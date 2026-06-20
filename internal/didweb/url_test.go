// Golden + error tests for the pure did:web identifier -> did.json URL mapping.
// The two live-hub cases are the load-bearing goldens the follower's
// outbound-fetch seam depends on; the path/port cases exercise the W3C did:web
// method rules ported in url.go.
package didweb

import "testing"

func TestDocumentURL(t *testing.T) {
	cases := []struct {
		name string
		did  string
		want string
	}{
		{
			name: "sb0 live hub, no path no port",
			did:  "did:web:sb0.iscc.id",
			want: "https://sb0.iscc.id/.well-known/did.json",
		},
		{
			name: "sb1 live hub, no path no port",
			did:  "did:web:sb1.amlet.id",
			want: "https://sb1.amlet.id/.well-known/did.json",
		},
		{
			name: "percent-encoded port plus path segments",
			did:  "did:web:example.com%3A3000:user:alice",
			want: "https://example.com:3000/user/alice/did.json",
		},
		{
			name: "single path segment, no port",
			did:  "did:web:example.com:user:alice",
			want: "https://example.com/user/alice/did.json",
		},
		{
			name: "host with port, no path",
			did:  "did:web:localhost%3A8080",
			want: "https://localhost:8080/.well-known/did.json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DocumentURL(tc.did)
			if err != nil {
				t.Fatalf("DocumentURL(%q) error: %v", tc.did, err)
			}
			if got != tc.want {
				t.Errorf("DocumentURL(%q) = %q, want %q", tc.did, got, tc.want)
			}
		})
	}
}

func TestDocumentURLErrors(t *testing.T) {
	cases := []struct {
		name string
		did  string
	}{
		{name: "empty input", did: ""},
		{name: "wrong method did:key", did: "did:key:z6Mkabc"},
		{name: "bare prefix, empty method-specific id", did: "did:web:"},
		{name: "empty host segment", did: "did:web::user:alice"},
		{name: "invalid percent-encoding", did: "did:web:example.com%zz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DocumentURL(tc.did); err == nil {
				t.Errorf("DocumentURL(%q) = nil error, want error", tc.did)
			}
		})
	}
}
