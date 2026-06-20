// Tests for the typed checkpoint-verdict persistence helpers (UpsertHub,
// RecordCheckpoint, FollowState, AdvanceFollowState). Each drives the public
// method against a t.TempDir() database and asserts on observable rows — raw
// db reads for counts and column values, independently-pinned expectations —
// never on Store internals, matching the style in sqlite_test.go.
package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// openTemp opens a fresh store in a temp dir and registers cleanup.
func openTemp(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// countRows reads a single COUNT(*) from the named table.
func countRows(t *testing.T, s *Store, table string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestUpsertHubIdempotent confirms a re-register with the same domain returns the
// same hub_id and leaves exactly one hubs row.
func TestUpsertHubIdempotent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	id1, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("first UpsertHub: %v", err)
	}
	id2, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("second UpsertHub: %v", err)
	}
	if id1 != id2 {
		t.Errorf("re-register hub_id = %d, want %d (same)", id2, id1)
	}
	if n := countRows(t, s, "hubs"); n != 1 {
		t.Errorf("hubs row count = %d, want 1", n)
	}

	// A different domain is a distinct row with a distinct id.
	id3, err := s.UpsertHub(ctx, "sb1.iscc.id", "sb1.iscc.id/log", "https://sb1.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub other domain: %v", err)
	}
	if id3 == id1 {
		t.Errorf("distinct domain reused hub_id %d", id3)
	}
	if n := countRows(t, s, "hubs"); n != 2 {
		t.Errorf("hubs row count = %d, want 2", n)
	}

	// Columns from the first insert are preserved.
	var origin, baseURL string
	err = s.db.QueryRow("SELECT origin, base_url FROM hubs WHERE hub_id = ?", id1).Scan(&origin, &baseURL)
	if err != nil {
		t.Fatalf("read hub columns: %v", err)
	}
	if origin != "sb0.iscc.id/log" || baseURL != "https://sb0.iscc.id" {
		t.Errorf("hub columns = (%q, %q), want (sb0.iscc.id/log, https://sb0.iscc.id)", origin, baseURL)
	}
}

// TestRecordCheckpointDedupes confirms a re-observed (hub_id, tree_size, root)
// returns the same id with inserted=false and leaves exactly one checkpoints row.
func TestRecordCheckpointDedupes(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	rec := CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   42,
		Root:       []byte("0123456789abcdef0123456789abcdef"),
		Raw:        []byte("sb0.iscc.id/log\n42\n...\n"),
		ObservedAt: time.Unix(1_700_000_000, 0),
	}

	id1, inserted, err := s.RecordCheckpoint(ctx, rec)
	if err != nil {
		t.Fatalf("first RecordCheckpoint: %v", err)
	}
	if !inserted {
		t.Errorf("first RecordCheckpoint inserted = false, want true")
	}

	id2, inserted, err := s.RecordCheckpoint(ctx, rec)
	if err != nil {
		t.Fatalf("second RecordCheckpoint: %v", err)
	}
	if inserted {
		t.Errorf("re-observed RecordCheckpoint inserted = true, want false")
	}
	if id2 != id1 {
		t.Errorf("re-observed id = %d, want %d (same)", id2, id1)
	}
	if n := countRows(t, s, "checkpoints"); n != 1 {
		t.Errorf("checkpoints row count = %d, want 1", n)
	}

	// observed_at persisted as unix-seconds; consistent/root_rebuilt stay NULL.
	var (
		observedAt  int64
		consistent  any
		rootRebuilt any
	)
	err = s.db.QueryRow(
		"SELECT observed_at, consistent, root_rebuilt FROM checkpoints WHERE id = ?", id1,
	).Scan(&observedAt, &consistent, &rootRebuilt)
	if err != nil {
		t.Fatalf("read checkpoint columns: %v", err)
	}
	if observedAt != 1_700_000_000 {
		t.Errorf("observed_at = %d, want 1700000000", observedAt)
	}
	if consistent != nil || rootRebuilt != nil {
		t.Errorf("consistent/root_rebuilt = (%v, %v), want (nil, nil)", consistent, rootRebuilt)
	}

	// A different root for the same hub is a distinct, recorded checkpoint.
	rec2 := rec
	rec2.TreeSize = 43
	rec2.Root = []byte("fedcba9876543210fedcba9876543210")
	_, inserted, err = s.RecordCheckpoint(ctx, rec2)
	if err != nil {
		t.Fatalf("RecordCheckpoint distinct root: %v", err)
	}
	if !inserted {
		t.Errorf("distinct (size, root) inserted = false, want true")
	}
	if n := countRows(t, s, "checkpoints"); n != 2 {
		t.Errorf("checkpoints row count = %d, want 2", n)
	}
}

