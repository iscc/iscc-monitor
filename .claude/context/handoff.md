## 2026-06-22 — Review of: Bind the §5 OTS proof's committed digest to §2's accepted root before rendering BITCOIN ANCHOR

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance added a digest-bound classifier `ots.ConfirmedFor(otsBytes, root)` that
fail-closes (wrapped error, same contract as a parse failure) unless the parsed proof's committed
SHA-256 `File.Digest` equals the caller's root, then pointed certificate §5 at it
(`ots.ConfirmedFor(rec.OTSBytes, root)`), so a mis-stamped (root, proof) row now declines §5 silently
instead of fabricating "block N". `Confirmed` and `ConfirmedFor` share one parse + one private
`classify` (attestation + `>MaxInt64` overflow guard). Scope is exemplary: exactly 2 non-test files +
2 test files, nothing from `## Not In Scope` touched, no template/SSR-output change, no dependency
drift; both claimed mutations independently reproduced.

**Verification:**
- [x] `mise run check` — green (build + vet + `go test ./...`, all 28 packages ok).
- [x] `gofmt -l .` — empty (no formatting failure).
- [x] `go test -count=1 -v -run TestOTSConfirmedFor ./internal/ots` — PASS; `-v` lists
      confirmed/pending/mismatch/parse subcases.
- [x] `go test -count=1 -v -run TestCertificateBitcoinAnchor ./internal/certificate` — PASS (Confirmed,
      Pending, DigestMismatch, Unanchored, EmptySentinel).
- [x] Mutation 1 (independent) — removing the `bytes.Equal(file.Digest, root)` gate from `ConfirmedFor`
      makes `TestOTSConfirmedForDigestBound/mismatch_declines` FAIL ("want digest-mismatch error, got
      nil"). Restored byte-identical.
- [x] Mutation 2 (independent) — reverting §5's call to `ots.Confirmed(rec.OTSBytes)` makes
      `TestCertificateBitcoinAnchorDigestMismatch` FAIL (renders §5 for the mismatched row). Restored
      byte-identical; tree clean.
- [x] WASM-pure invariant — `go list -deps` of `internal/{didweb,index,badge}` + `internal/proof/verify`
      each show 0 hits on `internal/ots`; all four still `GOOS=js GOARCH=wasm go build`. `go.mod`/`go.sum`
      byte-identical vs HEAD~1.
- [x] Oracle/conformance gate — applicable-adjacent: the change is OTS proof classification (a
      `bytes.Equal` on the proof's committed digest vs the accepted root), NOT
      signature/RFC-6962/Merkle/`proof`/`didweb`/fork-shrink-equivocation code. The trust-root oracles
      still pass independently: `python3 .claude/derive_vkey.py` reproduces both golden vectors
      (`+40b74463+`, `+22b08f3e+`); `go test ./internal/logclient ./internal/proof/verify` green
      (conformance/fsck/inclusion). `.claude/.scratch/` cleaned.
- [x] §3-coverage check — the two confirmed/pending tests legitimately drop their `§3 INCLUSION PROOF`
      assertion (per plan option A: forced `acceptedRoot != tree.Hash()` makes §3 decline). §3 stays
      asserted in 13 sites including the new `…DigestMismatch` (clean tree). Not a coverage hole.
- [x] Gate-circumvention scan over unpushed range (`@{upstream}..HEAD`) — no `//nolint`, `t.Skip`,
      build-tag exclusion, swallowed error, or deleted assertion. The one matched `-` line is the
      overflow-error string moving from `Confirmed` into the shared `classify` helper (re-prefixed `ots:`,
      re-added immediately). All test helpers (`mustHex`, `otsFixtureDigest`, `seedOTSAtRoot`) are used; no
      dead code.

**Issues found:** (none new). Resolved + deleted the `normal` "Certificate §5 BITCOIN ANCHOR does not
bind the OTS proof's committed digest to §2's accepted root" issue after independently mutation-verifying
the close (both mutations fail their guarding test, tree restored clean).

**Codex second opinion:** Clean — "The change correctly binds certificate §5 rendering to OTS proofs
whose parsed digest matches the accepted checkpoint root, while preserving the existing pending/error
handling behavior. Tests pass and I did not find any introduced correctness issues." Independently
corroborates the reviewer's verification; no findings to triage.

**Visual check:** n/a — no SSR surface changed. The diff touches only the server-side §5 gate
(`ots.go` + `handler.go` `buildData`) and tests; `cert.html` and every template are byte-unchanged
(`git diff HEAD~1..HEAD -- '*.html' '*.tmpl'` empty). The binding only decides WHICH already-defined §5
state (confirmed/pending/omitted) renders, not how it looks.

**Next:** Continue draining code-closable `normal`s in handoff-named order. Remaining §5/§6-adjacent:
(1) the certificate tier-2 honesty-copy overstatement (`cert.html:465` — the `HasBundle` header asserts
present-tense "This browser re-verifies" on the no-JS baseline; a copy-only fix, code-closable now);
(2) the §6 `· at` timestamp `normal` needs a store schema column on `RecordRow` (larger, store-touching —
define carefully). The front-of-queue WASM-verifier signature half stays design-first / a STOP-candidate
(browser did:web resolution) — do a design pass before touching `verifier.html`.

**Notes:**
- 1 advance commit ahead of `origin/develop` (plus the cid context commits). Pushing on PASS. The known
  `Pages` workflow failure on develop is the human-blocked custom-domain repo-settings step, not a code
  regression.
- Learnings: `ots.md` gained the two-classifiers-on-one-core trap (use `ConfirmedFor` for any
  root-asserting caller, never `Confirmed`); `certificate.md`'s §5 KNOWN-GAP note flipped to settled.
  Net-reduced `certificate.md` (collapsed settled §1/§3/§4 + COMPARISON-ANCHOR notes) — 157 lines, 15
  bullets (marginally over the ~150 soft cap because the new §5 note is load-bearing trust-root detail;
  well within the bullet budget). No durable cross-cutting rule to promote — the digest-binding is a
  package-local application of the already-indexed "gate a rendered ✓ on a re-VERIFICATION" rule.
- Open count after this close: 0 critical / 6 normal / several low. DONE still requires 0 normal.
