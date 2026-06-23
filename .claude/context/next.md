# Next Work Package

## Step: Add a `PRAGMA user_version`-gated migration runner to `store.Open`

## Advances
This closes the foundational half of the standing data-model `normal` filed in `issues.md`:

> **"No on-disk DB migration story — a column added to an existing table never reaches a
> pre-existing database"** — `store.Open` applies `schema.sql` as one `db.Exec` of nine
> `CREATE TABLE IF NOT EXISTS` (zero `ALTER TABLE`, no `PRAGMA user_version`, no migration
> framework), so a column added to an EXISTING table is a silent no-op on a pre-existing DB,
> and an upgraded node hits `no such column`.

It is the explicit next-scheduled code work the state names: *"Data-model `normal`s (code-closable,
paired): the on-disk DB migration story (`store.Open` `PRAGMA user_version`/idempotent `ALTER TABLE`)
and the `iscc_index.seq` single-global-PK multi-hub collision — a PK change needs the migration
mechanism, so they fold together."* The migration mechanism is the **prerequisite**: the
`iscc_index.seq → (hub_id, seq)` composite-PK fix is the *next* step and cannot land safely without it.
With both order-independent milestones (M-Deploy + M-API) complete in-repo and **0 critical / 0 started
milestone Verify open**, scheduling this paired data-model normal is the correct deliberate move (per the
state's "Next Milestone" guidance and the `loop-stalls-on-blocked-DONE` memory — code-closable work over
cosmetic chrome).

## Goal
Give the project its **first on-disk migration mechanism**: a `PRAGMA user_version`-gated runner that
applies an ordered, append-only migration list on `Open` *after* the idempotent `CREATE TABLE IF NOT
EXISTS` pass, advancing the stored `user_version` to the code's current version and never re-running an
already-applied step. This is the skeleton the `iscc_index` PK change (and every future column add) will
hang its `ALTER`/rebuild off. It ships verifiable on its own with a **no-op (baseline) migration set**, so
the mechanism is proven correct before any schema actually changes.

## Scope
- **Create**: (none — the runner lives in the existing `sqlite.go`)
- **Modify**:
  - `internal/store/sqlite.go` — add the migration runner + the migration list, call it from `Open` after
    the `schemaSQL` exec. (1 non-test file — the only one against the ≤3 budget.)
  - `internal/store/sqlite_test.go` — add the golden tests (test file, not counted).
  - `deploy/OPERATING.md` — §"Migration policy (interim)" (lines 71-84) is now stale: it states "there is
    **no on-disk migration mechanism yet**". Update it to describe the mechanism that now exists (the
    `user_version`-gated runner applied on `Open`) while keeping the honest caveat that the **current
    migration list is still a no-op baseline** — so the "recreate the volume on a schema change" interim
    policy *still holds today* until a real `ALTER`/rebuild migration is added (the next step). Doc-sync, not
    counted against the file budget.
- **Reference**:
  - `.claude/context/learnings/store.md` — the `internal/store` detail file (single-writer leaf rules;
    `Open` idempotency; the "No on-disk migration story" bullet at lines 73-79; the `iscc_index.seq`
    global-PK collision bullet at lines 49-59; the `AdvanceAccepted` post-commit `tx.Rollback()` idiom at
    lines 167-173). **Read it before writing.**
  - `internal/store/schema.sql` — the nine `CREATE TABLE IF NOT EXISTS` baseline the runner runs *after*.
  - `internal/store/sqlite.go` lines 50-77 — the current `Open` (pragmas → `schemaSQL` exec → return).
  - `internal/store/sqlite_test.go` — the existing `t.TempDir()` + reopen test pattern + `coreTables`
    pinning to mirror.

## Not In Scope
- **Do NOT change `iscc_index.seq` to a composite `(hub_id, seq)` PK in this step.** That is the *next*
  step, built on this mechanism. Adding it here would (a) blow the one-step boundary, (b) require an actual
  data-rebuilding migration whose correctness deserves its own focused increment, and (c) touch
  `iscc_index.go` writers/readers + `RecentRecords`. Land the runner first; the baseline migration list
  stays a no-op.
- Do NOT add an ORM / external migration framework (`golang-migrate`, etc.) — KISS: a tiny stdlib
  `user_version`-gated slice of migration funcs, matching the project's "no external dep we don't need"
  posture (ADR-0005 single SQLite adapter).
- Do NOT touch `schema.sql`'s `CREATE TABLE IF NOT EXISTS` statements — fresh DBs keep getting the full
  baseline from `schemaSQL`; the runner only handles the *delta* on already-created tables.
- Do NOT change `Open`'s signature, the single-writer pragmas, or `SetMaxOpenConns(1)`.

