// Golden + mutation HTTP-seam tests for the downloadable proof bundle
// (GET /inclusion/{iscc_id}.bundle). Each drives certificate.Handler over the same
// REAL mirrored-tile fixture the §3 tests use (fixtureStoreTiled), so the bundle is
// assembled from the verified §3 crypto path — not re-derived. The tests assert the
// observable response: a certifiable id's .bundle is application/json with the
// attachment header and a body that decodes to a self-contained bundle, AND its
// embedded inclusion proof re-verifies via logclient.VerifyInclusionEvidence (the
// external-oracle cross-check, the conformance gate this clause re-engages).
//
// The bundle is gated on the §3 re-verification: a non-certifiable id (or a tile gap /
// contradictory tree where §3 declined) is an honest 200 "no proof bundle available",
// never a fabricated bundle and never a 5xx. The contradictory-tree assertion proves
// the gate is load-bearing (the same fixture as TestCertificateInclusionProofContradictory).
package certificate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
)

// decodeBundle decodes a successful .bundle response body into a proofBundle, failing
// the test on any JSON error (a malformed bundle is a bug, not a verdict).
func decodeBundle(t *testing.T, body []byte) proofBundle {
	t.Helper()
	var b proofBundle
	if err := json.Unmarshal(body, &b); err != nil {
		t.Fatalf("decode bundle JSON: %v\nbody: %s", err, body)
	}
	return b
}

// TestCertificateProofBundle is the §-bundle oracle-gated test: a certifiable id whose
// §3 inclusion proof rebuilt the accepted root serves a self-contained proof bundle as
// JSON (attachment header), and the bundle's IsccLogInclusionProof member re-verifies
// against the mirrored tiles via logclient.VerifyInclusionEvidence (the external-oracle
// cross-check). The fixture carries a REAL signed-note checkpoint (the live sb0 note),
// so the key member populates and the bundle is complete.
func TestCertificateProofBundle(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	// Clean, non-frozen hub: the mirror and the accepted root are the SAME tree, so §3
	// re-verifies and the bundle is offered. Use the REAL signed note so the key member
	// populates (key id 0x40b74463 derived from the checkpoint's signature line).
	st, tree := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, raw)

	// Seed the cached did:web key the bundle's key member reads back, keyed on the SAME
	// key id the live checkpoint's signature line carries.
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
	rec := get(t, h, goldenID+bundleSuffix)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") || !strings.Contains(got, goldenID+".bundle.json") {
		t.Errorf("Content-Disposition = %q, want an attachment named %q", got, goldenID+".bundle.json")
	}

	bundle := decodeBundle(t, rec.Body.Bytes())

	// Subject id + resolved hub + DID.
	if bundle.IsccID != goldenID {
		t.Errorf("bundle iscc_id = %q, want %q", bundle.IsccID, goldenID)
	}
	if bundle.Hub.Domain != "sb1.amlet.id" {
		t.Errorf("bundle hub.domain = %q, want sb1.amlet.id", bundle.Hub.Domain)
	}
	if want := "did:web:sb1.amlet.id"; bundle.Hub.DID != want {
		t.Errorf("bundle hub.did = %q, want %q", bundle.Hub.DID, want)
	}

	// Checkpoint is the verbatim signed-note text (NOT base64), the body a client checks
	// the hub signature on.
	if bundle.Checkpoint != string(raw) {
		t.Errorf("bundle checkpoint is not the verbatim signed-note text\n got: %q\nwant: %q", bundle.Checkpoint, raw)
	}

	// The inclusion member is IsccLogInclusionProof-shaped.
	if bundle.Inclusion.Type != "IsccLogInclusionProof" {
		t.Errorf("bundle inclusion.type = %q, want IsccLogInclusionProof", bundle.Inclusion.Type)
	}
	if bundle.Inclusion.TreeSize != tree.Size() {
		t.Errorf("bundle inclusion.treeSize = %d, want %d", bundle.Inclusion.TreeSize, tree.Size())
	}
	if bundle.Inclusion.LeafIndex != seq {
		t.Errorf("bundle inclusion.leafIndex = %d, want %d", bundle.Inclusion.LeafIndex, seq)
	}

	// The inclusion proof hashes must equal the §3 page's ProofHashes (the independent
	// tree prover): a wrong/corrupt bundle hash fails this byte-match.
	want, err := tree.InclusionProof(seq, tree.Size())
	if err != nil {
		t.Fatalf("tree.InclusionProof: %v", err)
	}
	if len(want) < 2 {
		t.Fatalf("proof has %d hashes; want a multi-hash (substantive) proof", len(want))
	}
	if len(bundle.Inclusion.InclusionProof) != len(want) {
		t.Fatalf("bundle has %d proof hashes, want %d", len(bundle.Inclusion.InclusionProof), len(want))
	}
	for i, hsh := range want {
		enc := base64.StdEncoding.EncodeToString(hsh)
		if bundle.Inclusion.InclusionProof[i] != enc {
			t.Errorf("bundle inclusion proof hash %d = %q, want %q", i, bundle.Inclusion.InclusionProof[i], enc)
		}
	}

	// The record member is the base64-Std of the subject leaf's raw record bytes.
	if want := base64.StdEncoding.EncodeToString([]byte("leaf-0")); bundle.Record != want {
		t.Errorf("bundle record = %q, want %q", bundle.Record, want)
	}

	// The key member carries the derived key id (40b74463) + the seeded multibase.
	if bundle.Key == nil {
		t.Fatalf("bundle key member missing (the cached key was seeded)")
	}
	if bundle.Key.ID != "40b74463" {
		t.Errorf("bundle key.id = %q, want 40b74463", bundle.Key.ID)
	}
	if bundle.Key.Multibase != multibase {
		t.Errorf("bundle key.multibase = %q, want %q", bundle.Key.Multibase, multibase)
	}

	// The conformance gate: the assembled InclusionEvidence re-verifies against the
	// mirrored tiles via the external oracle. A bundle that does not feed straight into
	// VerifyInclusionEvidence (wrong shape, wrong hashes, wrong leaf/size) fails here.
	f := store.SQLiteFetcher{Store: st, HubID: hubIDForDomain(t, st, "sb1.amlet.id")}
	if err := logclient.VerifyInclusionEvidence(context.Background(), f.ReadTile, bundle.Inclusion); err != nil {
		t.Errorf("VerifyInclusionEvidence on the served bundle = %v, want nil", err)
	}
}

