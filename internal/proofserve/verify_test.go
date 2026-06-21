// Conformance test for the verify-for-me HTTP surface (/verify?iscc_id=<id>): it
// builds a 300-leaf internally-consistent tlog-tiles mirror — byte-accurate hash
// tiles AND entry bundles framed from the same leaf data — and drives
// proofserve.Handler's /verify route over the real SQLiteFetcher, asserting the
// single JSON verdict carries the accepted (size, root), the hub status, and a REAL
// RFC-6962 inclusion result Merkle-verified against the accepted root, for leaves on
// both sides of the 256-leaf tile boundary.
//
// Non-circularity / non-vacuousness (oracle gate APPLIES — the verdict's inclusion
// result is RFC-6962 crypto, not a stub). The fixture tree owns the real root; the
// handler reads the leaf bytes from the mirrored entry bundle, recomputes the
// inclusion proof over the mirrored tiles, and runs proof.VerifyInclusion itself;
// the test independently re-derives the same root from the tree and asserts the
// served root base64-decodes to it. A handler that hard-codes verified:true fails
// the unknown-id case; one that drops proof.VerifyInclusion fails the corrupted-root
// negative (TestVerifyInclusionIsNonVacuous).
package proofserve

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"

	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// verifyLeafData returns the raw leaf bytes the leaf at index i carries — the same
// bytes the fixture tree is built from, so rfc6962.HashLeaf(verifyLeafData(i)) equals
// tree.LeafHash(i) and the entry-bundle leaf the handler reads hashes to the tile the
// proof commits to. (buildMirror in handler_test.go uses the identical "leaf-%08d".)
func verifyLeafData(i int) []byte {
	return []byte(fmt.Sprintf("leaf-%08d", i))
}

// frameVerifyBundle frames raw records into a tlog-tiles entry bundle (big-endian
// uint16 length prefix per record), matching the C2SP encoding
// api.EntryBundle.UnmarshalText decodes — the independent encode path, never imported
// from the code under test.
func frameVerifyBundle(records [][]byte) []byte {
	var out []byte
	for _, rec := range records {
		var prefix [2]byte
		binary.BigEndian.PutUint16(prefix[:], uint16(len(rec)))
		out = append(out, prefix[:]...)
		out = append(out, rec...)
	}
	return out
}

// buildVerifyMirror builds a fresh store with a leaves-leaf testonly.Tree, mirrors
// its byte-accurate hash tiles AND entry bundles (framed from the tree's leaf data),
// folds one iscc_index projection per leaf, and records + advances the accepted
// checkpoint to size. When corruptRoot is true the recorded accepted root has its
// first byte flipped (the ONLY checkpoint row at size), so the recomputed inclusion
// proof builds from real tiles but does NOT verify against the stored root — the
// negative the non-vacuousness guard drives. This reproduces the state a verified
// poll leaves behind, which is all the handler reads.
func buildVerifyMirror(t *testing.T, leaves int, corruptRoot bool) mirrorTree {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(filepath.Join(t.TempDir(), "verify.db"))
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
		data[i] = verifyLeafData(i)
	}
	tree.AppendData(data...)
	size := tree.Size()

	at := time.Unix(1700000000, 0)

	// Mirror every hash tile byte-accurately (the same recompute buildMirror does).
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

	// Mirror every entry bundle, framed from the tree's leaf data, so the leaf the
	// handler reads back hashes to the tile the proof commits to.
	for first := 0; first < leaves; first += tiles.TileWidth {
		last := first + tiles.TileWidth
		if last > leaves {
			last = leaves
		}
		recs := make([][]byte, 0, last-first)
		for i := first; i < last; i++ {
			recs = append(recs, verifyLeafData(i))
		}
		width := last - first
		p := uint8(width)
		if width == tiles.TileWidth {
			p = 0
		}
		bundleIndex := uint64(first / tiles.TileWidth)
		if err := st.RecordEntryBundle(ctx, hubID, bundleIndex, p, frameVerifyBundle(recs), at); err != nil {
			t.Fatalf("RecordEntryBundle (index %d width %d): %v", bundleIndex, width, err)
		}
	}

	// One projection per leaf so SeqsForISCCID(leafISCCID(i)) resolves to [i].
	projs := make([]store.ProjectionRecord, leaves)
	for i := range projs {
		projs[i] = store.ProjectionRecord{HubID: hubID, Seq: uint64(i), IsccID: leafISCCID(i)}
	}
	if err := st.RecordProjections(ctx, projs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	root := append([]byte(nil), tree.Hash()...)
	if corruptRoot {
		root[0] ^= 0xff
	}
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hubID, TreeSize: size, Root: root, Raw: []byte("checkpoint"), ObservedAt: at,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, size); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	return mirrorTree{tree: tree, store: st, hubID: hubID, size: size}
}

