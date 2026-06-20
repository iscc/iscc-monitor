// Golden + error tests for the pure did:web document parser. The parsed pubkey,
// fed through verifierKey(origin, pub), must byte-match the recorded oracle
// vectors so the resolve -> vkey chain stays a single coherent golden. Fixtures
// in testdata/ are captured snapshots (see next.md re: the live sb1 rotation) and
// must not be hand-edited.
package didweb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDIDDocument(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		origin  string
		want    string
	}{
		{
			name:    "sb0",
			fixture: "sb0.iscc.id_did.json",
			origin:  "sb0.iscc.id/log",
			want:    "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5",
		},
		{
			name:    "sb1",
			fixture: "sb1.amlet.id_did.json",
			origin:  "sb1.amlet.id/log",
			want:    "sb1.amlet.id/log+22b08f3e+ATo2ruguSdJGh11PS76osrQf6OZKrufzzwH/HMwE3a8/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tc.fixture))
			if err != nil {
				t.Fatalf("read fixture %q: %v", tc.fixture, err)
			}
			key, err := parseDIDDocument(data)
			if err != nil {
				t.Fatalf("parseDIDDocument(%q) error: %v", tc.fixture, err)
			}
			if len(key.PublicKey) != 32 {
				t.Fatalf("pubkey is %d bytes, want 32", len(key.PublicKey))
			}
			got := verifierKey(tc.origin, key.PublicKey)
			if got != tc.want {
				t.Errorf("verifierKey(%q) = %q, want %q", tc.origin, got, tc.want)
			}
			// Live docs omit CID 1.0 validity fields -> "currently valid".
			if !key.Revoked.IsZero() {
				t.Errorf("Revoked = %v, want zero (fixture omits it)", key.Revoked)
			}
			if !key.ValidFrom.IsZero() || !key.ValidUntil.IsZero() {
				t.Errorf("ValidFrom/ValidUntil = %v/%v, want zero (fixture omits them)", key.ValidFrom, key.ValidUntil)
			}
		})
	}
}

func TestParseDIDDocumentErrors(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{
			name: "malformed JSON",
			data: `{"id": "did:web:sb0.iscc.id", `,
		},
		{
			name: "empty verificationMethod with string assertionMethod",
			data: `{"id": "did:web:sb0.iscc.id", "verificationMethod": [], "assertionMethod": ["did:web:sb0.iscc.id#z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"]}`,
		},
		{
			name: "missing publicKeyMultibase",
			data: `{"id": "did:web:sb0.iscc.id", "verificationMethod": [{"id": "did:web:sb0.iscc.id#k", "type": "Multikey", "controller": "did:web:sb0.iscc.id"}], "assertionMethod": ["did:web:sb0.iscc.id#k"]}`,
		},
		{
			name: "no assertionMethod",
			data: `{"id": "did:web:sb0.iscc.id", "verificationMethod": [{"id": "did:web:sb0.iscc.id#k", "publicKeyMultibase": "z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ"}]}`,
		},
		{
			name: "bad publicKeyMultibase",
			data: `{"id": "did:web:sb0.iscc.id", "verificationMethod": [{"id": "did:web:sb0.iscc.id#k", "publicKeyMultibase": "not-a-key"}], "assertionMethod": ["did:web:sb0.iscc.id#k"]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseDIDDocument([]byte(tc.data)); err == nil {
				t.Errorf("parseDIDDocument(%s) = nil error, want error", tc.name)
			}
		})
	}
}

// TestParseDIDDocumentInlineAssertion covers the inline-object assertionMethod
// form (an alternative to the #fragment string reference), parsing to the same
// sb0 golden verifier key.
func TestParseDIDDocumentInlineAssertion(t *testing.T) {
	doc := map[string]any{
		"id": "did:web:sb0.iscc.id",
		"assertionMethod": []any{
			map[string]any{
				"id":                 "did:web:sb0.iscc.id#z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ",
				"type":               "Ed25519VerificationKey2020",
				"controller":         "did:web:sb0.iscc.id",
				"publicKeyMultibase": "z6MkqbHELZopsq6eKrn6qxiAgRoVvwp2Vp7mPfKsvrbYmGwJ",
			},
		},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal doc: %v", err)
	}
	key, err := parseDIDDocument(data)
	if err != nil {
		t.Fatalf("parseDIDDocument(inline) error: %v", err)
	}
	got := verifierKey("sb0.iscc.id/log", key.PublicKey)
	want := "sb0.iscc.id/log+40b74463+AaV+ivnly67hhzQSQfGqCBP3PlOV2NBcmfGyzGdE2ZE5"
	if got != want {
		t.Errorf("verifierKey = %q, want %q", got, want)
	}
}
