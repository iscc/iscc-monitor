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
	"encoding/binary"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"

	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
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

// hostPortHubList is a Hub-List whose slot 1 (the slot goldenID decodes to) resolves
// to a host:port hub (https://localhost:8443 → domain localhost:8443). It exercises
// the certificate's did:web port-encoding path: the resolved domain carries a port
// colon, so the §4 DID and the bundle's hub.did must render did:web:localhost%3A8443.
func hostPortHubList() *registry.HubList {
	return &registry.HubList{
		Version: 1,
		Hubs: []registry.Hub{
			{HubID: hubID(0), URL: "https://sb0.iscc.id", Active: true},
			{HubID: hubID(1), URL: "https://localhost:8443", Active: true},
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
		`src="/_ds/iscc-logo-black.png"`, // the shared-chrome ISCC logo (byte-equal to web.LogoPath)
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

// TestCertificateQueryFallback asserts the no-JS hero-form fallback: the dashboard's
// claim-lookup <form method="get"> can only emit a query string, so it posts a bare
// /inclusion/?iscc_id=<id>. The handler must use the query id when the path id is
// empty, so GET /inclusion/?iscc_id=<golden> certifies the SAME subject as the path
// form GET /inclusion/<golden> (the query value reuses the decode→resolve→render
// chain). A bare /inclusion/ with no id and no query stays the honest "no id
// supplied" 200. Non-vacuous: removing the query fallback makes the query request
// render "no ISCC-ID supplied" instead of the subject banner.
func TestCertificateQueryFallback(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815)
	h := Handler(testnetHubList(), st, nil)

	// The path form: the certifying reference body to match against.
	pathRec := get(t, h, goldenID)
	if pathRec.Code != http.StatusOK {
		t.Fatalf("path form status = %d, want 200", pathRec.Code)
	}
	if !strings.Contains(pathRec.Body.String(), "is included in the transparency log") {
		t.Fatalf("path form did not certify the golden id\n%s", pathRec.Body.String())
	}

	// The query form the hero emits: bare /inclusion/ with ?iscc_id=<golden>.
	queryRec := httptest.NewRecorder()
	h.ServeHTTP(queryRec, httptest.NewRequest(http.MethodGet, PathPrefix+"?iscc_id="+goldenID, nil))
	if queryRec.Code != http.StatusOK {
		t.Fatalf("query form status = %d, want 200", queryRec.Code)
	}
	queryBody := queryRec.Body.String()
	for _, want := range []string{
		"is included in the transparency log", // the subject banner certifies
		"sb1.amlet.id",                        // the resolved hub domain
		"24815",                               // the subject position
	} {
		if !strings.Contains(queryBody, want) {
			t.Errorf("query form missing %q\n%s", want, queryBody)
		}
	}

	// A bare /inclusion/ with no id and no query stays the honest "no id supplied".
	bareRec := get(t, h, "")
	if bareRec.Code != http.StatusOK {
		t.Fatalf("bare form status = %d, want 200", bareRec.Code)
	}
	if !strings.Contains(bareRec.Body.String(), "no ISCC-ID supplied") {
		t.Errorf("bare /inclusion/ did not render the empty-id state\n%s", bareRec.Body.String())
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

// treeNodeHash recomputes the tree-node hash at (treeLevel, treeIndex) by folding
// the leaf hashes that node covers through a compact range — the same recompute the
// proof builder does, so the tiles ingested below are byte-consistent with the tree
// (ported from logclient/proofbuilder_test.go's nodeHash and follower/fsck_test.go's
// equivNodeHash). The node covers leaves [treeIndex<<treeLevel, (treeIndex+1)<<treeLevel)
// within size.
func treeNodeHash(t *testing.T, tree *testonly.Tree, treeLevel, treeIndex, size uint64) []byte {
	t.Helper()
	first := treeIndex << treeLevel
	last := (treeIndex + 1) << treeLevel
	if last > size {
		last = size
	}
	rf := compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}
	r := rf.NewEmptyRange(0)
	for i := first; i < last; i++ {
		if err := r.Append(tree.LeafHash(i), nil); err != nil {
			t.Fatalf("Append leaf %d: %v", i, err)
		}
	}
	h, err := r.GetRootHash(nil)
	if err != nil {
		t.Fatalf("GetRootHash for node (%d, %d): %v", treeLevel, treeIndex, err)
	}
	return h
}

// encodeBundle frames raw records into a tlog-tiles entry bundle (each record
// prefixed with its big-endian uint16 length, then concatenated) — the C2SP
// encoding api.EntryBundle.UnmarshalText decodes inside RecordBytesFromBundle. This
// is the test's independent encode path (copied from logclient/fsck_test.go), so the
// leaf hash the §3 verification derives matches tree.LeafHash(seq).
func encodeBundle(records [][]byte) []byte {
	var out []byte
	for _, rec := range records {
		var prefix [2]byte
		binary.BigEndian.PutUint16(prefix[:], uint16(len(rec)))
		out = append(out, prefix[:]...)
		out = append(out, rec...)
	}
	return out
}

// fixtureStoreTiled is fixtureStore with a REAL mirrored tile + entry-bundle backing,
// so the §3 inclusion-proof clause has genuine tiles to rebuild the proof from AND a
// genuine entry bundle to derive the subject leaf hash from. It builds an
// internally-consistent RFC-6962 tree of `leaves` records (merkle's testonly.Tree
// as the single source of truth), ingests every hash tile a complete mirror of that
// tree needs (the level-0 leaf-hash rows plus any upper levels, each node recomputed
// from the tree so the served tiles are byte-accurate), ingests every entry bundle
// (the record preimages framed via encodeBundle, so RecordBytesFromBundle returns the
// exact leaf preimage and rfc6962.HashLeaf(record) == tree.LeafHash(seq)), accepts the
// checkpoint at `acceptedRoot` (AdvanceAccepted sets LastSize), optionally freezes the
// hub, and indexes the golden leaf at `seq` under the production ISCC:-prefixed form.
// The returned tree is the independent prover the §3 test cross-checks against.
//
// acceptedRoot lets a caller deliberately make the mirror and the accepted root
// belong to DIFFERENT trees (the frozen-after-fork case): pass nil for the clean,
// consistent fixture (the accepted root defaults to the mirrored tree's own
// tree.Hash()), or a divergent root (a second tree's hash) to model a hub whose
// mirrored tiles do not rebuild the accepted root. freeze runs Freeze so ListHubs
// reports Frozen == true. Pick a `leaves`/`seq` that yields a multi-hash proof (e.g.
// a 5-leaf tree, leaf 0) so the proof is substantive, never empty.
//
// checkpointRaw is the raw checkpoint bytes the accepted-checkpoint record carries;
// KeyIDFromCheckpoint (the §4 signing-key derivation) reads its signature line. The
// §3 callers pass the cheap []byte("raw") (which KeyIDFromCheckpoint rejects — the
// honest §4 decline they don't exercise); the §4 callers pass a real signed-note
// checkpoint so the key id recovers.
func fixtureStoreTiled(t *testing.T, indexDomain, indexedID string, seq uint64, leaves int, acceptedRoot []byte, freeze bool, checkpointRaw []byte) (*store.Store, *testonly.Tree) {
	t.Helper()
	ctx := context.Background()

	tree := testonly.New(rfc6962.DefaultHasher)
	for i := range leaves {
		tree.AppendData([]byte(fmt.Sprintf("leaf-%d", i)))
	}
	size := tree.Size()
	if acceptedRoot == nil {
		acceptedRoot = tree.Hash() // clean fixture: mirror and accepted root agree
	}

	st, err := store.Open(filepath.Join(t.TempDir(), "certificate-tiled.db"))
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
	switch indexDomain {
	case "sb0.iscc.id":
		target = id0
	case "sb1.amlet.id":
		target = id1
	default:
		// A non-testnet indexDomain (e.g. the host:port DID fixture localhost:8443)
		// registers its own hub so the host:port path can be exercised.
		target, err = st.UpsertHub(ctx, indexDomain, indexDomain+"/log", "https://"+indexDomain)
		if err != nil {
			t.Fatalf("UpsertHub %q: %v", indexDomain, err)
		}
	}

	// Ingest every hash tile the mirror needs, byte-accurate against the tree. Each
	// tile's bottom row spans tree nodes at tree-level (tileLevel*TileHeight); a node
	// off the right edge (its leaves all >= size) is omitted, so a partial tile holds
	// exactly its live nodes (store.md widthForP: the partial qualifier is its width).
	for _, c := range tiles.TileCoords(size) {
		treeLevel := c.Level * uint64(tiles.TileHeight)
		first := c.Index * tiles.TileWidth
		var nodes [][]byte
		for n := uint64(0); n < tiles.TileWidth; n++ {
			treeIndex := first + n
			if (treeIndex << treeLevel) >= size {
				break
			}
			nodes = append(nodes, treeNodeHash(t, tree, treeLevel, treeIndex, size))
		}
		raw, err := api.HashTile{Nodes: nodes}.MarshalText()
		if err != nil {
			t.Fatalf("HashTile.MarshalText (level %d index %d): %v", c.Level, c.Index, err)
		}
		if err := st.RecordTile(ctx, target, c.Level, c.Index, c.Partial, raw, time.Unix(0, 0)); err != nil {
			t.Fatalf("RecordTile (level %d index %d): %v", c.Level, c.Index, err)
		}
	}

	// Ingest every entry bundle the mirror needs, framing the SAME leaf preimages the
	// tree was built from (leaf-i). RecordBytesFromBundle then returns the exact
	// preimage so rfc6962.HashLeaf(record) == tree.LeafHash(seq); that equality is what
	// lets the §3 verification pass on the clean fixture (and FAIL on a contradictory
	// accepted root). A <256-leaf tree is one partial bundle at index 0.
	for _, c := range tiles.BundleCoords(size) {
		first := c.Index * tiles.TileWidth
		var records [][]byte
		for n := uint64(0); n < tiles.TileWidth; n++ {
			leaf := first + n
			if leaf >= size {
				break
			}
			records = append(records, []byte(fmt.Sprintf("leaf-%d", leaf)))
		}
		if err := st.RecordEntryBundle(ctx, target, c.Index, c.Partial, encodeBundle(records), time.Unix(0, 0)); err != nil {
			t.Fatalf("RecordEntryBundle (index %d): %v", c.Index, err)
		}
	}

	if err := st.RecordProjections(ctx, []store.ProjectionRecord{
		{HubID: target, Seq: seq, IsccID: "ISCC:" + indexedID, NoteSchema: "iscc-note-0.8.0.json"},
	}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}
	// Accept the checkpoint at acceptedRoot (tree.Hash() for the clean fixture, a
	// divergent root for the frozen-after-fork case), so the accepted-tree cap
	// certifies the leaf and §2 renders that root. AdvanceAccepted sets LastSize.
	if err := st.AdvanceAccepted(ctx, store.CheckpointRecord{
		HubID:    target,
		TreeSize: size,
		Root:     acceptedRoot,
		Raw:      checkpointRaw,
	}); err != nil {
		t.Fatalf("AdvanceAccepted: %v", err)
	}
	if freeze {
		if err := st.Freeze(ctx, target); err != nil {
			t.Fatalf("Freeze: %v", err)
		}
	}
	return st, tree
}

// TestCertificateInclusionProof is the §3 oracle-gated test: with a real mirrored
// tile backing, the certificate recomputes the RFC-6962 inclusion proof of the
// subject leaf from the mirror and renders the leaf→siblings→root chain. The proof
// is mutation-proven non-vacuous against testonly.Tree.InclusionProof (the
// independent prover): the rendered hash chips must equal base64-Std of each hash in
// tree.InclusionProof(seq, size). A 5-leaf tree with the leaf at seq 0 yields a
// multi-hash proof, so the byte-match is substantive, not an empty-vs-empty pass.
func TestCertificateInclusionProof(t *testing.T) {
	const seq = 0
	const leaves = 5
	// Clean, non-frozen hub: the mirror and the accepted root are the SAME tree, so
	// §3 renders the proof that rebuilds the accepted root.
	st, tree := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, []byte("raw"))
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	// html/template HTML-escapes the rendered base64 chips ('+' -> "&#43;"), so
	// assert on the unescaped body — the semantic content is the base64-Std the
	// view-model carries, byte-identical to verify-for-me's writeEvidence.
	body := html.UnescapeString(rec.Body.String())

	for _, want := range []string{
		"§3 INCLUSION PROOF",
		fmt.Sprintf("leaf · seq %d", seq),
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing §3 marker %q\n%s", want, body)
		}
	}

	// The independent prover: the tree's own inclusion proof. Each rendered hash chip
	// must be the base64-Std of one of these hashes — a wrong/corrupt rendered hash
	// fails this assert, proving §3 is the real proof, not a literal.
	want, err := tree.InclusionProof(seq, tree.Size())
	if err != nil {
		t.Fatalf("tree.InclusionProof: %v", err)
	}
	if len(want) < 2 {
		t.Fatalf("proof has %d hashes; want a multi-hash (substantive) proof", len(want))
	}
	for i, hsh := range want {
		enc := base64.StdEncoding.EncodeToString(hsh)
		if !strings.Contains(body, enc) {
			t.Errorf("body missing §3 inclusion-proof hash %d %q\n%s", i, enc, body)
		}
	}
	// §3's root chip is the accepted root (the chain rebuilds it); it must be the
	// tree's REAL root, base64-Std, the same value §2 renders.
	if root := base64.StdEncoding.EncodeToString(tree.Hash()); !strings.Contains(body, root) {
		t.Errorf("body missing §3 accepted-root chip %q\n%s", root, body)
	}
	// The §3 note states the proof length (the actual count, not a literal "Five").
	if want := fmt.Sprintf("%d sibling hashes", len(want)); !strings.Contains(body, want) {
		t.Errorf("body missing §3 proof-length note %q\n%s", want, body)
	}
}

// TestCertificateRendersWasmVerifier is the tier-2 in-browser-verifier wiring test:
// a CERTIFIABLE id (whose §3 inclusion proof re-verified, so HasBundle) renders the
// progressive-enhancement loader — the same-origin /_ds/wasm_exec.js + /_ds/verify.wasm
// scripts, the tier-2 result region, and a <script type="application/json"> data
// island carrying the base64-Std record / accepted root / inclusion-proof hashes plus
// the integer index/size — while an UNCERTIFIABLE / !HasBundle id (a tile gap where §3
// declined) renders NONE of them (no fabricated verifier on an id the monitor cannot
// re-verify). The data island feeds the WASM isccVerifyInclusion so the BROWSER re-runs
// the same proof the server's §3 already re-verified (the two-tier honesty: the tier-2
// ✓ is a genuine re-VERIFICATION, gated on the SAME HasClause3 the page ✓ is).
//
// Mutation (non-vacuity): removing the data.RecordB64 population (the §3 success path)
// makes the data island carry "record":"" and fails the record-bytes assert; removing
// the {{if .HasBundle}} script block makes the loader/region asserts fail. The negative
// case fails the moment the scripts render unconditionally (an uncertifiable id would
// then carry the loader). The base64 record/root/proof appear verbatim (no '+'->&#43;
// entity escaping) because html/template JSON-context-escapes the <script> data island,
// the load-bearing reason the proof data is passed as a JSON island, not interpolated.
func TestCertificateRendersWasmVerifier(t *testing.T) {
	const seq = 0
	const leaves = 5

	t.Run("certifiable id wires the tier-2 verifier", func(t *testing.T) {
		// Clean tiled fixture: §3 re-verifies, so HasBundle is set and the tier-2
		// loader + data island render.
		st, tree := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, []byte("raw"))
		h := Handler(testnetHubList(), st, nil)

		rec := get(t, h, goldenID)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		// The JSON data island is in a <script> context, where html/template does NOT
		// entity-escape base64 '+'/'/', so the proof data appears verbatim in the RAW body.
		body := rec.Body.String()

		// The progressive-enhancement loader: the same-origin WASM runtime + verifier
		// module scripts and the tier-2 result region.
		for _, want := range []string{
			`<script src="/_ds/wasm_exec.js">`, // the Go WASM runtime loader
			"/_ds/verify.wasm",                 // the verifier module the inline script fetches
			`id="tier2-result"`,                // the result region the script writes the verdict into
			`type="application/json"`,          // the JSON data island (NOT interpolated JS)
			"isccVerifyInclusion",              // the WASM global the loader calls
		} {
			if !strings.Contains(body, want) {
				t.Errorf("certifiable body missing tier-2 verifier marker %q\n%s", want, body)
			}
		}

		// The data island must carry the REAL embedded proof data the WASM verifier
		// re-checks: the base64-Std record (leaf-0), the accepted root, every
		// inclusion-proof hash, and the integer index/size — verbatim, JSON-escaped.
		recordB64 := base64.StdEncoding.EncodeToString([]byte("leaf-0"))
		rootB64 := base64.StdEncoding.EncodeToString(tree.Hash())
		for _, want := range []string{
			`"record":"` + recordB64 + `"`, // the subject leaf's record bytes
			`"root":"` + rootB64 + `"`,     // the accepted root the proof rebuilds
		} {
			if !strings.Contains(body, want) {
				t.Errorf("data island missing %q\n%s", want, body)
			}
		}
		// Every inclusion-proof hash from the independent prover must appear verbatim in
		// the data island's proof array (base64-Std, un-entity-escaped in script context).
		want, err := tree.InclusionProof(seq, tree.Size())
		if err != nil {
			t.Fatalf("tree.InclusionProof: %v", err)
		}
		if len(want) < 2 {
			t.Fatalf("proof has %d hashes; want a multi-hash (substantive) proof", len(want))
		}
		for i, hsh := range want {
			enc := base64.StdEncoding.EncodeToString(hsh)
			if !strings.Contains(body, `"`+enc+`"`) {
				t.Errorf("data island proof array missing hash %d %q\n%s", i, enc, body)
			}
		}
		// The index/size are the subject position and accepted tree size, as bare JSON
		// numbers the WASM integer guard accepts.
		for _, want := range []string{
			fmt.Sprintf(`"index":%d`, seq),
			fmt.Sprintf(`"size":%d`, leaves),
		} {
			// html/template renders a JS-context number with surrounding whitespace
			// (e.g. `"index": 0 `), so match the key + the value tolerantly.
			key := strings.SplitN(want, ":", 2)[0]
			val := strings.SplitN(want, ":", 2)[1]
			idx := strings.Index(body, key+":")
			if idx < 0 || !strings.Contains(body[idx:idx+40], val) {
				t.Errorf("data island missing %q near %q\n%s", want, key, body)
			}
		}

		// No-JS baseline (target.md M-UI "complete with JavaScript disabled"): every
		// §1–§6 clause marker AND the honesty/actions region must render in the document
		// BODY (the <main> sheet), BEFORE the first executable <script src=...>/<script>
		// loader — never inside or gated by a <script>. The first loader <script> appears
		// after </main>, so a clause marker preceding it proves the clause is
		// server-rendered, not script-gated. (The JSON data island is a
		// type="application/json" <script>, which is data, not executable; the executable
		// loader is the <script src="/_ds/wasm_exec.js">.)
		loaderIdx := strings.Index(body, `<script src="/_ds/wasm_exec.js">`)
		if loaderIdx < 0 {
			t.Fatalf("no tier-2 loader script in certifiable body\n%s", body)
		}
		for _, marker := range []string{
			"§1 SUBJECT", "§2 CHECKPOINT", "§3 INCLUSION PROOF",
			"Tier 1", "Tier 2", // the two-tier honesty panel
			"Download proof bundle", // the actions region
		} {
			mIdx := strings.Index(body, marker)
			if mIdx < 0 {
				t.Errorf("no-JS baseline: certifiable body missing %q\n%s", marker, body)
				continue
			}
			if mIdx > loaderIdx {
				t.Errorf("no-JS baseline: %q renders AFTER the loader <script> (script-gated)\n%s", marker, body)
			}
		}
	})

	t.Run("uncertifiable id wires no verifier", func(t *testing.T) {
		// Tile-gap fixture: certifiable §1/§2 but no mirrored tiles, so §3 declines and
		// HasBundle stays false — the page must NOT render the tier-2 loader.
		st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815)
		h := Handler(testnetHubList(), st, nil)

		rec := get(t, h, goldenID)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		// Sanity: this id IS certifiable (§1/§2 render) but §3 declined (HasBundle false).
		if !strings.Contains(body, "§2 CHECKPOINT") {
			t.Fatalf("the negative fixture is not even certifiable (no §2)\n%s", body)
		}
		if strings.Contains(body, "§3 INCLUSION PROOF") {
			t.Fatalf("the negative fixture unexpectedly rendered §3 (HasBundle would be set)\n%s", body)
		}
		// No fabricated verifier on an id the monitor cannot re-verify: none of the
		// tier-2 markers may appear.
		for _, banned := range []string{
			"/_ds/wasm_exec.js",
			"/_ds/verify.wasm",
			`id="tier2-result"`,
			"isccVerifyInclusion",
		} {
			if strings.Contains(body, banned) {
				t.Errorf("uncertifiable body fabricated a tier-2 verifier marker %q\n%s", banned, body)
			}
		}
	})
}

