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

// TestCheckpointAt confirms a recorded (root, raw) round-trips through
// CheckpointAt and that an absent (hubID, treeSize) returns found=false with a
// nil error (mirroring FollowState's absent-row convention).
func TestCheckpointAt(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	wantRoot := []byte("checkpoint-at-root-padding-32byt")
	wantRaw := []byte("sb0.iscc.id/log\n42\n<root>\n")
	if _, _, err := s.RecordCheckpoint(ctx, CheckpointRecord{
		HubID:      hubID,
		TreeSize:   42,
		Root:       wantRoot,
		Raw:        wantRaw,
		ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}

	root, raw, found, err := s.CheckpointAt(ctx, hubID, 42)
	if err != nil {
		t.Fatalf("CheckpointAt: %v", err)
	}
	if !found {
		t.Fatalf("found = false for a recorded checkpoint, want true")
	}
	if string(root) != string(wantRoot) {
		t.Errorf("root = %q, want %q", root, wantRoot)
	}
	if string(raw) != string(wantRaw) {
		t.Errorf("raw = %q, want %q", raw, wantRaw)
	}

	// An absent (hubID, treeSize) is not an error.
	_, _, found, err = s.CheckpointAt(ctx, hubID, 99)
	if err != nil {
		t.Fatalf("CheckpointAt absent size: %v", err)
	}
	if found {
		t.Errorf("found = true for an absent tree_size, want false")
	}

	// An absent hub is likewise not an error.
	_, _, found, err = s.CheckpointAt(ctx, 999, 42)
	if err != nil {
		t.Fatalf("CheckpointAt absent hub: %v", err)
	}
	if found {
		t.Errorf("found = true for an absent hub, want false")
	}
}

// TestCheckpointAtDeterministicOnFork confirms that when a hub recorded two
// different roots at one tree_size (a fork's evidence: the prior accepted root
// first, then the contradicting-evidence row), CheckpointAt returns the
// first-recorded (lowest-rowid) row deterministically — the prior accepted root,
// never the later contradicting row — so both the prior-root read and the
// follower's fork re-detection stay deterministic (ORDER BY rowid).
func TestCheckpointAtDeterministicOnFork(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	priorRoot := []byte("prior-accepted-root-padding-32by")
	priorRaw := []byte("sb0.iscc.id/log\n42\nprior\n")
	contradictRoot := []byte("contradicting-evidence-root-32by")
	contradictRaw := []byte("sb0.iscc.id/log\n42\ncontradict\n")
	// Non-vacuous: the two roots must differ for the ordering to mean anything.
	if string(priorRoot) == string(contradictRoot) {
		t.Fatal("test fixture broken: the two roots are equal")
	}

	// Record the prior accepted root first (lowest rowid), then the later
	// contradicting-evidence row at the SAME tree_size.
	if _, _, err := s.RecordCheckpoint(ctx, CheckpointRecord{
		HubID: hubID, TreeSize: 42, Root: priorRoot, Raw: priorRaw, ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordCheckpoint prior: %v", err)
	}
	if _, _, err := s.RecordCheckpoint(ctx, CheckpointRecord{
		HubID: hubID, TreeSize: 42, Root: contradictRoot, Raw: contradictRaw, ObservedAt: time.Unix(1_700_000_001, 0),
	}); err != nil {
		t.Fatalf("RecordCheckpoint contradicting: %v", err)
	}
	if n := countRows(t, s, "checkpoints"); n != 2 {
		t.Fatalf("checkpoints row count = %d, want 2 (two roots at one size)", n)
	}

	root, raw, found, err := s.CheckpointAt(ctx, hubID, 42)
	if err != nil {
		t.Fatalf("CheckpointAt: %v", err)
	}
	if !found {
		t.Fatalf("found = false, want true")
	}
	if string(root) != string(priorRoot) {
		t.Errorf("root = %q, want %q (the first-recorded prior accepted root)", root, priorRoot)
	}
	if string(raw) != string(priorRaw) {
		t.Errorf("raw = %q, want %q (the first-recorded prior accepted row)", raw, priorRaw)
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

// TestAdvanceAccepted proves the store-owned advance transaction performs all
// three writes (checkpoint record, set-once coverage, follow-cursor advance)
// atomically: (1) one call records the checkpoint row, sets monitored_since_size,
// and sets follow_state.last_size; (2) a second call with the same (hub, size,
// root) is idempotent — exactly one checkpoints row, last_size unchanged; (3) a
// later call at a larger size advances last_size but never moves the coverage
// start (ADR-0001, set-once coverage).
func TestAdvanceAccepted(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	t0 := time.Unix(1_700_000_000, 0)
	rec := CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   100,
		Root:       []byte("advance-accepted-root-padding-32"),
		Raw:        []byte("sb0.iscc.id/log\n100\n<root>\n"),
		ObservedAt: t0,
	}

	// (1) One AdvanceAccepted records the checkpoint, sets coverage, advances cursor.
	if err := s.AdvanceAccepted(ctx, rec); err != nil {
		t.Fatalf("first AdvanceAccepted: %v", err)
	}
	if n := countRows(t, s, "checkpoints"); n != 1 {
		t.Errorf("checkpoints row count = %d, want 1", n)
	}
	cov, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if !cov.Set || cov.Size != 100 || !cov.Since.Equal(t0) {
		t.Errorf("Coverage = %+v, want {Size:100 Since:%v Set:true}", cov, t0)
	}
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if fs.LastSize != 100 || fs.Frozen {
		t.Errorf("FollowState = %+v, want {LastSize:100 Frozen:false}", fs)
	}

	// (2) A second AdvanceAccepted with the same (hub, size, root) is idempotent:
	// exactly one checkpoints row, last_size and coverage unchanged.
	if err := s.AdvanceAccepted(ctx, rec); err != nil {
		t.Fatalf("idempotent AdvanceAccepted: %v", err)
	}
	if n := countRows(t, s, "checkpoints"); n != 1 {
		t.Errorf("checkpoints row count after re-poll = %d, want 1", n)
	}
	fs, err = s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after re-poll: %v", err)
	}
	if fs.LastSize != 100 {
		t.Errorf("LastSize after re-poll = %d, want 100 (unchanged)", fs.LastSize)
	}

	// (3) A later observation at a larger size advances last_size but does NOT move
	// the set-once coverage start (ADR-0001).
	t1 := time.Unix(1_700_009_999, 0)
	rec2 := CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   500,
		Root:       []byte("advance-accepted-root2-paddng-32"),
		Raw:        []byte("sb0.iscc.id/log\n500\n<root>\n"),
		ObservedAt: t1,
	}
	if err := s.AdvanceAccepted(ctx, rec2); err != nil {
		t.Fatalf("growth AdvanceAccepted: %v", err)
	}
	if n := countRows(t, s, "checkpoints"); n != 2 {
		t.Errorf("checkpoints row count after growth = %d, want 2", n)
	}
	fs, err = s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after growth: %v", err)
	}
	if fs.LastSize != 500 {
		t.Errorf("LastSize after growth = %d, want 500", fs.LastSize)
	}
	cov, err = s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage after growth: %v", err)
	}
	if cov.Size != 100 || !cov.Since.Equal(t0) {
		t.Errorf("Coverage after growth = %+v, want {Size:100 Since:%v} (set-once)", cov, t0)
	}

	// Independently confirm the raw coverage columns were not overwritten by growth.
	var size, since int64
	err = s.db.QueryRow(
		"SELECT monitored_since_size, monitored_since_time FROM hubs WHERE hub_id = ?", hubID,
	).Scan(&size, &since)
	if err != nil {
		t.Fatalf("read raw coverage columns: %v", err)
	}
	if size != 100 || since != t0.Unix() {
		t.Errorf("raw (size, since) = (%d, %d), want (100, %d)", size, since, t0.Unix())
	}
}

