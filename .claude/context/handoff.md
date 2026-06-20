# Handoff

## 2026-06-20 — Review of: Poll-loop wrapper over PollHub (single-writer cadence + frozen back-off)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/follower/loop.go` — a per-network poll loop that drives the
existing `PollHub` over a fixed set of `HubTarget`s from one goroutine on a cadence, re-polling a
frozen hub only on a longer backed-off interval (ADR-0006). The cadence lives in the pure `due()`
predicate against an injected `now`; `Run` is thin `time.Ticker` plumbing. Diff is tight (1 production
+ 1 test file), all gates are green uncached, every `next.md` Verification criterion passes
individually, and no crypto/merkle/oracle/schema/dependency path was touched.

**Verification:**
- [x] `mise run check` — green (`go build ./...`, `go vet ./...`, `go test ./...` all `ok`:
  didweb/follower/logclient/store). Re-confirmed uncached (`go test -count=1 ./...`).
- [x] `gofmt -l .` — empty (clean; both new files individually clean).
- [x] `go test -count=1 -run TestDue ./internal/follower` — PASS; all 10 golden subtests run
  (verbose-confirmed, non-vacuous): zero `lastPoll` always due even frozen; unfrozen `< Normal` not
  due, due at/past `Normal`; frozen past `Normal` but `< Frozen` NOT due (back-off), due at/past
  `Frozen`. `due()` uses `>=`, so both boundary cases (`==Normal`, `==Frozen`) are covered.
- [x] `go test -count=1 -run 'TestTick$' ./internal/follower` — PASS (two clean hubs both advance to
  10183 on Tick 1; a second Tick at the same `now` re-polls neither — `checkpoints` count and both
  cursors unchanged, proving the in-memory `lastPoll` throttle).
- [x] `go test -count=1 -run TestTickFrozenUnaffected ./internal/follower` — PASS (shrink-seeded hub
  freezes on Tick 1 with 1 violation + 1 alert; a Tick past `Normal` but before `Frozen` does NOT
  re-poll it — `violations` stays 1; a Tick at `Frozen` re-polls it — `violations` → 2, alerts stays
  1, `Frozen` never cleared, `LastSize` stays 20000; clean hub advances to 10183 every pass).
- [x] `go test -count=1 ./internal/follower` — PASS (existing `TestPollHub*` stay green; `PollHub`
  unchanged).
- [x] `git status --short go.mod go.sum` empty — no dependency added; `go.mod`/`go.sum`
  byte-identical.
- [x] Scope — exactly 1 production file (`internal/follower/loop.go`) + 1 test file
  (`loop_test.go`); `follower.go`/`checkpoints.go`/`PollHub` NOT edited (the existing seam sufficed,
  as `next.md` anticipated); no `## Not In Scope` item done.
- [x] Gate integrity — scanned `@{upstream}..HEAD` (= `origin/develop..HEAD`, the 3 unpushed
  commits): no `//nolint`/`t.Skip`/build-tag/removed-assertion. The only swallowed errors are
  `_ = l.Tick(ctx, t)` in `Run` (documented inline; `Tick` surfaces the error to its caller, so this
  is not gate-dodging) and `_ = s.Close()` test cleanup (standard idiom).
- [x] Purity / leaf — follower production imports are exactly `{context, fmt, logclient, store,
  time}`; `go list -deps ./internal/store` has no `net/http` (store stays a leaf).
- [x] Conformance/oracle gate — correctly N/A: pure orchestration over `PollHub` + store reads;
  `git diff --name-only` touches no proof/verify/didweb/merkle/consistency path, and `internal/proof`
  does not exist yet.

**Issues found:** (none) — no minor fixes needed.

**Next:** The `cmd/iscc-monitor` binary entrypoint is the lowest-risk unblocked slice (wire DI + the
realm registry + config and call `Loop.Run` — the loop is now a pure library function with injected
targets/fetcher/alert) and makes the loop actually run. Alternatively the merkle-backed
**equivocation** trigger (highest value, highest risk — needs `transparency-dev/merkle`, a new dep,
plus tile fixtures, and trips the `fsck`/inclusion/golden-vector oracle gate) or the `hub_keys`
did:web cache write (couples to the stale `sb1.amlet.id_did.json` + `derive_vkey.py` HUBS refresh →
trips the crypto-parity oracle, so its own step).

**Notes:**
- **The back-off works *because a freeze returns `(StatusVerified, nil)`*.** `Tick` records
  `lastPoll[hub] = now` only on a nil-error `PollHub`, and a freeze is a nil error — so a just-frozen
  hub *does* get its `lastPoll` set and the next due decision correctly uses the longer `Frozen`
  interval. If a later change ever made freeze return a non-nil error, the frozen hub would be left
  unmarked and re-polled every `Normal` tick (no back-off). Captured in `learnings.md` as a
  guard-this invariant for the equivocation slice.
- **`Run` is untested and that is acceptable here.** It is a 12-line `select` over `ctx.Done()`/
  `ticker.C` with `defer ticker.Stop()`; all branching logic lives in the injected-`now` `Tick` + the
  pure `due()`, both covered. The spec forbids wall-clock sleeps, so testing `Run` would mean
  sleeping. `time.Now()` never appears in `loop.go` (the ticker delivers `t` via `ticker.C`); the
  only wall-clock source is `time.NewTicker(l.Normal)`.
- **`lastPoll` is in-memory only** (no `last_poll` column, per `## Not In Scope`): a process restart
  re-polls every hub on the first tick — harmless, `PollHub` is idempotent on an unchanged checkpoint.
  Persisting last-poll across restart is a future schema decision, not v1.
- Branch is `develop`; pushing on PASS per policy (never `main`).
