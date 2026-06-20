// Tests for the pure shrink- and fork-trigger detection. They drive CheckShrink
// and CheckFork over tables that pin every load-bearing boundary: a strict
// decrease is a shrink and a differing root at equal size is a fork, while equal
// size (re-observation), growth, and the fresh-store prev==0 zero are neither —
// so the fresh FollowState{}.LastSize zero is never misread as a violation.
package logclient

import "testing"

func TestCheckShrink(t *testing.T) {
	cases := []struct {
		name string
		prev uint64
		next uint64
		want bool
	}{
		{name: "strict shrink", prev: 10183, next: 10182, want: true},
		{name: "equal size is re-observation not shrink", prev: 10183, next: 10183, want: false},
		{name: "growth", prev: 10183, next: 10184, want: false},
		{name: "fresh store no prior accepted size", prev: 0, next: 5, want: false},
		{name: "both zero", prev: 0, next: 0, want: false},
		{name: "shrink to zero", prev: 1, next: 0, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CheckShrink(tc.prev, tc.next); got != tc.want {
				t.Errorf("CheckShrink(%d, %d) = %v, want %v", tc.prev, tc.next, got, tc.want)
			}
		})
	}
}

// TestViolationShrinkKind pins the kind string this slice owns to the exact value
// the store.Violation.Kind field and the violations.kind column expect.
func TestViolationShrinkKind(t *testing.T) {
	if string(ViolationShrink) != "shrink" {
		t.Errorf("ViolationShrink = %q, want %q", ViolationShrink, "shrink")
	}
}

func TestCheckFork(t *testing.T) {
	var (
		rootA    = [rootBytes]byte{0: 0x01, 1: 0x01, 31: 0x01}
		rootB    = [rootBytes]byte{0: 0x02, 1: 0x02, 31: 0x02}
		zeroRoot [rootBytes]byte
	)
	if rootA == rootB {
		t.Fatal("test setup: rootA and rootB must differ")
	}
	cases := []struct {
		name     string
		prevSize uint64
		prevRoot [rootBytes]byte
		nextSize uint64
		nextRoot [rootBytes]byte
		want     bool
	}{
		{name: "same size different root", prevSize: 10183, prevRoot: rootA, nextSize: 10183, nextRoot: rootB, want: true},
		{name: "same size identical root is re-observation", prevSize: 10183, prevRoot: rootA, nextSize: 10183, nextRoot: rootA, want: false},
		{name: "shrink is shrinks concern not fork", prevSize: 10183, prevRoot: rootA, nextSize: 10182, nextRoot: rootB, want: false},
		{name: "growth even with differing root", prevSize: 10183, prevRoot: rootA, nextSize: 10184, nextRoot: rootB, want: false},
		{name: "fresh store no prior accepted size", prevSize: 0, prevRoot: zeroRoot, nextSize: 5, nextRoot: rootB, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CheckFork(tc.prevSize, tc.prevRoot, tc.nextSize, tc.nextRoot); got != tc.want {
				t.Errorf("CheckFork(%d, %x, %d, %x) = %v, want %v", tc.prevSize, tc.prevRoot, tc.nextSize, tc.nextRoot, got, tc.want)
			}
		})
	}
}

// TestViolationForkKind pins the fork kind string to the exact value the
// store.Violation.Kind field and the violations.kind column expect.
func TestViolationForkKind(t *testing.T) {
	if string(ViolationFork) != "fork" {
		t.Errorf("ViolationFork = %q, want %q", ViolationFork, "fork")
	}
}
