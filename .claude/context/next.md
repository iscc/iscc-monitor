# Next Work Package

## Step: Carry the per-record `note.timestamp` through the iscc_index projection (store layer of the §6 `· at`)

## Advances
The `normal` issue **"Certificate §6 RECORD HISTORY omits the per-record `· at` timestamp the mockup
shows"** (issues.md), rooted in the **M-UI certificate Verify criterion** — target.md: the certificate
"renders the numbered evidence clauses (… full per-id record history incl. any deletion)". The mockup
(`.claude/design/ISCC Monitor - Certificate.dc.html:68`) renders each §6 row as `seq N · <at>`, but the
landed §6 renders only `label · seq N` because the `iscc_index` projection carries no per-record time
column. This step closes that issue's **store + projection prerequisite** (the larger half it names: "a
store schema column on the `iscc_index` projection + a follower-ingest write"); the §6 render itself is a
follow-up sub-step (see Not In Scope). This is the front-of-queue review recommendation: "Suggest
define-next picks the §6 timestamp … (store-scoped, self-contained) over the signature half (which wants
a STOP/design pass first)." The remaining `normal`s are design-first (WASM signature) or human-blocked
(Pages), so this self-contained store-schema slice is the right milestone-advancing pick.

## Goal
Thread the optional per-record `note.timestamp` (RFC-3339 string the log envelope carries for both
declarations and deletions) end-to-end through the schema-agnostic `iscc_index` projection — schema
column, the pure projection fold, the store write/read structs, and the follower's copy site — so the
verbatim time is persisted and readable per leaf. This is the foundation the certificate §6 `· at` and
(later) the log-browser record list can render, with the render itself deferred to keep this a clean,
≤3-file, store-tested skeleton.

## Scope
- **Modify** (3 non-test/doc production Go files):
  - `internal/logclient/projection.go` — add `Timestamp string` to `Projection`; add
    `Timestamp string \`json:"timestamp"\`` to the inner `note` of `recordEnvelope`; set
    `Timestamp: env.Note.Timestamp` in the fold. (Stays pure — no `time` import; the value is read
    verbatim, never parsed, per ADR-0008.)
  - `internal/store/iscc_index.go` — add `NoteTimestamp string` to `ProjectionRecord` and `RecordRow`;
    write it in `RecordProjections` (bind via `nullStringOrNil` so empty → SQL NULL, mirroring
    `iscc_id_str`/`note_schema`); read it in `RecordAt` and `ListRecords` through `sql.NullString`
    (NULL → "").
  - `internal/follower/ingest.go` — copy the new field at the `store.ProjectionRecord{…}` literal
    (`ingest.go:109`): `NoteTimestamp: p.Timestamp`.
- **Modify** (schema DDL — data file, not counted against the 3-file budget):
  - `internal/store/schema.sql` — add a nullable `note_timestamp TEXT` column to the `iscc_index`
    CREATE TABLE (after `note_schema`); keep the time-convention header comment honest (this column is
    the one RFC-3339 *string* exception to the unix-seconds convention — it is stored verbatim per
    ADR-0008, not parsed).
- **Modify** (docstrings already inside the files above — keep them evergreen): update the
  `ProjectionRecord`/`RecordRow`/`Projection` doc comments to list the new field; update
  `iscc_index.go`'s `SELECT`/`INSERT` column-list comments. No external README mentions this column.
- **Reference**:
  - `.claude/context/learnings/store.md` (iscc_index writer/reader section: `RecordProjections` per-row
    `ON CONFLICT(seq) DO UPDATE`, `sql.NullString` NULL→"" reads, `nullStringOrNil` convention).
  - `.claude/context/learnings/logclient.md` (Entry-bundle projection fold section: `recordEnvelope`
    reads inner `note`; file-level WASM purity — imports must stay `crypto/sha256`+`encoding/json`+`fmt`+
    `tessera/api`, no `time`).
  - `.claude/context/learnings/certificate.md` (§6 RECORD HISTORY bullet — the KNOWN GAP this unblocks).
  - `cauldron/iscc-hub/iscc_hub/schema.py` (the wire contract: `IsccNote.timestamp` and
    `IsccNoteDelete.timestamp` are both `Timestamp | None = None` — an optional RFC-3339 string,
    "IsccNote creation time in UTC", around lines 225-229 / 443 / 364).
  - `internal/logclient/projection_test.go` (golden frame helper `frameBundle` + decl/del fixtures to
    extend with a `note.timestamp`).
  - `internal/store/iscc_index_test.go` (existing `RecordProjections`/`RecordAt`/`ListRecords` round-trip
    tests to extend).

## Not In Scope
- **The certificate §6 render** (`internal/certificate/handler.go` `buildData` §6 loop + `cert.html`
  `{{.RecordHistory}}` rows). Wiring `RecordAt`'s new `NoteTimestamp` into a `HistoryRow.At` and
  rendering `seq N · <at>` is the explicit FOLLOW-UP step — do not touch certificate this iteration
  (keeps this ≤3 production files and store-testable in isolation).
