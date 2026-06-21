## 2026-06-21 — Frozen hubs are evidence-only: stop the clean re-poll from advancing accepted state

**Done:** Added the ADR-0006 already-frozen evidence-only short-circuit to `PollHub`: after the
self-consistency check and the `violated` branch, when `fs.Frozen` is true and the poll did NOT
re-detect a fresh violation, `PollHub` records the verdict metric (glossary `"frozen"`) and returns
`(status, nil)` WITHOUT `RecordCheckpoint`, `SetCoverage`, `AdvanceFollowState`, `cacheHubKey`, or
`fsckMirror`. This closes the silent freeze bypass where a frozen hub still advanced `last_size`,
coverage, the key cache, and ran fsck on a later clean-looking poll.

**Files changed:**
- `internal/follower/follower.go`: new `if fs.Frozen { recordVerdict(m, hubID, status, true,
  observedAt); return status, nil }` short-circuit placed after the `violated` branch and before
  building the `CheckpointRecord`; updated the `PollHub` doc comment (evergreen) to state an
  already-frozen hub is re-polled evidence-only and never advances accepted state. No other control
  flow touched — `ingestTiles` still runs before the consistency check, the `violated`/freeze path
  is untouched.
- `internal/follower/follower_test.go` (test): added `TestPollHubFrozenCleanRepollIsEvidenceOnly` —
  one clean verified poll seeds accepted state, `store.Freeze` freezes the hub directly, then a
  second poll over the SAME mirror (same size, same signed root → no shrink/fork/equivocation) is
  verified-and-clean. Asserts `LastSize` unchanged, `Coverage` unchanged, `hub_keys` count unchanged,
  hub stays `Frozen`, and the metric is `status="frozen"` (and NOT `status="verified"`). Reuses the
  existing `buildVerifiedMirror`/`mirrorLeaves`/`countRows`/`assertMetric` helpers — no new fixtures.

**Verification:** `mise run check` → green (build + vet + all 15 packages `ok`; `gofmt -l .` empty).
Per-criterion:
- `go test -count=1 -run TestPollHub ./internal/follower` → PASS (new frozen-clean-repoll test +
  `TestPollHubFork`/`Shrink`/`VerifiedAdvances`/`Unverified`/`CacheHit`/`Fsck`; no regression).
- `go test -count=1 -run TestTickFrozenUnaffected ./internal/follower` → PASS unchanged (the
  re-violation re-poll path is untouched; verbose output shows it still records 2 violations / 1
  alert and re-fires fsck on each due poll).
- New test asserts (verbose run confirms): after the clean frozen re-poll `LastSize` ==
  pre-repoll value, `Coverage().Set/Size/Since` unchanged, `hub_keys` count unchanged, hub stays
  `Frozen`, `hub_status{hub_id="1",status="frozen"} 1` emitted and no `status="verified"` line.
- `git diff --quiet -- go.mod go.sum internal/store/schema.sql` → exit 0 (pure control-flow fix).

**Mutation check (non-vacuous):** removed the short-circuit → the new test FAILS on the
`status="verified"` assertion (and the cursor/coverage/key would all advance), confirming a
green-but-wrong implementation cannot ship. Restored.

**Next:** The ADR-0006 frozen-advance gap (the highest-value `normal` issue) is now closed. Two
remaining cluster items from the same issue set: (1) the `TestPollHubFork` re-detection-via-second-
`PollHub` cleanup + the stale "unordered LIMIT 1" comment in `CheckpointAt` (a follower-*test*
behavior change, explicitly deferred this slice); (2) collapsing the self-consistency decision into a
pure `logclient.CheckConsistency` and/or adding a deep `store.AdvanceAccepted` transaction method.
Alternatively begin the proof-bundle JSON + verify-for-me arc.

**Notes:**
- Conformance/oracle gate: this slice touches the freeze *decision wiring* but NOT signature
  verification, RFC-6962 consistency math, or proof code — `checkConsistency`/`freeze`/`fsckMirror`/
  `RunFsck` are byte-unchanged. All existing conformance tests (`TestPollHubFsck`,
  `TestPollHubFork`/`Shrink`, the equivocation/inclusion tests, didweb/logclient goldens) pass under
  `mise run check`; the fsck root-rebuild runs green (`Successfully fsck'd log with size 300 and root
  e7077fda…`). The `notecheck` oracle runs in CI.
- Behavioral nuance for review: the short-circuit means `fsckMirror` no longer runs on a frozen clean
  re-poll. That is correct — a frozen hub is evidence-only; the mirror rebuild already ran on the
  polls that established accepted state, `ingestTiles` still runs (tiles stay mirrored for
  `fsck`-verifiability via the `SQLiteFetcher`), and re-running fsck would only re-verify already-
  accepted state on a hub that can no longer advance. The verbose test output confirms the re-poll
  emits no second fsck log line, unlike the unfrozen `TestTickFrozenUnaffected` clean hub B.
- The test seeds the freeze via `store.Freeze` directly (one of the two paths `next.md` allowed)
  rather than driving a first `PollHub` into a violation, because freezing directly keeps the seed
  deterministic and avoids the `CheckpointAt` unordered-`LIMIT 1` re-detection fragility that
  `TestPollHubFork` documents (and which is its own deferred slice). The re-poll is clean because it
  is the SAME mirror at the SAME size/root — neither shrink, fork, nor equivocation can trip.
