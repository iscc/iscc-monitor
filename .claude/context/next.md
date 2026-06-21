# Next Work Package

## Step: Paginated record list on the log browser (`GET /<domain>/log/records?from=…[&n=…]`)

## Advances
M-UI (Evidence Ledger frontend, ADR-0010) Verify criterion (target.md lines 113-116):

> "the log-browser record list paginates via plain links (`?from=…[&n=…]`, no-JS), newest-first,
> each row links to its single-record page, and an empty log renders the informative empty state (200)"

This is the first of the two M-UI record-list sub-steps the last `review` PASS handoff named ("Resume
the remaining M-UI dossier surfaces in the planned order: the paginated **record list**, then the
**single-record page** …"). It closes the *list-pagination* half of that clause; the single-record
page (next iteration) closes the rest. M1/M2/M3 are fully met; M-UI is the active milestone and this
is the nearest unstarted reachable Verify criterion. No open `critical`/`normal` issue preempts it.

## Goal
Add a no-JS, newest-first, plain-link-paginated record list over `iscc_index` at the new exact route
`GET /<domain>/log/records`, so a human can browse a hub's mirrored records and reach a per-record
view. This is the entry point for the certificate-of-inclusion arc and the first M-UI screen M3 left
entirely unbuilt.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/proofserve/records.html` (embedded template,
  Evidence-Ledger-dressed, mirroring `browser.html`'s DS-token `<style>` shell + badge partial).
- **Modify** (3 non-test/doc `.go` files):
  - `/workspace/iscc-monitor/internal/store/iscc_index.go` — add a leaf read
    `ListRecords(ctx context.Context, hubID int64, from uint64, n int) ([]RecordRow, int, error)`
    (newest-first window over `iscc_index`, scoped to one hub) + a `RecordRow` value struct. Keep
    `store` a leaf (stdlib only).
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — add `case "/records":
    serveRecords(...)` to the `Handler` switch, a `serveRecords` flow, a `recordsData` view-model, and
    the `records.html` embed + template-set wiring (associate the `hubStatusBadge` partial the same way
    `browserTmpl` does).
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — mount `mux.Handle("/records", proofs)` in
    `hubHandler` (the inner per-hub `ServeMux`, alongside `/inclusion`/`/consistency`/`/entries`/
    `/verify`) so the exact route beats the `/` subtree.
- **Modify (tests — uncounted)**:
  - `/workspace/iscc-monitor/internal/store/iscc_index_test.go` — `TestListRecords`.
  - `/workspace/iscc-monitor/internal/proofserve/handler_test.go` (or a new
    `records_test.go`) — `TestRecords...` HTTP-seam assertions.
- **Modify (docs — uncounted)**: `/workspace/iscc-monitor/CLAUDE.md` — add a
  `GET /<domain>/log/records?from=…[&n=…]` bullet to the route list (after the `GET /<domain>/log/`
  browser bullet, before `/checkpoint`).
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/iscc_index.go` (existing `RecordProjections` /
    `SeqsForISCCID` patterns + the one-to-many, schema-agnostic, interpret-nothing discipline).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` `ListViolations` (the newest-first
    leaf-read shape + empty-slice-not-error idiom + `sql.Null*` degrade to mirror).
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` `serveBrowser` + `browserData` +
    `browserTmpl` + `overlayStatus`/`parseUint` (the SSR render-into-buffer-then-200, badge partial
    wiring, the EXISTING overlay status to reuse).
  - `/workspace/iscc-monitor/internal/proofserve/browser.html` (the DS-token `<style>` shell, UNQUOTED
    `[data-status=…]` selectors, no-CDN baseline to copy).
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go:278-294` `hubHandler` (exact-route mount).
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` (iscc_index reader/writer bullets;
    NULL-time + ORDER quirks; the store leaf-purity rule).
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` (proofserve mount trap; the
    CSS-literal unquoted-selector trap; coverage-honesty status mapping).
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` (overlay-status precedence shape).

## Not In Scope
- The **single-record page** (declaration / deletion / unknown `note.$schema`) — the very next step;
  this step only LINKS each row to the existing `entries?index=<seq>` route as the row target until
  the single-record page lands. Do NOT build the single-record renderer now.
- The **certificate of inclusion** (`/inclusion/{iscc_id}`), the **proof-bundle assembler**, and the
  **Bitcoin-anchor vs comparison-anchor** panels — all later M-UI steps.
- Consolidating the triplicated `overlayStatus`/`hubStatus` into `internal/badge` (the open `low`
  issue). This step does NOT add a 4th copy: the record list shows ONE hub, so reuse proofserve's
  EXISTING `overlayStatus(fs, hubID, statuses)` already in `handler.go` — do not write a new copy.
- Any interpretation of `iscc_id` or `note.$schema` (ADR-0008: list verbatim, decode nothing).
- ETag / Cache-Control on the new route (off the M-UI Verify bar; size-dependent surface).
- Any schema change — `iscc_index` already has every column this needs; keep `schema.sql` byte-identical.

## Implementation Notes
- **`ListRecords` shape (newest-first window).** Mirror `ListViolations`: a leaf read returning plain
  Go values so `store` stays a leaf (verify `go list -deps ./internal/store | grep '^net/http$'` is
  empty; the package's own `.Imports` stay `context database/sql embed errors fmt time` + the sqlite
  driver). Return `[]RecordRow{Seq uint64; IsccID string; NoteSchema string}` plus the total row count
  (`SELECT COUNT(*) FROM iscc_index WHERE hub_id = ?`) so the template can render an honest "showing N
  of TOTAL" and decide whether an older/newer link is live. Scan `seq` as `int64` then `uint64(seq)`
  (symmetric with `RecordProjections`' `int64(r.Seq)` write — see store.md). Read
  `iscc_id_str`/`note_schema` through `sql.NullString` so a NULL column degrades to `""`, never an
  error. An empty result returns an empty slice + nil err (NOT an error) — the empty-log case the
  Verify clause requires. **Decode nothing** about the id or schema (ADR-0008 / learnings Correctness
  rule: "verification is schema-agnostic … Unknown schemas are indexed … but never interpreted").
- **Pagination semantics (no-JS, plain links).** Use a seq cursor, NOT OFFSET, so paging is stable
  under concurrent ingest: `from` is the (inclusive) upper-bound seq the newest-first page starts at;
  `n` is the page size. Query `SELECT seq, iscc_id_str, note_schema FROM iscc_index WHERE hub_id = ?
  AND seq <= ? ORDER BY seq DESC LIMIT ?` when `from` is present; when `from` is absent start from the
  newest (`WHERE hub_id = ? ORDER BY seq DESC LIMIT ?`). Parse `from`/`n` via the existing `parseUint`
  (empty or non-numeric → default, NOT a 400, so bare `/records` works). Default `n` to a small
  constant (e.g. 50) and clamp to a max (e.g. 200) so a hostile `n` cannot scan the whole index.
  Render older/newer links as plain `<a href="records?from=…&n=…">` computed from the first/last seq
  on the page — no JS, content in the served HTML. Pick ONE deterministic cursor scheme and pin it in
  tests; the exact convention is yours so long as it is no-JS and deterministic.
- **Row target.** Each row links to `entries?index=<seq>` (a relative link, like `browser.html`'s
  proof-surface list) — the existing single-leaf bytes route — as the per-record target until the
  single-record HTML page exists. The Verify clause says "each row links to its single-record page";
  `entries?index=<seq>` IS the per-record bytes view today, so the link is honest and will be
  re-pointed when the single-record page lands next step. Note this in the surrounding copy.
- **SSR discipline (copy from `serveBrowser`).** Read `FollowState` for the overlay status (reuse the
  EXISTING `overlayStatus(fs, hubID, statuses)` — no new copy), precompute the badge `.Label` via
  `badge.Label` with the defensive `!ok → label = status` fallback, render into a `bytes.Buffer` first
  so a template/store error is a 500 BEFORE any 200, then `buf.WriteTo(w)` (post-200 write-drop
  convention). `html/template` (NOT text/template) auto-escapes the id/schema strings.
- **DS shell + no-CDN (copy from `browser.html`).** Link `/_ds/tokens.css` + `/_ds/fonts.css`,
  page-scoped `<style>` over `var(--*)` tokens, no `http(s)://`/`cdn.`/`jsdelivr` anywhere in the
  body. CRITICAL CSS-LITERAL TRAP (http-surface.md): use the UNQUOTED `[data-status=verified]`
  attribute form in the badge color block, NOT `dashboard.html`'s quoted form — the record-list render
  test will carry a negative `data-status="…"`-style assert and the quoted selector would falsely
  satisfy it.
- **Mount trap (http-surface.md).** `/records` MUST be an explicit exact mount in `hubHandler`
  (`mux.Handle("/records", proofs)`) — the `/`-dispatch func only sends the bare path `/` to proofs
  and delegates every deeper path to tilesserve, so an unmounted `/records` would 404 via tilesserve.
- **Status mapping.** Non-GET → 405 and unmatched path → 404 are already owned by `Handler`. Within
  `serveRecords`: a `FollowState`/`ListRecords` DB error → 500; an empty index (or a hub with no
  accepted checkpoint) → 200 with the informative empty state (NEVER a 404 — coverage honesty,
  ADR-0001, mirroring `serveBrowser`'s `LastSize==0` → 200 empty page).
- **Oracle/conformance gate is N/A** for this slice — pure HTML render of persisted `iscc_index` rows
  via a leaf read; no signature / RFC-6962 / Merkle / did:web / fsck / proof path. Keep `go.mod`,
  `go.sum`, and `schema.sql` byte-identical.

## Verification
- `mise run check` is green (build + vet + all packages `ok`, `gofmt -l .` empty).
- `go test -run TestListRecords ./internal/store` passes: newest-first (`seq DESC`), hub-scoped, the
  page window respects `n`/`from`, the total count is correct, and an empty hub returns an empty slice
  + nil error. (Reviewer should be able to mutate `DESC → ASC` and see the test FAIL — non-vacuous.)
- `go test -run TestRecords ./internal/proofserve` passes: `GET /records` on a seeded multi-record
  fixture returns `200 text/html`, lists rows newest-first, each row contains an `entries?index=<seq>`
  link, a small `n` shows a working `?from=…&n=…` pagination link, and the body contains NO external
  CDN URL (no `http(s)://`/`cdn.`/`jsdelivr`).
- `go test -run TestRecordsEmpty ./internal/proofserve` (or an empty sub-case) passes: `GET /records`
  on a hub with an empty `iscc_index` returns `200` with the informative empty-state copy, never a
  404/5xx.
- `go list -deps ./internal/store | grep '^net/http$'` is empty (store stays a leaf after `ListRecords`).
- `go test ./cmd/iscc-monitor` passes (the `/records` mount does not regress the existing route table).

## Done When
`GET /<domain>/log/records[?from=…&n=…]` serves a no-JS, newest-first, plain-link-paginated, CDN-free
HTML record list (each row linking to its per-record view) with an informative 200 empty state, backed
by a leaf `store.ListRecords` read, and all Verification criteria pass.
