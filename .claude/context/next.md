# Next Work Package

## Step: `iscc_index` store writer + `iscc_id → []seq` read-back

## Goal
Persist the schema-agnostic projection records (`BundleProjections` output) into the
`iscc_index` table and add the `iscc_id → []seq` one-to-many read query. This lands the
store half of the M2 `iscc_index` projection (the pure decoder landed last slice) and
directly unblocks the next slice — resolving a sampled `iscc_id → leafIndex` to wire
`VerifyInclusionEvidence` into `PollHub`.

## Scope
- **Create**: `internal/store/iscc_index.go` — `ProjectionRecord` struct + `RecordProjections`
  writer + `SeqsForISCCID` reader.
- **Create (test, not counted)**: `internal/store/iscc_index_test.go` — table-driven round-trip +
  idempotency + one-to-many tests.
- **Modify**: (none — `schema.sql` already defines the `iscc_index` table and its index)
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/tiles.go` — the upsert/round-trip CRUD pattern to mirror
    (`RecordTile` composite-PK upsert at lines 37-51; `ReadTileBlob` "absent is not an error" at 76-89).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — the store-owned-plain-struct convention
    (`CheckpointRecord`/`HubKey` at lines 26-49, 62-70; `RecordHubKey` at 324; `LookupHubKey` at 365)
    and the `unixOrNil`/`nullStringOrNil` helpers.
  - `/workspace/iscc-monitor/internal/store/schema.sql` lines 99-113 — the `iscc_index` columns
    (`hub_id`, `seq` PRIMARY KEY, `iscc_id` BLOB, `iscc_id_str` TEXT, `note_schema` TEXT,
    `record_sha256` BLOB) and the `iscc_index_by_iscc_id` index on the `iscc_id` BLOB column.
  - `/workspace/iscc-monitor/internal/logclient/projection.go` — the `Projection` struct whose fields
    (`Seq`, `IsccID`, `NoteSchema`, `RecordSHA256`) the follower will later copy into
    `ProjectionRecord` at the call site (do NOT import logclient from store).
  - `/workspace/iscc-monitor/internal/store/sqlite_test.go` lines 1-40 — the `t.TempDir()` open +
    observable-state test setup to reuse; also `internal/store/tiles_test.go` for register-hub-then-CRUD.

## Not In Scope
- **Wiring into `PollHub`/`follower`** — no production caller this slice. `RecordProjections` is an
  intentional unused-until-wired export seam (like `RecordTile`/`LookupHubKey` were). Do not touch
  `internal/follower`.
- **Wiring `VerifyInclusionEvidence`** — that is the *next* slice; `SeqsForISCCID` exists to unblock
  it, not to call it now.
- **ISCC-ID interpretation** — do NOT decode the `ISCC:`-prefixed string into a structured ISCC-ID,
  do NOT import iscc-lib, do NOT validate maintype/subtype. ADR-0008: the index interprets nothing.
  Store the raw string and its UTF-8 bytes only.
- **`note_schema` validation** — store the raw `note.$schema` verbatim; never check it against a
  known-schema list.
- **`go.mod`/`go.sum`/`schema.sql` changes** — the `iscc_index` DDL already exists; no new dep
  (the new file needs only stdlib already in the store closure).

## Implementation Notes
- **Store stays a leaf (load-bearing).** `internal/store` must NOT import `internal/logclient` (it
  pulls `net/http` via `didresolve.go`). Mirror `checkpoints.go`'s convention: define a store-owned
  plain `ProjectionRecord` struct; the follower will copy `logclient.Projection` → `ProjectionRecord`
  at the call site in the later wiring slice. Verify with the package-imports grep in Verification.
  Imports for the new file: `context`, `database/sql`, `errors`, `fmt` only (no `crypto/sha256` —
  the hash arrives pre-computed in the record).
- **`ProjectionRecord` fields** map 1:1 to the columns: `HubID int64`, `Seq uint64`, `IsccID string`
  (the raw `ISCC:`-prefixed string from `Projection.IsccID`), `NoteSchema string`, `RecordSHA256
  [32]byte`. Keep it a plain value struct like `CheckpointRecord`/`HubKey`.
- **`iscc_id` BLOB vs `iscc_id_str` TEXT decision (the review's open question):** store the raw
  string into **both** — `iscc_id_str` gets the verbatim `ISCC:`-prefixed string, and `iscc_id` BLOB
  gets its UTF-8 bytes (`[]byte(r.IsccID)`). This keeps the existing `iscc_index_by_iscc_id` index
  (on the BLOB column) usable for `iscc_id → []seq` lookups WITHOUT any ISCC-ID codec, honoring
  ADR-0008's "interpret nothing". An empty `IsccID` writes an empty BLOB + empty string (not NULL) —
  the decoder already indexes empty ids verbatim (ADR-0008), so do not special-case it. (A real
  base32 BLOB decode is a deliberate future option if a binary-keyed lookup is ever needed; it is out
  of scope and explicitly NOT required now.)
- **`RecordProjections(ctx, recs []ProjectionRecord) error`** writes the batch. Because `seq` is the
  PRIMARY KEY, re-ingesting an already-mirrored bundle MUST be idempotent — use `INSERT INTO
  iscc_index (hub_id, seq, iscc_id, iscc_id_str, note_schema, record_sha256) VALUES (?,?,?,?,?,?)
  ON CONFLICT(seq) DO UPDATE SET hub_id=excluded.hub_id, iscc_id=excluded.iscc_id,
  iscc_id_str=excluded.iscc_id_str, note_schema=excluded.note_schema,
  record_sha256=excluded.record_sha256` per row, mirroring `RecordTile`'s composite-PK upsert. Pass
  `int64(r.Seq)`, `[]byte(r.IsccID)`, `r.IsccID`, `r.NoteSchema`, `r.RecordSHA256[:]` as the SQLite
  bindings. Wrap errors `%w` with a `store.RecordProjections: seq %d: %w` prefix. An empty slice is a
  no-op returning nil. Single-writer discipline holds — plain `db.ExecContext` calls on the capped
  pool (ADR-0005/0007); a per-row loop is fine (no explicit transaction needed; match `RecordTile`).
- **`SeqsForISCCID(ctx, hubID int64, isccID string) ([]uint64, error)`** is the one-to-many reader:
  `SELECT seq FROM iscc_index WHERE hub_id = ? AND iscc_id = ? ORDER BY seq` (query the BLOB column
  with `[]byte(isccID)` so it uses the index; `ORDER BY seq` makes the result deterministic). Return
  a nil/empty slice + nil error for no matches — "absent is not an error", mirroring `LookupHubKey`.
  Scan each row into an `int64` then convert to `uint64` (the column is INTEGER, matching how the
  codebase binds `int64(seq)` on the write side). `defer rows.Close()` and check `rows.Err()` after
  the loop. Wrap any query/scan fault `%w` with a `store.SeqsForISCCID: hub %d: %w` prefix.
- **Correctness rule (learnings.md):** "`iscc_id → seq` is one-to-many, schema-agnostic (ADR-0008) —
  index by `seq`; store the raw `note.$schema`; lookups return a list; unknown schemas are indexed +
  proof-able but never interpreted." The test MUST exercise the list-return (one `iscc_id` with a
  declaration + a deletion at two different seqs → `SeqsForISCCID` returns both).
- **FK note:** `iscc_index.hub_id` REFERENCES `hubs(hub_id)` and `foreign_keys=ON`, so the test must
  register a hub first (reuse the `RegisterHub`/insert helper that `tiles_test.go` already uses) before
  inserting projection rows — a dangling `hub_id` will fail the FK constraint.
- Start the file with a docstring (project convention) explaining it is the `iscc_index` writer +
  one-to-many reader, that it stores the raw `iscc_id`/`note.$schema` without interpretation
  (ADR-0008), and that it is an unwired-until-M2 export seam.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestRecordProjections -count=1 ./internal/store` passes (uncached).
