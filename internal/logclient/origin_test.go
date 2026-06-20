// Test for origin derivation: a hub base URL maps to its <domain>/log origin
// regardless of scheme or trailing slash, and never the bare domain.
package logclient

import "testing"

func TestOrigin(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "sb0 https", baseURL: "https://sb0.iscc.id", want: "sb0.iscc.id/log"},
		{name: "sb1 https", baseURL: "https://sb1.amlet.id", want: "sb1.amlet.id/log"},
		{name: "trailing slash", baseURL: "https://sb0.iscc.id/", want: "sb0.iscc.id/log"},
		{name: "http scheme", baseURL: "http://sb0.iscc.id", want: "sb0.iscc.id/log"},
		{name: "no scheme", baseURL: "sb0.iscc.id", want: "sb0.iscc.id/log"},
		{name: "host with port", baseURL: "https://localhost:8443", want: "localhost:8443/log"},
		{name: "surrounding whitespace", baseURL: "  https://sb0.iscc.id  ", want: "sb0.iscc.id/log"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := origin(tc.baseURL)
			if err != nil {
				t.Fatalf("origin(%q) error: %v", tc.baseURL, err)
			}
			if got != tc.want {
				t.Errorf("origin(%q) = %q, want %q", tc.baseURL, got, tc.want)
			}
		})
	}
}

func TestOriginErrors(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
	}{
		{name: "empty", baseURL: ""},
		{name: "whitespace only", baseURL: "   "},
		{name: "scheme only", baseURL: "https://"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := origin(tc.baseURL); err == nil {
				t.Errorf("origin(%q) = nil error, want error", tc.baseURL)
			}
		})
	}
}
