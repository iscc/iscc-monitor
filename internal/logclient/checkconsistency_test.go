// Tests for CheckConsistency, the composed self-consistency verdict the follower
// acts on (ADR-0006). They drive the full shrink → fork → equivocation branch order
// plus the proof build and the narrow missing-tile swallow over a table that pins
// every load-bearing case: shrink (next<prev), same-size split view (next==prev,
// differing root), growing split view (a corrupt new root so the real proof fails to
// verify → equivocation), clean growth (a valid proof → not violated), the
// fresh-store prevSize==0 (always clean, never reaches the fetcher), and a
// missing-tile fetcher (proof build errors → not violated, no error).
//
// The roots and the valid consistency proof come from a real RFC-6962 tree
// (transparency-dev/merkle's testonly.Tree over rfc6962.DefaultHasher), served
// through the same in-test TileFetcher as proofbuilder_test.go. The prover
// (testonly.Tree.ConsistencyProof, via tileFetcherFor + ConsistencyProofFromTiles)
// and the composed verifier (CheckConsistency → CheckEquivocation →
// VerifyConsistency) are independent merkle paths, so the growing-split-view
// cross-check is ground truth, not a tautology.
package logclient

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// TestCheckConsistency drives the composed verdict over the full case table. The
// tree is the boundary-crossing fixture from proofbuilder_test.go (treeLeaves), so
// the growing-pair proof build exercises real hash tiles at the partial-tile edge.
func TestCheckConsistency(t *testing.T) {
	ctx := context.Background()
	tree := buildTree(treeLeaves)
	fetch := tileFetcherFor(t, tree, treeLeaves)

	const (
		prevSize = 200 // a non-power-of-two prior accepted size
		nextSize = 260 // grows across the 256-leaf tile boundary
	)
	prevRoot := rootArray(t, tree.HashAt(prevSize))
	nextRoot := rootArray(t, tree.HashAt(nextSize))

	// A corrupt new root at the growing size: the real consistency proof built from
	// the mirror cannot relate prevRoot to it, so VerifyConsistency fails and the
	// verdict is equivocation. Mirrors TestCheckEquivocation's non-vacuousness guard.
	corruptNextRoot := nextRoot
	corruptNextRoot[0] ^= 0xff
	if corruptNextRoot == nextRoot {
		t.Fatal("test setup: corrupted nextRoot must differ from nextRoot")
	}

	// A differing root at the SAME size is a fork (no proof build needed).
	forkRoot := prevRoot
	forkRoot[0] ^= 0xff
	if forkRoot == prevRoot {
		t.Fatal("test setup: fork root must differ from prevRoot")
	}

	// A fetcher that fails every read with a wrapped os.ErrNotExist, modelling tiles
	// not yet mirrored: the growing-pair proof build errors and must be swallowed to
	// (false, "", nil), never freezing the hub on a missing tile.
	missingTileFetch := func(context.Context, uint64, uint64, uint8) ([]byte, error) {
		return nil, fmt.Errorf("read tile: %w", os.ErrNotExist)
	}

	cases := []struct {
		name      string
		fetch     TileFetcher
		prevSize  uint64
		prevRoot  [rootBytes]byte
		prevFound bool
		info      CheckpointInfo
		wantViol  bool
		wantKind  ViolationKind
		wantErr   bool
	}{
		{
			name:      "shrink: next strictly smaller than prev",
			fetch:     fetch,
			prevSize:  nextSize,
			prevRoot:  nextRoot,
			prevFound: true,
			info:      CheckpointInfo{TreeSize: prevSize, Root: prevRoot},
			wantViol:  true,
			wantKind:  ViolationShrink,
		},
		{
			name:      "fork: same size differing root",
			fetch:     fetch,
			prevSize:  prevSize,
			prevRoot:  prevRoot,
			prevFound: true,
			info:      CheckpointInfo{TreeSize: prevSize, Root: forkRoot},
			wantViol:  true,
			wantKind:  ViolationFork,
		},
		{
			name:      "growing split view: real proof fails to verify against corrupt new root",
			fetch:     fetch,
			prevSize:  prevSize,
			prevRoot:  prevRoot,
			prevFound: true,
			info:      CheckpointInfo{TreeSize: nextSize, Root: corruptNextRoot},
			wantViol:  true,
			wantKind:  ViolationEquivocation,
		},
		{
			name:      "clean growth: valid proof verifies",
			fetch:     fetch,
			prevSize:  prevSize,
			prevRoot:  prevRoot,
			prevFound: true,
			info:      CheckpointInfo{TreeSize: nextSize, Root: nextRoot},
			wantViol:  false,
		},
		{
			name:      "fresh store prevSize zero is always clean",
			fetch:     fetch,
			prevSize:  0,
			prevRoot:  [rootBytes]byte{},
			prevFound: false,
			info:      CheckpointInfo{TreeSize: nextSize, Root: nextRoot},
			wantViol:  false,
		},
		{
			name:      "missing tile on growing pair: proof build error swallowed",
			fetch:     missingTileFetch,
			prevSize:  prevSize,
			prevRoot:  prevRoot,
			prevFound: true,
			info:      CheckpointInfo{TreeSize: nextSize, Root: nextRoot},
			wantViol:  false,
		},
		{
			name:      "no prior checkpoint stored: root-dependent checks skipped on growth",
			fetch:     fetch,
			prevSize:  prevSize,
			prevRoot:  [rootBytes]byte{},
			prevFound: false,
			info:      CheckpointInfo{TreeSize: nextSize, Root: nextRoot},
			wantViol:  false,
		},
		{
			name:      "no prior checkpoint stored: shrink still fires (size-only)",
			fetch:     fetch,
			prevSize:  nextSize,
			prevRoot:  [rootBytes]byte{},
			prevFound: false,
			info:      CheckpointInfo{TreeSize: prevSize, Root: prevRoot},
			wantViol:  true,
			wantKind:  ViolationShrink,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			violated, kind, err := CheckConsistency(ctx, tc.fetch, tc.prevSize, tc.prevRoot, tc.prevFound, tc.info)
			if tc.wantErr != (err != nil) {
				t.Fatalf("CheckConsistency err = %v, wantErr %v", err, tc.wantErr)
			}
			if violated != tc.wantViol {
				t.Errorf("CheckConsistency violated = %v, want %v", violated, tc.wantViol)
			}
			if kind != tc.wantKind {
				t.Errorf("CheckConsistency kind = %q, want %q", kind, tc.wantKind)
			}
		})
	}
}
