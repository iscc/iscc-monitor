<!-- assessed-at: 79b48dabbb4cd2de7ed26117f8b39731fde155dd -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — the full equivocation toolkit is now **prerequisite-complete but still unwired**: pure `CheckEquivocation` verifier + `SQLiteFetcher` read seam + the new pure `ConsistencyProofFromTiles` proof builder all exist and are golden-tested, yet none are connected to the follower's `checkConsistency` (still shrink+fork only). Tile fixtures, `fsck` root-rebuild, structured logs, `/metrics`, and a real alert transport remain missing.

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, the full `hub_keys` did:web
key cache, the `internal/tiles` tlog-tiles layout seam, and the `internal/store` partial-tile mirror
CRUD + `SQLiteFetcher` have all landed. The newest slice adds `internal/logclient/proofbuilder.go` —
the pure `ConsistencyProofFromTiles` builder that sources every RFC-6962 consistency-proof node from
mirrored hash tiles via a fetch seam byte-identical to `SQLiteFetcher.ReadTile`. With this, the three
ingredients the deferred M1 equivocation branch needs (verdict function, tile read seam, proof source)
all exist — but they are **not yet wired together in the follower**. What still blocks M1: that wiring
+ an end-to-end test (which needs **tile fixtures, still absent**), structured logs, `/metrics`, and a
real alert transport.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache + pure equivocation
verifier + tlog-tiles layout seam + `SQLiteFetcher` mirror read-back + **pure `ConsistencyProofFromTiles`
proof builder**; the equivocation trigger is **still not wired into the follower**, and logs/metrics/
alert transport remain missing)

