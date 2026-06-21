<!-- assessed-at: 0a6d5134607f5c71db0126a5459d89f9ab1e6878 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 nearly complete — all **three** self-consistency triggers (shrink / fork / equivocation) are now **wired into the follower and golden-tested end-to-end**. The equivocation branch sources its RFC-6962 consistency proof from the local mirror (`SQLiteFetcher.ReadTile` → `ConsistencyProofFromTiles` → `CheckEquivocation`) and freezes on a non-verifying growing-pair proof. What still blocks full M1: structured logs (`slog`), `/metrics`, a real alert transport — plus the fact that the equivocation branch is **dormant in production** until M2's tile-ingestion writer lands (today `PollHub` never writes tiles, so the live path always hits the missing-tile skip).

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → all three freeze triggers),
the single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, the full `hub_keys` did:web
key cache, the `internal/tiles` tlog-tiles layout seam, the `internal/store` partial-tile mirror CRUD +
`SQLiteFetcher`, and the pure `ConsistencyProofFromTiles` builder have all landed. The newest slice
**wires `CheckEquivocation` into `follower.checkConsistency`** as the third branch, closing M1's last
open self-consistency trigger. The remaining M1 gaps are the lighter, independent pieces: structured
logs, `/metrics`, and a real alert transport.

## M1 — Read-only Monitor
**Status**: partially met (all three freeze triggers — shrink / fork / equivocation — wired + golden-
tested end-to-end; verify primitives + poll-loop cadence + realm-registry parser + config loader +
wired `cmd/` binary + coverage tracking + `hub_keys` cache + tlog-tiles seam + `SQLiteFetcher` +
`ConsistencyProofFromTiles`; **logs / metrics / real alert transport remain missing**)

