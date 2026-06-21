// This file holds CheckConsistency, the single pure home for the self-consistency
// verdict the follower acts on (ADR-0006). It composes the three pure triggers in
// consistency.go — shrink, fork, equivocation — plus the proof source in
// proofbuilder.go into one table-testable decision, so the follower only looks up
// the prior accepted evidence and acts on the returned verdict.
//
// The decision lives here (not in the follower) because it is pure policy over a
// tile-fetch seam: every input is a value (prior size/root, the new observation) or
// the injected TileFetcher, and the output is a (violated, kind, err) verdict with
// no store, net, or os dependency. Keeping logclient free of any store import
// preserves the dependency direction follower -> logclient.
package logclient

import (
	"context"
	"fmt"
)

// CheckConsistency runs the three RFC-6962 self-consistency triggers
// (shrink/fork/equivocation) against the prior accepted checkpoint for a hub and
// returns the verdict the follower acts on (ADR-0006). The follower owns the
// persistence concerns — reading the prior root/found flag from the checkpoints
// store and pairing the prior raw bytes with this verdict — and passes the pure
// inputs in: prevSize is FollowState.LastSize, prevRoot/prevFound come from a
// checkpoints lookup at prevSize, info is the new verified observation, and fetch
// reads mirrored hash tiles (the follower passes SQLiteFetcher.ReadTile).
//
// The CheckShrink/CheckFork/CheckEquivocation prevSize>0 guards mean a fresh-store
// zero never trips a violation, so a never-advanced hub is always clean; the
// prevSize == 0 early return makes that explicit and skips the proof build. When no
// checkpoint is stored at prevSize (prevFound is false — a hub that advanced before
// this code existed), the root-dependent fork/equivocation checks are skipped while
// the size-only shrink check still runs.
//
// The three triggers are mutually exclusive by size — shrink is next<prev, fork is
// next==prev, equivocation is the growing-pair next>prev case — so they are
// evaluated in shrink → fork → equivocation order and the first true kind is used.
//
// Equivocation sources its RFC-6962 consistency proof from the LOCAL mirror only —
// ConsistencyProofFromTiles over the injected fetch — and never re-hits the hub. The
// roots compared are the prior ACCEPTED root at prevSize and the new observation's
// root (info.Root), never the contradicting-evidence row.
//
// Error vs. violation discipline (ADR-0006 "freeze, never crash"): CheckEquivocation
// already turns a non-verifying proof into a (violated=true, err=nil) verdict. But
// ConsistencyProofFromTiles returns a genuine Go error on a tile-fetch/parse fault —
// most commonly a missing tile (a wrapped os.ErrNotExist), since production does not
// yet mirror tiles (that is M2 work). Such an error is NOT an equivocation verdict:
// freezing on a missing tile would be a false positive, and aborting the poll would
// break the loop. So a proof-build error is swallowed to (false, "", nil) — "cannot
// evaluate equivocation this poll" — and the poll proceeds. This swallow is
// deliberately narrow: it suppresses only the proof-build error so a missing tile
// cannot freeze a hub. Only a CheckEquivocation err (today unreachable-by-type, kept
// for signature symmetry) surfaces as a wrapped Go error.
func CheckConsistency(ctx context.Context, fetch TileFetcher, prevSize uint64, prevRoot [rootBytes]byte, prevFound bool, info CheckpointInfo) (violated bool, kind ViolationKind, err error) {
	if prevSize == 0 {
		return false, "", nil
	}

	shrink := CheckShrink(prevSize, info.TreeSize)
	fork := prevFound && CheckFork(prevSize, prevRoot, info.TreeSize, info.Root)
	switch {
	case shrink:
		return true, ViolationShrink, nil
	case fork:
		return true, ViolationFork, nil
	default:
		// Equivocation: the only growing-pair trigger (info.TreeSize > prevSize),
		// reached only when the prior accepted checkpoint is on record (prevFound) so
		// prevRoot is the real prior root, not a zero placeholder.
		if !prevFound || info.TreeSize <= prevSize {
			return false, "", nil
		}
		proofHashes, perr := ConsistencyProofFromTiles(ctx, fetch, prevSize, info.TreeSize)
		if perr != nil {
			// Cannot build the proof from the mirror (most often: tiles not mirrored
			// yet). Skip the equivocation branch — never freeze on a missing tile.
			return false, "", nil
		}
		eq, eerr := CheckEquivocation(prevSize, prevRoot, info.TreeSize, info.Root, proofHashes)
		if eerr != nil {
			return false, "", fmt.Errorf("check equivocation at sizes %d->%d: %w", prevSize, info.TreeSize, eerr)
		}
		if eq {
			return true, ViolationEquivocation, nil
		}
		return false, "", nil
	}
}
