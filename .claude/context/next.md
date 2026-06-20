# Next Work Package

## Step: Bootstrap the per-network SQLite store (`internal/store`): WAL + single-writer open + embedded schema

## Goal
Stand up the stateful foundation every later M1 unit needs: open a per-network SQLite database with
the ADR-0005/0007 single-writer discipline (WAL + `busy_timeout`), apply the core M1 schema idempotently
from an embedded `schema.sql`, and prove the DB + schema survive a close/reopen (restart). Nothing
consumes `AcceptCheckpoint` yet without somewhere to persist; this is that somewhere. It also wires the
pure-Go `modernc.org/sqlite` driver (`CGO_ENABLED=0`) for the first time.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/store/sqlite.go` — `Open(path string) (*Store, error)`, `Close()`,
    embedded-schema application, single-writer pragmas. (1 of ≤3 non-test/doc files.)
  - `/workspace/iscc-monitor/internal/store/schema.sql` — `//go:embed`-ed DDL for the core M1 tables
    (`CREATE TABLE IF NOT EXISTS …`). (Counts as a non-test/doc file: 2 of ≤3.)
  - `/workspace/iscc-monitor/internal/store/sqlite_test.go` — table/scenario tests (test file, not counted).
- **Modify**:
  - `/workspace/iscc-monitor/go.mod` + `/workspace/iscc-monitor/go.sum` — add `modernc.org/sqlite`
    (and its indirect deps). **Counts as the 3rd file slot conceptually but is dependency wiring, not logic.**
- **Reference**:
  - `/workspace/iscc-monitor/.claude/plans/cosmic-baking-octopus.md` — the "SQLite schema (core tables…)"
    block (lines ~135-147) is the authoritative column list; correctness rule 6 (single writer per DB).
  - `/workspace/iscc-monitor/.claude/adr/0005-single-sqlite-store.md` — WAL, single writer, one file,
    fetch-outside-the-write-transaction (the last point constrains *later* steps, not this one).
  - `/workspace/iscc-monitor/.claude/adr/0007-per-network-db-and-evidence-retention.md` — one file per
    network (`mainnet.db` / `testnet.db`); **no `network` column** in any table.
  - `/workspace/iscc-monitor/cauldron/tessera/client/fetcher.go` — the `Fetcher` 3-method interface
    (`ReadCheckpoint`, `ReadTile`, `ReadEntryBundle`) that a future `SQLiteFetcher` will implement over
    the `tiles`/`entry_bundles`/`checkpoints` BLOBs — informs the BLOB column shape, but **do not build
    the fetcher here**.
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` — existing `//go:embed`-free package style /
    file-docstring convention to match.

## Not In Scope
- **No follower loop, no polling, no goroutine supervisor, no `cmd/` binary.** This step only opens the
  store and applies the schema; the follower that calls `AcceptCheckpoint` and writes rows is the *next*
  step.
- **No typed insert/query/CRUD methods** (no `InsertCheckpoint`, no `hub_keys` upsert, no `follow_state`
  read/write helpers). Schema + open/close only. Row-access methods land with the follower so their
  shape is driven by a real caller (avoid speculative APIs — YAGNI).
- **No `SQLiteFetcher`** implementing `client.Fetcher` — that is M2 (Aggregator), and needs tiles in the
  store first.
- **No RFC-6962 consistency check, freeze/alert, OTS, metrics, or `iscc_index` logic.**
- **No sb1 fixture / `derive_vkey.py` `HUBS` refresh to `069d0f14`.** That belongs with the `hub_keys`
  write path (the follower step), not the empty-schema bootstrap.
- **Do NOT let `go get` bump the `go` directive to `1.25.0`** (see Implementation Notes) — the mise
  toolchain is pinned to `go 1.24`; a `go 1.25.0` minimum would fail the gate on a 1.24 toolchain.

## Implementation Notes
- **Driver + go-directive footgun (verified).** `go get modernc.org/sqlite@latest` currently pulls
  `v1.52.0` **and rewrites `go 1.24.0 → go 1.25.0`** plus a `toolchain go1.25.x` line. The mise gate runs
  `go = "1.24"`, so this *will* break `mise run check`. After adding the dep, **reset the directive back
  to `go 1.24.0`** and **delete any `toolchain` line** from `go.mod` (same discipline learnings already
  record for `x/mod`). Verify `mise run check` is green on the 1.24 toolchain afterward; if `v1.52.0`
  genuinely requires ≥1.25 to *build*, pin an older `modernc.org/sqlite` that builds on 1.24 rather than
  bumping the directive — and record which version in the commit body. The driver registers itself as
  the `"sqlite"` database/sql driver name (`import _ "modernc.org/sqlite"`), pure-Go, no cgo.
- **Open shape.** `func Open(path string) (*Store, error)` returning a small `type Store struct { db *sql.DB }`
  with a `func (s *Store) Close() error`. Use `database/sql` with driver name `"sqlite"`. This is the
  pure-Go driver chosen precisely because `CGO_ENABLED=0` forbids cgo SQLite (learnings).
