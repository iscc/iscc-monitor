// This file adds the typed OTS-table CRUD the (later) OpenTimestamps stamp loop,
// background upgrade loop, and certificate §5 / `.ots` route read and write
// through: record a stamped root once per distinct (hub, tree_size, root)
// (RecordOTS), read one root's anchor state back (OTSForRoot), list the pending
// roots the upgrade loop must poll (PendingOTS), and flip a pending row to
// Bitcoin-confirmed (MarkOTSUpgraded). It is the foundational, fully
// store-testable seam every later OTS sub-step reads/writes through.
//
// This step does NOT depend on the OpenTimestamps library: ots_bytes is stored
// and returned verbatim as an opaque []byte and status is an opaque string, so
// store stays a leaf (only database/sql + stdlib) and the crypto/proof path is
// untouched. The two status consts below are the single source of truth for the
// "pending" / "confirmed" literals so the writer, the PendingOTS filter, and
// MarkOTSUpgraded never drift apart (the literal-drift trap).
//
// Time convention mirrors schema.sql / checkpoints.go: timestamps are INTEGER
// unix-seconds, a zero time is written as NULL via the shared unixOrNil helper,
// and the nullable read columns degrade a NULL back to the zero value.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// OTSStatusPending is the status of a stamped root whose OpenTimestamps proof is
// not yet Bitcoin-confirmed — the rows the upgrade loop must re-poll. Carried as
// an opaque string on OTSRecord and shared by RecordOTS, PendingOTS, and
// MarkOTSUpgraded so the literal lives in exactly one place.
const OTSStatusPending = "pending"

// OTSStatusConfirmed is the status of a stamped root whose OpenTimestamps proof
// has been upgraded to a Bitcoin-confirmed attestation. MarkOTSUpgraded sets it;
// PendingOTS excludes it.
const OTSStatusConfirmed = "confirmed"

// OTSRecord is one OpenTimestamps anchor row for a distinct observed root,
// mirroring CheckpointRecord's plain-leaf shape. Status is an opaque string
// (pending → confirmed) carried on the struct so store stays import-free of any
// anchoring package; OTSBytes is the serialized proof stored/returned verbatim
// as an opaque BLOB. The nullable time / count columns (StampedAt, UpgradedAt,
// BTCHeight, Attempts, NextRetry) follow the schema's unix-seconds + zero-as-NULL
// convention. The row dedupes on UNIQUE(hub_id, tree_size, root).
type OTSRecord struct {
	HubID        int64
	TreeSize     uint64
	Root         []byte
	Status       string
	OTSBytes     []byte
	CalendarURLs string
	StampedAt    time.Time
	UpgradedAt   time.Time
	BTCHeight    int64
	Attempts     int64
	NextRetry    time.Time
}

