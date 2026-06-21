# Handoff

## 2026-06-21 — Pure entry-bundle coordinate enumeration for a tree of size N (`tiles.BundleCoords`)

**Done:** Added the pure, golden-testable `BundleCoords(treeSize uint64) []BundleCoord` to
`internal/tiles/coords.go` — the (index, partial) list naming which entry bundles a complete mirror of
a tree of size N must hold. It delegates the boundary math to tessera by iterating
`layout.Range(0, treeSize, treeSize)` and projecting each `RangeInfo` to `BundleCoord{Index, Partial}`.
No I/O, no store wiring, no hash-tile enumeration — purely additive and dependency-clean.

**Files changed:**
- `internal/tiles/coords.go` (new): `BundleCoord` type (`Index uint64`, `Partial uint8`) + `BundleCoords`.
- `internal/tiles/coords_test.go` (new): table-driven golden test asserting all 7 `next.md` vectors
  (length + each `{Index, Partial}` in order) plus a non-nil-empty-slice assertion for `treeSize == 0`.

**Verification:** `mise run check` (build + vet + test) → green (all 11 packages `ok`; `internal/tiles`
ran uncached `ok`). Per-criterion:
- [x] `gofmt -l internal/tiles/` → empty (after `mise run fmt`).
- [x] `go test -run TestBundleCoords ./internal/tiles` → passes (all 7 golden vectors).
- [x] `git diff --quiet HEAD -- go.mod go.sum` → exit 0 (no dependency change; `layout.Range` already
      compiled in via `layout.go`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/tiles` → exit 0 (leaf-purity invariant holds).
- [x] Boundary assertion: `BundleCoords(300)` returns exactly `[]BundleCoord{{0, 0}, {1, 44}}` — full
      bundle at index 0, partial-44 at index 1, never partial at index 0.

**Oracle/conformance gate (N/A — stated per `next.md`):** This slice is pure path/coordinate math
delegating to tessera; it introduces no signature / RFC-6962 / Merkle / did:web / fsck-rebuild path, so
the `notecheck` / `derive_vkey.py` / `fsck` ground-truth oracles do not apply. The gate **re-arms** when
the fetched bundles feed `LeafHashes` + the inclusion cross-check (the M2 ingestion writer slice).
Non-vacuousness is grounded externally: before writing, I ran `layout.Range(0, size, size)` in-module
over all 7 sizes (throwaway test, removed) and the output matched every pinned vector exactly, including
the `300 -> {0,0} {1,44}` boundary — so the golden is tessera ground truth, not author-asserted.

**Next:** The natural next slice is the **hash-tile (multi-level) coordinate enumeration** — the sibling
this step explicitly deferred. Hash tiles span tile-levels `0, 1, 2, …` (tree-levels `0, 8, 16, …`) and
need their own boundary reasoning (per-level tile counts + the top partial tile via
`layout.PartialTileSize(level, index, treeSize)`); unlike entry bundles there is no single `Range` call
covering all levels, so it needs its own loop over levels until the level has one (root) tile. After
both enumerations exist, the **live tile-ingestion writer** (fetch loop + `PollHub` wiring calling
`RecordTile` / `RecordEntryBundle`) becomes implementable, which in turn unblocks the still-deferred
`fsck` root-rebuild over `SQLiteFetcher` and the inclusion cross-check vs the hub's real
`IsccLogInclusionProof` (both want the same real tile/bundle fixtures in `testdata/live/`).

**Notes:**
- `BundleCoords` is an intentional unused-until-wired export seam (like `IsFull`, the consistency
  triggers, `LeafHashes`, `RunFsck`, the proof builders) — `go vet` clean, not dead code. Its first
  caller is the M2 live tile-ingestion fetch loop.
- The two-convention discipline is preserved: `BundleCoord.Partial` is the **path** qualifier
  (path-API "0 == full", the same `p` `EntriesPath`/`Range` use). The store's `width` column is a
  different encoding (`widthForP(Partial)`: `0 -> 256`), translated downstream by the ingestion writer
  via `RecordEntryBundle(hubID, bundleIndex, width, …)` — NOT this step's concern. The doc comments on
  both the type field and the function call this out so the eventual writer does not conflate them.
- `treeSize == 0` returns a non-nil length-0 slice (pre-sized `make([]BundleCoord, 0)` accumulator;
  `layout.Range` yields nothing). A dedicated test pins the non-nil contract.
- Used `reflect.DeepEqual` for the full-slice golden compare (plus an explicit length check first for a
  clearer failure message) — `BundleCoord` is a comparable value type, so this is exact.
- No `go.mod`/`go.sum` churn and no new import beyond the `layout` already in the package closure via
  `layout.go`, as required.
