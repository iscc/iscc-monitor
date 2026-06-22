## 2026-06-22 — `/` realm-index: add the per-hub Anchor projection + render the mockup's Checkpoint & Anchor ledger columns

**Done:** Added a read-only `Anchor` projection (latest-stamped-root OTS status, via a correlated
`ots` subselect) to `store.HubSummary`/`ListHubs`, threaded `Checkpoint` (= accepted `LastSize`),
`Anchor`, and a presentation-only `AnchorDot` into the dashboard `row` view-model, and widened the
`/` ledger from five to the mockup's six columns (`# | Hub · domain | Coverage since | Checkpoint |
Anchor | Status`). The store stays a leaf; this closes sub-delta (3) of the open `normal` `/`
realm-index column-deltas issue (issues.md:214).

**Files changed:**
- `internal/store/hubs.go`: added `Anchor string` to `HubSummary`; extended `ListHubs` with a
  correlated subselect `(SELECT o.status FROM ots o WHERE o.hub_id=h.hub_id ORDER BY o.stamped_at
  DESC, o.id DESC LIMIT 1)` read through `sql.NullString` (NULL → "").
- `internal/dashboard/handler.go`: added `Checkpoint`/`Anchor`/`AnchorDot` to `row`; new
  `anchorLabel` helper maps `store.OTSStatusConfirmed`/`OTSStatusPending` (the consts, not literals)
  → display label + dot keyword, defaulting any other/empty value to the honest "not anchored"/"none".
- `internal/dashboard/dashboard.html`: both grids → `34px 1.8fr 1.2fr 1fr 1.1fr 150px`; in-template
  column comment synced; added `.anchor-cell`/`.anchor-dot` CSS (decorative inline `<span>` dot keyed
  on `data-anchor`, DS-token color with literal-hue fallback, no `<img>`/CDN); added Checkpoint +
  Anchor colhead `<span>`s and the two data cells between Coverage and Status.
- `internal/store/hubs_test.go` (new): `TestListHubsAnchorStatus` — confirmed/pending(newest-wins)/
  never-stamped(empty) Anchor cases through the public API.
- `internal/dashboard/handler_test.go`: `fixtureStore` seeds a confirmed `RecordOTS` row on the
  verified hub (frozen hub left unstamped); `TestDashboardRendersEveryHub` asserts both colheads,
  `data-anchor="confirmed"`/`>confirmed<`, and `data-anchor="none"`/`>not anchored<`.

**Verification:** `mise run check` → green (build + vet + test, all 28 packages ok; dashboard + store
ran uncached). `gofmt -l .` empty outside `cauldron/`.
- `go test -run TestListHubs ./internal/store` → PASS (confirmed/pending/empty Anchor).
- `go test -run TestDashboard ./internal/dashboard` → PASS (six-column grid, confirmed + not-anchored
  cells, plus the pre-existing hero/overlay/no-CDN/method/path assertions).
- Leaf invariant: `go list -deps ./internal/store | grep '^net/http'` empty; internal iscc-monitor
  deps are only the pre-existing `internal/tiles` + the package itself (no `logclient`/`dashboard`).
- Mutation (both reverted byte-clean): dropping the `.anchor-cell` data cell from `dashboard.html`
  → `TestDashboardRendersEveryHub` FAILS the anchor assertion; reverting the `ots` subselect to a
  literal `''` → `TestListHubsAnchorStatus` FAILS (`Anchor = "" want "confirmed"`/`"pending"`).
- Oracle/conformance gate N/A — pure HTML render of a persisted leaf read; touches no
  signature/RFC-6962/Merkle/`proof`/`didweb`/fsck path. schema.sql/go.mod/go.sum byte-unchanged.

**Next:** Of the same `normal` (issues.md:214), the remaining slices are NOT pure code: sub-delta (2)
config-driven instance identity / realm name needs `internal/config` env wiring (a separate step);
sub-delta (4) "recent declarers checked" hero footer needs a recent-lookup history the store does not
track (out of scope). Both were deferred in this step's Not-In-Scope. A reasonable next pick is the
config-driven masthead identity (sub-delta 2) — it is self-contained but introduces new env config,
so it wants its own work package rather than riding a render slice.

**Notes:**
- The mockup's `coverageSince` cell shows date-then-size (`{{coverageSince}}` over `@ {{size}}`); the
  live page keeps its existing size-then-time order. That ordering delta was already present before
  this step and is outside scope (this step touched only the Checkpoint/Anchor columns) — not
  re-litigated here.
- `TestListHubsAnchorStatus` is a NEW store test (no prior `TestListHubs` existed; `ListHubs` was only
  exercised through the dashboard HTTP seam). `next.md`'s `-run TestListHubs` matches it by prefix.
- The Anchor humanization ("not anchored" vs the mockup's dot-only treatment of the empty state) is a
  deliberate coverage-honesty choice: a never-stamped hub must read as not-anchored, never as a silent
  confirmed-style green. The dot is decorative; the text label is the load-bearing grayscale-safe
  signal (ADR-0010 inv.4), matching the frozen-row tint precedent.
- No new migration: `Anchor` is a read-only projection off the existing `ots` table (no new column),
  so the no-migration `normal` is not tripped (next.md Not-In-Scope; store.md migration note).
