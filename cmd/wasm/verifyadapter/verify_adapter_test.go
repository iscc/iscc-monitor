// Tests for the pure WASM-marshaling adapter. They reuse the SAME 4-leaf
// base64-Std golden vector pinned in internal/proof/verify/verify_test.go
// (record "leaf-1", index 1, size 4) so VerifyJSON's verdict is proven identical
// to the server core's at the marshaling boundary — that IS the "identical
// vectors yield identical verdicts, WASM vs server" check. The cases assert the
// three-way contract survives marshaling: positive verified; a wrong record and a
// tampered root are negative VERDICTS (verified=false, errMsg==""), not errors;
// malformed base64 and an index >= size precondition are errors (errMsg!="").
package verifyadapter

import (
	"encoding/base64"
	"math"
	"strings"
	"testing"
)

// goldenRoot is the base64-Std root of the 4-leaf golden tree, copied verbatim
// from internal/proof/verify/verify_test.go.
const goldenRoot = "vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM="

// goldenProof is the base64-Std inclusion proof for index 1 in size 4, copied
// verbatim from internal/proof/verify/verify_test.go.
var goldenProof = []string{
	"MF31n5WQw8msY9KydDw4jjeSRJB4zr9/s9vmRxZDsrc=",
	"vUX/KHlnBNiL2sUbHfVT/aWYN7YW1tHLIRTbw7CH/2k=",
}

// goldenRecordB64 is base64-Std("leaf-1"), the record the adapter decodes before
// hashing it into the leaf hash. The monitor emits the record base64-Std.
const goldenRecordB64 = "bGVhZi0x"

func TestVerifyJSON(t *testing.T) {
	tamperedRoot := "AdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM=" // first byte flipped vs goldenRoot

	cases := []struct {
		name         string
		record       string
		root         string
		proof        []string
		index        uint64
		size         uint64
		wantVerified bool
		wantErr      bool // true => errMsg must be non-empty; false => errMsg must be ""
	}{
		{
			name:         "positive golden vector verifies",
			record:       goldenRecordB64,
			root:         goldenRoot,
			proof:        goldenProof,
			index:        1,
			size:         4,
			wantVerified: true,
			wantErr:      false,
		},
		{
			name:         "wrong record is a negative verdict not an error",
			record:       "V1JPTkc=", // base64-Std("WRONG")
			root:         goldenRoot,
			proof:        goldenProof,
			index:        1,
			size:         4,
			wantVerified: false,
			wantErr:      false,
		},
		{
			name:         "tampered root is a negative verdict not an error",
			record:       goldenRecordB64,
			root:         tamperedRoot,
			proof:        goldenProof,
			index:        1,
			size:         4,
			wantVerified: false,
			wantErr:      false,
		},
		{
			name:         "malformed base64 record is an error",
			record:       "not!valid!base64",
			root:         goldenRoot,
			proof:        goldenProof,
			index:        1,
			size:         4,
			wantVerified: false,
			wantErr:      true,
		},
		{
			name:         "index >= size is a precondition error",
			record:       goldenRecordB64,
			root:         goldenRoot,
			proof:        goldenProof,
			index:        4,
			size:         4,
			wantVerified: false,
			wantErr:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verified, errMsg := VerifyJSON(tc.record, tc.root, tc.proof, tc.index, tc.size)
			if verified != tc.wantVerified {
				t.Errorf("verified = %v, want %v (errMsg=%q)", verified, tc.wantVerified, errMsg)
			}
			if gotErr := errMsg != ""; gotErr != tc.wantErr {
				t.Errorf("errMsg = %q, want non-empty=%v", errMsg, tc.wantErr)
			}
		})
	}
}

