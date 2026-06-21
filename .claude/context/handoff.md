## 2026-06-21 — Pure entry-bundle → iscc_index projection decoder (`BundleProjections`)

**Done:** Added the pure, schema-agnostic `BundleProjections(bundle []byte, baseSeq uint64)
([]Projection, error)` in `internal/logclient/projection.go` — it decodes one tlog-tiles entry bundle
(via `api.EntryBundle.UnmarshalText`, mirroring `LeafHashes`) into one `Projection{Seq, IsccID,
NoteSchema, RecordSHA256}` per leaf (ADR-0008), extracting the committed `iscc_id` and the RAW inner
`note.$schema` discriminator plus the record-content SHA-256, without interpreting anything. It is an
unwired-until-M2 export seam (no production caller yet).

**Files changed:**
- `internal/logclient/projection.go` (new): the pure decoder + `Projection`/`recordEnvelope` types.
  Imports are exactly `crypto/sha256` + `encoding/json` + `fmt` + `tessera/api`. Records the INNER
  `note.$schema` (not the envelope's top-level `$schema`); `RecordSHA256` is `sha256.Sum256(record)`
  (NO `0x00` leaf prefix — that is `LeafHashes`' job). A JSON parse failure is a wrapped error naming
  the absolute seq; empty/unmodeled fields are indexed verbatim, never rejected.
- `internal/logclient/projection_test.go` (new): golden test with an independent uint16-BE framing
  helper (`frameBundle`, a third encode path); pins a declaration + a deletion record at `baseSeq=256`,
  plus empty-bundle, schema-agnostic (empty id + unknown URI), malformed-record, and truncated-bundle
  cases.

**Verification:** `mise run check` → GREEN (all 11 packages `ok`; build + vet + test). Per-criterion:
- [x] `go test -run TestBundleProjections -count=1 ./internal/logclient` — passes (uncached); all 5
  projection tests pass verbosely.
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — exit 0 (file stays WASM-shareable).
- [x] `gofmt -l .` — empty.
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` — exit 0 (no dep/schema change;
  `encoding/json`/`crypto/sha256` are stdlib, `tessera/api` already in the package closure).
- [x] `go list -f '{{join .Imports "\n"}}' ./internal/logclient | grep -E 'internal/store|database/sql'`
  — empty (store stays un-imported by logclient).
- [x] Assertion: two-record bundle (declaration + deletion) at `baseSeq=256` → length-2 with
  `Seq={256,257}` and `NoteSchema={iscc-note-0.8.0.json, iscc-note-delete-0.8.0.json}` verbatim, IDs
  round-trip, `RecordSHA256` matches an independent `sha256.Sum256`.

**Next:** The store writer — `store.RecordProjections`/`RecordProjection` upserting `[]Projection` into
`iscc_index` (with `iscc_id_str` = `IsccID` and `iscc_id` BLOB = whatever raw-id-bytes encoding the
schema column expects; note `schema.sql` has both `iscc_id` BLOB and `iscc_id_str` TEXT — clarify the
BLOB encoding when that slice is defined). Then wire `BundleProjections` into `PollHub`'s
tile-ingestion path and add the `iscc_id → []seq` read query, which together unblock wiring
`VerifyInclusionEvidence` into `PollHub` (resolve a sampled leaf index, pass `SQLiteFetcher.ReadTile`
straight in) — the remaining M2 second-half gap.

**Notes:**
- Oracle gate correctly N/A for this slice: pure JSON + `sha256` content-hash fold, no
  signature/RFC-6962/Merkle/did:web/`fsck` path introduced. `notecheck`/`derive_vkey.py` are untouched
  and re-arm at the `fsck`/inclusion-cross-check wiring slice. No signature/consistency/proof code
  changed.
- The `Projection.IsccID` is the raw `ISCC:`-prefixed string only; I deliberately did NOT decode it to
  the `iscc_id` BLOB column — `next.md` scopes ISCC-ID parsing OUT (schema-agnostic raw fold). The
  store-writer slice owns mapping `IsccID` → the `iscc_id`/`iscc_id_str` columns.
- `RecordSHA256` is `[32]byte` (value, not slice) to make the content hash a fixed-size field and let
  the test use `!=` directly; it maps to the nullable `record_sha256` BLOB column.
- Unwired export seam, same posture as `LeafHashes` / the consistency triggers /
  `VerifyInclusionEvidence` — `go vet` is clean, not dead code.
