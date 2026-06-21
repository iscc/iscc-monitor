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
// with is_full = 0. Partials are re-fetched and overwritten in place every poll via
// the composite-PK upsert; a full tile is immutable but an idempotent re-write of
// identical bytes is harmless. sha256 is recorded for fsck cross-checks and
// updated_at from the caller-supplied observedAt (mirroring RecordHubKey, which
// takes a time rather than calling time.Now()).
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
