// This file holds ingestTiles, the live tile/entry-bundle mirror writer PollHub
// calls on a verified, growing checkpoint. It walks the pure coordinate
// enumerations (tiles.TileCoords / tiles.BundleCoords) for the observed tree
// size, fetches each named hash tile and entry bundle over the existing transport
// primitives (logclient.FetchTile / FetchEntryBundle), and writes the raw BLOBs
// into the local store (store.RecordTile / RecordEntryBundle). It is the first
// production caller of those four seams: it feeds the SQLiteFetcher so the M2 fsck
// root-rebuild and the equivocation consistency-proof path read real mirrored
// tiles instead of always hitting the missing-tile skip. As it mirrors each entry
// bundle it also folds the bundle into the schema-agnostic iscc_index projection
// (logclient.BundleProjections -> store.RecordProjections), so a verified poll
// populates iscc_index for the later iscc_id -> leafIndex inclusion cross-check.
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
// bytes through the store CRUD, passing each coord's p qualifier straight through —
// the store owns the single p→width translation, so the follower no longer holds
// its own copy.
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
// records it in the store, passing the coord's Partial qualifier straight to
// RecordTile. The store maps p→width (a full tile, Partial == 0, lands at width
// tiles.TileWidth (256); a partial at its leaf count) — the follower performs no
// translation of its own.
func ingestHashTiles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time) error {
	for _, c := range tiles.TileCoords(treeSize) {
		raw, err := logclient.FetchTile(ctx, fetcher, baseURL, c.Level, c.Index, c.Partial)
		if err != nil {
			return fmt.Errorf("ingest tile level %d index %d p %d: %w", c.Level, c.Index, c.Partial, err)
		}
		if err := st.RecordTile(ctx, hubID, c.Level, c.Index, c.Partial, raw, observedAt); err != nil {
			return fmt.Errorf("record tile level %d index %d: %w", c.Level, c.Index, err)
		}
	}
	return nil
}

// ingestEntryBundles fetches every entry bundle named by
// tiles.BundleCoords(treeSize) and records it in the store, passing the coord's
// Partial qualifier straight to RecordEntryBundle (the store owns the p→width
// translation), mirroring ingestHashTiles for the entry-bundle table. After mirroring each bundle
// it folds the same raw bytes into the schema-agnostic iscc_index projection
// (ADR-0008) via projectEntryBundle, so a verified poll populates iscc_index for the
// later iscc_id -> leafIndex inclusion cross-check.
func ingestEntryBundles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time) error {
	for _, c := range tiles.BundleCoords(treeSize) {
		raw, err := logclient.FetchEntryBundle(ctx, fetcher, baseURL, c.Index, c.Partial)
		if err != nil {
			return fmt.Errorf("ingest entry bundle index %d p %d: %w", c.Index, c.Partial, err)
		}
		if err := st.RecordEntryBundle(ctx, hubID, c.Index, c.Partial, raw, observedAt); err != nil {
			return fmt.Errorf("record entry bundle index %d: %w", c.Index, err)
		}
		if err := projectEntryBundle(ctx, st, hubID, c.Index, raw); err != nil {
			return err
		}
	}
	return nil
}

// projectEntryBundle decodes one mirrored entry bundle into per-leaf projection
// records and upserts them into iscc_index. baseSeq is the bundle's first absolute
// leaf index (c.Index * tiles.TileWidth, 256 leaves per bundle), so each leaf's Seq
// is absolute. logclient.Projection is copied field-by-field into
// store.ProjectionRecord at the call site (the store stays a leaf, never importing
// logclient). A malformed record or a store fault is a genuine decode/store fault
// (ADR-0008 + ADR-0006): it is wrapped and returned up through ingestTiles -> PollHub
// to abort the poll before accepted state advances — it is NOT a self-consistency
// violation and must never freeze the hub.
func projectEntryBundle(ctx context.Context, st *store.Store, hubID int64, bundleIndex uint64, raw []byte) error {
	projections, err := logclient.BundleProjections(raw, bundleIndex*tiles.TileWidth)
	if err != nil {
		return fmt.Errorf("project entry bundle index %d: %w", bundleIndex, err)
	}
	recs := make([]store.ProjectionRecord, len(projections))
	for i, p := range projections {
		recs[i] = store.ProjectionRecord{
			HubID:        hubID,
			Seq:          p.Seq,
			IsccID:       p.IsccID,
			NoteSchema:   p.NoteSchema,
			RecordSHA256: p.RecordSHA256,
		}
	}
	if err := st.RecordProjections(ctx, recs); err != nil {
		return fmt.Errorf("project entry bundle index %d: %w", bundleIndex, err)
	}
	return nil
}
