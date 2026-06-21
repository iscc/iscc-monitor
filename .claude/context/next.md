# Next Work Package

## Step: Drive `TestPollHubFork` re-detection through a second `PollHub`

## Goal
Close the open `normal` ADR-0006 issue by re-arming `TestPollHubFork` to re-detect a fork through a
real second `PollHub` (proving the freeze + once-only-alert + evidence-accumulation invariants on the
production code path), now that `store.CheckpointAt`'s `ORDER BY rowid LIMIT 1` makes the prior-accepted
root deterministic. Remove the stale "unordered LIMIT 1 … non-deterministic" comment that no longer
describes the code.

## Scope
- **Create**: (none)
- **Modify**: `internal/follower/follower_test.go` (test-only — does not count against the 3-file cap)
- **Reference**:
  - `internal/follower/follower_test.go:185-326` (`TestPollHubFork`; the re-detection block at 276-290
    that currently calls `freeze(...)` directly)
  - `internal/follower/follower.go:136-246` (`PollHub`: `checkConsistency` at line 180 runs BEFORE the
    `fs.Frozen` short-circuit at 204; the `violated` branch at 184 re-freezes with `wasFrozen=fs.Frozen`)
  - `internal/follower/follower.go:420-461` (`checkConsistency`) and `:472-500` (`freeze` — `wasFrozen`
    gates the single alert; `RecordViolation` has no `ON CONFLICT`, so re-detection records a new row;
    `freeze` persists the contradicting checkpoint at a HIGHER rowid than the seed)
  - `internal/store/checkpoints.go:143-169` (`CheckpointAt` — deterministic `ORDER BY rowid LIMIT 1`,
    returns the lowest-rowid seed/prior-accepted root over the later contradicting-evidence row)
  - `.claude/context/issues.md` (the issue titled "`TestPollHubFork` re-detection still bypasses
    `PollHub`" — its "Verify fixed" clause is this step's acceptance bar)

## Not In Scope
- Touching any production file (`follower.go`, `checkpoints.go`, or any non-test `.go`). The store fix
  already shipped; this is purely the follower-test catch-up. If a second `PollHub` does NOT re-detect,
  STOP and report — do not "fix" production to make the test pass.
- The other four open `normal` issues (`AcceptCheckpoint` context reuse, tile-writer `p`-vocabulary,
  deep `AdvanceAccepted`, `logclient.CheckConsistency` collapse) — each is its own later slice.
- Proof-surface ETag/Cache-Control, verify-for-me, dashboard, WASM, OTS.
- Changing the seed-root setup, the first-poll assertions (lines 230-274), the unaffected-hub-B check,
  or the restart-survival check — only the re-detection block (276-290) changes its *driver*.

## Implementation Notes
- Replace the direct `freeze(ctx, s, hubID, logclient.ViolationFork, …)` call (currently at
  follower_test.go:282) with a **second** `PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id",
  time.Unix(2, 0), alert, reg)`. Use a *later* `observedAt` (`time.Unix(2, 0)`) than the first poll's
  `time.Unix(1, 0)` so the two detections are distinguishable in `detected_at`; assert the call returns
  `StatusVerified` with a nil error (a violation is a separate axis from the signature verdict).
- Why a second `PollHub` re-detects (do not weaken these assertions): after the first poll, `freeze` did
  NOT advance the cursor, so `fs.LastSize` stays `m.size` and `fs.Frozen` is now true. On the second poll
  `checkConsistency` runs BEFORE the `fs.Frozen` short-circuit, reads the prior accepted root via
  `CheckpointAt(hubID, m.size)` — which returns the **seed** root (lowest rowid) deterministically, not
  the contradicting mirror root that the first `freeze` persisted (higher rowid) — re-detects the fork,
  and re-freezes with `wasFrozen=true` (so `alert` is NOT fired a second time).
- Keep the existing post-re-detection assertions, adjusting only their *driver*: `violations` row count
  == 2 (re-detection is evidence — `RecordViolation` has no `ON CONFLICT`); `alerts` == 1 (exactly-one
  alert across the not-frozen→frozen transition only); hub stays `Frozen`; `LastSize` still `m.size`
  (a frozen hub never advances). If the violation-kind metric is re-asserted, it stays `kind="fork"`
  with the cumulative counter now at 2 (`iscc_monitor_violations_total{hub_id="1",kind="fork"} 2`).
- Delete the stale comment block at follower_test.go:276-281 (the "driven through freeze directly … and
  CheckpointAt's unordered LIMIT 1 makes a re-poll's fork re-comparison non-deterministic … pre-existing
  re-detection fragility" prose). Replace it with a short comment stating that the second `PollHub`
  re-detects against the deterministic prior-accepted root (seed, lowest rowid) and re-freezes without
  re-alerting.
- Correctness rule (learnings.md / ADR-0006): "A self-consistency violation freezes, never crashes …
  alert once, keep polling evidence-only … other hubs unaffected." Re-detection must compare against the
  prior **accepted** root, never the contradicting-evidence row — which is exactly what the deterministic
  `CheckpointAt` now guarantees and what this second-`PollHub` drive exercises end-to-end.
- The test is `package follower` (internal), and `PollHub`, `metrics`, `logclient`, and `store` are
  already imported/in-scope — no new imports. After the edit the test no longer references `freeze`; if
  `freeze` is now unused *only because of other test changes*, do not silence anything — `freeze` keeps
  its real production caller in `follower.go`, so `go vet`/`gofmt` stay clean.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass, `gofmt -l .`
  empty).
- `go test -count=1 -run TestPollHubFork ./internal/follower` passes.
- `grep -rn "unordered LIMIT 1" internal/` returns no matches.
- The re-detection in `TestPollHubFork` is driven by a second `PollHub(...)` call, not a direct
  `freeze(...)` — verifiable by `grep -n "freeze(" internal/follower/follower_test.go` showing no
  `freeze(` call inside `TestPollHubFork` (lines 185-326).
- After the second `PollHub`, the test asserts `violations` count == 2 and `alerts` == 1 (retained
  `countRows`/alert assertions), the hub stays `Frozen`, and `LastSize` is still `m.size`.
- Mutation sanity (advance runs once, then reverts): changing the expectation to `alerts == 2` (or
  removing the second `PollHub`) makes `TestPollHubFork` FAIL — confirming the re-detection drive is
  non-vacuous and the once-only-alert invariant is genuinely exercised on the production path.

## Done When
`TestPollHubFork` re-detects the fork via a second `PollHub` (asserting two violations and exactly one
alert with the hub still frozen and not advanced), the stale "unordered LIMIT 1" comment is gone, and
`mise run check` is green with no production file changed.
