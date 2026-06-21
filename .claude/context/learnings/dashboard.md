<!-- area: internal/dashboard (handler.go, dashboard.html) + internal/store/hubs.go -->
<!-- indexed-as: dashboard.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# Dashboard — server-rendered hub-list page at `GET /`

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## `GET /` hub-list dashboard (`internal/dashboard` + `store.ListHubs`)

- **The `/` mount is the lowest-priority `http.ServeMux` pattern, so it never shadows a more-specific
  route — but it DOES catch every unmatched path, so the handler must guard `r.URL.Path != "/"` → 404
  itself.** `buildMux` mounts `dashboard.Handler(st)` at `/` next to exact `/metrics`/`/healthz` and the
  per-hub `/<domain>/log/` subtrees; most-specific match means all of those still win. Reviewer
  re-confirmed with a standalone mux: a `/a/` subtree + `/` root resolves `/a`,`/a/`,`/a/x` to the
  subtree and only `/b` to root. The dashboard's own exact-path guard is what stops `/b`-style unknowns
  rendering the page. Keep BOTH the mount-at-`/` and the in-handler path guard — neither alone is enough.
- **`hubStatus(HubSummary)` is the store-provable subset ONLY (`inactive`>`frozen`>`verified`); the
  richer live verdicts come from an OVERLAY at `buildRows`, never from `ListHubs`.** `hubStatus`
  switches `!Active → inactive`, `Frozen → frozen`, else `verified` — it MIRRORS `proofserve.hubStatus`
  and EXTENDS it with the realm-registry `inactive`. Do NOT add `unverified`/`unresolvable`/`rotated` to
  `hubStatus`/`ListHubs`: those are not store-provable. settled: the thread-through landed —
  `overlayStatus(s, statuses StatusSource)` consults a tiny `dashboard.StatusSource` interface
  (`Status(hubID) (string,bool)`, satisfied structurally by `*metrics.Registry`) so the package never
  imports `internal/metrics`. **Precedence is load-bearing and must not regress:** overlay applies ONLY
  when `hubStatus(s) == "verified"`, adopts ONLY `unresolvable`/`unverified`, and is nil-tolerant
  (a fresher poll verdict is honest, but durable `inactive`/`frozen` must win). `TestOverlayStatusPrecedence`
  pins the table; `TestDashboardRendersInMemoryStatus` is the non-vacuous HTTP-seam render
  (reviewer mutation-confirmed: `overlayStatus`→`hubStatus` renders `data-status="verified"`, test FAILS).
  settled: the per-hub log-browser cell (`proofserve.serveBrowser`) now reuses this exact
  `StatusSource`-interface + `overlayStatus` shape (its own local copy, no `internal/dashboard` import —
  see `learnings/http-surface.md`). Reuse the same shape for the upcoming hub dossier / record pages.
- **`inactive` is currently unreachable through the public store API (no `SetActive` writer; `UpsertHub`
  inserts the schema default `active=1`).** So the golden HTTP-seam test cannot drive a hub to
  `inactive`; the advance covered it with a white-box table test on the package-private `hubStatus`
  (hence `package dashboard`, not `dashboard_test`). When a registry-deactivation writer lands, add an
  end-to-end inactive-render assertion through the public surface — until then the table test is the
  only coverage and is correct.
- **Coverage honesty (ADR-0001) is rendered, not just stored: `HasCoverage` false → literal "no coverage
  yet"; true → "size N at <RFC3339>".** `ListHubs` reads `monitored_since_{size,time}` via the
  `LEFT JOIN follow_state` so a never-polled hub still appears (its `last_size`/`frozen` are NULL → zero
  value). `coverageTime` returns "" when `!Set || Since.IsZero()` so "coverage started, time unknown"
  degrades to size-only. The fixture proves the split: the `AdvanceAccepted` hub shows `size 42 at …`,
  the `AdvanceFollowState`-then-`Freeze` hub shows `no coverage yet` (advance-follow-state does NOT set
  coverage). Never render the observed `last_size` as if it were a coverage guarantee.
