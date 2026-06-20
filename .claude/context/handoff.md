# Handoff

## 2026-06-20 — Poll-loop wrapper over PollHub (single-writer cadence + frozen back-off)

**Done:** Added a per-network poll loop (`Loop`) that drives the existing `PollHub` over a fixed set
of `HubTarget`s from one goroutine on a cadence, with a frozen hub re-polled only on a longer
backed-off interval (ADR-0006). The cadence decision is the pure `due()` predicate against an
injected `now`, so `Tick` is fully deterministic; `Run` is the thin `time.Ticker` plumbing. No new
dependency, no schema change, no crypto/merkle/oracle path touched.

**Files changed:**
- `internal/follower/loop.go` (new): `HubTarget{HubID,BaseURL}`; pure `due(frozen, lastPoll, now,
  normal, frozenInterval) bool`; `Loop` struct holding config + an in-memory `map[int64]time.Time`
  last-poll cache (no schema change, no lock — single goroutine); `Loop.Tick` (one sequential pass,
  reads `Frozen` fresh from `FollowState`, polls each due hub via `PollHub`, marks lastPoll only on
  success, returns the first error and continues the pass); `Loop.Run` (`time.Ticker` at `Normal`
  until `ctx.Done()`, returns `ctx.Err()`, never panics, ignores per-tick errors so one flaky hub
  never aborts the network's loop).
- `internal/follower/loop_test.go` (new): `TestDue` (10-row golden table), `TestTick`,
  `TestTickFrozenUnaffected`, reusing the existing offline helpers verbatim
  (`sb0VerifiedFetcher`/`openTemp`/`countRows`/`assertViolation`/`sb0ObservedAt`/`noopAlert`).

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all `ok`:
didweb/follower/logclient/store). `gofmt -l .` empty. Per-criterion:
- [x] `go test -count=1 -run TestDue ./internal/follower` — PASS (zero lastPoll always due, even
  frozen; unfrozen `< Normal` not due, due at/past `Normal`; frozen past `Normal` but `< Frozen` NOT
  due = back-off, due at/past `Frozen`).
- [x] `go test -count=1 -run 'TestTick$' ./internal/follower` — PASS (two clean hubs both advance to
  10183 on the first Tick; a second Tick at the same `now` re-polls neither — `checkpoints` row count
  and both cursors unchanged).
- [x] `go test -count=1 -run TestTickFrozenUnaffected ./internal/follower` — PASS (shrink-seeded hub
  freezes on Tick 1 with 1 violation + 1 alert; a Tick past `Normal` but before `Frozen` does NOT
  re-poll it — `violations` stays 1; a Tick at `Frozen` re-polls it — `violations` → 2, alerts stays
  1, `Frozen` never cleared, `LastSize` stays 20000; clean hub advances to 10183 every pass).
- [x] `go test -count=1 ./internal/follower` — PASS (existing `TestPollHub*` stay green; `PollHub`
  unchanged).
- [x] `git status --short go.mod go.sum` empty — no dependency added; `go.mod`/`go.sum`
  byte-identical.

**Next:** With the poll loop landed, the cleanest remaining M1 slices are: (a) the `cmd/iscc-monitor`
binary entrypoint that wires DI + the realm registry + `config` and calls `Loop.Run` (now unblocked —
the loop is a pure library function with injected targets/fetcher/alert); (b) the `hub_keys` did:web
key cache write (couples to the stale `sb1.amlet.id_did.json` + `derive_vkey.py` HUBS refresh → trips
the crypto-parity oracle, so its own step); or (c) the merkle-backed **equivocation** trigger (needs
`transparency-dev/merkle` — a new dep — + tile fixtures, trips the `fsck`/inclusion/golden-vector
oracle gate). The equivocation trigger remains the highest-value but highest-risk; the `cmd/` binary
is the lowest-risk next step and makes the loop actually run.

**Notes:**
- **Error policy is explicit (documented in `loop.go`):** `Tick` attempts every due hub, returns the
  first error encountered (nil if all succeeded), and never aborts the pass on a per-hub fault — a
  hub whose poll errored is left unmarked in `lastPoll` so it retries next due tick. `Run` ignores
  the per-tick `Tick` error (`_ = l.Tick(...)`) and continues; this is the ONE deliberate
  swallowed-error in the diff and it is justified inline (a flaky hub must not stop the network's
  loop; the error will surface via the store/metrics once structured logging lands). It is not
  dodging a gate — `Tick` itself surfaces the error to its caller.
- **`lastPoll` is in-memory only (no `last_poll` column)** per `## Not In Scope`. Consequence: a
  process restart re-polls every hub immediately on the first tick — harmless because `PollHub` is
  idempotent on an unchanged checkpoint (`RecordCheckpoint` dedupes on `UNIQUE(hub_id,tree_size,
  root)` and a re-observed clean checkpoint just re-advances to the same size). Persisting last-poll
  across restart is a future schema decision, not v1.
- **`Run` is not exercised by a test** (it is wall-clock ticker plumbing; the spec forbids wall-clock
  sleeps and scopes the binary/wiring to a later step). All logic worth testing lives in the pure
  `due()` and the injected-`now` `Tick`, both covered. `Run` is a thin 12-line `select` over
  `ctx.Done()`/`ticker.C` calling `Tick` — no branching logic beyond cancellation.
- **Conformance/oracle gate: correctly N/A** — pure orchestration over `PollHub` + store reads; no
  signature, RFC-6962 proof, didweb, or merkle code touched. The gate trips only when the
  merkle-backed equivocation slice lands.
- **Scope:** 1 production file + 1 test file (within the ≤3-file cap); `follower.go`,
  `checkpoints.go`, and `PollHub` were NOT edited (the existing seam was sufficient, as `next.md`
  anticipated). No `## Not In Scope` item done.
- Branch is `develop` (not `main`), per loop policy.
