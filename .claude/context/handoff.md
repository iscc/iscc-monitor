## 2026-06-21 — Review of: Wire `HubStatusBadge` into the `/` dashboard

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` wired the landed `internal/badge` leaf into the `/` dashboard: the bare
`{{.Status}}` cell now renders through the `hubStatusBadge` partial (icon + label + silhouette) for the
three store-provable statuses, backed by a new fail-closed `badge.Label(status) (string, bool)`
accessor over the same fixed `labels` table. The diff is scope-clean (2 non-test source files + 1 HTML
+ 2 tests), keeps `internal/badge` a WASM-shareable leaf and `internal/store` a leaf, and the golden
test is mutation-proven non-vacuous. The page stays honest — only `verified`/`frozen`/`inactive`
appear; no claim of five statuses on `/`.

**Verification:**
- [x] `mise run check` green — all 17 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty (clean) — ran over the whole tree, not just the two packages.
- [x] `go test -count=1 ./internal/badge ./internal/dashboard` PASS (uncached).
- [x] `go test -run TestDashboardRendersEveryHub ./internal/dashboard` PASS; body contains
  `class="hub-status-badge"`, verified marker `M8.4 12.3`, frozen marker `M8.2 3.3h7.6`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/badge` OK — badge stays WASM-shareable after `Label`.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/badge'` empty —
  store remains a leaf.
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` empty — byte-identical,
  no dep/schema change.
- [x] Mutation-proven non-vacuous (reviewer's OWN run, reverted): reverting the html cell to
  `{{.Status}}` FAILs `TestDashboardRendersEveryHub`; restored → green and tree byte-clean.
- [x] Scope discipline: advance commit touches `handoff.md` + `internal/badge/{badge.go,badge_test.go}`
  + `internal/dashboard/{dashboard.html,handler.go,handler_test.go}` (2 non-test source ≤3). Nothing
  from Not-In-Scope (no `metrics.Registry` thread-through, no CSS/DS tokens, no SVG markup change).
- [x] Gate-integrity scan over unpushed commits (`@{upstream}..HEAD`) — no `//nolint` / `t.Skip` /
  build-tag exclusion / swallowed error / deleted assertion in added Go/HTML. The other two unpushed
  commits (define-next, update-state) are context-file-only.
- [x] Badge fail-closed contract intact: `Label` and `Render` both gate on the same `labels` table;
  `Label` returns `("", false)` on `""`/`pwned`/`VERIFIED`/`rotated`/`<script>` (unit-tested). The
  dashboard's `data-status` only ever carries a validated `hubStatus` output.
- [x] Oracle/conformance gate correctly N/A — pure HTML composition; no
  signature/RFC-6962/Merkle/did:web/fsck/proof path.

**Issues found:** (none) — no defect. The lone open `low` notecheck item is unrelated and still valid;
no issues.md change this iteration.

**Codex second opinion:** Codex (gpt-5.5, xhigh, `reasoning summaries: none`) explored the increment
thoroughly — dumped the full diff, read both source files and the templates, fetched the prior
`badge.go`/`buildRows` state via git, and ran `git show --name-status` / `git diff` over the handoff —
then ended on the handoff diff with no explicit findings or verdict paragraph (summaries off), so the
transcript is a clean exploration with nothing to triage. Same behavior as last iteration with this
config. No defects surfaced; no action.

**Next:** Make the full five-status taxonomy store-provable on `/` — thread the in-memory
`metrics.Registry` statuses (`unresolvable`/`unverified`, with `follower.glossaryStatus` folding
`rotated → unverified`) through to the dashboard so all five badges can appear honestly. Per
learnings/dashboard.md that thread-through belongs at the **registry boundary**, NOT in
`store.ListHubs` or `dashboard.hubStatus` (the store cannot prove those statuses; they live in
memory). When it lands, add an end-to-end render assertion for at least one in-memory-only status and
extend the golden test markers. After that, wire the badge into the log browser / dossier /
certificate surfaces (later M-UI sub-steps).

**Notes:**
- The `tmpl` init closure (`template.Must(...Parse(pageTemplate))` then `template.Must(t.Parse(
  badge.Source))`) is the idiomatic way to associate two sources when `template.Must` returns one
  value; this is the same associated-template pattern the next M-UI page should reuse via
  `badge.Source` + `badge.Label`.
- The `!ok → label = status` fallback in `buildRows` is defensive-only: `hubStatus` only emits valid
  `labels` keys, so `ok` is always true on the `/` page today. It is correct to keep (never render an
  unlabeled badge if a future status path slips through), and the accessor itself is directly
  unit-tested in both the hit and fail-closed branches.
- Learnings: settled the "decide label-threading at wiring time" open note in `learnings/badge.md`
  (now points at the landed `dashboard.buildRows` pattern) and added a status-cell-via-partial bullet
  to `learnings/dashboard.md`. No cross-cutting promotion to the index (all package-local). Index
  82 lines; detail files within budget.
- Pre-existing uncommitted context-file edits from an earlier define-next (`target.md`, plan, prd,
  `CLAUDE.md`, `ADR-0010`) remain untracked/modified in the tree — not produced by this iteration and
  not mine to commit per protocol.
