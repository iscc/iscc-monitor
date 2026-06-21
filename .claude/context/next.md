# Next Work Package

## Step: Thread the five-status overlay + badge partial into the log-browser status cell

## Advances
M-UI — Evidence Ledger frontend, Verify criterion:
> "`HubStatusBadge` renders **all five** statuses each with a distinct text label **and** a distinct
> inline-SVG silhouette (golden-tested per status)" — and the framing criterion that "every SSR surface
> (`/` realm index, hub dossier, log-browser record list, single record, certificate) returns
> `200 text/html`" rendering the badge consistently.

This is also the explicit `review` handoff `**Next:**`: "Wire the same `StatusSource`-interface +
overlay shape into the per-hub log-browser status cell (`proofserve.hubStatus`) … so the richer
taxonomy is consistent across all M-UI surfaces." The `/` dashboard already renders all five statuses
via `dashboard.overlayStatus` + the `hubStatusBadge` partial; the log browser (`GET /<domain>/log/`)
still renders a bare `{{.Status}}` from the 2-element store subset (`frozen`/`verified` only). This
step makes the log browser's status cell render the same five-status badge honestly, closing the
per-surface consistency gap that keeps M-UI from being uniform.

## Goal
Render the log-browser (`serveBrowser`) hub-status cell through the `hubStatusBadge` partial, overlaid
with the in-memory `metrics.Registry` verdict so `unresolvable`/`unverified` show alongside the
store-provable `frozen`/`verified` — reusing the exact `StatusSource` interface + `overlayStatus`
precedence shape the dashboard already uses. After this step both `/` and `/<domain>/log/` render the
identical five-status badge taxonomy.

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test/doc source files):
  - `internal/proofserve/handler.go` — add a `StatusSource` interface (`Status(hubID int64) (string, bool)`),
    an `overlayStatus(fs, hubID, statuses)` helper mirroring `dashboard.overlayStatus` precedence, thread
    a `statuses StatusSource` param through `Handler` → `serveBrowser`, precompute the badge label via
    `badge.Label`, associate `badge.Source` into `browserTmpl`, and extend `browserData` with `Label`.
  - `internal/proofserve/browser.html` — render the status through `{{template "hubStatusBadge" .}}` in
    both the has-checkpoint table cell and the no-checkpoint state, instead of bare `{{.Status}}`.
  - `cmd/iscc-monitor/main.go` — thread the metrics registry `m` through `mirrorHandler` → `hubHandler`
    → `proofserve.Handler(st, hubID, m)` so the log browser gets the same overlay source `/` gets.
