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

// seqsOf extracts the seqs from a RecordRow page, so an order assertion reads as a
// plain []uint64 comparison.
func seqsOf(rows []RecordRow) []uint64 {
	out := make([]uint64, len(rows))
	for i, r := range rows {
		out[i] = r.Seq
	}
	return out
}

// TestListRecords confirms the record-list read is newest-first (seq DESC), respects
// the page window (n / from cursor), reports the correct total, and reads id / schema
// back verbatim — the load-bearing behaviour the log-browser record list renders.
func TestListRecords(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	// Five contiguous leaves (seq 0..4) with distinct ids and schemas.
	recs := make([]ProjectionRecord, 5)
	for i := range recs {
		recs[i] = ProjectionRecord{
			HubID:      hub,
			Seq:        uint64(i),
			IsccID:     leafID(i),
			NoteSchema: "iscc-note-0.8.0.json",
		}
	}
	if err := s.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	// A full page from the newest record is newest-first and reports the right total.
	// last == 5 (the accepted tree size) lets every seq 0..4 through.
	rows, total, err := s.ListRecords(ctx, hub, 5, false, 0, 10)
	if err != nil {
		t.Fatalf("ListRecords full page: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if got := seqsOf(rows); !reflect.DeepEqual(got, []uint64{4, 3, 2, 1, 0}) {
		t.Errorf("newest-first seqs = %v, want [4 3 2 1 0]", got)
	}
	// id / schema round-trip verbatim on the newest row.
	if rows[0].IsccID != leafID(4) {
		t.Errorf("newest IsccID = %q, want %q", rows[0].IsccID, leafID(4))
	}
	if rows[0].NoteSchema != "iscc-note-0.8.0.json" {
		t.Errorf("newest NoteSchema = %q, want iscc-note-0.8.0.json", rows[0].NoteSchema)
	}

	// A page size of 2 from the newest yields exactly the two newest, total unchanged.
	rows, total, err = s.ListRecords(ctx, hub, 5, false, 0, 2)
	if err != nil {
		t.Fatalf("ListRecords first page n=2: %v", err)
	}
	if total != 5 {
		t.Errorf("total on small page = %d, want 5", total)
	}
	if got := seqsOf(rows); !reflect.DeepEqual(got, []uint64{4, 3}) {
		t.Errorf("first page (n=2) seqs = %v, want [4 3]", got)
	}

	// The next page via the from cursor (one below the page's smallest seq, 3-1=2)
	// continues newest-first from seq 2.
	rows, _, err = s.ListRecords(ctx, hub, 5, true, 2, 2)
	if err != nil {
		t.Fatalf("ListRecords second page: %v", err)
	}
	if got := seqsOf(rows); !reflect.DeepEqual(got, []uint64{2, 1}) {
		t.Errorf("second page (from=2, n=2) seqs = %v, want [2 1]", got)
	}

	// from=0 is a real cursor now (NOT the start-at-newest sentinel): it must page to
	// exactly seq 0, never jump back to the newest leaf.
	rows, _, err = s.ListRecords(ctx, hub, 5, true, 0, 2)
	if err != nil {
		t.Fatalf("ListRecords from=0 cursor: %v", err)
	}
	if got := seqsOf(rows); !reflect.DeepEqual(got, []uint64{0}) {
		t.Errorf("from=0 cursor seqs = %v, want [0] (the oldest record, not the newest page)", got)
	}
}

// TestListRecordsCeiling confirms ListRecords caps at the accepted tree size: a
// frozen/fault hub whose iscc_index holds projections at seq >= last (ingest writes
// projections before AdvanceAccepted) must list ONLY seq < last rows, and the total
// must count only those — an uncapped list would imply the unaccepted leaves are
// vouched for (coverage honesty, ADR-0001). The ceiling guard mirrors every other
// record route's seq >= LastSize cap.
func TestListRecordsCeiling(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	// Six contiguous leaves (seq 0..5) but the accepted tree is only size 4, so seqs
	// 4 and 5 sit ABOVE the accepted ceiling (the frozen-hub-past-violation case).
	recs := make([]ProjectionRecord, 6)
	for i := range recs {
		recs[i] = ProjectionRecord{HubID: hub, Seq: uint64(i), IsccID: leafID(i), NoteSchema: "iscc-note-0.8.0.json"}
	}
	if err := s.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	const last = 4
	rows, total, err := s.ListRecords(ctx, hub, last, false, 0, 10)
	if err != nil {
		t.Fatalf("ListRecords with ceiling: %v", err)
	}
	if total != last {
		t.Errorf("total = %d, want %d (only seq < last counted)", total, last)
	}
	if got := seqsOf(rows); !reflect.DeepEqual(got, []uint64{3, 2, 1, 0}) {
		t.Errorf("ceiling-capped seqs = %v, want [3 2 1 0] (seq 4 and 5 excluded)", got)
	}
	for _, r := range rows {
		if r.Seq >= last {
			t.Errorf("row seq %d >= last %d leaked past the ceiling", r.Seq, last)
		}
	}
}

// TestListRecordsScopedByHub confirms the page is hub-scoped: a second hub's records
// never appear in the first hub's list, and the total counts only the first hub.
func TestListRecordsScopedByHub(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hubA := newHub(t, s)
	hubB, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub hubB: %v", err)
	}

	// seq is the global PRIMARY KEY, so the two hubs use disjoint seq ranges (a real
	// network never reuses a leaf index across hubs); the scope assertion is that
	// hubB's seqs never bleed into hubA's page or count.
	if err := s.RecordProjections(ctx, []ProjectionRecord{
		{HubID: hubA, Seq: 10, IsccID: leafID(0), NoteSchema: "iscc-note-0.8.0.json"},
		{HubID: hubA, Seq: 11, IsccID: leafID(1), NoteSchema: "iscc-note-0.8.0.json"},
		{HubID: hubB, Seq: 20, IsccID: leafID(99), NoteSchema: "iscc-note-0.8.0.json"},
	}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	// last == 12 admits hubA's seqs 10 and 11 (both < 12) while staying hub-scoped.
	rows, total, err := s.ListRecords(ctx, hubA, 12, false, 0, 10)
	if err != nil {
		t.Fatalf("ListRecords hubA: %v", err)
	}
	if total != 2 {
		t.Errorf("hubA total = %d, want 2 (hub-scoped count)", total)
	}
	if got := seqsOf(rows); !reflect.DeepEqual(got, []uint64{11, 10}) {
		t.Errorf("hubA seqs = %v, want [11 10] (hub-scoped)", got)
	}
}

// TestListRecordsEmpty confirms a hub with no indexed records returns an empty slice,
// a zero total, and a nil error — the empty-log case the record list must render.
func TestListRecordsEmpty(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	rows, total, err := s.ListRecords(ctx, hub, 0, false, 0, 10)
	if err != nil {
		t.Fatalf("ListRecords empty: unexpected error %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("empty hub rows = %v, want empty slice", rows)
	}
	if total != 0 {
		t.Errorf("empty hub total = %d, want 0", total)
	}
}

// leafID returns a synthetic ISCC:-prefixed id unique per leaf index.
func leafID(i int) string {
	return "ISCC:LEAF" + string(rune('A'+i%26)) + string(rune('0'+i%10))
}

// TestRecordAt confirms the single-row reader returns the persisted projection row
// (id / schema) for an existing seq with found=true, reads the columns back verbatim
// (ADR-0008: nothing is interpreted), and returns found=false with a nil error for an
// absent seq — the single-record page's "no projection indexed" miss, not an error.
func TestRecordAt(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	rec := ProjectionRecord{
		HubID:        hub,
		Seq:          7,
		IsccID:       "ISCC:MAAGZTFQTTVIZ3IS",
		NoteSchema:   "iscc-note-delete-0.8.0",
		RecordSHA256: [32]byte{0x42},
	}
	if err := s.RecordProjections(ctx, []ProjectionRecord{rec}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	row, found, err := s.RecordAt(ctx, hub, 7)
	if err != nil {
		t.Fatalf("RecordAt existing seq: %v", err)
	}
	if !found {
		t.Fatalf("RecordAt(7) found = false, want true")
	}
	if row.Seq != 7 {
		t.Errorf("row.Seq = %d, want 7", row.Seq)
	}
	if row.IsccID != rec.IsccID {
		t.Errorf("row.IsccID = %q, want %q", row.IsccID, rec.IsccID)
	}
	if row.NoteSchema != rec.NoteSchema {
		t.Errorf("row.NoteSchema = %q, want %q", row.NoteSchema, rec.NoteSchema)
	}

	// An absent seq is a plain miss: found=false, nil error, zero RecordRow.
	row, found, err = s.RecordAt(ctx, hub, 999)
	if err != nil {
		t.Fatalf("RecordAt absent seq: unexpected error %v", err)
	}
	if found {
		t.Errorf("RecordAt(999) found = true, want false")
	}
	if (row != RecordRow{}) {
		t.Errorf("absent RecordAt row = %+v, want zero RecordRow", row)
	}
}

// TestRecordAtScopedByHub confirms RecordAt is hub-scoped: a seq indexed under one hub
// is not returned for another hub even though seq is the global primary key (so the
// (hub_id, seq) pair, not seq alone, identifies the row the page reads).
func TestRecordAtScopedByHub(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hubA := newHub(t, s)
	hubB, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub hubB: %v", err)
	}

	if err := s.RecordProjections(ctx, []ProjectionRecord{
		{HubID: hubA, Seq: 5, IsccID: leafID(0), NoteSchema: "iscc-note-0.8.0"},
	}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	if _, found, err := s.RecordAt(ctx, hubA, 5); err != nil || !found {
		t.Fatalf("RecordAt(hubA, 5) found = %v, err = %v, want true / nil", found, err)
	}
	// hubB never indexed seq 5, so its read is a miss.
	if _, found, err := s.RecordAt(ctx, hubB, 5); err != nil || found {
		t.Errorf("RecordAt(hubB, 5) found = %v, err = %v, want false / nil (hub-scoped)", found, err)
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
