// Table-driven tests for DIDKey.ValidAt: the pure CID 1.0 validity-window
// predicate. Boundary instants are pinned explicitly so the half-open
// [ValidFrom, ValidUntil) semantics (and revoked-at-or-after) are exercised at
// the exact edges, not approximated.
package didweb

import (
	"testing"
	"time"
)

func TestValidAt(t *testing.T) {
	mustTime := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		return ts
	}

	now := mustTime("2026-06-20T00:00:00Z")
	until := mustTime("2026-12-31T00:00:00Z")
	past := mustTime("2020-01-01T00:00:00Z")
	future := mustTime("2030-01-01T00:00:00Z")

	cases := []struct {
		name string
		key  DIDKey
		at   time.Time
		want bool
	}{
		{
			name: "all zero fields is always valid",
			key:  DIDKey{},
			at:   now,
			want: true,
		},
		{
			name: "expired (ValidUntil in the past)",
			key:  DIDKey{ValidUntil: past},
			at:   now,
			want: false,
		},
		{
			name: "not yet active (ValidFrom in the future)",
			key:  DIDKey{ValidFrom: future},
			at:   now,
			want: false,
		},
		{
			name: "revoked in the past",
			key:  DIDKey{Revoked: past},
			at:   now,
			want: false,
		},
		{
			name: "active within [ValidFrom, ValidUntil)",
			key:  DIDKey{ValidFrom: past, ValidUntil: future},
			at:   now,
			want: true,
		},
		{
			name: "at ValidFrom is active (lower bound inclusive)",
			key:  DIDKey{ValidFrom: now},
			at:   now,
			want: true,
		},
		{
			name: "just before ValidFrom is not active",
			key:  DIDKey{ValidFrom: now},
			at:   now.Add(-time.Nanosecond),
			want: false,
		},
		{
			name: "at ValidUntil is expired (upper bound exclusive)",
			key:  DIDKey{ValidUntil: until},
			at:   until,
			want: false,
		},
		{
			name: "one nanosecond before ValidUntil is valid",
			key:  DIDKey{ValidUntil: until},
			at:   until.Add(-time.Nanosecond),
			want: true,
		},
		{
			name: "at Revoked is already revoked (boundary inclusive of revocation)",
			key:  DIDKey{Revoked: now},
			at:   now,
			want: false,
		},
		{
			name: "one nanosecond before Revoked is still valid",
			key:  DIDKey{Revoked: now},
			at:   now.Add(-time.Nanosecond),
			want: true,
		},
		{
			name: "Revoked overrides an otherwise-active window",
			key:  DIDKey{ValidFrom: past, ValidUntil: future, Revoked: past},
			at:   now,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.key.ValidAt(tc.at); got != tc.want {
				t.Errorf("ValidAt(%v) = %v, want %v", tc.at, got, tc.want)
			}
		})
	}
}

// TestValidAtZeroKeyNow confirms the live-fixture case from resolve_test.go: a
// key with no validity fields is valid right now, matching the "currently
// valid" assertion there.
func TestValidAtZeroKeyNow(t *testing.T) {
	if !(DIDKey{}).ValidAt(time.Now()) {
		t.Error("zero-value DIDKey.ValidAt(now) = false, want true (currently valid)")
	}
}