// TestCoverageSetOnce proves SetCoverage records the coverage start on the first
// call and never moves it (ADR-0001, coverage honesty): after SetCoverage(100, t0)
// then SetCoverage(500, t1), Coverage reports size 100 / since t0, ignoring the
// larger/later values. A re-call after the start is set returns a nil error.
func TestCoverageSetOnce(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	t0 := time.Unix(1_700_000_000, 0)
	t1 := time.Unix(1_700_009_999, 0)
	if err := s.SetCoverage(ctx, hubID, 100, t0); err != nil {
		t.Fatalf("first SetCoverage: %v", err)
	}
	// A larger, later observation must NOT move the start; the re-call is a no-op
	// and returns a nil error (it does not rely on RowsAffected to signal success).
	if err := s.SetCoverage(ctx, hubID, 500, t1); err != nil {
		t.Fatalf("second SetCoverage: %v", err)
	}

	info, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if !info.Set {
		t.Fatalf("Set = false after SetCoverage, want true")
	}
	if info.Size != 100 {
		t.Errorf("Size = %d, want 100 (set-once; the larger value is ignored)", info.Size)
	}
	if !info.Since.Equal(t0) {
		t.Errorf("Since = %v, want %v (set-once; the later time is ignored)", info.Since, t0)
	}

	// Independently confirm the raw columns were not overwritten.
	var size, since int64
	err = s.db.QueryRow(
		"SELECT monitored_since_size, monitored_since_time FROM hubs WHERE hub_id = ?", hubID,
	).Scan(&size, &since)
	if err != nil {
		t.Fatalf("read raw coverage columns: %v", err)
	}
	if size != 100 || since != t0.Unix() {
		t.Errorf("raw (size, since) = (%d, %d), want (100, %d)", size, since, t0.Unix())
	}
}

