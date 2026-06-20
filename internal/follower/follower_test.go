// Tests for the single-observation follower wiring (PollHub). They drive the full
// fetch -> AcceptCheckpoint -> persist chain against the captured sb0 fixtures
// through an in-test composite Fetcher that routes on the URL (the sb0 did.json
// for the did-document request, the sb0 checkpoint otherwise), so the live
// network is never touched. Assertions are on observable store outputs only — the
// follow cursor read back via the store's public FollowState — never on follower
// internals.
//
// AcceptCheckpoint re-derives the verifier-key origin from baseURL, and the sb0
// checkpoint is signed under "sb0.iscc.id/log"; the key is origin-bound, so the
// tests pass baseURL="https://sb0.iscc.id" with the captured sb0 did.json rather
// than a live httptest host (which would derive a non-matching origin).
package follower

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"

	_ "modernc.org/sqlite" // registers the "sqlite" driver for the inspector reads
)

// compositeFetcher routes Fetch on the URL: did.json requests get didDoc, every
// other URL (the checkpoint) gets checkpoint. AcceptCheckpoint resolves both the
// checkpoint and the hub's did.json through the same Fetcher, so one routing
// fetcher drives the whole fetch -> verify -> persist chain offline.
type compositeFetcher struct {
	checkpoint []byte
	didDoc     []byte
}

func (f compositeFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	if strings.HasSuffix(url, "did.json") {
		return f.didDoc, nil
	}
	return f.checkpoint, nil
}

// sb1Multibase is sb1's prior did:web key — a real Ed25519 key that does NOT sign
// the sb0 checkpoint, used to drive the non-advancing StatusUnverified path.
const sb1Multibase = "z6MkiNW46AUjNmKTV2YNyFi9ANG9wbfYQQoUQgADGwScd9jk"

// readFixture loads a captured did.json fixture from the logclient package's
// testdata/ directory (the committed did:web fixtures live alongside the resolver).
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "logclient", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}
	return data
}

// readCheckpoint loads a captured live checkpoint fixture from the module-root
// testdata/live/ directory (two levels up from this package).
func readCheckpoint(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "live", name))
	if err != nil {
		t.Fatalf("read checkpoint fixture %q: %v", name, err)
	}
	return data
}

// openTemp opens a fresh store in a temp dir and registers cleanup.
func openTemp(t *testing.T) *store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// didJSON builds a minimal one-method did:web document for the sb0 origin using
// the given publicKeyMultibase, mirroring the accept-test helper. It lets the
// non-advancing test advertise a mismatching key without editing a fixture.
func didJSON(multibase string) []byte {
	doc := `{"id": "did:web:sb0.iscc.id", "verificationMethod": [` +
		`{"id": "did:web:sb0.iscc.id#k", "type": "Multikey", "controller": "did:web:sb0.iscc.id", "publicKeyMultibase": "` + multibase + `"}` +
		`], "assertionMethod": ["did:web:sb0.iscc.id#k"]}`
	return []byte(doc)
}

// noopAlert is the no-op alert sink for tests that do not assert on alerting.
func noopAlert(int64, string) {}

// sb0FixtureRootB64 is the base64 root of the sb0 checkpoint fixture (line 3),
// pinned here so the synthetic fork seed can assert its seeded root differs from
// the real one — keeping the "different root at the same size" case non-vacuous.
const sb0FixtureRootB64 = "uir3z5T1yZVfmzWWPyagl0lifPjFdVxnGFquCSOOC8E="

// sb0VerifiedFetcher returns the composite fetcher that serves the real sb0
// checkpoint and its matching did.json, so PollHub yields StatusVerified at the
// sb0 fixture size (10183) and the real fixture root.
func sb0VerifiedFetcher(t *testing.T) compositeFetcher {
	t.Helper()
	return compositeFetcher{
		checkpoint: readCheckpoint(t, "sb0.iscc.id_checkpoint"),
		didDoc:     readFixture(t, "sb0.iscc.id_did.json"),
	}
}

