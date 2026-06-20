// Package didweb resolves a hub's signing key from its did:web/did:key material
// and derives the C2SP signed-note verifier key off-the-shelf tlog-tiles tooling
// (Tessera fsck, golang.org/x/mod/sumdb/note) expects.
//
// This file is a faithful port of .claude/derive_vkey.py: it maps a hub's
// published multibase Ed25519 public key (z6Mk… did:key) to the
// "<name>+<keyid>+<base64>" verifier-key string. The bytes are load-bearing —
// they must match the Go reference verifier exactly — so the derivation is
// golden-tested byte-for-byte against the Python oracle for both live testnet
// hubs. The signed-note name MUST equal the hub's served origin (e.g.
// sb0.iscc.id/log), never the bare domain.
package didweb

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
)

// b58Alphabet is the base58btc alphabet (Bitcoin ordering).
const b58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// algEd25519 is the signed-note algorithm identifier for Ed25519.
const algEd25519 = 0x01

// b58decode decodes a base58btc string to bytes, preserving leading-zero bytes.
//
// Each leading '1' in the input maps to one 0x00 prefix byte, matching the
// Python reference b58decode. It returns an error on any character outside the
// base58btc alphabet.
func b58decode(s string) ([]byte, error) {
	num := new(big.Int)
	radix := big.NewInt(58)
	for _, ch := range s {
		idx := strings.IndexRune(b58Alphabet, ch)
		if idx < 0 {
			return nil, fmt.Errorf("b58decode: invalid character %q", ch)
		}
		num.Mul(num, radix)
		num.Add(num, big.NewInt(int64(idx)))
	}
	body := num.Bytes() // big-endian, no leading zeros
	pad := len(s) - len(strings.TrimLeft(s, "1"))
	out := make([]byte, pad+len(body))
	copy(out[pad:], body)
	return out, nil
}

// pubkeyFromDID extracts the 32-byte Ed25519 public key from a z6Mk did:key.
//
// It strips the leading 'z' multibase prefix, base58btc-decodes the rest, and
// asserts the ed25519-pub multicodec header (0xED 0x01) before returning bytes
// [2:34]. It returns an error when the multibase prefix, multicodec header, or
// key length is wrong.
func pubkeyFromDID(z string) ([]byte, error) {
	if !strings.HasPrefix(z, "z") {
		return nil, fmt.Errorf("pubkeyFromDID: missing multibase 'z' prefix")
	}
	raw, err := b58decode(z[1:])
	if err != nil {
		return nil, fmt.Errorf("pubkeyFromDID: %w", err)
	}
	if len(raw) < 34 {
		return nil, fmt.Errorf("pubkeyFromDID: decoded key is %d bytes, expected >=34", len(raw))
	}
	if raw[0] != 0xED || raw[1] != 0x01 {
		return nil, fmt.Errorf("pubkeyFromDID: not ed25519-pub multicodec: %02x%02x", raw[0], raw[1])
	}
	return raw[2:34], nil
}

// keyID returns the signed-note 4-byte key id as a big-endian uint32.
//
// It is the first four bytes of SHA-256(name || 0x0A || 0x01 || pub), where 0x0A
// is the literal newline byte and 0x01 is the Ed25519 algorithm identifier.
func keyID(name string, pub []byte) uint32 {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte{0x0A})
	h.Write([]byte{algEd25519})
	h.Write(pub)
	return binary.BigEndian.Uint32(h.Sum(nil)[:4])
}

// verifierKey returns the signed-note verifier-key string for a name and pubkey.
//
// The format is "<name>+<keyid:08x>+<base64(0x01 || pub)>" using standard
// padded base64, matching .claude/derive_vkey.py byte-for-byte. The name is the
// hub origin (e.g. sb0.iscc.id/log).
func verifierKey(name string, pub []byte) string {
	encoded := append([]byte{algEd25519}, pub...)
	return fmt.Sprintf("%s+%08x+%s", name, keyID(name, pub), base64.StdEncoding.EncodeToString(encoded))
}