- **Verified incrementally** from the prior assessment at `79b48da`. The diff `79b48da..HEAD` touches
  **only** `internal/follower/` (one production file `follower.go` + one new test file
  `equivocation_test.go`) plus context files. Confirmed byte-unchanged (no diff): `internal/logclient/`,
  `internal/didweb/`, `internal/registry/`, `internal/config/`, `internal/tiles/`, `internal/store/`,
  `cmd/`, `go.mod`/`go.sum`, `schema.sql`, and the fixtures. All sections other than the follower
  equivocation wiring are re-confirmed unchanged and carried forward.

  - **NEW — equivocation trigger WIRED into the follower** (`internal/follower/follower.go`):
    `checkConsistency` (`follower.go:259`) now evaluates **shrink → fork → equivocation**. The third
    branch (lines 281-298) is the only growing-pair trigger (`info.TreeSize > prevSize`, `prevFound`):
    it builds the proof via `store.SQLiteFetcher{Store, HubID}.ReadTile` →
    `logclient.ConsistencyProofFromTiles(ctx, fetcher.ReadTile, prevSize, info.TreeSize)`, feeds the
    resulting `[][]byte` to `logclient.CheckEquivocation`, and on a `true` verdict returns
    `logclient.ViolationEquivocation` → `freeze` (RecordViolation + evidence RecordCheckpoint + Freeze +
    alert-once). Error-vs-violation discipline is sound (ADR-0006 "freeze, never crash"): a proof-build
    error — most often **tiles not mirrored yet** — returns `(false, "", nil, nil)` and **skips** the
    branch (never freezes on a missing tile); only a store fault propagates a real error. **+2 `func
    Test`** (follower 8 → 10): `TestPollHubEquivocation` (freeze: `violations.kind == "equivocation"`,
    `frozen == 1`, cursor stays at prevSize, exactly-one-alert, 2nd detection records a 2nd violation
    without re-alerting, evidence survives a store reopen) + `TestEquivocationMissingTilesDoesNotFreeze`
    (no tiles → proof-build error → branch skipped → no violation, the false-positive guard). `review`
    proved non-vacuousness by mutation (inverted verdict `if eq`→`if !eq` AND forced-skip both fail the
    freeze test) and confirmed the proof genuinely crosses the 256-leaf tile boundary over the real
    `SQLiteFetcher` (full tile at index 0 width 256, 44-leaf partial at index 1).

  - **PRODUCTION-DORMANT caveat**: the equivocation branch is wired but **dormant in the live path**
    until M2's tile-ingestion writer lands. Today `PollHub` never writes tiles, so a live observation
    always hits the missing-tile → skip case. This is the deliberate conservative choice; the branch is
    fully proven against in-test synthesized mirrored tiles (`seedMirrorTiles` via `RecordTile`, the
    stand-in for the future M2 writer) and becomes load-bearing the instant the M2 tile writer ships.

  - **Pure equivocation verifier** (`internal/logclient/consistency.go`, unchanged this slice):
    `CheckEquivocation(prevSize, prevRoot, nextSize, nextRoot, consistencyProof) (violated, err)` — for
    the strictly-growing case calls `proof.VerifyConsistency(rfc6962.DefaultHasher, ...)`; a non-verifying
    proof is a **verdict** (`violated=true, err=nil`), never a crash. Boundaries return `(false,nil)`.

  - **Pure `ConsistencyProofFromTiles` proof builder** (`internal/logclient/proofbuilder.go`, unchanged
    this slice): otel-/net-free port of tessera's `ProofBuilder.ConsistencyProof` that sources every
    proof node from mirrored hash tiles via the injected `TileFetcher func(ctx, level, index uint64,
    p uint8) ([]byte, error)` — signature **byte-identical to `store.SQLiteFetcher.ReadTile`**, which is
    exactly how the follower now passes it. Imports only `context`+`fmt`+`merkle/{proof,compact,rfc6962}`
    +`tessera/api{,/layout}` — WASM-clean. Golden-tested against a 300-leaf `testonly.Tree` across the
    256-leaf tile boundary (byte-equals `tree.ConsistencyProof` AND verifies via `VerifyConsistency`;
    corrupt-node mutation fails the golden).

  - **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
    possible. Documented limitation, not a green-but-wrong path.

  - **`SQLiteFetcher` + partial-tile mirror CRUD** (`internal/store/{tiles,fetcher}.go`, unchanged):
    `RecordTile`/`RecordEntryBundle`/`ReadTileBlob`/`ReadEntryBundleBlob`/`LatestCheckpointRaw` (partial-
    tile discipline: `is_full=1` only at `width==256`, sha256 column, composite-PK upsert,
    `os.ErrNotExist` on a missing row) + `SQLiteFetcher{Store,HubID}` with the three Fetcher methods
    (`ReadCheckpoint`/`ReadTile`/`ReadEntryBundle`), the `widthForP` p↔width mapping, and the inline
    `PartialOrFullResource` fallback. Store stays a **leaf**: structural `fsck.Fetcher` conformance via a
    local interface copy + `var _ fsckFetcher = SQLiteFetcher{}`, no `tessera/fsck`/`client` dep leak.

  - **reader→follower wiring** (`follower.go`): `cacheHubKeyFast` (fetch-free row refresh on a
    `LookupHubKey` hit) + `cacheHubKeyResolve` (cold fallback). Proven by
    `TestPollHubCacheHitSkipsDidFetch`.

  - **`KeyIDFromCheckpoint`** (`checkpointkey.go`) — golden `name=="sb0.iscc.id/log"`, `keyID==0x40b74463`.

  - **`hub_keys` key reader** (`store/checkpoints.go`): `LookupHubKey(ctx, hubID, keyID)`.

  - **coverage tracking** (`store/checkpoints.go` + `follower.go`): set-once `monitored_since_{size,time}`
    (ADR-0001), wired between `RecordCheckpoint` and `AdvanceFollowState`.

  - **`cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): wired M1 entrypoint; `alert(hubID, kind)`
    is a documented stderr placeholder.

  - **exported `logclient.Origin`** + golden `TestOrigin`; **config loader leaf**; **realm-registry
    parser** (domains-only, fails closed on URLs); **poll-loop cadence** (`loop.go`, single-writer);
    **freeze + alert-once** (`freeze` → `RecordViolation` + evidence `RecordCheckpoint` + `Freeze` + one-
    shot `AlertFunc`, gated on `!wasFrozen`).

  - `store/checkpoints.go` — typed CRUD: `UpsertHub`/`RecordCheckpoint`/`CheckpointAt`/`FollowState`/
    `AdvanceFollowState`/`RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`/`RecordHubKey`/`LookupHubKey`.

  - `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded **nine-table** `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
    `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`).

  - `logclient/{checkpoint,accept,verify,didresolve,origin,keyid,checkpointkey,consistency,proofbuilder}.go`
    — transport-only `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`/`VerifyCheckpoint`, networked
    `ResolveVerifierKey`, all three consistency triggers (`CheckShrink`/`CheckFork`/`CheckEquivocation`),
    and the pure `ConsistencyProofFromTiles` builder.

  - `didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/`DIDKey`
    (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.

  - `tiles/layout.go` — pure leaf re-exporting tessera's `api/layout` primitives (`TilePath`,
    `EntriesPath`, `PartialTileSize`, `TileWidth=256`, `TileHeight=8`) + the `IsFull(width)` predicate.

  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.

  - **Test totals (verified by grep)**: 9 (`didweb`) + 29 (`logclient`) + 48 (`store`) +
    **10 (`follower`, +2)** + 3 (`registry`) + 4 (`config`) + 5 (`tiles`) + 1 (`cmd/iscc-monitor`) =
    **109 `func Test`**.

