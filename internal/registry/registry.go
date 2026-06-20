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
	"strings"
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
