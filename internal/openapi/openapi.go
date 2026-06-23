// Package openapi serves the monitor's hand-authored OpenAPI 3.1 contract — the
// machine-readable description of its JSON / artifact / Prometheus-text surface — as
// a tiny stdlib + gopkg.in/yaml.v3 leaf. The in-repo document (openapi.yaml, with a
// byte-distinct but semantically identical openapi.json) is go:embed-ed at build
// time and served byte-verbatim at GET /openapi.json and GET /openapi.yaml, so a
// third-party integrator can generate a client, validate responses, and explore the
// API without reading the monitor's source (ADR-0014).
//
// The contract describes ONLY the machine-consumable surface: /healthz, /version,
// /metrics (as Prometheus text, not JSON-schema'd), the per-hub log routes
// (inclusion / consistency / entries / verify / checkpoint / checkpoint.ots / tile),
// the realm-wide proof bundle (/inclusion/{iscc_id}.bundle), and these two
// /openapi.* endpoints themselves. The human-facing HTML SSR surfaces (the realm
// index, the hub dossier, the log browser, the HTML certificate, and the /_ds/
// assets) are out of the contract by design — the route-vs-spec drift test (in
// cmd/iscc-monitor) keeps them on an explicit exclusion list. The verify-for-me path
// is documented in-band as the explicitly weaker, non-authoritative tier-1 path.
//
// DeclaredPaths exposes the document's declared path set so the drift test can
// reconcile it against the real mux. Handler mirrors internal/web's revalidating
// cache shape (no-cache + strong content ETag + If-None-Match -> 304); CORS rides the
// outer corsmw wrap at the mux convergence point, so this handler sets no CORS
// headers of its own.
//
// The package is a pure leaf: its only imports are crypto/sha256 / fmt / net/http /
// sort / sync / embed (stdlib) and gopkg.in/yaml.v3, with no internal/store,
// internal/metrics, or internal/logclient — the parser reads the embedded bytes, not
// the live mux. The oracle/conformance gate is N/A: this serves committed bytes and
// reconciles route strings, touching no signature, RFC-6962, Merkle, did:web, fsck,
// or proof path.
package openapi

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"net/http"
	"sort"
	"sync"

	yaml "gopkg.in/yaml.v3"
)

// JSONPath is the stable exact path the OpenAPI document is served at as JSON. The
// binary mounts Handler here; it stays in sync with the route the drift test
// reconciles.
const JSONPath = "/openapi.json"

// YAMLPath is the stable exact path the OpenAPI document is served at as YAML.
const YAMLPath = "/openapi.yaml"

// contentTypeJSON is the media type served for the JSON document.
const contentTypeJSON = "application/json"

// contentTypeYAML is the media type served for the YAML document.
const contentTypeYAML = "application/yaml"

// cacheControl is the revalidating Cache-Control policy for the served document.
// Like the /_ds/ assets it is a stable, overwrite-in-place URL (a redeploy reuses
// the path with changed bytes), so it must NOT carry immutable; no-cache forces a
// cheap ETag revalidation before reuse.
const cacheControl = "no-cache"

// JSON is the embedded OpenAPI 3.1 document as JSON — the byte-distinct twin of YAML,
// describing the identical path set (TestOpenAPIDocsAgree). It is served verbatim at
// JSONPath.
//
//go:embed openapi.json
var JSON []byte

// YAML is the embedded OpenAPI 3.1 document as YAML — the hand-authored source of
// truth, served verbatim at YAMLPath.
//
//go:embed openapi.yaml
var YAML []byte

// declaredPathsOnce memoizes the parse of the embedded YAML's paths key, since the
// document is fixed at build time and the drift test may call DeclaredPaths more than
// once.
var declaredPathsOnce = sync.OnceValue(func() []string {
	return parsePaths(YAML)
})

// DeclaredPaths returns the sorted set of path templates the embedded OpenAPI
// document declares (the keys of its top-level `paths` object, e.g.
// "/{domain}/log/inclusion"). The drift test consumes this to reconcile the
// document against the routes the real mux mounts. It parses the embedded YAML once.
func DeclaredPaths() []string {
	return declaredPathsOnce()
}

// parsePaths extracts the sorted top-level path-template keys from an OpenAPI
// document's `paths` object. JSON is valid YAML, so yaml.Unmarshal reads either the
// embedded YAML or JSON. A document with no paths yields an empty (non-nil) slice.
func parsePaths(doc []byte) []string {
	var parsed struct {
		Paths map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(doc, &parsed); err != nil {
		return []string{}
	}
	paths := make([]string, 0, len(parsed.Paths))
	for p := range parsed.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// Handler returns an http.Handler serving the OpenAPI document byte-verbatim: GET
// JSONPath as application/json (the embedded JSON bytes) and GET YAMLPath as
// application/yaml (the embedded YAML bytes). Only GET is served (any other method
// is 405); any other path is 404. Every 200 carries Cache-Control: no-cache and a
// strong content ETag, with an If-None-Match match short-circuiting to 304. It sets
// no CORS headers — the outer corsmw wrap owns the single CORS policy.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case JSONPath:
			writeDoc(w, r, JSON, contentTypeJSON)
		case YAMLPath:
			writeDoc(w, r, YAML, contentTypeYAML)
		default:
			http.NotFound(w, r)
		}
	})
}

// writeDoc sets the content type, the revalidating Cache-Control, and a strong
// content ETag, then either short-circuits to 304 or writes the document bytes. It
// is the internal/web.writeAsset shape: the ETag is the quoted hex of the bytes'
// SHA-256 (a STRONG validator, no W/ prefix), and an If-None-Match that is the
// wildcard "*" or echoes this exact tag yields 304 Not Modified with no body.
func writeDoc(w http.ResponseWriter, r *http.Request, data []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", cacheControl)
	etag := fmt.Sprintf("\"%x\"", sha256.Sum256(data))
	w.Header().Set("ETag", etag)

	if inm := r.Header.Get("If-None-Match"); inm == "*" || inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.WriteHeader(http.StatusOK)
	// Post-200 write-drop: the status is committed, so a copy error can only signal a
	// broken client connection, which a second status cannot fix.
	_, _ = w.Write(data)
}
