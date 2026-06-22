// Tests for the pure WASM-marshaling adapter. They reuse the SAME 4-leaf
// base64-Std golden vector pinned in internal/proof/verify/verify_test.go
// (record "leaf-1", index 1, size 4) so VerifyJSON's verdict is proven identical
// to the server core's at the marshaling boundary — that IS the "identical
// vectors yield identical verdicts, WASM vs server" check. The cases assert the
// three-way contract survives marshaling: positive verified; a wrong record and a
// tampered root are negative VERDICTS (verified=false, errMsg==""), not errors;
// malformed base64 and an index >= size precondition are errors (errMsg!="").
package verifyadapter

import "testing"

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
