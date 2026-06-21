// Tests for the poll-loop wrapper (Loop.Tick / due). They drive Tick and the pure
// due() predicate directly with an injected now — never a wall-clock sleep and
// never Loop.Run's ticker — so the cadence is fully deterministic. The store seam
// and the offline composite fetcher are reused verbatim from follower_test.go
// (sb0VerifiedFetcher, openTemp, countRows, assertViolation, sb0ObservedAt,
// noopAlert), so the loop is tested over the real sb0 fixtures with no live
// network. Assertions are on observable store outputs (FollowState, row counts)
// only, never on Loop internals.
package follower

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/store"
)

// TestDue is the golden table for the pure due() back-off predicate: a zero
// lastPoll is always due; an unfrozen hub polled less than Normal ago is not due
// but is due at/after Normal; a frozen hub polled past Normal but less than the
// longer Frozen interval is NOT due (the back-off) but is due at/after Frozen.
func TestDue(t *testing.T) {
	const (
		normal = 10 * time.Minute
		frozen = time.Hour
	)
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		frozen   bool
		lastPoll time.Time
		want     bool
	}{
		{"never polled is always due", false, time.Time{}, true},
		{"never polled is due even when frozen", true, time.Time{}, true},
		{"unfrozen polled just now is not due", false, now.Add(-time.Minute), false},
		{"unfrozen polled under normal is not due", false, now.Add(-(normal - time.Second)), false},
		{"unfrozen polled exactly normal ago is due", false, now.Add(-normal), true},
		{"unfrozen polled past normal is due", false, now.Add(-2 * normal), true},
		{"frozen polled past normal but under frozen is not due (back-off)", true, now.Add(-(normal + time.Minute)), false},
		{"frozen polled just under frozen is not due", true, now.Add(-(frozen - time.Second)), false},
		{"frozen polled exactly frozen ago is due", true, now.Add(-frozen), true},
		{"frozen polled past frozen is due", true, now.Add(-2 * frozen), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := due(tc.frozen, tc.lastPoll, now, normal, frozen); got != tc.want {
				t.Errorf("due(frozen=%v, lastPoll=%v) = %v, want %v", tc.frozen, tc.lastPoll, got, tc.want)
			}
		})
	}
}

// TestTick drives one pass over two registered clean hubs: a Tick at now polls
// each due hub through PollHub and advances its FollowState.LastSize to the sb0
// fixture size (10183). A second Tick at the *same* now must not re-poll either
// hub — the cursor is unchanged and no new checkpoints rows are written — proving
// the loop's in-memory lastPoll throttles to the cadence.
func TestTick(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hubA, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub A: %v", err)
	}
	hubB, err := s.UpsertHub(ctx, "sb1.iscc.id", "sb1.iscc.id/log", "https://sb1.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub B: %v", err)
	}

	loop := &Loop{
		Store:   s,
		Fetcher: sb0VerifiedFetcher(t),
		Targets: []HubTarget{
			{HubID: hubA, BaseURL: "https://sb0.iscc.id"},
			{HubID: hubB, BaseURL: "https://sb0.iscc.id"},
		},
		Normal: 10 * time.Minute,
		Frozen: time.Hour,
		Alert:  noopAlert,
	}

	now := sb0ObservedAt()
	if err := loop.Tick(ctx, now); err != nil {
		t.Fatalf("first Tick: %v", err)
	}
	for _, hubID := range []int64{hubA, hubB} {
		fs, err := s.FollowState(ctx, hubID)
		if err != nil {
			t.Fatalf("FollowState hub %d: %v", hubID, err)
		}
		if fs.LastSize != 10183 {
			t.Errorf("hub %d LastSize = %d, want 10183 (advanced by Tick)", hubID, fs.LastSize)
		}
	}
	rowsAfterFirst := countRows(t, path, "checkpoints")

	// A second Tick at the SAME now: neither hub is due (both polled this instant),
	// so nothing changes — no new checkpoints, no cursor movement.
	if err := loop.Tick(ctx, now); err != nil {
		t.Fatalf("second Tick: %v", err)
	}
	if n := countRows(t, path, "checkpoints"); n != rowsAfterFirst {
		t.Errorf("checkpoints after re-Tick = %d, want %d (not due, must not re-poll)", n, rowsAfterFirst)
	}
	for _, hubID := range []int64{hubA, hubB} {
		fs, err := s.FollowState(ctx, hubID)
		if err != nil {
			t.Fatalf("FollowState hub %d after re-Tick: %v", hubID, err)
		}
		if fs.LastSize != 10183 {
			t.Errorf("hub %d LastSize = %d after re-Tick, want 10183 (unchanged)", hubID, fs.LastSize)
		}
	}
}

