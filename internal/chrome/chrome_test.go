// Tests for the shared chrome: Site.View applies the static fallbacks and carries the
// co-branding, Verifier returns the fixed verifier-app chrome, the version links to
// its exact source, and the rendered partials show (and escape) exactly what the view
// holds — the operator co-brand only when configured, the instance links only on an
// instance.
package chrome

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/iscc/iscc-monitor/internal/version"
	"github.com/iscc/iscc-monitor/internal/web"
)

// page wraps both partials so a test renders the header and footer of one view.
var page = template.Must(template.Must(template.New("page").Parse(
	`{{template "siteHeader" .}}{{template "siteFooter" .}}`)).Parse(Source))

// render executes both partials over v and returns the markup.
func render(t *testing.T, v View) string {
	t.Helper()
	var buf bytes.Buffer
	if err := page.Execute(&buf, v); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// fullSite is a Site with every co-branding field set.
func fullSite() Site {
	return Site{
		Instance:     "monitor.example.com",
		Operator:     "Example realm line",
		Realm:        "ISCC mainnet",
		OperatorName: "Example GmbH",
		OperatorURL:  "https://example.com/",
		OperatorLogo: Logo{Data: []byte("<svg></svg>"), ContentType: "image/svg+xml"},
		Links: []Link{
			{Label: "Imprint", URL: "https://example.com/imprint"},
			{Label: "Privacy", URL: "https://example.com/privacy"},
		},
	}
}

// setVersion sets version.Version for one test and restores it afterwards.
func setVersion(t *testing.T, v string) {
	t.Helper()
	old := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = old })
}

func TestSiteViewFallbacks(t *testing.T) {
	v := Site{}.View()
	if v.Instance != instanceFallback || v.Operator != operatorFallback {
		t.Errorf("identity = %q / %q, want the static fallbacks", v.Instance, v.Operator)
	}
	if v.Sub != subEvidence || v.VerifyHref != verifierURL || !v.InstanceLinks {
		t.Errorf("view = %+v, want the evidence subtitle, the verifier link, and instance links", v)
	}
	if v.OperatorName != "" || v.OperatorURL != "" || v.OperatorLogo != "" || len(v.Links) != 0 {
		t.Errorf("view = %+v, want no operator co-brand", v)
	}
}

func TestSiteViewCarriesBranding(t *testing.T) {
	s := fullSite()
	v := s.View()
	if v.Instance != s.Instance || v.Operator != s.Operator {
		t.Errorf("identity = %q / %q, want the configured values", v.Instance, v.Operator)
	}
	if v.OperatorName != s.OperatorName || v.OperatorURL != s.OperatorURL || len(v.Links) != 2 {
		t.Errorf("view = %+v, want the configured operator name, URL, and links", v)
	}
	if v.OperatorLogo != web.OperatorLogoPath {
		t.Errorf("OperatorLogo = %q, want %q", v.OperatorLogo, web.OperatorLogoPath)
	}
}

func TestVerifierView(t *testing.T) {
	v := Verifier()
	if v.Instance != "monitor.iscc.codes" || v.VerifyHref != "/" || v.InstanceLinks {
		t.Errorf("Verifier() = %+v, want the fixed verifier identity, a self link, and no instance links", v)
	}
	if v.OperatorName != "" || v.OperatorLogo != "" || len(v.Links) != 0 {
		t.Errorf("Verifier() = %+v, want no operator co-brand", v)
	}
}

func TestSourceURL(t *testing.T) {
	const repo = "https://github.com/iscc/iscc-monitor"
	for _, tc := range []struct{ version, want string }{
		{"v0.1.0+2b73c0c", repo + "/releases/tag/v0.1.0"},
		{"v0.1.0", repo + "/releases/tag/v0.1.0"},
		{"v1.2.3-rc.1+abc1234", repo + "/releases/tag/v1.2.3-rc.1"},
		{"2b73c0c", repo + "/commit/2b73c0c"},
		{"2b73c0c4a1f3e5d7b9c0a2e4f6d8b0c2e4f6a8b0", repo + "/commit/2b73c0c4a1f3e5d7b9c0a2e4f6d8b0c2e4f6a8b0"},
		{"dev", ""},
		{"", ""},
		{"v0.1", ""},
		{"abc", ""},
		{"2B73C0C", ""},
		{"v0.1.0 ", ""},
	} {
		if got := sourceURL(tc.version); got != tc.want {
			t.Errorf("sourceURL(%q) = %q, want %q", tc.version, got, tc.want)
		}
	}
}

