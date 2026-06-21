<!-- assessed-at: 111b1a8a68f819045d922579cd5c74f75d3e1dfb -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 closing — the `internal/metrics` leaf is now **wired into the live verdict path**. The follower fires all four metric series (`SetHubStatus`/`SetLastObservedAt` on every non-error verdict via `recordVerdict`, `IncViolation` at freeze, `IncPollFailure` in `Tick` on the error branch), with the load-bearing verdict→glossary remap correct (`rotated`→`unverified`, freeze→`frozen`). What remains to close M1's `/metrics` deliverable is the **`net/http` handler** serving `Registry.WriteText` plus `metrics.New()` in `cmd/iscc-monitor/main.go` passed as `Loop.Metrics` (still left nil). A **real (non-log) alert transport** is the other outstanding M1 connective tissue. The equivocation branch stays production-dormant until M2's tile-ingestion writer lands.

All three self-consistency triggers (shrink/fork/equivocation) are wired into the follower and golden-tested end-to-end. The M1 crypto/verification core, the single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + config loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, the `hub_keys` did:web key cache, the `internal/tiles` tlog-tiles seam, the `internal/store` partial-tile mirror CRUD + `SQLiteFetcher`, the pure `ConsistencyProofFromTiles` builder, structured `slog` logging, and the now-wired `internal/metrics` leaf have all landed.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + all three freeze triggers wired & golden-tested end-to-end + poll-loop cadence + realm-registry parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache + tlog-tiles seam + `SQLiteFetcher` + `ConsistencyProofFromTiles` + structured `slog` logging + **metrics leaf now wired into the verdict path**; **`/metrics` HTTP handler + `cmd` wiring and a real alert transport remain missing**)

