## 2026-06-21 — Wire the iscc_index projection into PollHub's entry-bundle ingestion

**Done:** `ingestEntryBundles` now folds every mirrored entry bundle into the schema-agnostic
`iscc_index` projection: after `RecordEntryBundle` succeeds it decodes the same raw bytes via
`logclient.BundleProjections(raw, c.Index*tiles.TileWidth)`, copies each `logclient.Projection →
store.ProjectionRecord` at the call site (store stays a leaf), and `RecordProjections`. A verified
`PollHub` now populates `iscc_index`, unblocking the later `iscc_id → leafIndex` inclusion cross-check.

**Files changed:**
- `internal/follower/ingest.go`: added `projectEntryBundle` helper called from `ingestEntryBundles`
  right after `RecordEntryBundle`; updated the file/func doc comments. (1 non-test file.)
- `internal/follower/fsck_test.go`: `leafPreimages` now emits valid JSON-envelope records
  (`{iscc_id, note:{$schema}}`) with a distinct per-leaf `iscc_id` (`leafISCCID(i)`) and the real
  `iscc-note-0.8.0` declaration schema (`leafDeclSchema`), so `buildVerifiedMirror`'s bundles decode
  under `BundleProjections`. The tree is rebuilt from these same preimages, so the signed root stays
  self-consistent and fsck still rebuilds it.
- `internal/follower/ingest_test.go`: added `TestPollHubRecordsProjections` (the projection read-back
  integration test); made `recordingFetcher` serve a valid framed entry bundle for entries-path URLs
  (so the projection fold decodes them) via `recordingBundleBody`, and updated
  `TestIngestTilesWidthMapping`'s bundle read-back to compare against that body.

**Verification:** `mise run check` → GREEN (all 11 packages `ok`; build + vet + test). Per-criterion:
- [x] `go test -run TestPollHub -count=1 ./internal/follower` PASS (uncached) — existing verified-path
  + mirror + fsck tests survive the `leafPreimages` JSON-envelope change.
- [x] `go test -run TestPollHubRecordsProjections -count=1 ./internal/follower` PASS (uncached).
- [x] `gofmt -l .` empty.
- [x] store stays a leaf: `go list -f '{{join .Imports " "}}' ./internal/store | grep -E
  'internal/logclient|net/http'` empty.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).
- [x] Full `./internal/follower` + `./internal/logclient` pass uncached.

**Conformance (this slice touches the verified proof path):**
- fsck root-rebuild over the `SQLiteFetcher` (`TestPollHubFsck`) still passes over the changed JSON
  envelopes — "Successfully fsck'd log with size 5/300", proving the fixture change kept the signed
  root self-consistent. The corrupted-mirror subtest still FAILs the rebuild (non-vacuous).
- `internal/logclient` golden-vector / inclusion / equivocation suite passes uncached.
- The independent `notecheck` oracle runs in CI (not `go run` locally) — unchanged by this slice.

**Mutation check (advance, reverted):** forcing the baseSeq to constant `0`
(`BundleProjections(raw, 0)`) made `TestPollHubRecordsProjections` FAIL — bundle-1 leaves (seq ≥ 256)
collided onto bundle-0 seqs under `ON CONFLICT(seq) DO UPDATE`, so `SeqsForISCCID` returned the wrong
seqs. The `bundleIndex*tiles.TileWidth` absolute-seq math is load-bearing and proven non-vacuous.

**Next:** Wire the inclusion cross-check — resolve a sampled `iscc_id → leafIndex` via `SeqsForISCCID`
and feed `logclient.VerifyInclusionEvidence` over the `SQLiteFetcher` (`ReadTile`) against the hub's
own `IsccLogInclusionProof`. That needs an `IsccLogInclusionProof` fixture (none captured yet) and is
the natural moment to weigh the open `normal` follower issues (frozen-hub advance, `CheckpointAt`
ordering, `AcceptCheckpoint` did.json re-fetch, tile `p`/`width` duplication) since it reworks the
verified path more deeply.

**Notes:**
- **`TestIngestTilesWidthMapping` required a test-fixture fix, not just an additive test.** The
  width-mapping unit test's `recordingFetcher` served opaque `"body:"+url` bytes for entry bundles;
  once `ingestEntryBundles` decodes each bundle, those non-framed bytes are a genuine
  malformed-bundle fault and the poll aborts. The fetcher now frames a single valid JSON record (its
  `iscc_id` embeds the URL, so bytes stay URL-unique) for entries-path URLs only; tile URLs keep the
  opaque form, and the bundle read-back asserts against the same framed body. The width-mapping
  assertions (the test's actual purpose) are unchanged.
- **Error discipline:** a malformed/non-JSON record or a store fault propagates wrapped (`project
  entry bundle index %d: %w`) up through `ingestTiles → PollHub`, aborting the poll before accepted
  state advances — same posture as a tile-write fault, NOT a self-consistency violation, never freezes
  the hub (ADR-0008 + ADR-0006).
- **Store stays a leaf** — `logclient.Projection → store.ProjectionRecord` is copied field-by-field at
  the call site in `ingest.go`; the store never imports `logclient`.
- `equivocation_test.go` is unaffected — it builds its own inline `leaf-%d` tree and never calls
  `BundleProjections`. Nothing in `## Not In Scope` was touched (no `VerifyInclusionEvidence` wiring,
  no ISCC-ID codec, no deletion projection, no `schema.sql`/`go.mod`/`go.sum` change, open `normal`
  follower issues untouched).
