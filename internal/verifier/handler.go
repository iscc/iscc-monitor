// Package verifier serves the standalone Independent Verification page (Surface C,
// the monitor-agnostic verifier app hosted at monitor.iscc.codes). It renders ONE
// server-rendered HTML page: the masthead chrome, the .codes ↔ .id independence
// statement (the result is produced HERE; the monitor instance is not in the trust
// path), the verification-record step list, the skeptical-client (size, root)
// split-view input, and the guided split-view mismatch alert ("Mismatch — possible
// split view … do not discard either").
//
// The page is the no-JS, no-CDN, self-hosted SKELETON plus the live re-verification
// wiring. Given ?monitor=<instance-url>&id=<iscc_id>, the handler embeds a JSON
// data-island naming the target and a progressive-enhancement loader (the audited
// /_ds/wasm_exec.js + /_ds/verify.wasm); the BROWSER fetches
// <monitor>/inclusion/<id>.bundle, runs isccVerifyInclusion, and gates the verdict
// region on the genuine re-VERIFICATION — never a static render. The mismatch alert
// appears ONLY on a real negative verdict (failed), distinct from a broken-input
// error and a verified pass. With no target (or no JS) the page renders the honest
// baseline: it names the steps the verifier WILL run and asserts NO un-run verdict.
//
// The target fetch happens in the browser, never the handler: the handler only
// validates and reflects the target into the data-island. The leaf stays pure
// (stdlib only) — it imports no internal/store, internal/web, internal/metrics, or
// network, so it is golden-testable in isolation. The /_ds/... paths are template
// LITERALS kept in sync with web.TokensPath/FontsCSSPath/WasmVerifyPath/WasmExecPath/
// LogoPath by comment (the dashboard/dossier convention). The oracle/conformance
// gate is N/A: the live verdict runs in the BROWSER (verified by the review step's
// headless visual pass, ADR-0012), not in go test; this handler renders only HTML.
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
	"net/url"
)

// pageTemplate is the embedded verifier template, parsed once at package init so a
// malformed template fails the build, not a request.
//
//go:embed verifier.html
var pageTemplate string

// tmpl is the parsed verifier template. template.Must panics at init if the source
// fails to parse, surfacing a template bug at startup rather than per request.
var tmpl = template.Must(template.New("verifier").Parse(pageTemplate))

// pageData is the per-request view-model. HasTarget is the load-bearing gate: only a
// well-formed ?monitor=<url>&id=<iscc_id> pair sets it true, which is what the
// template keys the JSON data-island, the WASM loader, and the live verdict/mismatch
// region on. With HasTarget false the page renders the honest no-target baseline that
// asserts NO un-run verdict. Monitor and ID are reflected ONLY inside the JSON
// data-island (JSON-context-escaped by html/template), never into executable JS and
// never into the static body — so the user-supplied target URL cannot trip the
// no-CDN body ban on stylesheet/script/img origins.
type pageData struct {
	HasTarget bool
	Monitor   string
	ID        string
}

// Handler returns an http.Handler that renders the standalone Independent
// Verification page. Given ?monitor=<instance-url>&id=<iscc_id> it wires the live
// in-browser re-verification (data-island + WASM loader); with no valid target it
// renders the honest baseline. Only GET is served (any other method is 405). The
// page is rendered into a buffer first and only copied to the client on success, so
// a render error writes a 500 before any 200 is committed and a client never sees a
// half-rendered 200.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, parseTarget(r.URL.Query())); err != nil {
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

// parseTarget reads ?monitor= and ?id= from the query into a pageData, failing
// CLOSED to the no-target baseline (HasTarget=false) on any reject. It requires a
// non-empty id and a monitor URL that parses with an http/https scheme and a
// non-empty host and carries no fragment — the minimum for the browser to fetch
// <monitor>/inclusion/<id>.bundle. It does NOT fetch anything; the bundle fetch and
// verdict happen in the browser. Both values are reflected only inside the JSON
// data-island, so the validation here is a guard against an obviously unusable
// target, not a trust boundary (the browser re-validates the fetched bundle).
func parseTarget(q url.Values) pageData {
	monitor := q.Get("monitor")
	id := q.Get("id")
	if monitor == "" || id == "" {
		return pageData{}
	}
	u, err := url.Parse(monitor)
	if err != nil {
		return pageData{}
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Fragment != "" {
		return pageData{}
	}
	return pageData{HasTarget: true, Monitor: monitor, ID: id}
}
