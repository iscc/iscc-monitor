// Golden HTTP-seam tests for the realm-wide Certificate of Inclusion route
// (GET /inclusion/{iscc_id}). Each drives certificate.Handler over a fixture store
// (the testnet hubs registered through the public store API, with an indexed leaf
// for a known id) and the interim slot Hub-List, asserting the observable response:
// the §1 SUBJECT clause + subject banner for the known golden id resolve through
// the real index.Decode → registry.Resolve → store.SeqsForISCCID chain, and every
// malformed / unresolvable / not-followed / not-in-log id renders the documented
// "cannot certify" 200 state, never a 4xx/5xx. The oracle gate is N/A here: no
// signature, RFC-6962, Merkle, did:web, fsck, or proof path.
//
// Non-vacuity (reviewer-reproducible): the known-id test asserts the RESOLVED hub
// domain (sb1.amlet.id, derived from decoding MAIGHFECJMOPMIAB to hub_id 1 and
// resolving slot 1 through the Hub-List), not a literal the template carries
// unconditionally. TestCertificateResolvedDomainTracksHubList proves the rendered
// domain follows the Hub-List mapping — swapping slot 1 to sb0.iscc.id renders
// sb0.iscc.id — so a stubbed/wrong Resolve would fail the known-id assert.
package certificate

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
)

// goldenID is the realm-0 golden ISCC-IDv1 (internal/index): it decodes to
// {Realm:0, HubID:1}, so it resolves to the hub at slot 1.
const goldenID = "MAIGHFECJMOPMIAB"

// slot2ID is a constructed realm-1 ISCC-IDv1 (the internal/index test vector) that
// decodes to {Realm:1, HubID:2}, so it resolves to slot 2 — absent from the
// two-hub testnet list — exercising the "not in this realm" state.
const slot2ID = "MEIGHFECJMOPMIAC"

// hubID makes a *uint16 for a registry.Hub slot literal.
func hubID(v uint16) *uint16 { return &v }

// testnetHubList is the interim slot Hub-List the binary builds from realm.txt
// order: slot 0 -> sb0.iscc.id, slot 1 -> sb1.amlet.id. The certificate decodes an
// id's hub_id and resolves it through this list.
func testnetHubList() *registry.HubList {
	return &registry.HubList{
		Version: 1,
		Hubs: []registry.Hub{
			{HubID: hubID(0), URL: "https://sb0.iscc.id", Active: true},
			{HubID: hubID(1), URL: "https://sb1.amlet.id", Active: true},
		},
	}
}

// fixtureStore opens a fresh store, registers the two testnet hubs (sb0 then sb1,
// matching the slot order), indexes one leaf for the hub at the given domain under
// the PRODUCTION storage form (ISCC:-prefixed, matching logclient/projection.go),
// and seeds an accepted checkpoint covering that leaf (LastSize = seq+1) so the
// accepted-tree cap certifies it. indexedID is the BARE golden id; fixtureStore
// prefixes it with "ISCC:" before writing the projection, grounding the fixture in
// the wire format rather than in the handler's lookup. It returns the store; the
// caller drives the handler against it. To exercise the cannot-certify states a
// caller uses fixtureStoreUnaccepted instead.
func fixtureStore(t *testing.T, indexDomain, indexedID string, seq uint64) *store.Store {
	t.Helper()
	lastSize := uint64(0)
	if indexedID != "" {
		lastSize = seq + 1 // accept a checkpoint covering the leaf at seq
	}
	return fixtureStoreUnaccepted(t, indexDomain, indexedID, seq, lastSize)
}