// TestCoverageUnset confirms a hub that has never started coverage reports Set
// false with zero Size / Since (and a nil error), and that an absent hub likewise
// returns Set false — mirroring FollowState's "absent row is not an error".
func TestCoverageUnset(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	info, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage for a hub with no coverage: %v", err)
	}
	if info != (CoverageInfo{}) {
		t.Errorf("Coverage of an un-started hub = %+v, want zero value", info)
	}

	// An absent hub is likewise not an error.
	info, err = s.Coverage(ctx, 999)
	if err != nil {
		t.Fatalf("Coverage for an absent hub: %v", err)
	}
	if info.Set {
		t.Errorf("Set = true for an absent hub, want false")
	}
}

// TestCoverageZeroObservedAtNull confirms a zero observedAt writes monitored_since_time
// as NULL (size still set), keeping "never observed" distinct from the unix epoch,
// and that Coverage reads it back as a zero Since with Set true.
func TestCoverageZeroObservedAtNull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	if err := s.SetCoverage(ctx, hubID, 7, time.Time{}); err != nil {
		t.Fatalf("SetCoverage: %v", err)
	}

	var since any
	if err := s.db.QueryRow("SELECT monitored_since_time FROM hubs WHERE hub_id = ?", hubID).Scan(&since); err != nil {
		t.Fatalf("read monitored_since_time: %v", err)
	}
	if since != nil {
		t.Errorf("zero observedAt stored as %v, want NULL", since)
	}

	info, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if !info.Set || info.Size != 7 {
		t.Errorf("Coverage = %+v, want {Size:7 Set:true}", info)
	}
	if !info.Since.IsZero() {
		t.Errorf("Since = %v, want zero (NULL time)", info.Since)
	}
}

