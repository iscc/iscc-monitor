## 2026-06-22 — Review of: Harden the otsclient upgrade path — panic-recover + per-request timeout

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance routed the OTS calendar upgrade through a new `safeUpgrade` FFI-boundary
helper that derives a per-request `context.WithTimeout(ctx, 30s)` (one `defer cancel()` per call, no
defer-in-loop leak) and recovers any library panic into a wrapped fail-closed error — exactly what
`next.md` asked, closing the last open `normal` otsclient issue before `stampRoot`'s calendar-submit
makes `upgrade()` reachable. Scope is exemplary (1 non-test source file + 1 test file), both fixes are
mutation-proven non-vacuous, every isolation invariant holds, and Codex found nothing.

**Verification:**
- [x] `mise run check` — GREEN (build + vet + test, all 23 packages incl. `internal/otsclient`).
- [x] `gofmt -l .` (excl. `cauldron/`) — clean.
- [x] `go test -count=1 -v ./internal/otsclient` — pass (9 tests incl. the two new ones).
- [x] Mutation (reverted): removing the `safeUpgrade` `recover()` block → `TestUpgradePanicRecovered`
  PANICS the test binary (FAIL); restored → green. Confirmed by reviewer (panic stack traced to
  `client_test.go:158` propagating out of `safeUpgrade`).
- [x] Mutation (reverted): replacing `context.WithTimeout(ctx, upgradeTimeout)` with the bare ctx →
  `TestUpgradeBoundsContext` FAILS ("upgrade ctx carried no deadline"); restored → green. Restored
  file is byte-identical to HEAD (clean `git diff`).
- [x] `go test -count=1 -run TestUpgradeAlreadyConfirmed ./internal/otsclient` — pass; confirmed
  fixtures still classify at their exact heights, pending-only-upgrade invariant held.
- [x] Isolation: `GOOS=js GOARCH=wasm go build ./internal/{didweb,index,badge}` OK; badge ots/otsclient
  deps == 0; otsclient `database/sql` count == 1 (the `store.OTSRecord` seam, unchanged); store
  otsclient/follower/net-http deps == 0; follower anchoring deps == 0.
- [x] `go vet ./internal/otsclient` — clean (no `lostcancel`: the `defer cancel()` runs per-call inside
  `safeUpgrade`, not accumulated in the loop — the exact defer-in-loop trap `next.md` warned about,
  correctly avoided).
- [x] No bypass: the only `upgrade(...)` call lives inside `safeUpgrade` (client.go:166); the loop calls
  `safeUpgrade`, so no path reaches the seam unguarded.
- [x] `go.mod`/`go.sum` untouched in the advance commit; `go mod tidy -diff` clean.
- [x] Gate-circumvention scan over unpushed commits (`@{upstream}..HEAD`): the only `nolint`/`t.Skip`
  matches are prose (handoff/next.md + the docstring stating "never a //nolint or swallow"); no
  build-tag, swallowed error, or deleted assertion. The `recover()` is a documented FFI-boundary guard
  that re-surfaces the panic as a returned error — not a gate-dodge.
- [x] Scope: 1 non-test source file (`internal/otsclient/client.go`) + 1 test file; nothing in
  `## Not In Scope` touched (`follower.go`/`stampRoot`, `main.go`/`runOTSLoop`, the `.ots` route,
  certificate §5 all byte-unchanged; no `Stamper` seam / `MarkOTSSubmitted`; `Stamp` body untouched).

**Issues found:** (none new) — Resolved + deleted the open `normal` otsclient panic/timeout issue (both
defects fixed, mutation-proven). The remaining open `normal` issues (`hubDomain` ForceQuery; §4/bundle
`host:port` DID; §6 timestamp) are on surfaces this diff did not touch — none resolved or made stale.

**Codex second opinion:** Clean — explicit "did not find any introduced correctness, security, or
maintainability issues that warrant an inline finding." It independently confirmed the bounded context
and panic recovery do not break existing behavior and the updated tests exercise the new paths. No
findings to triage.

**Visual check:** n/a — no SSR surface changed (`internal/otsclient` is a network transport adapter; no
templates, no `internal/{dashboard,dossier,web,certificate}` touched).

**Next:** Wire `otsclient.Stamp` into `follower.stampRoot` (`follower.go:411`) so a stamped pending row
carries real `OTSBytes`/`CalendarURLs` — the deferred follow-up this step de-risked. With both
upgrade-path hazards now closed, that step can safely let `upgrade()` go live (a real proof reaching an
unimplemented op or a stalled calendar now degrades to a back-off, not a crash/hang). After it: the
`.ots` HTTP route (reads `OTSForRoot`) and certificate §5 BITCOIN ANCHOR (both depend on confirmed rows
existing).

**Notes:**
- The OTS milestone is now structurally de-risked: the upgrade path is crash- and hang-proof, but OTS is
  still NOT end-to-end functional in production — `stampRoot` writes pending rows with EMPTY `OTSBytes`,
  so `OTSTick`'s upgrade is a structural no-op until the calendar-submit step lands. That submit step is
  the single clear next increment on the OTS arc.
- `safeUpgrade` takes `(ctx, ...)` (ctx-first, Go convention) rather than `next.md`'s literal
  `(upgrade, ctx, ...)` sketch — same seam, idiomatic ordering; documented in the advance handoff.
- Learnings: collapsed the two now-resolved otsclient trap bullets (panic + timeout) into one
  `settled:` note in `learnings/otsclient.md` that preserves the forward-looking pitfall (keep all
  upgrade calls routed through `safeUpgrade`; do not strip the guard). No index promotion (the rule is
  package-local). otsclient.md now 49 lines / 6 bullets — within budget.
