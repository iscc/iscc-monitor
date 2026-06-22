# Next Work Package

## Step: Production multi-stage Dockerfile + CI container `/healthz` smoke job

## Advances
Milestone **M-Deploy — Packaged & operable instance** (ADR-0013, PRD story 13), its **first** Verify
bullet, quoted:

> a tracked **`Dockerfile`** builds `cmd/iscc-monitor` as a `CGO_ENABLED=0` static binary into a minimal
> **non-root** final image (`scratch`/distroless, CA roots present), and a **CI job builds it, runs the
> container** with a tmp `ISCC_MONITOR_DB` + the baked realm, and asserts `GET /healthz` → `200`;

This also closes the `critical` issue **"Publish a deployable container image to GHCR (Dockerfile +
push workflow)"** *in part* (the image-build half) and folds in the `normal` issue **"`build:monitor`'s
git-SHA command substitution empty-expands on git failure"** by computing the SHA as a Dockerfile
build-arg that **fails the stage on empty** — so the image never ships an empty `/version`. The GHCR
*publish workflow* (the second M-Deploy bullet) is deliberately a separate later step (see Not In Scope).

## Goal
Produce the deployable artifact M-Deploy is built around: a small, non-root, statically-linked container
image of `cmd/iscc-monitor` that CI proves actually boots and serves `/healthz` → 200. This is the
front-of-queue code-closable M-Deploy work and unblocks the GHCR publish step that follows.

## Scope
- **Create**: `Dockerfile` (repo root) — multi-stage `CGO_ENABLED=0` build into a non-root minimal final
  image with CA roots; `.dockerignore` (repo root) — exclude `.git`, the built `/iscc-monitor` binary,
  `cauldron/`, `*.db`, and `.claude/` from the build context.
- **Modify**: `.github/workflows/ci.yml` — add a job (`docker`) that builds the image, runs the
  container, and asserts `GET /healthz` → 200 (CI/config file, not a non-test/doc source file).
- **Reference**:
  - `.claude/context/learnings/version.md` — the `-X` empty-stamp-OVERRIDES-default trap this step must
    guard with a fail-fast build-arg.
  - `.claude/context/learnings/ci.md` — CI is one `ubuntu-latest`, `CGO_ENABLED=0` job; how the existing
    `check` job is structured (add the new job alongside it, do not fold into it).
  - `.claude/context/learnings/cmd-monitor.md` — the binary's entrypoint, required env, and the
    `/healthz` / `/version` reserved mounts.
  - `mise.toml` `[tasks."build:monitor"]` — the `-ldflags -X …version.Version=<sha>` stamp the Dockerfile
    build stage reproduces (and must guard against an empty SHA).
  - `internal/registry/testdata/realm.txt` — the interim realm baked into the image so the container has
    a valid `ISCC_MONITOR_REALM` for the smoke test.

## Not In Scope
- **The GHCR publish workflow** (push to `develop` → `ghcr.io/iscc/iscc-monitor` tagged `develop` +
  `sha-<short>`). That is M-Deploy's *second* Verify bullet and the second half of the GHCR `critical`
  issue — it is its own next step. Build the image and prove it boots first; publishing it follows.
- **The canonical `deploy/realm-testnet.txt`** + the operability/deployment doc + the root `README.md` —
  each is its own later M-Deploy slice. This step bakes the existing `internal/registry/testdata/realm.txt`
  as an *interim* realm purely so the CI smoke test has a registerable realm; do not invent `deploy/` here.
- **Fixing `mise.toml`'s `build:monitor` task** itself. The empty-SHA trap is fixed *for the image* via
  the Dockerfile build-arg guard here; whether to also harden the host `build:monitor` task is left to the
  `normal` issue (the Dockerfile is the consumer that actually hits the trap). Do not exceed the file
  budget chasing it.
- The on-disk DB migration mechanism, `/metrics` exposure policy, volume/egress docs — separate items.

## Implementation Notes
- **Multi-stage build.** Stage 1: `FROM golang:1.26.4 AS build` (matches the module's `go 1.26.1`
  directive + `mise.toml` toolchain `1.26.4`; pin the concrete tag for reproducibility). `WORKDIR /src`,
  `COPY go.mod go.sum ./` then `RUN go mod download`, then `COPY . .`. Build with
  `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X
  github.com/iscc/iscc-monitor/internal/version.Version=${VERSION}" -o /iscc-monitor ./cmd/iscc-monitor`.
