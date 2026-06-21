<!-- assessed-at: 8fd09e6d7ae82e23ea7f49a11a075280af9231eb -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 closing — all three self-consistency triggers (shrink / fork / equivocation) are wired into the follower and golden-tested end-to-end, and **structured logging (`slog`) now landed** at the loop + binary boundary (the previously swallowed per-tick error is now an observable ERROR record; the freeze alert is a WARN slog emit). The two remaining M1 gaps are `/metrics` and a real (non-log) alert transport; plus the equivocation branch stays **dormant in production** until M2's tile-ingestion writer lands.

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → all three freeze triggers), the single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, the full `hub_keys` did:web key cache, the `internal/tiles` tlog-tiles layout seam, the `internal/store` partial-tile mirror CRUD + `SQLiteFetcher`, the pure `ConsistencyProofFromTiles` builder, and now structured `slog` logging have all landed. The remaining M1 connective tissue is `/metrics` and a real alert transport.

## M1 — Read-only Monitor
**Status**: partially met (all three freeze triggers — shrink / fork / equivocation — wired + golden-tested end-to-end; verify primitives + poll-loop cadence + realm-registry parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache + tlog-tiles seam + `SQLiteFetcher` + `ConsistencyProofFromTiles` + **structured `slog` logging**; **`/metrics` + real alert transport remain missing**)

