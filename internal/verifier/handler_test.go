// Golden HTTP-seam test for the standalone Independent Verification page (Surface C).
// It drives verifier.Handler over an httptest request and asserts the observable
// response — 200 text/html, the named regions of the Independent Verification mockup
// (the eyebrow, the .codes ↔ .id independence statement, the five verification-record
// step labels, the guided split-view mismatch alert + its "do not discard either"
// guidance), the shared no-CDN DS shell wiring (token/font links, self-hosted logo,
// the audited /_ds/verify.wasm artifact reference), and the no-CDN body ban matching
// the sibling SSR seam tests. The oracle gate is N/A: no signature, RFC-6962, Merkle,
// did:web, fsck, or proof path — this is a pure static HTML render.
package verifier

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve renders the verifier page over a no-query GET request (the no-target
// baseline) and returns the recorder.
func serve(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	return serveTarget(t, "")
}

// serveTarget renders the verifier page over a GET request whose URL carries the
// given raw query (e.g. "monitor=https://monitor.iscc.id&id=ISCC:…") and returns the
// recorder. An empty query is the no-target baseline.
func serveTarget(t *testing.T, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	target := "/"
	if rawQuery != "" {
		target = "/?" + rawQuery
	}
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// TestVerifierRendersNamedRegions asserts GET returns 200 text/html with every named
// region of the Independent Verification mockup: the DS shell wiring, the eyebrow and
// head, the .codes ↔ .id independence statement, all five verification-record step
// labels, the split-view input, and the guided split-view mismatch alert.
func TestVerifierRendersNamedRegions(t *testing.T) {
	rec := serve(t)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	body := rec.Body.String()

	// The shared no-CDN DS shell: token + font stylesheets, the self-hosted ISCC
	// logo img, the DS font tokens, and the audited WASM artifacts the deferred
	// live-wiring sub-step instantiates.
	for _, want := range []string{
		`href="/_ds/tokens.css"`,
		`href="/_ds/fonts.css"`,
		`src="/_ds/iscc-logo-black.png"`,
		"/_ds/verify.wasm",
		"/_ds/wasm_exec.js",
		"var(--font-sans)",
		"var(--font-mono)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing DS-shell marker %q\n%s", want, body)
		}
	}

	// The eyebrow + head + the navigation-closure back-link to the certificate.
	for _, want := range []string{
		"Independent verification",
		"Re-run the proof yourself.",
		"← Certificate",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing page marker %q\n%s", want, body)
		}
	}

	// The five verification-record step labels (the mockup's vSteps) render as
	// static no-JS list rows.
	for _, want := range []string{
		"Fetch proof bundle from the monitor",
		"Recompute the leaf hash from the record bytes",
		"Walk the Merkle inclusion path",
		"Rebuild the root and match the hub-signed checkpoint",
		"Check the signature against the hub's did:web key",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing verification-record step %q\n%s", want, body)
		}
	}

	// The split-view input region: tree-size + root-hash fields and a Compare
	// affordance, rendered as a plain no-JS GET form.
	for _, want := range []string{
		"Compare your own (size, root)",
		"Tree size",
		"Root hash",
		">Compare<",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing split-view input marker %q\n%s", want, body)
		}
	}
}

// TestVerifierIndependenceStatement asserts the .codes ↔ .id independence statement
// names both origins and states the result is produced HERE while the monitor
// instance is not in the trust path (Handoff invariant 5 — the distinction stays
// visible). Both literal origins must appear in the body.
func TestVerifierIndependenceStatement(t *testing.T) {
	rec := serve(t)
	body := rec.Body.String()

	for _, want := range []string{
		"monitor.iscc.codes",
		"monitor.iscc.id",
		"not in the trust path",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing independence-statement marker %q\n%s", want, body)
		}
	}
}

// TestVerifierGuidedMismatchAlert asserts the load-bearing Verify fragment: the
// guided split-view mismatch alert renders the "Mismatch — possible split view"
// heading and the guidance to keep both signed histories ("do not discard either"),
// never a dead error. This is the criterion the advance step mutation-proves.
func TestVerifierGuidedMismatchAlert(t *testing.T) {
	rec := serve(t)
	body := rec.Body.String()

	for _, want := range []string{
		"Mismatch — possible split view",
		"do not discard either",
		"both signed histories are evidence",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing guided split-view alert marker %q\n%s", want, body)
		}
	}
}

