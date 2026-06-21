// Golden HTTP-seam test for the HTML log-browser route (GET /). It drives
// proofserve.Handler over the buildMirror fixture store and asserts the rendered
// page exposes the monitor's accepted checkpoint (size, root) — the store-provable
// state read back verbatim, never recomputed (the oracle gate is N/A here: no
// signature, RFC-6962, Merkle, did:web, fsck, or proof path). It also asserts the
// hub status renders through the five-status hubStatusBadge partial overlaid with the
// in-memory StatusSource verdict, mirroring dashboard.TestDashboardRendersInMemoryStatus.
// The mutation checks are that dropping the {{.Root}} or {{.Size}} cell from
// browser.html fails the golden body assert, and reverting serveBrowser to hubStatus(fs)
// instead of the overlay fails the unresolvable-overlay assert — proving both non-vacuous.
package proofserve

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/iscc/iscc-monitor/internal/store"
)

// TestBrowserExposesAcceptedCheckpoint asserts GET / returns 200 text/html and the
// body contains the accepted size and the base64-Std encoding of the accepted root.
func TestBrowserExposesAcceptedCheckpoint(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}

	body := rec.Body.String()
	wantSize := strconv.FormatUint(m.size, 10)
	if !strings.Contains(body, wantSize) {
		t.Errorf("body missing accepted size %s\n%s", wantSize, body)
	}
	wantRoot := base64.StdEncoding.EncodeToString(m.tree.Hash())
	if !strings.Contains(body, wantRoot) {
		t.Errorf("body missing accepted root %s\n%s", wantRoot, body)
	}
	// The page must advertise the proof surface so a client can discover it.
	for _, link := range []string{"entries?index=0", "inclusion?iscc_id", "consistency?from=0", "verify?iscc_id"} {
		if !strings.Contains(body, link) {
			t.Errorf("body missing proof link %q", link)
		}
	}
}

// TestBrowserNonGET asserts a non-GET method to the bare / is a 405 (the shared
// method-gate at the top of Handler covers it).
func TestBrowserNonGET(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID, nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// TestBrowserNoAcceptedCheckpoint asserts a followed-but-unpolled hub (LastSize == 0)
// renders a 200 "no accepted checkpoint yet" page, not a 404 — the browser exists for
// a hub before coverage starts, never implying a pre-coverage guarantee (ADR-0001).
func TestBrowserNoAcceptedCheckpoint(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	h := Handler(st, hubID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No accepted checkpoint yet") {
		t.Errorf("body missing no-coverage state\n%s", body)
	}
}

// fakeStatusSource is an in-memory StatusSource keyed by hub_id, standing in for the
// metrics registry so the log browser's status overlay is golden-testable without
// importing internal/metrics. A nil map (or a missing key) reports a miss.
type fakeStatusSource map[int64]string

func (f fakeStatusSource) Status(hubID int64) (string, bool) {
	s, ok := f[hubID]
	return s, ok
}

// TestBrowserRendersInMemoryStatus proves the in-memory status overlay reaches the
// log-browser page: a store-verified hub whose live verdict (from the StatusSource)
// is unresolvable renders that richer status through the hubStatusBadge partial —
// badge wrapper + label + the per-status silhouette marker — even though the store
// can only prove "verified". It is non-vacuous: reverting serveBrowser to
// hubStatus(fs) instead of overlayStatus renders data-status="verified" and fails
// these assertions (mirroring dashboard.TestDashboardRendersInMemoryStatus).
func TestBrowserRendersInMemoryStatus(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	statuses := fakeStatusSource{m.hubID: "unresolvable"}
	h := Handler(m.store, m.hubID, statuses)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// The in-memory-only status must render through the badge partial: the wrapper,
	// the fixed-table label, and the unresolvable silhouette marker (badge.md: M9.2 9.3).
	for _, want := range []string{
		`class="hub-status-badge"`,
		`data-status="unresolvable"`,
		">Unresolvable<",
		"M9.2 9.3",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing overlaid status markup %q\n%s", want, body)
		}
	}
	// And the store-only "verified" must NOT be the rendered status (the overlay won).
	if strings.Contains(body, `data-status="verified"`) {
		t.Errorf("body still renders store-only verified status; overlay did not apply\n%s", body)
	}
}
