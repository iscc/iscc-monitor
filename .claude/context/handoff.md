## 2026-06-21 — Review of: Close M2's inclusion cross-check Verify bar with a conformance test over the real mirror

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/follower/inclusion_test.go` — a conformance test that drives a
verified `PollHub` over the in-process byte-accurate 300-leaf mirror, resolves sampled leaves'
`iscc_id → leafIndex` via the production `SeqsForISCCID`, builds each leaf's hub-side
`IsccLogInclusionProof` from the fixture `testonly.Tree`, and asserts `logclient.VerifyInclusionEvidence`
(recomputing over the mirrored `SQLiteFetcher` tiles) byte-matches it, with wrong-leaf + corrupted-proof
negatives. This closes M2's second Verify criterion. Test-only, scope-clean (1 test file + handoff, 0
production lines), and reviewer-mutation-proven non-vacuous — a green-but-wrong verify cannot ship.

**Verification:**
- [x] `mise run check` — GREEN (11 packages `ok`; build + vet + test, re-run after cache).
- [x] `go test -run TestPollHubInclusion -count=1 ./internal/follower` — PASS uncached (verified poll
  `StatusVerified`; fsck "Successfully fsck'd log with size 300"; leaf 5 → `[5]`, leaf 260 → `[260]`;
  cross-check nil for both; both negatives hit `ErrInclusionMismatch`).
- [x] `go test -run TestPollHub -count=1 ./internal/follower` — PASS uncached (no regression).
- [x] `gofmt -l .` — empty.
- [x] `git diff --quiet HEAD~1 -- internal/store/schema.sql go.mod go.sum` — exit 0 (no schema/dep change).
- [x] `go list -deps ./internal/store | grep -E 'internal/logclient|net/http'` — empty (store stays a leaf).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0 (WASM purity intact).
- [x] **Oracle gate (RFC-6962 inclusion crypto — APPLIES):** `./internal/logclient ./internal/follower
  ./internal/didweb` all PASS uncached; `python3 .claude/derive_vkey.py` reproduces `40b74463`/`22b08f3e`
  (scratch removed); `notecheck --vkey <sb0> < sb0_checkpoint` → `OK sb0.iscc.id/log` (exit 0), a sig-byte
  flip → `invalid signature` (exit 1). CI `notecheck` signature-parity job present + unchanged by this diff.
- [x] **Mutation re-verification (reviewer, reverted, tree clean):** short-circuiting
  `VerifyInclusionEvidence` to `return nil` before the proof compare → BOTH negatives FAIL
  (`inclusion_test.go:125` wrong-leaf, `:145` corrupted-proof). A verify ignoring proof bytes cannot ship.
- [x] **Gate-integrity scan (all unpushed commits):** no `//nolint`, `t.Skip`, build-tag, swallowed error,
  or deleted assertion/test in the added Go lines; the only unpushed code commit is the additive test file.
- [x] **Scope:** exactly 1 test file (`internal/follower/inclusion_test.go`) + the handoff; 0 production
  files; nothing in `## Not In Scope` touched (no tautological `PollHub` caller, no live-fixture capture,
  no HTTP proof-serving, no open `normal` follower issues, no `schema.sql`/`go.mod`/`go.sum`).

**Issues found:** (none) — clean slice. No issues resolved (all 5 open `normal` + 1 `low` issue reference
live, unchanged code and were correctly out of scope); none filed; none stale vs `state.md`.

**Next:** Serve `inclusion`/`consistency`/`entries` proofs over HTTP from the mirror — the next M2 slice,
giving the cross-check an *inbound* hub-evidence transport / `verify-for-me` surface — OR pick up one of
the open `normal` follower issues that rework the verified path (`CheckpointAt` `ORDER BY` for fork
re-detection, `AcceptCheckpoint` did.json reuse, frozen-hub advance short-circuit, tile `p`/`width`
de-duplication, `AdvanceAccepted` store method, `CheckConsistency` collapse). Each is its own
store/refactor-touching slice; the HTTP proof-serving slice is the natural moment to revisit several.

**Notes:**
- **Test-only is the correct, honest scope.** `VerifyInclusionEvidence` is the *consumer* of a
  hub-supplied proof and there is no inbound hub-evidence transport on the follow path yet; a `PollHub`
  step recomputing the monitor's own proof and checking it against itself would be circular (forbidden by
  `next.md`/target.md). The follower already mirrors tiles + indexes leaves, so the conformance bar is
  satisfiable as ordinary `go test` with no production line. Reviewer independently confirmed the symbols
  the test relies on (`SeqsForISCCID`, `SQLiteFetcher.ReadTile ≅ logclient.TileFetcher`, `m.tree.InclusionProof`,
  `buildVerifiedMirror`/`leafISCCID`/`noopAlert`/`openTemp`) all exist with matching signatures.
- **Non-vacuous via three independent paths:** `m.tree.InclusionProof` (prover/hub),
  `InclusionProofFromTiles` inside `VerifyInclusionEvidence` (monitor recompute over the mirror written by
  `ingestTiles`), and the base64 round-trip. The wrong-leaf negative is the sharp one — a valid leaf-5
  proof labelled leaf 6 still fails because the monitor recomputes leaf 6's distinct proof. `TestPollHub`
  / `TestPollHubFsck` poll the same fixture clean, so the test is not "always mismatches".
- **M2 is not complete** (so not DONE): proof-serving over HTTP, sb1 fixture refresh, real alert
  transport, and the 5 open `normal` issues remain; M3 / WASM / OTS are not started. CONTINUE.
- **Push:** remote `origin` configured; `develop` is the working branch (3 commits ahead). Pushing
  `develop` per protocol — never `main`.
