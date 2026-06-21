// Golden HTTP-seam test for the per-hub dossier route (GET /<domain>). It drives
// dossier.Handler over a fixture store (the hub registered through the public store
// API) and asserts the observable response — 200 text/html, the Evidence-Ledger
// DS-shell wiring (token + font stylesheet links, DS font tokens, no <table>, no
// external CDN URL), the hub's domain rendered through the five-status hubStatusBadge
// partial overlaid with the in-memory StatusSource verdict, and ADR-0001 coverage
// honesty (a covered hub shows "size N at <RFC3339>"; an uncovered hub shows the
// explicit "no coverage yet", never a fabricated size+time). The oracle gate is N/A
// here: no signature, RFC-6962, Merkle, did:web, fsck, or proof path.
package dossier

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

// fakeStatusSource is an in-memory StatusSource keyed by hub_id, standing in for the
// metrics registry so the dossier's status overlay is golden-testable without
// importing internal/metrics. A nil map (or a missing key) reports a miss.
type fakeStatusSource map[int64]string

func (f fakeStatusSource) Status(hubID int64) (string, bool) {
	s, ok := f[hubID]
	return s, ok
}

// coveredHub opens a fresh store and registers one verified hub with an accepted
// checkpoint (which advances last_size and sets the coverage start). It returns the
// store and the hub_id; the caller asserts on the rendered dossier.
func coveredHub(t *testing.T) (*store.Store, int64) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "dossier.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	id, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	observed := time.Unix(1_700_000_000, 0).UTC()
	if err := st.AdvanceAccepted(ctx, store.CheckpointRecord{
		HubID:      id,
		TreeSize:   42,
		Root:       []byte("root-verified-32-bytes-padding!!"),
		Raw:        []byte("raw-checkpoint-bytes"),
		ObservedAt: observed,
	}); err != nil {
		t.Fatalf("AdvanceAccepted: %v", err)
	}
	return st, id
}

// frozenHub opens a fresh store, registers one hub, records two self-consistency
// violations (a fork then a later shrink), and freezes it — the state the dossier's
// non-dismissable Exhibit surfaces (ADR-0006). It returns the store and the hub_id.
func frozenHub(t *testing.T) (*store.Store, int64) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "frozen.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	id, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}
	if _, err := st.RecordViolation(ctx, store.Violation{
		HubID: id, Kind: "fork", RawA: []byte("a"), RawB: []byte("b"),
		DetectedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordViolation fork: %v", err)
	}
	if _, err := st.RecordViolation(ctx, store.Violation{
		HubID: id, Kind: "shrink", RawA: []byte("c"), RawB: []byte("d"),
		DetectedAt: time.Unix(1_700_009_999, 0),
	}); err != nil {
		t.Fatalf("RecordViolation shrink: %v", err)
	}
	if err := st.Freeze(ctx, id); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	return st, id
}

// TestDossierFrozenExhibit asserts a frozen hub's dossier renders the
// categorically-distinct, non-dismissable Exhibit panel (ADR-0006): the distinct
// panel class/heading, the literal "do not trust new state" notice, each violation
// kind + its detected-at, AND the frozen badge silhouette marker. There must be no
// dismiss/close control (non-dismissable: no <button>, no JS).
func TestDossierFrozenExhibit(t *testing.T) {
	st, id := frozenHub(t)
	rec := httptest.NewRecorder()
	Handler(st, id, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	// The frozen badge silhouette marker must be present (the five-status badge).
	if !strings.Contains(body, "M8.2 3.3h7.6") {
		t.Errorf("body missing the frozen badge silhouette marker\n%s", body)
	}
	// The Exhibit panel: its distinct class + heading, and the non-dismissable
	// "do not trust new state" notice (case-insensitive on the copy).
	for _, want := range []string{
		`class="exhibit"`,
		"Exhibit",
		"self-consistency violation",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing Exhibit marker %q\n%s", want, body)
		}
	}
	if !strings.Contains(strings.ToLower(body), "do not trust new state") {
		t.Errorf("body missing the non-dismissable \"do not trust new state\" notice\n%s", body)
	}
	// Each violation's kind and detected-at (newest-first: shrink then fork).
	for _, want := range []string{
		"shrink", "fork",
		"2023-11-14T22:13:20Z", // fork detected_at (earlier)
		"2023-11-15T00:59:59Z", // shrink detected_at (later)
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing violation detail %q\n%s", want, body)
		}
	}
	// Non-dismissable: no dismiss/close control and no JS. The HTML `hidden`
	// attribute (`hidden>` / `hidden=`) is banned specifically — not the CSS
	// `overflow: hidden`, which is a layout property, not a visibility toggle.
	for _, banned := range []string{"<button", "<script", " hidden>", " hidden="} {
		if strings.Contains(body, banned) {
			t.Errorf("Exhibit must be non-dismissable; body contains %q\n%s", banned, body)
		}
	}
}

