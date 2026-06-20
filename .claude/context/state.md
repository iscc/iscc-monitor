<!-- assessed-at: 623d866b6a2e6f740540e804c40bdf279af63141 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — freeze loop closed, poll-loop cadence wrapper now drives PollHub

The M1 crypto/verification core is complete and the shrink+fork freeze loop is wired end-to-end into
`PollHub` (freeze + alert-once, ADR-0006). Since the last assessment the **per-network poll loop**
(`internal/follower/loop.go`) landed: a single-goroutine cadence driver over `PollHub` with a backed-off
re-poll of frozen hubs. What still blocks M1: the merkle-backed equivocation trigger, the `hub_keys`
did:web cache write, config + realm registry, coverage, structured logs, `/metrics`, real alert
transport, and a `cmd/` binary entrypoint.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence done; equivocation
trigger, the connective config/coverage/logs/metrics/binary tissue, and real alert transport still missing)
- Verified present (incremental re-check; the only Go change since the prior assessment at `12f8155` is the
  new poll loop — all other sections carried forward and reconfirmed on disk):
  - **NEW — poll-loop cadence wrapper** (`internal/follower/loop.go`): `Loop` drives `PollHub` over a
    fixed `[]HubTarget` from one goroutine (the single writer per DB, ADR-0005/0007). The pure `due(frozen,
    lastPoll, now, normal, frozenInterval) bool` predicate (zero `lastPoll` → always due; `>=` boundary)
    is the only branching logic; `Tick(ctx, now)` reads the freeze flag fresh from `store.FollowState`
    each pass, polls every due hub, records `lastPoll` only on a nil-error poll, and returns the first
    error without aborting the pass (a flaky hub never stalls the network loop). `Run` is thin
    `time.Ticker` plumbing; `time.Now()` never appears in the file. Frozen hubs re-poll only at the longer
    `Frozen` interval — the backed-off evidence-only cadence. `lastPoll` is in-memory only (no schema
    change; restart re-polls all, harmless since `PollHub` is idempotent). Tested by `TestDue` (10 golden
    subtests), `TestTick` (two clean hubs advance, second tick at same now re-polls neither) and
    `TestTickFrozenUnaffected` (shrink-seeded hub freezes once, NOT re-polled before `Frozen`, re-polled
    at `Frozen` → 2nd violation, alert stays 1, clean hub advances every pass).
  - **Freeze + alert-once wiring** (`internal/follower/follower.go`): on `StatusVerified`, `PollHub` reads
    `FollowState`, runs `checkConsistency` (shrink-then-fork against the prior accepted checkpoint) BEFORE
    any record/advance; on a true verdict `freeze(...)` does `RecordViolation` + `RecordCheckpoint`
    (contradictory evidence, no advance) + `Freeze`, then fires the injected `AlertFunc func(int64,string)`
    once iff `!wasFrozen`. A violation returns `(StatusVerified, nil)` — freezes, never crashes (ADR-0006),
    no auto-unfreeze. Production imports stay `{context, fmt, logclient, store, time}`. Tested by
    `TestPollHubFork`/`TestPollHubShrink`.
  - **`store.CheckpointAt(ctx, hubID, treeSize)` → `(root, raw, found, err)`** (`internal/store/
    checkpoints.go`): recovers the prior accepted root (`follow_state` does not persist `LastRoot`);
    `LIMIT 1`, no `ORDER BY`. Tested by `TestCheckpointAt` + `TestCheckpointHelpersRestartSurvival`.
  - `internal/logclient/consistency.go` — the two dep-free RFC-6962 triggers `CheckShrink` (`prev>0 &&
    next<prev`) and `CheckFork` (`prevSize>0 && nextSize==prevSize && nextRoot!=prevRoot`, array `!=`),
    plus `ViolationKind`/`ViolationShrink`/`ViolationFork`. The `transparency-dev/merkle` mention is
    prose-only (confirmed: not in go.mod, not imported). Consumed by `follower.checkConsistency`.
  - `internal/store/{sqlite,schema}.go` + `checkpoints.go` CRUD — `Open`/`Close` over `modernc.org/sqlite`
    with ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`,
    `synchronous=NORMAL`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`, typed
    `UpsertHub`/`RecordCheckpoint`/`FollowState`/`AdvanceFollowState`/`CheckpointAt`/`RecordViolation`/
    `Freeze`. `store` stays a leaf (no `logclient` import; no `net/http` in its closure).
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin}.go` — transport-only
    `FetchCheckpoint`, pure 4-way `AcceptCheckpoint` (clock-injected), pure `VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/
    `DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    `internal/{didweb,logclient}/testdata/{sb0,sb1}_did.json` — did:web fixtures.
  - Test totals: 9 (`didweb`) + 18 (`logclient`) + 19 (`store`) + 7 (`follower`, +3 for due/tick/tick-frozen)
    = **53 `func Test`** (up from 50).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — the only remaining of the three triggers (RFC-6962 consistency-proof failure
    across *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and
    will trip the conformance/oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - config + realm registry (domains only), coverage (`monitored_since`), structured logs, `/metrics`.
  - **`cmd/` is absent** — no binary entrypoint yet (verified: `cmd/` does not exist). The poll loop is now
    a pure library (`Loop.Run` with injected targets/fetcher/alert) ready to be wired by a `main`.
  - **Real alert transport** — `AlertFunc` is a minimal func seam; production delivery (email/webhook/log
    sink) is unbuilt.
- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed for the
  equivocation trigger + M2). Known stale-fixture drift (not yet acted on): `derive_vkey.py` HUBS + both
  `sb1.amlet.id_did.json` did:web fixtures carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer
  is now `069d0f14`; refresh lands with the `hub_keys` cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero import statements + zero
  go.mod entries): `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey` byte-match
  for both hubs — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected + evidence-survives-
  restart, now also exercised through the poll loop). The **third trigger, equivocation, is not met**
  (absent), so the M1 Verify line is not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "Poll-loop wrapper over PollHub")
  records the gate green at HEAD `623d866`: `go build/vet/test ./...` all `ok` (re-confirmed uncached),
  `gofmt -l .` empty, all due/tick/tick-frozen tests PASS (alert count stable), `go.mod`/`go.sum`
  byte-identical (no dep added), no `//nolint`/`t.Skip`/build-tag/swallowed-error dodges. Conformance/
  oracle gate correctly N/A this step (pure orchestration over `PollHub` + store reads; `internal/proof`
  does not exist yet). The merkle-backed equivocation slice that follows *will* trip the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `623d866`. **No `.github/workflows/` and `gh run list` returns `[]` — no CI configured.** When CI is
  wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out the
  future `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. With the freeze loop and poll cadence both closed, the highest-value remaining slice is the
**merkle-backed equivocation trigger** (RFC-6962 consistency-proof failure across growing sizes): plug it
into the same `checkConsistency` seam as a third branch returning `ViolationEquivocation` + a real
`ProofJSON`; it needs `transparency-dev/merkle` (a new dep) + tile fixtures and **trips the conformance/
oracle gate** (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check vs the hub's
`IsccLogInclusionProof`, golden-vector parity). It MUST preserve "compare against the prior *accepted*
root, not the contradicting evidence" (the `CheckpointAt` `LIMIT 1` rowid-order learning). Lower-risk
unblocked alternatives: (a) the **`cmd/iscc-monitor` binary** (wire DI + realm registry + config and call
`Loop.Run` — the loop is now a pure library, so this is the cleanest no-crypto slice and makes the monitor
actually run); (b) the **`hub_keys` did:web cache write** (which MUST also refresh the stale
`sb1.amlet.id_did.json` + `derive_vkey.py` HUBS to signer `069d0f14`, re-triggering the parity oracle).
Coverage (`monitored_since`) + structured logs + `/metrics` + real alert transport then complete M1's
Verify criteria. No CI is configured — flag for whoever sets up the workflow.
