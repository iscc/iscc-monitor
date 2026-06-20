// Tests for the did:web verifier-key resolver. They drive ResolveVerifierKey
// against the captured sb0/sb1 did.json fixtures through an in-test fake Fetcher
// (asserting the byte-exact golden verifier keys and the resolved did.json URL),
// confirm every fetch/parse/derive failure surfaces as errors.Is(err,
// ErrUnresolvable), and exercise the real-HTTP httpFetcher over net/http/httptest
// so the live network is never touched.
package logclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// fakeFetcher returns canned bytes (or err) and records the URL it was asked for.
type fakeFetcher struct {
	data   []byte
	err    error
	gotURL string
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.gotURL = url
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

// readFixture loads a captured did.json fixture from testdata/.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}
	return data
}

func TestResolveVerifierKey(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		fixture string
		wantURL string
		wantKey string
	}{
		{
			name:    "sb0",
			baseURL: "https://sb0.iscc.id",
			fixture: "sb0.iscc.id_did.json",
			wantURL: "https://sb0.iscc.id/.well-known/did.json",
			wantKey: "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5",
		},
		{
			name:    "sb1",
			baseURL: "https://sb1.amlet.id",
			fixture: "sb1.amlet.id_did.json",
			wantURL: "https://sb1.amlet.id/.well-known/did.json",
			wantKey: "sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeFetcher{data: readFixture(t, tc.fixture)}
			key, didKey, err := ResolveVerifierKey(context.Background(), f, tc.baseURL)
			if err != nil {
				t.Fatalf("ResolveVerifierKey(%q) error: %v", tc.baseURL, err)
			}
			if key != tc.wantKey {
				t.Errorf("verifier key = %q, want %q", key, tc.wantKey)
			}
			if f.gotURL != tc.wantURL {
				t.Errorf("fetched URL = %q, want %q", f.gotURL, tc.wantURL)
			}
			if len(didKey.PublicKey) != 32 {
				t.Errorf("DIDKey.PublicKey is %d bytes, want 32", len(didKey.PublicKey))
			}
		})
	}
}

func TestResolveVerifierKeyUnresolvable(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		fetcher *fakeFetcher
	}{
		{
			name:    "fetch error",
			baseURL: "https://sb0.iscc.id",
			fetcher: &fakeFetcher{err: errors.New("connection refused")},
		},
		{
			name:    "not found",
			baseURL: "https://sb0.iscc.id",
			fetcher: &fakeFetcher{err: os.ErrNotExist},
		},
		{
			name:    "malformed JSON",
			baseURL: "https://sb0.iscc.id",
			fetcher: &fakeFetcher{data: []byte(`{"id": "did:web:sb0.iscc.id", `)},
		},
		{
			name:    "valid JSON, no assertionMethod",
			baseURL: "https://sb0.iscc.id",
			fetcher: &fakeFetcher{data: []byte(`{"id": "did:web:sb0.iscc.id"}`)},
		},
		{
			name:    "empty base URL",
			baseURL: "",
			fetcher: &fakeFetcher{data: []byte("unused")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ResolveVerifierKey(context.Background(), tc.fetcher, tc.baseURL)
			if !errors.Is(err, ErrUnresolvable) {
				t.Errorf("error = %v, want errors.Is(err, ErrUnresolvable)", err)
			}
		})
	}
}

// TestResolveVerifierKeyOverHTTP drives the resolver through the real httpFetcher
// against an httptest server serving the sb0 fixture at /.well-known/did.json.
func TestResolveVerifierKeyOverHTTP(t *testing.T) {
	fixture := readFixture(t, "sb0.iscc.id_did.json")
	// did:web is always HTTPS (DocumentURL emits https://), so serve TLS and use
	// srv.Client(), which trusts the test server's self-signed certificate.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/did.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	// The fixture's origin is sb0.iscc.id/log; the verifier key embeds that
	// origin regardless of the host we actually fetch from, so the golden holds.
	key, _, err := ResolveVerifierKey(context.Background(), NewHTTPFetcher(srv.Client()), srv.URL)
	if err != nil {
		t.Fatalf("ResolveVerifierKey over HTTP error: %v", err)
	}
	host := srv.Listener.Addr().String()
	want := host + "/log"
	if got := key[:len(want)]; got != want {
		t.Errorf("verifier key origin prefix = %q, want %q (full key %q)", got, want, key)
	}
}

// TestHTTPFetcherNotFound checks the 404 -> os.ErrNotExist contract and that the
// resolver wraps it as ErrUnresolvable.
func TestHTTPFetcherNotFound(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := NewHTTPFetcher(srv.Client()).Fetch(context.Background(), srv.URL+"/.well-known/did.json")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Fetch 404 error = %v, want errors.Is(err, os.ErrNotExist)", err)
	}

	_, _, rerr := ResolveVerifierKey(context.Background(), NewHTTPFetcher(srv.Client()), srv.URL)
	if !errors.Is(rerr, ErrUnresolvable) {
		t.Errorf("ResolveVerifierKey over 404 = %v, want errors.Is(err, ErrUnresolvable)", rerr)
	}
}
