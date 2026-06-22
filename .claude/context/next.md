# Next Work Package

## Step: Render the record-list `Type` column (per-row declaration/deletion/unknown badge)

## Advances
target.md **M-UI — Evidence Ledger frontend**, the design-parity (named-region) bar for the
**log browser / record list** surface:

> **log browser / record list** — `ISCC Monitor - Log Browser.dc.html`: … the record table
> (`Seq · Type · ISCC-ID · Logged`) with a per-row **type badge** (declaration/deletion/unknown)
> and **every row a link to its single record**; …

The `Logged` column landed last iteration; the record-list head still renders only
`Seq · ISCC-ID · Logged`. This step adds the missing **`Type`** column and per-row type badge,
completing the mockup's 4-column `Seq · Type · ISCC-ID · Logged` head — the cheapest remaining
code-only M-UI named-region slice (state.md "Next Milestone" #1, handoff `**Next:**`). It is the
last pure-code named-region slice before the backlog turns design-first or human-blocked.

## Goal
Render each `/records` row with a `Type` badge (Declaration / Deletion / Unknown record type),
mapped from the verbatim `note.$schema` (ADR-0008: the only interpretation), and add the `Type`
column header, so the served no-JS HTML carries the mockup's full 4-column record table.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/proofserve/handler.go` — give `recordsData.Records` a per-row view-model carrying the
    precomputed `Kind` label (and a stable kind key for the badge `data-*`), because `store.RecordRow`
    is a plain store value with **no `Kind` field**; precompute it in `serveRecords` via the existing
    `recordKind(row.NoteSchema)`.
  - `internal/proofserve/records.html` — add the `<span>Type</span>` header, a per-row Type badge
    cell, the 3-column→4-column grid-template update (head + row grids stay aligned), and the badge CSS
    (DS-token-only, grayscale-safe).
  - `internal/proofserve/records_test.go` — new `TestRecordsRendersTypeColumn` (test file; does not
    count against the ≤3 non-test/doc budget).
- **Reference**:
  - `.claude/context/learnings/http-surface.md` — the "HTML record list at `/records`" section
    (Type col is the named-deferred slice; head must list only columns with a data cell) and the
    "single-record page" section (the GROUND-TRUTH vacuity trap, and "match the FULL wire URI").
  - `internal/proofserve/handler.go` lines 111-115 (`schemaDeclaration`/`schemaDeletion` +
    `kindDeclaration`/`kindDeletion`/`kindUnknown` constants), 777-951 (`recordsData`, `serveRecords`,
    `recordData`, `recordKind`) — the existing wiring to extend.
  - `internal/proofserve/records.html` lines 136-198, 288-305 (the record-list grid CSS + the
    `range .Records` row markup).
  - `.claude/design/ISCC Monitor - Log Browser.dc.html` lines 67-77 (the authoritative 4-column head
    `Seq · Type · ISCC-ID · Logged` and the per-row inline-block type badge with the three labels:
    Declaration / Deletion / Unknown type).
  - `internal/store/iscc_index.go` lines 34-93 (`ProjectionRecord` + `RecordRow` fields — note both
    carry `NoteSchema`, the discriminator the badge keys on).
  - `internal/proofserve/records_test.go` lines 180-229 (`TestRecordsRendersLoggedColumn`, the
    direct-`store.Open` fixture pattern to copy) and `leafISCCID`.

## Not In Scope
- Do NOT add the styled pager buttons, the "Jump to sequence" input (it is a JS control — the no-JS
  constraint wins), the `← <hub> dossier` back-link, or the masthead instance-identity copy — those
  are separate named-region/issue items, not this slice.
- Do NOT touch `store.RecordRow` / `ProjectionRecord` / `schema.sql` — `note.$schema` is already
  stored verbatim; the kind label is a render-time projection of an existing field, computed in the
  handler, never persisted. (Adding a column would re-trigger the open `normal` DB-migration issue.)
- Do NOT change the badge's color-only semantics into the sole status signal — the **text label** is
  load-bearing and grayscale-safe; the mockup's `tone`/`toneBg` hue is decorative only.
- Do NOT alter `recordKind`'s mapping, the `Logged` column, pagination, the accepted-tree cap, or any
  proof/crypto path — this is a template + view-model render slice only (oracle gate N/A).

## Implementation Notes
- **View-model, not a store change.** `recordsData.Records` is `[]store.RecordRow` and `RecordRow`
  has no `Kind`. The template's `{{range .Records}}` can only read `RecordRow` fields, so introduce a
  small handler-local row VM (e.g. `recordRowVM struct { store.RecordRow; Kind string; KindKey string }`
  embedding the store row, or a flat struct copying `Seq`/`IsccID`/`NoteSchema`/`NoteTimestamp` plus
  `Kind`/`KindKey`). In `serveRecords`, after `ListRecords`, map each row through `recordKind(row.NoteSchema)`
  to fill `Kind` (the human label) and a stable lowercase `KindKey` (`declaration`/`deletion`/`unknown`)
  for the badge `data-kind` selector. Keep `recordKind` as the single mapping site — `serveRecord`
  already calls it, so reuse it; do not duplicate the switch.
- **Labels reuse the existing constants.** `kindDeclaration="Declaration"`, `kindDeletion="Deletion"`,
  `kindUnknown="Unknown record type"` (handler.go:113-115) already match the mockup's
  Declaration/Deletion (the mockup's third label is "Unknown type"; our existing constant
  "Unknown record type" is the in-repo wording — keep the existing constant, do not introduce a new
  literal). The `KindKey` is a SEPARATE short token for CSS/`data-kind`, distinct from the display
  label — derive it in the handler, do not parse it back out of the label.
- **Grid alignment.** The head (`.records-head`) and each row (`.record-row`) are CSS grids. Adding
  `Type` means changing the record-list grids from `120px 1fr 160px` to a 4-column template
  (e.g. `120px 130px 1fr 160px`) on BOTH `.records-head` and `.record-row` so the header columns stay
  aligned with the data cells (the `Logged` review explicitly checked this). The `.ledger-status` row
  is independent (`160px 1fr`) and must NOT change.
- **Badge CSS is DS-token-only, grayscale-safe.** Render the badge as an inline-block with a text
  label; hue (if any) keys on `[data-kind=…]` like the existing `.hub-status-badge[data-status=…]`
  block, and must use the UNQUOTED attribute-selector form (`[data-kind=declaration]`, valid CSS for
  identifier values) — the quoted form would emit a `data-kind="…"` literal into the `<style>` and
  could trip a future negative body assert (the cross-cutting CSS-literal trap from
  http-surface.md/web.md). Every property must resolve to an existing `var(--*)` token in
  `internal/web/tokens.css` (the `Logged` review verified all this page's tokens resolve — reuse
  `--font-mono`/`--text-2xs`/`--text-xs`/spacing/radius tokens already used here).
- **No-CDN / no-`<table>` invariants hold.** This page already has `TestRecordsLinksTokensNoCDN`
  (bans `http://`/`https://`/`cdn.`/`jsdelivr`) and renders as a CSS grid `<ul>`, not a `<table>`.
  The new markup must add no external URL and no `<table>`.
- **Vacuity trap (learnings, the open `low` on `record_test.go`).** The single-record label test went
  vacuous because it returned the `schemaDeclaration`/`schemaDeletion` *constants* and `recordKind`
  switched on the SAME constants — reverting both constants left the suite green. Avoid that here:
  seed the new test's `ProjectionRecord` rows with **HARDCODED literal** `note.$schema` URIs
  (`"http://purl.org/iscc/schema/iscc-note-0.8.0.json"`,
  `"http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"`, and a garbage string like
  `"iscc-note-future-9.9.9"` for the unknown case), NOT the package constants — so reverting a
  constant makes the test FAIL.
- **Test grounding.** Copy `TestRecordsRendersLoggedColumn`'s direct-`store.Open` + `RecordProjections`
  + `AdvanceFollowState` fixture (buildMirror seeds uniform schemas, so build the fixture directly).
  Seed three accepted leaves: one declaration URI, one deletion URI, one unknown schema; advance
  `LastSize` past all three; assert the body contains all three rendered labels ("Declaration",
  "Deletion", "Unknown record type") AND the `<span>Type</span>` header. ADR-0008: the verbatim
  `note.$schema` still renders in the ISCC-ID cell's `.record-schema` line; the Type badge is
  additive, not a replacement.

## Verification
- `mise run check` is green (build + vet + `go test ./...` all pass; `gofmt -l .` empty outside
  `cauldron/`).
- `go test -count=1 -run TestRecords ./internal/proofserve` passes (existing record-list tests +
  the new `TestRecordsRendersTypeColumn`).
- The new test asserts the served `/records` body contains `<span>Type</span>` (header) and the three
  literal labels `Declaration`, `Deletion`, `Unknown record type`, each driven from a HARDCODED
  literal `note.$schema` URI (constant revert → test FAIL).
- Mutation check (run + revert): deleting the per-row Type badge cell from `records.html` makes
  `TestRecordsRendersTypeColumn` FAIL; restore byte-clean (`git diff` clean).
- `go list -deps ./internal/store | grep -E 'net/http|proofserve'` is empty (store stays a leaf —
  no store change).
- The diff adds no `http://`/`https://`/`cdn.`/`jsdelivr` to the body and no `<table>`
  (`TestRecordsLinksTokensNoCDN` + `TestRecordsRendersInMemoryStatus` still pass).

## Done When
`mise run check` is green and `go test -count=1 -run TestRecords ./internal/proofserve` passes with a
mutation-proven `TestRecordsRendersTypeColumn`, so the served `/records` no-JS HTML renders the
mockup's full 4-column `Seq · Type · ISCC-ID · Logged` head with a per-row declaration/deletion/unknown
type badge.