// TestCertificateInclusionProofTileGap asserts the honest-gap path: a certifiable id
// whose tiles are NOT mirrored (the synthetic §1/§2 fixture, no RecordTile) renders
// §1 and §2 but NO §3 clause — the os.ErrNotExist tile miss leaves §3 unrendered
// rather than 500ing or fabricating a proof.
func TestCertificateInclusionProofTileGap(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815) // no tiles mirrored
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// §1 + §2 still render (the certifiable subject + accepted checkpoint).
	if !strings.Contains(body, "§2 CHECKPOINT") {
		t.Errorf("tile-less fixture lost its §2 CHECKPOINT clause\n%s", body)
	}
	// §3 must be ABSENT — the tile gap is honest, not a fabricated proof.
	if strings.Contains(body, "§3 INCLUSION PROOF") {
		t.Errorf("a tile-less fixture rendered a §3 INCLUSION PROOF clause\n%s", body)
	}
}

// TestCertificateInclusionProofContradictory asserts the fail-closed rebuild gate: a
// NON-frozen hub whose mirrored tiles disagree with its accepted root (the fork-poll
// window — the follower ingested the contradictory tree's tiles but has not committed
// the freeze flag, so ListHubs still reports Frozen == false) renders §1 SUBJECT + §2
// CHECKPOINT but NO §3 INCLUSION PROOF. The proof BUILDS from the contradictory tiles,
// but it does not rebuild the accepted root, so proof.VerifyInclusion rejects it and
// the certificate declines §3 rather than render a sibling chain under an accepted-root
// ✓ the chain does not rebuild (a self-contradictory certificate — ADR-0006 / the
// Proof-bundle contract). The fixture mirrors tree A's tiles + bundles but accepts tree
// B's root (a different 5-leaf tree of the same size) and is NOT frozen (freeze=false),
// proving the gate is the re-verification, not the status flag.
//
// Mutation (non-vacuity, review reproduces it): replacing the §3
// `proof.VerifyInclusion(...) == nil` guard in buildData with `true` (so §3 renders
// whenever the proof builds) makes this test FAIL — the non-frozen contradictory hub
// would then render §3 INCLUSION PROOF (the proof builds from the present tiles but
// does not rebuild tree B's accepted root). Restoring the guard passes.
func TestCertificateInclusionProofContradictory(t *testing.T) {
	const seq = 0
	const leaves = 5

	// Tree B is a DIFFERENT 5-leaf tree (distinct leaf bytes), so its root differs
	// from the mirrored tree A's root — the accepted root the §3 proof would have to
	// rebuild does not match the mirrored (tree-A) tiles (the contradictory-tile case).
	treeB := testonly.New(rfc6962.DefaultHasher)
	for i := range leaves {
		treeB.AppendData([]byte(fmt.Sprintf("forked-leaf-%d", i)))
	}
	if treeB.Size() != uint64(leaves) {
		t.Fatalf("treeB.Size() = %d, want %d", treeB.Size(), leaves)
	}

	// freeze=false: the hub is NOT frozen, so a status-flag gate would render §3. Only
	// the fail-closed re-verification against tree B's accepted root withholds it.
	st, treeA := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, treeB.Hash(), false, []byte("raw"))
	// Sanity: the two trees genuinely disagree, so this is a real contradictory-tile
	// fixture (not an accidental same-root coincidence).
	if string(treeA.Hash()) == string(treeB.Hash()) {
		t.Fatalf("treeA and treeB share a root; the fixture is not contradictory")
	}
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	body := rec.Body.String()

	// §1 + §2 still render — they read the irreplaceable accepted-checkpoint record,
	// which a fork cannot corrupt, so the page is NOT blank.
	for _, want := range []string{"§1 SUBJECT", "§2 CHECKPOINT"} {
		if !strings.Contains(body, want) {
			t.Errorf("contradictory certificate missing %q (the page must still render §1+§2)\n%s", want, body)
		}
	}
	// §3 must be ABSENT — the proof built from tree A's tiles does not rebuild tree B's
	// accepted root, so the re-verification declines the clause even though the hub is
	// not frozen.
	if strings.Contains(body, "§3 INCLUSION PROOF") {
		t.Errorf("a non-frozen contradictory-tile hub rendered a §3 INCLUSION PROOF clause\n%s", body)
	}
}

