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
- settled: `inactive` is unreachable through the public store API (no `SetActive` writer; `UpsertHub`
  inserts schema default `active=1`), so it is covered only by a white-box table test on package-private
  `hubStatus` (hence `package dashboard`, not `dashboard_test`). When a registry-deactivation writer lands,
  add an end-to-end inactive-render assertion through the public surface.
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
- settled: the frozen dossier (`internal/dossier`) adds a non-dismissable Exhibit `<section>` ON TOP OF the
  badge, gated `{{if .Frozen}}` (markup-distinct per ADR-0010, no `<button>`/`<script>`/` hidden` attr). The
  `ListViolations` read stays off the hot path (only a frozen hub queries it; safe because `overlayStatus`
  never downgrades `frozen`; a frozen-with-zero-rows hub still renders the panel header via `{{else}}`).
- **The `/` named-region parity landed: claim-lookup hero + per-row dossier `<a>` + masthead identity.**
  The hero is a no-JS `<form method="get" action="/inclusion/">` with `<input name="iscc_id">` — a
  `method=get` form can ONLY emit a query string, so the certificate handler gained the symmetric
  fallback (use `?iscc_id=` when the path id is empty) for the chain to resolve; keep the two in lockstep
  if you move either. Each data row is now `<a class="ledger-row" href="/{{.Domain}}">` (Navigation
  closure); wrapping a `display:grid` row in an `<a>` is valid HTML5 ONLY because the `hubStatusBadge`
  partial carries no nested interactive element (no `<a>`/`<button>`) — re-check that invariant before
  adding any control inside a row. `TestDashboardRendersHeroAndNavigation` pins all three regions and
  counts dossier links specifically (`href="/<domain>"`, not bare `<a `) so the masthead verify link +
  hero button don't inflate the count; mutation-proven (drop the row `<a>` → count 0; drop the hero
  `<section>` → `<form` missing).
