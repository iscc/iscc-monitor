# Next Work Package

## Step: Redress the `GET /` realm index into the Evidence-Ledger grid (DS token classes, no-JS)

## Advances
Directly continues the `review` handoff `**Next:**` (2026-06-21, PASS/CONTINUE):
> "The realm-index redress — replace the dashboard `<table>` with the Evidence-Ledger grid + DS token
> classes (`var(--font-sans)`/`--font-mono` now resolve to the embedded webfonts)."

It is the first SCREEN step of milestone **M-UI — Evidence Ledger frontend** (target.md), toward its
Verify criterion:
> "every SSR surface (`/` realm index, hub dossier, log-browser record list, single record,
> certificate) returns `200 text/html`, embeds the DS tokens + self-hosted fonts with **no external CDN
> URL in the body**, and is **complete with JavaScript disabled** … the coverage window
> (`monitored_since` size + RFC-3339 time, or an explicit … 'no coverage yet') shows for **every** hub
> on the index … a pre-coverage state never renders as a guarantee (ADR-0001)."

The shared DS shell (tokens + self-hosted webfonts under `/_ds/`) is now complete; this is the first
iteration that converts the *functional* `/` table into the designed Evidence-Ledger screen, per
state.md's watch-item ("the next iterations must build SCREENS, not more shared primitives").

## Goal
Replace the bare `<table>` in the dashboard template with the Evidence-Ledger ledger grid from the
design source, styled entirely through the already-embedded DS tokens (the warm canvas, the card +
ledger rows, `var(--font-sans)`/`--font-mono`, the frozen-row tint), so `GET /` looks like the
Realm Index while staying no-JS, CDN-free, and rendering every hub's badge + coverage window honestly.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/dashboard/dashboard.html` — the embedded SSR template (the `<table>` → ledger grid
    redress; the primary change). Treated as a doc/asset like the prior `.css`/`.html` work, NOT a
    source file against the ≤3 budget.
  - `internal/dashboard/handler_test.go` — extend the golden HTTP-seam test (a test file, uncounted).
- **Source files modified (the ≤3 budget): ZERO.** `handler.go` already supplies the full row
  view-model (`Domain`, `Origin`, `Status`, `Label`, `LastSize`, `HasCoverage`, `SinceSize`,
  `SinceTime`); the redress needs no new field, helper, or Go change. If you find you *must* touch
  `handler.go`, stop and re-scope — the row model is already complete.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/design/ISCC Monitor - Realm Index.dc.html` — the design source of
    truth for the grid: the document-chrome header, the realm-ledger card (`grid-template-columns`, the
    mono uppercase column-header row, per-row `name`/`domain` stack, coverage-since cell, checkpoint
    cell, status cell), the frozen-row background tint, and the coverage-honesty footnote. Port the
    *structure + token usage*, NOT the JS `<sc-for>`/`<dc-import>`/`support.js` machinery.
  - `/workspace/iscc-monitor/internal/web/tokens.css` — the embedded DS tokens you style against
    (`var(--font-sans)`, `var(--font-mono)`, `var(--surface-page)`, `var(--surface-card)`,
    `var(--text-heading)`, `var(--text-muted)`, `var(--border-default)`, `var(--radius-*)`,
    `var(--shadow-sm)`, etc.).
  - `/workspace/iscc-monitor/internal/badge/badge.html` — the `hubStatusBadge` partial markup already
    invoked per row (`class="hub-status-badge"` + `.hub-status-badge-label`); the redress keeps
    invoking it unchanged.
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` and
    `/workspace/iscc-monitor/.claude/context/learnings/web.md` — the package-local pitfalls
    (coverage-honesty render; the `noExternalCDN` ban is satisfied only because the body renders
    scheme-less `Origin`, never an `https://` `base_url`).

## Not In Scope
- **No anchor column / no anchor data.** The design grid has an "Anchor" column with a
  confirmed/pending dot, but OTS/Bitcoin anchoring is a later milestone and no anchor state is stored.
  Do NOT fabricate "confirmed"/"pending" anchor values — that would violate anchor honesty (target.md:
  "anchoring" copy is Bitcoin-only; a not-yet-anchored root renders "pending", not an error). The
  separate Bitcoin-anchor vs comparison-anchor panels are their own later M-UI sub-step. Omit the
  Anchor column entirely this step.
- **No `handler.go` change.** No new row fields, no row-number computation in Go, no new helper. (A
  cosmetic row number, if wanted, comes from a template index, not a struct field — but it is optional
  and not required by any Verify criterion; skip it if it complicates the template.)
- **No log-browser / dossier / record / certificate work.** Threading the token/font CSS into the log
  browser and building those screens are later steps in the same arc.
- **No new component classes added to `internal/web/tokens.css`.** Keep the page-specific layout in a
  scoped `<style>` block in `dashboard.html` (referencing the `var(--*)` tokens), so this step touches
  no shared asset and cannot regress another surface. tokens.css stays the token layer only.
- **No claim-lookup hero / search box.** The design's "Find evidence" ISCC-ID input belongs to the
  certificate flow (a later step); the realm index this step ships is the ledger + chrome only.

