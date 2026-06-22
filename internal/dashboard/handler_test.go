// Golden test for the dashboard handler at the HTTP seam: it populates a fixture
// store with two hubs (one verified with coverage, one frozen) through the public
// store API, drives Handler over an httptest.ResponseRecorder, and asserts the
// observable response — 200, the text/html content type, and that the rendered
// body names every hub plus its glossary status. It also pins the method gate
// (non-GET → 405) and the exact-path gate (GET /unknown → 404), unit-tests the
// status mapping (including the inactive case the public store API cannot set
// directly), and proves the in-memory status overlay so a store-verified hub
// whose live verdict is unresolvable / unverified renders that status honestly.
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

// fakeStatusSource is an in-memory StatusSource keyed by hub_id, standing in for
// the metrics registry so the dashboard's status overlay is golden-testable
// without importing internal/metrics. A nil map (or a missing key) reports a miss.
type fakeStatusSource map[int64]string

func (f fakeStatusSource) Status(hubID int64) (string, bool) {
	s, ok := f[hubID]
	return s, ok
}

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
	// nil StatusSource: every store-verified hub renders as "verified" (no overlay),
	// so this test pins the store-provable subset exactly as before the overlay.
	Handler(st, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

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
	// Both store-provable statuses must render through the hubStatusBadge partial,
	// not as the bare status word: the wrapper, the fixed-table label, and the
	// per-status distinguishing silhouette marker must all appear (icon + label +
	// silhouette, never hue alone — ADR-0010 invariant 4).
	if !strings.Contains(body, `class="hub-status-badge"`) {
		t.Errorf("body missing hub-status-badge wrapper (partial not rendered)\n%s", body)
	}
	for _, want := range []string{
		`data-status="verified"`, ">Verified<", "M8.4 12.3", // verified: badge + label + check-circle marker
		`data-status="frozen"`, ">Frozen<", "M8.2 3.3h7.6", // frozen: badge + label + octagon-x marker
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing badge markup %q\n%s", want, body)
		}
	}
	// Coverage honesty is rendered, not just stored (ADR-0001): the verified hub
	// shows its recorded start size, the frozen hub (no coverage set) shows the
	// explicit "no coverage yet" — the observed size is never passed off as a
	// coverage guarantee.
	if !strings.Contains(body, "size 42") {
		t.Errorf("body missing coverage start for the verified hub\n%s", body)
	}
	if !strings.Contains(body, "no coverage yet") {
		t.Errorf("body missing the no-coverage state for the frozen hub\n%s", body)
	}

	// The Evidence-Ledger redress is observable at the seam: the page is styled
	// through the DS font tokens (so the type resolves to the self-hosted
	// webfonts) and lays out the ledger as a CSS grid, not an HTML <table>.
	for _, want := range []string{"var(--font-sans)", "var(--font-mono)", "display: grid"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing redress marker %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "<table") {
		t.Errorf("body still contains a <table> element; the grid redress is incomplete\n%s", body)
	}
}

// TestDashboardRendersInMemoryStatus proves the in-memory status overlay at the
// HTTP seam: a store-verified hub whose live verdict (from the StatusSource) is
// unresolvable / unverified renders that richer status — badge label + per-status
// silhouette marker — even though the store can only prove "verified". This is the
// end-to-end render of a status that exists ONLY in the in-memory registry, and is
// non-vacuous: stubbing the overlay so the live status is ignored leaves the page
// showing "verified" and fails these assertions.
func TestDashboardRendersInMemoryStatus(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "overlay.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Two store-verified hubs (active, not frozen), so the store proves "verified"
	// for both and the overlay is the only thing that can change the displayed status.
	unresolvableID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub unresolvable: %v", err)
	}
	unverifiedID, err := st.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub unverified: %v", err)
	}

	statuses := fakeStatusSource{
		unresolvableID: "unresolvable",
		unverifiedID:   "unverified",
	}

	rec := httptest.NewRecorder()
	Handler(st, statuses).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	// Both in-memory-only statuses must render through the badge partial: the
	// fixed-table label AND the per-status distinguishing silhouette marker
	// (badge.md: unresolvable M9.2 9.3, unverified M12 3.4 21 19H3z).
	for _, want := range []string{
		`data-status="unresolvable"`, ">Unresolvable<", "M9.2 9.3",
		`data-status="unverified"`, ">Unverified<", "M12 3.4 21 19H3z",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing in-memory status markup %q\n%s", want, body)
		}
	}
}

