<!-- assessed-at: d837649c1ca6ad90dd632ecb62f6fe9b73a97249 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 Verify bar met; `go mod tidy` now idempotent — ready for the M2 `fsck` conformance slice

Since the prior assessment the only source change is a **go.sum-only commit** (`79e5e37`, +22 lines, 0
removed): the tessera module-graph checksums that `go mod tidy` previously added unstaged are now
committed, so `go mod tidy && git diff --exit-code -- go.sum` exits 0 (verified idempotent). No `.go`,
`go.mod`, `schema.sql`, or `testdata` path changed (`git diff --stat 16ef55f..HEAD` touches only
`go.sum` + context files). M1 still meets its full Verify bar; M2/M3/WASM/OTS are not started. **One**
open `normal` issue remains (no CI / `notecheck` not wired), keeping the project off DONE.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary). A real (non-log) alert transport and
live tile ingestion (the latter an M2 dependency) remain as connective tissue but are outside M1's
Verify bar.

- **Verified incrementally** from the prior assessment at `16ef55f`. The diff `16ef55f..HEAD` is
  go.sum-only on the source side (+22 checksum lines, confirmed by `git diff --stat`): all
  cmd/follower/logclient/store/didweb/tiles/metrics/metricshttp/registry/config sources, `go.mod`, and
  `schema.sql` are byte-unchanged and carried forward. **Test totals re-grepped and unchanged**: 9
  (didweb) + 29 (logclient) + 48 (store) + 14 (follower) + 3 (registry) + 4 (config) + 5 (tiles) + 6
  (metrics) + 1 (metricshttp) + 1 (cmd/iscc-monitor) = **120 `func Test`** across **10 packages**.

  - **`/metrics` served + wired** (carried forward, verified unchanged): `internal/metricshttp/
    handler.go` — `Handler(*metrics.Registry) http.Handler` sets the exposition Content-Type then
    streams `r.WriteText(w)`; its only internal dep is `internal/metrics`, keeping the metrics leaf
    WASM-pure. `cmd/iscc-monitor/main.go` constructs `m := metrics.New()`, serves it from a background
    `net/http.Server` (`srv.Shutdown(5s)` on `ctx.Done()`) on `ISCC_MONITOR_ADDR` (default `:9464`),
    and passes `m` as `Loop.Metrics`. `internal/config` adds `Addr` (default `:9464`). Latest review
    smoke confirmed 200 on `/metrics`, 404 on `/healthz` (only `/metrics` served — no M3 creep), clean
    SIGINT.

  - **`internal/metrics` leaf** (unchanged): `*Registry` (RWMutex-guarded) over four families —
    counters `iscc_monitor_violations_total{hub_id,kind}` / `iscc_monitor_poll_failures_total{hub_id}`
    and gauges `iscc_monitor_hub_status{hub_id,status}` (set-one-clear-rest) /
    `iscc_monitor_last_observed_at{hub_id}` — rendered in Prometheus text format, stdlib-only. The
    verdict→glossary remap (`glossaryStatus`: rotated→unverified, freeze→frozen) lives in the follower.

  - **equivocation trigger WIRED, production-dormant** (unchanged): `checkConsistency` evaluates
    shrink → fork → equivocation; the growing-pair branch builds the RFC-6962 consistency proof from
    the LOCAL mirror (`SQLiteFetcher.ReadTile` → `ConsistencyProofFromTiles` → `CheckEquivocation`) and
    on `true` → freeze. A proof-build error (tiles-not-mirrored-yet) is swallowed narrowly → branch
    skipped (ADR-0006 false-positive guard). Golden-tested + mutation-proven across the 256-leaf tile
    boundary. **Dormant on the live path until M2's tile-ingestion writer lands** — `PollHub` never
    writes tiles, so a live observation always hits the missing-tile skip.

  - **structured logging (`slog`)**, **`SQLiteFetcher` + partial-tile mirror CRUD**, **hub_keys cache +
    key readers**, **coverage tracking** (set-once `monitored_since_{size,time}`, ADR-0001),
    **`logclient.Origin`** + golden `TestOrigin`, **config loader**, **realm-registry parser**
    (domains-only), **poll-loop cadence** (single-writer), **freeze + alert-once** (gated on
    `!wasFrozen`), the pure `ConsistencyProofFromTiles` builder, and the `internal/tiles` tlog-tiles
    seam — all unchanged and carried forward.

  - `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
    `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a leaf.

  - **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
    possible. The window IS re-checked every poll via `AcceptCheckpoint`'s `ValidAt`, which gates
    `StatusVerified` before the cache fast path. Documented limitation, not a green-but-wrong path.

