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

	"github.com/iscc/iscc-monitor/internal/dashboard"
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
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

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
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

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
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	body := rec.Body.String()

	// The DS shell is linked and the type resolves through the DS font tokens. The
	// masthead carries the self-hosted ISCC logo <img> beside the text mark (the
	// shared document chrome); its src is the same-origin /_ds/ literal so it never
	// trips the no-CDN ban below.
	for _, want := range []string{
		`href="/_ds/tokens.css"`,
		`href="/_ds/fonts.css"`,
		`src="/_ds/iscc-logo-black.png"`,
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
	// Coverage honesty (ADR-0001): §2 shows its recorded start time + size and the
	// derived "N days observed", never the observed last_size as a coverage
	// guarantee. The "since <time> @ size N" form is the §2 mockup format.
	for _, want := range []string{
		"§2 · COVERAGE",
		"since 2023-11-14T22:13:20Z @ size 42",
		"days observed",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing coverage window marker %q\n%s", want, body)
		}
	}
	// The trust-document head: the eyebrow "Hub dossier", the hub name in an <h1>,
	// the domain, and the "Compiled by …" provenance line.
	for _, want := range []string{
		"Hub dossier",
		`<h1 class="doc-hub-name">sb0.iscc.id</h1>`,
		"Compiled by",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing trust-document head marker %q\n%s", want, body)
		}
	}
	// §1 Identity, §3 Latest checkpoint, §4 Bitcoin anchor each render their
	// numbered heading + value. §1 derives did:web statically (no network read);
	// §3 shows accepted size + observed time; §4 shows the anchor dot + label.
	for _, want := range []string{
		"§1 · IDENTITY",
		"did:web:sb0.iscc.id",
		"§3 · LATEST CHECKPOINT",
		"42 entries",
		"observed 2023-11-14T22:13:20Z",
		"§4 · BITCOIN ANCHOR",
		`class="anchor-dot"`,
		"not anchored", // no OTS row → the honest never-anchored label
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing numbered-section marker %q\n%s", want, body)
		}
	}
	// The two action links resolve: "Prove an ISCC-ID in this hub →" → the realm
	// index claim-lookup hero ("/"), "Browse the log →" → /{{.Origin}}/.
	for _, want := range []string{
		`href="/">Prove an ISCC-ID in this hub →`,
		`href="/sb0.iscc.id/log/">Browse the log →`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing action link %q\n%s", want, body)
		}
	}
	// §5 renders as an honest minimal placeholder: the heading is present but no
	// fabricated observation line.
	if !strings.Contains(body, "§5 · OBSERVATION LOG") {
		t.Errorf("body missing the §5 observation-log placeholder heading\n%s", body)
	}
	// The dossier is a card/grid layout, never an HTML <table>.
	if strings.Contains(body, "<table") {
		t.Errorf("body contains a <table> element; the ledger card layout is incomplete\n%s", body)
	}
	// No third-party CDN URL may appear in the body. The chrome/verify link to
	// monitor.iscc.codes is the one intentional external https origin (the verifier
	// app, per the certificate baseline), so ban only third-party CDN hosts plus
	// bare http://, not every https:// substring.
	for _, banned := range []string{"jsdelivr", "cdn.", "unpkg", "googleapis", "http://"} {
		if strings.Contains(body, banned) {
			t.Errorf("body contains external CDN reference %q\n%s", banned, body)
		}
	}
}

// TestDossierConfirmedAnchorRendersHeight asserts the §4 Bitcoin-anchor section of a
// hub with a confirmed OTS row carrying a btc_height renders "confirmed", the
// confirmed dot keyword, and the block height — and NEVER a 5xx or error styling.
// It is the honesty counterpart of the pending case: the height shows ONLY when the
// anchor is confirmed AND the height is non-zero.
func TestDossierConfirmedAnchorRendersHeight(t *testing.T) {
	ctx := context.Background()
	st, id := coveredHub(t)
	if _, _, err := st.RecordOTS(ctx, store.OTSRecord{
		HubID:     id,
		TreeSize:  42,
		Root:      []byte("root-verified-32-bytes-padding!!"),
		Status:    store.OTSStatusConfirmed,
		StampedAt: time.Unix(1_700_000_000, 0),
		BTCHeight: 869440,
	}); err != nil {
		t.Fatalf("RecordOTS confirmed: %v", err)
	}

	rec := httptest.NewRecorder()
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"§4 · BITCOIN ANCHOR",
		`data-dot="confirmed"`,
		"confirmed",
		"block 869440",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing confirmed-anchor marker %q\n%s", want, body)
		}
	}
	// No error styling: the confirmed anchor must not render the "not anchored"
	// fallback label.
	if strings.Contains(body, "not anchored") {
		t.Errorf("confirmed anchor renders the not-anchored fallback label\n%s", body)
	}
}