- **Verified incrementally** from the prior assessment at `4eaead8`. The diff `4eaead8..HEAD` touches
  **only** the logclient package — one new production file (`internal/logclient/proofbuilder.go`) + its
  test (`proofbuilder_test.go`) — plus context files. Confirmed unchanged (no diff): `internal/follower/`,
  `internal/didweb/`, `internal/registry/`, `internal/config/`, `internal/tiles/`, `internal/store/`,
  `cmd/`, `go.mod`/`go.sum`, `schema.sql`, and the fixtures. All sections below other than the new
  proofbuilder lines are re-confirmed unchanged and carried forward.

  - **NEW — pure `ConsistencyProofFromTiles` proof builder landed** (`internal/logclient/proofbuilder.go`):
    `ConsistencyProofFromTiles(ctx, fetch TileFetcher, smaller, larger uint64) ([][]byte, error)` — an
    otel-/net-free port of tessera's `ProofBuilder.ConsistencyProof + fetchNodes + nodeCache.GetNode`
    that sources every proof node from mirrored hash tiles via the injected `TileFetcher func(ctx,
    level, index uint64, p uint8) ([]byte, error)`, whose signature is **deliberately byte-identical to
    `store.SQLiteFetcher.ReadTile`** so the follower can pass `SQLiteFetcher.ReadTile` straight in.
    Imports only `context`+`fmt`+`merkle/{proof,compact,rfc6962}`+`tessera/api{,/layout}` — WASM-clean.
    **+3 `func Test`** (logclient 26 → 29), golden-tested as ground truth against a 300-leaf
    `testonly.Tree` across the 256-leaf tile boundary (byte-equals `tree.ConsistencyProof` AND verifies
    via `proof.VerifyConsistency`; `review` proved non-vacuousness — a one-byte corruption in the fetched
    node fails the golden). This is the proof **source** the deferred equivocation wiring needs; it is
    **not itself wired into anything** yet.

  - **Pure equivocation verifier** (`internal/logclient/consistency.go`, unchanged this slice):
    `CheckEquivocation(prevSize, prevRoot, nextSize, nextRoot, consistencyProof) (violated, err)` — for
    the strictly-growing case calls `proof.VerifyConsistency(rfc6962.DefaultHasher, ...)`; a non-verifying
    proof is a **verdict** (`violated=true, err=nil`), never a crash ("freeze, never crash", ADR-0006).
    Boundaries (`prevSize==0` / `nextSize==prevSize` / `nextSize<prevSize`) return `(false,nil)` and skip
    `VerifyConsistency`. Golden-tested against a genuine `testonly.New(rfc6962.DefaultHasher)` tree.

  - **CRITICAL CAVEAT — equivocation is STILL NOT wired into the follower** (confirmed: zero
    `CheckEquivocation`/`ConsistencyProofFromTiles`/`ViolationEquivocation`/`SQLiteFetcher` references in
    `internal/follower/` or `cmd/`). `follower.checkConsistency` (`follower.go:236`) still has only the
    shrink + fork branches (`logclient.CheckShrink` + `logclient.CheckFork`, no third branch). All three
    ingredients now exist as pure, golden-tested units — the verdict function (`CheckEquivocation`), the
    tile read seam (`SQLiteFetcher.ReadTile`), and the proof source (`ConsistencyProofFromTiles`) — but
    the connective wiring and an end-to-end test (which needs **tile fixtures, still absent**) are not
    built. The M1 equivocation trigger is **not met end-to-end**.

  - **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
    possible. Documented limitation, not a green-but-wrong path.

  - **`SQLiteFetcher` + partial-tile mirror CRUD** (`internal/store/{tiles,fetcher}.go`, unchanged this
    slice): `RecordTile`/`RecordEntryBundle`/`ReadTileBlob`/`ReadEntryBundleBlob`/`LatestCheckpointRaw`
    (partial-tile discipline: `is_full=1` only at `width==256`, sha256 column, composite-PK upsert,
    `os.ErrNotExist` on a missing row) + `SQLiteFetcher{Store,HubID}` with the three Fetcher methods
    (`ReadCheckpoint`/`ReadTile`/`ReadEntryBundle`), the `widthForP` p↔width mapping, and an inline
    `PartialOrFullResource` partial→full fallback honoring `errors.Is(err, os.ErrNotExist)`. Store stays
    a **leaf**: deliberately does NOT import `tessera/fsck` or `tessera/client`; conformance to
    `fsck.Fetcher` is structural (local interface copy + `var _ fsckFetcher = SQLiteFetcher{}` in
    `fetcher_test.go`, accepted by `review` as drift-detection equivalent without the dep leak).

  - **reader→follower wiring** (`internal/follower/follower.go`): `cacheHubKeyFast` (fetch-free row
    refresh on a `LookupHubKey` hit) + `cacheHubKeyResolve` (cold fallback). Proven by
    `TestPollHubCacheHitSkipsDidFetch`.

  - **`KeyIDFromCheckpoint`** (`internal/logclient/checkpointkey.go`) — golden `name=="sb0.iscc.id/log"`,
    `keyID==0x40b74463`.

  - **`hub_keys` key reader** (`internal/store/checkpoints.go`): `LookupHubKey(ctx, hubID, keyID)`.

  - **coverage tracking** (`internal/store/checkpoints.go` + `follower.go`): set-once
    `monitored_since_{size,time}` (ADR-0001), wired between `RecordCheckpoint` and `AdvanceFollowState`.

  - **`cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): wired M1 entrypoint; `alert(hubID, kind)`
    is a documented stderr placeholder.

  - **exported `logclient.Origin`** + golden `TestOrigin`; **config loader leaf**; **realm-registry
    parser** (domains-only, fails closed on URLs); **poll-loop cadence** (`loop.go`, single-writer);
    **freeze + alert-once** (`checkConsistency` shrink-then-fork; `freeze` → `RecordViolation` + evidence
    `RecordCheckpoint` + `Freeze` + one-shot `AlertFunc`, gated on `!wasFrozen`).

  - `internal/store/checkpoints.go` — typed CRUD: `UpsertHub`/`RecordCheckpoint`/`FollowState`/
    `AdvanceFollowState`/`RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`/`RecordHubKey`/`LookupHubKey`.

  - `internal/store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`,
    ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`,
    `synchronous=NORMAL`, `SetMaxOpenConns(1)`), embedded **nine-table** `schema.sql` (`hubs`,
    `hub_keys`, `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`,
    `ots`).

  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin,keyid,checkpointkey,consistency,proofbuilder}.go`
    — transport-only `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`/`VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, all three consistency triggers (dep-free
    `CheckShrink`/`CheckFork`, merkle-backed `CheckEquivocation`), and the pure
    `ConsistencyProofFromTiles` proof builder.

  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/
    `DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.

  - `internal/tiles/layout.go` — pure leaf re-exporting tessera's `api/layout` primitives (`TilePath`,
    `EntriesPath`, `PartialTileSize`, `TileWidth=256`, `TileHeight=8`) + the `IsFull(width)` predicate.
    `net/http`- and `database/sql`-free, WASM-green.

  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.

  - **Test totals (verified by grep)**: 9 (`didweb`) + **29 (`logclient`, +3)** + 48 (`store`) +
    8 (`follower`) + 3 (`registry`) + 4 (`config`) + 5 (`tiles`) + 1 (`cmd/iscc-monitor`) =
    **107 `func Test`**.

- **Missing (remaining M1 connective tissue):**
  - **Equivocation trigger end-to-end** — all three ingredients (`CheckEquivocation` verdict function,
    `SQLiteFetcher` read seam, `ConsistencyProofFromTiles` proof source) now exist as pure golden-tested
    units, but the follower wiring is unbuilt (no third branch in `checkConsistency`). The end-to-end
    test needs **tile fixtures (still absent)**, and the conformance/oracle gate re-arms once the
    consistency-proof path runs over real on-disk tiles end-to-end.
  - **`fsck` root-rebuild conformance** — `fsck.New(...).Check(...)` over `SQLiteFetcher` + the inclusion
    cross-check vs the hub's `IsccLogInclusionProof` is unbuilt (needs tile fixtures + the `fsck` dep,
    which must live in `cmd/` or a future conformance package, not the store leaf).
  - **Structured logs** — no `slog` anywhere in `internal`/`cmd` (confirmed: zero `log/slog` imports). The
    only non-test `net/http` import is the legitimate one in `didresolve.go`. Two stderr placeholders
    remain: `alert` in `cmd/iscc-monitor/main.go` and the per-tick swallowed error in `loop.go`.
  - **`/metrics`** — no metrics impl, no `expvar`/prometheus, no http server (confirmed).
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed
  for the equivocation follower wiring + the `fsck` root-rebuild + M2; confirmed still absent). **Known
  stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json` fixtures (both `internal/didweb/`
  and `internal/logclient/`) and `derive_vkey.py` still carry sb1's PRE-rotation key (`22b08f3e`); the
  live sb1 signer is `069d0f14`. Captured in `verify_test.go` prose/tests (not green-but-wrong), but the
  did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `github.com/transparency-dev/merkle v0.0.2` (`consistency.go` + `proofbuilder.go`), and
  `github.com/transparency-dev/tessera v1.0.2` (`internal/tiles/layout.go` — `api/layout`; and now
  `proofbuilder.go` — `api` + `api/layout`). The `SQLiteFetcher` proves conformance to
  `tessera/fsck.Fetcher` **structurally** (local interface copy), so heavy tessera deps stay out of the
  store closure. **Not yet wired** (confirmed zero non-comment imports): the rest of tessera
  (`client`/`fsck`), `transparency-dev/formats`, `nbd-wtf/opentimestamps`.