- **Verified incrementally** from the prior assessment at `12ea5cc`. The diff `12ea5cc..HEAD` touches
  **only** `internal/follower/{follower,follower_test,loop,loop_test}.go` (the metrics-wiring slice) plus
  context files (`handoff.md`, `learnings.md`, `next.md`, `state.md`). Confirmed by `git diff --stat`:
  all other source paths (`internal/{metrics,store,logclient,didweb,tiles,config,registry}`,
  `cmd/iscc-monitor`, `go.mod`, `go.sum`, `schema.sql`) are byte-unchanged and carried forward.

  - **NEW — metrics wired into the follower verdict path** (verified by reading
    `internal/follower/{follower,loop}.go`): `Loop` gained an optional `Metrics *metrics.Registry`
    field (nil-safe; a bare `&Loop{…}` still works). `PollHub` now takes a trailing
    `m *metrics.Registry` param and funnels all three non-error verdict branches (early-non-verified,
    freeze, verified-advance) through `recordVerdict(m, hubID, status, frozen, observedAt)`, the single
    nil-safe mutation point for `hub_status` + `last_observed_at`. The freeze branch additionally calls
    `m.IncViolation(hubID, kind)` (re-fires on every re-detection — correct, re-detection is evidence).
    `Tick` calls `l.Metrics.IncPollFailure(target.HubID)` on `PollHub`'s error return — the same branch
    as the `ErrorContext` log — because a transport/garbled-body fault returns early before the verdict
    is known. **Load-bearing verdict→glossary remap is correct** (`glossaryStatus`): `frozen` takes
    precedence → `"frozen"` (overriding the still-`StatusVerified` enum after a freeze);
    `StatusVerified`→`"verified"`, `StatusUnverified`/`StatusRotated`→`"unverified"` (glossary has no
    "rotated"), `StatusUnresolvable`→`"unresolvable"`. This is the carried-forward wiring gotcha,
    resolved correctly: it maps to the **glossary** status set, NOT `logclient.Status.String()`.
    **+2 `func Test`** in follower (12 → 14). Latest `review` handoff (2026-06-21, verdict
    **PASS / CONTINUE**) records the gate green at this HEAD, mutation-proves two asserts non-vacuous
    (drop the frozen flag → `TestPollHubFork` renders `status="verified"` and FAILS; remove
    `IncViolation` → FAILS on missing `violations_total`), and confirms `store` stays a leaf
    (`go list -deps ./internal/store | grep net/http` empty — the follower→metrics edge does not leak in).

  - **REMAINING M1 GAP — `/metrics` not served or wired into `cmd`:** the increment sites now fire, but
    (verified by grep) there is **no `net/http` handler** serving `Registry.WriteText`, and
    `cmd/iscc-monitor/main.go` leaves `Loop.Metrics` **nil** (zero `Metrics`/`/metrics` references in
    `cmd/`). The only `net/http` usage in the tree is `internal/logclient/didresolve.go` (the did.json
    fetcher) + two test servers — no monitor HTTP server exists (`ListenAndServe`/`HandlerFunc` appear
    only in test `httptest` code). So in production no metrics are collected (the loop is constructed
    with nil Metrics) and none are exposed. M1's `/metrics` Verify/scope criterion is still open: the
    next slice must add the `net/http` handler (`Content-Type: text/plain; version=0.0.4`) +
    `metrics.New()` in `main.go` passed as `Loop.Metrics` + served.

  - **`internal/metrics` leaf** (unchanged this slice, carried forward): `*Registry` (RWMutex-guarded)
    over four metric families — two counters (`iscc_monitor_violations_total{hub_id,kind}` /
    `iscc_monitor_poll_failures_total{hub_id}`) and two gauges
    (`iscc_monitor_hub_status{hub_id,status}` set-one-clear-rest / `iscc_monitor_last_observed_at{hub_id}`)
    — rendered in Prometheus text-exposition format. Imports stdlib-only `{fmt io sort strconv strings
    sync}` (net-free, db-free, log-free, clock-free). 6 `func Test`. No new dependency.

  - **equivocation trigger WIRED, production-dormant** (`internal/follower/follower.go`):
    `checkConsistency` evaluates shrink → fork → equivocation; the third (growing-pair) branch builds
    the RFC-6962 consistency proof from the LOCAL mirror (`store.SQLiteFetcher.ReadTile` →
    `logclient.ConsistencyProofFromTiles` → `logclient.CheckEquivocation`) and on a `true` verdict →
    `ViolationEquivocation` → `freeze`. A proof-build error (most often **tiles not mirrored yet**) is
    swallowed narrowly → branch skipped, never freezes on a missing tile (ADR-0006 false-positive
    guard). Golden-tested end-to-end + mutation-proven, crossing the 256-leaf tile boundary over the
    real `SQLiteFetcher`. **Dormant in the live path until M2's tile-ingestion writer lands** — today
    `PollHub` never writes tiles, so a live observation always hits the missing-tile → skip case.

  - **structured logging (`slog`)** (unchanged): `Loop` has a nil-safe `Logger` field; `Run`'s per-tick
    error is an `ErrorContext` emit that does not propagate; `Tick` logs per-hub faults with stable
    `hub_id`+`err` keys; `cmd/iscc-monitor/main.go` builds a `slog.NewTextHandler(os.Stderr)` logger and
    `alertFunc(logger)` is a WARN slog emit. `slog` confined to `loop.go` + `main.go`.

  - **Pure equivocation verifier + proof builder** (`logclient/{consistency,proofbuilder}.go`,
    unchanged): `CheckEquivocation` treats a non-verifying growing-pair proof as a verdict (never a
    crash); `ConsistencyProofFromTiles` is an otel-/net-free port of tessera's
    `ProofBuilder.ConsistencyProof` over the injected `TileFetcher`, WASM-clean, golden-tested across the
    tile boundary.

  - **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
    possible. Documented limitation, not a green-but-wrong path. (The validity window IS re-checked every
    poll via `AcceptCheckpoint`'s `ValidAt`, which gates `StatusVerified` before the cache fast path.)

  - **`SQLiteFetcher` + partial-tile mirror CRUD** (`store/{tiles,fetcher}.go`, unchanged): partial-tile
    discipline (`is_full=1` only at `width==256`, sha256 column, composite-PK upsert, `os.ErrNotExist`
    on a miss), `SQLiteFetcher{Store,HubID}` with the three Fetcher methods, the `widthForP` p↔width
    mapping, partial→full fallback. Store stays a **leaf**: structural `fsck.Fetcher` conformance via a
    local interface copy, no tessera dep leak.

  - **hub_keys cache + key readers**, **coverage tracking** (set-once `monitored_since_{size,time}`,
    ADR-0001), **exported `logclient.Origin`** + golden `TestOrigin`, **config loader leaf**,
    **realm-registry parser** (domains-only, fails closed on URLs), **poll-loop cadence** (single-writer),
    **freeze + alert-once** (gated on `!wasFrozen`) — all unchanged.

  - `store/checkpoints.go` — typed CRUD: `UpsertHub`/`RecordCheckpoint`/`CheckpointAt`/`FollowState`/
    `AdvanceFollowState`/`RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`/`RecordHubKey`/`LookupHubKey`.

  - `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded **nine-table** `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
    `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`).

  - `logclient/{checkpoint,accept,verify,didresolve,origin,keyid,checkpointkey,consistency,proofbuilder}.go`
    — transport-only `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`/`VerifyCheckpoint`, networked
    `ResolveVerifierKey`, all three consistency triggers, and the pure `ConsistencyProofFromTiles` builder.

  - `didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/`DIDKey`
    (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.

  - `tiles/layout.go` — pure leaf re-exporting tessera's `api/layout` primitives + the `IsFull(width)`
    predicate.

  - **Test totals (verified by grep)**: 9 (`didweb`) + 29 (`logclient`) + 48 (`store`) +
    **14 (`follower`)** + 3 (`registry`) + 4 (`config`) + 5 (`tiles`) + 6 (`metrics`) +
    1 (`cmd/iscc-monitor`) = **119 `func Test`**.

- **Missing (remaining M1 connective tissue):**
  - **`/metrics` HTTP handler + `cmd` wiring** — the metrics increment sites now fire on the verdict
    path, but there is no `net/http` handler serving `Registry.WriteText` and `cmd/iscc-monitor/main.go`
    still leaves `Loop.Metrics` nil. This is the immediate next M1 slice and closes the `/metrics`
    Verify/scope criterion.
  - **Real alert transport** — `alertFunc` is a WARN slog emit (a structured signal); real delivery
    (email/webhook) is still a later step. The `AlertFunc func(int64,string)` seam is unchanged.
  - **Live tile ingestion** (M2 dependency, blocks the equivocation branch from firing in production) —
    `PollHub` never writes tiles, so the wired equivocation branch is dormant on the live path. Not
    strictly an M1 Verify gap (the trigger is golden-tested end-to-end against seeded tiles), but it is
    the reason the live path can't yet detect a real equivocation.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — no tiles or entry bundles** (verified `ls`). The equivocation test
  synthesizes mirrored tiles in-process; real on-disk tile fixtures are still needed for the `fsck`
  root-rebuild (M2) and the live tile-ingestion path. **Known stale-fixture drift, still not acted on:**
  the `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and `internal/logclient/`) and
  `derive_vkey.py` still carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`.
  Captured in `verify_test.go` prose/tests (not green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `github.com/transparency-dev/merkle v0.0.2` (`consistency.go` + `proofbuilder.go`),
  `github.com/transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`), and stdlib
  `log/slog` (loop.go + main.go only). `internal/metrics` is stdlib-only. `SQLiteFetcher` conforms to
  `tessera/fsck.Fetcher` **structurally** (local interface copy). **Not yet wired** (confirmed): the rest
  of tessera (`client`/`fsck`), `transparency-dev/formats`, `nbd-wtf/opentimestamps`. `net/http` appears
  only in `logclient/didresolve.go` (the did.json fetcher) + test servers — no monitor HTTP server yet.

- **Verify criteria status**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **All three triggers met end-to-end** in golden tests (synthetic shrink, fork,
  AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — **tracked** and
  asserted set-once. Structured logs — **met**. M1 is not yet *fully* met because **`/metrics` is not yet
  served** (the data model exists, the increment sites now fire, but no HTTP endpoint and `main.go`
  leaves Metrics nil).

## M2 — Aggregator
**Status**: not started — but **three prerequisite slices have landed**: `internal/tiles` re-exports
tessera's tlog-tiles layout math + the `IsFull` predicate; `internal/store/{tiles,fetcher}.go` provides
the partial-tile mirror CRUD + `SQLiteFetcher` (structural `client.Fetcher`/`fsck.Fetcher`); and
`internal/logclient/proofbuilder.go` provides a pure `ConsistencyProofFromTiles` over the same tile-fetch
seam (also consumed by the follower's equivocation branch). What remains for M2: tile/entry-bundle
fixtures + the **live tile-ingestion writer** (which also un-dormants the equivocation branch), the actual
`fsck.New(...).Check(...)` root-rebuild over `SQLiteFetcher`, the `iscc_index` projection writer, and
`inclusion`/`consistency`/`entries` served via a full `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable.
  Latest `review` handoff (2026-06-21, "Thread `*metrics.Registry` into `PollHub`/`Tick` and fire the
  increment sites", verdict **PASS / CONTINUE**) records the gate green at HEAD `111b1a8`:
  `mise run check` green (build + vet + test, all 9 packages); `gofmt -l .` empty; `go.mod`/`go.sum`
  byte-identical to HEAD~1; `go list -deps ./internal/store` net/http-free (store stays a leaf).
  Mutation-proved two asserts non-vacuous. Oracle/conformance gate correctly **N/A** for this slice (no
  verify/proof/consistency/merkle logic line changed; metrics WASM build green, leaf purity intact).
  Gate-integrity scan over the unpushed commits (`origin/develop..HEAD`): no
  `//nolint`/`t.Skip`/build-tag/swallowed-error/deleted-test/deleted-assertion.
- **Two open `normal` issues** (neither blocks the loop, both gate-relevant for the next conformance
  slice): (1) `go mod tidy` adds **22 unstaged go.sum lines** (tessera transitive requires that never
  compile) — harmless today with no CI, but would fail a future `go mod tidy && git diff --exit-code`
  gate; (2) **no `.github/workflows/`** — the external `notecheck` signature-parity oracle and any
  build/test/format/tidy gate run only locally, never in CI.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**, in sync
  with `origin/develop` (0 ahead / 0 behind); tree clean at HEAD `111b1a8`. **No `.github/workflows/` —
  no CI configured** (verified `ls`), so no `gh run` check applies. When CI is wired it must avoid
  `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future `notecheck`
  oracle rather than `go run` from `cauldron/`.

## Next Milestone
Continue M1. The metrics increment sites now fire on the verdict path; the immediate next slice closes
the `/metrics` Verify criterion. Candidate slices, in rough order:
1. **`/metrics` HTTP handler + `cmd` wiring** — a `net/http` handler calling `Registry.WriteText` with
   `Content-Type: text/plain; version=0.0.4`, plus `metrics.New()` in `cmd/iscc-monitor/main.go` passed
   as `Loop.Metrics` and served. This completes M1's `/metrics` deliverable (the increment sites already
   fire; `main.go` still leaves `Metrics` nil and nothing serves the registry).
2. **`fsck` root-rebuild conformance slice** (M2 Verify, also the first real on-disk tile fixtures) —
   `fsck.New(...).Check(...)` over `SQLiteFetcher` + the inclusion cross-check vs the hub's
   `IsccLogInclusionProof`. First slice where the trust-root **oracle gate re-arms** for the mirror path.
   Needs the heavy `fsck`/otel/klog dep + real on-disk tile fixtures under `testdata/live/`. Wire
   CI/`notecheck` and resolve the `go mod tidy` go.sum divergence at the same time.
3. **Live tile-ingestion writer** — make `PollHub` mirror tiles so the wired equivocation branch becomes
   load-bearing on the live path. M2 territory.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
5. **Real alert transport** — replace the WARN slog placeholder with email/webhook delivery to fully
   close M1's alert path.

**CI is still unconfigured** (open `normal` issue): load-bearing now that the merkle crypto + equivocation
path has landed and the upcoming `fsck`-rebuild/fixture slices arm the external `notecheck` oracle — wire
CI **before** the `fsck`-rebuild conformance slice, and resolve the `go mod tidy` go.sum divergence at the
same time so the tidy gate passes.
