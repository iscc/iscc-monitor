# Next Work Package

## Step: GHCR publish workflow + fold in the `.dockerignore` secret/sidecar + `build:monitor` empty-SHA fixes

## Advances
`target.md` **M-Deploy — Packaged & operable instance**, the second Verify bullet:

> a **publish workflow** pushes `ghcr.io/iscc/iscc-monitor` on push to `develop`, tagged BOTH `develop`
> (floating) AND `sha-<short>` (immutable, for pin/rollback) — asserted by inspecting the workflow trigger
> + tag template;

This also closes the open half of the `critical` issue **"Publish a deployable container image to GHCR
(Dockerfile + push workflow)"** — the Dockerfile + container-build half landed last window; this slice is
the **push** half it was always paired with. It is the front-of-queue code-closable M-Deploy work, the
loop's standing source of progress while feature milestones (M-UI/WASM/OTS) are design- or human-blocked.

## Goal
Add `.github/workflows/publish.yml` that, on push to `develop`, builds the existing root `Dockerfile` and
pushes `ghcr.io/iscc/iscc-monitor` tagged BOTH `develop` and `sha-<short>`, so iscc-infra can pull a
pinnable image instead of building on the box. Fold in the two carried `normal` traps this slice is the
natural toucher of: harden `.dockerignore` to mirror the gitignore's never-commit secret/sidecar set, and
make the host `build:monitor` git-SHA lookup fail fast instead of stamping an empty `/version`.

## Scope
- **Create**: `.github/workflows/publish.yml`
- **Modify**: `.dockerignore`, `mise.toml` (the `build:monitor` task). *(2 non-test/doc edits + 1 new
  file; no Go source touched — the Dockerfile that this workflow builds already exists and is unchanged.)*
- **Reference**:
  - `/workspace/iscc-monitor/.github/workflows/ci.yml` — the `docker` job's exact build command + the
    sibling-job pattern to imitate (`docker build --build-arg VERSION="$(git rev-parse --short HEAD)" …`).
  - `/workspace/iscc-monitor/.github/workflows/pages.yml` — the in-repo precedent for a separate workflow
    file with top-level `permissions:` + `concurrency:` (mirror its shape for `packages: write`).
  - `/workspace/iscc-monitor/Dockerfile` — the artifact published; note `VERSION` is a REQUIRED,
    fail-fast-on-empty build-arg (the workflow MUST pass a non-empty `--build-arg VERSION`).
  - `/workspace/iscc-monitor/.gitignore` — the authoritative never-commit set the `.dockerignore` must
    become a superset of: `.env`, `.env.*`, `**/auth.json` (secrets), plus the WAL/SHM sidecars
    `*.db-wal`/`*.db-shm` (the existing `*.db`/`*.sqlite*` rules match NEITHER).
  - `/workspace/iscc-monitor/.claude/context/learnings/ci.md` — Read before editing: the CI-job
    mechanics, the `.dockerignore`-must-mirror-`.gitignore` rule, the `-X` empty-stamp image-side fix, and
    the note that the host `build:monitor` is a SEPARATE still-unguarded consumer of the same path.
  - `/workspace/iscc-monitor/.claude/context/learnings/version.md` — the `-X` empty-stamp trap detail
    (read for the `build:monitor` fail-fast fold-in).

## Not In Scope
- **Making the GHCR package public / issuing a `read:packages` token / DNS / Caddy** — explicitly
  iscc-infra's human/infra work per `target.md` M-Deploy "Out of the loop's scope"; they never gate DONE
  here. The workflow only needs the auto-provided `GITHUB_TOKEN` + `packages: write` to push.
- **The canonical `deploy/realm-testnet.txt` + the operability/deployment doc + the root `README.md`** —
  separate M-Deploy slices (state Next priorities 2-3); do NOT start them here. Leave the Dockerfile's
  baked realm pointing at `internal/registry/testdata/realm.txt` for now.
- **Touching the Dockerfile** — it already builds correctly and was just reviewed; the publish workflow
  builds it as-is. Do not change its stages, base image, or build args.
- **Bumping the deprecated Node-20 action majors** (`checkout@v4`, `setup-go@v5`) — that is a separate
  `low` issue scoped to `pages.yml`; match the version pins already used in `ci.yml`/`pages.yml` for
  consistency, do not chase the bump in this slice.
- **Adding a `docker push` to the existing `ci.yml` `docker` smoke job** — keep publish in its own
  workflow file (mirrors the `pages.yml` separation so triggers + permissions stay independent); the
  `ci.yml` `docker` job stays a pure build+smoke with no registry login.