// TestVerifierNoTargetBaselineIsHonest asserts the no-target baseline asserts NO
// un-run verdict: with no ?monitor=&id= the body must NOT contain the present-tense
// negative-verdict assertion "Your (size, root) does not match" (that copy lives only
// inside the {{if .HasTarget}} loader, revealed only on a real `failed` verdict),
// while the honest named regions + the "not yet run" run-label still render. This is
// the honesty fix the advance mutation-proves: it closes the open `normal` issue
// where the mismatch alert claimed an un-run negative verdict.
func TestVerifierNoTargetBaselineIsHonest(t *testing.T) {
	rec := serve(t)
	body := rec.Body.String()

	// The un-run present-tense negative verdict must be absent from the baseline.
	if strings.Contains(body, "Your (size, root) does not match") {
		t.Errorf("no-target baseline asserts an un-run negative verdict (found present-tense \"Your (size, root) does not match\")\n%s", body)
	}
	// The honest no-verdict markers still render: the run-label and the explicit
	// no-verdict-claimed copy, plus the illustrative framing on the mismatch example.
	for _, want := range []string{
		"not yet run",
		"no verdict is claimed",
		"Illustrative — what a real mismatch shows",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("no-target baseline missing honest marker %q\n%s", want, body)
		}
	}
	// The baseline wires no live target: no data-island, no end-of-body WASM loader.
	for _, absent := range []string{
		`id="verify-target"`,
		`<script src="/_ds/wasm_exec.js">`,
		"isccVerifyInclusion",
	} {
		if strings.Contains(body, absent) {
			t.Errorf("no-target baseline unexpectedly wired live verification (found %q)\n%s", absent, body)
		}
	}
}

// TestVerifierConfiguredTargetWiresLiveVerification asserts that a well-formed
// ?monitor=&id= renders the live re-verification wiring: the type="application/json"
// data-island, the /_ds/wasm_exec.js + /_ds/verify.wasm loader, the isccVerifyInclusion
// call, and the three distinct tier-2 data-states (error / failed / verified). The
// target URL is reflected ONLY inside the JSON data-island, never in the static body.
func TestVerifierConfiguredTargetWiresLiveVerification(t *testing.T) {
	const monitor = "https://monitor.iscc.id"
	const id = "ISCC:MAIGKSETI7MJ4EAB"
	rec := serveTarget(t, "monitor="+monitor+"&id="+id)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		`<script id="verify-target" type="application/json">`,
		`<script src="/_ds/wasm_exec.js">`,
		"/_ds/verify.wasm",
		"isccVerifyInclusion",
		`data-state="error"`,
		`data-state="failed"`,
		`data-state="verified"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("configured target missing live-wiring marker %q\n%s", want, body)
		}
	}

	// The target is reflected only inside the JSON data-island. The island content is
	// the only place the monitor URL / id appears, so it survives JSON-context-escaped
	// (the url.URL fragment/scheme guard in parseTarget already rejected an unusable
	// target before render).
	if !strings.Contains(body, `"monitor":"`+monitor+`"`) {
		t.Errorf("data-island missing the monitor target %q\n%s", monitor, body)
	}
	if !strings.Contains(body, `"id":"`+id+`"`) {
		t.Errorf("data-island missing the id target %q\n%s", id, body)
	}
}

// TestVerifierRejectsMalformedTarget asserts parseTarget fails CLOSED to the
// no-target baseline on an unusable ?monitor=: a non-http(s) scheme, a missing host,
// a fragment, or a missing id all render the baseline (no data-island, no loader),
// never a half-wired live page.
func TestVerifierRejectsMalformedTarget(t *testing.T) {
	for _, q := range []string{
		"monitor=ftp://x.example&id=ISCC:AAA",        // wrong scheme
		"monitor=https://&id=ISCC:AAA",               // empty host
		"monitor=https://x.example%23frag&id=A",      // fragment (encoded #)
		"monitor=https://monitor.iscc.id",            // no id
		"id=ISCC:AAA",                                // no monitor
		"monitor=https://x.example#frag&id=ISCC:AAA", // explicit fragment
	} {
		rec := serveTarget(t, q)
		if rec.Code != http.StatusOK {
			t.Fatalf("query %q: status = %d, want 200", q, rec.Code)
		}
		if strings.Contains(rec.Body.String(), `id="verify-target"`) {
			t.Errorf("query %q wired a live target but should fall back to the baseline", q)
		}
	}
}

// TestVerifierNoExternalCDN bans any external CDN origin in the body — same-origin /
// relative hrefs only — matching the sibling no-CDN SSR seam tests. The mockup's
// hashed jsDelivr-style token path + _ds_bundle.js are dropped (ADR-0010: the
// self-hosted/no-CDN constraint wins over the mockup). The test runs the no-target
// baseline, so the user-supplied ?monitor= URL (runtime data reflected only inside
// the JSON data-island) is absent — keeping the ban meaningful for stylesheet/script/
// img origins, its real purpose, without firing on a legitimate target URL.
func TestVerifierNoExternalCDN(t *testing.T) {
	rec := serve(t)
	body := rec.Body.String()

	for _, banned := range []string{"jsdelivr", "http://", "https://", "cdn."} {
		if strings.Contains(body, banned) {
			t.Errorf("body contains external CDN reference %q\n%s", banned, body)
		}
	}
}

// TestVerifierNonGET asserts a non-GET method is a 405 (the shared method gate at the
// top of Handler covers it).
func TestVerifierNonGET(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
