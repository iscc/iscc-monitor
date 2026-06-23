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
  richer live verdicts come from an OVERLAY at `buildRows`, never from `ListHubs`.** Do NOT add
  `unverified`/`unresolvable`/`rotated` to `hubStatus`/`ListHubs`: those are not store-provable.
  **Precedence is load-bearing and must not regress:** `overlayStatus(s, statuses StatusSource)` applies
  ONLY when `hubStatus(s) == "verified"`, adopts ONLY `unresolvable`/`unverified`, and is nil-tolerant
  (durable `inactive`/`frozen` must win). The `StatusSource` interface (`Status(hubID) (string,bool)`,
  satisfied structurally by `*metrics.Registry`) keeps the package off `internal/metrics`.
  `TestOverlayStatusPrecedence` + `TestDashboardRendersInMemoryStatus` pin it (mutation-confirmed).
  settled: this overlay shape is now copied verbatim in proofserve's log-browser cell AND the dossier (3x —
  the consolidation pressure is a filed `low`; see `learnings/http-surface.md`).
- settled: `inactive` is unreachable through the public store API (no `SetActive` writer), so it is
  white-box table-tested on package-private `hubStatus`; add an end-to-end inactive render when a
  registry-deactivation writer lands.
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
- settled: the frozen dossier adds a non-dismissable Exhibit `<section>` gated `{{if .Frozen}}`
  (markup-distinct per ADR-0010, no `<button>`/`<script>`); `ListViolations` stays off the hot path (only a
  frozen hub queries it; a frozen-with-zero-rows hub still renders the header via `{{else}}`).
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
- **The claim-lookup `.hero-input` is the coral "live ISCC code bar" — `background`+`border` both
  `var(--iscc-coral-red)` (#f56169), `color: var(--surface-card)` (white), `::placeholder` translucent
  white `rgba(255,255,255,.72)`.** Matches the `.dc.html` design source (`background:var(--iscc-coral-red)`,
  `color:#fff`, borderless). The border is set to the coral bg (not removed) on purpose — mirrors
  `.hero-submit`'s `border = bg` trick so the 1px box-model is preserved and the input stays height-aligned
  with the bordered submit button. Set as a direct human design tweak on 2026-06-23, OUTSIDE the loop; this
  is the ratified styling — do NOT revert it to the old `surface-page`/`text-body`/`border-default` neutral
  look. No test asserts the input CSS (tests pin only `name="iscc_id"` and the form structure).
- **The hero ships a "Recently declared:" shortcut row (`buildRecent` → `pageData.Recent`, rendered under
  `{{if .Recent}}`).** It is the mockup's "Recent declarers checked" footer reimagined with a twist: the
  store cannot track which ids were *checked* (no lookup history), but it CAN list what was recently
  *declared*, so the row links the single newest indexed declaration straight to its Certificate of
  Inclusion — one working example without typing. Data path: `store.RecentRecords(n)` (realm-wide,
  newest-first by the global seq, accepted-tree-bounded, SCHEMA-AGNOSTIC) → dashboard filters to
  `schemaDeclaration`, de-dupes by id, caps at `recentDisplay` (**1** — one example is enough; per Titusz),
  over-fetching `recentFetch` (60) so the filter can skip past deletions/dups to reach it. The cap is a
  one-line change to show more. The link is `/inclusion/{{.Body}}` (id with the `ISCC:` prefix STRIPPED so no colon enters the
  URL path — the cert handler re-adds it), link text shows the full `{{.IsccID}}`. Empty index → the whole
  row is omitted (honest empty state, no dangling label). Note: schema interpretation lives HERE in the
  view layer, not the store (ADR-0008 keeps the store schema-agnostic — the "projection" pattern); the
  `schemaDeclaration` URI is a 3rd literal copy (cert + proofserve have the other two — a documented
  shared-constant cleanup, see issues.md). Tests: `TestDashboardRecentlyDeclared` (order/dedup/filter/cap/
  link-form) + `TestDashboardNoRecentRowWhenIndexEmpty`. Set as a direct human design tweak 2026-06-23,
  OUTSIDE the loop — ratified, do not revert.
- **Hero copy + placeholder (same 2026-06-23 out-of-loop tweak):** the `<h1 class="hero-title">` reads
  **"ISCC-ID Verification"** (was "Prove a specific ISCC declaration is in the log."), and the coral
  `.hero-input::placeholder` is paler at `rgba(255,255,255,.5)` (was .72) so it reads clearly as a
  placeholder, not a filled value. The `.dc.html` design source was updated to match both the new h1 and
  the "Recently declared:" label. No test pins the h1 copy.
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
  Checkpoint = `s.LastSize` (a relabel, NOT a new read). Anchor is a NEW `store.HubSummary.Anchor`
  correlated subselect (`SELECT o.status … ORDER BY o.stamped_at DESC, o.id DESC LIMIT 1`, NULL→""),
  mapped by `anchorLabel` via `store.OTSStatus*` consts to label+decorative-dot (the LABEL is the
  grayscale-safe load-bearing signal, ADR-0010 inv.4; unknown/empty → honest "not anchored"). Both halves
  mutation-proven; visual pass: column order + naming match the mockup.
- **The realm-index Anchor column is a per-HUB anchoring-activity indicator, NOT a per-checkpoint
  attestation** (by design + mockup: `anchorState` is a free-standing hub property). It is DECOUPLED from
  the displayed Checkpoint (`f.last_size`) — the newest-stamped OTS row may describe an OLDER root, and
  since OTS is async/best-effort (ADR-0004) that is the NORMAL steady state. "confirmed" means "this hub
  anchors its roots", never "size N is Bitcoin-confirmed"; the AUTHORITATIVE per-checkpoint claim lives in
  **certificate §5** (`OTSForRoot` bound to the §2 accepted root). Do NOT "fix" the subselect to
  `o.tree_size = f.last_size` without a design pass (it would render "not anchored" for every actively-polling
  hub). Open `normal` records the design question.
- **All THREE SSR mastheads (dashboard/dossier/cert) are now byte-identical VERBATIM ports — edit all
  together (a masthead change is a three-site HTML edit + three const copies).** `dashboard.html`,
  `dossier.html`, AND `cert.html` share the same `.chrome-identity`/`.chrome-instance`/`.chrome-operator`
  CSS rule bodies AND the same `<div class="chrome-actions">` block (the two-line `chrome-identity`
  `{{.Instance}}`/`{{.Operator}}` div + `verify ↗ monitor.iscc.codes` tier-2 link to `https://monitor.iscc.codes/`).
  Each handler carries its OWN private `resolveIdentity` + `instanceFallback`/`operatorFallback` const copies
  (byte-identical literals, "MUST stay byte-identical" comment — neither package can import the other's
  unexported consts; consolidation into one shared `Resolve` leaf is the tracked `low`, fold once proofserve
  lands). Only the explanatory CSS comment differs per file (the dashboard's still says "static copy in this
  skeleton" — stale, flagged `low`; the dossier+cert copies are accurate). Same narrowed no-CDN ban
  (`jsdelivr`/`cdn.`/`unpkg`/`googleapis`/`http://`, NOT a blanket `https://`) so `monitor.iscc.codes` passes.
  Visual pass (cert, reviewer `3c64097`): the cert masthead renders the configured `monitor.iscc.id` /
  `instance operated by ISCC Foundation · ISCC mainnet` right-aligned beside the logo + tier-2 link, byte-identical
  to dashboard/dossier. STILL PENDING: the three proofserve mastheads (`browser.html`/`records.html`/`record.html`)
  — `internal/verifier` stays EXCLUDED (its `.codes` chrome is the verifier-app identity). If you edit any
  masthead, mirror it in all three SSR HTML files.
- **Config-driven masthead identity LANDED on the dossier too (`Handler(st, hubID, statuses, id dashboard.Identity)`).**
  The dossier REUSES `dashboard.Identity` (imports `internal/dashboard`) rather than redefining it, but applies its
  OWN private `resolveIdentity` fail-safe + dossier-local `instanceFallback`/`operatorFallback` consts (literal
  copies, byte-identical to dashboard's, with a comment that they MUST match — neither package can import the
  other's unexported consts). This dossier-side helper was chosen over exporting `dashboard.resolve` to keep the
  edit at ≤3 prod files. The dossier masthead has NO Realm slot (its title is the realm-subtitle-free "Hub
  dossier") — thread only `Instance`/`Operator`. `TestDossierRendersInstanceIdentity` pins both populated +
  zero-value paths; reviewer mutation-confirmed non-vacuous on BOTH the template `{{.Instance}}` binding AND the
  wiring (`resolveIdentity` dropping the supplied value → FAIL). The duplicated fallback consts are tracked `low`
  for consolidation when the masthead arc finishes across all surfaces (a shared identity-resolve leaf).
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