- **Fail-fast SHA (the `normal` empty-stamp fix).** Declare `ARG VERSION` and **guard it inside the build
  RUN** so an empty value aborts the stage, e.g. begin the RUN with
  `[ -n "$VERSION" ] || { echo 'VERSION build-arg is empty — refusing to ship an empty /version stamp'; exit 1; }`.
  Per `learnings/version.md`, an empty `-X …Version=` OVERRIDES the `dev` default (it is NOT a no-op), so
  the guard is load-bearing: never let `--build-arg VERSION=` (or an unset arg expanding empty) produce an
  empty-version image. The CI job passes `--build-arg VERSION=$(git rev-parse --short HEAD)`; locally a
  caller can pass `dev`. (Do NOT default `ARG VERSION=dev` — that would mask the very trap we are
  guarding; require the caller to pass a non-empty value.)
- **Final stage — non-root, CA roots, scratch-class.** Prefer
  `FROM gcr.io/distroless/static-debian12:nonroot` (ships CA roots, a `nonroot` uid 65532, and is
  `scratch`-class for a static binary — satisfies "non-root" + "CA roots present" in one line). If you
  instead use `scratch`, you MUST `COPY --from=build /etc/ssl/certs/ca-certificates.crt
  /etc/ssl/certs/` AND set a numeric `USER 65532:65532` (scratch has no `/etc/passwd`, so use a numeric
  uid, not a name). `COPY --from=build /iscc-monitor /iscc-monitor`. `COPY` the interim realm
  (`internal/registry/testdata/realm.txt`) to a fixed documented path, e.g. `/etc/iscc-monitor/realm.txt`.
  `EXPOSE 9464` (documents the default `ISCC_MONITOR_ADDR` port; publishes nothing). `ENTRYPOINT
  ["/iscc-monitor"]`. CA roots matter for runtime did:web + hub HTTPS even though the smoke test only
  needs `/healthz`; the Verify bar names them.
- **The container must bind a reachable address for the smoke test.** `cmd/iscc-monitor` defaults
  `ISCC_MONITOR_ADDR` to `:9464` (confirmed: `main.go` `serveMetrics(ctx, cfg.Addr, …)`, default port
  9464). For CI to curl it, run with `-p 9464:9464` and pass `ISCC_MONITOR_ADDR=0.0.0.0:9464` (a bare
  `:9464` binds all interfaces too, but be explicit). `ISCC_MONITOR_DB` must point at a **writable** path
  inside the container — distroless `nonroot` cannot write `/`; use `/tmp/monitor.db` (distroless has a
  writable `/tmp`). For the smoke test, `/tmp/monitor.db` is simplest.
- **The realm hubs will be unreachable in CI** (no egress to `sb0.iscc.id`); that is FINE — `/healthz`
  pings only the SQLite store (`healthz.Pinger` → `store.Ping`), not the hubs (confirmed in
  `internal/healthz/handler.go`), so it returns 200 as soon as the store opens, independent of poll
  success. The smoke test asserts store-readiness, not hub reach.
- **CI job shape (`ci.yml`).** Add a second job `docker` (sibling to `check`, NOT folded into it — keep
  the gate job fast and the image job independent, per `learnings/ci.md`). Steps: `actions/checkout@v4`
  (match the existing pins — do NOT bump action majors here; the Node-20 bump is a separate `low` issue);
  `docker build --build-arg VERSION="$(git rev-parse --short HEAD)" -t iscc-monitor:ci .`; run detached
  `docker run -d --name mon -p 9464:9464 -e ISCC_MONITOR_DB=/tmp/monitor.db -e
  ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt -e ISCC_MONITOR_ADDR=0.0.0.0:9464 iscc-monitor:ci`; poll
  `http://localhost:9464/healthz` for up to ~15s and assert HTTP 200 (a `for` loop curling with
  `--fail`, failing the job if it never returns 200); `docker logs mon` on failure for diagnosis;
  `docker rm -f mon` cleanup. Use `set -euo pipefail` in the run blocks. Keep it a plain `docker`
  invocation (ubuntu-latest has Docker preinstalled) — do not pull in buildx/registry actions (those
  belong to the publish step).
