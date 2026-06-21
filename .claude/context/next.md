# Next Work Package

## Step: Mirror candidate tiles before the consistency check so a growing split view freezes

## Goal
Close the open `critical` trust-root gap: `PollHub` currently builds the growing-pair RFC-6962
consistency proof from a mirror that holds tiles only up to the *previously* accepted size, so a
growing split view hits a missing-candidate-tile error that `checkConsistency` swallows as a clean
pass — and then advances `last_size` to the inconsistent root and never freezes. Mirroring the
candidate tiles *before* the self-consistency check makes the proof buildable, so a genuine growing
equivocation is detected and frozen (ADR-0006).

## Scope
- **Modify**: `internal/follower/follower.go` (reorder `PollHub`: call `ingestTiles(...info.TreeSize...)`
  BEFORE `checkConsistency`, so the candidate-size tiles are mirrored before the equivocation proof is
  built; update the file/`PollHub` doc comments to describe the new order).
- **Modify**: `internal/follower/equivocation_test.go` (add the end-to-end growing split-view test
  through `PollHub`; this is a test file and does not count against the 3-file budget).
- **Reference**:
  - `/workspace/iscc-monitor/internal/follower/follower.go` — current `PollHub` (lines 122-211),
    `checkConsistency` (lines 385-426; the missing-tile swallow at 412-415), the `ingestTiles` call at
    line 197 and `fsckMirror` call at line 206.
  - `/workspace/iscc-monitor/internal/follower/ingest.go` — `ingestTiles` signature (line 40); it is
    transport+CRUD only and returns a genuine fetch/store error (never a violation).
  - `/workspace/iscc-monitor/internal/follower/equivocation_test.go` — `buildEquivTree`,
    `equivNodeHash`, `seedMirrorTiles`, `flipByte`, `rootArray`, `equivPrevSize` (5),
    `equivTreeLeaves` (300), and `TestEquivocationMissingTilesDoesNotFreeze` (which calls
    `checkConsistency` directly and must keep passing unchanged — the missing-tile swallow stays as a
    robustness guard).
  - `/workspace/iscc-monitor/internal/follower/fsck_test.go` — `buildVerifiedMirror`,
    `mirrorBundleFetcher`, `fsckOrigin` ("sb0.iscc.id/log"), `mirrorLeaves` (300); the `byPath` fetcher
    fixture for a self-consistent candidate tree (did.json + signed checkpoint + byte-accurate tiles).
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` — `openTemp`, `noopAlert`,
    `countRows`, `assertViolation` helpers.
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder.go` — `ConsistencyProofFromTiles` wraps a
    missing tile as `os.ErrNotExist` via `%w` (lines 60-84, `getNode` 139-153); confirms why the
    pre-ingest order produced a swallowed missing-tile error.

## Not In Scope
- Wiring `VerifyInclusionEvidence` into `PollHub` or the `iscc_index` projection writer (the M2
  second-half slice — a separate, later step).
- Fixing the `normal` "frozen hubs still advance accepted state" issue or the `normal` `CheckpointAt`
  unordered `LIMIT 1` issue (separate backlog entries; do not bundle).
- Removing or rewriting the `checkConsistency` missing-tile swallow (lines 412-415). It stays as a
  robustness guard for the genuine no-mirror case (e.g. a hub that advanced before tiles were
  mirrored); the reorder is what closes the gap. Do not also turn it into a hard error in this step.
- Refactoring `fsckMirror`, `cacheHubKey`, or the metrics wiring.

## Implementation Notes
- The minimal correct change is a **reorder in `PollHub`**, not new logic. After `AcceptCheckpoint`
  returns `StatusVerified` and `st.FollowState(ctx, hubID)` is read, call
  `ingestTiles(ctx, st, fetcher, hubID, baseURL, info.TreeSize, observedAt)` **before**
  `checkConsistency(...)`. On a growing pair this mirrors the candidate-size tiles, so
  `ConsistencyProofFromTiles(prevSize, info.TreeSize)` can build the proof and `CheckEquivocation`
  returns a true verdict on an inconsistent root — the hub freezes instead of silently advancing.
