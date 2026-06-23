# Next Work Package

## Step: Rebuild `iscc_index` under a composite `(hub_id, seq)` PK as migration index 0, with the out-of-range `user_version` guard

## Advances
This step is justified by two open `normal` issues that the loop's scheduled-next handoff
(`review` 2026-06-23 **Next:**) and `state.md` "Next Milestone" both name as the next deliberate code
work, now that the code-closable feature/milestone backlog is drained:

1. **"`iscc_index.seq` is a single global PRIMARY KEY but ingest writes per-hub absolute leaf indices —
   multi-hub PK collision"** (`normal`, `issues.md`): two followed hubs sharing a leaf index (every realm
   with ≥2 active hubs: both have seq 0, 1, …) collide on the global PK and `RecordProjections`'
   `ON CONFLICT(seq) DO UPDATE` silently clobbers the earlier hub's row. This is a multi-hub data-model
   correctness defect on the M2 `iscc_index` projection (ADR-0008).
2. **"Migration runner does not bound the read-back `user_version` — future-version silent-accept +
   negative-version panic (both go live at migration index 0)"** (`normal`, `issues.md`): the moment the
   `migrations` slice becomes non-empty, a `user_version > len(migs)` opens silently and a `-1` panics on
   `migs[-1]`. The state, the review handoff, and `learnings/store.md` all say to fix this **with or
   before** the first real migration — which is exactly this step.

No milestone Verify criterion is unmet (all M1→M-API bars are MET, carried forward); these two paired
`normal`s are the standing code-closable work, and they close together because the migration runner is
the enabling mechanism the prior window deliberately landed (no-op baseline) for precisely this.

## Goal
Make `iscc_index` faithfully hold every hub's low leaves in a multi-hub realm by re-keying it on the
composite `(hub_id, seq)` PK, delivered as the project's FIRST real on-disk migration (index 0) so a
pre-existing single-PK database upgrades in place; and bound the migration runner's read-back
`user_version` so the now-non-empty slice stays fail-closed instead of silently accepting a future
version or panicking on a negative one.

## Scope
- **Modify**:
  - `internal/store/schema.sql` — change the `iscc_index` table's `seq INTEGER PRIMARY KEY` to a column
    `seq INTEGER NOT NULL` plus a table-level `PRIMARY KEY (hub_id, seq)` (mirroring the existing
    composite PKs on `tiles` / `entry_bundles`). This is the DDL a FRESH database gets directly; the
    migration below brings a PRE-EXISTING database to the same shape. Update the table's leading comment
    so it states the composite key (evergreen — describe the current state).
  - `internal/store/iscc_index.go` — change `RecordProjections`' upsert conflict target from
    `ON CONFLICT(seq)` to `ON CONFLICT(hub_id, seq)` (the new PK), and update the `RecordProjections` /
    `ProjectionRecord` / `iscc_index.go` package docstrings that call `seq` "the PRIMARY KEY" to say the
    composite `(hub_id, seq)`. The four readers (`ListRecords`, `RecordAt`, `RecentRecords`,
    `SeqsForISCCID`) already scope every query by `hub_id` and select `seq` plainly, so their SQL is
    unchanged — do not touch their queries.
  - `internal/store/sqlite.go` — (a) append migration index 0 to the `migrations` slice: a function
    `func(tx *sql.Tx) error` that rebuilds `iscc_index` under the composite PK via SQLite's standard
    table-rebuild dance (create the new-shape table under a temp name, `INSERT INTO … SELECT …` to copy
    existing rows, drop the old table, rename the new one into place, recreate the
    `iscc_index_by_iscc_id` index); update the `migrations` var docstring + the package/`Open` docstrings
    that say the list is an "empty no-op baseline" to describe the one entry now present. (b) Add the
    out-of-range guard in `applyMigrations`: before the loop, `if version < 0 || version > len(migs) {
    return error }` (wrapped), so an unsupported on-disk version is rejected fail-closed instead of
    opening silently / panicking on `migs[-1]`.

  That is exactly 3 non-test source files (`schema.sql`, `iscc_index.go`, `sqlite.go`). Test files
  (`internal/store/sqlite_test.go`, `internal/store/iscc_index_test.go`) and docs are not counted.