## Implementation Notes
- **Workflow shape** (mirror `pages.yml`'s separate-file structure):
  - `on: push: branches: [develop]` (plus `workflow_dispatch` for manual re-runs, matching `pages.yml`).
    Per the Verify bullet the trigger MUST be push-to-`develop`.
  - Top-level `permissions: { contents: read, packages: write }` — `packages: write` is what lets the
    auto-provided `GITHUB_TOKEN` push to GHCR; no PAT needed.
  - One `ubuntu-latest` job: checkout → `docker/login-action@v3` (registry `ghcr.io`, `username:
    ${{ github.actor }}`, `password: ${{ secrets.GITHUB_TOKEN }}`) → a build-push step. **Prefer the
    explicit `docker/build-push-action@v6`** with a hand-written `tags:` list — it is the
    in-repo-idiomatic, inspectable form the Verify bullet asks for ("asserted by inspecting the workflow
    trigger + tag template"). A plain `docker build … && docker push …` pair is an acceptable equivalent
    if you keep the two literal tags hand-written and inspectable.
  - **Both tags, image lowercase:** `ghcr.io/iscc/iscc-monitor:develop` AND
    `ghcr.io/iscc/iscc-monitor:sha-<short>`. The Verify says `sha-<short>` (immutable, for pin/rollback) —
    derive a short SHA (e.g. a `run:` step exporting `echo "short=${GITHUB_SHA::7}" >> "$GITHUB_OUTPUT"`,
    or `docker/metadata-action`'s `type=sha,prefix=sha-,format=short`) so the immutable tag is
    `sha-<7hex>`, not the full 40-char SHA. GHCR repo paths must be lowercase — `iscc/iscc-monitor`
    already is, so keep the literal lowercase, no `tr` needed.
  - **Pass the required non-empty `VERSION` build-arg:** `build-args: VERSION=${{ github.sha }}` (or the
    derived short SHA). The Dockerfile fail-fasts on empty, and `${{ github.sha }}` is ALWAYS populated in
    Actions and can't empty-expand (unlike a `$(git rev-parse)` substitution), so the image always carries
    a non-empty `/version` — sidestepping the same empty-stamp class the `build:monitor` fix below
    addresses for the host path. Do NOT add a `$(git rev-parse)` substitution inside the workflow; use the
    Actions context var.
- **`.dockerignore` fold-in** (the `normal` "secret patterns or WAL/SHM sidecars" issue): ADD `.env`,
  `.env.*`, `**/auth.json`, `*.db-wal`, `*.db-shm`. Rationale to capture in a comment: `COPY . .` would
  otherwise bake a developer's local secret/state into the build-STAGE layer; the published image is
  unaffected (final stage only `COPY --from=build`s the binary) but defense-in-depth hygiene is the rule —
  the `.dockerignore` for a `COPY . .` Dockerfile must be a SUPERSET of the repo's secret `.gitignore`
  lines. `.claude/settings.local.json` is already covered by the existing `.claude/` line.
- **`mise.toml build:monitor` fold-in** (the `normal` "git-SHA empty-expands" issue): split the lookup so
  it fails fast — per `learnings.md` "`-ldflags -X` with an empty value OVERRIDES the default", an empty
  `$(git rev-parse)` clobbers `dev` to `""`. Rewrite the `run` to something like
  `sha=$(git rev-parse --short HEAD) && [ -n "$sha" ] && go build -ldflags "-X …Version=$sha" -o ./iscc-monitor ./cmd/iscc-monitor`
  so a git-failure aborts with non-zero exit and emits NO binary, never an empty-version one. Keep it a
  single portable `run` line (mise runs it through `sh`); the plain `build`/`check` tasks stay git-free and
  unchanged (the gate must never depend on git).
- **Correctness rule applied** (learnings.md, durable): keep the plain `go build ./...` gate git-free — do
  NOT add the SHA stamp to `check`/`build`; only `build:monitor` and the image path carry the `-X` stamp,
  each now guarded against the empty-expand.
- **Docker is absent on this host** (CI-only, per `learnings/ci.md`), so the `docker push` itself cannot be
  run locally — verify the workflow by YAML validity + static inspection of its trigger/tags/permissions
  (the Verify bullet explicitly asks for exactly that: "asserted by inspecting the workflow trigger + tag
  template"). The `.dockerignore` and `build:monitor` fixes ARE fully testable on this host.

## Verification
- `mise run check` is green (no Go source changed; this confirms the `mise.toml` edit did not break the
  task table and all packages still build/vet/test).
- `gofmt -l .` is empty (outside `cauldron/`).
- `.github/workflows/publish.yml` is valid YAML AND its inspectable contract holds, checkable
  mechanically (the repo has no PyYAML; use the cached `gopkg.in/yaml.v3` via a throwaway Go snippet, the
  established pattern from `learnings/ci.md`):
  - the workflow parses;
  - `on.push.branches` contains `develop`;
  - top-level `permissions.packages == write`;
  - the tag set contains BOTH a literal `ghcr.io/iscc/iscc-monitor:develop` AND a
    `ghcr.io/iscc/iscc-monitor:sha-`-prefixed immutable tag;
  - a non-empty `VERSION` build-arg is passed (the Dockerfile fail-fasts on empty).
- `.dockerignore` now lists `.env`, `.env.*`, `**/auth.json`, `*.db-wal`, `*.db-shm` — assert each line is
  present (e.g. `grep -qxF` each pattern).
- The `build:monitor` fail-fast holds: the rewritten guard aborts when `git rev-parse` yields nothing.
  Mechanically: run a shell simulation of the guard shape with an empty SHA, e.g.
  `sha=$(true); [ -n "$sha" ] && echo built || echo "aborted: empty sha"` → prints `aborted: empty sha`
  (no build), while a non-empty `sha=abc1234` → `built` — proving the `[ -n "$sha" ]` gate is load-bearing;
  reverting to the bare `$(git rev-parse)` substitution makes the empty case proceed with an empty `-X`.

## Done When
`.github/workflows/publish.yml` exists with a push-to-`develop` trigger, `packages: write`, and a
build-push that tags `ghcr.io/iscc/iscc-monitor` BOTH `develop` and `sha-<short>` passing a non-empty
`VERSION` build-arg; `.dockerignore` mirrors the gitignore's secret + WAL/SHM sidecar patterns; the host
`build:monitor` task fails fast on an empty git SHA; and `mise run check` + `gofmt -l .` stay green.