// getVerdict drives the /verify route with the given query string and decodes the
// JSON verdict, returning the status code and the parsed verdict.
func getVerdict(t *testing.T, h http.Handler, query string) (int, VerifyVerdict) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/verify?"+query, nil))
	var v VerifyVerdict
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatalf("decode verdict body %q: %v", rec.Body.String(), err)
		}
	}
	return rec.Code, v
}

// TestVerifyKnownISCCID proves the verify-for-me verdict carries a real,
// Merkle-verified inclusion result and the accepted (size, root) for leaves on both
// sides of the 256-leaf tile boundary (the >=256 leaves exercise the partial
// entry-bundle read path).
func TestVerifyKnownISCCID(t *testing.T) {
	m := buildVerifyMirror(t, mirrorLeaves, false)
	h := Handler(m.store, m.hubID, nil)
	wantRoot := base64.StdEncoding.EncodeToString(m.tree.HashAt(m.size))

	for _, leaf := range []int{0, 5, 255, 256, 260, 299} {
		code, v := getVerdict(t, h, "iscc_id="+leafISCCID(leaf))
		if code != http.StatusOK {
			t.Fatalf("leaf %d: status = %d, want 200", leaf, code)
		}
		if !v.Verified {
			t.Errorf("leaf %d: verified = false (reason %q), want true", leaf, v.Reason)
		}
		if !v.Included {
			t.Errorf("leaf %d: included = false, want true", leaf)
		}
		if v.Reason != "" {
			t.Errorf("leaf %d: reason = %q, want empty", leaf, v.Reason)
		}
		if v.HubStatus != "verified" {
			t.Errorf("leaf %d: hub_status = %q, want verified", leaf, v.HubStatus)
		}
		if v.TreeSize != m.size {
			t.Errorf("leaf %d: tree_size = %d, want %d", leaf, v.TreeSize, m.size)
		}
		if v.LeafIndex != uint64(leaf) {
			t.Errorf("leaf %d: leaf_index = %d, want %d", leaf, v.LeafIndex, leaf)
		}
		if v.IsccID != leafISCCID(leaf) {
			t.Errorf("leaf %d: iscc_id = %q, want %q", leaf, v.IsccID, leafISCCID(leaf))
		}
		// The served root must base64-decode to the tree's real root — ground truth,
		// not a constant a green-but-wrong handler could fake.
		if v.Root != wantRoot {
			t.Errorf("leaf %d: root = %q, want %q", leaf, v.Root, wantRoot)
		}
	}
}

// TestVerifyUnknownISCCID asserts an iscc_id never indexed is a 200 verdict with
// verified:false and a non-empty reason — never a 5xx. This is the mutation guard
// for a handler that hard-codes verified:true.
func TestVerifyUnknownISCCID(t *testing.T) {
	m := buildVerifyMirror(t, mirrorLeaves, false)
	h := Handler(m.store, m.hubID, nil)
	code, v := getVerdict(t, h, "iscc_id=ISCC:NOSUCHLEAF")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if v.Verified {
		t.Error("verified = true, want false for an unknown id")
	}
	if v.Included {
		t.Error("included = true, want false for an unknown id")
	}
	if v.Reason == "" {
		t.Error("reason = empty, want a non-empty cause for an unknown id")
	}
}

