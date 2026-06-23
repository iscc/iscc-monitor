// Golden HTTP-seam test for the in-app API reference page (GET /docs). It drives
// docs.Handler over an httptest request and asserts the observable response — 200
// text/html, the Stoplight Elements mount wired against this instance's /openapi.json
// (apiDescriptionUrl + the <elements-api> element + the same-origin /_ds/ asset
// refs), the shared no-CDN DS shell wiring (token/font links, self-hosted logo), the
// deliberate absence of tryItCorsProxy (Try-It calls this instance directly on the
// existing CORS *), the no-CDN body ban matching the sibling SSR seam tests, and a
// 405 for a non-GET. The oracle gate is N/A: this is a pure static HTML render with no
// signature, RFC-6962, Merkle, did:web, fsck, or proof path.
package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve renders the docs page over a plain GET and returns the recorder. The page is
// a single static artifact rendered identically for every request, so this one render
// is the whole observable surface.
func serve(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))
	return rec
}

// TestDocsRendersElementsShell asserts GET /docs returns 200 text/html mounting the
// Stoplight Elements <elements-api> component against /openapi.json from the
// self-hosted /_ds/ assets, on the shared no-CDN DS shell.
func TestDocsRendersElementsShell(t *testing.T) {
	rec := serve(t)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	body := rec.Body.String()

	for _, want := range []string{
		// The Elements mount: the component, its data source, and the self-hosted
		// JS/CSS asset refs (the /_ds/ paths are template literals synced to web.*).
		`<elements-api`,
		`apiDescriptionUrl="/openapi.json"`,
		`src="/_ds/elements.min.js"`,
		`href="/_ds/elements.min.css"`,
		// The shared no-CDN DS shell: token + font stylesheets and the self-hosted logo.
		`href="/_ds/tokens.css"`,
		`href="/_ds/fonts.css"`,
		`src="/_ds/iscc-logo-black.png"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing marker %q\n%s", want, body)
		}
	}
}

// TestDocsNoTryItCorsProxy asserts the tryItCorsProxy attribute is left UNSET so the
// Try-It console issues requests browser → this instance directly on the existing
// CORS * policy, never a third-party proxy (ADR-0014 §4).
func TestDocsNoTryItCorsProxy(t *testing.T) {
	body := serve(t).Body.String()
	if strings.Contains(body, "tryItCorsProxy") {
		t.Errorf("body unexpectedly carries tryItCorsProxy (Try-It must call this instance directly)\n%s", body)
	}
}

// TestDocsNoExternalCDN bans any external CDN origin in the /docs HTML page BODY —
// same-origin / relative hrefs only — matching the sibling no-CDN SSR seam tests. The
// ban applies ONLY to the rendered HTML page, NEVER to the 2 MB Elements JS bytes
// (which carry hundreds of inert baked example URL strings that are data, not runtime
// fetches — see web.go's elementsJS doc comment).
func TestDocsNoExternalCDN(t *testing.T) {
	body := serve(t).Body.String()
	for _, banned := range []string{"jsdelivr", "http://", "https://", "cdn."} {
		if strings.Contains(body, banned) {
			t.Errorf("body contains external CDN reference %q\n%s", banned, body)
		}
	}
}

// TestDocsNonGET asserts a non-GET method is a 405 (the shared method gate at the top
// of Handler covers it).
func TestDocsNonGET(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/docs", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
