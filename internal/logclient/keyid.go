// This file recovers the signed-note key id from a verifier-key string. The
// verifier key didweb.VerifierKey produces is "<name>+<keyid:08x>+<base64>"; its
// middle "+<hex>+" field IS the C2SP signed-note keyhash (the BE-uint32 of
// SHA-256(name||0x0A||0x01||pub)[:4]). The follower caches that key id alongside
// the resolved did:web public key, so it recovers the id from the vkey string
// rather than re-deriving it through the package-private didweb.keyID.
//
// The base64 tail uses base64.StdEncoding, whose alphabet includes '+' and '/',
// so the tail itself can contain a '+'. Splitting on at most two '+' (SplitN with
// n=3) isolates the keyhash field exactly and leaves any '+' in the tail intact.
package logclient

import (
	"fmt"
	"strconv"
	"strings"
)

// KeyIDFromVerifier parses the signed-note key id out of a verifier-key string.
//
// vkey is the "<name>+<keyid:08x>+<base64>" string didweb.VerifierKey returns. It
// splits on at most two '+' (so a '+' inside the base64 tail is preserved),
// requires exactly three fields, and parses the middle field as a hex uint32. It
// returns the key id, or a wrapped error when the string is not "<name>+<hex>+
// <base64>" or the middle field is not a 32-bit hex number.
func KeyIDFromVerifier(vkey string) (uint32, error) {
	parts := strings.SplitN(vkey, "+", 3)
	if len(parts) != 3 {
		return 0, fmt.Errorf("KeyIDFromVerifier: %q is not <name>+<keyid>+<base64>", vkey)
	}
	id, err := strconv.ParseUint(parts[1], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("KeyIDFromVerifier: %q has a non-hex key id %q: %w", vkey, parts[1], err)
	}
	return uint32(id), nil
}
