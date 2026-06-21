## 2026-06-21 — Review of: Frozen hubs are evidence-only — stop the clean re-poll from advancing accepted state

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `PollHub` now short-circuits an already-frozen hub to evidence-only on a clean verified
re-poll: `if fs.Frozen { recordVerdict(m, hubID, status, true, observedAt); return status, nil }`,
placed after `checkConsistency` + the `violated` branch and before `RecordCheckpoint`, so no
checkpoint record, coverage, follow-cursor advance, key cache, or fsck runs (ADR-0006). The change is
scope-clean (1 production file + its test + handoff), trust-root code is byte-unchanged, the new test
is reviewer-mutation-proven non-vacuous, and the full gate is green.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test ./...` all 15 packages `ok`; re-ran
  `go test -count=1 ./...` uncached → all `ok`.
- [x] `gofmt -l .` empty — clean.
- [x] `go test -count=1 -run TestPollHub ./internal/follower` — PASS (new frozen-clean-repoll test +
  Fork/Shrink/VerifiedAdvances/Unverified/CacheHit/Fsck, no regression).
- [x] `go test -count=1 -run TestTickFrozenUnaffected ./internal/follower` — PASS unchanged (the
  re-violation re-poll path is untouched).
- [x] New test asserts after the clean frozen re-poll: `LastSize` unchanged, `Coverage` unchanged,
  `hub_keys` row count unchanged, hub stays `Frozen`, `hub_status{hub_id="1",status="frozen"} 1`
  emitted and NO `status="verified"` line. Verbose run confirms the fsck log line fires once (seed
  poll only) — the re-poll did not re-run fsck.
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 — no
  dependency or schema change.
- [x] Mutation check (reviewer-reproduced, reverted): deleting the short-circuit makes
  `TestPollHubFrozenCleanRepollIsEvidenceOnly` FAIL on `status="verified"` (the hub re-advances). A
  green-but-wrong implementation cannot ship.
- [x] Oracle/conformance gate — correctly N/A for the trust-root math, but exercised anyway: this
  slice touches only the freeze *decision wiring* (`checkConsistency`/`freeze`/`fsckMirror`/`RunFsck`/
  `AcceptCheckpoint`/merkle/didweb all byte-unchanged). Fresh `TestPollHubFsck`/`Equivocation`/
  `Inclusion` pass; `derive_vkey.py` reproduces both golden vectors (`40b74463`, `22b08f3e`). CI
  `notecheck` signature-parity oracle is unaffected (checkpoint bytes + verification code untouched).
- [x] Gate-integrity scan over unpushed commits (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-assertion in code (the only matches are context-doc prose).
- [x] Scope discipline — only the two authorized files; nothing from `## Not In Scope` touched
  (`freeze`, `CheckpointAt`, `TestPollHubFork`, metrics surface, HTTP surfaces all untouched). Purity
  invariant holds (`GOOS=js GOARCH=wasm go build ./internal/didweb` OK; `internal/proof` not yet
  born).

**Issues found:** (none) — the `normal` issue "Frozen hubs still advance accepted state on later
clean-looking polls" is verified fixed and removed from `issues.md`.

**Next:** Drain another ADR-0006 `normal` issue or begin the proof-surface cache arc. Highest-value
candidates: (1) the `TestPollHubFork` re-detection-via-second-`PollHub` cleanup + stale "unordered
LIMIT 1" comment removal (a quick win now that `CheckpointAt` is deterministic); (2) collapse the
self-consistency decision into a pure `logclient.CheckConsistency`; (3) `AcceptCheckpoint` resolved-
context reuse so verified polls stop re-fetching `did.json`; or (4) extend ETag/Cache-Control to the
size-varying `/inclusion`/`/consistency`/`/entries` proof surfaces.

**Notes:**
- 5 `normal` issues remain open (was 6); none is `critical`, none blocks this slice's PASS, all block
  DONE. M3 (verify-for-me, dashboard, log browser), the WASM verifier, and OTS anchoring are the bulk
  of the remaining v1 work — Loop is CONTINUE, not DONE.
- Behavioral nuance confirmed correct: the short-circuit means `fsckMirror` no longer runs on a frozen
  clean re-poll. That is right — the mirror rebuild already ran on the polls that established accepted
  state, `ingestTiles` still runs (tiles stay mirrored for `fsck`-verifiability), and re-running fsck
  would only re-verify already-accepted state on a hub that can no longer advance. Verbose test output
  shows exactly one fsck log line (the seed poll), none on the re-poll.
- The test seeds the freeze via `store.Freeze` directly rather than driving a first `PollHub` into a
  violation — the cleanest isolation of the already-frozen re-poll path, and it sidesteps the
  `TestPollHubFork` re-detection fragility (its own deferred issue).
- Branch is `develop`, in sync with `origin/develop`; remote `origin` configured. Pushing on PASS.
