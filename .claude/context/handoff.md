## 2026-06-21 — Thread in-memory hub status into `/` so all five badges render honestly

**Done:** Added a read accessor `metrics.Status(hubID) (string, bool)` over the existing
`hubStatus` map, defined a `dashboard.StatusSource` interface the `*metrics.Registry` satisfies
structurally, and overlaid the live verdict onto the store-provable subset in `dashboard.buildRows`
so `/` now renders all five glossary statuses honestly — store-provable `verified`/`frozen`/`inactive`
plus in-memory `unresolvable`/`unverified`. The precedence is load-bearing: `inactive`/`frozen` are
durable truths that the overlay never overrides; the live verdict applies only to a store-`verified`
hub.

**Files changed:**
- `internal/metrics/metrics.go`: added `Status(hubID int64) (string, bool)` — RLock-guarded scan of
  `hubStatus` returning the entry whose value == 1 for the hub, `("", false)` on miss. Additive,
  stdlib-only, does not touch the render path.
- `internal/dashboard/handler.go`: added the `StatusSource` interface; `Handler` now takes
  `(st *store.Store, statuses StatusSource)`; new `overlayStatus(s, statuses)` applies the live verdict
  only when the store status is `verified` (and the live value is `unresolvable`/`unverified`); package
  doc updated. Nil `statuses` is tolerated (leaves every verified hub as `verified`).
- `cmd/iscc-monitor/main.go`: `dashboard.Handler(st)` → `dashboard.Handler(st, m)`; `buildMux` doc
  notes the metrics registry is now also the dashboard's status overlay.
- `internal/dashboard/handler_test.go` (test): added `fakeStatusSource`; new
  `TestDashboardRendersInMemoryStatus` (end-to-end overlay render at the `/` seam) and
  `TestOverlayStatusPrecedence` (precedence table); updated existing call sites to the new signature.
- `internal/metrics/metrics_test.go` (test): added `TestStatus` (hit, transition, miss, no cross-hub
  leak).
- `CLAUDE.md` (doc): updated the `GET /` bullet — the page now overlays the store subset with the live
  verdict so all five statuses render.

**Verification:** `mise run check` → all 17 packages `ok` (build + vet + test). Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/metrics ./internal/dashboard ./cmd/iscc-monitor` PASS (uncached).
- [x] `go test -run TestDashboardRenders ./internal/dashboard` PASS; body contains
  `data-status="unresolvable"` + `>Unresolvable<` + `M9.2 9.3` AND `data-status="unverified"` +
  `>Unverified<` + `M12 3.4 21 19H3z` (plus the existing `>Verified<`/`>Frozen<`).
- [x] `go test -run TestStatus ./internal/metrics` PASS.
- [x] Mutation check (my own run, reverted): bypassing the overlay
  (`status := overlayStatus(...)` → `status := hubStatus(s)`) makes `TestDashboardRendersInMemoryStatus`
  FAIL (renders `data-status="verified"` for the overlaid hubs); restored → PASS. Non-vacuous.
- [x] `GOOS=js GOARCH=wasm go build ./internal/metrics ./internal/badge` OK (metrics leaf stays
  WASM-shareable; the new reader added no import).
- [x] `go list -deps ./internal/dashboard | grep internal/metrics` empty (depends on the interface,
  not the concrete package).
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/badge|internal/metrics'`
  empty (store stays a leaf).
- [x] `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` empty.

**Next:** Wire the same overlay/badge pattern into the per-hub log-browser status cell
(`proofserve.hubStatus`) and the upcoming hub dossier, so the richer taxonomy is consistent across all
M-UI surfaces — the `StatusSource` interface + `badge.Label` precompute is the reusable pattern. The
`inactive` end-to-end render is still a table test only (no public `SetActive` writer / registry-
deactivation path exists yet — out of scope here, per next.md); add an end-to-end `inactive` assertion
when that writer lands.

**Notes:**
- Scope: 3 non-test source files (`metrics.go`, `dashboard/handler.go`, `main.go`) + 2 test files +
  CLAUDE.md doc — at the ≤3 source-file limit.
- Precedence rationale baked into `overlayStatus`: a fresher in-memory verdict is honest for a
  currently-failing did:web resolve (`unresolvable`) or a non-matching signature (`unverified`), but a
  freeze (ADR-0006, restart-surviving) and registry `inactive` are harder durable truths the live
  verdict must not override. Only `verified` is overlaid; any other live value (incl. `verified`
  itself or no record) keeps `verified`. Verified by `TestOverlayStatusPrecedence`.
- `StatusSource` is a deliberately tiny interface in `dashboard` so the package does NOT import
  `internal/metrics` (closure stays `bytes embed html/template net/http internal/store internal/badge`);
  `*metrics.Registry` satisfies it structurally via the new `Status` method.
- Oracle/conformance gate correctly N/A — pure data plumbing + HTML composition; no
  signature/RFC-6962/Merkle/did:web/fsck/proof path; go.mod/go.sum/schema.sql byte-identical.
- `metrics.Status` relies on `SetHubStatus`'s "exactly one status=1 per hub" invariant (learnings/
  metrics.md). If a future change lets two statuses read 1 for one hub, `Status` returns the first one
  map iteration finds — re-check that invariant before relaxing `SetHubStatus`.
- The `!ok → label = status` fallback in `buildRows` stays as the defensive backstop; both overlaid
  values come from the registry (filled by `follower.glossaryStatus`, always a valid `labels` key), so
  `ok` is always true on the page today.
