## 2026-06-21 — Single-record page at `GET /record?index=<seq>` (declaration / deletion / unknown schema)

**Done:** Added a server-rendered, no-JS, DS-shell single-record page that renders one accepted leaf's
seq, a kind label derived from the verbatim `note.$schema` (Declaration / Deletion / Unknown record
type — the only interpretation, fail-open), the verbatim ISCC-ID and schema, and the raw record bytes
read from the mirror, and re-pointed every `/records` row from `entries?index=<seq>` to
`record?index=<seq>`. The route is mounted in `cmd/iscc-monitor`'s per-hub mux so it works in
production.

**Files changed:**
- `internal/store/iscc_index.go`: added `RecordAt(ctx, hubID, seq) (RecordRow, found, err)` — a
  single-row projection reader via `QueryRowContext` on the `(hub_id, seq)` PK; `sql.ErrNoRows` →
  `(RecordRow{}, false, nil)` (absent is a plain miss, not an error); reuses `RecordRow`; added the
  `errors` import (now consults `errors.Is(err, sql.ErrNoRows)`).
- `internal/proofserve/handler.go`: added `//go:embed record.html` + parsed-once `recordTmpl`, the
  schema→kind constants, `recordData` view-model, `recordKind` helper (fail-open mapping), `serveRecord`
  (copies `serveEntries`' bundle-read + accepted-tree-cap flow incl. `p := tiles.PartialTileSize(0,
  bundleIndex, size)`; reads labels via `RecordAt` but never 404s on a missing projection — bytes are
  the source of truth), and a `case "/record":` in the path switch.
- `internal/proofserve/record.html` (new): DS-shell page mirroring `records.html`/`browser.html` —
  `/_ds/tokens.css` + `/_ds/fonts.css`, `var(--*)` tokens, no CDN, `{{template "hubStatusBadge" .}}`,
  unquoted `[data-status=…]` selectors, raw bytes in a `<pre><code>` via auto-escape (string-cast).
- `internal/proofserve/records.html`: row link `entries?index=` → `record?index=`; footnote reworded.
- `cmd/iscc-monitor/main.go`: `mux.Handle("/record", proofs)` + hubHandler doc note.
- `CLAUDE.md`: added the `GET /<domain>/log/record?index=<seq>` endpoint line; updated the `/records`
  line to say each row links to the single-record page.
- Tests: new `internal/proofserve/record_test.go` (12 HTTP-seam cases), new
  `internal/store/iscc_index_test.go::TestRecordAt`/`TestRecordAtScopedByHub`; updated
  `records_test.go` row-link assertions; updated `cmd/iscc-monitor/main_test.go::TestMirrorRecordsRoute`
  (stale `entries?index=` → `record?index=`) and added `TestMirrorRecordRoute` (binary-level mount proof).

**Verification:** `mise run check` → green (`go build`, `go vet`, all 20 packages `ok`, `gofmt -l .`
empty). Per-criterion:
- `go test -run TestRecord ./internal/proofserve` — PASS (in-tree leaf 200 across the bundle boundary
  shows seq/id/schema/raw bytes; Declaration/Deletion/Unknown labels incl. empty-schema → 200 no-error
  clause; missing/non-numeric index → 400; `seq >= LastSize` → 404; `LastSize == 0` → 404; bundle not
  mirrored → 404; non-GET → 405; DS-shell + no-CDN + no `<table>`; overlay status renders).
- `TestRecordsListsNewestFirst` (updated) — PASS: each `/records` row links `record?index=<seq>`.
- `go test -run TestRecordAt ./internal/store` — PASS: persisted row for existing seq, `found=false`
  for absent, hub-scoped.
- `go list -deps ./internal/store | grep '^net/http$'` empty AND no `internal/logclient`; `git diff
  --stat HEAD -- internal/store/schema.sql go.mod go.sum` empty (byte-identical).
- Conformance: `TestInclusionServedProofVerifies` / consistency tests re-ran uncached, green.

**Next:** The certificate of inclusion (`/inclusion/{iscc_id}`) + downloadable proof-bundle assembler
— the next M-UI slice, which re-engages the oracle gate. The single-record page deliberately renders
ONLY this leaf; the per-id cross-record history (original declaration joined to its deletion) belongs
to that certificate slice.

**Notes:**
- Oracle/conformance gate is correctly N/A and NOT skipped: the page is a pure decode-and-index render
  of persisted projection rows + mirrored bundle bytes — no signature / RFC-6962 / Merkle / did:web /
  fsck / proof-build path. The existing inclusion/consistency conformance tests still run green.
- Scope: exactly 3 non-test/doc `.go` files touched (`iscc_index.go`, `handler.go`, `main.go`) — at
  budget. The `cmd/iscc-monitor/main_test.go` edits are a stale-assertion fix (the row link changed) +
  one new mount-proof test; the next.md only named `records_test.go` for the row-link update, but the
  same affordance is asserted at the binary level in `main_test.go`, so it had to move too. No
  assertion was deleted or weakened — the cmd test now asserts the new `record?index=` link, and the
  added `TestMirrorRecordRoute` is the binary-level proof the new mount works (the proofserve unit test
  drives the handler directly and cannot catch a missing mux mount, per next.md).
- The missing-projection case (next.md's explicit decision point) is handled as designed:
  `RecordAt` `found=false` still renders 200 from the bytes with id/schema shown as "no projection
  indexed", schema→kind defaulting to Unknown — covered by `TestRecordRendersWithoutProjection`.
- Pre-existing unstaged change to `.claude/context/target.md` was left untouched and NOT committed (not
  my file to modify per the protocol).
