// Tests for certificate §5's covering anchor: the earliest confirmed Bitcoin anchor
// whose log state includes the subject leaf. On a hub that grows daily the accepted
// root is always younger than a confirmation takes, so §5 must reach back to an older
// anchored state; these tests build exactly that (an accepted tree of 8 leaves whose
// own root is pending, with anchors at earlier sizes) over the real mirrored-tile
// fixture.
//
// The independent oracles are merkle's testonly.Tree (HashAt / InclusionProof for the
// older state, never the monitor's proof builder) and proof.VerifyInclusion run the
// way a client would on the downloaded bundle. A confirmed proof for an arbitrary root
// is the bundled real Bitcoin-attested vector re-pointed at that root: ots.ConfirmedFor
// checks the committed digest and the attestation, never the chain, so the swap is
// faithful to what production checks. Every decline case removes exactly one of the
// three checks' preconditions and asserts §5 falls back to the accepted root's
// "pending" state with no anchor link and no bundle anchor member.
package certificate

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"strings"
	"testing"
	"time"

	opentimestamps "github.com/nbd-wtf/opentimestamps"
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"

	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/store"
)

// anchorSeenAt is when the seeded confirmed anchors were seen confirmed.
var anchorSeenAt = time.Date(2026, 11, 6, 16, 28, 46, 0, time.UTC)

// anchorLeaves and anchorSeq shape every covering-anchor fixture: the accepted tree has
// 8 leaves and the subject is leaf 2, so states of size 3 and up include it.
const (
	anchorLeaves = 8
	anchorSeq    = 2
)

// otsFor returns the named .ots fixture re-pointed at root. A non-zero height replaces
// the Bitcoin attestation's block height so each anchor carries a distinct one.
func otsFor(t *testing.T, name string, root []byte, height uint64) []byte {
	t.Helper()
	file, err := opentimestamps.ReadFromFile(otsFixture(t, name))
	if err != nil {
		t.Fatalf("parse OTS fixture %q: %v", name, err)
	}
	file.Digest = root
	for _, seq := range file.Sequences {
		for _, inst := range seq {
			if height > 0 && inst.Attestation != nil && inst.Attestation.CalendarServerURL == "" {
				inst.Attestation.BitcoinBlockHeight = height
			}
		}
	}
	return file.SerializeToFile()
}

// anchorSeed describes one OTS row and the earlier poll that would have produced it.
// The zero value of each override means "as production writes it": the state is
// mirrored, the checkpoint at size is recorded with the tree's real root, the row is
// confirmed and keyed on that root, and its proof commits to it.
type anchorSeed struct {
	size           uint64
	height         uint64
	pending        bool
	skipMirror     bool
	skipCheckpoint bool
	checkpointRoot []byte
	rowRoot        []byte
	proofRoot      []byte
}

// seedAnchor records seed's earlier poll (mirror + checkpoint signed as raw) and its
// OTS row for the hub at domain.
func seedAnchor(t *testing.T, st *store.Store, domain string, tree *testonly.Tree, raw []byte, seed anchorSeed) {
	t.Helper()
	ctx := context.Background()
	hubID := hubIDForDomain(t, st, domain)
	treeRoot := tree.HashAt(seed.size)
	if !seed.skipMirror {
		ingestMirror(t, st, hubID, tree, seed.size)
	}
	if !seed.skipCheckpoint {
		root := treeRoot
		if seed.checkpointRoot != nil {
			root = seed.checkpointRoot
		}
		if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{HubID: hubID, TreeSize: seed.size, Root: root, Raw: raw}); err != nil {
			t.Fatalf("RecordCheckpoint size %d: %v", seed.size, err)
		}
	}
	rowRoot := treeRoot
	if seed.rowRoot != nil {
		rowRoot = seed.rowRoot
	}
	proofRoot := rowRoot
	if seed.proofRoot != nil {
		proofRoot = seed.proofRoot
	}
	row := store.OTSRecord{
		HubID:      hubID,
		TreeSize:   seed.size,
		Root:       rowRoot,
		Status:     store.OTSStatusConfirmed,
		OTSBytes:   otsFor(t, "hello-world.txt.ots", proofRoot, seed.height),
		StampedAt:  time.Unix(1700000000, 0),
		UpgradedAt: anchorSeenAt,
	}
	if seed.pending {
		row.Status = store.OTSStatusPending
		row.OTSBytes = otsFor(t, "merkle1.txt.ots", proofRoot, 0)
		row.UpgradedAt = time.Time{}
	}
	if _, _, err := st.RecordOTS(ctx, row); err != nil {
		t.Fatalf("RecordOTS size %d: %v", seed.size, err)
	}
}

