<!-- assessed-at: 16ef55f852ef0c94c20588a5221838eb8231bded -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 essentially complete — `/metrics` is now served over HTTP and the binary collects it

M1's last open Verify criterion (`/metrics`) closed this slice: the binary builds a live
`metrics.New()`, threads it into the loop as `Loop.Metrics`, and serves it at GET `/metrics` from a
background goroutine on the configurable `ISCC_MONITOR_ADDR` (default `:9464`). The remaining M1
connective tissue is a real (non-log) alert transport; the equivocation branch is wired and
golden-tested but stays production-dormant until M2's tile-ingestion writer lands. M2/M3/WASM/OTS are
not started. Two open `normal` issues (no CI; `go mod tidy` go.sum divergence) keep the project off
DONE regardless of milestone completeness.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, **`/metrics` now served over HTTP**). A real alert transport and live tile ingestion (the
latter M2) remain as connective tissue but are outside M1's Verify bar.

- **Verified incrementally** from the prior assessment at `111b1a8`. The diff `111b1a8..HEAD` touches
  **only** the `/metrics` HTTP slice: `cmd/iscc-monitor/main.go`, `internal/config/config.go(+_test)`,
  and a new `internal/metricshttp/{handler,handler_test}.go`, plus context files. Confirmed by
  `git diff --stat`: all follower/logclient/store/didweb/tiles/metrics/registry sources, `go.mod`,
  `go.sum`, and `schema.sql` are byte-unchanged and carried forward.

  - **NEW — `/metrics` served + wired (verified by reading the code):**
    - `internal/metricshttp/handler.go` — `Handler(*metrics.Registry) http.Handler`: a thin
      `http.HandlerFunc` that sets `Content-Type: text/plain; version=0.0.4; charset=utf-8` then
      streams `r.WriteText(w)`. Its only internal dep is `internal/metrics` (verified
      `go list -deps ./internal/metricshttp`), so `net/http` stays out of the metrics leaf's closure
      (`go list -deps ./internal/metrics | grep net/http` empty — leaf stays WASM-pure). The mid-write
      `_ = r.WriteText(w)` swallow is justified inline (200 already on the wire). **+1 `func Test`**.
    - `cmd/iscc-monitor/main.go` — constructs `m := metrics.New()`, starts
      `go serveMetrics(ctx, cfg.Addr, m, logger)` (a `net/http.Server` serving exactly `/metrics`,
      shut down via `srv.Shutdown(5s)` on `ctx.Done()`), and passes `m` as `Loop.Metrics` (no longer
      nil). `serveMetrics` logs (never fatals) on a bad addr; `http.ErrServerClosed` is the
      clean-shutdown discriminator. The loop stays the foreground blocker so serving never blocks
      polling.
    - `internal/config/config.go` — adds `Addr` (key `ISCC_MONITOR_ADDR`, default `:9464`) via an
      `optional(get,key,fallback)` absent-OR-empty helper; bad addresses surface at `ListenAndServe`,
      not config. **+0 net `func Test` count change (4)** but two new sub-assertions on `Addr`.
    - Latest `review` handoff (2026-06-21, verdict **PASS / CONTINUE**) records the gate green at this
      HEAD (`16ef55f`): `mise run check` green across all 10 packages, `gofmt -l .` empty,
      `go.mod`/`go.sum` byte-identical, metrics WASM build green. Mutation-proved the handler asserts
      non-vacuous (content-type `0.0.4→0.0.3` fails the test; append `EXTRA` after `WriteText` fails
      body byte-equality) and ran an end-to-end binary smoke (200 on `/metrics`, **404 on `/healthz`**
      = only `/metrics` served, no M3 creep; SIGINT exits 0). Oracle gate correctly N/A (no
      verify/proof/merkle/did:web/tile logic line changed).

  - **`internal/metrics` leaf** (unchanged, carried forward): `*Registry` (RWMutex-guarded) over four
    families — counters `iscc_monitor_violations_total{hub_id,kind}` /
    `iscc_monitor_poll_failures_total{hub_id}` and gauges
    `iscc_monitor_hub_status{hub_id,status}` (set-one-clear-rest) /
    `iscc_monitor_last_observed_at{hub_id}` — rendered in Prometheus text-exposition format. Imports
    stdlib-only `{fmt io sort strconv strings sync}`. 6 `func Test`. The verdict→glossary remap
    (`glossaryStatus`: `rotated`→`unverified`, freeze→`frozen`) lives in the follower and was verified
    correct in the prior slice. `recordVerdict` funnels all three non-error verdict branches;
    `IncPollFailure` fires in `Tick` on `PollHub`'s error return; `IncViolation` fires at freeze.

  - **equivocation trigger WIRED, production-dormant** (unchanged): `checkConsistency` evaluates
    shrink → fork → equivocation; the growing-pair branch builds the RFC-6962 consistency proof from
    the LOCAL mirror (`SQLiteFetcher.ReadTile` → `ConsistencyProofFromTiles` → `CheckEquivocation`)
    and on `true` → freeze. A proof-build error (most often tiles-not-mirrored-yet) is swallowed
    narrowly → branch skipped (ADR-0006 false-positive guard). Golden-tested + mutation-proven across
    the 256-leaf tile boundary over the real `SQLiteFetcher`. **Dormant on the live path until M2's
    tile-ingestion writer lands** — `PollHub` never writes tiles, so a live observation always hits
    the missing-tile skip.

  - **structured logging (`slog`)**, **`SQLiteFetcher` + partial-tile mirror CRUD**, **hub_keys cache
    + key readers**, **coverage tracking** (set-once `monitored_since_{size,time}`, ADR-0001),
    **`logclient.Origin`** + golden `TestOrigin`, **config loader**, **realm-registry parser**
    (domains-only, fails closed on URLs), **poll-loop cadence** (single-writer), **freeze + alert-once**
    (gated on `!wasFrozen`), the pure `ConsistencyProofFromTiles` builder, and the `internal/tiles`
    tlog-tiles seam — all unchanged and carried forward.

  - `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
    `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a leaf
    (`go list -deps ./internal/store | grep net/http` empty — verified).

  - **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
    possible. The window IS re-checked every poll via `AcceptCheckpoint`'s `ValidAt`, which gates
    `StatusVerified` before the cache fast path. Documented limitation, not a green-but-wrong path.

  - **Test totals (verified by grep)**: 9 (`didweb`) + 29 (`logclient`) + 48 (`store`) + 14
    (`follower`) + 3 (`registry`) + 4 (`config`) + 5 (`tiles`) + 6 (`metrics`) + **1 (`metricshttp`)**
    + 1 (`cmd/iscc-monitor`) = **120 `func Test`**.

