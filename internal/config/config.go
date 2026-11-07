// Package config turns a flat key/value lookup into the validated, typed startup
// values the iscc-monitor binary consumes: the SQLite database path, the
// realm-document path, and the two poll intervals the follower loop runs on.
//
// It is a pure leaf: it parses and validates values already in hand and performs
// no I/O. The raw lookup is injected as a get(key) closure (the binary backs it
// with os.LookupEnv next step), so the package never touches the environment,
// the filesystem, or the network and is fully unit-testable. Reading the realm
// document and opening the database belong to the binary; config only carries the
// realm-document path as a string and the database path as a string.
//
// Configuration keys (all read through get):
//
//	ISCC_MONITOR_DB     required — path to the network's SQLite file (ADR-0007:
//	                    one database file per network, so this is a single path).
//	ISCC_MONITOR_REALM  required — the realm-membership document source: an
//	                    http(s):// URL to the authoritative iscc-hub Hub-List YAML
//	                    (carrying each hub's real embedded hub_id, re-fetched hourly)
//	                    or a filesystem path. config only carries the string; the
//	                    binary fetches/reads and parses it.
//	ISCC_MONITOR_NORMAL optional — clean-hub poll interval (default 5m).
//	ISCC_MONITOR_FROZEN optional — frozen-hub backed-off poll interval, the
//	                    evidence-only re-poll cadence (ADR-0006); default 1h.
//	ISCC_MONITOR_ADDR   optional — listen address for the /metrics HTTP server;
//	                    default :9464.
//	ISCC_MONITOR_INSTANCE   optional — this instance's domain, rendered on the
//	                    server-rendered masthead (default empty).
//	ISCC_MONITOR_OPERATOR   optional — the operator/realm line beneath the
//	                    instance on the masthead (default empty).
//	ISCC_MONITOR_REALM_NAME optional — the human realm NAME ("ISCC mainnet")
//	                    shown in the ledger subtitle, deliberately distinct from
//	                    the required ISCC_MONITOR_REALM realm-document PATH so a
//	                    filename never leaks into the subtitle (default empty).
//	ISCC_MONITOR_OPERATOR_NAME optional — the name of the organization running this
//	                    instance, shown as "Operated by <name>" in the footer and as
//	                    the operator logo's text alternative (default empty).
//	ISCC_MONITOR_OPERATOR_URL  optional — the operator's website, an absolute
//	                    http(s) URL linking the operator name and logo.
//	ISCC_MONITOR_OPERATOR_LOGO optional — filesystem path of the operator's logo
//	                    (SVG, PNG, WebP, or JPEG), shown on the masthead beside the
//	                    ISCC mark. config only carries the path; the binary reads it.
//	ISCC_MONITOR_FOOTER_LINKS  optional — the operator's footer links, such as an
//	                    imprint or privacy policy: "Label=URL" pairs separated by
//	                    "|", e.g. "Imprint=https://example.com/imprint|Privacy=
//	                    https://example.com/privacy". A URL is http(s) or mailto.
//
// The three identity values are free-form display strings carried verbatim; an
// unset key leaves the field empty and the shared masthead applies its static
// fail-safe copy, so they are read through the no-validation optional helper (like
// ISCC_MONITOR_ADDR, but with an empty default). The co-branding values are
// validated: every URL must parse with an allowed scheme, and an operator URL or logo
// requires an operator name, which labels the link and the logo.
//
// Intervals are parsed with time.ParseDuration. The load-bearing cross-check is
// Frozen >= Normal: the follower loop encodes its back-off by polling a frozen
// hub on the longer Frozen interval (loop.go due()), so a Frozen < Normal config
// is rejected rather than silently accepted.
package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Configuration key names read through the injected get closure.
const (
	keyDB     = "ISCC_MONITOR_DB"
	keyRealm  = "ISCC_MONITOR_REALM"
	keyNormal = "ISCC_MONITOR_NORMAL"
	keyFrozen = "ISCC_MONITOR_FROZEN"
	keyAddr   = "ISCC_MONITOR_ADDR"
)