- **`store.ListHubs` is a pure read returning plain Go types (`HubSummary` + `CoverageInfo`), so store
  stays a leaf** (`go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/logclient'`
  empty). The dashboard imports `store` AND `internal/badge`, never the reverse; its closure is `bytes
  embed html/template net/http internal/store internal/badge`. `html/template` (NOT `text/template`)
  auto-escapes hub domains; render into a `bytes.Buffer` first so a template/store error is a 500 BEFORE
  any 200 (the post-200 `buf.WriteTo(w)` drop is the documented broken-client convention). Oracle gate
  N/A — pure HTML of persisted rows, no signature/RFC-6962/Merkle/did:web/fsck/proof path;
  go.mod/go.sum/schema byte-identical.
- **`GET /` is now the Evidence-Ledger grid (no `<table>`), styled by a page-scoped `<style>` block in
  `dashboard.html` over the shared `var(--*)` DS tokens.** Every token used resolves in `internal/web/tokens.css`
  EXCEPT `--status-error-bg` (the frozen-row tint), which has no token and is therefore used WITH a literal
  fallback `var(--status-error-bg, rgba(245,97,105,0.06))` — decorative only (ADR-0010 inv.4: badge silhouette
  + label carry the status grayscale-safe; the hue is never the sole signal). Keep page layout local to this
  `<style>`, NOT in tokens.css, so it cannot regress another surface. `TestDashboardRendersEveryHub` asserts
  `display: grid` + `var(--font-sans)`/`var(--font-mono)` present AND `<table>` absent (both mutation-confirmed
  non-vacuous); `TestDashboardLinksTokensNoCDN` still bans `http(s)://`/`cdn.`/`jsdelivr` — passes only because
  the body renders scheme-less `Origin`, never an `https://` `base_url` (see web.md trap).
- **CSS-Grid ellipsis trap: a grid cell that wraps no-wrap+`text-overflow:ellipsis` content needs `min-width:0`.**
  Grid items default to `min-width:auto`, so the no-wrap `.hub-name`/`.hub-origin` would set the column's
  min-content width and the ellipsis never engages — a long domain pushes the coverage/status columns out of
  view. Fixed by `.hub-cell { min-width: 0 }` on the wrapping div (Codex P2, confirmed + fixed in review). Any
  future ledger cell that intends to ellipsize must carry `min-width: 0` on the grid item, not just the children.
- **The frozen dossier (`internal/dossier`) adds an Exhibit panel ON TOP OF the badge — markup-distinct
  per ADR-0010, gated `{{if .Frozen}}` where `Frozen = status == "frozen"`.** It is a full bordered
  `<section class="exhibit">` (loud heading "Exhibit — self-consistency violation" + literal "Do not
  trust new state from this hub." + a per-violation `kind`/`detected_at` list), NOT a recolored chip.
  Non-dismissable by construction: no `<button>`, no `<script>`, no `hidden` attr (tests ban ` hidden>`/
  ` hidden=` specifically, NOT the CSS `overflow: hidden`). The `ListViolations` read stays off the hot
  path — only a frozen hub queries it (safe because `overlayStatus` never downgrades `frozen`, only
  promotes `verified`); a frozen-with-zero-rows hub still renders the panel header via an `{{else}}`
  fallback, never a broken `{{range}}`. Empty-list and NULL-detected-at ("detected at an unknown time")
  branches are coverage-honesty discipline applied to evidence timestamps. Both the ordering and the
  Frozen gate are mutation-proven non-vacuous (reviewer reconfirmed independently).
- **The status cell renders through the `hubStatusBadge` partial, not the bare word.** The partial is
  associated into the page set once at init (`template.Must(template.New("dashboard").Parse(pageTemplate))`
  then `template.Must(t.Parse(badge.Source))`, wrapped in an init closure since `template.Must` returns
  one value); `<td>{{template "hubStatusBadge" .}}</td>` resolves over each `row` (which carries `.Status`
  + a precomputed `.Label`). `buildRows` precomputes `.Label` via `badge.Label(status)` with a
  `!ok → label = status` fallback that is defensive-only (`hubStatus` only ever emits valid `labels` keys).
  Still only 3/5 statuses appear here (the store-provable subset); wiring the badge did NOT add the richer
  taxonomy — that is the pending `metrics.Registry` thread-through. `TestDashboardRendersEveryHub` asserts
  the badge markup (`class="hub-status-badge"` + `>Verified<`/`M8.4 12.3` + `>Frozen<`/`M8.2 3.3h7.6`),
  so a regression to `{{.Status}}` fails the test (reviewer mutation-confirmed).
