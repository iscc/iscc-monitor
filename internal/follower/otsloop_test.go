// Tests for the OpenTimestamps stamp-then-upgrade loop control core (OTSTick +
// backoff). Each seeds ots rows via the public store.RecordOTS seam, drives OTSTick
// with a fake Stamper + fake Upgrader (confirming / declining / erroring), and
// asserts on observable store read-back (OTSForRoot / PendingOTS) — never on loop
// internals. Both seams are injected so no calendar/Bitcoin network is touched; the
// backoff helper is a pure golden table. Assertions are mutation-targeted: a no-op
// MarkOTSUpgraded leaves the confirmed row pending; a no-op MarkOTSAttempted leaves
// Attempts at 0; reverting the next_retry WHERE clause re-processes a backed-off row
// before its retry elapses; removing the len(r.OTSBytes)==0 stamp branch leaves a
// not-yet-stamped row empty so it can never confirm.
package follower

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/store"
)

// otsNow is a fixed injected "now" for the upgrade-loop tests (no wall-clock).
func otsNow() time.Time { return time.Unix(1_700_000_000, 0) }

// seedPendingOTS stamps one already-stamped pending root for a hub (non-empty
// OTSBytes, so OTSTick skips the stamp branch and exercises the upgrade path) and
// returns its root bytes. The upgrade-path tests pre-stamp so they isolate the
// Upgrader; the new stamp-path test uses seedNotStampedOTS for the empty sentinel.
func seedPendingOTS(t *testing.T, s *store.Store, hubID int64, treeSize uint64, root string) []byte {
	t.Helper()
	rb := []byte(root)
	if _, _, err := s.RecordOTS(context.Background(), store.OTSRecord{
		HubID: hubID, TreeSize: treeSize, Root: rb, Status: store.OTSStatusPending,
		OTSBytes: []byte("pre-stamped-pending-proof-bytes!"), StampedAt: otsNow(),
	}); err != nil {
		t.Fatalf("RecordOTS %q: %v", root, err)
	}
	return rb
}

// seedNotStampedOTS records a pending root with the empty-OTSBytes sentinel the
// follower's poll path writes (the not-yet-stamped state the stamp branch fills in).
func seedNotStampedOTS(t *testing.T, s *store.Store, hubID int64, treeSize uint64, root string) []byte {
	t.Helper()
	rb := []byte(root)
	if _, _, err := s.RecordOTS(context.Background(), store.OTSRecord{
		HubID: hubID, TreeSize: treeSize, Root: rb, Status: store.OTSStatusPending, StampedAt: otsNow(),
	}); err != nil {
		t.Fatalf("RecordOTS %q: %v", root, err)
	}
	return rb
}

// confirmingUpgrader reports every root Bitcoin-confirmed with the given proof/height.
func confirmingUpgrader(proof []byte, height int64) Upgrader {
	return func(_ context.Context, _ store.OTSRecord) (UpgradeResult, error) {
		return UpgradeResult{Confirmed: true, OTSBytes: proof, BTCHeight: height}, nil
	}
}

// decliningUpgrader reports every root still pending (no Bitcoin attestation yet).
func decliningUpgrader() Upgrader {
	return func(_ context.Context, _ store.OTSRecord) (UpgradeResult, error) {
		return UpgradeResult{Confirmed: false}, nil
	}
}

// fakeStamper reports every root stamped with the given proof bytes / calendar URL.
func fakeStamper(proof []byte, calendars string) Stamper {
	return func(_ context.Context, _ [32]byte) ([]byte, string, error) {
		return proof, calendars, nil
	}
}

// noStamper fails the test if invoked: the upgrade-path tests seed pre-stamped rows,
// so the stamp branch must be skipped (len(r.OTSBytes) != 0).
func noStamper(t *testing.T) Stamper {
	t.Helper()
	return func(_ context.Context, _ [32]byte) ([]byte, string, error) {
		t.Errorf("Stamper invoked for an already-stamped row, want it skipped")
		return nil, "", nil
	}
}