// liveCheckpointRaw loads a captured live signed checkpoint from the module-root
// testdata/live/ directory (two levels up from this package, matching logclient's
// readCheckpoint). Its signature line carries a real BE-uint32 keyhash, so
// KeyIDFromCheckpoint recovers a key id the §4 clause can look up.
func liveCheckpointRaw(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "live", name))
	if err != nil {
		t.Fatalf("read checkpoint fixture %q: %v", name, err)
	}
	return data
}

// hubIDForDomain reads a followed hub's hub_id back from the store by domain — the
// §4 tests need it to seed RecordHubKey for the same hub the fixture indexed.
func hubIDForDomain(t *testing.T, st *store.Store, domain string) int64 {
	t.Helper()
	summaries, err := st.ListHubs(context.Background())
	if err != nil {
		t.Fatalf("ListHubs: %v", err)
	}
	for _, s := range summaries {
		if s.Domain == domain {
			return s.HubID
		}
	}
	t.Fatalf("no followed hub for domain %q", domain)
	return 0
}

// sb0CheckpointKeyID is the BE-uint32 signed-note keyhash recovered from the live
// sb0.iscc.id checkpoint's signature line (pinned in
// logclient/checkpointkey_test.go). KeyIDFromCheckpoint reads only the keyhash, not
// the signature, so this live sb0 note seeds an sb1-indexed fixture fine.
const sb0CheckpointKeyID = uint32(0x40b74463)

