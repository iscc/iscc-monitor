// Tests for the per-network SQLite store. They drive Open/Close through a
// t.TempDir() database path and assert on observable database state — the table
// set, the journal-mode pragma, and row persistence across a close/reopen —
// never on Store internals. Raw db access is reached via a fresh database/sql
// handle in the restart test so persistence is proven against the file on disk,
// not a leftover in-process connection.
package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// coreTables are the nine core M1 tables the embedded schema must create. Kept
// here (not derived from the DDL) so the test independently pins the expected
// data model.
var coreTables = []string{
	"hubs",
	"hub_keys",
	"checkpoints",
	"violations",
	"tiles",
	"entry_bundles",
	"iscc_index",
	"follow_state",
	"ots",
}

// tableSet reads the table names from sqlite_master into a set.
func tableSet(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer func() { _ = rows.Close() }()
	got := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sqlite_master: %v", err)
	}
	return got
}

// TestStoreOpenCreatesCoreTables opens a fresh database and confirms every core
// M1 table is present.
func TestStoreOpenCreatesCoreTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	got := tableSet(t, s.db)
	for _, name := range coreTables {
		if !got[name] {
			t.Errorf("missing core table %q (have %v)", name, got)
		}
	}
}

// TestStoreOpenEnablesWAL confirms the single-writer journal mode is WAL after
// open (ADR-0005/0007, correctness rule 6).
func TestStoreOpenEnablesWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

// TestStoreRestartSurvival writes a sentinel hubs row, closes the store, reopens
// the same path, and confirms the row and all tables persist. A direct INSERT
// drives raw SQL because the store exposes no insert method yet; the point is to
// prove on-disk persistence, not a public API.
func TestStoreRestartSurvival(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_, err = s.db.Exec(
		"INSERT INTO hubs (hub_id, domain, origin, base_url) VALUES (?, ?, ?, ?)",
		1, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id",
	)
	if err != nil {
		t.Fatalf("insert sentinel hub: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s2.Close() }()

	got := tableSet(t, s2.db)
	for _, name := range coreTables {
		if !got[name] {
			t.Errorf("table %q missing after reopen", name)
		}
	}

	var domain string
	err = s2.db.QueryRow("SELECT domain FROM hubs WHERE hub_id = 1").Scan(&domain)
	if err != nil {
		t.Fatalf("read sentinel hub after reopen: %v", err)
	}
	if domain != "sb0.iscc.id" {
		t.Errorf("sentinel domain = %q, want %q", domain, "sb0.iscc.id")
	}
}

// TestStoreReopenIdempotent confirms opening the same path twice (sequentially,
// after Close) is a no-op: schema re-apply must not error.
func TestStoreReopenIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open must be a no-op, got: %v", err)
	}
	if err := s2.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