// sb0ObservedAt is a time inside the sb0 fixture's (unconstrained) validity
// window, so AcceptCheckpoint yields StatusVerified rather than StatusRotated.
func sb0ObservedAt() time.Time {
	return time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
}

// countRows reads a single COUNT(*) from the named table by opening an
// independent read-only connection to the store's file, so the assertions pin
// observable evidence rows without reaching into follower or store internals. It
// requires the *Store to have flushed its writes (it shares the file, WAL mode).
func countRows(t *testing.T, dbPath, table string) int {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open inspector db: %v", err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestPollHubFork drives the fork freeze path end-to-end through the
// outbound-fetch seam: the prior accepted checkpoint is seeded at the sb0 fixture
// size (10183) with a deliberately different root, then PollHub observes the real
// sb0 checkpoint at that same size with its real (different) root. The result is a
// "fork" violation row, frozen=1, a cursor that does NOT advance past the prior
// size, and exactly one alert. A second poll re-detects (records another
// violation) without re-alerting, a clean second hub is unaffected, and the freeze
// survives a store reopen.
func TestPollHubFork(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	// A second, well-behaved hub to prove other hubs are unaffected by the freeze.
	hubB, err := s.UpsertHub(ctx, "sb1.iscc.id", "sb1.iscc.id/log", "https://sb1.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub B: %v", err)
	}

	// Seed the prior accepted checkpoint at size 10183 with a root that differs
	// from the real sb0 fixture root, so the live observation is a genuine fork.
	seedRoot := []byte("fork-seed-root-distinct-padding32")
	if string(seedRoot) == sb0FixtureRootB64 {
		t.Fatal("seed root must differ from the sb0 fixture root")
	}
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   10183,
		Root:       seedRoot,
		Raw:        []byte("sb0.iscc.id/log\n10183\nseed-root\n"),
		ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("seed RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, 10183); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}

	fetcher := sb0VerifiedFetcher(t)
	var alerts int
	alert := func(int64, string) { alerts++ }

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", sb0ObservedAt(), alert)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	// A violation freezes, never crashes: the signature was valid, so the verdict
	// is still StatusVerified with a nil error.
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified (violation is a separate axis)", status)
	}

	assertViolation(t, path, hubID, "fork")
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if !fs.Frozen {
		t.Errorf("Frozen = false after a fork, want true")
	}
	if fs.LastSize != 10183 {
		t.Errorf("LastSize = %d, want 10183 (a frozen hub must not advance)", fs.LastSize)
	}
	if alerts != 1 {
		t.Errorf("alerts = %d after first fork detection, want 1", alerts)
	}

	// Re-poll the already-frozen hub: re-detection is itself evidence, so a second
	// violation row is recorded, but the alert must not fire again.
	if _, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", sb0ObservedAt(), alert); err != nil {
		t.Fatalf("second PollHub: %v", err)
	}
	if n := countRows(t, path, "violations"); n != 2 {
		t.Errorf("violations after re-detection = %d, want 2 (re-detection is evidence)", n)
	}
	if alerts != 1 {
		t.Errorf("alerts after re-detection = %d, want 1 (exactly-one-alert)", alerts)
	}

	// A clean verified poll of the second hub advances normally and stays unfrozen.
	if _, err := PollHub(ctx, s, fetcher, hubB, "https://sb0.iscc.id", sb0ObservedAt(), noopAlert); err != nil {
		t.Fatalf("PollHub hub B: %v", err)
	}
	fsB, err := s.FollowState(ctx, hubB)
	if err != nil {
		t.Fatalf("FollowState hub B: %v", err)
	}
	if fsB.Frozen {
		t.Errorf("hub B frozen after freezing hub A, want unaffected")
	}
	if fsB.LastSize != 10183 {
		t.Errorf("hub B LastSize = %d, want 10183 (clean advance)", fsB.LastSize)
	}

	// Freeze + evidence survive a store reopen.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	s2, err := store.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })
	fs2, err := s2.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState after reopen: %v", err)
	}
	if !fs2.Frozen {
		t.Errorf("frozen not preserved across restart, want true")
	}
	if n := countRows(t, path, "violations"); n != 2 {
		t.Errorf("violations after reopen = %d, want 2 (evidence survives restart)", n)
	}
}

