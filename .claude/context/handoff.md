## 2026-06-21 — Close M2's inclusion cross-check Verify bar with a conformance test over the real mirror

**Done:** Added `internal/follower/inclusion_test.go`, a conformance test that drives a verified
`PollHub` over the in-process byte-accurate 300-leaf mirror, resolves sampled leaves'
`iscc_id → leafIndex` via the production `SeqsForISCCID`, builds each leaf's hub-side
`IsccLogInclusionProof` from the fixture `testonly.Tree`, and asserts `logclient.VerifyInclusionEvidence`
(recomputing the proof from the mirrored `SQLiteFetcher` tiles) byte-matches it. This re-arms the
inclusion cross-check oracle gate on the real verified path and completes M2's second Verify criterion.
Test-only by design — no production line added (see Notes).

**Files changed:**
- `internal/follower/inclusion_test.go` (new): the conformance test. Polls `buildVerifiedMirror(300)`
  → `StatusVerified`; for leaf 5 (bundle 0) and leaf 260 (past the 256-leaf boundary) asserts
  `SeqsForISCCID == [leafIndex]` and `VerifyInclusionEvidence(ctx, SQLiteFetcher.ReadTile, ev) == nil`;
  two negatives (wrong-leaf: valid leaf-5 proof re-labelled leaf 6; corrupted-proof: one decoded hash
  byte flipped) both assert `errors.Is(err, logclient.ErrInclusionMismatch)`.

**Verification:** `mise run check` → GREEN (11 packages `ok`, build + vet + test). Per criterion:
- `go test -run TestPollHubInclusion -count=1 ./internal/follower` → PASS (verified poll `StatusVerified`;
  leaf 5 → `[5]`, leaf 260 → `[260]`; cross-check nil for both; both negatives hit `ErrInclusionMismatch`).
- `go test -run TestPollHub -count=1 ./internal/follower` → PASS (no regression).
- `gofmt -l .` empty; `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exit 0;
  `git status --porcelain` shows only the new test file (+ this handoff).
- `go list -deps ./internal/store | grep -E 'internal/logclient|net/http'` empty (store stays a leaf).
- **Oracle gate (RFC-6962 inclusion crypto — APPLIES):** `./internal/logclient ./internal/follower
  ./internal/didweb` all PASS; `python3 .claude/derive_vkey.py` reproduces `40b74463`/`22b08f3e`
  (then `rm -rf .claude/.scratch`); `notecheck --vkey <sb0> < sb0_checkpoint` → `OK sb0.iscc.id/log`
  (exit 0) and a byte-flipped checkpoint → `invalid signature` (exit 1).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exit 0 (WASM purity intact).

**Next:** Serve `inclusion`/`consistency`/`entries` proofs over HTTP from the mirror (the next M2 slice
that gives the cross-check an *inbound* hub-evidence transport / `verify-for-me` surface), OR pick up one
of the open `normal` follower issues that rework the verified path (`CheckpointAt` ordering,
`AcceptCheckpoint` context/did.json reuse, frozen-hub advance, tile `p`/`width` duplication,
`AdvanceAccepted`, `CheckConsistency` collapse) — each its own store/refactor-touching slice.

**Notes:**
- **Test-only is the honest scope (deliberate, stated in the file docstring).** `VerifyInclusionEvidence`
  is the consumer of a hub-supplied proof; there is no inbound hub-evidence transport on the follow path
  yet (no `FetchInclusionEvidence`). A `PollHub` step recomputing the monitor's OWN proof and checking it
  against itself would be circular and is forbidden by `next.md` / target.md. The follower already mirrors
  tiles (`ingestTiles`) and indexes leaves (`projectEntryBundle`), so the conformance test needs no
  production change to drive the real path.
- **Non-vacuous via three independent paths:** `m.tree.InclusionProof` (prover/hub),
  `InclusionProofFromTiles` inside `VerifyInclusionEvidence` (monitor recompute over the mirror written
  by `ingestTiles`), and the base64 round-trip. The wrong-leaf negative is the sharp one — a *valid*
  leaf-5 proof labelled leaf 6 still fails because the monitor recomputes leaf 6's different proof, so a
  verify that ignored the proof bytes would not pass. Confirmed the negatives are real: `TestPollHub` and
  `TestPollHubFsck` poll the same fixture clean, so the test is not "always mismatches".
- `notecheck` reads the vkey from `--vkey` and the checkpoint from **stdin** (not a path arg) — corrected
  the earlier handoff's invocation shape; it still passes.
- Scope clean: 0 production files changed; no `schema.sql`/`go.mod`/`go.sum` touch; none of the open
  `normal` follower issues touched (all correctly out of scope for this slice). No new module dependency.
