// Tests for the tilesserve.Handler — the canonical tlog-tiles read surface served
// from one hub's SQLiteFetcher. They seed a real store.Open(t.TempDir()) with one
// full tile, one partial tile, one entry bundle, and one checkpoint via the
// existing store writers, then drive the handler over httptest and assert on
// observable HTTP outputs only (status + body bytes), never on handler internals.
// Request URLs are built with tiles.TilePath / tiles.EntriesPath so the paths are
// tessera-canonical, not author-asserted.
package tilesserve_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/tiles"
	"github.com/iscc/iscc-monitor/internal/tilesserve"
)

// seeded holds the bytes written into the mirror so the test can assert exact
// byte-equality on what the handler serves back.
type seeded struct {
	fullTile   []byte
	partTile   []byte
	bundle     []byte
	checkpoint []byte
}

// newServer opens a fresh store, registers one hub, seeds the four artifacts, and
// returns an httptest server fronting tilesserve.Handler plus the seeded bytes.
func newServer(t *testing.T) (*httptest.Server, seeded) {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(t.TempDir() + "/x.db")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	hub, err := s.UpsertHub(ctx, "sb0.iscc.id", "sb0.iscc.id/log", "https://sb0.iscc.id")
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	sd := seeded{
		fullTile:   bytes.Repeat([]byte{0x11}, 8192),
		partTile:   bytes.Repeat([]byte{0x22}, 1408),
		bundle:     bytes.Repeat([]byte{0x33}, 512),
		checkpoint: []byte("sb0.iscc.id/log\n300\nrootbytes==\n\n— sb0 sig\n"),
	}
	now := time.Unix(1700000000, 0)
	// Full tile at (level 0, index 0), width 256.
	if err := s.RecordTile(ctx, hub, 0, 0, tiles.TileWidth, sd.fullTile, now); err != nil {
		t.Fatalf("RecordTile full: %v", err)
	}
	// Partial tile at (level 0, index 1), width 44 (the 300-leaf-tree leftover).
	if err := s.RecordTile(ctx, hub, 0, 1, 44, sd.partTile, now); err != nil {
		t.Fatalf("RecordTile partial: %v", err)
	}
	// One full entry bundle at index 0.
	if err := s.RecordEntryBundle(ctx, hub, 0, tiles.TileWidth, sd.bundle, now); err != nil {
		t.Fatalf("RecordEntryBundle: %v", err)
	}
	// One checkpoint at tree_size 300.
	if _, _, err := s.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hub, TreeSize: 300, Root: []byte("root"), Raw: sd.checkpoint, ObservedAt: now,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}

	srv := httptest.NewServer(tilesserve.Handler(store.SQLiteFetcher{Store: s, HubID: hub}))
	t.Cleanup(srv.Close)
	return srv, sd
}

// get issues a GET for path and returns the status and full body bytes.
func get(t *testing.T, base, path string) (int, []byte) {
	t.Helper()
	status, _, body := getWithHeader(t, base, path)
	return status, body
}

// getWithHeader issues a GET for path and returns the status, response header,
// and full body bytes — the sibling of get for asserting on response headers
// (e.g. Cache-Control) without changing get's body-only call sites.
func getWithHeader(t *testing.T, base, path string) (int, http.Header, []byte) {
	t.Helper()
	resp, err := http.Get(base + "/" + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", path, err)
	}
	return resp.StatusCode, resp.Header, body
}

// getWithIfNoneMatch issues a GET for path with an If-None-Match request header
// and returns the status, response header, and full body bytes — the seam for
// exercising the conditional-GET / 304 short-circuit.
func getWithIfNoneMatch(t *testing.T, base, path, inm string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, base+"/"+path, nil)
	if err != nil {
		t.Fatalf("new request %s: %v", path, err)
	}
	req.Header.Set("If-None-Match", inm)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s (If-None-Match %q): %v", path, inm, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", path, err)
	}
	return resp.StatusCode, resp.Header, body
}

