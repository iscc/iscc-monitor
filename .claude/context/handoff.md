## 2026-06-21 — Wire `HubStatusBadge` into the `/` dashboard

**Done:** Replaced the bare `{{.Status}}` text cell on the `/` dashboard with the
`hubStatusBadge` partial, so every hub now renders its status as icon + label +
silhouette (grayscale-safe) for the three store-provable statuses
(`verified`/`frozen`/`inactive`). Added a thin exported `badge.Label(status)
(string, bool)` reading the existing fixed `labels` table so the dashboard row
precomputes its `.Label` from the single source of truth, preserving the
fail-closed label contract. This is the first wiring of the `internal/badge` leaf
into an HTTP surface.

**Files changed:**
- `internal/badge/badge.go`: added exported `Label(status) (string, bool)` accessor
  over the unexported `labels` map. `Render`, `Source`, `PartialName`, `labels`,
  and `view` are unchanged; package stays a pure WASM-shareable leaf.
- `internal/dashboard/handler.go`: import `internal/badge`; associate
  `badge.Source` into the dashboard template set (`template.New("dashboard")` →
  `t.Parse(badge.Source)`); add a `Label string` field to the `row` view-model;
  populate it in `buildRows` from `badge.Label(status)` with a defensive fallback
  to the status string if `ok==false` (never an unlabeled badge).
- `internal/dashboard/dashboard.html`: `<td>{{.Status}}</td>` →
  `<td>{{template "hubStatusBadge" .}}</td>`.
- `internal/badge/badge_test.go`: added `TestLabelMatchesFixedTable` — direct unit
  test of the new `Label` accessor (each status → its fixed-table label with
  `ok==true`; unknown/empty → `("", false)`, fail-closed).
- `internal/dashboard/handler_test.go`: extended `TestDashboardRendersEveryHub` to
  assert the badge markup renders — `class="hub-status-badge"` wrapper,
  `data-status="verified"`/`>Verified<`/`M8.4 12.3` and
  `data-status="frozen"`/`>Frozen<`/`M8.2 3.3h7.6` (badge + label + distinguishing
  silhouette per store-provable hub). The pre-existing domain/origin/coverage
  assertions are unchanged.

**Verification:** `mise run check` → green (all 17 packages `ok`, build + vet +
test). Per-criterion:
- [x] `mise run check` green.
- [x] `gofmt -l internal/badge internal/dashboard` empty.
- [x] `go test -count=1 ./internal/badge ./internal/dashboard` PASS (uncached).
- [x] `go test -run TestDashboardRendersEveryHub ./internal/dashboard` PASS; body
  contains `class="hub-status-badge"`, verified marker `M8.4 12.3`, frozen marker
  `M8.2 3.3h7.6`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/badge` OK (badge stays a
  WASM-shareable leaf after adding `Label`).
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/badge'`
  empty (store remains a leaf).
- [x] `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` empty (no
  dep/schema change).
- [x] Mutation-proven non-vacuous (my own run, reverted): reverting the html cell
  to `{{.Status}}` FAILs `TestDashboardRendersEveryHub`; restored → green and tree
  byte-clean.

**Next:** Make the full five-status taxonomy store-provable on `/` — thread the
in-memory `metrics.Registry` statuses (`unresolvable`/`unverified`, with
`follower.glossaryStatus` folding `rotated → unverified`) through to the dashboard
so all five badges can appear honestly. Per learnings/dashboard.md that thread-
through belongs at the registry boundary, NOT in `store.ListHubs` or
`dashboard.hubStatus`. After that, wire the badge into the log browser / dossier /
certificate surfaces (later M-UI sub-steps).

**Notes:**
- Scope clean: 2 non-test source files + 1 html + 1 test (≤3 source excluding
  tests/docs). The page still honestly shows only the 3 store-provable statuses
  (next.md Not-In-Scope); no claim of five statuses on `/`.
- Used a small init-time closure for `tmpl` so both `Parse` calls share one set
  (`template.Must(template.New("dashboard").Parse(pageTemplate))` then
  `template.Must(t.Parse(badge.Source))`), exactly the associated-template idiom
  proven by `badge.TestPartialComposesIntoParent`. Buffer-first render and the
  method/exact-path guards are untouched (correctness load-bearing).
- `badge.Label` always returns `ok==true` for the dashboard's outputs (all three
  are valid `labels` keys); the `!ok` fallback in `buildRows` is defensive only. The
  accessor itself is directly unit-tested by `TestLabelMatchesFixedTable` (hit +
  fail-closed miss); its dashboard use is also exercised by the golden test.
- Oracle/conformance gate is N/A (pure HTML composition; no
  signature/RFC-6962/Merkle/did:web/fsck/proof path; `go.mod`/`go.sum`/`schema.sql`
  byte-identical — verified).
- Pre-existing uncommitted context-file edits from `define-next` (`target.md`,
  plan, prd, `CLAUDE.md`, new `ADR-0010`) are left untouched per protocol — not
  mine to commit.
