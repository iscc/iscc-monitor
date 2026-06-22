# Next Work Package

## Step: GitHub-Pages publish workflow for the Surface-C verifier (`monitor.iscc.codes`)

## Advances
WASM verifier milestone — target.md: *"plus the standalone **Independent Verification** verifier app
(… Surface C) at `monitor.iscc.codes` (monitor-agnostic via `?monitor=<url>`); reproducible build +
published hash + SRI pin (ADR-0003, ADR-0010). **Verify:** … the verifier artifact hash matches the
published value …"*

This is the **front-of-queue open Verify** and the explicit `review` handoff `**Next:**`: "Build the
GitHub-Pages publish workflow (`.github/workflows/*`) that runs `go run ./cmd/verifier-site -out <dir>`
and deploys the tree to `monitor.iscc.codes` — this generator is its build command." state.md's
DRIFT WATCH (amber) says the next increment must **close** a Verify criterion, not add more build
plumbing around the still-unpublished artifact — this is that closer: the reproducible build command
(`cmd/verifier-site`) exists; this step is the missing PUBLISH half that makes the artifact actually
deployed (the only remaining step to a live, hash-published Surface-C artifact).

## Goal
Add a GitHub Actions workflow that runs the existing `cmd/verifier-site` generator and publishes the
rendered static tree to GitHub Pages at `monitor.iscc.codes`, so the verifier app is a *deployed*,
public, reproducible-from-commit artifact (ADR-0003 "Pages-from-repo ties the deployed WASM to a public
commit") — closing the "published" half of the WASM milestone's Verify bar.

## Scope
- **Create**: `.github/workflows/pages.yml` — the Pages build+deploy workflow (the publish half).
- **Create**: `.github/pages/CNAME` — a tracked file containing exactly `monitor.iscc.codes` (+ newline)
  that the workflow copies into the published tree so Pages serves the custom domain (ADR-0003 `.codes` =
  code). Keep it under `.github/pages/`, NOT a bare repo-root `CNAME` where tooling might trip on it.
- **Modify**: `CLAUDE.md` — the WASM/surfaces section documents the surfaces; add a one-line note that
  `monitor.iscc.codes` is published by `.github/workflows/pages.yml` (with `cmd/verifier-site` as its
  build command). Minimum needed to keep docs in sync (this is the only non-test/doc file; well within 3).
- **Reference**:
  - `cmd/verifier-site/main.go` — the build command this workflow invokes (`go run ./cmd/verifier-site
    -out <dir>`, exits 0, writes the 14-file tree; default out dir is `dist`; fails closed on any non-200).
  - `.github/workflows/ci.yml` — the existing workflow's shape to mirror: `runs-on: ubuntu-latest`,
    `actions/checkout@v4`, `actions/setup-go@v5` with `go-version: "1.26"`, `env: CGO_ENABLED: "0"`.
  - `.claude/adr/0003-client-verification-and-in-browser-verifier.md` (lines 51–74) — verifier hosting:
    independent origin `monitor.iscc.codes`, GitHub-Pages-from-repo, reproducible-build + published-hash.
  - `.claude/context/learnings/verifier-site.md` — the generator's contract (fail-closed on a non-200,
    copies the byte-pinned `verify.wasm` whose SHA-256 == `web.WasmVerifyHash`, non-atomic-output `low`).
  - `.claude/context/learnings/ci.md` — CI workflow conventions (one `ubuntu-latest` job,
    `CGO_ENABLED: "0"`; PyYAML is absent locally so validate YAML via the cached `gopkg.in/yaml.v3`).

## Not In Scope
- Do **not** modify `cmd/verifier-site/main.go` — it is the finished build command; this step only
  *invokes* it from CI. (Its non-atomic-output `low`, the Surface-C `readTarget`, and the WASM-scope
  `normal`s are separate later steps; folding them in here would blur the publish-workflow change.)
- Do **not** add the dossier tier-2 WASM caller — that is the *other* WASM sub-step, a separate ≤3-file
  increment; pick one front. This step lands the publish workflow.
- Do **not** touch `.github/workflows/ci.yml`; the publish workflow is a new, separate file so CI and
  Pages have independent triggers and the existing gate stays unchanged.
- Do **not** add the missing `safeStamp` OTS guard, the §5 digest binding, or any `host:port`-DID work —
  those are unrelated `normal` issues touched only when their exact lines are next edited.

