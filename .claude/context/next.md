# Next Work Package

## Step: Log-browser record list — render the mockup's `Logged` column from `RecordRow.NoteTimestamp`

## Advances
M-UI Evidence-Ledger frontend, the **design-parity (named-region)** bar for the log-browser record list.
target.md (per-surface landmark regions): *"**log browser / record list** — `ISCC Monitor - Log
Browser.dc.html`: … the record table (`Seq · Type · ISCC-ID · Logged`) … and **every row a link to its
single record**"*. The served record list today renders only a 2-column `seq · (id+schema)` grid; the
mockup's table head is the 4-column `Seq · Type · ISCC-ID · Logged` (mockup lines 67-68,
`grid-template-columns:90px 130px 1fr 150px`). This step closes the **`Logged`** column of that named
region by wiring the already-landed `store.RecordRow.NoteTimestamp` into the rendered row.

No `critical`/`normal` issue preempts milestone work here. This is the cheapest remaining code-only M-UI
slice (the store field already exists; no new store read) that the review handoff named:
*"a reasonable code-only next pick: wire `RecordRow.NoteTimestamp` into the log-browser record-list
`Logged` column (`internal/proofserve`) — a separate SSR surface reusing the landed store field, no new
store read."*

## Goal
Render each record-list row's logged time (the verbatim inner `note.timestamp`) as the mockup's `Logged`
column, with an honest fallback when a record carries no timestamp — advancing the log-browser surface
toward its mockup's named-region table layout.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/proofserve/records.html` — the only non-test production file; add the `Logged` cell to the
    record-row grid and a column-header row above the list.
  - `internal/proofserve/records_test.go` — add the HTTP-seam golden test (test file, not counted toward
    the ≤3 non-test budget).
- **Reference**:
  - `.claude/design/ISCC Monitor - Log Browser.dc.html` (lines 67-72: the `Seq · Type · ISCC-ID ·
    Logged` table head + row grid `90px 130px 1fr 150px`; lines 108-113: the mockup humanizes the time
    as `YYYY-MM-DD HH:MM UTC` — a JS-side cosmetic, NOT a required named-region item).
  - `.claude/context/learnings/http-surface.md` — the "HTML record list at `/records`" + "HTML
    single-record page" sections: verbatim-render posture, the `[data-status=…]` UNQUOTED-selector trap,
    buffer-then-200, store-leaf invariant, the no-CDN body ban.
  - `.claude/context/learnings/web.md` reference via the index — the `noExternalCDN` third-party-origin
    ban (same-origin `url(` OK).
  - `internal/store/iscc_index.go` lines 81-94 (`RecordRow` — `NoteTimestamp` is the verbatim optional
    `note.timestamp` RFC-3339 string, `""` for a NULL/empty column).
  - `internal/proofserve/handler.go` lines 785-795 (`recordsData` already carries
    `Records []store.RecordRow`, so each row's `.NoteTimestamp` is template-reachable with NO new struct
    field or handler change).

## Not In Scope
- **The `Type` column / per-row type badge** (`declaration`/`deletion`/`unknown`). The mockup's table has
  four columns; this step lands `Logged` only. A per-row Type badge needs per-row `recordKind(NoteSchema)`
  precomputation (the parent `recordsData` carries no per-row kind, and `RecordRow` is a plain store value
  the store package owns), so it is its own follow-up step. Render the existing verbatim `note.$schema`
  (`.record-schema`) where it is today; do not add a type badge now.
- Touching `internal/store` — `RecordRow.NoteTimestamp` already lands and is read by `ListRecords`; do
  NOT add a store column, read, or method (store stays a leaf; no schema change).
- Touching the single-record page (`record.html` / `serveRecord` / `recordData`) — it has no `Logged`
  field today; adding it there is a separate single-record-parity step.
- Time-zone math or RFC-3339 parse/re-format in Go (ADR-0008 verbatim posture: do not interpret the
  timestamp; see Implementation Notes for the humanization decision).
- The `← <hub> dossier` back-link / chrome instance-identity parity on this surface (separate
  named-region slice under the M-UI cross-cutting nav-closure requirement).

## Implementation Notes
- **No handler/struct change is required.** `recordsData.Records` is `[]store.RecordRow`, and
  `RecordRow.NoteTimestamp` is exported, so the template can render `{{.NoteTimestamp}}` inside the
  `{{range .Records}}` block directly. Keep the production change template-only. (If a humanization helper
  feels cleaner, a tiny pure func in `handler.go` is acceptable and still ≤3 non-test files — but prefer
  template-only.)
