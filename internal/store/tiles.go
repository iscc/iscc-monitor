// This file adds the write/read CRUD for the mirrored tlog-tiles BLOBs — the
// tiles and entry_bundles tables (ADR-0005) — plus the size-agnostic
// LatestCheckpointRaw read the SQLiteFetcher needs. RecordTile / RecordEntryBundle
// are the write side; ReadTileBlob / ReadEntryBundleBlob / LatestCheckpointRaw the
// read side composed by fetcher.go.
//
// The write side speaks the tlog-tiles partial qualifier p (path-API "0 == full"):
// RecordTile / RecordEntryBundle take p uint8 and translate it to the stored width
// via the package-private widthForP (fetcher.go) — the single p→width authority,
// shared with the SQLiteFetcher read side, so the translation can never drift in two
// places.
//
// Partial-tile discipline (ADR-0005): a row is keyed by width. is_full is set to 1
// only for a full tile (width == 256, per tiles.IsFull); a partial is width < 256
// with is_full = 0. A partial re-fetched at the SAME width overwrites in place via the
// composite-PK upsert; because width is part of that key, a partial that GAINS leaves
// lands as a new row instead (issues.md "Partial tile/bundle rows accumulate"). A full
// tile is immutable, so an idempotent re-write of identical bytes is harmless. sha256
// is recorded for fsck cross-checks and updated_at from the caller-supplied observedAt
// (mirroring RecordHubKey, which takes a time rather than calling time.Now()).
//
// MirroredFullTiles / MirroredFullEntryBundles are the set-shaped read that lets the
// ingest walk act on that discipline: they name the coords whose bytes are already
// mirrored at full width, so the mirror serves as the fetch cache for the immutable
// (completed) part of a hub's log and only partials are re-fetched each poll. Both
// answer "is a row present at full width", NOT "are the stored bytes correct" —
// nothing here validates length or re-checks the recorded sha256, so a completed coord
// admitted with bad bytes is trusted and skipped indefinitely.
//
// store stays a leaf: this file uses crypto/sha256 (stdlib) and database/sql only;
// it does NOT import internal/logclient or anything pulling net/http into the store
// closure.
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/iscc/iscc-monitor/internal/tiles"
)

// RecordTile upserts one mirrored hash tile BLOB into the tiles table, keyed by
// (hub_id, level, index, width). It takes the tlog-tiles partial qualifier p (path-
// API "0 == full") and translates it to the stored width via the package-private
// widthForP (the single p→width authority, shared with the SQLiteFetcher read
// side). is_full is set to 1 only when tiles.IsFull(width) (width == 256); a partial
// (width < 256) stores is_full = 0 and is overwritten in place on a re-fetch via the
// composite-PK upsert. sha256 is the SHA-256 of data (for fsck cross-checks) and
// updated_at the caller-supplied observedAt (zero → NULL via unixOrNil). A re-write
// of identical full-tile bytes is idempotent.
func (s *Store) RecordTile(ctx context.Context, hubID int64, level, index uint64, p uint8, data []byte, observedAt time.Time) error {
	width := widthForP(p)
	// Admission gate: a hash tile carries exactly the hash count its path advertises —
	// width 32-byte hashes, at every tile level (verified against a live 300258-leaf
	// log: the full tiles are 8192 bytes, the level-0 `.p/226` is 7232, the level-1
	// `.p/148` is 4736, and the level-2 `.p/4` is 128). Rejecting a wrong-length body
	// keeps the mirror trustworthy as a fetch cache: a completed coord, once stored, is
	// skipped by the ingest walk forever, so an HTML error page, a truncated body, or an
	// empty body admitted at full width would be served and proof-read as authoritative
	// indefinitely. Length is not integrity (a right-length wrong-bytes body still gets
	// in — the forced re-walk repairs that); it is the cheap invariant that closes the
	// malformed-body class at the door.
	if len(data) != width*sha256.Size {
		return fmt.Errorf("store.RecordTile: hub %d level %d index %d width %d: %d bytes, want %d (%d hashes)",
			hubID, level, index, width, len(data), width*sha256.Size, width)
	}
	sum := sha256.Sum256(data)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO tiles (hub_id, level, tile_index, width, data, is_full, sha256, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?) "+
			"ON CONFLICT(hub_id, level, tile_index, width) DO UPDATE SET "+
			"data = excluded.data, is_full = excluded.is_full, sha256 = excluded.sha256, "+
			"updated_at = excluded.updated_at",
		hubID, int64(level), int64(index), width, data, boolToInt(tiles.IsFull(width)), sum[:], unixOrNil(observedAt),
	)
	if err != nil {
		return fmt.Errorf("store.RecordTile: hub %d level %d index %d width %d: %w", hubID, level, index, width, err)
	}
	return nil
}

