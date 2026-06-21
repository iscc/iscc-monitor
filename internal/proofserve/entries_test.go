// Conformance test for the computed single-leaf record-bytes HTTP surface
// (/entries?index=<seq>). It mirrors entry bundles of known framed records into a
// fresh store, records + advances the accepted checkpoint, then drives
// proofserve.Handler over the real SQLiteFetcher and asserts /entries returns the
// exact record bytes for an in-tree leaf (across the 256-leaf bundle boundary) and
// the documented 400/404/405 statuses for the edge cases.
//
// Oracle gate is correctly N/A for this slice: /entries is a pure decode-and-index
// — it returns the leaf's bytes verbatim, never re-hashing or verifying (the proof
// crypto lives in serveInclusion / serveConsistency). The test asserts byte-equality
// against the records the bundle was framed from (the independent third path,
// distinct from the api.EntryBundle.UnmarshalText decode under test), so a
// green-but-wrong extractor that returned the wrong leaf would fail.
package proofserve

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// recordBytes returns the synthetic raw record bytes the leaf at absolute seq i
// carries — a distinct per-leaf string so a wrong-leaf extraction is caught.
func recordBytes(i int) []byte {
	return []byte(fmt.Sprintf("record-bytes-for-leaf-%08d", i))
}

// frameBundle frames raw records into a tlog-tiles entry bundle, mirroring the C2SP
// encoding api.EntryBundle.UnmarshalText decodes: each record is prefixed with its
// big-endian uint16 length, then concatenated. This is the test's independent encode
// path — never imported from the code under test.
func frameBundle(records [][]byte) []byte {
	var out []byte
	for _, rec := range records {
		var prefix [2]byte
		binary.BigEndian.PutUint16(prefix[:], uint16(len(rec)))
		out = append(out, prefix[:]...)
		out = append(out, rec...)
	}
	return out
}

// entriesMirror holds the store/hub a set of framed entry bundles was mirrored into,
// along with the accepted tree size, so a test can drive /entries over the real
// SQLiteFetcher.
type entriesMirror struct {
	store *store.Store
	hubID int64
	size  uint64
}

// buildEntriesMirror creates a fresh store, frames the leaves records into full
// 256-leaf entry bundles plus a final partial, mirrors each bundle BLOB, and records
// + advances the accepted checkpoint to size — the state a verified poll leaves
// behind, which is all /entries reads. It does NOT mirror hash tiles (the
// record-bytes route never touches them).
func buildEntriesMirror(t *testing.T, leaves int) entriesMirror {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(filepath.Join(t.TempDir(), "entries.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	at := time.Unix(1700000000, 0)
	for first := 0; first < leaves; first += tiles.TileWidth {
		last := first + tiles.TileWidth
		if last > leaves {
			last = leaves
		}
		recs := make([][]byte, 0, last-first)
		for i := first; i < last; i++ {
			recs = append(recs, recordBytes(i))
		}
		width := last - first
		bundleIndex := uint64(first / tiles.TileWidth)
		if err := st.RecordEntryBundle(ctx, hubID, bundleIndex, width, frameBundle(recs), at); err != nil {
			t.Fatalf("RecordEntryBundle (index %d width %d): %v", bundleIndex, width, err)
		}
	}

	size := uint64(leaves)
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hubID, TreeSize: size, Root: []byte("root"), Raw: []byte("checkpoint"), ObservedAt: at,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, size); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	return entriesMirror{store: st, hubID: hubID, size: size}
}

// TestServeEntries asserts /entries?index=<seq> returns the exact record bytes for
// in-tree leaves on both sides of the 256-leaf bundle boundary.
func TestServeEntries(t *testing.T) {
	m := buildEntriesMirror(t, 300)
	h := Handler(m.store, m.hubID)

	for _, seq := range []int{0, 5, 255, 256, 260, 299} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/entries?index=%d", seq), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("seq %d: status = %d, want 200 (body %q)", seq, rec.Code, rec.Body.String())
		}
		if got, want := rec.Body.Bytes(), recordBytes(seq); string(got) != string(want) {
			t.Errorf("seq %d: body = %q, want %q", seq, got, want)
		}
		if ct := rec.Header().Get("Content-Type"); ct != octetStreamType {
			t.Errorf("seq %d: Content-Type = %q, want %q", seq, ct, octetStreamType)
		}
	}
}

// TestServeEntriesMissingIndex asserts a request without index is a 400.
func TestServeEntriesMissingIndex(t *testing.T) {
	m := buildEntriesMirror(t, 8)
	h := Handler(m.store, m.hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/entries", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestServeEntriesNonNumericIndex asserts a non-numeric index is a 400.
func TestServeEntriesNonNumericIndex(t *testing.T) {
	m := buildEntriesMirror(t, 8)
	h := Handler(m.store, m.hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/entries?index=abc", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestServeEntriesBeyondAcceptedTree asserts a seq at or past LastSize is a 404 —
// the leaf is not in the monitor's accepted tree.
func TestServeEntriesBeyondAcceptedTree(t *testing.T) {
	m := buildEntriesMirror(t, 8)
	h := Handler(m.store, m.hubID)
	for _, seq := range []int{8, 9, 100} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/entries?index=%d", seq), nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("seq %d: status = %d, want 404", seq, rec.Code)
		}
	}
}

// TestServeEntriesNoAcceptedCheckpoint asserts that with no accepted checkpoint yet
// (LastSize == 0) any index is a 404 — there are no accepted leaves to serve.
func TestServeEntriesNoAcceptedCheckpoint(t *testing.T) {
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

	h := Handler(st, hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/entries?index=0", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// TestServeEntriesBundleNotMirrored asserts that when the leaf is within the accepted
// tree but the entry bundle BLOB was never mirrored, the route is a 404 (the
// SQLiteFetcher os.ErrNotExist contract), not a 500.
func TestServeEntriesBundleNotMirrored(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "nobundle.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Accept a tree of size 8 but never mirror any entry bundle BLOB: the leaf is
	// in-tree yet unbacked.
	at := time.Unix(1700000000, 0)
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hubID, TreeSize: 8, Root: []byte("root"), Raw: []byte("checkpoint"), ObservedAt: at,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, 8); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	h := Handler(st, hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/entries?index=3", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestServeEntriesNonGET asserts a non-GET method is a 405.
func TestServeEntriesNonGET(t *testing.T) {
	m := buildEntriesMirror(t, 8)
	h := Handler(m.store, m.hubID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/entries?index=1", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
