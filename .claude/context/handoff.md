# Handoff

## 2026-06-20 — Coverage tracking: persist `monitored_since` (size + time), set-once, on first verified observation

**Done:** Added `SetCoverage`/`Coverage` to the store (set-once guarded UPDATE of
`hubs.monitored_since_{size,time}` + a reader), and wired `PollHub` to record coverage once on the
verified, non-violation path before `AdvanceFollowState`. A later/larger observation never moves the
immutable start (ADR-0001 coverage honesty); violating and non-verified verdicts never start coverage.

**Files changed:**
- `internal/store/checkpoints.go`: added `CoverageInfo` struct, `SetCoverage(ctx, hubID, size,
  observedAt)` (guarded `UPDATE … WHERE monitored_since_size IS NULL`, no-op + nil error after the
  start is set, never relies on `RowsAffected`), and `Coverage(ctx, hubID)` reader (`sql.NullInt64`,
  absent/unset hub → `CoverageInfo{}` + nil err, zero-time NULL → zero `Since`).
- `internal/follower/follower.go`: call `SetCoverage(hubID, info.TreeSize, observedAt)` between
  `RecordCheckpoint` and `AdvanceFollowState` on the verified non-violation path, error wrapped like
  its siblings; kept out of the `freeze` helper so a contradictory observation never starts coverage;
  updated the package doc comment to drop "coverage" from the deferred later-steps list.
- `internal/store/checkpoints_test.go` (test): `TestCoverageSetOnce` (100/t0 not moved by 500/t1, raw
  columns checked), `TestCoverageUnset` (un-started + absent hub → unset, nil err), `TestCoverageZero
  ObservedAtNull` (zero time → NULL, Set true, zero Since).
- `internal/follower/follower_test.go` (test): extended `TestPollHubVerifiedAdvances` (coverage set to
  10183 + observedAt, immovable by a later second poll); added coverage-not-set assertions to
  `TestPollHubUnverifiedDoesNotAdvance` and `TestPollHubFork`.

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all `ok`, 7 packages).
Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -run TestCoverage ./internal/store` — PASS: after `SetCoverage(100, t0)` then
  `SetCoverage(500, t1)`, `Coverage` reports `size==100` / `since==t0`; unset hub reports `set==false`.
- [x] `go test -run TestPollHub ./internal/follower` — PASS: a verified `PollHub` sets coverage to the
  fixture size `10183` and the observed time; a second verified poll (later time) does not move it.
- [x] `git diff HEAD -- go.mod go.sum` empty (no new dependency).
- [x] `go list -deps ./internal/store | grep '^net/http$'` empty (store stays a leaf).

**Next:** The merkle-backed **equivocation** trigger — the heavy slice deferred across prior steps and
the first to trip the oracle/conformance gate (`transparency-dev/merkle` + tile fixtures, `fsck`
root-rebuild over a `SQLiteFetcher`, inclusion cross-check vs the hub's `IsccLogInclusionProof`,
`notecheck` parity in CI). Lighter alternatives still open: the `hub_keys` did:web cache write (+ the
stale sb1 fixture refresh `22b08f3e`→`069d0f14`), structured logging to replace the two stderr
placeholders, `/metrics`, and surfacing the coverage window through the dashboard/REST (M2/M3).

**Notes:**
- Oracle/conformance gate is correctly **N/A**: the diff touches no
  proof/verify/didweb/merkle/consistency/signature/fsck/notecheck path — `SetCoverage`/`Coverage` are
  plain `hubs`-column CRUD and the `PollHub` change only adds a set-once write on the already-verified
  path. go.mod/go.sum byte-identical (no dep entered the closure).
- `SetCoverage` deliberately ignores `RowsAffected` (per `next.md`): the `IS NULL` guard makes a
  re-call a silent, correct no-op, so zero rows affected after the start is set is not an error.
- `info.TreeSize` (not `fs.LastSize`) is the recorded size, matching the just-recorded checkpoint; in
  practice `PollHub` always injects a real `observedAt`, so size and time land together on the first
  verified poll. The zero-`observedAt`→NULL path is covered by a store unit test for completeness but is
  not reachable from the live `PollHub` call.
- The fork-path "coverage stays unset" assertion is non-vacuous because the test seed uses
  `RecordCheckpoint`/`AdvanceFollowState`, never `SetCoverage`, so only a clean verified observation
  could have set it — and the freeze branch returns before reaching `SetCoverage`.
