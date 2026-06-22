# Next Work Package

## Step: Static-site generator for the Surface-C verifier deploy (`cmd/verifier-site`)

## Advances
WASM verifier milestone — the still-open Verify front:
> "the standalone **Independent Verification** verifier app … (Surface C) at `monitor.iscc.codes`
> (monitor-agnostic via `?monitor=<url>`); reproducible build + published hash + SRI pin. **Verify:**
> identical vectors yield identical verdicts (WASM vs server); the verifier artifact hash matches the
> published value; a `(size, root)` mismatch renders the guided split-view alert, not a dead error."

State (state.md lines 32-37) and the review handoff both name this as the front-of-queue sub-step: the
client-side target gating just landed, so `internal/verifier` is now a single static artifact — but
**"only `.github/workflows/ci.yml` exists — no Pages publish workflow"** and there is no way to
materialize the deployable site (`index.html` + `/_ds/` assets) from the handlers. This step builds the
load-bearing, golden-testable **generator** that the GitHub-Pages publish step (a later sub-step) will
invoke. The review handoff `**Next:**` is exactly: "generate the static `index.html` (render
`verifier.Handler` once to a file) + the `/_ds/` assets and publish to GitHub Pages via a
`.github/workflows/*`." This step does the generate half (mechanically verifiable); the workflow YAML is
the follow-up sub-step.

## Goal
Add a small `cmd/verifier-site` Go program that renders the complete Surface-C static site
(`index.html` from `verifier.Handler()` + every `/_ds/` asset from `web.Handler()`) into an output
directory, so the `monitor.iscc.codes` GitHub-Pages deploy has a single reproducible build command. The
generator is the deployable-artifact core; without it the deploy cannot be assembled or tested.

## Scope
- **Create**: `cmd/verifier-site/main.go` — the static-site generator.
- **Create**: `cmd/verifier-site/main_test.go` — golden test (generates into a temp dir, asserts the
  tree).
- **Modify**: `CLAUDE.md` — add a short "Building the Surface-C verifier site" subsection under
  "Development" documenting the generate command (one paragraph; the usage this step introduces).
- **Reference**:
  - `.claude/context/learnings/verifier.md` (Surface-C handler shape, `/_ds/` literals, client-side
    target, no-CDN ban — Read before writing).
  - `.claude/context/learnings/web.md` (`web.Handler` `/_ds/` SUBTREE mount, the five exact asset
    paths + `fonts/*.woff2`, `noExternalCDN` rules, the `verify.wasm` SRI pin — Read before writing).
  - `.claude/context/learnings/cmd-monitor.md` (thin-main idiom: `cmd/iscc-monitor` owns the one
    `os.Exit`; mirror that structure).
  - `internal/verifier/handler.go` (`verifier.Handler()` — GET-only, no-arg, renders `index.html`).
  - `internal/web/web.go` (`web.Handler()`, `web.Prefix`, `TokensPath`/`FontsCSSPath`/`WasmExecPath`/
    `WasmVerifyPath`/`LogoPath`, exported `web.TokensCSS` + `web.WasmVerifyHash`; fonts are embedded but
    NOT exported as a list — derive font paths from `fonts.css` `src:` URLs the way `web_test.go`
    `TestFontsCSSReferencesEmbeddedSubsets` does).
  - `internal/web/web_test.go` (the existing `src:`-URL-extraction + "exactly 8 woff2" pattern to
    reuse for enumerating fonts).

## Not In Scope
- **The GitHub-Pages publish workflow itself** (`.github/workflows/*` to deploy to
  `monitor.iscc.codes`). That is the next sub-step and depends on this generator; do not add or edit a
  workflow YAML here.
- Mounting `verifier.Handler` in `cmd/iscc-monitor`'s `buildMux` — Surface C deliberately ships on a
  different origin (verifier.md). Keep it unmounted.
- The WASM-verifier-scope signature/id-binding gap, the `readTarget` `u.href` normalization, and the
  `safeIndex`-to-`verifyadapter` move (all open `normal`s) — those wait for a WASM-core touch, not this
  HTML-assembly step.
- Rebuilding `verify.wasm` or re-pinning `WasmVerifyHash` — the generator COPIES the already-built,
  byte-pinned embedded asset; it does not invoke `go build -GOOS=js`.
- Any new store read / projection or `/` sub-region parity — unrelated to this step.

