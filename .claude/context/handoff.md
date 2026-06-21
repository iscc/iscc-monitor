## 2026-06-21 — Fix single-record kind-label constants to the full `note.$schema` URIs

**Done:** Set the two single-record kind-label constants to the full wire URIs the projection fold
stores verbatim, so real declarations/deletions now render "Declaration"/"Deletion" instead of "Unknown
record type", and scoped the no-CDN body ban to the document head so it no longer false-fails on the
now-realistic schema URI rendered as record data. This closes the open `normal` issue and completes the
single-record M-UI Verify criterion that was failing for every production record.

**Files changed:**
- `internal/proofserve/handler.go`: changed `schemaDeclaration` / `schemaDeletion` from the glossary
  short forms (`iscc-note-0.8.0` / `iscc-note-delete-0.8.0`) to the full wire URIs
  (`http://purl.org/iscc/schema/iscc-note-0.8.0.json` / `…iscc-note-delete-0.8.0.json`), and updated the
  doc comment to state these are the verbatim wire URIs matching `projection_test.go`. `recordKind`'s
  `switch` is unchanged (it `switch`es on these constants).
- `internal/proofserve/record_test.go`: scoped `TestRecordLinksTokensNoCDN` so the CDN scheme ban
  (`jsdelivr`/`http://`/`https://`/`cdn.`) and the DS-shell/token markers (`/_ds/tokens.css`,
  `/_ds/fonts.css`, `var(--font-sans)`, `var(--font-mono)`) assert over the document head (sliced at
  `</style>`) — the region where a CDN `<link>`/`url(` would appear — while the `<table>` absence still
  holds over the whole body. `schemaForSeq` followed the constant rename automatically (seq 0/1 now feed
  the production wire URIs); seq 2 (non-URL unknown) and seq 3 (empty) are unchanged.

**Verification:** `mise run check` → green (build + vet + all 20 packages test; proofserve uncached
1.784s). Per-criterion:
- `gofmt -l .` empty — PASS
- `go test -run TestRecord ./internal/proofserve` — PASS (incl. `TestRecordKindLabels`
  declaration/deletion/unknown/empty, and `TestRecordLinksTokensNoCDN` no longer false-failing on the
  verbatim `http://purl.org/...` schema) — PASS
- Declaration URI → "Declaration", deletion URI → "Deletion", non-URL/empty → "Unknown record type"
  (200) — PASS (asserted by `TestRecordKindLabels` over the now-real wire URIs)
- `grep '"iscc-note-0.8.0"\|"iscc-note-delete-0.8.0"' internal/proofserve/handler.go` → no match (short
  forms gone) — PASS
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` empty (byte-identical) — PASS
- Scope: exactly 2 files (1 production + 1 test), both in `internal/proofserve` — PASS

**Next:** Proceed to the originally-planned next slice: the certificate of inclusion
(`/inclusion/{iscc_id}`) + downloadable proof-bundle assembler, which re-engages the oracle/conformance
gate (signature / RFC-6962 / Merkle path). That slice was explicitly deferred out of this fix's scope.

**Notes:**
- Oracle/conformance gate is correctly N/A here — pure render of persisted projection rows + mirrored
  bundle bytes; no signature/RFC-6962/Merkle/did:web/fsck/proof path touched, and `internal/store`
  stays a leaf. Not skipped, just genuinely not engaged.
- The no-CDN ban is now scoped to the head (up to `</style>`), not the whole body. This is the review's
  preferred fix (keep the ban honest where a CDN link would appear, and it survives future surfaces that
  render real URIs) rather than the allowed alternative of asserting against a non-URL schema. Verified
  the template carries no `http://`/`https://` in the head and all DS-shell/token markers live there.
- No `go.mod`/`go.sum`/`schema.sql` change; no new constant or codec added (ADR-0008 schema-agnostic).
