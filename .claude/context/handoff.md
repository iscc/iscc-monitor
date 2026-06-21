## 2026-06-21 — Paginated record list on the log browser (`GET /<domain>/log/records?from=…[&n=…]`)

**Done:** Added a no-JS, newest-first, plain-link-paginated HTML record list at the new exact route
`GET /<domain>/log/records`, backed by a new leaf read `store.ListRecords`. Each row links to that
leaf's per-record bytes (`entries?index=<seq>`) until the single-record page lands next step; an empty
index renders an informative 200 empty state (never a 404). This closes the list-pagination half of the
M-UI record-list clause (target.md lines 113-116).

**Files changed:**
- `internal/store/iscc_index.go`: added the `RecordRow` value struct and `ListRecords(ctx, hubID, from,
  n) ([]RecordRow, int, error)` — a newest-first (`seq DESC`), hub-scoped, seq-cursor window over
  `iscc_index` plus the hub's total count; `iscc_id_str`/`note_schema` read via `sql.NullString`
  (NULL → ""), empty index → empty slice + 0 + nil err. Added the `database/sql` import (already in the
  package closure via `checkpoints.go`; store stays a leaf).
- `internal/proofserve/handler.go`: added `case "/records": serveRecords(...)` to the switch, the
  `recordsData` view-model, the `serveRecords` flow (overlay status + page math + render-into-buffer-
  then-200), the `records.html` embed, the `recordsTmpl` template set (badge partial associated the
  same way `browserTmpl` does), and `defaultPageSize=50`/`maxPageSize=200` constants. Updated the
  `Handler` doc comment.
- `internal/proofserve/records.html`: new embedded Evidence-Ledger template (DS-token `<style>` shell +
  hubStatusBadge partial, UNQUOTED `[data-status=…]` selectors, no `<table>`, no CDN URL), the record
  list, plain `?from=…&n=…` pagination links, and the empty state.
- `cmd/iscc-monitor/main.go`: mounted `mux.Handle("/records", proofs)` in `hubHandler` (exact route
  beats the `/` subtree dispatch); updated the doc comment.
- `internal/store/iscc_index_test.go`: `TestListRecords` (newest-first, page window via `n`/`from`,
  total), `TestListRecordsScopedByHub`, `TestListRecordsEmpty`.
- `internal/proofserve/records_test.go` (new): `TestRecordsListsNewestFirst`, `TestRecordsPagination`,
  `TestRecordsLinksTokensNoCDN`, `TestRecordsRendersInMemoryStatus`, `TestRecordsNonGET`,
  `TestRecordsEmpty` — all at the HTTP seam over the existing `buildMirror` fixture.
- `cmd/iscc-monitor/main_test.go`: `TestMirrorRecordsRoute` (binary-level routing proof on `buildMux`).
- `CLAUDE.md`: added the `GET /<domain>/log/records?from=…[&n=…]` route bullet.

**Verification:** `mise run check` → green (build + vet + all 20 packages `ok`); `gofmt -l .` empty.
Per-criterion:
- `go test -run TestListRecords ./internal/store` PASS (newest-first, hub-scoped, page window, total,
  empty hub → empty slice + nil err). Mutation-proven non-vacuous (reverted): `DESC → ASC` FAILS
  `TestListRecordsScopedByHub`; dropping the `hub_id` filter FAILS the scope count + order.
- `go test -run TestRecords ./internal/proofserve` PASS (200 text/html, newest-first, per-row
  `entries?index=<seq>`, working `?from=…&n=…` link, no CDN URL, overlay status). Mutation-proven:
  `DESC → ASC` in the store FAILS the proofserve newest-first + pagination tests; changing the
  empty-state copy FAILS `TestRecordsEmpty`.
- `go test -run TestRecordsEmpty ./internal/proofserve` PASS (empty index → 200 empty state).
- `go list -deps ./internal/store | grep '^net/http$'` empty (store stays a leaf; its own `.Imports`
  are `context crypto/sha256 database/sql embed errors fmt internal/tiles modernc.org/sqlite os time`).
- `go test ./cmd/iscc-monitor` PASS (the `/records` mount does not regress the route table).
- `go.mod`/`go.sum`/`schema.sql` byte-identical to HEAD.

**Next:** Build the **single-record page** (declaration / deletion / unknown `note.$schema`) — the
record-row target should then be re-pointed from `entries?index=<seq>` to the single-record HTML page.
That step interprets `note.$schema` into a projection view (the first place the schema string is read,
not just listed), so it is the natural place to start the schema-aware projection layer. After that:
the **certificate of inclusion** (`/inclusion/{iscc_id}`) — the step that re-engages the
oracle/inclusion-proof conformance gate; give it a dedicated step.

**Notes:**
- **Pagination cursor is a deterministic seq scheme, pinned in tests.** `from` is the inclusive
  upper-bound seq the page starts at; `seq <= from ORDER BY seq DESC LIMIT n`. Older link =
  `oldest - 1` (one below the page's smallest seq); newer link = `newest + pageSize` and is shown only
  once `from > 0`. The newer cursor is intentionally over-tolerant: `ListRecords` clamps `seq <= cursor`
  with `LIMIT`, so an over-large newer cursor simply lands on the newest page — the link can never error
  or skip the newest run. The older-link liveness (`oldest > 0`) is conservative under contiguous leaf
  seqs (the projection writer indexes one row per accepted leaf); a gap below would render a live link to
  an empty next page, which cannot happen with the current contiguous indexing.
- **`seq` is the global PRIMARY KEY, not per-hub.** My first `TestListRecordsScopedByHub` reused seq 0
  across two hubs and the second insert's `ON CONFLICT(seq) DO UPDATE` reassigned the row to hub B (test
  failed, correctly). Fixed the fixture to disjoint seq ranges (a real network never reuses a leaf index
  across hubs). Worth knowing for any future multi-hub iscc_index test.
- **CSS-literal trap honored.** `records.html` uses the UNQUOTED `[data-status=verified]` selector form
  (NOT `dashboard.html`'s quoted form) so `TestRecordsRendersInMemoryStatus`'s negative
  `data-status="verified"` assert stays honest.
- **Overlay-status reuse, no 4th copy.** `serveRecords` reuses proofserve's EXISTING
  `overlayStatus(fs, hubID, statuses)` (the record list shows ONE hub) — it does not add a 4th copy of
  the precedence logic. The triplicated `overlayStatus`/`hubStatus` consolidation into `internal/badge`
  remains the open `low` issue, deliberately deferred per `next.md` Not-In-Scope.
- **Oracle/conformance gate correctly N/A** — pure HTML render of persisted `iscc_index` rows via a leaf
  read; no signature/RFC-6962/Merkle/did:web/fsck/proof path; go.mod/go.sum/schema.sql byte-identical.
- **Pre-existing uncommitted human edits left untouched (NOT mine, NOT committed).** The working tree
  still carries the human-authored edits to `.claude/adr/0007-*.md` and `.claude/context/issues.md` that
  the last review handoff flagged. I did NOT stage or commit them — the human should review/commit them
  deliberately when ready.