// Instance-identity key names: the optional operator-supplied display strings
// rendered on the server-rendered masthead so each deployment is honest about
// who runs it. keyRealmName is deliberately distinct from keyRealm (the required
// realm-document filesystem PATH): the masthead needs the human realm NAME
// ("ISCC mainnet"), not the on-disk path, so overloading the path var would leak
// a filename into the ledger subtitle.
const (
	keyInstance  = "ISCC_MONITOR_INSTANCE"
	keyOperator  = "ISCC_MONITOR_OPERATOR"
	keyRealmName = "ISCC_MONITOR_REALM_NAME"
)

// Co-branding key names: the optional operator name, website, logo file, and footer
// links an instance shows beside the ISCC mark.
const (
	keyOperatorName = "ISCC_MONITOR_OPERATOR_NAME"
	keyOperatorURL  = "ISCC_MONITOR_OPERATOR_URL"
	keyOperatorLogo = "ISCC_MONITOR_OPERATOR_LOGO"
	keyFooterLinks  = "ISCC_MONITOR_FOOTER_LINKS"
)

// linkSeparator separates the "Label=URL" pairs of ISCC_MONITOR_FOOTER_LINKS. A URL
// cannot contain a literal "|" (RFC 3986 requires it percent-encoded), so splitting on
// it never cuts a URL.
const linkSeparator = "|"

// Default poll intervals applied when the corresponding key is absent. Normal is
// the clean-hub cadence; Frozen is the longer evidence-only re-poll cadence for a
// frozen hub (ADR-0006). The defaults satisfy Frozen >= Normal.
const (
	defaultNormal = 5 * time.Minute
	defaultFrozen = time.Hour
)

// defaultAddr is the listen address for the /metrics HTTP server when
// ISCC_MONITOR_ADDR is absent: an exotic non-standard port per the project port
// convention, fixed so the same deployment reuses it across restarts.
const defaultAddr = ":9464"

// Config is the validated, typed startup configuration for the monitor binary.
// DBPath is the single network's SQLite file (ADR-0007), RealmPath the on-disk
// realm-membership document, Normal/Frozen the follower loop's poll intervals
// (Frozen >= Normal encodes the freeze back-off, ADR-0006), and Addr the listen
// address for the /metrics HTTP server (default :9464).
//
// Instance, Operator, and RealmName are the optional free-form masthead-identity
// display strings (empty when their keys are unset); the binary builds the shared
// chrome.Site from them and the masthead applies its static fail-safe copy for any
// empty field. RealmName is the human realm NAME, distinct from RealmPath (the
// realm-document filesystem path).
//
// OperatorName, OperatorURL, OperatorLogo, and FooterLinks are the validated
// co-branding values (empty when unset). OperatorLogo is a filesystem path the binary
// reads at startup.
type Config struct {
	DBPath       string
	RealmPath    string
	Normal       time.Duration
	Frozen       time.Duration
	Addr         string
	Instance     string
	Operator     string
	RealmName    string
	OperatorName string
	OperatorURL  string
	OperatorLogo string
	FooterLinks  []Link
}

// Link is one validated "Label=URL" footer link from ISCC_MONITOR_FOOTER_LINKS.
type Link struct {
	Label string
	URL   string
}

