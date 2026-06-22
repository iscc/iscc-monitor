// Package verifier serves the standalone Independent Verification page (Surface C,
// the monitor-agnostic verifier app hosted at monitor.iscc.codes). It renders ONE
// server-rendered HTML page: the masthead chrome, the .codes ↔ .id independence
// statement (the result is produced HERE; the monitor instance is not in the trust
// path), the verification-record step list, the skeptical-client (size, root)
// split-view input, and the guided split-view mismatch alert ("Mismatch — possible
// split view … do not discard either").
//
// The page is a single STATIC artifact: it carries no per-request data and is
// rendered identically for every request (tmpl.Execute(&buf, nil)). The
// ?monitor=<instance-url>&id=<iscc_id> target is read CLIENT-side from
// location.search by the always-emitted end-of-body loader (/_ds/wasm_exec.js +
// /_ds/verify.wasm) — it is NEVER parsed by the handler nor reflected into the body.
// This is what lets the one pre-generated index.html, served byte-for-byte by a
// static host (GitHub Pages), serve both the no-target baseline AND a live
// ?monitor=…&id=… run: the BROWSER fetches <monitor>/inclusion/<id>.bundle, runs
// isccVerifyInclusion, and gates the verdict region on the genuine re-VERIFICATION
// — never a static render. The mismatch alert appears ONLY on a real negative
// verdict (failed), distinct from a broken-input error and a verified pass. With no
// target (or no JS) the page renders the honest baseline: it names the steps the
// verifier WILL run and asserts NO un-run verdict.
//
// The leaf stays pure (stdlib only) — it imports no internal/store, internal/web,
// internal/metrics, network, or net/url, so it is golden-testable in isolation. The
// /_ds/... paths are template LITERALS kept in sync with web.TokensPath/
// FontsCSSPath/WasmVerifyPath/WasmExecPath/LogoPath by comment (the dashboard/dossier
// convention). The oracle/conformance gate is N/A: the live verdict runs in the
// BROWSER (verified by the review step's headless visual pass, ADR-0012), not in go
// test; this handler renders only HTML.
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
// Verification page. The page is a single static artifact with no per-request data:
// the ?monitor=&id= target is read CLIENT-side from location.search by the
// always-emitted loader, never by this handler, so the same bytes serve both the
// no-target baseline and a live run. Only GET is served (any other method is 405).
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
