# Next Work Package

## Step: Stop re-walking a hub's whole tile history every poll — mirror the immutable coords once

## Advances

The **Follow-traffic contract** in `target.md` (new standing bar, ratified with this step) plus the
**M2 — Aggregator** Verify clause that now points at it. This is a `critical` correctness-of-conduct
gap surfaced from production, not a milestone gap: the monitor's per-poll outbound cost was
proportional to a hub's **total log size** instead of to its **growth**.

Measured on `monitor.iscc.io` against the production ISCC Hub (reported by the hub operator):
~434k requests/day and ~57 GB/day of egress, decomposing as ~2350 requests per poll walk × ~187
walks/day. That decomposition matches the enumeration exactly for a 300k-entry log — `TileCoords`
yields 1178 hash tiles and `BundleCoords` 1172 entry bundles = 2350 — which pins `ingestTiles` as the
sole source. The cost multiplies with every hub and every appended record.

## Goal

Fetch only what can have changed. A completed tlog-tiles tile or entry bundle is immutable (hubs
already serve those paths `cache-control: public, max-age=31536000, immutable`), so once its bytes
are in the mirror there is nothing to re-fetch — **the mirror IS the cache** (ADR-0005: one store, no
second copy on disk, no new config, no eviction policy). Partials (`.p/<W>`) and the signed
checkpoint stay uncached: the partial gains leaves on every append, and fetching the checkpoint fresh
*is* the observation.

## Scope

- **Create**: none
- **Modify** (2 non-test source files, within the ≤3 budget):
  - `internal/store/tiles.go` — add the set-shaped read of already-mirrored-in-full coords
    (`TileKey`, `MirroredFullTiles`, `MirroredFullEntryBundles`), keyed on the same `widthForP(0)`
    the write and `SQLiteFetcher` read sides use, so the three cannot drift.
  - `internal/follower/ingest.go` — consult those sets in the two walks and skip a coord that is
    already mirrored at full width; keep every partial unconditionally re-fetched.
- **Tests**: `internal/store/tiles_test.go`, `internal/follower/ingest_test.go`,
  `internal/follower/follower_test.go` (the existing `countingFetcher` gains a `urls` record so a
  PollHub-level test can assert what one observation costs on the wire).
- **Reference**: `target.md` Follow-traffic contract; ADR-0005 partial-tile discipline;
  `learnings/follower.md` (the load-bearing `ingestTiles → checkConsistency` order, which this step
  must not disturb).

## Not In Scope

- **No HTTP-layer cache and no on-disk cache directory.** An `httpcache` transport or a second
  filesystem cache would duplicate every mirrored byte and add an eviction policy ADR-0005
  deliberately does not have. The store already holds exactly these bytes, keyed by coordinate.
- **Do NOT cache the checkpoint or `did.json`.** Both must be fetched fresh every poll.
- **Do NOT reorder `ingestTiles` relative to `checkConsistency`** — mirroring the candidate tiles
  before the consistency check is a closed `critical` (`learnings/follower.md`).
- **No mirror-repair path.** Making the mirror authoritative for completed coords removes the
  incidental self-heal the unconditional re-fetch provided; that is filed as its own `normal` issue,
  not folded in here.
- No metrics for skipped-vs-fetched coords (a `low`, folded into the existing scaling trip-wire issue).

## Implementation Notes

- The skip condition is `c.Partial == 0 && alreadyMirroredInFull`. **Both halves are load-bearing**:
  dropping the `Partial == 0` guard would let a coord that is full in the mirror suppress the fetch of
  the *same coord enumerated as a partial*, which happens on a **shrunk** observation — exactly the
  case where the contradicting hub's own bytes must be mirrored as evidence (ADR-0006).
- `ingestEntryBundles` writes the `iscc_index` projection **before** `RecordEntryBundle`. The bundle
  row is what makes a coord skippable, so writing it last upholds "a full bundle in the mirror implies
  its projection was written". With the writes the other way round a projection fault would leave a
  row every later walk skips, making the index gap permanent.
- The set reads are two queries per poll, not one query per coord: the store opens with
  `SetMaxOpenConns(1)`, so per-coord lookups would serialise thousands of round-trips against the
  single writer connection shared by every hub in the realm.

## Verification

- `go test -count=1 -run 'TestIngestTiles|TestPollHubRefetches|TestIngestEntryBundle' ./internal/follower`
  and `go test -count=1 -run TestMirroredFull ./internal/store` pass.
- **Mutation-proven non-vacuous** (each mutation applied, test FAILs, mutation reverted):
  - hash-tile skip never fires → `TestIngestTilesSkipsMirroredFullCoords`,
    `TestIngestTilesGrowthFetchesOnlyNewCoords`, `TestPollHubRefetchesCheckpointNotCompletedTiles` FAIL;
  - entry-bundle skip never fires → the same three FAIL;
  - drop the `Partial == 0` guard (cache partials too) → `TestIngestTilesAlwaysFetchesPartials` FAILs;
  - swap the projection/record order → `TestIngestEntryBundleProjectionFaultLeavesCoordRefetchable` FAILs.
- The checkpoint is still fetched every poll — `TestPollHubRefetchesCheckpointNotCompletedTiles`
  asserts the `https://sb0.iscc.id/log/checkpoint` URL appears in the *second* poll's fetches.
- `mise run check` green (30/30 packages), `gofmt -l .` empty.

## Done When

A poll of an unchanged tree fetches the checkpoint and zero completed tiles or entry bundles; a poll
of a grown tree fetches only the coords the mirror lacks; a `.p/<W>` path is never served from the
mirror; the `iscc_index` projection can never be permanently skipped; and `mise run check` is green.