// TestVerifyMissingISCCID asserts a request without iscc_id is a 200 verdict (not a
// 400 like /inclusion): verify-for-me always yields a verdict for id input.
func TestVerifyMissingISCCID(t *testing.T) {
	m := buildVerifyMirror(t, mirrorLeaves, false)
	h := Handler(m.store, m.hubID, nil)
	code, v := getVerdict(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if v.Verified {
		t.Error("verified = true, want false for a missing id")
	}
	if v.Reason == "" {
		t.Error("reason = empty, want a non-empty cause for a missing id")
	}
}

// TestVerifyInclusionIsNonVacuous proves the verdict's inclusion result is a REAL
// Merkle check, not a stub: over a byte-accurate mirror whose ONLY accepted
// checkpoint root is deliberately corrupted, the recomputed inclusion proof builds
// but does NOT verify against the stored root, so the verdict must be verified:false
// (a 200 verdict, not a 5xx). This is the mutation guard for a handler that drops
// proof.VerifyInclusion — such a handler would report verified:true here.
func TestVerifyInclusionIsNonVacuous(t *testing.T) {
	m := buildVerifyMirror(t, mirrorLeaves, true)
	h := Handler(m.store, m.hubID, nil)

	code, v := getVerdict(t, h, "iscc_id="+leafISCCID(5))
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if v.Verified {
		t.Error("verified = true against a corrupted accepted root; the handler dropped proof.VerifyInclusion")
	}
	if v.Included {
		t.Error("included = true against a corrupted accepted root; the inclusion check is vacuous")
	}
	if v.Reason == "" {
		t.Error("reason = empty, want a non-empty cause when the inclusion proof does not verify")
	}
	// The served (size, root) is still the accepted (corrupted) one and the leaf
	// resolved — only the Merkle verification fails, which is exactly the negative.
	if v.TreeSize != m.size {
		t.Errorf("tree_size = %d, want %d", v.TreeSize, m.size)
	}
	if v.LeafIndex != 5 {
		t.Errorf("leaf_index = %d, want 5", v.LeafIndex)
	}
}

// TestVerifyNonGET asserts a non-GET method is a 405 (the shared method gate).
func TestVerifyNonGET(t *testing.T) {
	m := buildVerifyMirror(t, 8, false)
	h := Handler(m.store, m.hubID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/verify?iscc_id="+leafISCCID(1), nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// TestVerifyNoAcceptedCheckpoint asserts that with no accepted checkpoint yet
// (LastSize == 0) the verdict is a 200 with verified:false and a non-empty reason —
// never a 5xx — even when the iscc_id is indexed.
func TestVerifyNoAcceptedCheckpoint(t *testing.T) {
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
	if err := st.RecordProjections(ctx, []store.ProjectionRecord{{HubID: hubID, Seq: 0, IsccID: leafISCCID(0)}}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}

	h := Handler(st, hubID, nil)
	code, v := getVerdict(t, h, "iscc_id="+leafISCCID(0))
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if v.Verified {
		t.Error("verified = true, want false with no accepted checkpoint")
	}
	if v.Reason == "" {
		t.Error("reason = empty, want a non-empty cause with no accepted checkpoint")
	}
}

// TestVerifyFrozenHubStatus asserts a frozen hub's verdict reports hub_status
// "frozen" (the glossary status the store can substantiate via FollowState.Frozen),
// while the inclusion result still verifies against the accepted root.
func TestVerifyFrozenHubStatus(t *testing.T) {
	m := buildVerifyMirror(t, mirrorLeaves, false)
	ctx := context.Background()
	if err := m.store.Freeze(ctx, m.hubID); err != nil {
		t.Fatalf("Freeze: %v", err)
	}

	h := Handler(m.store, m.hubID, nil)
	code, v := getVerdict(t, h, "iscc_id="+leafISCCID(5))
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if v.HubStatus != "frozen" {
		t.Errorf("hub_status = %q, want frozen", v.HubStatus)
	}
	if !v.Verified {
		t.Errorf("verified = false (reason %q), want true — freeze does not break the inclusion check", v.Reason)
	}
}

// TestVerifyLeafHashMatchesTree guards the fixture's load-bearing assumption: the
// entry-bundle leaf bytes the handler reads back hash (via the production hasher) to
// the tree's leaf hash, so the inclusion check the handler runs is against ground
// truth, not a coincidentally-passing constant.
func TestVerifyLeafHashMatchesTree(t *testing.T) {
	m := buildVerifyMirror(t, 8, false)
	want := m.tree.LeafHash(3)
	got := rfc6962.DefaultHasher.HashLeaf(verifyLeafData(3))
	if string(got) != string(want) {
		t.Fatal("leaf hash mismatch: the handler must hash record bytes the same way the tree does")
	}
}
