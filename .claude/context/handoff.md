## 2026-06-22 — Carry the per-record `note.timestamp` through the iscc_index projection (store layer of the §6 `· at`)

**Done:** Threaded the optional per-record `note.timestamp` (verbatim RFC-3339 string, both note types)
end-to-end through the schema-agnostic `iscc_index` projection — the pure fold, the schema column, the
store write/read structs, and the follower's copy site — so the value is persisted per leaf (present →
verbatim TEXT, absent → SQL NULL → ""). The certificate §6 `· at` render is left as the next sub-step
(explicitly out of scope), keeping this a clean ≤3-production-file, store-tested skeleton.

**Files changed:**
- `internal/logclient/projection.go`: added `Timestamp string` to `Projection`, `Timestamp` (json
  `timestamp`) to `recordEnvelope.Note`, and `Timestamp: env.Note.Timestamp` in the fold; docstrings
  updated. File-level imports UNCHANGED (`crypto/sha256`+`encoding/json`+`fmt`+`tessera/api`, no `time`)
  — WASM purity preserved.
- `internal/store/iscc_index.go`: added `NoteTimestamp string` to `ProjectionRecord` and `RecordRow`;
  bound it via `nullStringOrNil` in `RecordProjections`' INSERT col-list AND `DO UPDATE SET`; read it
  through `sql.NullString` (NULL→"") in `RecordAt` and `ListRecords`; docstrings/column comments updated.
- `internal/follower/ingest.go`: copied `NoteTimestamp: p.Timestamp` at the `store.ProjectionRecord{…}`
  literal in `projectEntryBundle`.
- `internal/store/schema.sql` (data file, not in the 3-file budget): added nullable `note_timestamp TEXT`
  to `iscc_index` (after `note_schema`); time-convention header now names this column as the one
  verbatim-RFC-3339-TEXT exception to the unix-seconds convention (ADR-0008).
- `internal/logclient/projection_test.go` (test): `TestBundleProjections` declaration now carries
  `note.timestamp`, deletion omits it; asserts the fold reads the inner per-record value (present;
  absent→"").
- `internal/store/iscc_index_test.go` (test): `TestRecordProjectionsRoundTrip` round-trips a present
  timestamp (true NOT-NULL column + `RecordAt`); new `TestRecordProjectionsNoTimestamp` proves absent→
  SQL NULL→"" via `RecordAt`+`ListRecords`; `TestRecordProjectionsIdempotent` now pins the upsert
  `DO UPDATE` of `note_timestamp` (second write wins).

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 28 packages ok),
`gofmt -l .` empty.
- `go test -count=1 -run TestBundleProjections ./internal/logclient` → PASS (asserts the fold reads
  `note.timestamp` per record; no-timestamp record yields "").
- `go test -count=1 -run 'TestRecordAt|TestRecordProjections|TestListRecords' ./internal/store` → PASS
  (round-trips `NoteTimestamp` present and absent→"" through write then read).
- `GOOS=js GOARCH=wasm go build ./internal/logclient` → exit 0 (projection.go stays WASM-pure, no `time`;
  file-level import set byte-unchanged).
- Mutation (self-checked, both reverted to byte-clean): (1) dropping `note_timestamp` from
  `RecordProjections`' `DO UPDATE SET` → `TestRecordProjectionsIdempotent` FAILS (stale first value);
  (2) `Projection.Timestamp = "CONSTANT"` instead of `env.Note.Timestamp` → `TestBundleProjections`
  FAILS on both the present and the absent (→"") leaf.
- Oracle/conformance gate N/A — confirmed by name-only diff over the trust-root globs (`internal/proof/`,
  `logclient/verify`, `didweb`, fork/shrink/equivocation/consistency, `derive_vkey`) → empty. This slice
  is a pure JSON-fold field + plain nullable-TEXT round-trip; it touches no signature/RFC-6962/Merkle/
  did:web/proof/fsck path. `projection.go` lives under `internal/logclient` but is the schema-agnostic
  fold, not a verify/Merkle path.

**Next:** The certificate §6 RECORD HISTORY render is now unblocked — the explicit FOLLOW-UP. Wire
`RecordAt`'s new `NoteTimestamp` into a `HistoryRow.At` field in `internal/certificate/handler.go`'s §6
loop and render `seq N · <at>` in `cert.html`'s `{{.RecordHistory}}` rows, matching the mockup
(`.dc.html:68`). That step should also pick the format/relativize policy for the verbatim string (deferred
here per ADR-0008 "store the raw value, never interpret"). The log-browser record-list `Logged` column
(`internal/proofserve`) is a separate later render that can reuse the same `RecordRow.NoteTimestamp`.

**Notes:**
- Per the §6 implementation note, the value MUST come from each record's own `note.timestamp` (not the
  ISCC-ID-embedded `body>>12` time), because a deletion carries the EXISTING declaration's `iscc_id` and
  thus the SAME id-embedded time as the declaration — only `note.timestamp` distinguishes the two §6 rows
  the mockup shows. The fold reads it from `recordEnvelope.Note`, which already decodes the inner note.
- `nullStringOrNil` chosen over plain TEXT binding (the way `iscc_id_str` is bound) so an absent timestamp
  is a true NULL distinct from a present empty string — matches the key-cache precedent and the
  `note_schema`/`iscc_id_str` nullable-column reader idiom; pinned by `TestRecordProjectionsNoTimestamp`
  reading the raw column as `sql.NullString` (Valid==false).
- No migration / `ALTER TABLE` (no schema-versioning framework here; the column is added to
  `CREATE TABLE IF NOT EXISTS`, so only fresh DBs get it — consistent with every prior column; dev DBs
  are ephemeral). Out-of-scope items (§6 render, proofserve `Logged` column, RFC-3339 parsing) all left
  untouched.
- No backward-incompatible API change and no design deviation — additive struct fields + a new nullable
  column; no HUMAN REVIEW needed.
