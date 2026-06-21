## 2026-06-21 — `iscc_index` store writer + `iscc_id → []seq` read-back

**Done:** Added `internal/store/iscc_index.go` — the store half of the M2 `iscc_index` projection
(ADR-0008). `RecordProjections` upserts a batch of per-leaf `ProjectionRecord`s keyed on `seq` (the
PRIMARY KEY) with an idempotent `ON CONFLICT(seq) DO UPDATE`, and `SeqsForISCCID` is the one-to-many
`iscc_id → []seq` reader (BLOB-column lookup over the existing `iscc_index_by_iscc_id` index,
ascending). The store stays a leaf (no logclient/net/http import) and `iscc_id` is stored raw into
both the BLOB and TEXT columns, interpreting nothing.

**Files changed:**
- `internal/store/iscc_index.go` (new, non-test): `ProjectionRecord` struct + `RecordProjections`
  writer + `SeqsForISCCID` reader. Mirrors `tiles.go`'s composite-PK upsert and `checkpoints.go`'s
  store-owned-plain-struct + "absent is not an error" conventions.
- `internal/store/iscc_index_test.go` (new, test, not counted): 8 table-style tests — round-trip,
  empty-batch no-op, one-to-many (declaration + deletion at two seqs), ORDER BY determinism over
  descending input, absent-is-empty, idempotency (seq-PK upsert leaves one row, second write wins),
  schema-agnostic (unmodeled `note_schema` + empty `iscc_id` persist verbatim), hub-scoped lookup.

**Verification:** `mise run check` → GREEN (all 11 packages `ok`; build + vet + test). Per criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -run TestRecordProjections -count=1 ./internal/store` PASS (uncached).
- [x] `go test -run TestSeqsForISCCID -count=1 ./internal/store` PASS (uncached).
- [x] One-to-many: two records, same `IsccID`, seqs 256/257 (distinct schemas) → `SeqsForISCCID`
  returns `[256 257]` ordered (`TestSeqsForISCCIDOneToMany`).
- [x] Idempotency: two `RecordProjections` calls at seq 256 → `COUNT(*) WHERE seq=256 == 1` and the
  `note_schema`/`record_sha256` round-trip to the second write (`TestRecordProjectionsIdempotent`).
- [x] Schema-agnostic: `"iscc-note-future-9.9.9"` and an empty `IsccID` both persist and read back
  verbatim; empty id is looked up via `SeqsForISCCID("")` (`TestRecordProjectionsSchemaAgnostic`).
- [x] Leaf-purity: `go list -f '{{join .Imports}}' ./internal/store | grep -E 'internal/logclient|net/http'`
  is empty.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).

**Next:** Wire `RecordProjections` into the `PollHub` tile-ingestion loop (decode each ingested
entry bundle via `logclient.BundleProjections`, copy `Projection → ProjectionRecord` at the call
site, `RecordProjections`), then use `SeqsForISCCID` to resolve a sampled `iscc_id → leafIndex` and
feed `logclient.VerifyInclusionEvidence` over the `SQLiteFetcher` — the remaining M2 second-half gap
the review flagged.

**Notes:**
- **Deviation from `next.md`'s suggested import list (no functional change):** `next.md` proposed
  imports `context`, `database/sql`, `errors`, `fmt`. The implementation genuinely needs only
  `context` + `fmt`: `RecordProjections` is a plain `ExecContext` loop, and `SeqsForISCCID` uses
  `QueryContext`/`rows.Next` with no `sql.ErrNoRows`/`errors.Is` (its absent case is the natural empty
  result, not a sentinel). I dropped the two unused imports rather than carry dead ones (gofmt/vet
  would reject unused imports anyway). The behavior matches `next.md` exactly; only the import set is
  leaner. This is the same posture as the existing helpers — `tiles.go`/`checkpoints.go` import
  `database/sql`/`errors` because they consult `sql.ErrNoRows`; this file's reader simply doesn't need
  to.
- **`RecordProjections` is an intentional unused-until-wired export seam** (like `RecordTile`/
  `LookupHubKey` were) — no production caller this slice, by design (`## Not In Scope`). `go vet` is
  clean; not dead code.
- Oracle/conformance gate correctly **N/A** for this slice: plain CRUD + BLOB round-trip with
  synthetic in-test records; no signature/RFC-6962/Merkle/did:web/`fsck`-rebuild path. `notecheck`/
  `derive_vkey.py`/`fsck` untouched; the gate re-arms when the inclusion cross-check is wired into
  `PollHub` (the next slice).
- Nothing from `## Not In Scope` touched: no follower/PollHub wiring, no `VerifyInclusionEvidence`
  call, no ISCC-ID interpretation/codec, no `note_schema` validation, no `schema.sql`/`go.mod`/`go.sum`
  change.
