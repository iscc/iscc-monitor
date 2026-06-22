## 2026-06-22 — Review of: Log-browser record list — render the mockup's `Logged` column from `RecordRow.NoteTimestamp`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** A clean, template-only M-UI slice that wires the already-landed `store.RecordRow.NoteTimestamp`
into the `/records` log-browser record list as the mockup's `Logged` column — verbatim RFC-3339 render
(ADR-0008, never re-formatted) with an honest `&mdash;` fallback for the NULL-timestamp common case, plus a
`Seq · ISCC-ID · Logged` column-header row. No handler/struct/store change; scope is exactly 1 production
file + 1 test file. All gates green, the new test is mutation-proven non-vacuous, Codex clean, and the
visual pass confirms the `Logged` column lands matching the mockup's named region with no new delta.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 28 packages ok.
- [x] `go test -count=1 -run TestRecords ./internal/proofserve` — PASS (existing record-list tests + new `TestRecordsRendersLoggedColumn`).
- [x] New test asserts the literal `2026-06-21T12:34:56Z`, the `&mdash;` empty fallback, and `<span>Logged</span>` — all present (verbatim verified by re-run).
- [x] Mutation (reviewer-run, restored via `git checkout`): deleting the `.record-logged` cell from `records.html` makes `TestRecordsRendersLoggedColumn` FAIL; restore byte-clean (`git diff`-clean).
- [x] `gofmt -l .` — empty outside `cauldron/`.
- [x] Store leaf invariant: `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` empty (store untouched).
- [x] Oracle/trust-path: name-only diff over `internal/proof/`, `logclient/verify`, `didweb`, `index`, `notecheck`, `go.mod`, `go.sum`, `schema.sql`, consistency/equivocation is empty — oracle gate correctly N/A (pure HTML render of a persisted leaf read).
- [x] No-CDN body ban + `<table>` ban + unquoted-`[data-status=]` trap: `TestRecordsLinksTokensNoCDN` + `TestRecordsRendersInMemoryStatus` PASS; diff adds no `http://`/`https://`/`cdn.`/`jsdelivr`/`<table>`/quoted-`data-status="…"`.
- [x] All 9 DS tokens used in the new CSS (`--space-3/4/5`, `--border-width`, `--border-default`, `--font-mono`, `--text-2xs`, `--tracking-wide`, `--text-muted`) resolve in `internal/web/tokens.css`.
- [x] Gate-circumvention scan over unpushed diff: no `nolint`/`t.Skip`/build-tag/swallowed-error in added code (the lone `//go:build` match is prose inside an earlier handoff).
- [x] Header columns align with data cells: head grid `120px 1fr 160px` (`Seq`/`ISCC-ID`/`Logged`) maps to the row's `.record-seq`/`.record-cell`/`.record-logged` cells.

**Issues found:** (none) — the deferred `Type` column is correctly Not-In-Scope and already named as the next step (next.md + handoff), not a defect; the head honestly lists only columns with a data cell.

**Codex second opinion:** Clean. Codex: "The change cleanly renders the existing NoteTimestamp field in the records template with appropriate escaping and fallback behavior, and the added test covers the new column. Existing tests pass and no blocking regressions were found." No findings to triage. Matches my own assessment.

**Visual check:** Performed (agent-browser 0.29.0 available). Built the binary, launched against a populated DB, and screenshotted `GET /sb0.iscc.id/log/records` vs `.claude/design/ISCC Monitor - Log Browser.dc.html`. The live page renders the `SEQ · ISCC-ID · LOGGED` header and per-row verbatim RFC-3339 timestamps in the rightmost column, matching the mockup's `LOGGED` named region — the increment's target. Deltas observed are all pre-existing cross-cutting M-UI gaps (the deferred `TYPE` column = next step; generic instance-identity masthead copy = tracked `normal` issue #214 sub-2; the `← <hub> dossier` back-link / styled pager buttons / JUMP-TO-SEQUENCE box). No NEW delta from this slice; nothing filed.

**Next:** The deferred sibling slice on this same surface — the **`Type` column / per-row type badge** (`declaration`/`deletion`/`unknown`) that completes the mockup's full 4-column `Seq · Type · ISCC-ID · Logged` head. Needs a per-row `recordKind(NoteSchema)` precompute: `recordsData` carries no per-row kind and `RecordRow` is a plain store value, so map each `RecordRow` to a small row VM in `serveRecords` (`handler.go`) carrying the kind label — reuse `recordKind` + the full-URI `schemaDeclaration`/`schemaDeletion` constants from `record.go`. Per the http-surface non-vacuous rule, seed the test with a HARDCODED literal schema URI (constant-vs-constant goes vacuous — see the open `low` issue on `record_test.go`). Stays code-only; store stays a leaf.

**Notes:**
- Visual-pass observation: the running monitor polled the live testnet `sb0.iscc.id` and overwrote the
  seeded fixture DB with REAL records — so the screenshot's timestamps are honest live `note.timestamp`
  values, an even stronger confirmation than synthetic fixtures. (Watch: a stale dev monitor can hold the
  configured port; bind a fresh exotic port and `pkill` cleanly before relaunch.)
- The `&mdash;` fallback is emitted as the HTML ENTITY (html/template passes the literal through), so the
  test asserts the entity string, not a rendered em-dash glyph — correct.
- No oracle/conformance path touched; `notecheck`/`derive_vkey.py` correctly N/A this iteration.
- Open `normal` issues remain (DB migration story #40; `/` realm-index instance-identity copy #214 sub-2;
  WASM verifier signature half; Pages custom-domain binding; realm-index Anchor design-honesty) — none
  preempt the cheapest M-UI code slice and none are touched here.
