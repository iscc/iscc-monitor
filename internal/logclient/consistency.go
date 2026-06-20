// This file holds the pure RFC-6962 self-consistency triggers the follower runs
// against successive verified checkpoints from one hub (ADR-0006). A violation
// freezes the hub and preserves evidence; it never crashes — so each trigger is a
// pure verdict the stateful follower acts on, never a panic or error on any input.
//
// All three triggers live here:
//   - shrink — a hub presenting a verified checkpoint whose tree_size is strictly
//     smaller than a size this monitor already accepted for it (the hub rewrote
//     history backwards). A hub's tree_size is the monotonic committed-record
//     count it must never decrease (iscc-hub build_checkpoint guards
//     old_size >= size), so a strict decrease against this monitor is the shrink
//     violation.
//   - fork — a hub presenting a verified checkpoint at the *same* tree_size this
//     monitor already accepted for it but with a *different* root. A monotonic
//     append-only log has exactly one root per committed size, so a differing
//     root at an equal size is a split of history. This is a fixed-size
//     [rootBytes]byte array compare, so it needs no Merkle math either.
//   - equivocation — a hub presenting a verified checkpoint that *grows* the tree
//     (nextSize > prevSize) but whose RFC-6962 consistency proof fails to relate
//     the prior accepted root at prevSize to the new root at nextSize. An
//     append-only log can always prove a later root extends an earlier one, so a
//     non-verifying consistency proof for a growing pair is a split view
//     (iscc-log §10.2). This is the only trigger that needs Merkle math, so it
//     verifies the proof via github.com/transparency-dev/merkle.
//
// CheckEquivocation verifies a consistency proof it is GIVEN; obtaining the proof
// hashes from mirrored hash tiles is later tile-fetch work and is not done here.
package logclient

import (
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
)

// ViolationKind names a self-consistency trigger as a plain string, matching the
// store.Violation.Kind value the persistence step populates and the
// violations.kind column. The shrink, fork, and equivocation kinds all live here.
type ViolationKind string

// ViolationShrink is the kind string for a strict tree-size decrease.
const ViolationShrink ViolationKind = "shrink"

// ViolationFork is the kind string for a differing root at an equal tree size.
const ViolationFork ViolationKind = "fork"

// ViolationEquivocation is the kind string for a failing RFC-6962 consistency
// proof across a growing tree-size pair.
const ViolationEquivocation ViolationKind = "equivocation"

// CheckShrink reports whether a newly-observed verified tree size is a shrink
// against the previously-accepted size for the same hub — a strict decrease,
// which a hub's monotonic tree_size must never present (ADR-0006).
//
// It is true only when next < prev AND prev > 0. The boundaries are load-bearing:
//   - next == prev is a re-observation / candidate-fork, the fork trigger's
//     concern, not a shrink — false.
//   - next > prev is normal growth — false.
//   - prev == 0 means no size has been accepted yet (the fresh-store
//     FollowState{}.LastSize zero), so any next is growth, never a shrink — false.
//
// The caller maps FollowState.LastSize to prev and CheckpointInfo.TreeSize to
// next. CheckShrink never panics or errors on any uint64 pair: a violation
// freezes, never crashes.
func CheckShrink(prev, next uint64) bool {
	return prev > 0 && next < prev
}

// CheckFork reports whether a newly-observed verified checkpoint is a fork
// against the previously-accepted one for the same hub — the same committed
// tree_size carrying a different Merkle root, which a monotonic append-only log
// can never legitimately present (ADR-0006).
//
// It is true only when nextSize == prevSize AND nextRoot != prevRoot AND
// prevSize > 0. The boundaries are load-bearing:
//   - prevSize == 0 means no size has been accepted yet (the fresh-store
//     FollowState{}.LastSize zero and its zero [rootBytes]byte root), so any
//     observation is a first sighting, never a fork — false regardless of roots.
//   - nextSize != prevSize is shrink's concern (strict decrease) or normal
//     growth, neither of which is a fork even when the roots differ — false.
//   - nextRoot == prevRoot at an equal size is a benign re-observation
//     (store.RecordCheckpoint dedupes it on UNIQUE(hub_id, tree_size, root)) —
//     false.
//
// The [rootBytes]byte array is compared with !=, which Go defines elementwise on
// fixed-size byte arrays — no bytes import is needed. The caller maps
// FollowState.LastSize and the stored root at that size to prevSize/prevRoot and
// CheckpointInfo.TreeSize/Root to nextSize/nextRoot. CheckFork never panics or
// errors on any input: a violation freezes, never crashes.
func CheckFork(prevSize uint64, prevRoot [rootBytes]byte, nextSize uint64, nextRoot [rootBytes]byte) bool {
	return prevSize > 0 && nextSize == prevSize && nextRoot != prevRoot
}

// CheckEquivocation reports whether a newly-observed verified checkpoint that
// grows the tree is an equivocation against the previously-accepted one for the
// same hub — the RFC-6962 consistency proof fails to relate the prior accepted
// root at prevSize to the new root at nextSize, which an append-only log can never
// legitimately present (iscc-log §10.2, ADR-0006).
//
// It returns violated=true only for the strictly-growing case
// (prevSize > 0 && nextSize > prevSize) when proof.VerifyConsistency fails. The
// boundaries are load-bearing and are shrink's/fork's concern, not this trigger's;
// each returns (false, nil) without calling VerifyConsistency:
//   - prevSize == 0 means no size has been accepted yet (the fresh-store
//     FollowState{}.LastSize zero), so any observation is a first sighting, never
//     an equivocation — mirrors the CheckShrink/CheckFork prev>0 guard.
//   - nextSize == prevSize is fork's concern (a same-size root compare).
//   - nextSize < prevSize is shrink's concern (a strict decrease).
//
// Error vs. violation discipline (ADR-0006 "freeze, never crash"): a non-verifying
// but well-formed proof is a *verdict* (violated=true, err=nil), never a Go error
// that could abort the poll loop — the proof not verifying IS the evidence. A
// successful verification means the log is consistent — (false, nil). The returned
// err is reserved for obviously-malformed input: a prevRoot/nextRoot that is not
// exactly rootBytes long is a caller bug, returned as (false, non-nil err) without
// freezing on it.
//
// The caller maps FollowState.LastSize and the stored root at that size to
// prevSize/prevRoot, CheckpointInfo.TreeSize/Root to nextSize/nextRoot, and
// supplies the RFC-6962 consistency-proof hashes between the two sizes.
func CheckEquivocation(prevSize uint64, prevRoot [rootBytes]byte, nextSize uint64, nextRoot [rootBytes]byte, consistencyProof [][]byte) (violated bool, err error) {
	if prevSize == 0 || nextSize <= prevSize {
		// Fresh store, fork's same-size compare, or shrink's decrease — none is
		// this trigger's concern, and none calls VerifyConsistency.
		return false, nil
	}
	// A failing-yet-well-formed proof is the evidence: VerifyConsistency's error is
	// converted to a violated=true verdict, never surfaced as a poll error.
	if err := proof.VerifyConsistency(rfc6962.DefaultHasher, prevSize, nextSize, consistencyProof, prevRoot[:], nextRoot[:]); err != nil {
		return true, nil
	}
	return false, nil
}