- **Build context hygiene (`.dockerignore`).** Without it, `COPY . .` would copy `.git`, the 26 MB
  gitignored `/iscc-monitor` binary, and `cauldron/` (which breaks `go build ./...` per the always-loaded
  learning) into the build context. Exclude at minimum: `.git`, `/iscc-monitor`,
  `/cmd/iscc-monitor/iscc-monitor`, `cauldron/`, `*.db`, `*.sqlite*`, `.claude/`, `.devcontainer/`,
  `.github/`. The Go build only needs the module sources + `go.mod`/`go.sum`.
- **No application source changes.** This step adds packaging + CI only; `internal/*` and `cmd/*` stay
  byte-identical. `go.mod`/`go.sum` unchanged (no new dependency).

## Local-verification fallback (Docker is NOT installed in this environment)
`docker` is confirmed absent on this host (the prior version-stamp handoff warned of this), so the
container-run Verify items below cannot run locally — they run in CI. `advance` should therefore verify
locally by **static inspection + a host-build dry run**, and rely on CI for the container assertions:
- Confirm the same `-ldflags -X` target builds on the host (no Docker):
  `go build -ldflags "-X github.com/iscc/iscc-monitor/internal/version.Version=testsha" -o /tmp/iscc-monitor-img ./cmd/iscc-monitor`
  succeeds, and running it (tmp `ISCC_MONITOR_DB` + `internal/registry/testdata/realm.txt` realm +
  `ISCC_MONITOR_ADDR=127.0.0.1:41466` + `ISCC_MONITOR_NORMAL=10m`) then `curl 127.0.0.1:41466/healthz`
  → `{"status":"ok"}` and `curl 127.0.0.1:41466/version` → `{"version":"testsha"}`. This proves the
  binary the image wraps boots, serves `/healthz` 200, and carries the stamp — the only parts that are
  host-independent of Docker.
- Validate `ci.yml` is well-formed YAML (the same `gopkg.in/yaml.v3` path `learnings/ci.md` notes the
  prior CI step used to validate the workflow locally, since PyYAML is absent).
- Lint the `Dockerfile` by reading it for the guard, the non-root user, the CA-roots base, and the
  `COPY`/`ENTRYPOINT` correctness; do NOT claim the `docker build`/`docker run` Verify lines passed
  locally — mark them "verified in CI" in the handoff.

## Verification
- `mise run check` is green (unchanged — no Go source touched; confirms the step did not regress the
  gate).
- `docker build --build-arg VERSION=testsha -t iscc-monitor:ci .` succeeds and produces an image. *(CI;
  host-equivalent: the `-ldflags -X` host build above succeeds.)*
- `docker run -d -p 9464:9464 -e ISCC_MONITOR_DB=/tmp/monitor.db -e
  ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt -e ISCC_MONITOR_ADDR=0.0.0.0:9464 iscc-monitor:ci`
  starts, and within ~15s `curl -fsS http://localhost:9464/healthz` returns HTTP 200 with
  `{"status":"ok"}`. *(CI; host-equivalent above.)*
- `curl -fsS http://localhost:9464/version` returns `{"version":"testsha"}` (the build-arg stamp flowed
  through `-ldflags -X`; proves the version stamp is non-empty in the image). *(CI; host-equivalent
  above.)*
- `docker build --build-arg VERSION= -t iscc-monitor:empty .` **FAILS** (the fail-fast guard aborts the
  build stage rather than shipping an empty `/version`) — the regression proof for the folded-in `normal`
  empty-stamp fix. *(CI; locally inspect the guard line in the Dockerfile and confirm the host build with
  an empty `-X …Version=` would be refused by the same `[ -n "$VERSION" ]` check.)*
- `docker image inspect iscc-monitor:ci` shows a non-root user (uid 65532 / `nonroot`, not root) and a
  small final image (well under ~30 MB). *(CI; locally confirm the `nonroot` base + `USER` directive in
  the Dockerfile.)*
- The new `docker` job in `.github/workflows/ci.yml` is valid YAML and its `run` blocks reproduce the
  build + run + `/healthz`-200 assertion above. *(Verifiable locally.)*

## Done When
A tracked root `Dockerfile` + `.dockerignore` build a non-root, CA-roots, statically-linked,
non-empty-version-stamped `cmd/iscc-monitor` image, a CI `docker` job builds it and asserts `GET
/healthz` → 200, an empty `VERSION` build-arg fails the build, and `mise run check` stays green.
