<!-- assessed-at: 4eaead8b5bbace793e56a63e37c358879a1cd2e0 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress (with M2-prerequisite seams landing) — verify core + cadence + parser + config + `cmd/` binary + coverage + `hub_keys` cache + **pure equivocation verifier** + **`internal/tiles` layout seam** + **`SQLiteFetcher` mirror read-back** are landed; equivocation is still **not wired into the follower**, and tile fixtures + `fsck` root-rebuild + `ProofBuilder` + structured logs + `/metrics` + real alert transport are still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, the full `hub_keys` did:web
key cache, the pure `CheckEquivocation` consistency-proof verifier, and the `internal/tiles` tlog-tiles
layout seam have all landed. The newest slice adds `internal/store/{tiles.go,fetcher.go}` — the
partial-tile-discipline mirror write/read CRUD plus `SQLiteFetcher`, which structurally satisfies
tessera's three-method Fetcher shape over the store. This is the read seam the deferred M1 equivocation
branch and the M2 `fsck` root-rebuild both depend on. What still blocks M1: wire `CheckEquivocation`
into the follower (now sourceable via `SQLiteFetcher`, but still needs tile fixtures for an end-to-end
test), structured logs, `/metrics`, and a real alert transport.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache + pure equivocation
verifier + tlog-tiles layout seam + `SQLiteFetcher` mirror read-back; the equivocation trigger is
**not yet wired into the follower**, and logs/metrics/alert transport remain missing)

