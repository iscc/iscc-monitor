// Golden tests pinning the tlog-tiles layout seam. Because internal/tiles only
// delegates to tessera's api/layout, these vectors prove the delegation wiring
// and lock the canonical paths so a future refactor cannot silently diverge. The
// path strings are ground truth lifted from tessera's own api/layout/paths_test.go
// and tile.go, not author-asserted.
package tiles

import "testing"

func TestTilePath(t *testing.T) {
	tests := []struct {
		name  string
		level uint64
		index uint64
		p     uint8
		want  string
	}{
		{"full first tile", 1, 0, 0, "tile/1/000"},
		{"partial first tile width 255", 0, 0, 255, "tile/0/000.p/255"},
		{"deep level grouped index", 15, 455667, 0, "tile/15/x455/667"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TilePath(tc.level, tc.index, tc.p); got != tc.want {
				t.Errorf("TilePath(%d, %d, %d) = %q, want %q", tc.level, tc.index, tc.p, got, tc.want)
			}
		})
	}
}

func TestEntriesPath(t *testing.T) {
	tests := []struct {
		name  string
		index uint64
		p     uint8
		want  string
	}{
		{"partial bundle width 8", 0, 8, "tile/entries/000.p/8"},
		{"full bundle index 255", 255, 0, "tile/entries/255"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := EntriesPath(tc.index, tc.p); got != tc.want {
				t.Errorf("EntriesPath(%d, %d) = %q, want %q", tc.index, tc.p, got, tc.want)
			}
		})
	}
}

func TestPartialTileSize(t *testing.T) {
	tests := []struct {
		name    string
		level   uint64
		index   uint64
		logSize uint64
		want    uint8
	}{
		// In a 300-leaf tree the first tile (index 0) is full (256 leaves -> 0),
		// and the leftover 44 leaves spill into the second tile (index 1).
		{"full first tile of 300-leaf tree reports 0", 0, 0, 300, 0},
		{"second tile of 300-leaf tree is partial-44", 0, 1, 300, 44},
		// A 44-leaf tree's first-and-only tile is itself the partial-44.
		{"only tile of 44-leaf tree is partial-44", 0, 0, 44, 44},
		{"exactly-full first tile reports 0", 0, 0, 256, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PartialTileSize(tc.level, tc.index, tc.logSize); got != tc.want {
				t.Errorf("PartialTileSize(%d, %d, %d) = %d, want %d", tc.level, tc.index, tc.logSize, got, tc.want)
			}
		})
	}
}

func TestIsFull(t *testing.T) {
	tests := []struct {
		name  string
		width int
		want  bool
	}{
		{"full at width 256", 256, true},
		{"partial at width 255", 255, false},
		{"partial at width 1", 1, false},
		{"partial at width 0", 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsFull(tc.width); got != tc.want {
				t.Errorf("IsFull(%d) = %v, want %v", tc.width, got, tc.want)
			}
		})
	}
}

// TestConstants pins the tlog-tiles spec constants this seam re-exports.
func TestConstants(t *testing.T) {
	if TileWidth != 256 {
		t.Errorf("TileWidth = %d, want 256", TileWidth)
	}
	if TileHeight != 8 {
		t.Errorf("TileHeight = %d, want 8", TileHeight)
	}
}
