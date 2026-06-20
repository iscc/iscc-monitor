// Tests for the pure shrink-trigger detection. They drive CheckShrink over a
// table that pins every load-bearing boundary: a strict decrease is a shrink,
// while equal size (re-observation), growth, and the fresh-store prev==0 zero are
// not — so the fresh FollowState{}.LastSize zero is never misread as a violation.
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
