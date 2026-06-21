## 2026-06-21 — Embed the DS v2 token CSS as a shared `internal/web` leaf and link it from `GET /`

**Done:** Created a stdlib-only `internal/web` leaf that `go:embed`s the concatenated, CDN-free ISCC
Design System v2 token CSS and serves it at the exact path `/_ds/tokens.css` (GET-only → 405, immutable
Cache-Control), mounted it in `buildMux`, and added the `<link rel="stylesheet" href="/_ds/tokens.css">`
to the dashboard `<head>`. This is the shared no-JS, no-CDN style shell every later M-UI surface links.

**Files changed:**
- `internal/web/web.go` (new): package doc + `//go:embed tokens.css var TokensCSS []byte`,
  `const TokensPath = "/_ds/tokens.css"`, `Handler()` serving `text/css; charset=utf-8` with
  `Cache-Control: public, max-age=31536000, immutable`; pure leaf (`bytes`/`embed`/`net/http` only).
- `internal/web/tokens.css` (new): `colors.css`+`typography.css`+`spacing.css`+`base.css` concatenated
  verbatim, in that order. `fonts.css` (the jsDelivr `@font-face` block) excluded; `.iscc-grain`
  `background-image: url("../assets/grain.png")` line dropped so the file has zero external/dangling URLs.
- `internal/web/web_test.go` (new): GET→200 `text/css` non-empty body with `--iscc-blue`; non-GET→405;
  CDN-free invariant (no `jsdelivr`/`http`/`url(`).
- `internal/dashboard/dashboard.html`: added the `<link>` in `<head>` (literal path; comment notes it
  must stay in sync with `web.TokensPath`).
- `internal/dashboard/handler_test.go`: added `TestDashboardLinksTokensNoCDN` (body links the stylesheet,
  no `jsdelivr`/`http://`/`https://`/`cdn.`).
- `cmd/iscc-monitor/main.go`: imported `internal/web`, mounted `web.Handler()` at `web.TokensPath` in
  `buildMux` next to `/metrics`/`/healthz`; updated the `buildMux` doc comment for the new exact route.
- `CLAUDE.md` (doc): added the `GET /_ds/tokens.css` bullet under `GET /`; updated the "(no CSS/JS yet)"
  note to "links the shared DS token stylesheet, no JS".

**Verification:** `mise run check` → green (all 18 packages `ok`; `gofmt -l .` empty). Per-criterion:
- `go test -run 'TestTokens|TestDashboard' ./internal/web ./internal/dashboard` → PASS.
- `internal/web` test: GET 200 `text/css` + `--iscc-blue` present; POST → 405; body has no
  `jsdelivr`/`http`/`url(` — PASS.
- Dashboard body: contains `href="/_ds/tokens.css"`, no `jsdelivr`/`http://`/`https://`/`cdn.` — PASS.
- `go list -deps ./internal/web | grep -E 'internal/(store|metrics|logclient)'` → empty (leaf pure;
  internal closure is only `internal/web` itself). `GOOS=js GOARCH=wasm go build ./internal/web` → OK.
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` → empty (no dep/schema change).
- Mutation sanity (mine, reverted): appended an `https://cdn.jsdelivr.net` line to `tokens.css` →
  `TestTokensCDNFree` FAILS on both the `jsdelivr` and `http` asserts; restored byte-identical → green.

**Next:** The fonts sub-step (deferred here): fetch + commit the Readex Pro / JetBrains Mono woff2
binaries, `go:embed` them, serve them under `/_ds/fonts/...`, and add a self-hosted `@font-face`
stylesheet (replacing the excluded CDN `fonts.css` block) — its own ≤3-file change. After that, the
realm-index redress sub-step (rewrite the `<table>` into the Evidence-Ledger realm-register grid and
apply token classes), then thread `/_ds/tokens.css` into the log browser and the future dossier/record/
certificate pages, surface by surface.

**Notes:**
- Scope: 2 new source files (`internal/web/web.go`, `internal/web/tokens.css`) + 1 modified source file
  (`cmd/iscc-monitor/main.go`) = 3 non-test files, at the ≤3 limit (the `.html` template change + tests +
  `CLAUDE.md` don't count). Nothing from `## Not In Scope` touched: no `fonts.css`/`@font-face`/CDN URL,
  no woff2, no grain asset (rule neutralized, not served), no markup restyle, no log-browser/dossier
  wiring, no `internal/store`/`internal/metrics` import, no dep/schema change.
- The `.iscc-grain` rule is kept (minus the `background-image` line) for verbatim-port fidelity; its
  `background-size`/`background-blend-mode` are harmless no-ops without an image. The handler sets no
  CORS headers — the outer `corsmw.Handler(mux)` wrap owns the single CORS policy (per
  `learnings/http-surface.md`). Exact-path mount (no trailing slash, no `/_ds/` subtree) keeps it
  isolated and avoids inviting the fonts sub-step in.
- Pre-existing uncommitted change `.claude/agents/review.md` (the loop-doc improvement the prior handoff
  flagged) is NOT mine and NOT staged — still flagging for whoever owns the loop docs.
- Oracle/conformance gate correctly N/A: pure static-asset transport + a static `<link>`; no signature/
  RFC-6962/Merkle/did:web/fsck/proof path. go.mod/go.sum/schema byte-identical (verified).
