// Package docs serves the monitor's in-app interactive API reference at GET /docs:
// a server-rendered HTML shell that mounts the self-hosted Stoplight Elements
// <elements-api> web component against this instance's own /openapi.json (ADR-0014
// §4). The page is the no-JS DS shell every server-rendered surface uses — it links
// /_ds/tokens.css + /_ds/fonts.css + /_ds/site.css and renders the shared masthead and
// footer (internal/chrome) — plus the two self-hosted
// Stoplight Elements assets served from /_ds/ (the web-component JS bundle and its
// stylesheet, each byte-pinned in internal/web alongside verify.wasm).
//
// The page carries no per-request data: it renders identically for every request,
// varying only with the instance's configured chrome.Site. Elements' "Try It"
// console issues requests browser → this instance directly on the existing CORS *
// policy, so the tryItCorsProxy attribute is left UNSET — no external CDN host
// appears in the page body and the surface makes no external runtime call.
//
// The leaf imports no internal/store, internal/metrics, network, or net/url (only the
// internal/chrome partials), so it is golden-testable in isolation. The /_ds/... paths
// are template LITERALS kept in sync with web.TokensPath / FontsCSSPath / SiteCSSPath /
// ElementsJSPath / ElementsCSSPath by comment (the
// dashboard/dossier/verifier convention). The oracle/conformance gate is N/A: this
// renders only HTML and serves no signature, RFC-6962, Merkle, did:web, fsck, or
// proof path.
package docs

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/iscc/iscc-monitor/internal/chrome"
)

// pageTemplate is the embedded docs template, parsed once at package init so a
// malformed template fails the build, not a request.
//
//go:embed docs.html
var pageTemplate string

// tmpl is the parsed docs template with the shared chrome partials associated.
// template.Must panics at init if a source fails to parse, surfacing a template bug at
// startup rather than per request.
var tmpl = func() *template.Template {
	t := template.Must(template.New("docs").Parse(pageTemplate))
	return template.Must(t.Parse(chrome.Source))
}()

// subAPI is the docs masthead line beneath the "Trust & Transparency Monitor" mark.
const subAPI = "API reference · ISCC-Hub network"

// pageData is the docs template context: the shared masthead and footer.
type pageData struct {
	Chrome chrome.View
}

// Handler returns an http.Handler that renders the in-app API reference page. The
// page has no per-request data: it mounts the self-hosted Stoplight Elements
// <elements-api> component against /openapi.json under the shared chrome resolved from
// site, so the same bytes serve every request. Only GET is served (any other method is 405).
// The page is rendered into a buffer first and only copied to the client on success,
// so a render error writes a 500 before any 200 is committed and a client never sees
// a half-rendered 200.
func Handler(site chrome.Site) http.Handler {
	data := pageData{Chrome: site.View()}
	data.Chrome.Sub = subAPI
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// Post-200 write-drop: the status is already committed, so a copy error can
		// only signal a broken client connection, which a second status cannot fix.
		_, _ = buf.WriteTo(w)
	})
}
