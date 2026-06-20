// This file builds an RFC-6962 consistency proof between two tree sizes purely
// from a hub's mirrored hash tiles — the proof *source* the equivocation trigger
// (CheckEquivocation) consumes. It is a port of tessera's
// client.ProofBuilder.ConsistencyProof / fetchNodes / nodeCache.GetNode
// (cauldron/tessera/client/client.go), stripped of its otel spans and its
// net/http TileFetcherFunc so it stays a pure, WASM-shareable leaf: the only
// dependency on the outside world is an injected tile-fetch closure.
//
// The fetch seam (TileFetcher) matches store.SQLiteFetcher.ReadTile exactly, so
// the follower can later pass SQLiteFetcher.ReadTile directly to source proof
// nodes from the local mirror without re-hitting the hub. The result is shaped as
// the [][]byte CheckEquivocation expects for its consistencyProof argument.
//
// Purity (Correctness rule: proof/verify is pure; keep WASM-shareable): this file
// imports only stdlib + the dep-clean merkle/{proof,compact,rfc6962} and
// tessera/api{,/layout} closures. It must not pull net / net/http /
// database/sql. The one I/O-ish import is os, used solely for the os.ErrNotExist
// sentinel surfaced from the fetcher (already in this package via checkpoint.go).
package logclient

import (
	"context"
	"fmt"

	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
)

// TileFetcher reads the raw bytes of one mirrored hash tile, identified by its
// tile-space level and index and the tlog-tiles partial qualifier p (p == 0 means
// "full"). Its shape matches tessera's TileFetcherFunc and, deliberately,
// store.SQLiteFetcher.ReadTile exactly — so the follower can pass
// SQLiteFetcher.ReadTile straight in. A missing tile must surface as a wrapped
// os.ErrNotExist so errors.Is survives (the SQLiteFetcher contract).
type TileFetcher func(ctx context.Context, level, index uint64, p uint8) ([]byte, error)

// ConsistencyProofFromTiles builds the RFC-6962 consistency proof relating the
// root at tree size smaller to the root at tree size larger, sourcing every proof
// node from mirrored hash tiles via fetch. The returned [][]byte feeds
// CheckEquivocation's consistencyProof argument directly.
//
// It ports tessera's ProofBuilder.ConsistencyProof + fetchNodes: proof.Consistency
// yields the node IDs sufficient to build the proof; each is resolved to a hash via
// getNode; proof.Nodes.Rehash then folds the ephemeral node and returns the proof.
//
// Boundaries: proof.Consistency returns no node IDs for smaller == 0 or
// smaller == larger, so Rehash yields an empty proof and this returns a nil slice
// cleanly — letting CheckEquivocation's guard short-circuit. A genuine tile-fetch
// or tile-parse fault is returned as a wrapped Go error (errors.Is(err,
// os.ErrNotExist) survives a missing tile), distinct from CheckEquivocation's
// "proof fails to verify = violation" verdict.
//
// larger is the log size passed to layout.PartialTileSize, so each tile's partial
// qualifier reflects the newer tree the proof is computed against (matching
// tessera's nodeCache.logSize).
func ConsistencyProofFromTiles(ctx context.Context, fetch TileFetcher, smaller, larger uint64) ([][]byte, error) {
	nodes, err := proof.Consistency(smaller, larger)
	if err != nil {
		return nil, fmt.Errorf("ConsistencyProofFromTiles: compute node list for (%d, %d): %w", smaller, larger, err)
	}

	// A simple per-call cache keyed by (tileLevel, tileIndex) avoids re-fetching
	// and re-parsing the same tile for sibling nodes within one proof (KISS — no
	// need to port the full nodeCache struct or its ephemeral-node map).
	tiles := make(map[tileKey]api.HashTile)
	hashes := make([][]byte, 0, len(nodes.IDs))
	for _, id := range nodes.IDs {
		h, err := getNode(ctx, fetch, tiles, larger, id)
		if err != nil {
			return nil, fmt.Errorf("ConsistencyProofFromTiles: get node %+v: %w", id, err)
		}
		hashes = append(hashes, h)
	}

	proofHashes, err := nodes.Rehash(hashes, rfc6962.DefaultHasher.HashChildren)
	if err != nil {
		return nil, fmt.Errorf("ConsistencyProofFromTiles: rehash proof: %w", err)
	}
	return proofHashes, nil
}

// tileKey identifies a fetched tile in the per-call cache.
type tileKey struct {
	level uint64
	index uint64
}

// getNode resolves one compact.NodeID to its tree-node hash by fetching the tile
// that contains it (caching it), then recomputing the node hash from the tile's
// bottom-row leaf hashes. It is a port of tessera's nodeCache.GetNode minus the
// ephemeral-node map (Rehash supplies ephemeral nodes here, never getNode).
func getNode(ctx context.Context, fetch TileFetcher, tiles map[tileKey]api.HashTile, logSize uint64, id compact.NodeID) ([]byte, error) {
	tileLevel, tileIndex, nodeLevel, nodeIndex := layout.NodeCoordsToTileAddress(uint64(id.Level), id.Index)
	key := tileKey{tileLevel, tileIndex}
	tile, ok := tiles[key]
	if !ok {
		p := layout.PartialTileSize(tileLevel, tileIndex, logSize)
		raw, err := fetch(ctx, tileLevel, tileIndex, p)
		if err != nil {
			return nil, fmt.Errorf("fetch tile (level %d, index %d, p %d): %w", tileLevel, tileIndex, p, err)
		}
		if err := tile.UnmarshalText(raw); err != nil {
			return nil, fmt.Errorf("parse tile (level %d, index %d): %w", tileLevel, tileIndex, err)
		}
		tiles[key] = tile
	}

	// Recompute the requested node hash from the tile's leaf hashes: a node at
	// nodeLevel spans 2**nodeLevel leaves, the nodeIndex-th such block.
	numLeaves := 1 << nodeLevel
	firstLeaf := int(nodeIndex) * numLeaves
	lastLeaf := firstLeaf + numLeaves
	if lastLeaf > len(tile.Nodes) {
		return nil, fmt.Errorf("tile (level %d, index %d) has %d leaves, need [%d, %d)", tileLevel, tileIndex, len(tile.Nodes), firstLeaf, lastLeaf)
	}

	rf := compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}
	r := rf.NewEmptyRange(0)
	for _, leaf := range tile.Nodes[firstLeaf:lastLeaf] {
		if err := r.Append(leaf, nil); err != nil {
			return nil, fmt.Errorf("fold tile leaf into range: %w", err)
		}
	}
	return r.GetRootHash(nil)
}
