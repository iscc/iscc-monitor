// This file resolves a hub's signed-note verifier key over the network: given a
// hub base URL it fetches the hub's /.well-known/did.json through an injected
// Fetcher and runs the pure did:web chain (DocumentURL -> ParseDIDDocument ->
// VerifierKey) to produce the key checkpoints are verified against. It is the
// non-WASM, networked half of did:web resolution (ADR-0009): it imports
// net/http, so it lives in logclient, not the WASM-pure internal/didweb. Any
// fetch/parse/derive failure collapses to ErrUnresolvable so the follower maps
// it to hub status "unresolvable" and keeps mirroring without inspecting the
// resolver's internals.
package logclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/iscc/iscc-monitor/internal/didweb"
)

// ErrUnresolvable is the sentinel every resolution failure wraps.
//
// The follower checks errors.Is(err, ErrUnresolvable) to set hub status
// "unresolvable" (keep mirroring, do not advance) for any failure to fetch,
// parse, or derive the hub's key — without depending on the underlying cause.
var ErrUnresolvable = errors.New("did:web key unresolvable")

// Fetcher is the outbound-fetch seam: it returns the raw bytes at a URL.
//
// The resolver and (later) the follower depend on this 1-method interface so
// tests inject fixture bytes instead of hitting the live network. Implementations
// must honor the context for cancellation and timeout.
type Fetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// httpFetcher fetches URLs over a *http.Client, mirroring Tessera's HTTPFetcher.
//
// A nil client falls back to http.DefaultClient. It is unexported; callers
// construct it through NewHTTPFetcher.
type httpFetcher struct {
	c *http.Client
}

// NewHTTPFetcher returns a Fetcher backed by the given HTTP client.
//
// A nil client uses http.DefaultClient.
func NewHTTPFetcher(c *http.Client) Fetcher {
	if c == nil {
		c = http.DefaultClient
	}
	return httpFetcher{c: c}
}

// Fetch GETs url and returns its body bytes.
//
// It maps a 404 to os.ErrNotExist (by contract, mirroring Tessera's fetcher),
// treats any other non-200 as an error, and always closes the response body. The
// returned errors are NOT wrapped in ErrUnresolvable; ResolveVerifierKey owns
// that mapping so the Fetcher stays reusable for non-did fetches.
func (h httpFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", url, err)
	}
	resp, err := h.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		// All good, continue below.
	case http.StatusNotFound:
		return nil, fmt.Errorf("fetch %q: %w", url, os.ErrNotExist)
	default:
		return nil, fmt.Errorf("fetch %q: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ResolveVerifierKey resolves a hub base URL to its signed-note verifier key.
//
// It derives the hub's did:web identifier from the base URL host, maps it to the
// did.json URL (didweb.DocumentURL), fetches the document through fetcher, parses
// it (didweb.ParseDIDDocument), and derives the verifier-key string under the
// hub's origin (didweb.VerifierKey). It returns the key string alongside the
// parsed DIDKey (carrying the public key and CID 1.0 validity window). Any
// failure to fetch, parse, or derive is wrapped with ErrUnresolvable.
func ResolveVerifierKey(ctx context.Context, fetcher Fetcher, baseURL string) (string, didweb.DIDKey, error) {
	name, err := origin(baseURL)
	if err != nil {
		return "", didweb.DIDKey{}, fmt.Errorf("%v: %w", err, ErrUnresolvable)
	}
	host := strings.TrimSuffix(name, "/log")
	// did:web requires a host:port to percent-encode the colon (W3C did:web
	// §3.2), so DocumentURL maps the first MSID segment back to host:port rather
	// than splitting the port off as a path component.
	did := "did:web:" + strings.Replace(host, ":", "%3A", 1)
	docURL, err := didweb.DocumentURL(did)
	if err != nil {
		return "", didweb.DIDKey{}, fmt.Errorf("%v: %w", err, ErrUnresolvable)
	}
	data, err := fetcher.Fetch(ctx, docURL)
	if err != nil {
		return "", didweb.DIDKey{}, fmt.Errorf("fetch did.json: %v: %w", err, ErrUnresolvable)
	}
	key, err := didweb.ParseDIDDocument(data)
	if err != nil {
		return "", didweb.DIDKey{}, fmt.Errorf("%v: %w", err, ErrUnresolvable)
	}
	return didweb.VerifierKey(name, key.PublicKey), key, nil
}
