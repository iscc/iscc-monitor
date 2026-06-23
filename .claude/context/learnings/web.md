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
- **`verify.wasm` is served at `WasmVerifyPath` (`/_ds/verify.wasm`, `application/wasm`) via the same
  `case`+`writeAsset` leaf, and its SHA-256 is pinned in `WasmVerifyHash` (TestWasmVerifyHashPinned,
  mutation-proven). settled: the reproducible-build trap is CLOSED — `build:wasm` now carries
  `-buildvcs=false`, the committed blob has zero `vcs.` strings, and `mise run build:wasm` is
  byte-identical (`f03b9b89…`) across clean / untracked-dirty / tracked-dirty / `go clean -cache`
  (reviewer + Codex both re-measured; pin == committed == rebuild).**
- **A `GOOS=js GOARCH=wasm` `go build` is NOT reproducible without `-buildvcs=false` — `-trimpath
  -ldflags=-buildid=` is INSUFFICIENT.** Go stamps `debug.ReadBuildInfo` VCS metadata (`vcs.revision`,
  `vcs.modified`, the `mod` `+dirty` suffix) into the wasm `data` section by default, so the hash changes
  with EVERY commit and flips on a dirty vs clean tree (an 8-byte `data`-section delta; reviewer measured
  3 distinct hashes — clean / dirty / parent-rev — and 0/10 reproductions of a committed-at-parent-rev
  artifact). The committed `verify.wasm`'s build settings are readable via
  `strings verify.wasm | grep vcs.` — confirm `vcs.modified=false` and that `vcs.revision` matches the
  commit. The fix is `-buildvcs=false` in the `mise run build:wasm` task (verified: stable single hash
  across clean / dirty / `go clean -cache`); then rebuild AND re-pin `WasmVerifyHash`. Any future
  committed wasm/binary artifact whose hash is published as an SRI pin MUST be built with `-buildvcs=false`
  or the published hash is unreproducible from a clean checkout (CI rebuild-and-compare can never pass).
- **The `noExternalCDN` helper strips each line's `//` comment tail before scanning (so the vendored
  `wasm_exec.js`'s one Go-issue-tracker comment URL stops false-positiving). The predicate is now
  comment-context only: `//` is a comment ONLY at line-start or when preceded by whitespace (space/tab)
  — `web_test.go:57`.** This closed the quoted-delimiter over-strip: a `src="//cdn..."` / `url("//cdn...")`
  is preceded by `"` (not whitespace) so it survives and trips the ban (`TestNoExternalCDNProtocolRelative`
  pins it, mutation-proven against the old `:`-only guard). The ban list is third-party-origin only
  (`jsdelivr`/`http://`/`https://`/`cdn.`); same-origin `/_ds/` paths pass.
- **`iscc-logo-black.png` is served at `LogoPath` (`/_ds/iscc-logo-black.png`, `image/png`) via the same
  `case`+`writeAsset` no-cache/strong-ETag/304 leaf, embedded with `//go:embed iscc-logo-black.png`.** Two
  forward rules: (1) the masthead `<img src>` literal lives in each template (`dashboard.html`,
  `dossier.html`, …) and templates CANNOT read the Go const — the string must stay byte-equal to `LogoPath`,
  asserted by `TestDashboard…`/`TestDossier…` (`src="/_ds/iscc-logo-black.png"`, mutation-proven). (2)
  `contentTypePNG = "image/png"` is a string literal ONLY — never import an `image/*` package, or the leaf
  stops being the pure-stdlib WASM-green closure (`GOOS=js GOARCH=wasm go build ./internal/web` must stay OK).
  The PNG is a committed, pre-downscaled artifact (199×76 gray+alpha, ~3 KB) — the downscale ran ONCE at
  commit time, never in a build/`mise` step (no image toolchain on the build path; ADR-0003 pure-Go). The
  same-origin `/_ds/` src passes `noExternalCDN` + the dashboard body ban (no `http(s)://`/`cdn.`/`jsdelivr`).
  settled: the logo now renders on ALL SIX SSR mastheads — `dashboard.html`, `dossier.html`, plus
  `certificate/cert.html` + `proofserve/{browser,record,records}.html` (the same `.chrome-brand`/`.chrome-logo`/
  `.chrome-divider` block + the one-line `<img>`, copied verbatim). Each surface's handler test asserts
  `src="/_ds/iscc-logo-black.png"` (mutation-proven on cert + browser). Forward rule when adding a NEW SSR
  surface: copy the same masthead block and add the same src assertion to its handler test — OR, if the
  six-way copy-paste is finally consolidated, a single `html/template` chrome partial (deferred KISS move).
  Never let an SSR masthead ship without the logo (target.md:148 "every surface").
