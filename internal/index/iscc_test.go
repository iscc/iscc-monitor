// Tests for the pure ISCC-IDv1 decoder. The golden vectors are grounded in
// external ground truth, not in the implementation's own arithmetic:
//
//   - The realm-0 vector is the hub's own published example. iscc-hub's
//     iscc_hub/schema.py documents the resolved metadata URL
//     "https://gateway.iscc.io/iscc_id/maighfecjmopmiab" for an ISCC-ID, and
//     describes the layout as a 64-bit identifier = 52-bit timestamp + 12-bit hub
//     ID. Uppercasing that example gives "MAIGHFECJMOPMIAB". Decoding it with the
//     standard RFC 4648 base32 codec (independently, via Python's base64.b32decode
//     in the derivation that produced these expectations) yields header bytes
//     0x60 0x10 → MainType 6 (ID), SubType 0 (realm 0), Version 1 — and body
//     0x6394824b1cf62001 → hub_id 1, timestamp 1751831876325218 µs (2025-07-06).
//   - The realm-1 vector is constructed from first principles per ADR-0010's
//     layout: header 0x61 0x10 (MainType 6, SubType 1 = operational realm,
//     Version 1) over the same timestamp with hub_id 2, base32-encoding to
//     "MEIGHFECJMOPMIAC". Its expected fields are the inputs to that construction.
//
// Neither expected value is computed by calling Decode/encode.
package index

import (
	"strings"
	"testing"
)

func TestDecodeGoldenVectors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want ISCCID
	}{
		{
			// iscc-hub schema.py example, uppercased; realm 0 (test/sandbox).
			name: "realm0-hub1-hub-example",
			in:   "MAIGHFECJMOPMIAB",
			want: ISCCID{Realm: 0, HubID: 1, Timestamp: 1751831876325218},
		},
		{
			// Same id with the optional human-readable "ISCC:" prefix.
			name: "realm0-hub1-with-prefix",
			in:   "ISCC:MAIGHFECJMOPMIAB",
			want: ISCCID{Realm: 0, HubID: 1, Timestamp: 1751831876325218},
		},
		{
			// Constructed per ADR-0010: realm 1 (operational), non-zero hub slot.
			name: "realm1-hub2-constructed",
			in:   "MEIGHFECJMOPMIAC",
			want: ISCCID{Realm: 1, HubID: 2, Timestamp: 1751831876325218},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode(tc.in)
			if err != nil {
				t.Fatalf("Decode(%q) returned error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("Decode(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestDecodeRoundTrip(t *testing.T) {
	// Round-trip both realms and a non-zero hub slot through encode→Decode so the
	// realm 0 vs realm 1 nibble and a real slot are both exercised. The inputs are
	// the ground truth; Decode must return exactly them.
	cases := []ISCCID{
		{Realm: 0, HubID: 1, Timestamp: 1751831876325218},
		{Realm: 1, HubID: 2, Timestamp: 1751831876325218},
		{Realm: 0, HubID: 4095, Timestamp: 0},    // max hub slot, zero timestamp
		{Realm: 1, HubID: 0, Timestamp: 1 << 51}, // zero slot, near-max timestamp
	}
	for _, want := range cases {
		s := encode(want.Realm, want.HubID, want.Timestamp)
		got, err := Decode(s)
		if err != nil {
			t.Fatalf("Decode(encode(%+v)=%q) error: %v", want, s, err)
		}
		if got != want {
			t.Fatalf("round-trip %q: got %+v, want %+v", s, got, want)
		}
	}
}

func TestDecodeRejectsMalformed(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"prefix-only", "ISCC:"},
		{"too-short", "MAIGHFEC"},
		{"too-long", "MAIGHFECJMOPMIABXX"},
		{"non-base32-chars", "MAIGHFECJMOPMIA1"}, // '1' and '0','8','9' are not in the alphabet
		{"lowercase", "maighfecjmopmiab"},        // ISCC base32 is uppercase only
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decode(%q) panicked: %v", tc.in, r)
				}
			}()
			if _, err := Decode(tc.in); err == nil {
				t.Fatalf("Decode(%q) = nil error, want non-nil", tc.in)
			}
		})
	}
}