// RecordOTS inserts one stamped root and returns its id. It dedupes on the
// UNIQUE(hub_id, tree_size, root) key so each distinct observed root is stamped
// once (the milestone's "stamp each distinct root daily" criterion): a re-stamp of
// the same (hub, size, root) returns the existing id with inserted=false and a nil
// error, preserving the first stamping. This ports RecordCheckpoint's
// ON CONFLICT … DO NOTHING + RowsAffected first-sighting dance verbatim. TreeSize
// is cast uint64→int64, zero times are written as NULL via unixOrNil, and an empty
// CalendarURLs as NULL via nullStringOrNil.
func (s *Store) RecordOTS(ctx context.Context, r OTSRecord) (int64, bool, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO ots (hub_id, tree_size, root, status, ots_bytes, calendar_urls, "+
			"stamped_at, upgraded_at, btc_height, attempts, next_retry) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "+
			"ON CONFLICT(hub_id, tree_size, root) DO NOTHING",
		r.HubID, int64(r.TreeSize), r.Root, r.Status, r.OTSBytes, nullStringOrNil(r.CalendarURLs),
		unixOrNil(r.StampedAt), unixOrNil(r.UpgradedAt), r.BTCHeight, r.Attempts, unixOrNil(r.NextRetry),
	)
	if err != nil {
		return 0, false, fmt.Errorf("store.RecordOTS: insert: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return 0, false, fmt.Errorf("store.RecordOTS: rows affected: %w", err)
	} else if n > 0 {
		id, err := res.LastInsertId()
		if err != nil {
			return 0, false, fmt.Errorf("store.RecordOTS: last insert id: %w", err)
		}
		return id, true, nil
	}
	// Conflict: the row already exists; read its id back.
	var id int64
	err = s.db.QueryRowContext(ctx,
		"SELECT id FROM ots WHERE hub_id = ? AND tree_size = ? AND root = ?",
		r.HubID, int64(r.TreeSize), r.Root,
	).Scan(&id)
	if err != nil {
		return 0, false, fmt.Errorf("store.RecordOTS: read existing id: %w", err)
	}
	return id, false, nil
}

// OTSForRoot reads the anchor state for one (hub, tree_size, root) back. An
// un-anchored root is a plain miss — sql.ErrNoRows returns (OTSRecord{}, false,
// nil), not an error — so certificate §5 / the `.ots` route can render the honest
// "pending / not-yet-anchored" state rather than a 5xx (the CheckpointAt
// absent-is-not-an-error convention). The nullable columns are read through
// sql.NullInt64 / sql.NullString (the inverse of the write), so a NULL degrades to
// the zero value; HubID / TreeSize / Root come from the in-args (the lookup key),
// so the returned struct is fully populated.
func (s *Store) OTSForRoot(ctx context.Context, hubID int64, treeSize uint64, root []byte) (OTSRecord, bool, error) {
	var (
		status     sql.NullString
		otsBytes   []byte
		calendars  sql.NullString
		stampedAt  sql.NullInt64
		upgradedAt sql.NullInt64
		btcHeight  sql.NullInt64
		attempts   sql.NullInt64
		nextRetry  sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT status, ots_bytes, calendar_urls, stamped_at, upgraded_at, btc_height, attempts, next_retry "+
			"FROM ots WHERE hub_id = ? AND tree_size = ? AND root = ?",
		hubID, int64(treeSize), root,
	).Scan(&status, &otsBytes, &calendars, &stampedAt, &upgradedAt, &btcHeight, &attempts, &nextRetry)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return OTSRecord{}, false, nil
	case err != nil:
		return OTSRecord{}, false, fmt.Errorf("store.OTSForRoot: hub %d size %d: %w", hubID, treeSize, err)
	}
	r := OTSRecord{
		HubID:        hubID,
		TreeSize:     treeSize,
		Root:         root,
		Status:       status.String,
		OTSBytes:     otsBytes,
		CalendarURLs: calendars.String,
		BTCHeight:    btcHeight.Int64,
		Attempts:     attempts.Int64,
	}
	if stampedAt.Valid {
		r.StampedAt = time.Unix(stampedAt.Int64, 0)
	}
	if upgradedAt.Valid {
		r.UpgradedAt = time.Unix(upgradedAt.Int64, 0)
	}
	if nextRetry.Valid {
		r.NextRetry = time.Unix(nextRetry.Int64, 0)
	}
	return r, true, nil
}

