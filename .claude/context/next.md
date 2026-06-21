# Next Work Package

## Step: Fix single-record kind-label constants to the full `note.$schema` URIs

## Advances
M-UI (Evidence Ledger frontend) Verify criterion — quoted from `target.md`:
> "the single-record page renders `declaration`, `deletion` (a new record — original preserved) **and an
> unknown `note.$schema`** without erroring"

This step closes the **open `normal` issue** "Single-record page kind-label constants miss the real
`note.$schema` URIs" and resolves the latest review NEEDS_WORK verdict (handoff `**Next:**`). It is not
a detour from milestone work — it *completes* an in-flight M-UI Verify criterion that currently fails
for every real declaration/deletion. Per protocol a NEEDS_WORK handoff is fixed before new milestone
work, and the single-record slice cannot be pushed until this lands.

## Goal
The single-record page currently labels every production declaration and deletion "Unknown record type"
because the kind-label constants are CLAUDE.md glossary short forms (`iscc-note-0.8.0`) while the
projection fold stores the verbatim wire value — the full URI
`http://purl.org/iscc/schema/iscc-note-0.8.0.json`. Set the two constants to the full URIs and reseed
the test fixture with the production URIs so the page labels real records correctly and the test
exercises the real wire value.

## Scope
- **Modify**: `internal/proofserve/handler.go` (the only production file — 2 constant lines)
- **Modify (test)**: `internal/proofserve/record_test.go` (the `schemaForSeq` fixture follows the
  constants automatically; scope the no-CDN ban so the now-real schema URI does not false-fail)
- **Reference**:
  - `internal/logclient/projection_test.go:18-21` — the golden ground truth for the exact wire URIs
    (`http://purl.org/iscc/schema/iscc-note-0.8.0.json`, `…iscc-note-delete-0.8.0.json`).
  - `.claude/context/learnings/store.md` (lines 43-48) — the "stored `note.$schema` is the VERBATIM wire
    value — a full URI, NOT a short name" trap.
  - `.claude/context/learnings/http-surface.md` (lines 85-99) — the single-record section's durable
    trap: match the full wire URI, seed the test with the URI, and scope a no-CDN `http://` body ban to
    the template/CDN region once a real schema URI renders.
  - `.claude/context/issues.md` — the full issue text (the issue `review` deletes on PASS).

## Not In Scope
- The certificate of inclusion (`/inclusion/{iscc_id}`) and the downloadable proof-bundle assembler —
  the NEXT slice (it re-engages the oracle/conformance gate). Do not start it here.
- Touching `cmd/iscc-monitor`, `internal/store`, `internal/logclient`, or any file outside
  `internal/proofserve` — the defect is entirely local to the kind-label map and its test.
- Reworking `recordKind`'s structure (the `switch` at handler.go:835-844 is correct); only the two
  `const` values change.
- Adding a separate "glossary short name" constant or any ISCC-ID codec — ADR-0008 is schema-agnostic;
  the page interprets only the verbatim `note.$schema` against the wire URI.
- Any `go.mod`/`go.sum`/`schema.sql` change (must stay byte-identical).

## Implementation Notes
- In `internal/proofserve/handler.go:108-109`, change the two constants to the exact wire URIs (port them
  verbatim from `projection_test.go:19-20`):
  - `schemaDeclaration = "http://purl.org/iscc/schema/iscc-note-0.8.0.json"`
  - `schemaDeletion    = "http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"`
  The comment at handler.go:101-106 already says "the verbatim `note.$schema`", so it stays accurate; the
  constants now match that prose. `recordKind` (835-844) needs no change — it `switch`es on these
  constants.
- In `internal/proofserve/record_test.go`, `schemaForSeq` (lines 40-52) returns the *constants* for seq 0
  (declaration) and seq 1 (deletion), so those two cases follow the rename automatically and now feed the
  production wire URIs — which is the whole point: the test must exercise the production path, not invent
  short forms. Keep seq 2 (`"iscc-note-future-9.9.9"`, a non-URL unknown) and seq 3 (`""`, empty →
  Unknown) as-is so the unknown/empty no-error clause stays exercised.
- **The no-CDN ban interaction (the review's explicit warning, the load-bearing part of this fix).**
  `TestRecordLinksTokensNoCDN` (record_test.go:369-396) renders `index=0` (a declaration) and bans
  `"jsdelivr"`, `"http://"`, `"https://"`, `"cdn."` across the WHOLE body. Once seq 0's schema is the
  full URI, `record.html:272` renders `http://purl.org/...` as legitimate record data, so the whole-body
  `http://` ban false-fails. Fix the test, NOT the page: scope the CDN ban to the template/CDN region —
  the document head where a CDN `<link>`/`url(` would actually appear (slice the body at `</style>` or at
  `<body`, and run the `http://`/`https://`/`jsdelivr`/`cdn.` ban only over that head region). The
  legitimate same-origin `/_ds/` link asserts (`href="/_ds/tokens.css"`, `href="/_ds/fonts.css"`,
  `var(--font-sans)`, `var(--font-mono)`) and the `<table>` absence assert must still hold over the
  appropriate region. (Alternative the review allows: render that one no-CDN assertion against a non-URL
  schema such as seq 2 — but prefer scoping the region; it keeps the ban honest where a CDN link would
  appear and survives future surfaces that render real URIs.)
- Correctness rule (learnings index "schema-agnostic" + store.md trap): the stored `note.$schema` is the
  verbatim wire value (a full URI), so any consumer matching it against a literal MUST use the full URI
  and its test MUST seed the URI, never the glossary short form. CLAUDE.md's `iscc-note-0.8.0` is prose
  shorthand only.
- Keep the slice a pure render (oracle gate N/A — no signature/RFC-6962/Merkle/did:web/fsck/proof path is
  touched); `internal/store` stays a leaf; `go.mod`/`go.sum`/`schema.sql` byte-identical.

## Verification
- `mise run check` is green (build + vet + all packages test; `gofmt -l .` empty).
- `go test -run TestRecord ./internal/proofserve` passes — including `TestRecordLinksTokensNoCDN` (no
  longer false-failing on the now-realistic `http://purl.org/...` schema data) and the
  declaration/deletion kind-label assertions.
- Assertion: a single-record page for a leaf whose projection `note.$schema` is
  `"http://purl.org/iscc/schema/iscc-note-0.8.0.json"` renders the label **"Declaration"** (not
  "Unknown record type"); a leaf with `"http://purl.org/iscc/schema/iscc-note-delete-0.8.0.json"` renders
  **"Deletion"**; a non-URL/empty schema still renders 200 with **"Unknown record type"**.
- `grep -n '"iscc-note-0.8.0"\|"iscc-note-delete-0.8.0"' internal/proofserve/handler.go` returns nothing
  (the short forms are gone from the production constants).
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` is empty (byte-identical).

## Done When
`mise run check` is green, the single-record page labels the full-URI declaration/deletion schemas as
"Declaration"/"Deletion" (and unknown/empty as "Unknown record type") proven by
`go test -run TestRecord ./internal/proofserve`, the production constants no longer hold the short forms,
and the no-CDN ban no longer false-fails on the verbatim schema URI.
