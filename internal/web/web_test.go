// Tests for the static-asset handler at the HTTP seam: they drive Handler over an
// httptest.ResponseRecorder and assert the observable response — GET returns 200
// text/css with a non-empty body carrying a known token, a non-GET is 405 — plus
// the load-bearing CDN-free invariant: the served stylesheet contains no jsdelivr
// substring and no "http" at all, so the body references no external origin.
package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokensServedAsCSS(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, TokensPath, nil))

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

// TestTokensCDNFree pins the load-bearing M-UI invariant: the served token CSS
// references no external CDN origin. The only file in the design bundle carrying
// CDN URLs is fonts.css (excluded), and the grain background-image URL is
// neutralized, so the body has zero "jsdelivr" and zero "http" substrings.
func TestTokensCDNFree(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, TokensPath, nil))
	body := rec.Body.Bytes()

	if bytes.Contains(body, []byte("jsdelivr")) {
		t.Error("served tokens.css contains jsdelivr")
	}
	if bytes.Contains(body, []byte("http")) {
		t.Error("served tokens.css contains an http(s) URL")
	}
	if bytes.Contains(body, []byte("url(")) {
		t.Error("served tokens.css contains a url() reference")
	}
}

func TestTokensMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, TokensPath, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d, want 405", rec.Code)
	}
}
