// Package metricshttp exposes the pure metrics leaf over net/http: it wraps a
// metrics.Registry in an http.Handler that renders the Prometheus
// text-exposition format on GET /metrics. It exists as a separate package
// precisely so net/http never enters internal/metrics, which must stay a
// stdlib-only, WASM-shareable leaf (the proof/verify-style purity rule). The
// binary mounts this handler; internal/metrics keeps only the renderer.
package metricshttp

import (
	"net/http"

	"github.com/iscc/iscc-monitor/internal/metrics"
)

// contentType is the Prometheus text-exposition content type (version 0.0.4),
// the value a scraper expects on a /metrics response.
const contentType = "text/plain; version=0.0.4; charset=utf-8"

// Handler returns an http.Handler that renders r in Prometheus text-exposition
// format. It sets the Content-Type header before writing the body and then calls
// r.WriteText, so the served bytes match the renderer exactly.
//
// r must be non-nil (the binary always passes a real metrics.New() registry);
// there is no nil-guard branch.
func Handler(r *metrics.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		// WriteText streams directly to the ResponseWriter, which sends a 200 on
		// the first write; a mid-write error cannot un-send that status, so it is
		// only the renderer's own writer error and there is nothing left to
		// recover. Drop it deliberately rather than write a misleading second
		// status — the only failure mode here is a broken client connection.
		_ = r.WriteText(w)
	})
}