// TestCanonicalAcceptedRepresentations asserts every accepted spelling of an
// ISCC-IDv1 maps to the single canonical form, and that the result is stable when
// canonicalized again. The forms come from IEP-0001 (canonical, lowercase URI) and
// the draft IEP-0015 (did:iscc).
func TestCanonicalAcceptedRepresentations(t *testing.T) {
	const want = "ISCC:MAIGHFECJMOPMIAB"
	cases := []struct {
		name string
		in   string
	}{
		{"canonical", "ISCC:MAIGHFECJMOPMIAB"},
		{"bare", "MAIGHFECJMOPMIAB"},
		{"bare-lowercase", "maighfecjmopmiab"},
		{"bare-mixed-case", "MaIgHfEcJmOpMiAb"},
		{"uri-lowercase", "iscc:maighfecjmopmiab"},
		{"lowercase-prefix-uppercase-body", "iscc:MAIGHFECJMOPMIAB"},
		{"uppercase-prefix-lowercase-body", "ISCC:maighfecjmopmiab"},
		{"did", "did:iscc:maighfecjmopmiab"},
		{"did-uppercase", "DID:ISCC:MAIGHFECJMOPMIAB"},
		{"surrounding-spaces", "  ISCC:MAIGHFECJMOPMIAB  "},
		{"surrounding-newline", "ISCC:MAIGHFECJMOPMIAB\r\n"},
		{"surrounding-non-breaking-space", string(rune(0xA0)) + "MAIGHFECJMOPMIAB" + string(rune(0xA0))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Canonical(tc.in)
			if got != want {
				t.Fatalf("Canonical(%q) = %q, want %q", tc.in, got, want)
			}
			if again := Canonical(got); again != want {
				t.Fatalf("Canonical(%q) = %q, want it stable at %q", got, again, want)
			}
		})
	}
}

// TestCanonicalLeavesOtherStringsUnchanged asserts a string that is not a valid
// ISCC-IDv1 comes back byte-for-byte, so an opaque id keeps matching verbatim and
// Decode still rejects it. The look-alike cases hold letters that Unicode case
// mapping folds onto base32 letters ("ı" to "I", "ſ" to "S"): accepting them would
// let a different string pass as this ID.
func TestCanonicalLeavesOtherStringsUnchanged(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"whitespace-only", "  "},
		{"padded-junk", "  not-an-iscc-id  "},
		{"opaque-prefixed", "ISCC:LEAF00000003"},
		{"too-short", "maighfec"},
		{"non-base32", "maighfecjmopmia1"},
		{"not-an-iscc-id-type", "iscc:eaawyal5vr274rqt"},
		{"double-prefix", "ISCC:ISCC:MAIGHFECJMOPMIAB"},
		{"did-with-iscc-prefix", "did:iscc:iscc:maighfecjmopmiab"},
		{"double-did", "did:iscc:did:iscc:maighfecjmopmiab"},
		{"other-did-method", "did:web:maighfecjmopmiab"},
		{"inner-space", "ISCC: MAIGHFECJMOPMIAB"},
		{"hyphenated", "MAIGHFEC-JMOPMIAB"},
		{"lookalike-dotless-i-in-body", "maıghfecjmopmiab"},
		{"lookalike-dotless-i-in-prefix", "ıscc:maighfecjmopmiab"},
		{"lookalike-long-s-in-prefix", "iſcc:maighfecjmopmiab"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Canonical(tc.in); got != tc.in {
				t.Fatalf("Canonical(%q) = %q, want the input unchanged", tc.in, got)
			}
		})
	}
}

