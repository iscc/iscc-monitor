# Next Work Package

## Step: Pure entry-bundle coordinate enumeration for a tree of size N (`tiles.BundleCoords`)

## Goal
Add the pure, golden-testable function that says *which* entry bundles a complete
mirror of a tree of size N must hold — the `(index, partial)` coordinate list. This is
the first, foundational slice of the M2 live tile-ingestion writer: the fetch loop and
the `PollHub` wiring both build on it, and it must be exact at the 256-leaf boundary.
Doing the pure enumeration first (before any I/O or store wiring) keeps the riskiest
part — the boundary math — isolated and oracle-checkable against tessera.

## Scope
- **Create**:
  - `internal/tiles/coords.go` — the new `BundleCoord` type + `BundleCoords(treeSize uint64) []BundleCoord`.
  - `internal/tiles/coords_test.go` — table-driven golden test (test file, not counted).
- **Modify**: (none — purely additive)
- **Reference**:
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/api/layout/paths.go`
    (`Range` at line 49, `RangeInfo` at line 99, `EntriesPath` at line 121) — the oracle to delegate to.
  - `/workspace/iscc-monitor/internal/tiles/layout.go` — the existing seam; match its delegation + doc style.
  - `/workspace/iscc-monitor/internal/tiles/layout_test.go` — match its table-driven golden-vector test style.
  - `/workspace/iscc-monitor/internal/store/tiles.go` — confirms the downstream
    `RecordEntryBundle(hubID, bundleIndex, width, …)` signature this list will eventually feed (width is
    the leaf count; `p==0` ⇒ 256, via `widthForP` in `internal/store/fetcher.go`).

## Not In Scope
- **No hash-tile (multi-level) enumeration.** Hash tiles span tile-levels `0, 1, 2, …` (tree-levels
  `0, 8, 16, …`) and need their own boundary reasoning — that is the *next* slice. This step is entry
  bundles only.
- **No I/O, no fetch, no store writes.** Do not add a tile/bundle *fetch* primitive, do not call
  `RecordEntryBundle`, do not touch `PollHub`/`follower`. This function is pure.
- **No `PollHub`/follower wiring** — the live ingestion writer that calls this is a later slice.
- Do not touch the open `low` issue (`cmd/notecheck` vestigial `out`).

## Implementation Notes
- **Delegate to tessera, never reimplement the boundary math** (target Stack rule, and the
  `internal/tiles` package is explicitly "a thin re-export … not a reimplementation"). Implement
  `BundleCoords` by iterating `layout.Range(0, treeSize, treeSize)` and projecting each `RangeInfo` to a
  `BundleCoord{Index: ri.Index, Partial: ri.Partial}`. Ignore `ri.First`/`ri.N` (those describe
  sub-ranges; for a full-tree mirror every bundle is taken whole — `Range(0, treeSize, treeSize)`
  already yields the complete cover, confirmed by running it in-module).
- `BundleCoord` carries `Index uint64` and `Partial uint8` (the path-API "0 == full" qualifier — the
  same `p` `EntriesPath` takes). Keep the existing two-convention discipline: `Partial` is the *path*
  qualifier here; the store's `width` column is `widthForP(Partial)` later, NOT this step's concern.
- `layout.Range` returns a Go 1.23 range-over-func iterator (`iter.Seq[layout.RangeInfo]`); consume it
  with `for ri := range layout.Range(0, treeSize, treeSize) { … }`. The 1.24 toolchain supports this.
- `treeSize == 0` must yield an **empty (len 0) slice** — `layout.Range` yields nothing, so a pre-sized
  `make([]BundleCoord, 0, …)` accumulator returns empty cleanly. Return a non-nil empty slice for the
  zero case; the test asserts `len == 0` (and reads fine for a nil slice too, but prefer non-nil).
- Keep the package a pure leaf: import only `github.com/transparency-dev/tessera/api/layout` (already in
  the closure via `layout.go`). Do **not** pull in `net`/`database/sql`. `go.mod`/`go.sum` must stay
  byte-identical (no new dependency — `layout.Range` is already compiled in).
- Correctness rule (learnings, "Partial-tile discipline" + the pinned `PartialTileSize(0,0,300)==0` /
  `(0,1,300)==44`): the boundary is the bug surface. The first 256-leaf bundle of a 300-leaf tree is
  **full** (`Partial==0`, index 0); the leftover 44 are a **partial** at **index 1** (`Partial==44`) —
  never index 0.
- Oracle/conformance gate is **N/A** for this slice: it is pure path/coordinate math delegating to
  tessera (no signature / RFC-6962 / Merkle / did:web / fsck-rebuild path). It re-arms when the fetched
  bundles feed `LeafHashes` + the inclusion cross-check. State this in the handoff rather than inventing
  a crypto check.

## Golden vectors (ground truth — pinned by running `layout.Range(0, size, size)` in-module)
- `BundleCoords(0)` → `[]` (len 0)
- `BundleCoords(1)` → `[{0, 1}]`
- `BundleCoords(255)` → `[{0, 255}]`
- `BundleCoords(256)` → `[{0, 0}]`
- `BundleCoords(257)` → `[{0, 0}, {1, 1}]`
- `BundleCoords(300)` → `[{0, 0}, {1, 44}]`   ← the boundary case
- `BundleCoords(513)` → `[{0, 0}, {1, 0}, {2, 1}]`

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass).
- `gofmt -l internal/tiles/` is empty.
- `go test -run TestBundleCoords ./internal/tiles` passes, asserting every golden vector above
  (length + each `{Index, Partial}` in order).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change).
- `GOOS=js GOARCH=wasm go build ./internal/tiles` exits 0 (the leaf-purity invariant holds).
- Assertion: `BundleCoords(300)` returns exactly `[]BundleCoord{{0, 0}, {1, 44}}` — full bundle at
  index 0, partial-44 at index 1, never partial at index 0.

## Done When
`internal/tiles/coords.go` provides `BundleCoords` returning the exact golden entry-bundle coordinate
lists above, and all Verification criteria pass with the tree clean and `go.mod`/`go.sum` byte-unchanged.
