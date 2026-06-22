## 2026-06-22 — Static-site generator for the Surface-C verifier deploy (`cmd/verifier-site`)

**Done:** Added `cmd/verifier-site`, a thin-main Go program that renders the complete Surface-C static
site into an output directory — `index.html` driven from `verifier.Handler()` over httptest, plus every
`/_ds/` asset (token/font CSS, woff2 binaries, `wasm_exec.js`, the SRI-pinned `verify.wasm`, the logo)
copied from `web.Handler()` at its URL path. The woff2 set is enumerated from the served `fonts.css`
`src:` URLs (not hardcoded), and `generate` fails closed if any handler returns non-200. This is the
generate half of the deploy; the GitHub-Pages publish workflow is the follow-up sub-step.

**Files changed:**
- `cmd/verifier-site/main.go`: new generator — thin `main` (one `-out` flag, default `dist/`, owns the
  single `os.Exit`) over a testable `generate(outDir)`; helpers `render` (GET-over-httptest, errors on
  non-200), `fontPaths` (scans `fonts.css` for `"/_ds/fonts/…"`), `assetPath` (URL path → on-disk path),
  `writeFile` (MkdirAll + WriteFile). Two internal imports only: `internal/verifier`, `internal/web`.
- `cmd/verifier-site/main_test.go`: golden `TestGenerate` — generates into `t.TempDir()`, asserts
  `index.html` non-empty + loader markers (`/_ds/wasm_exec.js`, `/_ds/verify.wasm`, `isccVerifyInclusion`,
  `URLSearchParams`) + no-CDN body (re-asserted on disk), all five named `/_ds/` assets present + non-empty,
  ≥1 `/_ds/fonts/*.woff2`, and the generated `verify.wasm` SHA-256 == `web.WasmVerifyHash`.
- `CLAUDE.md`: added a "Building the Surface-C verifier site" subsection under Development documenting
  `go run ./cmd/verifier-site -out dist`.

**Verification:** `mise run check` → GREEN (all 27 packages ok); `gofmt -l .` empty.
- `go test -count=1 -run TestGenerate ./cmd/verifier-site` → PASS.
- Tree assertion (index.html + 5 named `/_ds/` + 8 woff2) → PASS; `go run ./cmd/verifier-site -out <tmp>`
  exits 0 and writes `index.html` + `_ds/` (verified the full 13-file tree).
- Generated `_ds/verify.wasm` SHA-256 = `7d57ab1b…f22d2c` == `web.WasmVerifyHash` (copy, not rebuild) → PASS.
- No-CDN re-assertion on the generated `index.html` (no `jsdelivr`/`http://`/`https://`/`cdn.`) → PASS.
- Dep closure: only internal imports are `internal/verifier` + `internal/web`; no `database/sql`, no
  network fetch (`net/http` present solely via `net/http/httptest`, which next.md explicitly allows).
- Mutation-proven non-vacuous: renaming a `/_ds/` asset path → `web.Handler` 404s → `generate` aborts →
  `TestGenerate` FAILs with `GET /_ds/renamed.css = 404, want 200`; restored → green. (Note: a bad
  `verifier.Handler` URL does NOT trip fail-closed — Surface C is one static artifact rendered for every
  GET path incl. unknown ones, so it returns 200; the asset-rename mutation is the right fail-closed probe.)

**Next:** Build the GitHub-Pages publish workflow (`.github/workflows/*`) that runs `go run
./cmd/verifier-site -out <dir>` and deploys the tree to `monitor.iscc.codes` — this generator is its build
command. When that lands, fold in the two still-open Surface-C `normal`s the review handoff flagged: the
`readTarget` `u.href` normalization, and the WASM-verifier-scope honesty gap (the core proves inclusion
math only — no checkpoint-signature/did:web-key/id-binding check, but the step copy lists a did:web step).

**Notes:**
- `dist/` is NOT added to `.gitignore` here (out of scope: the test writes to `t.TempDir()`, and the
  deploy-output ignore belongs with the publish-workflow sub-step). `review` may want to add it when the
  workflow lands so a local `go run` doesn't dirty the tree.
- The generator copies the byte-pinned embedded `verify.wasm` (proven by the hash equality), per
  next.md's Not-In-Scope — it never invokes `go build -GOOS=js`. The published artifact hash therefore
  inherits `web.WasmVerifyHash`'s reproducibility (the `-buildvcs=false` build, web.md).
- Oracle/conformance gate is N/A: this is pure HTML/asset assembly from already-golden-tested embedded
  bytes — no signature/RFC-6962/Merkle/did:web/fsck/proof path touched. `verifier.Handler` stays unmounted
  in `cmd/iscc-monitor`'s `buildMux` (Surface C is a different origin); not touched.
