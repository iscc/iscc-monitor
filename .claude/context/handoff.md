## 2026-06-21 — Review of: `HubStatusBadge` five-status template partial (icon + label + silhouette)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/badge`, a pure leaf rendering the five-status `HubStatusBadge`
as inline SVG + text label, ported verbatim from `.claude/design/HubStatusBadge.dc.html`. The diff is
scope-clean (2 non-test source files + 1 test), the package is a stdlib-only WASM-shareable leaf,
fail-closed on unknown status, and golden + mutation-proven. This is the leaf-first foundation of M-UI
step (b); not wired into any surface yet (correct per next.md Not-In-Scope).

**Verification:**
- [x] `mise run check` green — all 17 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty (clean).
- [x] `go test -count=1 ./internal/badge` PASS uncached — 4 named tests all PASS verbosely.
- [x] Golden test asserts each status's distinct label AND distinguishing SVG marker (verified
  `M8.4 12.3`, unresolvable `M9.2 9.3`, unverified `M12 3.4 21 19H3z`, frozen `M8.2 3.3h7.6`, inactive
  two pause `<line>`s) — silhouette+label, not hue (ADR-0010 invariant 4). Markers verified verbatim
  against the `.dc.html` design source.
- [x] Pairwise-distinct: 5 renderings collected into a set, asserted `len == 5`.
- [x] Unknown/empty status fails closed — reviewer's own probe (`""`, `"verified "`, `" frozen"`,
  `"Frozen"`, `"VERIFIED"`) returns an error AND writes zero bytes; `data-status` only ever carries a
  validated status (the only entry point `Render` builds the `view` from the fixed `labels` table).
- [x] `go list -deps ./internal/badge | grep -E 'net/http|internal/store'` empty — leaf confirmed
  (full closure is stdlib only: `html/template`/`embed`/`bytes`/`io`/`fmt` + transitive stdlib).
- [x] `GOOS=js GOARCH=wasm go build ./internal/badge` OK — WASM-shareable (ADR-0010 surface C).
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` empty — byte-identical,
  no new deps / schema change.
- [x] Mutation-proven non-vacuous (reviewer's OWN run, reverted): corrupting the frozen octagon path
  to `MUTANT_PATH` FAILs `TestRenderLabelAndSilhouette`; reverted → green and tree byte-clean.
- [x] Scope discipline: advance commit touches only `handoff.md` + `internal/badge/{badge.go,badge.html,
  badge_test.go}` (2 non-test source ≤3). No `dashboard.html`/`handler.go`/`store`/`cmd` change — the
  Not-In-Scope "do not wire yet" is honored.
- [x] Gate-integrity scan over unpushed commits (`@{upstream}..HEAD`) — no `//nolint` / `t.Skip` /
  build-tag exclusion / swallowed error / deleted assertion in added Go/HTML. (define-next +
  update-state commits in the range are context-file-only.)
- [x] Oracle/conformance gate correctly N/A — pure static markup keyed on a status string; no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path.

**Issues found:** (none) — no defect. The lone open `low` notecheck item is unrelated and still valid;
no issues.md change.

**Codex second opinion:** Codex (gpt-5.5, xhigh, `reasoning summaries: none`) explored the diff
thoroughly — read the full diff, re-ran `go test ./...` (all green) and the named badge tests, ran
`git show --check HEAD` (no whitespace errors), and traced the status taxonomy through
`follower.glossaryStatus` + `metrics.Registry`. It surfaced no defects and emitted no explicit final
verdict paragraph (summaries off), so the transcript is a clean exploration with nothing to triage. Its
one useful observation — that `glossaryStatus` folds `rotated → unverified` and `inactive` is never
follower-produced, so the richer taxonomy isn't store-provable — matches next.md's Not-In-Scope and is
already captured in the badge learnings (no action).

**Next:** Wire the badge into `internal/dashboard` — replace the bare `{{.Status}}` text cell in
`dashboard.html` with `{{template "hubStatusBadge" .}}` after `template.Must(tmpl.Parse(badge.Source))`
in `handler.go`. The dashboard `row` view-model must expose BOTH `.Status` AND a fixed-table `.Label`
(the partial reads `.Label` directly and does NOT re-derive it), so the wiring step either exports
`badge.labels` or gives the row a pre-computed label. Only 3 of 5 badges (`frozen`/`verified`/
`inactive`) appear there until five-status resolution is made store-provable — a separate later M-UI
sub-step (ADR-0010). Keep that thread-through honest; do not claim five statuses on `/` before it lands.

**Notes:**
- **Design-fidelity (within next.md latitude):** advance correctly dropped the component's per-status
  inline `style=` hue colors and the chip `size`/`variant`/`padX/padY/fontPx` props (DS tokens + chip
  styling are a later M-UI step; accessibility rides icon+label+silhouette per ADR-0010). Added stable
  CSS hooks (`class="hub-status-badge"`, `data-status`, `.hub-status-badge-label`) and `aria-hidden` on
  the decorative SVG so the later token/CSS step can style without re-touching the partial. The label is
  always rendered (the design's `withLabel` toggle was dropped — the badge always needs a label here).
- **Learnings:** created `learnings/badge.md` (new area) + added a pointer row to the index. No
  cross-cutting promotion (all package-local: silhouette-port discipline, fail-closed label contract,
  minimal surface, leaf purity, 3/5-store-provable caveat). Index stays under ~120 lines.
- Pre-existing uncommitted context-file edits from `define-next` (`target.md`, plan, prd, `CLAUDE.md`,
  new `ADR-0010`) are left untouched per protocol — advance committed only impl/test/handoff, and they
  are not mine to commit.
