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
// schema.sql and applied idempotently on every Open, so opening an existing
// database is a no-op and the data survives a close/reopen.
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
// single-writer pragmas and the embedded core M1 schema, and returns a ready
// Store.
//
// The connection pool is capped at one open connection (SetMaxOpenConns(1)) so
// all access serializes through a single writer — reads serialize too for now,
// which is fine pre-serving and avoids SQLITE_BUSY flakes; a read-pool split can
// come when serving lands. Schema application is idempotent (every statement is
// CREATE … IF NOT EXISTS), so Open on an existing database is a no-op. The caller
// owns the returned Store and must Close it.
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
	return &Store{db: db}, nil
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
