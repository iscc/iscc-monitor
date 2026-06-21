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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/metrics"
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

// countingFetcher wraps a Fetcher and tallies how many times a did.json URL is
// fetched, so a test can prove the cache-hit fast path skips the second did.json
// resolution. It counts only did.json (the did:web resolution) — checkpoint
// fetches are not the thing under test.
type countingFetcher struct {
	inner    logclient.Fetcher
	didFetch int
}

func (f *countingFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	if strings.HasSuffix(url, "did.json") {
		f.didFetch++
	}
	return f.inner.Fetch(ctx, url)
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

// openTemp opens a fresh store in a temp dir, registers cleanup, and returns the
// store alongside its file path so a test can open an independent inspector
// connection (countRows/readHubKey) against the same WAL file.
func openTemp(t *testing.T) (*store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "testnet.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

// readHubKey reads the (key_id, pubkey_raw) of a hub's single cached key over an
// independent read-only connection, so the assertion pins the observable cache row
// without reaching into follower or store internals.
func readHubKey(t *testing.T, dbPath string, hubID int64) (uint32, []byte) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open inspector db: %v", err)
	}
	defer func() { _ = db.Close() }()
	var (
		keyID  int64
		pubkey []byte
	)
	err = db.QueryRow("SELECT key_id, pubkey_raw FROM hub_keys WHERE hub_id = ? LIMIT 1", hubID).Scan(&keyID, &pubkey)
	if err != nil {
		t.Fatalf("read hub_key for hub %d: %v", hubID, err)
	}
	return uint32(keyID), pubkey
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

// assertMetric confirms the registry's rendered Prometheus output contains an
// exact sample line, asserting on the observable rendered text (Registry.String())
// rather than on any follower or registry internal.
func assertMetric(t *testing.T, reg *metrics.Registry, line string) {
	t.Helper()
	rendered := reg.String()
	if !strings.Contains(rendered, line) {
		t.Errorf("metrics output missing %q\n--- rendered ---\n%s", line, rendered)
	}
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
	reg := metrics.New()

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", sb0ObservedAt(), alert, reg)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	// A violation freezes, never crashes: the signature was valid, so the verdict
	// is still StatusVerified with a nil error.
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified (violation is a separate axis)", status)
	}

	// The freeze fires the violations counter (keyed on the real kind) and maps the
	// hub status to the glossary "frozen" label, not StatusVerified's "verified".
	assertMetric(t, reg, `iscc_monitor_violations_total{hub_id="1",kind="fork"} 1`)
	assertMetric(t, reg, `iscc_monitor_hub_status{hub_id="1",status="frozen"} 1`)

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

	// A frozen/violating observation must not start coverage from the contradictory
	// checkpoint: the seed used RecordCheckpoint/AdvanceFollowState, not SetCoverage.
	cov, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.Set {
		t.Errorf("coverage Set after a fork freeze, want unset (a violation must not start coverage)")
	}

	// A frozen/violating observation must not cache a key either: the key cache is
	// on the verified, non-violation path, never inside freeze.
	if n := countRows(t, path, "hub_keys"); n != 0 {
		t.Errorf("hub_keys rows after a fork freeze = %d, want 0 (a violation must not cache a key)", n)
	}

	// Re-poll the already-frozen hub: re-detection is itself evidence, so a second
	// violation row is recorded, but the alert must not fire again.
	if _, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", sb0ObservedAt(), alert, nil); err != nil {
		t.Fatalf("second PollHub: %v", err)
	}
	if n := countRows(t, path, "violations"); n != 2 {
		t.Errorf("violations after re-detection = %d, want 2 (re-detection is evidence)", n)
	}
	if alerts != 1 {
		t.Errorf("alerts after re-detection = %d, want 1 (exactly-one-alert)", alerts)
	}

	// A clean verified poll of the second hub advances normally and stays unfrozen.
	if _, err := PollHub(ctx, s, fetcher, hubB, "https://sb0.iscc.id", sb0ObservedAt(), noopAlert, nil); err != nil {
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

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", sb0ObservedAt(), alert, nil)
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

	// A frozen/violating observation must not cache a key (verified-path only).
	if n := countRows(t, path, "hub_keys"); n != 0 {
		t.Errorf("hub_keys rows after a shrink freeze = %d, want 0 (a violation must not cache a key)", n)
	}
}

