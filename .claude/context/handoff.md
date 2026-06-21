## 2026-06-21 — Review of: Thread in-memory hub status into `/` so all five badges render honestly

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` threaded the in-memory glossary verdict onto the `/` dashboard via a tiny
`dashboard.StatusSource` interface (satisfied structurally by `*metrics.Registry`'s new
`Status(hubID) (string, bool)` reader) and an `overlayStatus` precedence function, so `/` now renders
all five glossary statuses honestly. The diff is scope-clean (3 non-test source files + 2 tests + 1 doc
at the ≤3 limit), keeps `internal/dashboard` free of an `internal/metrics` import and `internal/store` a
leaf, and the overlay test is mutation-proven non-vacuous. The precedence is correct: durable
`inactive`/`frozen` win; the live verdict overlays only a store-`verified` hub and only adopts
`unresolvable`/`unverified`.

**Verification:**
- [x] `mise run check` green — all 17 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty (whole tree).
- [x] `go test -count=1 ./internal/metrics ./internal/dashboard ./cmd/iscc-monitor` PASS (uncached).
- [x] `go test -run TestDashboardRenders ./internal/dashboard` PASS — body carries both
  `data-status="unresolvable"`/`>Unresolvable<`/`M9.2 9.3` and `data-status="unverified"`/`>Unverified<`/`M12 3.4 21 19H3z` plus the existing verified/frozen markers.
- [x] `go test -run TestStatus ./internal/metrics` PASS; `go test -run TestOverlayStatusPrecedence ./internal/dashboard` PASS.
- [x] Mutation check (my own run, reverted): `status := overlayStatus(s, statuses)` → `status := hubStatus(s)`
  makes `TestDashboardRendersInMemoryStatus` FAIL (overlaid hubs render `data-status="verified"`); restored → PASS, tree byte-clean.
- [x] `GOOS=js GOARCH=wasm go build ./internal/metrics ./internal/badge` OK — metrics leaf stays WASM-shareable (new reader added no import).
- [x] `go list -deps ./internal/dashboard | grep internal/metrics` empty — depends on the interface, not the concrete package.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/badge|internal/metrics'` empty — store stays a leaf.
- [x] `git diff --stat HEAD~1..HEAD -- go.mod go.sum internal/store/schema.sql` empty — no dep/schema change.
- [x] Gate-integrity scan over unpushed commits — no `//nolint` / `t.Skip` / build-tag exclusion / swallowed-error-to-dodge / deleted assertion in added Go/HTML (the lone `_ = buf.WriteTo(w)` is the pre-existing, documented post-200 write-drop; the other unpushed commits are context-file/doc-only).
- [x] Correctness cross-check: `follower.glossaryStatus` only ever emits the five glossary strings, so the overlay's `unresolvable`/`unverified` filter is exhaustive and any other live value (incl. in-memory `frozen`/`verified`) correctly keeps the store's `verified`.
- [x] Oracle/conformance gate correctly N/A — pure data plumbing + HTML composition; no signature/RFC-6962/Merkle/did:web/fsck/proof path.

**Issues found:** (none) — no defect. The lone open `low` notecheck `out io.Writer` item is unrelated and still valid; no issues.md change this iteration.

**Codex second opinion:** Codex (gpt-5.5, xhigh, `reasoning summaries: none`) explored the increment thoroughly — dumped the full diff at `--unified=80`, read `handler.go`/`metrics.go`/the tests/`store` `ListHubs`+schema, grepped `dashboard.Handler`/`buildMux` call sites, and ran `go test ./...` (all 17 green) — then ended on the test run with no explicit findings or verdict paragraph (summaries off), same behavior as the prior iterations with this config. Clean exploration, nothing to triage; no action.

**Next:** Wire the same `StatusSource`-interface + overlay shape into the per-hub log-browser status cell (`proofserve.hubStatus`) and the upcoming hub dossier, so the richer taxonomy is consistent across all M-UI surfaces — the `overlayStatus` precedence + `badge.Label` precompute is the reusable pattern (now settled in `learnings/dashboard.md`). The `inactive` end-to-end render stays a table test until a registry-deactivation writer / public `SetActive` lands; add an end-to-end `inactive` assertion then.

**Notes:**
- Scope: 3 non-test source files (`metrics.go`, `dashboard/handler.go`, `main.go`) + 2 tests + CLAUDE.md doc — at the ≤3 limit; nothing from `## Not In Scope` touched (no store status column, no Prometheus render-path change, no `internal/badge` change, no CSS).
- `metrics.Status` correctly relies on `SetHubStatus`'s "exactly one status=1 per hub" invariant (zeroes others before setting). Recorded as a forward-looking pitfall in `learnings/metrics.md`: if a future change lets two statuses read 1 for one hub, `Status` becomes map-iteration-order-dependent — re-check before relaxing `SetHubStatus`.
- Learnings: settled the "richer statuses belong at the thread-through" open notes in `learnings/dashboard.md` (now documents the landed `overlayStatus` + precedence rule) and `learnings/badge.md` (now states `/` renders all five); added the `Status`-accessor invariant note to `learnings/metrics.md`. No cross-cutting index promotion (all package-local). Budgets OK: index 82, dashboard ~57, metrics ~89, badge ~52 lines.
- M-UI is in progress (this slice closes the five-status render); v1 milestones (M3 verify-for-me, OTS) are not all met, so Loop = CONTINUE, not DONE.
