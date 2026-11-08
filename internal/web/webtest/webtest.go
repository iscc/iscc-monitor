// Package webtest holds test helpers for the monitor's server-rendered pages. Its one
// helper pins the no-CDN invariant (ADR-0010): a page loads every stylesheet, script,
// image, and font from its own origin, so it renders offline and from one binary with
// no third-party dependency in the page-load path. Hyperlinks are navigation, not page
// loads, so a page may link to another origin, such as the source repository or an
// operator's imprint.
package webtest

import (
	"regexp"
	"testing"
)

// externalLoad matches a resource reference to another origin: an absolute or
// protocol-relative URL in a src, srcset, poster, or data attribute, in a <link> href,
// in a CSS url(), or after an @import.
var externalLoad = regexp.MustCompile(`(?i)(?:\b(?:src|srcset|poster|data)\s*=\s*["']?\s*|<link\b[^>]*?\bhref\s*=\s*["']?\s*|url\(\s*["']?\s*|@import\s+["']?\s*)(?:https?:)?//`)

// ExternalLoads returns every resource reference in body that points to another
// origin; an empty result means the page loads only same-origin resources.
func ExternalLoads(body string) []string {
	return externalLoad.FindAllString(body, -1)
}

// AssertNoExternalLoads fails t for each resource reference in body that points to
// another origin. name identifies the page in the failure message.
func AssertNoExternalLoads(t testing.TB, name, body string) {
	t.Helper()
	for _, ref := range ExternalLoads(body) {
		t.Errorf("%s loads a resource from another origin: %q", name, ref)
	}
}
