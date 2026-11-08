// Tests that the published configuration reference (docs/operate/configuration.md, served at
// monitor.iscc.codes) names exactly the environment variables this package reads, so the
// operator documentation cannot silently fall behind a new or removed key.
package config

import (
	"os"
	"regexp"
	"slices"
	"testing"
)

// envKeyPattern matches an ISCC_MONITOR_* environment variable name.
var envKeyPattern = regexp.MustCompile(`ISCC_MONITOR_[A-Z_]+[A-Z]`)

// envKeys returns the sorted, distinct ISCC_MONITOR_* names in the file at path.
func envKeys(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	keys := envKeyPattern.FindAllString(string(b), -1)
	slices.Sort(keys)
	return slices.Compact(keys)
}

func TestConfigurationDocsNameEveryKey(t *testing.T) {
	code := envKeys(t, "config.go")
	docs := envKeys(t, "../../docs/operate/configuration.md")
	if len(code) == 0 {
		t.Fatal("found no ISCC_MONITOR_* keys in config.go")
	}
	for _, k := range code {
		if !slices.Contains(docs, k) {
			t.Errorf("docs/operate/configuration.md does not document %s", k)
		}
	}
	for _, k := range docs {
		if !slices.Contains(code, k) {
			t.Errorf("docs/operate/configuration.md documents %s, which config.go does not read", k)
		}
	}
}