// RecordEntryBundle upserts one mirrored entry bundle BLOB into the entry_bundles
// table, keyed by (hub_id, bundle_index, width). It takes the tlog-tiles partial
// qualifier p and translates it to the stored width via widthForP (the single
// p→width authority). Same partial-tile discipline as RecordTile: is_full = 1 only
// at width == 256; partials overwrite in place.
func (s *Store) RecordEntryBundle(ctx context.Context, hubID int64, bundleIndex uint64, p uint8, data []byte, observedAt time.Time) error {
	width := widthForP(p)
	sum := sha256.Sum256(data)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO entry_bundles (hub_id, bundle_index, width, data, is_full, sha256, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?) "+
			"ON CONFLICT(hub_id, bundle_index, width) DO UPDATE SET "+
			"data = excluded.data, is_full = excluded.is_full, sha256 = excluded.sha256, "+
			"updated_at = excluded.updated_at",
		hubID, int64(bundleIndex), width, data, boolToInt(tiles.IsFull(width)), sum[:], unixOrNil(observedAt),
	)
	if err != nil {
		return fmt.Errorf("store.RecordEntryBundle: hub %d index %d width %d: %w", hubID, bundleIndex, width, err)
	}
	return nil
}

// ReadTileBlob reads one mirrored hash tile BLOB back by (hub_id, level, index,
// width). An absent row returns (nil, false, nil) — absent is not an error,
// mirroring CheckpointAt / LookupHubKey — so the fetcher can map the miss to the
// os.ErrNotExist contract. width is the actual leaf count (256 for full).
func (s *Store) ReadTileBlob(ctx context.Context, hubID int64, level, index uint64, width int) ([]byte, bool, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT data FROM tiles WHERE hub_id = ? AND level = ? AND tile_index = ? AND width = ?",
		hubID, int64(level), int64(index), width,
	).Scan(&data)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, false, nil
	case err != nil:
		return nil, false, fmt.Errorf("store.ReadTileBlob: hub %d level %d index %d width %d: %w", hubID, level, index, width, err)
	}
	return data, true, nil
}

// ReadEntryBundleBlob reads one mirrored entry bundle BLOB back by (hub_id,
// bundle_index, width). An absent row returns (nil, false, nil), mirroring
// ReadTileBlob. width is the actual leaf count (256 for full).
func (s *Store) ReadEntryBundleBlob(ctx context.Context, hubID int64, bundleIndex uint64, width int) ([]byte, bool, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT data FROM entry_bundles WHERE hub_id = ? AND bundle_index = ? AND width = ?",
		hubID, int64(bundleIndex), width,
	).Scan(&data)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, false, nil
	case err != nil:
		return nil, false, fmt.Errorf("store.ReadEntryBundleBlob: hub %d index %d width %d: %w", hubID, bundleIndex, width, err)
	}
	return data, true, nil
}

// TileKey identifies one mirrored hash tile within a hub by its tile-level and its
// index in that level's tile space — the coordinate pair, without the width. It is
// the set key MirroredFullTiles returns.
type TileKey struct {
	// Level is the tile-level (0, 1, 2, …); tile-level L spans tree-level L*8.
	Level uint64
	// Index is the tile's index within its level in tile space.
	Index uint64
}

