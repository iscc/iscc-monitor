# Next Work Package

## Step: Hub dossier skeleton — `GET /<domain>` Evidence-Ledger page (status + coverage honesty), no-JS

## Advances
M-UI (Evidence Ledger frontend) Verify criterion:

> every SSR surface (`/` realm index, **hub dossier**, log-browser record list, single record,
> certificate) returns `200 text/html`, embeds the DS tokens + self-hosted fonts with **no external CDN
> URL in the body**, and is **complete with JavaScript disabled**… the coverage window
> (`monitored_since` size + RFC-3339 time, or an explicit "coverage just started" / "no coverage yet")
> shows for **every** hub on the index + **dossier** and a pre-coverage state never renders as a
> guarantee (ADR-0001).

This is the first of the still-open M-UI screens (state.md M-UI "Still open: **hub dossier**
(`/<domain>`…)"). It is also the review handoff's explicit `**Next:**` ("Thread the same DS-token/font
shell + Evidence-Ledger card pattern into the next SSR surface — the hub dossier (`/<domain>`)").
Skeleton-first: this step lands the dossier *page + route* (masthead, five-status badge, coverage
honesty); the categorically-distinct **frozen Exhibit** — which needs a NEW `store` violation read that
does not yet exist — is the next sub-step (see `## Not In Scope`).

## Goal
Stand up a per-hub dossier page at the bare `GET /<domain>` (no `/log`) — a new `internal/dossier` leaf
that renders one hub's status badge + coverage window in the Evidence-Ledger card pattern, wired into
the mux. It closes the bulk of the dossier Verify criterion (200 text/html, DS tokens + fonts, no-CDN,
no-JS, coverage honesty for that hub) and gives the next iteration a place to add the frozen Exhibit.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/dossier/handler.go` — handler + view-model + the
    `StatusSource`/`overlayStatus`/`hubStatus`/`coverageTime` shape, ported from
    `internal/dashboard/handler.go` (see Not In Scope on why the duplication is accepted here).
- **Modify**:
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — add `Domain string` to `hubRoute`, set it in
    `registerHubs`, and mount `dossier.Handler(st, r.HubID, m)` at the exact path `"/" + r.Domain`
    inside `mirrorHandler`'s per-route loop.
- **Create (asset/template — not a Go source file, like the prior `.html` work)**:
  - `/workspace/iscc-monitor/internal/dossier/dossier.html` — the Evidence-Ledger card template
    (port the `<head>` DS-shell + page-scoped `<style>` from `internal/dashboard/dashboard.html` and
    the single-hub card body from `internal/proofserve/browser.html`).
- **Create (test — uncounted)**:
  - `/workspace/iscc-monitor/internal/dossier/handler_test.go`
- **Modify (doc — uncounted)**:
  - `/workspace/iscc-monitor/CLAUDE.md` — add a `GET /<domain>` bullet to the route list (after the
    `GET /` line, before `GET /<domain>/log/`).
- **Reference**:
  - `/workspace/iscc-monitor/internal/dashboard/handler.go` — the EXACT pattern to mirror: handler
    shape, the `StatusSource` interface, `overlayStatus`/`hubStatus` precedence, `buildRows` +
    `badge.Label` `!ok → label = status` fallback, `coverageTime`, render-into-`bytes.Buffer`-then-200,
    the `template.Must(...).Parse(badge.Source)` init idiom.
  - `/workspace/iscc-monitor/internal/dashboard/dashboard.html` — the DS `<head>` (two `<link>`s +
    page-scoped `<style>` over `var(--*)` tokens, `.chrome` masthead).
  - `/workspace/iscc-monitor/internal/proofserve/browser.html` + `browser_test.go` — the single-hub
    `.ledger` card body layout + the `TestBrowserLinksTokensNoCDN` no-CDN assertion shape to copy.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` lines 165-244 (`buildMux`/`mirrorHandler`/
    `hubHandler`) + 246-271 (`registerHubs`, where `e.Domain` is available) + 42-50 (`hubRoute`).
  - `/workspace/iscc-monitor/internal/store/hubs.go` (`ListHubs` / `HubSummary` / `CoverageInfo`) — the
    read this handler filters by `hubID`; `HubSummary` carries `.HubID`, `.Domain`, `.Origin`,
    `.Active`, `.LastSize`, `.Frozen`, `.Coverage`.
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md`,
    `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md`,
    `/workspace/iscc-monitor/.claude/context/learnings/badge.md`,
    `/workspace/iscc-monitor/.claude/context/learnings/web.md` — Read all four before writing: overlay
    precedence, the CSS-literal `data-status` trap, the badge `.Label` precompute, the
    `noExternalCDN` / scheme-less-origin rule.

## Not In Scope
- **The frozen Exhibit** (violation kind + detected-at + "do not trust new state", non-dismissable,
  categorically-distinct markup, ADR-0006). It needs a NEW store read (`store.ListViolations(hubID)`
  over the `violations` table — only `RecordViolation` exists today, no read) plus the Exhibit block in
  the template. That is the next sub-step. This skeleton renders the `frozen` *status badge* honestly
  via the overlay, but does NOT fabricate or render violation detail.
- **De-duplicating `overlayStatus`/`hubStatus`** into `internal/badge` (the open `low` issue "Hub-status
  overlay precedence is duplicated"). This step copies the dashboard/proofserve shape a third time
  DELIBERATELY — consolidation is its own tracked step, not a prerequisite here. Do not refactor the two
  existing copies in this step (it would balloon scope and touch unrelated packages).
- **Any new `store` method, schema change, or write path.** Reuse `ListHubs` filtered by `hubID`.
- **The record list / single record / certificate / proof-bundle / anchor panels** — later M-UI steps.
- **ETag / Cache-Control / conditional-GET** on the dossier — not a Verify criterion (matches `/`).

## Implementation Notes
- **Route / mux.** Add `Domain string` to `hubRoute` (main.go:48-50) and set it from `e.Domain` in
  `registerHubs` alongside the existing `HubID`/`Origin`. In `mirrorHandler`'s per-route loop, ALSO
  mount `mux.Handle("/"+r.Domain, dossier.Handler(st, r.HubID, m))` — the bare-domain path
  (e.g. `/sb0.iscc.id`) is a distinct EXACT pattern, more-specific than and disjoint from the existing
  `/<domain>/log/` subtree (`"/" + r.Origin + "/"`), so `http.ServeMux` keeps both. Because the mount is
  an exact pattern (no trailing slash), the mux routes ONLY that path here — so the handler needs NO
  in-handler path guard (unlike `dashboard`, which owns the catch-all `/`). Still guard
  `r.Method != GET → 405`.
- **Handler.** Mirror `dashboard.Handler` but for ONE hub: signature `Handler(st *store.Store, hubID
  int64, statuses StatusSource) http.Handler`. On GET, call `st.ListHubs(ctx)`, find the `HubSummary`
  whose `.HubID == hubID`. The binary always registers the hub before mounting, so a not-found is a real
  store inconsistency → 500 (NOT a 404). Build a single view-model: status via `overlayStatus`, label via
  `badge.Label` with the `!ok → label = status` fallback, coverage via the `coverageTime` helper. Render
  into a `bytes.Buffer`, then `WriteHeader(200)` + `buf.WriteTo(w)` (post-200 write-drop). Use
  `html/template` (auto-escape); parse page + `badge.Source` once at init via the
  `template.Must(template.New("dossier").Parse(pageTemplate))` then `template.Must(t.Parse(badge.Source))`
  idiom dashboard uses.
- **Template.** Port `dossier.html` from `dashboard.html`'s `<head>` (the two `/_ds/tokens.css` +
  `/_ds/fonts.css` `<link>`s as literals — templates can't read Go consts — the page-scoped `<style>`
  over `var(--*)` tokens, the `.chrome` masthead) and `browser.html`'s single-card body: a
  bordered/shadowed `.ledger` card with definition rows — Domain, Origin, Status
  (`{{template "hubStatusBadge" .}}`), Coverage (the honest "size N at <RFC3339>" / "no coverage yet"
  split), Observed size — plus a relative link to that hub's log browser (`/{{.Origin}}/`). No
  `<table>`; no `http://`/`https://`/`cdn.`/`jsdelivr` in the body.
- **CSS-literal trap (learnings/http-surface.md).** If the test asserts a NEGATIVE `data-status="X"` to
  prove the overlay won, the badge color/tint selectors in the `<style>` MUST use the UNQUOTED CSS
  attribute form (`[data-status=verified]`), not `dashboard.html`'s QUOTED `[data-status="verified"]` —
  the quoted form emits that literal into the rendered `<style>` and falsely fails the negative assert.
  Copy `browser.html`'s unquoted selectors. (If your test does not include a negative `data-status`
  assert, either form is safe — but prefer the unquoted form for consistency with `browser.html`.)
