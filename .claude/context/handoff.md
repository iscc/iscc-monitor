## 2026-06-21 — Review of: Gate certificate §3 inclusion proof on `!hub.Frozen` (close the self-contradictory-proof critical)

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The advance lands exactly the one-line guard `next.md` asked for
(`} else if data.HasClause2 && !hub.Frozen {`, the only non-comment code change) plus thorough
docstrings, and adds a genuinely non-vacuous `TestCertificateInclusionProofFrozen` (I reproduced the
mutation: reverting the guard makes it FAIL, restoring passes). All gates are green, scope is clean
(1 source + 1 test file), no gate weakening. BUT the chosen fix (a status-flag gate) is necessary
but NOT sufficient: Codex raised a [P1] — reviewer-confirmed against the follower source — that a
fork-poll TOCTOU window leaves the same self-contradictory §3 certificate reachable under
concurrency, so the trust-root honesty critical is narrowed, not closed.

**Verification:**
- [x] `mise run check` green (build + vet + test, all 21 packages `ok`).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` → PASS uncached (full suite incl.
  the new frozen test + the unchanged clean §3 and tile-gap tests).
- [x] Mutation reproduced INDEPENDENTLY: reverting `&& !hub.Frozen` → `} else if data.HasClause2 {`
  makes `TestCertificateInclusionProofFrozen` FAIL (the frozen hub renders §3 again); restoring it
  passes. Backup-restored, tree clean (`git diff --stat` empty).
- [x] Oracle/conformance gate uncached: `go test -count=1 ./internal/logclient ./cmd/notecheck` → both
  `ok` (no crypto path changed — the §3 builder is byte-identical, only its render condition gained the
  flag).
- [x] `gofmt -l .` empty; `git diff --stat go.mod go.sum` empty; `GOOS=js GOARCH=wasm go build
  ./internal/index ./internal/didweb` → exit 0; `proof/verify` closure still has no `net`/`net/http`/
  `database/sql`/`sqlite` (purity intact).
- [x] No gate circumvention across the unpushed commits (`@{upstream}..HEAD`): no
  `nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion; the cert-code diff is add-only
  (no removed tests or assertions).
- [x] Scope: 1 source (`handler.go`) + 1 test (`handler_test.go`); no `## Not In Scope` item touched
  (no §4–§6, no proof-bundle download, no `registry.go`/`go.mod`/CI; `cert.html` correctly untouched —
  the template already gates §3 on `{{if .HasClause3}}`).
- [ ] §3 honesty under a fork poll — FAILS (concurrency): a request landing in the TOCTOU window
  between `ingestTiles` overwriting the fork's tiles and `st.Freeze` committing the frozen flag reads
  `hub.Frozen == false` while `CheckpointAt(LastSize)` returns the old accepted root, so §3 still
  renders the fork's siblings under the old root's `✓`. The original critical is narrowed, not closed.

**Issues found:**
- **[critical, updated]** Certificate §3 can still render a self-contradictory proof in the fork-poll
  TOCTOU window — the `!hub.Frozen` gate is necessary but incomplete because the HTTP server runs
  concurrently with the follower and the flag is set (`st.Freeze`) AFTER `ingestTiles` overwrites the
  same-size tiles, in separate transactions. Steady-state frozen case IS closed by this advance; the
  race remains. The durable fix is the fail-closed `proof.VerifyInclusion`-against-the-accepted-root
  variant the §3 plan explicitly deferred until "a non-frozen divergence is demonstrated" — which this
  finding now demonstrates. Full mechanism + line refs + fix + test recipe in issues.md.

**Codex second opinion:** One [P1] — "Bind §3 proofs to a consistent accepted root"
(`handler.go:369`). **Confirmed real.** I traced the fork-poll ordering in `internal/follower/`:
`ingestTiles` (follower.go:174) overwrites same-size tile rows in per-tile `RecordTile` transactions;
`checkConsistency` then detects the fork; `freeze` (follower.go:194) calls `st.Freeze` as its LAST
store write and records the contradictory checkpoint via `RecordCheckpoint` (NOT `AdvanceAccepted`), so
`CheckpointAt(LastSize)` keeps the old root throughout. The follower and HTTP server run concurrently
(no shared lock; the flag lives in a separately-updated `follow_state` row read via `ListHubs`), so the
window between the first overwritten tile and the `st.Freeze` commit is real and the same
self-contradictory certificate is reachable. The hard oracles (notecheck, golden vectors) are green and
do not contradict this — they cover the clean signature/key path, not this concurrency window. No Codex
findings dismissed.

**Next:** Replace the status-flag gate with the fail-closed verification (the variant the plan
deferred): in `buildData`'s §3 branch, before `HasClause3 = true`, read the subject leaf's entry bundle
from the mirror, derive the leaf hash, decode `data.CheckpointRoot`, and `proof.VerifyInclusion(hasher,
data.Position, hub.LastSize, proof, acceptedRootBytes, leafHash)` — set `HasClause3` only on a nil
verdict. This fails closed against ANY tile↔root divergence (race or steady state) and subsumes the
`!hub.Frozen` gate. Extend the test: a NON-frozen contradictory-tile fixture (mirror tree A, accept
tree B's root, drop the freeze) must render §1+§2 but NO §3 `✓`, mutation-proven; `fixtureStoreTiled`
must seed a byte-accurate entry bundle for the subject leaf (it currently seeds only hash tiles). Then
resume §4 SIGNING KEY → §5 anchor → §6 record history → proof-bundle download.

**Notes:**
- This advance is a real, correct improvement (steady-state frozen case is now honest, mutation-proven)
  — it is NOT a regression and weakens no gate; it just does not fully close the trust-root critical, so
  it cannot PASS and must not push (CI is green only at `17c4957`; the whole §3 effort is unpushed).
- The fail-closed fix subsumes the freeze gate, so the next advance can REPLACE `&& !hub.Frozen` with
  the `proof.VerifyInclusion` check (one source file again); keep `TestCertificateInclusionProofFrozen`
  but make its assertion hold via the verification, not the flag (or fold it into the non-frozen
  contradictory-tile test).
- The same latent "built ≠ verified" gap exists in proofserve's `serveInclusion`, but proofserve
  returns client-verifiable JSON without a `✓` validity assertion, so it is not an honesty defect there;
  the certificate is strictly worse because it renders a verdict the page asserts as true.
- Deferred backlog unchanged: `hubDomain` ForceQuery (`normal`), ADR-0011 Go 1.26/iscc-lib (`normal`),
  five `low` items. M-UI certificate Verify is §1+§2 real + §3 landed-but-incomplete (concurrency
  honesty hole); not yet DONE.
- No remote push this cycle (NEEDS_WORK).
