# Next Work Package

## Step: Version-stamp the binary (`-ldflags -X`) and surface it on `GET /version`

## Advances
M-Deploy Verify criterion (target.md, "Packaged & operable instance"):

> the binary is **version-stamped** (git SHA via `-ldflags`, default `dev` when unset) and reports it on
> `/healthz` JSON or `GET /version` — an HTTP-seam test asserts a non-empty version field;

This closes one standalone M-Deploy Verify item. It is the natural fold-in the `review` handoff named
("Fold in the `-ldflags` git-SHA build stamp surfaced on `/healthz` JSON or a tiny `GET /version` — the
version-stamp slice was deferred out of the SIGTERM step"). It is also a prerequisite the Dockerfile
step will consume: the production image build passes the git SHA via `-ldflags`, so landing the
testable injection seam FIRST means the later (locally-unverifiable, Docker-less) image step just wires
to an already-proven `var`. Skeleton-first ordering for M-Deploy: the runnable, pure-Go-testable slice
before the container infra (Docker is not available in this environment — the image step's
container-runs-`/healthz` check cannot be verified locally, so it must be its own later step).

## Goal
Give the running binary a build-provenance string (default `dev`, overridable to the git short SHA at
build time via `-ldflags -X`) and expose it at `GET /version` so an operator (and a CI smoke check) can
confirm exactly which build is live. This is the M-Deploy "which build is running?" requirement, and it
unblocks the Dockerfile step that will inject the real SHA.

## Scope
- **Create**: `internal/version/version.go` — a tiny HTTP leaf: an exported `var Version = "dev"` (the
  `-ldflags -X` injection target) plus a `Handler() http.Handler` that serves `GET /version` as JSON
  (`{"version":"<Version>"}`). Mirror the `internal/metricshttp` / `internal/healthz` HTTP-leaf style.
- **Modify**:
  - `cmd/iscc-monitor/main.go` — in `buildMux`, mount `mux.Handle("/version", version.Handler())`
    (an exact path, like `/metrics` and `/healthz`); and add `"version"` to the `reservedMountNames`
    map so a realm domain literally named `version` cannot collide with the exact mount and panic
    `http.ServeMux` (the same reserved-name discipline `learnings/cmd-monitor.md` documents for
    `/metrics`, `/healthz`, `/_ds`). Add the `internal/version` import.
  - `mise.toml` — add the `-ldflags` stamp so a real build injects the SHA: a `tasks."build:monitor"`
    that runs
    `go build -ldflags "-X github.com/iscc/iscc-monitor/internal/version.Version=$(git rev-parse --short HEAD)" -o ./iscc-monitor ./cmd/iscc-monitor`
    (the path the Dockerfile/CI will reuse). Keep `tasks.check` / `tasks.build` (plain `go build ./...`)
    UNTOUCHED so the default `dev` value still compiles and is what the gate builds — do NOT make `git`
    a hard requirement of the default `check` task.
- **Reference**:
  - `internal/metricshttp/handler.go` + `internal/healthz/handler.go` — the HTTP-leaf style to mirror
    (fixed Content-Type, drop the post-status write error deliberately, no nil-guard; `healthz` shows
    the 405-on-non-GET shape).
  - `.claude/context/learnings/cmd-monitor.md` — the `reservedMountNames` / `reservedDomain` discipline
    (an exact bare-domain mount over an operator-controlled realm domain MUST be reserved-name-gated or
    `http.ServeMux.Handle` PANICS on a duplicate pattern) and the thin-`main` / `buildMux` wiring rules.
  - `.claude/context/learnings/config.md` — confirms why this is NOT a config key (it is a build-time
    `-ldflags` stamp, not a runtime env value; `internal/config` stays a `{fmt time}`-only leaf).

## Not In Scope
- The production `Dockerfile` and the GHCR publish workflow (the next M-Deploy step — it CONSUMES this
  `-X` injection; Docker is not available in this environment, so its container-runs-`/healthz` CI check
  cannot be verified locally and must be its own step).
- Adding the version to `/healthz` JSON. Pick the `GET /version` surface (the target offers either);
  do NOT rewrite the healthz handler's fixed-byte-literal bodies (its "no marshal-failure branch"
  design is deliberate — leave `internal/healthz` byte-identical).
- A richer build-info struct (build time, Go version, `debug.ReadBuildInfo` VCS data). YAGNI — the
  Verify bar asks only for a non-empty version field; a single string default-`dev` meets it.
- Making `version` an `internal/config` key or an `ISCC_MONITOR_*` env var — it is a compile-time stamp,
  so `internal/config` is NOT touched.
- The `deploy/realm-testnet.txt` canonical realm doc, the operability/deployment doc, and the root
  `README.md` — separate M-Deploy slices, not this step.