// TestGlossaryStatus is the golden table for the verdict -> glossary-status
// mapper. It pins the load-bearing remaps: a frozen observation is "frozen" even
// though its enum is StatusVerified, and StatusRotated folds into "unverified"
// (the glossary has no "rotated") — both distinct from logclient.Status.String().
func TestGlossaryStatus(t *testing.T) {
	cases := []struct {
		name   string
		status logclient.Status
		frozen bool
		want   string
	}{
		{"verified", logclient.StatusVerified, false, "verified"},
		{"verified-but-frozen", logclient.StatusVerified, true, "frozen"},
		{"unverified", logclient.StatusUnverified, false, "unverified"},
		{"unresolvable", logclient.StatusUnresolvable, false, "unresolvable"},
		{"rotated folds into unverified", logclient.StatusRotated, false, "unverified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := glossaryStatus(tc.status, tc.frozen); got != tc.want {
				t.Errorf("glossaryStatus(%v, frozen=%v) = %q, want %q", tc.status, tc.frozen, got, tc.want)
			}
		})
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
	s, path := openTemp(t)
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
	reg := metrics.New()

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt, noopAlert, reg)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusVerified {
		t.Fatalf("status = %s, want verified", status)
	}

	// The verified verdict maps to the glossary "verified" status and records the
	// observation timestamp (observedAt.Unix(), a non-zero value).
	assertMetric(t, reg, `iscc_monitor_hub_status{hub_id="1",status="verified"} 1`)
	assertMetric(t, reg, fmt.Sprintf(`iscc_monitor_last_observed_at{hub_id="1"} %d`, observedAt.Unix()))

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

	// The resolved did:web key is cached exactly once: one hub_keys row, keyed by
	// sb0's signed-note keyhash (0x40b74463), with the 32-byte Ed25519 pubkey.
	if n := countRows(t, path, "hub_keys"); n != 1 {
		t.Errorf("hub_keys rows = %d, want 1 (a verified poll caches the key)", n)
	}
	keyID, pubkey := readHubKey(t, path, hubID)
	if keyID != 0x40b74463 {
		t.Errorf("hub_keys key_id = %08x, want 40b74463 (sb0 signed-note keyhash)", keyID)
	}
	if len(pubkey) != 32 {
		t.Errorf("hub_keys pubkey_raw = %d bytes, want 32 (Ed25519 key)", len(pubkey))
	}

	// A second verified poll refreshes the same key in place: still exactly one row.
	if _, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt, noopAlert, nil); err != nil {
		t.Fatalf("second PollHub for key cache: %v", err)
	}
	if n := countRows(t, path, "hub_keys"); n != 1 {
		t.Errorf("hub_keys rows after second poll = %d, want 1 (refresh in place)", n)
	}

	// Coverage is recorded on the first verified observation at the fixture size and
	// the observed time (ADR-0001, coverage honesty).
	cov, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if !cov.Set {
		t.Fatalf("coverage Set = false after a verified poll, want true")
	}
	if cov.Size != 10183 {
		t.Errorf("coverage Size = %d, want 10183 (sb0 fixture)", cov.Size)
	}
	if !cov.Since.Equal(observedAt) {
		t.Errorf("coverage Since = %v, want %v (the observed time)", cov.Since, observedAt)
	}

	// A second verified poll at a later time does not move the immutable start.
	later := observedAt.Add(24 * time.Hour)
	if _, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", later, noopAlert, nil); err != nil {
		t.Fatalf("second PollHub: %v", err)
	}
	cov2, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage after second poll: %v", err)
	}
	if cov2.Size != 10183 || !cov2.Since.Equal(observedAt) {
		t.Errorf("coverage moved on a second poll: got (size %d, since %v), want (10183, %v)",
			cov2.Size, cov2.Since, observedAt)
	}
}

