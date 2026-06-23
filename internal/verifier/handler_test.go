// Golden HTTP-seam test for the standalone Independent Verification page (Surface C).
// It drives verifier.Handler over an httptest request and asserts the observable
// response — 200 text/html, the named regions of the Independent Verification mockup
// (the eyebrow, the .codes ↔ .id independence statement, the five verification-record
// step labels, the guided split-view mismatch alert + its "do not discard either"
// guidance), the shared no-CDN DS shell wiring (token/font links, self-hosted logo,
// the audited /_ds/verify.wasm artifact reference), and the no-CDN body ban matching
// the sibling SSR seam tests. The page is a single STATIC artifact: the ?monitor=&id=
// target is read CLIENT-side from location.search by the always-emitted loader, never
// by the handler, so every request renders the same bytes — the tests assert the
// always-present loader, that no query value is reflected into the body, and that the
// static body claims no un-run verdict. The oracle gate is N/A: no signature,
// RFC-6962, Merkle, did:web, fsck, or proof path — this is a pure static HTML render.
package verifier

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// scriptBlock matches an end-of-body <script>…</script> block (incl. its contents),
// used to isolate the DISPLAYED no-JS body from the always-emitted loader's JS string
// literals. The present-tense verdict copy ("verifying…", "Your (size, root) does not
// match") legitimately lives inside the loader's setVerdict() as runtime textContent,
// so the honesty contract is about what the no-JS render DISPLAYS, not raw substring
// presence anywhere in the file. The loader is the only multi-line script and runs
// no display until a valid target is found.
var scriptBlock = regexp.MustCompile(`(?s)<script\b[^>]*>.*?</script>`)

// displayedBody returns the body with every <script>…</script> block removed — the
// static markup a no-JS reader actually sees. The honesty assertions run against this
// so they catch an un-run verdict in DISPLAYED text without false-positiving on the
// loader's JS string literals (which set textContent only at runtime, post-target).
func displayedBody(body string) string {
	return scriptBlock.ReplaceAllString(body, "")
}

// serve renders the verifier page over a plain no-query GET request and returns the
// recorder. The page is a single static artifact rendered identically for every
// request (the ?monitor=&id= target is read CLIENT-side from location.search by the
// loader, never by the handler), so this one render is the whole observable surface.
func serve(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
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
	// logo img, the DS font tokens, and the audited WASM artifacts the always-emitted
	// client-side loader instantiates.
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
	// static no-JS list rows. The fourth/fifth describe what the in-browser WASM run
	// ACTUALLY does (rebuild the committed root + confirm the id-binding) — it does NOT
	// list a did:web signature check it never runs (see TestVerifierDoesNotClaimSignatureCheck).
	for _, want := range []string{
		"Fetch proof bundle from the monitor",
		"Recompute the leaf hash from the record bytes",
		"Walk the Merkle inclusion path",
		"Rebuild the root and match the committed checkpoint root",
		"Confirm the record commits the requested ISCC-ID",
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

// TestVerifierDoesNotClaimSignatureCheck mutation-proves the copy-honesty fix: the
// in-browser WASM checker (isccVerifyInclusion) re-runs ONLY the RFC-6962 inclusion
// math and the id-binding — it does NOT fetch a did:web document or verify the
// checkpoint signature. So the served page must not (1) list "Check the signature
// against the hub's did:web key" as a step it runs, nor (2) claim in its `verified`
// verdict copy that the browser re-verified a "hub-signed checkpoint root". The
// `verified` copy must instead assert only the inclusion / accepted-root + id-binding
// check the WASM actually ran. Reverting the template (re-adding the did:web step or
// restoring "hub-signed checkpoint root" to the verdict) makes this test FAIL.
func TestVerifierDoesNotClaimSignatureCheck(t *testing.T) {
	body := serve(t).Body.String()

	// (1) The un-run did:web signature STEP is gone.
	if strings.Contains(body, "Check the signature against the hub's did:web key") {
		t.Errorf("body lists an in-browser did:web signature step the WASM never runs\n%s", body)
	}

	// (2) The verified-verdict copy must not claim the browser re-verified a
	// "hub-signed checkpoint root" — the WASM rebuilds the committed/accepted root from
	// the inclusion proof; it does not validate the root was hub-signed.
	verified := verifiedVerdictLine(t, body)
	if strings.Contains(verified, "hub-signed") {
		t.Errorf("verified-verdict copy claims a hub-signed-root check the WASM did not run: %q", verified)
	}

	// (3) The verified-verdict copy asserts only what the WASM ran: the inclusion proof
	// against the accepted root (matching the certificate's already-honest sibling copy).
	if !strings.Contains(verified, "re-verified this inclusion proof against the accepted root") {
		t.Errorf("verified-verdict copy must claim only the inclusion / accepted-root check: %q", verified)
	}
}

// verifiedVerdictLine returns the single setVerdict("verified", …) call's source line
// from the rendered body, so the honesty assertions inspect exactly the success-copy
// string (not the in-progress or failed strings that legitimately differ).
func verifiedVerdictLine(t *testing.T, body string) string {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, `setVerdict("verified"`) {
			return line
		}
	}
	t.Fatalf(`no setVerdict("verified", …) line found in body:\n%s`, body)
	return ""
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

// TestVerifierStaticBodyAlwaysCarriesLoader asserts the static-artifact contract: a
// plain no-query GET ALWAYS carries the client-side live-verification loader — the
// /_ds/wasm_exec.js + /_ds/verify.wasm scripts, the isccVerifyInclusion call, and the
// URLSearchParams(location.search) read that resolves the ?monitor=&id= target in the
// browser. Because GitHub Pages serves the one pre-generated index.html byte-for-byte
// for every path, the loader must be present on every render so that loading the same
// artifact at /?monitor=…&id=… runs the WASM verdict client-side.
func TestVerifierStaticBodyAlwaysCarriesLoader(t *testing.T) {
	rec := serve(t)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		`<script src="/_ds/wasm_exec.js">`,
		"/_ds/verify.wasm",
		"isccVerifyInclusion",
		"URLSearchParams",
		"location.search",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("static body missing always-emitted loader marker %q\n%s", want, body)
		}
	}
}