// TestCertificateSigningKey is the §4 happy path: with the accepted checkpoint
// carrying a REAL signed-note checkpoint (so KeyIDFromCheckpoint recovers its key id)
// and that key seeded into the hub_keys cache (RecordHubKey), the certificate renders
// the §4 SIGNING KEY clause — the hub's did:web identifier, the hex key id, and the
// cached multibase — while §1/§2/§3 still render (regression). The key id is the live
// sb0 keyhash (0x40b74463), derived from the checkpoint, not synthesized.
func TestCertificateSigningKey(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, _ := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)

	// Seed the cached did:web key the §4 clause reads back, keyed on the SAME key id
	// the live checkpoint's signature line carries (so the derive→lookup chain hits).
	const multibase = "z6MktestKeyMultibaseValue000000000000000000000"
	if err := st.RecordHubKey(context.Background(), store.HubKey{
		HubID:      hubIDForDomain(t, st, "sb1.amlet.id"),
		KeyID:      sb0CheckpointKeyID,
		PubkeyRaw:  make([]byte, 32),
		PubkeyZ:    multibase,
		ResolvedAt: time.Unix(1700000000, 0),
	}); err != nil {
		t.Fatalf("RecordHubKey: %v", err)
	}

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	// §1/§2/§3 must still render (regression) alongside the new §4.
	for _, want := range []string{
		"§1 SUBJECT",
		"§2 CHECKPOINT",
		"§3 INCLUSION PROOF",
		"§4 SIGNING KEY",
		"did:web:sb1.amlet.id", // the resolved hub's did:web identifier
		"40b74463",             // the hex key id derived from the accepted checkpoint
		multibase,              // the seeded cached multibase
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing §4 marker %q\n%s", want, body)
		}
	}
}

