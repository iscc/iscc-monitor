## 2026-06-23 — Review of: Rebuild `iscc_index` under a composite `(hub_id, seq)` PK as migration index 0, with the out-of-range `user_version` guard

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance re-keys `iscc_index` from the single-global `seq PRIMARY KEY` to the composite
`(hub_id, seq)` PK (fresh DB via `schema.sql`, pre-existing DB via the project's first real migration,
index 0) and bounds the migration runner's read-back `user_version` so the now-non-empty slice rejects an
out-of-range version fail-closed. Scope is exactly the 3 asked source files; `mise run check` is green,
`gofmt` clean, store leaf-purity intact, `go.mod`/`go.sum` byte-unchanged. I independently mutation-proved
all three new tests are load-bearing (single-PK revert, guard removal, neutered-migration each FAIL the
matching test) — the work is correct and well-gated.

**Verification:**
- [x] `mise run check` green (build + vet + test, 30 pkgs) — PASS.
- [x] `gofmt -l .` empty — PASS.
- [x] Composite-PK multi-hub round-trip — `TestRecordProjectionsMultiHubSeqZero` PASS; mutation (revert
  schema+writer to single-PK) → FAILS with `ON CONFLICT clause does not match any PRIMARY KEY` (load-bearing).
- [x] Migration upgrade-in-place — `TestMigrationIsccIndexCompositePK` PASS (seeded row survives, version →
  `len(migrations)`==1, PK columns `[hub_id seq]`, second-hub seq-0 insert no collision); mutation (neuter
  the migration body) → FAILS (PK stays `[seq]`, second insert hits the UNIQUE constraint).
- [x] Out-of-range guard fail-closed — `TestMigrationOutOfRangeVersion` PASS (future + negative both wrap
  `errUnsupportedSchemaVersion`); mutation (remove the guard) → FAILS (future returns nil; negative would panic).
- [x] All seven `TestMigration*` PASS (the runner change is additive; the fresh-DB test sees `len(migrations)==1`).
- [x] Store leaf purity intact — `go list -deps ./internal/store | grep '^net/http$'` empty.
- [x] Scope discipline — exactly 3 non-test/doc source files (`schema.sql`, `iscc_index.go`, `sqlite.go`),
  matching `next.md`; the four reader queries untouched (Not-In-Scope honored); no gate circumvention in the
  unpushed commits.

**Oracle/conformance gate:** N/A — touches no signature / RFC-6962 / Merkle / did:web / proof path; it is
`iscc_index` DDL + `user_version` bookkeeping (advance + `next.md` both call this N/A; confirmed by inspection).

**Issues found:** Two Codex-confirmed hardening follow-ups, both filed `low` (neither blocks; both
strictly-narrower defense-in-depth / performance, not reachable today, not regressions):
1. The out-of-range guard runs AFTER `db.Exec(schemaSQL)` in `Open`, so a downgrade-from-newer-binary
   re-applies the (idempotent, `CREATE … IF NOT EXISTS`-only) baseline DDL before the reject. Hoist the
   version read+guard ahead of the schema pass.
2. The composite-PK rebuild dropped `seq`'s standalone ordering path; `RecentRecords`' `ORDER BY i.seq DESC`
   now sorts instead of walking the old rowid order. Add a `seq` index (schema + migration 0) if a populated
   monitor shows it hot.
Resolved this iteration (deleted from `issues.md`, fix verified + mutation-proven): the `iscc_index` single-
global-PK multi-hub collision `normal`, the migration-runner out-of-range `normal`, and the "migration list
still EMPTY" `normal` (now `len(migrations)==1`).

**Codex second opinion:** Two `[P2]` findings, both reviewer-CONFIRMED as real but triaged DOWN to `low`
(not blockers) and filed as issues — see above. (1) `sqlite.go:181-184` schema-version check after the
baseline DDL: confirmed the ordering, but verified `schemaSQL` is entirely `CREATE … IF NOT EXISTS` (no
DROP/ALTER/DELETE/UPDATE/INSERT — grepped), so the residual is non-destructive, not reachable on a first
migration, and the guard is strictly stronger than the prior no-guard state. (2) `schema.sql:119` lost `seq`
ordering path: confirmed it is a performance regression (correct rows, just a sort vs index walk),
negligible at 2-hub testnet scale, and `next.md` explicitly scoped this step to the PK rework + left
`RecentRecords` ordering as-is. Neither contradicts a trust-root oracle (no crypto path touched). Codex did
NOT flag any correctness defect in the migration, the guard, or the writer.

**Visual check:** n/a — no SSR surface changed (the diff is `internal/store` DDL + migration bookkeeping;
no `internal/dashboard`/`dossier`/`certificate`/`web`/template was touched).

**Next:** Both paired data-model `normal`s and the migration-empty `normal` are now closed; the
migration mechanism + first real migration are proven end-to-end. No further code-closable
migration/`iscc_index` work is queued. `define-next` should pull from the open `normal` backlog — the
M-API contract-fidelity slice 4 (the phantom `verify` `index` param + the `checkpoint` media-type doc
fixes, both `normal`) is the most self-contained next deliberate step; the dossier §1/§3 honesty `normal`s
and the realm-index Anchor design `normal` are the alternative tracks (the last two want a design pass).

**Notes:**
- The two new `low`s I filed are the natural follow-ups WHEN `Open`/the migration runner or the
  dashboard-recent path is next touched — fold the guard-hoist into any future migration work and the
  `seq` index into the next `iscc_index` schema edit (a released migration's effect must converge with the
  fresh-DB schema, so both move together).
- Net-reduced `learnings/store.md` from 191 → 181 lines this iteration (rotation budget): collapsed the
  freeze/ListViolations/RecordHubKey/AdvanceAccepted/OTS-CRUD settled-landed bullets into compact
  `settled (landed, detail in git history)` one-liners, and rewrote the two superseded notes (single-PK
  collision → "settled: composite PK"; migration-runner "go live the instant" → "settled: the guard
  landed") to the post-landing state.
- Pre-existing `.claude/context/target.md` working-tree mod (from steer `d2f259e`) is still uncommitted —
  not mine to commit (review writes only learnings/handoff/issues + minor fixes); left for `update-state`/`steer`.
- No `critical` issues open; 7 real `normal`s remain → CONTINUE (DONE requires zero open `critical`/`normal`).
