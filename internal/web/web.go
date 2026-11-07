// Package web serves the monitor's shared static front-end assets — the ISCC
// Design System v2 token stylesheet, the shared site-chrome stylesheet (site.css), the
// self-hosted webfont @font-face shell, the woff2 binaries themselves, the Go WASM
// runtime loader (wasm_exec.js), the verifier WebAssembly artifact (verify.wasm), the
// ISCC masthead logo and favicon (byte-verbatim copies of the ISCC brand SVG masters),
// and the self-hosted Stoplight Elements assets
// (elements.min.js + elements.min.css) the /docs API reference mounts — over
// net/http as a tiny stdlib-only leaf. The stylesheets and fonts are the one no-JS,
// no-CDN style shell every server-rendered M-UI surface (the realm index at "/", the
// hub dossier, the log browser, the certificate page) links via stable paths under
// /_ds/, so the design tokens and fonts are defined once and shared. The wasm_exec.js
// loader is the same-origin Go runtime glue the later tier-2 progressive enhancement
// must load before instantiating the verifier .wasm; serving it here keeps that
// runtime CDN-free too.
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
// WASM-shareable, and never pulls a heavier dependency into the asset path. The
// wasm_exec.js asset adds no import (embed is already in use). CORS rides the outer
// corsmw.Handler wrap at the mux convergence point, so this handler sets no CORS
// headers of its own.
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

// WasmExecPath is the stable exact path the Go WASM runtime loader is served at. The
// later tier-2 progressive enhancement loads this same-origin script before
// instantiating the verifier .wasm; like TokensPath a page's <script src> literal
// must stay in sync with this value.
const WasmExecPath = "/_ds/wasm_exec.js"

// WasmVerifyPath is the stable exact path the verifier WebAssembly artifact is served
// at. The later tier-2 progressive enhancement fetches and instantiates this
// same-origin module (after loading WasmExecPath) to run the in-browser inclusion
// verifier; like TokensPath a page's fetch URL literal must stay in sync with it.
const WasmVerifyPath = "/_ds/verify.wasm"

// SiteCSSPath is the stable exact path the shared site-chrome stylesheet is served at:
// the masthead and footer partials (internal/chrome), the brand buttons, and the focus
// outline. Every page links it after TokensPath; like TokensPath the href literal must
// stay in sync with this value.
const SiteCSSPath = "/_ds/site.css"

// LogoPath is the stable exact path the self-hosted ISCC logo is served at: the
// near-black signature with the coral circle, a byte-verbatim copy of the ISCC brand
// master iscc-logo-black-coral.svg. The masthead partial (internal/chrome) renders it
// as an <img src> beside the "Trust & Transparency Monitor" text mark; that literal
// must stay in sync with this value, since templates cannot read the Go const.
const LogoPath = "/_ds/iscc-logo.svg"

// FaviconPath is the stable exact path the favicon is served at: the near-black ISCC
// symbol with its coral circle, a byte-verbatim copy of the brand master
// iscc-icon-black-coral.svg (the brand's favicon.svg). Every page links it as
// <link rel="icon">; that literal must stay in sync with this value.
const FaviconPath = "/_ds/favicon.svg"

// OperatorLogoPath is the exact path an instance serves its operator's logo at, when
// ISCC_MONITOR_OPERATOR_LOGO configures one. The binary mounts ImageHandler there; the
// /_ds/ subtree Handler itself never serves it, so the static verifier site has none.
const OperatorLogoPath = "/_ds/operator-logo"

// ElementsJSPath is the stable exact path the self-hosted Stoplight Elements
// web-component JS bundle is served at. The /docs page (internal/docs) loads it as a
// same-origin <script> to mount the <elements-api> component; like TokensPath that
// template's <script src> literal must stay in sync with this value.
const ElementsJSPath = "/_ds/elements.min.js"

// ElementsCSSPath is the stable exact path the self-hosted Stoplight Elements
// stylesheet is served at. The /docs page links it as a same-origin <link>; like
// TokensPath that template's href literal must stay in sync with this value.
const ElementsCSSPath = "/_ds/elements.min.css"

