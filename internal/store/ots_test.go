// Tests for the typed OTS-table CRUD helpers (RecordOTS, OTSForRoot, PendingOTS,
// MarkOTSUpgraded). Each drives the public method against a t.TempDir() database
// (via openTemp from checkpoints_test.go) and asserts on observable rows / return
// values — never on Store internals. The assertions are mutation-targeted: the
// dedupe id check breaks if DO NOTHING becomes a plain insert, the pending-order
// check breaks if ASC becomes DESC, and the pending-filter check breaks if the
// status WHERE is dropped.
package store

import (
	"context"
	"testing"
	"time"
)

// TestRecordOTSRoundTrip confirms a stamped root round-trips through RecordOTS →
// OTSForRoot with every stored field byte/value-equal (Status, OTSBytes,
// CalendarURLs, BTCHeight, and the times back as set), and that the first stamp
// reports inserted=true.
func TestRecordOTSRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	stamped := time.Unix(1_700_000_000, 0)
	nextRetry := time.Unix(1_700_003_600, 0)
	rec := OTSRecord{
		HubID:        hubID,
		TreeSize:     42,
		Root:         []byte("ots-round-trip-root-padding-32by"),
		Status:       OTSStatusPending,
		OTSBytes:     []byte("\x00OpenTimestamps\x00\x00Proof"),
		CalendarURLs: "https://alice.btc.calendar.opentimestamps.org",
		StampedAt:    stamped,
		BTCHeight:    0,
		Attempts:     1,
		NextRetry:    nextRetry,
	}

	id, inserted, err := s.RecordOTS(ctx, rec)
	if err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}
	if !inserted {
		t.Errorf("first RecordOTS inserted = false, want true")
	}
	if id <= 0 {
		t.Errorf("RecordOTS id = %d, want > 0", id)
	}

	got, found, err := s.OTSForRoot(ctx, hubID, rec.TreeSize, rec.Root)
	if err != nil {
		t.Fatalf("OTSForRoot: %v", err)
	}
	if !found {
		t.Fatalf("found = false for a stamped root, want true")
	}
	if got.HubID != hubID {
		t.Errorf("HubID = %d, want %d", got.HubID, hubID)
	}
	if got.TreeSize != rec.TreeSize {
		t.Errorf("TreeSize = %d, want %d", got.TreeSize, rec.TreeSize)
	}
	if string(got.Root) != string(rec.Root) {
		t.Errorf("Root = %q, want %q", got.Root, rec.Root)
	}
	if got.Status != OTSStatusPending {
		t.Errorf("Status = %q, want %q", got.Status, OTSStatusPending)
	}
	if string(got.OTSBytes) != string(rec.OTSBytes) {
		t.Errorf("OTSBytes = %q, want %q", got.OTSBytes, rec.OTSBytes)
	}
	if got.CalendarURLs != rec.CalendarURLs {
		t.Errorf("CalendarURLs = %q, want %q", got.CalendarURLs, rec.CalendarURLs)
	}
	if got.BTCHeight != rec.BTCHeight {
		t.Errorf("BTCHeight = %d, want %d", got.BTCHeight, rec.BTCHeight)
	}
	if got.Attempts != rec.Attempts {
		t.Errorf("Attempts = %d, want %d", got.Attempts, rec.Attempts)
	}
	if !got.StampedAt.Equal(stamped) {
		t.Errorf("StampedAt = %v, want %v", got.StampedAt, stamped)
	}
	if !got.NextRetry.Equal(nextRetry) {
		t.Errorf("NextRetry = %v, want %v", got.NextRetry, nextRetry)
	}
	// A pending root has not been upgraded: UpgradedAt is NULL → zero time.
	if !got.UpgradedAt.IsZero() {
		t.Errorf("UpgradedAt = %v, want zero (not yet upgraded)", got.UpgradedAt)
	}
}

// TestOTSForRootAbsent confirms an un-anchored (hub, size, root) returns
// found=false with a nil error (a plain miss, not a 5xx) — the convention
// certificate §5 / the .ots route render the "not-yet-anchored" state from.
func TestOTSForRootAbsent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	got, found, err := s.OTSForRoot(ctx, hubID, 99, []byte("never-stamped-root-padding-32byt"))
	if err != nil {
		t.Fatalf("OTSForRoot absent: %v", err)
	}
	if found {
		t.Errorf("found = true for an un-anchored root, want false")
	}
	if got.HubID != 0 || got.TreeSize != 0 || got.Root != nil || got.Status != "" {
		t.Errorf("OTSRecord = %+v, want zero value", got)
	}

	// An absent hub is likewise a miss, not an error.
	_, found, err = s.OTSForRoot(ctx, 999, 42, []byte("ots-round-trip-root-padding-32by"))
	if err != nil {
		t.Fatalf("OTSForRoot absent hub: %v", err)
	}
	if found {
		t.Errorf("found = true for an absent hub, want false")
	}
}