// TestOTSTickConfirms drives OTSTick over a seeded pending root with a confirming
// Upgrader and asserts the row reads back confirmed with the returned proof/height
// and drops out of PendingOTS. (Mutation: a no-op MarkOTSUpgraded leaves it pending.)
func TestOTSTickConfirms(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := seedPendingOTS(t, s, hubID, 100, "ots-confirm-target-root-pad-32by")

	proof := []byte("bitcoin-confirmed-ots-proof-blob")
	const height = int64(870_123)
	if err := OTSTick(ctx, s, noStamper(t), confirmingUpgrader(proof, height), otsNow(), nil); err != nil {
		t.Fatalf("OTSTick: %v", err)
	}

	got, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot: found=%v err=%v", found, err)
	}
	if got.Status != store.OTSStatusConfirmed {
		t.Errorf("Status = %q, want %q", got.Status, store.OTSStatusConfirmed)
	}
	if string(got.OTSBytes) != string(proof) {
		t.Errorf("OTSBytes = %q, want %q", got.OTSBytes, proof)
	}
	if got.BTCHeight != height {
		t.Errorf("BTCHeight = %d, want %d", got.BTCHeight, height)
	}
	if !got.UpgradedAt.Equal(otsNow()) {
		t.Errorf("UpgradedAt = %v, want %v (injected now)", got.UpgradedAt, otsNow())
	}

	pending, err := s.PendingOTS(ctx, otsNow().Add(time.Hour))
	if err != nil {
		t.Fatalf("PendingOTS: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("PendingOTS after confirm = %d rows, want 0 (confirmed drops out)", len(pending))
	}
}

// TestOTSTickDeclinesBacksOff drives a declining Upgrader and asserts the row stays
// pending with Attempts==1 and a future NextRetry, is excluded from PendingOTS at the
// same now, and re-surfaces at now+backoff where a second OTSTick bumps Attempts to 2.
// (Mutation: a no-op MarkOTSAttempted leaves Attempts at 0; reverting the next_retry
// WHERE clause re-processes the row before back-off elapses.)
func TestOTSTickDeclinesBacksOff(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := seedPendingOTS(t, s, hubID, 100, "ots-decline-target-root-pad-32by")

	now := otsNow()
	if err := OTSTick(ctx, s, noStamper(t), decliningUpgrader(), now, nil); err != nil {
		t.Fatalf("OTSTick (decline): %v", err)
	}

	got, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot: found=%v err=%v", found, err)
	}
	if got.Status != store.OTSStatusPending {
		t.Errorf("Status = %q, want %q (declined stays pending)", got.Status, store.OTSStatusPending)
	}
	if got.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", got.Attempts)
	}
	wantRetry := now.Add(backoff(1))
	if !got.NextRetry.Equal(wantRetry) {
		t.Errorf("NextRetry = %v, want %v (now+backoff(1))", got.NextRetry, wantRetry)
	}

	// At the same now the backed-off row is excluded, so a second OTSTick is a no-op
	// (Attempts stays 1).
	if err := OTSTick(ctx, s, noStamper(t), decliningUpgrader(), now, nil); err != nil {
		t.Fatalf("OTSTick (same now): %v", err)
	}
	got, _, err = s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil {
		t.Fatalf("OTSForRoot (same now): %v", err)
	}
	if got.Attempts != 1 {
		t.Errorf("Attempts after re-tick at same now = %d, want 1 (still backed off)", got.Attempts)
	}

	// At now+backoff the row re-surfaces and a third OTSTick bumps Attempts to 2.
	later := wantRetry
	if err := OTSTick(ctx, s, noStamper(t), decliningUpgrader(), later, nil); err != nil {
		t.Fatalf("OTSTick (after backoff): %v", err)
	}
	got, _, err = s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil {
		t.Fatalf("OTSForRoot (after backoff): %v", err)
	}
	if got.Attempts != 2 {
		t.Errorf("Attempts after backoff re-tick = %d, want 2", got.Attempts)
	}
	if !got.NextRetry.Equal(later.Add(backoff(2))) {
		t.Errorf("NextRetry = %v, want %v (later+backoff(2))", got.NextRetry, later.Add(backoff(2)))
	}
}

