// Tests for the /metrics HTTP handler: a round trip over httptest returns HTTP
// 200, the Prometheus text-exposition Content-Type, and a body byte-equal to the
// registry's own renderer (metrics.Registry.String), so serving never diverges
// from the rendered bytes.
package metricshttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iscc/iscc-monitor/internal/metrics"
)

func TestHandlerServesRegistry(t *testing.T) {
	m := metrics.New()
	m.SetHubStatus(1, "verified")
	m.IncViolation(1, "fork")

	srv := httptest.NewServer(Handler(m))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET %s: %v", srv.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	// Assert the literal Prometheus text-exposition content type (not the package
	// constant) so a change to the served value is caught here.
	const wantCT = "text/plain; version=0.0.4; charset=utf-8"
	if got := resp.Header.Get("Content-Type"); got != wantCT {
		t.Errorf("Content-Type = %q, want %q", got, wantCT)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if got, want := string(body), m.String(); got != want {
		t.Errorf("body does not match renderer\n got: %q\nwant: %q", got, want)
	}
	// The pre-populated samples must actually appear, so the byte-equality above
	// is not vacuously comparing two empty renders.
	if want := m.String(); want == "" {
		t.Fatal("renderer produced empty output for a populated registry")
	}
}