// TestRecordCheckpointZeroObservedAtNull confirms a zero ObservedAt is stored as
// NULL, keeping "never observed" distinct from the unix epoch.
func TestRecordCheckpointZeroObservedAtNull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	id, _, err := s.RecordCheckpoint(ctx, CheckpointRecord{
		HubID:    hubID,
		TreeSize: 1,
		Root:     []byte("zero-observed-at-root-padding-32"),
		Raw:      []byte("raw"),
		// ObservedAt left zero.
	})
	if err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	var observedAt any
	if err := s.db.QueryRow("SELECT observed_at FROM checkpoints WHERE id = ?", id).Scan(&observedAt); err != nil {
		t.Fatalf("read observed_at: %v", err)
	}
	if observedAt != nil {
		t.Errorf("zero ObservedAt stored as %v, want NULL", observedAt)
	}
}

// TestFollowStateUnknownHub confirms an unknown hub returns the zero FollowState
// and a nil error, not an error.
func TestFollowStateUnknownHub(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	fs, err := s.FollowState(ctx, 999)
	if err != nil {
		t.Fatalf("FollowState unknown hub: %v", err)
	}
	if fs != (FollowState{}) {
		t.Errorf("FollowState unknown hub = %+v, want zero value", fs)
	}
}

// TestAdvanceFollowState confirms the cursor advances and round-trips through
// FollowState, including across a fresh row creation and a subsequent update.
func TestAdvanceFollowState(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	if err := s.AdvanceFollowState(ctx, hubID, 10); err != nil {
		t.Fatalf("first AdvanceFollowState: %v", err)
	}
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if fs.LastSize != 10 || fs.Frozen {
		t.Errorf("FollowState = %+v, want {LastSize:10 Frozen:false}", fs)
	}

	if err := s.AdvanceFollowState(ctx, hubID, 25); err != nil {
		t.Fatalf("second AdvanceFollowState: %v", err)
	}
	fs, err = s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after advance: %v", err)
	}
	if fs.LastSize != 25 {
		t.Errorf("LastSize after advance = %d, want 25", fs.LastSize)
	}
	if n := countRows(t, s, "follow_state"); n != 1 {
		t.Errorf("follow_state row count = %d, want 1", n)
	}
}

// TestAdvanceFollowStateNoAutoUnfreeze proves AdvanceFollowState never clears the
// freeze flag (ADR-0006, no auto-unfreeze): after a raw UPDATE sets frozen=1, an
// advance leaves frozen=1 and updates last_size.
func TestAdvanceFollowStateNoAutoUnfreeze(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	// Seed a row, then freeze it via raw SQL (the freeze path is not in scope).
	if err := s.AdvanceFollowState(ctx, hubID, 5); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE follow_state SET frozen = 1 WHERE hub_id = ?", hubID); err != nil {
		t.Fatalf("raw freeze: %v", err)
	}

	if err := s.AdvanceFollowState(ctx, hubID, 99); err != nil {
		t.Fatalf("AdvanceFollowState on frozen hub: %v", err)
	}

	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if !fs.Frozen {
		t.Errorf("frozen cleared by AdvanceFollowState, want still frozen")
	}
	if fs.LastSize != 99 {
		t.Errorf("LastSize = %d, want 99", fs.LastSize)
	}

	// Independently confirm the raw column too.
	var frozen, lastSize int64
	err = s.db.QueryRow("SELECT frozen, last_size FROM follow_state WHERE hub_id = ?", hubID).Scan(&frozen, &lastSize)
	if err != nil {
		t.Fatalf("read raw follow_state: %v", err)
	}
	if frozen != 1 || lastSize != 99 {
		t.Errorf("raw (frozen, last_size) = (%d, %d), want (1, 99)", frozen, lastSize)
	}
}

// TestCheckpointHelpersRestartSurvival writes via the typed methods, closes the
// store, reopens the same path, and confirms the rows are still readable —
// mirroring TestStoreRestartSurvival but exercising the new helpers.
func TestCheckpointHelpersRestartSurvival(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "testnet.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	if _, _, err := s.RecordCheckpoint(ctx, CheckpointRecord{
		HubID:      hubID,
		TreeSize:   7,
		Root:       []byte("restart-survival-root-padding-32"),
		Raw:        []byte("raw"),
		ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, 7); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	if n := countRows(t, s2, "checkpoints"); n != 1 {
		t.Errorf("checkpoints after reopen = %d, want 1", n)
	}
	fs, err := s2.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after reopen: %v", err)
	}
	if fs.LastSize != 7 {
		t.Errorf("LastSize after reopen = %d, want 7", fs.LastSize)
	}
}
