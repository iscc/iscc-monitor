// Golden test for verifier-key derivation: the Go output must byte-match the
// .claude/derive_vkey.py oracle for both live testnet hubs (sb0, sb1). The
// literals below are the oracle's recorded output and are load-bearing.
package didweb

import "testing"

func TestVerifierKey(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		did    string
		want   string
	}{
		{
			name:   "sb0",
			origin: "sb0.iscc.id/log",
			did:    "z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ",
			want:   "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5",
		},
		{
			name:   "sb1",
			origin: "sb1.amlet.id/log",
			did:    "z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk",
			want:   "sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pub, err := pubkeyFromDID(tc.did)
			if err != nil {
				t.Fatalf("pubkeyFromDID(%q) error: %v", tc.did, err)
			}
			if len(pub) != 32 {
				t.Fatalf("pubkey is %d bytes, want 32", len(pub))
			}
			got := verifierKey(tc.origin, pub)
			if got != tc.want {
				t.Errorf("verifierKey(%q) = %q, want %q", tc.origin, got, tc.want)
			}
		})
	}
}

func TestPubkeyFromDIDErrors(t *testing.T) {
	cases := []struct {
		name string
		did  string
	}{
		{name: "missing z prefix", did: "6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"},
		{name: "invalid base58 char", did: "z0OIl"},
		{name: "too short", did: "z1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := pubkeyFromDID(tc.did); err == nil {
				t.Errorf("pubkeyFromDID(%q) = nil error, want error", tc.did)
			}
		})
	}
}
