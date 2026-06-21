# Next Work Package

## Step: Thread the in-memory hub status into `/` so all five badges can render honestly

## Advances
M-UI (Evidence Ledger frontend, ADR-0010) Verify criterion:

> "`HubStatusBadge` renders **all five** statuses each with a distinct text label **and** a distinct
> inline-SVG silhouette (golden-tested per status) … Make the full five-status taxonomy
> (`verified`/`unresolvable`/`unverified`/`frozen`/`inactive`) store-provable so the badge renders it
> honestly."

This is the review handoff's explicit `**Next:**`: the `HubStatusBadge` partial is wired into `/` but
only 3/5 statuses (`verified`/`frozen`/`inactive`) can appear, because `unresolvable`/`unverified` live
only in the in-memory `metrics.Registry` and are not threaded to the dashboard. Closing this is the
nearest unmet M-UI criterion (the per-status golden — all five rendered honestly across `/`).

## Goal
Give `internal/dashboard` a way to read each hub's current in-memory glossary status from the
`metrics.Registry` and overlay it onto the store-provable subset, so `/` renders `unresolvable` and
`unverified` honestly (in addition to `verified`/`frozen`/`inactive`) — completing the five-status
taxonomy at the `/` HTTP seam.

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test source files):
  - `internal/metrics/metrics.go` — add a read accessor `Status(hubID int64) (string, bool)` returning
    the single status currently set to 1 for that hub (RLock-guarded). This is the in-memory-status
    reader the dashboard consumes; it is additive and does not touch the Prometheus render path.
  - `internal/dashboard/handler.go` — define a small `StatusSource` interface
    (`Status(hubID int64) (string, bool)`), accept it in `Handler(st *store.Store, statuses StatusSource)`,
    and overlay the in-memory verdict in `buildRows`: store `inactive`/`frozen` still win; otherwise
    consult `statuses.Status(s.HubID)` and use `unresolvable`/`unverified` when reported.
  - `cmd/iscc-monitor/main.go` — pass the existing `m` (`*metrics.Registry`) into the dashboard:
    `buildMux(st, routes, m)` already has `m`; change `mux.Handle("/", dashboard.Handler(st))` →
    `dashboard.Handler(st, m)`.
- **Modify (test/doc, not counted):**
  - `internal/dashboard/handler_test.go` — extend `TestDashboardRendersEveryHub` (or add a sibling) to
    drive a hub whose in-memory status is `unresolvable`/`unverified` via a tiny fake `StatusSource`,
    asserting the badge markup + per-status silhouette marker appears. Update existing call sites to the
    new `Handler` signature (pass a fake or a nil-tolerant source).
  - `internal/metrics/metrics_test.go` — add a focused `Status` accessor test (hit + miss).
  - `CLAUDE.md` "Running a local dev instance" `GET /` bullet — update only if the visible
    status-coverage wording shifts (it now shows the full five-status taxonomy).

- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/context/learnings/dashboard.md` (status-subset rule + "richer
    statuses belong at the thread-through, not in `ListHubs`/`hubStatus`")
  - `/workspace/iscc-monitor/.claude/context/learnings/badge.md` (five labels already present; markers
    `M9.2 9.3` = unresolvable, `M12 3.4 21 19H3z` = unverified)
  - `/workspace/iscc-monitor/.claude/context/learnings/metrics.md` (the leaf is stdlib-only/WASM-pure;
    `SetHubStatus` keeps exactly one status=1 per hub — the reader relies on that invariant)
  - `/workspace/iscc-monitor/internal/follower/follower.go` `glossaryStatus` (the canonical
    store→glossary fold the registry already holds; folds `rotated → unverified`, `frozen` overrides)
  - `/workspace/iscc-monitor/internal/dashboard/handler.go`,
    `/workspace/iscc-monitor/internal/dashboard/handler_test.go`,
    `/workspace/iscc-monitor/internal/metrics/metrics.go`,
    `/workspace/iscc-monitor/cmd/iscc-monitor/main.go`

## Not In Scope
- DS v2 tokens / self-hosted fonts / any CSS (separate M-UI sub-step).
- Hub dossier, record list, single-record page, certificate, proof-bundle assembler.
- A registry-deactivation writer / public `SetActive` (the `inactive` end-to-end render stays a table
  test until that lands — do not add it here).
- Changing `proofserve.hubStatus` or the log-browser status cell (a later wire-the-badge sub-step).
- Adding `unresolvable`/`unverified` columns to `store.HubSummary` or `ListHubs` — the store cannot
  prove those; they MUST come from the in-memory `metrics.Registry`, per learnings/dashboard.md.
- Touching the metrics Prometheus render path / golden output (the new method is read-only, additive).
- Changing `internal/badge` — all five labels + silhouettes already exist there.

## Implementation Notes
- **Precedence is the load-bearing logic.** In `buildRows`, compute `status := hubStatus(s)` first.
  Keep `inactive` and `frozen` exactly as today (store/registry truth; `frozen` is restart-surviving
  ADR-0006 evidence that must win over any fresher in-memory verdict). ONLY when the store status is
  `verified` should you consult the `StatusSource`: if `statuses.Status(s.HubID)` returns
  (`unresolvable` or `unverified`, ok=true) use it; otherwise keep `verified`. Rationale: a hub with an
  old accepted checkpoint but a currently-failing did:web resolve is honestly `unresolvable` now — the
  in-memory poll verdict is fresher than the store's `verified` flag, but a freeze/inactive is a harder,
  durable truth that the live verdict must not override.
- **The reader (`metrics.Status`).** Iterate `r.hubStatus` under `r.mu.RLock()`, return
  `(k.status, true)` for the entry whose value == 1 and `k.hubID == hubID`; return `("", false)` if no
  status=1 entry exists for that hub. `SetHubStatus` guarantees at most one active per hub (it zeroes
  the others), so the first match is unambiguous. Keep the method stdlib-only — do NOT pull any new
  import; the leaf must stay WASM-pure (learnings/metrics.md).
- **Interface, not concrete dep.** Define `StatusSource` in `dashboard` and have `Handler` take it, so
  `internal/dashboard` does NOT import `internal/metrics` (keeps the closure
  `bytes embed html/template net/http internal/store internal/badge` and the dashboard golden-testable
  with a fake). `*metrics.Registry` satisfies it structurally via the new `Status` method.
- **Badge labels already exist** for all five statuses (`badge.Label` table has
  `unresolvable`/`unverified`); the partial already renders their silhouettes. So once `buildRows`
  emits those status strings, `badge.Label(status)` resolves and the partial renders the correct
  label + silhouette with no badge change. Do NOT modify `internal/badge`.
- **Status validity:** the overlaid value comes from the registry, which the follower fills via
  `glossaryStatus` (always a valid `labels` key). The existing `!ok → label = status` fallback in
  `buildRows` stays as the defensive backstop.
- Keep the buffer-first render (`tmpl.Execute(&buf, …)` → 500-before-200) and the exact-path/method
  guards untouched — they are correctness load-bearing (learnings/dashboard.md).
- Relevant correctness rule (learnings index): "did:web is the only key source (ADR-0009). Resolution
  failure → `unresolvable` (keep mirroring). A signature matching no listed key → `unverified`." This
  step makes those two glossary states visible on `/`.
- Oracle/conformance gate is N/A (pure data plumbing + HTML composition; no
  signature/RFC-6962/Merkle/did:web/fsck/proof path; `go.mod`/`go.sum`/`schema.sql` stay byte-identical).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/metrics ./internal/dashboard ./cmd/iscc-monitor` passes (uncached).
- `go test -run TestDashboardRendersEveryHub ./internal/dashboard` passes and the rendered body contains
  the unresolvable OR unverified badge markup — at least one of `>Unresolvable<` + `M9.2 9.3` or
  `>Unverified<` + `M12 3.4 21 19H3z` (in addition to the existing `>Verified<`/`>Frozen<` markers),
  proving an in-memory-only status renders honestly at the `/` seam. Mutation check (advance reverts to
  confirm non-vacuous): stubbing the overlay so the in-memory status is ignored must fail this test.
- `go test -run TestStatus ./internal/metrics` passes: `Status(hubID)` returns the active status for a
  hub set via `SetHubStatus`, and `("", false)` for an unknown hub.
- `GOOS=js GOARCH=wasm go build ./internal/metrics ./internal/badge` succeeds (the new reader keeps the
  metrics leaf WASM-shareable; badge unchanged).
- `go list -deps ./internal/dashboard | grep internal/metrics` is empty (dashboard depends on the
  `StatusSource` interface, not the concrete metrics package).
- `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/badge|internal/metrics'`
  is empty (store stays a leaf; no new store status column added).
- `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` is empty (no dep/schema change).

## Done When
`/` renders all five `HubStatusBadge` statuses honestly — store-provable `verified`/`frozen`/`inactive`
plus in-memory `unresolvable`/`unverified` overlaid via the new `metrics.Status` reader and
`dashboard.StatusSource` — with the per-status golden asserting at least one in-memory-only status at
the HTTP seam, and all Verification checks pass.
