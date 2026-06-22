// Tests for the typed OTS-table CRUD helpers (RecordOTS, OTSForRoot, PendingOTS,
// MarkOTSStamped, MarkOTSUpgraded, MarkOTSAttempted). Each drives the public method
// against a t.TempDir() database (via openTemp from checkpoints_test.go) and asserts
// on observable rows / return values — never on Store internals. The assertions are
// mutation-targeted: the dedupe id check breaks if DO NOTHING becomes a plain
// insert, the pending-order check breaks if ASC becomes DESC, the pending-filter
// check breaks if the status WHERE is dropped, the back-off-exclusion check breaks if
// MarkOTSAttempted is a no-op or PendingOTS drops the next_retry filter, and the
// stamp check breaks if MarkOTSStamped is a no-op or touches status.
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
	none, err := s.PendingOTS(ctx, time.Unix(1_700_000_000, 0))
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

	got, err := s.PendingOTS(ctx, later)
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

// TestMarkOTSStamped confirms MarkOTSStamped fills ots_bytes / calendar_urls on an
// existing not-yet-stamped pending row, keeps the row pending (status untouched), and
// so leaves it in PendingOTS for the upgrade loop. (Mutation: a no-op MarkOTSStamped
// leaves ots_bytes empty; touching status would drop the row from PendingOTS.)
func TestMarkOTSStamped(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := []byte("stamp-target-root-padding-32byte")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 100, Root: root, Status: OTSStatusPending,
		// OTSBytes empty (the not-yet-stamped sentinel), CalendarURLs empty.
		StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}

	proof := []byte("\x00OpenTimestamps\x00\x00InitialPending")
	const calendars = "https://alice.btc.calendar.opentimestamps.org"
	if err := s.MarkOTSStamped(ctx, hubID, 100, root, proof, calendars); err != nil {
		t.Fatalf("MarkOTSStamped: %v", err)
	}

	got, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot after stamp: found=%v err=%v", found, err)
	}
	if got.Status != OTSStatusPending {
		t.Errorf("Status = %q, want %q (stamp keeps it pending)", got.Status, OTSStatusPending)
	}
	if string(got.OTSBytes) != string(proof) {
		t.Errorf("OTSBytes = %q, want %q", got.OTSBytes, proof)
	}
	if got.CalendarURLs != calendars {
		t.Errorf("CalendarURLs = %q, want %q", got.CalendarURLs, calendars)
	}

	// The stamped row stays in PendingOTS for the upgrade loop.
	pending, err := s.PendingOTS(ctx, time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatalf("PendingOTS after stamp: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("PendingOTS after stamp returned %d rows, want 1 (still pending)", len(pending))
	}
	if string(pending[0].OTSBytes) != string(proof) {
		t.Errorf("pending OTSBytes = %q, want %q", pending[0].OTSBytes, proof)
	}
}