// TestRecordOTSDedupes confirms a re-stamp of the same (hub, tree_size, root)
// returns the existing id with inserted=false and leaves exactly one ots row — the
// UNIQUE(hub, tree_size, root) "stamp each distinct root once" guarantee. (Mutation:
// swapping DO NOTHING for a plain insert breaks the same-id check and the row count.)
func TestRecordOTSDedupes(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	rec := OTSRecord{
		HubID:     hubID,
		TreeSize:  42,
		Root:      []byte("ots-dedupe-root-padding-32-bytes"),
		Status:    OTSStatusPending,
		StampedAt: time.Unix(1_700_000_000, 0),
	}

	id1, inserted, err := s.RecordOTS(ctx, rec)
	if err != nil {
		t.Fatalf("first RecordOTS: %v", err)
	}
	if !inserted {
		t.Errorf("first RecordOTS inserted = false, want true")
	}

	id2, inserted, err := s.RecordOTS(ctx, rec)
	if err != nil {
		t.Fatalf("re-stamp RecordOTS: %v", err)
	}
	if inserted {
		t.Errorf("re-stamp RecordOTS inserted = true, want false")
	}
	if id2 != id1 {
		t.Errorf("re-stamp id = %d, want %d (same)", id2, id1)
	}
	if n := countRows(t, s, "ots"); n != 1 {
		t.Errorf("ots row count = %d, want 1", n)
	}

	// A different root for the same hub is a distinct, recorded stamp.
	rec2 := rec
	rec2.TreeSize = 43
	rec2.Root = []byte("ots-dedupe-root2-padding-32-byte")
	_, inserted, err = s.RecordOTS(ctx, rec2)
	if err != nil {
		t.Fatalf("RecordOTS distinct root: %v", err)
	}
	if !inserted {
		t.Errorf("distinct (size, root) inserted = false, want true")
	}
	if n := countRows(t, s, "ots"); n != 2 {
		t.Errorf("ots row count = %d, want 2", n)
	}
}

// TestRecordOTSZeroTimesNull confirms a zero StampedAt / NextRetry is written as
// NULL (not the unix epoch) and an empty CalendarURLs as NULL — the unixOrNil /
// nullStringOrNil convention, with OTSForRoot reading them back as the zero value.
func TestRecordOTSZeroTimesNull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := []byte("ots-zero-times-root-padding-32by")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID:    hubID,
		TreeSize: 7,
		Root:     root,
		Status:   OTSStatusPending,
		// StampedAt / NextRetry zero, CalendarURLs empty.
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}

	var stamped, nextRetry, calendars any
	if err := s.db.QueryRow(
		"SELECT stamped_at, next_retry, calendar_urls FROM ots WHERE hub_id = ? AND tree_size = ? AND root = ?",
		hubID, 7, root,
	).Scan(&stamped, &nextRetry, &calendars); err != nil {
		t.Fatalf("read ots columns: %v", err)
	}
	if stamped != nil || nextRetry != nil || calendars != nil {
		t.Errorf("(stamped_at, next_retry, calendar_urls) = (%v, %v, %v), want all NULL", stamped, nextRetry, calendars)
	}

	got, found, err := s.OTSForRoot(ctx, hubID, 7, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot: found=%v err=%v", found, err)
	}
	if !got.StampedAt.IsZero() || !got.NextRetry.IsZero() {
		t.Errorf("times = (%v, %v), want zero (NULL → zero time)", got.StampedAt, got.NextRetry)
	}
	if got.CalendarURLs != "" {
		t.Errorf("CalendarURLs = %q, want \"\" (NULL → empty)", got.CalendarURLs)
	}
}

// TestPendingOTS confirms PendingOTS returns only pending-status rows oldest-first
// (ORDER BY stamped_at ASC) and excludes a confirmed row. (Mutations: ASC → DESC
// flips the order; dropping the status WHERE includes the confirmed row.)
func TestPendingOTS(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// An empty index returns an empty slice and a nil error.
	none, err := s.PendingOTS(ctx)
	if err != nil {
		t.Fatalf("PendingOTS (none): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("PendingOTS with no rows = %v, want empty", none)
	}

	earlier := time.Unix(1_700_000_000, 0)
	later := time.Unix(1_700_009_999, 0)
	confirmedTime := time.Unix(1_700_005_000, 0)

	// A later-stamped pending root, an earlier-stamped pending root, and a
	// confirmed root the filter must exclude.
	laterRoot := []byte("pending-later-root-padding-32byt")
	earlierRoot := []byte("pending-earlier-root-padding-32b")
	confirmedRoot := []byte("pending-confirmed-root-paddng-32")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 200, Root: laterRoot, Status: OTSStatusPending, StampedAt: later,
	}); err != nil {
		t.Fatalf("RecordOTS later: %v", err)
	}
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 100, Root: earlierRoot, Status: OTSStatusPending, StampedAt: earlier,
	}); err != nil {
		t.Fatalf("RecordOTS earlier: %v", err)
	}
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 300, Root: confirmedRoot, Status: OTSStatusConfirmed, StampedAt: confirmedTime,
	}); err != nil {
		t.Fatalf("RecordOTS confirmed: %v", err)
	}

	got, err := s.PendingOTS(ctx)
	if err != nil {
		t.Fatalf("PendingOTS: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("PendingOTS returned %d rows, want 2 (confirmed excluded)", len(got))
	}
	// Oldest-first: the earlier-stamped pending root precedes the later one.
	if string(got[0].Root) != string(earlierRoot) {
		t.Errorf("got[0].Root = %q, want %q (oldest-first)", got[0].Root, earlierRoot)
	}
	if string(got[1].Root) != string(laterRoot) {
		t.Errorf("got[1].Root = %q, want %q (oldest-first)", got[1].Root, laterRoot)
	}
	for i, r := range got {
		if r.Status != OTSStatusPending {
			t.Errorf("got[%d].Status = %q, want %q (only pending rows)", i, r.Status, OTSStatusPending)
		}
		if string(r.Root) == string(confirmedRoot) {
			t.Errorf("got[%d] is the confirmed root, want it excluded", i)
		}
	}
	// The earlier row's stamped_at round-trips through the list read.
	if !got[0].StampedAt.Equal(earlier) {
		t.Errorf("got[0].StampedAt = %v, want %v", got[0].StampedAt, earlier)
	}
}

