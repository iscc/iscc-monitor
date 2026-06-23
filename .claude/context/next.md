# Next Work Package

## Step: Close the single-record navigation back-leg — chrome identity + `← Log browser` breadcrumb + older/newer stepper + actions on `record.html`

## Advances
The lone **`critical`** issue (`[human]`, "Hub-dossier 'Browse the log →' lands on a dead-end … the
record-list log browser is off-mockup"), whose navigation-closure clause is **not yet met**: per
`target.md` M-UI **"Navigation closure"** — *"The `/` → hub dossier → log browser → single record →
certificate chain is fully traversable with JavaScript disabled: forward links **and** `←` breadcrumb
back-links both present, so … no surface is a dead end."* `state.md` verifies the single-record page
`record.html` carries **zero `<a>` nav links** (confirmed: `grep -c '<a ' record.html` == 0), so the
dossier→record-list→single-record→**back** chain is broken at the single-record back-leg. This step
builds the `ISCC Monitor - Single Record.dc.html` named regions on `record.html` (breadcrumb, stepper,
actions, chrome identity), the load-bearing remaining half of the navigation-closure clause. Part-2b
(the record-list pager) is the other open half and is deferred to the next slice (Not In Scope).

## Goal
Make the single-record page a navigable node, not a dead end: add the shared chrome instance-identity
block + `verify ↗` link, the `← Log browser` breadcrumb back to the record list, the no-JS older/newer
record stepper (plain links disabled at the ends), and the "Prove this record's inclusion →" /
"Back to list" actions — so a no-JS reader can traverse the chain forward and back.

## Scope
- **Modify**:
  - `internal/proofserve/handler.go` — thread `domain, instance, operator` into the `serveRecord`
    dispatch (line 245: `serveRecord(w, r, st, f, hubID, domain, instance, operator, statuses)`) and
    into `serveRecord`'s signature; extend the `recordData` view-model (line 1022) with the new fields
    (`Domain, Instance, Operator`, the stepper `HasOlder/OlderIndex/HasNewer/NewerIndex`, the gated
    `ProveInclusionID`, and `Total`). Populate them in the `data := recordData{…}` literal (line 1136).
  - `internal/proofserve/record.html` — add the chrome instance-identity block + `verify ↗` link (port
    byte-for-byte from `records.html` lines 368-385 + their `.chrome-actions`/`.chrome-identity`/
    `.chrome-verify` CSS), the `← Log browser` breadcrumb, the older/newer stepper, and the two action
    links, per the mockup named regions below.
  - `internal/proofserve/record_test.go` *(test — not in the ≤3 non-test budget)* — add region tests
    (breadcrumb, stepper, identity, actions) and the stepper end-condition + honesty-gate tests.
- **Reference**:
  - `.claude/design/ISCC Monitor - Single Record.dc.html` — the authoritative mockup (lines 41-90: the
    breadcrumb, stepper, head, type notices, field grid, actions).
  - `.claude/context/learnings/http-surface.md` — §"HTML single-record page at `/record`" (the
    `record.html` durable traps) + §"HTML record list at `/records`" / "Chrome/breadcrumb/head dressing
    (part-2a)" (the breadcrumb must be the ABSOLUTE site-root `href="/{{.Domain}}"`; bare-`{{.Domain}}`,
    no fabricated display name; the unquoted `[data-status=…]`/`[data-kind=…]` CSS-literal trap).
  - `internal/proofserve/records.html` (the part-2a sibling — copy its `.chrome-actions` /
    `.chrome-identity` / `.chrome-verify` / `.backlink` markup + CSS verbatim so the surfaces stay
    byte-identical chrome; and its pager disabled-`<span>` pattern lines 427-438 for the stepper ends).
  - `internal/certificate/handler.go:109` (`const PathPrefix = "/inclusion/"`) — the
    "Prove this record's inclusion →" target is `/inclusion/<iscc_id>` (realm-root, NOT under `/log/`).

## Not In Scope
- **Part-2b record-list pager parity** (the `records.html` top+bottom "seq X–Y of Z" pager disabled at
  the ends, dropping the Status-badge row) — that is the OTHER open half of the critical and the next
  slice; do not touch `records.html` here beyond reading it for the chrome/pager-pattern port.
- **The masthead-identity const consolidation** (the tracked `low`, instanceFallback/operatorFallback
  duplicated 4×) — keep the byte-identical-copy pattern; do not refactor a shared resolve leaf here.
- **The proof-bundle / certificate `§6` flow itself** — only the link to the existing certificate route
  is added; no crypto/proof/Merkle path changes (oracle gate stays N/A).
- Touching `cmd/iscc-monitor/main.go` or the `Handler` constructor — the part-2a `Handler` already
  takes `domain` + `dashboard.Identity` and resolves `instance, operator` once; this change is internal
  to the `serveRecord` dispatch (the 11 `record_test.go` `Handler(...)` calls stay unchanged).
- **The mockup's type-notice callouts** (the deletion / unknown-type info boxes, mockup lines 64-75) and
  the field-grid restructure (HUB / POSITION / LOGGED rows) — `record.html`'s existing field rows
  already render the kind + deletion note; the navigation regions are this step's scope, not a full
  field-grid re-layout. Leave the field rows as they are.

## Implementation Notes
- **No constructor change.** `Handler(st, hubID, domain, statuses, id)` already resolves
  `instance, operator := resolveIdentity(id)` once (handler.go:231) and passes `domain` to
  `serveRecords`. Pass those same three (`domain, instance, operator`) into the `serveRecord` call at
  line 245 — a one-line edit; the 11 existing `record_test.go` calls that build `Handler(...)` stay
  unchanged (they already pass `"sb0.iscc.id"` + `dashboard.Identity{}`, which `resolveIdentity` maps to
  the static fallback copy, so non-identity assertions stay green).
