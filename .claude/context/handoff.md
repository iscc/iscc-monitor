# Handoff

## 2026-06-21 — Review of: Pure entry-bundle coordinate enumeration for a tree of size N (`tiles.BundleCoords`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the pure `BundleCoords(treeSize uint64) []BundleCoord` to
`internal/tiles/coords.go` (plus a table-driven golden test), naming which entry bundles a complete
mirror of a tree of size N must hold. It delegates the boundary math to tessera by iterating
`layout.Range(0, treeSize, treeSize)` and projecting `{Index, Partial}` — no I/O, no store wiring,
purely additive. Every Verification criterion passes and the goldens are independently confirmed as
tessera ground truth.

**Verification:**
- [x] `mise run check` (build + vet + test) — green (all 11 packages `ok`; `internal/tiles` re-run
      uncached → `ok`, `go vet ./...` clean).
- [x] `gofmt -l internal/tiles/` — empty. Also ran `gofmt -l .` over the whole tree → empty.
- [x] `go test -run TestBundleCoords ./internal/tiles` (uncached, `-count=1 -v`) — passes (all 7 golden
      vectors as named subtests + the non-nil-empty zero-case test).
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` — exit 0 (no dependency change; `layout.Range`
      already in the closure via `layout.go`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/tiles` — exit 0 (leaf-purity invariant holds). `didweb`
      WASM build also re-checked green (the trust-root WASM-shared seam).
- [x] Boundary assertion: `BundleCoords(300)` returns exactly `[]BundleCoord{{0, 0}, {1, 44}}` — full
      bundle at index 0, partial-44 at index 1, never partial at index 0.

**Oracle/conformance gate (N/A — correctly stated):** This slice is pure path/coordinate math
delegating to tessera; it introduces no signature / RFC-6962 / Merkle / did:web / fsck-rebuild path, so
the `notecheck` / `derive_vkey.py` / `fsck` ground-truth oracles do not apply. The gate re-arms when the
fetched bundles feed `LeafHashes` + the inclusion cross-check (the M2 ingestion writer slice).
Non-vacuousness is grounded externally, not author-asserted: I re-ran the real
`layout.Range(0, size, size)` in a throwaway module over all 7 sizes and the output matched every pinned
vector byte-for-byte — `0→[]`, `1→[{0,1}]`, `255→[{0,255}]`, `256→[{0,0}]`, `257→[{0,0},{1,1}]`,
`300→[{0,0},{1,44}]`, `513→[{0,0},{1,0},{2,1}]`. So the goldens are tessera ground truth.

**Issues found:** (none) — purely additive seam, no behavior to fix.

**Gate-integrity scan:** clean over all 3 unpushed commits (`@{upstream}..HEAD`). The added go lines are
only `coords.go` + `coords_test.go`: no `//nolint`, no `t.Skip`/`SkipNow`, no build-tag exclusions, no
swallowed errors, no deleted assertions. The `t.Fatal`/`t.Errorf` calls are genuine assertions.

**Scope discipline:** clean. 1 production file (`coords.go`) + 1 new test file + the handoff — within
≤3. No "Not In Scope" work leaked: no hash-tile (multi-level) enumeration, no I/O/fetch/`RecordEntryBundle`,
no `PollHub`/follower/store wiring (`grep` confirms `BundleCoords` has no production caller — it is an
intentional unwired seam), and the `cmd/notecheck out io.Writer` low issue was correctly left untouched.

**Next:** The hash-tile (multi-level) coordinate enumeration — the sibling this step deferred. Hash
tiles span tile-levels `0, 1, 2, …` (tree-levels `0, 8, 16, …`); unlike entry bundles there is no single
`Range` call covering all levels, so it needs its own loop over levels (per-level tile counts + the top
partial via `layout.PartialTileSize(level, index, treeSize)`) until a level has one root tile. After both
enumerations exist, the **live tile-ingestion writer** (fetch loop + `PollHub` wiring calling
`RecordTile` / `RecordEntryBundle`) becomes implementable — which in turn unblocks the still-deferred
`fsck` root-rebuild over `SQLiteFetcher` and the inclusion cross-check vs the hub's real
`IsccLogInclusionProof` (both want real tile/bundle fixtures in `testdata/live/`).

**Notes:**
- `BundleCoords` is an intentional unused-until-wired export seam (like `IsFull`, the consistency
  triggers, `LeafHashes`, `RunFsck`, the proof builders) — `go vet` clean, not dead code.
- The `Range(0,N,N)` projection ignores `ri.First`/`ri.N` correctly: that call IS the complete whole-tree
  cover, and `layout.Range` only sets `Partial` for the start/end indices — intermediate full bundles keep
  the zero=full value (verified via `513 → {1,0}` middle bundle).
- Two-convention discipline preserved: `BundleCoord.Partial` is the **path** qualifier (path-API
  "0 == full"); the store's `width` column is `widthForP(Partial)` (`0 → 256`), translated downstream by
  the ingestion writer — not this step's concern. Doc comments call this out.
- `internal/tiles` closure stays `api/layout`-only (verified `go list -deps` has no `net`/`net/http`/
  `database/sql`); pure leaf, WASM-green. `go.mod`/`go.sum` byte-identical.
- (low, still open, untouched) `cmd/notecheck`'s `run` has a vestigial `out io.Writer` param — correctly
  loop-skipped (low priority, skipped by the loop).
- Branch is `develop` with upstream `origin/develop` (ahead 3); remote configured → pushing on PASS.