// TestMarkOTSStampedAbsentAndEmptyCalendars confirms MarkOTSStamped is a no-op
// nil-error for an absent (hub, size, root) (like MarkOTSUpgraded it ignores
// RowsAffected) and writes an empty calendarURLs as NULL (the nullStringOrNil
// convention, read back as "").
func TestMarkOTSStampedAbsentAndEmptyCalendars(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Absent row: a no-op, not an error.
	if err := s.MarkOTSStamped(ctx, hubID, 999, []byte("absent-root-padding-32bytes-here"), []byte("x"), "cal"); err != nil {
		t.Fatalf("MarkOTSStamped absent: %v", err)
	}

	// Empty calendarURLs is written as NULL and reads back as "".
	root := []byte("stamp-empty-cal-root-padding-32b")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 100, Root: root, Status: OTSStatusPending, StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}
	if err := s.MarkOTSStamped(ctx, hubID, 100, root, []byte("proof"), ""); err != nil {
		t.Fatalf("MarkOTSStamped empty calendars: %v", err)
	}
	var calendars any
	if err := s.db.QueryRow(
		"SELECT calendar_urls FROM ots WHERE hub_id = ? AND tree_size = ? AND root = ?",
		hubID, 100, root,
	).Scan(&calendars); err != nil {
		t.Fatalf("read calendar_urls: %v", err)
	}
	if calendars != nil {
		t.Errorf("calendar_urls = %v, want NULL (empty → NULL)", calendars)
	}
	got, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot: found=%v err=%v", found, err)
	}
	if got.CalendarURLs != "" {
		t.Errorf("CalendarURLs = %q, want \"\" (NULL → empty)", got.CalendarURLs)
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
	pending, err := s.PendingOTS(ctx, time.Unix(1_700_086_400, 0))
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

// TestMarkOTSAttempted confirms a backed-off retry sets attempts / next_retry,
// keeps the row pending (status untouched), and so excludes it from PendingOTS until
// now reaches next_retry, after which it re-surfaces with the bumped attempts count.
// (Mutation: a no-op MarkOTSAttempted leaves attempts at 0 and next_retry NULL, so
// the row is never excluded — the back-off-exclusion assertion fails.)
func TestMarkOTSAttempted(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := []byte("backoff-target-root-padding-32by")
	stamped := time.Unix(1_700_000_000, 0)
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 100, Root: root, Status: OTSStatusPending, StampedAt: stamped,
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}

	// A freshly-stamped row (next_retry NULL) is immediately due.
	if pending, err := s.PendingOTS(ctx, stamped); err != nil {
		t.Fatalf("PendingOTS before back-off: %v", err)
	} else if len(pending) != 1 {
		t.Fatalf("PendingOTS before back-off returned %d rows, want 1", len(pending))
	}

	nextRetry := stamped.Add(time.Hour)
	if err := s.MarkOTSAttempted(ctx, hubID, 100, root, 1, nextRetry); err != nil {
		t.Fatalf("MarkOTSAttempted: %v", err)
	}

	// The row stays pending with attempts bumped and next_retry set.
	got, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot after back-off: found=%v err=%v", found, err)
	}
	if got.Status != OTSStatusPending {
		t.Errorf("Status = %q, want %q (back-off keeps it pending)", got.Status, OTSStatusPending)
	}
	if got.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", got.Attempts)
	}
	if !got.NextRetry.Equal(nextRetry) {
		t.Errorf("NextRetry = %v, want %v", got.NextRetry, nextRetry)
	}

	// Before next_retry elapses the row is excluded from PendingOTS.
	if pending, err := s.PendingOTS(ctx, nextRetry.Add(-time.Second)); err != nil {
		t.Fatalf("PendingOTS during back-off: %v", err)
	} else if len(pending) != 0 {
		t.Errorf("PendingOTS during back-off returned %d rows, want 0 (backed off)", len(pending))
	}

	// At next_retry (the <= boundary) the row is due again.
	pending, err := s.PendingOTS(ctx, nextRetry)
	if err != nil {
		t.Fatalf("PendingOTS at next_retry: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("PendingOTS at next_retry returned %d rows, want 1 (re-surfaced)", len(pending))
	}
	if pending[0].Attempts != 1 {
		t.Errorf("re-surfaced Attempts = %d, want 1", pending[0].Attempts)
	}

	// A second back-off bumps attempts and pushes next_retry further out.
	nextRetry2 := nextRetry.Add(2 * time.Hour)
	if err := s.MarkOTSAttempted(ctx, hubID, 100, root, pending[0].Attempts+1, nextRetry2); err != nil {
		t.Fatalf("second MarkOTSAttempted: %v", err)
	}
	got2, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot after second back-off: found=%v err=%v", found, err)
	}
	if got2.Attempts != 2 {
		t.Errorf("Attempts after second back-off = %d, want 2", got2.Attempts)
	}
	if !got2.NextRetry.Equal(nextRetry2) {
		t.Errorf("NextRetry after second back-off = %v, want %v", got2.NextRetry, nextRetry2)
	}
}

// TestMarkOTSAttemptedAbsent confirms marking a (hub, size, root) with no row is a
// no-op nil-error (like MarkOTSUpgraded, MarkOTSAttempted ignores RowsAffected).
func TestMarkOTSAttemptedAbsent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	if err := s.MarkOTSAttempted(ctx, hubID, 999, []byte("absent-root-padding-32bytes-here"), 1, time.Unix(1_700_003_600, 0)); err != nil {
		t.Fatalf("MarkOTSAttempted absent: %v", err)
	}
}

// TestMarkOTSAttemptedZeroNextRetryNull confirms a zero nextRetry is written as NULL
// (the unixOrNil convention), so the row immediately re-surfaces via PendingOTS's
// `next_retry IS NULL` leg rather than being permanently backed off.
func TestMarkOTSAttemptedZeroNextRetryNull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := []byte("zero-nextretry-root-padding-32by")
	if _, _, err := s.RecordOTS(ctx, OTSRecord{
		HubID: hubID, TreeSize: 100, Root: root, Status: OTSStatusPending, StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}
	if err := s.MarkOTSAttempted(ctx, hubID, 100, root, 1, time.Time{}); err != nil {
		t.Fatalf("MarkOTSAttempted zero nextRetry: %v", err)
	}

	var nextRetry any
	if err := s.db.QueryRow(
		"SELECT next_retry FROM ots WHERE hub_id = ? AND tree_size = ? AND root = ?",
		hubID, 100, root,
	).Scan(&nextRetry); err != nil {
		t.Fatalf("read next_retry: %v", err)
	}
	if nextRetry != nil {
		t.Errorf("next_retry = %v, want NULL (zero time → NULL)", nextRetry)
	}
	// The NULL row is immediately due again.
	if pending, err := s.PendingOTS(ctx, time.Unix(1_700_000_000, 0)); err != nil {
		t.Fatalf("PendingOTS: %v", err)
	} else if len(pending) != 1 {
		t.Errorf("PendingOTS with NULL next_retry returned %d rows, want 1", len(pending))
	}
}