// TestRecordViolation confirms a violation round-trips its kind, both raw
// contradictory checkpoints, and the proof JSON, and returns a positive id.
func TestRecordViolation(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	rawA := []byte("sb0.iscc.id/log\n42\nrootA...\n")
	rawB := []byte("sb0.iscc.id/log\n42\nrootB...\n")
	proof := `{"size_a":42,"size_b":42,"path":["aa","bb"]}`
	id, err := s.RecordViolation(ctx, Violation{
		HubID:      hubID,
		Kind:       "equivocation",
		RawA:       rawA,
		RawB:       rawB,
		ProofJSON:  proof,
		DetectedAt: time.Unix(1_700_000_000, 0),
	})
	if err != nil {
		t.Fatalf("RecordViolation: %v", err)
	}
	if id <= 0 {
		t.Errorf("RecordViolation id = %d, want > 0", id)
	}

	var (
		kind     string
		gotA     []byte
		gotB     []byte
		gotProof string
		detected int64
	)
	err = s.db.QueryRow(
		"SELECT kind, raw_a, raw_b, proof_json, detected_at FROM violations WHERE id = ?", id,
	).Scan(&kind, &gotA, &gotB, &gotProof, &detected)
	if err != nil {
		t.Fatalf("read violation columns: %v", err)
	}
	if kind != "equivocation" {
		t.Errorf("kind = %q, want equivocation", kind)
	}
	if string(gotA) != string(rawA) {
		t.Errorf("raw_a = %q, want %q", gotA, rawA)
	}
	if string(gotB) != string(rawB) {
		t.Errorf("raw_b = %q, want %q", gotB, rawB)
	}
	if gotProof != proof {
		t.Errorf("proof_json = %q, want %q", gotProof, proof)
	}
	if detected != 1_700_000_000 {
		t.Errorf("detected_at = %d, want 1700000000", detected)
	}
}

// TestRecordViolationNoDedupe confirms re-detecting the same violation records a
// distinct row (no UNIQUE constraint): repeated detection is itself evidence.
func TestRecordViolationNoDedupe(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	v := Violation{
		HubID:      hubID,
		Kind:       "shrink",
		RawA:       []byte("a"),
		RawB:       []byte("b"),
		ProofJSON:  "",
		DetectedAt: time.Unix(1_700_000_000, 0),
	}
	id1, err := s.RecordViolation(ctx, v)
	if err != nil {
		t.Fatalf("first RecordViolation: %v", err)
	}
	id2, err := s.RecordViolation(ctx, v)
	if err != nil {
		t.Fatalf("second RecordViolation: %v", err)
	}
	if id1 == id2 {
		t.Errorf("re-detection reused id %d, want a distinct row", id2)
	}
	if n := countRows(t, s, "violations"); n != 2 {
		t.Errorf("violations row count = %d, want 2", n)
	}

	// An empty ProofJSON is stored as an empty string, not NULL.
	var proof any
	if err := s.db.QueryRow("SELECT proof_json FROM violations WHERE id = ?", id1).Scan(&proof); err != nil {
		t.Fatalf("read proof_json: %v", err)
	}
	if proof != "" {
		t.Errorf("empty ProofJSON stored as %v, want empty string", proof)
	}
}

// TestRecordViolationZeroDetectedAtNull confirms a zero DetectedAt is stored as
// NULL, matching the unixOrNil convention.
func TestRecordViolationZeroDetectedAtNull(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	id, err := s.RecordViolation(ctx, Violation{
		HubID: hubID,
		Kind:  "fork",
		RawA:  []byte("a"),
		RawB:  []byte("b"),
		// DetectedAt left zero.
	})
	if err != nil {
		t.Fatalf("RecordViolation: %v", err)
	}
	var detected any
	if err := s.db.QueryRow("SELECT detected_at FROM violations WHERE id = ?", id).Scan(&detected); err != nil {
		t.Fatalf("read detected_at: %v", err)
	}
	if detected != nil {
		t.Errorf("zero DetectedAt stored as %v, want NULL", detected)
	}
}

// TestFreezeNoPriorRow confirms Freeze on a hub with no follow_state row creates
// the row with frozen=1 (a hub can be frozen before its first verified advance).
func TestFreezeNoPriorRow(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	if err := s.Freeze(ctx, hubID); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if !fs.Frozen {
		t.Errorf("Frozen = false after Freeze on a hub with no prior row, want true")
	}
	if n := countRows(t, s, "follow_state"); n != 1 {
		t.Errorf("follow_state row count = %d, want 1", n)
	}
}

// TestFreezeNoAutoUnfreeze proves a Freeze followed by AdvanceFollowState keeps
// frozen=1 while updating the cursor (ADR-0006, no auto-unfreeze): the advance
// moves last_size but never clears the freeze.
func TestFreezeNoAutoUnfreeze(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	if err := s.Freeze(ctx, hubID); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, 99); err != nil {
		t.Fatalf("AdvanceFollowState after Freeze: %v", err)
	}
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if !fs.Frozen {
		t.Errorf("frozen cleared by AdvanceFollowState after Freeze, want still frozen")
	}
	if fs.LastSize != 99 {
		t.Errorf("LastSize = %d, want 99", fs.LastSize)
	}
}

