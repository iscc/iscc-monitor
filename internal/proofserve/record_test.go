// Golden HTTP-seam tests for the single-record page (GET /record?index=<seq>). They
// drive proofserve.Handler over a fixture store that mirrors entry bundles (the raw
// record bytes, the source of truth) and seeds iscc_index projections (the id / schema
// labels), then assert the rendered page shows the leaf's seq, the verbatim ISCC-ID and
// note.$schema, the raw record bytes, and the correct kind label for the declaration /
// deletion / unknown schemas — including the load-bearing no-error clause that an
// unknown or empty schema still renders a 200. The status mapping (400 / 404 / 405) is
// pinned alongside, plus the DS-shell + no-CDN invariant the M-UI surfaces share.
//
// The oracle gate is N/A here: the page is a pure decode-and-index render of persisted
// projection rows and mirrored bundle bytes — no signature, RFC-6962, Merkle, did:web,
// fsck, or proof path (the proof-bundle assembler that re-engages it is the next slice).
package proofserve

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
)

// recordPageMirror holds the store/hub the single-record fixture was mirrored into,
// plus the accepted tree size, so a test can drive /record over the real SQLiteFetcher.
type recordPageMirror struct {
	store *store.Store
	hubID int64
	size  uint64
}

// schemaForSeq assigns each leaf a note.$schema so the kind-label mapping is exercised:
// seq 0 is a declaration, seq 1 a deletion, seq 2 a future/unknown schema, seq 3 an
// empty schema (still Unknown, the no-error clause), and the rest declarations.
func schemaForSeq(seq int) string {
	switch seq {
	case 0:
		return schemaDeclaration
	case 1:
		return schemaDeletion
	case 2:
		return "iscc-note-future-9.9.9"
	case 3:
		return "" // empty schema → Unknown, must still render
	default:
		return schemaDeclaration
	}
}

// buildRecordPageMirror creates a fresh store, frames the leaves records into full
// 256-leaf entry bundles plus a final partial (the raw bytes the page renders), seeds
// one iscc_index projection per leaf (the id / schema the page labels), and records +
// advances the accepted checkpoint to size — the state a verified poll leaves behind,
// which is all /record reads. It mirrors no hash tiles (the page never touches them).
func buildRecordPageMirror(t *testing.T, leaves int) recordPageMirror {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(filepath.Join(t.TempDir(), "record.db"))
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
		p := uint8(width)
		if width == tiles.TileWidth {
			p = 0
		}
		bundleIndex := uint64(first / tiles.TileWidth)
		if err := st.RecordEntryBundle(ctx, hubID, bundleIndex, p, frameBundle(recs), at); err != nil {
			t.Fatalf("RecordEntryBundle (index %d width %d): %v", bundleIndex, width, err)
		}
	}

	// One projection per leaf so RecordAt resolves each leaf's id / schema label.
	projs := make([]store.ProjectionRecord, leaves)
	for i := range projs {
		projs[i] = store.ProjectionRecord{
			HubID:      hubID,
			Seq:        uint64(i),
			IsccID:     leafISCCID(i),
			NoteSchema: schemaForSeq(i),
		}
	}
	if err := st.RecordProjections(ctx, projs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
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

	return recordPageMirror{store: st, hubID: hubID, size: size}
}

// getRecord drives the handler with the given /record query string and returns the
// status code and body.
func getRecord(t *testing.T, h http.Handler, query string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	url := "/record"
	if query != "" {
		url += "?" + query
	}
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec.Code, rec.Body.String()
}

// TestRecordServesInTreeLeaf asserts /record?index=<seq> returns 200 text/html for an
// in-tree leaf across the 256-leaf bundle boundary and shows the seq, the verbatim
// ISCC-ID, the verbatim note.$schema, and the raw record bytes.
func TestRecordServesInTreeLeaf(t *testing.T) {
	m := buildRecordPageMirror(t, 300)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	for _, seq := range []int{0, 5, 255, 256, 299} {
		code, body := getRecord(t, h, fmt.Sprintf("index=%d", seq))
		if code != http.StatusOK {
			t.Fatalf("seq %d: status = %d, want 200 (body %q)", seq, code, body)
		}
		// The leaf's seq value is shown.
		if !strings.Contains(body, fmt.Sprintf(">%d<", seq)) {
			t.Errorf("seq %d: body missing the rendered Seq value\n%s", seq, body)
		}
		// The verbatim ISCC-ID is shown.
		if !strings.Contains(body, leafISCCID(seq)) {
			t.Errorf("seq %d: body missing verbatim iscc_id %q\n%s", seq, leafISCCID(seq), body)
		}
		// The verbatim note.$schema is shown (except seq 3's empty schema, tested below).
		if want := schemaForSeq(seq); want != "" && !strings.Contains(body, want) {
			t.Errorf("seq %d: body missing verbatim note.$schema %q\n%s", seq, want, body)
		}
		// The raw record bytes are shown for human inspection.
		if want := string(recordBytes(seq)); !strings.Contains(body, want) {
			t.Errorf("seq %d: body missing raw record bytes %q\n%s", seq, want, body)
		}
	}
}

