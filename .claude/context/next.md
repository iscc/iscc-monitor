# Next Work Package

## Step: Self-host the DS v2 webfonts — embed Readex Pro / JetBrains Mono woff2 under `/_ds/fonts/...` (no CDN), fold in the stable-path cache fix

## Advances
M-UI (Evidence Ledger frontend, ADR-0010) Verify criterion:

> "every SSR surface ... embeds the DS tokens + self-hosted fonts with **no external CDN URL in the
> body** ... DS tokens + Readex Pro / JetBrains Mono fonts are **self-hosted** (embedded, no runtime CDN)."

The token CSS shell already ships CDN-free (`/_ds/tokens.css`), but its `--font-sans`/`--font-mono`
stacks currently fall back to Arial/Consolas because no webfont is served. This step lands the
self-hosted-fonts half of the shared shell: embed the woff2 binaries, serve them under `/_ds/fonts/...`,
and add a same-origin `@font-face` stylesheet — closing the "self-hosted fonts, no runtime CDN" clause
of the M-UI shell. This is the top item of the state's convergence-driven order and the explicit
`**Next:**` from the last `review` handoff.

It also preempts the OPEN `normal` issue *"`/_ds/tokens.css` uses `immutable` Cache-Control on a
stable, overwrite-in-place URL"* by switching ALL `/_ds/...` assets to the `tilesserve`
`no-cache` + strong-ETag + `If-None-Match`→304 pattern — the handoff explicitly directed folding this
fix in here, since fonts touch `internal/web` anyway.

## Goal
Make the monitor's UI render in Readex Pro / JetBrains Mono with zero runtime CDN dependency, served
from the binary's own `embed.FS` under `/_ds/fonts/...` with a correct revalidating cache policy.

## Scope
- **Create** (assets / docs, not counted against the 3-source limit):
  - `internal/web/fonts/*.woff2` — the **eight latin subsets** (commit the fetched binaries; see notes).
  - `internal/web/fonts.css` — the self-hosted `@font-face` stylesheet (same-origin
    `url("/_ds/fonts/...")`, ported from the design `fonts.css` with the CDN URLs rewritten).
  - `internal/web/fonts/OFL.txt` — the SIL Open Font License text covering both families (required by
    the OFL when redistributing the binaries).
