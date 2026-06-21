<!-- area: internal/badge (badge.go, badge.html) -->
<!-- indexed-as: badge.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# HubStatusBadge — five-status SSR template partial

Read this when a step touches `internal/badge` or wires the badge into a page. Durable
cross-cutting rules live in the index (`.claude/context/learnings.md`); package-local
mechanics are here.

## `internal/badge` (the grayscale-safe status primitive)

- **The five silhouette `<path>`/`<line>` markup is ported VERBATIM from
  `.claude/design/HubStatusBadge.dc.html` and is the design source of truth.** The golden test
  pins one distinguishing marker per status (verified `M8.4 12.3`, unresolvable `M9.2 9.3`,
  unverified `M12 3.4 21 19H3z`, frozen `M8.2 3.3h7.6`, inactive `x1="9.7" y1="9"`). If the
  design component changes, re-port the inner SVG AND update the fixture markers together — they
  must stay byte-identical to the `.dc.html`. The component's per-status hue/`style=` colors and
  the chip `size`/`variant`/`padX/padY/fontPx` props were deliberately DROPPED (ADR-0010: hue is
  non-load-bearing for accessibility; chip styling + DS tokens are a later M-UI step). Do NOT
  re-introduce inline `style=` colors here.
- **Fail-closed is the security contract: `Render` validates `status` against the `labels` map
  BEFORE executing the template; an unknown/empty status returns an error and writes ZERO bytes.**
  The label rendered is ALWAYS a fixed-table value, never the caller's string — so a
  caller-supplied status can never inject a label. `data-status="{{.Status}}"` only ever carries a
  validated status because `Render` is the only entry point that builds the `view`. If a later
  step adds a second entry point (e.g. a parent template passing an arbitrary `view`), it MUST
  route status through the same `labels` lookup or the fail-closed guarantee is lost. Keep the
  `view` struct unexported.
- **Surface is `Render(w, status)` (direct) + `Source` (embedded src) + `PartialName`
  ("hubStatusBadge") + `Label(status) (string, bool)` (fixed-table accessor).** A parent page
  composes via `template.Must(parent.Parse(badge.Source))` then `{{template "hubStatusBadge" .}}`
  over a value exposing `.Status` and `.Label` — the standard `html/template` associated-template
  idiom, tested by `TestPartialComposesIntoParent`. No `MustParseInto` wrapper was added (YAGNI).
  The partial reads `.Label` DIRECTLY (it does NOT re-derive the label from `.Status`), so a parent
  row MUST carry a precomputed `.Label`. `labels` stays unexported; `Label` is the single-sourced,
  fail-closed (`("", false)` on miss) way to precompute it — never re-implement the table at a
  caller. settled: dashboard wiring resolved this — `dashboard.buildRows` calls `badge.Label(status)`
  with a `!ok → label = status` defensive fallback (defensive-only: `dashboard.hubStatus` only ever
  yields valid `labels` keys). Reuse that exact pattern for the next page (dossier / log browser).
- **Pure WASM-shareable leaf: closure is stdlib only (`bytes`/`embed`/`html/template`/`io`/`fmt`
  and their transitive stdlib — `os` shows up only via `fmt`, unavoidable).** No `net/http`, no
  `internal/store`. `html/template` (NOT `text/template`) is mandatory for auto-escaping. Parsed
  once at init (`template.Must`) so a malformed partial fails the build, not a request. Verify
  WASM-shareability with `GOOS=js GOARCH=wasm go build ./internal/badge` (green), not by grepping
  `os` out of `go list -deps`. Oracle gate N/A — pure static markup keyed on a status string; no
  signature/RFC-6962/Merkle/did:web/fsck/proof path; go.mod/go.sum/schema byte-identical.
- **Only 3 of 5 statuses are store-provable (`frozen`/`verified`/`inactive`); the other two
  (`unresolvable`/`unverified`) come from the in-memory `metrics.Registry` (`follower.glossaryStatus`
  folds `rotated → unverified`).** settled: `/` now renders all five — `dashboard.overlayStatus`
  overlays the live verdict onto the store subset (see `learnings/dashboard.md` for the precedence
  rule). A new page that wants the full taxonomy must reuse that `StatusSource` overlay; the store
  alone still cannot prove `unresolvable`/`unverified`.
