# Next Work Package

## Step: Certificate §6 render — thread `RecordAt.NoteTimestamp` into a `HistoryRow.At` and render `seq N · <at>`

## Advances
Closes the open `normal` issue **"Certificate §6 RECORD HISTORY omits the per-record `· at` timestamp the
mockup shows (store prerequisite LANDED — render-only now)"** (`issues.md`), which is rooted in the
**M-UI certificate Verify criterion**:

> the **realm-wide certificate** … for a known id renders the numbered evidence clauses (subject + position;
> checkpoint `(size, root)`; inclusion proof; signing key; anchor state; **full per-id record history incl. any
> deletion**) …

and the named-region design-parity bar for the certificate §6 region — the mockup
`.claude/design/ISCC Monitor - Certificate.dc.html:68` renders each §6 row as
`seq {{ rec.seq }} · {{ rec.at }}`. The store half landed last iteration
(`store.RecordRow.NoteTimestamp`, read from the `iscc_index.note_timestamp` column); this is the
fully-unblocked, code-closable RENDER half that converts the narrowed §6 `normal` into a closed one. It is
the front-of-queue `**Next:**` from `handoff.md` and `state.md` "Next Milestone" item 1.

## Goal
Surface each §6 record's verbatim `note.timestamp` (RFC-3339) next to its seq, so the certificate's
record-history clause matches its mockup region and shows the per-record `· at` the M-UI criterion calls
for. This closes a `normal` and is the cheapest unblocked progress toward the M-UI exit.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/certificate/handler.go` — add an `At string` field (with docstring) to the `HistoryRow`
    struct (~line 181), and in the §6 loop (`for _, seq := range seqs`, ~line 999) capture
    `row.NoteTimestamp` into the new field, gated on the existing `found` check (mirroring how `schema` is
    captured). This is the ONLY production (non-test/template) file in scope.
  - `internal/certificate/cert.html` — in the `{{range .RecordHistory}}` block (line 476), render the
    timestamp after the seq: `{{.Label}} · seq {{.Seq}}{{if .At}} · {{.At}}{{end}}`. (Template/markup, not
    a non-test/doc Go file — it does not count against the ≤3-file production budget.)
  - `internal/certificate/handler_test.go` — seed a `NoteTimestamp` on the §6 fixture
    (`fixtureStoreHistory`, ~line 1234, the `RecordProjections` slice) and extend
    `TestCertificateRecordHistory` to assert the rendered timestamp is present. (Test file — not counted.)
- **Reference**:
  - `.claude/context/learnings/certificate.md` — the §6 KNOWN-GAP bullet (render-only, store landed) and
    the `html/template` entity-escaping note (`+`→`&#43;` in text nodes — relevant if a non-UTC offset is
    ever rendered).
  - `.claude/context/learnings/store.md` — the `note_timestamp` RFC-3339 verbatim-exception bullet
    (`""` → SQL NULL; read back as "" via `sql.NullString`; never parsed).
  - `internal/store/iscc_index.go` — `RecordRow.NoteTimestamp` (set by `RecordAt`, `""` for NULL) and
    `ProjectionRecord.NoteTimestamp` (written by `RecordProjections`).
  - `.claude/design/ISCC Monitor - Certificate.dc.html:68` — the mockup §6 row (`seq … · {{ rec.at }}`).

## Not In Scope
- Do **not** parse / relativize / reformat the timestamp. Render the verbatim RFC-3339 string the store
  returns — ADR-0008 leaves the format to the renderer, and a verbatim render keeps §6 a pure store-read
  with no `time` parsing and no new failure mode. (The mockup's humanized `2026-02-14 18:40 UTC` display is
  visual polish below the named-region bar, not this step's gate.)
- Do **not** also wire the log-browser record-list `Logged` column (`internal/proofserve`). It can reuse the
  same `RecordRow.NoteTimestamp` but is a separate surface and a separate follow-up — keep this slice to the
  certificate so it stays ≤3 files and closes exactly one `normal`.
- Do **not** add a DB migration for `note_timestamp` (that hazard is its own filed `normal` and a deliberate
  design step — the project's first migration mechanism).
- Do **not** touch the §3/§4/§5 crypto or OTS paths, the WASM verifier, or the comparison-anchor panel.

## Implementation Notes
- The `HistoryRow` struct already has `Seq`, `Label`, `IsDeletion` (handler.go ~line 181). Add:
  `// At is the record's verbatim note.timestamp (RFC-3339) from the iscc_index projection, or "" when the`
  `// note carried none (rendered conditionally). It is displayed verbatim and never parsed (ADR-0008).`
  then `At string`. Update the `HistoryRow` type doc comment to mention the new field.
- In the §6 loop the existing code reads `schema := ""` then `if found { schema = row.NoteSchema }`.
  Capture the timestamp the same way — `at := ""` then `if found { at = row.NoteTimestamp }` (or fold both
  into the one `if found` block) — and pass `At: at` into the appended `HistoryRow{...}`. A `RecordAt` miss
  leaves `At` empty — same honest-gap discipline as the schema, no new error path, no new `RecordAt` call.
- In `cert.html`, the current row is `<div class="clause-mono">{{.Label}} · seq {{.Seq}}</div>`. Make the
  timestamp conditional so the declaration-only / no-timestamp fixtures still render a clean row:
  `{{.Label}} · seq {{.Seq}}{{if .At}} · {{.At}}{{end}}`.
- Correctness rule (`learnings.md` index): the `iscc_id → seq` projection is schema-agnostic and verbatim —
  `note.timestamp` is carried, never interpreted. Keep that posture: no parse, no validation, no gate on it.
- `html/template` entity-escaping (certificate.md): an RFC-3339 string contains `:` (and `+` for non-UTC
  offsets); in a text node `html/template` escapes `+`→`&#43;`. Prefer a `Z`-suffixed UTC fixture timestamp
  (e.g. `"2026-02-14T18:40:00Z"`) so the test assertion needs no `html.UnescapeString`.
- For the test: in `fixtureStoreHistory`, add `NoteTimestamp: "2026-02-14T18:40:00Z"` to the declaration
  `ProjectionRecord` and leave the deletion's empty (also exercises the absent→"" path). In
  `TestCertificateRecordHistory`, add the seeded timestamp string to the `want` slice. Non-vacuity:
  dropping `At: at` from the appended `HistoryRow` (or the `{{if .At}}…{{end}}` from the template) must make
  the new assertion FAIL while `TestCertificateRecordHistoryDeclarationOnly` still passes.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`; `gofmt -l .` empty).
- `go test -count=1 -run TestCertificateRecordHistory ./internal/certificate` passes — the §6 declaration
  row renders `Declaration · seq 24815 · 2026-02-14T18:40:00Z` and the deletion row still renders.
- `go test -count=1 -run TestCertificateRecordHistoryDeclarationOnly ./internal/certificate` passes — a
  record with no timestamp renders `… · seq 24815` with no trailing `· ` artifact.
- Mutation (advance self-check, revert after, report in handoff): removing `At: at` from the appended
  `HistoryRow` makes `TestCertificateRecordHistory` FAIL on the missing timestamp marker.

## Done When
`mise run check` is green and `TestCertificateRecordHistory` asserts the verbatim RFC-3339 timestamp now
renders next to the seq in the §6 row (with the declaration-only / absent-timestamp case still clean),
closing the §6 render-only `normal`.
