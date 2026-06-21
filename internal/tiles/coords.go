// This file adds the pure coordinate enumerations for a complete mirror of a
// tree of size N: BundleCoords names which entry bundles the mirror must hold,
// and TileCoords names which hash tiles (across every tile-level) it must hold.
// Together they are the two coordinate sources the M2 live tile-ingestion writer
// builds on: the fetch loop and the PollHub wiring consume both to know what to
// fetch and how to width-key each RecordEntryBundle / RecordTile write. Like the
// rest of internal/tiles they delegate the boundary math to tessera's api/layout
// (never reimplement it), so the package stays a pure leaf with a stdlib-only
// closure.
package tiles

import "github.com/transparency-dev/tessera/api/layout"

// BundleCoord names one entry bundle a complete mirror of a tree must hold. Index
// is the bundle's position in tile space; Partial is the path-API "0 == full"
// qualifier — the same p argument EntriesPath takes (0 for a full 256-leaf
// bundle, else the partial leaf count). The store's width column is a different
// encoding (0 -> 256); the store translates Partial to that width on write, not here.
type BundleCoord struct {
	// Index is the entry bundle's index in tile space.
	Index uint64
	// Partial is the path-API partial qualifier: 0 for a full (256-leaf) bundle,
	// otherwise the bundle's leaf count.
	Partial uint8
}

// BundleCoords returns, in ascending index order, the coordinates of every entry
// bundle a complete mirror of a tree of size treeSize must hold. It delegates the
// boundary math to tessera by iterating layout.Range(0, treeSize, treeSize) — the
// complete cover of a full tree — and projecting each RangeInfo to a BundleCoord;
// the sub-range fields (First/N) are irrelevant for a whole-tree mirror. The first
// 256-leaf bundle of a non-multiple-of-256 tree is full (Partial 0, index 0) and
// the leftover leaves form a partial at the next index, never a partial at index
// 0. treeSize == 0 yields a non-nil empty slice (layout.Range yields nothing).
func BundleCoords(treeSize uint64) []BundleCoord {
	out := make([]BundleCoord, 0)
	for ri := range layout.Range(0, treeSize, treeSize) {
		out = append(out, BundleCoord{Index: ri.Index, Partial: ri.Partial})
	}
	return out
}

// TileCoord names one hash tile a complete mirror of a tree must hold. Level is
// the tile-level (0, 1, 2, … spanning tree-levels 0, 8, 16, …); Index is the
// tile's position within that level. Partial is the path-API "0 == full"
// qualifier — the same p argument TilePath takes (0 for a full 256-hash tile,
// else the partial hash count). The store's width column is a different encoding
// (0 -> 256); the store translates Partial to that width on write, not here.
type TileCoord struct {
	// Level is the tile-level (0, 1, 2, …); tile-level L spans tree-level L*8.
	Level uint64
	// Index is the tile's index within its level in tile space.
	Index uint64
	// Partial is the path-API partial qualifier: 0 for a full (256-hash) tile,
	// otherwise the tile's hash count.
	Partial uint8
}

// TileCoords returns, in ascending (level, index) order, the coordinates of
// every hash tile a complete mirror of a tree of size treeSize must hold —
// across every tile-level. Unlike entry bundles there is no single layout.Range
// covering all levels, so it climbs the tile-levels itself: at each level the
// number of hashes is sizeAtLevel = treeSize >> (level * TileHeight); the level
// holds sizeAtLevel/TileWidth full tiles followed, iff sizeAtLevel is not a
// multiple of TileWidth, by one partial tile. Each emitted tile's Partial is
// taken from PartialTileSize (tessera's "0 == full" oracle), not recomputed.
// Enumeration stops once a level has collapsed to a single root tile
// (sizeAtLevel <= TileWidth), so an exact-power-of-256 tree emits its lone root
// tile and no spurious empty level above it. treeSize == 0 yields a non-nil
// empty slice.
func TileCoords(treeSize uint64) []TileCoord {
	out := make([]TileCoord, 0)
	for level := uint64(0); ; level++ {
		sizeAtLevel := treeSize >> (level * TileHeight)
		if sizeAtLevel == 0 {
			break
		}
		fullTiles := sizeAtLevel / TileWidth
		for index := uint64(0); index < fullTiles; index++ {
			out = append(out, TileCoord{Level: level, Index: index, Partial: PartialTileSize(level, index, treeSize)})
		}
		if sizeAtLevel%TileWidth != 0 {
			out = append(out, TileCoord{Level: level, Index: fullTiles, Partial: PartialTileSize(level, fullTiles, treeSize)})
		}
		if sizeAtLevel <= TileWidth {
			break
		}
	}
	return out
}