// TestCertificateProofBundleDIDPortEncoded asserts the proof bundle's hub.did
// percent-encodes a host:port hub's port colon: a hub resolved to localhost:8443
// serves a bundle whose Hub.DID is did:web:localhost%3A8443 (NOT the malformed
// did:web:localhost:8443, which would name a different did.json than the key
// resolved from), while hub.domain stays the verbatim localhost:8443. A no-port
// hub's bundle DID is regression-covered by TestCertificateProofBundle
// (did:web:sb1.amlet.id).
//
// Mutation (non-vacuity): reverting didWeb at the bundle site to "did:web:" +
// data.Domain makes this test FAIL — the bundle DID would be did:web:localhost:8443.
func TestCertificateProofBundleDIDPortEncoded(t *testing.T) {
	const seq = 0
	const leaves = 5
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, _ := fixtureStoreTiled(t, "localhost:8443", goldenID, seq, leaves, nil, false, raw)

	h := Handler(hostPortHubList(), st, nil)
	rec := get(t, h, goldenID+bundleSuffix)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	bundle := decodeBundle(t, rec.Body.Bytes())

	if bundle.Hub.Domain != "localhost:8443" {
		t.Errorf("bundle hub.domain = %q, want localhost:8443 (verbatim, not encoded)", bundle.Hub.Domain)
	}
	if want := "did:web:localhost%3A8443"; bundle.Hub.DID != want {
		t.Errorf("bundle hub.did = %q, want %q (the port colon must be %%3A-encoded)", bundle.Hub.DID, want)
	}
}