- **Missing (remaining M1 connective tissue):**
  - **Structured logs** — no `slog` anywhere in `internal`/`cmd` (confirmed: zero `log/slog` imports).
    Two stderr placeholders remain: `alert` in `cmd/iscc-monitor/main.go` and the per-tick swallowed
    error in `loop.go` (`loop.go:131` notes the error "surfaces via the store/metrics later").
  - **`/metrics`** — no metrics impl, no `expvar`/prometheus, no http server (confirmed: the only
    `metrics` mention is a comment in `loop.go:131`).
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder.
  - **Live tile ingestion** (M2 dependency, blocks the equivocation branch from firing in production) —
    `PollHub` never writes tiles, so the wired equivocation branch is dormant on the live path until the
    M2 writer lands. Not strictly an M1 Verify gap (the trigger is golden-tested end-to-end against
    seeded tiles), but it is the reason the live path can't yet detect a real equivocation.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles**. The
  equivocation test synthesizes mirrored tiles in-process (`seedMirrorTiles`); real on-disk tile fixtures
  are still needed for the `fsck` root-rebuild (M2) and the live tile-ingestion path. **Known stale-
  fixture drift, still not acted on:** the `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and
  `internal/logclient/`) and `derive_vkey.py` still carry sb1's PRE-rotation key (`22b08f3e`); the live
  sb1 signer is `069d0f14`. Captured in `verify_test.go` prose/tests (not green-but-wrong), but the
  did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `github.com/transparency-dev/merkle v0.0.2` (`consistency.go` + `proofbuilder.go`), and
  `github.com/transparency-dev/tessera v1.0.2` (`tiles/layout.go` — `api/layout`; `proofbuilder.go` —
  `api` + `api/layout`). `SQLiteFetcher` conforms to `tessera/fsck.Fetcher` **structurally** (local
  interface copy), keeping heavy tessera deps out of the store closure. **Not yet wired** (confirmed zero
  non-comment imports): the rest of tessera (`client`/`fsck`), `transparency-dev/formats`,
  `nbd-wtf/opentimestamps`.

- **Verify criteria status**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **All three triggers now met end-to-end** in golden tests (synthetic shrink, fork,
  AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-
  unaffected + evidence-survives-restart). Coverage (`monitored_since`) — **tracked** and asserted set-
  once. M1's **Verify** line for the triggers is **satisfied in test**; M1 is not yet *fully* met because
  structured logs + `/metrics` (both in the M1 scope sentence) are still absent.

## M2 — Aggregator
**Status**: not started — but **three prerequisite slices have landed**: `internal/tiles` re-exports
tessera's tlog-tiles layout math + the `IsFull` predicate; `internal/store/{tiles,fetcher}.go` provides
the partial-tile mirror CRUD + `SQLiteFetcher` (structural `client.Fetcher`/`fsck.Fetcher`); and
`internal/logclient/proofbuilder.go` provides a pure `ConsistencyProofFromTiles` over the same tile-fetch
seam (a first piece of the M2 `ProofBuilder`, now also consumed by the follower's equivocation branch).
What remains for M2: tile/entry-bundle fixtures + the **live tile-ingestion writer** (which also
un-dormants the equivocation branch), the actual `fsck.New(...).Check(...)` root-rebuild over
`SQLiteFetcher` (the trust-root oracle re-arm), the `iscc_index` projection writer, and
`inclusion`/`consistency`/`entries` served via a full `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `merkle v0.0.2` + `tessera v1.0.2` +
  `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable. Latest `review` handoff (2026-06-21,
  "Wire `CheckEquivocation` into `follower.checkConsistency`", verdict **PASS / CONTINUE**) records the
  gate green at HEAD `0a6d513`: `mise run check` green (build + vet + test, all 8 packages ok),
  `gofmt -l .` empty, `go.mod`/`go.sum` + trust-root packages byte-identical to HEAD~1,
  `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (purity preserved), `go list -deps
  ./internal/store` has no `net/http`. Oracle gate **APPLIES** (RFC-6962/equivocation path) and is
  satisfied: merkle-backed `CheckEquivocation` + `ConsistencyProofFromTiles` golden tests pass,
  `derive_vkey.py` prints both golden vectors exactly (`40b74463` / `22b08f3e`), non-vacuousness proven
  by two mutations (inverted verdict + forced-skip both fail the freeze test). `notecheck`/`fsck` correctly
  N/A (no external oracle wired; tiles synthesized in-test, no on-disk-tile/fsck-rebuild path this slice).
  Gate-integrity scan over the unpushed commits: no `//nolint`/`t.Skip`/build-tag exclusion/swallowed-
  error dodge/deleted assertion.
