// Conformance test for the mirrored OpenTimestamps proof HTTP surface
// (GET /checkpoint.ots): it builds a byte-accurate tlog-tiles mirror with an
// accepted (size, root) (reusing handler_test.go's buildMirror), stores a real
// OpenTimestamps fixture proof against that root via store.RecordOTS, and drives
// proofserve.Handler's /checkpoint.ots route over the real store, asserting the
// served body is byte-equal to the stored OTSBytes, is application/octet-stream,
// carries a strong ETag with conditional-GET 304 support, and parses as a valid
// .ots with the standard opentimestamps client (the offline form of "verifies with
// the standard ots client"). It also pins the honest 404 cases: no accepted
// checkpoint, an accepted root with no ots row, and a row carrying the
// empty-OTSBytes sentinel.
//
// The opentimestamps import is test-only — production proofserve serves OTSBytes as
// opaque bytes and never parses them, so it stays off the non-WASM internal/ots /
// opentimestamps closure (asserted by the next.md dep-closure check). Oracle gate is
// N/A: this is an opaque-byte serve of an already-stored proof, no signature /
// RFC-6962 / Merkle / did:web / proof code is touched.
package proofserve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	opentimestamps "github.com/nbd-wtf/opentimestamps"

	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/store"
)

// otsFixture reads the real OpenTimestamps proof fixture internal/ots/testdata copies
// verbatim — note the .txt.ots suffix — so the served bytes are a genuine .ots, not a
// synthetic blob a green-but-wrong handler could fake.
func otsFixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "ots", "testdata", "hello-world.txt.ots"))
	if err != nil {
		t.Fatalf("read ots fixture: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("ots fixture is empty")
	}
	return b
}

// recordOTSForAccepted stores an ots row for the mirror's accepted (size, root) with
// the given proof bytes, so OTSForRoot(hubID, size, root) resolves it. It mirrors the
// keying the production stamp loop uses (hub_id, tree_size, root).
func recordOTSForAccepted(t *testing.T, m mirrorTree, otsBytes []byte) {
	t.Helper()
	ctx := context.Background()
	root, _, found, err := m.store.CheckpointAt(ctx, m.hubID, m.size)
	if err != nil || !found {
		t.Fatalf("CheckpointAt(%d): found=%v err=%v", m.size, found, err)
	}
	if _, _, err := m.store.RecordOTS(ctx, store.OTSRecord{
		HubID:     m.hubID,
		TreeSize:  m.size,
		Root:      root,
		Status:    store.OTSStatusPending,
		OTSBytes:  otsBytes,
		StampedAt: time.Unix(1700000000, 0),
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}
}

// getOTS drives the /checkpoint.ots route and returns the recorder so callers can
// assert status, headers, and body bytes.
func getOTS(t *testing.T, h http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/checkpoint.ots", nil))
	return rec
}

// TestOTSServesStoredProofVerbatim proves the route serves the stored OpenTimestamps
// proof byte-for-byte for an anchored accepted root, as application/octet-stream, and
// that the served bytes parse as a valid .ots with the standard opentimestamps client
// (the offline "verifies with the standard ots client" half).
func TestOTSServesStoredProofVerbatim(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	want := otsFixture(t)
	recordOTSForAccepted(t, m, want)

	rec := getOTS(t, Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != octetStreamType {
		t.Errorf("Content-Type = %q, want %q", ct, octetStreamType)
	}
	if got := rec.Body.Bytes(); string(got) != string(want) {
		t.Errorf("served %d bytes, want byte-equal to the stored %d-byte proof", len(got), len(want))
	}
	// The served bytes must parse as a real .ots — proving they are a genuine
	// OpenTimestamps proof, not opaque garbage (the offline form of the ots-client
	// verify). This is the test-only opentimestamps parse; production never parses.
	file, err := opentimestamps.ReadFromFile(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("served body does not parse as .ots: %v", err)
	}
	if file == nil {
		t.Fatal("opentimestamps.ReadFromFile returned a nil *File for the served body")
	}
}

// TestOTSConditionalGET asserts the strong content ETag + If-None-Match → 304 pattern
// (mirroring tilesserve.writeBlob): a first GET returns 200 with a quoted-hex ETag,
// and a follow-up GET echoing that ETag (and the wildcard "*") short-circuits to 304
// with an empty body.
func TestOTSConditionalGET(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	recordOTSForAccepted(t, m, otsFixture(t))
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	first := getOTS(t, h)
	if first.Code != http.StatusOK {
		t.Fatalf("first GET status = %d, want 200", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first GET has no ETag")
	}

	for _, inm := range []string{etag, "*"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/checkpoint.ots", nil)
		req.Header.Set("If-None-Match", inm)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotModified {
			t.Errorf("If-None-Match %q: status = %d, want 304", inm, rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("If-None-Match %q: 304 body = %d bytes, want empty", inm, rec.Body.Len())
		}
	}
}

// TestOTSNoAcceptedCheckpoint asserts a hub with no accepted checkpoint yet
// (LastSize == 0) is a 404 — there is no root to anchor — never a 5xx.
func TestOTSNoAcceptedCheckpoint(t *testing.T) {
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

	rec := getOTS(t, Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{}))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// TestOTSRootNotAnchored asserts an accepted root with no ots row is a 404 "root not
// yet anchored" (the honest pending state, OTSForRoot's plain miss) — never a 5xx.
func TestOTSRootNotAnchored(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	rec := getOTS(t, Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{}))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for an un-anchored root", rec.Code)
	}
}

// TestOTSEmptySentinelNotAnchored is the load-bearing edge case: a row exists for the
// accepted root but carries the empty-OTSBytes sentinel (stamped but not yet
// calendar-submitted), so there is no servable proof — the route must 404 "root not
// yet anchored" rather than serve zero bytes as an unparseable .ots.
func TestOTSEmptySentinelNotAnchored(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	recordOTSForAccepted(t, m, nil) // empty-OTSBytes sentinel row

	rec := getOTS(t, Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{}))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for the empty-OTSBytes sentinel row", rec.Code)
	}
}

// TestOTSNonGET asserts a non-GET method is a 405 (the shared method gate).
func TestOTSNonGET(t *testing.T) {
	m := buildMirror(t, 8)
	rec := httptest.NewRecorder()
	Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/checkpoint.ots", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