// TestRecordContentType asserts the single-record page is served as HTML.
func TestRecordContentType(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/record?index=0", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
}

// TestRecordKindLabels asserts the only interpretation the page performs — mapping the
// verbatim note.$schema to a kind label — is correct AND fail-open: the declaration and
// deletion schemas show their friendly labels, while an unknown schema AND an empty
// schema both render a 200 with the "Unknown record type" label (the explicit no-error
// clause). The verbatim schema string still appears alongside the label.
func TestRecordKindLabels(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	cases := []struct {
		seq       int
		wantKind  string
		wantNoErr bool // every case must be a 200; this documents the no-error clause
	}{
		{0, kindDeclaration, true},
		{1, kindDeletion, true},
		{2, kindUnknown, true}, // unknown schema renders, never a 4xx/5xx
		{3, kindUnknown, true}, // empty schema renders, never a 4xx/5xx
	}
	for _, c := range cases {
		code, body := getRecord(t, h, fmt.Sprintf("index=%d", c.seq))
		if code != http.StatusOK {
			t.Errorf("seq %d: status = %d, want 200 (the no-error clause)", c.seq, code)
			continue
		}
		if !strings.Contains(body, c.wantKind) {
			t.Errorf("seq %d: body missing kind label %q\n%s", c.seq, c.wantKind, body)
		}
	}

	// The deletion copy must note it is a NEW record that marks the declaration redacted
	// in derived views but never removes the committed declaration (glossary "Deletion").
	_, delBody := getRecord(t, h, "index=1")
	if !strings.Contains(delBody, "never removes the committed declaration") {
		t.Errorf("deletion page missing the new-record-preserves-original copy\n%s", delBody)
	}
}

// TestRecordRendersWithoutProjection asserts a leaf whose bytes ARE mirrored but whose
// iscc_index projection is absent still renders a 200 from the bytes (the bytes are the
// source of truth; the projection is a derived view, ADR-0008), showing a "no projection
// indexed" state rather than a 404. An absent projection maps note.$schema to Unknown.
func TestRecordRendersWithoutProjection(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "noproj.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Mirror a single 4-leaf entry bundle but seed NO projections, then accept size 4.
	at := time.Unix(1700000000, 0)
	recs := [][]byte{recordBytes(0), recordBytes(1), recordBytes(2), recordBytes(3)}
	if err := st.RecordEntryBundle(ctx, hubID, 0, 4, frameBundle(recs), at); err != nil {
		t.Fatalf("RecordEntryBundle: %v", err)
	}
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hubID, TreeSize: 4, Root: []byte("root"), Raw: []byte("checkpoint"), ObservedAt: at,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, 4); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, body := getRecord(t, h, "index=2")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (missing projection must NOT 404)\n%s", code, body)
	}
	// The bytes are still rendered — they are the source of truth.
	if want := string(recordBytes(2)); !strings.Contains(body, want) {
		t.Errorf("body missing raw record bytes %q\n%s", want, body)
	}
	// The id / schema cells show the no-projection state, not a fabricated id.
	if !strings.Contains(body, "no projection indexed") {
		t.Errorf("body missing the no-projection-indexed state\n%s", body)
	}
}

