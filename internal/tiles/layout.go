// Package tiles is the monitor's thin, golden-tested seam over the canonical
// tlog-tiles path/coordinate math (https://c2sp.org/tlog-tiles). It re-exports
// (never reimplements, per the target Stack rule) the layout primitives from
// github.com/transparency-dev/tessera/api/layout that every downstream piece —
// the SQLite mirror keys, the SQLiteFetcher, the ProofBuilder, and fsck — keys
// on, plus the one project-specific IsFull predicate (ADR-0005). The tessera
// api/layout closure is stdlib-only, so this package stays a pure leaf with no
// net/net-http/database-sql imports — shareable by the SQLite store keys and the
// WASM-bound proof/verify path, like internal/didweb.
package tiles

import "github.com/transparency-dev/tessera/api/layout"

const (
	// TileWidth is the maximum number of hashes in the bottom row of a tile —
	// fixed at 256 by the tlog-tiles spec. The tiles.width SQLite column stores
	// this as the full-tile leaf count (see IsFull).
	TileWidth = layout.TileWidth
	// TileHeight is the number of Merkle levels a tile spans — fixed at 8 by the
	// tlog-tiles spec.
	TileHeight = layout.TileHeight
)

// TilePath returns the canonical tlog-tiles path for the hash tile at the given
// level and tile-space index. A width p > 0 yields a partial-tile path
// (".p/<p>" suffix); p == 0 is the full-tile path. This is the path-API "0 ==
// full" convention — distinct from the tiles.width column convention IsFull
// owns. It delegates to tessera's layout.TilePath.
func TilePath(level, index uint64, p uint8) string {
	return layout.TilePath(level, index, p)
}

// EntriesPath returns the canonical tlog-tiles path for the nth entry bundle. A
// width p > 0 yields a partial-bundle path (".p/<p>" suffix); p == 0 is the full
// path. Same "0 == full" path-API convention as TilePath. It delegates to
// tessera's layout.EntriesPath.
func EntriesPath(index uint64, p uint8) string {
	return layout.EntriesPath(index, p)
}

// PartialTileSize returns the expected leaf count of the tile at the given level
// and index within a tree of logSize, or 0 if that tile is expected to be full.
// This is the partial qualifier the path functions take as their p argument
// (path-API "0 == full"). It delegates to tessera's layout.PartialTileSize.
func PartialTileSize(level, index, logSize uint64) uint8 {
	return layout.PartialTileSize(level, index, logSize)
}

// IsFull reports whether a tile/bundle of the given column width is a full,
// immutable tile under the ADR-0005 partial-tile discipline. This package owns
// the translation between the two encodings of "full": the path API above marks
// a full tile as width 0 (the partial suffix is dropped), while the tiles.width
// / entry_bundles.width SQLite column stores the actual leaf count — 256 for a
// full tile, anything less for a partial. IsFull adopts the column convention:
// width == TileWidth is full, any smaller width is partial. The argument is an
// int so it can hold the full sentinel 256 (which a uint8 cannot represent).
func IsFull(width int) bool {
	return width == TileWidth
}