// TestCertificateSigningKeyUncached is the §4 honest cache-miss decline: the SAME
// real-note tiled fixture but with NO RecordHubKey seeded, so the key the accepted
// checkpoint was signed with is not in hub_keys. The body renders §1/§2/§3 but NOT
// §4 SIGNING KEY (status 200, never a 500, never a fabricated key).
//
// Mutation (non-vacuity, review reproduces it): forcing HasClause4 = true
// unconditionally in buildData (or rendering §4 on a cache miss) makes this test FAIL
// — the uncached hub would then render §4 SIGNING KEY. Restoring the cache-miss gate
// passes.
func TestCertificateSigningKeyUncached(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	// No RecordHubKey: the key the checkpoint was signed with is not cached.
	st, _ := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	// §1/§2/§3 still render — the cache miss only withholds §4.
	for _, want := range []string{"§1 SUBJECT", "§2 CHECKPOINT", "§3 INCLUSION PROOF"} {
		if !strings.Contains(body, want) {
			t.Errorf("uncached-key certificate missing %q (the page must still render §1-§3)\n%s", want, body)
		}
	}
	// §4 must be ABSENT — an uncached key is an honest decline, not a fabricated key.
	if strings.Contains(body, "§4 SIGNING KEY") {
		t.Errorf("an uncached-key hub rendered a §4 SIGNING KEY clause\n%s", body)
	}
}

// TestCertificateSigningKeyDIDPortEncoded is the §4 did:web port-encoding test: a hub
// resolved to a host:port domain (localhost:8443) must render its §4 SIGNING KEY DID
// as did:web:localhost%3A8443 — the port colon percent-encoded — so the DID denotes
// the same host the key resolved from (didweb.DocumentURL reads a bare colon as a
// path-segment boundary). The malformed did:web:localhost:8443 (host localhost, path
// 8443) must NOT appear.
//
// Mutation (non-vacuity): reverting didWeb at the §4 site to "did:web:" + data.Domain
// makes this test FAIL — the rendered DID would be did:web:localhost:8443.
func TestCertificateSigningKeyDIDPortEncoded(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, _ := fixtureStoreTiled(t, "localhost:8443", goldenID, seq, leaves, nil, false, raw)

	// Seed the cached did:web key keyed on the SAME key id the live checkpoint carries,
	// so §4 renders its DID for the host:port hub.
	if err := st.RecordHubKey(context.Background(), store.HubKey{
		HubID:      hubIDForDomain(t, st, "localhost:8443"),
		KeyID:      sb0CheckpointKeyID,
		PubkeyRaw:  make([]byte, 32),
		PubkeyZ:    "z6MktestKeyMultibaseValue000000000000000000000",
		ResolvedAt: time.Unix(1700000000, 0),
	}); err != nil {
		t.Fatalf("RecordHubKey: %v", err)
	}

	h := Handler(hostPortHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	if !strings.Contains(body, "§4 SIGNING KEY") {
		t.Fatalf("body missing §4 SIGNING KEY clause for the host:port hub\n%s", body)
	}
	if want := "did:web:localhost%3A8443"; !strings.Contains(body, want) {
		t.Errorf("§4 DID missing the port-encoded form %q\n%s", want, body)
	}
	// The unencoded form denotes a different host (localhost, path 8443) — it must not render.
	if bad := "did:web:localhost:8443"; strings.Contains(body, bad) {
		t.Errorf("§4 DID rendered the unencoded host:port form %q (the port colon must be %%3A-encoded)\n%s", bad, body)
	}
}

// TestCertificateSigningKeyDIDCleanDomain is the §4 no-port regression: a clean
// domain (sb1.amlet.id, no colon) must still render did:web:sb1.amlet.id exactly,
// with no spurious encoding — didWeb replaces only a port colon, leaving a no-port
// domain byte-identical to the prior "did:web:" + Domain behavior.
func TestCertificateSigningKeyDIDCleanDomain(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, _ := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)
	if err := st.RecordHubKey(context.Background(), store.HubKey{
		HubID:      hubIDForDomain(t, st, "sb1.amlet.id"),
		KeyID:      sb0CheckpointKeyID,
		PubkeyRaw:  make([]byte, 32),
		PubkeyZ:    "z6MktestKeyMultibaseValue000000000000000000000",
		ResolvedAt: time.Unix(1700000000, 0),
	}); err != nil {
		t.Fatalf("RecordHubKey: %v", err)
	}

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())
	if want := "did:web:sb1.amlet.id"; !strings.Contains(body, want) {
		t.Errorf("§4 DID for a clean domain = missing %q (no-port domain must round-trip unchanged)\n%s", want, body)
	}
}