## Implementation Notes
- **Mechanism (KISS, stdlib only):** in `sqlite.go`, define an ordered, append-only `migrations` slice
  where index `i` is the step that lifts `user_version` from `i` to `i+1` (so `len(migrations)` is the
  code's current schema version). Each entry is a small `func(*sql.Tx) error`. Factor the loop into a
  package-level runner (e.g. `applyMigrations(db *sql.DB, migs []func(*sql.Tx) error) error`) and call it
  from `Open` with the production slice — making it a func taking the slice keeps it directly unit-testable
  with a synthetic slice (so the test does not depend on the production list ever being non-empty).
- **Runner body:** AFTER the existing `db.Exec(schemaSQL)` baseline pass, read `PRAGMA user_version` (a
  single-row `db.QueryRow("PRAGMA user_version").Scan(&v)` into an `int`), then for every version `v` from
  the stored value up to `len(migs)`, run `migs[v]` inside its **own transaction** and, on success, set the
  new version. **`PRAGMA user_version = ?` does NOT accept a bound parameter in SQLite** — build the
  statement with `fmt.Sprintf("PRAGMA user_version = %d", v+1)`; the value is an in-code `int` (never user
  input), so there is no injection surface. Wrap each migration + its version bump in one `*sql.Tx` using the
  established post-commit `defer func(){ _ = tx.Rollback() }()` idiom (per `learnings/store.md` —
  `AdvanceAccepted` is the precedent; a post-`Commit` rollback returns benign `sql.ErrTxDone`).
- **Baseline = no-op this step.** Ship the PRODUCTION `migrations` slice **empty** (with an explanatory
  comment that the `iscc_index` composite-PK migration is the first planned entry, landing in the next
  step). With an empty list the current version is `0`; a fresh DB created by `schemaSQL` is already at the
  baseline, so the runner advances `user_version` to `0` and runs nothing. The runner's correctness is
  proven by the unit test feeding it a **synthetic** migration slice — so this step changes no real schema
  yet ships a fully-exercised, non-vacuous mechanism.
- **Idempotency contract (the load-bearing property):** re-`Open` on an already-migrated DB must run ZERO
  migrations — `user_version` already equals `len(migs)`, so the loop body never executes. The existing
  `Open`-is-a-no-op-on-existing-DB invariant (docstring lines 11-12, 56-58) must still hold; update that
  docstring to mention the version-gated runner.
- **Fail-closed:** a migration error must `%w`-wrap and abort `Open` — return the error after `db.Close()`,
  mirroring the existing `schemaSQL`-exec error path at lines 72-75 — never leave a half-migrated DB. The
  per-migration transaction rolls back on error, so a failed step leaves `user_version` unadvanced.
- **Single-writer discipline holds:** `SetMaxOpenConns(1)` means the migration transactions serialize on the
  one connection like every other write — no concurrency concern. The runner runs once, synchronously, in
  `Open` before the Store handle is returned, so no follower goroutine races it.
- **Relevant Correctness rule (learnings.md):** *"SQLite single writer per DB — one goroutine owns all
  writes."* The migration runs inside `Open` before any other goroutine has the handle, so it is trivially
  the single writer. Combined with the fail-closed discipline above.
- **Oracle/conformance gate: N/A** — this touches no signature / RFC-6962 / Merkle / did:web / proof path;
  it is `user_version` bookkeeping + DDL plumbing. State so in the commit.

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...`, `gofmt -l .` empty).
- `go test -run TestMigration ./internal/store` passes (name the new tests `TestMigration…` so this filter
  catches them all).
- **Fresh-DB version assertion:** after `store.Open` on a `t.TempDir()` path, `PRAGMA user_version`
  equals `len(migrations)` (the code's current version) — a test opens a new DB and reads the pragma back
  via a raw `database/sql` handle.
- **Old-DB upgrade assertion (the non-vacuous core):** the runner, fed a synthetic one-entry migration
  slice against a DB whose `user_version` is `0`, runs that one migration exactly once, advances
  `user_version` to `1`, and the migration's observable effect (e.g. a sentinel `ALTER TABLE … ADD COLUMN`
  or a marker row) is present afterward.
- **Idempotency assertion:** running the runner a SECOND time over the now-migrated DB executes ZERO
  migrations (assert via a call counter on the synthetic migration, or that the side effect is not
  duplicated and `user_version` is unchanged).
- **Fail-closed assertion:** a synthetic migration that returns an error leaves `user_version` unadvanced
  and the DB unchanged, and the runner returns the wrapped error.
- **Mutation check** (record in the commit): deleting the `PRAGMA user_version = N` bump makes the
  idempotency test FAIL (the migration re-runs); reverting the runner call in `Open` makes the upgrade test
  FAIL.

## Done When
`store.Open` runs a `PRAGMA user_version`-gated migration runner after the baseline schema pass — proven by
the fresh-version, old-DB-upgrade, idempotency, and fail-closed tests above — with the production migration
list still a no-op baseline, `deploy/OPERATING.md` §Migration-policy updated to describe the mechanism, and
`mise run check` green.