- **Verify criteria status**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected + evidence-survives-
  restart). Coverage (`monitored_since`) — **tracked** and asserted set-once. The **third trigger,
  equivocation, has a verified pure verifier, a tile read seam, AND a pure proof builder, but is NOT
  wired into the follower**, so the M1 Verify line is **not satisfied**.

## M2 — Aggregator
**Status**: not started — but **three prerequisite slices have landed**: `internal/tiles` re-exports
tessera's tlog-tiles layout math + the `IsFull` partial-tile predicate; `internal/store/{tiles,fetcher}.go`
provides the partial-tile mirror CRUD + `SQLiteFetcher` (structural `client.Fetcher`/`fsck.Fetcher`); and
`internal/logclient/proofbuilder.go` provides a pure `ConsistencyProofFromTiles` over the same tile-fetch
seam (a first piece of the M2 `ProofBuilder`). What remains for M2: tile/entry-bundle fixtures, the actual
`fsck.New(...).Check(...)` root-rebuild over `SQLiteFetcher` (the trust-root oracle re-arm), the
`iscc_index` projection writer, and `inclusion`/`consistency`/`entries` served via a full `ProofBuilder`
from the local store.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `merkle v0.0.2` + `tessera v1.0.2` +
  `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable. Latest `review` handoff (2026-06-21,
  "Pure `ConsistencyProofFromTiles` builder over the tile-fetch seam", verdict
  **PASS_WITH_NOTES / CONTINUE**) records the gate green at HEAD `79b48da`: `mise run check` green
  (build + vet + test, all 8 packages ok), `gofmt -l .` empty, `go mod verify` passes, build reproducible
  under `-mod=readonly`, `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (purity preserved).
  Oracle gate **APPLIES** (RFC-6962 crypto) and is satisfied by three independent merkle paths (prover
  `tree.ConsistencyProof`, verifier `VerifyConsistency`, builder) + a corrupt-node mutation proving
  non-vacuousness; `notecheck`/`derive_vkey.py`/`fsck` correctly N/A (tiles synthesized in-test, no
  signature/did:web/on-disk-tile/fsck-rebuild path this slice).