- **Coverage honesty (Correctness rule, ADR-0001).** Never render the observed `last_size` as a coverage
  guarantee. `HasCoverage` false → literal "no coverage yet"; true → "size N at <RFC3339>". Copy
  `coverageTime` verbatim ("" when `!c.Set || c.Since.IsZero()`).
- **Purity / closure.** Depend on `internal/store` + `internal/badge` + the `StatusSource` *interface*
  (NOT `internal/metrics`) — `*metrics.Registry` satisfies it structurally, exactly like dashboard.
  Keep `net/http`/`internal/dossier` out of `store`/`badge` (they are leaves; do not introduce a cycle).
- **`--status-error-bg` token caveat.** If you reuse the frozen-row tint, use it WITH the literal
  fallback `var(--status-error-bg, rgba(245, 97, 105, 0.06))`, exactly as `dashboard.html` does — that
  token is not defined in `internal/web/tokens.css` (dashboard.md rule; the badge silhouette+label carry
  status grayscale-safe, hue is decorative).
- **Oracle/conformance gate is N/A** here: pure HTML render of persisted store rows + the in-memory
  overlay — no signature/RFC-6962/Merkle/did:web/fsck/proof path. go.mod/go.sum/schema must stay
  byte-identical. State this in the advance.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/dossier` passes (new package).
- A new `internal/dossier/handler_test.go` HTTP-seam test (httptest, no socket) asserts, for a fixture
  store with the hub registered: `GET /<domain>` returns `200` + `Content-Type: text/html`; the body
  contains `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`, `var(--font-sans)`; the body contains the
  hub's `Domain` and its `hubStatusBadge` markup (`class="hub-status-badge"` + the rendered label); the
  body contains NO `<table>` and NO `http://`/`https://`/`cdn.`/`jsdelivr`.
- Coverage-honesty assertion: a hub WITHOUT coverage renders "no coverage yet" (no fabricated
  size+time); a hub WITH coverage renders its `size N at <RFC3339>` (mirror the dashboard fixture split).
- A non-GET (`POST /<domain>`) returns `405`.
- The new DS-shell / coverage assertion is mutation-proven non-vacuous (advance: temporarily break a
  `<link>` href or the coverage branch → the assert FAILS; revert byte-identical, tree clean).
- `go list -deps ./internal/store ./internal/badge | grep -E 'net/http|internal/dossier'` is empty
  (store/badge stay leaves; no import cycle introduced).
- `go test -count=1 ./cmd/iscc-monitor` passes — the new `hubRoute.Domain` + `/<domain>` mount does not
  break `buildMux` routing (the `/<domain>/log/` subtree AND the `/<domain>` exact path both resolve).

## Done When
`GET /<domain>` serves a `200 text/html` Evidence-Ledger dossier page for the registered hub — DS tokens
+ self-hosted fonts linked, no CDN URL in the body, no-JS, the five-status badge via the overlay, and an
honest coverage window — with all Verification criteria passing and `mise run check` green.
