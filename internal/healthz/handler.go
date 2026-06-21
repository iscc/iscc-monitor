// Package healthz exposes the monitor's liveness + store-readiness probe over
// net/http as a tiny leaf package. It mirrors internal/metricshttp: it keeps
// net/http out of internal/store by depending only on a 1-method Pinger
// interface (which *store.Store satisfies structurally), so the dependency
// direction is binary -> healthz, never the reverse, and healthz imports neither
// store nor database/sql.
//
// The oracle/conformance gate is N/A here: this is pure HTTP wiring plus a DB
// ping, touching no signature, RFC-6962, Merkle, did:web, fsck, or proof path.
package healthz

import (
	"context"
	"io"
	"net/http"
)

// Pinger reports whether a backing resource (the SQLite store) is reachable. It
// is a 1-method interface so healthz depends on neither store nor database/sql;
// *store.Store satisfies it structurally via its Ping method.
type Pinger interface {
	Ping(ctx context.Context) error
}

// bodyOK and bodyUnavailable are the fixed JSON bodies. They are byte literals,
// not marshalled from dynamic data, so there is no marshal-failure branch — the
// post-status write error is dropped deliberately (it can only signal a broken
// client connection, which a second status cannot fix).
const (
	bodyOK          = `{"status":"ok"}`
	bodyUnavailable = `{"status":"unavailable"}`
)

// Handler returns an http.Handler that reports process liveness and store
// readiness. On GET it calls p.Ping(r.Context()): a nil error writes 200 with
// {"status":"ok"}, a non-nil error writes 503 with {"status":"unavailable"}. Any
// other method is 405. The status is chosen up front and WriteHeader is called
// before the body so the error case is never masked by an implicit 200, and the
// Content-Type is set before WriteHeader.
//
// p must be non-nil (the binary always passes the real store); there is no
// nil-guard branch.
func Handler(p Pinger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		status, body := http.StatusOK, bodyOK
		if err := p.Ping(r.Context()); err != nil {
			status, body = http.StatusServiceUnavailable, bodyUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
}
