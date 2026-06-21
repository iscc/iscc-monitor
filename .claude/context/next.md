# Next Work Package

## Step: Gate certificate §3 inclusion proof on `!hub.Frozen` (close the self-contradictory-proof critical)

## Advances
Closes the **open `critical`** that blocks the M-UI milestone (it preempts all feature work per
issues.md "Certificate §3 renders a self-contradictory proof for a frozen-after-fork hub (built ≠
verified)"). It restores honesty to the M-UI Verify criterion the §3 advance partially met:

> the **realm-wide certificate** (`/inclusion/{iscc_id}`) … renders the numbered evidence clauses
> (subject + position; checkpoint `(size, root)`; **inclusion proof**; …)

A certificate that renders a sibling chain under a `root … ✓` the siblings do not rebuild violates the
glossary "Proof bundle" / "Verifiable cache" contract (a client verifies the artifact itself) and
ADR-0006 (freeze preserves evidence but never advances accepted state). This must close before §4–§6
resume — the unsound clause sits on the trust-root self-verifiable surface, and the cycle cannot push
(CI is green only at `17c4957`, the last PASSed state, not at HEAD's unsound §3).

## Goal
For a frozen hub, omit the §3 INCLUSION PROOF clause entirely (the page still renders §1 SUBJECT + §2
CHECKPOINT from the irreplaceable accepted-checkpoint record), so the certificate never pairs a proof
built from the contradictory mirror tiles with an accepted root the proof does not rebuild.

## Scope
- **Create**: (none)
- **Modify**:
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — add `&& !hub.Frozen` to the §3 render
    guard at line 336 (`} else if data.HasClause2 {` → `} else if data.HasClause2 && !hub.Frozen {`),
    and update the §3 block comment + the file/`buildData`/`certData` docstrings to record the freeze
    gate and why it is the complete fix. This is the only non-test source file (≤3 budget: 1).
  - `/workspace/iscc-monitor/internal/certificate/handler_test.go` — add
    `TestCertificateInclusionProofFrozen`: a frozen hub whose mirrored tiles disagree with the accepted
    root renders §1+§2 but NO §3 `✓`; mutation-proven (reverting `&& !hub.Frozen` makes it FAIL). Test
    file, not counted toward the ≤3 budget.
- **Reference** (read before implementing):
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — the §3 "built ≠ verified"
    bullet (the exact defect + the two fix options) and the `html/template` base64-escape note.
  - `/workspace/iscc-monitor/.claude/context/learnings/follower.md` — freeze / evidence-only re-poll
    ordering (why the frozen path leaves contradictory tiles in the mirror while the accepted root is
    the old one).
  - `/workspace/iscc-monitor/internal/follower/follower.go` lines 174 (ingestTiles before
    checkConsistency), 204-207 (frozen early return skips AdvanceAccepted/fsckMirror), 221 + 241
    (verified path advances then `fsckMirror` — this proves a *non-frozen* hub's tiles already match
    the accepted root, so the freeze gate is the complete fix, not a partial one).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` line 339 (`store.Freeze(ctx, hubID)` —
    sets `frozen=1`, read back into `HubSummary.Frozen` by `ListHubs`) — the fixture's freeze lever.
  - `/workspace/iscc-monitor/internal/store/hubs.go` (`HubSummary.Frozen`, populated by `ListHubs`
    line 70) — already carried into `buildData` via `followedHub`, in hand at the §3 branch.

## Not In Scope
- The heavier `proof.VerifyInclusion`-against-the-accepted-root variant (read the entry bundle, derive
  the leaf hash, decode the accepted root, verify before `HasClause3 = true`). The verified-advance
  path already runs `fsckMirror` (follower.go:241), which root-rebuilds every non-frozen hub's mirror
  against its accepted root and would freeze on divergence, so the *only* path where mirrored tiles can
  diverge from the accepted root is the frozen path — the freeze gate closes the exact filed defect
  with no entry-bundle plumbing (KISS / YAGNI; `fixtureStoreTiled` does not even seed entry bundles).
  Revisit the full proof-verification approach only if a non-frozen divergence is ever demonstrated.
- §4 SIGNING KEY, §5 Bitcoin anchor, §6 record history, and the downloadable proof-bundle assembler —
  the next steps once this critical is cleared and pushed.
- The deferred `normal` registry `hubDomain` `ForceQuery` fix (fold in only when `registry.go` is next
  touched; this step does not touch it).
- The ADR-0011 Go 1.26 / iscc-lib stack bump (separate `normal`).
- Any `cert.html` change — the template already gates §3 on `{{if .HasClause3}}` (cert.html:345), so a
  false `HasClause3` omits the clause with no markup edit. Keeping `go.mod`/`go.sum` byte-unchanged
  (no new imports needed — `hub.Frozen` is already in scope).

## Implementation Notes
- **The one-line guard.** In `buildData`'s §3 branch (handler.go:336), change
  `} else if data.HasClause2 {` to `} else if data.HasClause2 && !hub.Frozen {`. `hub` is the
  `store.HubSummary` returned by `followedHub`; `hub.Frozen` is already populated by `ListHubs`
  (hubs.go:70) and is in scope here. Leave the `os.ErrNotExist` honest-gap branch and the
  non-`os.ErrNotExist` 500 branch unchanged — a frozen hub simply takes neither: the proof may still
  *build* (the tiles are present), but it is never rendered. No new imports.
- **Why the freeze gate is complete, not partial (record in the docstring).** ADR-0006 / follower.go:
  the verified-advance path calls `AdvanceAccepted` (221) then `fsckMirror` (241), which rebuilds the
  accepted root from the mirror and would freeze on divergence; the frozen path returns early
  (204-207) BEFORE `AdvanceAccepted`/`fsckMirror`, leaving the contradictory candidate tiles ingested
  at :174 in the mirror while `CheckpointAt(LastSize)` still returns the old accepted root. So a
  non-frozen hub's mirror is fsck-consistent with its accepted root by construction; the frozen hub is
  the sole divergence window. Fail-closed (ADR-0001): when in doubt about the mirror, decline the clause.
- **§1/§2 are unaffected** — they read the irreplaceable accepted-checkpoint *record* (`CheckpointAt`),
  which a fork cannot corrupt; the §1 cap reasoning ("a frozen hub's LastSize caps it at its accepted
  window") already covers them with no frozen branch. Only §3's mirror-tile read is corruptible, so
  only §3 gets the freeze gate. The frozen Exhibit / status surface is rendered elsewhere; this clause
  just declines to assert a Merkle proof it cannot honestly pair with the accepted root.
- **Test fixture (thin variant of the existing `fixtureStoreTiled`, handler_test.go:473).** Build a
  fixture where the mirrored tiles and the accepted root belong to DIFFERENT trees, then freeze:
  1. Build tree A (`leaves=5`) and seed its hash tiles via `RecordTile` (reuse the existing
     `fixtureStoreTiled` tile-ingest loop over `tiles.TileCoords(size)` and the `treeNodeHash` helper).
  2. `AdvanceAccepted` with `Root:` a DIFFERENT 32-byte root than `treeA.Hash()` — simplest is a second
     tree B of the same size (`Root: treeB.Hash()`), so the accepted root the §3 proof would have to
     rebuild does NOT match the mirrored (tree-A) tiles (the contradictory-tile case). Index the golden
     leaf under `"ISCC:" + goldenID` at a seq `< size`.
  3. `st.Freeze(ctx, target)` (checkpoints.go:339) so `ListHubs` reports `Frozen == true`.
     Prefer factoring a small `fixtureStoreFrozenContradictory` helper OR extending `fixtureStoreTiled`
     with an `acceptedRoot []byte` + `freeze bool` parameter (pass `treeA.Hash()`/`false` from the
     existing clean caller, divergent values from the new one) — keep it a thin variant, do NOT
     duplicate the whole tile loop.
  Assert (HTTP seam, the certificate's only contract): `200 text/html`; body contains `§1 SUBJECT`
  and `§2 CHECKPOINT` (the page is NOT blank); body does NOT contain `§3 INCLUSION PROOF`. The
  §3-absence asserts are on plain markers, so no `html.UnescapeString` is needed (it is required only
  when asserting on base64 chips — cert.html escapes `+`→`&#43;`; see learnings/certificate.md).
- **Non-vacuity (mandatory — review reproduces it).** Reverting `&& !hub.Frozen` must make the new test
  FAIL (the frozen hub would then render `§3 INCLUSION PROOF` again). State this in the test docstring
  and confirm it locally before handing off. Leave `TestCertificateInclusionProof` (clean, non-frozen)
  and `TestCertificateInclusionProofTileGap` (honest tile gap) untouched and green — together the three
  cover: clean→§3 renders, tile-gap→§3 omitted, frozen-contradictory→§3 omitted.
- **Correctness rules in play** (learnings.md index): "A self-consistency violation freezes, never
  crashes (ADR-0006)" — a frozen hub keeps polling evidence-only and never advances accepted state, so
  its mirror can hold post-freeze contradictory tiles; "Coverage honesty (ADR-0001)" — never render a
  guarantee the accepted state does not support.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes uncached (the full certificate
  suite, including the new frozen test and the unchanged clean §3 + tile-gap tests).
- Mutation check: reverting `&& !hub.Frozen` to `} else if data.HasClause2 {` makes
  `go test -run TestCertificateInclusionProofFrozen ./internal/certificate` FAIL; restoring it passes.
- The new test asserts the frozen-contradictory fixture's body contains `§1 SUBJECT` and
  `§2 CHECKPOINT` but NOT `§3 INCLUSION PROOF`, at `200 text/html`.
- Oracle/conformance gate stays green (no crypto path changed; the §3 builder is unchanged, only its
  render is gated): `go test -count=1 ./internal/logclient ./cmd/notecheck` passes uncached.
- `go.mod` / `go.sum` byte-unchanged (`git diff --stat go.mod go.sum` empty);
  `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0 (no new imports in
  `certificate`, purity unregressed).

## Done When
`mise run check` is green, the new `TestCertificateInclusionProofFrozen` passes and is mutation-proven
(reverting the `!hub.Frozen` guard fails it), and a frozen-after-fork hub's certificate renders §1+§2
but never a §3 `✓` — closing the open `critical` so the milestone can resume and the cycle can push.