// TestMarkOTSUpgraded confirms MarkOTSUpgraded flips a pending row to confirmed,
// sets ots_bytes / btc_height / upgraded_at, and drops it out of PendingOTS, while
// leaving another still-pending root in the list.
func TestMarkOTSUpgraded(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	upgradeRoot := []byte("upgrade-target-root-padding-32by")
	stayRoot := []byte("stays-pending-root-padding-32byt")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 100, Root: upgradeRoot, Status: OTSStatusPending,
		OTSBytes: []byte("pending-proof"), StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS upgrade target: %v", err)
	}
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 200, Root: stayRoot, Status: OTSStatusPending,
		StampedAt: time.Unix(1_700_009_999, 0),
	}); err != nil {
		t.Fatalf("RecordOTS stays-pending: %v", err)
	}

	upgradedProof := []byte("bitcoin-confirmed-ots-proof-blob")
	upgradedAt := time.Unix(1_700_086_400, 0)
	const btcHeight = int64(870_123)
	if err := s.MarkOTSUpgraded(ctx, hubID, 100, upgradeRoot, upgradedProof, btcHeight, upgradedAt); err != nil {
		t.Fatalf("MarkOTSUpgraded: %v", err)
	}

	// The upgraded row now reads back confirmed with the new proof / height / time.
	got, found, err := s.OTSForRoot(ctx, hubID, 100, upgradeRoot)
	if err != nil || !found {
		t.Fatalf("OTSForRoot upgraded: found=%v err=%v", found, err)
	}
	if got.Status != OTSStatusConfirmed {
		t.Errorf("Status = %q, want %q", got.Status, OTSStatusConfirmed)
	}
	if string(got.OTSBytes) != string(upgradedProof) {
		t.Errorf("OTSBytes = %q, want %q (upgraded proof)", got.OTSBytes, upgradedProof)
	}
	if got.BTCHeight != btcHeight {
		t.Errorf("BTCHeight = %d, want %d", got.BTCHeight, btcHeight)
	}
	if !got.UpgradedAt.Equal(upgradedAt) {
		t.Errorf("UpgradedAt = %v, want %v", got.UpgradedAt, upgradedAt)
	}

	// The upgraded root drops out of PendingOTS; the other pending root remains.
	pending, err := s.PendingOTS(ctx)
	if err != nil {
		t.Fatalf("PendingOTS after upgrade: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("PendingOTS returned %d rows, want 1 (upgraded row excluded)", len(pending))
	}
	if string(pending[0].Root) != string(stayRoot) {
		t.Errorf("remaining pending root = %q, want %q", pending[0].Root, stayRoot)
	}
}

// TestMarkOTSUpgradedIdempotent confirms re-marking an already-confirmed root is a
// no-op nil-error (like SetCoverage, MarkOTSUpgraded ignores RowsAffected), and
// marking an absent (hub, size, root) is likewise not an error.
func TestMarkOTSUpgradedIdempotent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := []byte("idempotent-upgrade-root-pad-32by")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 100, Root: root, Status: OTSStatusPending,
		StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}

	upgradedAt := time.Unix(1_700_086_400, 0)
	proof := []byte("confirmed-proof")
	if err := s.MarkOTSUpgraded(ctx, hubID, 100, root, proof, 870_123, upgradedAt); err != nil {
		t.Fatalf("first MarkOTSUpgraded: %v", err)
	}
	// Re-mark the already-confirmed row: a no-op, not an error.
	if err := s.MarkOTSUpgraded(ctx, hubID, 100, root, proof, 870_123, upgradedAt); err != nil {
		t.Fatalf("re-MarkOTSUpgraded: %v", err)
	}

	// Marking a (hub, size, root) with no row is likewise not an error.
	if err := s.MarkOTSUpgraded(ctx, hubID, 999, []byte("absent-root-padding-32bytes-here"), proof, 1, upgradedAt); err != nil {
		t.Fatalf("MarkOTSUpgraded absent: %v", err)
	}
}
