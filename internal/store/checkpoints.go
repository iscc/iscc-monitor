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

// CoverageInfo is the per-hub coverage start read from
// hubs.monitored_since_{size,time} (ADR-0001). Set reports whether coverage has
// been recorded yet: a hub that has never yielded a verified observation returns
// Set false with zero Size / Since, and guarantees hold only from the coverage
// start onward.
type CoverageInfo struct {
	Size  uint64
	Since time.Time
	Set   bool
}

// HubKey is one did:web-resolved hub signing key to cache into hub_keys
// (ADR-0009). The DID document is the source of truth; this row is a refreshed
// cache, never an independent key source. KeyID is the BE-uint32 signed-note
// keyhash that identifies the key (the follower derives it from
// ResolveVerifierKey's verifier-string middle field "+<hex>+" or keyID(name,
// pub); this method just persists what it is given). PubkeyRaw is the 32-byte
// Ed25519 key, PubkeyZ its z6Mk… multibase form (empty → SQL NULL). Revoked
// carries the DID doc's revocation instant (zero → not revoked, NULL) and
// ResolvedAt the time of resolution (zero → NULL).
type HubKey struct {
	HubID      int64
	KeyID      uint32
	PubkeyRaw  []byte
	PubkeyZ    string
	Revoked    time.Time
	ResolvedAt time.Time
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

// CheckpointAt reads the persisted (root, raw) bytes for a hub's checkpoint at a
// given tree_size. follow_state deliberately does not persist the accepted root,
// so the follower reads it back here to drive the fork check (prevRoot) and to
// supply the prior raw bytes as violation evidence (RawA).
//
// An absent (hubID, treeSize) returns found=false with a nil error (not an
// error), mirroring FollowState's "absent row is not an error" convention. store
// stays a leaf: it returns []byte, never a logclient type — the follower copies
// the root into a fixed-size array at the call site. The query is ORDER BY rowid
// LIMIT 1, so a hub that recorded two different roots at one size (a fork's
// evidence) returns the row recorded first — the prior accepted root, which has
// the lowest rowid — deterministically, never the later contradicting-evidence
// row; both this prior-root read and the follower's fork re-detection stay
// deterministic.
func (s *Store) CheckpointAt(ctx context.Context, hubID int64, treeSize uint64) (root []byte, raw []byte, found bool, err error) {
	err = s.db.QueryRowContext(ctx,
		"SELECT root, raw FROM checkpoints WHERE hub_id = ? AND tree_size = ? ORDER BY rowid LIMIT 1",
		hubID, int64(treeSize),
	).Scan(&root, &raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil, false, nil
	case err != nil:
		return nil, nil, false, fmt.Errorf("store.CheckpointAt: hub %d size %d: %w", hubID, treeSize, err)
	}
	return root, raw, true, nil
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

// AdvanceAccepted commits the whole "advance accepted state" invariant for one
// verified, non-violation observation as a single transaction so its ordering and
// partial-write atomicity live at the storage boundary (ADR-0005 single-writer
// locality), not in the follower. It performs, against one *sql.Tx, the three
// writes the follower used to sequence by hand: dedupe-insert the observed
// checkpoint (ON CONFLICT(hub_id, tree_size, root) DO NOTHING — an idempotent
// re-poll is a no-op), the set-once coverage UPDATE (guarded by
// monitored_since_size IS NULL so the start never moves, ADR-0001), and the
// follow-cursor upsert (omitting frozen so a frozen hub stays frozen, ADR-0006).
// On any error the transaction rolls back, leaving accepted state unchanged; the
// runs on the store's single capped connection and opens no second connection.
func (s *Store) AdvanceAccepted(ctx context.Context, c CheckpointRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store.AdvanceAccepted: begin: %w", err)
	}
	// After a successful Commit, Rollback returns sql.ErrTxDone, which is safe to
	// ignore; on any early return it rolls the transaction back.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO checkpoints (hub_id, tree_size, root, raw, observed_at) "+
			"VALUES (?, ?, ?, ?, ?) ON CONFLICT(hub_id, tree_size, root) DO NOTHING",
		c.HubID, int64(c.TreeSize), c.Root, c.Raw, unixOrNil(c.ObservedAt),
	); err != nil {
		return fmt.Errorf("store.AdvanceAccepted: record checkpoint: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE hubs SET monitored_since_size = ?, monitored_since_time = ? "+
			"WHERE hub_id = ? AND monitored_since_size IS NULL",
		int64(c.TreeSize), unixOrNil(c.ObservedAt), c.HubID,
	); err != nil {
		return fmt.Errorf("store.AdvanceAccepted: set coverage: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO follow_state (hub_id, last_size) VALUES (?, ?) "+
			"ON CONFLICT(hub_id) DO UPDATE SET last_size = excluded.last_size",
		c.HubID, int64(c.TreeSize),
	); err != nil {
		return fmt.Errorf("store.AdvanceAccepted: advance follow state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store.AdvanceAccepted: commit: %w", err)
	}
	return nil
}

// Violation is one self-consistency violation to persist permanently into the
// violations table (irreplaceable evidence, ADR-0006). Kind carries the trigger
// the consistency check supplies ("fork"/"shrink"/"equivocation") as a plain
// string, mirroring how Status rides on CheckpointRecord. RawA / RawB are the two
// contradictory hub-signed checkpoint bytes and ProofJSON the supporting
// consistency proof; a zero DetectedAt is written as NULL.
type Violation struct {
	HubID      int64
	Kind       string
	RawA       []byte
	RawB       []byte
	ProofJSON  string
	DetectedAt time.Time
}

// RecordViolation inserts one violation and returns its id. It is a plain INSERT
// with no ON CONFLICT: violations has no UNIQUE constraint because re-detecting a
// violation is itself evidence, so every detection is recorded. proof_json is a
// plain string (empty stays empty, not NULL); a zero DetectedAt is stored as NULL.
func (s *Store) RecordViolation(ctx context.Context, v Violation) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO violations (hub_id, kind, detected_at, raw_a, raw_b, proof_json) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		v.HubID, v.Kind, unixOrNil(v.DetectedAt), v.RawA, v.RawB, v.ProofJSON,
	)
	if err != nil {
		return 0, fmt.Errorf("store.RecordViolation: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("store.RecordViolation: last insert id: %w", err)
	}
	return id, nil
}

// ListViolations reads a hub's recorded self-consistency violations newest-first
// (ORDER BY detected_at DESC, id DESC) for the dossier's permanent-evidence
// Exhibit (ADR-0006). It is a pure leaf read returning plain []Violation: only
// hub_id, kind, and detected_at are read (the markup renders kind + detected_at;
// raw_a / raw_b / proof_json belong with the future proof-bundle surface and are
// left zero here). detected_at is read through sql.NullInt64 — the unixOrNil write
// inverse — so a NULL detected_at degrades to a zero time.Time. A hub with no
// violations returns an empty slice and a nil error (an absent row is not an
// error, mirroring the other reads).
func (s *Store) ListViolations(ctx context.Context, hubID int64) ([]Violation, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT hub_id, kind, detected_at FROM violations WHERE hub_id = ? "+
			"ORDER BY detected_at DESC, id DESC",
		hubID,
	)
	if err != nil {
		return nil, fmt.Errorf("store.ListViolations: hub %d: %w", hubID, err)
	}
	defer func() { _ = rows.Close() }()

	var violations []Violation
	for rows.Next() {
		var (
			v        Violation
			detected sql.NullInt64
		)
		if err := rows.Scan(&v.HubID, &v.Kind, &detected); err != nil {
			return nil, fmt.Errorf("store.ListViolations: scan: %w", err)
		}
		if detected.Valid {
			v.DetectedAt = time.Unix(detected.Int64, 0)
		}
		violations = append(violations, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListViolations: rows: %w", err)
	}
	return violations, nil
}

// Freeze sets frozen=1 on the hub's follow_state row, upserting so it works
// whether or not a row exists yet (a hub can be frozen before its first verified
// advance). It is the only writer of frozen; AdvanceFollowState deliberately omits
// frozen from its conflict update, so an advance after a freeze keeps frozen=1
// (ADR-0006, no auto-unfreeze).
func (s *Store) Freeze(ctx context.Context, hubID int64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO follow_state (hub_id, frozen) VALUES (?, 1) "+
			"ON CONFLICT(hub_id) DO UPDATE SET frozen = 1",
		hubID,
	)
	if err != nil {
		return fmt.Errorf("store.Freeze: hub %d: %w", hubID, err)
	}
	return nil
}