// TestFreezeOtherHubsUnaffected proves freezing one hub leaves another hub's
// follow_state untouched (a violation freezes only the offending hub, ADR-0006).
func TestFreezeOtherHubsUnaffected(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubA, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub A: %v", err)
	}
	hubB, err := s.UpsertHub(ctx, "sb1.iscc.id", "sb1.iscc.id/log", "https://sb1.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub B: %v", err)
	}
	if err := s.Freeze(ctx, hubA); err != nil {
		t.Fatalf("Freeze A: %v", err)
	}
	fsB, err := s.FollowState(ctx, hubB)
	if err != nil {
		t.Fatalf("FollowState B: %v", err)
	}
	if fsB.Frozen {
		t.Errorf("hub B frozen after freezing hub A, want unaffected")
	}
}

// TestFreezeRestartSurvival writes a violation and a freeze via the typed methods,
// closes the store, reopens the same path, and confirms the evidence and the
// freeze flag both survive — the "evidence survives restart" property (ADR-0006).
func TestFreezeRestartSurvival(t *testing.T) {
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
	if _, err := s.RecordViolation(ctx, Violation{
		HubID:      hubID,
		Kind:       "fork",
		RawA:       []byte("checkpoint-a"),
		RawB:       []byte("checkpoint-b"),
		ProofJSON:  `{"path":[]}`,
		DetectedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordViolation: %v", err)
	}
	if err := s.Freeze(ctx, hubID); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	if n := countRows(t, s2, "violations"); n != 1 {
		t.Errorf("violations after reopen = %d, want 1", n)
	}
	fs, err := s2.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after reopen: %v", err)
	}
	if !fs.Frozen {
		t.Errorf("frozen not preserved across restart, want true")
	}
}

// TestRecordHubKeyInsert confirms a first RecordHubKey for a seeded hub inserts
// exactly one row whose pubkey_raw / pubkey_z / resolved_at columns read back
// equal to the input.
func TestRecordHubKeyInsert(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	pubRaw := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	resolved := time.Unix(1_700_000_000, 0)
	k := HubKey{
		HubID:      hubID,
		KeyID:      0x40b74463,
		PubkeyRaw:  pubRaw,
		PubkeyZ:    "z6MkExampleMultibaseValue",
		ResolvedAt: resolved,
		// Revoked left zero (un-revoked key).
	}
	if err := s.RecordHubKey(ctx, k); err != nil {
		t.Fatalf("RecordHubKey: %v", err)
	}

	var n int
	if err := s.db.QueryRow(
		"SELECT count(*) FROM hub_keys WHERE hub_id = ? AND key_id = ?", hubID, int64(k.KeyID),
	).Scan(&n); err != nil {
		t.Fatalf("count hub_keys: %v", err)
	}
	if n != 1 {
		t.Fatalf("hub_keys count for (hub, key) = %d, want 1", n)
	}

	var (
		gotRaw      []byte
		gotZ        string
		gotResolved int64
	)
	err = s.db.QueryRow(
		"SELECT pubkey_raw, pubkey_z, resolved_at FROM hub_keys WHERE hub_id = ? AND key_id = ?",
		hubID, int64(k.KeyID),
	).Scan(&gotRaw, &gotZ, &gotResolved)
	if err != nil {
		t.Fatalf("read hub_keys columns: %v", err)
	}
	if string(gotRaw) != string(pubRaw) {
		t.Errorf("pubkey_raw = %q, want %q", gotRaw, pubRaw)
	}
	if gotZ != k.PubkeyZ {
		t.Errorf("pubkey_z = %q, want %q", gotZ, k.PubkeyZ)
	}
	if gotResolved != resolved.Unix() {
		t.Errorf("resolved_at = %d, want %d", gotResolved, resolved.Unix())
	}
}

