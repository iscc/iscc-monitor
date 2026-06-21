## 2026-06-21 — Mirror candidate tiles before the consistency check so a growing split view freezes

**Done:** Reordered `PollHub` so `ingestTiles(...info.TreeSize...)` runs BEFORE `checkConsistency`, so
the candidate-size tiles are mirrored before the equivocation consistency proof is built. This closes
the open `critical` trust-root gap: a growing split view now freezes instead of hitting a
missing-candidate-tile error that `checkConsistency` swallowed as a clean pass and silently advancing
`last_size` to the inconsistent root (ADR-0006). It is a pure reorder — no new logic, exactly one
`ingestTiles` invocation per poll, `fsckMirror` left where it is on the clean advance path.

**Files changed:**
- `internal/follower/follower.go`: moved the single `ingestTiles` call up to run right after
  `FollowState` and before `checkConsistency` (was after `cacheHubKey`, before `fsckMirror`); updated
  the file-level and inline doc comments to describe the mirror-before-check order; trimmed the
  now-stale "this must run AFTER ingestTiles" note on `fsckMirror` (the tiles are now mirrored earlier
  in the same poll).
- `internal/follower/equivocation_test.go`: added `TestPollHubGrowingSplitViewFreezes` — the
  end-to-end growing split-view freeze test through the full `PollHub` chain (self-consistent
  candidate via `buildVerifiedMirror(t, mirrorLeaves)` at size 300, prior accepted seeded at size 5
  with a `flipByte`'d wrong root, counting alert). Asserts via observable store outputs only:
  `(StatusVerified, nil)`, `assertViolation(..."equivocation")`, `Frozen == true`,
  `LastSize == equivPrevSize` (5, did NOT advance to 300), exactly one alert, `countRows(violations) == 1`.

**Verification:** `mise run check` → green (all 11 packages `ok`; build + vet + test). Per-criterion:
- [x] `go test -run TestPollHubGrowingSplitViewFreezes -count=1 ./internal/follower` — passes (the new freeze test).
- [x] `go test -run 'TestPollHubEquivocation|TestEquivocationMissingTilesDoesNotFreeze' -count=1 ./internal/follower` — passes (the direct-`checkConsistency` missing-tile robustness guard is unchanged).
- [x] `go test -run 'TestPollHubVerifiedAdvances|TestPollHubMirrorsTiles|TestPollHubFsck' -count=1 ./internal/follower` — passes (consistent-growing advance + fsck rebuild unaffected by the reorder).
- [x] `gofmt -l .` — empty.
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` — exit 0 (no dep/schema change; follower-only reorder).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0 (WASM purity guard unaffected).
- [x] Exactly one `ingestTiles(ctx, …)` call in production `follower.go` (line 163); `fsckMirror` still on the clean advance path (line 218).

**Next:** Wire `VerifyInclusionEvidence` into `PollHub` + the `iscc_index` projection writer (the M2
second-half slice) — the natural successor now that the candidate tiles are mirrored before the checks.
The two `normal` backlog issues also remain open and were correctly NOT bundled here: frozen hubs still
advance accepted state, and `CheckpointAt`'s unordered `LIMIT 1` (the re-detection fragility flagged in
`TestPollHubFork`).

**Notes:**
- The `checkConsistency` missing-tile swallow (`follower.go:412-415`) is intentionally left intact as a
  robustness guard for the genuine no-mirror case (a hub that advanced before tiles were mirrored), per
  `next.md` Not In Scope. The reorder is what closes the gap; the swallow was NOT turned into a hard error.
- On a freeze the candidate tiles are now already mirrored but the cursor is NOT advanced (freeze
  returns without `AdvanceFollowState`) — intended: tiles are rebuildable/evidence, not accepted state.
  No mirror-rollback logic was added (ADR-0006 "preserve evidence").
- Non-vacuity: `TestPollHubVerifiedAdvances`/`TestPollHubMirrorsTiles` poll the SAME `buildVerifiedMirror`
  at size 300 with a CONSISTENT prior and DO advance without freezing — so the suite catches both a
  "never freezes" and an "always freezes" wiring. I confirmed the new test exercises the moved path: it
  reuses the existing `flipByte`/`rootArray`/`equivPrevSize` helpers, and the freeze depends on the
  candidate tiles being present (which only the reorder provides through `PollHub`).
- Oracle gate: this slice touches consistency/proof code (the equivocation freeze path), and the
  monitor's conformance tests cover it and pass under `mise run check` — `TestPollHubGrowingSplitViewFreezes`
  (end-to-end equivocation freeze over the mirror), `TestPollHubEquivocation` (direct
  `checkConsistency` + the consistent-root non-vacuity pair), `TestPollHubFsck` (fsck root-rebuild over
  the `SQLiteFetcher`), and the `internal/logclient` inclusion cross-check tests all green. No
  signature-verification or proof crypto was changed — only the order in which the already-tested mirror
  and consistency checks run.
