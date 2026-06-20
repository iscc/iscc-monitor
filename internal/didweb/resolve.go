// This file parses a hub's /.well-known/did.json bytes into the Ed25519
// verifier key the hub signs checkpoints with. It is the pure (no net/os/sql)
// half of did:web resolution (ADR-0009): bytes in, key + validity metadata out.
// The HTTP fetch and the did:web:<domain> -> URL mapping live in the follower
// step's outbound-fetch seam, not here.
//
// Key selection follows the hub's signing convention: checkpoints are signed as
// an assertion (cryptosuite eddsa-jcs-2022, proofPurpose assertionMethod), so we
// take the verification method referenced by assertionMethod. The extracted
// publicKeyMultibase feeds the oracle-verified pubkeyFromDID/verifierKey path
// unchanged, keeping the trust root single-sourced.
package didweb

import (
	"encoding/json"
	"fmt"
	"time"
)

// verificationMethod is a CID 1.0 / W3C DID verification method entry.
//
// Only the fields the monitor needs are decoded; unknown fields are ignored.
// validFrom/validUntil/revoked are the CID 1.0 validity window the follower may
// later enforce against the observation time; they are absent in the live docs.
type verificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
	ValidFrom          string `json:"validFrom"`
	ValidUntil         string `json:"validUntil"`
	Revoked            string `json:"revoked"`
}

// didDocument is the subset of a did:web document the monitor parses.
//
// assertionMethod entries are either an inline verification-method object or a
// string DID-URL reference (#fragment) into verificationMethod, so it is decoded
// late as json.RawMessage.
type didDocument struct {
	ID                 string               `json:"id"`
	VerificationMethod []verificationMethod `json:"verificationMethod"`
	AssertionMethod    []json.RawMessage    `json:"assertionMethod"`
}

// DIDKey is the parsed assertion-method signing key from a did:web document.
//
// PublicKey is the 32-byte Ed25519 key (multicodec header already stripped by
// pubkeyFromDID); Multibase is its z6Mk… did:key form; the validity fields carry
// the parsed CID 1.0 timestamps (zero when the document omits them, meaning
// "currently valid" — the follower decides enforcement later, not this parser).
type DIDKey struct {
	Multibase  string
	PublicKey  []byte
	ValidFrom  time.Time
	ValidUntil time.Time
	Revoked    time.Time
}

// parseDIDDocument parses did.json bytes into the hub's assertion-method key.
//
// It JSON-decodes the document, resolves the verification method referenced by
// assertionMethod (inline object or #fragment reference into verificationMethod),
// extracts publicKeyMultibase through the oracle-verified pubkeyFromDID, and
// surfaces any CID 1.0 validity timestamps. It returns a wrapped error on invalid
// JSON, a missing or unresolvable assertion method, a missing publicKeyMultibase,
// or a pubkeyFromDID failure. It performs no I/O and no now-vs-window enforcement.
func parseDIDDocument(data []byte) (DIDKey, error) {
	var doc didDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return DIDKey{}, fmt.Errorf("parseDIDDocument: invalid JSON: %w", err)
	}
	vm, err := assertionKey(doc)
	if err != nil {
		return DIDKey{}, fmt.Errorf("parseDIDDocument: %w", err)
	}
	if vm.PublicKeyMultibase == "" {
		return DIDKey{}, fmt.Errorf("parseDIDDocument: verification method %q has no publicKeyMultibase", vm.ID)
	}
	pub, err := pubkeyFromDID(vm.PublicKeyMultibase)
	if err != nil {
		return DIDKey{}, fmt.Errorf("parseDIDDocument: %w", err)
	}
	return DIDKey{
		Multibase:  vm.PublicKeyMultibase,
		PublicKey:  pub,
		ValidFrom:  parseTime(vm.ValidFrom),
		ValidUntil: parseTime(vm.ValidUntil),
		Revoked:    parseTime(vm.Revoked),
	}, nil
}

// assertionKey selects the verification method named by the document's first
// assertionMethod entry.
//
// An entry may be an inline verification-method object or a JSON string holding a
// DID-URL reference; a string is resolved to the matching verificationMethod[i].id.
// It returns an error when assertionMethod is empty, malformed, or references an
// id that is not present in verificationMethod.
func assertionKey(doc didDocument) (verificationMethod, error) {
	if len(doc.AssertionMethod) == 0 {
		return verificationMethod{}, fmt.Errorf("document has no assertionMethod")
	}
	raw := doc.AssertionMethod[0]

	var ref string
	if err := json.Unmarshal(raw, &ref); err == nil {
		for _, vm := range doc.VerificationMethod {
			if vm.ID == ref {
				return vm, nil
			}
		}
		return verificationMethod{}, fmt.Errorf("assertionMethod references unknown verification method %q", ref)
	}

	var inline verificationMethod
	if err := json.Unmarshal(raw, &inline); err != nil {
		return verificationMethod{}, fmt.Errorf("assertionMethod entry is neither a DID-URL reference nor a verification method: %w", err)
	}
	if inline.PublicKeyMultibase == "" && inline.ID == "" {
		return verificationMethod{}, fmt.Errorf("inline assertionMethod entry has no key material")
	}
	return inline, nil
}

// parseTime parses an RFC 3339 timestamp, returning the zero time on empty or
// unparseable input.
//
// Absence of a CID 1.0 validity field means "no constraint" here; the follower,
// not this pure parser, decides what an unparseable value implies.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