## Implementation Notes
- **Workflow shape** — use the modern GitHub-Pages Actions deploy (no `gh-pages` branch). One job that
  builds the artifact + uploads it, and a second that deploys it, gated to the default branch:
  - `name`, `on: push: branches: [develop]` (Pages publishes from the active branch — the repo's default
    here is `develop` per the git state) plus `workflow_dispatch` for manual runs. Do NOT trigger on PRs.
  - Top-level `permissions: { contents: read, pages: write, id-token: write }` and
    `concurrency: { group: "pages", cancel-in-progress: false }` (the canonical Pages concurrency).
  - **build job** (`runs-on: ubuntu-latest`, `env: CGO_ENABLED: "0"`): `actions/checkout@v4` →
    `actions/setup-go@v5` (`go-version: "1.26"`) → `go run ./cmd/verifier-site -out dist` (the generator
    fails closed → a non-zero exit aborts the deploy, so a broken render is never published — the
    fail-closed contract from `verifier-site.md`) → `cp .github/pages/CNAME dist/CNAME` →
    `actions/configure-pages@v5` → `actions/upload-pages-artifact@v3` with `path: dist`.
  - **deploy job** (`needs: build`, `environment: { name: github-pages, url: ${{
    steps.deployment.outputs.page_url }}}`): `actions/deploy-pages@v4` (`id: deployment`).
  - Pin action **major** tags as above (matches the project's `@v4`/`@v5` style in `ci.yml`). These
    `actions/*-pages` versions are the current canonical set; keep the four-action build→deploy shape.
- **CNAME** — content is exactly `monitor.iscc.codes` + a trailing newline, nothing else (ADR-0003 the
  custom domain). It must land at the *root* of the published artifact (`dist/CNAME`) so Pages applies the
  custom domain. Keep `CNAME` as a tracked repo file (`.github/pages/CNAME`) the workflow `cp`s into
  `dist` after `go run` — do NOT make `cmd/verifier-site` write it (keep the generator a pure renderer of
  handler output, per `verifier-site.md`'s one-source-of-truth rule).
- **Reproducibility / published hash (ADR-0003).** The published `verify.wasm` inherits its byte-pinned
  hash from the generator (it *copies* the embedded blob whose SHA-256 == `web.WasmVerifyHash`, per
  `verifier-site.md`); the workflow must NOT rebuild the WASM (no `mise run build:wasm` step) — it only
  renders + uploads, so the deployed artifact hash equals the committed, golden-tested value. This is what
  "the verifier artifact hash matches the published value" means: build-from-commit, copy-not-rebuild.
- **Relevant learnings rule:** `ci.md` — the existing CI job uses `CGO_ENABLED: "0"` and pins
  `actions/checkout@v4` / `actions/setup-go@v5` / `go-version: "1.26"`; mirror these. PyYAML is absent
  locally, so validate the new YAML with the cached `gopkg.in/yaml.v3` (see Verification), not `python3 -c
  "import yaml"`. `verifier-site.md` — the generator is fail-closed (non-200 → abort), so a `go run` that
  exits 0 guarantees the complete 14-file tree; the workflow needs no extra completeness assertion.

## Verification
- `mise run check` is green (the workflow file does not touch the Go build, but confirm nothing else
  regressed): `go build ./... && go vet ./... && go test ./...` all pass, `gofmt -l .` empty.
- The new workflow is valid YAML — parse it with the cached `gopkg.in/yaml.v3` (PyYAML is absent
  locally): a tiny throwaway `go run` of a `yaml.Unmarshal([]byte(read .github/workflows/pages.yml),
  &map[string]any{})` exits 0 with no error.
- `go run ./cmd/verifier-site -out /tmp/pages-verify` exits 0 and writes the full **14-file** tree
  (`index.html` + `_ds/{tokens.css,fonts.css,wasm_exec.js,verify.wasm,iscc-logo-black.png}` + 8 woff2) —
  the workflow's build step reproduced locally: `find /tmp/pages-verify -type f | wc -l` prints `14`.
- The deployed-tree `verify.wasm` is the byte-pinned artifact, not a rebuild: `sha256sum
  /tmp/pages-verify/_ds/verify.wasm` matches `web.WasmVerifyHash` (the generator copies it; the existing
  `TestGenerate` already pins this — re-confirm it still passes).
- The `CNAME` content is exactly `monitor.iscc.codes`: `grep -qx "monitor.iscc.codes"
  .github/pages/CNAME` exits 0, and the workflow `cp`s it to `dist/CNAME`.
- The workflow declares the Pages permissions and the build→deploy job pair: `grep -q "pages: write"
  .github/workflows/pages.yml` and `grep -q "deploy-pages" .github/workflows/pages.yml` both exit 0.

## Done When
`mise run check` is green, `.github/workflows/pages.yml` parses as valid YAML and declares the Pages
build→deploy job pair (`pages: write` + `deploy-pages`) running `go run ./cmd/verifier-site`, the
generator reproduces the 14-file tree locally with `verify.wasm`'s SHA-256 == `web.WasmVerifyHash`, and
a tracked `CNAME` containing exactly `monitor.iscc.codes` is copied into the published artifact.
