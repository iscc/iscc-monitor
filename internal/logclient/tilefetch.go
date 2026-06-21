// This file holds the networked tile/entry-bundle fetch primitives: given a hub
// base URL and an injected Fetcher they return the raw bytes of one hash tile or
// one entry bundle from the canonical tlog-tiles paths
// (https://<domain>/log/tile/<l>/<i> and https://<domain>/log/tile/entries/<i>,
// iscc-log §9). They are the transport-only seam between the pure coordinate
// enumerations (tiles.TileCoords / tiles.BundleCoords) and the mirror writers
// (store.RecordTile / store.RecordEntryBundle), mirroring FetchCheckpoint exactly:
// origin() -> "https://" + name + "/" + path -> fetcher.Fetch -> body verbatim.
// They do not parse the tile/bundle (LeafHashes and the proof builders own that)
// and synthesize no os.ErrNotExist of their own — the Fetcher's 404 contract
// rides through the %w wrap so the ingestion loop can tell "tile not served yet"
// from a hard transport fault.
package logclient

import (
	"context"
	"fmt"

	"github.com/iscc/iscc-monitor/internal/tiles"
)

// FetchTile fetches the raw bytes of one hash tile from a hub base URL.
//
// It derives the canonical tile URL by reusing the shared origin() helper (which
// yields the scheme-less "<domain>/log") and appending tiles.TilePath(level,
// index, p), so a base URL such as "https://sb0.iscc.id" or the bare "sb0.iscc.id"
// resolves to "https://sb0.iscc.id/log/tile/<level>/<index>" (or the ".p/<p>"
// partial path when p > 0). It fetches through the injected Fetcher and returns
// the body verbatim — no parsing; decoding the tile is the caller's concern. The
// Fetcher's error is wrapped with %w so a 404's errors.Is(err, os.ErrNotExist)
// still holds, letting the ingestion loop tell "tile not served yet" from other
// transport faults; an origin() error is likewise wrapped and returned.
func FetchTile(ctx context.Context, fetcher Fetcher, baseURL string, level, index uint64, p uint8) ([]byte, error) {
	name, err := origin(baseURL)
	if err != nil {
		return nil, fmt.Errorf("fetch tile: %w", err)
	}
	url := "https://" + name + "/" + tiles.TilePath(level, index, p)
	raw, err := fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch tile %q: %w", url, err)
	}
	return raw, nil
}

// FetchEntryBundle fetches the raw bytes of one entry bundle from a hub base URL.
//
// It derives the canonical bundle URL by reusing the shared origin() helper and
// appending tiles.EntriesPath(index, p), so a base URL such as "https://sb0.iscc.id"
// resolves to "https://sb0.iscc.id/log/tile/entries/<index>" (or the ".p/<p>"
// partial path when p > 0). It fetches through the injected Fetcher and returns
// the body verbatim — no parsing; decoding the bundle into leaf hashes belongs to
// LeafHashes. The Fetcher's error is wrapped with %w so a 404's errors.Is(err,
// os.ErrNotExist) still holds; an origin() error is likewise wrapped and returned.
func FetchEntryBundle(ctx context.Context, fetcher Fetcher, baseURL string, index uint64, p uint8) ([]byte, error) {
	name, err := origin(baseURL)
	if err != nil {
		return nil, fmt.Errorf("fetch entry bundle: %w", err)
	}
	url := "https://" + name + "/" + tiles.EntriesPath(index, p)
	raw, err := fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch entry bundle %q: %w", url, err)
	}
	return raw, nil
}