- **Modify (doc, not counted against the 3-file limit)**:
  - `CLAUDE.md` — update the `GET /<domain>/log/` bullet (lines 58-59) to note the status renders via the
    five-status `HubStatusBadge` (store-provable subset + in-memory overlay), matching the `GET /` line.
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` — the landed `overlayStatus` +
    precedence rule to mirror (durable `inactive`/`frozen` win; overlay only over store-`verified`;
    adopt only `unresolvable`/`unverified`; nil-tolerant).
  - `/workspace/iscc-monitor/.claude/context/learnings/badge.md` — the `Render`/`Source`/`PartialName`/
    `Label` surface and the "parent row MUST carry a precomputed `.Label`" + fail-closed contract;
    silhouette markers (`unresolvable` `M9.2 9.3`, `unverified` `M12 3.4 21 19H3z`).
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — the `serveBrowser` posture
    (store-provable subset, render-into-buffer-then-200, oracle gate N/A) this step extends.
  - `/workspace/iscc-monitor/internal/dashboard/handler.go` — the canonical `StatusSource` +
    `overlayStatus` + `buildRows` shape to port (replicate the small shape locally; do NOT import
    `internal/dashboard`).
  - `/workspace/iscc-monitor/internal/metrics/metrics.go` (`func (r *Registry) Status`) — the structural
    satisfier already used by the dashboard.

## Not In Scope
- DS v2 tokens / self-hosted fonts / CSS — the badge stays the existing token-free SVG partial; no
  styling work in this step.
- The hub dossier, paginated record list, single-record page, certificate, proof-bundle assembler, or
  anchor panels — each is its own later M-UI step.
- Adding `unresolvable`/`unverified`/`inactive` to `proofserve.hubStatus` or `store.ListHubs` — those
  are NOT store-provable; the live verdicts come ONLY from the `StatusSource` overlay (do not regress
  the store-leaf boundary). `inactive` is unreachable here (`serveBrowser` reads `FollowState`, not the
  realm-active flag); the overlay only adds `unresolvable`/`unverified` over store-`verified`, exactly
  as the dashboard does over its `verified` subset.
- Importing `internal/metrics` into `internal/proofserve` — depend on the local `StatusSource`
  interface only, so proofserve does not gain a metrics import and stays interface-typed.
- Changing `internal/badge` or the badge labels/silhouettes — all five already exist there.

## Implementation Notes
- **Port the precedence verbatim from `dashboard.overlayStatus`** (handler.go:152-172): overlay applies
  ONLY when the store status is `verified`, adopts ONLY `unresolvable`/`unverified`, and is nil-tolerant
  (`statuses == nil` → keep the store status). Here the store status comes from the existing
  `hubStatus(fs store.FollowState)` (`frozen` else `verified`) — there is no `inactive` case in
  `serveBrowser` (it reads `FollowState`, not the realm-active flag), so the input subset is just
  `frozen`/`verified`, and the overlay can only ever flip `verified` → `unresolvable`/`unverified`.
  Signature suggestion: `overlayStatus(fs store.FollowState, hubID int64, statuses StatusSource) string`.
- **Define a local `StatusSource` interface in `proofserve`** identical to `dashboard.StatusSource`
  (`Status(hubID int64) (string, bool)`). `*metrics.Registry` satisfies it structurally (already proven
  for the dashboard). Do NOT import `internal/dashboard` or `internal/metrics` — keep proofserve's
  closure free of a metrics dependency, mirroring the dashboard's design.
- **Associate the badge partial into `browserTmpl`** the same way `dashboard` does (handler.go:46-49):
  wrap the parse in an init closure since `template.Must` returns one value —
  `t := template.Must(template.New("browser").Parse(browserSource)); return template.Must(t.Parse(badge.Source))`.
  Import `github.com/iscc/iscc-monitor/internal/badge`.
- **`browserData` must carry a precomputed `.Label`** — the `hubStatusBadge` partial reads `.Label`
  directly (it does NOT re-derive from `.Status`), per `learnings/badge.md`. Compute it with
  `label, ok := badge.Label(status); if !ok { label = status }` (the `!ok` fallback is defensive-only;
  `overlayStatus` only ever yields valid `labels` keys). Set both `Status` and `Label` on `browserData`.
- **`Handler` signature change ripples to every call of `proofserve.Handler(st, hubID)`** — grep the
  package tests first: `grep -rn "Handler(" internal/proofserve/*_test.go` (browser/inclusion/
  consistency/entries/verify tests all construct it). Add the new `statuses` arg (pass `nil` where the
  overlay is irrelevant — the proof/checkpoint tests). `go build` will catch any straggler.
- **In `main.go`**: `mirrorHandler(st, routes)` and `hubHandler(st, hubID)` must each take and forward
  `m *metrics.Registry`; `buildMux` already holds `m` (it passes it to `dashboard.Handler`/`metricshttp`).
  Update the `mirrorHandler(st, routes)` call in `buildMux` and the `hubHandler(st, r.HubID)` call inside
  `mirrorHandler` accordingly, then `proofserve.Handler(st, hubID, m)` in `hubHandler`. `m` is already
  `*metrics.Registry` at that seam and satisfies the proofserve `StatusSource` interface structurally.
- **Render the badge in BOTH branches of `browser.html`**: the has-checkpoint `<td>{{.Status}}</td>`
  becomes `<td>{{template "hubStatusBadge" .}}</td>`, and the no-checkpoint `<p>Status: {{.Status}}</p>`
  becomes a badge invocation too (so an unpolled hub that is `unresolvable` shows the honest badge, not
  a bare "verified" — consistent with ADR-0001 coverage honesty). The `browserData` value passed to
  `Execute` exposes `.Status` + `.Label`, exactly what the partial needs.
- **Correctness rule (learnings index):** the relevant cross-cutting rule is "did:web is the only key
  source (ADR-0009). Resolution failure → `unresolvable` (keep mirroring). A signature matching no
  listed key → `unverified`." This step makes those two glossary states visible on the log browser too.
  Preserve the dashboard's package-local precedence rule exactly (durable `frozen` must win; overlay
  only over store-`verified`; adopt only `unresolvable`/`unverified`). Oracle/conformance gate is N/A
  (pure HTML composition of persisted rows + an in-memory status overlay; no signature/RFC-6962/Merkle/
  did:web/fsck/proof path) — `go.mod`/`go.sum`/`schema.sql` must stay byte-identical.
- Keep `serveBrowser`'s buffer-first render (`Execute(&buf, …)` → 500-before-200), the method gate, and
  the no-checkpoint 200 honesty branch untouched — they are correctness load-bearing.

## Verification
- `mise run check` is green (build + vet + test for all packages; `gofmt -l .` empty).
- `go test -count=1 ./internal/proofserve ./cmd/iscc-monitor` passes (uncached).
- `go test -run TestBrowser ./internal/proofserve` passes — the existing log-browser tests still pass
  with the new badge markup.
- A new (or extended) HTTP-seam test drives `serveBrowser` (via `Handler`) with a fake `StatusSource`
  returning `"unresolvable"` for a store-`verified` hub and asserts the rendered body carries the badge
  markup (`class="hub-status-badge"`, `data-status="unresolvable"`, `>Unresolvable<`, and the
  unresolvable silhouette marker `M9.2 9.3`) — proving the overlay reaches the page (mirror
  `dashboard.TestDashboardRendersInMemoryStatus`).
- Mutation check (run then revert): reverting `serveBrowser` to use `hubStatus(fs)` instead of the
  overlay makes the new overlay test FAIL (renders `data-status="verified"`); restored → PASS.
- `go list -deps ./internal/proofserve | grep -E 'internal/metrics|internal/dashboard'` is empty
  (proofserve depends on the local interface, not the concrete metrics/dashboard packages).
- `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` is empty (store stays a leaf).
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` is empty (no dep/schema change).

## Done When
`mise run check` is green and the log-browser page (`GET /<domain>/log/`) renders the hub status through
the five-status `hubStatusBadge` partial overlaid with the in-memory verdict — golden-asserted at the
HTTP seam for an `unresolvable` store-`verified` hub and mutation-proven non-vacuous — with proofserve
holding no `internal/metrics`/`internal/dashboard` import and the store still a leaf.
