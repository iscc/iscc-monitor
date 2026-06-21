// Package tilesserve exposes one hub's mirrored tlog-tiles artifacts over
// net/http, served verbatim from the local store.SQLiteFetcher and never
// re-hitting the hub. It is the canonical static read surface M2 requires
// (iscc-log §9): GET /checkpoint, GET /tile/<L>/<index...>, and
// GET /tile/entries/<index...> — including the ".p/<W>" partial-tile suffix —
// each returning the raw BLOB bytes a verifier reads to compute proofs locally.
//
// There are no proof-computing endpoints here; the handler only serves the
// static files. It is a separate package from store precisely so net/http never
// enters the store closure (store stays a leaf): tilesserve depends on store,
// never the reverse. The canonical path parsers come from tessera's api/layout
// (ParseTileLevelIndexPartial / ParseTileIndexPartial), so the chunked index and
// partial-suffix math is never hand-rolled.
package tilesserve

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/transparency-dev/tessera/api/layout"

	"github.com/iscc/iscc-monitor/internal/store"
)

// contentType is the media type for every served BLOB. The bodies are the
// canonical tlog-tiles static files (signed-note checkpoint, hash tile, entry
// bundle), so they are returned as opaque octet streams.
const contentType = "application/octet-stream"

const (
	// cacheImmutable is the Cache-Control policy for a content-addressed FULL
	// tile/entry-bundle (width == 0 in the path-API vocabulary). A full resource
	// never changes once mirrored, so it is cached for a year and the RFC-8246
	// immutable directive lets browsers skip revalidation entirely.
	cacheImmutable = "public, max-age=31536000, immutable"
	// cacheRevalidate is the Cache-Control policy for resources that are
	// overwritten in place: a PARTIAL tile/bundle (re-fetched and overwritten
	// every poll, ADR-0005) and the size-varying checkpoint. no-cache lets a
	// cache store the response but forces revalidation before reuse — it must NOT
	// carry the immutable directive or a client would pin a soon-overwritten
	// partial forever.
	cacheRevalidate = "no-cache"
)

// Handler returns an http.Handler that serves one hub's mirrored tlog-tiles
// artifacts from f. It routes the three canonical paths to f.ReadCheckpoint /
// f.ReadTile / f.ReadEntryBundle and writes the raw BLOB bytes verbatim.
//
// Status mapping: a path-parse failure → 400; a missing mirror row
// (errors.Is(err, os.ErrNotExist), the SQLiteFetcher sentinel) → 404; any other
// read error → 500; a non-GET method → 405; an unmatched path → 404.
//
// Each 200 carries a Cache-Control policy keyed on cacheability: a content-
// addressed FULL tile/entry-bundle (path-API width == 0) is immutable, while a
// PARTIAL tile/bundle and the size-varying checkpoint revalidate (no-cache). CORS
// is set once by the corsmw wrap; conditional GET (ETag / If-None-Match) is
// intentionally out of scope for this slice.
func Handler(f store.SQLiteFetcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// THIS handler is mounted at the hub root, so it sees the path suffix of
		// the hub's /log origin: a single leading slash is trimmed and the
		// canonical relative paths (checkpoint, tile/..., tile/entries/...) are
		// matched directly.
		path := strings.TrimPrefix(r.URL.Path, "/")
		switch {
		case path == "checkpoint":
			serveCheckpoint(w, r, f)
		// tile/entries/ must be matched BEFORE tile/ — entries is a sub-prefix of
		// tile, and a bare tile/<L> parse of an entries path must not win.
		case strings.HasPrefix(path, "tile/entries/"):
			serveEntries(w, r, f, strings.TrimPrefix(path, "tile/entries/"))
		case strings.HasPrefix(path, "tile/"):
			serveTile(w, r, f, strings.TrimPrefix(path, "tile/"))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
}

// serveCheckpoint writes the hub's latest checkpoint BLOB. The checkpoint is
// size-varying (overwritten as the tree grows), so it is never immutable.
func serveCheckpoint(w http.ResponseWriter, r *http.Request, f store.SQLiteFetcher) {
	data, err := f.ReadCheckpoint(r.Context())
	if err != nil {
		writeReadError(w, err)
		return
	}
	writeBlob(w, data, false)
}

// serveTile parses rest as "<L>/<index...>" (the chunked index may itself contain
// "/" and a ".p/<W>" partial suffix), then writes the hash tile BLOB. tessera's
// returned width IS the fetcher's p (0 == full), so it is passed straight through.
func serveTile(w http.ResponseWriter, r *http.Request, f store.SQLiteFetcher, rest string) {
	level, index, ok := strings.Cut(rest, "/")
	if !ok {
		http.Error(w, "bad tile path", http.StatusBadRequest)
		return
	}
	l, i, width, err := layout.ParseTileLevelIndexPartial(level, index)
	if err != nil {
		http.Error(w, "bad tile index", http.StatusBadRequest)
		return
	}
	data, err := f.ReadTile(r.Context(), l, i, width)
	if err != nil {
		writeReadError(w, err)
		return
	}
	// In the tessera path-API vocabulary width == 0 is a FULL tile (immutable);
	// any width > 0 is a partial that is overwritten every poll (revalidate).
	writeBlob(w, data, width == 0)
}

// serveEntries parses index as the chunked bundle index (with optional ".p/<W>"
// partial suffix), then writes the entry bundle BLOB. tessera's returned width IS
// the fetcher's p (0 == full), so it is passed straight through.
func serveEntries(w http.ResponseWriter, r *http.Request, f store.SQLiteFetcher, index string) {
	i, width, err := layout.ParseTileIndexPartial(index)
	if err != nil {
		http.Error(w, "bad entries index", http.StatusBadRequest)
		return
	}
	data, err := f.ReadEntryBundle(r.Context(), i, width)
	if err != nil {
		writeReadError(w, err)
		return
	}
	// width == 0 is a FULL entry bundle (immutable); width > 0 is a partial that
	// is overwritten every poll (revalidate).
	writeBlob(w, data, width == 0)
}

// writeReadError maps a fetcher read error to its HTTP status: the
// os.ErrNotExist sentinel (a never-mirrored row) → 404, anything else → 500.
func writeReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, os.ErrNotExist) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// writeBlob sets the octet-stream content type, the Cache-Control policy, and
// writes the raw BLOB bytes. immutable selects the long-lived immutable policy
// for content-addressed full tiles/bundles; false selects the revalidating
// policy for partials and the size-varying checkpoint. Both headers are set
// before the first write because the 200 is sent on first write and freezes the
// header map.
func writeBlob(w http.ResponseWriter, data []byte, immutable bool) {
	w.Header().Set("Content-Type", contentType)
	if immutable {
		w.Header().Set("Cache-Control", cacheImmutable)
	} else {
		w.Header().Set("Cache-Control", cacheRevalidate)
	}
	// The 200 is sent on the first write; a mid-write error cannot un-send it and
	// the only failure mode here is a broken client connection, so it is dropped
	// deliberately rather than writing a misleading second status (matching
	// metricshttp). Not a gate dodge.
	_, _ = w.Write(data)
}
