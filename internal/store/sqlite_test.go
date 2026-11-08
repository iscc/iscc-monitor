// Tests for the per-network SQLite store. They drive Open/Close through a
// t.TempDir() database path and assert on observable database state — the table
// set, the journal-mode pragma, and row persistence across a close/reopen —
// never on Store internals. Raw db access is reached via a fresh database/sql
// handle in the restart test so persistence is proven against the file on disk,
// not a leftover in-process connection.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
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

// TestStorePing confirms Ping returns nil on a freshly opened store — the
// readiness signal the /healthz handler consults.
func TestStorePing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	if err := s.Ping(context.Background()); err != nil {
		t.Errorf("Ping on a freshly opened store = %v, want nil", err)
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

// userVersion reads PRAGMA user_version off a raw handle.
func userVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	return v
}

// rawOpen opens path with the single-writer pragmas through a raw database/sql
// handle, mirroring how the runner reaches the database. Used to drive
// applyMigrations directly with a synthetic migration slice and to read state
// back off disk independent of any in-process Store connection.
func rawOpen(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestMigrationFreshDBAtCurrentVersion confirms a freshly created database is left
// at the code's current schema version (len(migrations)) after Open — the runner
// advances user_version to the baseline even when nothing needs migrating.
func TestMigrationFreshDBAtCurrentVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	db := rawOpen(t, path)
	if got := userVersion(t, db); got != len(migrations) {
		t.Errorf("fresh-DB user_version = %d, want len(migrations) = %d", got, len(migrations))
	}
}

// TestMigrationUpgradesOldDB feeds the runner a synthetic one-entry migration
// slice against a version-0 database and confirms it runs the migration exactly
// once, advances user_version to 1, and the migration's observable effect (a
// marker row) is present afterward.
func TestMigrationUpgradesOldDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db := rawOpen(t, path)
	// Simulate a database created before this migration existed: reset the
	// stored version below len(migs) so the synthetic step is pending.
	if _, err := db.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatalf("reset user_version: %v", err)
	}

	calls := 0
	migs := []func(*sql.Tx) error{
		func(tx *sql.Tx) error {
			calls++
			_, err := tx.Exec("CREATE TABLE migration_marker (id INTEGER PRIMARY KEY)")
			return err
		},
	}
	if err := applyMigrations(db, migs); err != nil {
		t.Fatalf("applyMigrations: %v", err)
	}

	if calls != 1 {
		t.Errorf("migration ran %d times, want 1", calls)
	}
	if got := userVersion(t, db); got != 1 {
		t.Errorf("user_version after upgrade = %d, want 1", got)
	}
	if !tableSet(t, db)["migration_marker"] {
		t.Error("migration effect missing: marker table not created")
	}
}

// TestMigrationIdempotent confirms re-running the runner over an already-migrated
// database executes zero migrations: user_version already equals len(migs), so the
// loop body never fires. This is the load-bearing idempotency contract — deleting
// the user_version bump makes this test fail (the migration re-runs).
func TestMigrationIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db := rawOpen(t, path)
	if _, err := db.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatalf("reset user_version: %v", err)
	}

	calls := 0
	migs := []func(*sql.Tx) error{
		func(tx *sql.Tx) error {
			calls++
			_, err := tx.Exec("CREATE TABLE migration_marker (id INTEGER PRIMARY KEY)")
			return err
		},
	}
	if err := applyMigrations(db, migs); err != nil {
		t.Fatalf("first applyMigrations: %v", err)
	}
	if err := applyMigrations(db, migs); err != nil {
		t.Fatalf("second applyMigrations: %v", err)
	}

	if calls != 1 {
		t.Errorf("migration ran %d times across two runs, want 1", calls)
	}
	if got := userVersion(t, db); got != 1 {
		t.Errorf("user_version after second run = %d, want 1", got)
	}
}

