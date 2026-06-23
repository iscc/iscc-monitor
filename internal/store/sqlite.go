// Package store opens and owns one network's SQLite database — the stateful
// foundation every later M1 unit (follower, mirror, index, OTS) persists into.
//
// One file per network (mainnet.db / testnet.db) per ADR-0007, so no table
// carries a `network` column. The connection is opened with the ADR-0005 /
// ADR-0007 single-writer discipline: WAL journaling plus a busy_timeout, and the
// connection pool is capped at one open connection so every write serializes
// through a single writer (the goroutine-ownership wrapper that enforces "one
// goroutine owns all writes" lands with the follower; this layer just guarantees
// the pool can never open a second writer). The core M1 schema is embedded from
// schema.sql and applied idempotently on every Open, then a PRAGMA user_version-
// gated migration runner applies any append-only deltas the baseline DDL cannot
// reach (a column added to an existing table). So opening an existing database is
// a no-op once it is already at the code's schema version, and the data survives a
// close/reopen.
//
// Time convention: timestamps are stored as INTEGER unix-seconds (see
// schema.sql). The pure-Go modernc.org/sqlite driver is used because the project
// builds with CGO_ENABLED=0, which forbids a cgo SQLite driver.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" database/sql driver
)

// schemaSQL is the embedded core M1 DDL applied on every Open.
//
//go:embed schema.sql
var schemaSQL string

// migrations is the ordered, append-only list of on-disk schema deltas the
// baseline CREATE TABLE IF NOT EXISTS pass (schemaSQL) cannot reach — chiefly a
// column added to an existing table on a pre-existing database. Index i is the
// step that lifts PRAGMA user_version from i to i+1, so len(migrations) is the
// code's current schema version. The list is run after schemaSQL on every Open by
// applyMigrations; an already-migrated database runs zero steps.
//
// It is an empty no-op baseline today: a fresh database created by schemaSQL is
// already at version 0 = len(migrations), so Open advances nothing. The first
// planned entry is the iscc_index single-global-PK → composite (hub_id, seq)
// rebuild (the multi-hub leaf-index collision), landing in the next step; append
// migrations here, never edit or reorder an already-released entry.
var migrations = []func(*sql.Tx) error{}

// pragmas configure the single-writer discipline (ADR-0005/0007, correctness
// rule 6) and are executed in order immediately after open: WAL journaling so
// readers never block the writer, a 5s busy_timeout to ride out lock contention
// instead of failing with SQLITE_BUSY, enforced foreign keys, and synchronous=
// NORMAL which is safe and fast under WAL.
var pragmas = []string{
	"PRAGMA journal_mode=WAL",
	"PRAGMA busy_timeout=5000",
	"PRAGMA foreign_keys=ON",
	"PRAGMA synchronous=NORMAL",
}

// Store is an open handle to one network's SQLite database.
type Store struct {
	db *sql.DB
}

// Open opens (creating if absent) the SQLite database at path, applies the
// single-writer pragmas and the embedded core M1 schema, runs the
// PRAGMA user_version-gated migration list, and returns a ready Store.
//
// The connection pool is capped at one open connection (SetMaxOpenConns(1)) so
// all access serializes through a single writer — reads serialize too for now,
// which is fine pre-serving and avoids SQLITE_BUSY flakes; a read-pool split can
// come when serving lands. Schema application is idempotent (every statement is
// CREATE … IF NOT EXISTS); the version-gated runner that follows applies only the
// not-yet-applied append-only deltas and advances user_version, so Open on a
// database already at the code's schema version is a no-op (zero migrations run).
// The caller owns the returned Store and must Close it.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store.Open: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store.Open: %q: %w", p, err)
		}
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store.Open: apply schema: %w", err)
	}
	if err := applyMigrations(db, migrations); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store.Open: migrate: %w", err)
	}
	return &Store{db: db}, nil
}

// applyMigrations runs the version-gated migration list on db after the baseline
// schema pass. It reads PRAGMA user_version and, for every version v from the
// stored value up to len(migs), runs migs[v] inside its own transaction and bumps
// user_version to v+1 on success. So a database already at len(migs) runs nothing
// (the idempotency contract) and an out-of-date one applies exactly the missing
// steps in order. It takes the slice explicitly so it is unit-testable with a
// synthetic list independent of the production migrations.
//
// It is fail-closed: a migration error rolls back that step's transaction
// (leaving user_version unadvanced, so a re-run retries from the same point) and
// is %w-wrapped and returned, aborting Open — a half-migrated database is never
// returned. The per-step transaction follows the AdvanceAccepted idiom; the
// deferred Rollback after a successful Commit returns benign sql.ErrTxDone.
//
// PRAGMA user_version does not accept a bound parameter in SQLite, so the bump is
// built with fmt.Sprintf from an in-code int (v+1, never user input) — no
// injection surface.
func applyMigrations(db *sql.DB, migs []func(*sql.Tx) error) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	for v := version; v < len(migs); v++ {
		if err := applyMigration(db, migs[v], v); err != nil {
			return fmt.Errorf("migration %d: %w", v, err)
		}
	}
	return nil
}

// applyMigration runs one migration step inside its own transaction and bumps
// PRAGMA user_version from v to v+1 atomically on success. A failure rolls the
// transaction back (user_version stays v) and is %w-wrapped; the deferred
// Rollback after a successful Commit returns benign sql.ErrTxDone.
func applyMigration(db *sql.DB, mig func(*sql.Tx) error, v int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := mig(tx); err != nil {
		return fmt.Errorf("apply: %w", err)
	}
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", v+1)); err != nil {
		return fmt.Errorf("set user_version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Ping verifies the underlying SQLite connection is reachable by delegating to
// database/sql PingContext. It is the store-readiness probe the /healthz handler
// consults; with SetMaxOpenConns(1) the ping serializes on the single connection
// like every other read, which is fine for a liveness check. A failure is
// %w-wrapped so the caller can inspect the driver error.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("store.Ping: %w", err)
	}
	return nil
}

// Close releases the database handle. It is safe to call once; further use of
// the Store after Close is undefined.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("store.Close: %w", err)
	}
	return nil
}
