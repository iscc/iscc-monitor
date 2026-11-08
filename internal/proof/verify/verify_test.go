// Tests for the pure inclusion-verifier core. The golden vector is a 4-leaf tree
// of records "leaf-0".."leaf-3"; the proof for index 1 in size 4 is built in-test
// from rfc6962.DefaultHasher (HashLeaf + HashChildren) so a library bump cannot
// silently rot the vector, and is cross-checked against the base64-Std literals
// recorded in next.md. The positive case asserts the proof rebuilds the root; the
// negative cases assert the (false, nil) vs (false, err) split that both call
// sites depend on (a wrong record and a tampered root are negative VERDICTS, not
// errors; an index >= size precondition is an error).
package verify

import (
	"encoding/base64"
	"testing"

	"github.com/transparency-dev/merkle/rfc6962"
)

// goldenRoot is the expected base64-Std root of the 4-leaf tree (next.md).
const goldenRoot = "vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM="

// goldenProof is the base64-Std inclusion proof for index 1 in size 4 (next.md).
var goldenProof = []string{
	"MF31n5WQw8msY9KydDw4jjeSRJB4zr9/s9vmRxZDsrc=",
	"vUX/KHlnBNiL2sUbHfVT/aWYN7YW1tHLIRTbw7CH/2k=",
}

// buildGoldenTree builds the 4-leaf tree's root and the index-1 inclusion proof
// directly from rfc6962.DefaultHasher, so the vector is grounded in the library's
// own math rather than frozen literals. Layout (h = HashLeaf, p = HashChildren):
//
//	root = p( p(h0,h1), p(h2,h3) )
//	proof(index=1) = [ h0, p(h2,h3) ]  (sibling then uncle)
func buildGoldenTree(t *testing.T) (root []byte, proof [][]byte) {
	t.Helper()
	h := rfc6962.DefaultHasher
	h0 := h.HashLeaf([]byte("leaf-0"))
	h1 := h.HashLeaf([]byte("leaf-1"))
	h2 := h.HashLeaf([]byte("leaf-2"))
	h3 := h.HashLeaf([]byte("leaf-3"))
	left := h.HashChildren(h0, h1)
	right := h.HashChildren(h2, h3)
	root = h.HashChildren(left, right)
	proof = [][]byte{h0, right}
	return root, proof
}

// mustDecode base64-Std-decodes a literal, failing the test on error.
func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return b
}

// TestVerifyInclusionGoldenCrossCheck asserts the in-test tree matches the
// base64-Std literals from next.md, pinning both against each other.
func TestVerifyInclusionGoldenCrossCheck(t *testing.T) {
	root, proof := buildGoldenTree(t)
	if got := base64.StdEncoding.EncodeToString(root); got != goldenRoot {
		t.Fatalf("root = %s, want %s", got, goldenRoot)
	}
	if len(proof) != len(goldenProof) {
		t.Fatalf("proof len = %d, want %d", len(proof), len(goldenProof))
	}
	for i, want := range goldenProof {
		if got := base64.StdEncoding.EncodeToString(proof[i]); got != want {
			t.Fatalf("proof[%d] = %s, want %s", i, got, want)
		}
	}
}

// TestVerifyInclusion exercises the positive verdict and the negative/error
// split that the verify-for-me and certificate §3 call sites depend on.
func TestVerifyInclusion(t *testing.T) {
	root, proof := buildGoldenTree(t)

	t.Run("positive rebuilds root", func(t *testing.T) {
		ok, err := VerifyInclusion([]byte("leaf-1"), 1, 4, proof, root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("got (false, nil), want (true, nil) for the golden vector")
		}
	})

	t.Run("wrong record is a negative verdict not an error", func(t *testing.T) {
		ok, err := VerifyInclusion([]byte("WRONG"), 1, 4, proof, root)
		if err != nil {
			t.Fatalf("got error %v, want a (false, nil) negative verdict", err)
		}
		if ok {
			t.Fatal("got (true, nil), want (false, nil) for a wrong record")
		}
	})

	t.Run("tampered root is a negative verdict not an error", func(t *testing.T) {
		tampered := make([]byte, len(root))
		copy(tampered, root)
		tampered[0] ^= 0xFF
		ok, err := VerifyInclusion([]byte("leaf-1"), 1, 4, proof, tampered)
		if err != nil {
			t.Fatalf("got error %v, want a (false, nil) negative verdict", err)
		}
		if ok {
			t.Fatal("got (true, nil), want (false, nil) for a tampered root")
		}
	})

	t.Run("index >= size is a precondition error", func(t *testing.T) {
		ok, err := VerifyInclusion([]byte("leaf-1"), 4, 4, proof, root)
		if err == nil {
			t.Fatal("got nil error, want a precondition error for index >= size")
		}
		if ok {
			t.Fatal("got ok=true, want ok=false on the precondition error")
		}
	})
}

// TestVerifyInclusionAgainstLiteralVector decodes the next.md base64 literals
// directly (not the in-test tree) and verifies, so the wrapper is checked against
// the externally-recorded vector as well as the library-rebuilt one.
func TestVerifyInclusionAgainstLiteralVector(t *testing.T) {
	root := mustDecode(t, goldenRoot)
	proof := make([][]byte, len(goldenProof))
	for i, s := range goldenProof {
		proof[i] = mustDecode(t, s)
	}
	ok, err := VerifyInclusion([]byte("leaf-1"), 1, 4, proof, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("literal vector did not verify")
	}
}