- Move the single `ingestTiles` call up; do NOT duplicate it. There must remain exactly one
  `ingestTiles` invocation per `PollHub`. Keep `fsckMirror` where it is, on the clean advance path
  after `RecordCheckpoint`/`AdvanceFollowState` — it still has mirrored tiles because `ingestTiles`
  now runs earlier in the same call.
- On a violation the candidate tiles are now already mirrored but the cursor is NOT advanced (freeze
  returns without `AdvanceFollowState`). That is intended: tiles are rebuildable/evidence, not
  accepted state (Correctness rule "partial-tile discipline" + ADR-0006 "preserve evidence"). Do NOT
  add logic to roll back the mirror on a freeze.
- A genuine transport/store fault during the moved `ingestTiles` is returned as the existing
  `%w`-wrapped "ingest tiles" error and aborts the poll WITHOUT advancing — which is exactly the
  "missing proof tile → error, not clean pass" behavior the `critical` issue asks for. No accepted
  state changes on that path.
- `buildVerifiedMirror(t, mirrorLeaves)` already serves byte-accurate bytes for every coord
  `tiles.TileCoords(300)` / `BundleCoords(300)` enumerates, so the candidate side of the new test is a
  normal self-consistent tree whose signature verifies and whose tiles fsck-rebuild.
- **New test (`TestPollHubGrowingSplitViewFreezes`)** — construct a growing split view routed through
  the full `PollHub` chain:
  - Build a self-consistent candidate via `m := buildVerifiedMirror(t, mirrorLeaves)` (size 300). Its
    `m.fetcher` serves the did.json, the signed candidate checkpoint, and byte-accurate candidate tiles.
  - Seed a *prior accepted* checkpoint at a smaller size (e.g. `equivPrevSize` = 5) whose root is a
    **wrong/fabricated** prior root — use `flipByte(rootArray(t, m.tree.HashAt(equivPrevSize)))` so the
    consistency proof from prior→candidate cannot verify — then `AdvanceFollowState(ctx, hubID, 5)`.
    The candidate checkpoint the fetcher signs is internally valid; the inconsistency is between the
    seeded prior accepted root and the candidate root, which is exactly a growing split view against
    this monitor.
  - Call `PollHub(ctx, s, m.fetcher, hubID, "https://sb0.iscc.id", observedAt, alert, nil)` with a
    counting `AlertFunc`; assert it returns `(StatusVerified, nil)` (freeze, never crash — ADR-0006).
  - Assert via observable store outputs ONLY (PRD outbound-fetch seam rule — never follower
    internals): `assertViolation(t, path, hubID, "equivocation")`; `FollowState.Frozen == true`;
    `FollowState.LastSize == equivPrevSize` (did NOT advance to 300); exactly one alert;
    `countRows(t, path, "violations") == 1`.
  - Non-vacuity is provided by the pair with the existing `TestPollHubVerifiedAdvances` /
    `TestPollHubMirrorsTiles` (a *consistent* growing observation over the same mirror advances and
    does not freeze) — so the suite catches both a "never freezes" and an "always freezes" wiring.
- Correctness rule in play (learnings.md): "A self-consistency violation freezes, never crashes
  (ADR-0006) — three triggers fork/shrink/equivocation; persist evidence, set `frozen=1`, alert once,
  do not advance." The reorder is what makes the equivocation trigger actually fire on a growing pair.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestPollHubGrowingSplitViewFreezes -count=1 ./internal/follower` passes (the new
  end-to-end growing split-view freeze test).
- `go test -run 'TestPollHubEquivocation|TestEquivocationMissingTilesDoesNotFreeze' -count=1
  ./internal/follower` still passes (the direct-`checkConsistency` missing-tile robustness guard is
  unchanged).
- `go test -run 'TestPollHubVerifiedAdvances|TestPollHubMirrorsTiles|TestPollHubFsck' -count=1
  ./internal/follower` still passes (the consistent-growing advance path and the fsck rebuild are
  unaffected by the reorder).
- `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no dependency or schema
  change — this is a follower-only reorder).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (WASM purity guard unaffected).

## Done When
`PollHub` mirrors candidate tiles before the self-consistency check, a growing split view freezes
without advancing accepted state (`TestPollHubGrowingSplitViewFreezes` green), the pre-existing
equivocation/advance/fsck tests still pass, and `mise run check` is green — closing the open `critical`
issue.
