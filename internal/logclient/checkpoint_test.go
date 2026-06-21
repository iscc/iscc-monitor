// Tests for the networked checkpoint-fetch primitive. They drive FetchCheckpoint
// against an in-test fake Fetcher (asserting the canonical /log/checkpoint URL is
// derived from both scheme-ful and bare base URLs and that bytes pass through
// unchanged), confirm a 404 surfaces as errors.Is(err, os.ErrNotExist) through the
// %w wrapper and an empty base URL errors, and run one real-HTTP round-trip over
// httptest serving the captured sb0 checkpoint + did.json so fetch -> verify
// composes to StatusVerified without touching the live network.
package logclient

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestFetchCheckpoint(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
	}{
		{name: "with scheme", baseURL: "https://sb0.iscc.id"},
		{name: "bare host", baseURL: "sb0.iscc.id"},
	}
	want := []byte("sb0.iscc.id/log\n10183\nroot\n\n— sb0.iscc.id/log sig\n")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeFetcher{data: want}
			got, err := FetchCheckpoint(context.Background(), f, tc.baseURL)
			if err != nil {
				t.Fatalf("FetchCheckpoint(%q) error: %v", tc.baseURL, err)
			}
			const wantURL = "https://sb0.iscc.id/log/checkpoint"
			if f.gotURL != wantURL {
				t.Errorf("fetched URL = %q, want %q", f.gotURL, wantURL)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("bytes = %q, want %q (must pass through unchanged)", got, want)
			}
		})
	}
}

func TestFetchCheckpointNotFound(t *testing.T) {
	f := &fakeFetcher{err: os.ErrNotExist}
	_, err := FetchCheckpoint(context.Background(), f, "https://sb0.iscc.id")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want errors.Is(err, os.ErrNotExist)", err)
	}
}

func TestFetchCheckpointEmptyBaseURL(t *testing.T) {
	f := &fakeFetcher{data: []byte("unused")}
	_, err := FetchCheckpoint(context.Background(), f, "")
	if err == nil {
		t.Fatalf("FetchCheckpoint(\"\") = nil error, want error propagated from origin()")
	}
}

// TestFetchCheckpointOverHTTP drives the fetch over the real httpFetcher against an
// httptest server serving the sb0 checkpoint at /log/checkpoint, then feeds the
// fetched bytes straight into AcceptCheckpoint to prove fetch -> verify composes to
// StatusVerified with the sb0 fixture's tree size.
//
// AcceptCheckpoint re-derives the verifier-key origin from its baseURL, and the sb0
// checkpoint is signed under "sb0.iscc.id/log" — so verification is keyed to the sb0
// base URL with the captured did.json served by a fake Fetcher (the established
// accept-test pattern). A live httptest host (127.0.0.1:<port>) would derive a
// different origin and never match the fixture's signature, so the fetch is over
// real HTTP while the verify origin stays sb0.
func TestFetchCheckpointOverHTTP(t *testing.T) {
	checkpoint := readCheckpoint(t, "sb0.iscc.id_checkpoint")
	// did:web is always HTTPS (DocumentURL emits https://) and FetchCheckpoint
	// likewise, so serve TLS and use srv.Client(), which trusts the self-signed cert.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/log/checkpoint" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(checkpoint)
	}))
	defer srv.Close()

	raw, err := FetchCheckpoint(context.Background(), NewHTTPFetcher(srv.Client()), srv.URL)
	if err != nil {
		t.Fatalf("FetchCheckpoint over HTTP error: %v", err)
	}
	if !bytes.Equal(raw, checkpoint) {
		t.Errorf("fetched bytes differ from served fixture")
	}

	// The sb0 checkpoint is signed under sb0.iscc.id/log, so verify against that
	// origin with the captured did.json (no validity window) via a fake Fetcher.
	// observedAt is fixed inside sb0's validity window so the in-window key yields
	// StatusVerified rather than StatusRotated.
	didFetcher := &fakeFetcher{data: readFixture(t, "sb0.iscc.id_did.json")}
	observedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	status, info, _, err := AcceptCheckpoint(context.Background(), didFetcher, "https://sb0.iscc.id", raw, observedAt)
	if err != nil {
		t.Fatalf("AcceptCheckpoint error: %v", err)
	}
	if status != StatusVerified {
		t.Fatalf("status = %v, want StatusVerified", status)
	}
	if info.TreeSize != 10183 {
		t.Errorf("TreeSize = %d, want 10183 (sb0 fixture)", info.TreeSize)
	}
}
