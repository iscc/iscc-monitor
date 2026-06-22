## 2026-06-22 — GHCR publish workflow + `.dockerignore` secret/sidecar fold-in + `build:monitor` empty-SHA fail-fast

**Done:** Added `.github/workflows/publish.yml` that, on push to `develop` (plus `workflow_dispatch`),
builds the existing root `Dockerfile` unchanged and pushes `ghcr.io/iscc/iscc-monitor` tagged BOTH
`develop` (floating) AND `sha-<short>` (immutable), passing a non-empty `VERSION=${{ github.sha }}`
build-arg. Folded in the two carried `normal` traps this slice is the natural toucher of: hardened
`.dockerignore` to mirror the gitignore's secret/sidecar never-commit set, and split the host
`build:monitor` SHA lookup behind a `[ -n "$sha" ]` gate so a git failure aborts with no binary
instead of stamping an empty `/version`.

**Files changed:**
- `.github/workflows/publish.yml` (new): push-to-`develop` + `workflow_dispatch` trigger; top-level
  `permissions: {contents: read, packages: write}`; `concurrency: {group: publish}` (mirrors
  `pages.yml`'s shape); one `ubuntu-latest` `publish` job: checkout → derive 7-hex short SHA
  (`${GITHUB_SHA::7}`) → `docker/login-action@v3` (ghcr.io, `github.actor` + `GITHUB_TOKEN`) →
  `docker/build-push-action@v6` with hand-written inspectable `tags:` (`:develop` + `:sha-<short>`)
  and `build-args: VERSION=${{ github.sha }}`.
- `.dockerignore`: added `*.db-wal`, `*.db-shm` (the `*.db`/`*.sqlite*` globs match neither) and the
  secret set `.env`, `.env.*`, `**/auth.json`, each with a rationale comment. Now a superset of the
  repo `.gitignore`'s never-commit secret lines.
- `mise.toml` (`build:monitor` task only): rewrote `run` to
  `sha=$(git rev-parse --short HEAD) && [ -n "$sha" ] && go build -ldflags "-X …Version=$sha" -o …`
  so an empty SHA fails fast (non-zero exit, no binary). `build`/`check`/`vet`/`test`/`build:wasm`
  unchanged and still git-free.

**Verification:** `mise run check` → green (all 28 packages `ok`, build + vet clean; no Go source
touched). `gofmt -l .` → empty outside `cauldron/`. Per-criterion:
- [x] `publish.yml` parses as valid YAML (cached `gopkg.in/yaml.v3` snippet, the `learnings/ci.md`
  pattern) — `on.push.branches` contains `develop`; top-level `permissions.packages == write`
  (`contents: read`); tag set contains BOTH literal `ghcr.io/iscc/iscc-monitor:develop` AND a
  `ghcr.io/iscc/iscc-monitor:sha-`-prefixed tag; a non-empty `VERSION=${{ … }}` build-arg is passed.
  All contract checks PASS.
- [x] `.dockerignore` lists `.env`, `.env.*`, `**/auth.json`, `*.db-wal`, `*.db-shm` — each
  `grep -qxF` PRESENT.
- [x] `build:monitor` fail-fast holds: empty `sha=$(true)` → `aborted: empty sha`, exit 1, build step
  never runs; non-empty `sha=abc1234` → `built`. The real task on this host (git present) builds the
  26 MB binary, exit 0, and stamps the real short SHA `e0ea36c` — happy path intact.

**Next:** The remaining cheap M-Deploy slices are still open (none code-blocked): the canonical
`deploy/realm-testnet.txt`, the operability/deployment doc, and the root `README.md` (which
`target.md` "Done When" requires before DONE). Suggest `deploy/realm-testnet.txt` next — it is the
smallest, and the Dockerfile's baked realm (`internal/registry/testdata/realm.txt`) can then point at
the canonical file in a follow-up.

**Notes:**
- **Docker is absent on this host (CI-only, per `learnings/ci.md`)**, so the `docker push` itself
  cannot run locally — I verified `publish.yml` exactly as the Verify bullet asks: YAML validity +
  static inspection of trigger / tags / permissions / build-arg. The `.dockerignore` and
  `build:monitor` fixes ARE fully tested on this host.
- Used `docker/build-push-action@v6` + `docker/login-action@v3` (the in-repo-idiomatic explicit form
  the Verify bullet prefers) over a plain `docker build && docker push` pair. Both tags are
  hand-written literals (no `metadata-action`), so the tag template is directly inspectable.
- Action version pins match `ci.yml`/`pages.yml` for consistency (`checkout@v4`); the deprecated
  Node-20 major bumps are a separate `low` issue, deliberately not chased here.
- The `critical` GHCR issue ("Publish a deployable container image to GHCR") is now fully code-closed:
  the Dockerfile + build-smoke half landed last window, this slice is the push half. The two `normal`
  traps (`.dockerignore` secret/sidecar gap; `build:monitor` empty-SHA) are both closed.
- `ci.yml`'s `docker` job is left a pure build+smoke with no registry login (per Not-In-Scope); publish
  lives in its own file so triggers + permissions stay independent (mirrors the `pages.yml` separation).
- No Go source, `go.mod`/`go.sum`/`schema.sql`, or trust-root/proof/signature path touched — oracle /
  conformance gate is N/A for this slice (packaging + task-config only).
