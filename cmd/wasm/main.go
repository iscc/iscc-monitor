//go:build js && wasm

// Command wasm is the WASM-only entrypoint for the in-browser ISCC verifier.
// It is compiled solely by GOOS=js GOARCH=wasm (the build tag excludes it from
// the linux go build/vet/test gate, leaving this package with no Go files there,
// so go build ./... skips it). The testable marshaling lives in the untagged,
// non-main cmd/wasm/verifyadapter package imported here. This file is the thin
// syscall/js glue: it wraps verifyadapter.VerifyJSON in a js.FuncOf and exposes
// it as the global JS function globalThis.isccVerifyInclusion, then blocks so the
// runtime stays alive.
package main

import (
	"syscall/js"

	"github.com/iscc/iscc-monitor/cmd/wasm/verifyadapter"
)

// isccVerifyInclusion is the js.FuncOf shim bridging the JS call into the pure
// verifyadapter adapter. JS arg order: (record, root, proofArray, index, size[, id])
// — record and root are base64-Std strings, proofArray is a JS array of base64-Std
// proof-hash strings, index and size are numbers, and the optional 6th id is the
// requested ISCC-ID string. It returns a JS object {verified: bool, error: string}.
//
// With the 6th id arg present (the cross-origin Surface-C verifier), verified gates
// on BOTH the inclusion math (VerifyJSON) AND the id-binding (RecordCommitsID), so a
// monitor returning a valid-but-unrelated declaration's bundle yields a negative
// verdict, not a false green. The 6th arg is OPTIONAL so the same-origin certificate
// caller's 5-arg call (cert.html — id-binding is harmless there, the monitor already
// baked the bundle) keeps working against this shared verify.wasm; a 5-arg call gates
// on inclusion math alone. A wrong arg count, a non-integral / out-of-safe-range
// index or size, or broken record input is folded into an error result rather than
// panicking or silently truncating.
func isccVerifyInclusion(this js.Value, args []js.Value) any {
	if len(args) != 5 && len(args) != 6 {
		return map[string]any{
			"verified": false,
			"error":    "isccVerifyInclusion: expected 5 or 6 args (record, root, proof, index, size[, id])",
		}
	}
	record := args[0].String()
	root := args[1].String()
	proofArr := args[2]
	proof := make([]string, proofArr.Length())
	for i := range proof {
		proof[i] = proofArr.Index(i).String()
	}
	// Validate index/size BEFORE the uint64 conversion. js.Value.Int() is
	// int(v.Float()) — it silently TRUNCATES, so a non-integer JS Number (e.g. 1.9)
	// would verify against the wrong-but-truncated leaf, and a value beyond the JS
	// safe-integer range (2^53) would round to a different integer. Read each as a
	// float64 and fail closed on any value that cannot be a real leaf index/size.
	index, errMsg := verifyadapter.SafeIndex(args[3].Float(), "index")
	if errMsg != "" {
		return map[string]any{"verified": false, "error": errMsg}
	}
	size, errMsg := verifyadapter.SafeIndex(args[4].Float(), "size")
	if errMsg != "" {
		return map[string]any{"verified": false, "error": errMsg}
	}

	inclusionOK, inclusionErr := verifyadapter.VerifyJSON(record, root, proof, index, size)

	// The id-binding gates verified ONLY when the requested id is supplied (the 6th
	// arg). An error on EITHER side (broken input) beats a bare false: prefer the
	// first non-empty errMsg so a decode/parse fault renders as an error, not a
	// mismatch.
	idOK, idErr := true, ""
	if len(args) == 6 {
		idOK, idErr = verifyadapter.RecordCommitsID(record, args[5].String())
	}
	errMsg = inclusionErr
	if errMsg == "" {
		errMsg = idErr
	}
	return map[string]any{"verified": inclusionOK && idOK, "error": errMsg}
}

// main registers isccVerifyInclusion on globalThis and blocks so the Go runtime
// stays alive to service JS calls.
func main() {
	js.Global().Set("isccVerifyInclusion", js.FuncOf(isccVerifyInclusion))
	select {}
}
