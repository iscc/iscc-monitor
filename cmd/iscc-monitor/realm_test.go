// Tests for the realm-source loading + hourly-refresh wiring: loadRealm parses the
// authoritative iscc-hub Hub-List (real embedded hub_ids) from a URL or a file,
// falls back to the legacy domains-only document, and refreshRealmOnce swaps the
// live resolver while preserving the last good snapshot on a failed fetch.
//
// The headline regression reproduces the live monitor.iscc.io bug end-to-end:
// decoding the real ISCC-ID MEIGKUAHT3CSYAAC yields hub_id 2, and the mainnet
// Hub-List must resolve it to amlet.id — the document-order mapping the binary used
// before resolved the wrong (or no) hub, rendering "not found in this realm".
package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/iscc/iscc-monitor/internal/index"
	"github.com/iscc/iscc-monitor/internal/registry"
)

// slotPtr makes a *uint16 for a registry.Hub hub_id slot literal in tests.
func slotPtr(v uint16) *uint16 { return &v }

// mainnetHubListURL is the canonical authoritative source the production deployment
// points ISCC_MONITOR_REALM at; used here as the test source so the fixture is
// self-documenting about the real wiring.
const mainnetHubListURL = "https://raw.githubusercontent.com/iscc/iscc-hub/refs/heads/main/hubs/mainnet.yaml"

// mainnetHubListYAML mirrors iscc-hub/hubs/mainnet.yaml: hub_id 1 -> iscc.id,
// hub_id 2 -> amlet.id, with the deprecated-and-ignored pubkey field present.
const mainnetHubListYAML = `version: 1
network: mainnet
hubs:
    - hub_id: 1
      pubkey: z6MkeVVrr1dmNL4VsAWuP4tK1QAbi9AeAghVfkiHqGYdmsFX
      url: https://iscc.id
      active: true
    - hub_id: 2
      pubkey: z6MkpautqPyziUVAZBNfCsmdrqwCN6V5v8nGqKB3uyV3UvnY
      url: https://amlet.id
      active: true
`

// quietLogger returns a logger that discards output so test logs stay clean.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// staticGetter returns a realmGetter that serves fixed bytes for the expected URL
// and errs on any other URL, so a test asserts the exact source was fetched.
func staticGetter(wantURL, body string) realmGetter {
	return func(_ context.Context, url string) ([]byte, error) {
		if url != wantURL {
			return nil, os.ErrNotExist
		}
		return []byte(body), nil
	}
}

// TestLoadRealmResolvesRealHubIDFromURL is the live-bug regression: loading the
// mainnet Hub-List from its URL must resolve the real id MEIGKUAHT3CSYAAC (hub_id 2)
// to amlet.id, and yield the two follow entries. Non-vacuous: the pre-fix
// document-order mapping resolved hub_id 2 to nothing (only slots 0,1 existed).
func TestLoadRealmResolvesRealHubIDFromURL(t *testing.T) {
	hl, entries, err := loadRealm(context.Background(), mainnetHubListURL, staticGetter(mainnetHubListURL, mainnetHubListYAML), quietLogger())
	if err != nil {
		t.Fatalf("loadRealm: %v", err)
	}

	// Decode the real id from the bug report and resolve it the way the certificate
	// handler does: index.Decode -> registry.Resolve.
	id, err := index.Decode("MEIGKUAHT3CSYAAC")
	if err != nil {
		t.Fatalf("decode MEIGKUAHT3CSYAAC: %v", err)
	}
	if id.HubID != 2 {
		t.Fatalf("decoded hub_id = %d, want 2", id.HubID)
	}
	domain, ok := hl.Resolve(id.HubID)
	if !ok || domain != "amlet.id" {
		t.Fatalf("Resolve(hub_id %d) = %q, %v; want amlet.id, true", id.HubID, domain, ok)
	}

	wantDomains := []string{"iscc.id", "amlet.id"}
	if len(entries) != len(wantDomains) {
		t.Fatalf("entries len = %d, want %d: %+v", len(entries), len(wantDomains), entries)
	}
	for i, d := range wantDomains {
		if entries[i].Domain != d || entries[i].BaseURL != "https://"+d {
			t.Errorf("entries[%d] = %+v, want domain %q", i, entries[i], d)
		}
	}
}

// TestLoadRealmFromFile asserts a non-URL source is read from disk and parsed as a
// Hub-List, so tests and local dev can drive a file while production drives the URL.
func TestLoadRealmFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mainnet.yaml")
	if err := os.WriteFile(path, []byte(mainnetHubListYAML), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	hl, _, err := loadRealm(context.Background(), path, staticGetter("", ""), quietLogger())
	if err != nil {
		t.Fatalf("loadRealm(file): %v", err)
	}
	if domain, ok := hl.Resolve(2); !ok || domain != "amlet.id" {
		t.Errorf("Resolve(2) = %q, %v; want amlet.id, true", domain, ok)
	}
}

