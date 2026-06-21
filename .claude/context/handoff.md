## 2026-06-21 — `HubStatusBadge` five-status template partial (icon + label + silhouette)

**Done:** Added `internal/badge`, a pure leaf package rendering the five-status `HubStatusBadge`
partial as inline SVG + text label, ported faithfully from `.claude/design/HubStatusBadge.dc.html`. Each
status (`verified`/`unresolvable`/`unverified`/`frozen`/`inactive`) emits its own distinct silhouette
(check-circle · question-circle · triangle-warning · octagon-x · pause-circle) and its own label; the
label is mapped in Go from a fixed table so an unknown status fails closed. Not wired into any surface
yet (next step).

**Files changed:**
- `internal/badge/badge.go` (new): exports `Render(w io.Writer, status string) error` (direct render,
  fails closed on unknown/empty status), `Source` (the embedded partial src), and `PartialName`
  (`"hubStatusBadge"`) so a parent page can `template.Must(parent.Parse(badge.Source))` then
  `{{template "hubStatusBadge" .}}` — the standard `html/template` partial-include idiom. Label lives in
  a fixed `labels` table; caller's string is never echoed as a label.
- `internal/badge/badge.html` (new): `{{define "hubStatusBadge"}}…{{end}}` with one `{{if eq .Status …}}`
  arm per status emitting that status's exact SVG inner markup from the design component, plus a
  `<span>{{.Label}}</span>`. `html/template` (auto-escaping; SVG arms are static literal text).
- `internal/badge/badge_test.go` (new, test): golden test — per-status label + distinguishing SVG
  marker; pairwise-distinct (set of 5 == 5); unknown/empty status fails closed (error + zero output);
  parent-composition idiom test.

**Verification:** `mise run check` → green (all 17 packages `ok`, build + vet + test). Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/badge` PASS uncached; 4 named tests all PASS verbosely.
- [x] Golden test asserts each status's distinct label AND distinguishing SVG (verified `M8.4 12.3`,
  unresolvable `M9.2 9.3`, unverified `M12 3.4 21 19H3z`, frozen `M8.2 3.3h7.6`, inactive two pause
  `<line>`s) — silhouette+label, not hue (ADR-0010 invariant 4).
- [x] Pairwise-distinct: 5 renderings collected into a set, asserted `len == 5`.
- [x] Unknown/empty (`""`, `"pwned"`, `"VERIFIED"`, `"rotated"`, `"<script>"`) → error + nothing written.
- [x] `go list -deps ./internal/badge | grep -E 'net/http|internal/store'` empty — leaf confirmed.
- [x] `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` empty — no new deps / schema.
- [x] Mutation-proven non-vacuous (my own run, reverted): corrupting the frozen octagon path FAILs
  `TestRenderLabelAndSilhouette`; reverted → green, tree byte-clean.
- [x] Bonus: `GOOS=js GOARCH=wasm go build ./internal/badge` OK — the leaf is WASM-shareable for the
  verifier app (ADR-0010 surface C), matching the `proof/verify` purity posture.

**Next:** Wire the badge into `internal/dashboard` (replace the bare `{{.Status}}` text cell in
`dashboard.html` with `{{template "hubStatusBadge" .}}` after `template.Must(tmpl.Parse(badge.Source))`
in `handler.go`). The dashboard `row` view-model already carries `Status`; it needs a `Label`-bearing
shape OR the parent template can pass `.` if the row exposes `.Status`+`.Label`. Note: the dashboard
today resolves only the store-provable subset (`frozen`/`verified`/`inactive`), so only 3 of 5 badges
appear there until five-status resolution is threaded through (a separate, later M-UI sub-step per
next.md's Not-In-Scope). The badge `Render`/partial already accepts all five honestly.

**Notes:**
- **Oracle/conformance gate is N/A** (as next.md predicted): pure static-markup render keyed on a status
  string — no signature / RFC-6962 / Merkle / did:web / fsck / proof path. `go.mod`/`go.sum`/`schema.sql`
  byte-identical (verified empty diff). No new dependency.
- **Design-fidelity decisions (within next.md latitude):** dropped the component's per-status inline
  `style=`/hue colors and the chip `padX/padY/fontPx`/`variant`/`size` props (DS tokens + chip styling
  are explicitly out of scope for this step — accessibility rides icon+label+silhouette, not hue). SVG
  silhouette inner-markup is ported verbatim; `width/height` fixed at the design's `md` 15px. Added
  stable CSS hooks (`class="hub-status-badge"`, `data-status`, `.hub-status-badge-label`) and
  `aria-hidden="true"` on the decorative SVG (label is the accessible text) so the later token/CSS step
  can style without re-touching the partial. No `<head>`/`style` block — this is an embeddable fragment.
- **Surface kept minimal:** exposed `Render` (direct) + `Source` (parent composition) + `PartialName`,
  per next.md's "expose BOTH". Did NOT add a `MustParseInto` helper — `template.Must(parent.Parse(
  badge.Source))` is the idiom and is tested (`TestPartialComposesIntoParent`); adding a wrapper would be
  unused surface (YAGNI). Reviewer: confirm this is the wiring shape you want before the dashboard step.
- Pre-existing uncommitted context-file edits from `define-next` (`target.md`, plan, prd, `CLAUDE.md`,
  new `ADR-0010`) are left untouched and NOT committed by me (per protocol I commit only impl/test/
  handoff).