// TestOTSTickErrorBacksOffAndContinues drives an Upgrader that errors on the first
// (oldest) row and confirms the second, asserting the pass does NOT abort: the
// erroring row gets a back-off (Attempts==1, still pending) AND the second row is
// processed (confirmed), while OTSTick returns the wrapped first error.
func TestOTSTickErrorBacksOffAndContinues(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	// Two already-stamped pending roots (non-empty OTSBytes, so the stamp branch is
	// skipped and the upgrade path runs); PendingOTS orders oldest-first by stamped_at,
	// so the earlier-stamped root (size 100) is processed first.
	stamped := []byte("pre-stamped-pending-proof-bytes!")
	firstRoot := []byte("ots-error-first-root-padding-32b")
	if _, _, err := s.RecordOTS(ctx, store.OTSRecord{
		HubID: hubID, TreeSize: 100, Root: firstRoot, Status: store.OTSStatusPending,
		OTSBytes: stamped, StampedAt: otsNow(),
	}); err != nil {
		t.Fatalf("RecordOTS first: %v", err)
	}
	secondRoot := []byte("ots-error-second-root-paddng-32b")
	if _, _, err := s.RecordOTS(ctx, store.OTSRecord{
		HubID: hubID, TreeSize: 200, Root: secondRoot, Status: store.OTSStatusPending,
		OTSBytes: stamped, StampedAt: otsNow().Add(time.Second),
	}); err != nil {
		t.Fatalf("RecordOTS second: %v", err)
	}

	transportFault := errors.New("calendar unreachable")
	proof := []byte("second-row-confirmed-proof-blobs")
	up := func(_ context.Context, r store.OTSRecord) (UpgradeResult, error) {
		if r.TreeSize == 100 {
			return UpgradeResult{}, transportFault
		}
		return UpgradeResult{Confirmed: true, OTSBytes: proof, BTCHeight: 42}, nil
	}

	err = OTSTick(ctx, s, noStamper(t), up, otsNow(), nil)
	if err == nil {
		t.Fatalf("OTSTick returned nil, want the wrapped transport fault")
	}
	if !errors.Is(err, transportFault) {
		t.Errorf("OTSTick error = %v, want it to wrap %v", err, transportFault)
	}

	// The erroring row got a back-off (still pending, Attempts==1).
	first, found, err := s.OTSForRoot(ctx, hubID, 100, firstRoot)
	if err != nil || !found {
		t.Fatalf("OTSForRoot first: found=%v err=%v", found, err)
	}
	if first.Status != store.OTSStatusPending {
		t.Errorf("first Status = %q, want %q (errored stays pending)", first.Status, store.OTSStatusPending)
	}
	if first.Attempts != 1 {
		t.Errorf("first Attempts = %d, want 1 (back-off recorded)", first.Attempts)
	}

	// The pass continued: the second row was confirmed despite the first's fault.
	second, found, err := s.OTSForRoot(ctx, hubID, 200, secondRoot)
	if err != nil || !found {
		t.Fatalf("OTSForRoot second: found=%v err=%v", found, err)
	}
	if second.Status != store.OTSStatusConfirmed {
		t.Errorf("second Status = %q, want %q (pass not aborted)", second.Status, store.OTSStatusConfirmed)
	}
	if string(second.OTSBytes) != string(proof) {
		t.Errorf("second OTSBytes = %q, want %q", second.OTSBytes, proof)
	}
}

// TestOTSStampThenUpgrade drives the full stamp-then-upgrade arc over a row the poll
// path wrote with the empty-OTSBytes sentinel: the first OTSTick stamps it (fills
// OTSBytes/CalendarURLs via the fake Stamper, row stays pending), the second OTSTick
// upgrades the now-stamped row to confirmed. This is the increment that finally lets
// a root transit pending -> Bitcoin-confirmed end-to-end. (Mutation: removing the
// len(r.OTSBytes)==0 stamp branch leaves OTSBytes empty, so the row never confirms.)
func TestOTSStampThenUpgrade(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := seedNotStampedOTS(t, s, hubID, 100, "ots-stamp-target-root-padding32")

	// First pass: the not-yet-stamped row gets its calendar proof; it stays pending.
	stampedProof := []byte("calendar-pending-ots-proof-blob!")
	const calendars = "https://alice.btc.calendar.opentimestamps.org"
	// confirmingUpgrader would confirm, but the stamp branch continues before reaching
	// it on this pass — so a stamp-then-skip-upgrade leaves the row pending, not confirmed.
	if err := OTSTick(ctx, s, fakeStamper(stampedProof, calendars), confirmingUpgrader([]byte("ignored"), 1), otsNow(), nil); err != nil {
		t.Fatalf("OTSTick (stamp): %v", err)
	}
	got, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot after stamp: found=%v err=%v", found, err)
	}
	if got.Status != store.OTSStatusPending {
		t.Errorf("Status after stamp = %q, want %q (stamp keeps it pending)", got.Status, store.OTSStatusPending)
	}
	if string(got.OTSBytes) != string(stampedProof) {
		t.Errorf("OTSBytes after stamp = %q, want %q", got.OTSBytes, stampedProof)
	}
	if got.CalendarURLs != calendars {
		t.Errorf("CalendarURLs after stamp = %q, want %q", got.CalendarURLs, calendars)
	}

	// Second pass: the now-stamped row reaches the Upgrader and confirms.
	confirmedProof := []byte("bitcoin-confirmed-upgraded-proof!")
	const height = int64(880_456)
	if err := OTSTick(ctx, s, fakeStamper(stampedProof, calendars), confirmingUpgrader(confirmedProof, height), otsNow(), nil); err != nil {
		t.Fatalf("OTSTick (upgrade): %v", err)
	}
	got, found, err = s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot after upgrade: found=%v err=%v", found, err)
	}
	if got.Status != store.OTSStatusConfirmed {
		t.Errorf("Status after upgrade = %q, want %q (pending -> confirmed)", got.Status, store.OTSStatusConfirmed)
	}
	if string(got.OTSBytes) != string(confirmedProof) {
		t.Errorf("OTSBytes after upgrade = %q, want %q", got.OTSBytes, confirmedProof)
	}
	if got.BTCHeight != height {
		t.Errorf("BTCHeight after upgrade = %d, want %d", got.BTCHeight, height)
	}
}

