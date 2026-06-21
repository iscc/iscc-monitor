// Golden tests pinning the hash-tile (multi-level) coordinate enumeration.
// Because TileCoords climbs the tile-levels itself (no single layout.Range covers
// every level) but takes each tile's Partial from tessera's layout.PartialTileSize
// oracle, these vectors prove the per-level construction and lock the boundary
// math so a future refactor cannot silently diverge. The 256-leaf tile boundary
// (257 -> full level-0 tile, partial-1 level-0 tile, partial-1 level-1 root) and
// the exact-power-of-256 root collapse (256 -> lone {0,0,0}; 65536 -> 256 full
// level-0 tiles + lone {1,0,0} root) are the load-bearing cases.
package tiles

import (
	"reflect"
	"testing"

	"github.com/transparency-dev/tessera/api/layout"
)

func TestTileCoords(t *testing.T) {
	tests := []struct {
		name     string
		treeSize uint64
		want     []TileCoord
	}{
		{"empty tree yields no tiles", 0, []TileCoord{}},
		{"single leaf is a partial-1 root tile at level 0", 1, []TileCoord{{0, 0, 1}}},
		{"255 leaves is one partial-255 tile at level 0", 255, []TileCoord{{0, 0, 255}}},
		{"exactly 256 leaves is one full root tile at level 0", 256, []TileCoord{{0, 0, 0}}},
		// The 256-leaf boundary: the first level-0 tile is full (partial 0), the
		// leftover leaf is a partial-1 at level-0 index 1, and the level-1 root tile
		// holds the single level-1 node as a partial-1.
		{"257 leaves spills into a partial level-0 tile and a partial level-1 root", 257, []TileCoord{{0, 0, 0}, {0, 1, 1}, {1, 0, 1}}},
		{"300 leaves is a full + partial-44 level-0 tile and a partial-1 level-1 root", 300, []TileCoord{{0, 0, 0}, {0, 1, 44}, {1, 0, 1}}},
		{"513 leaves is two full + partial-1 level-0 tiles and a partial-2 level-1 root", 513, []TileCoord{{0, 0, 0}, {0, 1, 0}, {0, 2, 1}, {1, 0, 2}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TileCoords(tc.treeSize)
			if len(got) != len(tc.want) {
				t.Fatalf("TileCoords(%d) returned %d coords, want %d (got %+v)", tc.treeSize, len(got), len(tc.want), got)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("TileCoords(%d) = %+v, want %+v", tc.treeSize, got, tc.want)
			}
		})
	}
}

// TestTileCoordsExactPowerOf256 pins the root-collapse boundary: an exact
// 65536-leaf tree (256^2 leaves = 256 full level-0 tiles folding into one full
// level-1 root) emits exactly 257 tiles ending with the lone full root {1, 0, 0},
// and never a spurious empty level above the root.
func TestTileCoordsExactPowerOf256(t *testing.T) {
	got := TileCoords(65536)
	want := make([]TileCoord, 0, 257)
	for index := uint64(0); index < 256; index++ {
		want = append(want, TileCoord{Level: 0, Index: index, Partial: 0})
	}
	want = append(want, TileCoord{Level: 1, Index: 0, Partial: 0})
	if len(got) != 257 {
		t.Fatalf("TileCoords(65536) returned %d coords, want 257", len(got))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TileCoords(65536) = %+v, want %+v", got, want)
	}
	if last := got[len(got)-1]; last != (TileCoord{Level: 1, Index: 0, Partial: 0}) {
		t.Errorf("TileCoords(65536) ends with %+v, want the full root {1 0 0}", last)
	}
}

// TestTileCoordsExactPowerOf256PlusOne pins the off-by-one above the boundary: a
// 65537-leaf tree carries the lone extra leaf as a partial-1 at level-0 index 256
// and still folds up to the full level-1 root {1, 0, 0} (the level-1 row has
// exactly 256 nodes — 256 full level-0 tiles plus the extra leaf's hash splits
// only the level-0 row, so the root remains a full 256-hash tile).
func TestTileCoordsExactPowerOf256PlusOne(t *testing.T) {
	got := TileCoords(65537)
	if !containsCoord(got, TileCoord{Level: 0, Index: 256, Partial: 1}) {
		t.Errorf("TileCoords(65537) = %+v, missing the lone extra leaf {0 256 1}", got)
	}
	if last := got[len(got)-1]; last != (TileCoord{Level: 1, Index: 0, Partial: 0}) {
		t.Errorf("TileCoords(65537) ends with %+v, want the full root {1 0 0}", last)
	}
}

// TestTileCoordsZeroIsNonNilEmpty pins the documented zero-case contract: an empty
// tree returns a non-nil, length-0 slice (not a nil slice).
func TestTileCoordsZeroIsNonNilEmpty(t *testing.T) {
	got := TileCoords(0)
	if got == nil {
		t.Fatal("TileCoords(0) = nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("TileCoords(0) has length %d, want 0", len(got))
	}
}

// TestTileCoordsPartialMatchesOracle re-grounds every emitted Partial against the
// tessera layout.PartialTileSize oracle directly, so the goldens above are
// independently confirmed as tessera ground truth rather than self-asserted.
func TestTileCoordsPartialMatchesOracle(t *testing.T) {
	for _, size := range []uint64{1, 255, 256, 257, 300, 513, 65536, 65537} {
		for _, c := range TileCoords(size) {
			if want := layout.PartialTileSize(c.Level, c.Index, size); c.Partial != want {
				t.Errorf("TileCoords(%d) coord %+v has Partial %d, oracle PartialTileSize = %d", size, c, c.Partial, want)
			}
		}
	}
}

// containsCoord reports whether coords contains the exact tile coordinate.
func containsCoord(coords []TileCoord, target TileCoord) bool {
	for _, c := range coords {
		if c == target {
			return true
		}
	}
	return false
}