// The FULL wire URIs the iscc_index projection stores verbatim (CLAUDE.md's
// iscc-note-0.8.0 is prose shorthand). §6's recordKind maps these to the Declaration
// / Deletion labels; the fixture seeds the wire form so the kind labels resolve as
// production stores them.
const (
	wireSchemaDeclaration = "http://purl.org/iscc/schema/iscc-note-0.8.0.json"
	wireSchemaDeletion    = "http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"
)

// fixtureStoreHistory seeds a hub whose subject id has the one-to-many §6 record
// history: a declaration at declSeq and a LATER deletion at delSeq, both under the
// SAME ISCC:-prefixed id and both below the accepted checkpoint (LastSize = delSeq+1),
// so both rows fall within the accepted tree. seq is the PRIMARY KEY, so the two
// records use distinct seqs (a deletion is a new record at a higher seq). The schemas
// are the FULL wire URIs so §6's recordKind labels them Declaration / Deletion.
func fixtureStoreHistory(t *testing.T, indexDomain, indexedID string, declSeq, delSeq uint64) *store.Store {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "certificate-history.db"))
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

	prefixed := "ISCC:" + indexedID
	if err := st.RecordProjections(ctx, []store.ProjectionRecord{
		{HubID: target, Seq: declSeq, IsccID: prefixed, NoteSchema: wireSchemaDeclaration},
		{HubID: target, Seq: delSeq, IsccID: prefixed, NoteSchema: wireSchemaDeletion},
	}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}
	if err := st.AdvanceAccepted(ctx, store.CheckpointRecord{
		HubID:    target,
		TreeSize: delSeq + 1, // accept a checkpoint covering both records
		Root:     []byte("root"),
		Raw:      []byte("raw"),
	}); err != nil {
		t.Fatalf("AdvanceAccepted: %v", err)
	}
	return st
}

// TestCertificateRecordHistory is the §6 RECORD HISTORY test: a subject id with a
// declaration AND a later deletion (the one-to-many iscc_id → seq case, ADR-0008)
// renders BOTH rows — the declaration row labelled "Declaration · seq <declSeq>" and
// the deletion row labelled "Deletion · seq <delSeq>" — plus the deletion note that a
// deletion is a new record and the declaration is preserved. The subject position is
// the earliest seq (declSeq), so §1 still certifies the declaration.
//
// Non-vacuity: setting data.HasClause6 = false in buildData (or dropping the deletion
// row from RecordHistory) makes this FAIL — the body would then carry no §6 marker
// (or no Deletion row / no note). The declaration and deletion seqs are distinct
// asserted values, so a neutered §6 cannot pass.
func TestCertificateRecordHistory(t *testing.T) {
	const declSeq = uint64(24815)
	const delSeq = uint64(31002)
	st := fixtureStoreHistory(t, "sb1.amlet.id", goldenID, declSeq, delSeq)
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"§6 RECORD HISTORY",
		fmt.Sprintf("Declaration · seq %d", declSeq), // the declaration row
		fmt.Sprintf("Deletion · seq %d", delSeq),     // the later deletion row
		"A deletion is a new record",                 // the deletion note
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing §6 marker %q\n%s", want, body)
		}
	}
	// §1 still certifies the declaration at the earliest seq.
	if !strings.Contains(body, fmt.Sprintf("position <span class=\"subject-strong\">%d</span>", declSeq)) {
		t.Errorf("§1 subject position is not the earliest seq %d\n%s", declSeq, body)
	}
}

// TestCertificateRecordHistoryDeclarationOnly asserts the common single-record case:
// a subject id with only a declaration renders §6 with the one Declaration row and NO
// deletion note (HasDeletion is false). It guards against the note rendering
// unconditionally.
func TestCertificateRecordHistoryDeclarationOnly(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815)
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "§6 RECORD HISTORY") {
		t.Errorf("a certifiable id rendered no §6 RECORD HISTORY clause\n%s", body)
	}
	// fixtureStore seeds the bare "iscc-note-0.8.0.json" schema (not the full wire URI),
	// so recordKind labels it the catch-all unknown — but the seq still lists and the
	// clause renders. The deletion note must be ABSENT (a single record is no deletion).
	if !strings.Contains(body, "seq 24815") {
		t.Errorf("§6 did not list the subject seq\n%s", body)
	}
	if strings.Contains(body, "A deletion is a new record") {
		t.Errorf("a declaration-only id rendered the deletion note\n%s", body)
	}
}

// fixtureStoreCovered seeds a certifiable hub whose coverage window carries BOTH a
// monitored-since size and a non-zero monitored-since TIME, so the COMPARISON ANCHOR
// panel renders the full "since size N · <RFC-3339>" window. It sets the coverage start
// (SetCoverage, the set-once monitored_since_{size,time}) with coverSize/coverSince
// BEFORE AdvanceAccepted, so AdvanceAccepted's own set-once coverage UPDATE no-ops
// (monitored_since_size is already non-NULL) and the explicit time survives — the only
// way to get a coverage time in a fixture, since AdvanceAccepted writes a NULL time for
// the zero-ObservedAt CheckpointRecord the other fixtures pass. The leaf is indexed at
// seq under the production ISCC:-prefixed form and accepted at LastSize = seq+1.
func fixtureStoreCovered(t *testing.T, indexDomain, indexedID string, seq, coverSize uint64, coverSince time.Time) *store.Store {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "certificate-covered.db"))
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

	if err := st.RecordProjections(ctx, []store.ProjectionRecord{
		{HubID: target, Seq: seq, IsccID: "ISCC:" + indexedID, NoteSchema: "iscc-note-0.8.0.json"},
	}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}
	// Set the coverage start WITH a time before AdvanceAccepted so the explicit time is
	// what ListHubs reads back (AdvanceAccepted's set-once UPDATE then no-ops).
	if err := st.SetCoverage(ctx, target, coverSize, coverSince); err != nil {
		t.Fatalf("SetCoverage: %v", err)
	}
	if err := st.AdvanceAccepted(ctx, store.CheckpointRecord{
		HubID:    target,
		TreeSize: seq + 1,
		Root:     []byte("root"),
		Raw:      []byte("raw"),
	}); err != nil {
		t.Fatalf("AdvanceAccepted: %v", err)
	}
	return st
}

