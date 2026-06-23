# Next Work Package

## Step: M-API slice 3 — serve `GET /docs` + self-hosted byte-pinned Stoplight Elements assets

## Advances
The **M-API** milestone's third (and final functional) Verify criterion (`target.md` §M-API):

> `GET /docs` returns `200 text/html`, loads the **self-hosted, byte-pinned** Stoplight Elements assets
> from `/_ds/` (the `<elements-api>` web-component JS **and** its stylesheet, each served byte-verbatim
> with the `verify.wasm` strong-ETag + no-cache + 304 policy; each SHA-256 published as an `internal/web`
> constant next to `WasmVerifyHash`), points it at `/openapi.json` via `apiDescriptionUrl`, and has **no
> external CDN host in the body and makes no external runtime call** — Elements' "Try It" calls this
> instance directly (no `tryItCorsProxy`) on the existing CORS `*`.

This is the nearest unmet milestone Verify criterion (state.md "M-API: 2/4 met — slices 1+2 LANDED, slice
3 OPEN"), the `review` handoff's explicit `**Next:**`, and there are **0 critical** issues. M-API is
order-independent and entirely in-repo / CI-verifiable (oracle gate N/A — static-asset transport + one HTML
shell, no crypto/proof path).

## Goal
Vendor the two Stoplight Elements assets (web-component JS + stylesheet) into `internal/web`, serve them
byte-verbatim under `/_ds/` with published SHA-256 pins, and mount a server-rendered `GET /docs` page that
mounts `<elements-api apiDescriptionUrl="/openapi.json">` against those same-origin assets — making the
machine API explorable in-browser with zero external runtime call.

## Scope
- **Create**:
  - `internal/web/elements.min.js` — the vendored Stoplight Elements web-component JS bundle, committed
    verbatim (a downloaded/generated asset, never hand-edited — same discipline as `wasm_exec.js`).
  - `internal/web/elements.min.css` — the vendored Stoplight Elements stylesheet, committed verbatim.
  - `internal/docs/docs.go` + `internal/docs/docs.html` — a tiny pure-stdlib SSR leaf in the
    `internal/verifier` shape: `//go:embed docs.html`, `template.Must(...Parse...)`, `Handler()` that
    buffers-then-200s `text/html; charset=utf-8` (405 on non-GET). The page links `/_ds/tokens.css` +
    `/_ds/fonts.css` + the masthead chrome (logo), and mounts
    `<elements-api apiDescriptionUrl="/openapi.json" router="hash">` loading `/_ds/elements.min.js`
    (a `<script>`) + `/_ds/elements.min.css` (a `<link>`), with **no `tryItCorsProxy` attribute**.
    (One non-test/doc Go file — `docs.go`; the `.html` is a template and the two assets are vendored
    binaries-as-data, not hand-written source.)
- **Modify** (≤3 non-test/doc source files):
  1. `internal/web/web.go` — add `//go:embed` for the two Elements assets, two path consts
     (`ElementsJSPath = "/_ds/elements.min.js"`, `ElementsCSSPath = "/_ds/elements.min.css"`), two published
     pin consts (`ElementsJSHash`, `ElementsCSSHash`, lowercase-hex SHA-256, next to `WasmVerifyHash`), and
     two `case` arms in `Handler`'s path switch reusing `writeAsset` (JS → `contentTypeJS`, CSS →
     `contentTypeCSS`). Keep the leaf pure-stdlib / WASM-green.
  2. `cmd/iscc-monitor/main.go` — `mux.Handle("/docs", docs.Handler())` in `buildMux`; add `"docs"` to
     `reservedMountNames` (a realm domain `docs` would collide with the exact `/docs` mount, like
     `metrics`/`version`); extend the `buildMux` docstring with the `/docs` route.
  - **Doc sync (required):** `CLAUDE.md` — add a `GET /docs` bullet to the dev-instance endpoint list and a
    `GET /_ds/elements.min.{js,css}` note (mirrors the existing `/_ds/verify.wasm` bullet). A doc file (not
    counted against the 3-source budget), but in-scope: a behavior change must stay documented.
