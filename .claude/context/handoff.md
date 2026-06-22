## 2026-06-22 — Review of: Extract the pure `internal/proof/verify` inclusion-verifier core (WASM-shareable skeleton)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance creates the pure, WASM-shareable `internal/proof/verify` package wrapping the
RFC-6962 `HashLeaf → proof.VerifyInclusion` primitive behind a minimal-arg three-way verdict contract,
and routes the two production call sites (proofserve `serveVerify`, certificate §3 `buildData`) through
it — the single shared verifier core the WASM milestone needs, removing the verbatim-duplicated
leaf-hash-and-verify primitive from both handlers. Scope is tight (2 new files in the new package + 2
one-line call-site swaps), `mise run check` is green across all 24 packages, the new wrapper test is
mutation-proven non-vacuous, the WASM-shareability and purity gates hold, and both call sites are
byte-identical in observable behavior. Codex returned a clean verdict.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 24 packages incl. the new `internal/proof/verify`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/proof/verify` — succeeds (the load-bearing WASM-shareability gate).
- [x] `go test -count=1 -run TestVerifyInclusion ./internal/proof/verify` — PASS (positive + 3 negative/error cases).
- [x] Golden assertion `VerifyInclusion([]byte("leaf-1"),1,4,proof,root)` → `(true,nil)` — verified
  both via the library-rebuilt 4-leaf tree AND the next.md base64-Std literals (`root == vdHF/...KrM=`);
  the cross-check test pins them equal.
- [x] Negatives — wrong record + tampered root each `(false,nil)`; `index>=size` → non-nil error. PASS.
- [x] Behavior-preserving — `-run TestVerify ./internal/proofserve` and `-run TestCertificate
  ./internal/certificate` both PASS unchanged (verdict shape + §3 gate byte-identical after routing).
- [x] DRY — the inline `rfc6962.DefaultHasher.HashLeaf` + library `proof.VerifyInclusion` pair is gone
  from BOTH handlers (grep confirms; remaining `proof.VerifyInclusion` matches are all conceptual
  comments, no live code). Both now call `verify.VerifyInclusion`.
- [x] Purity closure — `go list -deps ./internal/proof/verify` has NO `net`/`net/http`/`database/sql`/
  `html/template`; `os` only transitively via `fmt` (the documented always-loaded nuance).
- [x] `gofmt -l .` (excl `cauldron/`) clean; `go mod tidy -diff` clean (no new prod dep — reuses the
  existing `transparency-dev/merkle`).
- [x] Conformance/oracle gate (touched proof code) — `TestVerifyInclusionIsNonVacuous` +
  `TestInclusionServedProofVerifies` (proofserve), `TestCertificateInclusionProofContradictory` +
  `…ProofBundleContradictory` (certificate §3 fork-tile fail-closed), and `logclient` fsck all PASS;
  `derive_vkey.py` prints both golden vectors byte-for-byte (`40b74463`/`22b08f3e`; `.scratch` cleaned).
- [x] Mutation (reviewer-reproduced + reverted) — forcing the wrapper verdict to `… == nil || true`
  fails `TestVerifyInclusion/{wrong_record,tampered_root}`; reverted → green.
- [x] Quality-gate integrity — no `nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in any
  unpushed Go diff (`@{upstream}..HEAD`, 3 commits; only the advance touched `.go`).
- [x] Scope — only the 2 new `internal/proof/verify` files + the 2 one-line call-site swaps; `go.mod`,
  `go.sum`, store schema, `cmd/iscc-monitor/main.go` all byte-unchanged.

**Issues found:** (none). The `, _` error discards at both call sites are the documented, justified
verdict-folding (a non-nil err — only reachable on `index >= size`, structurally unreachable behind each
site's accepted-tree pre-gate — folds to the fail-closed `false` exactly as the old `== nil` boolean did).
NOT a swallowed-error gate-dodge: the boolean carries the verdict. Verified the old inline library call
ALSO returned non-nil for `index >= size`, so the path is byte-identical. The certificate's conceptual
`proof.VerifyInclusion` comments (handler.go:24/314/375/590/733/740) remain accurate descriptions of the
gate (`verify.VerifyInclusion` IS that check); leaving them un-reworded is correct, not stale-reference debt.

**Codex second opinion:** Clean — one summary verdict, no `Review comment:` findings. Codex independently
confirmed the extraction preserves the existing inclusion-verification behavior at both production call
sites, the new core mirrors the prior hash-and-verify logic with the same fail-closed outcome, tests pass,
and no blocking correctness issues were introduced. Matches my independent assessment; nothing to triage.

**Visual check:** n/a — no SSR surface changed in a visually-meaningful way. The certificate handler edit
is a behavior-preserving internal swap (inline verify call → wrapper); rendered output is byte-identical
(the `TestCertificate` suite passed unchanged). No proof of a visual delta possible from a verbatim refactor.

**Next:** The single shared verifier core now exists and compiles under `GOOS=js GOARCH=wasm`, so the
natural next WASM sub-step is the `GOOS=js`/`syscall/js` entrypoint (`cmd/wasm`) that exports
`verify.VerifyInclusion` to JS, then the lazy tier-2 enhancement on the certificate/dossier and the
standalone `monitor.iscc.codes` Independent Verification app (the rest of the milestone arc). A
`VerifyConsistency` sibling can join this package when a caller needs it — it warrants its OWN wrapper
because the consistency primitive's arg order differs (`proof` precedes the two roots). Alternatively,
the highest-value standing `normal` hardening to fold in when its line is next touched is the `safeStamp`
panic-recover + timeout guard on the freshly-live OTS stamp path.

**Notes:**
- NOT DONE: WASM verifier milestone is now skeleton-landed but the `syscall/js` entrypoint + the in-browser
  verifier app are still open (the "identical verdicts WASM vs server" half is unblocked by this core but
  not yet built). OTS Verify keeps its offline-unprovable live-chain Bitcoin-confirmed half open; the M-UI
  exit visual-pass + human sign-off (ADR-0012) has not been run. Loop = CONTINUE.
- Open issues carried forward unchanged (none on the edited lines): `safeStamp` guard (`normal`, highest
  value), `hubDomain` ForceQuery (`normal`), §4/bundle `host:port` DID `%3A`-encode (`normal`), §5
  digest-binding (`normal`), §6 `· at` timestamp (`normal`), plus the `low` debt set (nil-Stamper
  fall-through, vacuous label test, `-run TestOTS` filter gap, notecheck `out` param, overlay-precedence
  3x dup, mirror seam, scaling trip-wire, proofserve writeReadError dup).
- New learnings file seeded: `learnings/proof-verify.md` (+ pointer row in the index) — the WASM-purity
  seam, the three-way verdict contract, the inclusion-vs-consistency arg-order divergence, and the
  golden-vector discipline are the durable facts the next WASM-entrypoint step will need.
