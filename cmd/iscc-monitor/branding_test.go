// Tests for the operator co-branding wiring: loadOperatorLogo accepts each supported
// logo type and fails startup on a missing, empty, oversize, unsupported, or
// mismatched file; site maps the validated config onto chrome.Site; and buildMux
// serves the operator logo, and renders it on the masthead, only when one is
// configured.
package main

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/iscc/iscc-monitor/internal/chrome"
	"github.com/iscc/iscc-monitor/internal/config"
	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/store"
	"github.com/iscc/iscc-monitor/internal/web"
)

// testSVG is a minimal operator logo.
const testSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`

// writeFile writes data to name in a fresh temp dir and returns its path.
func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// pngBytes encodes a 1x1 PNG.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestLoadOperatorLogo(t *testing.T) {
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 16)...)
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 16)...)
	for _, tc := range []struct {
		name, file, contentType string
		data                    []byte
	}{
		{"svg", "logo.svg", "image/svg+xml", []byte(testSVG)},
		{"uppercase extension", "LOGO.SVG", "image/svg+xml", []byte(testSVG)},
		{"png", "logo.png", "image/png", pngBytes(t)},
		{"webp", "logo.webp", "image/webp", webp},
		{"jpeg", "logo.jpeg", "image/jpeg", jpeg},
		{"jpg", "logo.jpg", "image/jpeg", jpeg},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logo, err := loadOperatorLogo(writeFile(t, tc.file, tc.data))
			if err != nil {
				t.Fatalf("loadOperatorLogo: %v", err)
			}
			if logo.ContentType != tc.contentType || !bytes.Equal(logo.Data, tc.data) {
				t.Errorf("logo = %s with %d bytes, want %s with %d bytes", logo.ContentType, len(logo.Data), tc.contentType, len(tc.data))
			}
		})
	}
}

func TestLoadOperatorLogoNone(t *testing.T) {
	logo, err := loadOperatorLogo("")
	if err != nil || len(logo.Data) != 0 || logo.ContentType != "" {
		t.Errorf("loadOperatorLogo(\"\") = %+v, %v, want the zero Logo and no error", logo, err)
	}
}

func TestLoadOperatorLogoRejects(t *testing.T) {
	big := append([]byte(testSVG), bytes.Repeat([]byte(" "), operatorLogoMaxBytes)...)
	for _, tc := range []struct {
		name, path string
	}{
		{"unsupported type", writeFile(t, "logo.gif", []byte("GIF89a"))},
		{"no extension", writeFile(t, "logo", []byte(testSVG))},
		{"missing file", filepath.Join(t.TempDir(), "missing.svg")},
		{"empty file", writeFile(t, "empty.svg", nil)},
		{"oversize file", writeFile(t, "big.svg", big)},
		{"svg without an svg element", writeFile(t, "text.svg", []byte("hello"))},
		{"png holding svg", writeFile(t, "fake.png", []byte(testSVG))},
		{"jpeg holding png", writeFile(t, "fake.jpg", pngBytes(t))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if logo, err := loadOperatorLogo(tc.path); err == nil {
				t.Errorf("loadOperatorLogo(%q) = %s with %d bytes, want an error", tc.path, logo.ContentType, len(logo.Data))
			}
		})
	}
}

func TestSiteFromConfig(t *testing.T) {
	cfg := config.Config{
		Instance:     "monitor.example.com",
		Operator:     "Example realm line",
		RealmName:    "ISCC mainnet",
		OperatorName: "Example GmbH",
		OperatorURL:  "https://example.com/",
		FooterLinks:  []config.Link{{Label: "Imprint", URL: "https://example.com/imprint"}},
	}
	logo := chrome.Logo{Data: []byte(testSVG), ContentType: "image/svg+xml"}
	want := chrome.Site{
		Instance:     "monitor.example.com",
		Operator:     "Example realm line",
		Realm:        "ISCC mainnet",
		OperatorName: "Example GmbH",
		OperatorURL:  "https://example.com/",
		OperatorLogo: logo,
		Links:        []chrome.Link{{Label: "Imprint", URL: "https://example.com/imprint"}},
	}
	if got := site(cfg, logo); !reflect.DeepEqual(got, want) {
		t.Errorf("site(cfg) = %+v, want %+v", got, want)
	}
}

// muxGet drives one GET through the full buildMux over an empty store.
func muxGet(t *testing.T, s chrome.Site, path string) *httptest.ResponseRecorder {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "branding.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rec := httptest.NewRecorder()
	buildMux(st, nil, nil, metrics.New(), s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestBuildMuxServesOperatorLogo(t *testing.T) {
	s := chrome.Site{
		OperatorName: "Example GmbH",
		OperatorLogo: chrome.Logo{Data: []byte(testSVG), ContentType: "image/svg+xml"},
	}
	rec := muxGet(t, s, web.OperatorLogoPath)
	if rec.Code != http.StatusOK || rec.Body.String() != testSVG {
		t.Fatalf("GET %s = %d %q, want 200 with the logo", web.OperatorLogoPath, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'none'") {
		t.Errorf("Content-Security-Policy = %q, want a default-src 'none' policy", got)
	}
	home := muxGet(t, s, "/").Body.String()
	if !strings.Contains(home, `src="/_ds/operator-logo" alt="Example GmbH"`) {
		t.Errorf("dashboard masthead missing the operator logo\n%s", home)
	}
}

func TestBuildMuxWithoutOperatorLogo(t *testing.T) {
	if rec := muxGet(t, chrome.Site{}, web.OperatorLogoPath); rec.Code != http.StatusNotFound {
		t.Errorf("GET %s with no logo = %d, want 404", web.OperatorLogoPath, rec.Code)
	}
	if home := muxGet(t, chrome.Site{}, "/").Body.String(); strings.Contains(home, "chrome-cobrand") {
		t.Errorf("dashboard renders a co-brand with no logo configured\n%s", home)
	}
}