- **Reference**:
  - `.claude/context/learnings/store.md` — the migration-runner note (append-only, never edit/reorder a
    released entry; the two out-of-range edges to guard) and the `iscc_index` writer/reader settled facts.
  - `internal/store/schema.sql` `tiles` / `entry_bundles` — the established composite-PK pattern to copy.
  - `internal/store/sqlite_test.go` — the existing `TestMigration*` harness (`rawOpen`, `userVersion`,
    synthetic-slice `applyMigrations` calls, `tableSet`) the new tests extend.
  - `internal/store/iscc_index_test.go` — the existing `RecordProjections` / reader round-trip tests
    whose multi-hub collision assertion you add.

## Not In Scope
- **Do NOT change the four reader queries** (`ListRecords`/`RecordAt`/`RecentRecords`/`SeqsForISCCID`).
  They already filter by `hub_id`; the composite PK is transparent to them. Touching them risks a
  regression with no benefit (only docstrings that name `seq` "the PRIMARY KEY" get a wording update).
- **Do NOT re-key `RecentRecords`' cross-hub ordering.** It orders newest-first by the global `seq`,
  which is no longer a true global recency once two hubs reuse low seqs — but that is a known, separately
  filed concern (the store.md note) and the only monotonic signal the store has. Leave its `ORDER BY
  i.seq DESC` as-is; do not invent an observed-at column.
- **Do NOT change `follower/ingest.go`** — it already writes the per-hub absolute leaf index as `Seq`;
  the bug was the PK, not the writer's seq value. The composite PK makes the existing per-hub seq correct.
- **Do NOT add a second migration or any other schema change.** Exactly one entry (index 0).
- **Do NOT touch the `iscc_index_by_iscc_id` index definition in `schema.sql`** beyond what the rebuild
  migration recreates — the BLOB-`iscc_id` lookup index is unchanged.

## Implementation Notes
- **SQLite cannot ALTER a PRIMARY KEY in place** — the migration must use the documented table-rebuild
  (here the simple form, since no FK references `iscc_index.seq`): inside the migration's `*sql.Tx`,
  `CREATE TABLE iscc_index_new (…composite PK…)`, `INSERT INTO iscc_index_new (hub_id, seq, iscc_id,
  iscc_id_str, note_schema, note_timestamp, record_sha256) SELECT hub_id, seq, iscc_id, iscc_id_str,
  note_schema, note_timestamp, record_sha256 FROM iscc_index`, `DROP TABLE iscc_index`,
  `ALTER TABLE iscc_index_new RENAME TO iscc_index`, then
  `CREATE INDEX IF NOT EXISTS iscc_index_by_iscc_id ON iscc_index (iscc_id)`. The new-table DDL inside the
  migration MUST match the `schema.sql` shape a fresh DB gets, so the two converge. Note: a fresh DB never
  runs this migration (it is created by `schema.sql` already at the composite shape, then `user_version`
  jumps straight to `len(migrations)`); only a pre-existing single-PK DB runs it. Existing-row copy is
  safe because a pre-existing DB had a single global PK, so no two copied rows can collide on
  `(hub_id, seq)` (each old `seq` was globally unique). `PRAGMA foreign_keys=ON` is set on the connection;
  the drop/rename of a table that nothing references is fine — prefer NOT to toggle `foreign_keys` unless
  a test forces it (no FK points at `iscc_index`).
- **The migration runs inside `applyMigration`'s per-step `*sql.Tx` and is fail-closed** — return any
  error from the `tx.Exec` calls; the runner rolls back and leaves `user_version` unadvanced. Do NOT bump
  `user_version` inside the migration (the runner does that).
- **Append-only discipline (store.md, correctness rule):** add the entry at index 0 of `migrations`;
  never edit or reorder it later. `len(migrations)` becomes 1, so a fresh DB ends at `user_version == 1`.
  This shifts the existing `TestMigrationFreshDBAtCurrentVersion` assertion from `len(migrations) == 0`
  to `== 1` — it already asserts `userVersion == len(migrations)`, so it stays green automatically.
