// This file is the golden test for ConsistencyProofFromTiles: it builds a real
// RFC-6962 tree with merkle's testonly.Tree (over rfc6962.DefaultHasher), serves
// its hash tiles through an in-test TileFetcher, and asserts the tile-sourced
// proof byte-equals the tree's own ConsistencyProof and verifies via
// VerifyConsistency. The prover (testonly.Tree.ConsistencyProof) and the
// tile-sourced builder are independent code paths, so the byte-match is ground
// truth, not a tautology. The ~300-leaf tree crosses a full 256-leaf tile boundary
// so the node→tile mapping is exercised at level-0 index 0 (full) and index 1
// (partial).
package logclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
)

// treeLeaves is the tree size for the golden vectors: > 256 so the level-0 tile at
// index 0 is full (256 leaves) and the tile at index 1 is partial (44 leaves),
// exercising both the full and partial qualifiers in the node→tile mapping.
const treeLeaves = 300

// buildTree returns a testonly.Tree of n leaves over the RFC-6962 default hasher.
func buildTree(n int) *testonly.Tree {
	tree := testonly.New(rfc6962.DefaultHasher)
	for i := range n {
		tree.AppendData([]byte(fmt.Sprintf("leaf-%d", i)))
	}
	return tree
}

// nodeHash recomputes the tree-node hash at (treeLevel, treeIndex) by folding the
// leaf hashes that node covers through a compact range — the same recompute the
// builder does, so the synthesized tiles are consistent with the tree. The node
// covers leaves [treeIndex<<treeLevel, (treeIndex+1)<<treeLevel) within size.
func nodeHash(t *testing.T, tree *testonly.Tree, treeLevel, treeIndex, size uint64) []byte {
	t.Helper()
	first := treeIndex << treeLevel
	last := (treeIndex + 1) << treeLevel
	if last > size {
		last = size
	}
	rf := compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}
	r := rf.NewEmptyRange(0)
	for i := first; i < last; i++ {
		if err := r.Append(tree.LeafHash(i), nil); err != nil {
			t.Fatalf("Append leaf %d: %v", i, err)
		}
	}
	h, err := r.GetRootHash(nil)
	if err != nil {
		t.Fatalf("GetRootHash for node (%d, %d): %v", treeLevel, treeIndex, err)
	}
	return h
}

// tileFetcherFor returns a TileFetcher serving the hash tiles of tree within a log
// of logSize leaves. A tile at (tileLevel, tileIndex) holds the tree's bottom-row
// nodes at tree-level tileLevel*layout.TileHeight, serialized in tlog-tiles
// concatenated-hash form via api.HashTile.MarshalText. A coordinate with no leaves
// (off the right edge) returns a wrapped os.ErrNotExist, matching the SQLiteFetcher
// contract.
func tileFetcherFor(t *testing.T, tree *testonly.Tree, logSize uint64) TileFetcher {
	t.Helper()
	return func(_ context.Context, level, index uint64, _ uint8) ([]byte, error) {
		treeLevel := level * layout.TileHeight
		// The tile's bottom row spans tree nodes at treeLevel; each such node
		// covers 1<<treeLevel leaves. Emit as many nodes as logSize provides for
		// this tile, up to a full tile (256).
		firstNode := index * layout.TileWidth
		var nodes [][]byte
		for n := uint64(0); n < layout.TileWidth; n++ {
			nodeIndex := firstNode + n
			if nodeIndex<<treeLevel >= logSize {
				break
			}
			nodes = append(nodes, nodeHash(t, tree, treeLevel, nodeIndex, logSize))
		}
		if len(nodes) == 0 {
			return nil, fmt.Errorf("tile (level %d, index %d) empty: %w", level, index, os.ErrNotExist)
		}
		raw, err := api.HashTile{Nodes: nodes}.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText tile (level %d, index %d): %v", level, index, err)
		}
		return raw, nil
	}
}

