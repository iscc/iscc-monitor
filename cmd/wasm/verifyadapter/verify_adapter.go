// Package verifyadapter is the pure WASM-marshaling adapter for the browser
// verifier. It has NO build tag and NO syscall/js import, so it compiles and is
// unit-tested on linux: it base64-Std-decodes the proof-bundle fields the monitor
// emits (record, proof hashes, root) into the args of the shared
// github.com/iscc/iscc-monitor/internal/proof/verify core and folds that core's
// three-way verdict into a flat (verified, errMsg) result for JavaScript.
//
// The marshaling lives here, untagged and in its own non-main package, precisely
// so the WASM-vs-server verdict parity is provable by an ordinary linux test
// against the same golden vector the core test pins. The syscall/js glue that
// hands these args off from JS lives in the GOOS=js GOARCH=wasm-only cmd/wasm
// main.go, which imports this package.
package verifyadapter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/iscc/iscc-monitor/internal/proof/verify"
)

// maxSafeInteger is JavaScript's Number.MAX_SAFE_INTEGER (2^53 - 1): the largest
// integer a JS Number can represent without losing precision. A leaf index or tree
// size at or above 2^53 cannot round-trip through a JS Number, so the adapter rejects
// it rather than truncate it to a wrong-but-plausible value.
const maxSafeInteger = float64(1<<53 - 1)

// SafeIndex converts a JS Number (read as float64) into a uint64 leaf index/size,
// failing closed when the value is not a non-negative integer within the JS
// safe-integer range. It returns a non-empty errMsg (and a zero value) on any
// violation — NaN, infinity, a fractional value, a negative value, or a value at
// or beyond 2^53 — so the caller folds it into the {verified:false, error:…} result.
// This is the JS→Go integer contract enforced at the one boundary where the
// float→uint64 narrowing would otherwise truncate silently: js.Value.Int() is
// int(v.Float()), so a non-integer JS Number (e.g. 1.9) would verify against the
// wrong-but-truncated leaf, and a value beyond the safe-integer range would round.
func SafeIndex(v float64, name string) (uint64, string) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, "isccVerifyInclusion: " + name + " is not a finite number"
	}
	if v != math.Trunc(v) {
		return 0, "isccVerifyInclusion: " + name + " is not an integer"
	}
	if v < 0 || v > maxSafeInteger {
		return 0, "isccVerifyInclusion: " + name + " is out of safe-integer range"
	}
	return uint64(v), ""
}

// VerifyJSON decodes the base64-Std proof-bundle inputs the monitor emits and
// re-verifies the inclusion proof via the shared verify.VerifyInclusion core,
// folding its three-way verdict into a flat (verified, errMsg) result for JS.
//
// record and root are base64-Std encodings of the raw leaf record bytes and the
// accepted root; proofB64 is the base64-Std-encoded inclusion proof hashes in
// order; index and size locate the leaf in the accepted tree.
//
// The verdict mapping preserves the core's three-way contract so the eventual
// split-view alert can tell a mismatch apart from broken input:
//   - a base64 decode error OR a non-nil verify error (the index >= size
//     precondition) → verified=false with a non-empty errMsg.
//   - otherwise the core's boolean verdict is returned with errMsg=="": a wrong
//     record or a tampered root is verified=false, errMsg=="" — a negative
//     VERDICT, not an error.
func VerifyJSON(record, root string, proofB64 []string, index, size uint64) (verified bool, errMsg string) {
	recordBytes, err := base64.StdEncoding.DecodeString(record)
	if err != nil {
		return false, fmt.Sprintf("decode record: %v", err)
	}
	rootBytes, err := base64.StdEncoding.DecodeString(root)
	if err != nil {
		return false, fmt.Sprintf("decode root: %v", err)
	}
	proof := make([][]byte, len(proofB64))
	for i, h := range proofB64 {
		hashBytes, err := base64.StdEncoding.DecodeString(h)
		if err != nil {
			return false, fmt.Sprintf("decode proof[%d]: %v", i, err)
		}
		proof[i] = hashBytes
	}
	ok, err := verify.VerifyInclusion(recordBytes, index, size, proof, rootBytes)
	if err != nil {
		return false, err.Error()
	}
	return ok, ""
}

// idEnvelope is the minimal view of the canonical log-entry envelope the id-binding
// reads: only the top-level committed iscc_id. It mirrors the iscc_id field of
// logclient.recordEnvelope (Projection's source), so RecordCommitsID binds against
// the SAME committed id the monitor's iscc_index keys on. Unknown members (note,
// the envelope $schema) are dropped by the standard unmarshal.
type idEnvelope struct {
	IsccID string `json:"iscc_id"`
}

// RecordCommitsID reports whether the proof-bundle record actually commits the
// requested ISCC-ID — the id-binding half of a full re-verification. Without it a
// monitor could return a valid-but-unrelated declaration's internally-consistent
// bundle and the browser would render a false green; binding the record's own
// committed iscc_id to the requested id closes that gap.
//
// recordB64 is the base64-Std-encoded canonical record bytes (the same record
// VerifyJSON hashes into the leaf); wantID is the requested ISCC-ID from the
// verifier's ?id= target. Both the committed iscc_id and wantID are canonicalized
// to the "ISCC:"-prefixed form before an exact-bytes compare, so a bare and a
// prefixed request bind identically (the certificate §1 lookup-key idiom).
//
// The three-way verdict contract is preserved so the eventual split-view alert can
// tell a mismatch apart from broken input:
//   - a base64 decode error OR a JSON parse error → ok=false with a non-empty errMsg
//     (broken input is an ERROR).
//   - a well-formed record whose committed iscc_id does NOT match the requested id
//     (including an empty/absent committed iscc_id) → ok=false, errMsg=="" (a negative
//     VERDICT, the same class as a wrong record or a tampered root).
//   - a match → ok=true, errMsg=="".
func RecordCommitsID(recordB64, wantID string) (ok bool, errMsg string) {
	recordBytes, err := base64.StdEncoding.DecodeString(recordB64)
	if err != nil {
		return false, fmt.Sprintf("decode record: %v", err)
	}
	var env idEnvelope
	if err := json.Unmarshal(recordBytes, &env); err != nil {
		return false, fmt.Sprintf("decode record envelope: %v", err)
	}
	return canonicalID(env.IsccID) == canonicalID(wantID), ""
}

// canonicalID normalizes an ISCC-ID to its single "ISCC:"-prefixed form so a bare
// and a prefixed string compare equal. It is the certificate §1 lookup-key idiom:
// trim any leading "ISCC:" then re-prefix exactly once.
func canonicalID(id string) string {
	return "ISCC:" + strings.TrimPrefix(id, "ISCC:")
}
