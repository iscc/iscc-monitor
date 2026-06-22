<!-- area: .github/workflows/ci.yml + .github/workflows/pages.yml + .github/workflows/publish.yml -->
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
  excludes `.git`/`cauldron/`/built binaries/`*.db`/`.claude/`/`.github/` etc. The SECRET patterns (`.env`,
  `.env.*`, `**/auth.json`) and WAL/SHM SIDECARS (`*.db-wal`/`*.db-shm` — `*.db`/`*.sqlite*` match NEITHER)
  are now listed (advance `a15a9f4`, the `normal` gap closed for ROOT-level files). General rule still
  holds: a `.dockerignore` for a `COPY . .` Dockerfile should be a superset of the repo's secret
  `.gitignore` lines. **`.dockerignore` matching is NOT `.gitignore` matching** (Codex P2, reviewer-
  confirmed): a slashless pattern (`.env`, `*.db-wal`) matches only the CONTEXT ROOT under Docker's
  `filepath.Match`, whereas `.gitignore` matches the basename at ANY depth — so a nested `deploy/.env` /
  `data/monitor.db-wal` is gitignored but still sent to the build stage. To truly mirror the gitignore you
  must use recursive `**/` forms (`**/.env`, `**/*.db-wal`); `**/auth.json` already does. Latent
  defense-in-depth only (build-STAGE layer, never the final image which only `COPY --from=build`s the
  binary, never CI which has none of these files) — tracked as a `low` issue.
- **The realm var is BAKED via `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` (`Dockerfile:51`) and
  the `docker` smoke job PROVES it by NOT passing `-e ISCC_MONITOR_REALM`** (advance `2f6d50a`): a
  missing/misspelled `ENV` makes `config.Load` exit non-zero, the container never serves, and the
  `/healthz` loop times out → the job FAILs. `ISCC_MONITOR_DB`/`ISCC_MONITOR_ADDR` are still passed (no
  safe image default). Note for `deploy/OPERATING.md`: a Docker **Compose** volume declared `monitor-data:`
  with no `name:`/`external:` is project-PREFIXED at runtime (`<project>_monitor-data`), so a separate
  `docker run -v monitor-data:/data … chown` prep targets a DIFFERENT volume than `docker compose up`
  mounts — pin `name:` or use a Compose-native prep (open `normal`).

## GHCR publish workflow (`.github/workflows/publish.yml`)

- **The image PUBLISH lives in its own file, separate from `ci.yml`'s build+smoke `docker` job** (advance
  `a15a9f4`, mirrors the `pages.yml` separation so triggers + permissions stay independent — the `ci.yml`
  `docker` job stays a pure build+smoke with NO registry login). Shape: `on: push: [develop]` +
  `workflow_dispatch`; top-level `permissions: {contents: read, packages: write}` (the auto-provided
  `GITHUB_TOKEN` + `packages: write` is all the push needs — no PAT); `concurrency: {group: publish,
  cancel-in-progress: false}`. One `ubuntu-latest` job: checkout → derive 7-hex short SHA
  (`echo "short=${GITHUB_SHA::7}" >> "$GITHUB_OUTPUT"`) → `docker/login-action@v3` (ghcr.io, `github.actor`
  + `GITHUB_TOKEN`) → `docker/build-push-action@v6` with hand-written inspectable `tags:`
  (`:develop` + `:sha-<short>`) and `build-args: VERSION=${{ github.sha }}`.
- **`github.sha` cannot empty-expand** (unlike a `$(git rev-parse)` substitution), so the Dockerfile's
  required non-empty `VERSION` build-arg is always satisfied in Actions — this is why the workflow uses
  the Actions context var, not a shell substitution, for the build-arg (sidesteps the `-X` empty-stamp
  class entirely on the CI path). Docker is absent on the dev host, so verify this workflow locally by
  YAML validity + static inspection only (trigger/tags/permissions/build-arg) — the Verify bar asks for
  exactly that.
- **`workflow_dispatch` has no ref guard** (Codex P2, reviewer-confirmed): the job pushes the floating
  `:develop` tag unconditionally, so a manual dispatch from a non-develop ref would publish that branch's
  code as `:develop`. Same pattern as `pages.yml` (intentionally mirrored) — a pre-existing repo
  convention, not a regression, and `workflow_dispatch` is maintainer-only. Hardening opportunity tracked
  as a `normal` issue: guard the publish job/`:develop` tag on `github.ref == 'refs/heads/develop'` (best
  applied to `pages.yml` too for consistency).

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
