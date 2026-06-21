## 2026-06-21 — Review of: Redress the `GET /<domain>/log/` log browser into the Evidence-Ledger design (DS token/font shell, no-JS)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance replaced the bare `<table>` in `internal/proofserve/browser.html` with the
Evidence-Ledger card screen — the two `/_ds/tokens.css` + `/_ds/fonts.css` `<link>`s, a page-scoped
`<style>` over the embedded DS `var(--*)` tokens, the `.chrome` masthead, a bordered/shadowed `.ledger`
card with definition rows (Status / Accepted size / Accepted root) and the proof-surface link list, plus
a new `TestBrowserLinksTokensNoCDN` HTTP-seam assert. Zero Go source files touched; scope held to the
HTML asset + its test. Functional content (accepted `(size, root)`, the five-status badge partial, all
five relative proof links, the no-checkpoint coverage-honesty state) is meaning-equivalent. Build/vet/test
green, gofmt clean, the new assertion independently mutation-confirmed non-vacuous, Codex found nothing.

**Verification:**
- [x] `mise run check` — green (all 18 packages `ok`; `go build`/`go vet`/`go test` pass).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -count=1 -run TestBrowser ./internal/proofserve` — PASS (all 4 existing browser tests +
  the new DS-shell/no-CDN assertion).
- [x] Body contains `href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`, `var(--font-sans)`,
  `var(--font-mono)` — asserted at the HTTP seam; test passes.
- [x] Body contains NO `<table>` and NO `http://`/`https://`/`cdn.`/`jsdelivr` — asserted at the seam.
- [x] `TestBrowserExposesAcceptedCheckpoint` still passes (accepted size, base64 root, all five relative
  proof links present in the redressed card).
- [x] `TestBrowserNoAcceptedCheckpoint` still passes (followed-but-unpolled hub → 200 "No accepted
  checkpoint yet"; no fabricated `(0,"")`, ADR-0001 coverage honesty intact).
- [x] `TestBrowserRendersInMemoryStatus` still passes (overlay `unresolvable` renders through the badge;
  body carries no `data-status="verified"` — the unquoted attribute-selector form keeps this honest).
- [x] New assertion mutation-proven non-vacuous (independently re-run): broke `href="/_ds/fonts.css"`
  href → FAIL; re-introduced a `<table>` → FAIL; both reverted byte-identical, `git status` clean,
  green again after revert.
- [x] `GOOS=js GOARCH=wasm go build ./internal/badge ./internal/web` — exit 0 (shared leaves WASM-green).
- [x] Token resolution: every `var(--*)` in `browser.html` resolves in `internal/web/tokens.css` EXCEPT
  `--status-error-bg`, which is correctly used WITH the literal fallback (decorative frozen tint only).
- [x] Scope: ZERO Go source files changed; only `browser.html` (asset) + `browser_test.go` (test).
  go.mod/go.sum untouched.
- [x] `.Status`/`.HasCheckpoint`/`.Size`/`.Root` referenced by the template all exist on `browserData`
  (the new `data-status="{{.Status}}"` references the pre-existing `.Status` field — no handler change).
- [x] Gate-integrity scan over the unpushed range (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/loosened gate; the diff only ADDS assertions, deletes none.

**Issues found:** (none new). The four open `issues.md` entries are pre-existing `low`-priority
architecture deepenings in untouched packages (notecheck `out` param, dashboard/proofserve overlay
duplication, Mirror seam, proofserve `writeReadError`); none resolved or affected this iteration, all
remain valid.

**Codex second opinion:** Completed (exit 0, clean): "The HTML redress preserves the existing rendered
checkpoint/status/proof-link behavior while adding local design-system styling and tests. I did not find
any introduced functional regressions or blocking issues." No findings to triage.

**Next:** Thread the same DS-token/font shell + Evidence-Ledger card pattern into the next SSR surface —
the hub dossier (`/<domain>`) or the paginated record-list / single-record / certificate-of-inclusion
pages. Those need NEW handlers and store reads (out of scope here) and re-engage the oracle gate for the
certificate (inclusion-proof) path. Carry the new CSS-literal trap (unquoted `data-status` selectors when
a surface also carries a negative `data-status="X"` assert — see `learnings/http-surface.md`) and the
scoped-`<style>`-over-shared-tokens approach into the next dressed surface.

**Notes:**
- Oracle/conformance gate N/A: pure HTML rendering of persisted store rows — no signature / RFC-6962 /
  Merkle / did:web / fsck / proof path touched. go.mod/go.sum byte-identical.
- Minor cosmetic (left as-is, not a fix): the `.ledger-count` class is reused from `dashboard.html` where
  it holds a hub count; here it holds the page's descriptive subtitle. Pure class-name reuse, no behavior
  impact — not worth a Go/asset change.
- Learnings: collapsed the landed `serveBrowser` notes in `learnings/http-surface.md` into one `settled:`
  summary + three durable traps (mux mount, coverage-honesty status mapping, the new CSS-literal trap),
  net-reducing the file 173 → 164 lines (back under the ~150-ish budget pressure). The CSS-literal trap
  stays package-detail (SSR-surface-specific, not durable-true if proofserve were deleted) rather than
  promoted to the index.
- Pushing `develop` (remote configured) on this PASS.
