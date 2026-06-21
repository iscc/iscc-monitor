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
	if indexDomain == "sb1.amlet.id" {
		target = id1
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
