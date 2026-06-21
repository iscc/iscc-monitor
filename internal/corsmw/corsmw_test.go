// Tests for the CORS middleware: a GET response carries
// Access-Control-Allow-Origin: * and the inner handler's status/body are
// delegated through unchanged; an OPTIONS preflight returns 204 with the CORS
// headers and never reaches the inner handler; and the Allow-Origin header is
// present even when the inner handler writes a non-200 (an http.Error 404 still
// carries it).
package corsmw

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	const (
		allowOrigin  = "Access-Control-Allow-Origin"
		allowMethods = "Access-Control-Allow-Methods"
		body200      = "inner body"
		body404      = "not found here"
	)

	tests := []struct {
		name        string
		method      string
		inner       http.Handler
		wantStatus  int
		wantBody    string
		wantInner   bool // whether the inner handler should be invoked
		wantMethods bool // whether Access-Control-Allow-Methods must contain GET
	}{
		{
			name:   "GET delegates and carries Allow-Origin",
			method: http.MethodGet,
			inner: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, body200)
			}),
			wantStatus: http.StatusOK,
			wantBody:   body200,
			wantInner:  true,
		},
		{
			name:   "OPTIONS preflight short-circuits to 204",
			method: http.MethodOptions,
			inner: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("inner handler must not run for an OPTIONS preflight")
			}),
			wantStatus:  http.StatusNoContent,
			wantBody:    "",
			wantInner:   false,
			wantMethods: true,
		},
		{
			name:   "inner non-200 still carries Allow-Origin",
			method: http.MethodGet,
			inner: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, body404, http.StatusNotFound)
			}),
			wantStatus: http.StatusNotFound,
			wantBody:   body404 + "\n", // http.Error appends a trailing newline
			wantInner:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ran := false
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ran = true
				tt.inner.ServeHTTP(w, r)
			})

			rec := httptest.NewRecorder()
			Handler(inner).ServeHTTP(rec, httptest.NewRequest(tt.method, "/anything", nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			// The CORS header must be present on every response regardless of the
			// inner status.
			if got := rec.Header().Get(allowOrigin); got != "*" {
				t.Errorf("%s = %q, want %q", allowOrigin, got, "*")
			}
			if ran != tt.wantInner {
				t.Errorf("inner handler invoked = %v, want %v", ran, tt.wantInner)
			}
			if tt.wantMethods {
				if got := rec.Header().Get(allowMethods); !strings.Contains(got, http.MethodGet) {
					t.Errorf("%s = %q, want it to contain %q", allowMethods, got, http.MethodGet)
				}
			}
		})
	}
}