// SetCoverage records the hub's coverage start (monitored_since_{size,time}) the
// first time the hub yields a verified observation, and never moves it thereafter
// (ADR-0001, coverage honesty: the start is immutable). It is a guarded UPDATE —
// the monitored_since_size IS NULL clause means a re-call after the start is set is
// a silent no-op, so it does NOT rely on RowsAffected to signal success (zero rows
// affected once the start is set is the correct, non-error case). A zero observedAt
// is written as NULL via unixOrNil, mirroring RecordCheckpoint.
func (s *Store) SetCoverage(ctx context.Context, hubID int64, size uint64, observedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE hubs SET monitored_since_size = ?, monitored_since_time = ? "+
			"WHERE hub_id = ? AND monitored_since_size IS NULL",
		int64(size), unixOrNil(observedAt), hubID,
	)
	if err != nil {
		return fmt.Errorf("store.SetCoverage: hub %d: %w", hubID, err)
	}
	return nil
}

// Coverage reads the hub's coverage start back from
// hubs.monitored_since_{size,time}. A hub that has never started coverage (or an
// absent hub) returns CoverageInfo{} with Set false and a nil error, mirroring
// FollowState's "absent row is not an error" convention. monitored_since_time is
// read through sql.NullInt64 so an unset time degrades to a zero time.Time.
func (s *Store) Coverage(ctx context.Context, hubID int64) (CoverageInfo, error) {
	var (
		info      CoverageInfo
		sinceSize sql.NullInt64
		sinceTime sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT monitored_since_size, monitored_since_time FROM hubs WHERE hub_id = ?", hubID,
	).Scan(&sinceSize, &sinceTime)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return CoverageInfo{}, nil
	case err != nil:
		return CoverageInfo{}, fmt.Errorf("store.Coverage: hub %d: %w", hubID, err)
	}
	if !sinceSize.Valid {
		return CoverageInfo{}, nil
	}
	info.Set = true
	info.Size = uint64(sinceSize.Int64)
	if sinceTime.Valid {
		info.Since = time.Unix(sinceTime.Int64, 0)
	}
	return info, nil
}