// fixtureStoreUnaccepted is fixtureStore with an explicit accepted LastSize, so a
// test can seed a leaf at seq >= lastSize (or lastSize == 0, no accepted checkpoint)
// to exercise the accepted-tree cap's cannot-certify states. The leaf is indexed
// under the ISCC:-prefixed form; the accepted checkpoint is committed through the
// store's real accept path (AdvanceAccepted, the same call checkpoints_test.go
// uses to set LastSize) only when lastSize > 0.
func fixtureStoreUnaccepted(t *testing.T, indexDomain, indexedID string, seq, lastSize uint64) *store.Store {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "certificate.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	id0, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub sb0: %v", err)
	}
	id1, err := st.UpsertHub(ctx, "sb1.amlet.id", "sb1.amlet.id/log", "https://sb1.amlet.id")
	if err != nil {
		t.Fatalf("UpsertHub sb1: %v", err)
	}

	target := id0
	if indexDomain == "sb1.amlet.id" {
		target = id1
	}
	if indexedID != "" {
		if err := st.RecordProjections(ctx, []store.ProjectionRecord{
			// Index under the production form: ISCC:-prefixed, verbatim
			// (logclient/projection.go), not the bare decode input.
			{HubID: target, Seq: seq, IsccID: "ISCC:" + indexedID, NoteSchema: "iscc-note-0.8.0.json"},
		}); err != nil {
			t.Fatalf("RecordProjections: %v", err)
		}
	}
	if lastSize > 0 {
		if err := st.AdvanceAccepted(ctx, store.CheckpointRecord{
			HubID:    target,
			TreeSize: lastSize,
			Root:     []byte("root"),
			Raw:      []byte("raw"),
		}); err != nil {
			t.Fatalf("AdvanceAccepted: %v", err)
		}
	}
	return st
}

// get drives the handler for the path suffix id and returns the recorder.
func get(t *testing.T, h http.Handler, id string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathPrefix+id, nil))
	return rec
}

