# Next Work Package

## Step: Collapse the self-consistency decision into `logclient.CheckConsistency`

## Goal
Move the load-bearing shrink/fork/equivocation branch order, proof construction, and missing-tile
handling out of the follower and into one deep, table-testable
`logclient.CheckConsistency(...) -> (violated, kind, err)`. This closes the ADR-0006 `normal` issue
"self-consistency policy is split across follower orchestration and logclient helpers": the verdict
gets a single home that can be table-tested in `logclient` (shrink, same-size split view, growing
split view, clean growth, missing-tile), leaving the follower to only look up prior accepted evidence
and act on the returned verdict.

## Scope
- **Create**: `internal/logclient/checkconsistency.go` — the deep `CheckConsistency` entry point that
  composes `CheckShrink` → `CheckFork` → (build proof from tiles) → `CheckEquivocation`.
- **Modify**: `internal/follower/follower.go` — replace the body of `checkConsistency` (currently
  follower.go:416-457) so it does only the prior-checkpoint store lookup (`CheckpointAt(prevSize)`)
  and delegates the pure decision to `logclient.CheckConsistency`, returning the same
  `(violated, kind, prevRaw, err)` tuple `PollHub` already consumes (PollHub call site at
  follower.go:180 is unchanged).
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/consistency.go` — the three pure triggers
    (`CheckShrink`/`CheckFork`/`CheckEquivocation`, `ViolationKind` constants) to compose.
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder.go` — `ConsistencyProofFromTiles(ctx,
    fetch TileFetcher, smaller, larger)` and the `TileFetcher` type (lines 33-40) to accept.
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — `CheckpointInfo{Origin, TreeSize, Root}`
    shape (lines 64-68); `Root` is `[rootBytes]byte` (= `[32]byte`, `rootBytes` const in verify.go:35).
  - `/workspace/iscc-monitor/internal/follower/follower.go` — current `checkConsistency` (lines
    386-457) is the exact logic to port; preserve its doc-comment intent (error-vs-violation
    discipline, the narrow missing-tile swallow).
  - `/workspace/iscc-monitor/internal/logclient/consistency_test.go` — the existing
    `testonly.New(rfc6962.DefaultHasher)` / `ConsistencyProof` golden pattern to reuse for the table
    tests.

## Not In Scope
- The store-owned `AdvanceAccepted` single-transaction write (a separate `normal` issue) — leave the
  three sequenced `RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState` writes in `PollHub` untouched.
- The tile-writer `p`-vocabulary unification (`widthForP`) — separate `normal` issue.
- Changing any of the three pure trigger functions (`CheckShrink`/`CheckFork`/`CheckEquivocation`) or
  their semantics — `CheckConsistency` only *composes* them; their bodies stay byte-identical.
- Adding an "explicit indeterminate result" return type — keep the existing missing-tile = clean-pass
  behavior (`violated=false, err=nil`) so `PollHub`'s contract is unchanged; richer indeterminate
  modeling is a later step if ever needed.
- Touching `freeze`, `recordVerdict`, `cacheHubKey`, `fsckMirror`, the `PollHub` body, or the `store`
  package.

## Implementation Notes
- **Signature (pure decision lives in `logclient`).** `func CheckConsistency(ctx context.Context,
  fetch TileFetcher, prevSize uint64, prevRoot [rootBytes]byte, prevFound bool, info CheckpointInfo)
  (violated bool, kind ViolationKind, err error)`. The follower keeps owning the
  `CheckpointAt(prevSize)` store read and the `prevRaw` evidence bytes (persistence concerns, not pure
  policy), then calls `CheckConsistency` and pairs its verdict with `prevRaw`.
- **Port, don't rewrite.** The body is the existing `follower.checkConsistency` logic (follower.go:
  416-457) minus the `st.CheckpointAt` call: the `prevSize == 0` early return, the `shrink`/`fork`
  precompute (keep the `prevFound &&` guard on `CheckFork`), the `switch` over shrink → fork →
  default(equivocation), the `!prevFound || info.TreeSize <= prevSize` equivocation guard, the
  `ConsistencyProofFromTiles(ctx, fetch, prevSize, info.TreeSize)` build, the **narrow missing-tile
  swallow** (`perr != nil → return false, "", nil`), and the `CheckEquivocation` call whose `eerr`
  surfaces as a wrapped Go error. Keep the exact branch order.
