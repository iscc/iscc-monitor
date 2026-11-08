// Tests for the /healthz HTTP handler: a GET with a clean Pinger returns 200 +
// {"status":"ok"} and the JSON content type; a GET with a failing Pinger returns
// 503 + {"status":"unavailable"}; a non-GET returns 405. The Pinger seam is
// faked in-test so the handler is exercised without a real store, proving healthz
// depends on neither store nor database/sql.
package healthz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakePinger returns its configured err from Ping, letting a test drive both the
// healthy and unavailable branches without a real database.
type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func TestHealthzOK(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(fakePinger{err: nil}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Body.String(), `{"status":"ok"}`; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestHealthzUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(fakePinger{err: errors.New("store down")}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got, want := rec.Body.String(), `{"status":"unavailable"}`; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestHealthzMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(fakePinger{err: nil}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