// TestPollHubUnverifiedDoesNotAdvance drives one non-verified observation: the
// did.json advertises a mismatching key, so the sb0 checkpoint's signature
// matches no listed key. PollHub returns StatusUnverified, persists nothing, and
// leaves the follow cursor at zero.
func TestPollHubUnverifiedDoesNotAdvance(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := compositeFetcher{
		checkpoint: readCheckpoint(t, "sb0.iscc.id_checkpoint"),
		didDoc:     didJSON(sb1Multibase),
	}
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	reg := metrics.New()

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt, noopAlert, reg)
	if err != nil {
		t.Fatalf("PollHub: %v", err)
	}
	if status != logclient.StatusUnverified {
		t.Fatalf("status = %s, want unverified", status)
	}

	// A non-verified verdict still records the glossary status (here "unverified")
	// and the observation timestamp on the early-return verdict path.
	assertMetric(t, reg, `iscc_monitor_hub_status{hub_id="1",status="unverified"} 1`)

	fs, err := s.FollowState(ctx, hubID)
	if err != nil {
		t.Fatalf("FollowState: %v", err)
	}
	if fs.LastSize != 0 {
		t.Errorf("LastSize = %d, want 0 (non-verified must not advance)", fs.LastSize)
	}

	// A non-verified verdict must not start coverage.
	cov, err := s.Coverage(ctx, hubID)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.Set {
		t.Errorf("coverage Set after an unverified poll, want unset (only a clean verified observation starts coverage)")
	}

	// A non-verified verdict must not cache a key either (the key cache is on the
	// verified, non-violation path only).
	if n := countRows(t, path, "hub_keys"); n != 0 {
		t.Errorf("hub_keys rows after an unverified poll = %d, want 0", n)
	}
}

// TestPollHubCacheHitSkipsDidFetch proves the warm-cache fast path: across two
// verified polls of the same hub, the first (cold cache) resolves did.json twice
// (once in AcceptCheckpoint, once in cacheHubKey's miss-path resolve), and the
// second (warm cache) resolves it only once (AcceptCheckpoint), because cacheHubKey
// recovers the key id from the raw checkpoint, hits store.LookupHubKey, and
// refreshes the cached row in place WITHOUT a second ResolveVerifierKey. The
// standing invariants — exactly one hub_keys row, key_id 0x40b74463, 32-byte pubkey
// — are re-asserted to confirm the fast path refreshes rather than duplicates or
// drops the key.
func TestPollHubCacheHitSkipsDidFetch(t *testing.T) {
	ctx := context.Background()
	s, path := openTemp(t)
	hubID, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	fetcher := &countingFetcher{inner: sb0VerifiedFetcher(t)}
	observedAt := sb0ObservedAt()

	// First poll (cold cache): AcceptCheckpoint resolves did.json once and the
	// cache miss path resolves it again -> two did.json fetches total.
	if status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt, noopAlert, nil); err != nil {
		t.Fatalf("first PollHub: %v", err)
	} else if status != logclient.StatusVerified {
		t.Fatalf("first poll status = %s, want verified", status)
	}
	coldFetches := fetcher.didFetch
	if coldFetches != 2 {
		t.Errorf("did.json fetches after cold poll = %d, want 2 (AcceptCheckpoint + cache miss resolve)", coldFetches)
	}

	// Second poll (warm cache): AcceptCheckpoint resolves did.json once, but the
	// cache hit in cacheHubKey must NOT resolve again -> exactly +1 fetch, not +2.
	if status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt, noopAlert, nil); err != nil {
		t.Fatalf("second PollHub: %v", err)
	} else if status != logclient.StatusVerified {
		t.Fatalf("second poll status = %s, want verified", status)
	}
	if got := fetcher.didFetch - coldFetches; got != 1 {
		t.Errorf("did.json fetches during warm poll = %d, want 1 (only AcceptCheckpoint; cacheHubKey hit the cache)", got)
	}

	// The fast path refreshes in place: still exactly one row, same key id and pubkey.
	if n := countRows(t, path, "hub_keys"); n != 1 {
		t.Errorf("hub_keys rows after warm poll = %d, want 1 (fast path refreshes, never duplicates)", n)
	}
	keyID, pubkey := readHubKey(t, path, hubID)
	if keyID != 0x40b74463 {
		t.Errorf("hub_keys key_id = %08x, want 40b74463 (sb0 signed-note keyhash)", keyID)
	}
	if len(pubkey) != 32 {
		t.Errorf("hub_keys pubkey_raw = %d bytes, want 32 (Ed25519 key)", len(pubkey))
	}
}