// TestSafeIndex exercises the JS→Go integer guard that protects the WASM-vs-server
// vector parity: a truncated / non-integer / out-of-safe-range JS Number is NOT an
// identical vector, so SafeIndex must fail closed before the float64→uint64 narrowing
// silently truncates. The table covers each reject branch (the issue's named cases),
// the accept branch, and the boundary precisely — maxSafeInteger (2^53 - 1) is
// ACCEPTED, 2^53 is REJECTED (the guard is > maxSafeInteger). The NaN/Inf cases pin
// the specific "not a finite number" message via wantMsg because NaN also fails the
// fractional check and Inf also fails the range check — without the message assertion
// dropping the finite-number branch would leave the test green (vacuous). Reverting
// any single reject branch makes this test fail (non-vacuity is mutation-verified).
func TestSafeIndex(t *testing.T) {
	cases := []struct {
		name    string
		v       float64
		want    uint64
		wantErr bool   // true => errMsg must be non-empty; false => errMsg must be ""
		wantMsg string // if non-empty, errMsg must contain this substring (pins the branch)
	}{
		{name: "fractional is rejected", v: 1.9, want: 0, wantErr: true, wantMsg: "not an integer"},
		{name: "NaN is rejected", v: math.NaN(), want: 0, wantErr: true, wantMsg: "not a finite number"},
		{name: "positive infinity is rejected", v: math.Inf(1), want: 0, wantErr: true, wantMsg: "not a finite number"},
		{name: "negative is rejected", v: -1, want: 0, wantErr: true, wantMsg: "out of safe-integer range"},
		{name: "2^53 (just above the cap) is rejected", v: float64(1 << 53), want: 0, wantErr: true, wantMsg: "out of safe-integer range"},
		{name: "zero is accepted", v: 0, want: 0, wantErr: false},
		{name: "small integer is accepted", v: 5, want: 5, wantErr: false},
		{name: "maxSafeInteger (2^53 - 1) is accepted", v: float64(1<<53 - 1), want: 1<<53 - 1, wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, errMsg := SafeIndex(tc.v, "index")
			if got != tc.want {
				t.Errorf("got = %d, want %d (errMsg=%q)", got, tc.want, errMsg)
			}
			if gotErr := errMsg != ""; gotErr != tc.wantErr {
				t.Errorf("errMsg = %q, want non-empty=%v", errMsg, tc.wantErr)
			}
			if tc.wantMsg != "" && !strings.Contains(errMsg, tc.wantMsg) {
				t.Errorf("errMsg = %q, want it to contain %q", errMsg, tc.wantMsg)
			}
		})
	}
}

// envelopeB64 base64-Std-encodes a minimal canonical log-entry envelope committing
// isccID, mirroring the {iscc_id, note.$schema} shape logclient.recordEnvelope reads
// (the same record VerifyJSON would hash into the leaf).
func envelopeB64(isccID string) string {
	json := `{"iscc_id":"` + isccID + `","note":{"$schema":"https://schema.iscc.codes/iscc-note-0.8.0.json"}}`
	return base64.StdEncoding.EncodeToString([]byte(json))
}

// TestRecordCommitsID exercises the id-binding half of a full re-verification: the
// record's own committed iscc_id must match the requested id before verified renders.
// The table pins the three-way contract — a prefixed and a bare request both bind a
// matching envelope (canonicalization), a different id is a negative VERDICT (not an
// error, the wrong-record class the split-view alert keys on), and broken base64 / a
// non-JSON record are ERRORs. A record with no iscc_id field is a non-match negative
// verdict, never a pass. Non-vacuity (review): dropping the strings.TrimPrefix
// canonicalization fails the bare-request match; collapsing the mismatch branch to
// always-true fails the different-id case; collapsing the parse-error branch to a
// bare false fails the broken-input cases.
func TestRecordCommitsID(t *testing.T) {
	const committed = "ISCC:MAIAGISETI7MJ4EAB"

	cases := []struct {
		name      string
		recordB64 string
		wantID    string
		wantOK    bool
		wantErr   bool // true => errMsg must be non-empty; false => errMsg must be ""
	}{
		{
			name:      "prefixed request matches the committed id",
			recordB64: envelopeB64(committed),
			wantID:    "ISCC:MAIAGISETI7MJ4EAB",
			wantOK:    true,
			wantErr:   false,
		},
		{
			name:      "bare request canonicalizes to a match",
			recordB64: envelopeB64(committed),
			wantID:    "MAIAGISETI7MJ4EAB",
			wantOK:    true,
			wantErr:   false,
		},
		{
			name:      "different id is a negative verdict not an error",
			recordB64: envelopeB64(committed),
			wantID:    "ISCC:MAIBOTHERIDXXXXXX",
			wantOK:    false,
			wantErr:   false,
		},
		{
			name:      "envelope with no iscc_id is a non-match negative verdict",
			recordB64: base64.StdEncoding.EncodeToString([]byte(`{"note":{"$schema":"x"}}`)),
			wantID:    "ISCC:MAIAGISETI7MJ4EAB",
			wantOK:    false,
			wantErr:   false,
		},
		{
			name:      "malformed base64 record is an error",
			recordB64: "not!base64",
			wantID:    "ISCC:MAIAGISETI7MJ4EAB",
			wantOK:    false,
			wantErr:   true,
		},
		{
			name:      "well-formed base64 of non-JSON is an error",
			recordB64: base64.StdEncoding.EncodeToString([]byte("not json")),
			wantID:    "ISCC:MAIAGISETI7MJ4EAB",
			wantOK:    false,
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, errMsg := RecordCommitsID(tc.recordB64, tc.wantID)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v (errMsg=%q)", ok, tc.wantOK, errMsg)
			}
			if gotErr := errMsg != ""; gotErr != tc.wantErr {
				t.Errorf("errMsg = %q, want non-empty=%v", errMsg, tc.wantErr)
			}
		})
	}
}