## Implementation Notes
- **The whole redress lives in `dashboard.html`.** Keep the two existing `<link>`s to `/_ds/tokens.css`
  and `/_ds/fonts.css` in `<head>` (already correct). Add a single scoped `<style>` block in `<head>`
  for the page layout, all rules expressed through the embedded `var(--*)` tokens so the type resolves
  to the self-hosted Readex Pro / JetBrains Mono and the surfaces use the DS palette. Do NOT inline a
  per-element `style=` on everything (the `.dc.html` does, but classes keep the template readable);
  classes referencing tokens are fine.
- **Port structure, drop the DC machinery.** The `.dc.html` uses `<sc-for>`, `<dc-import>`, and
  `support.js`. Replace `<sc-for list="{{ hubs }}">` with the existing Go `{{range .Hubs}}` loop, and
  replace `<dc-import name="HubStatusBadge" …>` with the existing `{{template "hubStatusBadge" .}}`
  invocation that already renders the five-status badge. Never reference `support.js` or any external
  script — the page must be complete with JS disabled.
- **Grid columns to keep (Anchor dropped):** `#`(optional, template index) · `Hub · domain` · `Coverage
  since` · `Observed size` · `Status`. Render the hub as a two-line stack: `{{.Domain}}` (the realm
  registry advertises domains only, so the domain *is* the name — do not invent a display name) over
  `{{.Origin}}` in mono. The Status cell keeps `{{template "hubStatusBadge" .}}`.
- **Coverage honesty is load-bearing (ADR-0001, dashboard.md).** Keep the existing
  `{{if .HasCoverage}} size {{.SinceSize}}{{if .SinceTime}} at {{.SinceTime}}{{end}} {{else}} no
  coverage yet {{end}}` branch verbatim in the new cell — never render the observed `last_size` as if
  it were a coverage guarantee. The fixture proves the split (verified hub: `size 42 at …`; frozen
  hub: `no coverage yet`).
- **Frozen-row tint (grayscale-safe, not hue-only).** The design tints the frozen row
  (`ledgerBg:#fff6f6`). You may add a token-based tint keyed on `data-status="frozen"` (e.g. a CSS
  attribute selector on the row), but the status is ALREADY conveyed grayscale-safe by the badge
  (icon + label + silhouette), so the tint is decorative only — never the sole status signal
  (ADR-0010 invariant 4: never hue alone).
- **CDN-free invariant (web.md trap).** `TestDashboardLinksTokensNoCDN` bans
  `jsdelivr`/`http://`/`https://`/`cdn.` in the body. The redress passes ONLY because the body still
  renders the scheme-less `Origin` (`sb0.iscc.id/log`), never an `https://` `base_url`. Do NOT
  introduce any `https://`/`http://` literal (logo URL, external link, font CDN) into the template
  body. If you want the ISCC logo or a "verify ↗" link from the chrome, use same-origin/relative
  hrefs or omit them — an external link would break this assertion.
- **Empty state.** Keep an informative empty branch (`{{else}}` of the range) for "No hubs followed."
  rendered inside the grid/card, not a bare `<td colspan>`.
- **Render-into-buffer is unchanged** (`handler.go` already buffers then writes 200), so a template
  parse/exec error is still a 500 before any 200 — do not change that path.
- **Test additions (`handler_test.go`):** Extend `TestDashboardRendersEveryHub` (or add a focused
  test) to assert the redressed structure at the seam WITHOUT over-pinning inline CSS: assert the body
  still contains every hub domain + origin, both badges (the existing `data-status`/label/silhouette
  asserts stay), `size 42` and `no coverage yet` (coverage-honesty split), and add an assertion that
  the body references the DS font tokens (e.g. `var(--font-sans)` and `var(--font-mono)` appear in the
  scoped `<style>`) so the redress is non-vacuous. Keep `TestDashboardLinksTokensNoCDN`,
  `TestDashboardMethodNotAllowed`, `TestDashboardUnknownPath`, `TestHubStatusMapping`,
  `TestOverlayStatusPrecedence`, and `TestDashboardRendersInMemoryStatus` green unchanged.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestDashboard ./internal/dashboard` passes (all dashboard HTTP-seam tests).
- `go test -count=1 -run 'TestDashboardRendersEveryHub|TestDashboardLinksTokensNoCDN' ./internal/dashboard`
  passes — the redressed body still renders every hub + both badges + the coverage-honesty split AND
  stays CDN-free (no `jsdelivr`/`http://`/`https://`/`cdn.` in the body).
- Assertion: the served `/` body contains `var(--font-sans)` and `var(--font-mono)` (the redress styles
  through DS tokens, so the type resolves to the self-hosted webfonts) — checkable mechanically by the
  new test assertion.
- Assertion: the served `/` body contains NO `<table>` element and uses the grid layout
  (`display:grid` or a grid class) — the table-to-grid redress is observable at the seam.
- `GOOS=js GOARCH=wasm go build ./internal/badge ./internal/web` still exit 0 (the shared leaves the
  template depends on stay WASM-green; the dashboard itself is not WASM-built).

## Done When
`mise run check` is green, every dashboard HTTP-seam test passes, and `GET /` serves the Evidence-Ledger
grid (no `<table>`, styled through DS tokens, self-hosted fonts) listing every realm hub with its
five-status badge and an honest coverage window, with no external CDN URL in the body.
