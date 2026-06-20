// This file holds the pure RFC-6962 self-consistency triggers the follower runs
// against successive verified checkpoints from one hub (ADR-0006). A violation
// freezes the hub and preserves evidence; it never crashes — so each trigger is a
// pure verdict the stateful follower acts on, never a panic or error on any input.
//
// Two of the three triggers are dep-free and land here:
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
//
// Equivocation (RFC-6962 consistency-proof failure across growing sizes) is the
// only trigger deferred to a merkle-backed slice that lands with the
// transparency-dev/merkle dependency and tile fixtures; it needs Merkle math a
// pure size/root comparison does not. This file stays import-free of any new dep.
package logclient

// ViolationKind names a self-consistency trigger as a plain string, matching the
// store.Violation.Kind value the persistence step populates and the
// violations.kind column. The shrink and fork kinds exist here; equivocation
// arrives with its merkle-backed detector.
type ViolationKind string

// ViolationShrink is the kind string for a strict tree-size decrease.
const ViolationShrink ViolationKind = "shrink"

// ViolationFork is the kind string for a differing root at an equal tree size.
const ViolationFork ViolationKind = "fork"

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
