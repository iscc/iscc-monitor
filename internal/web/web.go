// Package web serves the monitor's shared static front-end assets — currently the
// ISCC Design System v2 token stylesheet — over net/http as a tiny stdlib-only
// leaf. It is the one no-JS, no-CDN style shell every server-rendered M-UI surface
// (the realm index at "/", the hub dossier, the log browser, the certificate page)
// links via a single stable path, so the design tokens are defined once and shared.
//
// The token CSS is go:embed-ed at build time, so the served bytes are build-pinned
// and the stylesheet carries no external CDN URL (the load-bearing M-UI invariant:
// every SSR body is complete with JavaScript disabled and references no third-party
// origin). The self-hosted webfonts are a separate later sub-step; the token CSS's
// font stacks already list Arial / Consolas fallbacks so pages render correctly
// with no webfont loaded.
//
// The package is a pure leaf: its only imports are bytes / embed / net/http
// (stdlib), with no internal/store, no internal/metrics, and no internal/logclient,
// so it stays trivially testable and never pulls a heavier dependency into the
// asset path. CORS rides the outer corsmw.Handler wrap at the mux convergence
// point, so this handler sets no CORS headers of its own.
//
// The oracle/conformance gate is N/A: this is pure static-asset transport,
// touching no signature, RFC-6962, Merkle, did:web, fsck, or proof path.
package web

import (
	"bytes"
	_ "embed"
	"net/http"
)

// TokensPath is the stable exact path the token stylesheet is served at. The
// binary mounts Handler here and every SSR page links this literal in its <head>;
// templates cannot read this Go const, so a page's literal href must stay in sync
// with this value.
const TokensPath = "/_ds/tokens.css"

// contentType is the CSS content type served for the token stylesheet.
const contentType = "text/css; charset=utf-8"

// cacheControl marks the token stylesheet immutable for a year: the bytes are
// build-pinned (go:embed), so a client may cache them indefinitely and a redeploy
// changes the served bytes only when the binary itself changes.
const cacheControl = "public, max-age=31536000, immutable"

// TokensCSS is the embedded ISCC Design System v2 token stylesheet — a single
// concatenated, CDN-free file (colors, typography, spacing, base tokens). It is
// embedded at build time so the served bytes are fixed and carry no external URL.
//
//go:embed tokens.css
var TokensCSS []byte

// Handler returns an http.Handler that serves the embedded token stylesheet. Only
// GET is served (any other method is 405); on GET it sets the CSS content type and
// an immutable long-lived Cache-Control before writing the build-pinned bytes. It
// sets no CORS headers — the outer corsmw wrap at the mux convergence point owns
// the single CORS policy.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", cacheControl)
		w.WriteHeader(http.StatusOK)
		// Post-200 write-drop: the status is already committed, so a copy error can
		// only signal a broken client connection, which a second status cannot fix.
		_, _ = bytes.NewReader(TokensCSS).WriteTo(w)
	})
}