- **The log-browser record list `Logged` column** (`internal/proofserve` record list / single-record
  page). The mockup's `Logged` column also wants this time, but that render is a separate later step.
- **Parsing/normalizing the RFC-3339 string** to a `time.Time` or unix-seconds. Store it verbatim as
  TEXT (ADR-0008 "store the raw value, never interpret"); keeping `projection.go` `time`-free preserves
  its WASM purity. Formatting/relativizing belongs to the render step.
- **A migration / `ALTER TABLE` for existing on-disk DBs.** There is no schema-versioning framework;
  the column is added to `CREATE TABLE IF NOT EXISTS` and only fresh DBs get it (consistent with how
  every prior column landed — dev DBs are ephemeral). Do not add a migration path.

## Implementation Notes
- **The wire field is `note.timestamp`, not the ISCC-ID timestamp.** A deletion carries the EXISTING
  `iscc_id`, so the id-embedded `body>>12` time (`internal/index.Decode`) is the *declaration's* time
  for BOTH rows and cannot distinguish the deletion's logged time. The mockup's two distinct §6 times
  (declaration vs deletion) come from each record's own `note.timestamp` — the "creation/signing time"
  per `schema.py`. So the value MUST be read from the record envelope's `note` object, which the
  projection fold already decodes (`recordEnvelope.Note`).
- **It is OPTIONAL (`Timestamp | None`).** Treat absence as the empty string in `Projection`/
  `ProjectionRecord` and write SQL NULL via `nullStringOrNil` (the `iscc_id_str`/`note_schema`/key-cache
  idiom), read back as "" via `sql.NullString` — exactly mirroring the existing nullable TEXT columns in
  this file. A record with no `note.timestamp` must still index cleanly (schema-agnostic, ADR-0008).
- **Keep `projection.go` pure.** Do NOT add `time` (or any new import); read `env.Note.Timestamp` as a
  raw string. Verify the WASM build still passes: `GOOS=js GOARCH=wasm go build ./internal/logclient`
  exits 0 (logclient.md notes this stays green; the load-bearing purity check is the file-level import
  set, which gains nothing).
- **`RecordProjections` upsert must include the new column in BOTH the INSERT column list AND the
  `ON CONFLICT(seq) DO UPDATE SET` list** (store.md: a `DO NOTHING`/dropped-field mutation FAILS the
  idempotency test on stale fields). The `RecordRow` readers (`RecordAt`, `ListRecords`) add the column
  to their `SELECT` lists and scan it through `sql.NullString`. Note `iscc_id_str` is currently bound as
  a plain TEXT (not via `nullStringOrNil`) in `RecordProjections`; `note_timestamp` SHOULD use
  `nullStringOrNil` so an absent timestamp is a true NULL, distinct from a present empty string — match
  the key-cache `nullStringOrNil` precedent, and confirm the helper exists in `internal/store`
  (`checkpoints.go` uses it).
- **Extend the golden, don't replace it.** `projection_test.go`'s declaration record gains a
  `note.timestamp` and the deletion record either a *different* timestamp or omits it (to pin the
  optional/NULL path) — assert the fold reads the inner per-record value, mirroring how the existing
  test pins inner `$schema`. For the store seam, extend `iscc_index_test.go` to round-trip a row WITH a
  timestamp and a row WITHOUT (NULL → "") through `RecordProjections` + `RecordAt`.
- **Correctness rules (learnings.md):** schema-agnostic projection (ADR-0008) — index verbatim, never
  interpret, optional fields tolerated; store stays a leaf (no new imports beyond stdlib in
  `iscc_index.go`); single-writer discipline unchanged (plain per-row `ExecContext`); follower →
  {logclient, store} dependency direction unchanged (the follower copies `Projection` field-by-field,
  store never imports logclient).
- **Oracle/conformance gate is N/A** for this slice — it touches no signature / RFC-6962 / Merkle /
  did:web / proof / fsck path (pure JSON-fold field + plain TEXT-column round-trip). Confirm by name-only
  diff over the trust-root globs (empty), as prior store-slice reviews did.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestBundleProjections ./internal/logclient` passes and asserts the fold reads
  `note.timestamp` per record (declaration's value present; the no-timestamp record yields "").
- `go test -count=1 -run 'TestRecordAt|TestRecordProjections|TestListRecords' ./internal/store` passes,
  round-tripping a `NoteTimestamp` value (present and absent→"") through write then read.
- Mutation (advance self-checks, report in handoff): dropping `note_timestamp` from
  `RecordProjections`' `DO UPDATE SET` makes the store round-trip test FAIL on the upsert path; setting
  `Projection.Timestamp` to a constant instead of `env.Note.Timestamp` makes the projection golden FAIL.
- `GOOS=js GOARCH=wasm go build ./internal/logclient` exits 0 (projection.go stays WASM-pure, no `time`).

## Done When
`mise run check` is green, the projection fold and the store round-trip persist and return the verbatim
optional `note.timestamp` per leaf (present and NULL→"") under the named tests, and `projection.go`
gained no non-stdlib/`time` import — leaving the certificate §6 render as the next sub-step.