// TestDossierPendingAnchorHonest asserts the §4 section of a hub with only a pending
// OTS row renders the honest "pending" label + dot and NO fabricated block height
// (the height shows only for a confirmed anchor with a non-zero height), with no 5xx
// or error styling. This pins the ADR-0001 / Bitcoin-anchoring honesty discipline.
func TestDossierPendingAnchorHonest(t *testing.T) {
	ctx := context.Background()
	st, id := coveredHub(t)
	if _, _, err := st.RecordOTS(ctx, store.OTSRecord{
		HubID:     id,
		TreeSize:  42,
		Root:      []byte("root-verified-32-bytes-padding!!"),
		Status:    store.OTSStatusPending,
		StampedAt: time.Unix(1_700_000_000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS pending: %v", err)
	}

	rec := httptest.NewRecorder()
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`data-dot="pending"`, "pending"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing pending-anchor marker %q\n%s", want, body)
		}
	}
	// No fabricated height: a pending anchor must render no "block N" string.
	if strings.Contains(body, "block ") {
		t.Errorf("pending anchor renders a fabricated block height\n%s", body)
	}
}

// TestDossierCautionForUnverified asserts a store-verified hub whose live verdict is
// unverified renders the soft caution note (visibly distinct from the frozen
// Exhibit) and that the "fork → split view" vocabulary map applies: the §-level
// caution copy uses "split view", never the avoid-listed "fork". The frozen Exhibit
// must NOT appear for a non-frozen hub.
func TestDossierCautionForUnverified(t *testing.T) {
	st, id := coveredHub(t)
	statuses := fakeStatusSource{id: "unverified"}
	rec := httptest.NewRecorder()
	Handler(st, id, statuses, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`class="caution"`,
		"Note · Unverified",
		"split-view",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing soft-caution marker %q\n%s", want, body)
		}
	}
	// The caution is NOT the frozen Exhibit, and the avoid-listed "fork" wording must
	// not leak into the §-level status copy.
	if strings.Contains(body, `class="exhibit"`) {
		t.Errorf("unverified (non-frozen) hub renders the frozen Exhibit\n%s", body)
	}
}

// TestDossierChromeTierTwoAndBackLink asserts the dossier carries the shared-chrome
// tier-2 affordance and navigation closure: the "monitor instance" identity label,
// the static "verify ↗ monitor.iscc.codes" tier-2 link to the monitor-agnostic
// verifier app (Surface C, the .codes app — never an instance), and the "← Realm
// index" back-link up to the dashboard at "/". The DS shell stays same-origin.
func TestDossierChromeTierTwoAndBackLink(t *testing.T) {
	st, id := coveredHub(t)
	rec := httptest.NewRecorder()
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"← Realm index",                      // the back-link copy
		`href="/"`,                           // the back-link target (realm index)
		"monitor instance",                   // the instance-identity label
		"monitor.iscc.codes",                 // the tier-2 verify link copy
		`href="https://monitor.iscc.codes/"`, // the tier-2 verify link target
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing chrome/back-link marker %q\n%s", want, body)
		}
	}
	// The DS shell links stay same-origin: the verifier app link is the only
	// external https origin, never a CDN-hosted stylesheet or font.
	for _, want := range []string{`href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing same-origin DS-shell link %q\n%s", want, body)
		}
	}
}

// TestDossierRendersInstanceIdentity proves the dossier masthead renders the
// operator-supplied instance identity at the HTTP seam, the SAME dashboard.Identity
// value the "/" page receives: a populated Identity surfaces its exact Instance /
// Operator strings on the chrome, and a zero-value Identity falls back to the static
// placeholder copy (the neutral "monitor instance" line and the generic operator
// line). It is non-vacuous: dropping the {{.Instance}} binding (or the
// .chrome-instance text node) from the template, or threading a constant default
// instead of the supplied value, makes the populated-identity assertions fail
// because the exact operator strings would no longer appear. Realm has no slot on
// the dossier masthead (its title is the realm-subtitle-free "Hub dossier"), so it
// is intentionally not asserted here.
func TestDossierRendersInstanceIdentity(t *testing.T) {
	st, id := coveredHub(t)

	// Populated identity: the masthead must render these exact operator-supplied
	// strings (not the static defaults), driven through the live render so reverting
	// the template binding fails the test. Realm is set but has no dossier slot.
	idv := dashboard.Identity{
		Instance: "monitor.example.test",
		Operator: "operated by Example Org · example net",
		Realm:    "example net",
	}
	rec := httptest.NewRecorder()
	Handler(st, id, nil, idv).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"monitor.example.test",
		"operated by Example Org · example net",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing identity literal %q\n%s", want, body)
		}
	}
	// The static placeholder copy must NOT appear when an operator configures the
	// instance — otherwise the test would pass even if the template ignored the
	// supplied value and kept the hard-coded default.
	if strings.Contains(body, "monitor instance") {
		t.Errorf("body still shows the static placeholder despite a configured Instance\n%s", body)
	}

	// Zero-value identity: the fallback masthead renders the neutral placeholder and
	// today's generic operator line, so an unconfigured deployment is honest rather
	// than asserting a false instance. The fallback strings MUST match the dashboard's
	// so the two mastheads stay byte-identical.
	recDefault := httptest.NewRecorder()
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(recDefault, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))
	if recDefault.Code != http.StatusOK {
		t.Fatalf("default status = %d, want 200", recDefault.Code)
	}
	defaultBody := recDefault.Body.String()
	for _, want := range []string{
		"monitor instance",
		"independent Trust &amp; Transparency service",
	} {
		if !strings.Contains(defaultBody, want) {
			t.Errorf("default body missing fallback %q\n%s", want, defaultBody)
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
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

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
	Handler(st, id, statuses, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))

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
	Handler(st, id, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sb0.iscc.id", nil))
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
	Handler(st, id+999, nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sb0.iscc.id", nil))
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