- **Reference**:
  - `.claude/context/learnings/openapi.md` — drift-test gates PATHS ONLY; `/docs` is HTML SSR so it goes on
    the **exclusion list**, never in the contract.
  - `.claude/context/learnings/web.md` — `/_ds/` subtree mount mechanics, `writeAsset` no-cache+ETag+304
    shape, the `WasmVerifyHash` reproducible-pin discipline, the per-template literal-must-match-the-Go-const
    rule, the `noExternalCDN` helper's third-party-origin-only ban.
  - `internal/verifier/handler.go` + `internal/verifier/handler_test.go` — the exact SSR-leaf shape to copy
    (embed → parse-at-init → buffer-then-200) and the no-CDN body-ban test pattern (`TestVerifierNoExternalCDN`,
    handler_test.go:278-291).
  - `internal/web/web.go` `Handler()` switch + `writeAsset` + `WasmVerifyHash`/`contentTypeJS`/`contentTypeCSS`
    consts — the literal serve site to extend.
  - `cmd/iscc-monitor/openapi_drift_test.go` `ssrExclusions()` (line 78) — add `"/docs"` so the drift test
    asserts it is mounted AND not in the contract.
  - `.claude/adr/0014-*.md` decision §4 (the authoritative `/docs` + Elements spec: two pinned assets, no
    `tryItCorsProxy`, `apiDescriptionUrl="/openapi.json"`).

## Not In Scope
- **The two `normal` OpenAPI contract-accuracy fixes** (phantom `verify` `index` param + `checkpoint`
  `text/plain`→`application/octet-stream` media type) and the `low` `healthz` 503. They are a separate
  `openapi.yaml`/`.json` doc-touch (the drift test is path-only, blind to them) — fold them into the
  IMMEDIATE follow-on step, NOT this one. Touching `openapi.yaml` + regenerating the JSON twin here would
  blow the file budget and mix two concerns. (Breadcrumb for the next iteration.)
- **Re-pinning `verify.wasm` or adding a per-operation OpenAPI golden** — those belong to the contract-fix
  step.
- **A consolidated chrome/masthead partial** — keep copying the existing per-template masthead block (the
  six-way duplication is a tracked `low`); do not refactor it here.
- **Adding `/docs` to the OpenAPI contract or `machineProbes()`** — it is an HTML surface, excluded by
  design (ADR-0014 §1).

## Implementation Notes
- **Vendor the assets from a PINNED version, byte-verbatim, committed — never fetched at build time.** A
  known-good, network-confirmed source is Stoplight Elements `9.0.23`:
  `https://unpkg.com/@stoplight/elements@9.0.23/web-components.min.js` (~2.0 MB) and
  `https://unpkg.com/@stoplight/elements@9.0.23/styles.min.css` (~290 KB). Download once, commit the exact
  bytes, then compute `sha256sum` on the COMMITTED files and paste those into `ElementsJSHash` /
  `ElementsCSSHash`. (The bytes I downloaded this iteration hashed
  `46e5a044295bbd599772e1a5e678e807078a3d2cd43226640a50917cd88d6938` for the JS and
  `a52002228108fb567b75caff209c4d8aa256ae591de3dea7d3b1a384b1a27b06` for the CSS — re-verify against the
  committed files, do not trust this note blindly.) The assets join the "verify.wasm pin is fragile"
  discipline (MEMORY): re-fetch + re-pin deliberately, never hand-edit. A pin test
  (`TestElementsAssetsHashPinned`, mirroring `TestWasmVerifyHashPinned`) asserts
  `sha256.Sum256(asset) == hash` for both — reverting either const FAILS it.
- **CRITICAL no-CDN nuance — do NOT run a substring CDN ban over the 2 MB Elements JS bytes.** The Elements
  bundle contains hundreds of baked `http://…`/`https://…` example/documentation strings (`golang.org`,
  `example.com`, `…twitter.com`, etc. — confirmed this iteration via `grep -oE "https?://" elements.min.js`).
  These are inert data inside the minified bundle, **not** runtime CDN fetches; the bundle issues no
  external request at runtime, and the CSS has no `@import` / external-font `url()` (confirmed). So the
  no-CDN assertion must apply ONLY to the **`/docs` HTML page body** (the `verifier`
  `TestVerifierNoExternalCDN` pattern over the rendered HTML — bans `jsdelivr`/`http://`/`https://`/`cdn.`),
  NEVER over the served JS/CSS asset bytes. Document this in the asset's `web.go` doc comment and the test
  so a future reviewer does not "fix" it into a false-positive.
