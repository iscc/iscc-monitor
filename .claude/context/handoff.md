## 2026-06-21 — Collapse the self-consistency decision into `logclient.CheckConsistency`

**Done:** Moved the load-bearing shrink/fork/equivocation branch order, the consistency-proof build
from mirrored tiles, and the narrow missing-tile swallow out of the follower into one pure,
table-testable `logclient.CheckConsistency(ctx, fetch, prevSize, prevRoot, prevFound, info) ->
(violated, kind, err)`. The follower's `checkConsistency` now only does the `CheckpointAt(prevSize)`
store read + `prevRaw` evidence bookkeeping and delegates the verdict; `PollHub` is unchanged.

**Files changed:**
- `internal/logclient/checkconsistency.go` (new): `CheckConsistency` composes
  `CheckShrink → CheckFork → ConsistencyProofFromTiles → CheckEquivocation` with the exact ported
  branch order from the old follower body (prevSize==0 early return, shrink/fork precompute with the
  `prevFound &&` fork guard, the `!prevFound || info.TreeSize <= prevSize` equivocation guard, the
  proof-build-error swallow to `(false, "", nil)`, and the `eerr`-wraps-to-Go-error path). No `store`
  import — takes the `TileFetcher` closure.
- `internal/follower/follower.go`: `checkConsistency` body replaced with the store lookup + delegate
  (`logclient.CheckConsistency(ctx, fetcher.ReadTile, prevSize, prevRoot, prevFound, info)`), returning
  the same `(violated, kind, prevRaw, err)` tuple. The `ConsistencyProofFromTiles`/`CheckEquivocation`
  calls are gone from the file. Also retouched one `PollHub`-body comment (line ~164) that named
  `ConsistencyProofFromTiles` so the moved-out symbol no longer appears in follower.go.
- `internal/logclient/checkconsistency_test.go` (new): table test over the boundary-crossing
  `testonly.Tree` (reuses `buildTree`/`tileFetcherFor` from `proofbuilder_test.go`). Cases: shrink,
  same-size fork, growing split view (corrupt new root → real proof fails → equivocation), clean
  growth (valid proof → not violated), prevSize==0, missing-tile fetcher (proof build errors →
  swallowed), and two `!prevFound` cases (root-checks skipped on growth; shrink still fires size-only).

**Verification:** `mise run check` → green (all 15 packages `ok`, incl. local `cmd/notecheck`);
`gofmt -l .` empty.
- `go test -run TestCheckConsistency ./internal/logclient` — PASS.
- `go test -run TestPollHub ./internal/follower` — PASS (delegated path: freeze/fork/equivocation/
  inclusion/fsck unchanged).
- `grep -n "ConsistencyProofFromTiles" internal/follower/follower.go` — nothing (exit 1).
- `go list -deps ./internal/logclient | grep iscc-monitor/internal/store` — empty (no new store edge).
- Mutation check (reverted): forcing the equivocation branch to drop its `(true, ViolationEquivocation)`
  return FAILS the growing-split-view case (`violated=false`/`kind=""` vs want `true`/`"equivocation"`)
  — the table is non-vacuous; a green-but-wrong verdict cannot ship.

**Next:** Drain the next ADR-0006 `normal` from `issues.md`. Candidates: the store-owned
`AdvanceAccepted` single-transaction write (collapse the three sequenced
`RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState` writes in `PollHub`); OR the tile-writer
`p`-vocabulary unification (delete the follower's `widthForP` copy).

**Notes:**
- **Oracle gate APPLIES (composes RFC-6962 consistency-proof verification) and is satisfied.** The
  table test builds merkle ground truth via `testonly.Tree` (independent prover) served through the
  same `tileFetcherFor` as the existing `ConsistencyProofFromTiles` golden; the composed verifier
  (`CheckConsistency`→`CheckEquivocation`→`VerifyConsistency`) is an independent path, so the
  growing-split-view cross-check is not a tautology. Mutation-proven above. `notecheck` runs in
  `mise run check`'s `cmd/notecheck` package — green. `derive_vkey.py` N/A (no signature/did:web path
  touched).
- **Pure port, byte-for-byte branch order.** The three trigger functions and `ConsistencyProofFromTiles`
  were not touched; `CheckConsistency` only composes them. Behavior is identical to the old follower
  body (incl. the missing-tile = clean-pass contract `PollHub` depends on) — no `PollHub` contract
  change, no new return type.
- **`prevSize==0` early return is now duplicated** (once in `follower.checkConsistency` to skip the
  store read, once in `logclient.CheckConsistency` as the pure guard). This is intentional per
  `next.md`: the follower keeps its copy to avoid a wasteful `CheckpointAt` at size 0; the logclient
  copy keeps the pure function total/correct on its own. Not debt.
- Touched a `PollHub`-body comment to remove the now-moved `ConsistencyProofFromTiles` name so the
  `next.md` grep criterion holds and the comment stays accurate (it now says "the prevSize->
  info.TreeSize consistency proof"). No code in `PollHub` changed.
