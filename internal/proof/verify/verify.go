// Package verify is the pure, WASM-shareable RFC-6962 inclusion-verifier core.
// It wraps the two-step "hash the record into its leaf hash, then re-verify the
// inclusion proof against the accepted root" primitive that both server surfaces
// (verify-for-me and the certificate's §3 clause) gate their rendered ✓ on, so
// the future GOOS=js WASM build imports the SAME function the server already
// uses — identical inputs yield identical verdicts on either side.
//
// Purity is the load-bearing constraint: this package imports ONLY the
// transparency-dev/merkle proof + rfc6962 primitives (plus fmt/stdlib for the
// precondition error). It pulls in no net, net/http, os, database/sql, embed, or
// template, so GOOS=js GOARCH=wasm go build ./internal/proof/verify succeeds and
// the core stays importable from every surface (server, verify-for-me, WASM).
package verify

import (
	"fmt"

	merkleproof "github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
)

// VerifyInclusion re-verifies that record is the leaf at index in a tree of the
// given size whose root is root, using the supplied RFC-6962 inclusion proof. It
// hashes record into its leaf hash (rfc6962.DefaultHasher.HashLeaf) and runs the
// merkle proof.VerifyInclusion check.
//
// The three-way return is the whole point of the wrapper:
//   - (true, nil)  — the proof rebuilds root: a verified inclusion.
//   - (false, nil) — the proof is well-formed but does NOT rebuild root: a
//     genuine NEGATIVE verdict (a wrong record or a tampered root), not an error.
//   - (false, err) — only on a precondition the library rejects (index >= size),
//     surfaced explicitly so callers do not conflate it with a negative verdict.
//
// Arg-order gotcha (always-loaded learning): merkleproof.VerifyInclusion is
// (hasher, index, size, leafHash, proof, root) — leafHash precedes proof, UNLIKE
// VerifyConsistency. This wrapper hides that ordering so call sites cannot trip it.
//
// Both production call sites pre-gate the leaf against the accepted tree
// (leafIndex < size / seqs[0] < hub.LastSize), so the precondition branch is
// unreachable on their happy path; it exists so a misuse fails closed with a
// diagnosable error rather than a silent negative. Pure: shared verbatim by the
// server, verify-for-me, and the GOOS=js WASM build, so it must stay import-clean.
func VerifyInclusion(record []byte, index, size uint64, proof [][]byte, root []byte) (bool, error) {
	if index >= size {
		return false, fmt.Errorf("verify: index %d is beyond tree size %d", index, size)
	}
	leafHash := rfc6962.DefaultHasher.HashLeaf(record)
	return merkleproof.VerifyInclusion(rfc6962.DefaultHasher, index, size, leafHash, proof, root) == nil, nil
}
