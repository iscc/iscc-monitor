## 2026-06-21 — Review of: Drive `TestPollHubFork` re-detection through a second `PollHub`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `TestPollHubFork`'s fork re-detection block now drives a real second `PollHub` (later
`observedAt=time.Unix(2,0)`) instead of calling `freeze(...)` directly, exercising the freeze +
once-only-alert + evidence-accumulation invariants on the production code path; the stale "unordered
LIMIT 1 … non-deterministic" comment is gone. Test-only change — no production file touched, trust
root untouched and green. Scope-clean and reviewer-mutation-proven non-vacuous.

**Verification:**
- [x] `mise run check` — green (15 packages `ok`; re-ran `go test -count=1 ./...` uncached, all 15
  pass; `gofmt -l .` empty).
- [x] `go test -count=1 -run TestPollHubFork ./internal/follower` — PASS (uncached).
- [x] `grep -rn "unordered LIMIT 1" internal/` — no matches (stale comment removed).
- [x] Re-detection driven by a second `PollHub`, not `freeze(...)` — `grep -n "freeze(" internal/
  follower/follower_test.go` shows no `freeze(` anywhere in the test file; `freeze` keeps its
  production caller at `follower.go:194` and two `equivocation_test.go` callers (no unused-symbol /
  dead-code regression; `rootArray`/`m.checkpoint` still referenced elsewhere).
- [x] Post-re-detection asserts — `violations == 2`, `alerts == 1`, hub stays `Frozen`, `LastSize ==
  m.size`, cumulative metric `iscc_monitor_violations_total{hub_id="1",kind="fork"} 2`. All present.
- [x] Mutation sanity — reviewer independently reproduced BOTH (and reverted): (1) asserting `alerts
  == 2` → FAILS (`alerts=1`, once-only-alert genuinely exercised); (2) removing the second `PollHub`
  → FAILS three assertions (violations count, the `kind="fork"} 2` metric, reopen-survival). The
  re-detection drive is non-vacuous on the production path.
- [x] Production byte-unchanged — `git diff --quiet HEAD~1..HEAD -- internal/follower/follower.go
  internal/store/checkpoints.go` exits 0; the only `.go` file in the diff is `follower_test.go`.
- [x] Trust-root oracle gate — `derive_vkey.py` reproduces both golden vectors (`40b74463`,
  `22b08f3e`); conformance/consistency tests (`TestPollHubFork|Shrink|Equivocation|Inclusion|Fsck`)
  pass uncached. Scratch (`.claude/.scratch`) removed after.
- [x] Gate-integrity scan over the 3 unpushed commits (update-state/define-next/advance) — no
  `//nolint`, `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion.

**Issues found:** (none in scope). Resolved + deleted the `normal` issue "`TestPollHubFork`
re-detection still bypasses `PollHub`" (verified fixed). Four `normal` + one `low` issue remain open
in `issues.md`, untouched by this slice.

**Next:** Drain the next ADR-0006 `normal` issue. Highest-value follower slice is `AcceptCheckpoint`
resolved-context reuse (verified polls currently re-fetch `did.json` two/three times per poll — cut
the redundant resolves while preserving the ADR-0009 per-poll validity check). Other open `normal`s:
tile-writer `p`-vocabulary (push `widthForP` into the store, delete the follower copy); deep
store-owned `AdvanceAccepted` (one transaction for record+coverage+advance); collapse the
self-consistency decision into a pure `logclient.CheckConsistency`. Or begin the proof-surface
ETag/Cache-Control arc.

**Notes:**
- The re-detection determinism is genuine and verified from source: on the second poll
  `checkConsistency` (follower.go:180) runs BEFORE the `fs.Frozen` short-circuit (follower.go:204) and
  reads the prior accepted root via `CheckpointAt(hubID, m.size)`, whose EXPLICIT `ORDER BY rowid LIMIT
  1` (checkpoints.go:159) returns the lowest-rowid seed root — not the contradicting evidence the first
  `freeze` persisted at a higher rowid. `CheckFork(seed != mirror)` re-fires; `freeze(wasFrozen=true)`
  re-records evidence without re-alerting. Updated the now-obsolete "no ORDER BY / implicit insert-order
  dependency" learnings bullet to reflect the explicit ordering + this end-to-end drive.
- `internal/proof` still does not exist; the WASM purity invariant rides on `internal/didweb`
  (`GOOS=js GOARCH=wasm go build ./internal/didweb` green). This diff touches no purity-relevant code.
- CI (`.github/workflows/ci.yml`) carries the `notecheck` signature-parity oracle (accept + reject a
  one-char-flipped sig); it was green at the last pushed commit `be6ccd3` (run success). This slice is
  test-only and does not touch signature/RFC-6962/did:web code, so parity is unaffected; the 3 unpushed
  commits will be CI-verified on push.
- Out-of-band: `.devcontainer/devcontainer.json` carries an uncommitted local working-tree edit (adds
  a `runArgs` port mapping). It is environment infrastructure, unrelated to this iteration, and was
  NOT staged or committed by this review. Left in place for the human to handle.
- Branch is `develop`. Pushing on PASS.
