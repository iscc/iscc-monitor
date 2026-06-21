## 2026-06-21 — Review of: Certificate §3 INCLUSION PROOF — render the real RFC-6962 proof from the mirror

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The advance wires the §3 inclusion-proof clause exactly as `next.md` asked — `buildData`
recomputes the RFC-6962 proof of `seqs[0]` against `hub.LastSize` from the mirrored tiles via
`logclient.InclusionProofFromTiles` over a `SQLiteFetcher`, base64-Std encodes each sibling, and
`cert.html` renders the leaf→siblings→root chain; the test is genuinely non-vacuous (I independently
reproduced both mutations). All gates are green and scope is clean. BUT a reviewer-confirmed Codex P2
is a real trust-root defect: for a **frozen-after-fork** hub the mirror can hold the contradictory
tree's tiles while the accepted root is the old one, so §3 can build a valid proof and render it under
a `root … ✓` the siblings do not rebuild — a self-contradictory certificate. That blocks PASS.

**Verification:**
- [x] `mise run check` green (build + vet + test, all 21 packages `ok`).
- [x] `go test -count=1 ./internal/certificate` passes uncached; `-run TestCertificate` passes.
- [x] §3 test asserts each base64-Std hash of `tree.InclusionProof(0,5)` (3 hashes, substantive) on the
  `html.UnescapeString`d body, plus `§3 INCLUSION PROOF` / `leaf · seq 0` / the accepted-root chip /
  the `N sibling hashes` note.
- [x] Mutation (a) reproduced INDEPENDENTLY: `HasClause3 = false` → `TestCertificateInclusionProof`
  FAILS. Reverted.
- [x] Mutation (b) reproduced INDEPENDENTLY: corrupt one rendered hash (prepend 0x00) → test FAILS with
  `body missing §3 inclusion-proof hash 0 …`. Reverted; tree clean afterward.
- [x] Tile-less fixture renders §1+§2 but NO §3 (`TestCertificateInclusionProofTileGap`); §1/§2
  synthetic tests still pass.
- [x] Oracle/conformance gate (re-engaged at §3): `go test -count=1 ./internal/logclient ./cmd/notecheck`
  green uncached; `derive_vkey.py` reproduces both vectors (`40b74463`/`22b08f3e`); scratch removed.
- [x] `gofmt -l .` empty; `GOOS=js GOARCH=wasm go build ./internal/index` + `./internal/didweb` exit 0.
- [x] `proof/verify` purity intact (no `net`/`net/http`/`database/sql`/sqlite in its closure); the new
  `internal/logclient` import in certificate is fine (certificate is an HTTP-surface package, not
  WASM-shared; `proofbuilder.go` is net-free at file level).
- [x] `go.mod`/`go.sum` byte-unchanged across the advance commit.
- [x] No gate circumvention across the 3 unpushed commits (no `nolint`/`t.Skip`/build-tag/swallowed-
  error/deleted-assertion; the advance is the only code change, define-next/update-state touch context).
- [x] Scope: 2 source files (`handler.go`, `cert.html`) + 1 test file — within the ≤3 budget. No
  `## Not In Scope` item touched (no §4-§6, no proof-bundle download, no `registry.go`/`go.mod`/CI).
- [ ] §3 honesty under a frozen-after-fork hub — FAILS: §3 trusts the mirror tiles blindly and can pair
  a contradictory-tree proof with the accepted root under a `✓`. Root cause + fix in issues.md.

**Issues found:**
- **[critical, new]** Certificate §3 renders a self-contradictory proof for a frozen-after-fork hub
  (built ≠ verified). `InclusionProofFromTiles` is a pure builder that never checks the proof rebuilds
  the accepted root; the follower ingests contradictory candidate tiles before the freeze check and
  `RecordTile` overwrites the rows (full tiles included), while the frozen path skips
  `AdvanceAccepted`/`fsckMirror` and keeps the old accepted root. §3 ignores `hub.Frozen` (which it has
  in hand). Fix: verify the built proof rebuilds `data.CheckpointRoot` via `proof.VerifyInclusion`
  before `HasClause3 = true` (fail-closed against any tile divergence) OR gate §3 on `!hub.Frozen`.
  Filed in issues.md with the full mechanism + line refs.

**Codex second opinion:** One [P2] finding — "Bind §3 proofs to the accepted checkpoint root"
(`handler.go:343`). **Confirmed real** and promoted to a `critical` issue. I traced the freeze/ingest
ordering in `internal/follower/follower.go` (ingestTiles at :174 runs before checkConsistency; the
frozen branch returns at :204-207 without AdvanceAccepted/fsckMirror), the no-immutability-guard upsert
in `internal/store/tiles.go:49-56`, and confirmed `InclusionProofFromTiles`
(`internal/logclient/proofbuilder.go:103`) is a builder that never verifies a root. The certificate is
strictly worse than proofserve here because it renders a `✓` validity assertion, not a client-verified
JSON. The hard oracles (notecheck, golden vectors) are green and do not contradict this — they cover
the clean path; the defect is in an untested frozen/contradictory-tile path. No findings dismissed.

**Next:** Fix the confirmed critical first: in `buildData`'s §3 branch, before `HasClause3 = true`,
verify the built proof rebuilds the accepted root (`proof.VerifyInclusion` with the leaf hash from the
mirrored entry bundle + `data.CheckpointRoot` decoded) — the truly fail-closed choice for a
self-verifiable artifact — or, if simpler is preferred, gate §3 on `!hub.Frozen`. Add a frozen-hub /
contradictory-tile fixture test that asserts §1+§2 render but NO §3 `✓`, mutation-proven (reverting the
guard makes it FAIL). Then resume the §4 SIGNING KEY clause (did:web key via `hub_keys`/`LookupHubKey`),
§5 anchor, §6 record history, and the proof-bundle download.

**Notes:**
- The same latent "built ≠ verified against an accepted root" gap exists in proofserve's
  `serveInclusion`, but proofserve returns client-verifiable JSON without a validity assertion, so it is
  not an honesty defect there. Worth a glance when the proof-bundle assembler lands (the bundle must let
  the client rebuild the root, which it does by construction).
- §1/§2 are unaffected by this defect: they read the irreplaceable accepted-checkpoint *record*
  (`CheckpointAt`), which a fork cannot corrupt. Only §3's mirror-tile read is corruptible.
- `html/template` entity-escapes `+`/`/` in base64 chips (`+`→`&#43;`); the §3 test correctly
  `html.UnescapeString`s the body. View-model strings stay byte-identical to `writeEvidence`; the
  on-page escaping is correct rendering. Recorded in `learnings/certificate.md`.
- No remote push this cycle (NEEDS_WORK). Open backlog otherwise unchanged: `hubDomain` ForceQuery
  (`normal`), ADR-0011 Go 1.26/iscc-lib (`normal`), four `low` items. M-UI certificate Verify is
  §1+§2 real + §3 landed-but-needs-the-frozen-guard; not yet DONE.
