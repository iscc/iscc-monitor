// Tests for the Surface-C static-site generator: generate renders into a temp dir and
// the assertions check the deployable tree. index.html exists, is non-empty, carries
// the client-side loader markers, and inherits the no-CDN body guarantee (re-asserted
// on the generated file for defense in depth); every /_ds/ asset is materialized at its
// URL path; at least one woff2 binary lands under /_ds/fonts/; and the generated
// verify.wasm's SHA-256 equals web.WasmVerifyHash, proving the generator copies the
// byte-pinned artifact rather than rebuilding it.
package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iscc/iscc-monitor/internal/web"
)

// TestGenerate renders the full static site into a temp dir and asserts the deployable
// tree: index.html with the loader markers and no CDN reference, every /_ds/ asset, at
// least one woff2 binary, and the SRI-pinned verify.wasm hash.
func TestGenerate(t *testing.T) {
	out := t.TempDir()
	if err := generate(out); err != nil {
		t.Fatalf("generate: %v", err)
	}

	// index.html exists, is non-empty, and carries the client-side loader markers — so
	// the deployed page is the live-wiring artifact the golden tests gate, not a stub.
	index := readNonEmpty(t, filepath.Join(out, "index.html"))
	for _, marker := range []string{"/_ds/wasm_exec.js", "/_ds/verify.wasm", "isccVerifyInclusion", "URLSearchParams"} {
		if !strings.Contains(string(index), marker) {
			t.Errorf("index.html missing loader marker %q", marker)
		}
	}

	// The generated page inherits the no-CDN body guarantee; re-assert it on disk so a
	// future template drift that leaks a CDN URL is caught in the deployable artifact.
	for _, banned := range []string{"jsdelivr", "http://", "https://", "cdn."} {
		if strings.Contains(string(index), banned) {
			t.Errorf("index.html references external origin %q", banned)
		}
	}

	// Every named /_ds/ asset is materialized at its URL path and is non-empty.
	for _, p := range []string{
		web.TokensPath, web.FontsCSSPath, web.WasmExecPath, web.WasmVerifyPath, web.LogoPath,
	} {
		readNonEmpty(t, assetPath(out, p))
	}

	// At least one woff2 binary lands under /_ds/fonts/ (enumerated from fonts.css).
	matches, err := filepath.Glob(filepath.Join(out, "_ds", "fonts", "*.woff2"))
	if err != nil {
		t.Fatalf("glob fonts: %v", err)
	}
	if len(matches) == 0 {
		t.Error("no /_ds/fonts/*.woff2 written")
	}

	// The generated verify.wasm is the byte-pinned artifact — its SHA-256 equals
	// web.WasmVerifyHash — proving the generator copies the embedded bytes, not rebuilds.
	wasm := readNonEmpty(t, assetPath(out, web.WasmVerifyPath))
	if got := fmt.Sprintf("%x", sha256.Sum256(wasm)); got != web.WasmVerifyHash {
		t.Errorf("generated verify.wasm sha256 = %s, want web.WasmVerifyHash %s", got, web.WasmVerifyHash)
	}
}

// readNonEmpty reads path, failing the test if it is missing or empty, and returns its
// bytes.
func readNonEmpty(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if len(data) == 0 {
		t.Fatalf("%s is empty", path)
	}
	return data
}
