## 2026-06-21 — Self-host the DS v2 webfonts under `/_ds/fonts/...` + fold in the stable-path cache fix

**Done:** Embedded the eight latin woff2 subsets (Readex Pro 300/400/500/600/700 + JetBrains Mono
400/500/700) plus an `@font-face` stylesheet and the SIL OFL license into `internal/web`, served them
under a new `/_ds/` subtree, switched every `/_ds/...` asset (tokens.css, fonts.css, woff2) from the
`immutable` cache policy to the `tilesserve.writeBlob` shape (`no-cache` + strong content ETag +
`If-None-Match`→304), and linked `/_ds/fonts.css` from the dashboard `<head>`. The monitor UI now
renders in Readex Pro / JetBrains Mono with zero runtime CDN dependency; `internal/web` stays a pure
WASM-green stdlib leaf.

**Files changed:**
- `internal/web/web.go` — embed `fonts.css` + the `fonts` dir; one `Handler()` mounted at the `/_ds/`
  subtree dispatches tokens.css / fonts.css / `fonts/*.woff2` by path, sets explicit content types
  (`text/css` / `font/woff2`), and applies the `no-cache`+strong-ETag+304 `writeAsset` policy (the
  `immutable` const is gone). Added `Prefix`, `FontsCSSPath` consts (kept `TokensPath`).
- `internal/web/fonts.css` (new) — same-origin `@font-face` shell, ported from the design `fonts.css`
  with CDN URLs rewritten to `/_ds/fonts/...`; the Arabic-300 Readex subset omitted (English UI), noted
  in the header comment.
- `internal/web/fonts/*.woff2` (new, 8 files, ~140 KB) — the latin subsets fetched from the Fontsource
  jsDelivr CDN, each verified as "Web Open Font Format (Version 2)".
- `internal/web/fonts/OFL.txt` (new) — SIL OFL 1.1 with both families' copyright headers (the OFL bodies
  were byte-identical between the two npm packages).
- `internal/web/web_test.go` — narrowed the `url(` ban to "no external CDN" (`jsdelivr`/`http://`/
  `https://`/`cdn.`); added fonts.css, woff2-binary, all-8-subsets, css↔embed cross-check, 304-conditional,
  no-cache+strong-ETag, and 405 assertions.
- `cmd/iscc-monitor/main.go` — mount changed from exact `web.TokensPath` to the `web.Prefix` subtree;
  `buildMux` doc comment updated.
- `internal/dashboard/dashboard.html` — added `<link rel="stylesheet" href="/_ds/fonts.css">`.

**Verification:** `mise run check` → green (all 18 packages `ok`; `go build`/`go vet`/`go test` pass);
`gofmt -l .` empty. Per criterion:
- `go test -run 'TestTokens|TestFonts|TestDashboard'` → PASS (web + dashboard).
- `GOOS=js GOARCH=wasm go build ./internal/web` → OK; `go list -deps` internal closure is only
  `internal/web` (no store/metrics/logclient) → leaf purity holds.
- E2E through real `buildMux` (throwaway test, removed): `GET /_ds/fonts.css` → 200 text/css, no-cache +
  quoted-hex ETag, body has `@font-face` + `/_ds/fonts/`, no jsdelivr/http; `GET /_ds/fonts/readex-pro-400.woff2`
  → 200 `font/woff2` + strong ETag; same request with `If-None-Match` → 304; `POST` → 405; `GET /_ds/tokens.css`
  → 200 with `Cache-Control: no-cache` (no `immutable`).
- `GET /` body links `/_ds/fonts.css` and stays CDN-free (`TestDashboardLinksTokensNoCDN` passes).
- Each `internal/web/fonts/*.woff2` → "Web Open Font Format (Version 2)".
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` empty (no dep/schema change).
- No `immutable` Cache-Control value remains in web.go (only two doc-comment mentions explaining the
  policy choice).

**Next:** The realm-index redress — replace the dashboard `<table>` with the Evidence-Ledger grid +
DS token classes (`var(--font-sans)`/`--font-mono` now actually resolve to the embedded webfonts). After
that, thread the token/font CSS into the log browser / dossier / record / certificate surfaces.

**Notes:**
- Scope: 2 modified non-test source files (`internal/web/web.go`, `cmd/iscc-monitor/main.go`) — the
  `.css`/`.woff2`/`OFL.txt` assets and the `.html` template are uncounted per `next.md`; within the
  ≤3-file limit. Nothing from `## Not In Scope` touched (no realm-grid restyle, no log-browser wiring,
  no Arabic subset, no dep/schema change, no store/metrics import into `internal/web`).
- Mount changed exact→subtree per the learnings guard: this is the deliberate reverse of the prior
  exact-path note (fonts NEED `/_ds/fonts/` under a subtree). `http.ServeMux` most-specific match still
  keeps `/`, `/metrics`, `/healthz`, and the per-hub `/<domain>/log/` subtrees from being shadowed
  (E2E-confirmed via buildMux), and `web.Handler` 404s any non-asset `/_ds/` path.
- The woff2 binaries were fetched live from jsDelivr/Fontsource at build time and committed, so the
  served bytes are now build-pinned and never re-fetched at runtime (the CDN was used only as the asset
  source, not a runtime dependency). If the reviewer wants byte-reproducibility, the source URLs and
  resolved package versions (readex-pro@5.2.11, jetbrains-mono@5.2.8) are in this handoff.
- The OFL bodies from the two Fontsource npm packages were verified byte-identical (`diff -q`), so
  `OFL.txt` carries both copyright notices + one shared license body rather than duplicating the text.
- Oracle/conformance gate N/A: pure static-asset transport + a static `<link>`; no signature/RFC-6962/
  Merkle/did:web/fsck/proof path.
