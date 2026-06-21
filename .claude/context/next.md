# Next Work Package

## Step: Certificate §6 RECORD HISTORY — render the per-id record list (declaration + any deletion)

## Advances
target.md **M-UI — Evidence Ledger frontend**, Verify criterion: the realm-wide certificate
"for a known id renders the numbered evidence clauses (subject + position; checkpoint `(size, root)`;
inclusion proof; signing key; anchor state; **full per-id record history incl. any deletion**)".
This step closes the **record-history** sub-item of that criterion — the last clause that is fully
backed by data the monitor already holds. It is also a named-region of the certificate mockup
(`.claude/design/ISCC Monitor - Certificate.dc.html` §6 "RECORD HISTORY", lines 67-69): the per-record
`label · seq` rows plus the "a deletion is a new record — the declaration is preserved" note.

Milestone work, not a preempting issue. `HasClause6` is `false` and never set; its `cert.html`
placeholder (lines 373-378) is an empty `clause-value`.

## Goal
Render §6 RECORD HISTORY for a certifiable id: the full one-to-many list of seqs the hub indexed under
that ISCC-ID (`SeqsForISCCID`, already fetched in `buildData`), each row labelled by its
`note.$schema` kind (declaration / deletion / unknown), with the deletion note shown when any record
in the history is a deletion. This is the last data-grounded certificate clause; it reuses the
schema→kind interpretation pattern already proven in `internal/proofserve` and adds no new store seam.

## Scope
- **Create**: (none)
- **Modify** (2 non-test source files, within the ≤3 budget):
  - `/workspace/iscc-monitor/internal/certificate/handler.go` — in `buildData`, after the §4 block,
    build the §6 record-history view-model from the `seqs` already in hand (one `store.RecordAt` per
    seq for its `NoteSchema`), set `HasClause6 = true` for a certifiable id, derive `HasDeletion`; add
    the §6 view-model fields + a local `RecordKind` helper with evergreen docstrings; update the
    file/`buildData`/`certData` docstrings to describe §6.
  - `/workspace/iscc-monitor/internal/certificate/cert.html` — fill the existing empty
    `{{if .HasClause6}}` §6 `clause-value` placeholder (lines 373-378) with the record-history rows +
    the conditional deletion note, reusing the existing `clause-mono` / `clause-note` classes (no new
    CSS, no CDN URL).
  - `/workspace/iscc-monitor/internal/certificate/handler_test.go` — test file, not counted toward the
    ≤3 budget. Add `TestCertificateRecordHistory` + a fixture seeding a declaration and a later deletion
    under one id.
- **Reference** (read before implementing):
  - `/workspace/iscc-monitor/.claude/context/learnings/certificate.md` — package-local mechanics:
    buffer-then-200 500 discipline, fail-closed clause decline, base64-Std cross-surface, the
    `html/template` `+`→`&#43;` escape trap (not load-bearing here — seqs/labels are plain ASCII).
  - `/workspace/iscc-monitor/internal/proofserve/handler.go:111-112` — the `schemaDeclaration` /
    `schemaDeletion` full wire-URI constants; `:832-847` — `recordKind` (the schema→label+isDeletion
    mapping to mirror).
  - `/workspace/iscc-monitor/internal/store/iscc_index.go:80-84` (`RecordRow`), `:157-192` (`RecordAt`),
    `:194-208` (`SeqsForISCCID`).
  - `/workspace/iscc-monitor/.claude/design/ISCC Monitor - Certificate.dc.html` lines 67-69 + 98-101 —
    the §6 region (rows are `label` + `seq · at`; a `claimHistory` of declaration + deletion) and the
    deletion note wording.
  - `/workspace/iscc-monitor/internal/certificate/handler_test.go:77-130` (`fixtureStore` /
    `fixtureStoreUnaccepted` — they already seed `RecordProjections`, so a deletion row is one extra
    `ProjectionRecord` with the deletion `note.$schema`) and `:506-596` (`fixtureStoreTiled`).

## Not In Scope
- **§5 BITCOIN ANCHOR.** It needs the OTS/anchor store seam, which does **not** exist yet (OTS
  anchoring is "not started"; there is no `anchor`/`ots` store method — only an `ots` table name appears
  in a store test fixture). Doing §5 honestly now means either building that seam first (a separate,
  larger step) or rendering only a content-free "pending" placeholder. Defer until the anchor store seam
  lands; render §6 first because it is fully data-grounded today. (`HasClause5` stays `false`.)
- **The downloadable proof-bundle assembler** `{checkpoint, inclusion/consistency proof, record bytes,
  hub key, ots?}` — its own oracle-gated step (re-engages the conformance gate); the §6 list is a
  store-read render with no crypto path.
- **The §4 `did:web:` + raw-domain `host:port` mis-render fix** (filed `normal`). `internal/didweb`
  exposes only the DID→URL direction (`DocumentURL`), no domain→DID `%3A` encoder, so fixing it cleanly
  means adding an encoder — a separate concern that would dilute this single §6 increment. Leave it
  filed; fold it in when a step deliberately touches the §4 DID-building path or adds the encoder.
- **The `internal/registry` `hubDomain` `ForceQuery` fix** (filed `normal`) — different file
  (`registry.go`), not touched here.
- Interpreting the id or schema beyond the kind label (ADR-0008: verification is schema-agnostic; §6
  shows the verbatim seqs + a fail-open kind label, nothing else — no projection of owner/gateway/etc.).
- Re-resolving `did.json` or ANY `net`/`net/http` work — §6 is a pure store read + render.

