<!-- assessed-at: 8c5ccfb43c3d0e749485e66f2076e19dab57c099 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 Verify bar met; M2 prerequisites accumulating — last slice landed the `fsck` leaf-hasher, next is the real `fsck` root-rebuild conformance slice

Since the prior assessment (`d837649`) the only source change is the **M2 `fsck` leaf-hasher slice**
(`a260635`): `internal/logclient/leafhasher.go` + its test (+129 lines, 2 new files). `go.mod`,
`go.sum`, `schema.sql`, and `testdata` are byte-unchanged (`git diff d837649..HEAD --stat` touches only
those two files plus context). M1 still meets its full Verify bar; M2 remains not-started but gains a
fourth prerequisite. The sole open `normal` issue (no CI / `notecheck` not wired) keeps the project off
DONE.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary). A real (non-log) alert transport and
live tile ingestion (the latter an M2 dependency) remain as connective tissue but are outside M1's
Verify bar.

- **Verified incrementally** from the prior assessment at `d837649`. The diff `d837649..HEAD` on the
  source side is exactly two new files (`internal/logclient/leafhasher.go` + `_test.go`); every other
  cmd/follower/logclient/store/didweb/tiles/metrics/metricshttp/registry/config source, `go.mod`,
  `go.sum`, and `schema.sql` is byte-unchanged and carried forward. **Test totals re-grepped**: 9
  (didweb) + 32 (logclient, +3 `LeafHashes` tests) + 48 (store) + 14 (follower) + 3 (registry) + 4
  (config) + 5 (tiles) + 6 (metrics) + 1 (metricshttp) + 1 (cmd/iscc-monitor) = **123 `func Test`**
  across **10 packages**.

  - **`/metrics` served + wired** (carried forward, verified unchanged): `internal/metricshttp/
    handler.go` — `Handler(*metrics.Registry) http.Handler` sets the exposition Content-Type then
    streams `r.WriteText(w)`; its only internal dep is `internal/metrics`, keeping the metrics leaf
    WASM-pure. `cmd/iscc-monitor/main.go` constructs `m := metrics.New()`, serves it from a background
    `net/http.Server` (`srv.Shutdown(5s)` on `ctx.Done()`) on `ISCC_MONITOR_ADDR` (default `:9464`),
    and passes `m` as `Loop.Metrics`. `internal/config` adds `Addr` (default `:9464`).

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
  and `LeafHashes` tests synthesize their inputs in-process; real on-disk tile/entry-bundle fixtures are
  still needed for the `fsck` root-rebuild (M2). **Known stale-fixture drift, still not acted on:** the
  `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and `internal/logclient/`) and
  `derive_vkey.py` carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`.
  Captured in `verify_test.go` prose/tests (not green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2`, `transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`; now also
  `leafhasher.go` via `tessera/api` + `merkle/rfc6962`), stdlib `log/slog` + `net/http` (cmd/main.go +
  metricshttp + didresolve.go). `internal/metrics` is stdlib-only. `SQLiteFetcher` conforms to
  `tessera/fsck.Fetcher` **structurally** (local interface copy). **Not yet wired**: the rest of tessera
  (`client`/`fsck`), `transparency-dev/formats`, `nbd-wtf/opentimestamps`.

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: not started — but **four prerequisite slices have landed** (carried forward + one new):
`internal/tiles` re-exports tessera's tlog-tiles layout math + the `IsFull` predicate;
`internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
(structural `client.Fetcher`/`fsck.Fetcher`); `internal/logclient/proofbuilder.go` provides a pure
`ConsistencyProofFromTiles` over the same tile-fetch seam; and **NEW** `internal/logclient/leafhasher.go`
— `LeafHashes(bundle []byte) ([][]byte, error)`, a verbatim-in-shape port of `runfsck`'s `leafHasher`
(`api.EntryBundle{}.UnmarshalText` then `rfc6962.DefaultHasher.HashLeaf` per entry), the hasher
`fsck.New(...)` takes. Golden-tested with an independent third-path framer (mutation-proven non-vacuous).
What remains for M2: tile/entry-bundle fixtures + the **live tile-ingestion writer** (which also
un-dormants the equivocation branch), the actual `fsck.New(...).Check(...)` root-rebuild over
`SQLiteFetcher` (wiring `LeafHashes` + `SQLiteFetcher` in), the inclusion cross-check vs the hub's
`IsccLogInclusionProof`, the `iscc_index` projection writer, and `inclusion`/`consistency`/`entries`
served via a full `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started. (The binary has a `net/http` mux serving only `/metrics`; `/`, `/healthz`, and
the REST surface are explicitly out of scope until M3.)

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable.
  Latest `review` handoff (2026-06-21, "Port the M2 fsck leaf-hasher (`LeafHashes`)", verdict
  **PASS / CONTINUE**) records the gate green at HEAD `8c5ccfb`: `mise run check` green (build + vet +
  test, all 10 packages `ok`); `gofmt -l .` empty; `GOOS=js GOARCH=wasm go build ./internal/logclient`
  exits 0 (file-level WASM purity holds); `go test -run TestLeafHashes` 3 subtests PASS uncached;
  `git diff HEAD~1..HEAD -- go.mod go.sum` exits 0 (no new dep). Oracle gate APPLIED (RFC-6962
  leaf-hash crypto) and was satisfied by independent in-test ground truth (third framing path,
  mutation-proven); `notecheck`/`derive_vkey.py`/`fsck` correctly N/A for this slice; trust-root
  conformance (didweb/logclient/follower) unchanged and `derive_vkey.py` reproduces both vectors
  (`40b74463`/`22b08f3e`).
- **`go mod tidy` is idempotent** (resolved earlier, unchanged): the 22 tessera module-graph go.sum
  lines are committed, `go mod tidy && git diff --exit-code -- go.sum` exits 0.
- **One open `normal` issue**: no `.github/workflows/` (verified `ls`) — the external `notecheck`
  signature-parity oracle and any build/test/format/tidy gate run only locally, never in CI. Natural
  companion to the M2 `fsck` slice (which first arms the mirror-path trust-root oracle).
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**; tree clean
  at HEAD `8c5ccfb` (review pushed on PASS). **No `.github/workflows/` — no CI configured**, so no
  `gh run` check applies. When CI is wired it must avoid `go build ./...` over the gitignored
  `cauldron/` reference trees and shell out the future `notecheck` oracle rather than `go run` from
  `cauldron/`.

## Next Milestone
**M2 — Aggregator.** M1 meets its full Verify bar; the `fsck` leaf-hasher prerequisite is now in place,
so the loop moves toward the **`fsck` root-rebuild conformance slice** — wire `LeafHashes` +
`SQLiteFetcher` into `fsck.New(...).Check(...)` over real on-disk tile/entry-bundle fixtures under
`testdata/live/`, plus the inclusion cross-check against the hub's own `IsccLogInclusionProof`. This is
the first slice where the trust-root **oracle gate re-arms for the mirror path**; it pulls the heavy
`fsck`/`net/http`/otel/klog deps and needs the first real on-disk tile fixtures.

**Wire CI before that conformance slice** (the remaining open `normal` issue, the sole gate-relevant
gap): the upcoming `fsck`-rebuild/fixture slices arm the external `notecheck` oracle, and the tree is
tidy-clean so a `go mod tidy && git diff --exit-code` CI step will pass. Candidate order:
1. **Wire CI + `notecheck`** — `.github/workflows/` running `mise run check` + shelling out `notecheck`,
   avoiding `go build ./...` over gitignored `cauldron/`.
2. **`fsck` root-rebuild conformance slice** (M2 Verify + first real on-disk tile/entry-bundle fixtures).
3. **Live tile-ingestion writer** — make `PollHub` mirror tiles so the wired equivocation branch
   becomes load-bearing on the live path.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
5. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
