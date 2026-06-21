# Next Work Package

## Step: Frozen hubs are evidence-only — stop the clean re-poll from advancing accepted state

## Goal
A hub that is already `frozen` must never advance its accepted state on a later
clean-looking poll (ADR-0006: frozen = evidence-only until a manual unfreeze). Today
`PollHub` reaches `RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState`/`cacheHubKey`/
`fsckMirror` whenever a verified observation is not a *fresh* violation, even if the hub
is already frozen — a silent bypass of the freeze. This closes that gap and re-arms the
ADR-0006 trust path that the handoff flagged as the highest-value next slice.

## Scope
- **Modify**: `internal/follower/follower.go` (the single production file — add the
  already-frozen evidence-only short-circuit in `PollHub` after `checkConsistency`).
- **Modify (test)**: `internal/follower/follower_test.go` (new test for the
  frozen-clean-repoll case; does not count toward the 3-file budget).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/issues.md` → "Frozen hubs still advance
    accepted state on later clean-looking polls" (the issue this resolves).
  - `/workspace/iscc-monitor/internal/follower/follower.go` lines 125-223 (`PollHub`) and
    440-477 (`freeze`) — the control flow being changed; `fs` (with `fs.Frozen`) is already
    read at line 145.
  - `/workspace/iscc-monitor/internal/follower/loop_test.go` lines 135-244
    (`TestTickFrozenUnaffected`) — the existing re-poll-of-frozen-hub test (its shrink
    re-fires every poll, so it exercises the *re-violation* branch, NOT the clean-repoll
    branch; confirm it still passes unchanged).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` lines 196-210
    (`AdvanceFollowState`) and 247-262 (`Freeze`) — confirms `AdvanceFollowState` never
    clears `frozen`, so the bug is purely that the accepted-state writes still run, not
    that frozen gets cleared.

## Not In Scope
- The separate `TestPollHubFork` re-detection-via-second-`PollHub` cleanup + stale
  "unordered LIMIT 1" comment removal (its own `normal` issue, its own slice — a
  follower-*test* behavior change, explicitly deferred in state.md).
- Collapsing the self-consistency decision into a pure `logclient.CheckConsistency`
  (separate `normal` refactor issue).
- Adding a deep `store.AdvanceAccepted` transaction method (separate `normal` issue).
- Any manual-unfreeze / auto-unfreeze mechanism (v1 has no unfreeze; do not add one).
- Changing `freeze`, the metrics surface, the loop cadence, or proof/cache HTTP surfaces.

## Implementation Notes
- The minimal change is in `PollHub` (`follower.go`), immediately after the
  `if violated { … }` block returns (around line 184) and before building the
  `store.CheckpointRecord` at line 186. `fs.Frozen` is already in scope (read at line 145).
  Add: when `fs.Frozen` is true (and we did NOT just re-detect a violation — that path
  already `return`ed inside the `violated` branch), short-circuit to evidence-only:
  record the verdict metric and return, WITHOUT `RecordCheckpoint`, `SetCoverage`,
  `AdvanceFollowState`, `cacheHubKey`, or `fsckMirror`.
- **Metrics correctness:** call `recordVerdict(m, hubID, status, /*frozen=*/true, observedAt)`
  on this branch so the hub keeps mapping to the glossary `"frozen"` label
  (`glossaryStatus(status, true) == "frozen"`), never `"verified"`. Return
  `(status, nil)` — the signature was valid, freezing is a separate axis (ADR-0006).
- **Ordering matters:** keep this AFTER `checkConsistency` + the `violated` branch, so a
  frozen hub that re-serves a *fresh* contradiction (the existing `TestTickFrozenUnaffected`
  shrink case) still records the re-detected violation as evidence and re-fires the
  violations counter via the existing `violated` path. The new branch handles only the
  *clean* re-poll (no fresh violation) of an already-frozen hub.
- **Tiles already ingested:** `ingestTiles` runs at line 163, before `checkConsistency`.
  That is fine to leave running on a frozen poll — tiles are rebuildable evidence, not
  accepted state (the doc comment at lines 159-162 already says a violation leaves tiles
  mirrored but the cursor not advanced). Do NOT move or skip `ingestTiles`.
- Correctness rule from `learnings.md` / `CLAUDE.md` glossary: "A self-consistency
  violation freezes … keep polling evidence-only … no auto-unfreeze, survive restart"
  (ADR-0006). The accepted-state advance is exactly the side effect "evidence-only" must
  suppress. `AdvanceFollowState` deliberately omits `frozen` from its conflict update, so
  the bug never *un*froze the hub — it advanced `last_size`, coverage, key cache, and ran
  fsck on a frozen hub, which this step stops.
- **Update the `PollHub` doc comment** (lines 40-49) to state that an already-frozen hub
  is re-polled evidence-only and never advances accepted state — keep the comment evergreen
  (describe current behavior, not the fix).
- New test (mirror the structure of `TestPollHubFork`/`TestPollHubShrink` in
  `follower_test.go`): freeze a hub first (seed a prior accepted checkpoint at a size
  strictly *smaller* than the in-process mirror, record a violation, and call
  `store.Freeze`, OR drive one `PollHub` into a freeze), capture `LastSize`/`Coverage`/
  `hub_keys` count, then `PollHub` again with the verified mirror at a size that does NOT
  re-trigger a violation (verified AND clean relative to the seeded prior accepted size).
  Assert that after the clean frozen re-poll: `FollowState.LastSize` is unchanged (no
  advance), `Coverage().Set` is unchanged, `hub_keys` row count is unchanged, the hub stays
  `Frozen`, and the metric maps to `hub_status{…,status="frozen"}` (not `"verified"`).
  Reuse the existing `buildVerifiedMirror`/`assertMetric`/`countRows`/`rootArray` helpers
  in the package (do not add new fixtures).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`,
  `gofmt -l .` empty).
- `go test -count=1 -run TestPollHub ./internal/follower` passes (the new
  frozen-clean-repoll test plus the existing `TestPollHubFork`/`TestPollHubShrink`/
  clean-advance tests — no regression).
- `go test -count=1 -run TestTickFrozenUnaffected ./internal/follower` still passes
  unchanged (the re-violation re-poll path is untouched).
- The new test asserts, after a clean verified poll of an already-frozen hub at a size
  that does not re-trigger a violation: `FollowState.LastSize` equals the pre-repoll value
  (NOT the new size), `Coverage().Set` is unchanged, `hub_keys` row count is unchanged, the
  hub stays `Frozen`, and `hub_status{hub_id="…",status="frozen"} 1` is emitted (not
  `status="verified"`).
- `git diff --quiet -- go.mod go.sum internal/store/schema.sql` exits 0 (no dependency or
  schema change — pure follower control-flow fix).

## Done When
`PollHub` short-circuits an already-frozen hub to evidence-only (no checkpoint record,
coverage, follow-cursor advance, key cache, or fsck) on a clean verified re-poll, the new
`follower_test.go` test pins that behavior, and all Verification criteria pass.
