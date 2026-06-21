## 2026-06-21 — Certificate §6 RECORD HISTORY — render the per-id record list (declaration + any deletion)

**Done:** The §6 RECORD HISTORY clause now renders the full one-to-many list of accepted-tree seqs a
hub indexed under the subject id — the declaration plus any later deletion — each labelled by its
verbatim `note.$schema` kind (declaration / deletion / unknown), with the "a deletion is a new record"
note shown when any row is a deletion. It is a pure store read (one `RecordAt` per seq from the `seqs`
already in hand), capped to the accepted tree (`seq < hub.LastSize`, coverage honesty), rendering
unconditionally for a certifiable id — no crypto/cache gate. `HasClause5` (Bitcoin anchor) stays
false, deferred until the OTS store seam exists.

**Files changed:**
- `internal/certificate/handler.go`: added local `schemaDeclaration`/`schemaDeletion` constants +
  kind labels + a pure `recordKind(noteSchema) (label, isDeletion)` helper (mirrors
  `proofserve.recordKind`, the constants unexported in another package); added a `HistoryRow` type
  and `RecordHistory []HistoryRow` + `HasDeletion bool` view-model fields; added the §6 build block
  in `buildData` after §4 (seq-capped to `< LastSize`, `RecordAt` miss is an honest unknown-label row,
  only a real DB fault 500s); updated the file/`certData`/`buildData` docstrings + the `HasClause2..6`
  doc comment to describe §6.
- `internal/certificate/cert.html`: filled the empty `{{if .HasClause6}}` §6 `clause-value` with a
  `{{range .RecordHistory}}` of `clause-mono` `{{.Label}} · seq {{.Seq}}` rows and a conditional
  `{{if .HasDeletion}}` `clause-note` deletion note (reuses existing classes, no new CSS, no CDN URL).
- `internal/certificate/handler_test.go`: added `fixtureStoreHistory` (declaration + later deletion
  under one id, both `< LastSize`, FULL wire-URI schemas), `TestCertificateRecordHistory` (asserts
  both rows + the deletion note + §1 position is the earliest seq), and
  `TestCertificateRecordHistoryDeclarationOnly` (single record → §6 renders, no deletion note).

**Verification:** `mise run check` → green, all 21 packages `ok` (build + vet + test).
- [x] `go test -count=1 -run TestCertificate ./internal/certificate` — PASS (all §1-§4 + the two new
  §6 tests).
- [x] `go test -count=1 -v -run TestCertificateRecordHistory ./internal/certificate` — PASS: asserts
  `Declaration · seq 24815`, `Deletion · seq 31002`, the deletion note, and the §1 position.
- [x] Mutation (non-vacuity, reproduced + reverted): `data.HasClause6 = true` → `= false` makes
  `TestCertificateRecordHistory` FAIL (no §6 marker / Deletion row / note); reverted, tree clean,
  tests green.
- [x] Oracle gate unbroken: `go test -count=1 ./internal/logclient ./internal/follower ./cmd/notecheck`
  all `ok` (no crypto path touched).
- [x] WASM/purity: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exit 0.
- [x] `gofmt -l .` empty; `git diff --stat HEAD -- go.mod go.sum` empty (no new dependency).
- [x] Scope discipline: exactly 2 non-test source files (`handler.go`, `cert.html`) + the test file;
  no §5 / proof-bundle / §4-DID / registry work.

**Next:** §5 BITCOIN ANCHOR is the last remaining clause but is BLOCKED on the OTS/anchor store seam,
which does not exist (no `anchor`/`ots` store method — only an `ots` table name in a store test
fixture). Doing §5 honestly needs that seam built first (a separate, larger step) — defer it. The
data-grounded clauses (§1-§4, §6) are now complete; the next M-UI certificate work is the
**downloadable proof-bundle assembler** `{checkpoint, inclusion/consistency proof, record bytes, hub
key, ots?}`, which re-engages the oracle/conformance gate (reuses the §3 build+verify crypto path and
the §4 key read-path). Either lands next; the proof-bundle is the bigger criterion item.

**Notes:**
- The kind mapping is duplicated from `proofserve.recordKind` because the proofserve constants are
  unexported in another package (next.md explicitly directed defining them locally to stay
  import-clean). Two pure copies of a 6-line switch — minor DRY debt; promoting a shared
  schema-kind helper to e.g. `internal/index` or a tiny shared leaf is a clean future refactor if a
  third caller appears, but not warranted now (YAGNI).
- §6 lists an unknown/empty schema verbatim with the "Unknown record type" label and never errors
  (ADR-0008 schema-agnostic; correctness-rule "index by seq, never gate on schema"). The existing
  `fixtureStore` seeds the bare `"iscc-note-0.8.0.json"` short form (NOT the wire URI), so
  `TestCertificateRecordHistoryDeclarationOnly` exercises that unknown-label path for free while
  proving the seq still lists. The new `fixtureStoreHistory` seeds the FULL wire URIs production
  actually stores, so the declaration/deletion labels resolve correctly.
- The §4 `did:web:` + raw-domain `host:port` mis-render (filed `normal`) is untouched — handler.go was
  modified here but the §4 DID-building path was deliberately left alone (no encoder added) per the
  Not-In-Scope note. Still filed for a step that touches the DID path or adds a `%3A` encoder.
- No fixture seeds a `RecordAt` DB-fault row, so the §6 500-on-real-fault branch is
  untested-but-trivial (mirrors §2/§3/§4's identical buffer-then-200 pattern). Acceptable; flag only
  if it becomes load-bearing.
