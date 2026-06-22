## 2026-06-22 — Log-browser record list: render the mockup's `Logged` column from `RecordRow.NoteTimestamp`

**Done:** Wired the already-landed `store.RecordRow.NoteTimestamp` into the `/records` log-browser
record list as the mockup's `Logged` column — a verbatim RFC-3339 render (ADR-0008, never re-formatted)
with an honest em-dash fallback for a record carrying no timestamp, plus a `Seq · ISCC-ID · Logged`
column-header row above the list. Template-only production change (no handler/struct/store change).

**Files changed:**
- `internal/proofserve/records.html`: added a trailing `160px` grid column to `.record-row` and a new
  `.records-head` column-header row (lists only the columns that have a data cell — `Seq · ISCC-ID ·
  Logged`; the full 4-col `Type` head stays deferred per Not-In-Scope); added a `.record-logged` data
  cell `{{if .NoteTimestamp}}{{.NoteTimestamp}}{{else}}&mdash;{{end}}` and its DS-token style.
- `internal/proofserve/records_test.go`: added `TestRecordsRendersLoggedColumn` — builds a self-contained
  store (one row with `NoteTimestamp: "2026-06-21T12:34:56Z"`, one with `""`), advances LastSize past
  both, and asserts the served body contains the hardcoded literal timestamp, the `&mdash;` empty
  fallback, and the `<span>Logged</span>` column header.

**Verification:** `mise run check` → green (all 28 packages ok, build + vet + test).
- [x] `go test -count=1 -run TestRecords ./internal/proofserve` → PASS (all existing record-list tests +
      the new Logged-column test).
- [x] New test asserts the literal `2026-06-21T12:34:56Z`, the `&mdash;` empty fallback, and the `Logged`
      header — all present.
- [x] Mutation (reverted byte-clean): removing the `.record-logged` cell from `records.html` makes
      `TestRecordsRendersLoggedColumn` FAIL (missing timestamp + missing em-dash); restoring passes. Tree
      `git diff`-clean of the mutation after revert.
- [x] `gofmt -l .` empty outside `cauldron/`.
- [x] Store leaf invariant: `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'`
      empty (store untouched, still a leaf).
- [x] Oracle/trust-path: name-only diff over `internal/proof/`, `logclient/verify`, `didweb`, `index`,
      `notecheck`, `go.mod`, `go.sum` is empty. Pure HTML render of a persisted leaf read; oracle gate N/A.
- [x] Scope: 1 production file + 1 test file (within the ≤3 non-test budget; nothing from Not In Scope —
      no Type badge, no store change, no single-record-page change, no time-zone math).

**Next:** The deferred sibling slice on this same surface is the **`Type` column / per-row type badge**
(`declaration`/`deletion`/`unknown`) that completes the mockup's full 4-column `Seq · Type · ISCC-ID ·
Logged` head. It needs a per-row `recordKind(NoteSchema)` precomputation: the parent `recordsData` carries
no per-row kind today and `RecordRow` is a plain store value, so a small view-model wrap in
`handler.go`'s `serveRecords` (map each `RecordRow` to a row VM carrying the kind label) is the natural
shape — reuse the existing `recordKind` + `schemaDeclaration`/`schemaDeletion` constants from
`record.go`, and seed the test with a HARDCODED literal schema URI per the http-surface.md non-vacuous
rule (a constant-vs-constant test goes vacuous). That keeps the store a leaf and stays code-only.

**Notes:**
- Header lists only columns with a data cell (`Seq · ISCC-ID · Logged`) on purpose — the head never
  promises a `Type` column with no cell, per the Implementation Notes. The em-dash is emitted as the
  `&mdash;` entity (html/template passes the literal through), so the test asserts the entity string.
- Kept the production change template-only — no humanization helper added (the mockup's
  `YYYY-MM-DD HH:MM UTC` is the intentional ADR-0008-deferred JS-side cosmetic, NOT a named-region item).
- Used the UNQUOTED `[data-status=frozen]` selector form already in the file (no new `data-status="…"`
  literal added to `<style>`), so `TestRecordsRendersInMemoryStatus`'s negative `data-status="verified"`
  assert stays honest. No-CDN body ban holds (no `http://`/`https://`/`cdn.`/`jsdelivr` introduced).
