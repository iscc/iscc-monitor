// Golden test for the dashboard handler at the HTTP seam: it populates a fixture
// store with two hubs (one verified with coverage, one frozen) through the public
// store API, drives Handler over an httptest.ResponseRecorder, and asserts the
// observable response — 200, the text/html content type, and that the rendered
// body names every hub plus its store-provable glossary status. It also pins the
// method gate (non-GET → 405) and the exact-path gate (GET /unknown → 404), and
// unit-tests the status mapping (including the inactive case the public store API
// cannot set directly).
package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/store"
)

// fixtureStore opens a fresh store and registers two hubs: a verified one with an
// accepted checkpoint and a recorded coverage start, and a frozen one. It returns
// the store; the caller asserts on the rendered output.
func fixtureStore(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "dash.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Verified hub: an accepted checkpoint advances last_size and sets coverage.
	verifiedID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub verified: %v", err)
	}
	observed := time.Unix(1_700_000_000, 0).UTC()
	if err := st.AdvanceAccepted(ctx, store.CheckpointRecord{
		HubID:      verifiedID,
		TreeSize:   42,
		Root:       []byte("root-verified-32-bytes-padding!!"),
		Raw:        []byte("raw-checkpoint-bytes"),
		ObservedAt: observed,
	}); err != nil {
		t.Fatalf("AdvanceAccepted verified: %v", err)
	}

	// Frozen hub: a follow_state row advanced then frozen by the freeze path.
	frozenID, err := st.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub frozen: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, frozenID, 7); err != nil {
		t.Fatalf("AdvanceFollowState frozen: %v", err)
	}
	if err := st.Freeze(ctx, frozenID); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	return st
}

func TestDashboardRendersEveryHub(t *testing.T) {
	st := fixtureStore(t)
	rec := httptest.NewRecorder()
	Handler(st).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	body := rec.Body.String()

	// Every hub's domain and origin must appear.
	for _, want := range []string{
		"sb0.iscc.id", "sb0.iscc.id/log",
		"sb1.amlet.id", "sb1.amlet.id/log",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n%s", want, body)
		}
	}
	// Both store-provable statuses must render.
	if !strings.Contains(body, "verified") {
		t.Errorf("body missing verified status\n%s", body)
	}
	if !strings.Contains(body, "frozen") {
		t.Errorf("body missing frozen status\n%s", body)
	}
	// The verified hub recorded coverage; the page must show its start size.
	if !strings.Contains(body, "size 42") {
		t.Errorf("body missing coverage start for the verified hub\n%s", body)
	}
}

func TestDashboardMethodNotAllowed(t *testing.T) {
	st := fixtureStore(t)
	rec := httptest.NewRecorder()
	Handler(st).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST / status = %d, want 405", rec.Code)
	}
}

func TestDashboardUnknownPath(t *testing.T) {
	st := fixtureStore(t)
	rec := httptest.NewRecorder()
	Handler(st).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /unknown status = %d, want 404", rec.Code)
	}
}

// TestHubStatusMapping pins the store-provable glossary mapping, including the
// inactive case the public store API cannot set (active is registry-managed; no
// public setter exists, so it is exercised at the summary boundary the renderer
// consumes).
func TestHubStatusMapping(t *testing.T) {
	cases := []struct {
		name string
		in   store.HubSummary
		want string
	}{
		{"inactive wins over frozen", store.HubSummary{Active: false, Frozen: true}, "inactive"},
		{"frozen when active", store.HubSummary{Active: true, Frozen: true}, "frozen"},
		{"verified default", store.HubSummary{Active: true, Frozen: false}, "verified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hubStatus(tc.in); got != tc.want {
				t.Errorf("hubStatus(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