// WasmVerifyHash is the published lowercase-hex SHA-256 of the committed verify.wasm
// bytes — the reproducible-build artifact hash a client compares against to confirm it
// loaded the audited verifier (the SRI/verify-artifact pin). It is produced by the
// deterministic `mise run build:wasm` task (GOOS=js GOARCH=wasm CGO_ENABLED=0 go build
// -trimpath -ldflags=-buildid= -buildvcs=false) under the project-pinned Go toolchain;
// the -buildvcs=false is load-bearing — without it Go stamps VCS revision/dirty
// metadata into the wasm data section and the hash is unreproducible. Re-running that
// task without re-pinning this const fails TestWasmVerifyHashPinned. The bytes are
// toolchain-dependent, so this value tracks `mise run build:wasm`, not a bare go build.
const WasmVerifyHash = "6d060ca5e0b430e52e0fdd894a200bf72dbba960318f9a0611c70c96956d9ad6"

// ElementsJSHash is the published lowercase-hex SHA-256 of the committed
// elements.min.js bytes — the build-pinned hash a reviewer compares against to
// confirm the vendored Stoplight Elements web-component bundle is the audited one
// (the SRI / pinned self-hosted-asset discipline, alongside WasmVerifyHash). The
// bundle is vendored byte-verbatim from a PINNED upstream version
// (@stoplight/elements@9.0.23 web-components.min.js), committed and never hand-edited;
// re-fetch + re-pin deliberately on a version bump (the "verify.wasm pin is fragile"
// discipline). Its SHA-256 must equal this value (TestElementsAssetsHashPinned).
const ElementsJSHash = "46e5a044295bbd599772e1a5e678e807078a3d2cd43226640a50917cd88d6938"

// ElementsCSSHash is the published lowercase-hex SHA-256 of the committed
// elements.min.css bytes — the build-pinned hash for the vendored Stoplight Elements
// stylesheet (@stoplight/elements@9.0.23 styles.min.css), the sibling pin of
// ElementsJSHash. Committed byte-verbatim, never hand-edited; its SHA-256 must equal
// this value (TestElementsAssetsHashPinned).
const ElementsCSSHash = "a52002228108fb567b75caff209c4d8aa256ae591de3dea7d3b1a384b1a27b06"

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

// contentTypeJS is the media type served for the wasm_exec.js runtime loader. It is
// set explicitly so the browser executes it as a script regardless of content
// sniffing.
const contentTypeJS = "text/javascript; charset=utf-8"

// contentTypeWASM is the media type served for the verifier .wasm artifact. It is the
// IANA media type browsers require for WebAssembly.instantiateStreaming; it is set
// explicitly so a content sniffer cannot downgrade it and refuse the streaming
// instantiation.
const contentTypeWASM = "application/wasm"

// contentTypeSVG is the media type served for the embedded logo and favicon SVGs. It is
// set explicitly so the browser renders them as images without depending on content
// sniffing.
const contentTypeSVG = "image/svg+xml"

// imageCSP is the Content-Security-Policy of the operator logo response. The logo is an
// operator-supplied file that may be an SVG; this policy stops any script or external
// load in it from running if the URL is opened directly, rather than as an <img>.
const imageCSP = "default-src 'none'; style-src 'unsafe-inline'"

// TokensCSS is the embedded ISCC Design System v2 token stylesheet — a single
// concatenated, CDN-free file (colors, typography, spacing, base tokens), embedded
// at build time so the served bytes are fixed and carry no external URL.
//
//go:embed tokens.css
var TokensCSS []byte

// siteCSS is the embedded shared site-chrome stylesheet (masthead, footer, buttons,
// focus outline).
//
//go:embed site.css
var siteCSS []byte

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

// wasmExecJS is the embedded Go WASM runtime loader — a byte-verbatim copy of the Go
// 1.26.10 toolchain's lib/wasm/wasm_exec.js (BSD-licensed). It is a build-pinned,
// version-locked asset (the same posture as the committed woff2 binaries); never
// hand-edit it. The later tier-2 enhancement loads it to bootstrap the verifier .wasm.
//
//go:embed wasm_exec.js
var wasmExecJS []byte

// wasmVerify is the embedded verifier WebAssembly artifact — the deterministic output
// of `mise run build:wasm` over cmd/wasm (the GOOS=js GOARCH=wasm verifier entrypoint),
// committed so the served bytes are build-pinned. It is a generated asset, never
// hand-edited: regenerate it with `mise run build:wasm` on a source or toolchain change
// and re-pin WasmVerifyHash. Its SHA-256 must equal WasmVerifyHash (TestWasmVerifyHashPinned).
//
//go:embed verify.wasm
var wasmVerify []byte

