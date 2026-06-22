## 2026-06-22 — Version-stamp the binary (`-ldflags -X`) and surface it on `GET /version`

**Done:** Added `internal/version`, a pure stdlib HTTP leaf exporting `var Version = "dev"` (the
`-ldflags -X` injection target) and `Handler()` serving `GET /version` → `{"version":"<Version>"}`.
Mounted it as an exact path in `buildMux`, reserved the `version` mount name against realm-domain
collision, and added a `build:monitor` mise task that stamps the git short SHA into the binary at link
time.

**Files changed:**
- `internal/version/version.go` (new): `Version = "dev"` + `Handler()` — JSON body via `%q` so a stamped
  SHA stays valid JSON, non-GET → 405, post-status write drop (documented convention). Mirrors
  `metricshttp`/`healthz` leaf style; imports only `fmt`/`net/http`.
- `internal/version/version_test.go` (new): HTTP-seam tests — GET → 200/`application/json`/non-empty
  `version` field equal to `Version`; non-GET → 405; default-is-`dev`.
- `cmd/iscc-monitor/main.go`: import `internal/version`; `mux.Handle("/version", version.Handler())` in
  `buildMux`; added `"version"` to `reservedMountNames`; doc comments updated to name the new route.
- `cmd/iscc-monitor/main_test.go`: `TestMirrorRouter` now asserts `/version` on the shared mux (200,
  `application/json`, non-empty field); `TestRegisterHubsRejectsReserved` table extended with `version`.
- `mise.toml`: new `tasks."build:monitor"` (`go build -ldflags "-X …version.Version=$(git rev-parse
  --short HEAD)" -o ./iscc-monitor ./cmd/iscc-monitor`). `build` / `check` left git-free and untouched.

**Verification:** `mise run check` → GREEN (all 27 packages `ok`, including the new
`internal/version`; build + vet clean). Per-criterion:
- [x] `gofmt -l .` → empty (touched files clean).
- [x] `go test -count=1 -run TestVersion ./internal/version` → PASS (3 tests).
- [x] `go test -count=1 -run TestBuildMux ./cmd/iscc-monitor` → PASS (reserved-domain no-panic, with
  `version` now in `reservedMountNames`); `TestMirrorRouter` + `TestRegisterHubsRejectsReserved` PASS.
- [x] Mechanical `-X` injection check: `go build -ldflags "-X …version.Version=test123"` → run →
  `curl 127.0.0.1:41465/version` returned `{"version":"test123"}` (status 200, `application/json`); a
  default `go build` served `{"version":"dev"}`; `mise run build:monitor` served the real SHA
  `{"version":"7947a86"}`. The stamp flows env-free from `-ldflags` to the response.
- [x] `go.mod` / `go.sum` / `internal/store/schema.sql` byte-identical (empty `git diff --stat`); no new
  dependency (stdlib only). `internal/version` is a pure leaf (`go list -deps` shows no `internal/*`).
- [x] Build artifact `./iscc-monitor` is gitignored (not staged).

**Next:** The front-of-queue M-Deploy `critical` is the multi-stage Dockerfile + GHCR publish workflow
(ADR-0013). It now CONSUMES this seam: the image build passes the git SHA via the same
`-ldflags -X github.com/iscc/iscc-monitor/internal/version.Version=<sha>` path `build:monitor`
established, and its container-runs-`/version` (or `/healthz`) CI smoke check is the natural Verify
assertion. Docker is unavailable locally, so that step's container check is CI-only and must be its own
slice. Other cheap independent slices still open: `deploy/realm-testnet.txt`, the operability/deployment
doc, and the root `README.md`.

**Notes:**
- Picked `GET /version` over folding into `/healthz` (target offered either) and left `internal/healthz`
  byte-identical, per `next.md` Not-In-Scope (its fixed-byte-literal "no marshal branch" design is
  deliberate).
- `internal/config` untouched — this is a compile-time `-ldflags` stamp, not a runtime env value, so it
  is correctly NOT an `ISCC_MONITOR_*` key (`learnings/config.md` confirms config stays a `{fmt time}`
  leaf). The `build:monitor` task is the ONLY git-dependent build path; `check`/`build` stay git-free so
  the gate never depends on `git` and compiles the `dev` default.
- Oracle gate N/A — pure HTTP wiring + a build-stamp string; touches no signature / RFC-6962 / Merkle /
  did:web / proof / fsck path. `proof/verify` purity + WASM build unaffected (`internal/version` is an
  HTTP leaf, not WASM-shared).
- Scope: 1 new prod file + 1 new test file + 2 modified files (`main.go`, `mise.toml`) + 1 modified test
  — within the ≤3 non-test-file budget (`version.go`, `main.go`, `mise.toml`).
