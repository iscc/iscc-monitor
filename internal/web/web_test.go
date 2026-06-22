// Tests for the static-asset handler at the HTTP seam: they drive Handler over an
// httptest.ResponseRecorder and assert the observable response. The token and font
// stylesheets return 200 text/css with a non-empty body carrying a known marker; the
// wasm_exec.js loader returns 200 text/javascript with the Go runtime symbol; a woff2
// binary returns 200 font/woff2 with the woff2 magic bytes; a non-GET is 405; an
// If-None-Match match short-circuits to 304. They also pin the load-bearing CDN-free
// invariant: no served body references an external CDN origin in loadable content (no
// jsdelivr, no http(s) scheme, no cdn. host) — a same-origin url("/_ds/fonts/...") in
// fonts.css and a vendored runtime's source-comment URL are legitimate and not banned.
package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// get drives Handler with a GET for path and returns the recorder.
func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// noExternalCDN fails if body references a third-party origin in loadable content. A
// same-origin url("/_ds/...") is allowed; only jsdelivr / an http(s) scheme / a cdn.
// host are banned, so a self-hosted @font-face src passes. Each line's // comment is
// stripped before scanning, since a comment can never trigger a runtime CDN fetch —
// this keeps the ban exactly as strict for real (loadable) content while permitting a
// vendored runtime's source comment (e.g. wasm_exec.js's Go issue-tracker URL).
func noExternalCDN(t *testing.T, name string, body []byte) {
	t.Helper()
	scanned := stripLineComments(body)
	for _, banned := range []string{"jsdelivr", "http://", "https://", "cdn."} {
		if bytes.Contains(scanned, []byte(banned)) {
			t.Errorf("%s references external origin %q", name, banned)
		}
	}
}