// TestLoadRealmDomainsOnlyFallback asserts the legacy line-based document still
// loads (backward compatibility) via the document-order mapping, so an old baked
// realm.txt keeps the binary booting until it is pointed at the Hub-List URL.
func TestLoadRealmDomainsOnlyFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "realm.txt")
	if err := os.WriteFile(path, []byte("# legacy\nsb0.iscc.id\nsb1.amlet.id\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	hl, entries, err := loadRealm(context.Background(), path, staticGetter("", ""), quietLogger())
	if err != nil {
		t.Fatalf("loadRealm(domains-only): %v", err)
	}
	// Document-order mapping: line 0 -> slot 0, line 1 -> slot 1.
	if domain, ok := hl.Resolve(0); !ok || domain != "sb0.iscc.id" {
		t.Errorf("Resolve(0) = %q, %v; want sb0.iscc.id, true (order mapping)", domain, ok)
	}
	if len(entries) != 2 {
		t.Errorf("entries len = %d, want 2", len(entries))
	}
}

// TestLoadRealmFetchError asserts a fetch failure propagates as a startup error
// rather than booting against an empty realm.
func TestLoadRealmFetchError(t *testing.T) {
	failing := func(_ context.Context, _ string) ([]byte, error) { return nil, os.ErrDeadlineExceeded }
	if _, _, err := loadRealm(context.Background(), "https://example.test/hubs.yaml", failing, quietLogger()); err == nil {
		t.Fatal("loadRealm with a failing getter returned nil error; want a wrapped fetch error")
	}
}

// TestLoadRealmRejectsUnparseableDocument asserts a body that is neither a Hub-List
// nor a domains-only document is a hard error (fail closed), never a silent empty
// realm.
func TestLoadRealmRejectsUnparseableDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.txt")
	if err := os.WriteFile(path, []byte("https://x.example/log\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, _, err := loadRealm(context.Background(), path, staticGetter("", ""), quietLogger()); err == nil {
		t.Fatal("loadRealm on a URL-shaped line returned nil error; want a fail-closed parse error")
	}
}

// TestRefreshRealmOnceSwapsThenKeepsLastGood asserts the refresh tick body swaps a
// fresh Hub-List into the live resolver, and that a subsequent failed fetch keeps
// the last good snapshot (resolution never breaks on a transient remote outage).
func TestRefreshRealmOnceSwapsThenKeepsLastGood(t *testing.T) {
	ctx := context.Background()
	// Seed the resolver with a single-hub realm (hub_id 1 only).
	resolver := registry.NewAtomicHubList(&registry.HubList{Version: 1, Hubs: []registry.Hub{
		{HubID: slotPtr(1), URL: "https://iscc.id", Active: true},
	}})
	if _, ok := resolver.Resolve(2); ok {
		t.Fatal("precondition: hub_id 2 should not resolve before refresh")
	}

	// A refresh that returns the full mainnet list adds hub_id 2.
	refreshRealmOnce(ctx, mainnetHubListURL, staticGetter(mainnetHubListURL, mainnetHubListYAML), resolver, quietLogger())
	if domain, ok := resolver.Resolve(2); !ok || domain != "amlet.id" {
		t.Fatalf("after refresh Resolve(2) = %q, %v; want amlet.id, true", domain, ok)
	}

	// A failed refresh must keep the last good snapshot.
	failing := func(_ context.Context, _ string) ([]byte, error) { return nil, os.ErrDeadlineExceeded }
	refreshRealmOnce(ctx, mainnetHubListURL, failing, resolver, quietLogger())
	if domain, ok := resolver.Resolve(2); !ok || domain != "amlet.id" {
		t.Errorf("after failed refresh Resolve(2) = %q, %v; want amlet.id, true (last good kept)", domain, ok)
	}
}

// TestLoadRealmURLMalformedHubListErrors asserts a URL source whose body fails
// ParseHubList surfaces the real Hub-List parse error rather than falling back to
// the legacy domains-only parser — which would misparse YAML keys ("version:",
// "hubs:") as bogus document-order domains. The legacy fallback is for filesystem
// sources only, so a malformed authoritative Hub-List fails closed.
func TestLoadRealmURLMalformedHubListErrors(t *testing.T) {
	const url = "https://example.test/hubs.yaml"
	cases := map[string]string{
		"duplicate hub_id": "version: 1\nhubs:\n  - hub_id: 1\n    url: https://a.io\n    active: true\n  - hub_id: 1\n    url: https://b.io\n    active: true\n",
		"schemeless url":   "version: 1\nhubs:\n  - hub_id: 1\n    url: sb0.iscc.id\n    active: true\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, entries, err := loadRealm(context.Background(), url, staticGetter(url, body), quietLogger())
			if err == nil {
				t.Fatalf("loadRealm on a malformed URL Hub-List returned nil error; want a parse error (entries=%+v)", entries)
			}
		})
	}
}

