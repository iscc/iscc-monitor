// Package verifier serves the standalone Independent Verification page (Surface C,
// the monitor-agnostic verifier app hosted at monitor.iscc.codes). It renders ONE
// static server-rendered HTML page: the masthead chrome, the .codes ↔ .id
// independence statement (the result is produced HERE; the monitor instance is not
// in the trust path), the verification-record step list, the skeptical-client
// (size, root) split-view input, and the guided split-view mismatch alert
// ("Mismatch — possible split view … do not discard either").
//
// This is the verifiable SKELETON of Surface C. It is a no-JS, no-CDN, self-hosted
// page that links the shared DS-token/font shell and references the audited WASM
// artifacts (/_ds/wasm_exec.js + /_ds/verify.wasm) the later live-wiring sub-step
// instantiates. This step deliberately runs NO WASM, parses NO ?monitor=<url> query,
// and computes NO live verdict — the alert states render as static markup matching
// the no-JS baseline, and the verification-record block describes the steps the
// verifier WILL run (never asserting a verdict that has not run).
//
// The leaf stays pure: stdlib + internal/web only (for the shared path consts the
// template literals must stay in sync with). It imports no internal/store,
// internal/metrics, or network, so it is golden-testable in isolation. The
// oracle/conformance gate is N/A: this is a pure static HTML render touching no
// signature, RFC-6962, Merkle, did:web, fsck, or proof path.
//
// Surface C lives on a DIFFERENT origin (a static site), not the instance binary,
// so this handler is NOT registered in cmd/iscc-monitor's buildMux. The exported
// Handler exists for the golden test (and a future local-preview/deploy harness).
package verifier

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
)

// pageTemplate is the embedded verifier template, parsed once at package init so a
// malformed template fails the build, not a request.
//
//go:embed verifier.html
var pageTemplate string

// tmpl is the parsed verifier template. template.Must panics at init if the source
// fails to parse, surfacing a template bug at startup rather than per request.
var tmpl = template.Must(template.New("verifier").Parse(pageTemplate))

// Handler returns an http.Handler that renders the standalone Independent
// Verification page. The page has no per-request data (it is a static SSR
// skeleton), so Handler takes no arguments. Only GET is served (any other method is
// 405). The page is rendered into a buffer first and only copied to the client on
// success, so a render error writes a 500 before any 200 is committed and a client
// never sees a half-rendered 200.
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