// TestCertificateProofBundleLinkRendered asserts the certificate HTML page wires the
// download action: a certifiable id renders an enabled <a href="…bundle"> download
// link (not the disabled placeholder), and a non-certifiable id keeps the disabled
// placeholder. The link is what makes the bundle reachable from the page.
//
// The href is the canonical path-rooted, ISCC:-prefix-free form (/inclusion/<bare
// id>.bundle) for BOTH request id forms. The ISCC:-prefixed sub-case is the
// regression guard: rendering the raw .IsccID into the href made html/template's URL
// escaper read the leading ISCC: as a scheme and emit the #ZgotmplZ sentinel
// (href="#ZgotmplZ.bundle"), a dead link to the headline affordance. The bare form
// hid it because it has no leading scheme.
func TestCertificateProofBundleLinkRendered(t *testing.T) {
	const seq = 0
	const leaves = 5
	st, _ := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, nil, false, []byte("raw"))
	h := Handler(testnetHubList(), st, nil)

	// The canonical href both id forms must resolve to: path-rooted at the mount,
	// ISCC:-prefix-free. The .bundle handler decodes the bare form identically.
	wantHref := `href="` + PathPrefix + goldenID + bundleSuffix + `"`

	// Both id forms certify the same leaf and must render the SAME working href with no
	// #ZgotmplZ. The ISCC:-prefixed form is the regression guard for the URL-escaper bug.
	for _, reqID := range []string{goldenID, "ISCC:" + goldenID} {
		rec := get(t, h, reqID)
		if rec.Code != http.StatusOK {
			t.Fatalf("id %q: status = %d, want 200", reqID, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, wantHref) {
			t.Errorf("id %q: certifiable page missing working download link %q\n%s", reqID, wantHref, body)
		}
		if strings.Contains(body, "#ZgotmplZ") {
			t.Errorf("id %q: download href was filtered to the #ZgotmplZ sentinel (dead link)\n%s", reqID, body)
		}
		if strings.Contains(body, "Download proof bundle (coming soon)") {
			t.Errorf("id %q: certifiable page still shows the disabled placeholder\n%s", reqID, body)
		}
	}

	// Non-certifiable id (malformed): the disabled placeholder stays. (A malformed id
	// renders the not-found state, which carries no actions block, so the assertion is
	// on the bundle-link absence — there must be no enabled .bundle link.)
	bad := get(t, h, "not-an-iscc-id")
	if bad.Code != http.StatusOK {
		t.Fatalf("malformed id status = %d, want 200", bad.Code)
	}
	if strings.Contains(bad.Body.String(), ".bundle\"") {
		t.Errorf("a non-certifiable id rendered a .bundle download link\n%s", bad.Body.String())
	}
}

// TestCertificateProofBundleTileGap asserts the honest-gap path: a certifiable id whose
// tiles are NOT mirrored (the §1/§2-only fixture) declined §3, so its .bundle request
// is an honest 200 "no proof bundle available" — never a fabricated bundle, never a 5xx.
func TestCertificateProofBundleTileGap(t *testing.T) {
	st := fixtureStore(t, "sb1.amlet.id", goldenID, 24815) // no tiles mirrored
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID+bundleSuffix)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (an honest 'not available', never a 5xx)", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got := rec.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("a not-available response set an attachment header %q", got)
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode not-available JSON: %v\nbody: %s", err, rec.Body.String())
	}
	if resp["error"] == "" {
		t.Errorf("a tile-gap .bundle response carried no error message: %v", resp)
	}
	if resp["iscc_id"] != goldenID {
		t.Errorf("not-available response iscc_id = %q, want %q", resp["iscc_id"], goldenID)
	}
	// It must NOT be a fabricated bundle (no inclusion / checkpoint members).
	if strings.Contains(rec.Body.String(), "IsccLogInclusionProof") {
		t.Errorf("a tile-gap .bundle response fabricated an inclusion proof\n%s", rec.Body.String())
	}
}

// TestCertificateProofBundleContradictory is the mutation-proving fail-closed test
// (reviewer reproduces it): a NON-frozen hub whose mirrored tiles disagree with its
// accepted root (the fork-poll window) declined §3, so its .bundle is an honest 200
// "no proof bundle available" with NO bundle — never a fabricated bundle whose proof
// would not re-verify. This proves the bundle is gated on the §3 re-verification, not a
// status flag: replacing the §3 proof.VerifyInclusion guard with `|| true` (the
// load-bearing mutation) makes TestCertificateInclusionProofContradictory fail and
// would here serve a bundle whose VerifyInclusionEvidence returns non-nil.
func TestCertificateProofBundleContradictory(t *testing.T) {
	const seq = 0
	const leaves = 5

	// Tree B is a DIFFERENT 5-leaf tree, so its root differs from the mirrored tree A's
	// — the accepted root the §3 proof would rebuild does not match the mirrored tiles.
	treeB := testonly.New(rfc6962.DefaultHasher)
	for i := range leaves {
		treeB.AppendData([]byte(string(rune('a'+i)) + "-forked"))
	}
	// freeze=false: only the fail-closed re-verification withholds the bundle.
	st, treeA := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, seq, leaves, treeB.Hash(), false, []byte("raw"))
	if string(treeA.Hash()) == string(treeB.Hash()) {
		t.Fatalf("treeA and treeB share a root; the fixture is not contradictory")
	}
	h := Handler(testnetHubList(), st, nil)

	rec := get(t, h, goldenID+bundleSuffix)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (honest not-available, never a 5xx)", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "IsccLogInclusionProof") {
		t.Errorf("a non-frozen contradictory-tile hub served a fabricated proof bundle\n%s", rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode not-available JSON: %v\nbody: %s", err, rec.Body.String())
	}
	if resp["error"] == "" {
		t.Errorf("a contradictory-tile .bundle response carried no error message: %v", resp)
	}
}
