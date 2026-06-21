# Next Work Package

## Step: Pure hash-tile (multi-level) coordinate enumeration for a tree of size N (`tiles.TileCoords`)

## Goal
Add the pure, boundary-exact enumeration naming which **hash tiles** (across every tile-level) a
complete mirror of a tree of size N must hold — the sibling of `BundleCoords`. Together they are the
two coordinate sources the M2 live tile-ingestion writer needs to know what to fetch; this is the last
missing pure prerequisite before that writer (and thus `fsck` / the inclusion cross-check) becomes
implementable.

## Scope
- **Create**: `internal/tiles/tilecoords_test.go` (golden table test)
- **Modify**: `internal/tiles/coords.go` — add `TileCoord{Level, Index uint64; Partial uint8}` and
  `TileCoords(treeSize uint64) []TileCoord` (1 non-test file)
- **Reference**:
  - `/workspace/iscc-monitor/internal/tiles/coords.go` + `/workspace/iscc-monitor/internal/tiles/layout.go`
    — existing seam style + the `PartialTileSize`/`TilePath` wrappers and the two-convention
    (path-`p` vs store-`width`) doc rules
  - `/workspace/iscc-monitor/internal/tiles/coords_test.go` — golden-test style to mirror (table +
    non-nil-empty zero case)
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/api/layout/tile.go` —
    `PartialTileSize(level, index, logSize)` (`sizeAtLevel = logSize >> (level*8)`,
    `fullTiles = sizeAtLevel/256`), `TileHeight = 8`, `TileWidth = 256` — the boundary oracle to delegate to
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/api/layout/paths.go` — `TilePath`
    (already re-exported as `tiles.TilePath`)
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/fsck/fsck.go` (lines ~218–284) —
    confirms tessera derives tiles by *replaying entries* (`visit`), so there is **no** single upstream
    helper to port; the per-level loop below is the correct, minimal construction

## Not In Scope
- The **live tile-ingestion writer** (`PollHub` fetch loop calling `RecordTile`/`RecordEntryBundle`) —
  that is the next slice and consumes both `TileCoords` and `BundleCoords`. Do not wire any caller.
- Any I/O, store, fetcher, or follower change. `TileCoords` is a pure unwired export seam, exactly like
  `BundleCoords` / `IsFull` / the consistency triggers (`go vet` clean, not dead code).
- The `fsck` root-rebuild, the inclusion cross-check, real tile/bundle fixtures in `testdata/live/`, or
  the `iscc_index` writer — all downstream of the ingestion writer.
- The open `low` issue (`cmd/notecheck`'s vestigial `out io.Writer`) — loop-skipped.

## Implementation Notes
- **Construction (no single `layout.Range` covers all levels — climb tile-levels yourself):** loop
  `level := uint64(0), 1, 2, …`. At each level compute
  `sizeAtLevel := treeSize >> (level * tiles.TileHeight)`. When `sizeAtLevel == 0`, stop. Otherwise emit
  `fullTiles := sizeAtLevel / tiles.TileWidth` full tiles (`Index 0 .. fullTiles-1`, `Partial 0`)
  followed by, **iff** `sizeAtLevel % tiles.TileWidth != 0`, one partial tile at `Index == fullTiles`
  with `Partial == uint8(sizeAtLevel % tiles.TileWidth)`. After emitting the level, **stop once
  `sizeAtLevel <= tiles.TileWidth`** (that level has collapsed to a single root tile; there is no level
  above it). This stop-after-emit ordering is load-bearing: it correctly emits the lone root tile of an
  exact-power-of-256 tree (e.g. `256 → {0,0,0}` then stop, `65536 → …256 full…, {1,0,0}` then stop)
  and never adds a spurious empty level above the root.
- **Delegate the per-tile `Partial` to tessera, don't hand-roll it twice:** derive each tile's `Partial`
  via `tiles.PartialTileSize(level, index, treeSize)` (which already wraps `layout.PartialTileSize`)
  rather than recomputing `% TileWidth` for the emitted value. Use the inline `fullTiles` /
  `sizeAtLevel % TileWidth` arithmetic only to drive the loop bounds and the stop condition; `Partial`
  on the emitted `TileCoord` comes from `PartialTileSize`. That keeps tessera the single source of truth
  for the "0 == full" qualifier and matches how `coords.go` already leans on `layout` (Correctness rule:
  re-use the transparency stack, never reimplement it). Use `tiles.TileWidth` / `tiles.TileHeight`
  constants, not bare `256` / `8`.
- **Two-convention discipline (carry the `BundleCoord` doc pattern verbatim in spirit):**
  `TileCoord.Partial` is the **path-API** qualifier (`0 == full`, the same `p` arg `tiles.TilePath`
  takes) — NOT the store's `width` column (`widthForP`: `0 → 256`). State this in the doc comment exactly
  as `BundleCoord` does; the width translation belongs to the downstream ingestion writer, not here.
- **Zero case:** `TileCoords(0)` must return a **non-nil, length-0** slice (`make([]TileCoord, 0)`),
  mirroring `BundleCoords(0)`'s documented contract.
- Keep the file a pure leaf: the only import stays `github.com/transparency-dev/tessera/api/layout`
  (already present in `coords.go`); add **no** new dep, so `go.mod`/`go.sum` stay byte-identical
  (`PartialTileSize` is already in the closure via `layout.go`).
- **Oracle/conformance gate is correctly N/A** for this slice (pure path/coordinate math; no
  signature / RFC-6962 / Merkle / did:web / `fsck`-rebuild path introduced). It re-arms when the fetched
  tiles feed the `fsck` root-rebuild. Ground the goldens externally instead: re-run
  `layout.PartialTileSize(level, index, size)` per vector to confirm each `Partial`, exactly as the
  `BundleCoords` review re-ran `layout.Range`.

## Verification
- `mise run check` is green (build + vet + test; `go vet ./...` clean, all packages `ok`).
- `gofmt -l internal/tiles/` is empty.
- `go test -run TestTileCoords ./internal/tiles` passes (uncached `-count=1`).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change).
- `GOOS=js GOARCH=wasm go build ./internal/tiles` exits 0 (leaf-purity invariant holds).
- Golden assertions (ground truth from the per-level construction above; `{Level, Index, Partial}`):
  - `TileCoords(0)` is non-nil and length 0.
  - `TileCoords(1) == [{0,0,1}]`
  - `TileCoords(255) == [{0,0,255}]`
  - `TileCoords(256) == [{0,0,0}]`
  - `TileCoords(257) == [{0,0,0},{0,1,1},{1,0,1}]`
  - `TileCoords(300) == [{0,0,0},{0,1,44},{1,0,1}]`
  - `TileCoords(513) == [{0,0,0},{0,1,0},{0,2,1},{1,0,2}]`
  - `TileCoords(65536)` ends with the root tile `{1,0,0}` and has exactly 257 entries (256 full level-0
    tiles `{0,0,0} .. {0,255,0}` + one full level-1 root tile).
  - `TileCoords(65537)` contains `{0,256,1}` (the lone extra leaf as a partial-1 at level-0 index 256)
    and ends with `{1,0,0}`.

## Done When
`advance` adds `TileCoord` + `TileCoords` to `internal/tiles/coords.go` with the golden test, and every
Verification criterion above passes.
