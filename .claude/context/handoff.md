## 2026-06-21 — Redress the `GET /<domain>/log/` log browser into the Evidence-Ledger design (DS token/font shell, no-JS)

**Done:** Replaced the bare `<table>` markup in `internal/proofserve/browser.html` with the
Evidence-Ledger card screen — the two `/_ds/tokens.css` + `/_ds/fonts.css` `<link>`s, a page-scoped
`<style>` over the embedded DS `var(--*)` tokens, the `.chrome` masthead, and a bordered/shadowed
`.ledger` card holding definition-style rows (Status / Accepted size / Accepted root) plus the
proof-surface link list. Functional content (accepted `(size, root)`, the five-status badge partial,
the relative proof links, the no-checkpoint coverage-honesty state) is meaning-equivalent — a redress,
not a feature change. Zero Go source files touched.

**Files changed:**
- `internal/proofserve/browser.html` (asset): `<table>` → Evidence-Ledger card. Added the two DS `<link>`s
  and one scoped `<style>` block; `.ledger` card with a 2-column definition grid; the accepted root in a
  wrapping `<code class="row-root">` (`word-break: break-all` + `min-width:0` on the value cell, NOT
  ellipsized, so the full base64 stays selectable); the proof-surface `<ul>` re-styled but all five links
  kept relative; BOTH `{{if .HasCheckpoint}}`/`{{else}}` branches preserved (the no-checkpoint state keeps
  the literal "No accepted checkpoint yet" and still invokes the `hubStatusBadge` partial). The frozen-tint
  + badge color selectors use the UNQUOTED attribute form (`[data-status=frozen]`, valid CSS) so the
  stylesheet carries no `data-status="..."` literal — keeping the overlay-precedence test honest (see Notes).
- `internal/proofserve/browser_test.go` (test): added `TestBrowserLinksTokensNoCDN` mirroring
  `dashboard.TestDashboardLinksTokensNoCDN` — asserts `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`,
  `var(--font-sans)`, `var(--font-mono)` present; NO `<table>`; bans `jsdelivr`/`http://`/`https://`/`cdn.`.

**Verification:** `mise run check` → green (all 18 packages `ok`, `go build`/`go vet`/`go test` pass);
`gofmt -l .` empty.
- [x] `go test -count=1 -run TestBrowser ./internal/proofserve` — PASS (all 4 existing browser tests +
  the new DS-shell/no-CDN assertion).
- [x] Body contains `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`, `var(--font-sans)`, `var(--font-mono)`
  — asserted at the HTTP seam.
- [x] Body contains NO `<table>` and NO `http://`/`https://`/`cdn.`/`jsdelivr` — asserted at the seam.
- [x] `TestBrowserExposesAcceptedCheckpoint` still passes (accepted size `300`, base64 root, all relative
  proof links present in the redressed card).
- [x] `TestBrowserNoAcceptedCheckpoint` still passes (followed-but-unpolled hub → 200 "No accepted
  checkpoint yet", no fabricated `(0,"")`).
- [x] `TestBrowserRendersInMemoryStatus` still passes (overlay `unresolvable` renders through the badge;
  body carries no `data-status="verified"` — see Notes for the CSS-literal subtlety this surfaced).
- [x] New DS-shell assertion mutation-proven non-vacuous: broke `href="/_ds/fonts.css"` → test FAILS;
  swapped `var(--font-sans)` → `Arial` → test FAILS; both reverted byte-identical, tree clean (residue grep
  count 0). Tree shows only the two intended files modified.
- [x] `GOOS=js GOARCH=wasm go build ./internal/badge ./internal/web` — exit 0 (shared leaves WASM-green).
- [x] Scope: ZERO Go source files changed; only `browser.html` (asset) + `browser_test.go` (test).
  go.mod/go.sum untouched (`git status` empty for both).

**Next:** Thread the same DS-token/font shell + Evidence-Ledger card pattern into the next SSR surface —
the hub dossier (`/<domain>`) or the paginated record-list / single-record / certificate-of-inclusion
pages. Those need NEW handlers and store reads (out of scope here), so they re-engage the oracle gate
for the certificate (inclusion-proof) path. Reuse this page's scoped-`<style>`-over-shared-tokens
approach (page layout stays local; tokens.css stays the token layer) and carry the CSS-Grid `min-width:0`
rule into any ledger cell that wraps/ellipsizes.

**Notes:**
- **CSS-literal trap surfaced by the existing negative assertion (worth promoting to dashboard.md/web.md):**
  `TestBrowserRendersInMemoryStatus` asserts the body contains no `data-status="verified"` substring (it
  proves the overlay won). The DS badge-color/frozen-tint selectors in `dashboard.html` use the QUOTED form
  `[data-status="verified"]`, which would have put that literal in the rendered `<style>` and falsely failed
  the proofserve test. I used the UNQUOTED CSS attribute form (`[data-status=frozen]`,
  `.hub-status-badge[data-status=verified]`) — valid CSS for identifier values — so the only `data-status="…"`
  literals in the body are the ones the badge partial / card div emit for the ACTUAL status. The dashboard
  page does not hit this because its render test has no such negative assertion. If a future surface both
  carries this negative assertion AND uses quoted selectors, it will trip; the unquoted form is the
  load-bearing reason this redress is honest. (proofserve only — left as a package-local note for review to
  promote if it deems it cross-cutting.)
- Oracle/conformance gate N/A: pure HTML rendering of persisted store rows — no signature/RFC-6962/Merkle/
  did:web/fsck/proof path touched. go.mod/go.sum/schema byte-identical.
- CDN-ban invariant holds for the same reason as `/`: `browserData` carries no `base_url`, so nothing
  scheme-bearing renders (web.md trap). I introduced no logo URL, external link, or font CDN.
- `--status-error-bg` (frozen-card tint) is used WITH the literal fallback
  `var(--status-error-bg, rgba(245, 97, 105, 0.06))`, exactly as dashboard.html does; every other `var(--*)`
  resolves in `internal/web/tokens.css`. Hue is decorative only (ADR-0010 inv.4; badge silhouette+label
  carry status grayscale-safe).
- `serveBrowser` render-into-buffer/post-200-write-drop path is untouched (no `handler.go` change), so a
  template parse/exec error is still a 500 before any 200.
- Pre-existing `low`-priority `issues.md` entries (overlay duplication between dashboard/proofserve, etc.)
  are untouched by this HTML-only step and remain valid.