## Implementation Notes
- **Default and injection.** `var Version = "dev"` at package scope is the standard `-ldflags -X`
  target: `go build -ldflags "-X github.com/iscc/iscc-monitor/internal/version.Version=abc1234"`
  overrides it at link time; an un-stamped build (the gate's plain `go build ./...`) keeps `dev`. The
  `-X` path is `<module>/internal/version.Version` — module is `github.com/iscc/iscc-monitor`
  (confirmed via `go list -m`). `-X` only overrides a `string` var initialized to a constant, so keep
  it a plain `var Version = "dev"` (NOT a `const`, NOT computed/concatenated).
- **Handler style.** Mirror `metricshttp.Handler` / `healthz.Handler`: an `http.HandlerFunc` that, on
  `GET`, sets `Content-Type: application/json`, calls `WriteHeader(200)`, then writes
  `{"version":"<Version>"}`. Build the body so a stamped SHA stays valid JSON — `strconv.Quote(Version)`
  inside the object (or `fmt.Sprintf(\`{"version":%q}\`, Version)`) avoids a broken body if the stamp
  ever contains a quote. Reject non-GET with 405 exactly as `healthz` does (consistency across the leaf
  endpoints). Drop the post-status write error deliberately (documented convention — a derived body
  cannot fail for content reasons after `WriteHeader`; only a broken client conn, unrecoverable).
- **Purity / leaf.** `internal/version` imports only stdlib (`net/http`, `fmt`/`strconv`) and NO
  `internal/*` package, so it stays a leaf. It is NOT WASM-shared (it is an HTTP leaf like
  `metricshttp`), so the `proof/verify`/`didweb` import-purity rule does not bind it.
- **Reserved-mount discipline (load-bearing).** `/version` is an EXACT mount in `buildMux`. The
  existing `reservedMountNames` is `{metrics, healthz, _ds}` (the `_ds` derived from `web.Prefix`); a
  realm line `version` would otherwise build an exact `/version` dossier mount in `mirrorHandler` BEFORE
  `buildMux` registers the real `/version`, panicking `http.ServeMux.Handle` on the duplicate pattern.
  Add `"version"` to `reservedMountNames` so `reservedDomain("version")` is true → `registerHubs` fails
  loudly at startup AND `mirrorHandler` skips the dossier mount (defense-in-depth). This mirrors exactly
  how `/metrics` and `/healthz` are protected (see `learnings/cmd-monitor.md`).
- **Oracle gate is N/A** — pure HTTP wiring + a build-stamp string; touches no signature / RFC-6962 /
  Merkle / did:web / proof / fsck path. `go.mod` / `go.sum` / `internal/store/schema.sql` MUST stay
  byte-identical (no new dependency — stdlib only).
- **Gate honesty.** Do NOT weaken `mise run check` — leave its `go build ./...` as-is (it builds the
  `dev` default, correct for the gate). The `-ldflags` stamp is an ADDITIONAL build path for the
  image/CI, not a replacement for the gate build.

## Verification
- `mise run check` is green (build + vet + test all pass; the default `dev` build compiles).
- `gofmt -l .` lists nothing (touched files clean).
- `go test -count=1 -run TestVersion ./internal/version` passes — an HTTP-seam test that drives
  `version.Handler()` over `httptest` asserts `GET /version` → `200`, `Content-Type: application/json`,
  a body parsing as JSON with a **non-empty** `version` field equal to `version.Version` (default
  `"dev"`), and a non-GET → `405`. (Name it `TestVersion…` so the `-run TestVersion` filter catches it.)
- `go test -count=1 -run TestBuildMux ./cmd/iscc-monitor` passes — the existing buildMux routing /
  reserved-domain tests stay green with `/version` mounted and `"version"` added to
  `reservedMountNames` (extend the reserved-domain assertion to cover `version` if a table drives it).
- Mechanical injection check (proves the `-X` target path is correct and the var is overridable):
  `go build -ldflags "-X github.com/iscc/iscc-monitor/internal/version.Version=test123" -o /tmp/iscc-monitor-vt ./cmd/iscc-monitor`
  builds, and starting it (tmp `ISCC_MONITOR_DB` + the `internal/registry/testdata/realm.txt` realm +
  `ISCC_MONITOR_ADDR=127.0.0.1:41465` + a long `ISCC_MONITOR_NORMAL=10m`) then
  `curl -s 127.0.0.1:41465/version` returns a body containing `test123` — the stamp flows env-free from
  `-ldflags` to the served response. (A default un-stamped build serves `dev`.)

## Done When
`GET /version` serves a non-empty version string (default `dev`, overridable to the git SHA via the
documented `-ldflags -X github.com/iscc/iscc-monitor/internal/version.Version=<sha>`), the new
`internal/version` HTTP-seam test and the buildMux reserved-name tests pass, and `mise run check` is
green with `go.mod` / `go.sum` / `schema.sql` byte-identical.
