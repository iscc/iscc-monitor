# Next Work Package

## Step: Wire the iscc_index projection into PollHub's entry-bundle ingestion

## Goal
Turn the two built-but-unwired `iscc_index` halves into a running projection: as `ingestEntryBundles`
mirrors each entry bundle on a verified poll, decode it via `logclient.BundleProjections` and persist
the per-leaf records via `store.RecordProjections`. This is the first half of M2's projection bar —
after this slice a real poll populates `iscc_index`, so a later slice can resolve `iscc_id → leafIndex`
for the inclusion cross-check.

## Scope
- **Modify**: `internal/follower/ingest.go` — in `ingestEntryBundles`, after `RecordEntryBundle`
  succeeds for a bundle coord, decode the same raw bundle bytes with
  `logclient.BundleProjections(raw, baseSeq)` (baseSeq = `c.Index * tiles.TileWidth`), copy each
  `logclient.Projection → store.ProjectionRecord` at the call site (set `HubID`), and call
  `st.RecordProjections(ctx, recs)`. (1 non-test/doc file.)
- **Modify (tests, not counted toward the 3-file limit)**:
  - `internal/follower/fsck_test.go` — change `leafPreimages` to emit valid JSON-envelope records so
    `buildVerifiedMirror`'s entry bundles decode under `BundleProjections`.
  - `internal/follower/ingest_test.go` — add an integration test asserting the projection read-back.
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/projection.go` — `BundleProjections` /
    `Projection` field names + the JSON-envelope shape it decodes (`recordEnvelope`: top-level
    `iscc_id`, inner `note.$schema`).
  - `/workspace/iscc-monitor/internal/store/iscc_index.go` — `ProjectionRecord` fields +
    `RecordProjections` / `SeqsForISCCID` contracts (empty slice = no-op; absent lookup = nil).
  - `/workspace/iscc-monitor/internal/logclient/projection_test.go` lines 38-86, 101-116 — the
    canonical JSON-envelope fixture form to copy into `leafPreimages`.
  - `/workspace/iscc-monitor/internal/follower/fsck_test.go` lines 115-253 — `leafPreimages`,
    `encodeBundle`, `buildVerifiedMirror` (the tree is built from the same preimages, so changing them
    keeps the signed root self-consistent and fsck still rebuilds it).
  - `/workspace/iscc-monitor/internal/follower/ingest_test.go` lines 159-218 — `TestPollHubMirrorsTiles`,
    the integration-test pattern (poll `buildVerifiedMirror`, assert on store read-back) to mirror.

## Not In Scope
- **Do NOT wire `VerifyInclusionEvidence` / the inclusion cross-check** — that needs an
  `IsccLogInclusionProof` fixture (none captured yet) and `SeqsForISCCID`-driven `iscc_id → leafIndex`
  resolution; it is the very next slice. This step only persists the projection.
- Do NOT add an ISCC-ID codec, a deletion-status projection, or known-schema validation (ADR-0008:
  the index is a raw fold — interpret nothing).
- Do NOT touch `schema.sql`, `go.mod`, or `go.sum` (the `iscc_index` table and its index already
  exist; `BundleProjections`/`RecordProjections` are already in the build closure).
- Do NOT address the open `normal` follower issues (frozen-hub advance, `CheckpointAt` ordering,
  `AcceptCheckpoint` did.json re-fetch, tile `p`/`width` duplication) in this slice — they are weighed
  at the inclusion-cross-check slice that more deeply reworks the verified path.

## Implementation Notes
- **Where**: the projection write belongs in `ingestEntryBundles` (`ingest.go` lines 67-78), right
  after the successful `st.RecordEntryBundle` call, reusing the already-fetched `raw` bytes — do not
  re-fetch. Decode once per bundle coord and persist that bundle's slice. `RecordProjections` is
  idempotent (`ON CONFLICT(seq) DO UPDATE`), so a re-poll overwrites in place, matching the tile
  mirror.
- **baseSeq**: a bundle at tile-space `c.Index` starts at absolute leaf index `c.Index * tiles.TileWidth`
  (256). `tiles.TileWidth` is already imported in `ingest.go`. `BundleProjections` makes each leaf's
  `Seq` absolute via this base — pass `c.Index * tiles.TileWidth`, never a per-call counter.
- **Store stays a leaf (Correctness rule)**: copy `logclient.Projection → store.ProjectionRecord`
  field-by-field at the call site (`HubID: hubID, Seq, IsccID, NoteSchema, RecordSHA256`). The store
  must not import `logclient`; verify with `go list -f '{{join .Imports " "}}' ./internal/store`.
- **Error discipline (ADR-0008 + ADR-0006)**: a malformed/non-JSON record is a genuine fault, so
  propagate `BundleProjections`'s error and `RecordProjections`'s error wrapped with `%w` (e.g.
  `project entry bundle index %d: %w`). This is a decode/store fault, NOT a self-consistency violation
  — it returns up through `ingestTiles → PollHub` and aborts the poll before accepted state advances
  (same posture as a tile-write fault); it must never freeze the hub.
- **Fixture (test-only, load-bearing)**: `leafPreimages` currently emits `leaf-%d` plaintext, which is
  not valid JSON and would make `BundleProjections` error on every verified-path poll. Change it to
  emit one valid JSON envelope per leaf with a **distinct** `iscc_id` (e.g. `ISCC:` + a per-index
  suffix) and a real inner `note.$schema` (copy the `declSchema` URI from `projection_test.go`). The
  tree is rebuilt from these same preimages (`buildVerifiedMirror` calls `tree.AppendData(preimages)`
  and frames the SAME preimages into the bundles), so the signed root stays self-consistent and the
  existing fsck/mirror tests keep passing — confirm by re-running the WHOLE follower package, not just
  the new test. `leafPreimages` is used only by `buildVerifiedMirror`; `equivocation_test.go` builds
  its own inline `leaf-%d` tree and never calls `BundleProjections`, so it is unaffected.
- **Relevant learning**: "`iscc_id → seq` is one-to-many, verification is schema-agnostic (ADR-0008) —
  index by `seq`, store the raw `note.$schema`, lookups return a list." Use distinct ids per leaf in
  the fixture so the read-back assertion is a clean one-seq-per-id lookup (a one-to-many declaration +
  deletion sharing an id is already covered by the store-level `iscc_index_test.go` and can stay there).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestPollHub -count=1 ./internal/follower` passes (the existing verified-path + mirror
  tests survive the `leafPreimages` JSON-envelope change — proving the signed root stays consistent and
  fsck still rebuilds it).
- A new integration test in `ingest_test.go` (e.g. `TestPollHubRecordsProjections`) drives a verified
  `PollHub` over `buildVerifiedMirror(t, mirrorLeaves)` and asserts, via the store read seam, that the
  projection was persisted: for a known fixture leaf's `iscc_id`, `st.SeqsForISCCID(ctx, hubID, id)`
  returns the expected `[]uint64{seq}` (asserting on observable store output, never follower
  internals). `go test -run TestPollHubRecordsProjections -count=1 ./internal/follower` passes.
- `go list -f '{{join .Imports " "}}' ./internal/store | grep -E 'internal/logclient|net/http'` is
  empty (store stays a leaf).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).

## Done When
A verified `PollHub` persists the `iscc_index` projection for every mirrored entry bundle, the new
`SeqsForISCCID` read-back integration test passes, all existing follower tests stay green under the
JSON-envelope fixture, the store stays a leaf, and `mise run check` is green.
