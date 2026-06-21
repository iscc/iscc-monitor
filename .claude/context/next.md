# Next Work Package

## Step: Pure entry-bundle → iscc_index projection decoder (`BundleProjections`)

## Goal
Land the pure, schema-agnostic decode that turns one tlog-tiles entry bundle into per-leaf projection
records `{seq, iscc_id, note_schema, record_sha256}` (ADR-0008). This is the foundational prerequisite
for the M2 second half: the `iscc_index` table is wholly unwritten today, and both the store writer and
the `VerifyInclusionEvidence` PollHub wiring need a function that resolves a leaf's `iscc_id` + raw
`note.$schema` from mirrored bundle bytes. Keeping it pure (no store/net) makes it golden-testable in
isolation before any I/O wiring.

## Scope
- **Create**: `internal/logclient/projection.go` — the pure decoder.
- **Create (test, not counted)**: `internal/logclient/projection_test.go`.
- **Reference**:
  - `/workspace/iscc-monitor/cauldron/iscc-hub/specs/iscc-log.md` §5.1 (lines 168–195): a log record is
    the JCS canonicalization of a JSON object with members `$schema`, `iscc_id`, `note`; leaf hash is
    `SHA-256(0x00 || record_bytes)`. §5.1 also names the deletion discriminator
    (`note.$schema = iscc-note-delete-0.8.0`).
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/schema.py` (lines 405–445 `IsccNote`, 352–366
    `IsccNoteDelete`): `note.$schema` is the FieldSchema URI string; `iscc_id` is the `ISCC:`-prefixed
    string. ADR-0008 / `CLAUDE.md` "Projection": store the raw `note.$schema`, never interpret it.
  - `/workspace/iscc-monitor/internal/logclient/leafhasher.go` — the sibling that already iterates
    `api.EntryBundle.Entries` (`[][]byte`, length-prefix-stripped record bytes) and decodes via
    `api.EntryBundle{}.UnmarshalText`. Mirror its decode + import-purity posture exactly.
  - `/workspace/iscc-monitor/internal/store/schema.sql` (lines 99–113): the `iscc_index` columns the
    records map to (`seq`, `iscc_id` BLOB, `iscc_id_str` TEXT, `note_schema` TEXT, `record_sha256` BLOB).
  - `/workspace/iscc-monitor/internal/logclient/leafhasher_test.go` — its `encodeBundle` manual
    uint16-big-endian length framing; reuse that exact helper pattern as an independent encode path in
    the projection test.

## Not In Scope
- The store writer (`store.RecordProjections` / a `RecordProjection` upsert into `iscc_index`) — a
  separate next slice; do NOT add a `store` import or any `database/sql` to this file.
- Wiring into `PollHub` / `ingest.go` — no production caller yet (intentional unused-until-wired export
  seam, like `LeafHashes` / the consistency triggers / `VerifyInclusionEvidence`). `go vet` clean, not
  dead code.
- Wiring `VerifyInclusionEvidence` into `PollHub` (needs the store writer + a leaf-index lookup first).
- An `iscc_id → []seq` query method (the read side belongs with the store-writer slice).
- Resolving/validating the ISCC-ID semantically, parsing the ISCC-CODE, or interpreting deletion
  status — projection is a raw fold, schema-agnostic (ADR-0008); unknown `note.$schema` values are
  indexed verbatim, never rejected.
- Any `go.mod`/`go.sum`/`schema.sql` change (the `iscc_index` DDL already exists; no new dep —
  `encoding/json`/`crypto/sha256` are stdlib and `tessera/api` is already in the package closure).

## Implementation Notes
- Signature (pure): `func BundleProjections(bundle []byte, baseSeq uint64) ([]Projection, error)`
  where `baseSeq` is the bundle's first leaf index (`bundleIndex * 256`, supplied by the future
  caller) so each record's `Seq` is absolute. Define
  `type Projection struct { Seq uint64; IsccID string; NoteSchema string; RecordSHA256 [32]byte }`.
- Decode the bundle exactly as `LeafHashes` does: `eb := &api.EntryBundle{}; eb.UnmarshalText(bundle)`
  (wrap the error `%w`), then iterate `eb.Entries` (each `e []byte` is the canonical JCS record bytes).
  For entry `i`:
  - `RecordSHA256 = sha256.Sum256(e)` — note this is the record-content hash (`crypto/sha256`), NOT the
    RFC-6962 *leaf* hash (`0x00 || r`). It maps to the `record_sha256` column for fsck/debug
    cross-reference; do NOT prepend `0x00`.
  - `json.Unmarshal(e, &env)` into a minimal envelope struct, e.g.
    `struct { IsccID string `json:"iscc_id"`; Note struct { Schema string `json:"$schema"` } `json:"note"` }`.
    Use `encoding/json` (stdlib); JCS bytes are valid UTF-8 JSON so a standard unmarshal extracts the
    fields exactly. A JSON *parse* failure on a record is an error (wrapped `%w` with the absolute seq),
    not a silently-skipped leaf — a malformed record in a hub's bundle is a genuine fault to surface.
  - `Seq = baseSeq + uint64(i)`; `NoteSchema = env.Note.Schema`; `IsccID = env.IsccID`.
- **The recorded schema is the INNER `note.$schema`, not the envelope's top-level `$schema`.** The
  envelope has its own `$schema` (the log-entry schema) and `note.$schema` is the declaration/deletion
  discriminator the projection keys on (ADR-0008 / `CLAUDE.md` Declaration vs Deletion). Confirm against
  iscc-log.md §5.1 + the `IsccNote`/`IsccNoteDelete` shapes before extracting.
- **Schema-agnostic (Correctness rule, ADR-0008):** do NOT validate `IsccID` format or compare
  `NoteSchema` against a known list — store whatever the record carries. An empty `iscc_id` or empty
  `note.$schema` (a record type the monitor does not model) is indexed verbatim, never rejected; only a
  JSON parse failure is an error.
- **Purity (Correctness rule: `proof/verify` is pure / WASM-shareable):** imports must be exactly
  `crypto/sha256` + `encoding/json` + `fmt` + `github.com/transparency-dev/tessera/api`. No
  `net`/`os`/`database/sql`/`internal/store`. The file-level `GOOS=js GOARCH=wasm` build must stay green.
- Start the file with a docstring (project convention) explaining it is the pure projection fold and
  that interpretation is deferred (ADR-0008); note it is an unwired-until-M2 export seam.
- **Test (golden, ground-truth, non-vacuous):** in `package logclient` (white-box) or `logclient_test`,
  build a synthetic entry bundle with the manual uint16-big-endian length framing used by
  `leafhasher_test.go`'s `encodeBundle` (an independent third encode path) over hand-written JCS-style
  record JSON. Pin BOTH a declaration (`note.$schema` =
  `http://purl.org/iscc/schema/iscc-note-0.8.0.json`) and a deletion
  (`http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json`) record so the schema-agnostic claim is
  exercised across the two known types. Assert: `Seq == baseSeq + i`, `IsccID` round-trips,
  `NoteSchema` is the verbatim inner-note URI (declaration vs deletion), and `RecordSHA256 ==
  sha256.Sum256(recordBytes)` recomputed independently. Add an empty-bundle case (zero projections, nil
  err) and a malformed-record case (a non-JSON entry → wrapped error). Non-vacuity: the deletion vs
  declaration assertion proves the decoder reads the *inner* schema, not a constant.

## Verification
- `mise run check` is green (`go build`/`go vet`/`go test ./...` all pass, `gofmt -l .` empty).
- `go test -run TestBundleProjections -count=1 ./internal/logclient` passes (uncached).
- `GOOS=js GOARCH=wasm go build ./internal/logclient` exits 0 (file stays WASM-shareable).
- `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no dep/schema change).
- `go list -f '{{join .Imports "\n"}}' ./internal/logclient | grep -E 'internal/store|database/sql'`
  is empty (store stays un-imported by logclient).
- Assertion: for a two-record bundle (one declaration, one deletion) at `baseSeq == 256`,
  `BundleProjections` returns `[]Projection` of length 2 with `Seq` = `{256, 257}` and `NoteSchema` =
  `{"http://purl.org/iscc/schema/iscc-note-0.8.0.json",
  "http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"}` verbatim.

## Done When
`internal/logclient/projection.go` defines the pure `BundleProjections` decoder with a golden test, all
Verification criteria pass, and no store/network import or schema/dep change is introduced.