## Implementation Notes
- **Thin main, mirror `cmd/iscc-monitor`.** `main()` parses one flag `-out <dir>` (default e.g.
  `dist/`), calls a pure `generate(outDir string) error`, prints what it wrote, and owns the single
  `os.Exit(1)` on error. Keep `generate` package-private but testable (same package as the test) so the
  test calls it directly into `t.TempDir()` — do not shell out.
- **Render `index.html` via the real handler, not by re-embedding the template.** Drive
  `verifier.Handler()` with an `httptest.NewRecorder()` + `httptest.NewRequest(http.MethodGet, "/",
  nil)`, assert `rec.Code == 200`, and write `rec.Body` to `<out>/index.html`. This guarantees the
  deployed page is byte-identical to what the golden tests already gate (the client-side loader, the
  no-CDN body, the honest baseline) — no second source of truth.
- **Materialize `/_ds/` assets via `web.Handler()` over httptest, one GET per path.** The five exact
  paths are `web.TokensPath`, `web.FontsCSSPath`, `web.WasmExecPath`, `web.WasmVerifyPath`,
  `web.LogoPath`; plus the woff2 binaries under `web.Prefix + "fonts/"`. For each, issue a GET, assert
  200, and write the body to `<out>` at the URL path — `web.Prefix` (`/_ds/`) becomes a real
  subdirectory (`<out>/_ds/tokens.css`, `<out>/_ds/verify.wasm`,
  `<out>/_ds/fonts/readex-pro-400.woff2`, …). Create parent dirs with `os.MkdirAll`. Use the URL path
  verbatim so the on-disk layout matches what the page fetches at runtime — GitHub Pages serves files at
  their path.
- **Enumerate fonts from `fonts.css`, do not hardcode the 8 names.** Fetch `web.FontsCSSPath` first,
  extract each `url("/_ds/fonts/<name>.woff2")` `src:` path (same regex/scan as
  `web_test.go`'s `TestFontsCSSReferencesEmbeddedSubsets`), and GET each — so a future font add/remove
  flows through without editing the generator. This keeps `fonts.css` the single source of truth.
- **No CDN literals leak.** The generator writes only bytes the handlers already produce (golden-tested
  CDN-free), so the on-disk `index.html` inherits the no-CDN guarantee; the test re-asserts it on the
  generated file for defense in depth.
- **Correctness rule (learnings.md, always-loaded "`proof/verify` is pure" + verifier.md):** the
  generator is a pure-stdlib + two-internal-import leaf (`internal/verifier`, `internal/web`,
  `net/http/httptest`, `os`, `path/filepath`, `flag`, `regexp`/`strings`). Do not add a network fetch,
  a `database/sql` import, or a third internal dep — it assembles from embedded bytes only.
- **Edge case:** if `web.Handler()` returns non-200 for any expected path (a future asset rename),
  `generate` must error, not write a partial site — fail closed so a broken deploy is caught in the test
  and in CI, not in production.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`; `gofmt -l .` empty).
- `go test -count=1 -run TestGenerate ./cmd/verifier-site` passes.
- The golden test generates into `t.TempDir()` and asserts: `<out>/index.html` exists, is non-empty,
  and contains the client-side loader markers (`/_ds/wasm_exec.js`, `/_ds/verify.wasm`,
  `isccVerifyInclusion`, `URLSearchParams`); `<out>/_ds/tokens.css`, `<out>/_ds/fonts.css`,
  `<out>/_ds/wasm_exec.js`, `<out>/_ds/verify.wasm`, `<out>/_ds/iscc-logo-black.png` all exist and are
  non-empty; at least one `<out>/_ds/fonts/*.woff2` exists; and the generated `index.html` contains no
  `jsdelivr` / `http://` / `https://` / `cdn.` substring (inherited no-CDN, re-asserted).
- The generated `<out>/_ds/verify.wasm` SHA-256 equals `web.WasmVerifyHash` (the deployed WASM is the
  byte-pinned artifact — proves the generator copies, not rebuilds).
- `go run ./cmd/verifier-site -out <tmp>` exits 0 and the directory contains `index.html` + `_ds/`.

## Done When
`mise run check` is green and `go test -run TestGenerate ./cmd/verifier-site` passes, with the generator
materializing `index.html` (byte-identical to `verifier.Handler`'s output) plus every `/_ds/` asset
(including the SRI-pinned `verify.wasm`) into the output directory — the deployable Surface-C site the
Pages workflow will publish next.
