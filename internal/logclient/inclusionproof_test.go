// This file is the golden test for InclusionProofFromTiles: it builds a real
// RFC-6962 tree with merkle's testonly.Tree (over rfc6962.DefaultHasher), serves
// its hash tiles through an in-test TileFetcher, and asserts the tile-sourced
// inclusion proof byte-equals the tree's own InclusionProof and verifies via
// VerifyInclusion. The prover (testonly.Tree.InclusionProof), the verifier
// (proof.VerifyInclusion), and the tile-sourced builder are three independent
// merkle code paths, so the cross-check is ground truth, not a tautology. It
// reuses buildTree / tileFetcherFor / equalProof / fmtProof from
// proofbuilder_test.go (same package), exercising leaf indices that span the
// 256-leaf tile boundary (tile 0 full, tile 1 partial).
package logclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
)

// TestInclusionProofFromTiles asserts the tile-sourced inclusion proof matches the
// tree's own proof byte-for-byte and verifies via VerifyInclusion for leaf indices
// spanning the 256-leaf tile boundary — indices inside the full tile 0 (0, 5, 200,
// 255), on the boundary (256), and inside the partial tile 1 (260, 299).
func TestInclusionProofFromTiles(t *testing.T) {
	ctx := context.Background()
	tree := buildTree(treeLeaves)
	fetch := tileFetcherFor(t, tree, treeLeaves)

	root := tree.HashAt(treeLeaves)
	for _, index := range []uint64{0, 5, 200, 255, 256, 260, 299} {
		got, err := InclusionProofFromTiles(ctx, fetch, index, treeLeaves)
		if err != nil {
			t.Fatalf("InclusionProofFromTiles(%d, %d): %v", index, treeLeaves, err)
		}
		want, err := tree.InclusionProof(index, treeLeaves)
		if err != nil {
			t.Fatalf("tree.InclusionProof(%d, %d): %v", index, treeLeaves, err)
		}
		if !equalProof(got, want) {
			t.Errorf("InclusionProofFromTiles(%d, %d) =\n%s\nwant\n%s", index, treeLeaves, fmtProof(got), fmtProof(want))
		}
		if err := proof.VerifyInclusion(rfc6962.DefaultHasher, index, treeLeaves, tree.LeafHash(index), got, root); err != nil {
			t.Errorf("VerifyInclusion(%d, %d) on tile-built proof: %v", index, treeLeaves, err)
		}
	}
}

// TestInclusionProofFromTilesMissingTile asserts a genuine tile-fetch fault surfaces
// as a wrapped Go error preserving os.ErrNotExist, distinct from a proof-content
// fault, mirroring TestConsistencyProofFromTilesMissingTile.
func TestInclusionProofFromTilesMissingTile(t *testing.T) {
	ctx := context.Background()
	fetch := func(context.Context, uint64, uint64, uint8) ([]byte, error) {
		return nil, fmt.Errorf("read tile: %w", os.ErrNotExist)
	}
	_, err := InclusionProofFromTiles(ctx, fetch, 5, treeLeaves)
	if err == nil {
		t.Fatal("InclusionProofFromTiles with a missing tile returned nil error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error %v does not wrap os.ErrNotExist", err)
	}
}

// TestInclusionProofFromTilesIndexOutOfRange asserts index >= size (proof.Inclusion's
// precondition) returns a wrapped non-nil error and does not panic — the fetcher is
// never reached.
func TestInclusionProofFromTilesIndexOutOfRange(t *testing.T) {
	ctx := context.Background()
	fetch := func(context.Context, uint64, uint64, uint8) ([]byte, error) {
		t.Fatal("out-of-range index must not fetch any tile")
		return nil, nil
	}
	if _, err := InclusionProofFromTiles(ctx, fetch, treeLeaves, treeLeaves); err == nil {
		t.Fatal("InclusionProofFromTiles(index == size) returned nil error")
	}
}
