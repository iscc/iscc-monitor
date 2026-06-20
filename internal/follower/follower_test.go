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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
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

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt)
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

	status, err := PollHub(ctx, s, fetcher, hubID, "https://sb0.iscc.id", observedAt)
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
