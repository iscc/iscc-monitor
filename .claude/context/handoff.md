## 2026-06-21 — Review of: Certificate §2 CHECKPOINT clause — render the accepted (size, root)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance makes the certificate's §2 CHECKPOINT clause real exactly as `next.md`
asked: `buildData`'s certifiable branch reads the accepted root back via `CheckpointAt(hub.HubID,
hub.LastSize)` and populates `CheckpointSize = hub.LastSize` + `CheckpointRoot` (base64-Std,
cross-surface-identical) + `HasClause2`; the `cert.html` `{{if .HasClause2}}` block renders
`size N · root <b64>` with a coverage-honest note reusing existing `.clause-*` CSS. Scope is clean
(2 non-test source files + 1 test file), all gates green, and both mutation checks reproduced
independently.

**Verification:**
- [x] `mise run check` green (build + vet + test) — all 21 packages `ok`, certificate uncached.
- [x] `go test -count=1 ./internal/certificate` passes uncached.
- [x] `go test -count=1 -run TestCertificateKnownID ./internal/certificate` passes; body contains
  `§2 CHECKPOINT`, `size 24816` (fixture `seq=24815` → `LastSize=seq+1=24816`), and the real
  base64-Std accepted root `cm9vdA==` (`EncodeToString([]byte("root"))`). Fixture math verified
  against `AdvanceAccepted` (sets `last_size = TreeSize`); `Position=24815 < 24816` certifies.
- [x] Non-certifiable ids render NO §2 clause — `TestCertificateNotInLog` + `TestCertificateUnacceptedLeaf`
  (both subtests) assert `§2 CHECKPOINT` absent.
- [x] Mutation (a) — neuter `data.HasClause2 = true` → `false` → `TestCertificateKnownID` FAILS
  (no `§2 CHECKPOINT`). Reproduced; tree restored.
- [x] Mutation (b) — corrupt the rendered root (`EncodeToString(append(root, 0x00))`) →
  `TestCertificateKnownID` FAILS with `body missing the §2 accepted root "cm9vdA=="`. Proves the
  assertion is grounded in the real committed root, not a hardcoded value. Reproduced; tree restored.
- [x] `gofmt -l .` empty.
- [x] `GOOS=js GOARCH=wasm go build ./internal/index` OK (no-regression).
- [x] `go.mod`/`go.sum` byte-unchanged (`encoding/base64` is stdlib).
- [x] No gate circumvention across the 3 unpushed commits (no `nolint`/`t.Skip`/build-tag/swallowed-error/
  deleted-assertion patterns in the diff).
- [x] Scope: 2 non-test source files (`handler.go`, `cert.html`) + 1 test file — within the ≤3 budget.
  No `## Not In Scope` item touched (no `registry.go`, no §3-§6, no ADR-0011 bump).
- [x] Oracle/conformance gate correctly N/A this slice (pure store read `CheckpointAt` + HTML render;
  no signature/RFC-6962/Merkle/did:web/fsck/proof path). It re-engages at §3.

**Issues found:** (none)

**Codex second opinion:** Clean verdict, no findings. Codex: "The changes consistently populate and
render the accepted checkpoint for certifiable certificate pages, and the added tests cover the main
success and non-certifiable paths. I did not identify any discrete correctness issues introduced by
this commit." Nothing to triage.

**Next:** §2 is sound. Proceed to the §3 INCLUSION PROOF clause + the downloadable proof-bundle
assembler — this is where the oracle/conformance crypto gate RE-ENGAGES: the served inclusion proof
must be mutation-proven non-vacuous against the hub's `IsccLogInclusionProof` / `notecheck` and
rebuilt over the `SQLiteFetcher`. The raw signed-note bytes are already available from
`CheckpointAt`'s second return (`raw`, ignored at §2) — wire it in at §3. After §3: §4 signing key,
§5 Bitcoin anchor, §6 record history.

**Notes:**
- The `found == false` honesty branch (leave §2 unrendered, not a 500) deliberately diverges from
  proofserve verify-for-me (which 500s on `!found`). Reviewer agrees this is the more coverage-honest
  stance for a clause-by-clause page that can decline to assert §2 — it is unreachable in practice
  (`AdvanceAccepted` records the checkpoint at the size it advances to), and it is fail-closed
  (renders the §1 banner, never fabricates a root). No issue filed; this is a sound design judgment.
- `.clause-mono` / `.clause-note` CSS already existed (cert.html:206-217) — confirmed no new CSS added.
- Open backlog unchanged by this slice: the `hubDomain` ForceQuery fail-open (`normal`, waits for a
  step that edits `registry.go`), the ADR-0011 Go 1.26 / iscc-lib bump (`normal`, foundational), and
  four `low` items (skipped by the loop). None block progress.
- M-UI's certificate Verify criterion is partially landed (§1 + §2 of 6 clauses); the milestone's
  remaining clauses (§3-§6) are the in-flight arc, so M-UI is not yet DONE.
