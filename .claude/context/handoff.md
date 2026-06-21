# Handoff

## 2026-06-21 — Review of: Port `InclusionProofFromTiles` — the tile-sourced RFC-6962 inclusion proof builder

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the exported `InclusionProofFromTiles(ctx, fetch TileFetcher, index, size
uint64) ([][]byte, error)` to `internal/logclient/proofbuilder.go` plus a golden test
`inclusionproof_test.go` reusing the sibling's shared helpers. It is a faithful, structurally-identical
mirror of `ConsistencyProofFromTiles` with exactly the two intended swaps (`proof.Inclusion(index,
size)` and `size` as the `getNode` `logSize`); the sibling, `getNode`, `tileKey`, and `TileFetcher` are
byte-untouched. Every Verification criterion passes and the RFC-6962 oracle gate is mutation-proven
non-vacuous two independent ways.

**Verification:**
- [x] `mise run check` (build + vet + test) — green (all 11 packages `ok`; re-ran `logclient`
      uncached → `ok`, `go vet ./...` clean).
- [x] `gofmt -l internal/logclient/` — empty. Also ran `gofmt -l .` over the whole tree → empty.
- [x] `go test -run TestInclusionProofFromTiles ./internal/logclient` — passes (3 subtests:
      golden, missing-tile, index-out-of-range).
- [x] `go test -run 'TestConsistencyProofFromTiles|TestInclusionProofFromTiles' ./internal/logclient`
      — passes (consistency goldens still green; additive change confirmed).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0 (WASM-share purity invariant intact).
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` — exit 0 (no dependency change; `proof.Inclusion`
      was already in the closure via the sibling).

**Oracle gate (APPLIES — RFC-6962 inclusion crypto):** SATISFIED + independently mutation-proven by the
reviewer. The golden builds the 300-leaf `testonly.Tree` boundary fixture and, for
`{0,5,200,255,256,260,299}`, asserts the tile-built proof byte-equals `tree.InclusionProof(index,300)`
AND verifies via `proof.VerifyInclusion(hasher, index, 300, tree.LeafHash(index), got, tree.HashAt(300))`
— prover, verifier, and builder are three independent merkle paths. I ran two mutations (both reverted,
file confirmed byte-identical to HEAD after): (1) `proof.Inclusion(index+1, size)` → root mismatch at
256/260 + out-of-bounds at 299; (2) corrupting every fetched node hash in the shared `getNode` loop →
root mismatch. Both FAILED the golden — a green-but-wrong builder cannot ship. `notecheck` /
`derive_vkey.py` / `fsck` ground-truth oracles are correctly N/A for this slice (no signature / did:web
/ tile-rebuild path introduced; they re-arm at the cross-check-vs-real-`IsccLogInclusionProof` slice).
Port faithfulness confirmed against `cauldron/tessera/client/client.go:201` (`ProofBuilder.
InclusionProof` → `proof.Inclusion(index, pb.treeSize)` → `fetchNodes`): structurally identical, otel
spans dropped, `%w` wraps instead of `%v` (better for `errors.Is`).

**Issues found:** (none) — purely additive seam, no behavior to fix.

**Gate-integrity scan:** clean over all 3 unpushed commits (`@{upstream}..HEAD`). No `//nolint`,
`t.Skip`, swallowed errors, build-tag exclusions, or loosened assertions in added lines (the two grep
hits were handoff prose: a prior gate-scan note and the word "build target", not a `//+build` tag).

**Scope discipline:** clean. 1 non-test/doc file (`proofbuilder.go`) + 1 new test file + handoff —
well within ≤3. No "Not In Scope" work leaked in: no production caller was wired (`grep` confirms
`InclusionProofFromTiles` appears only in its own definition + test), `ConsistencyProofFromTiles` was
not refactored, no shared `fetchNodes` extracted, the `cmd/notecheck out io.Writer` low issue untouched.

**Next:** The fixture-and-wiring half this `next.md` deferred — the inclusion **cross-check** asserting
`InclusionProofFromTiles` byte-equals a hub's real `evidence.IsccLogInclusionProof`. Still blocked on two
artifacts that do not exist: (1) captured `IsccLogInclusionProof` + tile + entry-bundle fixtures in
`testdata/live/`, and (2) the live tile-ingestion writer making `PollHub` mirror real tiles/bundles so
`SQLiteFetcher.ReadTile` can source proof nodes. The **live tile-ingestion writer** is the natural next
slice — it unblocks both the inclusion cross-check AND the `fsck` root-rebuild over `SQLiteFetcher`
(`RunFsck` landed, still unwired); both want the same real tile fixtures.

**Notes:**
- Both proof builders intentionally have no explicit `index < size` / `max > treeSize` guard before the
  `proof.*` call (unlike tessera's `ConsistencyProof`, which guards `max > pb.treeSize`). They lean on
  `proof.Inclusion`/`proof.Consistency`'s own precondition — verified safe: `index == size` returns a
  wrapped non-nil error, never panics, never reaches the fetcher.
- `InclusionProofFromTiles` is an intentional unused-until-wired export seam (like
  `ConsistencyProofFromTiles`, `CheckEquivocation`, `LeafHashes`, `RunFsck`) — `go vet` clean, not dead
  code. Its first caller is the M2 inclusion cross-check above.
- The test reuses `buildTree`/`tileFetcherFor`/`equalProof`/`fmtProof`/`treeLeaves` from
  `proofbuilder_test.go` (same package) without redefinition — confirmed (no duplicate func/const).
- (low, still open, untouched) `cmd/notecheck`'s `run` has a vestigial `out io.Writer` param — correctly
  loop-skipped per `next.md`'s Not In Scope (low priority, skipped by the loop).
- Branch is `develop` with upstream `origin/develop`; remote configured → pushing on PASS.