- **The no-CDN ban is now narrowed to third-party hosts (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`), NOT a
  blanket `https://` ban** — the masthead's intentional `https://monitor.iscc.codes/` tier-2 verify link
  must pass, so the test now ALSO positively asserts `monitor.iscc.codes` is present. This mirrors the
  certificate's already-merged posture and `web.md`'s `noExternalCDN` rule (third-party origins only).
  The load-bearing rule (no third-party CDN origin) still holds; this is the precedent, not a weakening.
  Mockup deviations accepted as constraint-wins (all flagged): "recent declarers"
  hero footer omitted (no store history). (Logo + config-driven instance identity have since landed — see
  the instance-identity bullet below.) The `#` row number is `RowNo = fmt.Sprintf("%02d", i+1)` on the
  view-model — presentation only, no store value; the new `34px` grid column is fixed-width mono and does
  not ellipsize (the `min-width:0` trap applies only to `.hub-cell`).
- **The Checkpoint + Anchor columns LANDED (six-column grid `34px 1.8fr 1.2fr 1fr 1.1fr 150px`).**
  Checkpoint = `s.LastSize` (a relabel of the old "Observed size" cell, NOT a new read). Anchor is a NEW
  `store.HubSummary.Anchor` projection: a correlated subselect `(SELECT o.status FROM ots o WHERE
  o.hub_id=h.hub_id ORDER BY o.stamped_at DESC, o.id DESC LIMIT 1)` → the hub's LATEST-STAMPED-root OTS
  status, NULL→"". `anchorLabel` (handler.go) maps it via the `store.OTSStatus*` consts (not literals) to
  label+dot keyword; an unknown/empty value renders the honest "not anchored"/no-dot. The dot is a
  decorative inline `<span data-anchor>`-keyed DS-token color with a literal-hue fallback (no `<img>`,
  no-CDN); the LABEL is the grayscale-safe load-bearing signal (ADR-0010 inv.4). Both halves
  mutation-proven (reviewer reran: subselect→`''` FAILS the store anchor case; drop the data cell FAILS
  the dashboard render). Visual pass vs the mockup: column ORDER + naming match exactly.
- **The realm-index Anchor column is a per-HUB anchoring-activity indicator, NOT a per-checkpoint
  attestation — by design and by mockup (`anchorState` is a free-standing hub property).** It is
  DECOUPLED from the displayed Checkpoint (`f.last_size`): the newest-stamped OTS row may describe an
  OLDER root than the accepted size, and because OTS is async/best-effort (ADR-0004, never blocks the
  poll), that is the NORMAL steady state — the displayed checkpoint is almost always ahead of the latest
  CONFIRMED anchor. So "confirmed" here means "this hub anchors its roots", never "size N is
  Bitcoin-confirmed". The AUTHORITATIVE per-checkpoint claim lives in **certificate §5**, which binds
  `OTSForRoot(hubID, treeSize, root)` to the §2 accepted root via `ots.ConfirmedFor`. Do NOT "fix" the
  realm-index subselect to `o.tree_size = f.last_size` (Codex's P2 suggestion) without a design pass — it
  would render "not anchored" for virtually every actively-polling hub and defeat the column's purpose.
  Open `normal` records the design question.
- **The dossier masthead chrome is a VERBATIM port of `certificate/cert.html` — keep the two byte-identical.**
  `dossier.html` carries the same `.chrome-actions`/`.chrome-instance`/`.chrome-verify` CSS + `<div
  class="chrome-actions">` (static `monitor instance` label + `verify ↗ monitor.iscc.codes` tier-2 link to
  `https://monitor.iscc.codes/`) and the `.backlink-row`/`.backlink` `← Realm index` link (`href="/"`) the
  certificate ships. The dossier's tier-2 affordance is correctly the cross-surface link to Surface C (the
  `.codes` verifier app), NOT a baked-in WASM proof island — it has no single ISCC-ID subject to re-verify.
  The dossier no-CDN ban is the same narrowed list (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://`, NOT a
  blanket `https://`) so `monitor.iscc.codes` passes; `TestDossierChromeTierTwoAndBackLink` pins all three
  affordances (mutation-proven: change the back-link copy → FAIL). Visual pass vs the dossier mockup: chrome,
  back-link, tier-2 chip all match named regions; the only deltas are already-filed (static instance identity
  vs config-driven; mockup Checkpoint/Anchor columns — those belong to the `/` realm-index issue). If you edit
  either masthead, mirror the change in both `cert.html` and `dossier.html`.
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
- **Config-driven masthead identity LANDED on `/` (the `dashboard.Identity` view value).** Three
  operator-supplied strings flow env → binary → page: `Identity{Instance, Operator, Realm}`, passed as the
  3rd arg to `Handler(st, statuses, id)`. **The fail-safe fallback lives in `Identity.resolve()` INSIDE the
  package (not main.go)** — a blank `Instance`/`Operator` falls back to the `instanceFallback`/`operatorFallback`
  consts (today's static copy), a blank `Realm` stays "" so the template's `Realm register{{if .Realm}} · {{.Realm}}{{end}}`
  renders the bare subtitle with NO trailing separator. Keeping the default in the handler is what makes the
  fallback golden-testable at the HTTP seam regardless of env. `TestDashboardRendersInstanceIdentity` pins
  BOTH the populated path (exact operator strings present, static placeholder ABSENT) and the zero-value
  fallback path; reviewer mutation-confirmed non-vacuous on BOTH the template binding (`{{.Instance}}`→literal
  FAILS) AND the wiring (`resolve` ignoring the supplied value FAILS). Visual pass: live binary renders
  `monitor.iscc.id` / `instance operated by ISCC Foundation · ISCC mainnet` / `REALM REGISTER · ISCC MAINNET`
  exactly matching the Realm-Index mockup.
- **The realm-name env var is `ISCC_MONITOR_REALM_NAME`, NOT `ISCC_MONITOR_REALM`.** `ISCC_MONITOR_REALM`
  is ALREADY the REQUIRED realm-document filesystem PATH in `internal/config` — overloading it would leak a
  filename (`…/realm.txt`) into the ledger subtitle. The masthead needs the human realm NAME, so it uses a
  distinct optional key. `ISCC_MONITOR_INSTANCE`/`ISCC_MONITOR_OPERATOR` are genuinely new. These three keys
  are read inline in `cmd/iscc-monitor/main.go` `identity()` (deferred from `internal/config` for the ≤3-file
  budget); the follow-on sub-step that threads identity to the other five SSR mastheads SHOULD move parsing
  into the config leaf — adopt `ISCC_MONITOR_REALM_NAME` (or finalize the name) there. Reuse the SAME
  `dashboard.Identity` value across dossier/cert/proofserve mastheads (the dossier+cert mastheads are
  byte-identical ports — keep them in lockstep).
