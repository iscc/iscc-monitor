## 2026-06-22 — Review of: `/` realm-index — add the per-hub Anchor projection + render the mockup's Checkpoint & Anchor ledger columns

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance closes sub-delta (3) of the `/` realm-index column-deltas `normal` exactly as
`next.md` scoped it: a read-only `Anchor` projection (latest-stamped-root OTS status via a correlated
`ots` subselect) on `store.HubSummary`/`ListHubs`, the `Checkpoint`/`Anchor`/`AnchorDot` view-model
fields, and a six-column ledger grid matching the mockup. Scope is exemplary (1 store file + 1 handler +
1 template + 2 tests, nothing from `## Not In Scope`); store stays a leaf; all gates green; both mutations
reproduced independently. Codex raised one [P2] design-honesty observation (per-hub vs per-checkpoint
Anchor) that I confirmed is a real M-UI-exit design question but NOT a defect of this spec-faithful
increment — filed `normal`, hence PASS_WITH_NOTES rather than a clean PASS.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 28 packages ok).
- [x] `gofmt -l .` — empty outside `cauldron/` (no formatting failure).
- [x] `go test -count=1 -run TestListHubs ./internal/store` — PASS (uncached): confirmed / pending
      (newest-wins) / never-stamped(empty) Anchor cases.
- [x] `go test -count=1 -run TestDashboard ./internal/dashboard` — PASS (uncached): six-column grid,
      `Checkpoint` + `Anchor` colheads, `data-anchor="confirmed"`/`>confirmed<`, `data-anchor="none"`/
      `>not anchored<`, plus the pre-existing hero/overlay/no-CDN/badge assertions.
- [x] Store leaf invariant — `go list -deps ./internal/store | grep '^net/http$'` empty; the only
      iscc-monitor internal dep is the pre-existing `internal/tiles` (no `logclient`/`dashboard`).
- [x] Mutation (both reproduced independently, reverted byte-clean): reverting the `ots` subselect to a
      literal `''` → `TestListHubsAnchorStatus` FAILS (`Anchor "" want "confirmed"`/`"pending"`); dropping
      the `.anchor-cell` data cell from `dashboard.html` → `TestDashboardRendersEveryHub` FAILS on
      `>confirmed<` / `data-anchor="none"` / `>not anchored<`. Tree `git diff`-clean after both reverts.
- [x] Oracle/conformance gate — N/A. No trust-root path touched (name-only diff over `internal/proof/`,
      `logclient/verify`, `didweb`, fork/shrink/equivocation/consistency, `derive_vkey` → empty). Pure
      HTML render of a persisted leaf read; go.mod/go.sum/schema.sql byte-unchanged.
- [x] Gate-circumvention scan over the unpushed range (`@{upstream}..HEAD`, 3 commits) — no `//nolint`,
      `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion. The sole `-` lines are the
      "Observed size" colhead/cell being RELABELLED to "Checkpoint", not a weakening.

**Issues found:** One `normal` filed from Codex's confirmed [P2] (realm-index Anchor is per-hub, not tied
to the displayed Checkpoint — a design-honesty question for the M-UI exit). No reviewer-originated defect.
Sub-delta (3) of issue:214 marked CLOSED (Checkpoint/Anchor columns land + visual-confirmed).

**Codex second opinion:** One [P2] finding (`hubs.go:50-51`): "Tie anchor status to the displayed
checkpoint — the subselect picks any OTS row, so the Anchor cell can describe an older root while the
Checkpoint cell shows a newer `f.last_size`, overstating current anchoring." TRIAGE — **confirmed real
as a design observation, but NOT a defect of this increment** (filed `normal`, did not block PASS): the
implementation faithfully matches `next.md`'s "latest-stamped-root" projection AND the mockup's
free-standing per-hub `anchorState` model; the decoupling is the EXPECTED steady state because OTS is
async/best-effort (ADR-0004) so the displayed checkpoint is almost always ahead of the latest confirmed
anchor; the authoritative per-checkpoint claim already lives in certificate §5 (`OTSForRoot` bound to the
§2 root via `ots.ConfirmedFor`). Codex's suggested fix (`o.tree_size = f.last_size`) is REJECTED without
a design pass — it would render "not anchored" for nearly every actively-polling hub and defeat the
column. Recorded as a design-pass question, not a code change here.

**Visual check:** Performed (ADR-0012) — `internal/dashboard/dashboard.html` is an SSR surface.
agent-browser 0.29.0 (bundles its own Chromium) launched headless against a throwaway in-package server
seeded by the test `fixtureStore` (rich states: a confirmed-OTS verified hub + a never-stamped frozen
hub) with `/_ds/` mounted so the page renders styled. Screenshotted the live `/` and the
`.dc.html` mockup. The live ledger renders the mockup's exact six-column order `# | HUB · DOMAIN |
COVERAGE SINCE | CHECKPOINT | ANCHOR | STATUS`: row 01 `sb0.iscc.id` → Checkpoint `42`, green-dot
`confirmed`, Verified; row 02 `sb1.amlet.id` → `no coverage yet`, Checkpoint `7`, faint-dot
`not anchored`, Frozen (tint). The grayscale-safe label is the load-bearing signal. No NEW visual delta;
the remaining sub-region deltas (config-driven masthead identity sub-delta 2, "recent declarers" footer
sub-delta 4) are already filed under issue:214 and out of this step's scope. Throwaway harness removed;
tree verified clean.

**Next:** Of the same `normal` (issue:214), the remaining slices are NOT pure code: sub-delta (2)
config-driven instance identity / realm name needs `internal/config` env wiring (its own work package);
sub-delta (4) "recent declarers checked" hero footer needs a recent-lookup history the store does not
track. The new per-hub-Anchor design `normal` and the existing no-migration + WASM-verifier-signature
`normal`s all want a deliberate design pass (STOP/design candidates), not a code-only slice. A reasonable
code-only next pick: wire `RecordRow.NoteTimestamp` into the log-browser record-list `Logged` column
(`internal/proofserve`) — a separate SSR surface reusing the landed store field, no new store read.

**Notes:**
- Open count after this review: 0 critical / 5 normal / 10 low (sub-delta (3) of issue:214 closed; one new
  per-hub-Anchor `normal` filed → net normal 4→5). DONE still requires 0 normal, so the loop continues.
- Learnings: `dashboard.md` gained the Checkpoint/Anchor-landed bullet + the per-hub-vs-per-checkpoint
  honesty rule (with the Codex-fix rejection); `store.md` gained the `Anchor`-projection settled note and
  was net-reduced 171→162 lines (collapsed the OTS-CRUD, OTS-back-off, and `AdvanceAccepted` settled
  bullets into tighter `settled:` summaries to stay within the rotation budget). Nothing promoted to the
  always-loaded index (package-local render/projection mechanics).
- Push: 4 commits ahead of `origin/develop` after this review commit (update-state + define-next + advance
  + review). Pushing on PASS_WITH_NOTES. The known `Pages` workflow failure on develop is the
  human-blocked custom-domain repo-settings step (a documented `normal`), not a code regression here.