// Load reads the configuration from the injected get closure, applies the
// documented interval defaults, validates, and returns a typed Config.
//
// get(key) returns a value and whether the key was present (the closure the
// binary builds from os.LookupEnv has this shape). DBPath and RealmPath are
// required: an absent or empty value yields a wrapped error naming the missing
// key. The intervals default to Normal=5m and Frozen=1h when absent, are parsed
// with time.ParseDuration (an unparseable value is wrapped and named, never
// swallowed), and must each be positive. The cross-check Frozen >= Normal is
// enforced so the loop's freeze back-off (loop.go due()) actually backs a frozen
// hub off; a Frozen < Normal config is rejected. The co-branding values are
// validated as described in the package documentation.
//
// It fails closed: on any validation failure it returns the zero Config alongside
// a wrapped error so the binary surfaces a precise startup failure rather than a
// half-built config.
func Load(get func(key string) (string, bool)) (Config, error) {
	dbPath, err := required(get, keyDB)
	if err != nil {
		return Config{}, err
	}
	realmPath, err := required(get, keyRealm)
	if err != nil {
		return Config{}, err
	}
	normal, err := duration(get, keyNormal, defaultNormal)
	if err != nil {
		return Config{}, err
	}
	frozen, err := duration(get, keyFrozen, defaultFrozen)
	if err != nil {
		return Config{}, err
	}
	if frozen < normal {
		return Config{}, fmt.Errorf("config: %q (%s) must be >= %q (%s): freeze cadence backs off, never speeds up", keyFrozen, frozen, keyNormal, normal)
	}
	addr := optional(get, keyAddr, defaultAddr)
	instance := optional(get, keyInstance, "")
	operator := optional(get, keyOperator, "")
	realmName := optional(get, keyRealmName, "")
	operatorName := optional(get, keyOperatorName, "")
	operatorURL := optional(get, keyOperatorURL, "")
	operatorLogo := optional(get, keyOperatorLogo, "")
	if operatorURL != "" {
		if err := checkURL(operatorURL, false); err != nil {
			return Config{}, fmt.Errorf("config: key %q: %w", keyOperatorURL, err)
		}
	}
	if (operatorURL != "" || operatorLogo != "") && operatorName == "" {
		return Config{}, fmt.Errorf("config: %q is required when %q or %q is set: it labels the operator link and logo", keyOperatorName, keyOperatorURL, keyOperatorLogo)
	}
	links, err := parseLinks(optional(get, keyFooterLinks, ""))
	if err != nil {
		return Config{}, fmt.Errorf("config: key %q: %w", keyFooterLinks, err)
	}
	return Config{
		DBPath:       dbPath,
		RealmPath:    realmPath,
		Normal:       normal,
		Frozen:       frozen,
		Addr:         addr,
		Instance:     instance,
		Operator:     operator,
		RealmName:    realmName,
		OperatorName: operatorName,
		OperatorURL:  operatorURL,
		OperatorLogo: operatorLogo,
		FooterLinks:  links,
	}, nil
}

// parseLinks parses "Label=URL" pairs separated by linkSeparator. Whitespace around a
// pair, label, or URL is trimmed and empty pairs are skipped, so a trailing separator
// is harmless. Every pair needs a non-empty label and a URL checkURL accepts (http,
// https, or mailto). An empty value yields no links.
func parseLinks(value string) ([]Link, error) {
	var links []Link
	for i, pair := range strings.Split(value, linkSeparator) {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		label, target, ok := strings.Cut(pair, "=")
		label, target = strings.TrimSpace(label), strings.TrimSpace(target)
		if !ok || label == "" {
			return nil, fmt.Errorf("link %d (%q) must be Label=URL", i+1, pair)
		}
		if err := checkURL(target, true); err != nil {
			return nil, fmt.Errorf("link %d (%q): %w", i+1, label, err)
		}
		links = append(links, Link{Label: label, URL: target})
	}
	return links, nil
}

// checkURL accepts an absolute http(s) URL with a host, or, when allowMailto is set, a
// mailto: URL with an address. Restricting the schemes keeps every configured link
// one the browser opens as a page or a mail draft, never a script.
func checkURL(raw string, allowMailto bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	switch {
	case (u.Scheme == "http" || u.Scheme == "https") && u.Host != "":
		return nil
	case allowMailto && u.Scheme == "mailto" && u.Opaque != "":
		return nil
	}
	if allowMailto {
		return fmt.Errorf("%q is not an absolute http(s) or mailto URL", raw)
	}
	return fmt.Errorf("%q is not an absolute http(s) URL", raw)
}

// required returns the value for key, failing closed with a wrapped error naming
// the key when it is absent or empty (whitespace is not trimmed — a path is taken
// verbatim).
func required(get func(key string) (string, bool), key string) (string, error) {
	value, ok := get(key)
	if !ok || value == "" {
		return "", fmt.Errorf("config: required key %q is missing", key)
	}
	return value, nil
}

// optional returns the value for key, falling back to fallback when the key is
// absent or empty. A bad value is not validated here (the listen address surfaces
// at ListenAndServe), matching the optional-key pattern's no-validation-beyond-
// default contract.
func optional(get func(key string) (string, bool), key, fallback string) string {
	value, ok := get(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}

// duration returns the interval for key, falling back to fallback when the key is
// absent. A present value is parsed with time.ParseDuration (an error is wrapped
// and names the key) and must be positive; a non-positive interval is rejected so
// the loop never polls on a zero or negative cadence.
func duration(get func(key string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value, ok := get(key)
	if !ok || value == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("config: key %q: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("config: key %q (%s) must be a positive interval", key, d)
	}
	return d, nil
}