// TestDashboardLinksTokensNoCDN pins the shared-shell wiring: the rendered "/" body
// links the embedded DS token stylesheet at the literal web.TokensPath and carries
// no third-party CDN URL — the load-bearing M-UI invariant that every SSR body is
// complete with JavaScript disabled and references no third-party origin. The
// masthead "verify ↗ monitor.iscc.codes" link is the one intentional external
// https origin (the monitor-agnostic verifier app), so the ban is narrowed to
// third-party CDN hosts only, mirroring the certificate's no-CDN posture.
func TestDashboardLinksTokensNoCDN(t *testing.T) {
	st := fixtureStore(t)
	rec := httptest.NewRecorder()
	Handler(st, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `href="/_ds/tokens.css"`) {
		t.Errorf("body missing token stylesheet link\n%s", body)
	}
	// The intentional tier-2 verifier link must be present and same as the
	// certificate's; real CDN hosts must not.
	if !strings.Contains(body, "monitor.iscc.codes") {
		t.Errorf("body missing the tier-2 verify link\n%s", body)
	}
	for _, banned := range []string{"jsdelivr", "cdn.", "unpkg", "googleapis"} {
		if strings.Contains(body, banned) {
			t.Errorf("body contains external CDN reference %q\n%s", banned, body)
		}
	}
}

// TestDashboardRendersHeroAndNavigation pins the three headline landmark regions the
// mockup demands on "/": (a) the claim-lookup hero as a no-JS GET form whose action
// resolves to /inclusion/ with an iscc_id input, (b) a dossier link per hub row (the
// realm-index → dossier navigation closure: anchor count >= hub count), and (c) the
// masthead instance-identity block with the tier-2 verify link. It is non-vacuous:
// dropping the row <a> wrapper or the hero form fails these assertions.
func TestDashboardRendersHeroAndNavigation(t *testing.T) {
	st := fixtureStore(t)
	rec := httptest.NewRecorder()
	Handler(st, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	// (a) The claim-lookup hero: a no-JS GET form posting ?iscc_id=… to /inclusion/.
	for _, want := range []string{
		`<form`,
		`method="get"`,
		`action="/inclusion/"`,
		`name="iscc_id"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("hero form missing %q\n%s", want, body)
		}
	}

	// (b) Navigation closure: one dossier <a href="/<domain>"> per hub. Count the
	// dossier links specifically (not the masthead verify link or the hero button),
	// and require anchor count >= hub count (here both fixture hubs).
	dossierLinks := 0
	for _, domain := range []string{"sb0.iscc.id", "sb1.amlet.id"} {
		if strings.Contains(body, `href="/`+domain+`"`) {
			dossierLinks++
		}
	}
	if dossierLinks < 2 {
		t.Errorf("dossier link count = %d, want >= 2 (one per hub)\n%s", dossierLinks, body)
	}

	// (c) The masthead instance-identity block + the tier-2 verify link.
	for _, want := range []string{
		"monitor instance",
		"monitor.iscc.codes",
		"hubs followed",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("masthead/heading missing %q\n%s", want, body)
		}
	}
}

func TestDashboardMethodNotAllowed(t *testing.T) {
	st := fixtureStore(t)
	rec := httptest.NewRecorder()
	Handler(st, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST / status = %d, want 405", rec.Code)
	}
}

func TestDashboardUnknownPath(t *testing.T) {
	st := fixtureStore(t)
	rec := httptest.NewRecorder()
	Handler(st, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/unknown", nil))
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

// TestOverlayStatusPrecedence pins the load-bearing precedence: the store's durable
// inactive / frozen truths are never overridden by a fresher in-memory verdict, the
// overlay applies only to a store-verified hub, and only the two richer live states
// (unresolvable / unverified) are adopted — any other live value (or a nil source)
// keeps "verified".
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
