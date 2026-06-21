# Next Work Package

## Step: `HubStatusBadge` five-status template partial (icon + label + silhouette)

## Advances
M-UI — Evidence Ledger frontend. Closes a self-contained slice of the M-UI Verify bar:

> `HubStatusBadge` renders **all five** statuses each with a distinct text label **and** a distinct
> inline-SVG silhouette (golden-tested per status) … status stays legible in grayscale + colorblind-safe
> (icon+label+silhouette, not hue).

This is the leaf-first foundation of M-UI step (b) in `state.md`'s convergence order. M-UI is the
nearest unmet milestone (M1/M2/M3 all met; no open `critical`/`normal` issue, so feature work proceeds).

## Goal
Build a pure, reusable server-rendered template partial that renders the five-status `HubStatusBadge`
(`verified` / `unresolvable` / `unverified` / `frozen` / `inactive`) as inline SVG + text label, ported
from the handoff's React component. This is the grayscale-safe status primitive every M-UI surface
(realm index, hub dossier, log browser, certificate) reuses; it lands first as a golden-tested leaf so
later steps can embed it and wire real five-status resolution.

## Scope
- **Create**: `internal/badge/badge.go` — a pure leaf package exporting the parsed partial + a `Render`
  helper (or an `html/template` `*template.Template` other packages associate via `template.Must(t.Parse(...))`).
- **Create**: `internal/badge/badge.html` — the embedded template partial (`{{define "hubStatusBadge"}}…{{end}}`),
  one `{{if}}` arm per status emitting that status's distinct inline `<svg>` silhouette + a `<span>` label.
- **Create (test)**: `internal/badge/badge_test.go` — golden test (test file, does not count against the ≤3).
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/design/HubStatusBadge.dc.html` — the source component: the five SVG
    silhouettes (exact `viewBox`/`path`/`circle`/`line` markup), the five labels, and the `META` table.
  - `/workspace/iscc-monitor/.claude/adr/0010-evidence-ledger-frontend.md` — the partial spec (lines
    47–52: inline SVG, five silhouettes check-circle · cloud-? · triangle · octagon-x · pause-circle;
    icon+label+silhouette, never hue alone) and the status palette (line 104).
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` — the SSR-leaf render posture this
    package mirrors (`html/template` NOT text; render-into-buffer; leaf with no `net/http`/`store` dep).
  - `/workspace/iscc-monitor/internal/dashboard/handler.go` + `/workspace/iscc-monitor/internal/dashboard/dashboard.html`
    — the established `//go:embed` + `template.Must` pattern to copy (this step does NOT modify them).

## Not In Scope
- **Do NOT wire the badge into `internal/dashboard` / `internal/proofserve` yet.** No change to
  `dashboard.html`, `handler.go`, `store.ListHubs`, or `cmd/iscc-monitor`. Wiring + replacing the bare
  `Status` text with the partial is the next step (and depends on this leaf existing).
- **Do NOT make the full taxonomy store-provable.** `unresolvable`/`unverified` are still not resolvable
  from the store (they live in the in-memory `metrics.Registry`). The partial accepts a status *string*
  and renders any of the five honestly; threading real five-status resolution through is a separate,
  later M-UI sub-step. Until then the dashboard still resolves only `frozen`/`verified`/`inactive`.
- **Do NOT embed DS tokens / self-hosted fonts / external CSS** here. The partial carries only the inline
  SVG + label markup (the handoff's inline `style=` colors may be ported as-is or dropped — accessibility
  rides icon+label+silhouette, not hue). Font/token embedding is its own M-UI step.
- **Do NOT build the frozen Exhibit panel.** The `frozen` *badge* (octagon-x silhouette + "Frozen"
  label) is in scope; the categorically-distinct non-dismissable Exhibit page-element is a later screen.

