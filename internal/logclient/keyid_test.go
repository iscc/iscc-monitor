// Golden + error tests for KeyIDFromVerifier. The golden vectors are the two live
// testnet vkeys (sb0, sb1) didweb.VerifierKey produces; sb0's tail deliberately
// contains a '+' inside its base64, proving the SplitN(…, 3) does not over-split.
package logclient

import "testing"

// TestKeyIDFromVerifier round-trips the live vkeys (including sb0's '+'-bearing
// base64 tail) and confirms malformed inputs return a non-nil error.
func TestKeyIDFromVerifier(t *testing.T) {
	t.Parallel()

	golden := []struct {
		name string
		vkey string
		want uint32
	}{
		{
			// sb0's base64 tail has a '+' (AaV+ivnly67…), so a plain Split would
			// over-split and read the wrong middle field.
			name: "sb0 with plus in base64 tail",
			vkey: "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5",
			want: 0x40b74463,
		},
		{
			name: "sb1",
			vkey: "sb1.amlet.id/log+069d0f14+AW9UGZSxDvYFeewtbNU74zEMv12ChQPcuE4veN80nNtb",
			want: 0x069d0f14,
		},
	}
	for _, tc := range golden {
		t.Run(tc.name, func(t *testing.T) {
			got, err := KeyIDFromVerifier(tc.vkey)
			if err != nil {
				t.Fatalf("KeyIDFromVerifier(%q): unexpected error: %v", tc.vkey, err)
			}
			if got != tc.want {
				t.Errorf("KeyIDFromVerifier(%q) = %08x, want %08x", tc.vkey, got, tc.want)
			}
		})
	}

	malformed := []struct {
		name string
		vkey string
	}{
		{name: "no plus fields", vkey: "no-plus-fields"},
		{name: "one plus only", vkey: "name+40b74463"},
		{name: "non-hex middle field", vkey: "a+zzzz+b"},
		{name: "empty middle field", vkey: "name++base64"},
		{name: "empty string", vkey: ""},
	}
	for _, tc := range malformed {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := KeyIDFromVerifier(tc.vkey); err == nil {
				t.Errorf("KeyIDFromVerifier(%q) = nil error, want a non-nil error", tc.vkey)
			}
		})
	}
}