// TestRecordHubKeyRefresh proves a second RecordHubKey with the same
// (hub_id, key_id) but a later ResolvedAt and a set Revoked refreshes the row in
// place (count stays 1; resolved_at and revoked_at are updated) — the DID
// document is the source of truth, so a re-resolve overwrites, never accumulates.
func TestRecordHubKeyRefresh(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	const keyID = uint32(0x069d0f14)
	first := HubKey{
		HubID:      hubID,
		KeyID:      keyID,
		PubkeyRaw:  []byte("first-resolution-pubkey-padding3"),
		ResolvedAt: time.Unix(1_700_000_000, 0),
	}
	if err := s.RecordHubKey(ctx, first); err != nil {
		t.Fatalf("first RecordHubKey: %v", err)
	}

	// Re-resolve the same key later, now revoked in the DID doc.
	laterResolved := time.Unix(1_700_009_999, 0)
	revoked := time.Unix(1_700_005_000, 0)
	second := HubKey{
		HubID:      hubID,
		KeyID:      keyID,
		PubkeyRaw:  []byte("second-resolution-pubkey-paddin3"),
		Revoked:    revoked,
		ResolvedAt: laterResolved,
	}
	if err := s.RecordHubKey(ctx, second); err != nil {
		t.Fatalf("second RecordHubKey: %v", err)
	}

	var n int
	if err := s.db.QueryRow(
		"SELECT count(*) FROM hub_keys WHERE hub_id = ? AND key_id = ?", hubID, int64(keyID),
	).Scan(&n); err != nil {
		t.Fatalf("count hub_keys: %v", err)
	}
	if n != 1 {
		t.Fatalf("hub_keys count after refresh = %d, want 1 (in-place update)", n)
	}

	var (
		gotRaw      []byte
		gotResolved int64
		gotRevoked  int64
	)
	err = s.db.QueryRow(
		"SELECT pubkey_raw, resolved_at, revoked_at FROM hub_keys WHERE hub_id = ? AND key_id = ?",
		hubID, int64(keyID),
	).Scan(&gotRaw, &gotResolved, &gotRevoked)
	if err != nil {
		t.Fatalf("read refreshed hub_keys columns: %v", err)
	}
	if string(gotRaw) != string(second.PubkeyRaw) {
		t.Errorf("pubkey_raw = %q, want %q (refreshed)", gotRaw, second.PubkeyRaw)
	}
	if gotResolved != laterResolved.Unix() {
		t.Errorf("resolved_at = %d, want %d (refreshed)", gotResolved, laterResolved.Unix())
	}
	if gotRevoked != revoked.Unix() {
		t.Errorf("revoked_at = %d, want %d (now revoked)", gotRevoked, revoked.Unix())
	}
}

