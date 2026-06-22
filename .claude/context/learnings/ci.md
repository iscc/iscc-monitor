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

## Production Dockerfile + `docker` CI job (`Dockerfile`, `.dockerignore`, `ci.yml` job `docker`)

- **The image build is a sibling `docker` job, NOT folded into `check`** (keeps the gate fast + the image
  build parallel). It is the ONLY place the container is actually built/run — `docker` is absent on the dev
  host, so every container-run Verify item (`docker build`/`docker run`/`docker image inspect`) is CI-only.
  Verify the host-independent half locally instead: the exact image build command
  `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X …version.Version=<sha>" -o /iscc-monitor
  ./cmd/iscc-monitor` produces a STATICALLY-linked, stripped ELF (`file` → "statically linked", `ldd` →
  "not a dynamic executable"), so `distroless/static-debian12:nonroot` (no libc, uid 65532, CA roots) runs
  it; ~19 MB binary + ~2 MB base ≈ under the 30 MB target. `/healthz` pings only the store (not the hubs),
  so the smoke test returns 200 even with no CI egress to the realm hubs.
- **The VERSION build-arg is REQUIRED and guarded inside the build RUN** with
  `[ -n "$VERSION" ] || { echo …; exit 1; } && go build …` — `ARG VERSION` is deliberately NOT defaulted
  (`=dev` would mask the empty-stamp trap). This is the image-side fix for the `-X` empty-stamp trap
  (`learnings/version.md`): an empty `-X …Version=` clobbers the `dev` default, so the guard fails the
  stage rather than shipping a blank `/version`. The CI job passes
  `--build-arg VERSION="$(git rev-parse --short HEAD)"` computed on the HOST (the `.git` dir is excluded
  from the build context, so the SHA must come via build-arg, never read inside the image). The HOST
  `mise.toml build:monitor` task is a SEPARATE consumer of the same `-X` path and is still unguarded — its
  `normal` issue stays open; this slice only hardened the image.
- **`.dockerignore` keeps the context lean but must MIRROR the repo `.gitignore`'s never-commit set.** It
  excludes `.git`/`cauldron/`/built binaries/`*.db`/`.claude/`/`.github/` etc. Two gaps to close when this
  area is next touched (open `normal` issue): the gitignored SECRET patterns (`.env`, `.env.*`,
  `**/auth.json`) and the WAL/SHM SIDECARS (`*.db-wal`/`*.db-shm` — `*.db`/`*.sqlite*` match NEITHER) are
  not excluded, so `COPY . .` would bake a developer's local secret/state into the build-STAGE layer (never
  the final image — it only `COPY --from=build`s the binary; never CI — a fresh checkout has none). General
  rule: a `.dockerignore` for a `COPY . .` Dockerfile should be a superset of the repo's secret `.gitignore`
  lines, not just the large/breaks-the-build ones.

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
