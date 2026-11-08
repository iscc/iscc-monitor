// Tests for the networked tile/entry-bundle fetch primitives. They drive
// FetchTile and FetchEntryBundle against the in-test fake Fetcher (reused from
// didresolve_test.go), asserting the exact canonical tlog-tiles URL is built from
// the shared origin() + tiles.TilePath/EntriesPath for representative coords (full
// and partial, both tile and bundle), that bytes pass through unchanged, that a
// 404's os.ErrNotExist survives the %w wrapper via errors.Is, and that an empty
// base URL errors out of origin(). The golden URLs anchor on the same path strings
// as internal/tiles/layout_test.go (tile/1/000, tile/0/000.p/255,
// tile/entries/255, tile/entries/000.p/8), prefixed with the sb0 log origin.
package logclient

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
)

func TestFetchTile(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		level   uint64
		index   uint64
		p       uint8
		wantURL string
	}{
		{
			name:    "full tile level 1, scheme base",
			baseURL: "https://sb0.iscc.id",
			level:   1, index: 0, p: 0,
			wantURL: "https://sb0.iscc.id/log/tile/1/000",
		},
		{
			name:    "full tile level 1, bare base",
			baseURL: "sb0.iscc.id",
			level:   1, index: 0, p: 0,
			wantURL: "https://sb0.iscc.id/log/tile/1/000",
		},
		{
			name:    "partial tile width 255",
			baseURL: "https://sb0.iscc.id",
			level:   0, index: 0, p: 255,
			wantURL: "https://sb0.iscc.id/log/tile/0/000.p/255",
		},
	}
	want := []byte("tile-bytes-verbatim")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeFetcher{data: want}
			got, err := FetchTile(context.Background(), f, tc.baseURL, tc.level, tc.index, tc.p)
			if err != nil {
				t.Fatalf("FetchTile error: %v", err)
			}
			if f.gotURL != tc.wantURL {
				t.Errorf("fetched URL = %q, want %q", f.gotURL, tc.wantURL)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("bytes = %q, want %q (must pass through unchanged)", got, want)
			}
		})
	}
}

func TestFetchEntryBundle(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		index   uint64
		p       uint8
		wantURL string
	}{
		{
			name:    "full bundle index 255, scheme base",
			baseURL: "https://sb0.iscc.id",
			index:   255, p: 0,
			wantURL: "https://sb0.iscc.id/log/tile/entries/255",
		},
		{
			name:    "full bundle index 255, bare base",
			baseURL: "sb0.iscc.id",
			index:   255, p: 0,
			wantURL: "https://sb0.iscc.id/log/tile/entries/255",
		},
		{
			name:    "partial bundle width 8",
			baseURL: "https://sb0.iscc.id",
			index:   0, p: 8,
			wantURL: "https://sb0.iscc.id/log/tile/entries/000.p/8",
		},
	}
	want := []byte("bundle-bytes-verbatim")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeFetcher{data: want}
			got, err := FetchEntryBundle(context.Background(), f, tc.baseURL, tc.index, tc.p)
			if err != nil {
				t.Fatalf("FetchEntryBundle error: %v", err)
			}
			if f.gotURL != tc.wantURL {
				t.Errorf("fetched URL = %q, want %q", f.gotURL, tc.wantURL)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("bytes = %q, want %q (must pass through unchanged)", got, want)
			}
		})
	}
}

func TestFetchTileNotFound(t *testing.T) {
	f := &fakeFetcher{err: os.ErrNotExist}
	_, err := FetchTile(context.Background(), f, "https://sb0.iscc.id", 1, 0, 0)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want errors.Is(err, os.ErrNotExist)", err)
	}
}

func TestFetchEntryBundleNotFound(t *testing.T) {
	f := &fakeFetcher{err: os.ErrNotExist}
	_, err := FetchEntryBundle(context.Background(), f, "https://sb0.iscc.id", 255, 0)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want errors.Is(err, os.ErrNotExist)", err)
	}
}

func TestFetchTileEmptyBaseURL(t *testing.T) {
	f := &fakeFetcher{data: []byte("unused")}
	_, err := FetchTile(context.Background(), f, "", 1, 0, 0)
	if err == nil {
		t.Fatalf("FetchTile(\"\") = nil error, want error propagated from origin()")
	}
}

func TestFetchEntryBundleEmptyBaseURL(t *testing.T) {
	f := &fakeFetcher{data: []byte("unused")}
	_, err := FetchEntryBundle(context.Background(), f, "", 255, 0)
	if err == nil {
		t.Fatalf("FetchEntryBundle(\"\") = nil error, want error propagated from origin()")
	}
}