- **Masthead logo height is `38px` (`.chrome-logo { height: 38px; width: auto }`), set per-template in each
  surface's page-scoped `<style>` (the same seven-way copy as the `<img>`: the six SSR mastheads above +
  `verifier/verifier.html`).** Doubled from the original `19px` on 2026-06-23 by a deliberate human design
  tweak made directly, OUTSIDE the CID loop. The `.claude/design` source mockups (`*.dc.html`,
  `style="height:38px;..."`) were updated in the same change, so spec and implementation stay in sync — `38px`
  is now the ratified value; do NOT "fix" it back to `19px`. No test asserts the pixel height (handler tests
  only assert the `src` literal), so this is template-CSS-only with no test coupling. If/when the chrome is
  consolidated into one partial, this height lives there too.
- **Residual whitespace-prefixed hole (open `low` issue, Codex P2):** the same predicate still treats a
  `//` preceded by whitespace as a comment, so the (rare, mostly-invalid) whitespace-before-URL forms
  `<script src = //cdn...>` and CSS `url( //cdn...)` are stripped and the ban misses them. This is NOT a
  regression — the old `:`-only guard stripped these too (verified), and no served asset uses the form.
  The robust fix when next touched is a tokenizer-grade check (only treat `//` as a comment outside a
  quoted string / `url(...)` token), not another delimiter blocklist. Until then, do not add an asset
  with a whitespace-prefixed protocol-relative URL.
- **Stoplight Elements assets (`/_ds/elements.min.js` + `.css`) are served via the same `case`+`writeAsset`
  no-cache/strong-ETag/304 leaf (JS→`contentTypeJS`, CSS→`contentTypeCSS`), SHA-256-pinned in
  `ElementsJSHash`/`ElementsCSSHash` (TestElementsAssetsHashPinned, sibling of the wasm pin).** They join
  the "verify.wasm pin is fragile" discipline: vendored byte-verbatim from `@stoplight/elements@9.0.23`,
  NEVER hand-edited (the JS is a 2 MB minified bundle), re-fetch+re-pin on a version bump. CRITICAL no-CDN
  nuance: do NOT run a substring CDN ban over the 2 MB JS bytes — it carries hundreds of inert baked
  `http(s)://` example strings that are DATA, not fetches (the no-CDN ban runs ONLY over the rendered
  `/docs` HTML body; doc-commented in `web.go` + `TestElementsJSServed`). The CSS has no `@import`/external
  `url()` (reviewer re-confirmed) so it makes no external request.
- **Latent no-CDN gap in the Elements bundle (open `normal` issue, Codex P2, reviewer-confirmed): the
  vendored `elements.min.js` hardcodes `https://unpkg.com/mermaid@9.4.3/dist/mermaid.min.js`** and
  lazy-loads it the first time a Markdown description renders a fenced ```mermaid block. Our served
  `/openapi.json` has ZERO `mermaid` today (reviewer-grepped), so `/docs` makes no external request now
  (live smoke + visual pass confirmed) — but ANY future OpenAPI description with a mermaid block would fire
  a third-party CDN fetch, breaking the no-CDN invariant. Forward rule: never put a fenced `mermaid` block
  in a served OpenAPI description; the durable guard is a test banning `mermaid` in the served doc body
  (hand-patching the pinned minified bundle would violate the never-hand-edit pin discipline). The bundle
  has exactly ONE such dynamic external asset loader (the mermaid `bE` const); speakerdeck/vimeo strings
  are oEmbed example data, not unconditional loads.
