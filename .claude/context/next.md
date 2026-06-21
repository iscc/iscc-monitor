# Next Work Package

## Step: Single-record page at `GET /record?index=<seq>` (declaration / deletion / unknown schema)

## Advances
M-UI (Evidence Ledger frontend) Verify criterion (`target.md`):

> "the single-record page renders `declaration`, `deletion` (a new record — original preserved) **and
> an unknown `note.$schema`** without erroring"

and the row-repoint half of the record-list criterion:

> "the log-browser record list paginates via plain links … each row links to its single-record page".

This is the nearest unmet M-UI criterion and the planned next slice (state.md "Next Milestone" #1, and
the latest `review` handoff `**Next:**`). It precedes the certificate-of-inclusion slice (which
re-engages the oracle gate) and stays on the same leaf-first M-UI arc — no switch to an unrelated
refactor. No `critical`/`normal` issue is open (the 5 open issues are all `low`, loop-skipped), so
milestone work takes precedence.

## Goal
Add a server-rendered, no-JS single-record page showing one accepted leaf's seq, verbatim `iscc_id`,
verbatim `note.$schema`, a human-readable record-kind label (declaration / deletion / unknown), and its
raw record bytes — and re-point every `/records` row from `entries?index=<seq>` to it. This closes the
single-record M-UI Verify criterion and completes the record-list row affordance.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/proofserve/record.html` (the single-record template; not counted
    toward the ≤3 `.go` budget) — a DS-shell page mirroring `records.html`/`browser.html`: same `<link>`
    to `/_ds/tokens.css` + `/_ds/fonts.css`, `var(--*)` tokens, no CDN, `{{template "hubStatusBadge" .}}`
    partial, ledger card, and a graceful no-projection state.
- **Modify** (3 non-test/doc `.go` files — at budget):
  - `/workspace/iscc-monitor/internal/store/iscc_index.go` — add a `RecordAt(ctx, hubID, seq)`
    single-row reader returning the one projection row (`iscc_id_str`, `note_schema`) + a `found` bool;
    leaf read, store stays a leaf.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — add `serveRecord` + a `case "/record":`
    in `Handler`'s path switch; parse `index` via the existing `parseUint`; apply the accepted-tree cap;
    read the projection via `RecordAt` and the raw record bytes via the same bundle math
    `serveEntries`/`serveVerify` use; render `record.html`; add a parsed-once `recordTmpl` var +
    `//go:embed record.html`.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — add `mux.Handle("/record", proofs)` in
    `hubHandler` (the inner ServeMux mounts each proofserve route explicitly at lines 293-297; without
    this `/record` falls through to `tilesserve` in production even though the proofserve unit test
    passes by driving the handler directly).
  - **Modify (template/doc, not counted)**: `/workspace/iscc-monitor/internal/proofserve/records.html`
    — re-point each row's link from `entries?index={{.Seq}}` to `record?index={{.Seq}}` (keep the
    `record-seq` styling/markup).
  - **Modify (docs, not counted)**: `/workspace/iscc-monitor/CLAUDE.md` "Running a local dev instance"
    endpoint list — add a `GET /<domain>/log/record?index=<seq>` line (keeping docs in sync is part of
    the step).
  - **Tests (uncounted)**: `/workspace/iscc-monitor/internal/proofserve/record_test.go` (new) and
    `/workspace/iscc-monitor/internal/store/iscc_index_test.go`; update the row-link assertion in
    `/workspace/iscc-monitor/internal/proofserve/records_test.go`.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — the `## HTML record list` +
    `## Computed record-bytes HTTP surface` sections: the `>= LastSize` accepted-tree cap is the contract
    EVERY record-facing route follows; bundle reads keyed on an absolute index MUST compute
    `p := tiles.PartialTileSize(0, bundleIndex, size)`, never `p == 0`; the unquoted-`[data-status=…]`
    CSS trap for any surface carrying a negative `data-status="X"` assert.
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — the `## iscc_index writer/reader`
    section: scan `seq` int64→uint64; read `iscc_id_str`/`note_schema` via `sql.NullString` (NULL→"");
    absent row is the no-row case (return `found=false`, not an error); store stays a leaf
    (`go list -deps … | grep '^net/http$'` empty); keep `schema.sql`/`go.mod`/`go.sum` byte-identical.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go:329-401` (`serveEntries`) — copy the
    bundle-read + accepted-tree-cap flow verbatim; `serveRecords`:682-766 + `serveBrowser`:581-629
    (render-into-`bytes.Buffer`-then-200; `overlayStatus(fs, hubID, statuses)` + `badge.Label`).
  - `/workspace/iscc-monitor/internal/proofserve/records.html` + `browser.html` — the DS-shell HTML to
    mirror; `/workspace/iscc-monitor/internal/proofserve/handler_test.go:42-160` (`buildMirror`,
    `leafISCCID`, `mirrorLeaves`) and `entries_test.go` (`buildEntriesMirror`, `recordBytes`,
    `frameBundle`) — the fixtures the new tests reuse.

## Not In Scope
- The certificate of inclusion (`/inclusion/{iscc_id}`), the downloadable proof-bundle assembler, and
  the Bitcoin-anchor vs comparison-anchor panels — the NEXT M-UI slice (re-engages the oracle gate).
- Interpreting the ISCC-ID (no codec, no `hub_id`/realm decode) — ADR-0008: the page lists the id
  verbatim. The ONLY interpretation allowed is mapping the verbatim `note.$schema` string to a display
  label (declaration / deletion / unknown), and that mapping must never gate or error.
- Fetching/joining the original declaration for a deletion's per-id history — that cross-record view
  belongs to the certificate slice. Render only THIS leaf here.
- Any WASM / tier-2 "verify in your browser" affordance (later milestone).
- Consolidating the 3x `overlayStatus` precedence into `internal/badge` (open `low` issue, loop-skipped).
- Any schema change (`internal/store/schema.sql` byte-identical) or any change to `RecordProjections` /
  the `PollHub` ingest write path — the slice is read-side + render-side only.

## Implementation Notes
- **Schema → kind label (the only interpretation, fail-open).** Map the verbatim `note.$schema`:
  `iscc-note-0.8.0` → "Declaration", `iscc-note-delete-0.8.0` → "Deletion" (`CLAUDE.md` glossary lines
  174/178), anything else INCLUDING empty → "Unknown record type". An unknown/empty schema MUST render
  the page, never a 4xx/5xx — that is the explicit Verify clause. Keep the verbatim raw `note.$schema`
  string visible ALONGSIDE the friendly label (honest: the monitor interprets nothing). For a deletion,
  the copy should note it is a *new* record that marks the declaration redacted in derived views but
  never removes the committed declaration (glossary "Deletion") — but DO NOT fetch the original here.
- **Status mapping (mirror `serveEntries`, the record-facing contract).** `parseUint` empty/non-numeric/
  overflow → 400; `FollowState`/`RecordAt`/bundle-read DB error → 500; `LastSize == 0` or
  `seq >= LastSize` → 404 "leaf not covered by accepted checkpoint"; bundle not mirrored
  (`errors.Is(err, os.ErrNotExist)`) → 404; `logclient.ErrLeafOutOfBundle` → 404; non-GET → 405 (the
  shared gate at the top of `Handler`). **Decide and test the missing-projection case explicitly:** a
  leaf that is in-tree (`seq < LastSize`) and whose bytes ARE mirrored but has NO `iscc_index` projection
  row (`RecordAt` found=false) should still render the page from the raw bytes with id/schema shown as a
  "no projection indexed" state — do NOT 404 it solely on a missing projection (the bytes are the source
  of truth; the projection is a derived view). The page does its work from the bytes; the projection only
  supplies the id/schema labels.
- **Bundle math (learnings trap).** `bundleIndex := seq / tiles.TileWidth`; `offset := seq %
  tiles.TileWidth`; `p := tiles.PartialTileSize(0, bundleIndex, size)` — never pass `p == 0`
  unconditionally or the final partial bundle of a non-multiple-of-256 tree 404s a real leaf. Then
  `logclient.RecordBytesFromBundle(bundle, offset)`.
- **Raw bytes rendering.** The record envelope is opaque JCS/JSON-ish bytes. Render them inside a
  `<pre>`/`<code>` block via `html/template` auto-escape (string-cast, NOT `template.HTML`). They are
  bytes for human inspection, not parsed. Page must be complete with JS off.
- **Buffer-then-200 + overlay.** Render into a `bytes.Buffer`, set `Content-Type: text/html;
  charset=utf-8`, `WriteHeader(200)`, `buf.WriteTo(w)` (post-200 write-drop). Reuse `overlayStatus(fs,
  hubID, statuses)` + `badge.Label(status)` exactly as `serveRecords` does so the five-status badge
  partial renders, and `record.html` invokes `{{template "hubStatusBadge" .}}` over a view-model that
  carries `.Status` + `.Label`.
- **CSS-literal trap (cross-cutting).** If `record.html` inlines the badge color block AND a test
  asserts the body contains NO `data-status="verified"`, use the UNQUOTED CSS attribute form
  (`[data-status=verified]`), like `records.html`/`browser.html`, not the quoted form `dashboard.html`
  gets away with.
- **`recordTmpl`** is parsed once at package init —
  `template.Must(template.New("record").Parse(recordSource))` then `template.Must(t.Parse(badge.Source))`
  — exactly like `recordsTmpl`/`browserTmpl`, so a malformed template fails the build, not a request.
- **Store reader.** `RecordAt` is `SELECT seq, iscc_id_str, note_schema FROM iscc_index WHERE hub_id=?
  AND seq=?` via `QueryRowContext`; map `sql.ErrNoRows` → `found=false, nil`; scan the two text columns
  through `sql.NullString` (NULL→""), `seq` int64→uint64. `iscc_index.go` currently imports only
  `context, database/sql, fmt`; add `errors` if you consult `errors.Is(err, sql.ErrNoRows)`. Return a
  small store-owned value (e.g. reuse `RecordRow`).
- **Correctness rules in play:** `iscc_id → seq` is one-to-many and verification is schema-agnostic
  (ADR-0008) — this route reads ONE leaf by absolute seq, interpreting nothing beyond the display-label
  map; coverage honesty (ADR-0001) — never imply a leaf above `LastSize` is accepted; store stays a leaf
  (no `net/http`/`internal/logclient` in its closure).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestRecord ./internal/proofserve` passes, including new HTTP-seam tests (driving
  `proofserve.Handler` over `buildMirror`/`buildEntriesMirror`):
  - `GET /record?index=<seq>` returns `200 text/html` for an in-tree leaf and shows `seq`, the verbatim
    `iscc_id`, the verbatim `note.$schema`, and the raw record bytes;
  - a projection with schema `iscc-note-0.8.0` shows the "Declaration" label, one with
    `iscc-note-delete-0.8.0` shows "Deletion", and one with an unknown/empty schema renders 200 with the
    "Unknown" label (the no-error clause) — seed three projections with those schemas via
    `store.RecordProjections`;
  - `index` missing/non-numeric → 400; `seq >= LastSize` → 404; `LastSize == 0` → 404; bundle not
    mirrored → 404; non-GET → 405;
  - body links `/_ds/tokens.css` + `/_ds/fonts.css`, contains no `http://`/`https://`/`jsdelivr`/`cdn.`,
    and has no `<table>` (DS-shell + no-CDN invariant).
- `go test -run TestRecordsListsNewestFirst ./internal/proofserve` (updated) passes asserting each
  `/records` row now links to `record?index=<seq>` (not `entries?index=<seq>`).
- `go test -run TestRecordAt ./internal/store` passes: `RecordAt` returns the persisted row for an
  existing seq and `found=false` for an absent one.
- `go list -deps ./internal/store | grep '^net/http$'` is empty (store stays a leaf) AND
  `git diff --stat HEAD -- internal/store/schema.sql go.mod go.sum` is empty (byte-identical).
- Oracle/conformance gate is N/A and NOT skipped: the page is a pure decode + index render of persisted
  rows + mirrored bytes — no signature / RFC-6962 / Merkle / did:web / fsck / proof-build path (the
  proof-bundle assembler that DOES re-engage it is the next slice). Confirm `go test ./...` still runs
  the existing inclusion/consistency conformance tests green.

## Done When
`GET /<domain>/log/record?index=<seq>` serves a no-JS, DS-shell, CDN-free 200 page rendering one
accepted leaf (declaration / deletion / unknown schema, with raw bytes) and errors per the mapping
above, every `/records` row links to it, `cmd/iscc-monitor` mounts the route, and all Verification
checks pass with `mise run check` green.
