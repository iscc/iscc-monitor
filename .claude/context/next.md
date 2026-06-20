# Next Work Package

## Step: Pure `ConsistencyProofFromTiles` builder over the tile-fetch seam (`internal/logclient`)

## Goal
Build the RFC-6962 consistency proof between two tree sizes from mirrored hash tiles, as a pure
function over a tile-level fetcher — the missing proof *source* that `CheckEquivocation` needs. This
is the prerequisite that unblocks wiring the third self-consistency trigger (equivocation) into the
follower without yet needing real on-disk tile fixtures: it is golden-testable against a synthetic
`testonly.Tree` whose tiles are computed in-test.

## Scope
- **Create**: `internal/logclient/proofbuilder.go` — the pure proof builder.
- **Create**: `internal/logclient/proofbuilder_test.go` — golden test against a real `testonly.Tree`
  (tests do not count against the 3-file budget).
- **Modify**: (none expected — see Implementation Notes; if a tiny re-export is genuinely needed,
  add ONE delegate to `internal/tiles/layout.go`, still ≤3 non-test files.)
- **Reference**:
  - `cauldron/tessera/client/client.go` — port `ProofBuilder.ConsistencyProof` (~lines 214–231),
    `fetchNodes` (~233–245), and `nodeCache.GetNode` (~385–435). This is the structure to port; do
    **not** import `tessera/client` (it pulls `net/http`/`otel`/`klog`). Drop the otel spans.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/merkle@v0.0.2/proof/proof.go` —
    `proof.Consistency(size1,size2)`, `Nodes.IDs`, `Nodes.Rehash(h, hc)`, `Nodes.Ephem`.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/merkle@v0.0.2/compact/nodes.go` +
    `.../compact/range.go` — `compact.NodeID{Level,Index}`, and
    `compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}.NewEmptyRange(0)` + `Append` +
    `GetRootHash(nil)` to recompute a node hash from a tile's leaves.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/api/layout/tile.go` —
    `layout.NodeCoordsToTileAddress(treeLevel, treeIndex)` and `layout.PartialTileSize`.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/api/state.go` —
    `api.HashTile{}.UnmarshalText` (the tlog-tiles concatenated-32-byte format).
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/merkle@v0.0.2/testonly/tree.go` —
    `testonly.New`, `Tree.HashAt(size)`, `Tree.ConsistencyProof(s1,s2)` for the golden vector.
  - `internal/logclient/consistency.go` — the `CheckEquivocation` consumer this feeds (its
    `consistencyProof [][]byte` arg). `rootBytes = 32` lives in `verify.go`.
  - `internal/store/fetcher.go` — `SQLiteFetcher.ReadTile(ctx, l, i uint64, p uint8) ([]byte, error)`
    is the production fetcher whose shape the new `TileFetcher` seam must match exactly.

## Not In Scope
- **Do NOT wire anything into `internal/follower/`** — no third branch in `checkConsistency`, no
  `SQLiteFetcher` call from the follower. That is the *next* step, which this one unblocks.
- **Do NOT add real tile fixtures** under `testdata/live/`. The golden tree is synthesized in-test
  (`testonly.Tree`); on-disk fixtures + the `fsck` root-rebuild are a separate later slice.
