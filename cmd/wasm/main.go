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
// verifyadapter.VerifyJSON adapter. JS arg order: (record, root, proofArray,
// index, size) — record and root are base64-Std strings, proofArray is a JS array
// of base64-Std proof-hash strings, index and size are numbers. It returns a JS
// object {verified: bool, error: string}. A wrong arg count is folded into an
// error result rather than panicking.
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
	index := uint64(args[3].Int())
	size := uint64(args[4].Int())

	verified, errMsg := verifyadapter.VerifyJSON(record, root, proof, index, size)
	return map[string]any{"verified": verified, "error": errMsg}
}

// main registers isccVerifyInclusion on globalThis and blocks so the Go runtime
// stays alive to service JS calls.
func main() {
	js.Global().Set("isccVerifyInclusion", js.FuncOf(isccVerifyInclusion))
	select {}
}