- **`/docs` is HTML SSR ⇒ exclusion list, not the contract.** The drift test (`learnings/openapi.md`)
  reconciles PATHS: machine routes ↔ `machineProbes()`, SSR routes ↔ `ssrExclusions()`. Add `"/docs"` to
  `ssrExclusions()` (it has no template params → `ssrTemplate` falls through to itself). Do NOT add it to
  `machineProbes()` or the OpenAPI doc — that would FAIL the SSR "must be excluded" assertion.
- **Reserved-name guard:** `docs` must join `reservedMountNames` (a plain literal like
  `metrics`/`healthz`/`version`), so a realm domain literally named `docs` fails loudly at startup instead
  of panicking `http.ServeMux` against the exact `/docs` mount. Mirror the existing entries.
- **Page shell:** copy the `internal/verifier` no-JS DS shell — `<head>` links `/_ds/tokens.css` +
  `/_ds/fonts.css`, masthead chrome with the `/_ds/iscc-logo-black.png` logo `<img>` (the same per-template
  `.chrome-*` block + the height-`38px` page-scoped style the other surfaces carry; `learnings/web.md`),
  then `<elements-api apiDescriptionUrl="/openapi.json" router="hash">` + the `<script src="/_ds/elements.min.js">`
  and `<link rel="stylesheet" href="/_ds/elements.min.css">`. The `/_ds/` paths are template LITERALS that
  must stay byte-equal to the Go consts (templates cannot read consts — the dashboard/verifier convention);
  assert them in the handler test. Leave `tryItCorsProxy` UNSET (ADR-0014 §4: Try-It goes browser→instance
  directly on the existing CORS `*`).
- **Leaf purity:** `internal/docs` stays pure stdlib (`bytes`/`embed`/`html/template`/`net/http`) — no
  `internal/store|metrics|web` import (it links `/_ds/` by literal, like `verifier`). `internal/web` stays
  WASM-green: the two new `case` arms add no import; verify `GOOS=js GOARCH=wasm go build ./internal/web`
  still OK.
- **Relevant rule:** no crypto-path learning applies (oracle gate N/A — static transport + HTML). The
  load-bearing rules here are the asset-pin discipline (MEMORY "verify.wasm pin is fragile") and the
  no-CDN-over-HTML-only nuance above.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestDocs ./internal/docs` passes: `GET /docs` → `200 text/html`; body contains
  `apiDescriptionUrl="/openapi.json"`, `<elements-api`, `src="/_ds/elements.min.js"`,
  `href="/_ds/elements.min.css"`, `href="/_ds/tokens.css"`, `src="/_ds/iscc-logo-black.png"`; body does
  NOT contain `tryItCorsProxy`; the no-CDN ban (`jsdelivr`/`http://`/`https://`/`cdn.`) finds none in the
  HTML body; non-GET → 405.
- `go test -run TestElements ./internal/web` passes: `GET /_ds/elements.min.js` → `200`,
  `Content-Type: text/javascript; charset=utf-8`, body byte-equal to the embedded JS, strong ETag, and a
  matching `If-None-Match` → `304`; `GET /_ds/elements.min.css` → `200`, `text/css; charset=utf-8`,
  byte-equal, 304-on-match; `sha256.Sum256` of each served asset equals its published `Elements*Hash`
  (`TestElementsAssetsHashPinned`, reverting a const FAILS).
- `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` passes with `/docs` on the exclusion list — it is
  probed MOUNTED (non-404) and asserted NOT declared in the contract; removing it from `ssrExclusions`
  while it is mounted, or declaring it in the doc, FAILS.
- `GOOS=js GOARCH=wasm go build ./internal/web` succeeds (the leaf stays WASM-green).
- `sha256sum internal/web/elements.min.js internal/web/elements.min.css` equals the two published consts
  (the pin is reproducible from the committed bytes).

## Done When
`mise run check` is green and all the criteria above pass: `GET /docs` serves the no-CDN Stoplight Elements
shell pointed at `/openapi.json`, both Elements assets are served byte-verbatim from `/_ds/` under
published SHA-256 pins with the no-cache+ETag+304 policy, and the drift test confirms `/docs` is a mounted,
contract-excluded HTML surface — closing the M-API `/docs` Verify criterion.
