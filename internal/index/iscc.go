// Package index decodes a self-describing ISCC-IDv1 string into its component
// fields: the realm, the 12-bit hub_id (issuing-hub slot), and the 52-bit
// microsecond timestamp. The monitor's realm-wide inclusion certificate is keyed
// on the id alone (ADR-0010): decoding (realm, hub_id) is what later resolves the
// issuing hub via the registry, so a wrong decode would prove the wrong leaf —
// this decoder is the trust root of that path.
//
// The codec is an external fact owned by ISO 24138 / iscc/iscc-core's iscc_id.py
// (mirroring the hub side iscc_hub/iscc_id.py), ported here, not invented. An
// ISCC-IDv1 is an 80-bit code = a 16-bit header + a 64-bit body, written as 16
// RFC 4648 base32 characters (uppercase alphabet ABCDEFGHIJKLMNOPQRSTUVWXYZ234567,
// no '=' padding), optionally prefixed with "ISCC:". The header's four nibbles are
// MainType, SubType, Version, Length; for an ISCC-IDv1 MainType = 6 (ID) and
// Version = 1, and the SubType nibble is the realm (0 = test/sandbox, 1 =
// operational). The body is a big-endian uint64 split into timestamp = body >> 12
// (52 bits, µs since epoch) and hub_id = body & 0xFFF (low 12 bits, slot 0-4095).
//
// This is a pure leaf: it performs no I/O and imports no net/sql/os, so it stays
// shareable with the WASM verifier. It fails closed — malformed input always
// returns a descriptive error and never panics.
package index

import (
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
)

// mainTypeID is the ISCC MainType nibble for an ISCC-ID (per ISO 24138).
const mainTypeID = 6

// versionV1 is the only ISCC-ID Version this package supports.
const versionV1 = 1

// hubIDMask isolates the low 12 bits of the body — the hub_id slot (0-4095).
const hubIDMask = 0xFFF

// timestampShift is the bit width of the hub_id field; the timestamp is the
// remaining high 52 bits of the body.
const timestampShift = 12

// iscPrefix is the optional human-readable scheme prefix on an ISCC string.
const iscPrefix = "ISCC:"

// bodyChars is the number of base32 characters in an ISCC-IDv1: an 80-bit code
// (16-bit header + 64-bit body) encodes to exactly 16 characters with no padding.
const bodyChars = 16

// iscBase32 is the ISCC base32 codec: the RFC 4648 uppercase alphabet with no
// '=' padding, as used by iscc/iscc-core's encode_base32/decode_base32.
var iscBase32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// ISCCID is a decoded ISCC-IDv1: the realm (SubType nibble), the 12-bit hub_id
// (issuing-hub slot within the realm), and the 52-bit microsecond timestamp.
type ISCCID struct {
	// Realm is the SubType nibble: 0 = test/sandbox, 1 = operational.
	Realm uint8
	// HubID is the low-12-bit issuing-hub slot (0-4095) the certificate resolves.
	HubID uint16
	// Timestamp is the high-52-bit value: microseconds since the Unix epoch.
	Timestamp uint64
}

// Decode parses an ISCC-IDv1 string into its realm, hub_id, and timestamp.
//
// It accepts an optional "ISCC:" prefix, base32-decodes the 16 body characters,
// validates the header is an ISCC-IDv1 (MainType = ID, Version = 1), reads the
// SubType nibble as the realm, and splits the big-endian uint64 body into
// timestamp = body >> 12 and hub_id = body & 0xFFF.
//
// It fails closed: an empty string, non-base32 characters, the wrong length, a
// header that is not an ISCC-IDv1, or a body shorter than eight bytes each return
// a descriptive error rather than panicking.
func Decode(isccID string) (ISCCID, error) {
	body := strings.TrimPrefix(isccID, iscPrefix)
	if len(body) != bodyChars {
		return ISCCID{}, fmt.Errorf("index: ISCC-IDv1 has %d base32 chars, want %d: %q", len(body), bodyChars, isccID)
	}
	raw, err := iscBase32.DecodeString(body)
	if err != nil {
		return ISCCID{}, fmt.Errorf("index: base32 decode %q: %w", isccID, err)
	}
	if len(raw) < 10 {
		return ISCCID{}, fmt.Errorf("index: decoded ISCC is %d bytes, want 10: %q", len(raw), isccID)
	}
	mainType := raw[0] >> 4
	if mainType != mainTypeID {
		return ISCCID{}, fmt.Errorf("index: MainType nibble %d is not an ISCC-ID (want %d): %q", mainType, mainTypeID, isccID)
	}
	version := raw[1] >> 4
	if version != versionV1 {
		return ISCCID{}, fmt.Errorf("index: ISCC-ID Version nibble %d is unsupported (want %d): %q", version, versionV1, isccID)
	}
	realm := raw[0] & 0xF
	value := binary.BigEndian.Uint64(raw[2:10])
	return ISCCID{
		Realm:     realm,
		HubID:     uint16(value & hubIDMask),
		Timestamp: value >> timestampShift,
	}, nil
}

// encode builds the canonical ISCC-IDv1 string (no "ISCC:" prefix) for a realm,
// hub_id, and timestamp. It is unexported because the monitor only ever decodes
// ids it is handed; it exists solely to prove Decode round-trips a constructed
// vector in the test.
func encode(realm uint8, hubID uint16, timestamp uint64) string {
	raw := make([]byte, 10)
	raw[0] = mainTypeID<<4 | realm&0xF
	raw[1] = versionV1 << 4
	value := timestamp<<timestampShift | uint64(hubID)&hubIDMask
	binary.BigEndian.PutUint64(raw[2:10], value)
	return iscBase32.EncodeToString(raw)
}
