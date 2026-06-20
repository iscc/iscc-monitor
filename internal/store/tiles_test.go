// Tests for the mirrored tile / entry-bundle CRUD (RecordTile, RecordEntryBundle,
// ReadTileBlob, ReadEntryBundleBlob, LatestCheckpointRaw). Each drives the public
// method against a t.TempDir() database and asserts on observable rows — round-trip
// bytes, the is_full column via a raw SELECT, the absent-is-not-an-error contract —
// never on Store internals, matching the style in checkpoints_test.go.
package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"testing"
	"time"
)

// readIsFull reads the raw is_full column for a tiles row (1 for a full tile, 0 for
// a partial), so the test pins the ADR-0005 partial-tile discipline from the column
// itself rather than trusting the writer.
func readIsFull(t *testing.T, s *Store, hubID int64, level, index uint64, width int) int {
	t.Helper()
	var isFull int
	err := s.db.QueryRow(
		"SELECT is_full FROM tiles WHERE hub_id = ? AND level = ? AND tile_index = ? AND width = ?",
		hubID, int64(level), int64(index), width,
	).Scan(&isFull)
	if err != nil {
		t.Fatalf("read is_full: %v", err)
	}
	return isFull
}

// readBundleIsFull reads the raw is_full column for an entry_bundles row.
func readBundleIsFull(t *testing.T, s *Store, hubID int64, bundleIndex uint64, width int) int {
	t.Helper()
	var isFull int
	err := s.db.QueryRow(
		"SELECT is_full FROM entry_bundles WHERE hub_id = ? AND bundle_index = ? AND width = ?",
		hubID, int64(bundleIndex), width,
	).Scan(&isFull)
	if err != nil {
		t.Fatalf("read bundle is_full: %v", err)
	}
	return isFull
}

// readTileSha reads the raw sha256 column for a tiles row.
func readTileSha(t *testing.T, s *Store, hubID int64, level, index uint64, width int) []byte {
	t.Helper()
	var sum []byte
	err := s.db.QueryRow(
		"SELECT sha256 FROM tiles WHERE hub_id = ? AND level = ? AND tile_index = ? AND width = ?",
		hubID, int64(level), int64(index), width,
	).Scan(&sum)
	if err != nil {
		t.Fatalf("read sha256: %v", err)
	}
	return sum
}

// newHub registers a hub and returns its id.
func newHub(t *testing.T, s *Store) int64 {
	t.Helper()
	id, err := s.UpsertHub(context.Background(), "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	return id
}

// TestRecordTileRoundTrip confirms a stored full tile reads back identical bytes
// and that the is_full / sha256 columns reflect a full (width 256) write.
func TestRecordTileRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	data := bytes.Repeat([]byte{0xab}, 8192)
	if err := s.RecordTile(ctx, hub, 0, 0, 256, data, time.Unix(1700000000, 0)); err != nil {
		t.Fatalf("RecordTile: %v", err)
	}

	got, found, err := s.ReadTileBlob(ctx, hub, 0, 0, 256)
	if err != nil {
		t.Fatalf("ReadTileBlob: %v", err)
	}
	if !found {
		t.Fatal("ReadTileBlob: full tile not found")
	}
	if !bytes.Equal(got, data) {
		t.Errorf("ReadTileBlob bytes = %x…, want %x…", got[:4], data[:4])
	}
	if f := readIsFull(t, s, hub, 0, 0, 256); f != 1 {
		t.Errorf("is_full for width 256 = %d, want 1", f)
	}
	want := sha256.Sum256(data)
	if got := readTileSha(t, s, hub, 0, 0, 256); !bytes.Equal(got, want[:]) {
		t.Errorf("sha256 = %x, want %x", got, want[:])
	}
}

// TestRecordTilePartialIsNotFull confirms a width < 256 partial sets is_full = 0
// and round-trips at its own width (ADR-0005: never promote a partial).
func TestRecordTilePartialIsNotFull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	data := bytes.Repeat([]byte{0xcd}, 44*32)
	if err := s.RecordTile(ctx, hub, 0, 1, 44, data, time.Unix(1700000001, 0)); err != nil {
		t.Fatalf("RecordTile partial: %v", err)
	}

	got, found, err := s.ReadTileBlob(ctx, hub, 0, 1, 44)
	if err != nil {
		t.Fatalf("ReadTileBlob partial: %v", err)
	}
	if !found || !bytes.Equal(got, data) {
		t.Errorf("partial round-trip found=%v equal=%v", found, bytes.Equal(got, data))
	}
	if f := readIsFull(t, s, hub, 0, 1, 44); f != 0 {
		t.Errorf("is_full for width 44 = %d, want 0", f)
	}
}