// TestRecordMissingIndex asserts a request without index is a 400.
func TestRecordMissingIndex(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getRecord(t, h, "")
	if code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

// TestRecordNonNumericIndex asserts a non-numeric index is a 400.
func TestRecordNonNumericIndex(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getRecord(t, h, "index=abc")
	if code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

// TestRecordBeyondAcceptedTree asserts a seq at or past LastSize is a 404 — the leaf is
// not in the monitor's accepted tree (coverage honesty, ADR-0001).
func TestRecordBeyondAcceptedTree(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	for _, seq := range []int{8, 9, 100} {
		code, _ := getRecord(t, h, fmt.Sprintf("index=%d", seq))
		if code != http.StatusNotFound {
			t.Errorf("seq %d: status = %d, want 404", seq, code)
		}
	}
}

// TestRecordNoAcceptedCheckpoint asserts that with no accepted checkpoint yet
// (LastSize == 0) any index is a 404 — there are no accepted leaves to render.
func TestRecordNoAcceptedCheckpoint(t *testing.T) {
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

	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, _ := getRecord(t, h, "index=0")
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

// TestRecordBundleNotMirrored asserts that when the leaf is within the accepted tree but
// the entry bundle BLOB was never mirrored, the route is a 404 (the SQLiteFetcher
// os.ErrNotExist contract), not a 500.
func TestRecordBundleNotMirrored(t *testing.T) {
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

	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, body := getRecord(t, h, "index=3")
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (body %q)", code, body)
	}
}

// TestRecordNonGET asserts a non-GET method to /record is a 405 (the shared method-gate
// at the top of Handler covers it).
func TestRecordNonGET(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/record?index=0", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// TestRecordLinksTokensNoCDN pins the Evidence-Ledger shell wiring on the single-record
// page: the rendered body links the embedded DS token + self-hosted-font stylesheets at
// their literal /_ds/ paths, is styled through the DS font tokens, lays out without an
// HTML <table>, and carries no external CDN URL (mirrors the records / browser pages).
//
// index=0 is a declaration, so its verbatim note.$schema —
// http://purl.org/iscc/schema/iscc-note-0.8.0.json — renders as legitimate record data
// in the <body>. The no-CDN scheme ban therefore runs ONLY over the document head (up to
// </style>), the region where a CDN <link>/url( would actually appear; banning http:// over
// the whole body would false-fail on that verbatim schema URI. The DS-shell + token markers
// live in the head too, so they assert over the same region; the <table> absence holds for
// the whole body.
func TestRecordLinksTokensNoCDN(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	code, body := getRecord(t, h, "index=0")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	// Scope the CDN ban + DS-shell asserts to the template/CDN region: the document
	// head, ending at </style>. A CDN <link> or url() reference can only appear here;
	// the <body> below carries verbatim record data (including the full schema URI).
	head := body
	if i := strings.Index(body, "</style>"); i >= 0 {
		head = body[:i]
	}

	for _, want := range []string{
		`href="/_ds/tokens.css"`,
		`href="/_ds/fonts.css"`,
		"var(--font-sans)",
		"var(--font-mono)",
	} {
		if !strings.Contains(head, want) {
			t.Errorf("head missing DS-shell marker %q\n%s", want, head)
		}
	}
	if strings.Contains(body, "<table") {
		t.Errorf("body contains a <table> element; the ledger redress is incomplete\n%s", body)
	}
	for _, banned := range []string{"jsdelivr", "http://", "https://", "cdn."} {
		if strings.Contains(head, banned) {
			t.Errorf("head contains external CDN reference %q\n%s", banned, head)
		}
	}
}

// TestRecordRendersInMemoryStatus proves the in-memory status overlay reaches the
// single-record page: a store-verified hub whose live verdict is unresolvable renders
// that richer status through the hubStatusBadge partial, and the store-only "verified"
// is NOT the rendered status (the overlay won). The negative data-status="verified"
// assert is what the unquoted CSS selector in record.html keeps honest.
func TestRecordRendersInMemoryStatus(t *testing.T) {
	m := buildRecordPageMirror(t, 8)
	statuses := fakeStatusSource{m.hubID: "unresolvable"}
	h := Handler(m.store, m.hubID, "sb0.iscc.id", statuses, dashboard.Identity{})

	code, body := getRecord(t, h, "index=0")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	for _, want := range []string{
		`class="hub-status-badge"`,
		`data-status="unresolvable"`,
		">Unresolvable<",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing overlaid status markup %q\n%s", want, body)
		}
	}
	if strings.Contains(body, `data-status="verified"`) {
		t.Errorf("body still renders store-only verified status; overlay did not apply\n%s", body)
	}
}
