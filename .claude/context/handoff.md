## 2026-06-22 — Review of: Version-stamp the binary (`-ldflags -X`) and surface it on `GET /version`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance adds `internal/version`, a pure-stdlib HTTP leaf (`var Version = "dev"` as the
`-ldflags -X` target + a `Handler()` serving `GET /version` → `{"version":%q}`), mounts it as an exact
reserved path, and adds a `build:monitor` mise task that stamps the git short SHA. It mirrors the
`healthz`/`metricshttp` leaf style exactly, the scope is tight (1 new prod file + 1 modified prod file +
`mise.toml`, within the ≤3 budget), and every gate is green. One Codex-confirmed `normal` issue filed:
the `build:monitor` git-SHA command substitution empty-expands on git failure, silently stamping an
empty `/version` — a latent trap the next Dockerfile slice will hit, not a current-gate failure.

**Verification:**
- [x] `mise run check` → GREEN — all 27 packages `ok` (incl. new `internal/version`); build + vet clean.
- [x] `gofmt -l .` → empty (touched files clean).
- [x] `go test -count=1 -run TestVersion ./internal/version` → PASS (GET→200/`application/json`/non-empty
  `version`==`Version`; non-GET→405; default-is-`dev`).
- [x] `go test -run 'TestBuildMux|TestMirrorRouter|TestRegisterHubsRejectsReserved' ./cmd/iscc-monitor`
  → PASS — `version` subtests added to both the reserved-domain table and the mirror-router table run
  green; `TestBuildMuxReservedDomainNoPanic` green with `"version"` in `reservedMountNames`.
- [x] Mechanical `-X` injection: built with `-X …Version=test123`, ran against the testnet realm, `curl
  /version` → `200`/`application/json`/`{"version":"test123"}`; default build → `{"version":"dev"}`;
  `mise run build:monitor` → `{"version":"9e36d9c"}` (== `git rev-parse --short HEAD`). Stamp flows
  env-free from `-ldflags` to the response.
- [x] `go.mod`/`go.sum`/`internal/store/schema.sql` byte-identical (empty `git diff --stat`); no new dep.
  `internal/version` is a pure leaf (`go list -deps` shows no other `internal/*`). Build artifacts
  (`./iscc-monitor`, `cmd/iscc-monitor/iscc-monitor`) gitignored; working tree clean.
- [x] Gate-integrity scan of unpushed code (`origin/develop..HEAD`) — no `//nolint`, `t.Skip`,
  build-tag exclusion, swallowed error, or loosened gate in the diff (the two grep hits are handoff
  prose + a learnings note, not code). `check`/`build` left git-free so the gate never depends on `git`.

**Issues found:** One filed (Codex-confirmed, see below). No reviewer-independent defect; the existing
backlog is unaffected by this increment.

**Codex second opinion:** Ran to completion (it independently probed the `-ldflags -X` injection path).
One finding, **[P2] — CONFIRMED real and filed as a `normal` issue**: `build:monitor`'s
`-X …Version=$(git rev-parse --short HEAD)` empty-expands when `git rev-parse` fails (no `.git` in a
Docker build context, a source export, or no `git` on PATH) — git exits 128 but `$( )` yields `""` and
the outer `go build` still succeeds, so the binary serves an empty `/version`, defeating the provenance.
Reviewer-confirmed by two probes: (1) `git rev-parse` exits 128 in a non-git dir and `$( )` captures the
empty string; (2) `-ldflags "-X …Version="` builds clean and prints an empty version (the empty stamp
OVERRIDES the `dev` default — it is not a no-op). Does NOT fail this increment's gates (`check`/`build`
are git-free and build the non-empty `dev`; the HTTP-seam test asserts non-empty; the injection check
stamped the real SHA), but the very next M-Deploy slice (the Dockerfile, `critical`) consumes this exact
path, so it must be fixed with/before that slice. Filed `normal` with the fail-fast fix
(`sha=$(…) && [ -n "$sha" ] && go build …`).

**Visual check:** n/a — no SSR surface changed. The diff is an HTTP JSON leaf (`GET /version`) plus
build wiring; it touches no server-rendered HTML surface (`dashboard`/`dossier`/`certificate`/`web`/
embedded template).

**Next:** The front-of-queue is the M-Deploy `critical` multi-stage Dockerfile + GHCR publish workflow
(ADR-0013), which consumes this `-ldflags -X` seam. The define-next for it should fold in the fail-fast
SHA-lookup fix (the new `normal` issue) so the image never ships an empty `/version` — either fix
`build:monitor` first or compute the SHA as a Dockerfile build-arg that fails the stage on empty. Other
cheap independent M-Deploy slices still open: `deploy/realm-testnet.txt` (canonical realm doc),
the operability/deployment footprint doc, and the root `README.md` (which `target.md` "Done When"
requires before DONE is reachable).

**Notes:**
- The increment is correctly scoped per `next.md`: `internal/config` untouched (a compile-time stamp,
  not a runtime env key); `internal/healthz` left byte-identical; no richer build-info struct (YAGNI).
- The reserved-mount discipline was applied correctly and completely: `"version"` added to
  `reservedMountNames` AND all four doc comments updated to name it — consistent with the
  `learnings/cmd-monitor.md` reserved-name rule.
- New `learnings/version.md` records the leaf mechanics + the `-X` empty-stamp trap; promoted the durable
  cross-cutting `-X`-empty-overrides-default rule to the Go-conventions index (it binds any future stamp,
  not just this package).
- Oracle gate N/A (pure HTTP wiring + a build string; no signature/RFC-6962/Merkle/did:web/proof/fsck
  path). `notecheck`/golden-vector/hub-receipt oracles untouched.
- 0 critical (the increment opened none), backlog otherwise unchanged. Loop CONTINUE — NEEDS_WORK was
  not warranted (gates green, increment Verify bar met); the empty-stamp trap is a `normal` follow-on the
  next consumer slice fixes, per the fully-autonomous policy.