// TestCanonicalKeepsDecodedFields asserts canonicalizing never changes which ID a
// string names: the realm, hub_id and timestamp match the strict decode of the
// canonical spelling.
func TestCanonicalKeepsDecodedFields(t *testing.T) {
	want := ISCCID{Realm: 1, HubID: 2, Timestamp: 1751831876325218}
	got, err := Decode(Canonical("did:iscc:meighfecjmopmiac"))
	if err != nil {
		t.Fatalf("Decode(Canonical(...)) error: %v", err)
	}
	if got != want {
		t.Fatalf("decoded %+v, want %+v", got, want)
	}
}

func TestDecodeWrongMainType(t *testing.T) {
	// A well-formed 16-char base32 string whose header MainType nibble is not 6
	// must be rejected even though it decodes cleanly to 10 bytes. Build one whose
	// first nibble is 0 (a different MainType) via encode-then-rewrite is overkill;
	// instead use a constructed string whose first byte's high nibble is 0.
	// "AAAAAAAAAAAAAAAA" decodes to all-zero bytes → MainType nibble 0.
	if _, err := Decode("AAAAAAAAAAAAAAAA"); err == nil {
		t.Fatal("Decode of MainType-0 string returned nil error, want rejection")
	}
}

func TestDecodeWrongVersion(t *testing.T) {
	// Header MainType 6 (ID) but Version nibble 0, not 1: byte0=0x60, byte1=0x00.
	// Encode that header over an all-zero body and confirm Decode rejects it.
	raw := []byte{0x60, 0x00, 0, 0, 0, 0, 0, 0, 0, 0}
	s := iscBase32.EncodeToString(raw)
	if _, err := Decode(s); err == nil {
		t.Fatalf("Decode(%q) with Version 0 returned nil error, want rejection", s)
	}
	// Sanity: the same header with Version 1 (0x10) must decode fine, proving the
	// rejection above is specifically the version check, not a length/codec issue.
	raw[1] = 0x10
	ok := iscBase32.EncodeToString(raw)
	if _, err := Decode(ok); err != nil {
		t.Fatalf("Decode(%q) with Version 1 errored: %v", ok, err)
	}
}

func TestDecodeRejectsNonzeroLength(t *testing.T) {
	// A canonical ISCC-IDv1 has a Length nibble of 0 (a 64-bit body). Header
	// MainType 6 (ID) and Version 1 but Length nibble 1 (byte0=0x60, byte1=0x11)
	// is non-canonical and must be rejected before the body is read — otherwise
	// bytes 2:10 are mis-interpreted as a 64-bit body and the wrong leaf is proved.
	raw := []byte{0x60, 0x11, 0, 0, 0, 0, 0, 0, 0, 0}
	s := iscBase32.EncodeToString(raw)
	if _, err := Decode(s); err == nil {
		t.Fatalf("Decode(%q) with Length nibble 1 returned nil error, want rejection", s)
	}
	// The issue/handoff name this literal explicitly: byte1 = 0x11.
	if _, err := Decode("MAIQAAAAAAAAAAAA"); err == nil {
		t.Fatal(`Decode("MAIQAAAAAAAAAAAA") returned nil error, want rejection`)
	}
	// Sanity: the same header with Length nibble 0 (0x10) must decode fine, proving
	// the rejection above is specifically the Length check, not a length/codec issue.
	raw[1] = 0x10
	ok := iscBase32.EncodeToString(raw)
	if _, err := Decode(ok); err != nil {
		t.Fatalf("Decode(%q) with Length nibble 0 errored: %v", ok, err)
	}
}

func TestDecodeNeverPanicsOnJunk(t *testing.T) {
	// Cheap fuzz-style insurance: random/short/garbage input must error, never
	// panic. Vary length around the 16-char boundary and across the alphabet.
	junk := []string{
		"", "Z", "ISCC", "ISCC:Z", strings.Repeat("A", 15), strings.Repeat("A", 17),
		strings.Repeat("=", 16), "================", "????????????????", "MAIG HFEC JMOP MIAB",
	}
	for _, in := range junk {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decode(%q) panicked: %v", in, r)
				}
			}()
			_, _ = Decode(in)
		}()
	}
}
