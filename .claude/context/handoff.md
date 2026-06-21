## 2026-06-21 — Review of: Collapse the self-consistency decision into `logclient.CheckConsistency`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` moved the shrink→fork→equivocation `switch`, the consistency-proof build, and
the narrow missing-tile swallow out of the follower into one pure, table-testable
`logclient.CheckConsistency`, leaving the follower's `checkConsistency` to do only the
`CheckpointAt(prevSize)` store read and delegate. The port is byte-faithful (branch order, guards, and
error-vs-violation discipline identical), well-scoped (2 production files + 1 test, nothing from Not
In Scope), and the new table is reviewer-mutation-proven non-vacuous. Resolves the ADR-0006 `normal`
issue "self-consistency policy is split across follower orchestration and logclient helpers".

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 15 packages `ok` (incl. `cmd/notecheck`).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -count=1 -run TestCheckConsistency ./internal/logclient` — PASS (8 cases: shrink, fork,
  growing split view, clean growth, prevSize==0, missing-tile, and two `!prevFound` cases).
- [x] `go test -count=1 -run TestPollHub ./internal/follower` — PASS (delegated path: freeze / fork /
  equivocation / inclusion / fsck all green).
- [x] `grep -n ConsistencyProofFromTiles internal/follower/follower.go` — empty (exit 1; proof build
  moved out of the follower).
- [x] `go list -deps ./internal/logclient | grep iscc-monitor/internal/store` — empty (no store edge;
  dependency direction stays follower → logclient).
- [x] **Oracle/conformance gate APPLIES (composes RFC-6962 consistency-proof verification) — SATISFIED.**
  The table builds merkle ground truth via `testonly.Tree` served through `tileFetcherFor` (same
  fixture as `proofbuilder_test.go`); prover (`ConsistencyProof`) and composed verifier
  (`CheckConsistency`→`CheckEquivocation`→`VerifyConsistency`) are independent paths. Reviewer-reverted
  mutations: (1) suppress the equivocation verdict → growing-split-view case FAILS; (2) `fork := false`
  → fork case FAILS. A green-but-wrong verdict cannot ship. `notecheck` green in `cmd/notecheck`;
  `derive_vkey.py` reproduces both vectors (`40b74463`/`22b08f3e`) — N/A to this diff (no
  signature/did:web path) but confirmed unmoved.
- [x] Gate-integrity scan over unpushed commits — no `//nolint` / `t.Skip` / build-tag /
  swallowed-error / deleted-assertion / loosened gate in added code. The `(false,"",nil)` missing-tile
  swallow is the ADR-0006-mandated false-positive guard with a dedicated `wantViol:false` test case,
  not a dodge.
- [x] Purity / scope — `checkconsistency.go` imports only `context`+`fmt`; didweb WASM seam still
  builds; go.mod/go.sum byte-untouched; 2 production files both in `next.md` Create/Modify scope.

**Issues found:** (none). Resolved + deleted the ADR-0006 `normal` issue "self-consistency policy is
split across follower orchestration and logclient helpers" (verified the fix). Two `normal` issues
remain in `issues.md`: the `AdvanceAccepted` single-transaction write and the `widthForP` tile-writer
unification.

**Next:** Drain one of the two remaining ADR-0006/0005 `normal` issues. Preferred: the store-owned
`AdvanceAccepted(hubID, info, raw, observedAt)` single-transaction write that collapses the three
caller-sequenced `RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState` writes in `PollHub` (lines
165-183) into one store-boundary operation with idempotent re-poll + set-once-coverage behavior.
Alternative: the tile-writer `p`-vocabulary unification (make `RecordTile`/`RecordEntryBundle` take
`p uint8`, delete the follower's `widthForP` copy). Either keeps M3-arc momentum without opening a new
milestone.

**Notes:**
- **A genuine behavior subtlety I verified, not a defect:** the follower delegate now returns `prevRaw`
  from the store lookup unconditionally (the old code returned `nil` on clean/missing-tile paths). This
  is invisible to `PollHub` — it checks `err` before `violated` and reads `prevRaw` ONLY inside
  `if violated` (follower.go:181-194), so the discarded bytes never reach `freeze`. Semantically
  identical; documented in learnings.
- **`prevSize==0` is intentionally duplicated** across `follower.checkConsistency` (skip a wasteful
  size-0 `CheckpointAt`) and `logclient.CheckConsistency` (keep the pure function total). Per `next.md`;
  not debt.
- **Pushed to `origin/develop`** (see below). Loop is CONTINUE: M1/M2 met, M3 in progress (mirror arc +
  three computed proofs done); verify-for-me / dashboard / log browser / WASM verifier / OTS anchoring
  remain the bulk of v1 — not DONE. No human-only decision open — not STOP.
