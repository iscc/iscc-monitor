## 2026-06-21 — Frozen Exhibit on the hub dossier (`store.ListViolations` + non-dismissable markup)

**Done:** Added a leaf read `store.ListViolations(hubID)` over the `violations` table and wired it into
the per-hub dossier (`GET /<domain>`) so a frozen hub renders a categorically-distinct, non-dismissable
**Exhibit** panel (ADR-0006): a bordered "do not trust new state" panel listing each violation's `kind`
+ `detected_at`, newest-first. Non-frozen dossiers issue no extra query and render no Exhibit.

**Files changed:**
- `internal/store/checkpoints.go`: added `ListViolations(ctx, hubID) ([]Violation, error)` — a pure
  read of `hub_id, kind, detected_at` (raw_a/raw_b/proof_json left zero, out of scope here), ordered
  `detected_at DESC, id DESC`, NULL `detected_at` → zero `time.Time` via `sql.NullInt64` (the `unixOrNil`
  inverse). No new imports (uses the existing `context`/`database/sql`/`fmt`/`time`).
- `internal/dossier/handler.go`: gated `st.ListViolations(...)` on the resolved status being `frozen`
  (off the hot path); added `Frozen bool` + `Violations []violationRow` to `dossierData`, a `violationRow`
  render struct, and `violationRows`/`violationTime` helpers (RFC-3339-or-empty, mirroring `coverageTime`).
  Refactored `buildData` to take the already-resolved `status` + violations. Added the `time` import.
- `internal/dossier/dossier.html`: added the `{{if .Frozen}}` Exhibit `<section class="exhibit">` —
  distinct panel markup (accent-rail border, loud heading "Exhibit — self-consistency violation", literal
  "Do not trust new state from this hub.", per-violation `kind` + `detected_at` list, empty-list fallback
  copy). No `<button>`/`<script>`/`hidden` (non-dismissable, no-JS baseline). Page-scoped `<style>` over
  DS `var(--*)` tokens only; no external/CDN URL.
- `internal/store/checkpoints_test.go` (tests): `TestListViolations` (two violations newest-first, hub-scoped,
  non-zero detected-at, empty result for a hub with none) + `TestListViolationsNullDetectedAt` (NULL → zero time).
- `internal/dossier/handler_test.go` (tests): `frozenHub` helper, `TestDossierFrozenExhibit` (panel class +
  heading + "do not trust new state" + both kinds + both RFC-3339 detected-at + frozen badge silhouette
  `M8.2 3.3h7.6`; bans `<button`/`<script`/` hidden>`/` hidden=`), `TestDossierNoExhibitWhenNotFrozen`
  (verified dossier renders no `class="exhibit"` and no "do not trust new state" copy).

**Verification:** `mise run check` → green (build + vet + all 19 packages `ok`); `gofmt -l .` empty.
- `go test -run TestListViolations ./internal/store` → PASS (newest-first, hub-scoped, non-zero + NULL cases).
- `go test -run TestDossier ./internal/dossier` → PASS; existing covered/no-coverage/in-memory-overlay/405/
  not-in-store/mapping/precedence tests all still PASS; verified dossier renders no Exhibit.
- `go list -deps ./internal/store | grep '^net/http'` empty; store `.Imports` unchanged
  (`context database/sql embed errors fmt time` + tiles + sqlite) — leaf purity intact.
- `GOOS=js GOARCH=wasm go build ./internal/badge` → green (badge untouched).
- Dossier closure unchanged (`bytes embed html/template net/http time` + badge + store); no `internal/metrics`.

**Next:** Resume the remaining M-UI dossier surfaces. Per `next.md` Not-In-Scope, the planned order is the
paginated **record list**, then the **single-record page**, then the **certificate of inclusion** (the one
that re-engages the oracle/inclusion-proof conformance gate), then the **proof-bundle assembler**. A small
separate clause is the distinct **Bitcoin-anchor vs comparison-anchor** panels. The triplicated
`overlayStatus`/`hubStatus` (the `low` issue) remains deferred — a fourth copy on the record pages would
strengthen the case to consolidate into `internal/badge` then.

**Notes:**
- **Mutation-proven non-vacuous (both reverted byte-identical):** (1) `ListViolations` ordering
  `DESC → ASC` → `TestListViolations` FAILs (`kinds = [fork shrink]`); (2) forcing `Frozen: true` in
  `buildData` → `TestDossierNoExhibitWhenNotFrozen` FAILs (verified dossier renders the Exhibit). So a
  green-but-wrong ordering or a leaked-Exhibit cannot ship.
- **Non-dismissable test nuance:** an early version banned the bare substring `hidden`, which false-matched
  the CSS `overflow: hidden` layout property. Narrowed the ban to the HTML attribute forms ` hidden>` /
  ` hidden=` (plus `<button`/`<script`) — that is the actual dismiss/visibility-toggle surface.
- **Out-of-scope working-tree changes I did NOT commit:** the session-start git snapshot said "clean", but
  the tree already carried unrelated edits to `.claude/adr/0007-*.md` (+34 lines on per-hub-vs-network DB
  rationale) and `.claude/context/issues.md` (+17 lines). These are not mine and not part of this work
  package; I staged only my 5 implementation/test files + this handoff. Someone should confirm those two
  files are intended before they get swept into a later commit.
- Oracle/conformance gate correctly N/A: this is a pure HTML render of one persisted store row plus a leaf
  read — no signature, RFC-6962, Merkle, did:web, fsck, or proof path touched; go.mod/go.sum/schema.sql
  byte-identical.
- ADR-0006 honesty: the Exhibit copy ("this freeze is not cleared automatically") never implies the freeze
  can be cleared, and there is no unfreeze affordance (auto-unfreeze is forbidden).
