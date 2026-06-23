# Next Work Package

## Step: Record-list pager parity (part-2b) — top+bottom "seq X–Y of Z" pager + drop the off-mockup Status row

## Advances
Closes the **last code-closable half of the lone `critical`** ("Hub-dossier 'Browse the log →' …
record-list log browser is off-mockup", `issues.md`), whose navigation-closure clause is already MET —
only its **part-2b cosmetic pager parity** + the human M-UI exit sign-off remain. The `critical`
preempts milestone-fresh work; it is rooted in the M-UI Verify criterion:

> "the log-browser record list paginates via plain links (`?from=…[&n=…]`, no-JS), newest-first, each
> row links to its single-record page" — and the design-parity named region: "the **plain-link pager**
> (newer/older + 'seq X–Y of Z' range, disabled at the ends — the no-JS `?from=…` form of the mockup's
> buttons)" (`target.md` M-UI "log browser / record list").

The current `records.html` renders a **single bottom-only** pager labelled "showing N of M" and carries
a **Status-badge row the mockup's Log Browser omits** — this step brings both to named-region parity.

## Goal
Rework the record-list pager into the mockup's top+bottom pagers — each with newer/older affordances
disabled (rendered as a `<span>`) at the ends and a center "seq X – Y of Z" range label — and drop the
off-mockup Status-badge row, so the served no-JS HTML carries the Log Browser mockup's pager region.
This is the final code slice needed before the human M-UI exit sign-off.

## Scope
- **Modify**: `internal/proofserve/handler.go` (add precomputed `RangeTop`/`RangeBottom` fields to
  `recordsData` in `serveRecords`; no new store read — top/bottom seq come from the already-fetched
  `records` slice)
- **Modify**: `internal/proofserve/records.html` (replace the single bottom pager + the `.ledger-status`
  Status row with the top+bottom "seq X–Y of Z" pager region matching the mockup; remove the now-dead
  `.ledger-status` / `.row-label` CSS)
- **Modify (test)**: `internal/proofserve/records_test.go` (new region/golden tests for the pager parity
  + the dropped Status row; update `olderHref` + the `older &rarr;` / `showing N of M` markers in the
  EXISTING pagination tests to the reworked pager markup — these are test-only edits, off the ≤3 budget)
- **Reference**:
  - `.claude/design/ISCC Monitor - Log Browser.dc.html` lines 58–85 (the authoritative top pager / record
    list / bottom pager markup: `seq {top} – {bottom} of {hubMax}` range label, opacity-disabled at the
    ends, the append-only footnote in the bottom pager's center)
  - `.claude/context/learnings/http-surface.md` §"HTML record list at `/records`" + §"Chrome/breadcrumb/
    head dressing (part-2a)" (the no-`<table>`/CDN rule, the unquoted `[data-status=…]`/`[data-kind=…]`
    CSS-literal trap, HARDCODED-literal test grounding, the seq-cursor pager liveness rules already settled)
  - `internal/proofserve/records.html` lines 398–447 (current ledger head, `.ledger-status` Status row,
    single bottom pager, footnote) and `internal/proofserve/handler.go` lines 846–998 (the `recordsData`
    view-model + `serveRecords` cursor math — `HasNewer/NewerFrom/HasOlder/OlderFrom`, `Total`, `PageSize`
    already exist)

## Not In Scope
- The **single-record page** (`record.html`) — its navigation closure (part-2) already landed; do not
  touch `serveRecord`/`record.html`.
- The **`/<domain>/log/` checkpoint-summary landing** (`browser.html`/`serveBrowser`) — it keeps its
  Status badge; only the *record-list* (`records.html`) drops the off-mockup Status row.
- A **"Jump to sequence" input** — the mockup's `<input onKeyDown>` is JS-driven; the no-JS constraint
  wins (the plain-link `?from=…` pager IS its no-JS equivalent). Do not add a form/input; flag the
  deviation in a comment, as the existing surfaces do.
- Type-badge **exact tint/hex** + append-only **footnote wording** micro-deltas beyond moving the
  append-only line into the bottom pager's center — leave residual aesthetic deltas to the ADR-0012 visual
  pass / human exit sign-off (they are not named regions).
- Requesting the **human M-UI exit sign-off** itself — that is the next step *after* this lands.
- The masthead-identity **const consolidation** (`low`, tracked) — out of scope; would add a 4th file.

## Implementation Notes
- **Range label is pure-derived, no new store read.** The page's top seq is `records[0].Seq` (largest,
  newest-first) and the bottom seq is `records[len(records)-1].Seq` (smallest); `Total` is already in
  `recordsData`. Add `RangeTop uint64` + `RangeBottom uint64` to `recordsData` and set them inside the
  EXISTING `if len(records) > 0 {` block in `serveRecords` (alongside the `HasOlder/HasNewer` math at
  `handler.go:976-998`) — the slice is already in hand, so this adds zero queries and keeps the store a
  leaf. Render the label as `seq {{.RangeTop}} – {{.RangeBottom}} of {{.Total}}` (the mockup's order: top
  is the *larger* seq because the list is newest-first; matches mockup line 61/134
  `${logTop} – ${logBottom} of ${hubMax}`).
