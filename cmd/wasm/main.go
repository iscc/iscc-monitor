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
	"math"
	"syscall/js"

	"github.com/iscc/iscc-monitor/cmd/wasm/verifyadapter"
)

// maxSafeInteger is JavaScript's Number.MAX_SAFE_INTEGER (2^53 - 1): the largest
// integer a JS Number can represent without losing precision. A leaf index or tree
// size at or above 2^53 cannot round-trip through a JS Number, so the shim rejects
// it rather than truncate it to a wrong-but-plausible value.
const maxSafeInteger = float64(1<<53 - 1)

// isccVerifyInclusion is the js.FuncOf shim bridging the JS call into the pure
// verifyadapter.VerifyJSON adapter. JS arg order: (record, root, proofArray,
// index, size) — record and root are base64-Std strings, proofArray is a JS array
// of base64-Std proof-hash strings, index and size are numbers. It returns a JS
// object {verified: bool, error: string}. A wrong arg count, or a non-integral /
// out-of-safe-range index or size, is folded into an error result rather than
// panicking or silently truncating.
func isccVerifyInclusion(this js.Value, args []js.Value) any {
	if len(args) != 5 {
		return map[string]any{
			"verified": false,
			"error":    "isccVerifyInclusion: expected 5 args (record, root, proof, index, size)",
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
	index, errMsg := safeIndex(args[3].Float(), "index")
	if errMsg != "" {
		return map[string]any{"verified": false, "error": errMsg}
	}
	size, errMsg := safeIndex(args[4].Float(), "size")
	if errMsg != "" {
		return map[string]any{"verified": false, "error": errMsg}
	}

	verified, errMsg := verifyadapter.VerifyJSON(record, root, proof, index, size)
	return map[string]any{"verified": verified, "error": errMsg}
}

// safeIndex converts a JS Number (read as float64) into a uint64 leaf index/size,
// failing closed when the value is not a non-negative integer within the JS
// safe-integer range. It returns a non-empty errMsg (and a zero value) on any
// violation — NaN, infinity, a fractional value, a negative value, or a value at
// or beyond 2^53 — so the caller folds it into the {verified:false, error:…} result
// the arg-count guard uses. This is the JS→Go integer contract enforced at the one
// boundary where the float→uint64 narrowing would otherwise truncate silently.
func safeIndex(v float64, name string) (uint64, string) {
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

// main registers isccVerifyInclusion on globalThis and blocks so the Go runtime
// stays alive to service JS calls.
func main() {
	js.Global().Set("isccVerifyInclusion", js.FuncOf(isccVerifyInclusion))
	select {}
}