- **Verified incrementally** from the prior assessment at `ac4a643`. The diff `ac4a643..HEAD` touches
  **only** the store package — two new production files (`internal/store/tiles.go`,
  `internal/store/fetcher.go`) + their tests — and context files. Confirmed unchanged (no diff):
  `internal/follower/`, `internal/logclient/`, `internal/didweb/`, `internal/registry/`,
  `internal/config/`, `internal/tiles/`, `cmd/`, `go.mod`/`go.sum`, `schema.sql`, and the fixtures. All
  sections below other than the new store lines are re-confirmed unchanged and carried forward.

  - **NEW — `SQLiteFetcher` + partial-tile mirror CRUD landed** (`internal/store/tiles.go` +
    `internal/store/fetcher.go`): `tiles.go` adds `RecordTile`/`RecordEntryBundle`/`ReadTileBlob`/
    `ReadEntryBundleBlob`/`LatestCheckpointRaw` (partial-tile discipline: `is_full=1` only at
    `width==256`, sha256 column, composite-PK upsert, `os.ErrNotExist` on a missing row). `fetcher.go`
    adds `SQLiteFetcher{Store,HubID}` with the three Fetcher methods (`ReadCheckpoint`/`ReadTile`/
    `ReadEntryBundle`), the **load-bearing `widthForP` p↔width mapping** (`p==0` → `tiles.TileWidth=256`
    full, else `int(p)`), and an inline `PartialOrFullResource` partial→full fallback honoring the
    `errors.Is(err, os.ErrNotExist)` contract. **+17 `func Test`** (store 31 → 48), all uncached-green
    per `review`. Store stays a **leaf**: it deliberately does NOT import `tessera/fsck` or
    `tessera/client` (that closure pulls `net/http`/`otel`/`klog`); conformance to `fsck.Fetcher` is
    structural, pinned by a local `fsckFetcher` interface copy + `var _ fsckFetcher = SQLiteFetcher{}`
    in `fetcher_test.go` (accepted by `review` as drift-detection equivalent without the dep leak).
    `go.mod`/`go.sum` byte-identical, `go mod tidy` a verified no-op.

  - **Pure equivocation verifier** (`internal/logclient/consistency.go`, unchanged this slice):
    `CheckEquivocation(prevSize, prevRoot, nextSize, nextRoot, consistencyProof) (violated, err)` — for
    the strictly-growing case calls `proof.VerifyConsistency(rfc6962.DefaultHasher, ...)`; a non-verifying
    proof is a **verdict** (`violated=true, err=nil`), never a crash ("freeze, never crash", ADR-0006).
    Boundaries (`prevSize==0` / `nextSize==prevSize` / `nextSize<prevSize`) return `(false,nil)` and skip
    `VerifyConsistency`. Golden-tested against a genuine `testonly.New(rfc6962.DefaultHasher)` tree.

  - **CRITICAL CAVEAT — `CheckEquivocation` is STILL NOT wired into the follower** (confirmed: zero
    `CheckEquivocation`/`ViolationEquivocation`/`SQLiteFetcher` references in `internal/follower/` or
    `cmd/`). It remains a pure verdict function; `follower.checkConsistency` still has only the shrink +
    fork branches (`logclient.CheckShrink` + `logclient.CheckFork`, no third branch). The
    `SQLiteFetcher` read seam needed to source the consistency-proof hashes now exists, but the wiring
    and an end-to-end test (which needs **tile fixtures, still absent**) are not built. The M1
    equivocation trigger is **not met end-to-end**.

  - **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
    possible. Documented limitation, not a green-but-wrong path.

  - **reader→follower wiring** (`internal/follower/follower.go`): `cacheHubKey` splits into
    `cacheHubKeyFast` (fetch-free, refreshes the row on a `LookupHubKey` hit with NO second did.json
    fetch) and `cacheHubKeyResolve` (cold fallback). Proven by `TestPollHubCacheHitSkipsDidFetch`.

  - **`KeyIDFromCheckpoint`** (`internal/logclient/checkpointkey.go`) — pure raw-checkpoint key-id reader;
    golden `name=="sb0.iscc.id/log"`, `keyID==0x40b74463`.

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

  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin,keyid,checkpointkey,consistency}.go`
    — transport-only `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`/`VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, and all three consistency triggers (dep-free
    `CheckShrink`/`CheckFork`, merkle-backed `CheckEquivocation`).

  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/
    `DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.

  - `internal/tiles/layout.go` — pure leaf re-exporting tessera's `api/layout` primitives (`TilePath`,
    `EntriesPath`, `PartialTileSize`, `TileWidth=256`, `TileHeight=8`) + the `IsFull(width)` predicate.
    `net/http`- and `database/sql`-free, WASM-green.

  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.

  - **Test totals (verified by grep)**: 9 (`didweb`) + 26 (`logclient`) + **48 (`store`, +17)** +
    8 (`follower`) + 3 (`registry`) + 4 (`config`) + 5 (`tiles`) + 1 (`cmd/iscc-monitor`) =
    **104 `func Test`**.

- **Missing (remaining M1 connective tissue):**
  - **Equivocation trigger end-to-end** — the pure `CheckEquivocation` verifier AND the `SQLiteFetcher`
    read seam now both exist, but the follower wiring is unbuilt (no third branch in `checkConsistency`).
    The end-to-end test needs **tile fixtures (still absent)**, and the conformance/oracle gate re-arms
    once the consistency-proof hashes flow through `SQLiteFetcher`.
  - **`fsck` root-rebuild conformance** — `fsck.New(...).Check(...)` over `SQLiteFetcher` + the inclusion
    cross-check vs the hub's `IsccLogInclusionProof` is unbuilt (needs tile fixtures + the `fsck` dep,
    which must live in `cmd/` or a future conformance package, not the store leaf).
  - **Structured logs** — no `slog` anywhere in `internal`/`cmd` (confirmed; all `slog`/`net/http`-string
    grep hits in non-test code are comments, except `didresolve.go`'s legitimate `net/http` import for
    did resolution). Two stderr placeholders remain: `alert` in `cmd/iscc-monitor/main.go` and the
    per-tick swallowed error in `loop.go`.
  - **`/metrics`** — no metrics impl, no `expvar`/prometheus, no http server (confirmed).
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed
  for the equivocation follower wiring + the `fsck` root-rebuild + M2; confirmed still absent). **Known
  stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json` fixtures (both `internal/didweb/`
  and `internal/logclient/`) and `derive_vkey.py` still carry sb1's PRE-rotation key (`22b08f3e`); the
  live sb1 signer is `069d0f14`. Captured in `verify_test.go` prose/tests (not green-but-wrong), but the
  did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `github.com/transparency-dev/merkle v0.0.2` (`logclient/consistency.go`), and
  `github.com/transparency-dev/tessera v1.0.2` (`internal/tiles/layout.go` — `api/layout` only). The
  `SQLiteFetcher` proves conformance to `tessera/fsck.Fetcher` **structurally** (local interface copy),
  so heavy tessera deps stay out of the store closure. **Not yet wired** (confirmed zero non-comment
  imports): the rest of tessera (`client`/`fsck`), `transparency-dev/formats`, `nbd-wtf/opentimestamps`.

