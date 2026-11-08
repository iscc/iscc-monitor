// This file is the golden test for BundleProjections. It frames hand-written
// JCS-style record JSON into a tlog-tiles entry bundle with the C2SP framing
// (2-byte big-endian length prefix + data per entry) itself — an independent third
// encode path, distinct from both the decode (api.EntryBundle.UnmarshalText) and the
// content hash (crypto/sha256) the code under test uses, so every cross-check is
// ground truth, not a tautology. It pins BOTH a declaration and a deletion record so
// the schema-agnostic claim is exercised across the two known note types: the
// deletion-vs-declaration assertion proves the decoder reads the INNER note.$schema,
// never a constant.
package logclient

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

const (
	declSchema = "http://purl.org/iscc/schema/iscc-note-0.8.0.json"
	delSchema  = "http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"
)

// frameBundle frames raw records into a tlog-tiles entry bundle, mirroring the C2SP
// encoding api.EntryBundle.UnmarshalText decodes: each record is prefixed with its
// big-endian uint16 length, then concatenated. This is the test's independent encode
// path — never imported from the code under test.
func frameBundle(records [][]byte) []byte {
	var out []byte
	for _, rec := range records {
		var prefix [2]byte
		binary.BigEndian.PutUint16(prefix[:], uint16(len(rec)))
		out = append(out, prefix[:]...)
		out = append(out, rec...)
	}
	return out
}

func TestBundleProjections(t *testing.T) {
	// JCS-style envelopes: top-level {$schema, iscc_id, note}. The envelope's own
	// $schema is the log-entry schema; note.$schema is the discriminator the
	// projection must key on. Record 0 is a declaration, record 1 a deletion, so the
	// decoder must read the INNER schema to tell them apart. The declaration carries
	// an inner note.timestamp (its own creation time) and the deletion OMITS it, so
	// the fold must read the per-record value and yield "" for the absent (optional)
	// case (ADR-0008: read verbatim, never parse, tolerate absence).
	declID := "ISCC:MAIGIIFJRDGEQQAA"
	delID := "ISCC:KUABKF53JEQIDQQ7YU7LB4VHCHGWU"
	const declTime = "2026-06-21T12:34:56Z"
	declRecord := []byte(`{"$schema":"log-entry","iscc_id":"` + declID +
		`","note":{"$schema":"` + declSchema + `","timestamp":"` + declTime + `"}}`)
	delRecord := []byte(`{"$schema":"log-entry","iscc_id":"` + delID +
		`","note":{"$schema":"` + delSchema + `"}}`)
	records := [][]byte{declRecord, delRecord}

	const baseSeq = 256
	bundle := frameBundle(records)
	got, err := BundleProjections(bundle, baseSeq)
	if err != nil {
		t.Fatalf("BundleProjections returned error: %v", err)
	}
	if len(got) != len(records) {
		t.Fatalf("BundleProjections returned %d projections, want %d", len(got), len(records))
	}

	// Guard non-vacuity: the two records carry different inner schemas, so an
	// assertion that the decoder reads the inner schema cannot be trivially true.
	if declSchema == delSchema {
		t.Fatal("declaration and deletion schemas must differ for a meaningful test")
	}

	wantIDs := []string{declID, delID}
	wantSchemas := []string{declSchema, delSchema}
	// The declaration's own timestamp is read verbatim; the deletion omits it → "".
	wantTimes := []string{declTime, ""}
	for i := range records {
		p := got[i]
		if p.Seq != baseSeq+uint64(i) {
			t.Errorf("projection[%d].Seq = %d, want %d", i, p.Seq, baseSeq+uint64(i))
		}
		if p.IsccID != wantIDs[i] {
			t.Errorf("projection[%d].IsccID = %q, want %q", i, p.IsccID, wantIDs[i])
		}
		if p.NoteSchema != wantSchemas[i] {
			t.Errorf("projection[%d].NoteSchema = %q, want %q (verbatim inner note schema)",
				i, p.NoteSchema, wantSchemas[i])
		}
		if p.Timestamp != wantTimes[i] {
			t.Errorf("projection[%d].Timestamp = %q, want %q (verbatim inner note.timestamp)",
				i, p.Timestamp, wantTimes[i])
		}
		want := sha256.Sum256(records[i])
		if p.RecordSHA256 != want {
			t.Errorf("projection[%d].RecordSHA256 = %x, want %x", i, p.RecordSHA256, want)
		}
	}
}

func TestBundleProjectionsEmpty(t *testing.T) {
	// nil and an explicitly empty bundle both decode to zero leaves with no error.
	for _, bundle := range [][]byte{nil, {}} {
		got, err := BundleProjections(bundle, 0)
		if err != nil {
			t.Errorf("BundleProjections(%v) returned error: %v", bundle, err)
		}
		if len(got) != 0 {
			t.Errorf("BundleProjections(%v) returned %d projections, want 0", bundle, len(got))
		}
	}
}

func TestBundleProjectionsSchemaAgnostic(t *testing.T) {
	// A record with an empty iscc_id and an unmodeled note.$schema is indexed
	// verbatim, never rejected (ADR-0008).
	record := []byte(`{"iscc_id":"","note":{"$schema":"urn:example:unknown-note-type"}}`)
	got, err := BundleProjections(frameBundle([][]byte{record}), 7)
	if err != nil {
		t.Fatalf("BundleProjections returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("BundleProjections returned %d projections, want 1", len(got))
	}
	if got[0].IsccID != "" {
		t.Errorf("IsccID = %q, want empty (indexed verbatim)", got[0].IsccID)
	}
	if got[0].NoteSchema != "urn:example:unknown-note-type" {
		t.Errorf("NoteSchema = %q, want the verbatim unknown URI", got[0].NoteSchema)
	}
	if got[0].Seq != 7 {
		t.Errorf("Seq = %d, want 7", got[0].Seq)
	}
}

func TestBundleProjectionsMalformedRecord(t *testing.T) {
	// A record whose bytes are not valid JSON is a genuine fault: a wrapped error
	// naming the absolute seq, not a silently skipped leaf.
	bundle := frameBundle([][]byte{[]byte("not json at all")})
	if _, err := BundleProjections(bundle, 256); err == nil {
		t.Fatal("BundleProjections(malformed record) returned nil error, want non-nil")
	}
}

func TestBundleProjectionsTruncatedBundle(t *testing.T) {
	// A 2-byte length prefix claiming 5 data bytes with only 1 present must surface
	// the wrapped UnmarshalText error.
	truncated := []byte{0x00, 0x05, 0x01}
	if _, err := BundleProjections(truncated, 0); err == nil {
		t.Fatal("BundleProjections(truncated bundle) returned nil error, want non-nil")
	}
}
