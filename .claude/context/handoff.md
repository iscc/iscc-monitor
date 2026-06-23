## 2026-06-23 — Rebuild `iscc_index` under a composite `(hub_id, seq)` PK as migration index 0, with the out-of-range `user_version` guard

**Done:** Re-keyed `iscc_index` from the original single-global `seq` PRIMARY KEY to the composite
`(hub_id, seq)` PK (fresh DB via `schema.sql`, pre-existing DB via the project's first real migration,
index 0), so two hubs in a multi-hub realm both index low leaves (both seq 0, 1, …) without one
clobbering the other. Added the fail-closed out-of-range `user_version` guard to `applyMigrations` so the
now-non-empty migration slice rejects a future version (silent-accept) or a negative version (the
`migs[-1]` panic) instead of either. Both paired data-model `normal`s close together.

**Files changed:**
- `internal/store/schema.sql`: `iscc_index` `seq INTEGER PRIMARY KEY` → `seq INTEGER NOT NULL` + table-level
  `PRIMARY KEY (hub_id, seq)` (the shape a FRESH DB gets directly); leading comment rewritten to describe
  the composite key (evergreen). `iscc_index_by_iscc_id` index unchanged.
- `internal/store/iscc_index.go`: writer `RecordProjections` conflict target `ON CONFLICT(seq)` →
  `ON CONFLICT(hub_id, seq)`; dropped the now-meaningless `hub_id = excluded.hub_id` from the `DO UPDATE SET`
  (hub_id is part of the conflict key, so it can never change on a conflict). Package/`RecordProjections`/
  `ProjectionRecord` docstrings that named `seq` "the PRIMARY KEY" now say the composite `(hub_id, seq)`.
  The four readers' SQL is untouched (they already scope by `hub_id`).
- `internal/store/sqlite.go`: appended migration index 0 (`migrateISCCIndexCompositePK`) doing the standard
  SQLite table-rebuild dance (create new-shape table → `INSERT … SELECT` copy → drop old → rename → recreate
  the iscc_id index), all inside the caller's `*sql.Tx`, no `user_version` bump (the runner does that). Added
  the `version < 0 || version > len(migs)` guard before the loop wrapping a new `errUnsupportedSchemaVersion`
  sentinel. Updated the `migrations` var + package/`Open`/`applyMigrations` docstrings off the "empty no-op
  baseline" wording to describe the one entry now present and the new guard.
- `internal/store/iscc_index_test.go` (test): added `TestRecordProjectionsMultiHubSeqZero` — two hubs both
  index seq 0; `RecordAt(hubA,0)→"ISCC:A"` and `RecordAt(hubB,0)→"ISCC:B"`, two rows survive.
- `internal/store/sqlite_test.go` (test): added `TestMigrationIsccIndexCompositePK` (production `migrations`
  slice over a seeded single-PK v0 DB → seeded row survives, `user_version == len(migrations)`, PK columns
  are `[hub_id seq]` via `pragma_table_info`, a second hub's matching seq inserts with no collision) and
  `TestMigrationOutOfRangeVersion` (future + negative `user_version` both return the
  `errUnsupportedSchemaVersion`-wrapped error). Added `reflect`/`fmt` imports + `oldISCCIndexDDL`,
  `pkColumns`, `countRows2` helpers.

**Verification:** `mise run check` → GREEN (build + vet + test, 30 pkgs). `gofmt -l .` → empty.
- Composite-PK multi-hub round-trip (the bug fix): `TestRecordProjectionsMultiHubSeqZero` PASS. Mutation —
  reverting writer + schema to single-PK makes it FAIL (`ON CONFLICT clause does not match any PRIMARY KEY`,
  proving the composite target is load-bearing; on a clean single-PK revert the later write would clobber).
- Migration upgrade-in-place: `TestMigrationIsccIndexCompositePK` PASS (seeded row survives, version → 1,
  PK = `[hub_id seq]`, no collision on the second-hub insert).
- Out-of-range guard fail-closed: `TestMigrationOutOfRangeVersion` PASS. Mutation — removing the guard makes
  it FAIL (future version returns nil; negative would panic on `migs[-1]`).
- All five existing `TestMigration*` PASS; the fresh-DB test now sees `len(migrations) == 1` automatically
  (it asserts `userVersion == len(migrations)`).
- All existing `iscc_index` tests (idempotency, one-to-many, ListRecords/RecordAt/RecentRecords, hub-scope)
  PASS — the `DO UPDATE SET` body is unchanged, idempotency intact.
- Store leaf purity intact: `go list -deps ./internal/store | grep '^net/http$'` empty. `go.mod`/`go.sum`
  byte-unchanged. The `internal/follower` caller of `RecordProjections` is unaffected (signature unchanged) —
  its suite is green.

**Oracle/conformance gate:** N/A — touches no signature / RFC-6962 / Merkle / did:web / proof path; it is
`iscc_index` DDL + `user_version` bookkeeping (the advance and `next.md` both call this N/A).

**Next:** Both paired data-model `normal`s are now closed. The migration mechanism + first real migration are
proven end-to-end, so the on-disk migration story is no longer a no-op baseline — a future `update-state`
should close the standing "No on-disk DB migration story" `normal` (the mechanism landed AND now has a real
entry). No further code-closable migration/iscc_index work is queued; the next deliberate step should come
from `issues.md` priority or the milestone backlog.

**Notes:**
- I dropped `hub_id = excluded.hub_id` from `RecordProjections`' `DO UPDATE SET` because `hub_id` is now
  part of the conflict-target key: an `ON CONFLICT(hub_id, seq)` only fires when both already match the
  inserted row, so re-assigning `hub_id` to its own value was dead. The remaining six `SET` columns are
  byte-identical to before, so the second-write-wins idempotency (incl. `note_timestamp`) is unchanged and
  `TestRecordProjectionsIdempotent` still passes. Not a behavioural change — flagging it as a deliberate,
  in-scope tidy of the upsert that the PK change made vacuous.
- Used a sentinel `errUnsupportedSchemaVersion` (matched via `errors.Is`) rather than a bare string so the
  test assertion is robust to message wording, per the `next.md` implementation note's "sentinel var or
  wrapped error" option.
- The migration's new-table DDL is a verbatim copy of the `schema.sql` `iscc_index` body (composite PK), so
  the fresh-DB path and the upgrade path converge on one shape; if either is edited later, edit both (the PK
  rework is append-only — this entry must never be reordered/edited per store.md).
- Pre-existing `.claude/context/target.md` working-tree mod (from steer `d2f259e`) is still uncommitted — not
  mine to commit (advance writes only source/test + handoff); left for `update-state`/`steer`.
- `TestMigrationOutOfRangeVersion` builds the `PRAGMA user_version = N` via `fmt.Sprintf`, not a `?`
  placeholder — SQLite rejects a bound parameter on that pragma (the exact nuance the production runner
  already documents; I hit it once and corrected it).
