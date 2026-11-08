// Tests for KeyIDFromCheckpoint, the raw-checkpoint key-id recovery. They drive it
// against the live sb0 checkpoint fixture (testdata/live/) and pin the golden
// (name, keyID) = ("sb0.iscc.id/log", 0x40b74463). A cross-check asserts the same
// keyID comes out of KeyIDFromVerifier(sb0VKey) — the two recovery paths (raw sig
// line vs vkey string) must agree for one hub. Garbled input must error, not panic.
package logclient

import "testing"

func TestKeyIDFromCheckpoint(t *testing.T) {
	raw := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	name, keyID, err := KeyIDFromCheckpoint(raw)
	if err != nil {
		t.Fatalf("KeyIDFromCheckpoint error: %v", err)
	}
	if name != "sb0.iscc.id/log" {
		t.Errorf("name = %q, want sb0.iscc.id/log", name)
	}
	if keyID != 0x40b74463 {
		t.Errorf("keyID = %#08x, want 0x40b74463", keyID)
	}
}

// TestKeyIDFromCheckpointMatchesVerifier proves the raw-checkpoint and vkey-string
// recovery paths agree: KeyIDFromCheckpoint's keyID equals KeyIDFromVerifier of the
// hub's golden verifier key (sb0VKey, defined in verify_test.go). A divergence here
// would mean the cache lookup key recovered before resolution differs from the one
// recovered after — defeating the cache fast path.
func TestKeyIDFromCheckpointMatchesVerifier(t *testing.T) {
	raw := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	_, fromCheckpoint, err := KeyIDFromCheckpoint(raw)
	if err != nil {
		t.Fatalf("KeyIDFromCheckpoint error: %v", err)
	}
	fromVerifier, err := KeyIDFromVerifier(sb0VKey)
	if err != nil {
		t.Fatalf("KeyIDFromVerifier error: %v", err)
	}
	if fromCheckpoint != fromVerifier {
		t.Errorf("keyID disagreement: from checkpoint %#08x, from verifier %#08x", fromCheckpoint, fromVerifier)
	}
}

// TestKeyIDFromCheckpointGarbled proves a non-note input errors cleanly without
// panicking or indexing an empty sig slice.
func TestKeyIDFromCheckpointGarbled(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{"not a note", []byte("not a note")},
		{"empty", []byte{}},
		{"nil", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, keyID, err := KeyIDFromCheckpoint(tc.raw)
			if err == nil {
				t.Fatalf("KeyIDFromCheckpoint(%q) = (%q, %#x, nil), want error", tc.raw, name, keyID)
			}
		})
	}
}
