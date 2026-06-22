<!-- area: .github/workflows/ci.yml + .github/workflows/pages.yml -->
<!-- indexed-as: ci.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# CI + Pages workflows (`.github/workflows/`)

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## CI workflow (`.github/workflows/ci.yml`)

- **CI is one `ubuntu-latest` job, `env: CGO_ENABLED: "0"`, that inlines `mise run check` then shells
  the `notecheck` oracle.** The three inlined lines (`go build ./...` / `go vet ./...` / `go test
  ./...`) are byte-identical to `mise.toml [tasks.check]` (`… && … && …`); GitHub Actions runs `run:`
  blocks with `bash -e` by default, so a non-zero `go build`/`go vet` aborts the step exactly like the
  `&&` short-circuit — behaviorally equivalent. `cauldron/` is gitignored so `go build ./...` on a fresh
  CI checkout is safe (the cauldron build pitfall never reaches CI). All five `next.md` local
  reproductions pass: ci.yml is valid YAML (validated via the cached `gopkg.in/yaml.v3` — PyYAML is
  absent locally), accept → `OK sb0.iscc.id/log` exit 0, reject(corrupted) → exit 1, bad vkey → exit 2,
  the verbatim `run:` block → exit 0, `mise run check` green (11 pkgs uncached), `gofmt -l` clean.
- **The CI `notecheck` reject guard catches every *realistic* green-but-wrong oracle, but its
  robustness leans on the downstream process draining stdin.** The guard is `if sed 's/QLdEY/QLdEZ/'
  … | ./notecheck …; then echo ERROR; exit 1; fi` under `set -o pipefail`. The real binary reads all of
  stdin via `io.ReadAll` *before* deciding, so on a corrupted checkpoint the pipeline returns
  notecheck's own exit 1 (guard false → CI continues = correct), and a regression that *accepts*
  corrupted input would return exit 0 (guard true → CI fails = caught — reviewer mutation-proved this
  with a stdin-draining always-accept stub: CI exits 1). The ONE artificial case the guard misses: a
  process that exits 0 *without* reading stdin makes `sed` die of SIGPIPE (141), and `pipefail` then
  reports the pipeline as 141≠0 (guard false → false pass). That is not a behavior a signature verifier
  can exhibit (it must read the checkpoint to verify it), so the gate is sound for the trust root — but
  if a future CI guard ever pipes into a tool that may short-circuit before draining, prefer a temp-file
  + explicit `$?` check over `sed | tool` under `pipefail` to avoid the SIGPIPE masking.

## Pages publish workflow (`.github/workflows/pages.yml`)

- **`pages.yml` is the modern Actions Pages build→deploy of the Surface-C verifier-site** (separate
  file from `ci.yml` so triggers stay independent): `push: [develop]` + `workflow_dispatch`, top-level
  `permissions: {contents:read, pages:write, id-token:write}`, `concurrency: {group:pages,
  cancel-in-progress:false}`; `build` job runs `go run ./cmd/verifier-site -out dist` → `cp
  .github/pages/CNAME dist/CNAME` → configure-pages@v5 → upload-pages-artifact@v3 (`path: dist`);
  `deploy` job (`needs: build`, `environment: github-pages`) → deploy-pages@v4. NO `mise run build:wasm`
  step — the generator COPIES the byte-pinned `verify.wasm` (deployed hash == `web.WasmVerifyHash`,
  re-verified `7d57ab1b…`), so the published artifact is reproducible-from-commit, not a rebuild.
- **The artifact `CNAME` is a NO-OP for the custom domain under Actions-based Pages — the binding lives
  in repo Settings, not in code (Codex P2, reviewer-confirmed mechanism).** With `actions/deploy-pages`
  GitHub ignores a `CNAME` in the uploaded artifact; the custom domain `monitor.iscc.codes` must be set
  in Settings → Pages (or via API) once, AND the Pages source must be switched to "GitHub Actions" — both
  are one-time repo-config steps a workflow file cannot assert. Consequence: until the custom domain is
  configured, the site lands at the default project URL `iscc.github.io/iscc-monitor/` where the page's
  root-absolute `/_ds/...` asset paths break (they resolve against the apex, not the project base path).
  The site is correct ONLY on the apex custom domain. Tracked as a `normal` issue + flagged for the human
  in the handoff; keep the `cp CNAME` step (harmless, documents intent, and is the correct mechanism if
  Pages source is ever switched back to branch-based).
