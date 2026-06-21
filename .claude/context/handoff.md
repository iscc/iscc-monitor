## 2026-06-21 — Gate certificate §3 inclusion proof on `!hub.Frozen` (close the self-contradictory-proof critical)

**Done:** Gated the certificate's §3 INCLUSION PROOF clause on `!hub.Frozen` (the one-line guard
`} else if data.HasClause2 && !hub.Frozen {` in `buildData`), so a frozen-after-fork hub renders §1
SUBJECT + §2 CHECKPOINT (from the irreplaceable accepted-checkpoint record) but never a §3 proof built
from contradictory mirror tiles under an accepted root those tiles do not rebuild. Recorded the
freeze-gate mechanism and why it is the complete (not partial) fix in the §3 block comment + the
file/`buildData`/`certData` docstrings. Closes the open `critical` blocking M-UI.

**Files changed:**
- `internal/certificate/handler.go`: added `&& !hub.Frozen` to the §3 render guard; added a freeze-gate
  paragraph at the §3 branch explaining the follower freeze/ingest ordering (ingest tiles at :174 →
  frozen early-return at :204-207 before AdvanceAccepted/fsckMirror at :221/:241, so the frozen path is
  the sole mirror↔accepted-root divergence window); updated the file-level, `buildData` step-7,
  `certData`, `ProofHashes`, and `HasClause3` docstrings to record the gate. (1 source file, ≤3 budget.)
- `internal/certificate/handler_test.go`: extended `fixtureStoreTiled` with `acceptedRoot []byte` (nil →
  the mirrored tree's own root for the clean caller) + `freeze bool` parameters — a thin variant, no
  duplicated tile loop; added `TestCertificateInclusionProofFrozen` (mirror tree A's tiles, accept tree
  B's root, freeze) asserting `200 text/html` with `§1 SUBJECT` + `§2 CHECKPOINT` present and
  `§3 INCLUSION PROOF` absent. Updated the existing clean §3 caller to pass `(nil, false)`.

**Verification:** `mise run check` → green (build + vet + test, all 21 packages `ok`). Per-criterion:
- `go test -count=1 -run TestCertificate ./internal/certificate` → PASS (full suite incl. the new frozen
  test + the unchanged clean §3 and tile-gap tests).
- Mutation check → PASS: reverting `&& !hub.Frozen` to `} else if data.HasClause2 {` makes
  `TestCertificateInclusionProofFrozen` FAIL (frozen hub renders §3 again, confirmed by run); restoring
  it passes. Backup-restored, tree clean afterward.
- Oracle/conformance gate uncached: `go test -count=1 ./internal/logclient ./cmd/notecheck` → both `ok`.
- `gofmt -l .` empty; `git diff --stat go.mod go.sum` empty; `GOOS=js GOARCH=wasm go build
  ./internal/index ./internal/didweb` → exit 0 (no new imports, purity unregressed).
- Scope: 2 files (1 source + 1 test), no `## Not In Scope` item touched.

**Next:** Resume the §4 SIGNING KEY clause (did:web key via `hub_keys`/`LookupHubKey`), then §5 Bitcoin
anchor, §6 record history, and the downloadable proof-bundle assembler — the milestone can now push.
When the proof-bundle assembler lands, mind the same "built ≠ verified against an accepted root" gap
noted for proofserve's `serveInclusion` (the bundle must let the client rebuild the root, which it does
by construction).

**Notes:**
- The freeze gate is the COMPLETE fix, not partial: the verified-advance path runs
  `AdvanceAccepted`→`fsckMirror` (which root-rebuilds every non-frozen hub's mirror against its accepted
  root and freezes on divergence), so a non-frozen hub's mirror is fsck-consistent with its accepted
  root by construction. The frozen path is the only divergence window. The heavier
  `proof.VerifyInclusion`-against-the-accepted-root variant was deliberately NOT taken (Not In Scope,
  KISS/YAGNI — `fixtureStoreTiled` does not even seed entry bundles); revisit only if a non-frozen
  divergence is ever demonstrated.
- §1/§2 unaffected (they read `CheckpointAt`, the irreplaceable accepted-checkpoint record a fork cannot
  corrupt); only §3's mirror-tile read needed the gate.
- `cert.html` was NOT touched — the template already gates §3 on `{{if .HasClause3}}`, so a false
  `HasClause3` omits the clause with no markup edit.
- The frozen test's §3-absence assert is on the plain `§3 INCLUSION PROOF` marker, so no
  `html.UnescapeString` is needed (that is required only when asserting on base64 chips, which escape
  `+`→`&#43;`; see learnings/certificate.md).
- Deferred backlog unchanged: `hubDomain` ForceQuery (`normal`), ADR-0011 Go 1.26/iscc-lib (`normal`).
