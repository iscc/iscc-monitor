<!-- assessed-at: the follow-traffic + mirror-repair increment committed on develop (child of 1ebdd32) -->

# Project State

## Status: IN_PROGRESS

## Phase: The production-reported egress `critical` (issue #4) is fixed, human-reviewed, COMMITTED and pushed; the M-UI human sign-off gate is unchanged.
The ISCC Hub operator reported that `monitor.iscc.io` was drawing ~434k requests/day and ~57 GB/day
of egress from a single production hub, because every poll re-walked the hub's **entire** tile
history instead of only the new coords. The fix landed in one human-directed increment (two passes:
the mirror-as-cache skip, then the trust-model consequences — admission gates, force-repair walk,
`fsckTimeout`), was reviewed by the human + Claude outside the loop, and is committed on `develop`.
The tree is CLEAN; the loop resumes its normal cadence from here. See `handoff.md` for the full
increment report.

## Convergence
- **New standing bar in `target.md`: the Follow-traffic contract.** Per-poll outbound cost must be
  proportional to a hub's **growth**, never its total log size; completed (full-width) coords are
  fetched at most once, while partials (`.p/<W>`), the signed checkpoint and `did.json` are never
  cached. It binds every present and future outbound path, is referenced from M2's Verify and from
  "Done When", and carries its own 4-item Verify list. **Currently MET.**
- **Remaining Verify criteria (per unmet milestone):**
  - **M1: 0 open. M2: 0 open** (the mirror-fill walk now also meets the new Follow-traffic clause).
    **M3: 0 open (4/4). M-Deploy: 0 open. M-API: 0 open (4/4).**
  - **M-UI: 1 Verify criterion still REOPENED** (log-browser record-list named-region parity) —
    `critical`, **FULLY code-closed**; only the human M-UI exit sign-off remains. Unchanged by this work.
  - **WASM verifier: 0 target.md Verify criteria open.** The open item is the cross-origin **signature
    half** — a design-blocked honesty `normal`, not a target.md criterion.
  - **OTS: 1/1 Verify open** — only a real Bitcoin confirmation remains (offline-unprovable). The
    daily-latest-root cadence `normal` (issues.md) is still open and IS code-closable.
- **Autonomous code-closable work EXISTS**: the OTS per-poll→daily-cadence `normal`, and the
  dead-baked-realm-hosts `normal` (`deploy/realm-testnet.txt` names hosts that no longer serve; the
  authoritative Hub-List has different ones). Neither is human- or design-blocked. The mirror-repair
  `normal` the follow-traffic change created was CLOSED in the same increment (force-repair walk +
  admission gates); only `low` residuals remain (`issues.md`).

## M1 — Read-only Monitor
**Status**: **met** — carried forward. `internal/follower/ingest.go` changed (below), but no M1
contract did: the verdict chain, freeze semantics, coverage, key resolution and `/metrics` are untouched.

## M2 — Aggregator
**Status**: **met**, and now also meets the new Follow-traffic clause.
- **`internal/store/tiles.go`** gained `TileKey` + `MirroredFullTiles` / `MirroredFullEntryBundles` —
  two hub-scoped set reads filtered on `widthForP(0)`, the same p→width authority `RecordTile` and
  `SQLiteFetcher.readTileAt` use. Empty-is-not-an-error, matching `ReadTileBlob`.
- **`internal/follower/ingest.go`**: both walks skip a coord when `c.Partial == 0 && alreadyFull`, so a
  completed tile/bundle is fetched at most once; every partial is still re-fetched every poll.
  `ingestEntryBundles` now writes the `iscc_index` projection BEFORE `RecordEntryBundle`, so the
  skippable row is the last write and "full bundle mirrored ⟹ projection written" holds.
- **Measured shape of the fix:** a 300k-entry hub enumerated 1178 hash tiles + 1172 entry bundles =
  2350 fetches per poll (matching the operator's report exactly, which is what pinned `ingestTiles` as
  the sole source). Steady state is now ~6: checkpoint + `did.json` + 3 partial tiles + 1 partial
  bundle, plus one per newly-completed coord.
- **Equivocation detection re-derived, not assumed:** `ingestTiles` still runs before
  `checkConsistency`, so every coord the proof needs is present and the missing-tile swallow is not
  newly reachable; on a rewriting hub the mirror is a MIX that cannot reconstruct both the stored prior
  root and the new signed root, so the freeze still fires; for an honest hub the mix is byte-identical
  to the real tree, so no false freeze. Detail in `learnings/follower.md`.
