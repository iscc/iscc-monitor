// This file holds the pure RFC-6962 self-consistency triggers the follower runs
// against successive verified checkpoints from one hub (ADR-0006). A violation
// freezes the hub and preserves evidence; it never crashes — so each trigger is a
// pure verdict the stateful follower acts on, never a panic or error on any input.
//
// Of the three triggers this slice lands only the size-only one: shrink — a hub
// presenting a verified checkpoint whose tree_size is strictly smaller than a
// size this monitor already accepted for it (the hub rewrote history backwards).
// A hub's tree_size is the monotonic committed-record count it must never
// decrease (iscc-hub build_checkpoint guards old_size >= size), so a strict
// decrease against this monitor is the shrink violation.
//
// Fork (same size, different root) and equivocation (RFC-6962 consistency-proof
// failure) are deferred to the merkle-backed slices that land with the
// transparency-dev/merkle dependency and tile fixtures; they need Merkle math a
// pure size comparison does not. This file stays import-free of any new dep.
package logclient

// ViolationKind names a self-consistency trigger as a plain string, matching the
// store.Violation.Kind value the persistence step populates and the
// violations.kind column. Only the shrink kind exists here; fork and equivocation
// arrive with their merkle-backed detectors.
type ViolationKind string

// ViolationShrink is the kind string for a strict tree-size decrease.
const ViolationShrink ViolationKind = "shrink"

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
