// Golden HTTP-seam tests for the HTML record-list route (GET /records). They drive
// proofserve.Handler over the buildMirror fixture store and assert the rendered page
// lists the hub's indexed records newest-first, links each row to its single-record
// page (record?index=<seq>), paginates via plain no-JS ?from=…&n=… links, and
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

	"github.com/iscc/iscc-monitor/internal/dashboard"
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
// row to its single-record page (record?index=<seq>), and shows the honest total.
func TestRecordsListsNewestFirst(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

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
	newestLink := "record?index=299"
	olderOnPageLink := "record?index=250"
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
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	// First page, n=2: the two newest (299, 298) and an older link with from=297&n=2.
	code, body := getRecords(t, h, "n=2")
	if code != http.StatusOK {
		t.Fatalf("first page status = %d, want 200", code)
	}
	for _, want := range []string{"record?index=299", "record?index=298"} {
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
	for _, want := range []string{"record?index=297", "record?index=296"} {
		if !strings.Contains(body, want) {
			t.Errorf("second page missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "record?index=299") {
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
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

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
	// monitor.iscc.codes is the one intentional external https origin (the verifier
	// app, same-federation, not a third-party asset CDN), so the ban targets known
	// third-party CDN hosts and bare http://, not every https:// substring — matching
	// the dossier / browser masthead no-CDN ban now that the chrome carries the link.
	for _, banned := range []string{"jsdelivr", "cdn.", "unpkg", "googleapis", "http://"} {
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
	h := Handler(m.store, m.hubID, "sb0.iscc.id", statuses, dashboard.Identity{})

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

// TestRecordsRendersLoggedColumn asserts the record list renders the mockup's Logged
// column: a row whose RecordRow.NoteTimestamp is set shows that verbatim RFC-3339 time
// (ADR-0008 — never re-formatted), a row with no timestamp shows the honest em-dash
// placeholder, and the list carries a Logged column header. The store is built directly
// (buildMirror seeds no timestamps) so the seeded literal time is the non-vacuous gate.
func TestRecordsRendersLoggedColumn(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "logged.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Seq 0 carries a real timestamp; seq 1 carries none (the common NULL-column case
	// the buildMirror fixture seeds). Both seqs are accepted (LastSize advanced to 2).
	const loggedAt = "2026-06-21T12:34:56Z"
	recs := []store.ProjectionRecord{
		{HubID: hubID, Seq: 0, IsccID: leafISCCID(0), NoteTimestamp: loggedAt},
		{HubID: hubID, Seq: 1, IsccID: leafISCCID(1)},
	}
	if err := st.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, 2); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	// The seeded timestamp renders verbatim (hardcoded literal, so the gate is real).
	if !strings.Contains(body, loggedAt) {
		t.Errorf("body missing verbatim logged time %q\n%s", loggedAt, body)
	}
	// The timestamp-less row renders the honest em-dash placeholder, never a fabricated
	// or zero time. The entity reference &mdash; is how the template emits it.
	if !strings.Contains(body, "&mdash;") {
		t.Errorf("body missing the empty-timestamp em-dash fallback\n%s", body)
	}
	// The named region heads its columns including Logged.
	if !strings.Contains(body, "<span>Logged</span>") {
		t.Errorf("body missing the Logged column header\n%s", body)
	}
}

// TestRecordsRendersTypeColumn asserts the record list renders the mockup's Type
// column: each row carries a per-kind badge (Declaration / Deletion / Unknown record
// type) mapped from the verbatim note.$schema (ADR-0008 — the only interpretation), and
// the list heads a Type column completing the 4-column Seq · Type · ISCC-ID · Logged
// head. The schema URIs are HARDCODED literals, NOT the package constants, so reverting
// schemaDeclaration / schemaDeletion makes this test FAIL (the constant-vs-constant
// vacuity trap the single-record test fell into). The verbatim schema still renders in
// the ISCC-ID cell — the Type badge is additive, never a replacement.
func TestRecordsRendersTypeColumn(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "type.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Seq 0 is a declaration, seq 1 a deletion, seq 2 an unknown schema. The schema URIs
	// are hardcoded literals (not schemaDeclaration / schemaDeletion) so a constant
	// regression breaks the kind mapping and FAILS this test. All three are accepted
	// (LastSize advanced to 3).
	const (
		declSchema    = "http://purl.org/iscc/schema/iscc-note-0.8.0.json"
		delSchema     = "http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"
		unknownSchema = "iscc-note-future-9.9.9"
	)
	recs := []store.ProjectionRecord{
		{HubID: hubID, Seq: 0, IsccID: leafISCCID(0), NoteSchema: declSchema},
		{HubID: hubID, Seq: 1, IsccID: leafISCCID(1), NoteSchema: delSchema},
		{HubID: hubID, Seq: 2, IsccID: leafISCCID(2), NoteSchema: unknownSchema},
	}
	if err := st.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, 3); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	// The named region heads its columns including Type, completing the 4-column head.
	if !strings.Contains(body, "<span>Type</span>") {
		t.Errorf("body missing the Type column header\n%s", body)
	}
	// Each kind's human label renders, mapped from the hardcoded schema URI.
	for _, want := range []string{"Declaration", "Deletion", "Unknown record type"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing Type label %q\n%s", want, body)
		}
	}
	// The Type badge is additive: the verbatim schema still renders alongside it.
	if !strings.Contains(body, declSchema) {
		t.Errorf("body no longer renders the verbatim note.$schema %q (Type badge wrongly replaced it)\n%s", declSchema, body)
	}
}

// countRecordLinks counts the per-record rows rendered in a /records body by counting
// the single-record-page links each row emits (one record?index= link per row).
func countRecordLinks(body string) int {
	return strings.Count(body, "record?index=")
}

// TestRecordsClampsHostilePageSize asserts a hostile n that wraps int(n) negative is
// clamped to maxPageSize BEFORE the int() conversion, so the page renders at most
// maxPageSize rows — never the whole index. n=9223372036854775808 (MaxInt64+1) wraps
// int(n) negative; modernc SQLite reads a negative LIMIT as unlimited, so without the
// pre-int() uint64 clamp the whole 300-leaf index would render.
func TestRecordsClampsHostilePageSize(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	code, body := getRecords(t, h, "n=9223372036854775808")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if got := countRecordLinks(body); got > maxPageSize {
		t.Errorf("rendered %d record rows for a hostile n, want at most %d (clamp bypassed)", got, maxPageSize)
	}
	// The whole index (300) must never render — the anti-DoS clamp is the point.
	if countRecordLinks(body) >= mirrorLeaves {
		t.Errorf("a hostile n rendered the whole index (%d rows); the clamp was bypassed", mirrorLeaves)
	}
}

// olderHref extracts the older-pagination link's query (the part after records?) from
// a rendered /records body, or returns "" when no older link is live. It reads the
// next href= after the "older &rarr;" anchor so the test follows the chain the page
// actually emits rather than a hand-built URL.
func olderHref(body string) string {
	const marker = `older &rarr;</a>`
	i := strings.Index(body, marker)
	if i < 0 {
		return "" // no live older link (the anchor is rendered as a <span>)
	}
	seg := body[:i]
	j := strings.LastIndex(seg, `href="records?`)
	if j < 0 {
		return ""
	}
	q := seg[j+len(`href="records?`):]
	end := strings.IndexByte(q, '"')
	return strings.ReplaceAll(q[:end], "&amp;", "&")
}

// TestRecordsOlderLinkReachesSeq0 walks the older-link chain emitted by the page (with
// n=1, so the chain emits a from=0 older link from the page ending at seq 1) all the
// way to the oldest record and asserts seq 0 is reached. Before the from=0 un-overload,
// the older link from a page ending at seq 1 emitted from=0, which ListRecords read as
// "start at newest", so following it jumped back to the newest record and seq 0 was
// unreachable via navigation (the reviewer-confirmed bug: from=1&n=1's older link
// from=0&n=1 showed the newest seq, not seq 0).
func TestRecordsOlderLinkReachesSeq0(t *testing.T) {
	// A small mirror (seqs 0..3, n=1) so the chain emits a literal from=0 older link.
	m := buildMirror(t, 4)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	query := "n=1"
	reached0 := false
	// Bound the walk so a chain that loops (the old overloaded behaviour jumps back to
	// the newest page forever) trips the cap instead of hanging.
	for step := 0; step < 16; step++ {
		code, body := getRecords(t, h, query)
		if code != http.StatusOK {
			t.Fatalf("step %d (%q) status = %d, want 200", step, query, code)
		}
		if strings.Contains(body, "record?index=0") {
			reached0 = true
			// The page containing seq 0 is the bottom of the chain — no live older link.
			if next := olderHref(body); next != "" {
				t.Errorf("page containing seq 0 still offers an older link %q (broken chain)\n%s", next, body)
			}
			break
		}
		next := olderHref(body)
		if next == "" {
			t.Fatalf("step %d (%q): older chain ended before reaching seq 0\n%s", step, query, body)
		}
		query = next
	}
	if !reached0 {
		t.Fatalf("older-link chain never reached seq 0 within the step cap (from=0 still overloaded)")
	}
}

// TestRecordsCeilingHidesUnacceptedLeaves asserts the list caps at the accepted tree
// size: a hub whose iscc_index holds projections at seq >= LastSize (a freeze/fault
// leaves them, since ingest writes projections before AdvanceAccepted) lists ONLY the
// accepted leaves, never the unaccepted ones whose record?index= links would then 404.
func TestRecordsCeilingHidesUnacceptedLeaves(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "ceiling.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hubID, err := st.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	// Index six leaves (seq 0..5) but advance the accepted tree to only size 4, so
	// seqs 4 and 5 are above the accepted ceiling — the frozen-hub-past-violation case.
	recs := make([]store.ProjectionRecord, 6)
	for i := range recs {
		recs[i] = store.ProjectionRecord{HubID: hubID, Seq: uint64(i), IsccID: leafISCCID(i)}
	}
	if err := st.RecordProjections(ctx, recs); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hubID, 4); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	// Accepted leaves (seq < 4) list.
	for _, want := range []string{"record?index=3", "record?index=0"} {
		if !strings.Contains(body, want) {
			t.Errorf("accepted leaf link %q missing\n%s", want, body)
		}
	}
	// Unaccepted leaves (seq >= 4) must NOT list — they are not vouched for and their
	// record?index= links would 404.
	for _, banned := range []string{"record?index=4", "record?index=5"} {
		if strings.Contains(body, banned) {
			t.Errorf("unaccepted leaf link %q rendered past the LastSize ceiling\n%s", banned, body)
		}
	}
	// The honest total reflects only the accepted leaves.
	if !strings.Contains(body, "of 4") {
		t.Errorf("body total does not reflect the accepted-only count (of 4)\n%s", body)
	}
}

// TestParseUintOverflow pins parseUint's contract, including the overflow guard: a
// value that would wrap uint64 is rejected with the same error type as a non-numeric
// input (so an over-large index never silently aliases a small one), while empty /
// non-numeric still error and a valid value parses. MaxUint64 itself must parse;
// MaxUint64+1 (the 20-digit "18446744073709551616") must be rejected.
func TestParseUintOverflow(t *testing.T) {
	cases := []struct {
		in      string
		want    uint64
		wantErr bool
	}{
		{"", 0, true},
		{"12a", 0, true},
		{"0", 0, false},
		{"42", 42, false},
		{"18446744073709551615", 18446744073709551615, false}, // MaxUint64 fits
		{"18446744073709551616", 0, true},                     // MaxUint64+1 overflows
		{"99999999999999999999999999", 0, true},               // far past MaxUint64
	}
	for _, c := range cases {
		got, err := parseUint(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("parseUint(%q) err = %v, wantErr = %v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && got != c.want {
			t.Errorf("parseUint(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestRecordsNonGET asserts a non-GET method to /records is a 405 (the shared
// method-gate at the top of Handler covers it).
func TestRecordsNonGET(t *testing.T) {
	m := buildMirror(t, 8)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
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

	h := Handler(st, hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if !strings.Contains(body, "No records yet") {
		t.Errorf("body missing empty-state copy\n%s", body)
	}
}

// TestRecordsHeadAndBreadcrumb asserts the record list carries the Log-Browser mockup's
// named head regions: the "← <domain> dossier" breadcrumb linking to the SITE-ROOT
// dossier mount (an absolute /<domain> path, OUTSIDE this /log/ subtree), the eyebrow
// "Log browser", the head domain, and the "<domain> · N records mirrored" sub-line whose
// count is the honest accepted-tree total. It is mutation-proven: dropping the breadcrumb
// OR the "Log browser" eyebrow from records.html FAILS it. The domain is the threaded
// "sb0.iscc.id" (matching the buildMirror fixture's hub).
func TestRecordsHeadAndBreadcrumb(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})

	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	for _, want := range []string{
		`href="/sb0.iscc.id"`,          // the breadcrumb target: the absolute site-root dossier mount
		"← sb0.iscc.id dossier",        // the breadcrumb copy
		"Log browser",                  // the head eyebrow
		`class="log-name">sb0.iscc.id`, // the head name = the hub domain (constraint-win)
		"records mirrored",             // the "<domain> · N records mirrored" sub-line copy
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing head/breadcrumb region %q\n%s", want, body)
		}
	}
	// The "N records mirrored" count is the honest accepted-tree total (mirrorLeaves).
	if !strings.Contains(body, "sb0.iscc.id · 300 records mirrored") {
		t.Errorf("body missing the honest records-mirrored sub-line (300)\n%s", body)
	}
	// The breadcrumb must be an ABSOLUTE site-root path, never a relative ../ walk out of
	// the /log/ subtree (the record rows are relative, but the dossier is at the site root).
	if strings.Contains(body, `href="../`) {
		t.Errorf("breadcrumb wrongly uses a relative ../ walk instead of the absolute site-root path\n%s", body)
	}
}

// TestRecordsChromeInstanceIdentity asserts the record list carries the shared chrome's
// instance-identity masthead + the static "verify ↗ monitor.iscc.codes" tier-2 link, the
// SAME dashboard.Identity value the dashboard and dossier mastheads receive: a populated
// Identity surfaces its exact Instance / Operator strings, while the verify link is the
// fixed monitor.iscc.codes target. It is non-vacuous — threading a constant default
// instead of the supplied value, or dropping the {{.Instance}} / {{.Operator}} bindings,
// fails the populated-identity assertions because the exact operator strings would no
// longer appear. The no-CDN body invariant still holds (the verify host is the only
// external https origin, asserted CDN-free by TestRecordsLinksTokensNoCDN).
func TestRecordsChromeInstanceIdentity(t *testing.T) {
	m := buildMirror(t, mirrorLeaves)
	idv := dashboard.Identity{
		Instance: "monitor.example.test",
		Operator: "operated by Example Org · example net",
		Realm:    "example net",
	}
	h := Handler(m.store, m.hubID, "sb0.iscc.id", nil, idv)

	code, body := getRecords(t, h, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	for _, want := range []string{
		`class="chrome-identity"`,               // the instance-identity block
		"monitor.example.test",                  // the operator-supplied instance line
		"operated by Example Org · example net", // the operator-supplied operator line
		"monitor.iscc.codes",                    // the tier-2 verify link copy
		`href="https://monitor.iscc.codes/"`,    // the tier-2 verify link target
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing chrome instance-identity / verify marker %q\n%s", want, body)
		}
	}
	// A zero-value Identity falls back to the static placeholder copy (the neutral
	// "monitor instance" line), proving resolveIdentity is wired, not bypassed.
	h0 := Handler(m.store, m.hubID, "sb0.iscc.id", nil, dashboard.Identity{})
	_, body0 := getRecords(t, h0, "")
	if !strings.Contains(body0, "monitor instance") {
		t.Errorf("zero-value Identity did not fall back to the static placeholder copy\n%s", body0)
	}
}