// anchoredFixture builds the growing-hub fixture: 8 accepted leaves whose accepted root
// has only a pending, calendar-submitted proof, plus the given earlier anchors. Every
// checkpoint carries the live signed note, so the anchor's signing key resolves once a
// test seeds it.
func anchoredFixture(t *testing.T, seeds ...anchorSeed) (*store.Store, *testonly.Tree, []byte) {
	t.Helper()
	raw := liveCheckpointRaw(t, "sb0.iscc.id_checkpoint")
	st, tree := fixtureStoreTiled(t, "sb1.amlet.id", goldenID, anchorSeq, anchorLeaves, nil, false, raw)
	if _, _, err := st.RecordOTS(context.Background(), store.OTSRecord{
		HubID:     hubIDForDomain(t, st, "sb1.amlet.id"),
		TreeSize:  tree.Size(),
		Root:      tree.Hash(),
		Status:    store.OTSStatusPending,
		OTSBytes:  otsFor(t, "merkle1.txt.ots", tree.Hash(), 0),
		StampedAt: time.Unix(1700000000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS accepted root: %v", err)
	}
	for _, seed := range seeds {
		seedAnchor(t, st, "sb1.amlet.id", tree, raw, seed)
	}
	// Mirror the accepted size again, last, the order the follower polls in. A mirror
	// write that dropped a coord's narrower partial rows would now drop the earlier
	// states' rows, and the covering-anchor tests would fail.
	ingestMirror(t, st, hubIDForDomain(t, st, "sb1.amlet.id"), tree, tree.Size())
	return st, tree, raw
}

// TestCertificateCoveringAnchor is the case §5's covering anchor exists for: the
// accepted root is pending, and anchors exist at sizes 2, 5 and 7. Size 2 ends before
// leaf 2, so the earliest anchor that includes the leaf is size 5. §5 must show that
// anchor's own block, the size it covers, its .ots link and the digest to check it
// against; the bundle must carry the checkpoint at size 5, a leaf-2 inclusion proof
// against it that equals the independent prover's and verifies like a client would
// verify it, the anchor's signing key, and the .ots that commits to the size-5 root.
func TestCertificateCoveringAnchor(t *testing.T) {
	st, tree, raw := anchoredFixture(t,
		anchorSeed{size: 2, height: 900002},
		anchorSeed{size: 5, height: 900005},
		anchorSeed{size: 7, height: 900007},
	)
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
	root5 := tree.HashAt(5)
	h := Handler(testnetHubList(), st, nil, dashboard.Identity{})

	body := renderCertBody(t, st, goldenID)
	for _, want := range []string{
		"block 900005 · seen confirmed 2026-11-06T16:28:46Z",
		"log state · size 5",
		`href="/sb1.amlet.id/log/checkpoint.ots?size=5"`,
		"ots verify -d " + hex.EncodeToString(root5) + " checkpoint.ots",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n%s", want, body)
		}
	}
	for _, absent := range []string{"awaiting Bitcoin confirmation", "block 900002", "block 900007"} {
		if strings.Contains(body, absent) {
			t.Errorf("body contains %q\n%s", absent, body)
		}
	}

	rec := get(t, h, goldenID+bundleSuffix)
	if rec.Code != http.StatusOK {
		t.Fatalf("bundle status = %d, want 200", rec.Code)
	}
	bundle := decodeBundle(t, rec.Body.Bytes())
	if bundle.Inclusion.TreeSize != anchorLeaves {
		t.Errorf("top-level inclusion.treeSize = %d, want the accepted %d", bundle.Inclusion.TreeSize, anchorLeaves)
	}
	a := bundle.Anchor
	if a == nil {
		t.Fatalf("bundle has no anchor member\n%s", rec.Body.String())
	}
	if a.Checkpoint != string(raw) || a.Inclusion.Checkpoint != string(raw) {
		t.Errorf("anchor checkpoint is not the signed note recorded at size 5")
	}
	if a.Inclusion.Type != "IsccLogInclusionProof" || a.Inclusion.TreeSize != 5 || a.Inclusion.LeafIndex != anchorSeq {
		t.Errorf("anchor inclusion = %s size %d leaf %d, want IsccLogInclusionProof size 5 leaf %d", a.Inclusion.Type, a.Inclusion.TreeSize, a.Inclusion.LeafIndex, anchorSeq)
	}
	want, err := tree.InclusionProof(anchorSeq, 5)
	if err != nil {
		t.Fatalf("tree.InclusionProof: %v", err)
	}
	if got := strings.Join(a.Inclusion.InclusionProof, ","); got != strings.Join(encodeHashes(want), ",") {
		t.Errorf("anchor proof = %s, want the independent prover's %s", got, strings.Join(encodeHashes(want), ","))
	}
	decoded := make([][]byte, len(a.Inclusion.InclusionProof))
	for i, s := range a.Inclusion.InclusionProof {
		if decoded[i], err = base64.StdEncoding.DecodeString(s); err != nil {
			t.Fatalf("decode proof hash %d: %v", i, err)
		}
	}
	record, err := base64.StdEncoding.DecodeString(bundle.Record)
	if err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if err := proof.VerifyInclusion(rfc6962.DefaultHasher, anchorSeq, 5, rfc6962.DefaultHasher.HashLeaf(record), decoded, root5); err != nil {
		t.Errorf("anchor proof does not verify against the size-5 root: %v", err)
	}
	if a.Key == nil || a.Key.ID != fmt.Sprintf("%08x", sb0CheckpointKeyID) || a.Key.Multibase != multibase {
		t.Errorf("anchor key = %+v, want the cached key that signed the size-5 checkpoint", a.Key)
	}
	otsBytes, err := base64.StdEncoding.DecodeString(a.OTS)
	if err != nil {
		t.Fatalf("decode anchor ots: %v", err)
	}
	file, err := opentimestamps.ReadFromFile(otsBytes)
	if err != nil {
		t.Fatalf("anchor ots does not parse: %v", err)
	}
	if hex.EncodeToString(file.Digest) != hex.EncodeToString(root5) {
		t.Errorf("anchor ots commits to %x, want the size-5 root %x", file.Digest, root5)
	}
}

// TestCertificateCoveringAnchorWithoutCachedKey asserts the anchor member omits its key,
// rather than fabricate one, when the key that signed the anchored checkpoint is not
// cached, while §5 and the rest of the anchor still render.
func TestCertificateCoveringAnchorWithoutCachedKey(t *testing.T) {
	st, _, _ := anchoredFixture(t, anchorSeed{size: 5, height: 900005})
	if body := renderCertBody(t, st, goldenID); !strings.Contains(body, "log state · size 5") {
		t.Errorf("§5 did not render the covering anchor\n%s", body)
	}
	bundle := decodeBundle(t, get(t, Handler(testnetHubList(), st, nil, dashboard.Identity{}), goldenID+bundleSuffix).Body.Bytes())
	if bundle.Anchor == nil {
		t.Fatal("bundle has no anchor member")
	}
	if bundle.Anchor.Key != nil {
		t.Errorf("anchor key = %+v, want omitted for an uncached key", bundle.Anchor.Key)
	}
}

// TestCertificateCoveringAnchorDeclines removes one precondition per case. §5 must fall
// back to the accepted root's pending state, with no anchor link, no covered size and
// no bundle anchor member: the page never vouches for an anchor it could not check.
func TestCertificateCoveringAnchorDeclines(t *testing.T) {
	other := testonly.New(rfc6962.DefaultHasher)
	other.AppendData([]byte("x"), []byte("y"), []byte("z"), []byte("w"), []byte("v"))
	otherRoot := other.Hash()

	cases := []struct {
		name string
		seed anchorSeed
	}{
		{"state ends before the leaf", anchorSeed{size: 2}},
		{"anchor still pending", anchorSeed{size: 5, pending: true}},
		{"proof commits to another root", anchorSeed{size: 5, proofRoot: otherRoot}},
		{"no checkpoint accepted at the anchored size", anchorSeed{size: 5, skipCheckpoint: true}},
		{"checkpoint at that size has another root", anchorSeed{size: 5, checkpointRoot: otherRoot}},
		{"older state not mirrored", anchorSeed{size: 5, skipMirror: true}},
		{"leaf proof does not rebuild the anchored root", anchorSeed{size: 5, checkpointRoot: otherRoot, rowRoot: otherRoot}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st, _, _ := anchoredFixture(t, tc.seed)
			body := renderCertBody(t, st, goldenID)
			if !strings.Contains(body, "awaiting Bitcoin confirmation") {
				t.Errorf("§5 did not fall back to the accepted root's pending state\n%s", body)
			}
			for _, absent := range []string{"checkpoint.ots?size=", "log state · size", "ots verify", "block "} {
				if strings.Contains(body, absent) {
					t.Errorf("body contains %q\n%s", absent, body)
				}
			}
			rec := get(t, Handler(testnetHubList(), st, nil, dashboard.Identity{}), goldenID+bundleSuffix)
			if bundle := decodeBundle(t, rec.Body.Bytes()); bundle.Anchor != nil {
				t.Errorf("bundle carries an anchor member it could not check: %+v", bundle.Anchor)
			}
		})
	}
}

// TestCertificateCoveringAnchorHTMLSafe guards the link's encoding: the href is a
// path-rooted URL the template passes through, never the #ZgotmplZ sentinel, and the
// rendered page stays parseable after unescaping.
func TestCertificateCoveringAnchorHTMLSafe(t *testing.T) {
	st, _, _ := anchoredFixture(t, anchorSeed{size: 5, height: 900005})
	rec := get(t, Handler(testnetHubList(), st, nil, dashboard.Identity{}), goldenID)
	raw := rec.Body.String()
	if strings.Contains(raw, "ZgotmplZ") {
		t.Errorf("anchor link rendered the template's unsafe-URL sentinel\n%s", raw)
	}
	if !strings.Contains(html.UnescapeString(raw), `href="/sb1.amlet.id/log/checkpoint.ots?size=5" download`) {
		t.Errorf("anchor link missing or not a download link\n%s", raw)
	}
}
