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

// tileBytes builds a well-formed hash-tile body for the partial qualifier p: exactly
// widthForP(p) 32-byte hashes, each filled with fill. RecordTile rejects any other
// length, so every tile fixture is sized through this helper rather than by hand.
func tileBytes(p uint8, fill byte) []byte {
	return bytes.Repeat([]byte{fill}, widthForP(p)*sha256.Size)
}

// seedProjection writes one iscc_index row at the FIRST leaf seq of a bundle, which
// is what MirroredFullEntryBundles requires before it will call that bundle
// skippable. Fixtures that want a bundle treated as fully mirrored call this.
func seedProjection(t *testing.T, s *Store, hubID int64, bundleIndex uint64) {
	t.Helper()
	if err := s.RecordProjections(context.Background(), []ProjectionRecord{{
		HubID:      hubID,
		Seq:        bundleIndex * 256,
		IsccID:     "ISCC:SEED",
		NoteSchema: "seed",
	}}); err != nil {
		t.Fatalf("seedProjection bundle %d: %v", bundleIndex, err)
	}
}

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
	if err := s.RecordTile(ctx, hub, 0, 0, 0, data, time.Unix(1700000000, 0)); err != nil {
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
	if err := s.RecordTile(ctx, hub, 0, 1, uint8(44), data, time.Unix(1700000001, 0)); err != nil {
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

	first := tileBytes(100, 0x01)
	second := tileBytes(100, 0x02)
	if err := s.RecordTile(ctx, hub, 0, 0, uint8(100), first, time.Unix(1, 0)); err != nil {
		t.Fatalf("first RecordTile: %v", err)
	}
	if err := s.RecordTile(ctx, hub, 0, 0, uint8(100), second, time.Unix(2, 0)); err != nil {
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
	if err := s.RecordEntryBundle(ctx, hub, 0, 0, full, time.Unix(1700000002, 0)); err != nil {
		t.Fatalf("RecordEntryBundle full: %v", err)
	}
	partial := bytes.Repeat([]byte{0xff}, 512)
	if err := s.RecordEntryBundle(ctx, hub, 1, uint8(44), partial, time.Unix(1700000003, 0)); err != nil {
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

// TestMirroredFullTiles pins the set-shaped read the ingest walk skips on: only
// coords stored at the FULL width appear, partial rows never do, and a coord that
// holds both a stale partial row and a full row is reported once (as full). The
// partial-must-be-absent half is the load-bearing one — a partial that leaked into
// the set would be skipped by the walk and never re-fetched as the tree grew.
func TestMirroredFullTiles(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	// (0,0) full; (0,1) partial only; (0,2) partial first, then promoted to full;
	// (1,0) full at a higher tile-level.
	if err := s.RecordTile(ctx, hub, 0, 0, 0, tileBytes(0, 0x11), time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile full: %v", err)
	}
	if err := s.RecordTile(ctx, hub, 0, 1, 44, tileBytes(44, 0x11), time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile partial: %v", err)
	}
	if err := s.RecordTile(ctx, hub, 0, 2, 44, tileBytes(44, 0x11), time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile partial-then-full: %v", err)
	}
	if err := s.RecordTile(ctx, hub, 0, 2, 0, tileBytes(0, 0x11), time.Unix(2, 0)); err != nil {
		t.Fatalf("RecordTile promoted full: %v", err)
	}
	if err := s.RecordTile(ctx, hub, 1, 0, 0, tileBytes(0, 0x11), time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile level 1 full: %v", err)
	}

	got, err := s.MirroredFullTiles(ctx, hub)
	if err != nil {
		t.Fatalf("MirroredFullTiles: %v", err)
	}
	want := map[TileKey]struct{}{
		{Level: 0, Index: 0}: {},
		{Level: 0, Index: 2}: {},
		{Level: 1, Index: 0}: {},
	}
	if len(got) != len(want) {
		t.Errorf("MirroredFullTiles = %v (%d coords), want %v (%d)", got, len(got), want, len(want))
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("full tile L%d I%d missing from the set", k.Level, k.Index)
		}
	}
	// The partial-only coord must NOT be reported as mirrored-in-full.
	if _, ok := got[TileKey{Level: 0, Index: 1}]; ok {
		t.Error("partial-only tile L0 I1 reported as mirrored in full, want absent (it must stay re-fetchable)")
	}
}

// TestMirroredFullEntryBundles is the entry-bundle twin of TestMirroredFullTiles:
// full indexes appear, a partial-only index does not. It also pins the projection
// half — a full bundle whose iscc_index projection is MISSING must stay out of the
// set, so a legacy or half-written database re-fetches and re-folds it instead of
// skipping it forever with a permanent index gap.
func TestMirroredFullEntryBundles(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	data := bytes.Repeat([]byte{0x22}, 32)
	if err := s.RecordEntryBundle(ctx, hub, 0, 0, data, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordEntryBundle full: %v", err)
	}
	seedProjection(t, s, hub, 0)
	if err := s.RecordEntryBundle(ctx, hub, 1, 44, data, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordEntryBundle partial: %v", err)
	}
	// Bundle 2: mirrored in FULL but never projected — the legacy shape.
	if err := s.RecordEntryBundle(ctx, hub, 2, 0, data, time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordEntryBundle unprojected full: %v", err)
	}

	got, err := s.MirroredFullEntryBundles(ctx, hub)
	if err != nil {
		t.Fatalf("MirroredFullEntryBundles: %v", err)
	}
	if _, ok := got[0]; !ok {
		t.Error("full bundle 0 missing from the set")
	}
	if _, ok := got[1]; ok {
		t.Error("partial-only bundle 1 reported as mirrored in full, want absent")
	}
	if _, ok := got[2]; ok {
		t.Error("full-but-unprojected bundle 2 reported as mirrored in full, want absent (it must stay re-fetchable so its iscc_index gap heals)")
	}
	if len(got) != 1 {
		t.Errorf("MirroredFullEntryBundles = %v, want exactly {0}", got)
	}

	// Once the missing projection lands, bundle 2 becomes skippable like any other.
	seedProjection(t, s, hub, 2)
	healed, err := s.MirroredFullEntryBundles(ctx, hub)
	if err != nil {
		t.Fatalf("MirroredFullEntryBundles after backfill: %v", err)
	}
	if _, ok := healed[2]; !ok {
		t.Error("bundle 2 still absent after its projection landed, want present")
	}
}

// TestRecordTileRejectsWrongLength pins the admission gate: a hash tile body that is
// not exactly width*32 bytes is refused, so a truncated body, an HTML error page, or
// an empty body can never be admitted at full width and then trusted forever by the
// ingest walk's skip.
//
// The width*32 rule holds at EVERY tile level, verified against a live 300258-leaf
// log: the full tiles are 8192 bytes, the level-0 `.p/226` is 7232, the level-1
// `.p/148` is 4736, and the level-2 `.p/4` is 128. The partial cases below are the
// ones that matter for the gate's reach — an upper-level partial carrying the hashes
// of INCOMPLETE subtrees (which no hub publishes) is rejected too.
func TestRecordTileRejectsWrongLength(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hub := newHub(t, s)

	rejected := []struct {
		name         string
		level, index uint64
		p            uint8
		data         []byte
	}{
		{"empty at full width", 0, 0, 0, nil},
		{"truncated full tile", 0, 0, 0, bytes.Repeat([]byte{0x01}, 8191)},
		{"html error page at full width", 0, 0, 0, []byte("<html>502 Bad Gateway</html>")},
		{"over-long full tile", 0, 0, 0, bytes.Repeat([]byte{0x01}, 8193)},
		{"not a whole number of hashes", 0, 0, 0, bytes.Repeat([]byte{0x01}, 8190)},
		{"empty partial", 0, 1, 44, nil},
		{"truncated partial", 0, 1, 44, bytes.Repeat([]byte{0x01}, 44*32-1)},
		{"upper-level partial with an incomplete subtree's hash", 1, 0, 1, bytes.Repeat([]byte{0x01}, 64)},
	}
	for _, tc := range rejected {
		if err := s.RecordTile(ctx, hub, tc.level, tc.index, tc.p, tc.data, time.Unix(1, 0)); err == nil {
			t.Errorf("%s: RecordTile = nil, want a rejection", tc.name)
		}
	}
	if n := countRows(t, s, "tiles"); n != 0 {
		t.Errorf("tiles row count after %d rejected writes = %d, want 0", len(rejected), n)
	}

	// The well-formed body at each of those widths is accepted.
	accepted := []struct {
		name         string
		level, index uint64
		p            uint8
	}{
		{"full tile", 0, 0, 0},
		{"level-0 partial", 0, 1, 44},
		{"level-1 partial", 1, 0, 1},
	}
	for _, tc := range accepted {
		if err := s.RecordTile(ctx, hub, tc.level, tc.index, tc.p, tileBytes(tc.p, 0x01), time.Unix(1, 0)); err != nil {
			t.Errorf("%s: RecordTile with a well-formed body = %v, want accepted", tc.name, err)
		}
	}
}

// TestMirroredFullSetsAreHubScoped proves one hub's mirrored coords never leak into
// another's set — a leak would make the walk skip a coord this hub has never
// fetched, leaving a permanent mirror hole. It also pins the empty-is-not-an-error
// contract for a hub with nothing mirrored.
func TestMirroredFullSetsAreHubScoped(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hubA := newHub(t, s)
	hubB, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub B: %v", err)
	}

	if err := s.RecordTile(ctx, hubA, 0, 7, 0, tileBytes(0, 0x33), time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordTile hubA: %v", err)
	}
	if err := s.RecordEntryBundle(ctx, hubA, 7, 0, bytes.Repeat([]byte{0x33}, 32), time.Unix(1, 0)); err != nil {
		t.Fatalf("RecordEntryBundle hubA: %v", err)
	}
	seedProjection(t, s, hubA, 7)

	tilesB, err := s.MirroredFullTiles(ctx, hubB)
	if err != nil {
		t.Fatalf("MirroredFullTiles hubB: %v", err)
	}
	if len(tilesB) != 0 {
		t.Errorf("MirroredFullTiles(hubB) = %v, want empty (hubA's tiles must not leak)", tilesB)
	}
	bundlesB, err := s.MirroredFullEntryBundles(ctx, hubB)
	if err != nil {
		t.Fatalf("MirroredFullEntryBundles hubB: %v", err)
	}
	if len(bundlesB) != 0 {
		t.Errorf("MirroredFullEntryBundles(hubB) = %v, want empty", bundlesB)
	}
}