// TestMigrationFailClosed confirms a migration that returns an error leaves
// user_version unadvanced and the database unchanged, and the runner returns the
// wrapped error. The per-step transaction rolls back, so a failed step never
// leaves a half-migrated database.
func TestMigrationFailClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db := rawOpen(t, path)
	if _, err := db.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatalf("reset user_version: %v", err)
	}

	boom := errors.New("boom")
	migs := []func(*sql.Tx) error{
		func(tx *sql.Tx) error {
			// Write a row, then fail: the rollback must discard it.
			if _, err := tx.Exec("CREATE TABLE migration_marker (id INTEGER PRIMARY KEY)"); err != nil {
				return err
			}
			return boom
		},
	}
	err = applyMigrations(db, migs)
	if err == nil {
		t.Fatal("applyMigrations on a failing migration returned nil, want error")
	}
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap %v", err, boom)
	}
	if got := userVersion(t, db); got != 0 {
		t.Errorf("user_version after failed migration = %d, want 0 (unadvanced)", got)
	}
	if tableSet(t, db)["migration_marker"] {
		t.Error("failed migration was not rolled back: marker table present")
	}
}

// oldISCCIndexDDL is the original single-global-PK iscc_index shape (seq INTEGER
// PRIMARY KEY), pinned here so the migration test can recreate a pre-existing
// database the production migration 0 must upgrade in place.
const oldISCCIndexDDL = `CREATE TABLE iscc_index (
    hub_id         INTEGER NOT NULL REFERENCES hubs(hub_id),
    seq            INTEGER PRIMARY KEY,
    iscc_id        BLOB,
    iscc_id_str    TEXT,
    note_schema    TEXT,
    note_timestamp TEXT,
    record_sha256  BLOB
)`

// pkColumns reads the PRIMARY KEY column set of a table from PRAGMA table_info, so a
// test can assert the migration re-keyed iscc_index onto the composite (hub_id, seq)
// rather than trusting a row-collision side effect alone.
func pkColumns(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query("SELECT name, pk FROM pragma_table_info(?) WHERE pk > 0 ORDER BY pk", table)
	if err != nil {
		t.Fatalf("pragma_table_info(%s): %v", table, err)
	}
	defer func() { _ = rows.Close() }()
	var cols []string
	for rows.Next() {
		var name string
		var pk int
		if err := rows.Scan(&name, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}
	return cols
}

// TestMigrationIsccIndexCompositePK runs the PRODUCTION migrations slice against a
// pre-existing single-global-PK iscc_index database (a seeded row at user_version 0)
// and confirms migration 0 re-keys it onto the composite (hub_id, seq) PRIMARY KEY:
// the seeded row survives, user_version reaches len(migrations), and a second hub can
// then index a seq matching the first hub's seq without a PK collision.
func TestMigrationIsccIndexCompositePK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db := rawOpen(t, path)
	// Recreate the pre-migration world: drop the composite-PK table Open just created
	// and put back the original single-PK shape with a seeded row, at user_version 0.
	for _, stmt := range []string{
		"DROP TABLE iscc_index",
		oldISCCIndexDDL,
		"INSERT INTO hubs (hub_id, domain, origin, base_url) VALUES (1, 'sb0.iscc.id', 'sb0.iscc.id/log', 'https://sb0.iscc.id')",
		"INSERT INTO hubs (hub_id, domain, origin, base_url) VALUES (2, 'sb1.amlet.id', 'sb1.amlet.id/log', 'https://sb1.amlet.id')",
		"INSERT INTO iscc_index (hub_id, seq, iscc_id, iscc_id_str, note_schema) VALUES (1, 0, X'41', 'ISCC:A', 'iscc-note-0.8.0.json')",
		"PRAGMA user_version = 0",
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed pre-migration DB (%q): %v", stmt, err)
		}
	}

	// Run the PRODUCTION migration slice, not a synthetic one.
	if err := applyMigrations(db, migrations); err != nil {
		t.Fatalf("applyMigrations: %v", err)
	}

	// The version advanced to the code's current schema version.
	if got := userVersion(t, db); got != len(migrations) {
		t.Errorf("user_version after migration = %d, want len(migrations) = %d", got, len(migrations))
	}
	// The table is now keyed on the composite (hub_id, seq).
	if got := pkColumns(t, db, "iscc_index"); !reflect.DeepEqual(got, []string{"hub_id", "seq"}) {
		t.Errorf("iscc_index PRIMARY KEY columns = %v, want [hub_id seq]", got)
	}
	// The seeded row survived the rebuild.
	var idStr string
	if err := db.QueryRow("SELECT iscc_id_str FROM iscc_index WHERE hub_id = 1 AND seq = 0").Scan(&idStr); err != nil {
		t.Fatalf("read seeded row after migration: %v", err)
	}
	if idStr != "ISCC:A" {
		t.Errorf("seeded iscc_id_str = %q, want ISCC:A (row must survive the rebuild)", idStr)
	}
	// A second hub can now index a seq matching the first hub's seq — the very
	// collision the single global PK forbade.
	if _, err := db.Exec(
		"INSERT INTO iscc_index (hub_id, seq, iscc_id, iscc_id_str, note_schema) VALUES (2, 0, X'42', 'ISCC:B', 'iscc-note-0.8.0.json')",
	); err != nil {
		t.Fatalf("insert second hub's seq 0 after migration: %v", err)
	}
	if n := countRows2(t, db, "iscc_index"); n != 2 {
		t.Errorf("iscc_index row count after second-hub insert = %d, want 2 (no PK collision)", n)
	}
}

