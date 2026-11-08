// This file provides SQLiteFetcher — a read-only view over a hub's mirrored tiles,
// entry bundles, and latest checkpoint that structurally satisfies tessera's
// three-method Fetcher interface (ReadCheckpoint / ReadTile / ReadEntryBundle).
// It is the seam the deferred M1 equivocation branch, the M2 fsck root-rebuild,
// and the M3 canonical-path mirror read mirror BLOBs through — always from the
// local store, never re-hitting the hub.
//
// It deliberately does NOT import tessera/fsck or tessera/client: the conformance
// to fsck.Fetcher is structural (pinned by a var-assertion in the test only), so
// no net/http / otel / klog enters the store closure and store stays a leaf.
//
// The p-argument ↔ width mapping is the load-bearing translation. The Fetcher
// methods take p uint8 where p == 0 means "full" (the tlog-tiles path-API
// convention), while the SQLite width column stores the actual leaf count (256 for
// full). So a p == 0 request queries width = 256 and a p > 0 request queries
// width = int(p). A missing row is mapped to os.ErrNotExist so errors.Is survives
// for the partial→full fallback (replicated inline from tessera's
// internal/fetcher.PartialOrFullResource, which is not importable).
package store

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/iscc/iscc-monitor/internal/tiles"
)

// SQLiteFetcher reads one hub's mirrored tlog-tiles artifacts back out of the
// store, satisfying tessera's Fetcher interface (ReadCheckpoint / ReadTile /
// ReadEntryBundle) structurally. It holds a *Store and the HubID it serves; reads
// go through the same-package ReadTileBlob / ReadEntryBundleBlob /
// LatestCheckpointRaw helpers, so the store's single open connection is reused and
// no net/http dependency enters the closure.
type SQLiteFetcher struct {
	Store *Store
	HubID int64
}

// ReadCheckpoint returns the raw signed-note bytes of the hub's most-recently
// observed checkpoint (the highest tree_size). A hub with no checkpoint row yet
// returns a wrapped os.ErrNotExist so errors.Is(err, os.ErrNotExist) holds for any
// fsck consumer.
func (f SQLiteFetcher) ReadCheckpoint(ctx context.Context) ([]byte, error) {
	raw, found, err := f.Store.LatestCheckpointRaw(ctx, f.HubID)
	if err != nil {
		return nil, fmt.Errorf("SQLiteFetcher.ReadCheckpoint: hub %d: %w", f.HubID, err)
	}
	if !found {
		return nil, fmt.Errorf("SQLiteFetcher.ReadCheckpoint: hub %d: %w", f.HubID, os.ErrNotExist)
	}
	return raw, nil
}

// ReadTile returns the mirrored hash tile BLOB at (level l, index i) for the
// partial qualifier p. It maps p to the stored width (p == 0 → 256 full, else
// int(p)) and replicates tessera's PartialOrFullResource fallback: a p > 0 request
// that misses falls back to the full (width 256) tile; a p == 0 miss returns a
// wrapped os.ErrNotExist.
func (f SQLiteFetcher) ReadTile(ctx context.Context, l, i uint64, p uint8) ([]byte, error) {
	data, err := f.readTileAt(ctx, l, i, p)
	if errors.Is(err, os.ErrNotExist) && p > 0 {
		// The partial may have been promoted to a full tile as the tree grew; try
		// the full resource (PartialOrFullResource semantics).
		return f.readTileAt(ctx, l, i, 0)
	}
	return data, err
}

// readTileAt reads exactly the tile at the width that p maps to (no fallback),
// returning a wrapped os.ErrNotExist when the row is absent.
func (f SQLiteFetcher) readTileAt(ctx context.Context, l, i uint64, p uint8) ([]byte, error) {
	width := widthForP(p)
	data, found, err := f.Store.ReadTileBlob(ctx, f.HubID, l, i, width)
	if err != nil {
		return nil, fmt.Errorf("SQLiteFetcher.ReadTile: hub %d level %d index %d width %d: %w", f.HubID, l, i, width, err)
	}
	if !found {
		return nil, fmt.Errorf("SQLiteFetcher.ReadTile: hub %d level %d index %d width %d: %w", f.HubID, l, i, width, os.ErrNotExist)
	}
	return data, nil
}

// ReadEntryBundle returns the mirrored entry bundle BLOB at index i for the partial
// qualifier p, with the same p→width mapping and partial→full fallback as ReadTile.
func (f SQLiteFetcher) ReadEntryBundle(ctx context.Context, i uint64, p uint8) ([]byte, error) {
	data, err := f.readEntryBundleAt(ctx, i, p)
	if errors.Is(err, os.ErrNotExist) && p > 0 {
		return f.readEntryBundleAt(ctx, i, 0)
	}
	return data, err
}

// readEntryBundleAt reads exactly the entry bundle at the width that p maps to (no
// fallback), returning a wrapped os.ErrNotExist when the row is absent.
func (f SQLiteFetcher) readEntryBundleAt(ctx context.Context, i uint64, p uint8) ([]byte, error) {
	width := widthForP(p)
	data, found, err := f.Store.ReadEntryBundleBlob(ctx, f.HubID, i, width)
	if err != nil {
		return nil, fmt.Errorf("SQLiteFetcher.ReadEntryBundle: hub %d index %d width %d: %w", f.HubID, i, width, err)
	}
	if !found {
		return nil, fmt.Errorf("SQLiteFetcher.ReadEntryBundle: hub %d index %d width %d: %w", f.HubID, i, width, os.ErrNotExist)
	}
	return data, nil
}

// widthForP maps a tlog-tiles partial qualifier p (path-API "0 == full") to the
// stored width column (the actual leaf count: tiles.TileWidth for full, int(p) for
// a partial). This is the load-bearing translation: a full-tile request (p == 0)
// must look up width 256, not 0.
func widthForP(p uint8) int {
	if p == 0 {
		return tiles.TileWidth
	}
	return int(p)
}