// TestVerifierNoServerSideTarget asserts the handler reflects NO ?monitor=/?id= into
// the body and emits no server-rendered data-island: the target is resolved entirely
// CLIENT-side from location.search, so a query-bearing request renders byte-for-byte
// the same artifact as a plain one (the static-deployment contract). This replaces the
// former server-side parse-and-reflect contract (parseTarget / the verify-target
// island), which no longer exists.
func TestVerifierNoServerSideTarget(t *testing.T) {
	const monitor = "https://monitor.iscc.id"
	const id = "ISCC:MAIGKSETI7MJ4EAB"
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?monitor="+monitor+"&id="+id, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	// No server-emitted data-island remains.
	if strings.Contains(body, `id="verify-target"`) {
		t.Errorf("body unexpectedly carries a server-emitted data-island\n%s", body)
	}
	// The handler does not reflect the query into the body.
	for _, leaked := range []string{monitor, id} {
		if strings.Contains(body, leaked) {
			t.Errorf("body reflected query value %q (target must be read client-side, not server-rendered)\n%s", leaked, body)
		}
	}
	// The query-bearing render is byte-identical to the plain baseline render.
	if body != serve(t).Body.String() {
		t.Errorf("query-bearing render differs from the plain baseline render (the static artifact must be identical for every path)")
	}
}

// TestVerifierNoTargetBaselineIsHonest asserts the DISPLAYED no-JS body asserts NO
// un-run verdict: the markup a no-JS reader sees (the body with the loader's <script>
// blocks stripped) must NOT contain the present-tense negative-verdict assertion "Your
// (size, root) does not match" nor the "verifying…" in-progress copy. Those strings
// legitimately live inside the loader's setVerdict() as runtime textContent (set only
// after a valid target is found and a real `failed`/in-progress state runs) — moving
// them OUT of the static markup and into the JS is exactly what keeps the static
// artifact honest. The honest no-verdict markers ("not yet run", "no verdict is
// claimed", the illustrative mismatch example) still render in the displayed body.
// This is the honesty fix the advance mutation-proves: it keeps the open `normal`
// honesty fix intact under the new client-side contract.
func TestVerifierNoTargetBaselineIsHonest(t *testing.T) {
	rec := serve(t)
	shown := displayedBody(rec.Body.String())

	// The un-run present-tense verdicts must be absent from the DISPLAYED body (they
	// are set only by the loader's JS, after a valid target is found).
	for _, banned := range []string{
		"Your (size, root) does not match",
		"verifying in your browser…",
		"Re-verifying this inclusion proof in your browser",
	} {
		if strings.Contains(shown, banned) {
			t.Errorf("displayed (no-JS) body asserts an un-run verdict (found present-tense %q)\n%s", banned, shown)
		}
	}
	// The honest no-verdict markers still render in the displayed body: the run-label
	// and the explicit no-verdict-claimed copy, plus the illustrative mismatch example.
	for _, want := range []string{
		"not yet run",
		"no verdict is claimed",
		"Illustrative — what a real mismatch shows",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("displayed (no-JS) body missing honest marker %q\n%s", want, shown)
		}
	}
}

// TestVerifierLoaderHasThreeDistinctStates asserts the loader keeps the three render
// states strictly distinct (error / failed / verified): the verdict CSS keys all
// three, and only `failed` is the split-view signal that lifts the guided mismatch
// alert. A transport/parse fault must surface as `error`, never masquerade as a
// mismatch.
func TestVerifierLoaderHasThreeDistinctStates(t *testing.T) {
	body := serve(t).Body.String()

	for _, want := range []string{
		`data-state="error"`,
		`data-state="failed"`,
		`data-state="verified"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("static body missing distinct verdict state %q\n%s", want, body)
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

// TestVerifierReadTargetReturnsNormalizedURL asserts readTarget returns the WHATWG-
// normalized monitor URL (the parsed u.href), not the raw ?monitor= query string, so
// an opaque-scheme form (https:example.com) flows downstream as https://example.com/
// — the guard's own parsed-and-validated URL is the single source of truth, closing
// the Surface-C readTarget normal where the JS port was more permissive than the Go
// parseTarget. The assertion is mutation-provable: reverting the return to the raw
// string ({ monitor: monitor, id: id }) makes this test FAIL on both the positive and
// the negative check.
func TestVerifierReadTargetReturnsNormalizedURL(t *testing.T) {
	body := serve(t).Body.String()

	if !strings.Contains(body, "return { monitor: u.href, id: id };") {
		t.Errorf("readTarget must return the normalized parsed URL (monitor: u.href)\n%s", body)
	}
	if strings.Contains(body, "return { monitor: monitor, id: id };") {
		t.Errorf("readTarget must NOT return the raw monitor query string (monitor: monitor)\n%s", body)
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