- **Correctness rule (learnings.md, ADR-0006): a self-consistency violation freezes, never crashes.**
  The missing-tile / proof-build error stays swallowed to `(false, "", nil)` — freezing on a missing
  tile is a false positive, and a non-verifying proof is `CheckEquivocation`'s `(violated=true,
  err=nil)` verdict, never a Go error. Only a `CheckEquivocation` `err` (today unreachable-by-type,
  kept for symmetry) wraps as a returned `err`.
- **TileFetcher seam.** `CheckConsistency` takes the `TileFetcher` closure (matches
  `store.SQLiteFetcher.ReadTile` exactly), so the follower passes `fetcher.ReadTile` in. This keeps
  `logclient` free of any `store` import (dependency direction stays follower → logclient).
- **Follower delegate.** After porting, `follower.checkConsistency` becomes: keep the `prevSize == 0`
  early return (a store lookup at size 0 is wasteful), call `st.CheckpointAt(ctx, hubID, prevSize)`
  for `prevRootBytes`/`prevRaw`/`prevFound`, copy into `var prevRoot [32]byte`, then `violated, kind,
  err := logclient.CheckConsistency(ctx, store.SQLiteFetcher{Store: st, HubID: hubID}.ReadTile,
  prevSize, prevRoot, prevFound, info)` and return `(violated, kind, prevRaw, err)`. The follower no
  longer constructs the proof itself (`ConsistencyProofFromTiles` / `CheckEquivocation` calls move out
  of `follower.go`).
- **Test ground truth.** Reuse `consistency_test.go`'s `testonly.New(rfc6962.DefaultHasher)` pattern:
  build an append-only tree, `HashAt(M)`/`HashAt(N)` for roots, `ConsistencyProof(M,N)` for the valid
  proof, serve tiles through an in-test `TileFetcher`. The prover (`ConsistencyProof`) and the composed
  verifier (`CheckConsistency`→`CheckEquivocation`→`VerifyConsistency`) are independent merkle paths,
  so the cross-check is not a tautology. Cover: shrink (`next<prev`), same-size fork (`next==prev`,
  differing root), growing split view (corrupt the new root so the real proof fails to verify →
  `equivocation`), clean growth (valid proof → not violated), `prevSize==0` (always clean), and a
  missing-tile fetcher (`fetch` returns wrapped `os.ErrNotExist` → not violated, no error).
- **Oracle/conformance gate APPLIES** (this composes RFC-6962 consistency-proof verification): the new
  table tests must include the `testonly.Tree` merkle ground truth above, and the existing follower
  `TestPollHub*` conformance tests must stay green with the delegated call. Do not weaken any gate;
  mutation-check mentally — forcing `violated=false` must fail the growing-split-view case. The
  fully-independent `notecheck` oracle runs in `mise run check`'s `cmd/notecheck` package.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestCheckConsistency ./internal/logclient` passes (table tests: shrink, fork,
  growing split view, clean growth, prevSize==0, missing-tile).
- `go test -count=1 -run TestPollHub ./internal/follower` passes (the delegated follower path stays
  green: freeze/fork/equivocation/inclusion/fsck unchanged).
- `grep -n "ConsistencyProofFromTiles" internal/follower/follower.go` returns nothing (the proof build
  moved out of the follower into `logclient.CheckConsistency`).
- `go list -deps ./internal/logclient | grep "iscc-monitor/internal/store"` is empty (`logclient`
  gained no `store` import edge).

## Done When
`logclient.CheckConsistency` owns the pure shrink/fork/equivocation decision (branch order + proof
build + missing-tile swallow), the follower's `checkConsistency` only looks up prior accepted evidence
and delegates, and every Verification check passes with `mise run check` green.
