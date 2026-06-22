## 2026-06-22 — Render the record-list `Type` column (per-row declaration/deletion/unknown badge)

**Done:** Added the deferred `Type` column to the `/records` log-browser record list, completing the
mockup's full 4-column `Seq · Type · ISCC-ID · Logged` head. Each row now carries a per-kind badge
(Declaration / Deletion / Unknown record type) mapped from the verbatim `note.$schema` via the existing
`recordKind` single mapping site, precomputed in `serveRecords` into a small handler-local row VM (the
store's `RecordRow` has no `Kind` field). Template-only render slice — no store/struct/schema change.

**Files changed:**
- `internal/proofserve/handler.go`: added `recordRowVM` (embeds `store.RecordRow`, adds `Kind` +
  `KindKey`); added `recordKindKey` (mirrors `recordKind`'s switch for the stable lowercase CSS key);
  changed `recordsData.Records` from `[]store.RecordRow` to `[]recordRowVM`; `serveRecords` now maps each
  store row through `recordKind`/`recordKindKey` into the VM (cursor arithmetic still reads the store
  slice).
- `internal/proofserve/records.html`: 3→4-column grid template on both `.records-head` and `.record-row`
  (`120px 130px 1fr 160px`, `.ledger-status` untouched); added the `<span>Type</span>` header and the
  per-row `.record-type` badge cell; added DS-token-only `.record-type` CSS with unquoted
  `[data-kind=declaration]`/`[data-kind=deletion]` decorative-hue selectors.
- `internal/proofserve/records_test.go`: new `TestRecordsRendersTypeColumn` — seeds three accepted leaves
  with HARDCODED literal `note.$schema` URIs (declaration / deletion / garbage-unknown), asserts the
  `<span>Type</span>` header + all three labels + the additive verbatim schema render.

**Verification:** `mise run check` → green (build + vet + `go test ./...`, all 28 packages ok).
- `go test -count=1 -run TestRecords ./internal/proofserve` → PASS (existing record-list tests + new test).
- New test asserts `<span>Type</span>` header + `Declaration`/`Deletion`/`Unknown record type` labels +
  the additive verbatim `http://...iscc-note-0.8.0.json` schema, each driven from a HARDCODED literal URI.
- Mutation (run + restored byte-clean): deleting the `.record-type` badge cell → `TestRecordsRendersTypeColumn`
  FAILS. Extra non-vacuity check: reverting the `schemaDeclaration` constant → test FAILS (hardcoded-literal
  grounding works, avoiding the constant-vs-constant vacuity trap).
- `go list -deps ./internal/store | grep -E 'net/http|proofserve'` → empty (store stays a leaf).
- `gofmt -l .` empty outside `cauldron/`. No oracle/crypto/proof path touched (oracle gate N/A).

**Next:** This is the last pure-code M-UI named-region slice on the log-browser surface. Per the prior
handoff + state.md, the remaining log-browser deltas are design-first or human-blocked: the `← <hub>
dossier` back-link, the styled pager buttons, and the "Jump to sequence" input (a JS control — the no-JS
constraint defers it). The generic-instance-identity masthead copy is tracked `normal` issue #214 sub-2.
The next cheapest slice likely shifts to another surface or to one of the open `normal` issues (DB
migration #40, WASM verifier signature half).

**Notes:**
- The `Type` badge is purely additive: the verbatim `note.$schema` still renders in the ISCC-ID cell's
  `.record-schema` line (ADR-0008), so nothing was replaced — the new test pins this so a future "replace
  the schema with the badge" refactor would FAIL.
- Kept `recordKind` as the single label-mapping site (`serveRecord` already calls it; reused, not
  duplicated). `recordKindKey` is a separate CSS-token mapping deliberately switching on the SAME schema
  constants, so the label and the `data-kind` key can never drift; it derives the key from the schema, never
  by parsing the display label.
- Used the UNQUOTED attribute-selector form (`[data-kind=declaration]`) to keep the cross-cutting
  CSS-literal trap closed — no `data-kind="..."` literal leaks into the `<style>`. All `.record-type` CSS
  properties resolve to existing tokens in `internal/web/tokens.css` (`--font-mono`, `--text-2xs`,
  `--weight-bold`, `--tracking-wide`, `--space-1`, `--space-2`, `--border-width`, `--border-subtle`,
  `--radius-xs`, `--text-muted`, `--status-success-text`, `--status-warning-text`).
- The hue maps only declaration→success and deletion→warning; unknown keeps the neutral `--text-muted`
  base (decorative only — the text label is the grayscale-safe load-bearing signal, ADR-0010 invariant 4).
- The mockup's third label is "Unknown type"; kept the existing in-repo `kindUnknown = "Unknown record
  type"` constant rather than introducing a new literal (next.md directive). `buildMirror` seeds empty
  schemas, so existing record-list tests now render the neutral "Unknown record type" badge — confirmed it
  carries no URL, so `TestRecordsLinksTokensNoCDN` still passes.
