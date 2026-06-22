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
	"fmt"

	"github.com/iscc/iscc-monitor/internal/proof/verify"
)

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
