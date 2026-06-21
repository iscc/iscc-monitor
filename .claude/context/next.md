# Next Work Package

## Step: Redress the `GET /<domain>/log/` log browser into the Evidence-Ledger design (DS token/font shell, no-JS)

## Advances
M-UI — Evidence Ledger frontend. Verify criterion:

> every SSR surface (`/` realm index, hub dossier, log-browser record list, single record, certificate)
> returns `200 text/html`, embeds the DS tokens + self-hosted fonts with **no external CDN URL in the
> body**, and is **complete with JavaScript disabled** (content in the served HTML, not script-gated)

This step closes the **log-browser** slice of that "every SSR surface … embeds the DS tokens + fonts,
no external CDN URL" requirement. Today `/` is the only screen wired to the DS shell; the log browser
at `/<domain>/log/` still renders a bare `<table>` and links no token/font CSS (state.md M-UI "Still
open": "`proofserve.serveBrowser` still links no token/font CSS"). It is the handoff's explicit
`**Next:**` (2026-06-21 PASS_WITH_NOTES) and the lowest-risk next screen because its view-model
(`browserData`, `overlayStatus`, the `hubStatusBadge` partial) is already complete — only the template
changes.

## Goal
Replace the bare `<table>` markup in `internal/proofserve/browser.html` with the Evidence-Ledger card
screen (DS-token-styled scoped `<style>`, `<link>`s to `/_ds/tokens.css` + `/_ds/fonts.css`, masthead +
bordered/shadowed card), so the log browser matches the `/` index. The functional content (accepted
`(size, root)`, the five-status badge, the proof-surface links, the no-checkpoint coverage-honesty
state) stays equivalent in meaning — this is a redress, not a feature change.

## Scope
- **Create**: (none)
- **Modify**:
  - `/workspace/iscc-monitor/internal/proofserve/browser.html` — the embedded log-browser template (the
    `<table>` → ledger card redress; the primary change). Treated as a doc/asset like the prior
    `.html`/`.css` work, NOT a source file against the ≤3 budget.
  - `/workspace/iscc-monitor/internal/proofserve/browser_test.go` — add the DS-shell + no-CDN HTTP-seam
    assertions (a test file, uncounted).
- **Source files modified (the ≤3 budget): ZERO.** `handler.go` already supplies the full view-model
  (`browserData{Status, Label, HasCheckpoint, Size, Root}`) and the `hubStatusBadge` partial is already
  associated into `browserTmpl`; the redress needs no new field, helper, or Go change. If you find you
  *must* touch `handler.go`, stop and re-scope.
- **Reference**:
  - `/workspace/iscc-monitor/internal/dashboard/dashboard.html` — the canonical Evidence-Ledger page to
    mirror: the two `<link>`s, the scoped `<style>` over `var(--*)` tokens, the `.chrome` masthead, the
    `.ledger` bordered/shadowed card, the `.hub-status-badge[data-status=…]` color block, the
    `--status-error-bg` literal-fallback frozen tint, and the `.hub-cell { min-width: 0 }` ellipsis fix.
  - `/workspace/iscc-monitor/internal/dashboard/handler_test.go` — `TestDashboardLinksTokensNoCDN`
    (~191-210, the `href="/_ds/tokens.css"` + banned-substring asserts) and the
    `var(--font-sans)`/`display: grid`/no-`<table>` asserts (~128-135): mirror these for the browser.
  - `/workspace/iscc-monitor/internal/web/tokens.css` — the available `var(--*)` token names (every
    dashboard token resolves here EXCEPT `--status-error-bg`, which needs a literal fallback).
  - `/workspace/iscc-monitor/internal/badge/badge.html` — the `hubStatusBadge` partial already invoked
    via `{{template "hubStatusBadge" .}}`; the redress keeps invoking it unchanged in both branches.
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` — the scoped-`<style>`-over-shared-
    tokens rule, the `--status-error-bg` literal-fallback rule, the CSS-Grid `min-width:0` trap.
  - `/workspace/iscc-monitor/.claude/context/learnings/web.md` — the CDN-ban trap: the body's `https://`
    ban survives only because the page renders scheme-less values; do NOT render a hub `base_url` here.
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — the `serveBrowser` section
    (status mapping, post-200 write-drop, `html/template`, the overlay shape) so the redress keeps that
    posture.

## Not In Scope
- Adding the **paginated record list** / single-record page / certificate of inclusion — those are later
  M-UI screens with new handlers and store reads; this step touches only the existing log-browser
  template. (They remain in `## Not In Scope` so the same arc continues, not an unrelated refactor.)
- **No `handler.go` change.** No new `browserData` fields, no new helper, no route change. The
  `<link>` paths are literals in the template (templates can't read Go consts), exactly as in
  `dashboard.html`.
- Refactoring the duplicated `overlayStatus`/`hubStatus` out of dashboard + proofserve (open `low`
  issue) — leave the Go handler untouched; this step is HTML + test only.
- Adding ETag/Cache-Control/conditional-GET to `/<domain>/log/` (not a Verify criterion).
- **No new component classes added to `internal/web/tokens.css`.** Keep the page-specific layout in a
  scoped `<style>` block in `browser.html` (referencing the `var(--*)` tokens), so this step touches no
  shared asset and cannot regress `/` or any other surface. tokens.css stays the token layer only.
- No anchor panel / no proof-bundle assembler / no tier-1-vs-tier-2 affordance — those are later M-UI
  sub-steps (the proof-bundle assembler re-engages the oracle gate).

## Implementation Notes
- **The whole redress lives in `browser.html`.** Add the two `<link rel="stylesheet">` tags
  (`/_ds/tokens.css`, `/_ds/fonts.css`) to `<head>` and one scoped `<style>` block, all rules expressed
  through the embedded `var(--*)` tokens so the type resolves to the self-hosted Readex Pro / JetBrains
  Mono and the surfaces use the DS palette. Port the `.chrome` masthead and `.ledger` card chrome from
  `dashboard.html` (reuse the `body { font-family: var(--font-sans); color: var(--text-body);
  background: var(--surface-page); }` base and the `.hub-status-badge[data-status]` color block verbatim).
- **No `<table>`.** Replace the existing `<table>`/`<tr>`/`<th>` block with a card layout — definition-
  style rows (Status / Accepted size / Accepted root) inside the bordered `.ledger` card. A small
  `display: grid` for the label/value rows mirrors the index and satisfies the no-`<table>` intent; the
  browser shows ONE hub's checkpoint, not a multi-hub grid, so trim the multi-row grid down.
- **The accepted root is long.** Keep it in a `<code>`/mono cell that wraps or scrolls (e.g.
  `word-break: break-all` or `overflow-wrap`), NOT an ellipsized cell — the full base64 root must remain
  readable/selectable. Only add `min-width: 0` (dashboard.md CSS-Grid trap) if you choose to ellipsize
  some cell; for a wrapping `<code>` it is not needed.
- **Coverage honesty (ADR-0001) must survive the redress.** Keep BOTH template branches: `{{if
  .HasCheckpoint}}` renders the `(size, root)` + proof links; `{{else}}` renders the no-checkpoint state.
  `TestBrowserNoAcceptedCheckpoint` asserts the literal string `No accepted checkpoint yet` — preserve it
  exactly. Never fabricate a `(0,"")` checkpoint as a guarantee.
- **The five-status badge partial is already wired** via `{{template "hubStatusBadge" .}}` over
  `browserData` (carries `.Status` + precomputed `.Label`); keep that invocation in BOTH branches
  (`HasCheckpoint` and the no-checkpoint state both invoke it today — see browser.html lines 16 and 37).
  Do NOT re-derive the label or inline the SVG — the partial owns rendering (badge.md single-source rule).
- **Proof-surface links stay relative and stay present.** The existing `<ul>` of relative links
  (`entries?index=0`, `inclusion?iscc_id=…`, `consistency?from=0`, `verify?iscc_id=…`, `checkpoint`) is
  what `TestBrowserExposesAcceptedCheckpoint` asserts — keep all of them, relative, in the
  `HasCheckpoint` branch. Re-style them (a list inside the card) but do not drop or absolutise them.
- **CDN-free invariant (web.md trap).** The body must contain no `http://`, `https://`, `cdn.`, or
  `jsdelivr`. `browserData` carries no `base_url`, so this holds today — do NOT introduce a logo URL,
  external link, or font CDN. Same-origin/relative hrefs only.
- **`--status-error-bg` has no token** — if you reuse the dashboard's frozen-row/exhibit tint, use it
  WITH the literal fallback `var(--status-error-bg, rgba(245, 97, 105, 0.06))`, exactly as
  `dashboard.html` does (dashboard.md rule; ADR-0010 inv.4 — hue is never the sole status signal, the
  badge silhouette+label carries it grayscale-safe).
- **Render-into-buffer is unchanged** (`serveBrowser` already buffers then writes 200, then post-200
  write-drop), so a template parse/exec error is still a 500 before any 200 — do not touch that path.
- **Test additions (`browser_test.go`), mirror `TestDashboardLinksTokensNoCDN`:** assert the rendered
  body contains `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`, `var(--font-sans)`, `var(--font-mono)`;
  assert it contains NO `<table>`; assert it bans `jsdelivr`/`http://`/`https://`/`cdn.`. Do NOT weaken
  the existing `TestBrowserExposesAcceptedCheckpoint` / `TestBrowserRendersInMemoryStatus` /
  `TestBrowserNonGET` / `TestBrowserNoAcceptedCheckpoint` — they must still pass against the redress.
- **Mutation check (non-vacuity, advance must perform).** Temporarily mutate the redressed template
  (e.g. `var(--font-sans)` → a literal, or delete a `<link>` href) and confirm the new assertion FAILS;
  then revert byte-identical and leave the tree clean. Record this in the review handoff.
- Oracle/conformance gate is **N/A** for this step: pure HTML rendering of persisted store rows — no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path is touched. go.mod/go.sum/schema must stay
  byte-identical.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass, `gofmt -l .` empty).
- `go test -count=1 -run TestBrowser ./internal/proofserve` passes (all four existing browser tests plus
  the new DS-shell / no-CDN assertion).
- The rendered `GET /` log-browser body contains `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`,
  `var(--font-sans)`, and `var(--font-mono)` — asserted at the HTTP seam.
- The rendered body contains NO `<table>` substring and contains NO `http://` / `https://` / `cdn.` /
  `jsdelivr` substring — asserted at the HTTP seam.
- `TestBrowserExposesAcceptedCheckpoint` still passes: the redressed body still contains the accepted
  size, the base64 accepted root, and every relative proof link.
- `TestBrowserNoAcceptedCheckpoint` still passes: the followed-but-unpolled hub renders the literal
  "No accepted checkpoint yet" state (200, not a fabricated checkpoint).
- New DS-shell assertion is mutation-proven non-vacuous (mutating the template fails it; revert leaves
  the tree byte-identical).
- `GOOS=js GOARCH=wasm go build ./internal/badge ./internal/web` still exit 0 (the shared leaves the
  template depends on stay WASM-green; proofserve itself is not WASM-built).

## Done When
The log browser at `GET /<domain>/log/` renders the Evidence-Ledger card screen wired to the DS token +
self-hosted-font shell with no external CDN URL in the body, all existing browser tests plus the new
DS-shell/no-CDN HTTP-seam assertion pass, and `mise run check` is green.
