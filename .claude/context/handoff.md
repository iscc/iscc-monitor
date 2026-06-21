## 2026-06-21 — Review of: Land the pure inclusion cross-check `VerifyInclusionEvidence`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `ParseInclusionEvidence` + `VerifyInclusionEvidence` land in a new
`internal/logclient/inclusioncheck.go` — the pure core of M2's second Verify half: decode a hub's
`IsccLogInclusionProof`, recompute the inclusion proof from mirrored tiles via
`InclusionProofFromTiles` (its first production-shaped caller), and byte-compare against the
hub-supplied base64-Std proof. Strictly additive (1 production file + 1 test file in `internal/logclient`,
nothing else touched), oracle-exact against the `iscc_hub` reference, and the byte-comparison is
mutation-proven load-bearing (reviewer reproduced + reverted). Every handoff claim I checked held.

**Verification:**
- [x] `mise run check` — green (all 11 packages `ok`; build + vet + test).
- [x] `go test -run TestVerifyInclusionEvidence -count=1 ./internal/logclient` — passes.
- [x] `go test -run TestParseInclusionEvidence -count=1 ./internal/logclient` — passes.
- [x] Golden returns `nil` for `{0,5,255,256,299}` on the 300-leaf `testonly.Tree`, proof built
  base64-Std from `tree.InclusionProof(index,300)` — verified across the 256-leaf tile boundary.
- [x] Mutation: corrupted proof hash AND wrong `LeafIndex` both `errors.Is(err, ErrInclusionMismatch)`;
  missing-tile fetcher `errors.Is(err, os.ErrNotExist)` and NOT `ErrInclusionMismatch` — all pass.
- [x] `gofmt -l .` — empty.
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` — exit 0 (no dep/schema change).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0 (WASM purity guard unaffected;
  `./internal/logclient` also builds for js/wasm).
- [x] **Oracle gate (APPLIES — RFC-6962 inclusion crypto):** struct + guards diffed against
  `cauldron/iscc-hub/iscc_hub/log_tree.py inclusion_evidence` + `schema.py Evidence` (shape, `treeSize
  ge=1`, `leafIndex ge=0` all match); Python `base64.b64encode` == Go `base64.StdEncoding` re-confirmed.
  Three independent merkle paths (prover / tile-builder / base64+bytes), not a tautology. Trust-root
  oracles green: `derive_vkey.py` reproduces `40b74463`/`22b08f3e`, `notecheck` passes,
  `TestRunFsck`/`TestEquivocation*`/`TestInclusionProofFromTiles*` all pass uncached.
- [x] **Mutation (reviewer-reproduced + reverted):** neutering the length+`bytes.Equal` compares
  (short-circuit to `return nil`, `bytes` still referenced) → BOTH `…CorruptedProof` and
  `…WrongLeafIndex` FAIL; revert → green. The byte-comparison is genuinely load-bearing.
- [x] **Purity:** `inclusioncheck.go` imports exactly `bytes/context/encoding/base64/encoding/json/
  errors/fmt` — no `net`/`os`/`sqlite`. The bundled `checkpoint` is decoded but never re-parsed.
- [x] **Scope:** exactly the 2 files `next.md` scoped; nothing in `## Not In Scope` was done (no
  `PollHub`/`iscc_index` wiring, no signature re-verify, no live fixtures, no go.mod/schema change).
- [x] **Gate-integrity scan** of all unpushed commits (`cfb3f48`/`0ea6c9e`/`f5cfcc7`/`275bb95`) — no
  `nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in the code diff (the two grep hits are
  removed handoff prose, not code).

**Issues found:** (none new from this slice). The four pre-existing backlog issues are untouched by this
additive slice and remain open — notably the **`critical`** "Growing equivocations can be accepted
before candidate tiles are mirrored" (`follower.go:410-416` builds the consistency proof from the local
mirror BEFORE `ingestTiles` at line 197, so a growing inconsistent root hits a missing-tile clean pass
and advances accepted state — I re-confirmed this in code). That `critical` blocks DONE but not this
slice's PASS; it lives in the follower, correctly out of scope for a logclient-only step.

**Next:** Two strong candidates for `define-next`:
1. **Fix the `critical`** growing-equivocation gap (`issues.md`): ensure the candidate-size tiles are
   mirrored before the consistency check, or make a missing proof tile a retry/error rather than a clean
   consistency pass — so a growing split view freezes and never advances to the inconsistent root.
2. **Wire `VerifyInclusionEvidence` into `PollHub`** (this slice's natural successor): needs the
   `iscc_index` projection writer (`iscc_id → seq`) to resolve a sampled leaf index + an entry-bundle to
   sample from; pass `store.SQLiteFetcher.ReadTile` straight in (signature already matches `TileFetcher`).
The `CheckpointAt ORDER BY` fix (`normal`) and the `fsckMirror` redundant-resolve (`normal`) also remain.

**Notes:**
- M2's Verify bar now has BOTH pure halves built: `fsck` root-rebuild (wired into `PollHub`) and the
  inclusion cross-check (this slice, pure + unwired). The inclusion half still needs `PollHub`/`iscc_index`
  wiring before M2 is fully met — the loop continues.
- `hubEvidenceFor(t, *testonly.Tree, index, leaves)` is the reusable "hub side" evidence builder for the
  future wiring test; it reuses `buildTree`/`tileFetcherFor`/`treeLeaves` from `proofbuilder_test.go`.
- Pushed to `origin/develop` (remote configured; branch was ahead). A human merges develop→main via CI.
