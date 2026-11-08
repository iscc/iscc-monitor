// Tests for the OpenAPI document handler at the HTTP seam, plus the two committed
// artifacts' agreement. They drive Handler over an httptest.ResponseRecorder and
// assert the observable response: GET /openapi.json -> 200 application/json with a
// body byte-equal to the embedded JSON; GET /openapi.yaml -> 200 application/yaml
// byte-equal to the embedded YAML; a non-GET -> 405; an unknown path -> 404; an
// If-None-Match echoing the served ETag -> 304. They also assert the embedded JSON
// and YAML declare the IDENTICAL path set (so the two committed artifacts cannot
// drift) and that the document is a valid OpenAPI 3.1 skeleton (openapi: 3.1.x, a
// non-empty paths object).
package openapi

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

// req drives Handler with method+path (and optional headers) and returns the recorder.
func req(t *testing.T, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	Handler().ServeHTTP(rec, r)
	return rec
}

func TestOpenAPIServesJSONVerbatim(t *testing.T) {
	rec := req(t, http.MethodGet, JSONPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
	}
	if got := rec.Body.Bytes(); !equalBytes(got, JSON) {
		t.Errorf("body is not byte-equal to embedded JSON (got %d bytes, want %d)", len(got), len(JSON))
	}
}

func TestOpenAPIServesYAMLVerbatim(t *testing.T) {
	rec := req(t, http.MethodGet, YAMLPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != contentTypeYAML {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeYAML)
	}
	if got := rec.Body.Bytes(); !equalBytes(got, YAML) {
		t.Errorf("body is not byte-equal to embedded YAML (got %d bytes, want %d)", len(got), len(YAML))
	}
}

func TestOpenAPIRejectsNonGet(t *testing.T) {
	rec := req(t, http.MethodPost, JSONPath, nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestOpenAPIUnknownPathIs404(t *testing.T) {
	rec := req(t, http.MethodGet, "/openapi.txt", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestOpenAPIConditionalGet304(t *testing.T) {
	etag := fmt.Sprintf("\"%x\"", sha256.Sum256(JSON))
	rec := req(t, http.MethodGet, JSONPath, map[string]string{"If-None-Match": etag})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304 carried a body of %d bytes, want empty", rec.Body.Len())
	}
	if rec.Header().Get("ETag") != etag {
		t.Errorf("304 ETag = %q, want %q", rec.Header().Get("ETag"), etag)
	}
}

// TestOpenAPIDocsAgree pins that the two committed artifacts declare the IDENTICAL
// set of paths. JSON is valid YAML, so yaml.Unmarshal reads both; comparing the
// parsed `paths` key sets fails the gate the moment the JSON and YAML drift from each
// other (e.g. a path edited in one but not the other).
func TestOpenAPIDocsAgree(t *testing.T) {
	jsonPaths := parsePaths(JSON)
	yamlPaths := parsePaths(YAML)
	if len(jsonPaths) == 0 {
		t.Fatal("embedded JSON declares no paths")
	}
	if !reflect.DeepEqual(jsonPaths, yamlPaths) {
		t.Errorf("JSON and YAML declare different path sets:\n json = %v\n yaml = %v", jsonPaths, yamlPaths)
	}
	// DeclaredPaths (the drift-test entry point) reads the YAML and must agree too.
	if !reflect.DeepEqual(DeclaredPaths(), yamlPaths) {
		t.Errorf("DeclaredPaths = %v, want %v", DeclaredPaths(), yamlPaths)
	}
}

// TestOpenAPIDocumentIsValid3_1 asserts the served JSON parses as an OpenAPI 3.1
// skeleton: a 3.1.x `openapi` field and a non-empty `paths` object.
func TestOpenAPIDocumentIsValid3_1(t *testing.T) {
	var doc struct {
		OpenAPI string         `yaml:"openapi"`
		Paths   map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(JSON, &doc); err != nil {
		t.Fatalf("served JSON does not parse: %v", err)
	}
	if len(doc.OpenAPI) < 4 || doc.OpenAPI[:4] != "3.1." {
		t.Errorf("openapi = %q, want a 3.1.x version", doc.OpenAPI)
	}
	if len(doc.Paths) == 0 {
		t.Error("paths object is empty, want the machine surface")
	}
}

// TestVerifyForMeFlaggedWeaker is the honesty golden: the served document MUST flag
// the verify-for-me path as the explicitly weaker, non-authoritative tier-1 path, so
// the trust framing can never be silently dropped from the contract (ADR-0014 §1).
func TestVerifyForMeFlaggedWeaker(t *testing.T) {
	for _, phrase := range []string{"weaker", "non-authoritative", "the monitor reports"} {
		if !containsFold(JSON, phrase) {
			t.Errorf("OpenAPI JSON does not flag verify-for-me with %q", phrase)
		}
		if !containsFold(YAML, phrase) {
			t.Errorf("OpenAPI YAML does not flag verify-for-me with %q", phrase)
		}
	}
}

// equalBytes reports byte equality without pulling bytes into the import set for a
// one-liner the test already needs elsewhere.
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// containsFold reports whether body contains phrase case-insensitively over ASCII.
func containsFold(body []byte, phrase string) bool {
	lower := make([]byte, len(body))
	for i, c := range body {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		lower[i] = c
	}
	p := []byte(phrase)
	for i, c := range p {
		if c >= 'A' && c <= 'Z' {
			p[i] = c + 'a' - 'A'
		}
	}
	return indexOf(lower, p) >= 0
}

// indexOf returns the first index of sub in s, or -1.
func indexOf(s, sub []byte) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := range sub {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
