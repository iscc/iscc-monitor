## 2026-06-22 — GitHub-Pages publish workflow for the Surface-C verifier (`monitor.iscc.codes`)

**Done:** Added `.github/workflows/pages.yml` (the build→deploy publish half) that runs the existing
`go run ./cmd/verifier-site -out dist` generator and deploys the rendered tree to GitHub Pages at
`monitor.iscc.codes`, plus a tracked `.github/pages/CNAME` the workflow copies to `dist/CNAME` for the
custom domain. This closes the "published" half of the WASM milestone's Verify bar: the artifact is now
deployed-from-commit (copy-not-rebuild), so the deployed `verify.wasm` hash equals the golden-pinned
`web.WasmVerifyHash`.

**Files changed:**
- `.github/workflows/pages.yml` (new): modern GitHub-Pages Actions deploy — `on: push: [develop]` +
  `workflow_dispatch` (no PR trigger); top-level `permissions: {contents: read, pages: write, id-token:
  write}`; `concurrency: {group: "pages", cancel-in-progress: false}`. `build` job (`ubuntu-latest`,
  `CGO_ENABLED: "0"`): checkout@v4 → setup-go@v5 (`go-version: "1.26"`) → `go run ./cmd/verifier-site
  -out dist` → `cp .github/pages/CNAME dist/CNAME` → configure-pages@v5 → upload-pages-artifact@v3
  (`path: dist`). `deploy` job (`needs: build`, `environment: github-pages`): deploy-pages@v4
  (`id: deployment`).
- `.github/pages/CNAME` (new): exactly `monitor.iscc.codes` + one trailing newline (19 bytes, od -c
  confirmed).
- `CLAUDE.md`: one-line note in the "Building the Surface-C verifier site" section that
  `.github/workflows/pages.yml` is the publish workflow (build command `cmd/verifier-site`, custom
  domain via the tracked CNAME). The one allowed non-test/doc edit.

**Verification:** `mise run check` → GREEN, all 27 packages ok (incl. `cmd/verifier-site`); `gofmt -l .`
empty after `mise run fmt`. Per-criterion:
- [x] `mise run check` green (build + vet + test); nothing regressed (workflow file is outside the Go build).
- [x] `pages.yml` is valid YAML — parsed via cached `gopkg.in/yaml.v3` `Unmarshal` into `map[string]any`,
  exit 0, 5 top-level keys (name/on/permissions/concurrency/jobs). PyYAML absent locally, used yaml.v3 per `ci.md`.
- [x] `go run ./cmd/verifier-site -out /tmp/pages-verify` exits 0 and writes the full 14-file tree
  (`find … | wc -l` = 14: index.html + 5 named `/_ds/` assets + 8 woff2).
- [x] Deployed-tree `verify.wasm` SHA-256 = `7d57ab1b…f22d2c` == `web.WasmVerifyHash` (web.go:93) — copy,
  not rebuild; the workflow has no `mise run build:wasm` step. `TestGenerate` (uncached) re-pins this: PASS.
- [x] CNAME content exactly `monitor.iscc.codes` (`grep -qx` exit 0); workflow `cp`s it to `dist/CNAME`.
- [x] `grep -q "pages: write"` and `grep -q "deploy-pages"` both exit 0; build command `go run
  ./cmd/verifier-site` present.

**Next:** The "published" half of the Surface-C WASM milestone is now in place; the front-of-queue items
that remain are the still-open Surface-C `normal`s and the new `low`: (1) the WASM-verifier honesty gap
(the proof core verifies inclusion math only — no checkpoint-signature / did:web-key / id-binding check,
yet the step copy lists a did:web step); (2) the `verifier.readTarget` `u.href` normalization; (3) fold
in the `cmd/verifier-site` non-atomic-output fix (stage to temp dir + rename). The OTHER WASM sub-step
(the dossier tier-2 WASM caller) is also still open. Pick one front next.

**Notes:**
- `dist/` deliberately left OUT of `.gitignore` (matches the prior review note: the test uses
  `t.TempDir()`; the deploy output never lands in the repo tree since CI builds it fresh). Working tree
  stayed clean — only the three intended paths are modified/untracked.
- Action versions are the current canonical Pages set (`configure-pages@v5`, `upload-pages-artifact@v3`,
  `deploy-pages@v4`) pinned at major tags, matching the `@v4`/`@v5` style in `ci.yml`. Did NOT touch
  `ci.yml` — the publish workflow is a separate file so CI and Pages keep independent triggers.
- The workflow triggers on `push: [develop]` (the repo's active/default branch per git state, per the
  step's Implementation Notes) — NOT on PRs and NOT on `main`. GitHub Pages must be configured to
  "GitHub Actions" source (one-time repo Settings step) for `deploy-pages@v4` to publish; this is repo
  config, not something a workflow file can assert. Worth a human confirming Pages source + the
  `monitor.iscc.codes` DNS CNAME are set on first deploy.
- Oracle/conformance gate — N/A: this increment touches no signature/RFC-6962/Merkle/did:web/fsck/proof
  code (CI plumbing + a domain file + a doc line); the `verify.wasm` it publishes is the already-golden
  byte-pinned blob, hash re-verified equal here.