- **The writer conflict target** `ON CONFLICT(hub_id, seq)` must name the exact composite PK columns in
  that order; SQLite matches the conflict target to the PK's column set. Verify the existing
  `iscc_index` idempotency test (re-ingesting a bundle overwrites in place) still passes — the
  `DO UPDATE SET` body is unchanged.
- **The out-of-range guard** (`learnings/store.md`, the filed `normal`): in `applyMigrations`, after
  reading `version` and before the loop, `if version < 0 || version > len(migs) { return
  fmt.Errorf("unsupported on-disk schema version %d (code supports up to %d): %w", version, len(migs),
  …) }`. Use a sentinel `var` or a plain wrapped error the test can assert on. This makes a future-version
  DB fail-closed and a `-1` return an error instead of panicking on `migs[-1]`.
- **Relevant correctness rules:** learnings.md "fail-closed" discipline (the guard); store.md "Append
  migrations, never edit/reorder a released entry"; ADR-0008 schema-agnostic index (the rebuild copies
  the verbatim columns, interprets nothing); the store stays a **leaf** — no new imports
  (`go list -deps ./internal/store | grep '^net/http$'` must stay empty).
- **Oracle/conformance gate: N/A** — this touches no signature / RFC-6962 / Merkle / did:web / proof
  path; it is `iscc_index` DDL + `user_version` bookkeeping. State the N/A in the advance.

## Verification
- `mise run check` is green (build + vet + test over all packages) and `gofmt -l .` is empty.
- **Composite-PK multi-hub round-trip (the bug fix):** a new `internal/store/iscc_index_test.go` test
  seeds two hubs (hub_id 1 and 2, via the same `INSERT INTO hubs …` the existing tests use to satisfy the
  FK), `RecordProjections` for `{HubID:1, Seq:0, IsccID:"ISCC:A"}` and `{HubID:2, Seq:0, IsccID:"ISCC:B"}`,
  then `RecordAt(ctx,1,0)` returns `"ISCC:A"` (found) AND `RecordAt(ctx,2,0)` returns `"ISCC:B"` (found) —
  neither clobbers the other. `go test -run TestRecordProjectionsMultiHubSeqZero ./internal/store` passes;
  on the OLD single-PK schema (or with `ON CONFLICT(seq)`) one of the two reads would return the wrong id.
- **Migration upgrade-in-place:** a new test in `internal/store/sqlite_test.go` creates a DB with the
  OLD single-PK `iscc_index` DDL + a seeded row at `user_version = 0` (drop+recreate the table on a raw
  handle, or seed via `rawOpen`), then runs the PRODUCTION `migrations` slice via `applyMigrations(db,
  migrations)` and asserts: the seeded row survives, `user_version == len(migrations)` (== 1), and a
  subsequent insert of a second hub's `seq` matching the first hub's `seq` succeeds (no PK collision).
  `go test -run TestMigrationIsccIndexCompositePK ./internal/store` passes.
- **Out-of-range guard fail-closed:** extend the migration tests so `applyMigrations(db, migs)` with
  `user_version` set to `len(migs)+1` returns a (wrapped) error, and `user_version = -1` returns an error
  (not a panic). `go test -run TestMigrationOutOfRangeVersion ./internal/store` passes; reverting the
  guard makes both FAIL (the future-version case opens silently; the `-1` case panics with a non-empty
  slice).
- **All five existing `TestMigration*` tests still pass:**
  `go test -run TestMigration ./internal/store` is green (the runner change is additive; the fresh-DB
  test now sees `len(migrations) == 1`).
- **Store leaf purity intact:** `go list -deps ./internal/store | grep '^net/http$'` is empty.

## Done When
`mise run check` is green and every Verification check passes: `iscc_index` is keyed on the composite
`(hub_id, seq)` PK (fresh DB via `schema.sql`, pre-existing DB via migration index 0), two hubs can both
index `seq 0` without clobbering, and the migration runner rejects an out-of-range `user_version`
fail-closed — closing both paired data-model `normal`s together.