// RecordHubKey caches a did:web-resolved hub key into hub_keys, deduped per
// (hub_id, key_id) and refreshed on every poll (ADR-0009: the DID document is the
// source of truth, so a re-resolve overwrites the cached row rather than
// accumulating). hub_keys carries no UNIQUE constraint, so this is a guarded
// UPDATE keyed on (hub_id, key_id) — mirroring SetCoverage — followed by an INSERT
// only when no row matched: the same key re-resolved refreshes pubkey/revoked/
// resolved_at in place (count stays 1), while a rotation to a different key_id
// inserts a second row so both the old and new keys stay cached (revocation is
// recorded via revoked_at, never by deleting the old row). An empty PubkeyZ writes
// NULL (distinct from ""), and a zero Revoked / ResolvedAt writes NULL via
// unixOrNil. The hub_id REFERENCES hubs(hub_id) FK is enforced, so a key for an
// unknown hub returns a non-nil error.
func (s *Store) RecordHubKey(ctx context.Context, k HubKey) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE hub_keys SET pubkey_raw = ?, pubkey_z = ?, revoked_at = ?, resolved_at = ? "+
			"WHERE hub_id = ? AND key_id = ?",
		k.PubkeyRaw, nullStringOrNil(k.PubkeyZ), unixOrNil(k.Revoked), unixOrNil(k.ResolvedAt),
		k.HubID, int64(k.KeyID),
	)
	if err != nil {
		return fmt.Errorf("store.RecordHubKey: update hub %d key %08x: %w", k.HubID, k.KeyID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store.RecordHubKey: rows affected: %w", err)
	}
	if n > 0 {
		return nil
	}
	// No existing row for (hub_id, key_id): insert the full row.
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO hub_keys (hub_id, key_id, pubkey_raw, pubkey_z, revoked_at, resolved_at) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		k.HubID, int64(k.KeyID), k.PubkeyRaw, nullStringOrNil(k.PubkeyZ),
		unixOrNil(k.Revoked), unixOrNil(k.ResolvedAt),
	)
	if err != nil {
		return fmt.Errorf("store.RecordHubKey: insert hub %d key %08x: %w", k.HubID, k.KeyID, err)
	}
	return nil
}

// LookupHubKey reads one cached did:web key back by (hub_id, key_id) — the read
// side of RecordHubKey. An absent (hub_id, key_id) returns (HubKey{}, false, nil)
// (not an error), mirroring FollowState / Coverage's "absent row is not an error"
// convention so callers can treat "not cached yet" as a plain miss; only a real
// query fault returns a non-nil error. The nullable columns are the exact inverse
// of RecordHubKey's nullStringOrNil / unixOrNil: a NULL pubkey_z reads back as ""
// and a NULL revoked_at / resolved_at as the zero time.Time, so a RecordHubKey →
// LookupHubKey round-trip is lossless for the empty/zero cases. HubID / KeyID come
// from the in-args (the lookup key), so the returned struct is fully populated.
// LIMIT 1 is defensive: hub_keys has no UNIQUE, but the write path keeps at most
// one row per (hub_id, key_id), so a match is single by construction.
func (s *Store) LookupHubKey(ctx context.Context, hubID int64, keyID uint32) (HubKey, bool, error) {
	var (
		pubkeyRaw  []byte
		pubkeyZ    sql.NullString
		revoked    sql.NullInt64
		resolvedAt sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT pubkey_raw, pubkey_z, revoked_at, resolved_at FROM hub_keys "+
			"WHERE hub_id = ? AND key_id = ? LIMIT 1",
		hubID, int64(keyID),
	).Scan(&pubkeyRaw, &pubkeyZ, &revoked, &resolvedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return HubKey{}, false, nil
	case err != nil:
		return HubKey{}, false, fmt.Errorf("store.LookupHubKey: hub %d key %08x: %w", hubID, keyID, err)
	}
	k := HubKey{
		HubID:     hubID,
		KeyID:     keyID,
		PubkeyRaw: pubkeyRaw,
		PubkeyZ:   pubkeyZ.String,
	}
	if revoked.Valid {
		k.Revoked = time.Unix(revoked.Int64, 0)
	}
	if resolvedAt.Valid {
		k.ResolvedAt = time.Unix(resolvedAt.Int64, 0)
	}
	return k, true, nil
}

// unixOrNil maps a time.Time to the schema's INTEGER unix-seconds, writing a zero
// time as NULL so "never observed" stays distinct from the unix epoch.
func unixOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Unix()
}

// nullStringOrNil maps a string to a nullable TEXT column, writing an empty string
// as NULL so "no value" (e.g. no multibase) stays distinct from the empty string.
func nullStringOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}
