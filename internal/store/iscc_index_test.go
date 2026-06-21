// Tests for the schema-agnostic iscc_index writer/reader (RecordProjections,
// SeqsForISCCID). Each drives the public method against a t.TempDir() database and
// asserts on observable rows — the one-to-many iscc_id → []seq lookup, the
// idempotent seq upsert via a raw COUNT, the verbatim round-trip of unmodeled
// schemas and an empty id — never on Store internals, matching tiles_test.go.
//
// The hub is registered first (newHub) because iscc_index.hub_id REFERENCES
// hubs(hub_id) and foreign_keys are ON, so a dangling hub_id would fail the FK.
package store

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

// readProjectionRow reads the iscc_id_str, note_schema and record_sha256 columns for
// one seq, so the test pins the round-trip from the columns themselves rather than
// trusting the writer.
func readProjectionRow(t *testing.T, s *Store, seq uint64) (isccIDStr, noteSchema string, sha []byte) {
	t.Helper()
	err := s.db.QueryRow(
		"SELECT iscc_id_str, note_schema, record_sha256 FROM iscc_index WHERE seq = ?",
		int64(seq),
	).Scan(&isccIDStr, &noteSchema, &sha)
	if err != nil {
		t.Fatalf("read projection row seq %d: %v", seq, err)
	}
	return isccIDStr, noteSchema, sha
}

// TestRecordProjectionsRoundTrip confirms a stored projection reads its columns back
// verbatim, including the iscc_id BLOB driving SeqsForISCCID and the iscc_id_str /
// note_schema / record_sha256 columns.
func TestRecordProjectionsRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	rec := ProjectionRecord{
		HubID:        hub,
		Seq:          256,
		IsccID:       "ISCC:MAAGZTFQTTVIZ3IS",
		NoteSchema:   "iscc-note-0.8.0.json",
		RecordSHA256: [32]byte{1, 2, 3, 4},
	}
	if err := s.RecordProjections(ctx, []ProjectionRecord{rec}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	gotID, gotSchema, gotSha := readProjectionRow(t, s, 256)
	if gotID != rec.IsccID {
		t.Errorf("iscc_id_str = %q, want %q", gotID, rec.IsccID)
	}
	if gotSchema != rec.NoteSchema {
		t.Errorf("note_schema = %q, want %q", gotSchema, rec.NoteSchema)
	}
	if !bytes.Equal(gotSha, rec.RecordSHA256[:]) {
		t.Errorf("record_sha256 = %x, want %x", gotSha, rec.RecordSHA256[:])
	}

	seqs, err := s.SeqsForISCCID(ctx, hub, rec.IsccID)
	if err != nil {
		t.Fatalf("SeqsForISCCID: %v", err)
	}
	if !reflect.DeepEqual(seqs, []uint64{256}) {
		t.Errorf("SeqsForISCCID = %v, want [256]", seqs)
	}
}

// TestRecordProjectionsEmptySlice confirms an empty batch is a no-op returning nil
// and writes no rows.
func TestRecordProjectionsEmptySlice(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	if err := s.RecordProjections(ctx, nil); err != nil {
		t.Fatalf("RecordProjections(nil): %v", err)
	}
	if n := countRows(t, s, "iscc_index"); n != 0 {
		t.Errorf("iscc_index row count after empty batch = %d, want 0", n)
	}
}

// TestSeqsForISCCIDOneToMany confirms one iscc_id shared by a declaration and a
// deletion at two seqs returns both, ordered (ADR-0008: iscc_id → seq is one-to-many,
// schema-agnostic; lookups return a list).
func TestSeqsForISCCIDOneToMany(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	id := "ISCC:MAAGZTFQTTVIZ3IS"
	recs := []ProjectionRecord{
		{HubID: hub, Seq: 256, IsccID: id, NoteSchema: "iscc-note-0.8.0.json", RecordSHA256: [32]byte{0xaa}},
		{HubID: hub, Seq: 257, IsccID: id, NoteSchema: "iscc-note-delete-0.8.0.json", RecordSHA256: [32]byte{0xbb}},
	}
	if err := s.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	seqs, err := s.SeqsForISCCID(ctx, hub, id)
	if err != nil {
		t.Fatalf("SeqsForISCCID: %v", err)
	}
	if !reflect.DeepEqual(seqs, []uint64{256, 257}) {
		t.Errorf("SeqsForISCCID = %v, want [256 257]", seqs)
	}
}

// TestSeqsForISCCIDOrderedDescendingInput confirms ORDER BY seq sorts the result even
// when the rows are inserted in descending seq order, so the list is deterministic.
func TestSeqsForISCCIDOrderedDescendingInput(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	id := "ISCC:MAAGZTFQTTVIZ3IS"
	recs := []ProjectionRecord{
		{HubID: hub, Seq: 300, IsccID: id, NoteSchema: "iscc-note-0.8.0.json"},
		{HubID: hub, Seq: 42, IsccID: id, NoteSchema: "iscc-note-0.8.0.json"},
		{HubID: hub, Seq: 100, IsccID: id, NoteSchema: "iscc-note-0.8.0.json"},
	}
	if err := s.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	seqs, err := s.SeqsForISCCID(ctx, hub, id)
	if err != nil {
		t.Fatalf("SeqsForISCCID: %v", err)
	}
	if !reflect.DeepEqual(seqs, []uint64{42, 100, 300}) {
		t.Errorf("SeqsForISCCID = %v, want [42 100 300] (ascending)", seqs)
	}
}

