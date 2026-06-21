// This file holds ingestTiles, the live tile/entry-bundle mirror writer PollHub
// calls on a verified, growing checkpoint. It walks the pure coordinate
// enumerations (tiles.TileCoords / tiles.BundleCoords) for the observed tree
// size, fetches each named hash tile and entry bundle over the existing transport
// primitives (logclient.FetchTile / FetchEntryBundle), and writes the raw BLOBs
// into the local store (store.RecordTile / RecordEntryBundle). It is the first
// production caller of those four seams: it feeds the SQLiteFetcher so the M2 fsck
// root-rebuild and the equivocation consistency-proof path read real mirrored
// tiles instead of always hitting the missing-tile skip.
//
// The writer is transport + CRUD only — no signature, RFC-6962, or Merkle math —
// so a fetch fault is a genuine transport error returned up to PollHub (NOT a
// self-consistency violation, ADR-0006). Every coord is re-fetched and overwritten
// in place each verified growing poll (ADR-0005: re-fetch partials, idempotent
// full-tile re-write), so a mid-ingest fault leaves a partial mirror the next poll
// completes via the idempotent upsert.
package follower

import (
	"context"
	"fmt"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// ingestTiles mirrors every hash tile and entry bundle a complete mirror of a
// tree of treeSize must hold into the local store. It walks tiles.TileCoords then
// tiles.BundleCoords (both pure coordinate enumerations), fetches each over the
// injected Fetcher via the logclient transport primitives, and writes the raw
// bytes through the store CRUD keyed by the widthForP-translated width.
//
// observedAt is the verified observation time (injected by PollHub, never
// time.Now()) and is recorded as each row's updated_at. A fetch or store fault is
// wrapped and returned — it is a genuine transport/store error, not a violation,
// so PollHub surfaces it without freezing the hub and the next poll re-fetches the
// missing coords (idempotent upsert).
func ingestTiles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time) error {
	if err := ingestHashTiles(ctx, st, fetcher, hubID, baseURL, treeSize, observedAt); err != nil {
		return err
	}
	return ingestEntryBundles(ctx, st, fetcher, hubID, baseURL, treeSize, observedAt)
}

// ingestHashTiles fetches every hash tile named by tiles.TileCoords(treeSize) and
// records it in the store at the width the coord's Partial maps to (widthForP). A
// full tile (Partial == 0) is stored at width tiles.TileWidth (256), a partial at
// its leaf count.
func ingestHashTiles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time) error {
	for _, c := range tiles.TileCoords(treeSize) {
		raw, err := logclient.FetchTile(ctx, fetcher, baseURL, c.Level, c.Index, c.Partial)
		if err != nil {
			return fmt.Errorf("ingest tile level %d index %d p %d: %w", c.Level, c.Index, c.Partial, err)
		}
		if err := st.RecordTile(ctx, hubID, c.Level, c.Index, widthForP(c.Partial), raw, observedAt); err != nil {
			return fmt.Errorf("record tile level %d index %d: %w", c.Level, c.Index, err)
		}
	}
	return nil
}

// ingestEntryBundles fetches every entry bundle named by
// tiles.BundleCoords(treeSize) and records it in the store at the widthForP width,
// mirroring ingestHashTiles for the entry-bundle table.
func ingestEntryBundles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time) error {
	for _, c := range tiles.BundleCoords(treeSize) {
		raw, err := logclient.FetchEntryBundle(ctx, fetcher, baseURL, c.Index, c.Partial)
		if err != nil {
			return fmt.Errorf("ingest entry bundle index %d p %d: %w", c.Index, c.Partial, err)
		}
		if err := st.RecordEntryBundle(ctx, hubID, c.Index, widthForP(c.Partial), raw, observedAt); err != nil {
			return fmt.Errorf("record entry bundle index %d: %w", c.Index, err)
		}
	}
	return nil
}

// widthForP maps a tlog-tiles partial qualifier p (path-API "0 == full") to the
// store's width column (the actual leaf count: tiles.TileWidth for full, int(p)
// for a partial). This re-derives the store's unexported widthForP one-liner (the
// store's copy is private and must not be exported): the load-bearing translation
// is that a full coord (Partial == 0) must be stored at width 256, not 0, or the
// SQLiteFetcher (which looks a full tile up at width 256) cannot read it back.
func widthForP(p uint8) int {
	if p == 0 {
		return tiles.TileWidth
	}
	return int(p)
}
