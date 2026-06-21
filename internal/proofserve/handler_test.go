// Conformance test for the computed-inclusion-proof HTTP surface: it builds one
// internally-consistent 300-leaf tlog-tiles log in-process with merkle's
// testonly.Tree (the single source of truth for leaf and node hashes), mirrors its
// hash tiles into a fresh store, indexes each leaf's iscc_id via the production
// iscc_index writer, and records + advances the accepted checkpoint — exactly what a
// verified poll leaves behind. It then drives proofserve.Handler over the real
// SQLiteFetcher and asserts the served inclusionProof base64-decodes to bytes that
// proof.VerifyInclusion ACCEPTS against the tree's root, for leaves on both sides of
// the 256-leaf tile boundary.
//
// Non-circularity / non-vacuousness (oracle gate APPLIES — this serves RFC-6962
// inclusion crypto). Three independent paths meet: the tree (the prover, owning the
// real inclusion proof and the root), InclusionProofFromTiles inside the handler (the
// monitor recompute over the mirror this test wrote), and proof.VerifyInclusion (the
// independent verifier). A green-but-wrong handler serving an empty or constant proof
// would fail proof.VerifyInclusion; the WrongLeaf negative pins that serving leaf A's
// proof for leaf B does not verify against B's leaf hash.
package proofserve

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"

	"github.com/iscc/iscc-monitor/internal/logclient"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// mirrorLeaves is the fixture log size: > 256 so the mirror crosses the 256-leaf
// tile boundary (a full level-0 tile at index 0, a 44-leaf partial at index 1, and a
// level-1 root tile), exercising the proof builder over both the full and partial
// tile qualifiers.
const mirrorLeaves = 300

// leafISCCID returns the distinct iscc_id the leaf at index i carries — a synthetic
// ISCC:-prefixed string unique per leaf so the projection read-back is a clean
// one-seq-per-id lookup (SeqsForISCCID returns exactly [seq]).
func leafISCCID(i int) string {
	return fmt.Sprintf("ISCC:LEAF%08d", i)
}

// nodeHash recomputes the tree-node hash at (treeLevel, treeIndex) by folding the
// leaf hashes it covers through a compact range, matching the proof builder's
// recompute so the synthesized tiles are byte-accurate with the tree. The node
// covers leaves [treeIndex<<treeLevel, (treeIndex+1)<<treeLevel), clamped to size.
func nodeHash(t *testing.T, tree *testonly.Tree, treeLevel, treeIndex, size uint64) []byte {
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

// mirrorTree holds the fixture's tree and the store/hub it was mirrored into, so a
// test can compute the hub-side proof (tree.InclusionProof) and the accepted root
// (tree.HashAt) while driving the handler over the real SQLiteFetcher.
type mirrorTree struct {
	tree  *testonly.Tree
	store *store.Store
	hubID int64
	size  uint64
}

// buildMirror creates a fresh store, builds a leaves-leaf testonly.Tree, mirrors its
// byte-accurate hash tiles into the tiles table, folds one iscc_index projection per
// leaf (so SeqsForISCCID resolves each leaf), and records + advances the accepted
// checkpoint to size (so FollowState.LastSize is the tree size). This reproduces the
// state a verified poll leaves behind, which is all the handler reads.
func buildMirror(t *testing.T, leaves int) mirrorTree {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(filepath.Join(t.TempDir(), "mirror.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	tree := testonly.New(rfc6962.DefaultHasher)
	data := make([][]byte, leaves)
	for i := range data {
		data[i] = []byte(fmt.Sprintf("leaf-%08d", i))
	}
	tree.AppendData(data...)
	size := tree.Size()

	// Mirror every hash tile byte-accurately: each node at (tileLevel, tileIndex) is
	// the RFC-6962 node hash at the matching tree position, recomputed via a compact
	// range over the leaf hashes it covers — the same recompute the proof builder's
	// getNode performs — so the served proof is real ground truth.
	at := time.Unix(1700000000, 0)
	for _, c := range tiles.TileCoords(size) {
		treeLevel := c.Level * uint64(tiles.TileHeight)
		first := c.Index * tiles.TileWidth
		var nodes [][]byte
		for n := uint64(0); n < tiles.TileWidth; n++ {
			treeIndex := first + n
			if (treeIndex << treeLevel) >= size {
				break
			}
			nodes = append(nodes, nodeHash(t, tree, treeLevel, treeIndex, size))
		}
		raw, err := api.HashTile{Nodes: nodes}.MarshalText()
		if err != nil {
			t.Fatalf("HashTile.MarshalText (level %d index %d): %v", c.Level, c.Index, err)
		}
		if err := st.RecordTile(ctx, hubID, c.Level, c.Index, c.Partial, raw, at); err != nil {
			t.Fatalf("RecordTile (level %d index %d): %v", c.Level, c.Index, err)
		}
	}

	// One projection per leaf so SeqsForISCCID(leafISCCID(i)) resolves to [i].
	recs := make([]store.ProjectionRecord, leaves)
	for i := range recs {
		recs[i] = store.ProjectionRecord{HubID: hubID, Seq: uint64(i), IsccID: leafISCCID(i)}
	}
	if err := st.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	// Record the accepted checkpoint and advance the follow cursor to size, so the
	// handler proves against the tree the monitor vouches for.
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hubID, TreeSize: size, Root: tree.Hash(), Raw: []byte("checkpoint"), ObservedAt: at,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, size); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	return mirrorTree{tree: tree, store: st, hubID: hubID, size: size}
}

// getEvidence drives the handler with the given query string and decodes the JSON
// body, returning the status code and the parsed evidence.
func getEvidence(t *testing.T, h http.Handler, query string) (int, logclient.InclusionEvidence) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/inclusion?"+query, nil))
	if rec.Code != http.StatusOK {
		return rec.Code, logclient.InclusionEvidence{}
	}
	var ev logclient.InclusionEvidence
	if err := json.Unmarshal(rec.Body.Bytes(), &ev); err != nil {
		t.Fatalf("decode evidence body %q: %v", rec.Body.String(), err)
	}
	return rec.Code, ev
}

// decodeProof base64-Std-decodes the served inclusionProof hashes.
func decodeProof(t *testing.T, ev logclient.InclusionEvidence) [][]byte {
	t.Helper()
	out := make([][]byte, len(ev.InclusionProof))
	for i, enc := range ev.InclusionProof {
		h, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			t.Fatalf("decode inclusionProof[%d] %q: %v", i, enc, err)
		}
		out[i] = h
	}
	return out
}