- **Breadcrumb (no-JS back-leg, the load-bearing fix).** `← Log browser` links to the record list:
  `href="records"` (relative — the record list shares the `/log/` subtree, same as the `record?index=`
  row links). The mockup also shows a `/ <hub name>` trailing crumb; render the bare `{{.Domain}}` (no
  fabricated display name — same constraint-win the records/dossier heads make; flag it in a comment).
- **Older/newer stepper = plain links, disabled at the ends.** The mockup's JS buttons become no-JS
  `<a>`s: **older** → `record?index=<Seq-1>`, present only when `Seq > 0` (`HasOlder`); **newer** →
  `record?index=<Seq+1>`, present only when `Seq+1 < size` (`HasNewer`, the accepted-tree ceiling
  `LastSize`, already in scope at handler.go:1081 as `size`). At an end, render a disabled `<span>` (the
  records.html pager pattern at lines 427-438). The position label: `Seq` is 0-based — show
  `seq {{.Seq}} of {{.Total}}` (`Total` = `size`), keeping the existing single-record `seq` wording
  rather than the mockup's 1-based "record N of M" (honest 0-based seq). Carry `OlderIndex`/`NewerIndex`
  as precomputed `uint64` so the template does no arithmetic.
- **"Prove this record's inclusion →" MUST be honesty-gated.** The certificate route is keyed on the
  ISCC-ID (`/inclusion/<iscc_id>`), but a leaf can have NO projection (`HasProjection=false`) or an
  empty/unknown id. Set `ProveInclusionID` only when `HasProjection && IsccID != ""`; in the template,
  render the "Prove this record's inclusion →" action ONLY `{{if .ProveInclusionID}}` (href
  `/inclusion/{{.ProveInclusionID}}`), so a no-id / no-projection leaf does not link to a certificate it
  cannot produce. This is the recurring honesty rule (MEMORY "Verdict-UI honesty recurring gap";
  learnings.md): never render an affordance asserting a capability the data does not support.
  "Back to list" is unconditional: `href="records"` (same as the breadcrumb target).
- **Chrome port + CSS-literal trap.** Port `records.html`'s `.chrome-actions`/`.chrome-identity`/
  `.chrome-verify`/`.backlink` markup AND their CSS rules verbatim into `record.html` (it currently
  lacks the identity block + verify link — it has only the brand half of the chrome). `record.html`
  already uses the UNQUOTED `[data-status=frozen]` form (lines 110, 224-238) — keep any new selector
  unquoted so a future negative `data-status="…"` body assert stays honest (http-surface.md trap).
- **Buffer-then-200 unchanged + store stays a leaf.** `serveRecord` already renders into a `bytes.Buffer`
  then writes 200 (handler.go:1150-1159); the new fields are pure view-model derived from data already in
  scope (`seq`, `size`, `row.IsccID`, `found`), no new store read, no schema/migration. `go.mod`/`go.sum`
  stay byte-identical (`dashboard.Identity` is already imported via the part-2a chrome).
- **No-CDN body ban.** The `verify ↗ monitor.iscc.codes` host is same-federation (the `.codes` verifier
  app), so the existing no-CDN body ban tolerates it (records.html already carries it past
  `TestRecordsLinksTokensNoCDN`); do NOT regress `TestRecordLinksTokensNoCDN`.
- **Oracle/conformance gate is N/A** — pure SSR chrome + view-model threading; no signature, RFC-6962,
  Merkle, did:web, or proof path is touched. State the N/A in the advance.

## Verification
- `mise run check` is green (build + vet + test across all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestRecord ./internal/proofserve` passes (all existing single-record tests
  still green under the threaded `serveRecord` signature).
- A new region test asserts the served `/record?index=<mid-seq>` body contains the breadcrumb
  `← Log browser` with `href="records"`, the chrome instance-identity binding (a populated
  `dashboard.Identity` renders its `Instance`/`Operator` strings), and the `verify ↗ monitor.iscc.codes`
  link — and is **mutation-proven**: removing the breadcrumb link FAILS; replacing the `{{.Instance}}`
  binding with a constant FAILS.
- A stepper end-condition test asserts: at `index=0` the older affordance is a disabled `<span>` (no
  older `record?index=` link) and newer is a live `<a href="record?index=1">`; at the topmost in-tree
  seq (`size-1`) newer is disabled and older is live — reverting the `HasOlder`/`HasNewer` guards FAILS.
- An honesty-gate test asserts: a leaf WITH a projected id renders the `href="/inclusion/<iscc_id>"`
  "Prove this record's inclusion →" action, and a leaf with NO projection (the existing
  `TestRecordRendersWithoutProjection` fixture, `index=2`) renders NO `/inclusion/` link — removing the
  `{{if .ProveInclusionID}}` gate FAILS the no-link assertion.
- `go list -deps ./internal/proofserve | grep -qx internal/metrics && exit 1 || true` (proofserve still
  does not pull `internal/metrics` into its dep closure — the load-bearing import-direction check).

## Done When
`record.html` carries the `ISCC Monitor - Single Record.dc.html` navigation named regions — chrome
instance identity + `verify ↗`, the `← Log browser` breadcrumb, the no-JS older/newer stepper disabled
at the ends, and the honesty-gated "Prove this record's inclusion →" / "Back to list" actions — every
new region is mutation-proven by a test, and `mise run check` is green; the no-JS
dossier→record-list→single-record→**back** chain is then traversable end-to-end (only the part-2b
record-list pager remains to fully close the `critical`).