// TestRecordTilePartialOverwrite confirms a re-fetched partial overwrites in place
// via the composite-PK upsert (still one row, the latest bytes win).
func TestRecordTilePartialOverwrite(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	first := bytes.Repeat([]byte{0x01}, 64)
	second := bytes.Repeat([]byte{0x02}, 96)
	if err := s.RecordTile(ctx, hub, 0, 0, 100, first, time.Unix(1, 0)); err != nil {
		t.Fatalf("first RecordTile: %v", err)
	}
	if err := s.RecordTile(ctx, hub, 0, 0, 100, second, time.Unix(2, 0)); err != nil {
		t.Fatalf("second RecordTile: %v", err)
	}

	got, _, err := s.ReadTileBlob(ctx, hub, 0, 0, 100)
	if err != nil {
		t.Fatalf("ReadTileBlob: %v", err)
	}
	if !bytes.Equal(got, second) {
		t.Errorf("overwrite bytes = %x…, want %x…", got[:2], second[:2])
	}
	if n := countRows(t, s, "tiles"); n != 1 {
		t.Errorf("tiles row count after overwrite = %d, want 1", n)
	}
}

// TestReadTileBlobAbsent confirms an un-written tile returns found=false, nil error
// (absent is not an error).
func TestReadTileBlobAbsent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	got, found, err := s.ReadTileBlob(ctx, hub, 3, 9, 256)
	if err != nil {
		t.Fatalf("ReadTileBlob absent: unexpected error %v", err)
	}
	if found || got != nil {
		t.Errorf("absent tile found=%v got=%v, want false/nil", found, got)
	}
}

// TestRecordEntryBundleRoundTrip confirms an entry bundle round-trips and its
// is_full column tracks the full/partial width like tiles.
func TestRecordEntryBundleRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	full := bytes.Repeat([]byte{0xee}, 4096)
	if err := s.RecordEntryBundle(ctx, hub, 0, 256, full, time.Unix(1700000002, 0)); err != nil {
		t.Fatalf("RecordEntryBundle full: %v", err)
	}
	partial := bytes.Repeat([]byte{0xff}, 512)
	if err := s.RecordEntryBundle(ctx, hub, 1, 44, partial, time.Unix(1700000003, 0)); err != nil {
		t.Fatalf("RecordEntryBundle partial: %v", err)
	}

	gotFull, found, err := s.ReadEntryBundleBlob(ctx, hub, 0, 256)
	if err != nil || !found || !bytes.Equal(gotFull, full) {
		t.Errorf("full bundle round-trip err=%v found=%v equal=%v", err, found, bytes.Equal(gotFull, full))
	}
	gotPartial, found, err := s.ReadEntryBundleBlob(ctx, hub, 1, 44)
	if err != nil || !found || !bytes.Equal(gotPartial, partial) {
		t.Errorf("partial bundle round-trip err=%v found=%v equal=%v", err, found, bytes.Equal(gotPartial, partial))
	}
	if f := readBundleIsFull(t, s, hub, 0, 256); f != 1 {
		t.Errorf("bundle is_full for width 256 = %d, want 1", f)
	}
	if f := readBundleIsFull(t, s, hub, 1, 44); f != 0 {
		t.Errorf("bundle is_full for width 44 = %d, want 0", f)
	}
}

// TestReadEntryBundleBlobAbsent confirms an un-written bundle returns found=false,
// nil error.
func TestReadEntryBundleBlobAbsent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	_, found, err := s.ReadEntryBundleBlob(ctx, hub, 7, 256)
	if err != nil {
		t.Fatalf("ReadEntryBundleBlob absent: unexpected error %v", err)
	}
	if found {
		t.Error("absent bundle found=true, want false")
	}
}

// TestLatestCheckpointRawHighestSize confirms LatestCheckpointRaw returns the
// raw bytes of the highest-tree_size checkpoint when several sizes are stored.
func TestLatestCheckpointRawHighestSize(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	rawLow := []byte("checkpoint-at-size-100")
	rawHigh := []byte("checkpoint-at-size-200")
	mustRecordCP(t, s, hub, 200, []byte("root-hi"), rawHigh)
	mustRecordCP(t, s, hub, 100, []byte("root-lo"), rawLow)

	got, found, err := s.LatestCheckpointRaw(ctx, hub)
	if err != nil {
		t.Fatalf("LatestCheckpointRaw: %v", err)
	}
	if !found {
		t.Fatal("LatestCheckpointRaw: not found")
	}
	if !bytes.Equal(got, rawHigh) {
		t.Errorf("LatestCheckpointRaw = %q, want %q (highest tree_size)", got, rawHigh)
	}
}

// TestLatestCheckpointRawAbsent confirms a hub with no checkpoint row returns
// found=false, nil error.
func TestLatestCheckpointRawAbsent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	_, found, err := s.LatestCheckpointRaw(ctx, hub)
	if err != nil {
		t.Fatalf("LatestCheckpointRaw absent: unexpected error %v", err)
	}
	if found {
		t.Error("absent checkpoint found=true, want false")
	}
}

// mustRecordCP records one checkpoint via the public RecordCheckpoint helper.
func mustRecordCP(t *testing.T, s *Store, hubID int64, size uint64, root, raw []byte) {
	t.Helper()
	_, _, err := s.RecordCheckpoint(context.Background(), CheckpointRecord{
		HubID:      hubID,
		TreeSize:   size,
		Root:       root,
		Raw:        raw,
		ObservedAt: time.Unix(1700000000, 0),
	})
	if err != nil {
		t.Fatalf("RecordCheckpoint size %d: %v", size, err)
	}
}
