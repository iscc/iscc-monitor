## 2026-06-21 — Thread the five-status overlay + badge partial into the log-browser status cell

**Done:** The log-browser page (`GET /<domain>/log/`) now renders its hub-status cell through the
five-status `hubStatusBadge` partial overlaid with the in-memory `metrics.Registry` verdict — reusing
the exact `StatusSource`-interface + `overlayStatus`-precedence shape the `/` dashboard uses — so both
surfaces render the identical five-status taxonomy honestly. Proofserve gained a package-local
`StatusSource` interface (no `internal/metrics`/`internal/dashboard` import); `main.go` threads the
metrics registry `m` through `mirrorHandler` → `hubHandler` → `proofserve.Handler`.

**Files changed:**
- `internal/proofserve/handler.go`: added a local `StatusSource` interface (`Status(hubID int64) (string, bool)`)
  and an `overlayStatus(fs, hubID, statuses)` helper mirroring `dashboard.overlayStatus` precedence
  verbatim (frozen wins; overlay only over store-`verified`; adopt only `unresolvable`/`unverified`;
  nil-tolerant); threaded `statuses StatusSource` through `Handler` → `serveBrowser`; precomputed
  `.Label` via `badge.Label`; associated `badge.Source` into `browserTmpl`; extended `browserData` with `Label`.
- `internal/proofserve/browser.html`: status now renders via `{{template "hubStatusBadge" .}}` in both
  the has-checkpoint table cell and the no-checkpoint `<p>` state (was bare `{{.Status}}`).
- `cmd/iscc-monitor/main.go`: `mirrorHandler` and `hubHandler` now take/forward `m *metrics.Registry`;
  `hubHandler` calls `proofserve.Handler(st, hubID, m)` so the browser gets the same overlay `/` gets.
- `internal/proofserve/browser_test.go`: added `fakeStatusSource` + `TestBrowserRendersInMemoryStatus`
  (overlay HTTP-seam render, asserts `class="hub-status-badge"`, `data-status="unresolvable"`,
  `>Unresolvable<`, marker `M9.2 9.3`, and that no `data-status="verified"` remains); existing 3 tests
  pass `nil`.
- `internal/proofserve/{consistency,entries,handler,verify}_test.go`: `Handler(...)` call sites pass `nil`
  (overlay irrelevant for proof/checkpoint tests).
- `CLAUDE.md`: `GET /<domain>/log/` bullet notes the status renders via the five-status `HubStatusBadge`
  (store-provable subset + in-memory overlay), matching the `GET /` line.

**Verification:** `mise run check` → green (all 17 packages `ok`, build + vet + test); `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/proofserve ./cmd/iscc-monitor` PASS (uncached).
- [x] `go test -run TestBrowser ./internal/proofserve` PASS — existing browser tests pass with new markup.
- [x] New `TestBrowserRendersInMemoryStatus` PASS — overlay reaches the page for an `unresolvable`
  store-`verified` hub (badge wrapper + label + silhouette marker; no residual `verified`).
- [x] Mutation (run then reverted): `overlayStatus(fs, hubID, statuses)` → `hubStatus(fs)` makes
  `TestBrowserRendersInMemoryStatus` FAIL (page renders bare `verified`); restored → PASS, tree byte-clean.
- [x] `go list -deps ./internal/proofserve | grep -E 'internal/metrics|internal/dashboard'` empty.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` empty (store stays a leaf).
- [x] `git diff --stat HEAD -- go.mod go.sum internal/store/schema.sql` empty (no dep/schema change).

**Next:** Wire the same `StatusSource`-overlay + `badge.Label`-precompute pattern into the upcoming hub
dossier and the paginated record-list / single-record pages so the five-status taxonomy stays uniform
across every M-UI surface. `inactive` remains unreachable in `serveBrowser` (it reads `FollowState`, not
the realm-active flag) and through the dashboard's public surface (no `SetActive` writer yet) — add an
end-to-end `inactive` render assertion once a registry-deactivation writer lands. The `out io.Writer`
notecheck item (issues.md, low) is still unrelated and open.

**Notes:**
- Scope: 3 non-test source files (`proofserve/handler.go`, `proofserve/browser.html`,
  `cmd/iscc-monitor/main.go`) + tests + `CLAUDE.md` doc — at the ≤3 limit. Nothing from `## Not In Scope`
  touched: no new status added to `proofserve.hubStatus`/`store.ListHubs`, no `internal/badge` change, no
  `internal/metrics` import in proofserve, no CSS/DS-token work, `serveBrowser`'s buffer-first render +
  method gate + no-checkpoint 200 branch left intact.
- Oracle/conformance gate is N/A (pure HTML composition of persisted rows + an in-memory status overlay;
  no signature/RFC-6962/Merkle/did:web/fsck/proof path). `go.mod`/`go.sum`/`schema.sql` byte-identical.
- The `overlayStatus` here is structurally identical to `dashboard.overlayStatus` but takes a
  `store.FollowState` + `hubID` (proofserve has no `HubSummary`), and the input subset is just
  `frozen`/`verified` (no `inactive` — `serveBrowser` reads `FollowState`), so the overlay can only ever
  flip `verified` → `unresolvable`/`unverified`. This is a deliberate small local replica per the plan
  (no `internal/dashboard` import), not a shared helper.
- Out-of-scope, NOT staged by me: the working tree carries a pre-existing modification to
  `.claude/agents/review.md` (review-agent doc) that I did not touch and will not commit — flagging it so
  `review` is aware it predates this increment.