// TestDossierNoExhibitWhenNotFrozen asserts a verified (non-frozen) hub renders NO
// Exhibit panel: neither the distinct panel class nor the "do not trust new state"
// copy may appear, so the Exhibit is categorically reserved for the frozen state.
func TestDossierNoExhibitWhenNotFrozen(t *testing.T) {
	st, id := coveredHub(t)
	rec := httptest.NewRecorder()
	Handler(st, id, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, banned := range []string{`class="exhibit"`} {
		if strings.Contains(body, banned) {
			t.Errorf("verified dossier renders an Exhibit marker %q\n%s", banned, body)
		}
	}
	if strings.Contains(strings.ToLower(body), "do not trust new state") {
		t.Errorf("verified dossier renders the frozen \"do not trust new state\" copy\n%s", body)
	}
}

// TestDossierRendersCoveredHub asserts GET /<domain> returns 200 text/html with the
// Evidence-Ledger DS shell, the hub's domain rendered through the hubStatusBadge
// partial, the honest coverage window for a covered hub, and no <table> / CDN URL.
func TestDossierRendersCoveredHub(t *testing.T) {
	st, id := coveredHub(t)
	rec := httptest.NewRecorder()
	// nil StatusSource: a store-verified hub renders as "verified" (no overlay).
	Handler(st, id, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	body := rec.Body.String()

	// The DS shell is linked and the type resolves through the DS font tokens.
	for _, want := range []string{
		`href="/_ds/tokens.css"`,
		`href="/_ds/fonts.css"`,
		"var(--font-sans)",
		"var(--font-mono)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing DS-shell marker %q\n%s", want, body)
		}
	}
	// The hub's domain and origin must appear.
	for _, want := range []string{"sb0.iscc.id", "sb0.iscc.id/log"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n%s", want, body)
		}
	}
	// The status renders through the five-status badge partial: the wrapper, the
	// data-status, the fixed-table label, and the verified silhouette marker.
	for _, want := range []string{
		`class="hub-status-badge"`,
		`data-status="verified"`,
		">Verified<",
		"M8.4 12.3",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing badge markup %q\n%s", want, body)
		}
	}
	// Coverage honesty (ADR-0001): the covered hub shows its recorded start size and
	// RFC-3339 time, never the observed last_size as a coverage guarantee.
	for _, want := range []string{"size 42", "at 2023-11-14T22:13:20Z"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing coverage window marker %q\n%s", want, body)
		}
	}
	// The dossier is a card/grid layout, never an HTML <table>.
	if strings.Contains(body, "<table") {
		t.Errorf("body contains a <table> element; the ledger card layout is incomplete\n%s", body)
	}
	// No external CDN URL may appear in the body — same-origin/relative hrefs only.
	for _, banned := range []string{"jsdelivr", "http://", "https://", "cdn."} {
		if strings.Contains(body, banned) {
			t.Errorf("body contains external CDN reference %q\n%s", banned, body)
		}
	}
}

