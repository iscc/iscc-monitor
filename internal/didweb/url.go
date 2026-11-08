// This file maps a did:web:<method-specific-id> identifier to the HTTPS URL of
// its did.json document, per the W3C did:web method spec. It is the pure
// string->URL half of did:web resolution (ADR-0009): the follower's
// outbound-fetch seam feeds this URL to its injected Fetcher. No net/net-http/
// os/sql imports, so the package stays WASM-shareable.
package didweb

import (
	"fmt"
	"net/url"
	"strings"
)

// didWebPrefix is the method prefix every did:web identifier carries.
const didWebPrefix = "did:web:"

// DocumentURL maps a did:web identifier to its did.json HTTPS URL.
//
// The method-specific id is colon-separated: the first segment is the
// percent-encoded host[:port], any later segments are path components. With no
// path segments the URL is https://<host>/.well-known/did.json; with path
// segments it is https://<host>/<seg1>/<seg2>/…/did.json. It returns a wrapped
// error when the did:web: prefix is missing, the identifier is empty, or a
// segment is not valid percent-encoding. It performs no I/O.
func DocumentURL(did string) (string, error) {
	if !strings.HasPrefix(did, didWebPrefix) {
		return "", fmt.Errorf("DocumentURL: %q is not a did:web identifier", did)
	}
	msid := strings.TrimPrefix(did, didWebPrefix)
	if msid == "" {
		return "", fmt.Errorf("DocumentURL: %q has an empty method-specific id", did)
	}
	segments := strings.Split(msid, ":")
	decoded := make([]string, len(segments))
	for i, seg := range segments {
		d, err := url.PathUnescape(seg)
		if err != nil {
			return "", fmt.Errorf("DocumentURL: invalid percent-encoding in segment %q: %w", seg, err)
		}
		decoded[i] = d
	}
	host := decoded[0]
	if host == "" {
		return "", fmt.Errorf("DocumentURL: %q has an empty host", did)
	}
	path := decoded[1:]
	if len(path) == 0 {
		return "https://" + host + "/.well-known/did.json", nil
	}
	return "https://" + host + "/" + strings.Join(path, "/") + "/did.json", nil
}