// TestInclusionServedProofVerifies proves the handler serves a real RFC-6962
// inclusion proof: for leaves on both sides of the 256-leaf tile boundary, the served
// proof decodes to bytes proof.VerifyInclusion ACCEPTS against the tree's root, and
// re-labelling it for a different leaf is REJECTED.
func TestInclusionServedProofVerifies(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID)
	root := m.tree.HashAt(m.size)

	for _, leaf := range []int{0, 5, 255, 256, 260, 299} {
		code, ev := getEvidence(t, h, "iscc_id="+leafISCCID(leaf))
		if code != http.StatusOK {
			t.Fatalf("leaf %d: status = %d, want 200", leaf, code)
		}
		if ev.Type != "IsccLogInclusionProof" {
			t.Errorf("leaf %d: type = %q, want IsccLogInclusionProof", leaf, ev.Type)
		}
		if ev.TreeSize != m.size {
			t.Errorf("leaf %d: treeSize = %d, want %d", leaf, ev.TreeSize, m.size)
		}
		if ev.LeafIndex != uint64(leaf) {
			t.Errorf("leaf %d: leafIndex = %d, want %d", leaf, ev.LeafIndex, leaf)
		}

		got := decodeProof(t, ev)
		// The served proof must verify against the tree's root — real ground truth,
		// not just a 200 with an empty/constant proof.
		if err := proof.VerifyInclusion(rfc6962.DefaultHasher, uint64(leaf), m.size, m.tree.LeafHash(uint64(leaf)), got, root); err != nil {
			t.Errorf("leaf %d: served proof does not verify: %v", leaf, err)
		}
		// The same proof must NOT verify for a different leaf's leaf hash (the sharp
		// negative — a green-but-wrong constant proof would also fail VerifyInclusion).
		other := uint64((leaf + 1) % int(m.size))
		if err := proof.VerifyInclusion(rfc6962.DefaultHasher, other, m.size, m.tree.LeafHash(other), got, root); err == nil {
			t.Errorf("leaf %d: served proof wrongly verifies for leaf %d", leaf, other)
		}
	}
}

// TestInclusionUnknownISCCID asserts an iscc_id never indexed is a 404, not a 200
// with an empty proof.
func TestInclusionUnknownISCCID(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID)
	code, _ := getEvidence(t, h, "iscc_id=ISCC:NOSUCHLEAF")
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

// TestInclusionMissingParam asserts a request without iscc_id is a 400.
func TestInclusionMissingParam(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/inclusion", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestInclusionNonGET asserts a non-GET method is a 405.
func TestInclusionNonGET(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/inclusion?iscc_id="+leafISCCID(1), nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// TestInclusionNoAcceptedCheckpoint asserts that with no accepted checkpoint yet
// (LastSize == 0) the handler is a 404 — there is nothing to prove against — even
// when the iscc_id is indexed.
func TestInclusionNoAcceptedCheckpoint(t *testing.T) {
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
	// Index a leaf but never advance the follow cursor: LastSize stays 0.
	if err := st.RecordProjections(ctx, []store.ProjectionRecord{{HubID: hubID, Seq: 0, IsccID: leafISCCID(0)}}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	h := Handler(st, hubID)
	code, _ := getEvidence(t, h, "iscc_id="+leafISCCID(0))
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

// TestInclusionUnmatchedPath asserts a path other than /inclusion under the handler
// is a 404 (the handler owns only the proof route; the mirror routes belong to
// tilesserve).
func TestInclusionUnmatchedPath(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/checkpoint", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
