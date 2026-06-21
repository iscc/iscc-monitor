<!-- assessed-at: 648158069b0e8a6fea407dd3da2faa75ae7760f8 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 Verify bar met; M2 conformance gate landing — the `fsck` root-rebuild self-check (`RunFsck`) is now wired and oracle-mutation-proven, but M2's live ingestion / index / inclusion / served-proofs bar is still not started

Since the prior assessment (`8c5ccfb`) the only source change is the **M2 `fsck` root-rebuild
conformance slice** (`cc818c4`): `internal/logclient/fsck.go` + `fsck_test.go` (+254 lines, 2 new
files), plus the `tessera/fsck` require-graph entering `go.mod`/`go.sum` (additive, tidy-idempotent).
`RunFsck` rebuilds a hub-signed checkpoint root from a `SQLiteFetcher` mirror via `LeafHashes`; the
test synthesizes a 5-leaf in-process log, seeds it into a real `store.SQLiteFetcher`, and proves the
rebuild matches the signed root and rejects a one-byte corruption of either a tile or an entry-bundle
BLOB. M1 still meets its full Verify bar. M2's structural self-check is now in place but its full
Verify bar (live tile ingestion, `iscc_index`, served proofs, inclusion cross-check) remains
not-started. The sole open `normal` issue (no CI / `notecheck` not wired) keeps the project off DONE.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary). A real (non-log) alert transport and
live tile ingestion (the latter an M2 dependency) remain as connective tissue but are outside M1's
Verify bar.

