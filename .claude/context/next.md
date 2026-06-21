# Next Work Package

## Step: Port `InclusionProofFromTiles` — the tile-sourced RFC-6962 inclusion proof builder

## Goal
Add the pure inclusion-proof *source* the M2 inclusion cross-check needs: build an RFC-6962
inclusion proof for a leaf index from a hub's mirrored hash tiles, mirroring the already-landed
`ConsistencyProofFromTiles`. This is the first, golden-testable half of the handoff's "inclusion
cross-check vs the hub's `IsccLogInclusionProof`" — broken out as a pure unit before the
fixture-and-wiring half that depends on captured `IsccLogInclusionProof`/tile fixtures and the
live tile-ingestion writer (neither exists yet).

## Scope
- **Modify**: `internal/logclient/proofbuilder.go` — add one exported function
  `InclusionProofFromTiles(ctx, fetch TileFetcher, index, size uint64) ([][]byte, error)` plus a
  doc comment. Reuse the existing `getNode`, `tileKey`, and `TileFetcher` already in this file
  unchanged. (1 non-test/doc file.)
- **Create**: `internal/logclient/inclusionproof_test.go` — golden test in `package logclient`
  reusing the helpers already defined in `proofbuilder_test.go` (`buildTree`, `tileFetcherFor`,
  `equalProof`, `fmtProof`, `bytesEqual`) — do NOT redefine them (same package → redefinition won't
  compile).
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder.go` (the sibling
    `ConsistencyProofFromTiles` to mirror exactly).
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder_test.go` (the golden-test pattern + the
    shared helpers to reuse).
  - `/workspace/iscc-monitor/cauldron/tessera/client/client.go` lines ~197-249
    (`ProofBuilder.InclusionProof` + `fetchNodes` — the upstream this ports; note it calls
    `proof.Inclusion(index, treeSize)`).

## Not In Scope
- The inclusion *cross-check* itself (asserting our proof byte-equals the hub's
  `evidence.IsccLogInclusionProof`) — needs real captured `IsccLogInclusionProof` + tile +
  entry-bundle fixtures that do not exist in `testdata/live/` yet. Defer.
- Wiring `InclusionProofFromTiles` into `follower.PollHub` or any production caller — it lands as an
  intentional unused-until-wired export seam (like `CheckEquivocation`, `LeafHashes`, `RunFsck`).
- The live tile-ingestion writer (making `PollHub` mirror real tiles/bundles) — separate later slice.
- Refactoring `ConsistencyProofFromTiles` or extracting a shared `fetchNodes` helper — keep the change
  additive; duplicating the tiny per-call fetch loop is the KISS choice and leaves the existing
  function byte-identical.
- The `cmd/notecheck` `out io.Writer` low issue — loop-skipped.

## Implementation Notes
- Port from tessera `ProofBuilder.InclusionProof`: it is `proof.Inclusion(index, treeSize)` then
  `fetchNodes`. Your version is structurally identical to `ConsistencyProofFromTiles` with exactly two
  swaps: call `proof.Inclusion(index, size)` instead of `proof.Consistency(smaller, larger)`, and pass
  `size` (the only tree size) as the `logSize` argument to `getNode` (matching how `larger` is the
  logSize in the consistency builder — `index` selects the leaf, `size` is the tree the proof is
  computed against). Reuse the same per-call `tiles := make(map[tileKey]api.HashTile)` cache + loop +
  final `nodes.Rehash(hashes, rfc6962.DefaultHasher.HashChildren)`.
- `proof.Inclusion` is already in the closure (`github.com/transparency-dev/merkle/proof`, used by
  `ConsistencyProofFromTiles`). Signature: `func Inclusion(index, size uint64) (Nodes, error)`,
  requires `0 <= index < size`. `proof.VerifyInclusion(hasher, index, size, leafHash, proof, root)` is
  the verifier for the test. No new import, no `go.mod`/`go.sum` change — confirm after with
  `git diff --quiet HEAD -- go.mod go.sum`.
- Keep purity (Correctness rule: `proof/verify` is pure / WASM-shareable): import only what
  `proofbuilder.go` already imports; do NOT add `net`/`net/http`/`database/sql`/`os`. A missing tile
  must stay a `%w`-wrapped error so `errors.Is(err, os.ErrNotExist)` survives — `getNode` already does
  this; you only forward its error wrapped with an `InclusionProofFromTiles:` prefix (this file never
  references `os` directly, same as the sibling).
- Error wrap style: mirror `ConsistencyProofFromTiles` exactly — wrap `proof.Inclusion`'s error as
  `InclusionProofFromTiles: compute node list for (index %d, size %d): %w`, `getNode`'s error as
  `InclusionProofFromTiles: get node %+v: %w`, and `nodes.Rehash`'s error as
  `InclusionProofFromTiles: rehash proof: %w`.
- Test (oracle gate APPLIES — RFC-6962 inclusion crypto): reuse `buildTree(treeLeaves)` (300 leaves,
  crosses the 256-leaf tile boundary) and `tileFetcherFor(t, tree, treeLeaves)` from
  `proofbuilder_test.go`. For leaf indices that exercise both tile 0 (full) and tile 1 (partial) —
  e.g. `{0, 5, 200, 255, 256, 260, 299}` — assert `InclusionProofFromTiles(ctx, fetch, index, 300)`
  byte-equals `tree.InclusionProof(index, 300)` (independent prover = ground truth, not a tautology)
  AND verifies via `proof.VerifyInclusion(rfc6962.DefaultHasher, index, 300, tree.LeafHash(index),
  got, tree.HashAt(300))`. Add a missing-tile case (fetcher returns a wrapped `os.ErrNotExist`) that
  asserts `errors.Is(err, os.ErrNotExist)` survives, mirroring `TestConsistencyProofFromTilesMissingTile`.
  The prover (`testonly.Tree.InclusionProof`), the verifier (`proof.VerifyInclusion`), and the builder
  are three independent merkle paths, so the cross-check is non-circular.
- Mutation hint for `advance`/`review`: forcing the builder to return the proof from a wrong index
  (or a one-byte-corrupted node) must make the byte-equality AND `VerifyInclusion` fail for at least
  one boundary index — confirm before declaring done so a green-but-wrong builder can't ship.
- Edge case: `proof.Inclusion` requires `index < size`; pick all test indices `< 300`. An `index >=
  size` test is optional (it is `proof.Inclusion`'s precondition, not this layer's contract); if added,
  assert a wrapped non-nil error and no panic.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass).
- `gofmt -l internal/logclient/` is empty.
- `go test -run TestInclusionProofFromTiles ./internal/logclient` passes.
- `go test -run 'TestConsistencyProofFromTiles|TestInclusionProofFromTiles' ./internal/logclient`
  passes (the consistency goldens still pass — the change is additive).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (the WASM-share purity invariant unbroken).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change).

## Done When
`InclusionProofFromTiles` builds a tile-sourced inclusion proof that byte-equals
`testonly.Tree.InclusionProof` and verifies via `proof.VerifyInclusion` for leaf indices spanning the
256-leaf tile boundary, the missing-tile error preserves `os.ErrNotExist`, and every Verification
check passes with no `go.mod`/`go.sum` change.