// TestCertificateComparisonAnchor is the COMPARISON ANCHOR test: a certifiable id on a
// hub with a recorded coverage window renders the distinctly-labelled comparison-anchor
// panel — the monitor's independently-observed (size, root) of what this hub showed
// THIS monitor, plus the coverage window (size + since) that bounds it — and that panel
// carries NO "Bitcoin"/"anchoring"/"OpenTimestamps" copy, proving it is a SEPARATE,
// distinctly-labelled element from the §5 Bitcoin anchor (target.md: "anchoring" copy is
// Bitcoin-only).
//
// Mutation (non-vacuity, review reproduces it): forcing HasComparisonAnchor = false in
// buildData (or removing the {{if .HasComparisonAnchor}} template block) makes this test
// FAIL — the body would then carry no COMPARISON ANCHOR marker. Restoring it passes.
func TestCertificateComparisonAnchor(t *testing.T) {
	const seq = uint64(24815)
	coverSince := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	st := fixtureStoreCovered(t, "sb1.amlet.id", goldenID, seq, 24000, coverSince)
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	// The distinctly-labelled comparison-anchor panel + its bounding coverage window.
	for _, want := range []string{
		"COMPARISON ANCHOR",             // the distinct label (NOT "§5"/"BITCOIN ANCHOR")
		"size 24000",                    // the coverage-window size (monitored_since_size)
		coverSince.Format(time.RFC3339), // the coverage-window since-time (RFC-3339)
		"detect a split view",           // the comparison-anchor affordance copy
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing comparison-anchor marker %q\n%s", want, body)
		}
	}
	// The §1/§2 clauses still render alongside it (regression).
	for _, want := range []string{"§1 SUBJECT", "§2 CHECKPOINT"} {
		if !strings.Contains(body, want) {
			t.Errorf("comparison-anchor certificate missing %q\n%s", want, body)
		}
	}

	// Distinctness: the comparison-anchor PANEL must carry no "anchoring"/Bitcoin copy.
	// Slice the COMPARISON ANCHOR clause out of the rendered page (from its marker to the
	// next clause marker) and assert the Bitcoin/OTS lexicon never appears inside it —
	// proving the two anchor panels are separate, distinctly-labelled elements.
	panel := comparisonAnchorPanel(t, body)
	for _, banned := range []string{"Bitcoin", "anchoring", "OpenTimestamps", "BITCOIN ANCHOR", "ots verify"} {
		if strings.Contains(panel, banned) {
			t.Errorf("comparison-anchor panel contains Bitcoin/anchoring copy %q\n%s", banned, panel)
		}
	}
}

// comparisonAnchorPanel slices the COMPARISON ANCHOR clause out of the rendered body —
// from its clause marker up to the next clause marker (or end of document) — so a test
// can assert on the panel's OWN copy in isolation, never catching a §5 Bitcoin string
// from elsewhere on the page.
func comparisonAnchorPanel(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "COMPARISON ANCHOR")
	if start < 0 {
		t.Fatalf("no COMPARISON ANCHOR panel in body\n%s", body)
	}
	rest := body[start+len("COMPARISON ANCHOR"):]
	// The next clause marker bounds this panel; §6 is the only clause that can follow it.
	if end := strings.Index(rest, "§6 RECORD HISTORY"); end >= 0 {
		return rest[:end]
	}
	// No §6 follows (e.g. an empty record history) — bound at the honesty panel instead.
	if end := strings.Index(rest, "honesty"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// TestCertificateComparisonAnchorIndependentOfOTS asserts the two anchor panels are
// DECOUPLED: a certifiable id on a hub with NO OTS row (so §5 BITCOIN ANCHOR is omitted)
// STILL renders the COMPARISON ANCHOR panel. The comparison anchor does not depend on
// the Bitcoin anchor — exactly the "separate, distinctly-labelled elements" the Verify
// criterion requires (a hub with no §5 still shows the comparison anchor).
func TestCertificateComparisonAnchorIndependentOfOTS(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	// No seedOTS: the accepted root has no mirrored OTS row, so §5 is omitted.
	st, _ := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	// §5 is ABSENT (no OTS row) but the COMPARISON ANCHOR is PRESENT — the two are decoupled.
	if strings.Contains(body, "§5 BITCOIN ANCHOR") {
		t.Errorf("an un-anchored root rendered a §5 BITCOIN ANCHOR clause\n%s", body)
	}
	if !strings.Contains(body, "COMPARISON ANCHOR") {
		t.Errorf("a hub with no §5 OTS row dropped the COMPARISON ANCHOR panel\n%s", body)
	}
}

// TestCertificateComparisonAnchorCoverageJustStarted asserts the honest no-window state:
// a certifiable hub whose coverage time is not recorded (the zero-time case) renders the
// COMPARISON ANCHOR panel WITHOUT a since-time chip — the honest "coverage just started"
// copy — never implying a pre-coverage guarantee (ADR-0001). fixtureStore's
// AdvanceAccepted writes a NULL monitored_since_time (zero ObservedAt), so Coverage.Since
// is zero while Coverage.Size is set.
func TestCertificateComparisonAnchorCoverageJustStarted(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815)
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	if !strings.Contains(body, "COMPARISON ANCHOR") {
		t.Errorf("a certifiable id with NULL coverage-time dropped the COMPARISON ANCHOR panel\n%s", body)
	}
	// fixtureStore's AdvanceAccepted set monitored_since_size (the leaf's accepted size)
	// with a NULL time, so the panel states the size window but no RFC-3339 since-time.
	if !strings.Contains(body, "since size 24816") {
		t.Errorf("comparison-anchor panel missing the NULL-time coverage size window\n%s", body)
	}
}

// otsFixture loads one bundled .ots vector from this package's testdata/ (copied
// verbatim from internal/ots/testdata so the certificate test is hermetic and never
// reads another package's testdata at runtime). hello-world.txt.ots is the external
// `ots verify` oracle's Bitcoin-confirmed vector (block height 358391); merkle1.txt.ots
// is a calendar-only (pending) vector that ots.Confirmed classifies as (false, 0, nil).
func otsFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read OTS fixture %q: %v", name, err)
	}
	return data
}