// stripLineComments removes the // comment tail from each line so noExternalCDN scans
// only loadable content. A comment is non-executable text, so a URL in it cannot be a
// runtime resource reference. It treats // as a comment only when it is NOT the //
// inside a scheme (a preceding ':' as in https://), so a real loadable https:// URL is
// never truncated and the ban stays exactly as strict for executable content; CSS uses
// /* */ block comments (no // tail) and is unaffected.
func stripLineComments(body []byte) []byte {
	lines := bytes.Split(body, []byte("\n"))
	for i, line := range lines {
		for j := 0; j+1 < len(line); j++ {
			if line[j] == '/' && line[j+1] == '/' && (j == 0 || line[j-1] != ':') {
				lines[i] = line[:j]
				break
			}
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

func TestTokensServedAsCSS(t *testing.T) {
	rec := get(t, TokensPath)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Errorf("Content-Type = %q, want text/css prefix", got)
	}
	body := rec.Body.Bytes()
	if len(body) == 0 {
		t.Fatal("body is empty")
	}
	// A known token must be present so the served bytes are the real stylesheet.
	if !bytes.Contains(body, []byte("--iscc-blue")) {
		t.Errorf("body missing --iscc-blue token\n%s", body)
	}
}

// TestTokensCDNFree pins the load-bearing M-UI invariant for the token CSS: it
// references no external CDN origin. tokens.css excludes the jsDelivr @font-face
// block and neutralizes the grain background-image, so the body carries no jsdelivr,
// no http(s) URL, and no cdn. host.
func TestTokensCDNFree(t *testing.T) {
	noExternalCDN(t, "served tokens.css", get(t, TokensPath).Body.Bytes())
}

func TestTokensRevalidateETag(t *testing.T) {
	rec := get(t, TokensPath)
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	etag := rec.Header().Get("ETag")
	if !strings.HasPrefix(etag, "\"") || strings.HasPrefix(etag, "W/") {
		t.Errorf("ETag = %q, want a strong quoted-hex tag", etag)
	}
}

// TestFontsCSSServed checks the @font-face stylesheet is served as CSS, carries an
// @font-face rule pointing at a same-origin /_ds/fonts/ path, and references no CDN.
func TestFontsCSSServed(t *testing.T) {
	rec := get(t, FontsCSSPath)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Errorf("Content-Type = %q, want text/css prefix", got)
	}
	body := rec.Body.Bytes()
	if !bytes.Contains(body, []byte("@font-face")) {
		t.Errorf("fonts.css missing @font-face\n%s", body)
	}
	if !bytes.Contains(body, []byte("/_ds/fonts/")) {
		t.Errorf("fonts.css missing same-origin /_ds/fonts/ src\n%s", body)
	}
	noExternalCDN(t, "served fonts.css", body)
}

// TestFontBinaryServed checks a woff2 binary is served with the font/woff2 content
// type, a strong ETag, no-cache, and the woff2 magic header ("wOF2").
func TestFontBinaryServed(t *testing.T) {
	rec := get(t, "/_ds/fonts/readex-pro-400.woff2")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "font/woff2" {
		t.Errorf("Content-Type = %q, want font/woff2", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	if etag := rec.Header().Get("ETag"); !strings.HasPrefix(etag, "\"") || strings.HasPrefix(etag, "W/") {
		t.Errorf("ETag = %q, want a strong quoted-hex tag", etag)
	}
	if got := rec.Body.Bytes(); len(got) < 4 || string(got[:4]) != "wOF2" {
		t.Errorf("body is not a woff2 (magic = %x)", got[:min(4, len(got))])
	}
}

// TestFontBinaryAllSubsets checks every one of the eight embedded latin subsets is
// served — the self-hosted @font-face stylesheet references them all.
func TestFontBinaryAllSubsets(t *testing.T) {
	subsets := []string{
		"readex-pro-300.woff2", "readex-pro-400.woff2", "readex-pro-500.woff2",
		"readex-pro-600.woff2", "readex-pro-700.woff2",
		"jetbrains-mono-400.woff2", "jetbrains-mono-500.woff2", "jetbrains-mono-700.woff2",
	}
	for _, name := range subsets {
		rec := get(t, "/_ds/fonts/"+name)
		if rec.Code != http.StatusOK {
			t.Errorf("GET /_ds/fonts/%s = %d, want 200", name, rec.Code)
		}
	}
}

// TestFontsCSSReferencesEmbeddedSubsets cross-checks that every src="/_ds/fonts/..."
// path named in the served fonts.css resolves to an embedded binary (200), so the
// stylesheet never points at a missing file. It scans only inside the double-quoted
// src values, so prose mentioning the marker in a comment is not treated as a URL.
func TestFontsCSSReferencesEmbeddedSubsets(t *testing.T) {
	css := get(t, FontsCSSPath).Body.String()
	const marker = `"/_ds/fonts/`
	found := 0
	for rest := css; ; {
		i := strings.Index(rest, marker)
		if i < 0 {
			break
		}
		rest = rest[i+1:] // step past the opening quote
		path, _, ok := strings.Cut(rest, `"`)
		if !ok {
			t.Fatalf("unterminated font URL in fonts.css near %q", rest[:min(40, len(rest))])
		}
		if rec := get(t, path); rec.Code != http.StatusOK {
			t.Errorf("fonts.css references %s but GET it = %d", path, rec.Code)
		}
		found++
	}
	if found != 8 {
		t.Errorf("fonts.css references %d font URLs, want 8 latin subsets", found)
	}
}

// TestWasmExecServed checks the Go WASM runtime loader is served at WasmExecPath with
// the text/javascript content type, the revalidating no-cache + strong ETag policy, a
// non-empty body, and no external CDN reference — the same-origin runtime the tier-2
// progressive enhancement loads before instantiating the verifier .wasm.
func TestWasmExecServed(t *testing.T) {
	rec := get(t, WasmExecPath)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/javascript; charset=utf-8", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	if etag := rec.Header().Get("ETag"); !strings.HasPrefix(etag, "\"") || strings.HasPrefix(etag, "W/") {
		t.Errorf("ETag = %q, want a strong quoted-hex tag", etag)
	}
	body := rec.Body.Bytes()
	if len(body) == 0 {
		t.Fatal("body is empty")
	}
	// A known runtime symbol must be present so the served bytes are the real loader.
	if !bytes.Contains(body, []byte("globalThis.Go")) {
		t.Errorf("body missing globalThis.Go runtime symbol")
	}
	noExternalCDN(t, "served wasm_exec.js", body)
}

func TestFontMissingIs404(t *testing.T) {
	if rec := get(t, "/_ds/fonts/does-not-exist.woff2"); rec.Code != http.StatusNotFound {
		t.Errorf("missing font status = %d, want 404", rec.Code)
	}
}

// TestIfNoneMatch304 checks the conditional-GET short-circuit: a request echoing the
// served ETag yields 304 with an empty body and the same validating ETag.
func TestIfNoneMatch304(t *testing.T) {
	for _, path := range []string{TokensPath, FontsCSSPath, WasmExecPath, "/_ds/fonts/jetbrains-mono-700.woff2"} {
		etag := get(t, path).Header().Get("ETag")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("If-None-Match", etag)
		Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNotModified {
			t.Errorf("%s with If-None-Match = %d, want 304", path, rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("%s 304 body not empty (%d bytes)", path, rec.Body.Len())
		}
		if got := rec.Header().Get("ETag"); got != etag {
			t.Errorf("%s 304 ETag = %q, want %q", path, got, etag)
		}
	}
}

func TestMethodNotAllowed(t *testing.T) {
	for _, path := range []string{TokensPath, FontsCSSPath, WasmExecPath, "/_ds/fonts/readex-pro-400.woff2"} {
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s = %d, want 405", path, rec.Code)
		}
	}
}
