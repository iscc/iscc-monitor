// This file holds ingestTiles, the live tile/entry-bundle mirror writer PollHub
// calls on a verified, growing checkpoint. It walks the pure coordinate
// enumerations (tiles.TileCoords / tiles.BundleCoords) for the observed tree
// size, fetches the named hash tiles and entry bundles the mirror does not already
// hold over the existing transport primitives (logclient.FetchTile /
// FetchEntryBundle), and writes the raw BLOBs
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
// self-consistency violation, ADR-0006). A mid-ingest fault leaves the coords it had
// not yet reached ABSENT, and the next poll fetches them (a coord is skipped only
// when its bytes are already stored). Recovery is therefore fill-the-gaps, never
// overwrite: bytes already mirrored at full width are trusted as-is and are never
// re-fetched, so a completed coord that is wrong in the mirror stays wrong. Nothing
// in this walk repairs it — see issues.md "The mirror has no repair path".
//
// Fetch only what can have changed (ADR-0005 partial-tile discipline). A completed
// tlog-tiles tile or entry bundle is immutable — hubs serve those paths
// `cache-control: immutable` — so once its bytes are mirrored at full width there is
// nothing to re-fetch, and the mirror IS the cache. Each walk consults the store's
// already-mirrored-in-full sets (store.MirroredFullTiles / MirroredFullEntryBundles)
// and skips those coords, so a poll costs one request per coord the mirror lacks plus
// one per partial (`.p/<W>`), which gains leaves as the tree grows and is re-fetched
// and overwritten every poll. Per-poll OUTBOUND cost is therefore proportional to the
// tree's GROWTH, not to its total size; the LOCAL cost is not — each walk still
// enumerates every coord and reads the whole already-mirrored-in-full set. The signed
// checkpoint is deliberately outside this rule: PollHub re-fetches it every poll —
// that fetch is the observation — and it is never cached.
package follower

import (
	"context"
	"fmt"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// ingestTiles brings the local mirror up to a complete copy of a tree of treeSize.
// It walks tiles.TileCoords then tiles.BundleCoords (both pure coordinate
// enumerations), fetches over the injected Fetcher every coord the mirror does not
// already hold immutably, and writes the raw bytes through the store CRUD, passing
// each coord's p qualifier straight through — the store owns the single p→width
// translation, so the follower holds no copy of it.
//
// force is the repair path. Normally (force false) a coord already mirrored at full
// width is served from the mirror, which is the whole point of the cache. But the
// mirror is then authoritative for those bytes, and nothing else re-checks them: a
// completed coord that is wrong locally would be trusted forever, and both the
// consistency proof and the fsck root-rebuild read it. force true bypasses the skip
// and re-fetches EVERY enumerated coord from the hub, so the caller can re-derive a
// verdict from authoritative bytes instead of a possibly-corrupt cache. PollHub uses
// it in exactly two places — before convicting a hub of a self-consistency violation,
// and after a failed root rebuild — so a healthy realm never pays for it.
//
// observedAt is the verified observation time (injected by PollHub, never
// time.Now()) and is recorded as each row's updated_at. A fetch or store fault is
// wrapped and returned — it is a genuine transport/store error, not a violation,
// so PollHub surfaces it without freezing the hub and the next poll fetches the
// coords still ABSENT from the mirror (a coord already stored in full is skipped,
// not re-fetched).
func ingestTiles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time, force bool) error {
	if err := ingestHashTiles(ctx, st, fetcher, hubID, baseURL, treeSize, observedAt, force); err != nil {
		return err
	}
	return ingestEntryBundles(ctx, st, fetcher, hubID, baseURL, treeSize, observedAt, force)
}

// mirroredFullTiles reads the already-mirrored-in-full hash-tile set the walk skips
// on, or an empty set when force is set (nothing is skippable on the repair path).
func mirroredFullTiles(ctx context.Context, st *store.Store, hubID int64, force bool) (map[store.TileKey]struct{}, error) {
	if force {
		return nil, nil
	}
	return st.MirroredFullTiles(ctx, hubID)
}

