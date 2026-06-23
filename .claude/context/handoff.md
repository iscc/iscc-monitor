## 2026-06-23 — Add a `PRAGMA user_version`-gated migration runner to `store.Open`

**Done:** Added the project's first on-disk migration mechanism — a `PRAGMA user_version`-gated runner
(`applyMigrations` + per-step `applyMigration`) that runs in `store.Open` after the baseline
`CREATE TABLE IF NOT EXISTS` pass, applying an ordered, append-only `migrations` slice and advancing
`user_version` to the code's current version (`len(migrations)`). The production `migrations` slice
ships **empty** (no-op baseline this step, per scope), so no real schema changed yet; correctness is
proven by tests feeding the runner a synthetic slice.

**Files changed:**
- `internal/store/sqlite.go`: added the `migrations` append-only slice (empty baseline + explanatory
  comment naming the `iscc_index` composite-PK rebuild as the first planned entry), `applyMigrations`
  (version-gated loop) and `applyMigration` (one step per `*sql.Tx`, `fmt.Sprintf` `user_version` bump
  since SQLite won't bind a parameter there, fail-closed `%w`-wrap, `AdvanceAccepted` `defer Rollback`
  idiom); wired the call into `Open` after the `schemaSQL` exec mirroring the existing error path;
  updated the package + `Open` docstrings to describe the runner and the version-gated no-op invariant.
- `internal/store/sqlite_test.go`: added `errors` import, `userVersion`/`rawOpen` helpers, and five
  `TestMigration…` tests (fresh-DB version, old-DB upgrade, idempotency, fail-closed rollback, in-order).
- `deploy/OPERATING.md`: rewrote §"Migration policy (interim)" to describe the mechanism that now exists
  while keeping the honest caveat that the list is still a no-op baseline, so "recreate the volume on a
  schema change" still holds today.

**Verification:** `mise run check` → GREEN (build + vet + test, 30 pkgs); `gofmt -l .` empty.
- Fresh-DB version assertion → `TestMigrationFreshDBAtCurrentVersion` PASS (`user_version == len(migrations)`).
- Old-DB upgrade assertion → `TestMigrationUpgradesOldDB` PASS (synthetic 1-entry slice over a v0 DB runs
  once, advances to 1, marker table present).
- Idempotency assertion → `TestMigrationIdempotent` PASS (second run executes zero migrations).
- Fail-closed assertion → `TestMigrationFailClosed` PASS (errored step leaves `user_version` 0, rolls
  back its write, returns the `%w`-wrapped error via `errors.Is`).
- In-order assertion → `TestMigrationAppliesInOrder` PASS.
- `go test -run TestMigration ./internal/store` → all 5 PASS.
- **Mutation checks (both confirmed, source restored pristine):** deleting the `PRAGMA user_version = %d`
  bump → `TestMigrationIdempotent` FAILS (`table migration_marker already exists` — migration re-ran);
  reverting the `applyMigrations` call in `Open` → with the empty production slice it does NOT change
  observable state (a fresh DB is already at v0), so to demonstrate the wiring I seeded one production
  migration locally and re-dropped the call → `TestMigrationFreshDBAtCurrentVersion` FAILS
  (`user_version = 0, want len(migrations) = 1`). Both reverted; `TestMigration` green afterward.

**Oracle/conformance gate: N/A** — this touches no signature / RFC-6962 / Merkle / did:web / proof path;
it is `user_version` bookkeeping + DDL plumbing.

**Next:** The migration mechanism is now in place, so the paired data-model `normal` can be closed: the
`iscc_index.seq` single-global-PK → composite `(hub_id, seq)` fix is the next step. It appends migration
index 0 (a real `ALTER`/table-rebuild that re-keys `iscc_index` and copies existing rows under the new
PK), and touches `iscc_index.go` writers/readers (`RecordProjections`, `SeqsForISCCID`, `RecordAt`) +
`RecentRecords`/`ListRecords` — its own focused increment with a real data-rebuild correctness test.

**Notes:**
- The "reverting the runner call makes the upgrade test FAIL" mutation in `next.md` is only literally true
  once a real migration exists — with the no-op baseline (`len(migrations)==0`) a fresh DB is already at
  v0, so removing the call leaves observable state identical and `applyMigrations`/`migrations` stay
  referenced by the synthetic-slice tests (still compiles). I proved the wiring is load-bearing by the
  seed-one-prod-migration probe above; review can reproduce it. This is expected given the deliberate
  no-op-baseline design, not a gap.
- The runner runs once, synchronously, in `Open` before the `Store` handle is returned, so no follower
  goroutine races it; `SetMaxOpenConns(1)` serializes the per-step transactions like every other write.
- Did NOT touch `schema.sql`, `Open`'s signature, the single-writer pragmas, or `iscc_index`'s PK — all
  per Not-In-Scope. The pre-existing `.claude/context/target.md` working-tree mod (from steer `d2f259e`)
  is still uncommitted and is not mine to commit.
