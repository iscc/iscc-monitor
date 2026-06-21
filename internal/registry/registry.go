// Package registry parses a realm-membership document into the list of hubs a
// monitor follows. The realm registry advertises membership by domain only — no
// keys (ADR-0009): a hub's signing key comes from its did:web document, and
// domain ownership is the hub's cryptographic identity. This package therefore
// carries no key field and rejects anything URL-shaped; it is purely a domain
// list.
//
// The document format is a pilot static document (KISS): one hub domain per
// line. Blank lines and lines whose first non-whitespace character is '#'
// (comments) are ignored; surrounding whitespace on each kept line is trimmed.
// A kept line is a bare host (e.g. "sb0.iscc.id", optionally "host:port"), never
// a URL — a scheme ("://") or a path ('/') is rejected so a URL is never
// silently coerced into a domain.
//
// This is a pure leaf: it parses bytes already in hand and performs no I/O. The
// caller reads the file (or fetches the remote document) and passes the bytes;
// reading is the wiring step's job, not this package's.
package registry

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

// Entry is one hub's realm membership: its bare domain and the derived base URL
// the follower polls. Domain is the host as written in the document (e.g.
// "sb0.iscc.id"), suitable for store.UpsertHub; BaseURL is "https://" + Domain
// (hubs are served over TLS), the scheme+host the follower turns into the log
// origin and verifier key.
type Entry struct {
	Domain  string
	BaseURL string
}

// Parse turns a realm-membership document into hub entries in input order.
//
// It reads the line-based format documented on the package: one domain per
// line, ignoring blank lines and '#' comment lines and trimming surrounding
// whitespace on each kept line. Input order is preserved (no sorting or
// deduping — reconciliation is a separate concern), so the poll loop and tests
// stay deterministic.
//
// It fails closed on a URL-shaped line: a kept line containing a scheme
// ("://") or a path separator ('/') yields a wrapped error naming the offending
// line rather than coercing a URL into a domain. An all-comment or all-blank
// document yields an empty slice and a nil error.
func Parse(data []byte) ([]Entry, error) {
	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain == "" || strings.HasPrefix(domain, "#") {
			continue
		}
		if strings.Contains(domain, "://") {
			return nil, fmt.Errorf("registry: line %q contains a scheme; expected a bare domain", domain)
		}
		if strings.Contains(domain, "/") {
			return nil, fmt.Errorf("registry: line %q contains a path separator; expected a bare domain", domain)
		}
		entries = append(entries, Entry{Domain: domain, BaseURL: "https://" + domain})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("registry: scan document: %w", err)
	}
	return entries, nil
}

// maxHubID is the largest valid 12-bit hub_id slot (4095). The embedded hub_id is
// the low 12 bits of an ISCC-IDv1 body, so a Hub-List slot outside 0-4095 cannot
// be the issuing hub of any id and is rejected fail-closed.
const maxHubID = 4095

// Hub is one entry in the iscc-hub Hub-List: the issuing hub's embedded 12-bit
// hub_id slot (0-4095, the value an ISCC-IDv1 carries in its low 12 bits — NOT
// the surrogate hubs.hub_id PK from store.UpsertHub), the hub's base URL, and
// whether the realm registry still lists it as active. The Hub-List's optional
// pubkey field is deprecated and ignored (ADR-0009): the signing key comes from
// the hub's did:web document, so this type carries no key field.
type Hub struct {
	HubID  uint16 `yaml:"hub_id"`
	URL    string `yaml:"url"`
	Active bool   `yaml:"active"`
}

// HubList is the parsed iscc-hub Hub-List for one network: the schema version,
// the network name (e.g. "testnet"), and the hubs in document order. It maps a
// decoded (realm, hub_id) — from internal/index — to the issuing hub's domain
// via Resolve.
type HubList struct {
	Version int    `yaml:"version"`
	Network string `yaml:"network"`
	Hubs    []Hub  `yaml:"hubs"`
}

// ParseHubList parses an iscc-hub Hub-List YAML document into a HubList.
//
// The document schema is {version, network, hubs:[{hub_id, url, active, pubkey?}]}
// (ADR-0010). The pubkey field, if present, is parsed-and-ignored — keys come
// from did:web (ADR-0009), so it is never read into any type. Hubs are kept in
// document order; a document with zero hubs is valid (an empty Hubs slice, not an
// error), mirroring Parse's all-comment behavior.
//
// Like Parse this is a pure leaf: it parses bytes already in hand and performs no
// I/O — the caller reads the file. It fails closed, returning nil alongside a
// wrapped error naming the fault, on invalid YAML, a hub_id outside 0-4095, a
// duplicate hub_id, or a url that is empty or path-bearing (so a path is never
// silently coerced into a domain).
func ParseHubList(data []byte) (*HubList, error) {
	var hl HubList
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false)
	if err := dec.Decode(&hl); err != nil {
		return nil, fmt.Errorf("registry: parse Hub-List: %w", err)
	}
	seen := make(map[uint16]struct{}, len(hl.Hubs))
	for _, h := range hl.Hubs {
		if h.HubID > maxHubID {
			return nil, fmt.Errorf("registry: hub_id %d is outside 0-%d", h.HubID, maxHubID)
		}
		if _, dup := seen[h.HubID]; dup {
			return nil, fmt.Errorf("registry: duplicate hub_id %d", h.HubID)
		}
		seen[h.HubID] = struct{}{}
		if _, err := hubDomain(h.URL); err != nil {
			return nil, fmt.Errorf("registry: hub_id %d: %w", h.HubID, err)
		}
	}
	return &hl, nil
}

// Resolve maps a 12-bit hub_id slot to the issuing hub's domain (the host of its
// url, scheme stripped — suitable for the /<domain>/log/... mount). ok is true
// for any listed slot, including an inactive hub: the monitor still follows and
// mirrors inactive hubs (ADR-0010) and lets the downstream badge convey
// inactivity, so resolution returns the domain regardless of Active. ok is false
// only for an unknown slot (no such hub_id in the list). A linear scan suffices —
// a realm has at most 4096 slots and a Hub-List a handful of hubs.
func (hl *HubList) Resolve(hubID uint16) (domain string, ok bool) {
	for _, h := range hl.Hubs {
		if h.HubID == hubID {
			domain, err := hubDomain(h.URL)
			if err != nil {
				return "", false
			}
			return domain, true
		}
	}
	return "", false
}

// hubDomain extracts the host (domain, scheme stripped) from a hub url using
// net/url, which is stdlib and WASM-safe (it does not pull in net/http). It fails
// closed on a url that is empty, unparseable, or carries no host (e.g. a bare
// path), so a path is never coerced into a domain.
func hubDomain(rawURL string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", fmt.Errorf("url is empty")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("url %q is not a valid URL: %w", rawURL, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("url %q has no host", rawURL)
	}
	return u.Host, nil
}
