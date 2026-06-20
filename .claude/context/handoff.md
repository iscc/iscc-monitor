# Handoff

## 2026-06-20 — Pure `ConsistencyProofFromTiles` builder over the tile-fetch seam (`internal/logclient`)

**Done:** Added `ConsistencyProofFromTiles(ctx, fetch TileFetcher, smaller, larger uint64) ([][]byte,
error)` — a pure RFC-6962 consistency-proof builder ported from tessera's
`client.ProofBuilder.ConsistencyProof` / `fetchNodes` / `nodeCache.GetNode` (otel spans and the
net/http fetcher dropped). It sources every proof node from mirrored hash tiles via an injected
`TileFetcher` closure whose shape matches `store.SQLiteFetcher.ReadTile` exactly, so the follower can
later pass it straight in. The result is shaped to feed `CheckEquivocation`'s `consistencyProof`
argument directly. No follower wiring (deliberately the next step).

**Files changed:**
- `internal/logclient/proofbuilder.go` (new): `TileFetcher` type, `ConsistencyProofFromTiles`,
  and the `getNode` helper (node→tile mapping via `layout.NodeCoordsToTileAddress` +
  `PartialTileSize`, tile parse via `api.HashTile.UnmarshalText`, node recompute via
  `compact.RangeFactory`). Per-call `map[tileKey]api.HashTile` cache (KISS; no full `nodeCache` port).
- `internal/logclient/proofbuilder_test.go` (new): golden test against a real ~300-leaf
  `testonly.Tree`. An in-test `TileFetcher` serializes the tree's level-0 hash tiles via
  `api.HashTile.MarshalText`; asserts the built proof byte-equals `tree.ConsistencyProof(s1,s2)` AND
  verifies via `proof.VerifyConsistency` for 5 growing pairs (incl. boundary-crossing `(5,300)` /
  `(260,300)` into the partial tile index 1), plus the empty-proof boundaries (`s1==0`, `s1==s2`,
  never touches the fetcher) and a missing-tile fault that preserves `os.ErrNotExist`.

**Verification:** `mise run check` → green, all 8 packages ok. Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 -run TestConsistencyProofFromTiles ./internal/logclient` passes (3 tests).
- [x] Golden holds: byte-equality vs `tree.ConsistencyProof` on all 5 pairs incl. boundary-crossing
  `(5,300)`; `VerifyConsistency` accepts the built proof for all 5 (≥3 required). Proof lengths are
  non-trivial (9/7/10/6/8 hashes), so the byte-match is not comparing empties.
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (purity preserved).
- [x] `git diff --quiet HEAD -- go.mod go.sum` exits 0 (go.mod AND go.sum byte-identical to HEAD).
- [x] Non-vacuousness proven: a throwaway mutation corrupting the first folded leaf in `getNode` made
  the golden FAIL (then reverted) — a green-but-wrong builder cannot ship.

**Next:** Wire `CheckEquivocation` into `follower.checkConsistency` as the third trigger. The two
seams now meet: source the proof via `ConsistencyProofFromTiles(ctx, SQLiteFetcher.ReadTile, prevSize,
nextSize)` (the `TileFetcher` signature was made identical to `SQLiteFetcher.ReadTile` for exactly
this), feed its `[][]byte` to `CheckEquivocation`, on `true` → `RecordViolation`("equivocation") +
`Freeze` + alert-once (mirroring the shrink/fork branches). That step needs real on-disk tile fixtures
under `testdata/live/` (deferred from here) so the end-to-end follower test can mirror tiles and prove
a real-tree equivocation freezes. Order against fork/shrink in `checkConsistency` and the
"compare against the prior accepted root, not the contradicting evidence" caveat (learnings) apply.

**Notes:**
- **`next.md` check "`go list -deps ./internal/logclient | grep net/http` is empty" does NOT hold —
  but it is PRE-EXISTING, not introduced here.** `net/http` is in the package closure via
  `didresolve.go` (the networked did:web resolver, which has lived in `logclient` since that slice —
  see learnings). I confirmed it by removing both new files and re-checking: `net/http` is present at
  baseline. My `proofbuilder.go` imports only `context fmt` + `merkle/{proof,compact,rfc6962}` +
  `tessera/api{,/layout}` — no net/database. The load-bearing purity invariant (WASM-shareable) is the
  **`GOOS=js GOARCH=wasm` build**, which passes. The package as a whole is not net-free (it never was
  after the resolver landed); the *new file* is. No action needed, but the `next.md` criterion as
  literally worded was already unsatisfiable for this package.
- **`go mod tidy` is NOT a no-op (adds 22 go.sum lines) — but the committed go.sum is byte-identical
  to HEAD and the build is fully reproducible under `-mod=readonly`.** `next.md` claimed adding
  `tessera/api` needs no go.sum change; that holds for `tessera/api/layout` (already imported by
  `internal/tiles`) but NOT for `tessera/api`, which has a broader module-graph require footprint.
  `go mod tidy` wants checksums for tessera's transitive *requires* (otel/klog/x-crypto/formats/
  backoff) — but those modules **never compile** (`go list -deps tessera/api` is stdlib-only; the
  `crypto/*` matches are stdlib, not `x/crypto`). I verified a clean `go clean -cache` +
  `GOFLAGS=-mod=readonly go build/test ./internal/logclient` PASSES with go.sum at HEAD. So I did NOT
  run `go mod tidy` (keeping go.sum byte-identical as `next.md` requires). **Heads-up for `review`/CI:**
  a CI step that runs `go mod tidy` then `git diff --exit-code` WOULD fail here (22 unstaged go.sum
  lines). Same class of metadata-only churn the learnings already noted for `internal/tiles`/go-cmp,
  but here it actually surfaces under tidy. If CI enforces tidy-cleanliness, those 22 go.sum entries
  should be added in a deliberate go.sum-only commit; I left them out to honor the byte-identical
  requirement. Flagging so review can decide which constraint wins.
- **Oracle gate APPLIES (RFC-6962 crypto) and is satisfied by in-test merkle ground truth.** The
  prover (`testonly.Tree.ConsistencyProof`) and the tile-sourced builder are independent code paths,
  and the proof additionally verifies via `proof.VerifyConsistency` (a third independent path), so the
  match is not a tautology — confirmed by the corrupt-leaf mutation. `notecheck`/`derive_vkey.py`/
  `fsck` are N/A (no signature/did:web/on-disk-tile path; tiles are synthesized in-test, as scoped).
- Scope: 1 production file + 1 test file (≤3 budget; no re-export delegate to `internal/tiles` was
  needed — `layout.NodeCoordsToTileAddress`/`PartialTileSize` are used directly, the latter already
  re-exported by `tiles` but the builder calls `layout` to match the port 1:1). No follower touch, no
  `testdata/live/` fixtures, no `tessera/client`/`fsck` import — all as Not-In-Scope required.
