// This file is the golden test for RecordBytesFromBundle. It builds a tlog-tiles
// entry bundle by encoding raw record bytes with the C2SP framing (2-byte
// big-endian length prefix + data per entry) itself — the independent third path,
// distinct from RecordBytesFromBundle's decode path (api.EntryBundle.UnmarshalText)
// — then asserts each in-range offset returns the original record bytes verbatim,
// that an out-of-range offset returns ErrLeafOutOfBundle (via errors.Is), and that a
// truncated bundle frame surfaces a wrapped error. The shared encodeBundle helper is
// defined in leafhasher_test.go (same package).
package logclient

import (
	"bytes"
	"errors"
	"testing"
)

func TestRecordBytesFromBundle(t *testing.T) {
	// Records of differing lengths, including a zero-length entry, so the framing
	// walk visits short, empty, and longer entries.
	records := [][]byte{
		[]byte("first record"),
		{},
		[]byte("a much longer third record with more bytes than the first"),
	}

	bundle := encodeBundle(records)

	for i, want := range records {
		got, err := RecordBytesFromBundle(bundle, uint64(i))
		if err != nil {
			t.Fatalf("RecordBytesFromBundle(offset %d) returned error: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("offset %d = %q, want %q", i, got, want)
		}
	}
}

func TestRecordBytesFromBundleOutOfRange(t *testing.T) {
	records := [][]byte{[]byte("only one record")}
	bundle := encodeBundle(records)

	// offset == len(Entries) and beyond are both out of range and must surface
	// ErrLeafOutOfBundle (checkable via errors.Is), not a panic or a 500.
	for _, offset := range []uint64{1, 2, 100} {
		_, err := RecordBytesFromBundle(bundle, offset)
		if !errors.Is(err, ErrLeafOutOfBundle) {
			t.Errorf("offset %d: err = %v, want ErrLeafOutOfBundle", offset, err)
		}
	}
}

func TestRecordBytesFromBundleTruncated(t *testing.T) {
	// A 2-byte length prefix claiming 5 data bytes with only 1 present must surface
	// the wrapped UnmarshalText error.
	truncated := []byte{0x00, 0x05, 0x01}
	if _, err := RecordBytesFromBundle(truncated, 0); err == nil {
		t.Fatal("RecordBytesFromBundle(truncated bundle) returned nil error, want non-nil")
	}
}

func TestRecordBytesFromBundleEmpty(t *testing.T) {
	// An empty bundle decodes to zero entries; any offset is out of range.
	for _, bundle := range [][]byte{nil, {}} {
		_, err := RecordBytesFromBundle(bundle, 0)
		if !errors.Is(err, ErrLeafOutOfBundle) {
			t.Errorf("RecordBytesFromBundle(%v, 0): err = %v, want ErrLeafOutOfBundle", bundle, err)
		}
	}
}