- **Verbatim, not parsed (ADR-0008 + learnings render-values-verbatim posture).** The record list already
  renders `IsccID` and `NoteSchema` verbatim. Render `NoteTimestamp` the same way — emit the stored
  RFC-3339 string as-is. Do NOT `time.Parse`/re-format it; the mockup's `YYYY-MM-DD HH:MM UTC`
  humanization is a JS-side cosmetic and (per state.md) intentional ADR-0008-deferred polish, NOT a
  required named-region item. The named-region bar is satisfied by the `Logged` cell *existing and showing
  the logged time*, not by a particular format.
- **Honest empty fallback (coverage/honesty discipline).** `NoteTimestamp == ""` (NULL column) is the
  common case — the `buildMirror` fixture seeds projections with NO timestamp (handler_test.go:144-147),
  so most fixture rows are `""`. Render an explicit placeholder for the empty case, mirroring the existing
  `{{if .IsccID}}…{{else}}(no iscc_id){{end}}` / `(no schema)` pattern — e.g.
  `{{if .NoteTimestamp}}{{.NoteTimestamp}}{{else}}—{{end}}`. Never render a fabricated or zero time.
- **Grid layout — render only the columns that have data cells.** The current `.record-row` is
  `grid-template-columns: 120px 1fr` (seq + a stacked id/schema cell). Add a trailing `Logged` grid column
  (e.g. `120px 1fr 160px`; mono, `--text-2xs`/`--text-xs`, `var(--text-muted)`). Add a column-header row
  above the `<ul class="records">` listing exactly the columns you render (e.g. `Seq · ISCC-ID · Logged`)
  so the head never promises a `Type` column that has no data cell — the full 4-column `Seq · Type ·
  ISCC-ID · Logged` head waits for the deferred Type-badge step. Keep it a CSS grid (no `<table>`; the
  existing `TestRecordsLinksTokensNoCDN` bans `<table>`).
- **DS-token + no-CDN discipline (learnings/web.md, http-surface.md):** every new rule uses `var(--*)`
  tokens; the body must carry NO `http://`/`https://`/`cdn.`/`jsdelivr` (TestRecordsLinksTokensNoCDN bans
  all four). Use the UNQUOTED `[data-status=frozen]` attribute-selector form already in this file — do
  NOT add a quoted `data-status="…"` literal into the `<style>` (it would trip
  `TestRecordsRendersInMemoryStatus`'s negative `data-status="verified"` assert).
- **The Logged-column test needs a fixture with a real `NoteTimestamp`.** `buildMirror` seeds none, so
  build a small self-contained store the way `TestRecordsCeilingHidesUnacceptedLeaves` does: `store.Open`
  on a `t.TempDir()`, `UpsertHub`, `RecordProjections` with at least one row carrying
  `NoteTimestamp: "2026-06-21T12:34:56Z"` AND one row with `NoteTimestamp: ""`, then `AdvanceFollowState`
  past both seqs so they are accepted. Assert the body contains the HARDCODED literal timestamp string
  (not a code constant — so the gate is non-vacuous), the empty-fallback placeholder, and the `Logged`
  column header.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty
  outside `cauldron/`).
- `go test -count=1 -run TestRecords ./internal/proofserve` passes (all existing record-list tests + the
  new `Logged`-column test).
- The new test asserts the served `/records` body contains the literal seeded timestamp
  `2026-06-21T12:34:56Z`, the empty-timestamp fallback placeholder for a row with no timestamp, and the
  `Logged` column header.
- Mutation (advance proves non-vacuous, then reverts): removing the `Logged` cell from `records.html`
  makes the new test FAIL; restoring it passes. Tree `git diff`-clean after revert.
- `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` stays empty (store untouched,
  still a leaf).
- No oracle/conformance path touched: the name-only diff over `internal/proof/`, `logclient/verify`,
  `didweb`, fork/shrink/equivocation/consistency, `derive_vkey` is empty (pure HTML render of a persisted
  leaf read; go.mod/go.sum/schema.sql byte-unchanged).

## Done When
`mise run check` is green, `go test -run TestRecords ./internal/proofserve` passes including the new
`Logged`-column golden test (verbatim seeded timestamp + honest empty fallback + `Logged` header
rendered), the cell is mutation-proven non-vacuous, and the store stays a leaf with no schema or
trust-path change.
