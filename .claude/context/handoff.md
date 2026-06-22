## 2026-06-22 — Production multi-stage Dockerfile + CI container `/healthz` smoke job

**Done:** Added a tracked root `Dockerfile` (multi-stage `CGO_ENABLED=0` static build of
`cmd/iscc-monitor` into `gcr.io/distroless/static-debian12:nonroot` — non-root uid 65532, CA roots
present, baked interim realm) and a `.dockerignore` (excludes `.git`, the built binary, `cauldron/`,
`*.db`, `.claude/`, etc.), plus a sibling `docker` job in `.github/workflows/ci.yml` that builds the
image, runs it with a tmp DB + baked realm, and asserts `GET /healthz` → 200. The image's `VERSION` is
a required, fail-fast build-arg: an empty value aborts the build stage rather than shipping a blank
`/version` (folds in the `normal` empty-stamp fix). No Go source touched.

**Files changed:**
- `Dockerfile` (new): stage 1 `golang:1.26.4` builds with `-trimpath -ldflags "-s -w -X …version.Version=${VERSION}"`,
  guarded by `[ -n "$VERSION" ] || { …; exit 1; } &&` before the build; stage 2 distroless static
  nonroot, `COPY --from=build /iscc-monitor`, `COPY internal/registry/testdata/realm.txt
  /etc/iscc-monitor/realm.txt`, `EXPOSE 9464`, `ENTRYPOINT ["/iscc-monitor"]`. `ARG VERSION` is NOT
  defaulted (defaulting would mask the trap).
- `.dockerignore` (new): keeps `.git`, `/iscc-monitor`, `/cmd/iscc-monitor/iscc-monitor`, `cauldron/`,
  `*.db`/`*.sqlite*`, `.claude/`, `.devcontainer/`, `.github/`, the Dockerfile + dockerignore itself out
  of the build context.
- `.github/workflows/ci.yml`: added `docker` job (sibling to `check`, NOT folded in) — `checkout@v4`
  (matched existing pins), `docker build --build-arg VERSION="$(git rev-parse --short HEAD)"`, detached
  `docker run -d -p 9464:9464` with `ISCC_MONITOR_DB=/tmp/monitor.db`,
  `ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt`, `ISCC_MONITOR_ADDR=0.0.0.0:9464`, a 15s
  `curl -fsS …/healthz` poll loop that fails the job + dumps `docker logs` if it never returns 200, and
  an `if: always()` cleanup. `set -euo pipefail` in the run blocks.

**Verification:** `mise run check` → GREEN (all 28 packages `ok`, build + vet clean — confirms no Go
regression; no source touched). `gofmt -l .` clean outside `cauldron/`. `go.mod`/`go.sum`/`schema.sql`
byte-identical (empty `git diff --stat`; no new dependency). Per criterion:
- [x] `mise run check` green — verified.
- [x] image build / `-ldflags -X` host-equivalent: `go build -ldflags "-X …Version=testsha" -o …
  ./cmd/iscc-monitor` succeeds; running it (tmp DB + testnet realm + `127.0.0.1:41466` +
  `NORMAL=10m`) → `curl /healthz` = `{"status":"ok"}` (HTTP 200) and `curl /version` =
  `{"version":"testsha"}`. The stamp flows through. **(`docker build`/`docker run` themselves run in
  CI — Docker is absent on this host; marked verified-in-CI.)**
- [x] empty-stamp regression: `go build -ldflags "-X …Version=" …` produces `{"version":""}` (proves
  the empty stamp clobbers `dev` — not a no-op), and the Dockerfile guard `[ -n "$VERSION" ] || { …;
  exit 1; } &&` was POSIX-shell-verified: empty `VERSION` → refuse + exit 1 (build never runs),
  non-empty → build runs + exit 0. So `docker build --build-arg VERSION=` FAILS in CI. **(CI;
  host-inspected + shell-proven.)**
- [x] non-root + small image: Dockerfile uses `distroless/static-debian12:nonroot` (uid 65532, CA
  roots) — confirmed by reading the base + the absence of any `USER root`. **(`docker image inspect`
  in CI.)**
- [x] `ci.yml` valid YAML and the `docker` job reproduces build+run+`/healthz`-200: validated via
  `gopkg.in/yaml.v3` (YAML OK), and parsed to confirm two sibling jobs (`check`, `docker`) each with
  their steps. **(Verified locally.)**

**Next:** The GHCR **publish workflow** — M-Deploy's second Verify bullet and the second half of the
GHCR `critical` issue: on push to `develop`, build + push `ghcr.io/iscc/iscc-monitor` tagged `develop`
+ `sha-<short>` (this Dockerfile is the artifact it publishes; the publish step adds
`docker/login-action` + `docker/build-push-action` or a plain `docker push` with `GITHUB_TOKEN` and
`packages: write` permission — a new `.github/workflows/publish.yml` or a job in `ci.yml`). Other cheap
M-Deploy slices still open: the canonical `deploy/realm-testnet.txt`, the operability/deployment doc,
and the root `README.md` (which `target.md` "Done When" requires before DONE is reachable).

**Notes:**
- **Docker is NOT installed on this host** (confirmed: `which docker` empty), so the three
  container-run Verify items (`docker build`/`docker run`/`docker image inspect`) genuinely run in CI,
  not locally — I verified the host-independent equivalents (the exact `-ldflags -X` host build boots,
  serves `/healthz` 200, carries the stamp) and inspected the Dockerfile + CI run blocks statically, as
  the `next.md` Local-verification fallback directs. I did NOT claim the `docker` commands passed
  locally.
- The empty-SHA `normal` issue is fixed **for the image** via the Dockerfile build-arg guard, as
  scoped. The host `mise.toml` `build:monitor` task is deliberately left untouched (out of scope per
  `next.md`; the Dockerfile is the consumer that actually hits the trap). Whether to also harden
  `build:monitor` remains the open `normal` issue.
- The image bakes `internal/registry/testdata/realm.txt` at `/etc/iscc-monitor/realm.txt` as an
  **interim** realm purely so the CI smoke test has a registerable realm — `next.md` explicitly defers
  inventing `deploy/realm-testnet.txt` to a later slice. An operator overrides `ISCC_MONITOR_REALM` to
  point at their own document.
- `EXPOSE 9464` documents the default `ISCC_MONITOR_ADDR` port (`:9464`, confirmed in
  `internal/config` `defaultAddr`); it publishes nothing by itself (CI uses `-p 9464:9464`).
- Oracle/conformance gate is **N/A**: this slice is packaging + CI only, touches no signature /
  RFC-6962 / Merkle / did:web / proof / fsck path; `internal/*` and `cmd/*` are byte-identical.
- The `.gitattributes`-free repo will commit the Dockerfile with whatever line endings git defaults to;
  the RUN guard uses `sh` `[ -n … ]` which is LF-agnostic in a Linux build image.