// TestRecordHubKeyRotation proves a RecordHubKey with a different key_id for the
// same hub inserts a second row (count == 2): a key rotation caches both the old
// and new keys; the old key is not deleted (its revocation, if any, is recorded
// via revoked_at).
func TestRecordHubKeyRotation(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Old key (sb1's prior signer 22b08f3e) then the rotated key (069d0f14).
	if err := s.RecordHubKey(ctx, HubKey{
		HubID:      hubID,
		KeyID:      0x22b08f3e,
		PubkeyRaw:  []byte("old-rotated-out-pubkey-padding-3"),
		ResolvedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordHubKey old key: %v", err)
	}
	if err := s.RecordHubKey(ctx, HubKey{
		HubID:      hubID,
		KeyID:      0x069d0f14,
		PubkeyRaw:  []byte("new-rotated-in-pubkey-padding-32"),
		ResolvedAt: time.Unix(1_700_009_999, 0),
	}); err != nil {
		t.Fatalf("RecordHubKey new key: %v", err)
	}

	var n int
	if err := s.db.QueryRow(
		"SELECT count(*) FROM hub_keys WHERE hub_id = ?", hubID,
	).Scan(&n); err != nil {
		t.Fatalf("count hub_keys: %v", err)
	}
	if n != 2 {
		t.Errorf("hub_keys count after rotation = %d, want 2 (both keys cached)", n)
	}
}

// TestRecordHubKeyNullable confirms a HubKey with an empty PubkeyZ and a zero
// Revoked writes pubkey_z IS NULL and revoked_at IS NULL — keeping "no multibase"
// and "not revoked" distinct from the empty string and the unix epoch.
func TestRecordHubKeyNullable(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	const keyID = uint32(0x40b74463)
	if err := s.RecordHubKey(ctx, HubKey{
		HubID:      hubID,
		KeyID:      keyID,
		PubkeyRaw:  []byte("nullable-test-pubkey-padding-32b"),
		ResolvedAt: time.Unix(1_700_000_000, 0),
		// PubkeyZ empty, Revoked zero.
	}); err != nil {
		t.Fatalf("RecordHubKey: %v", err)
	}

	// The row is selectable via the IS NULL predicates (proves NULL, not "").
	var n int
	if err := s.db.QueryRow(
		"SELECT count(*) FROM hub_keys WHERE hub_id = ? AND key_id = ? "+
			"AND pubkey_z IS NULL AND revoked_at IS NULL", hubID, int64(keyID),
	).Scan(&n); err != nil {
		t.Fatalf("count hub_keys with NULL columns: %v", err)
	}
	if n != 1 {
		t.Errorf("rows with pubkey_z IS NULL AND revoked_at IS NULL = %d, want 1", n)
	}
}

// TestRecordHubKeyForeignKey confirms RecordHubKey for a hub_id with no hubs row
// returns a non-nil error (the hub_id REFERENCES hubs(hub_id) FK is enforced).
func TestRecordHubKeyForeignKey(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	err := s.RecordHubKey(ctx, HubKey{
		HubID:      999, // no hubs row
		KeyID:      0x40b74463,
		PubkeyRaw:  []byte("orphan-key-pubkey-padding-32byte"),
		ResolvedAt: time.Unix(1_700_000_000, 0),
	})
	if err == nil {
		t.Fatalf("RecordHubKey for an unknown hub_id returned nil, want a FK error")
	}
}

// TestLookupHubKeyRoundTrip confirms a RecordHubKey then LookupHubKey returns
// found=true with every field byte/value-equal to what was written (the read side
// of the cache is lossless for the set columns).
func TestLookupHubKeyRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	pubRaw := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	resolved := time.Unix(1_700_000_000, 0)
	revoked := time.Unix(1_700_005_000, 0)
	want := HubKey{
		HubID:      hubID,
		KeyID:      0x40b74463,
		PubkeyRaw:  pubRaw,
		PubkeyZ:    "z6MkExampleMultibaseValue",
		Revoked:    revoked,
		ResolvedAt: resolved,
	}
	if err := s.RecordHubKey(ctx, want); err != nil {
		t.Fatalf("RecordHubKey: %v", err)
	}

	got, found, err := s.LookupHubKey(ctx, hubID, want.KeyID)
	if err != nil {
		t.Fatalf("LookupHubKey: %v", err)
	}
	if !found {
		t.Fatalf("LookupHubKey found = false, want true")
	}
	if got.HubID != hubID {
		t.Errorf("HubID = %d, want %d", got.HubID, hubID)
	}
	if got.KeyID != want.KeyID {
		t.Errorf("KeyID = %08x, want %08x", got.KeyID, want.KeyID)
	}
	if string(got.PubkeyRaw) != string(pubRaw) {
		t.Errorf("PubkeyRaw = %q, want %q", got.PubkeyRaw, pubRaw)
	}
	if got.PubkeyZ != want.PubkeyZ {
		t.Errorf("PubkeyZ = %q, want %q", got.PubkeyZ, want.PubkeyZ)
	}
	if !got.Revoked.Equal(revoked) {
		t.Errorf("Revoked = %v, want %v", got.Revoked, revoked)
	}
	if !got.ResolvedAt.Equal(resolved) {
		t.Errorf("ResolvedAt = %v, want %v", got.ResolvedAt, resolved)
	}
}

// TestLookupHubKeyNullableRoundTrip proves a key written with an empty PubkeyZ and
// a zero Revoked reads back PubkeyZ == "" and Revoked.IsZero() == true — the NULL
// → zero inverse of nullStringOrNil / unixOrNil.
func TestLookupHubKeyNullableRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	const keyID = uint32(0x40b74463)
	if err := s.RecordHubKey(ctx, HubKey{
		HubID:      hubID,
		KeyID:      keyID,
		PubkeyRaw:  []byte("nullable-test-pubkey-padding-32b"),
		ResolvedAt: time.Unix(1_700_000_000, 0),
		// PubkeyZ empty, Revoked zero.
	}); err != nil {
		t.Fatalf("RecordHubKey: %v", err)
	}

	got, found, err := s.LookupHubKey(ctx, hubID, keyID)
	if err != nil {
		t.Fatalf("LookupHubKey: %v", err)
	}
	if !found {
		t.Fatalf("LookupHubKey found = false, want true")
	}
	if got.PubkeyZ != "" {
		t.Errorf("PubkeyZ = %q, want \"\" (NULL → empty)", got.PubkeyZ)
	}
	if !got.Revoked.IsZero() {
		t.Errorf("Revoked = %v, want zero (NULL → zero time)", got.Revoked)
	}
}