// TestConsistencyProofFromTiles asserts the tile-sourced consistency proof matches
// the tree's own proof byte-for-byte on a boundary-crossing pair and verifies via
// VerifyConsistency for several growing pairs, plus the empty-proof boundaries.
func TestConsistencyProofFromTiles(t *testing.T) {
	ctx := context.Background()
	tree := buildTree(treeLeaves)
	fetch := tileFetcherFor(t, tree, treeLeaves)

	// Pairs chosen to exercise the node→tile mapping across the 256-leaf boundary:
	// (1) both sizes inside the first full tile; (2) smaller inside tile 0, larger
	// inside the partial tile 1 (the boundary-crossing case); (3) both inside the
	// partial tile 1. (5,300) and (260,300) require nodes from tile index 1.
	pairs := []struct{ smaller, larger uint64 }{
		{5, 200},
		{200, 300},
		{5, 300},
		{260, 300},
		{1, 256},
	}
	for _, p := range pairs {
		got, err := ConsistencyProofFromTiles(ctx, fetch, p.smaller, p.larger)
		if err != nil {
			t.Fatalf("ConsistencyProofFromTiles(%d, %d): %v", p.smaller, p.larger, err)
		}
		want, err := tree.ConsistencyProof(p.smaller, p.larger)
		if err != nil {
			t.Fatalf("tree.ConsistencyProof(%d, %d): %v", p.smaller, p.larger, err)
		}
		if !equalProof(got, want) {
			t.Errorf("ConsistencyProofFromTiles(%d, %d) =\n%s\nwant\n%s", p.smaller, p.larger, fmtProof(got), fmtProof(want))
		}
		if err := proof.VerifyConsistency(rfc6962.DefaultHasher, p.smaller, p.larger, got, tree.HashAt(p.smaller), tree.HashAt(p.larger)); err != nil {
			t.Errorf("VerifyConsistency(%d, %d) on tile-built proof: %v", p.smaller, p.larger, err)
		}
	}
}

// TestConsistencyProofFromTilesEmptyBoundaries asserts the smaller==0 and
// smaller==larger boundaries return an empty proof cleanly without touching the
// fetcher (proof.Consistency yields no node IDs), so CheckEquivocation's guard
// short-circuits correctly.
func TestConsistencyProofFromTilesEmptyBoundaries(t *testing.T) {
	ctx := context.Background()
	// A fetcher that fails the test if ever called — the empty boundaries must not
	// fetch any tile.
	fetch := func(context.Context, uint64, uint64, uint8) ([]byte, error) {
		t.Fatal("empty-boundary proof must not fetch any tile")
		return nil, nil
	}
	for _, p := range []struct{ smaller, larger uint64 }{{0, 300}, {0, 0}, {300, 300}} {
		got, err := ConsistencyProofFromTiles(ctx, fetch, p.smaller, p.larger)
		if err != nil {
			t.Fatalf("ConsistencyProofFromTiles(%d, %d): %v", p.smaller, p.larger, err)
		}
		if len(got) != 0 {
			t.Errorf("ConsistencyProofFromTiles(%d, %d) = %v, want empty", p.smaller, p.larger, got)
		}
	}
}

// TestConsistencyProofFromTilesMissingTile asserts a genuine tile-fetch fault
// surfaces as a wrapped Go error preserving os.ErrNotExist, distinct from a
// "proof fails to verify = violation" verdict.
func TestConsistencyProofFromTilesMissingTile(t *testing.T) {
	ctx := context.Background()
	fetch := func(context.Context, uint64, uint64, uint8) ([]byte, error) {
		return nil, fmt.Errorf("read tile: %w", os.ErrNotExist)
	}
	_, err := ConsistencyProofFromTiles(ctx, fetch, 5, 300)
	if err == nil {
		t.Fatal("ConsistencyProofFromTiles with a missing tile returned nil error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error %v does not wrap os.ErrNotExist", err)
	}
}

// equalProof reports whether two proofs are byte-identical element by element.
func equalProof(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytesEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// bytesEqual compares two byte slices without importing bytes (keeping the test's
// import surface minimal and matching consistency.go's no-bytes convention).
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// fmtProof renders a proof as hex-ish strings for a readable failure message.
func fmtProof(p [][]byte) string {
	s := ""
	for _, h := range p {
		s += fmt.Sprintf("  %x\n", h)
	}
	return s
}