## Implementation Notes
- **Reuse, don't re-derive, the kind mapping.** Mirror `proofserve.recordKind` (handler.go:838-847):
  switch the verbatim `NoteSchema` on the two full wire URIs
  (`http://purl.org/iscc/schema/iscc-note-0.8.0.json` → declaration;
  `http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json` → deletion; everything else, incl. empty →
  unknown). The proofserve constants are unexported in another package, so define the two schema
  constants LOCALLY in `certificate` and a small pure `recordKind(noteSchema) (label string,
  isDeletion bool)` helper (keeps it testable and import-clean). **Correctness rule (learnings
  index):** `iscc_id → seq` is one-to-many and verification is schema-agnostic — index by seq, never
  gate on schema; an unknown/empty schema is listed verbatim with the unknown label and never errors
  the page.
- **Data source already in hand.** `buildData` already holds `seqs []uint64` (ascending) from
  `SeqsForISCCID`. For each seq call `st.RecordAt(r.Context(), hub.HubID, seq)` to read its
  `NoteSchema`. A `found == false` row (a projection gap) is an honest gap — list the seq with the
  unknown/empty-schema label, do NOT 500. A real `RecordAt` DB error IS a 500 — keep the buffer-then-200
  discipline: `return certData{}, http.StatusInternalServerError` BEFORE any 200 is committed (every
  other clause in `buildData` does this).
- **Cap the listed seqs to the accepted tree.** §1 already certified `seqs[0] < hub.LastSize`; list only
  rows with `seq < hub.LastSize` in §6 so a deletion (or any record) indexed ABOVE the accepted
  checkpoint — an unaccepted projection left by a frozen/failed poll — is never implied to be vouched
  for (ADR-0001 coverage honesty; the same cap `ListRecords`/`serveRecord`/§1 apply). The history always
  has at least `seqs[0]`, so the list is non-empty for a certifiable id.
- **View-model.** Add to `certData`: `RecordHistory []HistoryRow` (each
  `HistoryRow{Seq uint64; Label string; IsDeletion bool}`) and `HasDeletion bool`; set
  `HasClause6 = true` for a certifiable id. Render §6 unconditionally for a certifiable id — unlike
  §3/§4 there is no crypto/cache gate to fail closed on; the seqs are the irreplaceable accepted-tree
  projections §1 already certified against. Give the fields evergreen docstrings parallel to the §2/§3/§4
  fields, and bump the `HasClause6` doc comment (currently "false so their gated placeholders render
  nothing").
- **Template (`cert.html` §6 block).** Fill the existing `{{if .HasClause6}}` `clause-value`: a
  `{{range .RecordHistory}}` over the rows, each a `clause-mono` line like `{{.Label}} · seq {{.Seq}}`,
  then `{{if .HasDeletion}}<div class="clause-note">This id was later deleted. A deletion is a new
  record — the declaration above is preserved and still proves inclusion.</div>{{end}}` (wording from
  mockup line 69). Reuse `clause-mono` / `clause-note`; add no `<style>` rule and no external/CDN URL
  (the `noExternalCDN` body invariant + the no-JS baseline hold; §6 is plain server-rendered markup).
- **Test (`handler_test.go`).** Add `TestCertificateRecordHistory`: seed a fixture where the subject id
  has TWO projections under the SAME id — a declaration at `seqs[0]` and a later deletion at a higher
  seq still `< LastSize`. Extend `fixtureStore`/`fixtureStoreUnaccepted` to take extra
  `ProjectionRecord`s, or add a sibling helper (note `RecordProjections` keys uniqueness on
  `(hub, seq)`, so the deletion row needs a distinct seq under the same `iscc_id`). Assert the rendered
  body contains BOTH `seq <decl>` with the declaration label AND `seq <del>` with the deletion label AND
  the deletion note. **Make it non-vacuous**: the assertion must fail if §6 is neutered.
- **Oracle/conformance gate is N/A for this step** (a store read + render; no signature / Merkle / proof
  code touched), but still run the oracle suite to prove no regression (see Verification).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestCertificate ./internal/certificate` passes (all existing §1-§4 tests plus
  the new `TestCertificateRecordHistory`).
- `go test -count=1 -v -run TestCertificateRecordHistory ./internal/certificate` shows the new test
  PASS, asserting BOTH the declaration row (`seq <decl>` + declaration label) AND the deletion row
  (`seq <del>` + deletion label) AND the deletion note are present in the rendered HTML.
- **Mutation (non-vacuity — advance runs + reverts):** setting `data.HasClause6 = false` in `buildData`
  (or dropping the deletion row from `RecordHistory`) makes `TestCertificateRecordHistory` FAIL; revert
  leaves the tree clean and tests green.
- Oracle gate unbroken: `go test -count=1 ./internal/logclient ./internal/follower ./cmd/notecheck`
  all `ok` (no crypto path touched, so this must stay green).
- WASM/purity unaffected: `GOOS=js GOARCH=wasm go build ./internal/index ./internal/didweb` exits 0
  (no `net`/`net/http` added to any WASM-shared leaf).
- No new dependency: `git diff --stat HEAD -- go.mod go.sum` is empty.
- Scope discipline: exactly 2 non-test source files changed
  (`internal/certificate/handler.go`, `internal/certificate/cert.html`) plus the test file; no §5 /
  proof-bundle / §4-DID / registry work done.

## Done When
`mise run check` is green, `TestCertificateRecordHistory` passes and is mutation-proven non-vacuous
(`HasClause6 = false` makes it fail), the oracle suite stays `ok`, and `git diff` shows no `go.mod` /
`go.sum` change — i.e. the certificate renders the full per-id record history incl. any deletion for a
certifiable id, closing the record-history sub-item of the M-UI certificate Verify criterion.
