// Package web serves the monitor's shared static front-end assets — the ISCC
// Design System v2 token stylesheet, the self-hosted webfont @font-face shell, and
// the woff2 binaries themselves — over net/http as a tiny stdlib-only leaf. It is
// the one no-JS, no-CDN style shell every server-rendered M-UI surface (the realm
// index at "/", the hub dossier, the log browser, the certificate page) links via
// stable paths under /_ds/, so the design tokens and fonts are defined once and
// shared.
//
// Everything is go:embed-ed at build time, so the served bytes are build-pinned and
// no asset references an external CDN URL (the load-bearing M-UI invariant: every
// SSR body is complete with JavaScript disabled and references no third-party
// origin). The fonts.css @font-face src URLs are same-origin /_ds/fonts/ paths and
// the woff2 binaries are committed and embedded, so the typeface renders with zero
// runtime CDN dependency; the token CSS still lists Arial / Consolas fallbacks so a
// page renders correctly before a webfont loads.
//
// All /_ds/ assets are served by one handler mounted at the /_ds/ subtree. Each is
// an overwrite-in-place resource at a stable (non-content-addressed) path — a
// redeploy reuses the URL with changed bytes — so none may carry the immutable
// directive. The handler uses the project's revalidating cache shape (the
// internal/tilesserve.writeBlob policy): Cache-Control: no-cache plus a strong
// content ETag (quoted hex of the bytes' SHA-256), with an If-None-Match match
// short-circuiting to 304 Not Modified, so a client revalidates cheaply without
// pinning a soon-overwritten asset.
//
// The package is a pure leaf: its only imports are crypto/sha256 / embed / fmt /
// net/http / strings / io / fs (stdlib), with no internal/store, no
// internal/metrics, and no internal/logclient, so it stays trivially testable,
// WASM-shareable, and never pulls a heavier dependency into the asset path. CORS
// rides the outer corsmw.Handler wrap at the mux convergence point, so this handler
// sets no CORS headers of its own.
//
// The oracle/conformance gate is N/A: this is pure static-asset transport,
// touching no signature, RFC-6962, Merkle, did:web, fsck, or proof path.
package web

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

// Prefix is the stable subtree every shared static asset is served under. The
// binary mounts Handler here so GET /_ds/tokens.css, GET /_ds/fonts.css, and GET
// /_ds/fonts/<file>.woff2 all reach this package.
const Prefix = "/_ds/"

// TokensPath is the stable exact path the token stylesheet is served at. Every SSR
// page links this literal in its <head>; templates cannot read this Go const, so a
// page's literal href must stay in sync with this value.
const TokensPath = "/_ds/tokens.css"

// FontsCSSPath is the stable exact path the self-hosted @font-face stylesheet is
// served at. A page links it (or tokens.css @imports it) to load the webfonts; like
// TokensPath it is duplicated as a template literal that must stay in sync.
const FontsCSSPath = "/_ds/fonts.css"

// cacheControl is the Cache-Control policy for every /_ds/ asset. Each is served at
// a stable, overwrite-in-place URL (not content-addressed), so it must NOT carry the
// immutable directive — that would let a client pin a soon-overwritten asset for a
// year. no-cache lets a cache store the response but forces revalidation (cheap via
// the strong ETag below) before reuse, mirroring internal/tilesserve.cacheRevalidate.
const cacheControl = "no-cache"

// contentTypeCSS is the media type served for the .css stylesheets.
const contentTypeCSS = "text/css; charset=utf-8"

// contentTypeWOFF2 is the media type served for the .woff2 binaries. It is set
// explicitly because a content sniffer would otherwise classify woff2 as an opaque
// octet stream, which some browsers refuse to use as a webfont.
const contentTypeWOFF2 = "font/woff2"

// TokensCSS is the embedded ISCC Design System v2 token stylesheet — a single
// concatenated, CDN-free file (colors, typography, spacing, base tokens), embedded
// at build time so the served bytes are fixed and carry no external URL.
//
//go:embed tokens.css
var TokensCSS []byte

// fontsCSS is the embedded self-hosted @font-face stylesheet. Its src URLs are
// same-origin /_ds/fonts/ paths, so the served bytes reference no external CDN.
//
//go:embed fonts.css
var fontsCSS []byte

// fontsFS is the embedded directory of woff2 font binaries (the eight latin subsets:
// Readex Pro 300/400/500/600/700 + JetBrains Mono 400/500/700) plus the SIL OFL
// license text. The binaries are committed assets, so the served bytes are
// build-pinned and never re-fetched at runtime.
//
//go:embed fonts
var fontsFS embed.FS

// Handler returns an http.Handler for the /_ds/ static-asset subtree. It serves the
// token stylesheet, the @font-face stylesheet, and the woff2 binaries; the content
// type is chosen by extension (text/css for .css, font/woff2 for .woff2). Only GET
// is served (any other method is 405); an unknown /_ds/ path is 404. Every 200
// carries Cache-Control: no-cache and a strong content ETag, with an If-None-Match
// match short-circuiting to 304. It sets no CORS headers — the outer corsmw wrap at
// the mux convergence point owns the single CORS policy.
//
// It must be mounted at Prefix (a trailing-slash subtree pattern) so the whole
// /_ds/ tree, including /_ds/fonts/, routes here; an exact-path mount would 404 the
// woff2 binaries.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case TokensPath:
			writeAsset(w, r, TokensCSS, contentTypeCSS)
		case FontsCSSPath:
			writeAsset(w, r, fontsCSS, contentTypeCSS)
		default:
			serveFont(w, r)
		}
	})
}

// serveFont serves a woff2 binary from the embedded fonts directory. It accepts only
// paths of the form /_ds/fonts/<name>.woff2 with no further "/" (the embed has no
// nested dirs), reading the file via its embed.FS name. A missing file or any
// non-woff2 path is 404.
func serveFont(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, Prefix)
	if !strings.HasSuffix(name, ".woff2") || strings.Contains(strings.TrimPrefix(name, "fonts/"), "/") {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(fontsFS, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeAsset(w, r, data, contentTypeWOFF2)
}

// writeAsset sets the content type, the revalidating Cache-Control, and a strong
// content ETag, then either short-circuits to 304 or writes the asset bytes. It is
// the internal/tilesserve.writeBlob shape: the ETag is the quoted hex of the bytes'
// SHA-256 (a STRONG validator, no W/ prefix), and an If-None-Match that is the
// wildcard "*" or echoes this exact tag yields 304 Not Modified with no body. All
// three headers are set BEFORE the conditional branch because the 200/304 status is
// sent on first write and freezes the header map (and per RFC 7232 §4.1 a 304 still
// carries the validating ETag and Cache-Control).
func writeAsset(w http.ResponseWriter, r *http.Request, data []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", cacheControl)
	etag := fmt.Sprintf("\"%x\"", sha256.Sum256(data))
	w.Header().Set("ETag", etag)

	// A client echoes back the exact ETag the server sent, so an exact-token match
	// (or the "*" wildcard) is sufficient — no comma-separated list parser needed.
	if inm := r.Header.Get("If-None-Match"); inm == "*" || inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.WriteHeader(http.StatusOK)
	// Post-200 write-drop: the status is already committed, so a copy error can
	// only signal a broken client connection, which a second status cannot fix.
	_, _ = w.Write(data)
}
