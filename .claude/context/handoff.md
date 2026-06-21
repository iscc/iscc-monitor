## 2026-06-21 — Review of: Fail-close certificate §3 — verify the built proof rebuilds the accepted root (close the TOCTOU critical)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The §3 INCLUSION PROOF clause now gates `HasClause3` on a fail-closed
`proof.VerifyInclusion` against the §2 accepted root — a faithful port of proofserve's `serveVerify`
crypto path — replacing the racily-read `!hub.Frozen` flag entirely. This closes the open `critical`
(fork-poll TOCTOU window) by construction: the rendered ✓ is now true iff the proof it shows rebuilds
the root it shows. Scope is tight (1 non-test source file), the mutation is reproducible and
non-vacuous, and Codex independently found no bugs.

**Verification:**
- [x] `mise run check` green — all 21 packages `ok` (build + vet + test).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — PASS uncached (clean §3 proof,
  tile-gap honest-decline, and the new non-frozen contradictory test).
- [x] Oracle/conformance gate (crypto path touched): `go test -count=1 ./internal/logclient
  ./internal/proofserve ./cmd/notecheck` — all `ok`.
- [x] Mutation (non-vacuity): replacing the §3 `proof.VerifyInclusion(...) == nil` guard with
  `... == nil || true` makes `TestCertificateInclusionProofContradictory` FAIL while the clean
  `TestCertificateInclusionProof` stays PASS. Reproduced by reviewer; reverted, tree clean. (The bare
  `if true` from next.md won't compile — unused `proof`/`leafHash`; `|| true` is the same logical
  mutation.)
- [x] `gofmt -l .` empty.
- [x] `git diff --stat go.mod go.sum` empty (byte-unchanged).
- [x] `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0 (purity/WASM intact).
- [x] `git diff --stat HEAD~1..HEAD -- internal/certificate/cert.html` empty (template untouched).
- [x] Scope discipline — exactly 1 non-test source file (`handler.go`); no `## Not In Scope` work done.
- [x] No gate circumvention across unpushed Go commits (no `nolint`/`t.Skip`/build-tag/swallowed-error;
  the `else`/`case` branches that decline a clause are legitimate fail-closed, not error-swallowing).
- [x] Port fidelity — §3 mirrors `serveVerify` (handler.go:535-575): same
  `bundleIndex/offset/p` → `ReadEntryBundle` → `RecordBytesFromBundle` → `HashLeaf` → `VerifyInclusion`
  arg order. `root` reused from §2's `CheckpointAt` (meaningful only inside the `HasClause2` guard).

**Issues found:** (none) — critical `Certificate §3 ... TOCTOU` deleted as verified-fixed. Backlog
unchanged: `hubDomain` ForceQuery (`normal`), ADR-0011 Go 1.26/iscc-lib (`normal`), four `low` items.

**Codex second opinion:** "No actionable bugs were found in the HEAD diff. The new certificate §3
verification follows the existing inclusion-verification pattern and the relevant tests pass." No
findings to triage — matches the reviewer's independent conclusion.

**Next:** Resume the §3-plan continuation now that the critical is closed and this can push: §4 SIGNING
KEY → §5 Bitcoin anchor → §6 record history, then the downloadable proof-bundle assembler (which shares
this same build+verify crypto path and must keep the oracle/conformance gate green). The clean caps
already proven (`LastSize > 0`, `Position < LastSize`, entry-bundle seeding in `fixtureStoreTiled`)
carry forward.

**Notes:**
- The unpushed range is 11 commits (CI was green only at `17c4957`); this PASS pushes the whole §3
  effort (build → freeze-gate → fail-closed) to `develop` in one go. CI on `develop` is the gate.
- Out-of-scope working-tree changes are present and were correctly left uncommitted by advance:
  modified `.claude/agents/review.md`, `.claude/context/target.md`, `.devcontainer/Dockerfile`, and
  untracked `.claude/adr/0012-agent-browser-visual-verification.md` + `.claude/skills/agent-browser/`.
  These are a separate human-driven agent-browser workstream, not this work package. The reviewer did
  NOT commit them (not loop-owned); flagging for human/loop hygiene. They do not affect the verdict.
- Durable rule promoted to the index: "on a self-verifiable surface, gate a rendered ✓/Merkle assertion
  on a re-VERIFICATION, not a status flag." Applies forward to the proof-bundle assembler and the
  in-browser verifier.
