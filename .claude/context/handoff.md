## 2026-06-22 — Review of: Render the record-list `Type` column (per-row declaration/deletion/unknown badge)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** A clean, tightly-scoped M-UI render slice that completes the mockup's full 4-column
`Seq · Type · ISCC-ID · Logged` record-list head by adding a per-row Type badge mapped from the verbatim
`note.$schema` (ADR-0008). It introduces a handler-local `recordRowVM` (embedding `store.RecordRow` +
`Kind`/`KindKey`) and a `recordKindKey` CSS-token mapping that mirrors `recordKind`'s switch on the SAME
schema constants, so the label and the badge key can never drift. No store/struct/schema change; scope is
exactly 1 production file + 1 template + 1 test file. All gates green, the new test is mutation-proven
non-vacuous (both the badge-cell deletion AND the constant revert make it FAIL — the constant-vs-constant
vacuity trap is avoided), Codex clean, and the visual pass confirms the Type column lands matching the
mockup's named region with no new delta.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 28 packages ok.
- [x] `go test -count=1 -run TestRecords ./internal/proofserve` — PASS (existing record-list tests + new `TestRecordsRendersTypeColumn`, verbose-confirmed it actually runs).
- [x] New test asserts `<span>Type</span>` header + `Declaration`/`Deletion`/`Unknown record type` labels + the additive verbatim `http://purl.org/...iscc-note-0.8.0.json` schema, each driven from a HARDCODED literal URI.
- [x] Mutation 1 (reviewer-run, restored byte-clean): deleting the `.record-type` badge cell from `records.html` → `TestRecordsRendersTypeColumn` FAILS; restore byte-clean (`git diff`-clean, HEAD unchanged).
- [x] Mutation 2 / non-vacuity (reviewer-run, restored byte-clean): reverting `schemaDeclaration` to the short form `iscc-note-0.8.0` → test FAILS (the hardcoded-literal grounding works; not tied to the constant under test).
- [x] `gofmt -l .` — empty outside `cauldron/`.
- [x] Store leaf invariant: `go list -deps ./internal/store | grep -E 'net/http|proofserve'` empty (store untouched).
- [x] Grid alignment: `.records-head` + `.record-row` both `120px 130px 1fr 160px` (head columns aligned to data cells); `.ledger-status` (`160px 1fr`) correctly untouched.
- [x] All 12 new `.record-type` CSS tokens resolve in `internal/web/tokens.css` (`--font-mono`, `--text-2xs`, `--weight-bold`, `--tracking-wide`, `--space-1/2`, `--border-width`, `--border-subtle`, `--radius-xs`, `--text-muted`, `--status-success-text`, `--status-warning-text`).
- [x] CSS-literal trap closed: the `.record-type[data-kind=…]` selectors use the UNQUOTED form; the only `data-kind="…"` literals are an HTML attribute on the row (templated) + a prose CSS comment, never a quoted selector in `<style>`. `TestRecordsLinksTokensNoCDN` + `TestRecordsRendersInMemoryStatus` still pass.
- [x] No-CDN/`<table>` ban holds: `buildMirror` fixtures (which `TestRecordsLinksTokensNoCDN` uses) seed empty schemas → the badge is the neutral "Unknown record type" carrying no URL, so the whole-body `http://` ban still passes.
- [x] Oracle/trust-path: name-only diff over `internal/proof/`, `logclient/verify`, `didweb`, `index`, `notecheck`, `go.mod`, `go.sum`, `schema.sql`, consistency/equivocation/fork/shrink is empty — oracle gate correctly N/A (pure HTML render of the persisted, schema-agnostic `note.$schema`).
- [x] Gate-circumvention scan over all unpushed commits: no `nolint`/`t.Skip`/build-tag/swallowed-error in added code (the two matches are prose inside handoff text). The other two unpushed commits are pure CID-context (`next.md`, `state.md`), no code.
- [x] Scope discipline: 1 prod file (`handler.go`) + 1 template (`records.html`) + 1 test file; nothing from `## Not In Scope` (pager buttons / jump-to-sequence / back-link / masthead / store change) was touched.

**Issues found:** (none) — the increment does exactly what `next.md` asked. The deferred siblings (pager buttons, jump-to-sequence JS input, `← <hub> dossier` back-link, masthead instance-identity) remain correctly Not-In-Scope and are already-tracked issues, not defects of this slice.

**Codex second opinion:** Clean. Codex: "The change cleanly projects record schemas into a view-model-backed Type badge and updates the template/tests accordingly. I did not find any actionable regressions in the modified code." Codex independently grepped `recordKind` across `internal/certificate` + `internal/proofserve` and confirmed the mapping-consistency design. No findings to triage; matches my own assessment.

**Visual check:** Performed (agent-browser 0.29.0). Built the binary, seeded a fixture DB with declaration/deletion/unknown records (the live testnet renders empty schemas, so a fixture is needed to exercise the badge's rich states), launched against a long-poll realm so the fixture survives, and screenshotted `GET /sb0.iscc.id/log/records` vs `.claude/design/ISCC Monitor - Log Browser.dc.html`. The live page renders the full `SEQ · TYPE · ISCC-ID · LOGGED` head with per-row badges — Declaration green (`--status-success-text`), Deletion amber (`--status-warning-text`), Unknown neutral muted — and the verbatim `note.$schema` still rendering below each ISCC-ID (additive, ADR-0008). Matches the mockup's Type named region. Deltas observed are all pre-existing tracked items (pager buttons, jump-to-sequence, back-link, masthead identity, and the decorative blue-vs-green badge hue — text label is the load-bearing signal per ADR-0010 inv. 4). No NEW delta from this slice; nothing filed.

**Next:** This was the last pure-code M-UI named-region slice on the log-browser surface. The remaining log-browser deltas are design-first or human-blocked (pager buttons, jump-to-sequence JS input, `← <hub> dossier` back-link). The next cheapest slice likely shifts to another surface or to one of the open `normal` issues — the strongest candidates: the `/` realm-index config-driven instance-identity copy (#214 sub-2, blocks honest per-deployment masthead and recurs across all six SSR mastheads), the on-disk DB migration story (#40, the first operational hazard once a populated prod DB needs in-place upgrade), or the WASM verifier signature half (design-first; the cross-origin trust gap). define-next should weigh the instance-identity copy first — it is code-only, unblocks multiple mastheads, and is the most-referenced remaining `normal`.

**Notes:**
- Mapping consistency is sound: `recordKind` (label) and `recordKindKey` (CSS token) switch on the SAME `schemaDeclaration`/`schemaDeletion` constants — the key is derived from the schema, never parsed from the label, so they cannot drift. This is the right factoring for the cross-cutting "single mapping site" rule.
- The `recordRowVM` embeds `store.RecordRow`, so the template still reads `.Seq`/`.IsccID`/`.NoteSchema`/`.NoteTimestamp` verbatim and only adds `.Kind`/`.KindKey` — no field collision (build + tests + field-ref audit confirm). The kind is render-time only, never a stored column (store stays a leaf; the open DB-migration issue is NOT re-triggered).
- The vacuity trap that left the single-record label test green (`record_test.go`, open `low`) is correctly avoided here by seeding HARDCODED literal schema URIs — reviewer-proven by the constant-revert mutation. That `low` on `record_test.go` is a DIFFERENT file (single-record page), untouched here, so it stays open.
- Open `normal` issues remain (DB migration #40; `/` realm-index instance-identity copy #214 sub-2; WASM verifier signature half; Pages custom-domain binding; realm-index Anchor design-honesty) — none preempt the M-UI named-region work and none are touched here.