- **Verified incrementally** from the prior assessment at `0a6d513`. The diff `0a6d513..HEAD` touches
  **only** the composition layer: `cmd/iscc-monitor/main.go` + `internal/follower/loop.go` (production)
  plus one new test `internal/follower/logging_test.go`, plus `.gitignore` and context files. Confirmed
  byte-unchanged (`git diff --quiet` exit 0): all six leaf packages (`internal/{store,logclient,didweb,
  tiles,config,registry}`) and `go.mod`/`go.sum`/`schema.sql`. All sections other than the slog wiring
  are re-confirmed unchanged and carried forward from the prior assessment.

  - **NEW — structured logging (`slog`) at the loop + binary boundary** (verified by reading both files):
    - `internal/follower/loop.go`: `Loop` gained a nil-safe `Logger *slog.Logger` field + unexported
      `logger()` accessor (falls back to `slog.Default()`), so every bare `&Loop{…}` still compiles. The
      previously `_ =`-discarded per-tick error in `Run` is now an `ErrorContext(ctx, "poll tick failed",
      "err", …)` emit that **still does not propagate** (log-and-continue invariant intact — confirmed the
      `_ = l.Tick(…)` line is gone, replaced by an `if err != nil { logger().ErrorContext }` branch with no
      `return`). `Tick` logs each per-hub fault with stable keys `hub_id`+`err` at two sites
      (`"poll hub failed"` loop.go:119, `"follow state read failed"` loop.go:109).
    - `cmd/iscc-monitor/main.go`: `run()` builds a `slog.NewTextHandler(os.Stderr)` logger,
      `slog.SetDefault`s it, injects it into the `Loop`, and replaces the old stderr-`Fprintf` `alert` with
      `alertFunc(logger)` — a WARN slog emit (`logger.Warn("hub frozen", "hub_id", …, "kind", …)`). The old
      `func alert(…)` is fully removed.
    - **+2 `func Test`** (follower 10 → 12): `TestLoopLogsTickError` (a faulting `Tick` through the real
      outbound-fetch seam emits exactly one ERROR record with populated `hub_id`+`err`) + `TestLoopLoggerNilSafe`
      (a bare `&Loop{}` with no Logger does not panic). `slog` appears **only** in `loop.go` + `main.go`
      (+ the test) — confirmed zero `log/slog` in any leaf package. `go.mod`/`go.sum` byte-identical
      (stdlib-only change). Latest `review` handoff (2026-06-21, verdict **PASS / CONTINUE**) records the
      gate green at this HEAD and probed the record stream independently (exactly one record, non-vacuous).

  - **equivocation trigger WIRED into the follower** (`internal/follower/follower.go`, unchanged this slice):
    `checkConsistency` evaluates **shrink → fork → equivocation**; the third (growing-pair) branch builds
    the RFC-6962 consistency proof from the LOCAL mirror (`store.SQLiteFetcher{Store,HubID}.ReadTile` →
    `logclient.ConsistencyProofFromTiles(ctx, …, prevSize, info.TreeSize)` → `logclient.CheckEquivocation`)
    and on a `true` verdict → `ViolationEquivocation` → `freeze` (RecordViolation + evidence RecordCheckpoint
    + Freeze + alert-once). A proof-build error (most often **tiles not mirrored yet**) is swallowed narrowly
    → branch skipped, never freezes on a missing tile (ADR-0006 false-positive guard). Golden-tested end-to-end
    (`TestPollHubEquivocation` + `TestEquivocationMissingTilesDoesNotFreeze`), mutation-proven non-vacuous by
    `review`, crosses the 256-leaf tile boundary over the real `SQLiteFetcher`.

  - **PRODUCTION-DORMANT caveat** (carried forward): the equivocation branch is wired but **dormant in the
    live path** until M2's tile-ingestion writer lands. Today `PollHub` never writes tiles, so a live
    observation always hits the missing-tile → skip case. Fully proven against in-test synthesized mirrored
    tiles; becomes load-bearing the instant the M2 tile writer ships.

  - **Pure equivocation verifier + proof builder** (`logclient/{consistency,proofbuilder}.go`, unchanged):
    `CheckEquivocation` treats a non-verifying growing-pair proof as a verdict (`violated=true`), never a
    crash; `ConsistencyProofFromTiles` is an otel-/net-free port of tessera's `ProofBuilder.ConsistencyProof`
    over the injected `TileFetcher` (signature byte-identical to `store.SQLiteFetcher.ReadTile`), WASM-clean,
    golden-tested across the tile boundary against a 300-leaf `testonly.Tree`.

  - **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
    possible. Documented limitation, not a green-but-wrong path. (Validity window IS re-checked every poll
    via `AcceptCheckpoint`'s `ValidAt`, which gates `StatusVerified` before the cache fast path runs.)

  - **`SQLiteFetcher` + partial-tile mirror CRUD** (`store/{tiles,fetcher}.go`, unchanged): partial-tile
    discipline (`is_full=1` only at `width==256`, sha256 column, composite-PK upsert, `os.ErrNotExist` on a
    miss), `SQLiteFetcher{Store,HubID}` with the three Fetcher methods, the `widthForP` p↔width mapping, and
    the partial→full fallback. Store stays a **leaf**: structural `fsck.Fetcher` conformance via a local
    interface copy + `var _ fsckFetcher = SQLiteFetcher{}`, no tessera dep leak.

  - **hub_keys cache + key readers** (unchanged): `cacheHubKeyFast` (fetch-free row refresh on a `LookupHubKey`
    hit) + `cacheHubKeyResolve` cold fallback; `KeyIDFromCheckpoint` golden (`name=="sb0.iscc.id/log"`,
    `keyID==0x40b74463`); `LookupHubKey(ctx, hubID, keyID)`.

  - **coverage tracking** (unchanged): set-once `monitored_since_{size,time}` (ADR-0001), wired between
    `RecordCheckpoint` and `AdvanceFollowState` on the verified non-violation path only.

  - **exported `logclient.Origin`** + golden `TestOrigin`; **config loader leaf**; **realm-registry parser**
    (domains-only, fails closed on URLs); **poll-loop cadence** (`loop.go`, single-writer); **freeze +
    alert-once** (gated on `!wasFrozen`).

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

  - `tiles/layout.go` — pure leaf re-exporting tessera's `api/layout` primitives + the `IsFull(width)` predicate.

  - **Test totals (verified by grep)**: 9 (`didweb`) + 29 (`logclient`) + 48 (`store`) +
    **12 (`follower`, +2)** + 3 (`registry`) + 4 (`config`) + 5 (`tiles`) + 1 (`cmd/iscc-monitor`) =
    **111 `func Test`**.

- **Missing (remaining M1 connective tissue):**
  - **`/metrics`** — no metrics impl, no `expvar`/prometheus, no http server (confirmed by grep: no
    `expvar`/`ListenAndServe`/`/metrics`/`prometheus` anywhere in `internal`/`cmd`). With the injected
    `Logger` and the `firstErr`/`PollHub` fault points now in place, this is the natural next M1 slice.
  - **Real alert transport** — `alertFunc` is a WARN slog emit (a structured signal, no longer a stderr
    placeholder); real delivery (email/webhook) is still a later step. The `AlertFunc func(int64,string)`
    seam is unchanged.
  - **Live tile ingestion** (M2 dependency, blocks the equivocation branch from firing in production) —
    `PollHub` never writes tiles, so the wired equivocation branch is dormant on the live path. Not strictly
    an M1 Verify gap (the trigger is golden-tested end-to-end against seeded tiles), but it is the reason the
    live path can't yet detect a real equivocation.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — no tiles or entry bundles** (verified `ls`). The equivocation test synthesizes
  mirrored tiles in-process; real on-disk tile fixtures are still needed for the `fsck` root-rebuild (M2) and
  the live tile-ingestion path. **Known stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json`
  fixtures (both `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py` still carry sb1's
  PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`. Captured in `verify_test.go` prose/tests
  (not green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `github.com/transparency-dev/merkle v0.0.2` (`consistency.go` + `proofbuilder.go`),
  `github.com/transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`), and stdlib `log/slog`
  (loop.go + main.go only). `SQLiteFetcher` conforms to `tessera/fsck.Fetcher` **structurally** (local
  interface copy). **Not yet wired** (confirmed zero non-comment imports): the rest of tessera
  (`client`/`fsck`), `transparency-dev/formats`, `nbd-wtf/opentimestamps` (grep: no `opentimestamps`).

- **Verify criteria status**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **All three triggers met end-to-end** in golden tests (synthetic shrink, fork, AND
  equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected +
  evidence-survives-restart). Coverage (`monitored_since`) — **tracked** and asserted set-once. **Structured
  logs — now met** (`slog` at the loop/binary boundary). M1 is not yet *fully* met because `/metrics`
  (in the M1 scope sentence) is still absent.

## M2 — Aggregator
**Status**: not started — but **three prerequisite slices have landed**: `internal/tiles` re-exports
tessera's tlog-tiles layout math + the `IsFull` predicate; `internal/store/{tiles,fetcher}.go` provides the
partial-tile mirror CRUD + `SQLiteFetcher` (structural `client.Fetcher`/`fsck.Fetcher`); and
`internal/logclient/proofbuilder.go` provides a pure `ConsistencyProofFromTiles` over the same tile-fetch
seam (also consumed by the follower's equivocation branch). What remains for M2: tile/entry-bundle fixtures +
the **live tile-ingestion writer** (which also un-dormants the equivocation branch), the actual
`fsck.New(...).Check(...)` root-rebuild over `SQLiteFetcher`, the `iscc_index` projection writer, and
`inclusion`/`consistency`/`entries` served via a full `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable. Latest
  `review` handoff (2026-06-21, "Structured logging (`slog`) at the loop + binary boundary", verdict
  **PASS / CONTINUE**) records the gate green at HEAD `8fd09e6`: `mise run check` green (build + vet + test,
  all 8 packages ok), `gofmt -l .` empty, `go.mod`/`go.sum` + all six leaf packages byte-identical to HEAD~1,
  `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (WASM purity preserved), `log/slog` confined to
  loop.go + main.go. Oracle/conformance gate correctly **N/A** for this slice (stdlib `slog` at the loop/binary
  layer only — no signature/RFC-6962/Merkle/did:web/proof/tile path touched). Gate-integrity scan over the
  unpushed commits: no `//nolint`/`t.Skip`/build-tag exclusion/swallowed-error dodge/deleted assertion (the
  `_ = l.Tick(…)` discard is genuinely gone, replaced by a non-propagating logging branch).
- **Two open `normal` issues** (neither blocks the loop, both gate-relevant for the next conformance slice):
  (1) `go mod tidy` adds **22 unstaged go.sum lines** (tessera transitive requires that never compile) —
  harmless today with no CI, but would fail a future `go mod tidy && git diff --exit-code` gate; (2) **no
  `.github/workflows/`** — the external `notecheck` signature-parity oracle and any build/test/format/tidy
  gate run only locally, never in CI.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**, in sync with
  `origin/develop` (0 ahead / 0 behind); tree clean at HEAD `8fd09e6`. **No `.github/workflows/` — no CI
  configured** (verified `ls`), so no `gh run` check applies. When CI is wired it must avoid `go build ./...`
  over the gitignored `cauldron/` reference trees and shell out the future `notecheck` oracle rather than
  `go run` from `cauldron/`.

## Next Milestone
Continue M1. All three self-consistency triggers are wired + golden-tested and structured logging has landed;
the remaining M1 gaps are `/metrics` plus the M2-prerequisite that un-dormants equivocation in production.
Candidate slices, in rough order:
1. **`/metrics` (expvar/HTTP)** — the natural M1 follow-on now that the injected `Logger` and the
   `firstErr`/`PollHub` fault points are the metric-increment sites. Completes M1's `/metrics` Verify/scope
   criterion with a small, leaf-free composition-layer slice.
2. **`fsck` root-rebuild conformance slice** (M2 Verify, also the first real on-disk tile fixtures) —
   `fsck.New(...).Check(...)` over `SQLiteFetcher` + the inclusion cross-check vs the hub's
   `IsccLogInclusionProof`. First slice where the trust-root **oracle gate re-arms** for the mirror path.
   Needs the heavy `fsck`/otel/klog dep (copy/scope as the `SQLiteFetcher` slice did) + real on-disk tile
   fixtures under `testdata/live/`. Wire CI/`notecheck` and resolve the `go mod tidy` go.sum divergence at
   the same time so the trust-root oracle has CI coverage when the mirror path first faces it.
3. **Live tile-ingestion writer** — make `PollHub` mirror tiles so the wired equivocation branch becomes
   load-bearing on the live path (today it always hits the missing-tile skip). M2 territory, but what turns
   the in-test-proven equivocation trigger into a real detector.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`) —
   its own trust-root step that re-arms the oracle gate.
5. **Real alert transport** — replace the WARN slog placeholder with email/webhook delivery to fully close
   M1's alert path.

**CI is still unconfigured** (open `normal` issue): load-bearing now that the merkle crypto + equivocation
path has landed and the upcoming `fsck`-rebuild/fixture slices arm the external `notecheck` oracle — wire CI
**before** the `fsck`-rebuild conformance slice, and resolve the `go mod tidy` go.sum divergence at the same
time so the tidy gate passes.
