// Tests for ExternalLoads: every form of cross-origin resource load is found, and
// same-origin loads, hyperlinks, and XML namespaces are not.
package webtest

import "testing"

func TestExternalLoadsFindsCrossOriginLoads(t *testing.T) {
	for _, body := range []string{
		`<script src="https://cdn.example/x.js"></script>`,
		`<script src='//cdn.example/x.js'></script>`,
		`<img src=https://example.com/a.png>`,
		`<img srcset="https://example.com/a.png 2x">`,
		`<video poster="http://example.com/p.jpg"></video>`,
		`<object data="https://example.com/x.svg"></object>`,
		`<link rel="stylesheet" href="https://fonts.example/css">`,
		`<link rel="icon" type="image/svg+xml" href="//example.com/favicon.svg">`,
		`@font-face { src: url("https://fonts.example/a.woff2"); }`,
		`.x { background: url(//example.com/grain.png); }`,
		`@import "https://example.com/base.css";`,
	} {
		if got := ExternalLoads(body); len(got) != 1 {
			t.Errorf("ExternalLoads(%q) = %q, want one match", body, got)
		}
	}
}

func TestExternalLoadsIgnoresSameOriginAndLinks(t *testing.T) {
	for _, body := range []string{
		`<link rel="stylesheet" href="/_ds/tokens.css">`,
		`<img class="chrome-logo" src="/_ds/iscc-logo.svg" alt="ISCC">`,
		`@font-face { src: url("/_ds/fonts/readex-pro-400.woff2"); }`,
		`<a href="https://github.com/iscc/iscc-monitor">Source</a>`,
		`<a class="chrome-verify" href="https://monitor.iscc.codes/">verify</a>`,
		`<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
		`<span data-state="verified"></span>`,
	} {
		if got := ExternalLoads(body); len(got) != 0 {
			t.Errorf("ExternalLoads(%q) = %q, want none", body, got)
		}
	}
}