// TestCertificateKnownID asserts the golden known-id chain over the PRODUCTION path:
// the leaf is indexed under the ISCC:-prefixed form and covered by an accepted
// checkpoint, so the bare-suffix request GET /inclusion/<golden> certifies through
// the real index.Decode → registry.Resolve → SeqsForISCCID(prefixed) → seqs[0] <
// LastSize chain. It returns 200 text/html and the body carries the subject id, the
// RESOLVED hub domain (sb1.amlet.id, from decoding hub_id 1 -> slot 1), the position
// (seqs[0]), the §1 SUBJECT clause marker, the back-link, the tier-2 verify link,
// the two-tier honesty panel, and no external CDN URL outside the same-origin /_ds/
// links.
func TestCertificateKnownID(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815)
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	body := rec.Body.String()

	for _, want := range []string{
		goldenID,             // the subject id
		"sb1.amlet.id",       // the RESOLVED hub domain (the non-vacuous derived value)
		"24815",              // the subject position seqs[0]
		"§1 SUBJECT",         // the §1 clause marker
		"§2 CHECKPOINT",      // the §2 clause marker
		"size 24816",         // the accepted checkpoint tree size (hub.LastSize = seq+1)
		"← Realm index",      // the back-link
		"monitor.iscc.codes", // the tier-2 verify link
		"Tier 1",             // the two-tier honesty panel
		"Tier 2",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n%s", want, body)
		}
	}
	// §2 renders the REAL accepted root the fixture committed (base64-Std), not a
	// fabricated/hardcoded one — rendering a wrong root fails this assert.
	if wantRoot := base64.StdEncoding.EncodeToString([]byte("root")); !strings.Contains(body, wantRoot) {
		t.Errorf("body missing the §2 accepted root %q\n%s", wantRoot, body)
	}
	// The resolved hub at slot 1 is sb1.amlet.id, NOT the slot-0 sb0.iscc.id; if the
	// decode/resolve chain were bypassed the wrong (or no) domain would render.
	if strings.Contains(body, "transparency log of <span class=\"subject-strong subject-mono\">sb0.iscc.id</span>") {
		t.Errorf("subject banner resolved to sb0.iscc.id; decode->resolve picked the wrong slot\n%s", body)
	}
	// The DS shell is linked (same-origin /_ds/), and no external CDN URL appears
	// outside those links. The chrome/verify link to monitor.iscc.codes is the one
	// intentional external https origin, so ban only third-party CDN hosts.
	for _, want := range []string{`href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing DS-shell link %q\n%s", want, body)
		}
	}
	for _, banned := range []string{"jsdelivr", "cdn.", "unpkg", "googleapis"} {
		if strings.Contains(body, banned) {
			t.Errorf("body contains external CDN reference %q\n%s", banned, body)
		}
	}
}

// TestCertificateResolvedDomainTracksHubList is the reviewer-reproducible
// non-vacuity proof: the rendered subject domain follows the Hub-List mapping, not
// a template literal. With slot 1 remapped to sb0.iscc.id (and the leaf indexed
// under sb0), the same golden id renders sb0.iscc.id, NOT sb1.amlet.id — so a
// stubbed/wrong Resolve would fail TestCertificateKnownID's sb1.amlet.id assert.
func TestCertificateResolvedDomainTracksHubList(t *testing.T) {
	st := fixtureStore(t, "sb0.iscc.id", goldenID, 7)
	// Remap slot 1 (the golden id's hub_id) to sb0.iscc.id.
	remapped := &registry.HubList{
		Version: 1,
		Hubs: []registry.Hub{
			{HubID: hubID(0), URL: "https://sb0.iscc.id", Active: true},
			{HubID: hubID(1), URL: "https://sb0.iscc.id", Active: true},
		},
	}
	h := Handler(remapped, st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "sb0.iscc.id") {
		t.Errorf("remapped slot 1 did not resolve to sb0.iscc.id\n%s", body)
	}
	// The subject banner must NOT carry the original sb1.amlet.id — the rendered
	// domain tracks the Hub-List, proving the decode->resolve chain is load-bearing.
	if strings.Contains(body, "sb1.amlet.id") {
		t.Errorf("remapped certificate still shows sb1.amlet.id; resolve is not load-bearing\n%s", body)
	}
}

// TestCertificateNotInLog asserts an id that decodes and resolves to a followed hub
// but has no indexed leaf renders the documented "not found in log" 200 state,
// never a 4xx/5xx.
func TestCertificateNotInLog(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", "", 0) // no leaf indexed
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "not found in log") {
		t.Errorf("body missing the not-found-in-log state\n%s", body)
	}
	// It must NOT claim a certifiable subject banner.
	if strings.Contains(body, "is included in the transparency log") {
		t.Errorf("not-in-log id rendered a certifiable subject banner\n%s", body)
	}
	// A non-certifiable id renders NO §2 CHECKPOINT clause.
	if strings.Contains(body, "§2 CHECKPOINT") {
		t.Errorf("a not-in-log id rendered a §2 CHECKPOINT clause\n%s", body)
	}
}

// TestCertificatePrefixedLookup asserts the canonicalization: with the leaf indexed
// under the production ISCC:-prefixed form, the bare-suffix request /inclusion/<id>
// AND the prefixed request /inclusion/ISCC:<id> both certify the same leaf. Reverting
// the lookup to the bare rawID makes the first request report "not found in log".
func TestCertificatePrefixedLookup(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815)
	h := Handler(testnetHubList(), st, nil)

	for _, id := range []string{goldenID, "ISCC:" + goldenID} {
		rec := get(t, h, id)
		if rec.Code != http.StatusOK {
			t.Fatalf("id %q: status = %d, want 200", id, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "is included in the transparency log") {
			t.Errorf("id %q did not certify the prefixed leaf\n%s", id, body)
		}
		if !strings.Contains(body, "24815") {
			t.Errorf("id %q missing the subject position\n%s", id, body)
		}
	}
}

// TestCertificateUnacceptedLeaf asserts the accepted-tree cap: a leaf indexed at a
// seq >= LastSize (a frozen/failed poll left an unaccepted projection above the
// accepted checkpoint) AND a hub with no accepted checkpoint yet (LastSize == 0)
// both render the cannot-certify state, NOT the affirmative subject banner.
// Reverting the seqs[0] < LastSize cap makes this FAIL.
func TestCertificateUnacceptedLeaf(t *testing.T) {
	t.Run("above accepted tree", func(t *testing.T) {
		// Leaf at seq 24815, accepted checkpoint only at size 24815, so
		// seqs[0] (24815) >= LastSize (24815): indexed but not yet accepted.
		st := fixtureStoreUnaccepted(t, "sb1.amlet.id", goldenID, 24815, 24815)
		h := Handler(testnetHubList(), st, nil)

		rec := get(t, h, goldenID)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "not in accepted tree") {
			t.Errorf("body missing the not-in-accepted-tree state\n%s", body)
		}
		if strings.Contains(body, "is included in the transparency log") {
			t.Errorf("an unaccepted leaf rendered a certifiable subject banner\n%s", body)
		}
		// A non-certifiable id renders NO §2 CHECKPOINT clause.
		if strings.Contains(body, "§2 CHECKPOINT") {
			t.Errorf("an unaccepted leaf rendered a §2 CHECKPOINT clause\n%s", body)
		}
	})

	t.Run("no accepted checkpoint yet", func(t *testing.T) {
		// Leaf indexed, but no accepted checkpoint (LastSize == 0).
		st := fixtureStoreUnaccepted(t, "sb1.amlet.id", goldenID, 24815, 0)
		h := Handler(testnetHubList(), st, nil)

		rec := get(t, h, goldenID)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "no accepted checkpoint yet") {
			t.Errorf("body missing the no-accepted-checkpoint-yet state\n%s", body)
		}
		if strings.Contains(body, "is included in the transparency log") {
			t.Errorf("a hub with no accepted checkpoint rendered a certifiable subject banner\n%s", body)
		}
		// A non-certifiable id renders NO §2 CHECKPOINT clause.
		if strings.Contains(body, "§2 CHECKPOINT") {
			t.Errorf("a hub with no accepted checkpoint rendered a §2 CHECKPOINT clause\n%s", body)
		}
	})
}

// TestCertificateMalformedID asserts a non-decodable id renders the documented
// invalid-id 200 state, never a 4xx/5xx — a decode error is a verdict, not a fault.
func TestCertificateMalformedID(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 1)
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, "NOTANISCCID")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Cannot certify", "not a valid ISCC-ID"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing invalid-id marker %q\n%s", want, body)
		}
	}
}

// TestCertificateUnresolvableSlot asserts an id whose hub_id slot is not in the
// Hub-List renders the documented "not in this realm" 200 state.
func TestCertificateUnresolvableSlot(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 1)
	h := Handler(testnetHubList(), st, nil)

	// slot2ID decodes to hub_id 2, which is not in the two-hub list.
	rec := get(t, h, slot2ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not found in this realm") {
		t.Errorf("body missing the not-in-this-realm state\n%s", rec.Body.String())
	}
}

// TestCertificateResolvedButNotFollowed asserts an id resolving to a domain the
// monitor does not follow renders the "hub not followed" 200 state. The Hub-List
// resolves slot 1 to an unlisted domain, while the store holds only sb0/sb1.
func TestCertificateResolvedButNotFollowed(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 1)
	unfollowed := &registry.HubList{
		Version: 1,
		Hubs: []registry.Hub{
			{HubID: hubID(1), URL: "https://elsewhere.example", Active: true},
		},
	}
	h := Handler(unfollowed, st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "hub not followed by this monitor") {
		t.Errorf("body missing the hub-not-followed state\n%s", rec.Body.String())
	}
}

// TestCertificateEmptyID asserts a bare /inclusion/ (empty id) renders the honest
// "no id supplied" 200 state, never a 5xx.
func TestCertificateEmptyID(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 1)
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "no ISCC-ID supplied") {
		t.Errorf("body missing the empty-id state\n%s", rec.Body.String())
	}
}

// TestCertificateNilHubList asserts a nil Hub-List makes every id resolve to "not
// in this realm" (fail-closed), never a panic or 5xx.
func TestCertificateNilHubList(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 1)
	h := Handler(nil, st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not found in this realm") {
		t.Errorf("nil Hub-List did not fail closed\n%s", rec.Body.String())
	}
}

// TestCertificateNonGET asserts a non-GET method is a 405 (the shared method gate).
func TestCertificateNonGET(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 1)
	h := Handler(testnetHubList(), st, nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, PathPrefix+goldenID, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