- **Verify criteria status**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected + evidence-survives-
  restart). Coverage (`monitored_since`) — **tracked** and asserted set-once. The **third trigger,
  equivocation, has a verified pure verifier and now a read seam, but is NOT wired into the follower**,
  so the M1 Verify line is **not satisfied**.

## M2 — Aggregator
**Status**: not started — but **two prerequisite slices have landed**: `internal/tiles` re-exports
tessera's tlog-tiles layout math + the `IsFull` partial-tile predicate, and `internal/store/{tiles,
fetcher}.go` now provides the partial-tile mirror CRUD + `SQLiteFetcher` (structural `client.Fetcher`/
`fsck.Fetcher`). What remains for M2: tile/entry-bundle fixtures, the actual `fsck.New(...).Check(...)`
root-rebuild over `SQLiteFetcher` (the trust-root oracle re-arm), the `iscc_index` projection writer,
and `inclusion`/`consistency`/`entries` served via a `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `merkle v0.0.2` + `tessera v1.0.2` +
  `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable. Latest `review` handoff (2026-06-20,
  "SQLiteFetcher — read mirrored tiles/bundles/checkpoint back as a `fsck.Fetcher`", verdict
  **PASS_WITH_NOTES / CONTINUE**) records the gate green at HEAD `4eaead8`: `mise run check` green
  (build + vet + test, all 8 packages ok), `gofmt -l .` empty, `go mod tidy` no-op, `go mod verify`
  passes, store stays a leaf (`go list -deps` has no `net/http`). **Oracle/conformance gate correctly
  N/A this slice** (plain CRUD + synthetic BLOB round-trip — no signature/RFC-6962/Merkle/did:web/`fsck`-
  rebuild path); it re-arms at the `fsck` root-rebuild + equivocation-wiring slice. The one accepted
  note: the `var _ fsckFetcher` assertion uses a local interface copy (not the real `fsck` import) to
  avoid the `net/http`/`otel`/`klog` dep leak — equivalent drift-detection, recorded in learnings.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**, in sync
  with `origin/develop` (0/0); tree clean at HEAD `4eaead8`. **No `.github/workflows/` — no CI
  configured.** When CI is wired it must avoid `go build ./...` over the gitignored `cauldron/` reference
  trees and shell out the future `notecheck` oracle rather than `go run` from `cauldron/`.

## Next Milestone
Continue M1. The pure equivocation verifier, the `internal/tiles` layout seam, and the `SQLiteFetcher`
read seam are all done; the immediate gap is wiring the equivocation trigger into the follower and
capturing the tile fixtures that both the wiring test and the `fsck` root-rebuild need. Candidate
slices, in rough order:
1. **Wire `CheckEquivocation` into `follower.checkConsistency`** as the third branch (map
   `FollowState.LastSize → prevSize`, stored root → `prevRoot`, `info.TreeSize → nextSize`, `info.Root →
   nextRoot`, consistency proof sourced via `SQLiteFetcher`) → `ViolationEquivocation` → `RecordViolation`
   + evidence + `Freeze` + alert-once, no advance — closing the M1 Verify line. Needs **tile fixtures**
   for an end-to-end test, which re-arms the conformance/oracle gate.
2. **`fsck` root-rebuild conformance slice** — real tile fixtures + `fsck.New(...).Check(...)` over
   `SQLiteFetcher` + the inclusion cross-check vs the hub's `IsccLogInclusionProof`. First slice where
   the trust-root **oracle gate re-arms** for the mirror path; lives in `cmd/` or a future conformance
   package that *can* take the `fsck` dep (otel/klog closure), not the store leaf.
3. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
4. The lighter remaining M1 gaps — structured logs (`slog`, replacing the two stderr placeholders),
   `/metrics`, real alert transport — then complete M1's Verify criteria.
No CI is configured: flag for whoever sets up the workflow (load-bearing now that the merkle crypto path
has landed and the upcoming `fsck`-rebuild/fixture slices arm the external `notecheck` oracle — wire CI
**before** the `fsck`-rebuild conformance slice so the trust-root oracle has CI coverage when the mirror
path first faces it).
