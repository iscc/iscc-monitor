// Tests for the third-party notices generator. They run against the real module graph
// and module cache, so a dependency change without a regenerated file fails the gate.
package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const repoRoot = "../.."

// TestCommittedNoticesAreCurrent fails when THIRD_PARTY_NOTICES.txt needs regenerating.
func TestCommittedNoticesAreCurrent(t *testing.T) {
	got, err := generate(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(repoRoot, "THIRD_PARTY_NOTICES.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Error("THIRD_PARTY_NOTICES.txt is stale: run `mise run notices`")
	}
}

// TestNoticesCoverShippedComponents checks each kind of source reaches the document.
func TestNoticesCoverShippedComponents(t *testing.T) {
	got, err := generate(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Go standard library and runtime",
		"modernc.org/sqlite v",
		"SQLite Is Public Domain",             // a secondary license file in a module
		"--- LICENSE-3RD-PARTY.md ---",        // modernc.org/libc's bundled musl terms
		"gopkg.in/yaml.v3 v",                  // an Apache-licensed module with a NOTICE
		"--- NOTICE ---",                      // that module's NOTICE is carried over
		"Stoplight Elements 9.0.23",           // vendored browser bundle
		"--- elements.min.js.LICENSE.txt ---", // notices of libraries inside the bundle
		"SIL Open Font License",               // vendored fonts
	} {
		if !strings.Contains(got, want) {
			t.Errorf("notices are missing %q", want)
		}
	}
	if strings.Contains(got, "\r") {
		t.Error("notices contain carriage returns")
	}
}

// TestNoticesExcludeTestOnlyModules keeps modules that never ship out of the document.
func TestNoticesExcludeTestOnlyModules(t *testing.T) {
	got, err := generate(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "github.com/iscc/iscc-lib/packages/go") {
		t.Error("notices include iscc-lib, which only a test imports")
	}
}

// TestParseModules reads listFormat lines and skips packages without a module.
func TestParseModules(t *testing.T) {
	got := parseModules("\na.io/x\tv1.0.0\t/cache/x\r\n\nb.io/y\tv2.0.0\t/cache/y\n")
	want := []module{
		{path: "a.io/x", version: "v1.0.0", dir: "/cache/x"},
		{path: "b.io/y", version: "v2.0.0", dir: "/cache/y"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("parseModules = %v, want %v", got, want)
	}
}

// TestUniqueModulesSortsAndDeduplicates covers a module shared by two builds.
func TestUniqueModulesSortsAndDeduplicates(t *testing.T) {
	a := module{path: "a.io/x", version: "v1.0.0", dir: "/a"}
	b := module{path: "b.io/y", version: "v1.0.0", dir: "/b"}
	got := uniqueModules([]module{b, a, b})
	if !slices.Equal(got, []module{a, b}) {
		t.Errorf("uniqueModules = %v", got)
	}
}

// TestLicenseFilesMatchesVariants finds license and notice names and ignores the rest.
func TestLicenseFilesMatchesVariants(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"LICENSE.txt", "COPYING", "NOTICE", "SQLITE-LICENSE", "main.go", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "licenses"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := licenseFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range got {
		names = append(names, filepath.Base(f))
	}
	want := []string{"COPYING", "LICENSE.txt", "NOTICE", "SQLITE-LICENSE"}
	if !slices.Equal(names, want) {
		t.Errorf("licenseFiles = %v, want %v", names, want)
	}
}

// TestLicenseFilesRejectsUnlicensedModule refuses a module with unknown terms.
func TestLicenseFilesRejectsUnlicensedModule(t *testing.T) {
	if _, err := licenseFiles(t.TempDir()); err == nil {
		t.Error("expected an error for a directory without a license file")
	}
}

// TestNormalizeUnifiesLineEndings keeps output identical across platforms.
func TestNormalizeUnifiesLineEndings(t *testing.T) {
	if got := normalize("\r\nline one\r\nline two\r\n\r\n"); got != "line one\nline two" {
		t.Errorf("normalize = %q", got)
	}
}
