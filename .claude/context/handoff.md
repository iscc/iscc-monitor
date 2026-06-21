## 2026-06-21 — Review of: Wire RunFsck into PollHub over the live SQLiteFetcher mirror

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** `fsckMirror` is wired into `PollHub` on the verified, non-violation path AFTER
`ingestTiles` and BEFORE the final `recordVerdict` — `logclient.RunFsck`'s first production caller —
and rebuilds the RFC-6962 root from the local `SQLiteFetcher` mirror, cross-checking it against the
signed checkpoint root. A rebuild mismatch is surfaced as a genuine fault (NOT a violation, no freeze
— ADR-0006). The diff is clean (1 production file + tests), the mutation is non-vacuous, and the
oracle parity holds. The advance's HUMAN REVIEW REQUESTED (converting 6 real-sb0 tests to the
in-process mirror) is, on independent review, a **sound equivalent with no trust-root coverage loss**
— I cleared it; no human ratification needed.

**Resolution of the advance's HUMAN REVIEW REQUESTED (the 3 questions posed):**
1. **Coverage/gate weakened?** No. Net test assertions went UP (32 added, 16 removed); the 16 removed
   were hardcoded `10183`/`0x40b74463` literals replaced by fixture-relative `m.size`/`m.keyID` (same
   assertions, no magic numbers). No `t.Skip`/`//nolint`/build-tag/swallowed error introduced. The
   converted tests now assert MORE — they exercise the full mirror→fsck→rebuild path the real-sb0
   fixture could never reach (it would fail fsck for lack of captured leaf preimages).
2. **Real-sb0 parity still covered elsewhere?** Yes, at the correct (verification) layer.
   `internal/logclient/accept_test.go::TestAcceptCheckpoint/"verified"` verifies the REAL sb0
   checkpoint (size 10183, real ed25519 sig) against the REAL captured `sb0.iscc.id_did.json` key →
   `StatusVerified`/`Origin=sb0.iscc.id/log`/`TreeSize=10183`. The `notecheck` external oracle accepts
   it (`OK sb0.iscc.id/log`, exit 0); `derive_vkey.py` reproduces `40b74463`/`22b08f3e`; the real sb0
   checkpoint fixture is still consumed by `TestPollHubUnverifiedDoesNotAdvance`.
3. **Need human ratification?** No. The retirement is a forced *technical* consequence of the slice
   `next.md` asked for (fsck on the live path requires a byte-accurate mirror; the real log's leaf
   preimages were never captured, live capture is Not In Scope), resolved with the SAME `testonly.Tree`
   fixture style `next.md` itself prescribed for `fsck_test.go`. No plan/ADR deviation, no public-API
   change, no gate weakening — out of STOP scope. PASS, not STOP.

**Verification:**
- [x] `mise run check` — green (all 11 packages `ok`; re-ran follower/logclient/didweb/notecheck uncached).
- [x] `gofmt -l .` — empty.
- [x] `TestPollHubFsck` (good=nil, corrupt=non-nil) — both subtests pass.
- [x] Existing `TestPollHub|TestIngest|TestWidthForP|TestEquivocation|TestTick` paths — pass.
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` — exit 0 (byte-identical).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` — exit 0.
- [x] Production follower import set unchanged — no import outside `{context, fmt, logclient, metrics,
  store, tiles, log/slog}`.
- [x] **Mutation (reviewer-reproduced + reverted):** `fsckMirror` early-`return nil` →
  `RejectsCorruptedMirror` FAILS; revert → green. Rebuild genuinely compares against the signed root.
- [x] **Oracle gate (APPLIES — RFC-6962 root-rebuild on the live path):** `derive_vkey.py` reproduces
  both golden vectors; `notecheck` accepts the real sb0 checkpoint (`OK sb0.iscc.id/log`, exit 0);
  `TestRunFsck` (logclient) + `TestEquivocation*` + `TestPollHubFsck` all pass uncached. Trust root green.
- [x] Gate-integrity scan of all 3 unpushed commits — no nolint/skip/build-tag/swallowed-error/deleted-assertion.
- [x] Scope — exactly 1 production file (`follower.go`); `internal/store`, `cmd/`, the inclusion
  cross-check (`InclusionProofFromTiles`) all untouched, as scoped.

**Issues found:** (no blockers) Filed two `normal` `[review]` issues, both surfaced/observed by this slice:
- `CheckpointAt` unordered `LIMIT 1` → fork re-detection compares against an undefined row post-freeze
  (forced `TestPollHubFork` re-detection to use `freeze` directly; first detection still via `PollHub`,
  shrink re-detection still via `Tick`). Fix belongs to the store-touching equivocation/serving slice.
- `fsckMirror` re-resolves the did:web key every verified poll (the +1 did.json fetch the cache-hit
  test now pins at cold=3/warm=2). Efficiency-only; thread the already-resolved vkey through.

**Next:** The **inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof` — the SECOND
half of M2's Verify bar. `InclusionProofFromTiles` already exists as an unwired seam; it needs captured
`IsccLogInclusionProof` fixtures (real or in-process). Alternatively, `define-next` could pick up the
`CheckpointAt ORDER BY` fix (`normal`, unblocks deterministic fork re-detection via `PollHub`) or the
`fsckMirror` redundant-resolve efficiency item — both are now tracked in `issues.md`.

**Notes:**
- M2's Verify bar is now HALF landed: `fsck` rebuilds each accepted root from the `SQLiteFetcher` on
  every verified poll (this slice). The inclusion cross-check is the remaining half — not yet a DONE
  milestone, so the loop continues.
- `buildVerifiedMirror(t, leaves)` is now the reusable follower verified-path fixture (per-run keypair,
  byte-accurate tiles/bundles across the 256-leaf boundary at `mirrorLeaves=300`). Future verified-path
  follower tests should use it, not the real sb0 checkpoint (which fails fsck on the live path).
- `mirrorBundleFetcher` routes by `strings.HasSuffix` over `byPath`; verified no suffix collisions for
  size-300 tlog-tiles paths. A future colliding-path fixture would want exact/longest-suffix matching.
- Pushed to `origin/develop` (remote configured; branch was ahead 3). A human merges develop→main via CI.
