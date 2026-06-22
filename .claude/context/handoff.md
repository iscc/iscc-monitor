## 2026-06-22 — Extract the pure `internal/proof/verify` inclusion-verifier core (WASM-shareable skeleton)

**Done:** Created the pure, WASM-shareable `internal/proof/verify` package wrapping the RFC-6962
inclusion-verify primitive (`HashLeaf(record)` → `proof.VerifyInclusion`) behind a minimal-arg
`VerifyInclusion(record, index, size, proof, root) (bool, error)`, and routed the two production call
sites (proofserve `serveVerify` and certificate §3 `buildData`) through it. This establishes the single
verifier core the WASM milestone needs and removes the verbatim-duplicated leaf-hash-and-verify
primitive (with its identical arg-order gotcha comment) that lived in both handlers.

**Files changed:**
- `internal/proof/verify/verify.go` (new): the pure core. Imports ONLY `fmt` +
  `transparency-dev/merkle/proof` (aliased `merkleproof` to avoid the package-name collision) +
  `.../rfc6962`. Three-way contract: `(true,nil)` rebuilds root, `(false,nil)` well-formed-but-negative
  verdict, `(false,err)` only on the `index >= size` precondition (pre-checked up front).
- `internal/proof/verify/verify_test.go` (new, not counted): golden 4-leaf vector built in-test from
  the hasher AND cross-checked against the next.md base64-Std literals (`root ==
  vdHF/1WxnLaw58dhv5psyqJ/u/wHt08fq7bpEaC9KrM=`); positive + the three negative/error cases.
- `internal/proofserve/handler.go`: replaced the inline `leafHash := …; included := proof.VerifyInclusion(…) == nil`
  pair in `serveVerify` with `included, _ := verify.VerifyInclusion(record, leafIndex, size, builtProof, root)`;
  dropped the now-unused `merkle/proof` + `rfc6962` imports, added `internal/proof/verify`.
- `internal/certificate/handler.go`: replaced the inline pair in `buildData` §3 with
  `if ok, _ := verify.VerifyInclusion(record, data.Position, hub.LastSize, builtProof, root); ok {`;
  same import swap. The `record` var is still used (`arts.record = record` inside the `ok` block).

**Verification:** `mise run check` → green (build + vet + test, all 24 packages incl. the new one).
Per-criterion:
- [x] `GOOS=js GOARCH=wasm go build ./internal/proof/verify` succeeds (the load-bearing WASM-shareability gate).
- [x] `go test -count=1 -run TestVerifyInclusion ./internal/proof/verify` passes.
- [x] Golden assertion `VerifyInclusion([]byte("leaf-1"),1,4,proof,root)` → `(true,nil)` (both the
  library-rebuilt tree AND the next.md base64 literals; they cross-check equal).
- [x] Negatives: wrong record + tampered root each `(false,nil)`; `index>=size` returns a non-nil error.
- [x] Behavior-preserving: `-run TestVerify ./internal/proofserve` and `-run TestCertificate
  ./internal/certificate` pass unchanged (verify-for-me verdict shape + §3 gate byte-identical).
- [x] DRY: the inline `rfc6962.DefaultHasher.HashLeaf` + `proof.VerifyInclusion(rfc6962…)` pair no
  longer appears in either handler (grep confirms; both now call `verify.VerifyInclusion`).
- [x] `gofmt -l .` (excl `cauldron/`) clean; `go mod tidy -diff` clean (no new prod dep — reuses
  the existing `transparency-dev/merkle`).
- [x] Conformance/oracle gate (touched proof code): `TestVerifyInclusionIsNonVacuous` +
  `TestInclusionServedProofVerifies` (proofserve), `TestCertificateInclusionProofContradictory` +
  `…ProofBundleContradictory` (certificate §3 fork-tile fail-closed) all PASS; `derive_vkey.py` prints
  both golden vectors byte-for-byte (`40b74463`/`22b08f3e`, `.scratch` cleaned).
- [x] Purity closure: `go list -deps ./internal/proof/verify` has NO `net`/`net/http`/`database/sql`/
  `html/template`; `os` appears only transitively via `fmt` (the documented always-loaded nuance — the
  WASM build is the load-bearing proof, not the grep).
- [x] Mutation (non-vacuous, reproduced + reverted): forcing the wrapper verdict to `… == nil || true`
  fails `TestVerifyInclusion/{wrong_record,tampered_root}`; reverted → green.

**Next:** The single shared verifier core now exists and compiles under `GOOS=js GOARCH=wasm`. The
natural next WASM sub-step is the `GOOS=js`/`syscall/js` entrypoint (`cmd/wasm` or similar) that exports
`verify.VerifyInclusion` to JS, then the lazy tier-2 enhancement on the certificate/dossier and the
standalone `monitor.iscc.codes` Independent Verification app (the rest of the milestone arc). A
`VerifyConsistency` sibling in this same package can follow when a caller needs it (the consistency
primitive in proofserve has a DIFFERENT arg order — `proof` precedes the two roots — so it warrants its
own wrapper to hide that, just as this one hides the inclusion order).

**Notes:**
- Behavior-preserving extraction, byte-identical observable outputs. The certificate's package doc and
  in-body comments still say "proof.VerifyInclusion against the §2 root" — left intact because they
  describe the gate conceptually and remain accurate (`verify.VerifyInclusion` IS that check); not in
  scope to reword and would have been noise.
- Both call sites discard the error (`, _`) deliberately: per next.md, a non-nil error (only reachable on
  `index >= size`, which both sites pre-gate via the accepted-tree cap) is folded into the fail-closed
  negative verdict exactly as the old `== nil` boolean did — proofserve reports "inclusion proof did not
  verify", certificate silently declines §3. No new 5xx branch introduced. The discard is documented in
  the call-site comments and is NOT a swallowed-error gate-dodge: the boolean carries the verdict and the
  precondition is structurally unreachable on the happy path.
- No change to any standing open issue (none sat on the edited lines). The certificate's `did:web:`
  `host:port` `%3A`-encode gap, §5 digest-binding, §6 timestamp, `safeStamp`/`hubDomain` defects are all
  untouched and remain filed.
- New package learnings file recommended for `review` to seed: `learnings/proof-verify.md` (pointer row
  in the index) — the WASM-purity seam, the three-way verdict contract, and the inclusion-vs-consistency
  arg-order divergence are the durable facts a future WASM-entrypoint step will need.