func TestDisplayVersion(t *testing.T) {
	for _, tc := range []struct{ version, want string }{
		{"2b73c0c4a1f3e5d7b9c0a2e4f6d8b0c2e4f6a8b0", "2b73c0c"},
		{"2b73c0c", "2b73c0c"},
		{"v0.1.0+2b73c0c", "v0.1.0+2b73c0c"},
		{"dev", "dev"},
	} {
		if got := displayVersion(tc.version); got != tc.want {
			t.Errorf("displayVersion(%q) = %q, want %q", tc.version, got, tc.want)
		}
	}
}

func TestRenderUnconfigured(t *testing.T) {
	setVersion(t, "dev")
	out := render(t, Site{}.View())
	for _, want := range []string{
		`<a class="chrome-home" href="/"><img class="chrome-logo" src="/_ds/iscc-logo.svg" alt="ISCC Monitor home"`,
		`<div class="chrome-instance">monitor instance</div>`,
		`<a class="chrome-verify" href="https://monitor.iscc.codes/">`,
		`ISCC Trust &amp; Transparency Monitor · <span class="site-footer-version">dev</span>`,
		`Open source software by the ISCC Foundation · Apache-2.0 · <a href="https://github.com/iscc/iscc-monitor">Source</a>`,
		`<a href="/docs">API docs</a> · <a href="https://monitor.iscc.codes/">Verify independently ↗</a>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("unconfigured chrome missing %q\n%s", want, out)
		}
	}
	for _, absent := range []string{"chrome-cobrand", "site-footer-operator", "Operated by"} {
		if strings.Contains(out, absent) {
			t.Errorf("unconfigured chrome contains %q\n%s", absent, out)
		}
	}
}

func TestRenderCoBranded(t *testing.T) {
	out := render(t, fullSite().View())
	for _, want := range []string{
		`<span class="chrome-cobrand-label">Operated by</span>`,
		`<a class="chrome-cobrand-link" href="https://example.com/"><img class="chrome-cobrand-logo" src="/_ds/operator-logo" alt="Example GmbH"></a>`,
		`<div class="chrome-instance">monitor.example.com</div>`,
		`<div class="chrome-operator">Example realm line</div>`,
		`Operated by <a href="https://example.com/">Example GmbH</a> · <a href="https://example.com/imprint">Imprint</a> · <a href="https://example.com/privacy">Privacy</a>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("co-branded chrome missing %q\n%s", want, out)
		}
	}
}

func TestRenderOperatorVariants(t *testing.T) {
	for _, tc := range []struct {
		name string
		site Site
		want string
	}{
		{"name without url or logo", Site{OperatorName: "Example GmbH"},
			`<div class="site-footer-operator">Operated by Example GmbH</div>`},
		{"links without a name", Site{Links: []Link{{"Imprint", "https://a.example/i"}, {"Privacy", "https://a.example/p"}}},
			`<div class="site-footer-operator"><a href="https://a.example/i">Imprint</a> · <a href="https://a.example/p">Privacy</a></div>`},
		{"logo without a url", Site{OperatorName: "Example GmbH", OperatorLogo: Logo{Data: []byte("x")}},
			`<span class="chrome-cobrand-label">Operated by</span>
      <img class="chrome-cobrand-logo" src="/_ds/operator-logo" alt="Example GmbH">`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if out := render(t, tc.site.View()); !strings.Contains(out, tc.want) {
				t.Errorf("chrome missing %q\n%s", tc.want, out)
			}
		})
	}
}

func TestRenderEscapesOperatorValues(t *testing.T) {
	out := render(t, Site{
		Instance:     `<b>x</b>`,
		OperatorName: `<script>alert(1)</script>`,
		OperatorURL:  `javascript:alert(1)`,
		Links:        []Link{{Label: `"><img src=x>`, URL: `javascript:alert(2)`}},
	}.View())
	for _, banned := range []string{"<script>", "<b>x</b>", `"><img src=x>`, "javascript:"} {
		if strings.Contains(out, banned) {
			t.Errorf("chrome renders unescaped %q\n%s", banned, out)
		}
	}
}

func TestRenderVerifier(t *testing.T) {
	out := render(t, Verifier())
	if !strings.Contains(out, `<a class="chrome-verify" href="/">`) {
		t.Errorf("verifier chrome missing its self link\n%s", out)
	}
	for _, absent := range []string{`href="/docs"`, "Verify independently", "Operated by"} {
		if strings.Contains(out, absent) {
			t.Errorf("verifier chrome contains %q\n%s", absent, out)
		}
	}
}

func TestRenderVersionLink(t *testing.T) {
	setVersion(t, "v0.1.0+2b73c0c")
	out := render(t, Site{}.View())
	want := `<span class="site-footer-version"><a href="https://github.com/iscc/iscc-monitor/releases/tag/v0.1.0">v0.1.0&#43;2b73c0c</a></span>`
	if !strings.Contains(out, want) {
		t.Errorf("footer missing release link %q\n%s", want, out)
	}
}