// TestTickFrozenUnaffected proves the backed-off evidence-only re-poll: hub A is
// seeded to freeze on its next poll (a shrink, prior size 20000 > sb0 fixture
// 10183) and hub B is clean. The first Tick freezes hub A (1 violation, 1 alert)
// and advances hub B to 10183. A Tick past Normal but before Frozen does NOT
// re-poll the frozen hub (no new violation) while hub B still re-polls. A Tick at
// Frozen re-polls hub A, recording a second violation as evidence WITHOUT a new
// alert, never clearing Frozen, and hub B stays advanced — other hubs unaffected.
func TestTickFrozenUnaffected(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hubA, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub A: %v", err)
	}
	hubB, err := s.UpsertHub(ctx, "sb1.iscc.id", "sb1.iscc.id/log", "https://sb1.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub B: %v", err)
	}

	// Seed hub A's prior accepted checkpoint strictly larger than the fixture, so
	// the next observation at 10183 is a shrink that freezes it.
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubA,
		Status:     "verified",
		TreeSize:   20000,
		Root:       []byte("shrink-seed-root-distinct-pad-32"),
		Raw:        []byte("sb0.iscc.id/log\n20000\nseed-root\n"),
		ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("seed RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubA, 20000); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}

	var alerts int
	loop := &Loop{
		Store:   s,
		Fetcher: sb0VerifiedFetcher(t),
		Targets: []HubTarget{
			{HubID: hubA, BaseURL: "https://sb0.iscc.id"},
			{HubID: hubB, BaseURL: "https://sb0.iscc.id"},
		},
		Normal: 10 * time.Minute,
		Frozen: time.Hour,
		Alert:  func(int64, string) { alerts++ },
	}

	t0 := sb0ObservedAt()

	// First Tick: hub A freezes (shrink), hub B advances cleanly.
	if err := loop.Tick(ctx, t0); err != nil {
		t.Fatalf("first Tick: %v", err)
	}
	assertViolation(t, path, hubA, "shrink")
	if n := countRows(t, path, "violations"); n != 1 {
		t.Fatalf("violations after first Tick = %d, want 1", n)
	}
	if alerts != 1 {
		t.Errorf("alerts after first Tick = %d, want 1", alerts)
	}
	fsA, err := s.FollowState(ctx, hubA)
	if err != nil {
		t.Fatalf("FollowState A: %v", err)
	}
	if !fsA.Frozen {
		t.Errorf("hub A Frozen = false after a shrink, want true")
	}
	if fsA.LastSize != 20000 {
		t.Errorf("hub A LastSize = %d, want 20000 (frozen hub does not advance)", fsA.LastSize)
	}
	assertCleanAdvance(t, ctx, s, hubB)

	// Tick past Normal but before Frozen: the frozen hub A is NOT due (back-off), so
	// no new violation is recorded. Hub B (unfrozen) IS due and re-polls cleanly.
	if err := loop.Tick(ctx, t0.Add(loop.Normal+time.Minute)); err != nil {
		t.Fatalf("back-off Tick: %v", err)
	}
	if n := countRows(t, path, "violations"); n != 1 {
		t.Errorf("violations during back-off = %d, want 1 (frozen hub not re-polled before Frozen)", n)
	}
	if alerts != 1 {
		t.Errorf("alerts during back-off = %d, want 1 (no new alert)", alerts)
	}
	assertCleanAdvance(t, ctx, s, hubB)

	// Tick at the Frozen interval: hub A is now due on the backed-off cadence and is
	// re-polled, recording a second violation as evidence — but no new alert — and
	// never clearing Frozen. Hub B stays advanced.
	if err := loop.Tick(ctx, t0.Add(loop.Frozen)); err != nil {
		t.Fatalf("frozen-interval Tick: %v", err)
	}
	if n := countRows(t, path, "violations"); n != 2 {
		t.Errorf("violations after Frozen re-poll = %d, want 2 (re-detection is evidence)", n)
	}
	if alerts != 1 {
		t.Errorf("alerts after Frozen re-poll = %d, want 1 (exactly-one-alert)", alerts)
	}
	fsA, err = s.FollowState(ctx, hubA)
	if err != nil {
		t.Fatalf("FollowState A after Frozen re-poll: %v", err)
	}
	if !fsA.Frozen {
		t.Errorf("hub A Frozen cleared on re-poll, want still true (no auto-unfreeze)")
	}
	if fsA.LastSize != 20000 {
		t.Errorf("hub A LastSize = %d after re-poll, want 20000 (still no advance)", fsA.LastSize)
	}
	assertCleanAdvance(t, ctx, s, hubB)
}

// TestTickMetricsPollFailure proves the Tick-level poll-failure counter: a hub
// whose fetcher always fails makes PollHub return early on the transport fault, so
// Tick (not PollHub) increments iscc_monitor_poll_failures_total for that hub. The
// counter is fired on the same error branch as the structured log, and is keyed by
// the hub_id. Asserted on the registry's rendered output, not on Loop internals.
func TestTickMetricsPollFailure(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	reg := metrics.New()
	loop := &Loop{
		Store:   s,
		Fetcher: errFetcher{},
		Targets: []HubTarget{{HubID: hubID, BaseURL: "https://sb0.iscc.id"}},
		Normal:  10 * time.Minute,
		Frozen:  time.Hour,
		Alert:   noopAlert,
		Metrics: reg,
	}

	// The faulting fetch surfaces as a non-nil Tick error (log-and-continue), and on
	// that branch Tick fires the poll-failure counter for the hub.
	if err := loop.Tick(ctx, sb0ObservedAt()); err == nil {
		t.Fatal("Tick error = nil, want the per-hub fetch fault surfaced")
	}
	assertMetric(t, reg, `iscc_monitor_poll_failures_total{hub_id="1"} 1`)
}

// assertCleanAdvance confirms a hub advanced to the sb0 fixture size and stayed
// unfrozen — the "other hubs unaffected" half of the freeze contract.
func assertCleanAdvance(t *testing.T, ctx context.Context, s *store.Store, hubID int64) {
	t.Helper()
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState hub %d: %v", hubID, err)
	}
	if fs.Frozen {
		t.Errorf("hub %d frozen, want unaffected by another hub's freeze", hubID)
	}
	if fs.LastSize != 10183 {
		t.Errorf("hub %d LastSize = %d, want 10183 (clean advance)", hubID, fs.LastSize)
	}
}
