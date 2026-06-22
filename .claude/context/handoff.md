## 2026-06-22 — Review of: Carry the per-record `note.timestamp` through the iscc_index projection (store layer of the §6 `· at`)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance threads the optional per-record `note.timestamp` (verbatim RFC-3339 string, both
note types) end-to-end through the schema-agnostic `iscc_index` projection — the pure fold
(`logclient.Projection.Timestamp`), the nullable `note_timestamp TEXT` column, the store write/read
structs (`ProjectionRecord`/`RecordRow`), and the follower's copy site — exactly as `next.md` asked.
Scope is exemplary (3 production files + schema DDL + 2 test files, nothing from `## Not In Scope`
touched), all gates are green, and I independently reproduced both mutations and the WASM-purity check.
The one Codex `[P1]` (no on-disk migration for the new column) is a correct description of an
intentional, codebase-wide, `next.md`-documented posture — refuted as a blocker, filed as a `normal`
backlog issue.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 28 packages ok; record tests re-run uncached PASS).
- [x] `gofmt -l .` — empty (no formatting failure).
- [x] `go test -count=1 -run TestBundleProjections ./internal/logclient` — PASS (fold reads `note.timestamp`
      per record: declaration present, deletion absent → "").
- [x] `go test -count=1 -run 'TestRecordAt|TestRecordProjections|TestListRecords' ./internal/store` — PASS
      (round-trips `NoteTimestamp` present and absent→"" through write then read; new
      `TestRecordProjectionsNoTimestamp` proves absent → true SQL NULL via the raw `sql.NullString` column).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — exit 0; `projection.go` import set is exactly
      `crypto/sha256`+`encoding/json`+`fmt`+`tessera/api` (no `time` — WASM purity preserved).
- [x] Mutation 1 (independent, reverted byte-clean) — dropping `note_timestamp` from `RecordProjections`'
      `DO UPDATE SET` → `TestRecordProjectionsIdempotent` FAILS on the stale first value.
- [x] Mutation 2 (independent, reverted byte-clean) — `Projection.Timestamp = "CONSTANT"` → `TestBundleProjections`
      FAILS on BOTH the present and the absent (→"") leaf. Tree confirmed clean after both reverts.
- [x] Wire contract — `cauldron/iscc-hub/iscc_hub/schema.py:364/443` confirm `timestamp: Timestamp | None`
      for BOTH `IsccNote` and `IsccNoteDelete` (genuinely optional, matching the absent→NULL handling).
- [x] Oracle/conformance gate — N/A. Name-only diff over the trust-root globs (`internal/proof/`,
      `logclient/verify`, `didweb`, fork/shrink/equivocation/consistency, `derive_vkey`) → empty. Pure
      JSON-fold field + nullable-TEXT round-trip; no signature/RFC-6962/Merkle/did:web/proof/fsck path.
- [x] Gate-circumvention scan over the unpushed range (`@{upstream}..HEAD`, 3 commits) — no `//nolint`,
      `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion in added lines (the one `-`
      `rows.Scan` hit is the 3-col scan being EXTENDED to 4 cols in the same reader, not a removal).

**Issues found:** (none reviewer-originated). Codex `[P1]` (no-migration) filed as a new `normal` issue
(out of scope for this slice). Existing §6 issue updated: its store prerequisite is now LANDED, so it is
narrowed to render-only.

**Codex second opinion:** One finding, triaged REFUTED-as-blocker (logged + filed as backlog):
- `[P1]` "Add a migration for note_timestamp" (`schema.sql:114`) — Codex is mechanically CORRECT:
  `store.Open` applies `schema.sql` as one `CREATE TABLE IF NOT EXISTS` pass, so the new column is never
  added to a pre-existing `iscc_index`, and the new INSERT/SELECT paths would fail `no such column` on an
  upgraded node over an old DB. But this is (a) NOT a regression of this slice — it is a pre-existing,
  codebase-wide property (zero `ALTER TABLE` / no migration framework anywhere; EVERY prior column landed
  identically, reviewer grep-confirmed), and (b) EXPLICITLY out of scope per `next.md` `## Not In Scope`
  ("no schema-versioning framework; dev DBs are ephemeral; do not add a migration path"). Adding an
  `ALTER`/migration here would introduce the project's FIRST migration mechanism as a side effect of a
  field slice — a deliberate, design-reviewed decision, not this increment's job. Does not block PASS;
  filed as a `normal` backlog issue so the real future operational hazard (in-place upgrade over a
  populated prod DB) is tracked for a dedicated migration step. The hard oracles (N/A here) are unaffected.

**Visual check:** n/a — no SSR surface changed. This is a pure store/projection slice; the certificate §6
render (the only surface that will eventually show `· at`) is explicitly the deferred follow-up and was
not touched, so there is no rendered delta to screenshot.

**Next:** The certificate §6 RECORD HISTORY render is now fully unblocked (store prerequisite landed) —
the front-of-queue, self-contained pick: wire `RecordAt`'s `NoteTimestamp` into a `HistoryRow.At` field
in `internal/certificate/handler.go`'s §6 loop and render `seq N · <at>` in `cert.html`
(`.dc.html:68`), choosing the format/relativize policy for the verbatim RFC-3339 string. Alternatives:
the `/` Checkpoint/Anchor data columns + config-driven instance identity (store/projection + config),
or the design-first WASM-verifier signature half (still a STOP/design candidate). The new no-migration
`normal` and the WASM-signature `normal` both want a deliberate design pass, not a code-only slice.

**Notes:**
- Open count after this review: 0 critical / 5 normal / 10 low. DONE still requires 0 normal. The §6
  store half closing did NOT close a normal (it narrowed the existing §6 issue to render-only); the
  no-migration issue is newly articulated, so net normal count went 4→5.
- Learnings: `store.md` gained the `note_timestamp` RFC-3339-exception bullet + a durable codebase-wide
  no-migration-posture bullet (kept package-local — it is store-mechanics, already covered at the index
  level by ADR-0007). `logclient.md` gained the `Projection.Timestamp` reads-inner-`note.timestamp`
  bullet (with the deletion-vs-declaration why and the WASM-purity import pin). `certificate.md` §6
  KNOWN-GAP bullet updated to "store prerequisite landed, render-only remaining". Nothing promoted to the
  always-loaded index. All three detail files remain within the rotation budget.
- 4 commits ahead of `origin/develop` after this review commit (update-state + define-next + advance +
  review). Pushing on PASS_WITH_NOTES. The known `Pages` workflow failure on develop is the human-blocked
  custom-domain repo-settings step (a documented `normal`), not a code regression from this slice.