- **Do NOT import `tessera/client` or `tessera/fsck`** (they pull `net/http`/`otel`/`klog`, breaking
  `logclient`'s WASM-clean seam and churning `go.mod`). Port the logic; reuse only
  `merkle/{proof,compact,rfc6962}` and `tessera/api{,/layout}` (all dep-clean, already required).
- Inclusion-proof building, entry-bundle reads, the M2 serving `ProofBuilder`, and `iscc_index`.

## Implementation Notes
- **Signature.** Define a tile-fetch seam matching tessera's `TileFetcherFunc` and the existing
  `store.SQLiteFetcher.ReadTile` exactly, so the follower can later pass `SQLiteFetcher.ReadTile`
  directly:
  ```go
  type TileFetcher func(ctx context.Context, level, index uint64, p uint8) ([]byte, error)
  func ConsistencyProofFromTiles(ctx context.Context, fetch TileFetcher, smaller, larger uint64) ([][]byte, error)
  ```
  Return `[][]byte` shaped to feed `CheckEquivocation`'s `consistencyProof` argument directly.
- **Algorithm (ported from `client.go`).** `proof.Consistency(smaller, larger)` → a `proof.Nodes`;
  fetch each `compact.NodeID` hash via a `getNode` helper; collect them in `nodes.IDs` order
  (mirroring `fetchNodes`); then `nodes.Rehash(hashes, rfc6962.DefaultHasher.HashChildren)` to fold
  the ephemeral node and return the proof.
- **Node → tile mapping (the load-bearing port of `nodeCache.GetNode`).** For a `compact.NodeID{Level,
  Index}`: `tileLevel, tileIndex, nodeLevel, nodeIndex := layout.NodeCoordsToTileAddress(uint64(Level),
  uint64(Index))`; the tile's partial qualifier is `p := layout.PartialTileSize(tileLevel, tileIndex,
  larger)`; fetch `fetch(ctx, tileLevel, tileIndex, p)`; `var t api.HashTile; t.UnmarshalText(raw)`.
  Then recompute the node hash from the tile leaves: `numLeaves := 1 << nodeLevel; firstLeaf :=
  int(nodeIndex) * numLeaves; lastLeaf := firstLeaf + numLeaves`; guard `lastLeaf <= len(t.Nodes)`;
  fold `t.Nodes[firstLeaf:lastLeaf]` through `compact.RangeFactory{Hash:
  rfc6962.DefaultHasher.HashChildren}.NewEmptyRange(0)` + `Append(leaf, nil)` + `GetRootHash(nil)`. A
  simple per-call map cache keyed by `(tileLevel, tileIndex)` is fine (KISS) — no need to port the full
  `nodeCache` struct or its ephemeral-node map.
- **`larger` is the log size** passed to `PartialTileSize` (tessera uses `n.logSize`), so the partial
  qualifier reflects the *newer* tree the proof is computed against.
- **Boundaries / errors.** `proof.Consistency` returns `Nodes{IDs: []}` for `smaller==0` or
  `smaller==larger`, so `Rehash` yields an empty proof — return an empty/`nil` slice cleanly so the
  `CheckEquivocation` guard short-circuits correctly. A genuine tile-fetch fault (a real
  `os.ErrNotExist` or transport error) is a Go error returned to the caller — distinct from
  `CheckEquivocation`'s "proof fails to verify = violation" verdict. Wrap fetch/parse errors with
  `%w` so `errors.Is(err, os.ErrNotExist)` survives a missing tile (matches the `SQLiteFetcher`
  contract).
- **Purity / WASM (Correctness rule: `proof/verify` is pure; keep WASM-shareable).** `tessera/api`
  and `tessera/api/layout` closures are stdlib-only (verified: no `net/http`/`otel`/`klog`), and
  `merkle/{proof,compact,rfc6962}` are already in `logclient`'s closure. This file must NOT pull
  `net`/`database/sql`/`net/http`. The one allowed I/O-ish import is `os` only as the `os.ErrNotExist`
  sentinel surfaced from the fetcher (already in `logclient` via `checkpoint.go`). Verify the package
  still builds under `GOOS=js GOARCH=wasm`.
- **`go.mod` must stay byte-identical.** Both `merkle v0.0.2` and `tessera v1.0.2` are already direct
  requires, `merkle/compact` is already in the closure, and `tessera/api` is a dep-clean graph member
  — adding it is not a new require. `go mod tidy` must remain a no-op; any `go get` is a red flag.
- **Golden test = ground truth, not author-asserted (Oracle gate APPLIES — RFC-6962 crypto).** Build
  a `testonly.New(rfc6962.DefaultHasher)` tree of ~300 leaves (crosses a full 256-tile boundary so the
  node→tile mapping is exercised at level-0 index 0 *and* index 1). Serialize its hash tiles into an
  in-test `TileFetcher` (fill each tile's `api.HashTile.Nodes` with the tree's level-0 leaf hashes for
  that tile range — `tree.LeafHash(i)` — and `MarshalText` them, since tiles store the bottom-row
  leaves and the builder recomputes interior nodes via the `RangeFactory`). Assert
  `ConsistencyProofFromTiles(ctx, fetch, s1, s2)` byte-equals `tree.ConsistencyProof(s1, s2)` for
  several `(s1<s2)` pairs, and that the result verifies via
  `proof.VerifyConsistency(rfc6962.DefaultHasher, s1, s2, got, tree.HashAt(s1), tree.HashAt(s2))`. The
  prover (`tree.ConsistencyProof`) and the tile-sourced builder are independent code paths, so the
  match is not a tautology. Include the empty-proof boundary (`s1==0`, `s1==s2`). If reconstructing
  tiles from leaf hashes proves fiddly, the equivalent ground-truth check is asserting the built proof
  *verifies* via `VerifyConsistency` (independent of `tree.ConsistencyProof`) — keep at least the
  byte-equality on one boundary-crossing pair so a green-but-wrong builder cannot ship.

## Verification
- `mise run check` is green (build + vet + test, all packages).
- `gofmt -l .` is empty.
- `go test -count=1 -run TestConsistencyProofFromTiles ./internal/logclient` passes.
- The golden assertion holds: for a ~300-leaf `testonly.Tree`, `ConsistencyProofFromTiles(ctx, fetch,
  s1, s2)` byte-equals `tree.ConsistencyProof(s1, s2)` for at least one boundary-crossing `(s1<s2)`
  pair, and `proof.VerifyConsistency(rfc6962.DefaultHasher, s1, s2, got, tree.HashAt(s1),
  tree.HashAt(s2))` accepts the built proof for at least three pairs.
- `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (purity preserved).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dep churn); `go mod tidy` is a no-op.
- `go list -deps ./internal/logclient | grep -E '^net/http$|^database/sql$'` is empty (still a
  network-free pure unit).

## Done When
`ConsistencyProofFromTiles` reconstructs the `testonly.Tree` consistency proof from tiles for the
boundary-crossing pair(s) and the built proofs verify, all Verification checks pass, and `logclient`
still builds for `GOOS=js GOARCH=wasm` with `go.mod`/`go.sum` unchanged.
