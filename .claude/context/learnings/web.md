<!-- area: internal/web (web.go, tokens.css) + the dashboard <link> / buildMux mount -->
<!-- indexed-as: web.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# Static front-end assets — the shared `internal/web` leaf (`/_ds/...`)

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## `internal/web` — embedded DS token/font shell (`GET /_ds/...`)

- settled: the `immutable`-on-a-stable-path trap is fixed and the surface is now a `/_ds/` SUBTREE mount.
  Every `/_ds/...` asset (tokens.css, fonts.css, woff2) is served by one `web.Handler` mounted at
  `web.Prefix` with the `no-cache` + strong content-ETag + `If-None-Match`→304 policy (the
  `tilesserve.writeBlob` shape, hex of `sha256.Sum256(data)`, no `W/` prefix). The two `immutable`
  mentions left in `web.go` are doc comments explaining WHY the directive is avoided, not a header value.
  Forward rule: NEVER serve a stable (non-content-addressed) `/_ds/` asset with `immutable`; redeploy
  reuses the URL with changed bytes, so a client would pin stale CSS/fonts for a year. Content-fingerprint
  the path or revalidate.
- **The mount is now a `/_ds/` SUBTREE (`mux.Handle(web.Prefix, web.Handler())`), the deliberate reverse
  of the prior exact-path mount** — fonts need `/_ds/fonts/x.woff2` to route here. `http.ServeMux`
  most-specific match still keeps `/`, `/metrics`, `/healthz`, and each `/<domain>/log/` subtree from being
  shadowed (reviewer E2E-confirmed through the real `buildMux`), and `web.Handler` 404s any non-asset
  `/_ds/` path. If you add another `/_ds/...` family, extend the in-handler path switch — do NOT add a
  second competing `/_ds/...` mux pattern.
- **`serveFont` is the one request-path→filesystem-read site; its guard is load-bearing.** It rejects
  anything not ending `.woff2` and anything with a `/` after `fonts/`, so `../`, nested paths, and
  non-woff2 names all 404 before the `fs.ReadFile`. Reviewer probed `..`/nested/double-`fonts/` paths →
  all 404. Keep both clauses if you touch this; `http.ServeMux` also path-cleans `..` in production, but
  the guard must stand alone.
- **The CDN-free invariant is now enforced by `noExternalCDN` banning `jsdelivr`/`http://`/`https://`/`cdn.`
  — the bare `http`/`url(` substring bans are GONE on purpose.** Self-hosted `@font-face` legitimately
  needs same-origin `src: url("/_ds/fonts/...woff2")`, so a bare `url(` ban is wrong. The new helper bans
  only third-party origins; a same-origin `url(` and a relative `/_ds/` path pass. `TestFontsCSSReferencesEmbeddedSubsets`
  also cross-checks that every `src` path in fonts.css resolves to an embedded woff2 (200) — exactly 8.
  Watch: the `http://`/`https://` bans (not bare `http`) still false-positive if a future SSR surface
  renders a hub `base_url` (an `https://...`); scope to a non-self host then.
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
- **`wasm_exec.js` is served at `WasmExecPath` (`/_ds/wasm_exec.js`), byte-verbatim from
  `$(go env GOROOT)/lib/wasm/wasm_exec.js` (Go 1.26.1, `cmp`-identical), via the same in-handler `case`
  + `writeAsset` + no-cache/strong-ETag/304 leaf (`contentTypeJS = text/javascript; charset=utf-8`).
  Never hand-edit it; re-`cp` it on a toolchain bump.**
- **The `noExternalCDN` helper strips each line's `//` comment tail before scanning (so the vendored
  `wasm_exec.js`'s one Go-issue-tracker comment URL stops false-positiving). The predicate is now
  comment-context only: `//` is a comment ONLY at line-start or when preceded by whitespace (space/tab)
  — `web_test.go:57`.** This closed the quoted-delimiter over-strip: a `src="//cdn..."` / `url("//cdn...")`
  is preceded by `"` (not whitespace) so it survives and trips the ban (`TestNoExternalCDNProtocolRelative`
  pins it, mutation-proven against the old `:`-only guard). The ban list is third-party-origin only
  (`jsdelivr`/`http://`/`https://`/`cdn.`); same-origin `/_ds/` paths pass.
- **Residual whitespace-prefixed hole (open `low` issue, Codex P2):** the same predicate still treats a
  `//` preceded by whitespace as a comment, so the (rare, mostly-invalid) whitespace-before-URL forms
  `<script src = //cdn...>` and CSS `url( //cdn...)` are stripped and the ban misses them. This is NOT a
  regression — the old `:`-only guard stripped these too (verified), and no served asset uses the form.
  The robust fix when next touched is a tokenizer-grade check (only treat `//` as a comment outside a
  quoted string / `url(...)` token), not another delimiter blocklist. Until then, do not add an asset
  with a whitespace-prefixed protocol-relative URL.
