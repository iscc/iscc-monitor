// Tests for the version HTTP leaf: GET /version drives version.Handler over
// httptest and asserts a 200 application/json body whose "version" field is
// non-empty and equals version.Version (the default "dev" or a -ldflags-injected
// SHA), and a non-GET method is rejected with 405 (matching internal/healthz).
package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestVersionHandlerGet proves GET /version returns 200, Content-Type
// application/json, and a JSON body with a non-empty "version" field equal to the
// package-level Version var (default "dev").
func TestVersionHandlerGet(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not valid JSON: %v", rec.Body.String(), err)
	}
	if body.Version == "" {
		t.Errorf("version field is empty, want a non-empty build string")
	}
	if body.Version != Version {
		t.Errorf("version field = %q, want %q (the package Version var)", body.Version, Version)
	}
}

// TestVersionHandlerRejectsNonGet proves a non-GET method is rejected with 405,
// matching the leaf-endpoint method gate in internal/healthz.
func TestVersionHandlerRejectsNonGet(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, httptest.NewRequest(method, "/version", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s status = %d, want 405", method, rec.Code)
		}
	}
}

// TestVersionDefaultIsDev proves the package default is "dev" — an un-stamped
// build (the quality gate's plain go build) serves "dev", so the -ldflags -X
// injection target var is correctly initialized to a constant.
func TestVersionDefaultIsDev(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version = %q, want %q (the un-stamped default; a stamped build overrides via -ldflags -X)", Version, "dev")
	}
}