// logoSVG is the embedded ISCC logo, a byte-verbatim copy of the ISCC brand master
// iscc-logo-black-coral.svg (never hand-edited; the brand forbids rebuilding the logo).
// Its viewBox includes the brand's clear space.
//
//go:embed iscc-logo.svg
var logoSVG []byte

// faviconSVG is the embedded favicon, a byte-verbatim copy of the ISCC brand master
// iscc-icon-black-coral.svg.
//
//go:embed favicon.svg
var faviconSVG []byte

// elementsJS is the embedded Stoplight Elements web-component JS bundle — vendored
// byte-verbatim from a PINNED upstream version (@stoplight/elements@9.0.23
// web-components.min.js), committed so the served bytes are build-pinned. It is a
// downloaded, generated asset, never hand-edited: re-fetch + re-pin ElementsJSHash
// deliberately on a version bump (the "verify.wasm pin is fragile" discipline). The
// /docs page (internal/docs) loads it same-origin to mount the <elements-api>
// component. IMPORTANT for any no-CDN reviewer: this ~2 MB minified bundle contains
// hundreds of inert, baked http(s):// example/documentation strings (golang.org,
// example.com, etc.) that are DATA, not runtime fetches — the bundle issues no
// external request at runtime. Do NOT run a substring CDN ban over these bytes; the
// no-CDN assertion applies only to the rendered /docs HTML page body.
//
//go:embed elements.min.js
var elementsJS []byte

// elementsCSS is the embedded Stoplight Elements stylesheet — vendored byte-verbatim
// from the same pinned version (@stoplight/elements@9.0.23 styles.min.css), committed
// and build-pinned. It has no @import and no external-font url() (confirmed), so it
// references no external CDN at runtime. The /docs page links it same-origin. Like
// elementsJS it is a downloaded asset, never hand-edited; re-pin ElementsCSSHash on a
// version bump.
//
//go:embed elements.min.css
var elementsCSS []byte

// Handler returns an http.Handler for the /_ds/ static-asset subtree. It serves the
// token and site-chrome stylesheets, the @font-face stylesheet, the woff2 binaries, the
// wasm_exec.js runtime loader, the verifier verify.wasm artifact, the logo and favicon
// SVGs, and the Stoplight Elements JS bundle + stylesheet; the content type is chosen
// per path (text/css for the .css, text/javascript for wasm_exec.js and
// elements.min.js, application/wasm for verify.wasm, image/svg+xml for the logo and
// favicon, font/woff2 for .woff2).
// Only GET is served
// (any other method is 405); an unknown /_ds/ path is 404. Every 200 carries
// Cache-Control: no-cache and a strong content ETag, with an If-None-Match match
// short-circuiting to 304. It sets no CORS headers — the outer corsmw wrap at the mux
// convergence point owns the single CORS policy.
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
		case SiteCSSPath:
			writeAsset(w, r, siteCSS, contentTypeCSS)
		case FontsCSSPath:
			writeAsset(w, r, fontsCSS, contentTypeCSS)
		case WasmExecPath:
			writeAsset(w, r, wasmExecJS, contentTypeJS)
		case WasmVerifyPath:
			writeAsset(w, r, wasmVerify, contentTypeWASM)
		case LogoPath:
			writeAsset(w, r, logoSVG, contentTypeSVG)
		case FaviconPath:
			writeAsset(w, r, faviconSVG, contentTypeSVG)
		case ElementsJSPath:
			writeAsset(w, r, elementsJS, contentTypeJS)
		case ElementsCSSPath:
			writeAsset(w, r, elementsCSS, contentTypeCSS)
		default:
			serveFont(w, r)
		}
	})
}

// ImageHandler returns an http.Handler that serves one image, such as the operator logo
// at OperatorLogoPath, with the same GET-only, revalidating cache policy as Handler. It
// adds a restrictive Content-Security-Policy and X-Content-Type-Options: nosniff, so an
// operator-supplied SVG opened directly cannot run script on the instance's origin.
func ImageHandler(data []byte, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Security-Policy", imageCSP)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		writeAsset(w, r, data, contentType)
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