- **Two open `normal` issues** (neither blocks the loop, both gate-relevant for the next conformance
  slice): (1) `go mod tidy` adds **22 unstaged go.sum lines** (tessera transitive requires that never
  compile) — harmless today with no CI, but would fail a future `go mod tidy && git diff --exit-code`
  gate; (2) **no `.github/workflows/`** — the external `notecheck` signature-parity oracle and any
  build/test/format/tidy gate run only locally, never in CI.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**, in sync
  with `origin/develop`; tree clean at HEAD `0a6d513`. **No `.github/workflows/` — no CI configured.**
  When CI is wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and
  shell out the future `notecheck` oracle rather than `go run` from `cauldron/`.

## Next Milestone
Continue M1. All three self-consistency triggers are now wired and golden-tested; the remaining M1 gaps
are the lighter independent slices plus the M2-prerequisite that un-dormants equivocation in production.
Candidate slices, in rough order:
1. **`fsck` root-rebuild conformance slice** (M2 Verify, also the first real on-disk tile fixtures) —
   `fsck.New(...).Check(...)` over `SQLiteFetcher` + the inclusion cross-check vs the hub's
   `IsccLogInclusionProof`. First slice where the trust-root **oracle gate re-arms** for the mirror path.
   Needs the heavy `fsck`/otel/klog dep (cannot land in the store leaf — copy or scope it as the
   `SQLiteFetcher` slice did) + real on-disk tile fixtures under `testdata/live/`. Wire CI/`notecheck`
   and resolve the `go mod tidy` go.sum divergence at the same time so the trust-root oracle has CI
   coverage when the mirror path first faces it.
2. **Live tile-ingestion writer** — make `PollHub` mirror tiles so the wired equivocation branch becomes
   load-bearing on the live path (today it always hits the missing-tile skip). This is M2 territory but
   is what turns the in-test-proven equivocation trigger into a real detector.
3. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
4. **Lighter remaining M1 gaps** — structured logs (`slog`, replacing the two stderr placeholders),
   `/metrics`, real alert transport — to fully complete M1's Verify/scope criteria.

**CI is still unconfigured** (open `normal` issue): load-bearing now that the merkle crypto + equivocation
path has landed and the upcoming `fsck`-rebuild/fixture slices arm the external `notecheck` oracle — wire
CI **before** the `fsck`-rebuild conformance slice, and resolve the `go mod tidy` go.sum divergence at the
same time so the tidy gate passes.
