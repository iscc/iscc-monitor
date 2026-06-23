## 2026-06-23 — Close the single-record navigation back-leg: chrome identity + `← Log browser` breadcrumb + older/newer stepper + actions on `record.html`

**Done:** Made the single-record page (`GET /<domain>/log/record?index=<seq>`) a navigable
node instead of a dead end: threaded `domain, instance, operator` into the `serveRecord`
dispatch, extended the `recordData` view-model, and dressed `record.html` with the
`ISCC Monitor - Single Record.dc.html` navigation named regions — the shared chrome
instance-identity block + `verify ↗ monitor.iscc.codes` link, the `← Log browser` breadcrumb
back to the record list, the no-JS older/newer stepper (plain links, disabled `<span>` at the
ends), and the honesty-gated "Prove this record's inclusion →" / unconditional "Back to list"
actions. The no-JS dossier→record-list→single-record→**back** chain is now traversable
end-to-end (only the part-2b record-list pager remains to fully close the `critical`).

**Files changed:**
- `internal/proofserve/handler.go`: threaded `domain, instance, operator` into the
  `serveRecord` call (line 245) and its signature + doc comment; extended `recordData` with
  `Domain/Instance/Operator`, the stepper `HasOlder/OlderIndex/HasNewer/NewerIndex`, `Total`,
  and the gated `ProveInclusionID`; populated them in the `data := recordData{…}` literal
  (older when `seq > 0`, newer when `seq+1 < size`, `Total = size`, `ProveInclusionID` set only
  when `found && row.IsccID != ""`). No new store read, no constructor change, no schema/migration.
- `internal/proofserve/record.html`: ported the `.chrome-actions`/`.chrome-identity`/
  `.chrome-instance`/`.chrome-operator`/`.chrome-verify` CSS + markup verbatim from `records.html`;
  added the `.breadcrumb`, `.stepper`, and `.actions` CSS + markup; the breadcrumb back-link is the
  relative `records` target, the stepper links are `record?index=<n>` (disabled `<span>` at the
  ends), the position label is the honest 0-based `seq {{.Seq}} of {{.Total}}`, the prove-inclusion
  action is `{{if .ProveInclusionID}}` → `/inclusion/{{.ProveInclusionID}}`.
- `internal/proofserve/record_test.go` *(test)*: added `TestRecordBreadcrumbAndChromeIdentity`,
  `TestRecordStepperEnds`, `TestRecordProveInclusionHonestyGate` + a `recordBody` head/body splitter
  helper. All three are mutation-proven (see below).

**Verification:** `mise run check` → green (build + vet + test across all 30 packages).
- [x] `gofmt -l .` — empty.
- [x] `go test -count=1 -run TestRecord ./internal/proofserve` — PASS (all existing single-record
  tests still green under the threaded `serveRecord` signature).
- [x] Region test (`TestRecordBreadcrumbAndChromeIdentity`): asserts `<a href="records">← Log browser</a>`,
  the `breadcrumb-here` bare-domain crumb, the populated `chrome-identity` Instance/Operator strings,
  and the `verify ↗ monitor.iscc.codes` link; zero-value Identity falls back to "monitor instance".
- [x] Stepper end-condition test (`TestRecordStepperEnds`): seq 0 → older disabled `<span>`, newer →
  `record?index=1`; seq 7 (size-1) → newer disabled, older → `record?index=6`; mid-tree seq 4 → both
  live, no disabled span in the body region (scoped past `</style>` to dodge the CSS-literal trap).
- [x] Honesty-gate test (`TestRecordProveInclusionHonestyGate`): a projected leaf renders
  `href="/inclusion/<iscc_id>"`; the no-projection fixture (index=2) renders NO `/inclusion/` link;
  "Back to list" unconditional on both.
- [x] Import-direction: `go list -deps ./internal/proofserve | grep -qx internal/metrics` → absent.
- [x] `go.mod`/`go.sum` byte-identical (no new module dep; `dashboard.Identity` already imported).
- [x] **Mutation-proven (all 4 FAIL when reverted):** breadcrumb anchor removed → FAIL;
  `{{.Instance}}` → constant → FAIL; `HasOlder`/`HasNewer` guards → `if true` → FAIL;
  `{{if .ProveInclusionID}}` → `{{if true}}` → FAIL.
- Oracle/conformance gate: **N/A** — pure SSR chrome + view-model threading; no signature, RFC-6962,
  Merkle, did:web, fsck, or proof path touched; store stays a leaf, no new read.

**Next:** **Part-2b — the record-list pager parity** (the OTHER open half of the `critical`): rework
`records.html`'s pager to the mockup's top+bottom "seq X–Y of Z" form disabled at the ends, and drop
the Status-badge row the mockup's log browser omits. That is the last code-closable slice before the
`critical` can be closed and the human M-UI exit sign-off requested. After that, the masthead-identity
const consolidation (the tracked `low`, now duplicated 5× with this slice) is the natural cleanup.

**Notes:**
- **CSS-literal trap (recorded in http-surface.md) bit once during dev:** the mid-tree negative
  "no disabled stepper" assert initially tripped on the `.stepper-disabled` CSS rule in the head
  `<style>`. Fixed by scoping the assert to the rendered body (after `</style>`) via the new
  `recordBody` helper — same pattern the no-CDN test uses. The disabled-stepper class selector itself
  is plain (no `data-*` literal), so no quoted/unquoted attribute-selector concern here.
- **Breadcrumb back-link is `href="records"` (relative), NOT the absolute `/{{.Domain}}` the
  record-list breadcrumb uses.** Deliberate: the record list shares the `/log/` subtree with this page
  (same as the `record?index=` row links + "Back to list" action), whereas the record-list page's
  `← <domain> dossier` crumb points OUT of the subtree to the site-root dossier mount. Two different
  back-legs, two different href styles — both correct.
- **The masthead-identity fallback consts are now duplicated a 5th time** (dashboard, dossier,
  certificate, records — already 4× — and now the single-record path reuses proofserve's existing
  `resolveIdentity`, so no NEW const copy was added; the duplication count is unchanged at the package
  level). Tracked `low` (consolidation); correctly out of scope here.
- **Kept the existing field rows as-is** (per Not-In-Scope): the mockup's type-notice callouts and the
  HUB/POSITION/LOGGED field-grid re-layout were left untouched — `record.html`'s existing rows already
  render the kind + deletion note. Only the navigation regions were added.
- No backward-incompatible API change, no design deviation — the `serveRecord` signature change is
  internal to the package (the 11 `record_test.go` `Handler(...)` calls are unchanged).
