## 2026-06-21 — Review of: Make the certificate §1 inclusion claim sound — accepted-tree cap + ISCC:-prefixed lookup

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance closes both open `critical` certificate defects in one slice exactly as
`next.md` asked: `buildData` now canonicalizes the lookup id to the stored `ISCC:`-prefixed form and
gates the affirmative §1 inclusion claim on the accepted-tree cap (`len(seqs) > 0 && seqs[0] <
LastSize`), with honest cannot-certify states otherwise. `LastSize` is carried out of the existing
`ListHubs` scan via a `followedHub` return-type change (no second store round-trip), and fixtures are
re-grounded to production's wire format (prefixed id + an accepted checkpoint). Scope is clean (one
non-test source file), all gates green, and both mutation checks reproduced independently.

**Verification:**
- [x] `mise run check` green (build + vet + test) — confirmed uncached: build + vet pass, all 21 packages `ok`.
- [x] `go test -count=1 ./internal/certificate` passes uncached — 11 tests pass.
- [x] `TestCertificateKnownID` — passes: bare-suffix request certifies the prefixed leaf within the accepted tree, position 24815, sb1.amlet.id.
- [x] `TestCertificateUnacceptedLeaf` (both subtests) — `seqs[0] >= LastSize` → "not in accepted tree"; `LastSize == 0` → "no accepted checkpoint yet"; neither renders the subject banner.
- [x] `TestCertificatePrefixedLookup` — `/inclusion/MAIGHFECJMOPMIAB` and `/inclusion/ISCC:MAIGHFECJMOPMIAB` both certify the same leaf.
- [x] Mutation (a) — neutering the `LastSize == 0` / `seqs[0] >= LastSize` cap → `TestCertificateUnacceptedLeaf` FAILS (renders the certifiable banner). Reproduced; tree restored.
- [x] Mutation (b) — reverting the lookup to bare `rawID` → `TestCertificateKnownID` + `TestCertificatePrefixedLookup` FAIL ("not found in log"). Reproduced; tree restored.
- [x] `gofmt -l .` empty.
- [x] No gate circumvention in unpushed commits (no `nolint`/`t.Skip`/build-tag/deleted-test patterns); `go.mod`/`go.sum` byte-unchanged (no new deps).
- [x] Scope: exactly 1 non-test source file (`handler.go`); 2 test files (`certificate/handler_test.go` + `cmd/iscc-monitor/main_test.go`, both re-grounding the same two bugs). Within budget.

**Issues found:** (none) — both `critical` issues are resolved by this slice and deleted from `issues.md`.

**Codex second opinion:** Clean verdict, no findings. Codex confirms: "the certificate handler now
correctly normalizes lookups to the prefixed stored form and gates affirmative inclusion on the
accepted tree size. The added tests cover the key regression paths, and the full test suite passes."
Nothing to triage.

**Next:** §1 is now sound. Proceed to the §2 Checkpoint clause (`HasClause2`): render the accepted
`(size, root)` the cap already keys on, reusing the `HubSummary.LastSize` carry this slice introduced
(read `FollowState`/`CheckpointAt(hubID, LastSize)`). After §2, §3 Inclusion proof re-engages the
oracle/conformance gate — the served proof bundle's inclusion proof must be mutation-proven
non-vacuous against the hub's `IsccLogInclusionProof`.

**Notes:**
- Oracle/conformance gate is N/A this slice: still a pure HTML render of decode + registry resolve +
  store reads (no signature/RFC-6962/Merkle/did:web/fsck/proof path). The gate APPLIES starting at §3.
- The boundary is correct: a tree of size N has leaves 0..N-1, so seq `LastSize-1` is the last
  certifiable leaf and `seqs[0] >= LastSize` is the right reject condition (verified against
  `AdvanceAccepted`, which sets `last_size = TreeSize`).
- No template edit was needed (as predicted): both new cannot-certify states route through the
  existing `cert.html` `{{else}}` not-found branch via `data.Reason`; the "is included in the
  transparency log of" banner stays under `{{if .Certifiable}}`.
- `cmd/iscc-monitor/main_test.go` was a justified third file (TEST file, not counted against the
  ≤3 non-test source budget): the same two bugs were baked into its end-to-end fixture, so re-grounding
  it to ground truth was required to keep `mise run check` green.
- `learnings/certificate.md`'s two OPEN review-blocking bullets are marked resolved (collapsed into
  settled notes with the mutation proof recorded); the index gist updated to drop "OPEN".
