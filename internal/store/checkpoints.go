// This file adds the typed insert/query methods the follower drives to persist
// one observed checkpoint verdict: register a hub (hubs), record an observed
// checkpoint (checkpoints), and read/advance the per-hub follow cursor
// (follow_state). These are plain methods on *Store using the single open
// connection (the pool is already capped at one writer in sqlite.go), so they
// run db.ExecContext / db.QueryRowContext directly and open no new connections.
//
// store stays a leaf: it depends only on database/sql + stdlib and deliberately
// does NOT import internal/logclient. The follower maps the logclient verdict
// (Status.String(), CheckpointInfo) into the plain store-owned structs below at
// the call site, so net/http-bearing deps never enter this package's closure.
//
// Time convention mirrors schema.sql: timestamps are INTEGER unix-seconds. A
// zero CheckpointRecord.ObservedAt is written as NULL (not 0) so "never observed"
// is distinguishable from "observed at the unix epoch".
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CheckpointRecord is one observed hub-signed checkpoint to persist into the
// checkpoints table. Status carries the follower's logclient verdict
// ("verified"/"unverified"/"unresolvable"/"rotated") for the caller's
// verified-only-advances decision; the checkpoints table has no status column,
// so it is not persisted here. Root is the RFC-6962 tree head and Raw the full
// signed-note bytes; consistent / root_rebuilt stay NULL until the
// consistency-check step fills them.
type CheckpointRecord struct {
	HubID      int64
	Status     string
	TreeSize   uint64
	Root       []byte
	Raw        []byte
	ObservedAt time.Time
}

// FollowState is the per-hub poll cursor and freeze flag read from follow_state.
// The zero value (LastSize 0, Frozen false, empty LastError) is what FollowState
// returns for a hub that has no row yet.
type FollowState struct {
	LastSize  uint64
	Frozen    bool
	LastError string
}

// UpsertHub inserts-or-gets the hubs row for a hub and returns its hub_id. It is
// idempotent on the domain: a re-register with the same domain returns the
// existing id without rewriting columns. hubs carries no UNIQUE on domain, so
// this selects first and inserts only when absent (origin / base_url are set on
// that first insert).
func (s *Store) UpsertHub(ctx context.Context, domain, origin, baseURL string) (int64, error) {
	var hubID int64
	err := s.db.QueryRowContext(ctx, "SELECT hub_id FROM hubs WHERE domain = ?", domain).Scan(&hubID)
	switch {
	case err == nil:
		return hubID, nil
	case !errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("store.UpsertHub: select %q: %w", domain, err)
	}
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO hubs (domain, origin, base_url) VALUES (?, ?, ?)",
		domain, origin, baseURL,
	)
	if err != nil {
		return 0, fmt.Errorf("store.UpsertHub: insert %q: %w", domain, err)
	}
	hubID, err = res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("store.UpsertHub: last insert id: %w", err)
	}
	return hubID, nil
}

// RecordCheckpoint inserts one observed checkpoint and returns its id. It dedupes
// on the UNIQUE(hub_id, tree_size, root) key: a re-observed (size, root) returns
// the existing id with inserted=false and a nil error, preserving the first
// sighting. consistent / root_rebuilt are left NULL for the consistency-check
// step; a zero ObservedAt is stored as NULL.
func (s *Store) RecordCheckpoint(ctx context.Context, c CheckpointRecord) (int64, bool, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO checkpoints (hub_id, tree_size, root, raw, observed_at) "+
			"VALUES (?, ?, ?, ?, ?) ON CONFLICT(hub_id, tree_size, root) DO NOTHING",
		c.HubID, int64(c.TreeSize), c.Root, c.Raw, unixOrNil(c.ObservedAt),
	)
	if err != nil {
		return 0, false, fmt.Errorf("store.RecordCheckpoint: insert: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return 0, false, fmt.Errorf("store.RecordCheckpoint: rows affected: %w", err)
	} else if n > 0 {
		id, err := res.LastInsertId()
		if err != nil {
			return 0, false, fmt.Errorf("store.RecordCheckpoint: last insert id: %w", err)
		}
		return id, true, nil
	}
	// Conflict: the row already exists; read its id back.
	var id int64
	err = s.db.QueryRowContext(ctx,
		"SELECT id FROM checkpoints WHERE hub_id = ? AND tree_size = ? AND root = ?",
		c.HubID, int64(c.TreeSize), c.Root,
	).Scan(&id)
	if err != nil {
		return 0, false, fmt.Errorf("store.RecordCheckpoint: read existing id: %w", err)
	}
	return id, false, nil
}

// FollowState reads the per-hub poll cursor and freeze flag. A hub with no
// follow_state row yet returns the zero FollowState{} and a nil error (not an
// error), so the follower can treat "never polled" as last_size 0 / not frozen.
func (s *Store) FollowState(ctx context.Context, hubID int64) (FollowState, error) {
	var (
		fs        FollowState
		lastSize  sql.NullInt64
		lastError sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT last_size, frozen, last_error FROM follow_state WHERE hub_id = ?", hubID,
	).Scan(&lastSize, &fs.Frozen, &lastError)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return FollowState{}, nil
	case err != nil:
		return FollowState{}, fmt.Errorf("store.FollowState: hub %d: %w", hubID, err)
	}
	if lastSize.Valid {
		fs.LastSize = uint64(lastSize.Int64)
	}
	fs.LastError = lastError.String
	return fs, nil
}

// AdvanceFollowState upserts the per-hub follow_state, setting last_size to the
// newly-accepted size. It deliberately omits frozen from the conflict update so a
// frozen hub stays frozen (ADR-0006, no auto-unfreeze); only the freeze path may
// set frozen, and nothing here clears it.
func (s *Store) AdvanceFollowState(ctx context.Context, hubID int64, lastSize uint64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO follow_state (hub_id, last_size) VALUES (?, ?) "+
			"ON CONFLICT(hub_id) DO UPDATE SET last_size = excluded.last_size",
		hubID, int64(lastSize),
	)
	if err != nil {
		return fmt.Errorf("store.AdvanceFollowState: hub %d: %w", hubID, err)
	}
	return nil
}

// unixOrNil maps a time.Time to the schema's INTEGER unix-seconds, writing a zero
// time as NULL so "never observed" stays distinct from the unix epoch.
func unixOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Unix()
}
