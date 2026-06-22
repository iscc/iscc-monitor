## 2026-06-22 — Certificate §6 render: thread `RecordAt.NoteTimestamp` into `HistoryRow.At` and render `seq N · <at>`

**Done:** Wired the per-record verbatim `note.timestamp` (RFC-3339) into the certificate §6 RECORD
HISTORY clause — added an `At` field to `HistoryRow`, captured `row.NoteTimestamp` in the §6 loop's
existing `if found` block, and render it conditionally after the seq (`seq N · <at>`) in `cert.html`,
matching the mockup region. This closes the §6 render-only `normal`; the store half landed last
iteration. Verbatim render only — no parse, no reformat (ADR-0008).

**Files changed:**
- `internal/certificate/handler.go`: added `At string` to `HistoryRow` (with docstring; type doc
  updated); in the §6 loop captured `at := ""` then `at = row.NoteTimestamp` under the existing
  `found` gate, and passed `At: at` into the appended `HistoryRow`.
- `internal/certificate/cert.html`: §6 row now renders `{{.Label}} · seq {{.Seq}}{{if .At}} · {{.At}}{{end}}`
  (conditional so timestamp-less rows render no trailing `· ` artifact).
- `internal/certificate/handler_test.go`: added `historyDeclTimestamp = "2026-02-14T18:40:00Z"`
  (Z-suffixed UTC so no `html.UnescapeString` needed); seeded it on `fixtureStoreHistory`'s
  declaration `ProjectionRecord` (deletion left empty → absent→"" path); extended
  `TestCertificateRecordHistory` to assert the full row `Declaration · seq 24815 · 2026-02-14T18:40:00Z`
  renders and that the deletion row carries no trailing `· `.

**Verification:** `mise run check` → green (build + vet + test, all 28 packages ok; certificate
re-ran uncached PASS). `gofmt -l .` → empty.
- [x] `go test -count=1 -run TestCertificateRecordHistory ./internal/certificate` — PASS (the §6
      declaration row renders `Declaration · seq 24815 · 2026-02-14T18:40:00Z`; deletion row renders).
- [x] `go test -count=1 -run TestCertificateRecordHistoryDeclarationOnly ./internal/certificate` —
      PASS (a record with no timestamp renders `… · seq 24815` with no trailing `· ` artifact).
- [x] Mutation (self-check, reverted byte-clean): dropping the `{{if .At}}…{{end}}` from `cert.html`
      makes `TestCertificateRecordHistory` FAIL on the exact marker
      `body missing §6 marker "Declaration · seq 24815 · 2026-02-14T18:40:00Z"` while
      `TestCertificateRecordHistoryDeclarationOnly` still PASSES. (See Notes for why I mutated the
      template rather than `At: at` — the latter is a compile error, not an assertion failure.)
- [x] Oracle/conformance gate — N/A. Pure store-read into an `html/template` text node; touches no
      signature/RFC-6962/Merkle/did:web/proof/fsck path (handler.go diff confirms: just the struct
      field + the `if found` capture + the appended literal).

**Next:** The mockup §6 also humanizes the display to `2026-02-14 18:40 UTC` (visual polish below the
named-region bar, deferred per ADR-0008) — could be picked up later if design-parity wants it, but it
introduces `time` parsing and a format policy, so it is a deliberate step, not a free follow-up. The
sibling unblocked pick is wiring the same `RecordRow.NoteTimestamp` into the log-browser record-list
`Logged` column (`internal/proofserve`) — explicitly out of scope here, a separate surface and a clean
self-contained slice. The no-migration `normal` and the WASM-signature `normal` still want a deliberate
design pass, not a code-only slice.

**Notes:**
- Mutation nuance: `next.md` specifies the self-check as "removing `At: at` from the appended
  `HistoryRow`". Doing exactly that leaves `at := ""` unused, so the package fails to COMPILE
  (`declared and not used: at`) rather than producing a clean test assertion failure — it proves the
  field is load-bearing but does not isolate the assertion. To prove the assertion itself is
  non-vacuous I instead dropped the `{{if .At}}…{{end}}` template render (compiles, runs): that fails
  `TestCertificateRecordHistory` on the timestamp marker while the declaration-only test still passes.
  Both mutations reverted byte-clean; the committed tree is the unmutated version.
- Verbatim render, per `## Not In Scope`: no parse/relativize/reformat, no new `RecordAt` call, no new
  error path, no DB migration, no log-browser wiring. The deletion's empty timestamp exercises the
  absent→"" no-render path in the same test.
- Scope: 1 production Go file + 1 template + 1 test file. Within the ≤3 production-file budget (template
  and test do not count). Nothing from `## Not In Scope` touched.
- Learnings to update (review's call): `certificate.md` §6 KNOWN-GAP bullet now resolved — the
  render-only remainder is LANDED (`HistoryRow.At` + `{{if .At}}` conditional render of the verbatim
  RFC-3339 string); the humanized `2026-02-14 18:40 UTC` display stays deferred.
