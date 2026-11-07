// Package chrome renders the document chrome every server-rendered page shares: the
// masthead (the ISCC signature, the "Trust & Transparency Monitor" mark, this instance's
// identity, an optional operator co-brand, and the link to the independent verifier) and
// the footer (the running software version linked to its exact source, the open-source
// credit, the instance links, and the operator's own links such as an imprint or privacy
// policy).
//
// Source holds both partials. A page template associates it next to its own source and
// invokes {{template "siteHeader" .Chrome}} and {{template "siteFooter" .Chrome}}, where
// .Chrome is a View. Site is the operator-configured identity and co-branding of an
// instance; Site.View resolves it into the View an instance page renders, and Verifier
// returns the fixed View of the monitor-agnostic verifier app (monitor.iscc.codes),
// which carries no operator identity. The partials' styles live in the shared
// /_ds/site.css stylesheet.
//
// Every operator-supplied string reaches the page through html/template's contextual
// escaping, and every operator URL is validated at startup (internal/config), so a
// configured value can never inject markup.
//
// The oracle/conformance gate is N/A: this is static-markup rendering, touching no
// signature, RFC-6962, Merkle, did:web, fsck, or proof path.
package chrome

import (
	_ "embed"
	"regexp"

	"github.com/iscc/iscc-monitor/internal/version"
	"github.com/iscc/iscc-monitor/internal/web"
)

// Source is the embedded template source defining the "siteHeader" and "siteFooter"
// partials. Page templates associate it with template.Must(parent.Parse(chrome.Source)).
//
//go:embed chrome.html
var Source string

// Static masthead copy. instanceFallback and operatorFallback stand in for an unset
// ISCC_MONITOR_INSTANCE / ISCC_MONITOR_OPERATOR, so an unconfigured binary shows a
// neutral placeholder rather than claiming a specific domain or operator.
const (
	instanceFallback = "monitor instance"
	operatorFallback = "independent Trust & Transparency service · ISCC-Hub network"
	subEvidence      = "Evidence of record · ISCC-Hub network"
	verifierURL      = "https://monitor.iscc.codes/"
	repoURL          = "https://github.com/iscc/iscc-monitor"
)

// Link is one operator-configured footer link, such as an imprint or privacy policy.
type Link struct {
	Label string
	URL   string
}

// Logo is an operator logo file the binary serves at web.OperatorLogoPath. The zero
// value means no logo is configured.
type Logo struct {
	Data        []byte
	ContentType string
}

// Site is the operator-configured identity and co-branding of a monitor instance. Every
// field is optional; View applies the static fallbacks for an empty Instance or Operator.
type Site struct {
	// Instance is this deployment's domain, shown on the masthead.
	Instance string
	// Operator is the free-form line beneath the instance on the masthead.
	Operator string
	// Realm is the human realm name the dashboard shows in its register heading.
	Realm string
	// OperatorName names the organization running this instance ("Operated by …").
	OperatorName string
	// OperatorURL is the operator's website; it links the operator name and logo.
	OperatorURL string
	// OperatorLogo is the operator's logo, shown on the masthead beside the ISCC mark.
	OperatorLogo Logo
	// Links are the operator's footer links, in configured order.
	Links []Link
}

// View is the render context of the siteHeader and siteFooter partials.
type View struct {
	// Sub is the masthead line beneath the "Trust & Transparency Monitor" mark.
	Sub string
	// Instance and Operator are the resolved instance-identity lines.
	Instance string
	Operator string
	// VerifyHref is the target of the masthead's verifier link.
	VerifyHref string
	// OperatorName, OperatorURL, and Links carry the operator co-brand.
	OperatorName string
	OperatorURL  string
	Links        []Link
	// OperatorLogo is the served path of the operator logo, or empty when none is set.
	OperatorLogo string
	// Version is the running software version as displayed; VersionURL links it to its
	// exact source, or is empty when the version names no published source.
	Version    string
	VersionURL string
	// InstanceLinks shows the footer's links to this instance's API docs and to the
	// independent verifier. The verifier app itself omits them.
	InstanceLinks bool
}

// View resolves s into the chrome an instance page renders, applying the static
// fallbacks for an empty Instance or Operator.
func (s Site) View() View {
	v := base()
	v.Instance = fallback(s.Instance, instanceFallback)
	v.Operator = fallback(s.Operator, operatorFallback)
	v.VerifyHref = verifierURL
	v.OperatorName = s.OperatorName
	v.OperatorURL = s.OperatorURL
	v.Links = s.Links
	v.InstanceLinks = true
	if len(s.OperatorLogo.Data) > 0 {
		v.OperatorLogo = web.OperatorLogoPath
	}
	return v
}

// Verifier returns the chrome of the monitor-agnostic verifier app: a fixed identity, a
// verifier link to its own root, and a footer without instance or operator links.
func Verifier() View {
	v := base()
	v.Instance = "monitor.iscc.codes"
	v.Operator = "independent verifier app · audits any monitor instance"
	v.VerifyHref = "/"
	return v
}

// base returns the fields every View shares: the evidence subtitle and the version.
func base() View {
	return View{
		Sub:        subEvidence,
		Version:    displayVersion(version.Version),
		VersionURL: sourceURL(version.Version),
	}
}

// fallback returns value, or def when value is empty.
func fallback(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

var (
	// releaseRE matches a release version, "vX.Y.Z" with an optional pre-release and
	// build metadata; the first group is the tag.
	releaseRE = regexp.MustCompile(`^(v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)(?:\+[0-9A-Za-z.-]+)?$`)
	// commitRE matches a bare git commit hash, abbreviated or full.
	commitRE = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

// sourceURL returns the published source of version v: the release page for a release
// tag, the commit page for a commit hash, or empty for anything else (such as "dev").
func sourceURL(v string) string {
	if m := releaseRE.FindStringSubmatch(v); m != nil {
		return repoURL + "/releases/tag/" + m[1]
	}
	if commitRE.MatchString(v) {
		return repoURL + "/commit/" + v
	}
	return ""
}

// displayVersion shortens a full commit hash to the conventional seven characters and
// returns any other version unchanged.
func displayVersion(v string) string {
	if len(v) == 40 && commitRE.MatchString(v) {
		return v[:7]
	}
	return v
}
