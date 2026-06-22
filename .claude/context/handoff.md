## 2026-06-22 — Review of: Production multi-stage Dockerfile + CI container `/healthz` smoke job

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance adds a tracked root `Dockerfile` (multi-stage `CGO_ENABLED=0` static build of
`cmd/iscc-monitor` into `distroless/static-debian12:nonroot`, CA roots, baked interim realm), a
`.dockerignore`, and a sibling `docker` CI job that builds the image, runs it, and asserts `GET /healthz`
→ 200. Scope-clean (2 new config files + 1 CI-config edit, no Go source touched), gates green, and the
empty-`VERSION` fail-fast guard is independently proven. One reviewer- and Codex-confirmed build-context
hygiene gap (`.dockerignore` does not mirror the gitignored secret/sidecar patterns) is filed `normal` —
it does not reach the published image or CI, so it does not block PASS.

**Verification:**
- [x] `mise run check` green — verified: all 28 packages `ok`, build + vet clean (no Go source touched).
- [x] `gofmt -l .` clean — verified (no files listed outside `cauldron/`).
- [x] `go.mod`/`go.sum`/`schema.sql` byte-identical — verified (empty `git diff --stat`); `go mod verify`
  → "all modules verified" (the image's `go mod download` will succeed).
- [x] Image build host-equivalent — verified: the exact image build command
  (`CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X …version.Version=imgtest" …`) produces a
  **statically linked, stripped** 19 MB ELF (`file` → "statically linked", `ldd` → "not a dynamic
  executable"), confirming it runs on `distroless/static`. *(`docker build`/`run`/`inspect` themselves are
  CI-only — Docker absent on this host; the host-independent half is what I verified.)*
- [x] Boot + `/healthz` 200 + version stamp — verified: the `-X`-stamped host binary boots against the
  testnet realm and serves `/healthz` → `{"status":"ok"}` and `/version` → `{"version":"testsha"}`.
- [x] Empty-stamp regression — verified: `-X …Version=` produces `{"version":""}` (clobbers `dev`, so the
  guard is load-bearing), and the POSIX guard `[ -n "$VERSION" ] || { …; exit 1; } &&` fails fast (exit 1,
  build never runs) on empty and proceeds (exit 0) on non-empty — precedence tested with the exact line.
- [x] Non-root + CA roots — verified by reading the base (`distroless/static-debian12:nonroot`, uid 65532,
  CA bundle) and the absence of any `USER root`. *(`docker image inspect` runs in CI.)*
- [x] `ci.yml` valid YAML, two sibling jobs (`check`, `docker`) — verified via `gopkg.in/yaml.v3` (parses;
  `jobs: check docker`). The `docker` job reproduces build → run → 15s `/healthz`-200 poll → `if: always()`
  cleanup. *(Verified locally.)*
- [x] `.dockerignore` does not drop a needed fixture — verified: zero tracked `.db`/`.sqlite` files exist,
  `internal/registry/testdata/realm.txt` is tracked (the `COPY` source) and not git-ignored, no embedded
  `.db` fixtures, and the build context still carries `cmd/`/`internal/`/`testdata/`/`go.mod`/`go.sum`.

**Issues found:** One filed `normal`: **`.dockerignore` does not mirror the gitignored secret patterns or
the WAL/SHM DB sidecars** — `.env`/`.env.*`/`**/auth.json` and `*.db-wal`/`*.db-shm` (the `*.db`/`*.sqlite*`
rules match neither) are not excluded, so `COPY . .` could bake a developer's local secret/state into the
build-STAGE layer. Reviewer-confirmed via fnmatch + `.gitignore` cross-check; does NOT reach the published
image (final stage only `COPY --from=build`s the binary) and does NOT affect CI (a fresh checkout has none
of these files). Defense-in-depth hardening for the GHCR-publish slice to fold in, not a leak in the shipped
artifact. No gate-circumvention found across unpushed commits (the `t.Skip`/`//go:build` grep hits are prose
inside the prior handoff narrative, not the code diff).

**Codex second opinion:** One P2: "Exclude local secrets and DB sidecars from the Docker context"
(`.dockerignore:20-29`) — **confirmed real** (verified `*.db` does not fnmatch `monitor.db-wal`, and
`.gitignore` lists `.env`/`.env.*`/`**/auth.json` which `.dockerignore` omits), but **does not block PASS**:
the gap is confined to the build-STAGE layer/cache, never the published image and never the clean CI
checkout. Filed as the `normal` issue above for the next M-Deploy slice. No trust-root surface touched, so
no oracle conflict to adjudicate.

**Visual check:** n/a — no SSR surface changed (packaging + CI config only; `internal/dashboard`,
`internal/dossier`, `internal/web`, `internal/certificate`, and all templates are byte-identical).

**Next:** The GHCR **publish workflow** — M-Deploy's second Verify bullet and the second half of the GHCR
`critical` issue: on push to `develop`, build + push `ghcr.io/iscc/iscc-monitor` tagged `develop` +
`sha-<short>` (this Dockerfile is the artifact it publishes; add `docker/login-action` +
`docker/build-push-action` or a plain `docker push` with `GITHUB_TOKEN` + `packages: write`, as a new
`.github/workflows/publish.yml` or a `ci.yml` job). Fold the `.dockerignore` secret/sidecar hardening
(the `normal` filed this iteration) into that slice since it is the natural next toucher of these files.
Other cheap M-Deploy slices still open: `deploy/realm-testnet.txt`, the operability/deployment doc, and
the root `README.md` (which `target.md` "Done When" requires before DONE).

**Notes:**
- **Docker is NOT installed on this host** (`which docker` empty), so the three container-run Verify items
  genuinely run in CI; I verified the host-independent equivalents (the exact `-trimpath -ldflags -X` build
  is static/stripped, boots, serves `/healthz` 200, carries the stamp) and inspected the Dockerfile + CI
  run blocks statically. I did NOT claim the `docker` commands passed locally.
- The empty-SHA `normal` issue ("`build:monitor`'s git-SHA empty-expands") is fixed **for the image** via
  the Dockerfile build-arg guard, but the host `mise.toml build:monitor` task is deliberately left untouched
  (out of scope per `next.md`), so that issue stays OPEN — do not delete it.
- The `critical` GHCR issue is now half-closed (image-build + boot proven); the publish half remains.
- Oracle/conformance gate is **N/A**: packaging + CI only, no signature / RFC-6962 / Merkle / did:web /
  proof / fsck path; `internal/*` and `cmd/*` byte-identical, `go.sum` byte-identical (no new dependency).
- Final image size: ~19 MB binary + ~2 MB distroless static base ≈ comfortably under the ~30 MB target.