// PendingOTS reads the still-pending stamped roots whose back-off has elapsed at
// now — the rows the upgrade loop must re-poll this pass — oldest-first (ORDER BY
// stamped_at ASC, id ASC) for a fair upgrade order. It is a leaf []OTSRecord read
// scoped by status = OTSStatusPending (a confirmed row drops out) AND a back-off
// filter `next_retry IS NULL OR next_retry <= now`: a freshly-stamped row has a
// NULL next_retry so it is immediately due, while a row a MarkOTSAttempted back-off
// pushed into the future is excluded until now reaches its next_retry. A network
// with no due rows returns an empty slice and a nil error. The nullable columns are
// read through sql.NullInt64 / sql.NullString exactly as OTSForRoot, so a NULL
// degrades to the zero value. now is bound directly via now.Unix() (NOT unixOrNil —
// now is never the zero time on the upgrade path).
func (s *Store) PendingOTS(ctx context.Context, now time.Time) ([]OTSRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT hub_id, tree_size, root, status, ots_bytes, calendar_urls, "+
			"stamped_at, upgraded_at, btc_height, attempts, next_retry "+
			"FROM ots WHERE status = ? AND (next_retry IS NULL OR next_retry <= ?) "+
			"ORDER BY stamped_at ASC, id ASC",
		OTSStatusPending, now.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("store.PendingOTS: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var pending []OTSRecord
	for rows.Next() {
		var (
			r          OTSRecord
			treeSize   int64
			status     sql.NullString
			calendars  sql.NullString
			stampedAt  sql.NullInt64
			upgradedAt sql.NullInt64
			btcHeight  sql.NullInt64
			attempts   sql.NullInt64
			nextRetry  sql.NullInt64
		)
		if err := rows.Scan(
			&r.HubID, &treeSize, &r.Root, &status, &r.OTSBytes, &calendars,
			&stampedAt, &upgradedAt, &btcHeight, &attempts, &nextRetry,
		); err != nil {
			return nil, fmt.Errorf("store.PendingOTS: scan: %w", err)
		}
		r.TreeSize = uint64(treeSize)
		r.Status = status.String
		r.CalendarURLs = calendars.String
		r.BTCHeight = btcHeight.Int64
		r.Attempts = attempts.Int64
		if stampedAt.Valid {
			r.StampedAt = time.Unix(stampedAt.Int64, 0)
		}
		if upgradedAt.Valid {
			r.UpgradedAt = time.Unix(upgradedAt.Int64, 0)
		}
		if nextRetry.Valid {
			r.NextRetry = time.Unix(nextRetry.Int64, 0)
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.PendingOTS: rows: %w", err)
	}
	return pending, nil
}

// MarkOTSUpgraded flips one stamped root to OTSStatusConfirmed and records the
// upgraded OpenTimestamps proof bytes, the Bitcoin block height, and the upgrade
// instant. It is a plain UPDATE keyed on (hub_id, tree_size, root); like
// SetCoverage it ignores RowsAffected so an idempotent re-mark of an
// already-confirmed (or absent) row is not an error. upgradedAt is written as NULL
// when zero via unixOrNil.
func (s *Store) MarkOTSUpgraded(ctx context.Context, hubID int64, treeSize uint64, root []byte, otsBytes []byte, btcHeight int64, upgradedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE ots SET status = ?, ots_bytes = ?, btc_height = ?, upgraded_at = ? "+
			"WHERE hub_id = ? AND tree_size = ? AND root = ?",
		OTSStatusConfirmed, otsBytes, btcHeight, unixOrNil(upgradedAt),
		hubID, int64(treeSize), root,
	)
	if err != nil {
		return fmt.Errorf("store.MarkOTSUpgraded: hub %d size %d: %w", hubID, treeSize, err)
	}
	return nil
}

// MarkOTSAttempted records a backed-off retry for a still-pending stamped root:
// the upgrade loop sets attempts to the new count and next_retry to the future
// instant before which PendingOTS must not re-surface the row. It does NOT touch
// status — the row stays OTSStatusPending so PendingOTS re-serves it once next_retry
// elapses. It is a plain UPDATE keyed on (hub_id, tree_size, root) and, like
// SetCoverage / MarkOTSUpgraded, ignores RowsAffected so an absent row is a silent
// no-op rather than an error. nextRetry is written as NULL when zero via unixOrNil.
func (s *Store) MarkOTSAttempted(ctx context.Context, hubID int64, treeSize uint64, root []byte, attempts int64, nextRetry time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE ots SET attempts = ?, next_retry = ? "+
			"WHERE hub_id = ? AND tree_size = ? AND root = ?",
		attempts, unixOrNil(nextRetry),
		hubID, int64(treeSize), root,
	)
	if err != nil {
		return fmt.Errorf("store.MarkOTSAttempted: hub %d size %d: %w", hubID, treeSize, err)
	}
	return nil
}
