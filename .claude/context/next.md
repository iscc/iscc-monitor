# Next Work Package

## Step: Wire `HubStatusBadge` into the `/` dashboard

## Advances
M-UI (Evidence Ledger frontend) Verify criterion:
> "`HubStatusBadge` renders **all five** statuses each with a distinct text label **and** a distinct
> inline-SVG silhouette (golden-tested per status) … status conveyed by icon + label + silhouette and
> never by hue alone."

This is the first wiring of the landed `internal/badge` leaf into an HTTP surface, closing the start of
the M-UI requirement that "every hub [is rendered] via a five-status `HubStatusBadge` template partial
(icon + label + silhouette)" on the `/` realm index. It is the exact `**Next:**` from the latest
`review` handoff (PASS / CONTINUE for the badge partial). Only 3 of 5 statuses
(`verified`/`frozen`/`inactive`) are store-provable on `/` today, so the page honestly shows those three
through the badge; full five-status store-provability is a separate later M-UI sub-step (left in Not In
Scope).

## Goal
Replace the bare `{{.Status}}` text cell in the dashboard with the `hubStatusBadge` partial so every hub
on `/` renders its status as icon + label + silhouette (grayscale-safe), turning the badge leaf into the
first wired M-UI surface.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/badge/badge.go` — add ONE small exported label accessor (e.g. `Label(status string)
    (string, bool)`) that reads the existing unexported `labels` table, so a parent page can precompute a
    row's `.Label` from the single source of truth without re-deriving it. Do not change `Render`,
    `Source`, `PartialName`, `labels`, or `view`.
  - `internal/dashboard/handler.go` — associate the partial via
    `template.Must(template.New("dashboard").Parse(pageTemplate))` followed by
    `template.Must(tmpl.Parse(badge.Source))` (one parsed set), add a `Label string` field to the `row`
    view-model, and populate it in `buildRows` from `badge.Label(hubStatus(s))`.
  - `internal/dashboard/dashboard.html` — replace the `<td>{{.Status}}</td>` cell with
    `<td>{{template "hubStatusBadge" .}}</td>`.
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/context/learnings/badge.md` (the partial's surface + fail-closed
    contract + 3/5 store-provable caveat)
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` (mount/path-guard, store-provable
    status subset, coverage-honesty render)
  - `/workspace/iscc-monitor/internal/badge/badge.go`, `/workspace/iscc-monitor/internal/badge/badge.html`
    (partial expects `.Status` + `.Label`)
  - `/workspace/iscc-monitor/internal/dashboard/handler.go`,
    `/workspace/iscc-monitor/internal/dashboard/dashboard.html`,
    `/workspace/iscc-monitor/internal/dashboard/handler_test.go` (existing golden test to extend)
  - `/workspace/iscc-monitor/.claude/design/HubStatusBadge.dc.html` (design source of truth — do NOT
    change the SVG markup)

## Not In Scope
- Making the full five-status taxonomy (`unresolvable`/`unverified`/`rotated`) store-provable on `/` —
  separate M-UI sub-step; the `metrics.Registry` thread-through belongs there, NOT in `ListHubs` or
  `dashboard.hubStatus` (learnings/dashboard.md). Keep the page honest: only 3 statuses appear.
- DS v2 tokens, self-hosted fonts (`go:embed`), CSS, or chip styling — a later M-UI step. Do not add
  inline `style=` hue colors to the partial (ADR-0010 invariant 4; learnings/badge.md).
- Wiring the badge into the log browser, dossier, or certificate — later sub-steps in the same arc.
- Changing the badge SVG markup, `Render`, or the silhouette golden markers.
- ETag/Cache-Control on `/` (not a Verify criterion).

## Implementation Notes
- The partial reads `.Label` DIRECTLY and does NOT re-derive it from `.Status` (learnings/badge.md), so
  the `row` MUST carry a precomputed `.Label`. `badge.labels` is unexported — add a thin exported
  `Label(status) (string, bool)` reading that same table so the label stays single-sourced and the
  fail-closed contract is preserved (caller status never trusted verbatim for the label).
- The dashboard only ever produces `hubStatus(s)` ∈ {`verified`,`frozen`,`inactive`} — all valid
  `labels` keys — so `badge.Label` always returns `ok == true` here. Still, in `buildRows`, treat a
  `false` from `badge.Label` defensively (the page should not render an unlabeled/blank badge); do not
  silently emit an empty label.
- Use `html/template` (NOT `text/template`) — already the case; associating `badge.Source` into the same
  template set via `tmpl.Parse(badge.Source)` makes `{{template "hubStatusBadge" .}}` resolve over the
  `row` value (which exposes `.Status` and `.Label`). This is the associated-template idiom proven by
  `badge.TestPartialComposesIntoParent`.
- Keep the buffer-first render (`tmpl.Execute(&buf, …)` → 500-before-200) and the exact-path/method
  guards untouched — they are correctness load-bearing (learnings/dashboard.md).
- Correctness rule from learnings.md index: store stays a leaf — do NOT import `net/http` or
  `internal/badge` into `internal/store`; the dashboard imports both, never the reverse. `internal/badge`
  must remain a pure WASM-shareable leaf (no `net/http`/`internal/store`) after adding `Label`.
- Extend `internal/dashboard/handler_test.go`'s `TestDashboardRendersEveryHub` to assert the badge markup
  now appears: the body must contain the `hub-status-badge` wrapper and a per-status label (`>Verified<`,
  `>Frozen<`) AND a distinguishing silhouette marker (verified `M8.4 12.3`, frozen `M8.2 3.3h7.6`) —
  proving the partial rendered, not just the raw status word. Do NOT weaken the existing
  domain/origin/coverage assertions.
- Oracle/conformance gate is N/A for this step (pure HTML composition; no signature/RFC-6962/Merkle/
  did:web/fsck/proof path; `go.mod`/`go.sum`/`schema.sql` stay byte-identical).

## Verification
- `mise run check` is green (build + vet + test all pass).
- `gofmt -l internal/badge internal/dashboard` is empty.
- `go test -count=1 ./internal/badge ./internal/dashboard` passes (uncached).
- `go test -run TestDashboardRendersEveryHub ./internal/dashboard` passes and the rendered body contains
  `class="hub-status-badge"`, the verified silhouette marker `M8.4 12.3`, and the frozen marker
  `M8.2 3.3h7.6` (badge rendered for both store-provable hubs, not the bare status word).
- `GOOS=js GOARCH=wasm go build ./internal/badge` succeeds (the new `Label` accessor keeps the badge a
  WASM-shareable leaf).
- `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/badge'` is empty
  (store remains a leaf).
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` is empty (no dep/schema change).

## Done When
`advance` is done when the `/` dashboard renders each hub's status through the `hubStatusBadge` partial
(icon + label + silhouette) for the three store-provable statuses, the extended golden test asserts the
badge markup, and all Verification criteria pass.
