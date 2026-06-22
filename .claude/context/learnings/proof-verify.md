<!-- area: internal/proof/verify (pure RFC-6962 inclusion-verifier core, WASM-shareable) -->
<!-- indexed-as: proof-verify.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/proof/verify` — pure inclusion-verifier core

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## The shared verifier core (`VerifyInclusion`)

- **This is the ONE function the server, verify-for-me, and the future WASM build all share.**
  `VerifyInclusion(record, index, size, proof, root) (bool, error)` wraps the two-step
  `rfc6962.DefaultHasher.HashLeaf(record)` → `merkleproof.VerifyInclusion(...)` primitive that BOTH
  server surfaces gate their rendered ✓ on (proofserve `serveVerify`, certificate §3 `buildData`). The
  WASM-milestone Verify criterion ("identical vectors yield identical verdicts, WASM vs server") is only
  achievable because both sides call this same code — do not fork a second verify path for WASM.

- **Three-way verdict contract is the whole point of the wrapper — preserve it exactly.**
  `(true, nil)` proof rebuilds root (verified); `(false, nil)` well-formed but does NOT rebuild root (a
  genuine NEGATIVE verdict — wrong record or tampered root, NOT an error); `(false, err)` ONLY on the
  `index >= size` precondition the library rejects, surfaced so callers don't conflate it with a
  negative. Both call sites discard the err (`, _`) and route the boolean exactly where the old
  `proof.VerifyInclusion(...) == nil` boolean went — folding a non-nil err into the fail-closed negative.
  This is NOT a swallowed-error gate-dodge: the boolean carries the verdict and the precondition is
  structurally unreachable on the happy path (both sites pre-gate the leaf against the accepted tree,
  `leafIndex < size` / `seqs[0] < hub.LastSize`). The old inline library call ALSO returned non-nil for
  `index >= size`, so the `== nil` path yielded the same `false` — behavior is byte-identical.

- **Purity is the LOAD-BEARING constraint (always-loaded rule).** Imports ONLY `fmt` +
  `transparency-dev/merkle/proof` (aliased `merkleproof` to dodge the `proof` package-name collision
  with this dir) + `.../rfc6962`. NO `net`/`net/http`/`os`/`database/sql`/`embed`/`html/template`. The
  load-bearing gate is `GOOS=js GOARCH=wasm go build ./internal/proof/verify` succeeding, NOT grepping
  the dep list (`os` appears transitively via `fmt`, same nuance as `internal/didweb`/`internal/index`).
  Keep it import-clean or the future WASM build breaks.

- **Arg-order gotcha lives HERE now, hidden from call sites.** `merkleproof.VerifyInclusion(hasher,
  index, size, leafHash, proof, root)` — `leafHash` precedes `proof`, UNLIKE `VerifyConsistency`
  (`hasher, size1, size2, proof, root1, root2` — `proof` precedes the two roots). When a
  `VerifyConsistency` sibling joins this package (next.md flagged it), it needs its OWN wrapper to hide
  the DIFFERENT arg order — do not reuse this one's signature shape blindly.

## Golden vector + test discipline

- **The 4-leaf golden vector is grounded in the library's own hasher, then cross-checked against frozen
  literals.** `buildGoldenTree` builds `root = p(p(h0,h1), p(h2,h3))` and `proof(index=1) = [h0,
  p(h2,h3)]` from `rfc6962.DefaultHasher` (HashLeaf + HashChildren) so a library bump can't silently rot
  the vector; `TestVerifyInclusionGoldenCrossCheck` pins it against the base64-Std literals
  (`root == vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM=`). A separate test decodes the literals directly
  and verifies, so the wrapper is checked against the externally-recorded vector too. Mutation-proven
  (review): forcing the verdict to `... == nil || true` fails `wrong_record` + `tampered_root`; reverted.
