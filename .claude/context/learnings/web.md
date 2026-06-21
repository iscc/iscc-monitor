<!-- area: internal/web (web.go, tokens.css) + the dashboard <link> / buildMux mount -->
<!-- indexed-as: web.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# Static front-end assets — the shared `internal/web` leaf (`/_ds/...`)

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## `internal/web` — embedded DS token CSS shell (`GET /_ds/tokens.css`)

- **`immutable` Cache-Control is ONLY safe on a content-addressed URL; `/_ds/tokens.css` is a STABLE
  overwrite-in-place path, so `immutable` is the wrong policy here.** The landed handler sets
  `public, max-age=31536000, immutable`, which contradicts the project's own convention:
  `internal/tilesserve` uses `cacheImmutable` ONLY for FULL content-addressed tiles (URL changes when
  bytes change) and `cacheRevalidate = "no-cache"` for anything overwritten in place at a fixed URL —
  its comment warns such a resource "must NOT carry the immutable directive or a client would pin a
  soon-overwritten" version. A redeploy that changes `tokens.css` reuses the same URL, so an `immutable`
  client can keep stale CSS for a year. Cosmetic (stale tokens, no correctness impact) — tracked in
  issues.md as a `normal` follow-up. For ANY future static asset served at a stable path (the upcoming
  self-hosted woff2 at `/_ds/fonts/...` too): use `no-cache` + a strong content ETag + `If-None-Match`→304
  (the `tilesserve.writeBlob` shape), OR a build-fingerprinted path (`/_ds/tokens.<hash>.css`). Do NOT
  copy the `immutable`-on-a-stable-path pattern forward.
- **Mount at the EXACT path, never a `/_ds/` subtree.** `mux.Handle(web.TokensPath, web.Handler())` is an
  exact `http.ServeMux` pattern (no trailing slash). Reviewer end-to-end-confirmed through the full
  `buildMux`: `GET /_ds/tokens.css` → 200 text/css; `GET /_ds/fonts/x.woff2` → 404 (nothing else under
  `/_ds/` is served); `GET /` still hits the dashboard; CORS (`Allow-Origin: *`) rides the outer
  `corsmw.Handler(mux)` wrap so the handler sets none. An exact mount keeps the surface isolated and
  stops a later sub-step (fonts) from silently leaking under a subtree.
- **The CDN-free test bans `http` AND `url(` substrings — this only stays true because the ported token
  files are scheme-less.** `internal/web/web_test.go` asserts the served bytes contain no `jsdelivr`, no
  `http`, no `url(`. Source-of-truth: the only CDN URLs in the DS bundle live in `fonts.css` (excluded),
  and the only `url(` in colors/typography/spacing/base is the `.iscc-grain background-image:
  url("../assets/grain.png")` in `base.css` — which the port neutralizes (drops the `background-image`
  line, keeps `background-size`/`background-blend-mode` as harmless no-ops). When the fonts sub-step
  lands self-hosted `@font-face`, those `src: url("/_ds/fonts/...woff2")` lines WILL reintroduce `url(`
  (a same-origin relative URL, which is fine) — the `url(` ban must then narrow to "no external/CDN
  `url(`", not "no `url(` at all". Keep the `jsdelivr`/`http`-substring bans; relax only the bare `url(`.
- **The dashboard body's `http://`/`https://` ban is satisfied only because `ListHubs` renders the
  scheme-less `h.origin` (`<domain>/log`), NOT the `https://...` `base_url`.** `TestDashboardLinksTokensNoCDN`
  bans `http://`/`https://`/`cdn.` in the rendered `/` body; this passes because the template renders
  `.Domain` + `.Origin` (origin = `sb0.iscc.id/log`), never the hub's `base_url`. If a future SSR surface
  ever renders a hub `base_url` (an `https://...` value), this anti-CDN assertion will false-positive on a
  legitimate same-hub URL — scope the ban to third-party origins (e.g. `jsdelivr`/`cdn.`/a non-self host),
  not the bare `https://` substring, the moment a real `https://` value is intentionally rendered.
- **Pure stdlib leaf, WASM-green.** Closure is `bytes`/`embed`/`net/http` only; `go list -deps
  ./internal/web` contains no `internal/store|metrics|logclient` (only `internal/web` itself), and
  `GOOS=js GOARCH=wasm go build ./internal/web` is OK. Mirror the `internal/badge` `go:embed` idiom for
  any future static asset; go.mod/go.sum/schema stay byte-identical. Oracle gate N/A (static transport).