// TestLookupHubKeyAbsent confirms a lookup for a (hub_id, key_id) with no row
// returns (HubKey{}, false, nil) — an absent key is a miss, not an error.
func TestLookupHubKeyAbsent(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	got, found, err := s.LookupHubKey(ctx, hubID, 0xdeadbeef)
	if err != nil {
		t.Fatalf("LookupHubKey for absent key returned err = %v, want nil", err)
	}
	if found {
		t.Errorf("found = true for an absent (hub, key), want false")
	}
	// HubKey holds a []byte (not comparable with ==); assert each field is zero.
	if got.HubID != 0 || got.KeyID != 0 || got.PubkeyRaw != nil || got.PubkeyZ != "" ||
		!got.Revoked.IsZero() || !got.ResolvedAt.IsZero() {
		t.Errorf("HubKey = %+v, want zero value", got)
	}
}

// TestLookupHubKeyDiscriminatesKeyID proves that after a rotation (two rows with
// distinct key ids and distinct pubkey_raw), each LookupHubKey returns its own
// key's bytes, not the other's.
func TestLookupHubKeyDiscriminatesKeyID(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)

	hubID, err := s.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	const oldKeyID = uint32(0x22b08f3e)
	const newKeyID = uint32(0x069d0f14)
	oldRaw := []byte("old-rotated-out-pubkey-padding-3")
	newRaw := []byte("new-rotated-in-pubkey-padding-32")
	// Non-vacuous: the two keys must differ for the discrimination to mean anything.
	if string(oldRaw) == string(newRaw) {
		t.Fatal("test fixture broken: old and new pubkey_raw are equal")
	}
	if err := s.RecordHubKey(ctx, HubKey{
		HubID:      hubID,
		KeyID:      oldKeyID,
		PubkeyRaw:  oldRaw,
		ResolvedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordHubKey old key: %v", err)
	}
	if err := s.RecordHubKey(ctx, HubKey{
		HubID:      hubID,
		KeyID:      newKeyID,
		PubkeyRaw:  newRaw,
		ResolvedAt: time.Unix(1_700_009_999, 0),
	}); err != nil {
		t.Fatalf("RecordHubKey new key: %v", err)
	}

	gotOld, found, err := s.LookupHubKey(ctx, hubID, oldKeyID)
	if err != nil || !found {
		t.Fatalf("LookupHubKey old: found=%v err=%v", found, err)
	}
	if string(gotOld.PubkeyRaw) != string(oldRaw) {
		t.Errorf("old key PubkeyRaw = %q, want %q", gotOld.PubkeyRaw, oldRaw)
	}

	gotNew, found, err := s.LookupHubKey(ctx, hubID, newKeyID)
	if err != nil || !found {
		t.Fatalf("LookupHubKey new: found=%v err=%v", found, err)
	}
	if string(gotNew.PubkeyRaw) != string(newRaw) {
		t.Errorf("new key PubkeyRaw = %q, want %q", gotNew.PubkeyRaw, newRaw)
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