- **Two open `normal` issues** (neither blocks the loop, both gate-relevant for the next conformance
  slice): (1) `go mod tidy` now adds **22 unstaged go.sum lines** (tessera transitive requires that never
  compile) — harmless today with no CI, but would fail a future `go mod tidy && git diff --exit-code`
  gate; (2) **no `.github/workflows/`** — the external `notecheck` signature-parity oracle and any
  build/test/format/tidy gate run only locally, never in CI.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**, in sync
  with `origin/develop` (0/0); tree clean at HEAD `79b48da`. **No `.github/workflows/` — no CI
  configured** (`gh run list` returns empty). When CI is wired it must avoid `go build ./...` over the
  gitignored `cauldron/` reference trees and shell out the future `notecheck` oracle rather than `go run`
  from `cauldron/`.

## Next Milestone
Continue M1. All three equivocation ingredients (verifier `CheckEquivocation`, read seam `SQLiteFetcher`,
proof source `ConsistencyProofFromTiles`) are now done and golden-tested; the immediate gap is wiring them
together in the follower and capturing the tile fixtures the wiring test + the `fsck` root-rebuild both
need. Candidate slices, in rough order:
1. **Wire equivocation into `follower.checkConsistency`** as the third branch — source the proof via
   `ConsistencyProofFromTiles(ctx, SQLiteFetcher.ReadTile, prevSize, nextSize)` (the `TileFetcher`
   signature was made byte-identical to `SQLiteFetcher.ReadTile` for exactly this), feed its `[][]byte`
   to `CheckEquivocation`, and on a true verdict `RecordViolation`("equivocation") + `RecordCheckpoint`
   (evidence) + `Freeze` + alert-once, mirroring the fork/shrink branches (honor the shrink→fork→
   equivocation order and "compare against the prior accepted root, not the contradicting evidence").
   Needs **real on-disk tile fixtures** under `testdata/live/` for an end-to-end test — closing the M1
   Verify line and re-arming the conformance/oracle gate.
2. **`fsck` root-rebuild conformance slice** — real tile fixtures + `fsck.New(...).Check(...)` over
   `SQLiteFetcher` + the inclusion cross-check vs the hub's `IsccLogInclusionProof`. First slice where
   the trust-root **oracle gate re-arms** for the mirror path; lives in `cmd/` or a future conformance
   package that *can* take the `fsck` dep (otel/klog closure), not the store leaf.
3. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
4. The lighter remaining M1 gaps — structured logs (`slog`, replacing the two stderr placeholders),
   `/metrics`, real alert transport — then complete M1's Verify criteria.

**CI is still unconfigured** (open `normal` issue): load-bearing now that the merkle crypto path has
landed and the upcoming `fsck`-rebuild/fixture slices arm the external `notecheck` oracle — wire CI
**before** the `fsck`-rebuild conformance slice so the trust-root oracle has CI coverage when the mirror
path first faces it, and resolve the `go mod tidy` go.sum divergence at the same time so the tidy gate
passes.