// TestOTSStampBacksOff drives a Stamper transport fault over a not-yet-stamped row and
// asserts the row stays pending and un-stamped (empty OTSBytes) with a back-off
// recorded (Attempts==1, future NextRetry), the pass does NOT abort, and the Upgrader
// is never invoked for the still-unstamped row. A stamp fault is best-effort: a
// back-off, never a freeze, never an abort (OTS never blocks the follower, ADR-0004).
func TestOTSStampBacksOff(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	root := seedNotStampedOTS(t, s, hubID, 100, "ots-stamp-fault-root-padding-32")

	stampFault := errors.New("calendar unreachable")
	stamper := func(_ context.Context, _ [32]byte) ([]byte, string, error) {
		return nil, "", stampFault
	}
	upCalled := false
	up := func(_ context.Context, _ store.OTSRecord) (UpgradeResult, error) {
		upCalled = true
		return UpgradeResult{}, nil
	}

	now := otsNow()
	err = OTSTick(ctx, s, stamper, up, now, nil)
	if err == nil {
		t.Fatalf("OTSTick returned nil, want the wrapped stamp fault")
	}
	if !errors.Is(err, stampFault) {
		t.Errorf("OTSTick error = %v, want it to wrap %v", err, stampFault)
	}
	if upCalled {
		t.Errorf("Upgrader invoked for an unstamped row, want the stamp branch to continue first")
	}

	got, found, err := s.OTSForRoot(ctx, hubID, 100, root)
	if err != nil || !found {
		t.Fatalf("OTSForRoot: found=%v err=%v", found, err)
	}
	if got.Status != store.OTSStatusPending {
		t.Errorf("Status = %q, want %q (stamp fault stays pending)", got.Status, store.OTSStatusPending)
	}
	if len(got.OTSBytes) != 0 {
		t.Errorf("OTSBytes = %q, want empty (stamp fault leaves it un-stamped)", got.OTSBytes)
	}
	if got.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1 (back-off recorded)", got.Attempts)
	}
	if !got.NextRetry.Equal(now.Add(backoff(1))) {
		t.Errorf("NextRetry = %v, want %v (now+backoff(1))", got.NextRetry, now.Add(backoff(1)))
	}
}

// TestOTSTickNoPending confirms OTSTick over an empty (or fully backed-off) index is
// a clean nil-error no-op that never invokes the Upgrader.
func TestOTSTickNoPending(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	if _, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id"); err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	called := false
	up := func(_ context.Context, _ store.OTSRecord) (UpgradeResult, error) {
		called = true
		return UpgradeResult{}, nil
	}
	// nil Stamper here also exercises the nil-tolerant skip (no panic over an empty index).
	if err := OTSTick(ctx, s, nil, up, otsNow(), nil); err != nil {
		t.Fatalf("OTSTick over empty index: %v", err)
	}
	if called {
		t.Errorf("Upgrader invoked with no pending rows, want it skipped")
	}
}

// TestOTSBackoff is the golden table for the pure backoff helper: capped exponential,
// monotonic up to the shift cap, then clamped to otsBackoffMax.
func TestOTSBackoff(t *testing.T) {
	cases := []struct {
		attempts int64
		want     time.Duration
	}{
		{0, otsBackoffBase},  // non-positive treated as 1
		{1, otsBackoffBase},  // 1h
		{2, 2 * time.Hour},   // base << 1
		{3, 4 * time.Hour},   // base << 2
		{4, 8 * time.Hour},   // base << 3
		{5, 16 * time.Hour},  // base << 4
		{6, otsBackoffMax},   // base << 5 = 32h, clamped to 24h
		{100, otsBackoffMax}, // shift cap + max clamp
	}
	var prev time.Duration
	for _, c := range cases {
		got := backoff(c.attempts)
		if got != c.want {
			t.Errorf("backoff(%d) = %v, want %v", c.attempts, got, c.want)
		}
		if got < prev {
			t.Errorf("backoff(%d) = %v < previous %v (not monotonic)", c.attempts, got, prev)
		}
		prev = got
	}
}
