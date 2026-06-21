## 2026-06-21 — Review of: Mirror candidate tiles before the consistency check so a growing split view freezes

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** A pure reorder in `PollHub` moves the single `ingestTiles(...info.TreeSize...)` call ahead
of `checkConsistency`, so the candidate-size tiles are mirrored before the equivocation consistency
proof is built — closing the open `critical` trust-root gap where a growing split view hit a
missing-candidate-tile error that `checkConsistency` swallowed as a clean pass and then advanced
`last_size` to the inconsistent root. No new logic, exactly one `ingestTiles` invocation per poll,
`checkConsistency`'s missing-tile swallow left intact as a robustness guard, scope clean (1 production
file + 1 test file). The new end-to-end `TestPollHubGrowingSplitViewFreezes` is non-vacuous and every
handoff claim held under my own uncached re-runs.

**Verification:**
- [x] `mise run check` — green (all 11 packages `ok`; build + vet + test).
- [x] `go test -run TestPollHubGrowingSplitViewFreezes -count=1 ./internal/follower` — passes (uncached).
- [x] `go test -run 'TestPollHubEquivocation|TestEquivocationMissingTilesDoesNotFreeze' -count=1
  ./internal/follower` — passes (the direct-`checkConsistency` missing-tile robustness guard unchanged).
- [x] `go test -run 'TestPollHubVerifiedAdvances|TestPollHubMirrorsTiles|TestPollHubFsck' -count=1
  ./internal/follower` — passes (the consistent-growing advance + fsck rebuild unaffected by the reorder).
- [x] `go test -run TestPollHubFork -count=1 ./internal/follower` — passes (fork re-detection unaffected).
- [x] `gofmt -l .` — empty.
- [x] `git diff --quiet HEAD~1 -- go.mod go.sum internal/store/schema.sql` — exit 0 (no dep/schema change).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0 (WASM purity guard unaffected).
- [x] Structural: exactly one `ingestTiles(ctx, …)` call in production `follower.go` (line 163);
  `fsckMirror` still on the clean advance path (line 218). Order verified by grep:
  `FollowState`(145) → `ingestTiles`(163) → `checkConsistency`(169) → `freeze`(183) /
  `RecordCheckpoint`(194)+`AdvanceFollowState`(202).
- [x] **Oracle gate (APPLIES — RFC-6962 / consistency-proof / equivocation path):** full
  `./internal/follower ./internal/logclient` conformance suite green uncached (`fsck` root-rebuild +
  equivocation + inclusion cross-check); `derive_vkey.py` reproduces both golden vectors
  (`40b74463`/`22b08f3e`); the fully-independent `cmd/notecheck` oracle holds locally (accept
  `OK sb0.iscc.id/log` + reject a one-char-flipped sig) and is wired in CI (`.github/workflows/ci.yml`).
  No signature/proof crypto changed — only the order in which already-tested checks run.
- [x] **Gate-integrity scan** of all unpushed commits (`@{upstream}..HEAD`: `5a10f7c`/`9f43e18`/`05d7ee6`/
  `dbff98e`) — no `nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in the Go code diff.
- [x] **Scope:** exactly the 2 files `next.md` scoped; nothing in `## Not In Scope` was done (the
  `checkConsistency` swallow body is untouched — only its describing comment moved; no `fsckMirror`
  refactor, no frozen-hub or `CheckpointAt` fix bundled).

**Issues found:** (none). The resolved `critical` "Growing equivocations can be accepted before
candidate tiles are mirrored" was deleted from `issues.md` after verifying the fix end-to-end.

**Next:** Wire `VerifyInclusionEvidence` into `PollHub` + the `iscc_index` projection writer (the M2
second-half slice). This needs the `iscc_id → seq` projection to resolve a sampled leaf index and an
entry bundle to sample from; pass `store.SQLiteFetcher.ReadTile` straight in (signature already matches
`TileFetcher`). That wiring is the remaining gap before M2's Verify bar is met.

**Notes:**
- The reorder runs `ingestTiles` on the freeze path too: on a violation the candidate tiles are
  mirrored but the cursor is NOT advanced (freeze returns without `AdvanceFollowState`) — intended,
  tiles are rebuildable/evidence, not accepted state (ADR-0006 "preserve evidence"). No mirror-rollback
  was added.
- An `ingestTiles` fault (genuine transport/store error on a missing candidate tile) now aborts the
  poll before `checkConsistency`, so accepted state never advances — exactly the "missing proof tile →
  error, not clean pass" behavior the `critical` asked for.
- `mise run check` is GREEN. Two `normal` follower issues remain open and were correctly NOT bundled:
  frozen hubs still advance accepted state on later clean polls (`follower.go` reaches
  `AdvanceFollowState` when `fs.Frozen` is already true), and `CheckpointAt`'s unordered `LIMIT 1`
  re-detection fragility. M2 second-half wiring blocks DONE (not this slice's PASS).
- Active branch is `develop` (the session-start snapshot showing `main` was stale). Remote `origin`
  configured; pushing `develop` on this PASS.