// countRows2 reads a single COUNT(*) from the named table off a raw handle (the
// store-level countRows helper takes a *Store, which the raw-handle migration tests
// do not hold).
func countRows2(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestMigrationOutOfRangeVersion confirms the runner rejects an out-of-range stored
// user_version fail-closed instead of opening silently or panicking: a version above
// len(migs) (a database from a NEWER binary) and a negative version (corruption) both
// return the errUnsupportedSchemaVersion-wrapped error, and migs[-1] is never
// indexed. Reverting the guard makes the future-version case open silently and the
// negative case panic.
func TestMigrationOutOfRangeVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db := rawOpen(t, path)
	migs := []func(*sql.Tx) error{
		func(tx *sql.Tx) error { return nil },
	}

	// A future version (> len(migs)) must be rejected, not silently accepted. PRAGMA
	// user_version takes no bound parameter, so the value is formatted in (an in-test
	// int, never user input).
	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", len(migs)+1)); err != nil {
		t.Fatalf("set future user_version: %v", err)
	}
	err = applyMigrations(db, migs)
	if err == nil {
		t.Fatal("applyMigrations on a future user_version returned nil, want error")
	}
	if !errors.Is(err, errUnsupportedSchemaVersion) {
		t.Errorf("future-version error = %v, want it to wrap errUnsupportedSchemaVersion", err)
	}

	// A negative version (corruption) must return an error, not panic on migs[-1].
	if _, err := db.Exec("PRAGMA user_version = -1"); err != nil {
		t.Fatalf("set negative user_version: %v", err)
	}
	err = applyMigrations(db, migs)
	if err == nil {
		t.Fatal("applyMigrations on a negative user_version returned nil, want error")
	}
	if !errors.Is(err, errUnsupportedSchemaVersion) {
		t.Errorf("negative-version error = %v, want it to wrap errUnsupportedSchemaVersion", err)
	}
}

// TestMigrationAppliesInOrder confirms a multi-entry slice over a version-0
// database runs every step in order and advances user_version to len(migs).
func TestMigrationAppliesInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db := rawOpen(t, path)
	if _, err := db.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatalf("reset user_version: %v", err)
	}

	var order []int
	migs := []func(*sql.Tx) error{
		func(tx *sql.Tx) error { order = append(order, 0); return nil },
		func(tx *sql.Tx) error { order = append(order, 1); return nil },
		func(tx *sql.Tx) error { order = append(order, 2); return nil },
	}
	if err := applyMigrations(db, migs); err != nil {
		t.Fatalf("applyMigrations: %v", err)
	}

	if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Errorf("migrations ran in order %v, want [0 1 2]", order)
	}
	if got := userVersion(t, db); got != 3 {
		t.Errorf("user_version after three migrations = %d, want 3", got)
	}
}
