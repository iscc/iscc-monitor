## 2026-06-21 — Review of: Frozen Exhibit on the hub dossier (`store.ListViolations` + non-dismissable markup)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance closes the open M-UI Verify clause for the frozen **Exhibit** (ADR-0010): a
new leaf read `store.ListViolations(hubID)` over the `violations` table feeds a categorically-distinct,
non-dismissable Exhibit panel on the per-hub dossier (`GET /<domain>`), listing each violation's `kind`
+ `detected_at` newest-first, with a "do not trust new state" notice. Clean, minimal, well-tested;
`mise run check` green, both mutations independently reproduced as non-vacuous, scope honored, leaf
purity intact, Codex agrees (no issues).

**Verification:**
- [x] `mise run check` — green (build + vet + all 19 packages `ok`).
- [x] `gofmt -l .` — empty (no files listed).
- [x] `go test -run TestListViolations ./internal/store` — PASS (newest-first, hub-scoped, non-zero +
  NULL-detected-at cases).
- [x] `go test -run TestDossier ./internal/dossier` — PASS (frozen Exhibit renders panel + both kinds +
  both RFC-3339 timestamps + frozen badge silhouette `M8.2 3.3h7.6`; verified dossier renders NO Exhibit).
- [x] Store leaf purity — `go list -deps ./internal/store | grep '^net/http$'` empty; only internal dep
  is `internal/tiles`; direct imports unchanged (`context crypto/sha256 database/sql embed errors fmt
  internal/tiles modernc.org/sqlite os time`). Dossier closure has no `internal/metrics`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/badge` — green (badge untouched).
- [x] No-CDN / non-dismissable — `dossier.html` carries no `http(s)://`/`cdn.`/`jsdelivr` and no
  `<button>`/`<script>`/` hidden>`/` hidden=` (the no-JS, non-dismissable baseline holds).
- [x] Mutation-proven non-vacuous INDEPENDENTLY (reviewer re-ran, both reverted byte-identical):
  (1) `ListViolations` `DESC → ASC` → `TestListViolations` FAILs (`kinds = [fork shrink]`, wrong order);
  (2) forcing `Frozen: true` in `buildData` → `TestDossierNoExhibitWhenNotFrozen` FAILs (verified dossier
  leaks the Exhibit).
- [x] Scope — 2 non-test `.go` files (`store/checkpoints.go`, `dossier/handler.go`) + 1 template
  (`dossier.html`) ≤ 3; all Not-In-Scope items honored (no record list, single-record page, certificate,
  proof-bundle, anchor panels, or overlay consolidation; raw evidence bytes deliberately left out).
- [x] Gate-integrity scan over all unpushed commits — no `//nolint`/`t.Skip`/build-tag/swallowed-error/
  loosened gate; the diff only ADDS a read + render + tests.
- [x] Oracle/conformance gate correctly N/A — pure HTML render of a persisted store row + a leaf read;
  no signature/RFC-6962/Merkle/did:web/fsck/proof path; go.mod/go.sum/schema.sql byte-identical.

**Issues found:** (none) — no new problems. The triplicated `overlayStatus`/`hubStatus` (now 3x with
the dossier copy) remains the open `low` issue, deliberately deferred per `next.md` Not-In-Scope.

**Codex second opinion:** Completed (exit 0). Explicit no-issues verdict: "No actionable correctness
issues were identified in the HEAD changes. The new frozen-dossier exhibit and ListViolations read are
consistently wired and existing tests pass." Matches my independent review; nothing to triage.

**Next:** Resume the remaining M-UI dossier surfaces in the planned order: the paginated **record list**,
then the **single-record page**, then the **certificate of inclusion** (the one that re-engages the
oracle/inclusion-proof conformance gate — give it a dedicated step), then the **proof-bundle assembler**.
A small separate clause is the distinct **Bitcoin-anchor vs comparison-anchor** panels. When a fourth
copy of the `overlayStatus`/`hubStatus` precedence would land on the record pages, that is the moment to
consolidate it into `internal/badge` (the `low` issue) instead of adding copy #4.

**Notes:**
- **Correctness confirmed at the read seam.** `ListViolations` mirrors the `RecordViolation` write
  exactly: column order `hub_id, kind, detected_at`, `unixOrNil` ↔ `sql.NullInt64` NULL-time symmetry,
  newest-first `ORDER BY detected_at DESC, id DESC`, empty-slice-not-error for a hub with none. SQLite
  quirk worth knowing (recorded in store.md): a NULL `detected_at` sorts LAST under `DESC`, so a
  time-unknown violation lands at the bottom of the newest-first list — acceptable for the Exhibit.
- **Frozen gate is safe and off the hot path.** `status == "frozen"` is the only path that issues the
  extra query; `overlayStatus` never downgrades `frozen` (it only promotes `verified`), so the gate
  cannot be defeated by a fresher poll verdict. Non-frozen dossiers issue zero extra queries.
- **ADR-0006 honesty preserved.** The Exhibit copy ("this freeze is not cleared automatically") never
  implies the freeze can be cleared; there is no unfreeze affordance (auto-unfreeze forbidden). The
  zero-rows-but-frozen edge renders the panel header via an `{{else}}` fallback, never a broken `{{range}}`.
- **Pre-existing uncommitted human edits (NOT mine, NOT this advance's, NOT staged by review).** The
  working tree carries uncommitted edits to `.claude/adr/0007-*.md` (+34 lines: the network-level-vs-
  hub-level DB rationale) and `.claude/context/issues.md` (+17 lines: the `[human]` scaling-trip-wire
  `low` issue). These are absent from `HEAD` and are clearly human-authored (`Source: [human]`); the
  advance flagged them and left them uncommitted, and I have likewise NOT swept them into the review
  commit. **Action for the human:** review and commit these two files deliberately when ready — they are
  consistent (the ADR cross-references the trip-wire issue) and harmless to carry, but should not be
  laundered into a CID commit.
- Learnings net-reduced: `store.md` 165→149 lines (collapsed two settled SQLiteFetcher bullets into one
  `settled:` line, added the `ListViolations` read bullet); `dashboard.md` gained one concise Exhibit
  bullet (the dossier-render home; no separate dossier.md detail file warranted yet).
