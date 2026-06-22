# Next Work Package

## Step: Extract the pure `internal/proof/verify` inclusion-verifier core (WASM-shareable skeleton)

## Advances
The **WASM verifier upgrade** milestone (target.md, `[not started]`, **1/1 Verify open — the
longest-standing untouched criterion**, re-confirmed by `state.md`: no `internal/proof` package, no
`syscall/js`, no `GOOS=js` target). Its Verify criterion:

> **Verify:** identical vectors yield identical verdicts (WASM vs server); the verifier artifact hash
> matches the published value; a `(size, root)` mismatch renders the guided split-view alert, not a
> dead error.

This is the **skeleton-first** opening move toward that milestone: the "identical verdicts (WASM vs
server)" half is only achievable once the server and the future `GOOS=js` build share **one** pure
verification core. This step creates that core (`internal/proof/verify`), golden-tests it, proves it
builds under `GOOS=js GOARCH=wasm`, and routes the two existing server call sites through it — so the
WASM build later imports the *same* function the server already uses. Remaining WASM sub-steps are
listed under `## Not In Scope`.

## Goal
Create a pure, WASM-shareable `internal/proof/verify` package wrapping the RFC-6962 inclusion-verify
primitive (`HashLeaf(record)` → `proof.VerifyInclusion`), and route the two production call sites
(`proofserve` verify-for-me and `certificate` §3) through it. This establishes the single verifier
core the WASM milestone needs and removes the verbatim-duplicated primitive (with its identical
"arg-order gotcha" comment) that currently lives in both handlers.

## Scope
- **Create**: `internal/proof/verify/verify.go` (the pure package) and
  `internal/proof/verify/verify_test.go` (golden-vector + parity test — does not count toward the
  3-file limit).
- **Modify** (≤3 non-test files):
  1. `internal/proofserve/handler.go` — replace the inline `leafHash := …; included := proof.VerifyInclusion(…) == nil`
     pair at lines ~680-681 (`serveVerify`) with a call to the new package.
  2. `internal/certificate/handler.go` — replace the inline pair at lines ~805-806 (`buildData` §3)
     with the same call.
- **Reference**:
  - `.claude/context/learnings.md` — the always-loaded rules: "`proof/verify` is pure (no
    `net`/`os`/`sqlite` imports) … shared by the server, `verify-for-me`, and the WASM build. Keep it
    import-clean or the WASM build breaks." AND "gate a rendered ✓ on a re-VERIFICATION, not a status
    flag" (this core IS the re-verification).
  - `.claude/context/learnings/certificate.md` — §3 gate mechanics (the re-verify rule, the
    `HasClause3`-only-on-nil contract) so the certificate edit preserves exact behavior.
  - `.claude/context/learnings/http-surface.md` — proofserve `serveVerify` semantics (the
    `Verified`/`Included`/`Reason` verdict shape the call must not change).
  - `internal/proofserve/handler.go:666-696` and `internal/certificate/handler.go:799-819` — the two
    verbatim call sites to replace.
  - `github.com/transparency-dev/merkle/proof.VerifyInclusion` (signature:
    `(hasher merkle.LogHasher, index, size uint64, leafHash []byte, proof [][]byte, root []byte) error`)
    and `github.com/transparency-dev/merkle/rfc6962.DefaultHasher` (`HashLeaf`). Both build under
    `GOOS=js GOARCH=wasm` (verified during scoping).

## Not In Scope
- **No `GOOS=js GOARCH=wasm` main / `syscall/js` glue, no `cmd/wasm`, no JS shim.** This step only
  creates the *pure Go* core and proves it *compiles* under the WASM target via `go build`; the actual
  WASM entrypoint, the lazy-loaded tier-2 enhancement on the certificate/dossier, and the standalone
  `monitor.iscc.codes` Independent Verification app are later sub-steps of this same arc.
- **No consistency-proof verifier in the new package yet.** Inclusion only (the verify-for-me + §3
  primitive). A `VerifyConsistency` sibling can join in a later step when a caller needs it.
- **No change to the verdict shapes, error mapping, or HTTP status** of `serveVerify` or the
  certificate — this is a behavior-preserving extraction; the observable outputs must be byte-identical.
- **Do NOT touch the proof-*builder*** (`logclient.InclusionProofFromTiles`) — it stays in
  `logclient` (it reads tiles; it is not pure). Only the final hash-leaf-and-verify step moves.
