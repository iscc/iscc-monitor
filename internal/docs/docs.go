// Package docs serves the monitor's in-app interactive API reference at GET /docs:
// a server-rendered HTML shell that mounts the self-hosted Stoplight Elements
// <elements-api> web component against this instance's own /openapi.json (ADR-0014
// §4). The page is the no-JS DS shell every server-rendered surface uses — it links
// /_ds/tokens.css + /_ds/fonts.css and the masthead logo — plus the two self-hosted
// Stoplight Elements assets served from /_ds/ (the web-component JS bundle and its
// stylesheet, each byte-pinned in internal/web alongside verify.wasm).
//
// The page is a single STATIC artifact: it carries no per-request data and renders
// identically for every request (tmpl.Execute(&buf, nil)). Elements' "Try It"
// console issues requests browser → this instance directly on the existing CORS *
// policy, so the tryItCorsProxy attribute is left UNSET — no external CDN host
// appears in the page body and the surface makes no external runtime call.
//
// The leaf stays pure (stdlib only) — it imports no internal/store, internal/web,
// internal/metrics, network, or net/url, so it is golden-testable in isolation. The
// /_ds/... paths are template LITERALS kept in sync with web.TokensPath /
// FontsCSSPath / LogoPath / ElementsJSPath / ElementsCSSPath by comment (the
// dashboard/dossier/verifier convention). The oracle/conformance gate is N/A: this
// renders only HTML and serves no signature, RFC-6962, Merkle, did:web, fsck, or
// proof path.
package docs

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
)

// pageTemplate is the embedded docs template, parsed once at package init so a
// malformed template fails the build, not a request.
//
//go:embed docs.html
var pageTemplate string

// tmpl is the parsed docs template. template.Must panics at init if the source
// fails to parse, surfacing a template bug at startup rather than per request.
var tmpl = template.Must(template.New("docs").Parse(pageTemplate))

// Handler returns an http.Handler that renders the in-app API reference page. The
// page is a single static artifact with no per-request data: it mounts the
// self-hosted Stoplight Elements <elements-api> component against /openapi.json, so
// the same bytes serve every request. Only GET is served (any other method is 405).
// The page is rendered into a buffer first and only copied to the client on success,
// so a render error writes a 500 before any 200 is committed and a client never sees
// a half-rendered 200.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, nil); err != nil {
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
