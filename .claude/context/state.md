<!-- assessed-at: ac4a643b3c5181018a93620043ab6c30fe577c0a -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress (with first M2-prerequisite slice landed) — verify core + cadence + parser + config + `cmd/` binary + coverage + `hub_keys` cache + **pure equivocation verifier** + **`internal/tiles` tlog-tiles layout seam** are landed; equivocation is still **not wired into the follower**, and `SQLiteFetcher`/`ProofBuilder` + tile fixtures + structured logs + `/metrics` + real alert transport are still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, the full `hub_keys` did:web
key cache, and the pure `CheckEquivocation` consistency-proof verifier have all landed. The newest slice
adds `internal/tiles` — a thin golden-tested re-export of tessera's tlog-tiles layout primitives — which
is the first building block of the **tile-fetch / `SQLiteFetcher` / `ProofBuilder`** infrastructure that
both M2 and the M1 equivocation-wiring depend on. What still blocks M1: wire `CheckEquivocation` into
the follower (needs the `SQLiteFetcher`/`ProofBuilder` to source proof hashes — not yet built),
structured logs, `/metrics`, and a real alert transport.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache + pure equivocation
verifier + tlog-tiles layout seam; the equivocation trigger is **not yet wired into the follower**, and
logs/metrics/alert transport remain missing)