// MirroredFullTiles returns the set of coords at which this hub already holds a FULL
// hash tile — a row stored at the width widthForP maps the path-API "full" qualifier
// (p == 0) to, i.e. tiles.TileWidth.
//
// A completed tlog-tiles hash tile is immutable: its 256 hashes are fixed once the
// tree covers them, which is why hubs serve those paths `cache-control: immutable`.
// So a coord in this set never needs re-fetching from the hub, and the mirror is
// itself the cache. Partial rows are deliberately excluded — a partial gains hashes
// as the tree grows and must be re-fetched every poll (ADR-0005 partial-tile
// discipline). A hub with nothing mirrored yet returns an empty (non-nil) set, not an
// error: absent is not an error, mirroring ReadTileBlob.
func (s *Store) MirroredFullTiles(ctx context.Context, hubID int64) (map[TileKey]struct{}, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT level, tile_index FROM tiles WHERE hub_id = ? AND width = ?",
		hubID, widthForP(0),
	)
	if err != nil {
		return nil, fmt.Errorf("store.MirroredFullTiles: hub %d: query: %w", hubID, err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[TileKey]struct{})
	for rows.Next() {
		var level, index int64
		if err := rows.Scan(&level, &index); err != nil {
			return nil, fmt.Errorf("store.MirroredFullTiles: hub %d: scan: %w", hubID, err)
		}
		out[TileKey{Level: uint64(level), Index: uint64(index)}] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.MirroredFullTiles: hub %d: rows: %w", hubID, err)
	}
	return out, nil
}

// MirroredFullEntryBundles returns the set of bundle indexes at which this hub
// already holds a FULL entry bundle AND that bundle's iscc_index projection is
// present, the entry-bundle twin of MirroredFullTiles (same immutability rationale,
// same full-width key, same empty-set-not-error contract).
//
// The projection half is what makes the set safe to skip on: presence of the BLOB
// alone does not establish the target.md invariant "a full entry bundle present in
// the mirror implies its iscc_index projection was written". A database written by
// the earlier record-then-project order — or one whose bundles were mirrored before
// projection support existed — can hold a full bundle with no index rows, and a
// presence-only skip would strip it of its projection forever. Requiring the
// projection means such a bundle is simply re-fetched and re-folded once, then
// skipped like any other.
//
// The check is an EXISTS on the bundle's FIRST leaf seq (bundle_index * TileWidth),
// a primary-key seek on iscc_index (hub_id, seq). It is exact for anything the
// current write order produces (projections land before the BLOB, so a mirrored full
// bundle has either all its rows or none) and catches the dominant legacy case of a
// bundle with no projections at all; a legacy bundle whose fold died PART-way through
// (RecordProjections is a per-row loop, not one transaction) still reads as projected.
func (s *Store) MirroredFullEntryBundles(ctx context.Context, hubID int64) (map[uint64]struct{}, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT b.bundle_index FROM entry_bundles b WHERE b.hub_id = ? AND b.width = ? "+
			"AND EXISTS (SELECT 1 FROM iscc_index i WHERE i.hub_id = b.hub_id "+
			"AND i.seq = b.bundle_index * ?)",
		hubID, widthForP(0), tiles.TileWidth,
	)
	if err != nil {
		return nil, fmt.Errorf("store.MirroredFullEntryBundles: hub %d: query: %w", hubID, err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[uint64]struct{})
	for rows.Next() {
		var index int64
		if err := rows.Scan(&index); err != nil {
			return nil, fmt.Errorf("store.MirroredFullEntryBundles: hub %d: scan: %w", hubID, err)
		}
		out[uint64(index)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.MirroredFullEntryBundles: hub %d: rows: %w", hubID, err)
	}
	return out, nil
}

// LatestCheckpointRaw reads the raw signed-note bytes of the most-recently observed
// checkpoint for a hub — the highest tree_size row. The SQLiteFetcher's
// ReadCheckpoint needs this size-agnostic read (the existing CheckpointAt is
// size-keyed). A hub with no checkpoint row returns (nil, false, nil); absent is
// not an error.
func (s *Store) LatestCheckpointRaw(ctx context.Context, hubID int64) ([]byte, bool, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT raw FROM checkpoints WHERE hub_id = ? ORDER BY tree_size DESC LIMIT 1",
		hubID,
	).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, false, nil
	case err != nil:
		return nil, false, fmt.Errorf("store.LatestCheckpointRaw: hub %d: %w", hubID, err)
	}
	return raw, true, nil
}

// boolToInt maps a Go bool to the schema's INTEGER 0/1 boolean convention.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