// TestSeqsForISCCIDAbsent confirms an un-indexed iscc_id returns an empty slice and a
// nil error (absent is not an error).
func TestSeqsForISCCIDAbsent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	seqs, err := s.SeqsForISCCID(ctx, hub, "ISCC:NEVERINDEXED")
	if err != nil {
		t.Fatalf("SeqsForISCCID absent: unexpected error %v", err)
	}
	if len(seqs) != 0 {
		t.Errorf("absent iscc_id seqs = %v, want empty", seqs)
	}
}

// TestRecordProjectionsIdempotent confirms re-ingesting the same seq overwrites in
// place (still one row) and that an UPDATEd note_schema round-trips to the second
// write's value — re-ingesting a mirrored bundle must be idempotent (seq is the PK).
func TestRecordProjectionsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	first := ProjectionRecord{
		HubID: hub, Seq: 256, IsccID: "ISCC:MAAGZTFQTTVIZ3IS",
		NoteSchema: "iscc-note-0.8.0.json", RecordSHA256: [32]byte{0x11},
	}
	second := ProjectionRecord{
		HubID: hub, Seq: 256, IsccID: "ISCC:MAAGZTFQTTVIZ3IS",
		NoteSchema: "iscc-note-delete-0.8.0.json", RecordSHA256: [32]byte{0x22},
	}
	if err := s.RecordProjections(ctx, []ProjectionRecord{first}); err != nil {
		t.Fatalf("first RecordProjections: %v", err)
	}
	if err := s.RecordProjections(ctx, []ProjectionRecord{second}); err != nil {
		t.Fatalf("second RecordProjections: %v", err)
	}

	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM iscc_index WHERE seq = ?", int64(256)).Scan(&count); err != nil {
		t.Fatalf("count seq 256: %v", err)
	}
	if count != 1 {
		t.Errorf("rows for seq 256 = %d, want 1 (idempotent upsert)", count)
	}

	_, gotSchema, gotSha := readProjectionRow(t, s, 256)
	if gotSchema != second.NoteSchema {
		t.Errorf("note_schema after re-ingest = %q, want %q (second write wins)", gotSchema, second.NoteSchema)
	}
	if !bytes.Equal(gotSha, second.RecordSHA256[:]) {
		t.Errorf("record_sha256 after re-ingest = %x, want %x", gotSha, second.RecordSHA256[:])
	}
}

// TestRecordProjectionsSchemaAgnostic confirms an unmodeled note_schema and an empty
// iscc_id both persist and read back verbatim (ADR-0008: the index interprets and
// validates nothing — no rejection of unknown schemas or empty ids).
func TestRecordProjectionsSchemaAgnostic(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	future := ProjectionRecord{
		HubID: hub, Seq: 1, IsccID: "ISCC:FUTUREID",
		NoteSchema: "iscc-note-future-9.9.9", RecordSHA256: [32]byte{0x99},
	}
	emptyID := ProjectionRecord{
		HubID: hub, Seq: 2, IsccID: "",
		NoteSchema: "iscc-note-0.8.0.json", RecordSHA256: [32]byte{0x88},
	}
	if err := s.RecordProjections(ctx, []ProjectionRecord{future, emptyID}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	// The unmodeled schema persisted verbatim.
	if _, gotSchema, _ := readProjectionRow(t, s, 1); gotSchema != future.NoteSchema {
		t.Errorf("unmodeled note_schema = %q, want %q", gotSchema, future.NoteSchema)
	}
	// The empty iscc_id persisted (empty string, not NULL) and is looked up verbatim.
	gotID, _, _ := readProjectionRow(t, s, 2)
	if gotID != "" {
		t.Errorf("empty iscc_id_str = %q, want empty string", gotID)
	}
	seqs, err := s.SeqsForISCCID(ctx, hub, "")
	if err != nil {
		t.Fatalf("SeqsForISCCID empty id: %v", err)
	}
	if !reflect.DeepEqual(seqs, []uint64{2}) {
		t.Errorf("SeqsForISCCID(\"\") = %v, want [2] (empty id indexed verbatim)", seqs)
	}
}

// TestSeqsForISCCIDScopedByHub confirms the lookup is hub-scoped: two hubs indexing
// the same iscc_id do not bleed into each other's result.
func TestSeqsForISCCIDScopedByHub(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hubA := newHub(t, s)
	hubB, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub hubB: %v", err)
	}

	id := "ISCC:MAAGZTFQTTVIZ3IS"
	if err := s.RecordProjections(ctx, []ProjectionRecord{
		{HubID: hubA, Seq: 10, IsccID: id, NoteSchema: "iscc-note-0.8.0.json"},
		{HubID: hubB, Seq: 20, IsccID: id, NoteSchema: "iscc-note-0.8.0.json"},
	}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	seqs, err := s.SeqsForISCCID(ctx, hubA, id)
	if err != nil {
		t.Fatalf("SeqsForISCCID hubA: %v", err)
	}
	if !reflect.DeepEqual(seqs, []uint64{10}) {
		t.Errorf("hubA SeqsForISCCID = %v, want [10] (hub-scoped)", seqs)
	}
}