- **Verified incrementally** from the prior assessment at `8a8279d`. The diff `8a8279d..HEAD` touches
  **only** the new `internal/tiles/` package (`layout.go` + `layout_test.go`), `go.mod`/`go.sum` (the
  tessera dep add), and context files. Confirmed byte-unchanged: `internal/follower/`, `internal/store/`,
  `internal/logclient/`, `internal/didweb/`, `internal/registry/`, `internal/config/`, `cmd/`, and the
  fixtures. All sections below other than the new `internal/tiles` line are re-confirmed unchanged and
  carried forward.

  - **NEW — `internal/tiles` tlog-tiles layout seam landed** (`internal/tiles/layout.go`): a pure leaf
    that re-exports (never reimplements, per the Stack rule) tessera's `api/layout` primitives —
    `TilePath`, `EntriesPath`, `PartialTileSize`, consts `TileWidth=256` / `TileHeight=8` — plus the one
    ADR-0005 predicate `IsFull(width int) bool` (column convention: `width==256` is full). `net/http`-
    and `database/sql`-free, WASM-green. **+5 `func Test`**, golden vectors confirmed by `review` against
    tessera ground truth (path strings verbatim from tessera's own `paths_test.go`). Review also caught
    and corrected a wrong `PartialTileSize(0,0,300)` vector from `next.md` (correct: `(0,0,300)=0`,
    `(0,1,300)=44`). This is the **first building block** of the deferred tile-fetch infrastructure; it
    is an intentional unused-until-wired export seam (`go vet` clean, not dead code) whose first caller
    will be the `SQLiteFetcher` store-key slice.

  - **Pure equivocation verifier** (`internal/logclient/consistency.go`, unchanged this slice):
    `CheckEquivocation(prevSize, prevRoot, nextSize, nextRoot, consistencyProof) (violated, err)` — for
    the strictly-growing case calls `proof.VerifyConsistency(rfc6962.DefaultHasher, ...)`; a non-verifying
    proof is a **verdict** (`violated=true, err=nil`), never a crash ("freeze, never crash", ADR-0006).
    Boundaries (`prevSize==0` / `nextSize==prevSize` / `nextSize<prevSize`) return `(false,nil)` and skip
    `VerifyConsistency`. Golden-tested against a genuine `testonly.New(rfc6962.DefaultHasher)` tree.

  - **CRITICAL CAVEAT — `CheckEquivocation` is STILL NOT wired into the follower** (confirmed: zero
    `CheckEquivocation`/`ViolationEquivocation` references in `internal/follower/` or `cmd/`). It is a
    pure verdict function only; `follower.checkConsistency` still has only the shrink + fork branches.
    Wiring needs the consistency-proof hashes, which only the `SQLiteFetcher`/`ProofBuilder` can source
    from mirrored hash tiles — **not yet built**. So the equivocation trigger is **not met end-to-end**.

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

  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (incl. `hub_keys`).

  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin,keyid,checkpointkey,consistency}.go`
    — transport-only `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`/`VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, and all three consistency triggers (dep-free
    `CheckShrink`/`CheckFork`, merkle-backed `CheckEquivocation`).

  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/
    `DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.

  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.

  - **Test totals (verified by grep)**: 9 (`didweb`) + 26 (`logclient`) + 31 (`store`) + 8 (`follower`) +
    3 (`registry`) + 4 (`config`) + **5 (`tiles`, new)** + 1 (`cmd/iscc-monitor`) = **87 `func Test`**.

- **Missing (remaining M1 connective tissue):**
  - **Equivocation trigger end-to-end** — the pure `CheckEquivocation` verifier exists, but the follower
    wiring is unbuilt (no third branch in `checkConsistency`, no source for the proof hashes). That source
    is the **`SQLiteFetcher` / `ProofBuilder`** slice (M2 territory, partially scaffolded now by
    `internal/tiles`) — needs tile fixtures (still absent) and will arm the conformance/oracle gate.
  - **Structured logs** — no `slog` anywhere in `internal`/`cmd` (confirmed). Two stderr placeholders
    remain: `alert` in `cmd/iscc-monitor/main.go` and the per-tick swallowed error in `loop.go`.
  - **`/metrics`** — no metrics impl, no `expvar`/prometheus, no http server (confirmed).
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed
  for the equivocation follower wiring + M2; confirmed still absent). **Known stale-fixture drift, still
  not acted on:** the `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and `internal/logclient/`)
  and `derive_vkey.py` still carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`.
  Captured in `verify_test.go` prose/tests (not green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `github.com/transparency-dev/merkle v0.0.2` (`logclient/consistency.go`), and **now**
  `github.com/transparency-dev/tessera v1.0.2` (`internal/tiles/layout.go` — `api/layout` only;
  module-graph-only heavy deps stay out of the package closure). **Not yet wired** (confirmed zero
  non-comment imports): the rest of tessera (`client`/`api`/`fsck`), `transparency-dev/formats`,
  `nbd-wtf/opentimestamps`.

- **Verify criteria status**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected + evidence-survives-
  restart). Coverage (`monitored_since`) — **tracked** and asserted set-once. The **third trigger,
  equivocation, has a verified pure verifier but is NOT wired into the follower**, so the M1 Verify line
  is **not satisfied**.

## M2 — Aggregator
**Status**: not started — but the **first prerequisite slice has landed**: `internal/tiles` re-exports
tessera's tlog-tiles layout math + the `IsFull` partial-tile predicate. The next M2 slice is the
`SQLiteFetcher` (tessera `client.Fetcher` seam over the store's `tiles`/`entry_bundles` tables), which
is also the prerequisite for wiring the M1 equivocation trigger, so it is being pulled forward. Tile
fixtures are still absent and must be captured for both the SQLiteFetcher and the `fsck`/inclusion oracle.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `merkle v0.0.2` + `tessera v1.0.2` +
  `x/mod v0.33.0` + `sqlite v1.46.1`); `mise run check` runnable. Latest `review` handoff (2026-06-20,
  "Add the canonical tlog-tiles layout layer (`internal/tiles`)", verdict **PASS / CONTINUE**) records
  the gate green at HEAD `ac4a643`: `mise run check` green (build + vet + test, all 8 packages ok),
  `gofmt -l .` empty, `go mod tidy` no-op, `go mod verify` passes, `GOOS=js GOARCH=wasm go build
  ./internal/tiles` green. **Oracle/conformance gate correctly N/A this slice** (pure path strings — no
  signature/RFC-6962/Merkle/did:web/fsck path); it re-arms at the `SQLiteFetcher` + `fsck` slice.
- Remote `origin` configured (github.com/iscc/iscc-monitor); working branch is **`develop`**, in sync
  with `origin/develop` (0/0); tree clean at HEAD `ac4a643`. **No `.github/workflows/` — no CI
  configured.** When CI is wired it must avoid `go build ./...` over the gitignored `cauldron/` reference
  trees and shell out the future `notecheck` oracle rather than `go run` from `cauldron/`.

## Next Milestone
Continue M1. The pure equivocation verifier and the `internal/tiles` layout seam are done; the immediate
gap is sourcing the equivocation proof hashes and wiring the trigger into the follower. Candidate slices,
in rough order:
1. **`SQLiteFetcher`** — implement the tessera `client.Fetcher` seam (`{ReadCheckpoint, ReadTile,
   ReadEntryBundle}`) over the store's `tiles`/`entry_bundles` tables, keyed by `tiles.TilePath`/
   `EntriesPath` and gated by `IsFull`. The only way to source the RFC-6962 consistency-proof hashes
   `CheckEquivocation` consumes, and the foundation for the M3 canonical-path mirror. **Needs tile
   fixtures (still absent) and re-arms the conformance/oracle gate** (`fsck` root-rebuild over the
   `SQLiteFetcher`, inclusion cross-check, golden-vector parity, `notecheck` in CI).
2. **Wire `CheckEquivocation` into `follower.checkConsistency`** as the third branch (map
   `FollowState.LastSize → prevSize`, stored root → `prevRoot`, `info.TreeSize → nextSize`, `info.Root →
   nextRoot`, fetched consistency proof) → `ViolationEquivocation` → `RecordViolation` + evidence +
   `Freeze` + alert-once, no advance — closing the M1 Verify line. Depends on (1).
3. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
4. The lighter remaining M1 gaps — structured logs (`slog`, replacing the two stderr placeholders),
   `/metrics`, real alert transport — then complete M1's Verify criteria.
No CI is configured: flag for whoever sets up the workflow (load-bearing now that the merkle crypto path
has landed and the upcoming `SQLiteFetcher`/fixture slices arm the external `notecheck` oracle).
