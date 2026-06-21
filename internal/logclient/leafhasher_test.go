// This file is the golden test for LeafHashes. It builds a tlog-tiles entry
// bundle by encoding raw record bytes with the C2SP framing (2-byte big-endian
// length prefix + data per entry) itself, then asserts each LeafHashes(bundle)[i]
// byte-equals rfc6962.DefaultHasher.HashLeaf(records[i]) computed independently in
// the test. The in-test encode path is distinct from both LeafHashes' decode path
// (api.EntryBundle.UnmarshalText) and its hash path (rfc6962), so the cross-check
// is ground truth, not a tautology. Records of differing lengths (including a
// zero-length entry) make the framing walk non-vacuous.
package logclient

import (
	"encoding/binary"
	"testing"

	"github.com/transparency-dev/merkle/rfc6962"
)

// encodeBundle frames raw records into a tlog-tiles entry bundle, mirroring the
// C2SP encoding api.EntryBundle.UnmarshalText decodes: each record is prefixed
// with its big-endian uint16 length, then concatenated. This is the test's
// independent encode path — never imported from the code under test.
func encodeBundle(records [][]byte) []byte {
	var out []byte
	for _, rec := range records {
		var prefix [2]byte
		binary.BigEndian.PutUint16(prefix[:], uint16(len(rec)))
		out = append(out, prefix[:]...)
		out = append(out, rec...)
	}
	return out
}

func TestLeafHashes(t *testing.T) {
	// Records of differing lengths, including a zero-length entry, so the framing
	// walk visits short, empty, and longer entries.
	records := [][]byte{
		[]byte("first record"),
		{},
		[]byte("a much longer third record with more bytes than the first"),
	}

	bundle := encodeBundle(records)
	got, err := LeafHashes(bundle)
	if err != nil {
		t.Fatalf("LeafHashes returned error: %v", err)
	}
	if len(got) != len(records) {
		t.Fatalf("LeafHashes returned %d hashes, want %d", len(got), len(records))
	}

	for i, rec := range records {
		want := rfc6962.DefaultHasher.HashLeaf(rec)
		if len(got[i]) != len(want) {
			t.Errorf("hash[%d] length = %d, want %d", i, len(got[i]), len(want))
		}
		if len(got[i]) != 32 {
			t.Errorf("hash[%d] length = %d, want 32 bytes", i, len(got[i]))
		}
		if string(got[i]) != string(want[:]) {
			t.Errorf("hash[%d] = %x, want %x", i, got[i], want)
		}
	}
}

func TestLeafHashesEmpty(t *testing.T) {
	// nil and an explicitly empty bundle both decode to zero entries with no error.
	for _, bundle := range [][]byte{nil, {}} {
		got, err := LeafHashes(bundle)
		if err != nil {
			t.Errorf("LeafHashes(%v) returned error: %v", bundle, err)
		}
		if len(got) != 0 {
			t.Errorf("LeafHashes(%v) returned %d hashes, want 0", bundle, len(got))
		}
	}
}

func TestLeafHashesTruncated(t *testing.T) {
	// A 2-byte length prefix claiming 5 data bytes with only 1 present must surface
	// the wrapped UnmarshalText error.
	truncated := []byte{0x00, 0x05, 0x01}
	if _, err := LeafHashes(truncated); err == nil {
		t.Fatal("LeafHashes(truncated bundle) returned nil error, want non-nil")
	}
}