// TestHandlerServesSeededBytes drives every served path and asserts a 200 with
// bytes byte-equal to what was seeded, plus the 404/400/405 error mappings.
func TestHandlerServesSeededBytes(t *testing.T) {
	srv, sd := newServer(t)

	// Canonical paths built from the tessera layout (not hand-written).
	fullTilePath := tiles.TilePath(0, 0, 0)    // tile/0/000
	partTilePath := tiles.TilePath(0, 1, 44)   // tile/0/001.p/44
	bundlePath := tiles.EntriesPath(0, 0)      // tile/entries/000
	missingTilePath := tiles.TilePath(0, 5, 0) // tile/0/005 (never mirrored)

	const (
		cacheImmutable  = "public, max-age=31536000, immutable"
		cacheRevalidate = "no-cache"
	)

	t.Run("full tile", func(t *testing.T) {
		status, header, body := getWithHeader(t, srv.URL, fullTilePath)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if !bytes.Equal(body, sd.fullTile) {
			t.Errorf("body mismatch for %s", fullTilePath)
		}
		// A full, content-addressed tile is cached as immutable.
		if got := header.Get("Cache-Control"); got != cacheImmutable {
			t.Errorf("Cache-Control = %q, want %q", got, cacheImmutable)
		}
	})

	t.Run("partial tile", func(t *testing.T) {
		status, header, body := getWithHeader(t, srv.URL, partTilePath)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if !bytes.Equal(body, sd.partTile) {
			t.Errorf("body mismatch for %s", partTilePath)
		}
		// The load-bearing assertion: a partial tile is overwritten every poll, so
		// it must revalidate and must NOT carry the immutable directive.
		if got := header.Get("Cache-Control"); got != cacheRevalidate {
			t.Errorf("Cache-Control = %q, want %q", got, cacheRevalidate)
		}
	})

	t.Run("entry bundle", func(t *testing.T) {
		status, header, body := getWithHeader(t, srv.URL, bundlePath)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if !bytes.Equal(body, sd.bundle) {
			t.Errorf("body mismatch for %s", bundlePath)
		}
		// A full entry bundle is cached as immutable.
		if got := header.Get("Cache-Control"); got != cacheImmutable {
			t.Errorf("Cache-Control = %q, want %q", got, cacheImmutable)
		}
	})

	t.Run("checkpoint", func(t *testing.T) {
		status, header, body := getWithHeader(t, srv.URL, "checkpoint")
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if !bytes.Equal(body, sd.checkpoint) {
			t.Errorf("checkpoint body = %q, want %q", body, sd.checkpoint)
		}
		// The checkpoint is size-varying, so it revalidates and is never immutable.
		if got := header.Get("Cache-Control"); got != cacheRevalidate {
			t.Errorf("Cache-Control = %q, want %q", got, cacheRevalidate)
		}
	})

	t.Run("never-mirrored tile is 404", func(t *testing.T) {
		status, _ := get(t, srv.URL, missingTilePath)
		if status != http.StatusNotFound {
			t.Errorf("status = %d, want 404", status)
		}
	})

	t.Run("malformed tile index is 400", func(t *testing.T) {
		// "abc" is not a valid 3-digit chunked index segment.
		status, _ := get(t, srv.URL, "tile/0/abc")
		if status != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", status)
		}
	})

	t.Run("malformed entries index is 400", func(t *testing.T) {
		status, _ := get(t, srv.URL, "tile/entries/abc")
		if status != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", status)
		}
	})

	t.Run("unmatched path is 404", func(t *testing.T) {
		status, _ := get(t, srv.URL, "nope")
		if status != http.StatusNotFound {
			t.Errorf("status = %d, want 404", status)
		}
	})

	t.Run("POST is 405", func(t *testing.T) {
		resp, err := http.Post(srv.URL+"/"+fullTilePath, "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", resp.StatusCode)
		}
	})
}

// TestHandlerConditionalGet covers the strong-ETag + If-None-Match → 304 path:
// every 200 carries a quoted, strong (no W/) content ETag; re-requesting with
// that exact tag yields a 304 with an empty body and the same ETag; a
// non-matching tag still returns the full body; and two distinct resources
// produce distinct ETags (proving the tag is content-derived, not a constant).
func TestHandlerConditionalGet(t *testing.T) {
	srv, sd := newServer(t)

	fullTilePath := tiles.TilePath(0, 0, 0) // tile/0/000
	bundlePath := tiles.EntriesPath(0, 0)   // tile/entries/000

	t.Run("200 carries a strong quoted ETag", func(t *testing.T) {
		status, header, _ := getWithHeader(t, srv.URL, fullTilePath)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		etag := header.Get("ETag")
		if etag == "" {
			t.Fatalf("ETag is empty, want a non-empty strong tag")
		}
		if strings.HasPrefix(etag, "W/") {
			t.Errorf("ETag = %q is weak (W/ prefix), want a strong validator", etag)
		}
		if !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
			t.Errorf("ETag = %q is not quoted, want a quoted token per RFC 7232", etag)
		}
	})

	t.Run("If-None-Match with the exact ETag returns 304 with empty body", func(t *testing.T) {
		// First fetch the resource to learn its ETag and body.
		status, header, body := getWithHeader(t, srv.URL, fullTilePath)
		if status != http.StatusOK {
			t.Fatalf("priming GET status = %d, want 200", status)
		}
		etag := header.Get("ETag")
		if !bytes.Equal(body, sd.fullTile) {
			t.Fatalf("priming GET body mismatch")
		}
		// Re-issue the same GET conditionally with that exact ETag.
		status, header2, body2 := getWithIfNoneMatch(t, srv.URL, fullTilePath, etag)
		if status != http.StatusNotModified {
			t.Fatalf("status = %d, want 304", status)
		}
		if len(body2) != 0 {
			t.Errorf("304 body = %q, want empty", body2)
		}
		if got := header2.Get("ETag"); got != etag {
			t.Errorf("304 ETag = %q, want %q echoed back", got, etag)
		}
	})

	t.Run("If-None-Match wildcard returns 304", func(t *testing.T) {
		status, _, body := getWithIfNoneMatch(t, srv.URL, fullTilePath, "*")
		if status != http.StatusNotModified {
			t.Fatalf("status = %d, want 304", status)
		}
		if len(body) != 0 {
			t.Errorf("304 body = %q, want empty", body)
		}
	})

	t.Run("non-matching If-None-Match returns 200 with the full body", func(t *testing.T) {
		status, _, body := getWithIfNoneMatch(t, srv.URL, fullTilePath, `"deadbeef"`)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		if !bytes.Equal(body, sd.fullTile) {
			t.Errorf("non-matching tag suppressed the body for %s", fullTilePath)
		}
	})

	t.Run("distinct resources produce distinct ETags", func(t *testing.T) {
		_, tileHdr, _ := getWithHeader(t, srv.URL, fullTilePath)
		_, bundleHdr, _ := getWithHeader(t, srv.URL, bundlePath)
		tileTag := tileHdr.Get("ETag")
		bundleTag := bundleHdr.Get("ETag")
		if tileTag == "" || bundleTag == "" {
			t.Fatalf("missing ETag: tile=%q bundle=%q", tileTag, bundleTag)
		}
		// The seeded full tile and bundle are distinct byte streams, so a
		// content-derived ETag must differ; a constant tag would fail here.
		if tileTag == bundleTag {
			t.Errorf("ETags collide (%q) — tag is not content-derived", tileTag)
		}
	})
}
