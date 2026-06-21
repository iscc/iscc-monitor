## 2026-06-21 — Redress the `GET /` realm index into the Evidence-Ledger grid (DS token classes, no-JS)

**Done:** Replaced the bare `<table>` in `dashboard.html` with the Evidence-Ledger ledger grid ported
from the design source — document-chrome masthead, a bordered/shadowed ledger card with a mono
uppercase column-header row and per-hub grid rows (`domain`/`origin` stack, coverage-since cell,
observed-size cell, five-status badge), the decorative frozen-row tint, and the coverage-honesty
footnote — all styled through a page-scoped `<style>` block referencing only the embedded DS `var(--*)`
tokens so the type resolves to the self-hosted webfonts. Zero Go source files touched (the row
view-model already had every field); the Anchor column, claim-lookup hero, and all DC `<sc-for>`/
`<dc-import>`/`support.js` machinery were dropped per scope.

**Files changed:**
- `internal/dashboard/dashboard.html`: `<table>` → CSS-grid Evidence-Ledger; added a scoped `<style>`
  block (chrome, ledger card, `.ledger-row` grid, frozen-row `data-status` tint, badge coloring keyed
  on `data-status`, footnote) all via DS tokens; kept the two `/_ds/...` `<link>`s, the coverage-honesty
  branch verbatim, the `{{template "hubStatusBadge" .}}` invocation, and an informative empty state.
- `internal/dashboard/handler_test.go`: extended `TestDashboardRendersEveryHub` with the
  coverage-honesty split (`no coverage yet` for the frozen hub) and three non-vacuous redress markers
  (`var(--font-sans)`, `var(--font-mono)`, `display: grid`) plus a `<table>`-absence assertion.

**Verification:** `mise run check` → green, all 18 packages `ok` (`go build`/`go vet`/`go test`);
`gofmt -l .` empty.
- `go test -run TestDashboard ./internal/dashboard` → PASS (all HTTP-seam tests incl. the unchanged
  `TestDashboardLinksTokensNoCDN`, `TestDashboardMethodNotAllowed`, `TestDashboardUnknownPath`,
  `TestHubStatusMapping`, `TestOverlayStatusPrecedence`, `TestDashboardRendersInMemoryStatus`).
- `go test -run 'TestDashboardRendersEveryHub|TestDashboardLinksTokensNoCDN' ./internal/dashboard` →
  PASS — redressed body renders every hub + both badges + the coverage split AND stays CDN-free.
- Per-criterion: body contains `var(--font-sans)` + `var(--font-mono)` ✓; body contains NO `<table>`
  and uses `display: grid` ✓; coverage split (`size 42` / `no coverage yet`) ✓; no
  `jsdelivr`/`http://`/`https://`/`cdn.` in body ✓ (grepped `dashboard.html` directly → none).
- `GOOS=js GOARCH=wasm go build ./internal/badge ./internal/web` → exit 0 (shared leaves stay
  WASM-green).
- Throwaway full-render check (removed): the served `/` body is the chrome + ledger card grid;
  verified hub shows `size 42 at 2023-11-14T22:13:20Z`, frozen hub shows `no coverage yet` with
  `data-status="frozen"` on the row and the octagon-x badge.

**Next:** Thread the same DS token/font shell + the Evidence-Ledger card/grid pattern into the next SSR
surface — the hub dossier (`/<domain>`) or the log-browser record list (`proofserve.serveBrowser`),
which already reuses the dashboard's `StatusSource`/`overlayStatus` shape. Reuse this page's scoped-
`<style>`-over-shared-tokens approach (keep page layout local, tokens.css stays the token layer).

**Notes:**
- Scope held: 2 modified files, both uncounted against the ≤3 source budget (`dashboard.html` is a
  doc/asset like prior `.css`/`.html` work; `handler_test.go` is a test). ZERO Go source files changed —
  `handler.go` already supplied the full row model, so no new field/helper was needed.
- Badge coloring decision: the badge partial emits `class="hub-status-badge"`/`-label` but tokens.css
  has no badge CSS, so before this step the badge was unstyled. I added decorative `data-status`-keyed
  coloring in the scoped block (e.g. frozen/unverified → `--status-error-text`). This is grayscale-safe
  (ADR-0010 invariant 4): the silhouette + label remain the load-bearing status signal; color is never
  the sole signal. The frozen-row background tint uses a literal `rgba(245,97,105,0.06)` fallback
  (`var(--status-error-bg, …)`) since tokens.css has no surface-tint token — also decorative-only.
- Design fidelity trade-offs (intentional, per scope): the per-row hub line shows `{{.Domain}}` as the
  name (the realm registry advertises domains only — no display name to invent) over `{{.Origin}}` in
  mono; the design's row `#` column was skipped (optional, no Verify criterion); the `ledger-count`
  span carries the explanatory subtitle rather than a fabricated "N hubs followed" total.
- CDN-free invariant preserved: the only host-like strings in the body are the scheme-less origins
  (`sb0.iscc.id/log`); no `https://` literal was introduced (no logo/external-link/CDN). The
  `web.md`/`dashboard.md` trap (ban false-positives the day a real `base_url` is rendered) is untouched
  — still not rendered here.
- Oracle/conformance gate N/A: pure HTML rendering of persisted store rows; no signature/RFC-6962/
  Merkle/did:web/fsck/proof path. go.mod/go.sum/schema byte-identical.
