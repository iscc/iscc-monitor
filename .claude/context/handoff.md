## 2026-06-23 — Review of: Add a `PRAGMA user_version`-gated migration runner to `store.Open`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance added the project's first on-disk migration mechanism — a `PRAGMA user_version`-gated,
append-only `migrations []func(*sql.Tx) error` run inside `store.Open` after the baseline `CREATE TABLE IF
NOT EXISTS` pass (`applyMigrations` + per-step `applyMigration`, each step in its own `*sql.Tx`,
fail-closed, idempotent). Exactly in-scope (1 non-test file `sqlite.go` + 1 test file + 1 doc), every gate
green, both mutation claims independently reproduced. PASS_WITH_NOTES (not PASS) only because Codex surfaced
a real, currently-latent fail-open in the runner's version handling (no bound on the read-back
`user_version`) that I reproduced and filed `normal` — it does not block this increment but goes live at the
very next step.

**Verification:**
- [x] `mise run check` (build + vet + test, 30 pkgs) — GREEN.
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -run TestMigration ./internal/store -v` — all 5 PASS (`FreshDBAtCurrentVersion`,
  `UpgradesOldDB`, `Idempotent`, `FailClosed`, `AppliesInOrder`).
- [x] **Fresh-DB version assertion** — `TestMigrationFreshDBAtCurrentVersion` PASS (`user_version ==
  len(migrations)`); independently confirmed via the broader `TestStoreReopenIdempotent` pass.
- [x] **Old-DB upgrade assertion** — `TestMigrationUpgradesOldDB` PASS (synthetic 1-entry slice over a v0 DB
  runs once → version 1 → marker table present).
- [x] **Idempotency assertion** — `TestMigrationIdempotent` PASS (second run executes zero migrations).
- [x] **Fail-closed assertion** — `TestMigrationFailClosed` PASS (errored step leaves `user_version` 0,
  rolls back its write, returns the `%w`-wrapped error via `errors.Is`).
- [x] **Mutation 1 (reviewer-reproduced):** removing the `PRAGMA user_version = %d` bump → `TestMigrationIdempotent`
  FAILS (`table migration_marker already exists` — re-ran) AND `TestMigrationUpgradesOldDB` FAILS (version
  stays 0). Source restored pristine.
- [x] **Mutation 2 (reviewer-reproduced):** swallowing the migration error in `applyMigration` →
  `TestMigrationFailClosed` FAILS (`returned nil, want error`). Source restored pristine.
- [x] **Open ordering correct** — pragmas → `db.Exec(schemaSQL)` → `applyMigrations(db, migrations)` →
  return Store; the migrate error path mirrors the existing `schemaSQL` path (`db.Close()` + `%w`-wrap).
- [x] **Store leaf purity intact** — `go list -deps ./internal/store | grep '^net/http$'` empty.
- [x] `go.mod` / `go.sum` byte-identical to HEAD~1; only the pre-existing `target.md` steer-artifact mod is
  in the working tree (correctly not committed by advance).
- [x] Gate-circumvention scan over the 3 unpushed commits — no `nolint`/`t.Skip`/swallowed-err/build-tag
  dodge, no deleted tests/assertions; the source diff is purely additive. (The `defer _ = tx.Rollback()` is
  the established post-commit `sql.ErrTxDone` idiom, not an error-swallowing dodge.)
- [x] `deploy/OPERATING.md` §Migration-policy — accurately describes the mechanism that now exists AND keeps
  the honest "recreate the volume on a schema change" caveat (the list is still a no-op baseline).

**Oracle/conformance gate:** N/A — touches no signature / RFC-6962 / Merkle / did:web / proof path; it is
`user_version` bookkeeping + DDL plumbing. Reviewer concurs with the advance's N/A call.

**Issues found:** One new `normal` (Codex-confirmed, reviewer-reproduced) — the runner does not bound the
read-back `user_version`, so a future `user_version > len(migrations)` opens SILENTLY (unsupported schema
accepted) and a negative `user_version = -1` PANICS on `migs[-1]` once the slice is non-empty. Latent today
(empty production slice → only `version == 0` ever occurs), but BOTH go live at migration index 0 (the
scheduled next step), so fix the `version < 0 || version > len(migs)` guard with or before it. Filed. Also
updated the standing "No on-disk DB migration story" `normal` to reflect that the mechanism landed but the
production list is still empty (close it when the first real migration lands).

**Codex second opinion:** Returned exactly one finding, `[P2]` — "Reject out-of-range user_version values"
(`sqlite.go:124-125`): the loop treats an out-of-range stored `user_version` as valid — a newer-binary DB
(`> len(migrations)`) opens silently and a corrupt `-1` panics by indexing `migs[-1]`. **CONFIRMED** —
reproduced both halves with a throwaway probe (future version 5 over the empty prod slice → `Open` SUCCEEDS;
`-1` with a 1-entry synthetic slice → `index out of range [-1]` panic). Triaged `normal`, not a blocker:
not reachable in production today (empty slice → only `version == 0`), all gates green, but it becomes live
at the next step. Filed as an `issues.md` entry for a later advance; not fixed here (review is read-only).
No other findings. (Codex ran ~5 min, including its own negative-`user_version` probe program.)

**Visual check:** n/a — no SSR surface changed. The diff touches only `internal/store/sqlite.go`
(non-rendering data-layer plumbing), its test file, and `deploy/OPERATING.md`; no server-rendered HTML
template was edited.

**Next:** The migration mechanism is in place, so the scheduled next step is the `iscc_index.seq`
single-global-PK → composite `(hub_id, seq)` rebuild, landing as migration index 0 (re-keys `iscc_index`,
copies existing rows under the new PK) plus the `iscc_index.go` writer/reader updates. **Fold the
out-of-range `user_version` guard into that same step** (or do it first) — the negative-version panic goes
live the instant the slice becomes non-empty, and the future-version silent-accept undermines the
fail-closed contract that step relies on. Both data-model `normal`s then close together.

**Notes:**
- Advance's "reverting the runner call makes the upgrade test FAIL" caveat is accurate: with the empty
  production slice that mutation is observably a no-op (a fresh DB is already at v0), so the wiring is
  proven load-bearing only via the synthetic-slice path / a seed-one-prod-migration probe — expected by the
  deliberate no-op-baseline design, not a gap. I confirmed Mutations 1 and 2 (the user_version bump and the
  error-propagation) ARE non-vacuous against the synthetic-slice tests.
- `learnings/store.md` net-reduced this iteration (192 → ~186 lines): collapsed the verbose freeze /
  `ListViolations` / `RecordHubKey` settled bullets into one-line `settled:` summaries (git history keeps
  the detail) and replaced the stale "no migration story exists" bullet with the migration-runner note
  (including the two out-of-range edges to guard).
- The pre-existing `.claude/context/target.md` working-tree mod (from steer `d2f259e`) is still uncommitted
  — not review's to commit (must not modify target.md); left for the next `update-state`/`steer`.
