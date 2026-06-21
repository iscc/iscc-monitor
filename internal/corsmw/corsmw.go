// Package corsmw is the monitor's single CORS policy leaf: one middleware that
// wraps the public mux so every served surface (/metrics, /healthz, /inclusion,
// /consistency, /entries, and the raw tlog-tiles mirror) answers cross-origin
// browser GETs uniformly. The policy lives here in exactly one place and rides
// every route via the single buildMux wrap, never duplicated per handler — that
// is why CORS is intentionally out of scope in the proofserve/tilesserve
// handlers and belongs in this leaf instead. The monitor serves public,
// credential-free, read-only data, so a wildcard origin is the correct and
// simplest policy. It imports only net/http, keeping the leaf trivially correct
// and import-clean.
package corsmw

import "net/http"

// Handler wraps next so every response carries Access-Control-Allow-Origin: *
// and OPTIONS preflights succeed with 204 No Content.
//
// The Allow-Origin header is set BEFORE delegating to next, so it lands on every
// response — 200, 404, 405, and 500 alike — even when the inner handler calls
// w.WriteHeader (via http.Error or the first body write), after which header
// mutations are ignored.
//
// An OPTIONS request is treated as a preflight: it also advertises the allowed
// methods and headers, replies 204, and returns WITHOUT calling next. The inner
// handlers only speak GET and would 405 an OPTIONS, so the preflight is
// short-circuited here to let the browser proceed to the real GET. Every other
// method flows through to next unchanged, so a non-GET non-OPTIONS request still
// reaches the inner handler and gets its existing 405.
func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