- **Missing (remaining M1 connective tissue, outside the Verify bar):**
  - **Real alert transport** — `alertFunc` is a WARN `slog` emit (a structured signal); real delivery
    (email/webhook) is still a later step. The `AlertFunc func(int64,string)` seam is unchanged.
  - **Live tile ingestion** (M2 dependency) — `PollHub` never writes tiles, so the wired equivocation
    branch is dormant on the live path. Not an M1 Verify gap (the trigger is golden-tested end-to-end
    against seeded tiles), but it is why the live path can't yet detect a real equivocation.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — no tiles or entry bundles (verified `ls`). The equivocation test
  synthesizes mirrored tiles in-process; real on-disk tile fixtures are still needed for the `fsck`
  root-rebuild (M2) and the live tile-ingestion path. **Known stale-fixture drift, still not acted
  on:** the `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and `internal/logclient/`) and
  `derive_vkey.py` still carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`.
  Captured in `verify_test.go` prose/tests (not green-but-wrong), but the did.json fixtures remain
  stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle v0.0.2` (`consistency.go` + `proofbuilder.go`),
  `transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`), and stdlib `log/slog`
  (loop.go + main.go only) + `net/http` (now in `cmd/main.go` + `metricshttp` + `didresolve.go`).
  `internal/metrics` is stdlib-only. `SQLiteFetcher` conforms to `tessera/fsck.Fetcher` **structurally**
  (local interface copy). **Not yet wired** (confirmed): the rest of tessera (`client`/`fsck`),
  `transparency-dev/formats`, `nbd-wtf/opentimestamps`.

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. **`/metrics` — now served over HTTP and collected by the binary.**

## M2 — Aggregator
**Status**: not started — but **three prerequisite slices have landed** (carried forward):
`internal/tiles` re-exports tessera's tlog-tiles layout math + the `IsFull` predicate;
`internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
(structural `client.Fetcher`/`fsck.Fetcher`); and `internal/logclient/proofbuilder.go` provides a pure
`ConsistencyProofFromTiles` over the same tile-fetch seam (also consumed by the follower's
equivocation branch). What remains for M2: tile/entry-bundle fixtures + the **live tile-ingestion
writer** (which also un-dormants the equivocation branch), the actual `fsck.New(...).Check(...)`
root-rebuild over `SQLiteFetcher`, the `iscc_index` projection writer, and
`inclusion`/`consistency`/`entries` served via a full `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started. (The binary now has a `net/http` mux serving only `/metrics`; `/`, `/healthz`,
and the REST surface are explicitly out of scope until M3.)

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable.
  Latest `review` handoff (2026-06-21, "Serve `/metrics` over HTTP and wire `metrics.New()` into the
  binary", verdict **PASS / CONTINUE**) records the gate green at HEAD `16ef55f`: `mise run check`
  green (build + vet + test, all 10 packages); `gofmt -l .` empty; `go.mod`/`go.sum` byte-identical;
  metrics leaf WASM-pure (`go list -deps ./internal/metrics | grep net/http` empty — independently
  re-verified here). Mutation-proved the handler asserts non-vacuous; end-to-end binary smoke
  confirmed 200 on `/metrics`, 404 on `/healthz`, clean SIGINT. Oracle/conformance gate correctly
  **N/A** for this slice (no verify/proof/consistency/merkle/did:web logic line changed).
- **Two open `normal` issues** (neither blocks the loop, both gate-relevant for the next conformance
  slice): (1) `go mod tidy` adds **22 unstaged go.sum lines** (tessera transitive requires that never
  compile) — **re-confirmed here** (`go mod tidy` → `go.sum` diverges by 22 lines); harmless today
  with no CI, but would fail a future `go mod tidy && git diff --exit-code` gate; (2) **no
  `.github/workflows/`** — the external `notecheck` signature-parity oracle and any build/test/format/
  tidy gate run only locally, never in CI.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**; tree
  clean at HEAD `16ef55f` (review pushed on PASS). **No `.github/workflows/` — no CI configured**
  (verified `ls`), so no `gh run` check applies. When CI is wired it must avoid `go build ./...` over
  the gitignored `cauldron/` reference trees and shell out the future `notecheck` oracle rather than
  `go run` from `cauldron/`.

## Next Milestone
**M2 — Aggregator.** M1 now meets its full Verify bar, so the loop moves to M2. The natural first
slice (per the review handoff) is the **`fsck` root-rebuild conformance slice** — `fsck.New(...).Check(...)`
over `SQLiteFetcher` + the inclusion cross-check vs the hub's `IsccLogInclusionProof`. This is the
first slice where the trust-root **oracle gate re-arms for the mirror path**; it needs the heavy
`fsck`/otel/klog dep and real on-disk tile fixtures under `testdata/live/`.

**Wire CI before that conformance slice** (open `normal` issue, now load-bearing): the upcoming
`fsck`-rebuild/fixture slices arm the external `notecheck` oracle, and `go mod tidy` already diverges
by 22 go.sum lines — resolve the go.sum divergence at the same time so the tidy gate passes. Candidate
order:
1. **Wire CI + `notecheck`** and resolve the `go mod tidy` go.sum divergence (both open issues).
2. **`fsck` root-rebuild conformance slice** (M2 Verify + first real on-disk tile fixtures).
3. **Live tile-ingestion writer** — make `PollHub` mirror tiles so the wired equivocation branch
   becomes load-bearing on the live path.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
5. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