- **Missing (remaining M1 connective tissue, outside the Verify bar):**
  - **Real alert transport** — `alertFunc` is a WARN `slog` emit; real delivery (email/webhook) is a
    later step. The `AlertFunc func(int64,string)` seam is unchanged.
  - **Live tile ingestion** (M2 dependency) — `PollHub` never writes tiles, so the wired equivocation
    branch is dormant on the live path. Not an M1 Verify gap (the trigger is golden-tested end-to-end
    against seeded tiles), but it is why the live path can't yet detect a real equivocation.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — no tiles or entry bundles (verified `ls`, unchanged). The equivocation
  test synthesizes mirrored tiles in-process; real on-disk tile fixtures are still needed for the
  `fsck` root-rebuild (M2). **Known stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json`
  fixtures (both `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py` carry sb1's
  PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`. Captured in `verify_test.go`
  prose/tests (not green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2`, `transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`), stdlib `log/slog`
  + `net/http` (cmd/main.go + metricshttp + didresolve.go). `internal/metrics` is stdlib-only.
  `SQLiteFetcher` conforms to `tessera/fsck.Fetcher` **structurally** (local interface copy). **Not yet
  wired**: the rest of tessera (`client`/`fsck`), `transparency-dev/formats`, `nbd-wtf/opentimestamps`.
  (The 22 new go.sum lines record formats/otel/klog/backoff/x-crypto module-graph requires of
  `tessera/api` that are NOT compiled into any monitor package — `go mod why -m` traces through
  `tessera/api.test`, never our packages.)

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: not started — but **three prerequisite slices have landed** (carried forward):
`internal/tiles` re-exports tessera's tlog-tiles layout math + the `IsFull` predicate;
`internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
(structural `client.Fetcher`/`fsck.Fetcher`); and `internal/logclient/proofbuilder.go` provides a pure
`ConsistencyProofFromTiles` over the same tile-fetch seam (also consumed by the follower's equivocation
branch). What remains for M2: tile/entry-bundle fixtures + the **live tile-ingestion writer** (which
also un-dormants the equivocation branch), the actual `fsck.New(...).Check(...)` root-rebuild over
`SQLiteFetcher`, the `iscc_index` projection writer, and `inclusion`/`consistency`/`entries` served via
a full `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started. (The binary has a `net/http` mux serving only `/metrics`; `/`, `/healthz`, and
the REST surface are explicitly out of scope until M3.)

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable.
  Latest `review` handoff (2026-06-21, "Commit the 22 `go mod tidy` go.sum lines so the tidy gate is
  idempotent", verdict **PASS / CONTINUE**) records the gate green at HEAD `d837649`: `mise run check`
  green (build + vet + test, all 10 packages); `gofmt -l .` empty; readonly build+test green;
  `go mod verify` → all modules verified. Trust root intact (didweb/logclient/follower conformance
  uncached + `derive_vkey.py` reproduces both golden vectors `40b74463`/`22b08f3e`). Oracle/conformance
  gate correctly **N/A** for this diff (touches no signature/RFC-6962/Merkle/proof/did:web code).
- **`go mod tidy` is now idempotent — RESOLVED.** The prior open `normal` issue (`go mod tidy` adding 22
  unstaged go.sum lines) is closed: those checksums are committed, `go mod tidy && git diff --exit-code
  -- go.sum` exits 0, and the recorded checksums were independently confirmed genuine via a proxy
  re-fetch (`go mod download -x`), not a local-cache tautology. A future `go mod tidy && git diff
  --exit-code` CI gate will pass on these lines.
- **One open `normal` issue**: no `.github/workflows/` — the external `notecheck` signature-parity
  oracle and any build/test/format/tidy gate run only locally, never in CI. Natural companion to the M2
  `fsck` slice (which first arms the mirror-path trust-root oracle).
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**; tree clean
  at HEAD `d837649` (review pushed on PASS). **No `.github/workflows/` — no CI configured** (verified
  `ls`), so no `gh run` check applies. When CI is wired it must avoid `go build ./...` over the
  gitignored `cauldron/` reference trees and shell out the future `notecheck` oracle rather than `go
  run` from `cauldron/`.

## Next Milestone
**M2 — Aggregator.** M1 meets its full Verify bar and the tree is now tidy-clean, so the loop moves to
M2. The natural first slice (per the review handoff) is the **`fsck` root-rebuild conformance slice** —
`fsck.New(...).Check(...)` over `SQLiteFetcher` + the inclusion cross-check vs the hub's
`IsccLogInclusionProof`. This is the first slice where the trust-root **oracle gate re-arms for the
mirror path**; it needs the heavy `fsck`/otel/klog dep and real on-disk tile fixtures under
`testdata/live/`.

**Wire CI before that conformance slice** (the remaining open `normal` issue, now the sole gate-relevant
gap): the upcoming `fsck`-rebuild/fixture slices arm the external `notecheck` oracle, and the tree is
now tidy-clean so a `go mod tidy && git diff --exit-code` CI step will pass. Candidate order:
1. **Wire CI + `notecheck`** (the one remaining open issue) — `.github/workflows/` running `mise run
   check` + shelling out `notecheck`, avoiding `go build ./...` over gitignored `cauldron/`.
2. **`fsck` root-rebuild conformance slice** (M2 Verify + first real on-disk tile fixtures).
3. **Live tile-ingestion writer** — make `PollHub` mirror tiles so the wired equivocation branch
   becomes load-bearing on the live path.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
5. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
