// Package version exposes the monitor's build-provenance string over net/http as
// a tiny HTTP leaf, mirroring internal/metricshttp and internal/healthz. The
// exported Version var defaults to "dev" and is the -ldflags -X injection target:
// a release build overrides it with the git short SHA via
//
//	go build -ldflags "-X github.com/iscc/iscc-monitor/internal/version.Version=<sha>"
//
// so an operator (and a CI smoke check) can confirm exactly which build is live by
// reading GET /version. It is a build-time stamp, not a runtime config value, so it
// is deliberately NOT an internal/config key or an ISCC_MONITOR_* env var.
//
// The oracle/conformance gate is N/A here: this is pure HTTP wiring plus a build
// stamp, touching no signature, RFC-6962, Merkle, did:web, fsck, or proof path.
package version

import (
	"fmt"
	"net/http"
)

// Version is the build-provenance string the binary reports at GET /version. It
// defaults to "dev" for an un-stamped build (the quality gate's plain
// `go build ./...`); a release build overrides it at link time via
// `-ldflags "-X github.com/iscc/iscc-monitor/internal/version.Version=<sha>"`. It
// must stay a plain string var initialized to a constant — -X only overrides such
// a var, never a const or a computed/concatenated value.
var Version = "dev"

// Handler returns an http.Handler that reports the build version. On GET it sets
// Content-Type: application/json, writes 200, then writes {"version":"<Version>"}.
// The body is built with %q so a stamped SHA (or any value) stays valid JSON even
// if it ever contains a quote. Any other method is 405, matching internal/healthz
// for consistency across the leaf endpoints. The post-status write error is dropped
// deliberately (documented convention): a derived body cannot fail for content
// reasons after WriteHeader, so the only failure mode is a broken client connection,
// which a second status cannot fix.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"version":%q}`, Version)
	})
}