// TestPollHubShrink drives the shrink freeze path: the prior accepted size is
// seeded strictly larger (20000) than the sb0 fixture size (10183), so the live
// verified observation at 10183 is a strict tree-size decrease. The result is a
// "shrink" violation row, frozen=1, a cursor that does not advance, and exactly
// one alert.
func TestPollHubShrink(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	// Seed a prior accepted checkpoint at a size strictly larger than the fixture.
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID:      hubID,
		Status:     "verified",
		TreeSize:   20000,
		Root:       []byte("shrink-seed-root-distinct-pad-32"),
		Raw:        []byte("sb0.iscc.id/log\n20000\nseed-root\n"),
		ObservedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("seed RecordCheckpoint: %v", err)
	}
	if err := s.AdvanceFollowState(ctx, hubID, 20000); err != nil {
		t.Fatalf("seed AdvanceFollowState: %v", err)
	}

	fetcher := sb0VerifiedFetcher(t)
	var alerts int
	alert := func(int64, string) { alerts++ }

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", sb0ObservedAt(), alert)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified (violation is a separate axis)", status)
	}

	assertViolation(t, path, hubID, "shrink")
	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if !fs.Frozen {
		t.Errorf("Frozen = false after a shrink, want true")
	}
	if fs.LastSize != 20000 {
		t.Errorf("LastSize = %d, want 20000 (a frozen hub must not advance)", fs.LastSize)
	}
	if alerts != 1 {
		t.Errorf("alerts = %d after shrink detection, want 1", alerts)
	}
}

// assertViolation confirms a violation of the given kind exists for a hub by
// reading the violations table over an independent connection (observable
// evidence, not follower internals).
func assertViolation(t *testing.T, dbPath string, hubID int64, kind string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open inspector db: %v", err)
	}
	defer func() { _ = db.Close() }()
	var gotKind string
	err = db.QueryRow("SELECT kind FROM violations WHERE hub_id = ? LIMIT 1", hubID).Scan(&gotKind)
	if err != nil {
		t.Fatalf("read violation for hub %d (want kind %q): %v", hubID, kind, err)
	}
	if gotKind != kind {
		t.Errorf("violation kind = %q, want %q", gotKind, kind)
	}
}

// TestPollHubVerifiedAdvances drives one verified observation end-to-end: the
// composite fetcher serves the sb0 checkpoint and its matching did.json, so
// PollHub returns StatusVerified and the follow cursor advances to the sb0 fixture
// tree size.
func TestPollHubVerifiedAdvances(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := compositeFetcher{
		checkpoint: readCheckpoint(t, "sb0.iscc.id_checkpoint"),
		didDoc:     readFixture(t, "sb0.iscc.id_did.json"),
	}
	// observedAt inside sb0's validity window (the captured did.json is
	// unconstrained, so any time is in-window) yields StatusVerified, not Rotated.
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt, noopAlert)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified", status)
	}

	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if fs.LastSize != 10183 {
		t.Errorf("LastSize = %d, want 10183 (sb0 fixture)", fs.LastSize)
	}
	if fs.Frozen {
		t.Errorf("Frozen = true after a clean verified poll, want false")
	}
}

// TestPollHubUnverifiedDoesNotAdvance drives one non-verified observation: the
// did.json advertises a mismatching key, so the sb0 checkpoint's signature
// matches no listed key. PollHub returns StatusUnverified, persists nothing, and
// leaves the follow cursor at zero.
func TestPollHubUnverifiedDoesNotAdvance(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := compositeFetcher{
		checkpoint: readCheckpoint(t, "sb0.iscc.id_checkpoint"),
		didDoc:     didJSON(sb1Multibase),
	}
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt, noopAlert)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusUnverified {
		t.Fatalf("status = %s, want unverified", status)
	}

	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if fs.LastSize != 0 {
		t.Errorf("LastSize = %d, want 0 (non-verified must not advance)", fs.LastSize)
	}
}
