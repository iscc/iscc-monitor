<!-- area: internal/tiles (layout re-export, coords.go) -->
<!-- indexed-as: tiles.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/tiles` — tlog-tiles layout seam

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## tlog-tiles layout seam (`internal/tiles`)

- **`internal/tiles` is a thin re-export of `tessera/api/layout`, not a reimplementation** — wrappers
  `TilePath`/`EntriesPath`/`PartialTileSize` delegate 1:1 (matching arg order: `TilePath(level, index
  uint64, p uint8)`, `EntriesPath(n uint64, p uint8)`), plus `const TileWidth/TileHeight = layout.*`
  and the one project predicate `IsFull(width int) bool == width == TileWidth`. The package's compiled
  closure is `api/layout` only, and `api/layout`'s own closure is **stdlib-only** — so `net/http`/
  `database/sql` stay out and the WASM build is green (verified). `IsFull` takes `int` (not `uint8`)
  because the column-convention full sentinel 256 cannot fit in `uint8`; `next.md` left the choice open
  and `int` is the clean pick.
- **`next.md`'s `PartialTileSize(0,0,300)==44` golden was WRONG — advance correctly pinned `0`.** Per
  tessera `tile.go`: `sizeAtLevel=300`, `fullTiles=300/256=1`, `index 0 < fullTiles` → **0** (the first
  256-leaf tile is *full*); the leftover 44 spill into index **1**. Reviewer re-ran the real
  `layout.PartialTileSize` independently: `(0,0,300)=0`, `(0,1,300)=44`, `(0,0,44)=44`, `(0,0,256)=0`.
  Downstream load-bearing: the SQLiteFetcher slice must address the 44-leaf partial of a 300-leaf tree
  at **index 1**, never index 0. All four `TilePath`/`EntriesPath` golden strings are verbatim from
  tessera's `api/layout/paths_test.go` (ground truth, not author-asserted).
- **tessera v1.0.2 is a clean dep on the 1.24 toolchain.** Its go directive is `go 1.24.0`; the heavy
  otel/klog/formats/`x/crypto`/backoff deps land in `go.sum` as module-graph requirements only (never
  compiled, so absent from `go.mod`'s indirect block and the `internal/tiles` closure) — same pattern as
  go-cmp for merkle. `go mod tidy` is a verified no-op, `go mod verify` passes, directive stays `go
  1.24.0` with no `toolchain` line. The tessera require graph bumped `x/sys 0.37→0.41` (benign, pure-Go).
- **`IsFull`/`internal/tiles` are an intentional unused-until-wired export seam** (like the consistency
  triggers) — `go vet` clean, not dead code; the SQLiteFetcher store-key slice is its first caller. The
  oracle/conformance gate is correctly N/A here (pure path strings, no signature/RFC-6962/did:web/fsck
  path); it re-arms at the SQLiteFetcher + `fsck` slice.
- **`BundleCoords(treeSize)` enumerates the entry bundles a full mirror must hold by iterating
  `layout.Range(0, treeSize, treeSize)` and projecting `{Index, Partial}`** (`coords.go`). `Range(0,N,N)`
  IS the complete whole-tree cover, so `ri.First`/`ri.N` (sub-range fields) are correctly ignored.
  Reviewer re-ran the real `layout.Range(0,size,size)` in a throwaway module over all 7 vectors and it
  matched byte-for-byte — including the boundary `300 → [{0,0},{1,44}]` (first 256 full at index 0, 44
  partial at index 1) and `513 → [{0,0},{1,0},{2,1}]` (middle bundle index 1 is full: `Range` only sets
  `Partial` for the start/end indices, leaving intermediate bundles at the zero=full value). So the
  goldens are tessera ground truth, not author-asserted. Closure stays `api/layout`-only (pure leaf,
  WASM-green); go.mod/go.sum byte-identical. Unwired seam — its first caller is the M2 live
  tile-ingestion fetch loop. **Sibling still deferred:** hash-tile (multi-level) enumeration needs its
  own per-level loop (no single `Range` call covers all levels), via `PartialTileSize(level,index,size)`.
- **`TileCoords(treeSize)` is the multi-level hash-tile sibling — both pure coordinate sources now
  exist** (`coords.go`). It climbs tile-levels itself (`sizeAtLevel = treeSize >> (level*TileHeight)`),
  emits `fullTiles` then one partial per level, and takes each `Partial` from `PartialTileSize` (never
  recomputed). The **stop-after-emit** ordering (`break` *after* emitting once `sizeAtLevel <=
  TileWidth`) is load-bearing: it emits the lone root tile of an exact-power-of-256 tree
  (`256→{0,0,0}`; `65536→…256 full…,{1,0,0}`) and adds no spurious empty level above the root. Reviewer
  re-derived all 9 vectors against the tessera oracle in a throwaway module (matched byte-for-byte,
  incl. `65537` ending `{1,0,0}` — a *full* root, since the extra leaf splits only the level-0 row) and
  mutation-proved the golden non-vacuous two ways (reverted): (1) `Partial` from `index+1` → golden +
  oracle-cross-check FAIL; (2) drop the stop-after-emit → all three boundary tests FAIL. Oracle gate
  correctly N/A (pure coordinate math, no signature/RFC-6962/Merkle/did:web/fsck path);
  go.mod/go.sum byte-identical, WASM-green, unwired seam. Its first caller is the M2 tile-ingestion writer.
