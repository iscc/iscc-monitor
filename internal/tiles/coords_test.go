// Golden tests pinning the entry-bundle coordinate enumeration. Because
// BundleCoords only delegates to tessera's layout.Range, these vectors prove the
// projection wiring and lock the boundary math so a future refactor cannot
// silently diverge. The vectors are ground truth pinned by running
// layout.Range(0, size, size) in-module — the 256-leaf boundary (300 -> full
// bundle at index 0 + partial-44 at index 1) is the load-bearing case.
package tiles

import (
	"reflect"
	"testing"
)

func TestBundleCoords(t *testing.T) {
	tests := []struct {
		name     string
		treeSize uint64
		want     []BundleCoord
	}{
		{"empty tree yields no bundles", 0, []BundleCoord{}},
		{"single leaf is a partial-1 bundle at index 0", 1, []BundleCoord{{0, 1}}},
		{"255 leaves is one partial-255 bundle at index 0", 255, []BundleCoord{{0, 255}}},
		{"exactly 256 leaves is one full bundle at index 0", 256, []BundleCoord{{0, 0}}},
		{"257 leaves is a full bundle then a partial-1 at index 1", 257, []BundleCoord{{0, 0}, {1, 1}}},
		// The 256-leaf boundary case: the first bundle is full (partial 0) and the
		// leftover 44 leaves are a partial at index 1, never a partial at index 0.
		{"300 leaves is a full bundle then a partial-44 at index 1", 300, []BundleCoord{{0, 0}, {1, 44}}},
		{"513 leaves is two full bundles then a partial-1 at index 2", 513, []BundleCoord{{0, 0}, {1, 0}, {2, 1}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BundleCoords(tc.treeSize)
			if len(got) != len(tc.want) {
				t.Fatalf("BundleCoords(%d) returned %d coords, want %d (got %+v)", tc.treeSize, len(got), len(tc.want), got)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("BundleCoords(%d) = %+v, want %+v", tc.treeSize, got, tc.want)
			}
		})
	}
}

// TestBundleCoordsZeroIsNonNilEmpty pins the documented zero-case contract: an
// empty tree returns a non-nil, length-0 slice (not a nil slice).
func TestBundleCoordsZeroIsNonNilEmpty(t *testing.T) {
	got := BundleCoords(0)
	if got == nil {
		t.Fatal("BundleCoords(0) = nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("BundleCoords(0) has length %d, want 0", len(got))
	}
}
