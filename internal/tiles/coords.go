// This file adds the pure entry-bundle coordinate enumeration for a complete
// mirror of a tree of size N — the (index, partial) list naming which entry
// bundles the mirror must hold. It is the foundational, boundary-exact slice the
// M2 live tile-ingestion writer builds on: the fetch loop and the PollHub wiring
// both consume BundleCoords to know what to fetch and how to width-key each
// RecordEntryBundle write. Like the rest of internal/tiles it delegates the
// boundary math to tessera's api/layout (never reimplements it), so the package
// stays a pure leaf with a stdlib-only closure.
package tiles

import "github.com/transparency-dev/tessera/api/layout"

// BundleCoord names one entry bundle a complete mirror of a tree must hold. Index
// is the bundle's position in tile space; Partial is the path-API "0 == full"
// qualifier — the same p argument EntriesPath takes (0 for a full 256-leaf
// bundle, else the partial leaf count). The store's width column is a different
// encoding (widthForP(Partial): 0 -> 256), translated downstream, not here.
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
