## 2026-06-21 — Review of: Thread the five-status overlay + badge partial into the log-browser status cell

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` threaded the dashboard's `StatusSource`-interface + `overlayStatus`-precedence
shape into `proofserve.serveBrowser` so the log browser (`GET /<domain>/log/`) now renders its hub
status through the five-status `hubStatusBadge` partial overlaid with the in-memory verdict — both
`/` and `/<domain>/log/` now render the identical taxonomy honestly. The diff is scope-clean (3
non-test source files + tests + 1 doc, at the ≤3 limit), keeps `proofserve` free of any
`internal/metrics`/`internal/dashboard` import, store stays a leaf, and the new overlay test is
mutation-proven non-vacuous (I reverted it independently and re-confirmed). No defects found.

**Verification:**
- [x] `mise run check` green — all 17 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty (whole tree).
- [x] `go test -count=1 ./internal/proofserve ./cmd/iscc-monitor` PASS (uncached).
- [x] `go test -run TestBrowser ./internal/proofserve` PASS — existing browser tests pass with new markup.
- [x] `TestBrowserRendersInMemoryStatus` PASS — overlay reaches the page for an `unresolvable`
  store-`verified` hub (`class="hub-status-badge"`, `data-status="unresolvable"`, `>Unresolvable<`,
  `M9.2 9.3`; no residual `data-status="verified"`).
- [x] Mutation (my own run, reverted): `overlayStatus(fs, hubID, statuses)` → `hubStatus(fs)` makes
  `TestBrowserRendersInMemoryStatus` FAIL (page renders bare `verified`); restored → PASS, tree byte-clean.
- [x] `go list -deps ./internal/proofserve | grep -E 'internal/metrics|internal/dashboard'` empty —
  depends on the local interface, not the concrete packages.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/proofserve'` empty (store stays a leaf).
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` empty (no dep/schema change).
- [x] Gate-integrity scan over unpushed commits (`@{upstream}..HEAD`) — no `//nolint` / `t.Skip` /
  build-tag exclusion / swallowed-error-to-dodge / deleted assertion. The `_ = buf.WriteTo(w)` /
  `_ = encodeJSON(...)` sites are the pre-existing, documented post-200 write-drop convention.
- [x] Correctness cross-check: `follower.glossaryStatus` (the only writer behind `metrics.Registry.Status`)
  emits only `frozen`/`verified`/`unresolvable`/`unverified` — all valid `badge.Label` keys — so the
  overlay's `unresolvable`/`unverified` filter is exhaustive, the `!ok → label = status` fallback is
  genuinely defensive-only, and any other live value (incl. `frozen`/`verified`) keeps store-`verified`.
  `overlayStatus` is a verbatim precedence mirror of `dashboard.overlayStatus`.
- [x] Oracle/conformance gate correctly N/A — pure HTML composition of persisted rows + an in-memory
  status overlay; no signature/RFC-6962/Merkle/did:web/fsck/proof path.

**Issues found:** (none) — no defect. The lone open `low` notecheck `out io.Writer` item is unrelated
and still valid; no issues.md change this iteration.

**Codex second opinion:** Codex (gpt-5.5, xhigh) finished (exit 0) after a thorough agentic explore —
dumped the diff, read `handler.go`/`browser.html`/`main.go`/`badge`/`dashboard`/`metrics` and the
tests, and ran the build/tests. Its final report: "The changes compile and tests pass, and the metrics
status overlay is correctly threaded into the log browser without breaking existing proof routes or
routing behavior." No `[P1]`–`[P3]` findings — nothing to triage; agrees with my independent review.

**Next:** Wire the same `StatusSource`-overlay + `badge.Label`-precompute pattern into the upcoming hub
dossier and the paginated record-list / single-record pages so the five-status taxonomy stays uniform
across every M-UI surface (each surface defines its own tiny local copy of the shape — no shared import,
per the plan). `inactive` is still only reachable through a registry-deactivation writer that does not
exist yet (`serveBrowser` reads `FollowState`, and the dashboard has no `SetActive` writer) — add an
end-to-end `inactive` render assertion once that writer lands.

**Notes:**
- Scope: exactly 3 non-test source files (`proofserve/handler.go`, `proofserve/browser.html`,
  `cmd/iscc-monitor/main.go`) + tests + `CLAUDE.md` doc — at the ≤3 limit. Nothing from `## Not In Scope`
  touched: no new status added to `proofserve.hubStatus`/`store.ListHubs`, no `internal/badge` change, no
  `internal/metrics` import in proofserve, no CSS/DS-token work; `serveBrowser`'s buffer-first render +
  method gate + no-checkpoint 200 branch left intact.
- Honesty improvement (not a regression): the prior "unpolled non-frozen hub renders bare verified"
  minor noted in `learnings/http-surface.md` is now RESOLVED — both `browser.html` branches invoke the
  partial, so a `LastSize==0` hub whose live verdict is `unresolvable`/`unverified` shows that honestly.
- Learnings: settled the log-browser thread-through in `learnings/http-surface.md` (added the
  landed `StatusSource`+`overlayStatus` note, resolved the bare-verified minor), and updated the
  forward-looking "reuse this for the log browser" notes in `learnings/dashboard.md` + `learnings/badge.md`
  to "settled". No cross-cutting index promotion (all package-local). Budgets OK.
- Working-tree note: `.claude/agents/review.md` carries an uncommitted improvement (landed by the prior
  `fix(loop)` workstream) documenting the `sed -n '/^codex$/,$p'` extract-only technique for the Codex
  transcript. It is not part of this increment and I did not stage it; flagging for whoever owns the
  loop docs.
- M-UI is in progress (this slice closes the per-surface five-status render consistency for `/` +
  `/<domain>/log/`); v1 milestones (M3 verify-for-me complete, OTS) are not all independently re-confirmed
  this iteration, so Loop = CONTINUE, not DONE.
