# Handoff

## 2026-06-21 — Port `InclusionProofFromTiles` — the tile-sourced RFC-6962 inclusion proof builder

**Done:** Added the exported `InclusionProofFromTiles(ctx, fetch TileFetcher, index, size uint64)
([][]byte, error)` to `internal/logclient/proofbuilder.go` — a faithful mirror of the sibling
`ConsistencyProofFromTiles`, swapping `proof.Consistency(smaller, larger)` for `proof.Inclusion(index,
size)` and passing `size` as the `logSize` arg to the unchanged `getNode`. Added the golden test
`inclusionproof_test.go` reusing the shared helpers from `proofbuilder_test.go`. Purely additive — no
production caller yet (intentional unused-until-wired seam, like `CheckEquivocation`/`LeafHashes`/
`RunFsck`), no `go.mod`/`go.sum` change.

**Files changed:**
- `internal/logclient/proofbuilder.go`: added `InclusionProofFromTiles` + doc comment, placed between
  `ConsistencyProofFromTiles` and `tileKey`. Reuses `getNode`/`tileKey`/`TileFetcher` unchanged;
  `ConsistencyProofFromTiles` is byte-identical (verified — the change is strictly additive).
- `internal/logclient/inclusionproof_test.go` (new): `package logclient` golden test, reuses
  `buildTree`/`tileFetcherFor`/`equalProof`/`fmtProof` from `proofbuilder_test.go` (not redefined).

**Verification:** `mise run check` → green (all 11 packages `ok`; logclient/follower ran uncached).
Per-criterion:
- [x] `mise run check` (build + vet + test) — green.
- [x] `gofmt -l internal/logclient/` — empty (ran `mise run fmt` first).
- [x] `go test -run TestInclusionProofFromTiles ./internal/logclient` — passes (3 subtests).
- [x] `go test -run 'TestConsistencyProofFromTiles|TestInclusionProofFromTiles' ./internal/logclient`
      — passes (consistency goldens still green; additive change confirmed).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exits 0 (WASM-share purity invariant intact).
- [x] `git diff --quiet HEAD -- go.mod go.sum` — exits 0 (no dependency change; no new import —
      `proof.Inclusion`/`proof.VerifyInclusion`/`tree.InclusionProof` were already in the closure).

**Oracle gate (APPLIES — RFC-6962 inclusion crypto):** satisfied by three independent merkle paths.
The golden builds a 300-leaf `testonly.Tree` (crosses the 256-leaf tile boundary: tile 0 full, tile 1
= 44-leaf partial at index 1), serves its tiles via `tileFetcherFor`, and for indices
`{0, 5, 200, 255, 256, 260, 299}` asserts the tile-built proof **byte-equals** `tree.InclusionProof`
AND **verifies** via `proof.VerifyInclusion(rfc6962.DefaultHasher, index, 300, tree.LeafHash(index),
got, tree.HashAt(300))`. The prover (`testonly.Tree.InclusionProof`), the verifier
(`proof.VerifyInclusion`), and the builder are independent → not a tautology. Mutation-proven
non-vacuous: forcing `proof.Inclusion(index+1, size)` (wrong leaf) made the byte-equality AND
`VerifyInclusion` FAIL for boundary indices (e.g. 256, 260) and tripped out-of-bounds at 299 — then
reverted. A green-but-wrong builder cannot ship. `notecheck`/`derive_vkey.py`/`fsck` ground-truth
oracles are correctly N/A for this slice (no signature/did:web/tile-rebuild path introduced; they
re-arm at the cross-check-vs-real-`IsccLogInclusionProof` slice).

**Next:** The fixture-and-wiring half deferred by this `next.md` — the inclusion **cross-check** that
asserts `InclusionProofFromTiles` byte-equals a hub's real `evidence.IsccLogInclusionProof`. Blocked
on two artifacts that do not exist yet: (1) captured `IsccLogInclusionProof` + tile + entry-bundle
fixtures in `testdata/live/`, and (2) the live tile-ingestion writer (making `PollHub` mirror real
tiles/bundles so `SQLiteFetcher.ReadTile` can source proof nodes). Pair it with the `fsck` root-rebuild
over `SQLiteFetcher` (`RunFsck` landed, still unwired) once that writer exists — both want the same
real tile fixtures. The live tile-ingestion writer is the natural next slice since it unblocks both.

**Notes:**
- The change is intentionally additive — `ConsistencyProofFromTiles` and its `getNode` helper are
  untouched, per `next.md`'s "do not extract a shared `fetchNodes`" KISS direction. Duplicating the
  tiny per-call fetch loop keeps the sibling byte-identical.
- Added one beyond-`next.md`-minimum subtest (`TestInclusionProofFromTilesIndexOutOfRange`) — `next.md`
  marked the `index >= size` case optional; it asserts a wrapped non-nil error and no panic (the
  fetcher is never reached), exercising `proof.Inclusion`'s precondition. No new assertions weakened.
- `proof.VerifyInclusion`'s arg order is `(hasher, index, size, leafHash, proof, root)` — note
  `leafHash` precedes `proof`, unlike `VerifyConsistency`'s `(…, size1, size2, proof, root1, root2)`.
  Confirmed against `merkle@v0.0.2/proof/verify.go:46`. The test uses `tree.HashAt(300)` for `root` and
  `tree.LeafHash(index)` for `leafHash`.
- (low, still open, untouched) `cmd/notecheck`'s `run` has a vestigial `out io.Writer` param — loop-
  skipped per `next.md`'s Not In Scope.