## Implementation Notes
- **Port faithfully from `HubStatusBadge.dc.html`.** Map the five `sc-if` arms to five Go-template arms.
  Use the component's exact SVG inner markup per status so the silhouettes match the design:
  - `verified` → `<circle cx=12 cy=12 r=9>` + check `<path d="M8.4 12.3l2.5 2.5 4.7-5.2">` (check-circle)
  - `unresolvable` → `<circle r=9>` + question `<path d="M9.2 9.3a3 3 0 0 1 5.6 1.2…">` + dot (question-circle)
  - `unverified` → triangle `<path d="M12 3.4 21 19H3z">` + exclamation line + dot (triangle-warning)
  - `frozen` → octagon `<path d="M8.2 3.3h7.6L20.7 8.2v7.6L15.8 20.7H8.2L3.3 15.8V8.2z">` + X lines (octagon-x)
  - `inactive` → `<circle r=9>` + two vertical `<line>`s (pause-circle)
  Labels (from `META`): `Verified` / `Unresolvable` / `Unverified` / `Frozen` / `Inactive`.
- **Choose the selection mechanism deliberately.** A single `{{define "hubStatusBadge"}}` with an
  `{{if eq .Status "verified"}}…{{else if eq .Status "unverified"}}…{{end}}` chain over a small
  view-model (`{Status, Label}`) is the simplest. Expose BOTH (a) an exported
  `Render(w io.Writer, status string) error` for direct use AND (b) the embedded source string (or a
  `MustParseInto(parent *template.Template)` helper) so a parent page template can `{{template
  "hubStatusBadge" .}}` it later — the standard `html/template` partial-include idiom (parent
  `template.Must(parent.Parse(badgeSrc))`, then invoke by name). Decide the exact surface from how
  `html/template` associated templates compose; keep it minimal.
- **Map the label inside Go from a fixed table, never trust the caller's string verbatim for the label**,
  so an unknown status fails closed (return an error, or render an explicit fallback) rather than emitting
  an attacker-controlled label. The five SVG arms are static literal template text (not `template.HTML`
  from input), so they auto-escape-safely; `html/template` (NOT `text/template`) is mandatory.
- **Keep the package a pure leaf.** Closure must be `bytes`/`embed`/`html/template`/`io` + stdlib only —
  NO `net/http`, NO `internal/store`. Verify with `go list -deps ./internal/badge`. Mirror the
  `internal/dashboard` embed + `template.Must(...Parse)` pattern (learnings/dashboard.md): parse the
  embedded source once at init so a malformed partial fails the build, not a request.
- **Oracle/conformance gate is N/A** — pure static-markup rendering keyed on a status string; no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path. `go.mod`/`go.sum`/`schema.sql` must be
  byte-identical (no new deps). State this in the review handoff.
- **Correctness rule (learnings index):** status is conveyed by icon + label + silhouette, never hue
  alone (ADR-0010 invariant 4). The golden test must assert the *silhouette* and *label* differ across
  statuses — not merely a color attribute.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/badge` passes uncached.
- The golden test asserts, for each of the five statuses, the rendered output contains that status's
  distinct text label (`Verified`/`Unresolvable`/`Unverified`/`Frozen`/`Inactive`) AND its distinct
  distinguishing SVG element — e.g. `verified` contains the check path `M8.4 12.3`, `unverified` the
  triangle `M12 3.4 21 19H3z`, `frozen` the octagon `M8.2 3.3h7.6`, `inactive` two pause `<line>`s,
  `unresolvable` the question `M9.2 9.3`.
- The golden test asserts the five rendered outputs are **pairwise distinct** (collect the five into a
  set and assert `len == 5`) — proving no two statuses collapse to the same silhouette.
- An unknown/empty status fails closed: the test asserts `Render` returns an error OR renders an explicit
  fallback, never an arbitrary attacker-controlled label.
- `go list -deps ./internal/badge | grep -E 'net/http|internal/store'` is empty (badge stays a leaf).
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` is empty (no new deps / schema change).

## Done When
`internal/badge` exists as a pure leaf rendering the five-status `HubStatusBadge` partial, its golden
test proves all five statuses produce distinct labels + distinct inline-SVG silhouettes (pairwise
unique) with an unknown status failing closed, and `mise run check` is green.