- `go test -run TestSeqsForISCCID -count=1 ./internal/store` passes (uncached).
- One-to-many assertion: write two `ProjectionRecord`s with the SAME `IsccID` but different `Seq`
  (e.g. a declaration at seq 256 + a deletion at seq 257, distinct `NoteSchema`) →
  `SeqsForISCCID(hubID, id)` returns `[]uint64{256, 257}` (both, ordered).
- Idempotency assertion: calling `RecordProjections` twice with the same `seq` leaves exactly one
  row for that `seq` (re-query `SELECT COUNT(*) … WHERE seq = ?` == 1; an `UPDATE`d `note_schema`
  field round-trips to the second write's value).
- Schema-agnostic assertion: a record with an unmodeled `NoteSchema` (e.g. `"iscc-note-future-9.9.9"`)
  and a record with an empty `IsccID` both persist and read back verbatim (no rejection).
- Leaf-purity assertion: `go list -f '{{join .Imports "\n"}}' ./internal/store | grep -E
  'internal/logclient|net/http'` is empty.
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).

## Done When
`internal/store/iscc_index.go` exposes `RecordProjections` (idempotent upsert keyed on `seq`) and
`SeqsForISCCID` (ordered one-to-many `iscc_id → []seq`), all Verification checks pass, store stays a
leaf (no logclient/net/http import), and `schema.sql`/`go.mod`/`go.sum` are byte-unchanged.