// seedOTS records an OpenTimestamps row for the accepted (hubID, LastSize, root) of the
// certifiable hub the tiled fixture seeded, mirroring the (hub_id, tree_size, root) key
// the §5 read (and the .ots route / stamp loop) use. The root is the tree's own
// tree.Hash() (the clean fixture's accepted root) and the size its leaf count, matching
// what AdvanceAccepted committed. upgradedAt is the confirmation instant the §5 clause
// renders when the proof is Bitcoin-confirmed (zero → no time chip).
func seedOTS(t *testing.T, st *store.Store, domain string, tree *testonly.Tree, otsBytes []byte, upgradedAt time.Time) {
	t.Helper()
	if _, _, err := st.RecordOTS(context.Background(), store.OTSRecord{
		HubID:      hubIDForDomain(t, st, domain),
		TreeSize:   tree.Size(),
		Root:       tree.Hash(),
		Status:     store.OTSStatusConfirmed,
		OTSBytes:   otsBytes,
		StampedAt:  time.Unix(1700000000, 0),
		UpgradedAt: upgradedAt,
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}
}

// TestCertificateBitcoinAnchorConfirmed is the §5 confirmed path: a certifiable id whose
// accepted root has a mirrored OTS row carrying a Bitcoin-confirmed proof renders the §5
// BITCOIN ANCHOR clause with the confirming block height (358391, the external `ots
// verify` oracle's ground-truth height for hello-world.txt.ots) and the confirmation
// time, while §1-§4/§6 still render (regression). The height is the oracle literal, not
// derived from the certificate code.
func TestCertificateBitcoinAnchorConfirmed(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, tree := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)
	upgradedAt := time.Date(2026, 2, 14, 18, 40, 0, 0, time.UTC)
	seedOTS(t, st, "sb1.amlet.id", tree, otsFixture(t, "hello-world.txt.ots"), upgradedAt)

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	for _, want := range []string{
		"§1 SUBJECT",
		"§2 CHECKPOINT",
		"§3 INCLUSION PROOF",
		"§5 BITCOIN ANCHOR",
		"block 358391", // the external oracle's ground-truth confirmed height
		"§6 RECORD HISTORY",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing §5 confirmed marker %q\n%s", want, body)
		}
	}
	// The confirmation time chip is rendered from the row's upgraded_at (RFC-3339).
	if want := upgradedAt.Format(time.RFC3339); !strings.Contains(body, want) {
		t.Errorf("body missing §5 confirmation time %q\n%s", want, body)
	}
	// The honest "pending" copy must NOT appear for a confirmed anchor.
	if strings.Contains(body, "awaiting Bitcoin confirmation") {
		t.Errorf("a confirmed anchor rendered the pending state\n%s", body)
	}
}

// TestCertificateBitcoinAnchorPending is the §5 honest pending path: a certifiable id
// whose accepted root has a mirrored OTS row carrying a calendar-only (not yet
// Bitcoin-confirmed) proof renders §5 BITCOIN ANCHOR in the "pending" state — never an
// error and never a block height (target.md: a not-yet-anchored root renders the normal
// "pending" state). merkle1.txt.ots is the bundled calendar-only vector ots.Confirmed
// classifies (false, 0, nil).
func TestCertificateBitcoinAnchorPending(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, tree := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)
	seedOTS(t, st, "sb1.amlet.id", tree, otsFixture(t, "merkle1.txt.ots"), time.Time{})

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	if !strings.Contains(body, "§5 BITCOIN ANCHOR") {
		t.Errorf("a pending-anchor id rendered no §5 BITCOIN ANCHOR clause\n%s", body)
	}
	if !strings.Contains(body, "awaiting Bitcoin confirmation") {
		t.Errorf("body missing the §5 honest pending state\n%s", body)
	}
	// A pending anchor must NOT render a confirmed block height (no "block " literal,
	// which would imply a confirmation it does not have).
	if strings.Contains(body, "block ") {
		t.Errorf("a pending anchor rendered a confirmed block height\n%s", body)
	}
}

// TestCertificateBitcoinAnchorUnanchored is the §5 un-anchored decline: a certifiable id
// whose accepted root has NO OTS row renders the page WITHOUT the §5 BITCOIN ANCHOR
// marker (HasClause5 == false), while §1-§4/§6 are unaffected. An un-anchored root
// simply omits §5; it is NOT an error.
//
// Mutation (non-vacuity, review reproduces it): forcing HasClause5 = true
// unconditionally in buildData renders §5 for this un-anchored fixture and makes this
// test FAIL; reverting restores green.
func TestCertificateBitcoinAnchorUnanchored(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	// No seedOTS: the accepted root has no mirrored OTS row.
	st, _ := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	// The rest of the certificate is unaffected (§1-§3 + §6 still render).
	for _, want := range []string{"§1 SUBJECT", "§2 CHECKPOINT", "§3 INCLUSION PROOF", "§6 RECORD HISTORY"} {
		if !strings.Contains(body, want) {
			t.Errorf("un-anchored certificate missing %q (the page must still render §1-§3+§6)\n%s", want, body)
		}
	}
	// §5 must be ABSENT — an un-anchored root omits the clause, never fabricates one.
	if strings.Contains(body, "§5 BITCOIN ANCHOR") {
		t.Errorf("an un-anchored root rendered a §5 BITCOIN ANCHOR clause\n%s", body)
	}
}

// TestCertificateBitcoinAnchorEmptySentinel asserts the empty-OTSBytes sentinel decline:
// a row CAN exist for the accepted root yet carry zero ots_bytes (stamped at observation
// but not yet calendar-submitted, the load-bearing edge case the .ots route guards). §5
// must be OMITTED (HasClause5 == false), never rendered against a zero-byte proof.
func TestCertificateBitcoinAnchorEmptySentinel(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, tree := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)
	// Seed a row with the empty-OTSBytes sentinel for the accepted root.
	seedOTS(t, st, "sb1.amlet.id", tree, nil, time.Time{})

	h := Handler(testnetHubList(), st, nil)
	rec := get(t, h, goldenID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := html.UnescapeString(rec.Body.String())

	if strings.Contains(body, "§5 BITCOIN ANCHOR") {
		t.Errorf("an empty-OTSBytes-sentinel row rendered a §5 BITCOIN ANCHOR clause\n%s", body)
	}
}
