## 2026-06-23 — M-API slice 3 — serve `GET /docs` + self-hosted byte-pinned Stoplight Elements assets

**Done:** Vendored the two Stoplight Elements `9.0.23` assets (web-component JS + stylesheet)
byte-verbatim into `internal/web`, served them under `/_ds/` with published SHA-256 pins on the same
no-cache + strong-ETag + 304 policy as `verify.wasm`, and added a pure-stdlib SSR leaf `internal/docs`
that mounts `<elements-api apiDescriptionUrl="/openapi.json">` against those same-origin assets — making
the machine API explorable in-browser with no external CDN host and no external runtime call (no
`tryItCorsProxy`). Mounted `GET /docs` in `buildMux`, reserved the `docs` mount name, and put `/docs` on
the drift test's SSR exclusion list. Closes the M-API `/docs` Verify criterion.

**Files changed:**
- `internal/web/elements.min.js` (new): vendored `@stoplight/elements@9.0.23/web-components.min.js`
  (~2.0 MB), committed verbatim. SHA-256 `46e5a044…d6938`.
- `internal/web/elements.min.css` (new): vendored `@stoplight/elements@9.0.23/styles.min.css` (~290 KB),
  committed verbatim. SHA-256 `a5200222…27b06`.
- `internal/docs/docs.go` + `internal/docs/docs.html` (new): pure-stdlib SSR leaf in the `verifier`
  shape — `//go:embed docs.html`, parse-at-init, buffer-then-200 `text/html`, 405 on non-GET. Static no-JS
  DS shell (tokens/fonts/logo) mounting `<elements-api … router="hash">` from `/_ds/elements.min.js` +
  `/_ds/elements.min.css`, `tryItCorsProxy` left unset.
- `internal/web/web.go`: two path consts (`ElementsJSPath`, `ElementsCSSPath`), two pin consts
  (`ElementsJSHash`, `ElementsCSSHash`) next to `WasmVerifyHash`, two `//go:embed` vars, two `Handler`
  `case` arms reusing `writeAsset` (JS→`contentTypeJS`, CSS→`contentTypeCSS`). Doc comments updated.
- `cmd/iscc-monitor/main.go`: `mux.Handle("/docs", docs.Handler())`, `"docs"` added to
  `reservedMountNames`, `buildMux` docstring extended.
- `internal/web/web_test.go`: `TestElementsJSServed` / `TestElementsCSSServed` (200, content-type,
  byte-equal, strong ETag, 304-on-match) + `TestElementsAssetsHashPinned` (mutation-proven pin).
- `internal/docs/docs_test.go` (new): 200 text/html, Elements mount + `/_ds/` refs + DS shell,
  no-`tryItCorsProxy`, no-CDN body ban, 405 non-GET.
- `cmd/iscc-monitor/openapi_drift_test.go`: `/docs` added to `ssrExclusions()`.
- `CLAUDE.md`: added `GET /docs` and `GET /_ds/elements.min.{js,css}` endpoint bullets.

**Verification:** `mise run check` → GREEN (build + vet + test, all 30 pkgs).
- `go test -run TestDocs ./internal/docs` → PASS (200 text/html; `apiDescriptionUrl="/openapi.json"`,
  `<elements-api`, `src="/_ds/elements.min.js"`, `href="/_ds/elements.min.css"`, `href="/_ds/tokens.css"`,
  `src="/_ds/iscc-logo-black.png"` present; `tryItCorsProxy` absent; no-CDN body ban clean; 405 non-GET).
- `go test -run TestElements ./internal/web` → PASS (both assets: 200, correct content-type, byte-equal,
  strong ETag, 304-on-match; `sha256.Sum256` of each equals its published `Elements*Hash`).
- `go test -run TestOpenAPIDrift ./cmd/iscc-monitor` → PASS with `/docs` probed MOUNTED and asserted NOT
  declared in the contract.
- `GOOS=js GOARCH=wasm go build ./internal/web` and `./internal/docs` → both succeed (WASM-green).
- `sha256sum internal/web/elements.min.{js,css}` equals the two published consts (pin reproducible from
  committed bytes).
- `internal/docs` is a pure leaf (`go list -deps` shows only itself in the internal closure); `go.mod`/
  `go.sum` byte-identical to HEAD (the assets are embedded data, no new dependency).

**Next:** The IMMEDIATE follow-on (this slice's `Not In Scope` breadcrumb): the two `normal` OpenAPI
contract-accuracy fixes in `internal/openapi/openapi.yaml` + regenerated `openapi.json` twin — remove the
phantom `verify` `index` query param (handler always uses `seqs[0]`) and fix `/{domain}/log/checkpoint`'s
media type `text/plain`→`application/octet-stream` — plus the `low` `healthz` 503 response. Add a
per-operation golden pinning each documented op's params + `200` media type against the handler's real
behavior, since the path-only drift test is structurally blind to params/media-types/responses
(`learnings/openapi.md`). Regenerate the JSON twin deterministically after the YAML edit. That closes the
last M-API contract-fidelity gap and would meet the 4th M-API criterion fully.

**Notes:**
- **CRITICAL no-CDN nuance (do not "fix" into a false positive):** the no-CDN ban runs ONLY over the
  rendered `/docs` HTML body, NEVER over the 2 MB Elements JS bytes. The minified bundle carries hundreds
  of inert baked `http(s)://` example/doc strings (golang.org, example.com, …) that are DATA, not runtime
  fetches; the bundle issues no external request at runtime, and the CSS has no `@import`/external-font
  `url()` (both re-confirmed this iteration). This is documented in `web.go`'s `elementsJS` doc comment and
  in `TestElementsJSServed` so a future reviewer does not run a substring ban over the asset bytes.
- The `/docs` page is a STATIC artifact (`tmpl.Execute(&buf, nil)`, no Identity data), so its masthead uses
  the static proofserve-style chrome (brand + mark + sub, no `{{.Instance}}`/`{{.Operator}}` binding) — the
  `internal/docs` leaf deliberately takes no `dashboard.Identity` to stay import-pure. If a future step wants
  the instance-identity line on `/docs`, that needs an `Identity` parameter (and a `dashboard` import) — a
  deliberate, separate change, not done here.
- The Elements assets are minified third-party bundles (not hand-written source), so `gofmt` does not touch
  them and they join the "verify.wasm pin is fragile" discipline: re-fetch + re-pin deliberately on a
  version bump, never hand-edit (`TestElementsAssetsHashPinned` gates it).
- `.claude/context/target.md` (modified) and the untracked `.claude/adr/0014-…md` are pre-existing artifacts
  from the prior `cid(steer)` commit, NOT touched/authored by this advance — left out of this commit per the
  context-file rule (advance writes only `handoff.md` + source/test). Review may want to confirm ADR-0014
  gets tracked.