- Dossier §4 Bitcoin-anchor named region (the handoff's alternative suggestion) — deferred; this step
  prioritizes the longest-standing untouched milestone Verify criterion per the state's #1 ranking.
- The standing `normal` hardening defects (§5 digest-binding, `host:port` DID `%3A`-encode,
  `safeStamp`, `hubDomain` ForceQuery, §6 timestamp) — none are on the lines this step edits.

## Implementation Notes
- **Purity is the load-bearing constraint.** `internal/proof/verify` must import ONLY
  `github.com/transparency-dev/merkle/proof` and `github.com/transparency-dev/merkle/rfc6962` (plus
  `fmt`/`errors` for the precondition error). NO `net`, `net/http`, `os`, `database/sql`, `embed`,
  `html/template`. Per the didweb learning, `os` may appear transitively via `fmt`; that is fine — the
  load-bearing test is `GOOS=js GOARCH=wasm go build ./internal/proof/verify` succeeding, NOT grepping
  the dep list.
- **API shape (pure, minimal-arg, KISS):**
  ```go
  // VerifyInclusion re-verifies that record is the leaf at index in a tree of the
  // given size whose root is root, using the supplied RFC-6962 inclusion proof.
  // It hashes record into its leaf hash (rfc6962.DefaultHasher.HashLeaf) and runs
  // proof.VerifyInclusion. Returns (true, nil) when the proof rebuilds root,
  // (false, nil) when the proof is well-formed but does NOT rebuild root (a genuine
  // negative verdict — not an error), and (false, err) only on a precondition the
  // library rejects (e.g. index >= size). Pure: shared verbatim by the server,
  // verify-for-me, and the GOOS=js WASM build, so it must stay import-clean.
  func VerifyInclusion(record []byte, index, size uint64, proof [][]byte, root []byte) (bool, error)
  ```
  The package dir is `proof/verify` and is named `verify`; alias the merkle proof import to avoid the
  `proof` package-name collision (e.g. `import merkleproof "github.com/transparency-dev/merkle/proof"`),
  matching how the existing handlers alias as needed.
- **The (false, nil) vs (false, err) split is the whole point.** Both call sites today treat a non-nil
  library `VerifyInclusion` result as a *silent negative*, never a 500 — proofserve sets
  `Verified:false, Reason:"inclusion proof did not verify"`; certificate silently omits §3. The library
  returns an error for BOTH a real mismatch AND a precondition violation (`index >= size`). To preserve
  exact behavior: check `index >= size` up front and return `(false, fmt.Errorf(...))`; otherwise
  `return merkleproof.VerifyInclusion(rfc6962.DefaultHasher, index, size, rfc6962.DefaultHasher.HashLeaf(record), proof, root) == nil, nil`.
  Both call sites pre-gate the leaf against the accepted tree (`leafIndex < size` /
  `seqs[0] < hub.LastSize`), so the precondition branch is unreachable on the happy path — note that in
  a comment. At the call sites, route the returned `ok` boolean exactly where the old
  `included`/`== nil` boolean went; treating a non-nil error as a non-verified verdict keeps both sites
  strictly fail-closed (the certificate already silently declines §3; proofserve already reports
  "did not verify") — do NOT introduce a new 5xx branch.
- **Arg order is the classic bug (always-loaded learning, repeated verbatim in both call sites):**
  `VerifyInclusion(hasher, index, size, leafHash, proof, root)` — `leafHash` precedes `proof`. The new
  wrapper hides this; keep the gotcha comment on the wrapper, and trim the now-redundant duplicate
  comments at the two call sites to a one-liner ("// re-verify via proof/verify").
- **This wrapper IS the "re-verify, not a flag" rule (always-loaded).** Both surfaces render a ✓ a
  reader trusts; they must keep gating on this re-verification. The extraction must not regress that —
  the boolean the wrapper returns is the same `proof.VerifyInclusion(...) == nil` gate, just centralized.
- **Golden vector for the test (computed with the library during scoping, reproducible).** A 4-leaf
  tree of records `"leaf-0".."leaf-3"`: `root = vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM=`
  (base64-Std); the inclusion proof for index 1 in size 4 is
  `["MF31n5WQw8msY9KydDw4jjeSRJB4zr9/s9vmRxZDsrc=", "vUX/KHlnBNiL2sUbHfVT/aWYN7YW1tHLIRTbw7CH/2k="]`.
  Decode these (base64-Std) in the test and assert `VerifyInclusion([]byte("leaf-1"), 1, 4, proof,
  root)` returns `(true, nil)`. Negative cases: a wrong record (`"WRONG"`) → `(false, nil)`; a tampered
  root → `(false, nil)`; `index >= size` (e.g. `4, 4`) → `(false, err)`. Prefer building the proof
  in-test from `rfc6962.DefaultHasher` (HashLeaf + HashChildren) so a library bump can't silently rot
  the vector; the base64 literals above are the cross-check.

## Verification
- `mise run check` is green (build + vet + test all packages; `gofmt -l .` excl. `cauldron/` empty;
  `go mod tidy -diff` clean — no new prod dep).
- `GOOS=js GOARCH=wasm go build ./internal/proof/verify` succeeds (the WASM-shareability gate — the
  load-bearing proof the core can compile into the future WASM build).
- `go test -count=1 -run TestVerifyInclusion ./internal/proof/verify` passes.
- Golden assertion: `verify.VerifyInclusion([]byte("leaf-1"), 1, 4, proof, root)` returns `(true, nil)`
  for the 4-leaf vector above (`root == vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM=`).
- Negative assertions: a wrong record and a tampered root each return `(false, nil)` (not an error);
  `index >= size` returns a non-nil error.
- Behavior-preserving: `go test -count=1 -run TestVerify ./internal/proofserve` and
  `go test -count=1 -run TestCertificate ./internal/certificate` both still pass unchanged (the
  verify-for-me verdict shape and the §3 gate behavior are byte-identical after routing through the
  new core).
- DRY: the inline `proof.VerifyInclusion(rfc6962.DefaultHasher, …leafHash…, builtProof, root) == nil`
  pair no longer appears in either `internal/proofserve/handler.go` or `internal/certificate/handler.go`
  (each now calls `verify.VerifyInclusion`).

## Done When
`internal/proof/verify` exists as a pure, WASM-buildable package whose golden + negative tests pass,
both server call sites route through it with byte-identical observable behavior, and `mise run check`
plus the `GOOS=js GOARCH=wasm go build ./internal/proof/verify` gate are green.