- **Single-writer pragmas (ADR-0005/0007, correctness rule 6).** Set on open, in order:
  `PRAGMA journal_mode=WAL;`, `PRAGMA busy_timeout=5000;` (ms), `PRAGMA foreign_keys=ON;`,
  `PRAGMA synchronous=NORMAL;` (safe + fast under WAL). Enforce the single-writer invariant at the
  connection-pool level with `db.SetMaxOpenConns(1)` so all writes serialize through one connection —
  the goroutine-ownership wrapper comes with the follower; this step just guarantees the pool can't
  open a second writer. (Reads will also serialize for now; that is fine pre-serving and avoids
  `SQLITE_BUSY` flakes in tests. A read pool split can come when serving lands.)
- **Embedded schema.** Put the DDL in `schema.sql`, embed it with
  `import _ "embed"` + `//go:embed schema.sql` into a `var schemaSQL string`, and apply it on open via
  one `db.Exec(schemaSQL)`. Every statement is `CREATE TABLE IF NOT EXISTS …` (and `CREATE INDEX IF NOT
  EXISTS …`) so `Open` on an existing DB is a no-op — that is what makes restart-survival trivially true.
- **Schema = the plan's core tables verbatim, minus the dropped `network` column (ADR-0007).** Create
  exactly these, with the columns from the plan's "SQLite schema" block:
  `hubs`, `hub_keys`, `checkpoints` (`UNIQUE(hub_id,tree_size,root)`), `violations`, `tiles`
  (`PK(hub_id,level,tile_index,width)`), `entry_bundles` (`PK(hub_id,bundle_index,width)`), `iscc_index`
  (`seq` PK, `INDEX(iscc_id)`), `follow_state` (`hub_id` PK), `ots` (`UNIQUE(hub_id,tree_size,root)`).
  Use `BLOB` for `root`/`data`/`iscc_id`/`raw_*`/`ots_bytes`, `INTEGER` for sizes/booleans/timestamps
  (store times as unix-seconds or RFC-3339 TEXT — pick one and document it in the file docstring; unix
  INTEGER is simplest and matches `observed_at`-style comparisons). No `cosigs` table (deferred to M7).
  Keep the DDL readable and commented per table; this file is load-bearing documentation of the data
  model.
- **Correctness rules in play (learnings.md / plan):** rule 6 "SQLite single writer per DB — WAL +
  busy_timeout; one goroutine owns all writes" (enforced here by `SetMaxOpenConns(1)` + WAL; the
  goroutine wrapper is the follower's job); ADR-0007 "no `network` column"; `CGO_ENABLED=0` →
  `modernc.org/sqlite`, never a cgo driver, and **no `go test -race`**.
- **Tests** (offline, deterministic): use `t.TempDir()` for the DB path.
  1. `Open` on a fresh path succeeds and the expected tables exist — assert via
     `SELECT name FROM sqlite_master WHERE type='table'` and check the set contains every table above.
  2. `PRAGMA journal_mode` returns `wal` after open.
  3. **Restart survival**: `Open` → write a sentinel row into one table (e.g. an INSERT into `hubs`) →
     `Close()` → `Open()` the same path again → the row and all tables are still present. (A direct
     `db.Exec` INSERT in the test is fine even though the store exposes no insert method yet — the test
     drives raw SQL to prove persistence, not a public API.)
  4. `Open` twice on the same path (sequentially, after Close) is idempotent (schema re-apply is a
     no-op, no error).
  Assert on observable DB state (rows, pragma values), never on `Store` internals.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` exit 0; `gofmt -l .` empty)
  **on the pinned `go 1.24` toolchain** — i.e. `go.mod` still reads `go 1.24.0` with no `toolchain` line.
- `go test -run TestStore ./internal/store` passes (fresh-open, pragma, restart-survival, idempotent-reopen).
- Assertion: after `Open(filepath.Join(t.TempDir(), "testnet.db"))`, a
  `SELECT name FROM sqlite_master WHERE type='table'` result set contains all of:
  `hubs hub_keys checkpoints violations tiles entry_bundles iscc_index follow_state ots`.
- Assertion: `PRAGMA journal_mode` query returns `"wal"`.
- Assertion: a row inserted before `Close()` is still readable after reopening the same file path
  (restart survival).
- Assertion: `grep -R "network" internal/store/schema.sql` finds no `network` *column* (ADR-0007 — the
  word may appear only in a comment; the column is gone).
- `go.mod` `go` directive is `1.24.0` and contains no `toolchain` directive (run `grep -E '^(go|toolchain)' go.mod`).

## Done When
`internal/store.Open` opens a WAL, single-writer SQLite DB with all nine core M1 tables applied
idempotently from the embedded `schema.sql`, the data survives a close/reopen, and every Verification
criterion passes with `mise run check` green on the `go 1.24` toolchain.