// ingestHashTiles fetches every hash tile named by tiles.TileCoords(treeSize) that
// the mirror does not already hold in full, and records it in the store, passing the
// coord's Partial qualifier straight to RecordTile. The store maps p→width (a full
// tile, Partial == 0, lands at width tiles.TileWidth (256); a partial at its leaf
// count) — the follower performs no translation of its own.
//
// The one set read (MirroredFullTiles) replaces one network round-trip per already-
// mirrored coord: a full coord (Partial == 0) whose bytes are already stored is
// immutable and skipped, while every partial is always re-fetched. A skipped coord is
// not a gap — the bytes are in the store, which is what the fsck rebuild, the
// consistency proof, and the served mirror all read. Membership is decided by the
// width column alone: the stored bytes are trusted UNVALIDATED, so a completed coord
// that is wrong in the mirror is skipped forever (issues.md "The mirror has no repair
// path").
func ingestHashTiles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time, force bool) error {
	mirrored, err := mirroredFullTiles(ctx, st, hubID, force)
	if err != nil {
		return fmt.Errorf("mirrored full tiles: %w", err)
	}
	for _, c := range tiles.TileCoords(treeSize) {
		if _, full := mirrored[store.TileKey{Level: c.Level, Index: c.Index}]; c.Partial == 0 && full {
			continue
		}
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
// tiles.BundleCoords(treeSize) that the mirror does not already hold in full and
// records it in the store, passing the coord's Partial qualifier straight to
// RecordEntryBundle (the store owns the p→width translation), mirroring
// ingestHashTiles for the entry-bundle table — including its already-mirrored-in-full
// skip, which is where the bulk of the saved bytes are (a full bundle is 256 whole
// records; a full hash tile is 256 hashes).
//
// Each fetched bundle's raw bytes are folded into the schema-agnostic iscc_index
// projection (ADR-0008) via projectEntryBundle BEFORE RecordEntryBundle stores them.
// The order is load-bearing: RecordEntryBundle is what makes the coord skippable, so
// writing it last is what upholds the invariant "a full bundle in the mirror implies
// its projection was written". A projection fault therefore leaves the coord absent
// from the mirror, and the next poll re-fetches and re-folds it.
func ingestEntryBundles(ctx context.Context, st *store.Store, fetcher logclient.Fetcher, hubID int64, baseURL string, treeSize uint64, observedAt time.Time, force bool) error {
	mirrored, err := mirroredFullEntryBundles(ctx, st, hubID, force)
	if err != nil {
		return fmt.Errorf("mirrored full entry bundles: %w", err)
	}
	for _, c := range tiles.BundleCoords(treeSize) {
		if _, full := mirrored[c.Index]; c.Partial == 0 && full {
			continue
		}
		raw, err := logclient.FetchEntryBundle(ctx, fetcher, baseURL, c.Index, c.Partial)
		if err != nil {
			return fmt.Errorf("ingest entry bundle index %d p %d: %w", c.Index, c.Partial, err)
		}
		leaves, err := projectEntryBundle(ctx, st, hubID, c.Index, raw)
		if err != nil {
			return err
		}
		// Admission gate, the entry-bundle twin of RecordTile's length check: a
		// COMPLETED bundle carries exactly TileWidth records. An empty body decodes to
		// zero leaves without error, so without this a zero-byte 200 would be admitted
		// at full width and skipped forever, silently truncating both the mirror and
		// the iscc_index. Only full coords are checked — a partial legitimately holds
		// any count up to TileWidth.
		if c.Partial == 0 && leaves != tiles.TileWidth {
			return fmt.Errorf("ingest entry bundle index %d: full bundle decoded %d records, want %d", c.Index, leaves, tiles.TileWidth)
		}
		if err := st.RecordEntryBundle(ctx, hubID, c.Index, c.Partial, raw, observedAt); err != nil {
			return fmt.Errorf("record entry bundle index %d: %w", c.Index, err)
		}
	}
	return nil
}

// mirroredFullEntryBundles reads the already-mirrored-and-projected entry-bundle set
// the walk skips on, or an empty set when force is set (the repair path re-fetches
// every coord).
func mirroredFullEntryBundles(ctx context.Context, st *store.Store, hubID int64, force bool) (map[uint64]struct{}, error) {
	if force {
		return nil, nil
	}
	return st.MirroredFullEntryBundles(ctx, hubID)
}

// projectEntryBundle decodes one mirrored entry bundle into per-leaf projection
// records and upserts them into iscc_index, returning the number of leaves the bundle
// decoded to (the caller's completeness check). baseSeq is the bundle's first absolute
// leaf index (bundleIndex * tiles.TileWidth, 256 leaves per bundle), so each leaf's
// Seq is absolute. logclient.Projection is copied field-by-field into
// store.ProjectionRecord at the call site (the store stays a leaf, never importing
// logclient). A malformed record or a store fault is a genuine decode/store fault
// (ADR-0008 + ADR-0006): it is wrapped and returned up through ingestTiles -> PollHub
// to abort the poll before accepted state advances — it is NOT a self-consistency
// violation and must never freeze the hub.
func projectEntryBundle(ctx context.Context, st *store.Store, hubID int64, bundleIndex uint64, raw []byte) (int, error) {
	projections, err := logclient.BundleProjections(raw, bundleIndex*tiles.TileWidth)
	if err != nil {
		return 0, fmt.Errorf("project entry bundle index %d: %w", bundleIndex, err)
	}
	recs := make([]store.ProjectionRecord, len(projections))
	for i, p := range projections {
		recs[i] = store.ProjectionRecord{
			HubID:         hubID,
			Seq:           p.Seq,
			IsccID:        p.IsccID,
			NoteSchema:    p.NoteSchema,
			NoteTimestamp: p.Timestamp,
			RecordSHA256:  p.RecordSHA256,
		}
	}
	if err := st.RecordProjections(ctx, recs); err != nil {
		return 0, fmt.Errorf("project entry bundle index %d: %w", bundleIndex, err)
	}
	return len(projections), nil
}
