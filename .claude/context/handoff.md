## 2026-06-21 — Review of: Redress the `GET /` realm index into the Evidence-Ledger grid (DS token classes, no-JS)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance replaced the bare `<table>` on `GET /` with the Evidence-Ledger CSS-grid screen
(document-chrome masthead, bordered/shadowed ledger card, mono uppercase column-header row, per-hub grid
rows with the two-line `Domain`/`Origin` stack, coverage-since cell, observed-size cell, five-status badge,
frozen-row tint, coverage-honesty footnote), all styled through a page-scoped `<style>` over the embedded
DS `var(--*)` tokens. Zero Go source files touched (the row view-model was already complete); scope held to
the HTML asset + its golden test. Build/vet/test green, gofmt clean, both new assertions mutation-confirmed
non-vacuous. One Codex P2 (CSS-Grid ellipsis needs `min-width:0`) was confirmed and fixed in-review as a
minor layout-robustness change.

**Verification:**
- [x] `mise run check` green — all 18 packages `ok` (`go build`/`go vet`/`go test`), `gofmt -l .` empty.
- [x] `go test -count=1 -run TestDashboard ./internal/dashboard` — PASS (all HTTP-seam tests).
- [x] `go test -count=1 -run 'TestDashboardRendersEveryHub|TestDashboardLinksTokensNoCDN'` — PASS (every hub
  + both badges + coverage split rendered, body stays CDN-free).
- [x] Body contains `var(--font-sans)` + `var(--font-mono)` — confirmed at the seam and in the rendered dump.
- [x] Body contains NO `<table>` and uses `display: grid` — confirmed (negative assertion present + dump).
- [x] `GOOS=js GOARCH=wasm go build ./internal/badge ./internal/web` — exit 0 (shared leaves WASM-green).
- [x] Non-vacuity (mutation): `display: grid`→`block` fails the redress assertion; `no coverage yet`→`size 0`
  fails the coverage-honesty assertion. Both reverted byte-identical, tree clean.
- [x] Token resolution: every `var(--*)` in the template resolves in `internal/web/tokens.css` except
  `--status-error-bg`, which is correctly used WITH a literal fallback (decorative frozen-row tint only).
- [x] Scope: zero Go source files changed; only `dashboard.html` (asset) + `handler_test.go` (test).
- [x] Gate-integrity scan over the unpushed range — no `//nolint`/`t.Skip`/build-tag/swallowed-error/loosened
  gate; no deleted assertions (the diff only adds them).
- [x] Rendered-body inspection (throwaway test, removed): verified hub → `size 42 at 2023-11-14T22:13:20Z`,
  observed `42`, verified badge + check-circle; frozen hub → `no coverage yet`, observed `7`,
  `data-status="frozen"` row + octagon-x badge. Coverage honesty intact (observed size never shown as coverage).

**Issues found:** (none new) — the open `issues.md` entries are pre-existing `low`-priority architecture
deepenings in untouched packages (notecheck `out` param, dashboard/proofserve overlay duplication, Mirror
seam, proofserve `writeReadError`); none are resolved or affected by this iteration, all remain valid.

**Codex second opinion:** One finding.
- **[P2] hub cell missing `min-width:0` (`dashboard.html:217`)** — CONFIRMED. The unclassed grid `<div>`
  wrapping `.hub-name`/`.hub-origin` defaults to `min-width:auto`, so the author's no-wrap `text-overflow:
  ellipsis` rules never engage and a long domain could push the coverage/status columns out of view. Fixed
  in-review (step 9 minor fix): added `.hub-cell { min-width: 0 }` and applied the class to the wrapping div.
  Pure layout robustness, no behavior/architecture change, tests stay green. Not a gate failure (cosmetic,
  short realm domains render fine today), so verdict is PASS_WITH_NOTES, not NEEDS_WORK.

**Next:** Thread the same DS-token/font shell + Evidence-Ledger card/grid pattern into the next SSR surface —
the hub dossier (`/<domain>`) or the log-browser record list (`proofserve.serveBrowser`), which already
reuses the dashboard's `StatusSource`/`overlayStatus` shape. Reuse this page's scoped-`<style>`-over-shared-
tokens approach (page layout stays local; tokens.css stays the token layer), and carry the CSS-Grid
`min-width:0` ellipsis rule into any new ledger cell that intends to truncate.

**Notes:**
- Oracle/conformance gate N/A: pure HTML rendering of persisted store rows; no signature/RFC-6962/Merkle/
  did:web/fsck/proof path touched. go.mod/go.sum/schema byte-identical.
- The `web.md`/`dashboard.md` CDN-ban trap is still latent (the body's `https://` ban survives only because
  it renders scheme-less `Origin`, never a hub `base_url`); the redress did not change that. The day a real
  `https://` value is intentionally rendered, scope the ban to third-party origins.
- Minor cosmetic: the column header reads "Hub · domain" while the cell stacks `Domain` over `Origin`; the
  realm advertises domains only (no display name to invent), so the wording is accurate — left as-is.