- **Modify** (≤3 non-test/doc source files):
  - `internal/web/web.go` — embed the fonts dir + `fonts.css`; serve `tokens.css`, `fonts.css`, and the
    woff2 under a `/_ds/` surface; replace the `immutable` const with the `no-cache`+strong-ETag+304
    policy for every `/_ds/...` asset. [1 source file]
  - `cmd/iscc-monitor/main.go` — change the exact `mux.Handle(web.TokensPath, web.Handler())` mount to
    what the new surface needs (a `/_ds/` subtree mount, or an added fonts mount); update the `buildMux`
    doc comment. [1 source file]
  - `internal/web/web_test.go` — narrow the `url(` ban to "no external/CDN `url(`" and add the fonts
    assertions (test file — not counted).
  - `internal/dashboard/dashboard.html` — add a `<link rel="stylesheet" href="/_ds/fonts.css">` (or have
    `tokens.css` `@import "/_ds/fonts.css";`); keep the `<head>` CDN-free. [doc/template — not counted]
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/context/learnings/web.md` — the `immutable`-on-stable-path trap, the
    exact-path-vs-subtree mount guard, and the explicit instruction to **narrow the `url(` ban** when
    self-hosted `@font-face` lands. READ THIS FIRST.
  - `/workspace/iscc-monitor/internal/tilesserve/handler.go` — `cacheRevalidate` const and `writeBlob`
    (strong-ETag + `If-None-Match`→304). Port this exact cache shape; do NOT hand-roll.
  - `/workspace/iscc-monitor/.claude/design/_ds/iscc-design-system-v2-50a54aa2-10e0-420b-8f94-b11168b55f5e/tokens/fonts.css`
    — the source `@font-face` blocks (families, weights, `font-display: swap`); rewrite each CDN `src:`
    URL to a same-origin `/_ds/fonts/...` path.
  - `/workspace/iscc-monitor/internal/web/web.go` + `/workspace/iscc-monitor/internal/web/web_test.go` —
    the existing leaf to extend.
  - `/workspace/iscc-monitor/internal/dashboard/dashboard.html` — the `<head>` to add the `<link>` to.

## Not In Scope
- The realm-index redress (table → Evidence-Ledger grid + token classes) — the next sub-step.
- Threading the token/font CSS into the log browser / dossier / record / certificate surfaces.
- Any markup restyle beyond the single `<link>`/`@import` needed to load the fonts.
- The **Arabic-300 Readex subset** from the design `fonts.css` — the monitor UI is English; embed the
  eight **latin** subsets only (Readex Pro 300/400/500/600/700 + JetBrains Mono 400/500/700) to keep the
  embed lean and the `@font-face` honest. Note the omission in a comment.
- Importing any new module dependency; touching `go.mod`/`go.sum`/`schema.sql`.
- Pulling `internal/store`/`metrics`/`logclient` into the `internal/web` closure — it MUST stay a pure
  stdlib leaf (`embed`/`net/http`/`crypto/sha256` etc.), WASM-green.

## Implementation Notes
- **Fetch the binaries deterministically.** The eight latin subsets are live on jsDelivr (probed: all
  HTTP 200, ~15-22 KB each, ~140 KB total embedded). Fetch each into `internal/web/fonts/` with `curl`
  from `https://cdn.jsdelivr.net/fontsource/fonts/<family>@latest/latin-<weight>-normal.woff2`:
  `readex-pro` weights 300/400/500/600/700, `jetbrains-mono` weights 400/500/700. Verify each is a real
  woff2 (`file <f>` reports "Web Open Font Format (Version 2)") before committing. These are committed
  binary assets, so the served bytes are build-pinned and never re-fetched at runtime. Name them plainly,
  e.g. `readex-pro-400.woff2` / `jetbrains-mono-700.woff2`, and reference those exact names in `fonts.css`.
- **Subtree vs exact mount (the learnings guard).** The current `/_ds/tokens.css` is an EXACT mount; the
  reviewer confirmed `GET /_ds/fonts/x.woff2` → 404 today. Fonts need a `/_ds/` subtree. Prefer ONE
  coherent `/_ds/` surface: a thin handler (or `http.FileServer(http.FS(embedded))` wrapped) mounted at
  `/_ds/` that serves `tokens.css`, `fonts.css`, and `fonts/*.woff2`. Whichever you pick, set the content
  type explicitly — `text/css; charset=utf-8` for `.css`, `font/woff2` for `.woff2` (`http.FileServer`
  would sniff woff2 as octet-stream). A thin extension-switch handler is cleanest and lets you apply the
  cache policy uniformly. Keep the `web.TokensPath` const (it is linked from the dashboard); add a
  matching `FontsCSSPath`/`FontsPath` const if helpful.
- **Cache policy (the issue fix).** Drop the `immutable` const. Every `/_ds/...` 200 gets
  `Cache-Control: no-cache`, a STRONG content ETag (`fmt.Sprintf("\"%x\"", sha256.Sum256(data))`, no
  `W/` prefix), and an `If-None-Match` (`"*"` or exact-tag) → `304 Not Modified` short-circuit — exactly
  the `tilesserve.writeBlob` shape. Set all headers BEFORE the conditional branch (the status freezes the
  header map). Keep the post-200 write-drop convention (`_, _ = w.Write(...)`).
- **CDN-free invariant (Correctness: "no external CDN URL in the body").** `fonts.css` `src:` URLs become
  same-origin `url("/_ds/fonts/readex-pro-400.woff2")`. The web_test `url(` ban must narrow to "no
  `jsdelivr`, no `http`/`https`, no `cdn.`" — KEEP those substring bans (a same-origin `url(` is fine),
  relax only the bare `url(` ban that `TestTokensCDNFree` currently asserts. Add a `TestFonts*` that the
  served `fonts.css` contains `@font-face` + a `/_ds/fonts/` path and NO `jsdelivr`/`http`.
- **Method discipline.** Non-GET on any `/_ds/...` asset → 405 (carry the existing convention forward).
- **CORS** still rides the outer `corsmw.Handler(mux)` wrap — set no CORS headers in this package.
- **Purity (Correctness: `internal/web` is a WASM-shareable leaf).** Verify with
  `GOOS=js GOARCH=wasm go build ./internal/web` and `go list -deps ./internal/web` (internal closure must
  be only `internal/web`). `embed` + `crypto/sha256` + `net/http` + `fmt`/`strings`/`bytes` are all fine.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run 'TestTokens|TestFonts|TestDashboard' ./internal/web ./internal/dashboard` passes.
- `GOOS=js GOARCH=wasm go build ./internal/web` exits 0 (leaf stays WASM-green).
- `go list -deps ./internal/web | grep -E 'internal/(store|metrics|logclient)'` is empty (leaf purity).
- Through the real `buildMux` (a throwaway test or `curl` against a built binary):
  `GET /_ds/fonts.css` → `200 text/css`, body contains `@font-face` + `/_ds/fonts/` and NO `jsdelivr`/`http`;
  `GET /_ds/fonts/<one>.woff2` → `200` with `Content-Type: font/woff2` and a quoted-hex `ETag`;
  re-request with that `If-None-Match` → `304` empty body; `POST /_ds/fonts/<one>.woff2` → `405`.
- `GET /_ds/tokens.css` now returns `Cache-Control: no-cache` + a strong ETag (NOT `immutable`); no
  `immutable` Cache-Control remains in `internal/web` (`grep -R "immutable" internal/web/web.go` → no
  Cache-Control hit).
- `GET /` body links the fonts stylesheet (or tokens.css `@import`s it) and stays CDN-free
  (no `jsdelivr`/`http://`/`https://`/`cdn.` in the rendered body).
- Each committed `internal/web/fonts/*.woff2` is a valid woff2 (`file` → "Web Open Font Format (Version 2)").
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` is empty (no dep/schema change).

## Done When
`mise run check` is green and all Verification criteria pass: the eight latin woff2 subsets are embedded
and served under `/_ds/fonts/...` with a `no-cache`+strong-ETag+304 policy, `/_ds/tokens.css` no longer
carries `immutable`, a same-origin CDN-free `@font-face` stylesheet is linked from `/`, and
`internal/web` remains a WASM-green pure leaf.