- **Mirror-authoritative hardening (same increment, second pass):** `RecordTile` rejects a full tile
  that is not exactly `TileWidth*32` bytes; `ingestEntryBundles` rejects a full bundle decoding to
  fewer than `TileWidth` records; `MirroredFullEntryBundles` requires the bundle's `iscc_index`
  projection to exist before calling a coord skippable; `PollHub` force-re-ingests (authoritative
  re-walk) BOTH before convicting a hub of a self-consistency violation and after a failed root
  rebuild; `fsckMirror` runs under `fsckTimeout` on its own goroutine because tessera's fsck
  deadlocks (not errors) on a corrupt completed tile — reported upstream as
  transparency-dev/tessera#1098.
- **Not touched:** `SQLiteFetcher`, `ProofBuilder`, the `iscc_index` schema, the served mirror.

## M3 · M-UI · WASM · OTS · M-Deploy · M-API
**Status**: all **carried forward unchanged** — no handler, template, `/_ds/` asset, OpenAPI document,
workflow, Dockerfile or WASM artifact was touched this window.
- **M-UI** remains blocked ONLY on the human exit sign-off (ADR-0012); `define-next` must not
  re-attempt it. The realm-index Anchor-honesty `normal` remains design-blocked.
- **OTS** carries the open, code-closable per-poll→daily-cadence `normal`.
- Carried `low` traps are unchanged; three new issues are filed (below).

## Quality gates
**Status**: **GREEN at the increment commit.** `mise run check` passes: `go build ./...`,
`go vet ./...`, `go test ./...` all `ok` across **30/30 packages**; `gofmt -l .` empty (ignoring
gitignored `cauldron/`).
- **Oracle / conformance gate: APPLIES** (the change sits upstream of the RFC-6962 consistency-proof
  input) **and is satisfied** — the equivocation argument above plus the unchanged
  `logclient`/`proof/verify`/`derive_vkey.py` vectors. `review` must re-derive it independently rather
  than accept the green suite.
- **Mutation-proven 4/4** (pass 1), each applied → FAIL observed → reverted (file restored
  byte-identical): hash-tile skip disabled · bundle skip disabled · `Partial == 0` guard dropped ·
  projection/record order swapped. The third is the one that matters most — the first version of the
  tests did NOT catch it, and `TestIngestTilesAlwaysFetchesPartials` was added specifically to close
  that hole. The 5 pass-2 load-bearing halves (length gate, short-bundle gate, projection-EXISTS
  requirement, force re-ingest before conviction, `fsckTimeout` non-stranding) are likewise
  mutation-proven (`issues.md`, mirror-repair residuals entry).
- **New tests:** `internal/store` — `TestMirroredFullTiles`, `TestMirroredFullEntryBundles`,
  `TestMirroredFullSetsAreHubScoped`. `internal/follower` — `TestIngestTilesSkipsMirroredFullCoords`,
  `TestIngestTilesGrowthFetchesOnlyNewCoords`, `TestIngestTilesAlwaysFetchesPartials`,
  `TestPollHubRefetchesCheckpointNotCompletedTiles`,
  `TestIngestEntryBundleProjectionFaultLeavesCoordRefetchable`; `repair_test.go` —
  `TestPollHubDoesNotFreezeHonestHubOnCorruptMirror`, `TestPollHubBoundsAndRepairsWedgedRootRebuild`,
  `TestIngestRejectsShortFullEntryBundle`, `TestIngestRefoldsMirroredBundleWithMissingProjection`.
  All assert at the outbound-fetch seam (the URLs the injected `Fetcher` saw) or on store read-back —
  never on follower internals.
- **Open issues: 1 critical (human-blocked), 4 normal, ~26 low.** The normals are: OTS daily cadence
  (code-closable), the dead baked realm-fallback hosts (code-closable), the WASM signature half
  (design-blocked), the realm-index Anchor honesty (design-blocked). DONE still requires 0 critical
  AND 0 normal.

## Next Milestone
The loop has autonomous work. In priority order:
1. **The OTS per-poll → daily latest-root cadence `normal`** — code-closable, spec-pinned by the
   ADR-0004 amendment, untouched by this window.
2. **The dead baked realm-fallback hosts `normal`** — `deploy/realm-testnet.txt` (and the registry
   fixture) name hosts that no longer serve; the authoritative Hub-List names different ones. A fresh
   deploy without `ISCC_MONITOR_REALM` set follows nothing.
3. **Ops (human): deploy the committed follow-traffic increment to `monitor.iscc.io`** — that is what
   actually stops the production egress.
4. **Still human/design-blocked, do NOT code-attempt:** the M-UI exit sign-off, the WASM cross-origin
   signature half, the realm-index Anchor semantics.
