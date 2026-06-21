## 2026-06-21 — Pure hash-tile (multi-level) coordinate enumeration for a tree of size N (`tiles.TileCoords`)

**Done:** Added the pure `TileCoord{Level, Index uint64; Partial uint8}` + `TileCoords(treeSize uint64)
[]TileCoord` to `internal/tiles/coords.go`, the multi-level sibling of `BundleCoords`. It climbs the
tile-levels itself (no single `layout.Range` covers all levels), emitting per-level full tiles + one
top partial and stopping once a level has collapsed to a single root tile; each emitted `Partial` is
taken from tessera's `PartialTileSize` oracle, not hand-rolled. Purely additive, unwired — `go.mod`/
`go.sum` byte-identical.

**Files changed:**
- `internal/tiles/coords.go`: added `TileCoord` + `TileCoords` (per-level loop delegating the partial
  qualifier to `PartialTileSize`); refreshed the file doc comment to cover both enumerations.
- `internal/tiles/tilecoords_test.go` (new): table-driven golden test + power-of-256 root-collapse
  cases + non-nil-empty zero case + an oracle-cross-check test that re-grounds every emitted `Partial`
  against `layout.PartialTileSize` directly.

**Verification:** `mise run check` (build + vet + test) → green (all 11 packages `ok`; `internal/tiles`
re-run uncached → `ok`, `go vet ./...` clean). Per-criterion:
- [x] `gofmt -l internal/tiles/` — empty.
- [x] `go test -run TestTileCoords -count=1 ./internal/tiles` — passes (all 7 table vectors + the two
      power-of-256 boundary tests + zero-case + oracle cross-check).
- [x] `git diff --quiet HEAD -- go.mod go.sum` — exit 0 (no dependency change; `PartialTileSize`
      already in the closure via `layout.go`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/tiles` — exit 0 (leaf-purity invariant holds).
- [x] Golden assertions all hold: `TileCoords(0)` non-nil len 0; `1→[{0,0,1}]`; `255→[{0,0,255}]`;
      `256→[{0,0,0}]`; `257→[{0,0,0},{0,1,1},{1,0,1}]`; `300→[{0,0,0},{0,1,44},{1,0,1}]`;
      `513→[{0,0,0},{0,1,0},{0,2,1},{1,0,2}]`; `65536` ends `{1,0,0}` with exactly 257 entries; `65537`
      contains `{0,256,1}` and ends `{1,0,0}`.

**Oracle/conformance gate (N/A — correctly stated in `next.md`):** This slice is pure path/coordinate
math delegating to tessera; it introduces no signature / RFC-6962 / Merkle / did:web / `fsck`-rebuild
path, so the `notecheck` / `derive_vkey.py` / `fsck` ground-truth oracles do not apply. The gate
re-arms when the fetched tiles feed the `fsck` root-rebuild + inclusion cross-check (the M2 ingestion
writer slice). Non-vacuousness is grounded externally, not author-asserted: I re-ran the real
`layout.PartialTileSize(level, index, size)` per-tile in a throwaway module over all 9 sizes and the
reconstructed coordinate lists matched every pinned vector byte-for-byte — including
`65537 → last {1,0,0}` (a full root, not a partial). So the goldens are tessera ground truth.

**Next:** Both pure coordinate sources now exist (`BundleCoords` + `TileCoords`), so the **live
tile-ingestion writer** is implementable: a `PollHub` fetch loop that, on a verified growing
checkpoint, walks `TileCoords(size)` + `BundleCoords(size)`, fetches each tile/bundle over the
`logclient` `Fetcher` seam at its `TilePath`/`EntriesPath`, and writes via `RecordTile` /
`RecordEntryBundle` (translating the path-API `Partial` to the store's `width` via `widthForP`,
re-fetching partials every poll per ADR-0005). That writer in turn unblocks the still-deferred `fsck`
root-rebuild over `SQLiteFetcher` and the inclusion cross-check vs the hub's real
`IsccLogInclusionProof` — both want real tile/bundle fixtures in `testdata/live/`.

**Notes:**
- **Two-convention discipline preserved (matches `BundleCoord`):** `TileCoord.Partial` is the **path**
  qualifier (path-API "0 == full", the same `p` arg `TilePath` takes), NOT the store's `width` column
  (`widthForP`: `0 → 256`). The doc comment calls this out; the width translation belongs to the
  downstream ingestion writer.
- **The `Partial` value comes from `PartialTileSize`, never recomputed.** The inline `fullTiles` /
  `sizeAtLevel % TileWidth` arithmetic only drives the loop bounds and the stop condition; the emitted
  qualifier delegates to tessera so it stays the single source of truth for "0 == full". Uses the
  `TileWidth`/`TileHeight` constants, never bare `256`/`8`.
- **The stop-after-emit ordering is load-bearing and correct:** stopping once `sizeAtLevel <= TileWidth`
  *after* emitting the level emits the lone root tile of an exact-power-of-256 tree (`256→{0,0,0}`;
  `65536→…256 full level-0…,{1,0,0}`) and never adds a spurious empty level above the root.
- **`TileCoords` is an intentional unused-until-wired export seam** (like `BundleCoords`, `IsFull`, the
  consistency triggers, `LeafHashes`, `RunFsck`, the proof builders) — `go vet` clean, not dead code.
  No production caller; its first caller is the M2 live tile-ingestion fetch loop. Confirmed no caller
  wired (purely additive).
- **`next.md`'s 65537 verification text says only "ends with `{1,0,0}`"** — my first test draft wrongly
  asserted a partial-1 root `{1,0,1}`; the tessera oracle confirms the level-1 root of a 65537-leaf tree
  is a *full* 256-hash tile (the extra leaf splits only the level-0 row). Fixed to `{1,0,0}`; the
  oracle cross-check test guards against any such hand-derivation drift going forward.
- `internal/tiles` closure stays `api/layout`-only (pure leaf, WASM-green); `go.mod`/`go.sum`
  byte-identical (`PartialTileSize` already in the closure via `layout.go`). No new import.
- Scope: 1 production file (`coords.go`) + 1 new test file + the handoff — within ≤3. No "Not In Scope"
  work leaked: no ingestion writer / `PollHub` wiring / I/O / store-fetcher change, and the open `low`
  issue (`cmd/notecheck`'s vestigial `out io.Writer`) was correctly left untouched.
- Branch is `develop`.
