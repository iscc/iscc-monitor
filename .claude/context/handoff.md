# Handoff

## 2026-06-20 — Pure fork-trigger detection (`CheckFork`) in `internal/logclient`

**Done:** Landed M1's second RFC-6962 self-consistency trigger as a pure, golden-testable verdict —
`CheckFork(prevSize uint64, prevRoot [rootBytes]byte, nextSize uint64, nextRoot [rootBytes]byte) bool
== prevSize > 0 && nextSize == prevSize && nextRoot != prevRoot` — plus a `ViolationFork ViolationKind
= "fork"` const next to `ViolationShrink`. Extended the existing `consistency.go` (no new file), with
zero new dependency, no fixtures, no follower wiring, and no store calls. Updated the package doc so
fork is no longer listed among the deferred merkle-backed triggers (only equivocation remains
deferred).

**Files changed:**
- `internal/logclient/consistency.go`: added `ViolationFork` const + the pure `CheckFork` function;
  rewrote the package doc prose to state fork landed dep-free (same-size, different-root `[32]byte`
  array compare via `!=`, no `bytes` import) and only equivocation still awaits
  `transparency-dev/merkle` + tiles.
- `internal/logclient/consistency_test.go`: added table-driven `TestCheckFork` (5 distinct, non-vacuous
  cases using `rootA`/`rootB`/`zeroRoot` literals with a setup guard that `rootA != rootB`) and
  `TestViolationForkKind` pinning `string(ViolationFork) == "fork"`.

**Verification:** `mise run check` → green (`go build ./...`, `go vet ./...`, `go test ./...` all `ok`
across didweb/follower/logclient/store on go1.24). Per-criterion:
- [x] `gofmt -l internal/logclient/consistency.go internal/logclient/consistency_test.go` — empty.
- [x] `go test -count=1 -run TestCheckFork ./internal/logclient` — PASS, 5 subtests (true + false).
- [x] `string(ViolationFork) == "fork"` — PASS (`TestViolationForkKind`).
- [x] `CheckFork(10183, rootA, 10183, rootB) == true` (same size, different root) — PASS.
- [x] `CheckFork(10183, rootA, 10183, rootA) == false` (identical root, re-observation) — PASS.
- [x] `CheckFork(10183, rootA, 10182, rootB) == false` (shrink) — PASS.
- [x] `CheckFork(10183, rootA, 10184, rootB) == false` (growth with differing roots) — PASS.
- [x] `CheckFork(0, zeroRoot, 5, rootB) == false` (fresh-store `prevSize == 0` guard) — PASS.
- [x] `git status --short go.mod go.sum` empty; `go list -m github.com/transparency-dev/merkle` →
  "not a known dependency" (no dep added).
- [x] `go list -deps ./internal/logclient | grep -c '^net/http'` → 4 (unchanged; the extended file
  imports nothing new).

**Next:** Wire `CheckShrink` + `CheckFork` into `follower.PollHub` — the composition slice both pure
verdicts were left unwired for. Map `FollowState.LastSize → prevSize` and the stored root at that size
(a `checkpoints` lookup, since `LastRoot` is intentionally NOT persisted in `follow_state`) → prevRoot;
map `CheckpointInfo.TreeSize/Root → nextSize/nextRoot`; on a true verdict call `RecordViolation` +
`Freeze`, with the "exactly one alert" mechanism. Keep `transparency-dev/merkle` + tile fixtures
deferred to the single equivocation slice that genuinely needs RFC-6962 consistency-proof math.

**Notes:**
- `CheckFork` / `ViolationFork` are an *intentional* unused-until-wired export seam (same as
  `CheckShrink` / `ViolationShrink`), referenced only by their own tests until the follower wiring
  slice lands — not dead code. `go vet` is clean; do not flag them.
- Used array `!=` on `[rootBytes]byte` (Go's elementwise comparison on fixed-size byte arrays), NOT
  `bytes.Equal`, so the file stays import-free of `bytes`/any new dep, preserving the slice's whole
  point. The `prevSize > 0` guard is load-bearing the same way shrink's `prev > 0` is: it keeps the
  fresh-store `FollowState{}.LastSize == 0` + its zero root from being misread as a fork.
- Conformance/oracle gate is correctly N/A here: this is a pure size/root array comparison touching no
  signature, RFC-6962 proof, didweb, or merkle code — that gate trips only when the equivocation slice
  lands. The didweb + logclient golden suites still pass (no regression).
- Branch is `develop`; committing implementation + test + this handoff only.
