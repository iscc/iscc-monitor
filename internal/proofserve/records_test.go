// Golden HTTP-seam tests for the HTML record-list route (GET /records). They drive
// proofserve.Handler over the buildMirror fixture store and assert the rendered page
// lists the hub's indexed records newest-first, links each row to its per-record
// bytes view (entries?index=<seq>), paginates via plain no-JS ?from=…&n=… links, and
// renders an informative 200 empty state for a hub with no indexed records. The oracle
// gate is N/A here: pure HTML render of persisted iscc_index rows, no signature,
// RFC-6962, Merkle, did:web, fsck, or proof path. The body is also pinned CDN-free —
// the load-bearing M-UI invariant that every SSR body is complete with JavaScript
// disabled and references no third-party origin.
package proofserve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iscc/iscc-monitor/internal/store"
)

// getRecords drives the handler with the given /records query string and returns the
// status code and body.
func getRecords(t *testing.T, h http.Handler, query string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	url := "/records"
	if query != "" {
		url += "?" + query
	}
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec.Code, rec.Body.String()
}

// TestRecordsListsNewestFirst asserts GET /records returns 200 text/html, lists the
// hub's records newest-first (the highest seq appears before a lower one), links each
// row to its per-record bytes view (entries?index=<seq>), and shows the honest total.
func TestRecordsListsNewestFirst(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/records", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
	body := rec.Body.String()

	// The default page size is 50, so the newest 50 seqs (299..250) appear; the page
	// must be newest-first: the newest seq's row precedes the oldest-on-page seq's row.
	newestLink := "entries?index=299"
	olderOnPageLink := "entries?index=250"
	iNewest := strings.Index(body, newestLink)
	iOlder := strings.Index(body, olderOnPageLink)
	if iNewest < 0 {
		t.Fatalf("body missing newest record link %q", newestLink)
	}
	if iOlder < 0 {
		t.Fatalf("body missing on-page record link %q", olderOnPageLink)
	}
	if iNewest > iOlder {
		t.Errorf("records not newest-first: %q at %d should precede %q at %d", newestLink, iNewest, olderOnPageLink, iOlder)
	}
	// The honest "showing N of TOTAL" line reflects the full index size.
	if !strings.Contains(body, "of 300") {
		t.Errorf("body missing honest total (of 300)\n%s", body)
	}
}

// TestRecordsPagination asserts a small n shows a working ?from=…&n=… older link and
// that following it serves the next newest-first window without re-listing the first
// page's records.
func TestRecordsPagination(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, nil)

	// First page, n=2: the two newest (299, 298) and an older link with from=297&n=2.
	code, body := getRecords(t, h, "n=2")
	if code != http.StatusOK {
		t.Fatalf("first page status = %d, want 200", code)
	}
	for _, want := range []string{"entries?index=299", "entries?index=298"} {
		if !strings.Contains(body, want) {
			t.Errorf("first page missing %q\n%s", want, body)
		}
	}
	// The older link is a plain anchor with the seq cursor one below the page's
	// smallest seq (298-1=297) and the page size echoed (the &amp; is the escaped &).
	wantOlder := `href="records?from=297&amp;n=2"`
	if !strings.Contains(body, wantOlder) {
		t.Fatalf("first page missing older pagination link %q\n%s", wantOlder, body)
	}

	// Follow the older link: the next window is 297, 296 — and must NOT re-list 299.
	code, body = getRecords(t, h, "from=297&n=2")
	if code != http.StatusOK {
		t.Fatalf("second page status = %d, want 200", code)
	}
	for _, want := range []string{"entries?index=297", "entries?index=296"} {
		if !strings.Contains(body, want) {
			t.Errorf("second page missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "entries?index=299") {
		t.Errorf("second page wrongly re-lists the first page's newest record (299)\n%s", body)
	}
	// On a non-first page a newer link is live too.
	if !strings.Contains(body, `href="records?from=`) {
		t.Errorf("second page missing a newer pagination link\n%s", body)
	}
}

// TestRecordsLinksTokensNoCDN pins the Evidence-Ledger shell wiring on the record
// list: the rendered body links the embedded DS token + self-hosted-font stylesheets
// at their literal /_ds/ paths, is styled through the DS font tokens, lays out the
// list without an HTML <table>, and carries no external CDN URL (mirrors the browser
// page's invariant).
func TestRecordsLinksTokensNoCDN(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, nil)

	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	for _, want := range []string{
		`href="/_ds/tokens.css"`,
		`href="/_ds/fonts.css"`,
		"var(--font-sans)",
		"var(--font-mono)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing DS-shell marker %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "<table") {
		t.Errorf("body still contains a <table> element; the ledger redress is incomplete\n%s", body)
	}
	for _, banned := range []string{"jsdelivr", "http://", "https://", "cdn."} {
		if strings.Contains(body, banned) {
			t.Errorf("body contains external CDN reference %q\n%s", banned, body)
		}
	}
}

// TestRecordsRendersInMemoryStatus proves the in-memory status overlay reaches the
// record-list page: a store-verified hub whose live verdict is unresolvable renders
// that richer status through the hubStatusBadge partial, and the store-only "verified"
// is NOT the rendered status (the overlay won). The negative data-status="verified"
// assert is what the unquoted CSS selector in records.html keeps honest.
func TestRecordsRendersInMemoryStatus(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	statuses := fakeStatusSource{m.hubID: "unresolvable"}
	h := Handler(m.store, m.hubID, statuses)

	code, body := getRecords(t, h, "")
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

// TestRecordsNonGET asserts a non-GET method to /records is a 405 (the shared
// method-gate at the top of Handler covers it).
func TestRecordsNonGET(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/records", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

// TestRecordsEmpty asserts GET /records on a hub with an empty iscc_index returns 200
// with the informative empty-state copy, never a 404/5xx — coverage honesty (ADR-0001).
func TestRecordsEmpty(t *testing.T) {
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

	h := Handler(st, hubID, nil)
	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if !strings.Contains(body, "No records yet") {
		t.Errorf("body missing empty-state copy\n%s", body)
	}
}
