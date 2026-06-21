## 2026-06-21 — Drive `TestPollHubFork` re-detection through a second `PollHub`

**Done:** Re-armed `TestPollHubFork`'s re-detection block to re-detect the fork through a real second
`PollHub` (later `observedAt=time.Unix(2,0)`) instead of calling `freeze(...)` directly, proving the
freeze + once-only-alert + evidence-accumulation invariants on the production code path now that
`store.CheckpointAt`'s `ORDER BY rowid LIMIT 1` makes the prior-accepted root deterministic. Removed the
stale "unordered LIMIT 1 … non-deterministic" comment that no longer described the code. No production
file changed.

**Files changed:**
- `internal/follower/follower_test.go`: replaced the direct `freeze(...)` re-detection driver (and its
  stale ~6-line comment block) with a second `PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id",
  time.Unix(2,0), alert, reg)`; asserted it returns `StatusVerified` with nil error, `violations` count
  == 2, `alerts` == 1, the cumulative `iscc_monitor_violations_total{hub_id="1",kind="fork"} 2` metric,
  the hub stays `Frozen`, and `LastSize` is still `m.size`.

**Verification:** `mise run check` → green (`go build ./...`, `go vet ./...`, `go test ./...` all 15
packages `ok`; `gofmt -l .` empty). Per-criterion:
- [x] `go test -count=1 -run TestPollHubFork ./internal/follower` PASSES.
- [x] `grep -rn "unordered LIMIT 1" internal/` → no matches.
- [x] No `freeze(` call inside `TestPollHubFork` (lines 185-345); re-detection driven by a second
  `PollHub(...)`.
- [x] After the second `PollHub`: `violations` == 2, `alerts` == 1, hub stays `Frozen`, `LastSize` ==
  `m.size`.
- [x] Mutation sanity (both reverted): (1) asserting `alerts == 2` → FAILS (`alerts=1`); (2) removing the
  second `PollHub` → FAILS (`violations=1` at reopen, `kind="fork"} 1`). Re-detection drive is
  non-vacuous and the once-only-alert invariant is genuinely exercised on the production path.
- [x] Conformance/consistency tests pass uncached
  (`TestPollHubFork|Shrink|Equivocation|Inclusion|Fsck` → `ok`); `derive_vkey.py` reproduces both
  golden vectors (`40b74463`, `22b08f3e`). `.claude/.scratch` removed after.
- [x] Production byte-unchanged: `git diff --quiet -- internal/follower/follower.go
  internal/store/checkpoints.go` exits 0. Only `follower_test.go` in the diff.

**Next:** Drain another ADR-0006 `normal` issue. Remaining open `normal` issues (none `critical`):
`AcceptCheckpoint` resolved-context reuse (verified polls re-fetch `did.json`); tile-writer
`p`-vocabulary; deep `AdvanceAccepted`; collapse the self-consistency decision into a pure
`logclient.CheckConsistency`. Or begin the proof-surface ETag/Cache-Control arc for the size-varying
`/inclusion`/`/consistency`/`/entries` surfaces. The `AcceptCheckpoint` context-reuse one is the
highest-value follower slice (cuts a redundant did.json fetch per verified poll).

**Notes:**
- Why the second `PollHub` re-detects (load-bearing, do not weaken): the first `freeze` did NOT advance
  the cursor, so `fs.LastSize` stays `m.size` and `fs.Frozen` becomes true. On the second poll
  `checkConsistency` runs BEFORE the `fs.Frozen` short-circuit, reads the prior accepted root via
  `CheckpointAt(hubID, m.size)` — which deterministically returns the **seed** root (lowest rowid), not
  the contradicting mirror root the first `freeze` persisted at a higher rowid — re-detects the fork, and
  re-freezes with `wasFrozen=true` so `alert` is NOT fired a second time. `RecordViolation` has no
  `ON CONFLICT`, so the re-detection records a fresh second row.
- `freeze`, `rootArray`, and `m.checkpoint` are all still referenced elsewhere
  (`equivocation_test.go`, `inclusion_test.go`), so dropping the direct `freeze(...)` call introduced no
  unused-symbol problem — `go vet`/`gofmt` stay clean. `freeze` keeps its real production caller in
  `follower.go:194`.
- No new imports (`PollHub`, `metrics`, `logclient`, `store` already in scope). The closed `normal`
  issue "`TestPollHubFork` re-detection still bypasses `PollHub`" is verified fixed.
- Branch is `develop`. Committed implementation test + handoff only.
