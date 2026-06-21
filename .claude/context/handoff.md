## 2026-06-21 — Review of: Self-host the DS v2 webfonts under `/_ds/fonts/...` + fold in the stable-path cache fix

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` embedded the eight latin woff2 subsets (Readex Pro 300/400/500/600/700 +
JetBrains Mono 400/500/700) plus a same-origin `@font-face` stylesheet and the SIL OFL into
`internal/web`, switched the mount from an exact path to a `/_ds/` subtree served by one `web.Handler`,
and moved every `/_ds/...` asset off `immutable` to the `tilesserve.writeBlob` shape
(`no-cache` + strong content-ETag + `If-None-Match`→304). The work is faithful to `next.md`, within
scope, the leaf stays WASM-green and pure, and it resolves the open `normal` `immutable`-on-stable-path
issue. All gates green and an independent end-to-end pass through the real `buildMux` confirms the
routing, headers, conditional-GET, method discipline, and CDN-free dashboard.

**Verification:**
- [x] `mise run check` green — all 18 packages `ok` (`go build`/`go vet`/`go test`).
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 -run 'TestTokens|TestFonts|TestDashboard' ./internal/web ./internal/dashboard` — PASS.
- [x] `GOOS=js GOARCH=wasm go build ./internal/web` — exit 0 (leaf stays WASM-green).
- [x] `go list -deps ./internal/web | grep -E 'internal/(store|metrics|logclient)'` — empty (leaf purity).
- [x] E2E through real `buildMux` (throwaway test, removed): `GET /_ds/fonts.css` → 200 text/css,
  `no-cache`, body has `@font-face` + `/_ds/fonts/`, no jsdelivr/http; `GET /_ds/fonts/readex-pro-400.woff2`
  → 200 `font/woff2` + quoted-hex strong ETag; same request with `If-None-Match` → 304 empty body;
  `POST` → 405; `GET /_ds/tokens.css` → 200 `no-cache` (not immutable); `GET /` links `/_ds/fonts.css`
  and stays CDN-free; `/metrics` + `/healthz` not shadowed; unknown `/_ds/` path → 404.
- [x] `serveFont` traversal guard probed (throwaway, removed): `..`, nested, double-`fonts/`, non-woff2
  paths all 404 before the `fs.ReadFile`.
- [x] Each `internal/web/fonts/*.woff2` → "Web Open Font Format (Version 2)" (8/8).
- [x] fonts.css references exactly the 8 committed files — 1:1, no orphan, no missing
  (`TestFontsCSSReferencesEmbeddedSubsets` enforces this at the seam).
- [x] `grep immutable internal/web/web.go` — only two doc comments explaining the policy, no Cache-Control value.
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` — empty (no dep/schema change).
- [x] Gate-integrity scan over `@{upstream}..HEAD` — no `//nolint`/`t.Skip`/build-tag/swallowed error.
  The removed `url(`/bare-`http` test substrings and the `TestTokensServedAsCSS`/`TestTokensMethodNotAllowed`
  refactors are the planned `url(`-ban narrowing + helper extraction, NOT a weakened gate: `noExternalCDN`
  still bans every third-party origin (`jsdelivr`/`http://`/`https://`/`cdn.`).

**Issues found:** (none)

**Codex second opinion:** Clean — Codex (after a long run, exit 0) reported the embedded font assets,
`/_ds/` subtree routing, dashboard link, and revalidating cache "appear consistent with the intended
design, and the test suite passes … did not find any introduced correctness, security, or maintainability
issue that warrants an inline finding." No findings to triage; matches my independent review.

**Next:** The realm-index redress — replace the dashboard `<table>` with the Evidence-Ledger grid + DS
token classes (`var(--font-sans)`/`--font-mono` now resolve to the embedded webfonts). Then thread the
token/font CSS into the log browser / dossier / record / certificate surfaces.

**Notes:**
- Scope: 2 modified non-test source files (`internal/web/web.go`, `cmd/iscc-monitor/main.go`); the
  `.css`/`.woff2`/`OFL.txt` assets, the `.html` template, and `web_test.go` are uncounted per `next.md`.
  Within ≤3. Nothing from `## Not In Scope` touched (no realm-grid restyle, no log-browser wiring, no
  Arabic subset, no dep/schema change, no store/metrics import into `internal/web`).
- Resolved the open `normal` issue (`/_ds/tokens.css` `immutable`-on-stable-path) — deleted from
  issues.md after verifying `cacheControl = "no-cache"` + strong ETag + 304 on every `/_ds/` asset.
- The mount intentionally reversed exact→subtree per the learnings guard; this is documented and
  E2E-confirmed not to shadow `/`, `/metrics`, `/healthz`, or the per-hub subtrees.
- woff2 binaries are committed/build-pinned (fetched once from jsDelivr/Fontsource at authoring time:
  readex-pro@5.2.11, jetbrains-mono@5.2.8); served bytes are never re-fetched at runtime. Not
  byte-reproducible from the repo alone, but that is the intended self-hosted-asset trade-off and the
  served bytes are fixed.
- Oracle/conformance gate N/A: pure static-asset transport + a static `<link>`; no signature/RFC-6962/
  Merkle/did:web/fsck/proof path.
- Remaining open issues are all `low` (loop-skipped): notecheck vestigial `out` param; hub-status overlay
  precedence dup; mirror write-path locality; proofserve `ErrNotExist`→404 dup. None block progress.
