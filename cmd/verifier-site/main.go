// Command verifier-site renders the standalone Independent Verification site
// (Surface C, the monitor-agnostic verifier app published at monitor.iscc.codes)
// into an output directory as a complete, self-contained static tree: the single
// index.html page plus every /_ds/ asset it loads (the Design System token and
// @font-face stylesheets, the woff2 binaries, the Go WASM runtime loader, the
// SRI-pinned verify.wasm artifact, and the masthead logo). The GitHub-Pages deploy
// invokes this one command to assemble the deployable artifact, so the published
// site is byte-identical to what the golden tests already gate.
//
// The page and assets are materialized by driving the REAL handlers
// (verifier.Handler and web.Handler) over httptest, never by re-embedding the
// template or hardcoding the asset list — so there is one source of truth. index.html
// is whatever verifier.Handler renders for GET /, and each /_ds/ path is written at
// its URL path (web.Prefix becomes a real subdirectory) so the on-disk layout matches
// what the page fetches at runtime, exactly as GitHub Pages serves it. The font set is
// enumerated from the served fonts.css src URLs, so a future font add/remove flows
// through without editing this generator.
//
// main stays thin and owns the single os.Exit; all the rendering and writing lives in
// the testable generate helper. generate fails closed: if any expected handler returns
// non-200 (a future asset rename), it errors rather than writing a partial site, so a
// broken deploy is caught in the golden test and in CI, not in production. The
// generator is a pure-stdlib + two-internal-import leaf — it copies the already-built,
// byte-pinned embedded bytes the handlers produce and never invokes a network fetch, a
// database, or a `go build -GOOS=js`.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/iscc/iscc-monitor/internal/verifier"
	"github.com/iscc/iscc-monitor/internal/web"
)

// defaultOutDir is the output directory used when -out is not given. The Pages deploy
// publishes this directory's tree to monitor.iscc.codes.
const defaultOutDir = "dist"

func main() {
	out := flag.String("out", defaultOutDir, "output directory for the rendered static site")
	flag.Parse()
	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, "verifier-site:", err)
		os.Exit(1)
	}
	fmt.Println("verifier-site: wrote static site to", *out)
}

// generate renders the complete Surface-C static site into outDir: index.html from
// verifier.Handler and every /_ds/ asset from web.Handler, each written at its URL
// path so the on-disk tree matches what the page fetches. It enumerates the woff2
// binaries from the served fonts.css rather than hardcoding them, so the font set
// tracks fonts.css. It fails closed — a non-200 from any handler aborts before a
// partial site is written — so a future asset rename is caught here, not in production.
func generate(outDir string) error {
	index, err := render(verifier.Handler(), "/")
	if err != nil {
		return fmt.Errorf("render index.html: %w", err)
	}
	if err := writeFile(filepath.Join(outDir, "index.html"), index); err != nil {
		return err
	}

	ds := web.Handler()
	fontsCSS, err := render(ds, web.FontsCSSPath)
	if err != nil {
		return fmt.Errorf("render %s: %w", web.FontsCSSPath, err)
	}

	paths := []string{
		web.TokensPath,
		web.FontsCSSPath,
		web.WasmExecPath,
		web.WasmVerifyPath,
		web.LogoPath,
	}
	paths = append(paths, fontPaths(string(fontsCSS))...)

	for _, p := range paths {
		body, err := render(ds, p)
		if err != nil {
			return fmt.Errorf("render %s: %w", p, err)
		}
		if err := writeFile(assetPath(outDir, p), body); err != nil {
			return err
		}
	}
	return nil
}

// render drives handler with a GET for urlPath and returns the response body. It fails
// closed on any non-200 status so a renamed or missing asset aborts the generate run
// rather than writing a broken site.
func render(handler http.Handler, urlPath string) ([]byte, error) {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, urlPath, nil))
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("GET %s = %d, want 200", urlPath, rec.Code)
	}
	return rec.Body.Bytes(), nil
}

// fontPaths extracts every src="/_ds/fonts/..." URL named in the served fonts.css, so
// the woff2 binaries are enumerated from the single source of truth (fonts.css) rather
// than hardcoded. It scans only inside the double-quoted src values, the same pattern
// internal/web's TestFontsCSSReferencesEmbeddedSubsets uses.
func fontPaths(css string) []string {
	const marker = `"` + web.Prefix + `fonts/`
	var paths []string
	for rest := css; ; {
		i := strings.Index(rest, marker)
		if i < 0 {
			break
		}
		rest = rest[i+1:] // step past the opening quote
		path, after, ok := strings.Cut(rest, `"`)
		if !ok {
			break
		}
		paths = append(paths, path)
		rest = after
	}
	return paths
}

// assetPath maps a served /_ds/ URL path to its on-disk location under outDir, turning
// the URL path's slashes into directory separators (web.Prefix becomes a real
// subdirectory). The URL path is used verbatim so the on-disk layout matches what the
// page fetches at runtime — a static host serves files at their path.
func assetPath(outDir, urlPath string) string {
	return filepath.Join(outDir, filepath.FromSlash(strings.TrimPrefix(urlPath, "/")))
}

// writeFile creates the parent directories and writes data to path. It is the one disk
// write site, so the generate flow stays free of repeated MkdirAll/WriteFile pairs.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