- **Top + bottom pager, both gated.** Port the mockup's two pager blocks (lines 58–63 top, 80–85 bottom).
  Each block has three slots: Newer (left), center label, Older (right). Reuse the EXISTING liveness flags
  — `HasNewer` → `<a href="records?from={{.NewerFrom}}&amp;n={{.PageSize}}">` else a disabled `<span>`;
  `HasOlder` → `<a href="records?from={{.OlderFrom}}&amp;n={{.PageSize}}">` else a `<span>`. The mockup's
  `opacity`/`cursor` "disabled" styling becomes the `<span>` (no href) form — the no-JS equivalent of the
  mockup's `op:.4` button, the same pattern the single-record stepper already uses (`record.html`
  older/newer stepper) and the current bottom pager already uses (`records.html:427-438`). The TOP pager's
  center is the `seq X–Y of Z` range; the BOTTOM pager's center is the **append-only footnote** ("Records
  are append-only — a deletion is itself a new entry, never a removal.", mockup line 83). Keep the
  standalone `<p class="footnote">` (`records.html:447`) for the coverage-honesty clause OR fold its
  append-only sentence into the bottom pager — either way the append-only statement must appear and the
  coverage-honesty clause ("guarantees hold only from coverage start") must NOT be dropped.
- **Drop the Status-badge row.** Remove the `.ledger-status` block (`records.html:402-406`, the
  `<span class="row-label">Status</span>` + `{{template "hubStatusBadge" .}}` partial) AND its now-unused
  `.ledger-status` (`:194`) / `.row-label` (`:202`) CSS. NOTE: `serveRecords` still computes
  `status`/`label` for the `data-status="{{.Status}}"` attribute on `.ledger` (frozen-row tinting) — KEEP
  that attribute and the `Status`/`Label` fields; only the rendered *badge row* goes. After editing, grep
  the template to confirm no remaining line references the dropped CSS classes.
- **CSS-literal trap (learnings, recurring).** `TestRecordsRendersInMemoryStatus` asserts the body
  contains NO `data-status="verified"`. Any badge-color/frozen-tint CSS that survives MUST use the
  UNQUOTED attribute selector (`[data-status=verified]`), never the quoted `[data-status="verified"]`
  form — the quoted form leaks that literal into `<style>` and falsely fails the negative assert. The
  record list already follows this; do not regress it when editing the CSS block.
- **Honesty rule (learnings always-loaded).** This surface renders no `✓`/verification — pure store-read,
  oracle gate N/A. Do not introduce any verification-shaped copy. The range/pager is derived purely from
  the page window + `Total`.
- **Test grounding = HARDCODED literals, not constants.** Per the settled record-list rule, seed the
  pager-region tests against literal seq values + the literal range string the page renders (e.g. a
  5-record mirror `seq 0..4`, full page → top pager asserts `seq 4 – 0 of 5`; with `n=2` from the newest
  → `seq 4 – 3 of 5`), NOT against view-model field reads, so a regression of the range math FAILS. Use
  the existing `buildMirror` / fixture helpers in `records_test.go`.
- **Update the existing pagination-chain tests' markers.** `TestRecordsPagination`,
  `TestRecordsOlderLinkReachesSeq0`, and the `olderHref` helper (`records_test.go:330`, keyed on
  `older &rarr;</a>`) plus `TestRecordsListsNewestFirst`'s `showing N of M` assertion all key off the
  current pager markup — both change shape in this rework. Update those markers to the reworked pager's
  anchor text so the link-chain assertions still verify the `?from=…` chain reaches seq 0 (the
  load-bearing seq-cursor-reaches-0 guard must stay green). These are test-file edits, off the ≤3 budget.

## Verification
- `mise run check` is green (build + vet + test across all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestRecords ./internal/proofserve` passes (all existing record-list tests green
  under the reworked pager, including the updated `TestRecordsPagination` + `TestRecordsOlderLinkReachesSeq0`).
- A NEW pager-parity test asserts the served `/records` body for a multi-page fixture contains the
  mockup's range label form `seq <top> – <bottom> of <total>` with hardcoded literal seqs, and contains a
  newer/older affordance BOTH above (top pager) and below (bottom pager) the record list; reverting the
  range computation (swapping top/bottom, or dropping one of the two pagers) makes it FAIL.
- A NEW test asserts the served `/records` body NO LONGER contains the dropped Status-badge row markup
  (e.g. the `row-label">Status` marker is absent), while `data-status="<overlaid>"` on `.ledger` is still
  present (frozen-tint attribute kept); re-adding the badge row makes it FAIL.
- `go list -deps ./internal/proofserve | grep -qx internal/metrics && exit 1 || true` — `internal/metrics`
  stays out of the proofserve closure (no new dep; this is a template/view-model-only change).
- `go.mod` / `go.sum` are byte-identical (no new module dependency).

## Done When
`mise run check` is green and the served `/records` HTML carries the mockup's top+bottom "seq X–Y of Z"
pager region disabled at the ends with no off-mockup Status-badge row, each new assertion mutation-proven
— closing the last code-closable half of the `critical` so only the human M-UI exit sign-off remains.