// TestLoadRealmRejectsEmptyRealm asserts a realm that lists no hubs fails closed
// rather than booting a monitor that follows nothing: a zero-hub Hub-List (`hubs: []`
// from a URL or file) and an all-comment legacy document both error.
func TestLoadRealmRejectsEmptyRealm(t *testing.T) {
	const url = "https://example.test/hubs.yaml"
	if _, _, err := loadRealm(context.Background(), url, staticGetter(url, "version: 1\nnetwork: mainnet\nhubs: []\n"), quietLogger()); err == nil {
		t.Error("loadRealm on a zero-hub URL Hub-List returned nil error; want an empty-realm error")
	}

	commentOnly := filepath.Join(t.TempDir(), "realm.txt")
	if err := os.WriteFile(commentOnly, []byte("# only comments\n\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, _, err := loadRealm(context.Background(), commentOnly, staticGetter("", ""), quietLogger()); err == nil {
		t.Error("loadRealm on an all-comment legacy document returned nil error; want an empty-realm error")
	}
}

// TestRefreshRealmOnceKeepsLastGoodOnMalformedBody asserts an hourly refresh that
// fetches a malformed Hub-List body keeps the last-good resolver rather than swapping
// in misparsed legacy entries (the bad-remote-response keep-last-good contract).
func TestRefreshRealmOnceKeepsLastGoodOnMalformedBody(t *testing.T) {
	ctx := context.Background()
	resolver := registry.NewAtomicHubList(&registry.HubList{Version: 1, Hubs: []registry.Hub{
		{HubID: slotPtr(1), URL: "https://iscc.id", Active: true},
		{HubID: slotPtr(2), URL: "https://amlet.id", Active: true},
	}})

	// A scheme-less url body fails ParseHubList; the URL refresh must NOT fall back.
	const malformed = "version: 1\nhubs:\n  - hub_id: 9\n    url: sb9.iscc.id\n    active: true\n"
	refreshRealmOnce(ctx, mainnetHubListURL, staticGetter(mainnetHubListURL, malformed), resolver, quietLogger())
	if domain, ok := resolver.Resolve(2); !ok || domain != "amlet.id" {
		t.Errorf("after malformed refresh Resolve(2) = %q, %v; want amlet.id, true (last good kept)", domain, ok)
	}
	if _, ok := resolver.Resolve(9); ok {
		t.Error("malformed refresh swapped in the garbage snapshot; want last good kept")
	}
}

// TestRefreshRealmOnceKeepsLastGoodOnEmptyRealm asserts that an hourly refresh which
// successfully fetches a zero-hub Hub-List keeps the last-good resolver: an empty
// realm is rejected by loadRealm, so refreshRealmOnce preserves the working snapshot
// instead of wiping resolution mid-run.
func TestRefreshRealmOnceKeepsLastGoodOnEmptyRealm(t *testing.T) {
	ctx := context.Background()
	resolver := registry.NewAtomicHubList(&registry.HubList{Version: 1, Hubs: []registry.Hub{
		{HubID: slotPtr(2), URL: "https://amlet.id", Active: true},
	}})

	const empty = "version: 1\nnetwork: mainnet\nhubs: []\n"
	refreshRealmOnce(ctx, mainnetHubListURL, staticGetter(mainnetHubListURL, empty), resolver, quietLogger())
	if domain, ok := resolver.Resolve(2); !ok || domain != "amlet.id" {
		t.Errorf("after empty-realm refresh Resolve(2) = %q, %v; want amlet.id, true (last good kept)", domain, ok)
	}
}

// TestHTTPGetBytesFailsClosed asserts the production getter rejects a non-200 status
// and an oversize body, so neither a 404/5xx error page nor a runaway response is
// ever parsed as (truncated) realm data.
func TestHTTPGetBytesFailsClosed(t *testing.T) {
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer notFound.Close()
	if _, err := httpGetBytes(context.Background(), notFound.URL); err == nil {
		t.Error("httpGetBytes on a 404 returned nil error; want a status error")
	}

	oversize := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, realmDocMaxBytes+10))
	}))
	defer oversize.Close()
	if _, err := httpGetBytes(context.Background(), oversize.URL); err == nil {
		t.Error("httpGetBytes on an oversize body returned nil error; want an exceeds-limit error")
	}
}