// TestDossierCoverageHonestyNoCoverage asserts a followed-but-unpolled hub (no
// coverage set) renders the explicit "no coverage yet" state, never a fabricated
// size+time — the ADR-0001 coverage-honesty rule at the dossier seam.
func TestDossierCoverageHonestyNoCoverage(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "uncovered.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	id, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	rec := httptest.NewRecorder()
	Handler(st, id, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "no coverage yet") {
		t.Errorf("body missing the no-coverage state\n%s", body)
	}
	// The coverage cell must not fabricate a "size N" guarantee: the covered branch
	// renders `row-mono">size N`, which must be absent for an uncovered hub.
	if strings.Contains(body, `row-mono">size `) {
		t.Errorf("uncovered hub renders a fabricated coverage size\n%s", body)
	}
}

// TestDossierRendersInMemoryStatus proves the in-memory status overlay reaches the
// dossier: a store-verified hub whose live verdict (from the StatusSource) is
// unresolvable renders that richer status through the hubStatusBadge partial — the
// wrapper, the fixed-table label, and the unresolvable silhouette marker — even
// though the store can only prove "verified". The store-only "verified" must NOT be
// the rendered status (the overlay won), so the body carries no data-status="verified".
func TestDossierRendersInMemoryStatus(t *testing.T) {
	st, id := coveredHub(t)
	statuses := fakeStatusSource{id: "unresolvable"}
	rec := httptest.NewRecorder()
	Handler(st, id, statuses).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
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
	if strings.Contains(body, `data-status="verified"`) {
		t.Errorf("body still renders store-only verified status; overlay did not apply\n%s", body)
	}
}

// TestDossierNonGET asserts a non-GET method is a 405 (the shared method gate at the
// top of Handler covers it).
func TestDossierNonGET(t *testing.T) {
	st, id := coveredHub(t)
	rec := httptest.NewRecorder()
	Handler(st, id, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sb0.iscc.id", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// TestDossierHubNotInStore asserts a hub_id with no matching store summary is a 500
// (a real store inconsistency — the binary always registers the hub before mounting
// this handler), never a 404.
func TestDossierHubNotInStore(t *testing.T) {
	st, id := coveredHub(t)
	rec := httptest.NewRecorder()
	Handler(st, id+999, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// TestHubStatusMapping pins the store-provable glossary mapping, including the
// inactive case the public store API cannot set (active is registry-managed; no
// public setter exists, so it is exercised at the summary boundary the renderer
// consumes), mirroring dashboard.TestHubStatusMapping.
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

// TestOverlayStatusPrecedence pins the load-bearing precedence: the store's durable
// inactive / frozen truths are never overridden by a fresher in-memory verdict, the
// overlay applies only to a store-verified hub, and only the two richer live states
// (unresolvable / unverified) are adopted — any other live value (or a nil source)
// keeps "verified". It mirrors dashboard.TestOverlayStatusPrecedence.
func TestOverlayStatusPrecedence(t *testing.T) {
	verified := store.HubSummary{HubID: 1, Active: true, Frozen: false}
	frozen := store.HubSummary{HubID: 1, Active: true, Frozen: true}
	inactive := store.HubSummary{HubID: 1, Active: false, Frozen: false}

	cases := []struct {
		name     string
		in       store.HubSummary
		statuses StatusSource
		want     string
	}{
		{"verified overlaid to unresolvable", verified, fakeStatusSource{1: "unresolvable"}, "unresolvable"},
		{"verified overlaid to unverified", verified, fakeStatusSource{1: "unverified"}, "unverified"},
		{"verified stays when live is verified", verified, fakeStatusSource{1: "verified"}, "verified"},
		{"verified stays when no live record", verified, fakeStatusSource{}, "verified"},
		{"verified stays with nil source", verified, nil, "verified"},
		{"frozen wins over live unresolvable", frozen, fakeStatusSource{1: "unresolvable"}, "frozen"},
		{"inactive wins over live unverified", inactive, fakeStatusSource{1: "unverified"}, "inactive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := overlayStatus(tc.in, tc.statuses); got != tc.want {
				t.Errorf("overlayStatus(%+v, %v) = %q, want %q", tc.in, tc.statuses, got, tc.want)
			}
		})
	}
}