- **Verified incrementally** from the prior assessment at `8c5ccfb`. The diff `8c5ccfb..HEAD` on the
  source side is exactly two new files (`internal/logclient/fsck.go` + `_test.go`) plus additive
  `go.mod`/`go.sum` entries; every other cmd/follower/logclient/store/didweb/tiles/metrics/metricshttp/
  registry/config source and `schema.sql` is byte-unchanged and carried forward. **Test totals
  re-grepped**: 9 (didweb) + 33 (logclient, +1 `TestRunFsck` with 3 subtests) + 48 (store) + 14
  (follower) + 3 (registry) + 4 (config) + 5 (tiles) + 6 (metrics) + 1 (metricshttp) + 1
  (cmd/iscc-monitor) = **124 `func Test`** across **10 packages**.

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
  `sb1.amlet.id_checkpoint`) — no tiles or entry bundles (verified `ls`, unchanged). The equivocation,
  `LeafHashes`, AND the new `RunFsck` tests all synthesize their inputs in-process (`testonly.Tree` +
  manual bundle/tile framing); real on-disk tile/entry-bundle fixtures are still needed for the
  inclusion cross-check vs the hub's `IsccLogInclusionProof` and a 256-crossing live `fsck` case (M2).
  **Known stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json` fixtures (both
  `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py` carry sb1's PRE-rotation key
  (`22b08f3e`); the live sb1 signer is `069d0f14`. Captured in `verify_test.go` prose/tests (not
  green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2`, `transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`; `leafhasher.go` via
  `tessera/api` + `merkle/rfc6962`; and now **`tessera/fsck`** in `logclient/fsck.go`),
  `transparency-dev/formats` (now an indirect via the `tessera/fsck` graph), stdlib `log/slog` +
  `net/http` (cmd/main.go + metricshttp + didresolve.go). `internal/metrics` is stdlib-only.
  `SQLiteFetcher` conforms to `tessera/fsck.Fetcher` **structurally** (store keeps a local interface
  copy; `logclient/fsck.go` consumes the real `fsck.Fetcher` interface). **Not yet wired**: the rest of
  tessera (`client` proof-builder), `nbd-wtf/opentimestamps`.

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: not started — but **five prerequisite slices have now landed** (four carried forward + one
new this iteration):
- `internal/tiles` re-exports tessera's tlog-tiles layout math + the `IsFull` predicate;
- `internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
  (structural `client.Fetcher`/`fsck.Fetcher`);
- `internal/logclient/proofbuilder.go` provides a pure `ConsistencyProofFromTiles` over the same
  tile-fetch seam;
- `internal/logclient/leafhasher.go` — `LeafHashes(bundle []byte) ([][]byte, error)`, the per-entry
  RFC-6962 leaf hasher `fsck.New(...)` takes;
- **NEW** `internal/logclient/fsck.go` — `RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error`,
  thin glue over `note.NewVerifier(vkey)` + `fsck.New(origin, v, f, LeafHashes, fsck.Opts{N:1}).Check`.
  This is the first slice where the **root-rebuild conformance gate re-arms for the mirror path** —
  `RunFsck` rebuilds a checkpoint root from a `SQLiteFetcher` and is mutation-proven (review forced
  `return nil` → both corruption subtests fail). It is an in-process **structural self-check** (shares
  the monitor's own `LeafHashes`/RFC-6962 code), not the fully-independent oracle; `notecheck` remains
  the truly-external oracle (still un-wired, still in CI). `RunFsck` has **no production caller yet** —
  an intentional unused-until-wired seam (its first caller needs the live tile-ingestion writer).

What remains for M2's Verify bar (all not-started): the **live tile-ingestion writer** (make `PollHub`
mirror real tiles/bundles — also un-dormants the equivocation branch and enables a *production*
`RunFsck` caller); the **inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof`
(needs real captured `IsccLogInclusionProof` fixtures + an inclusion `ProofBuilder` — the SECOND half
of M2's Verify); the `iscc_index` projection writer (schema-agnostic, `iscc_id → seq` one-to-many);
and `inclusion`/`consistency`/`entries` served via a full `ProofBuilder` from the local store (never
re-hitting the hub).

## M3 — Trust API + dashboard
**Status**: not started. (The binary has a `net/http` mux serving only `/metrics`; `/`, `/healthz`, and
the REST surface are explicitly out of scope until M3.)

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable.
  Latest `review` handoff (2026-06-21, "Wire `LeafHashes` + `SQLiteFetcher` into `fsck.New(...).
  Check(...)`", verdict **PASS / CONTINUE**) records the gate green at HEAD `648158069`: `mise run
  check` green (build + vet + test, all 10 packages `ok`); `gofmt -l .` empty; `go test -count=1 -run
  TestRunFsck ./internal/logclient` PASS uncached (3 subtests — `RebuildsSignedRoot` nil,
  `RejectsCorruptedTile`/`RejectsCorruptedBundle` non-nil; klog root `00d21829…`); `go mod tidy &&
  git diff --exit-code -- go.mod go.sum` exits 0; `go mod verify` → all modules verified. Oracle gate
  **APPLIED** (RFC-6962 root-rebuild crypto) and was satisfied + **independently mutation-proven** by
  the reviewer (forced `RunFsck` → `return nil` ⇒ both corruption subtests fail); `notecheck`
  (fully-independent oracle) still N/A only because it is un-wired. Trust-root goldens reproduce —
  `derive_vkey.py` prints both vectors (`40b74463`/`22b08f3e`) byte-for-byte.
- **`go mod tidy` is idempotent**: the `tessera/fsck` require-graph entered cleanly (klog/otel/formats
  in go.mod indirect; testify-family testify/go-spew/difflib/yaml.v3 in go.sum only as transitive
  test-deps). `go mod tidy && git diff --exit-code -- go.mod go.sum` exits 0.
- **One open `normal` issue**: no `.github/workflows/` (verified `ls`) — the external `notecheck`
  signature-parity oracle and any build/test/format/tidy gate run only locally, never in CI. Now that
  the mirror path exercises the trust root structurally (`RunFsck`) but the truly-independent
  `notecheck` runs only in CI, this is the natural next slice.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**; tree
  clean at HEAD `648158069` (review pushed on PASS). **No `.github/workflows/` — no CI configured**;
  `gh run list --branch develop` returns `[]` (no runs), confirming no CI applies. When CI is wired it
  must avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future
  `notecheck` oracle rather than `go run` from `cauldron/`.

## Next Milestone
**M2 — Aggregator.** M1 meets its full Verify bar and the `fsck` root-rebuild self-check is now landed,
so the loop moves toward closing M2's remaining Verify bar. The structural rebuild oracle is armed but
fully-independent CI verification (`notecheck`) is still missing — that gap takes priority.

Candidate order:
1. **Wire CI + `notecheck`** (the sole open `normal` issue, the only gate-relevant gap) —
   `.github/workflows/` running `mise run check` + shelling out the external `notecheck`
   signature-parity oracle, avoiding `go build ./...` over the gitignored `cauldron/`. The tree is
   tidy-clean so a `go mod tidy && git diff --exit-code` CI step will pass.
2. **Inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof` — the SECOND half of
   M2's Verify bar; needs real captured `IsccLogInclusionProof` fixtures + an inclusion `ProofBuilder`.
3. **Live tile-ingestion writer** — make `PollHub` mirror real tiles/bundles, which un-dormants the
   wired equivocation branch on the live path AND gives `RunFsck` its first production caller.
4. **`iscc_index` projection writer** + serving `inclusion`/`consistency`/`entries` from the local
   store via a full `ProofBuilder`.
5. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
6. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
