<!-- assessed-at: 8a8279d821f385ed634ded3f5ac555b49950ffe0 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verify core + cadence + parser + config + `cmd/` binary + coverage + `hub_keys` cache (write + read + reader→follower wiring) are landed; the **pure equivocation verifier** (`CheckEquivocation` + real `transparency-dev/merkle`) has now landed but is **not yet wired into the follower**; logs, `/metrics`, real alert transport, tile fetch still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, and the full `hub_keys`
did:web key cache (write + read + fast-path-consulted-on-verified-poll) have landed. As of this slice
the **third self-consistency trigger's pure building block exists**: `CheckEquivocation` verifies an
RFC-6962 consistency proof via a genuine `github.com/transparency-dev/merkle v0.0.2`, golden-tested
against a real tree. What remains to close M1: **wire that verifier into the follower** (which needs
tile fetch / `SQLiteFetcher` / `ProofBuilder` to source the proof hashes — both still absent),
structured logs, `/metrics`, and a real alert transport (still a stderr placeholder).

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache write/read/wiring +
**pure equivocation verifier**; the equivocation trigger is not yet wired into the follower, and
logs/metrics/alert transport remain missing)

- Verified present (incremental re-check from prior assessment at `7a7a3fa`; the diff
  `7a7a3fa..HEAD` touches **only** `internal/logclient/consistency.go` + `consistency_test.go`, plus
  `go.mod`/`go.sum` (the new merkle dep), plus context files — confirmed: `internal/follower/` and
  `internal/store/` (incl. `schema.sql`) are **byte-unchanged**, fixtures unchanged. All non-consistency
  sections re-confirmed unchanged and carried forward):
  - **NEW — pure equivocation verifier landed** (`internal/logclient/consistency.go`): the third
    self-consistency trigger's Merkle building block. `CheckEquivocation(prevSize, prevRoot
    [rootBytes]byte, nextSize, nextRoot [rootBytes]byte, consistencyProof [][]byte) (violated bool, err
    error)` — for the strictly-growing case (`prevSize > 0 && nextSize > prevSize`) it calls
    `proof.VerifyConsistency(rfc6962.DefaultHasher, prevSize, nextSize, consistencyProof, prevRoot[:],
    nextRoot[:])`; a non-verifying proof is a **verdict** (`violated=true, err=nil`), never a Go error
    that could abort the poll loop ("freeze, never crash", ADR-0006). Boundaries `prevSize==0` /
    `nextSize==prevSize` / `nextSize<prevSize` all return `(false, nil)` AND skip `VerifyConsistency`
    (fork's / shrink's concern). New `ViolationEquivocation ViolationKind = "equivocation"`. Import
    block adds `merkle/proof` + `merkle/rfc6962`. **+3 `func Test`** (logclient 23→26): the review
    handoff records the golden built from a genuine `testonly.New(rfc6962.DefaultHasher)` tree (ground
    truth, not author-asserted), with valid `(M=7,N=11)`→`(false,nil)`, corrupted root / corrupted
    proof element →`(true,nil)`, and the three boundary-skip cases. The reviewer independently re-ran
    two mutations that each fail the suite — non-vacuous oracle gate.
  - **CRITICAL CAVEAT — `CheckEquivocation` is NOT yet wired into the follower** (confirmed: zero
    `CheckEquivocation`/`ViolationEquivocation` references in `internal/follower/` or `cmd/`). It is a
    *pure verdict function only*; the follower's `checkConsistency` still has only the shrink + fork
    branches. Wiring it in requires the consistency-proof hashes, which only a tile-fetch /
    `SQLiteFetcher` / `ProofBuilder` slice can source from mirrored hash tiles (per review's "Next"
    note: realistic order is tile-fetch first, then the follower branch). So the equivocation trigger
    is **not met end-to-end** through the poll loop.
  - **Schema caveat carried forward** (handoff): the cached `HubKey` row carries only `revoked_at` (no
    `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check **from the cache alone** is
    still not possible. The fast path refreshes/reuses the key but does not re-check the validity window
    from cache; documented limitation, not a green-but-wrong path.
  - **reader→follower wiring** (`internal/follower/follower.go`, unchanged this slice): `cacheHubKey`
    splits into `cacheHubKeyFast` (fetch-free: `KeyIDFromCheckpoint(raw) → (name, keyID)`, guards
    `name == Origin(baseURL)`, on `LookupHubKey` hit refreshes the row via `RecordHubKey` with NO
    second `ResolveVerifierKey`/did.json fetch) and `cacheHubKeyResolve` (cold fallback). Proven by
    `TestPollHubCacheHitSkipsDidFetch` (cold poll = 2 did.json fetches, warm poll = +1).
  - **`KeyIDFromCheckpoint`** (`internal/logclient/checkpointkey.go`) — pure raw-checkpoint key-id
    reader via `note.Open(raw, note.VerifierList())` (empty list) + `errors.As` on `*UnverifiedNoteError`.
    Golden `name=="sb0.iscc.id/log"`, `keyID==0x40b74463`, cross-check `==KeyIDFromVerifier(sb0VKey)`.
  - **`hub_keys` key reader** (`internal/store/checkpoints.go`): `LookupHubKey(ctx, hubID, keyID
    uint32) (HubKey, bool, error)` — inverse of `RecordHubKey`; "absent → zero + found=false + nil err".
  - **coverage tracking** (`internal/store/checkpoints.go` + `follower.go`): set-once
    `monitored_since_{size,time}` (ADR-0001), wired between `RecordCheckpoint` and `AdvanceFollowState`.
  - **`cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): wired M1 entrypoint; `alert(hubID,
    kind)` is a documented stderr placeholder (`fmt.Fprintln(os.Stderr, ...)`).
  - **exported `logclient.Origin`** + golden `TestOrigin`; **config loader leaf** (4 tests, `Frozen >=
    Normal` cross-check); **realm-registry parser** (3 tests, domains-only, fails closed on URLs);
    **poll-loop cadence** (`loop.go`, single-writer, pure `due`, frozen hubs re-poll at `Frozen`);
    **freeze + alert-once** (`checkConsistency` shrink-then-fork; `freeze` → `RecordViolation` +
    evidence `RecordCheckpoint` + `Freeze` + one-shot `AlertFunc`, gated on `!wasFrozen`).
  - `internal/store/checkpoints.go` — typed CRUD: `UpsertHub`/`RecordCheckpoint`/`FollowState`/
    `AdvanceFollowState`/`RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`/`RecordHubKey`/
    `LookupHubKey`; `CheckpointAt` recovers the prior accepted root via `LIMIT 1` (rowid order).
  - `internal/logclient/consistency.go` — now holds **all three** triggers: dep-free `CheckShrink` +
    `CheckFork`, and the merkle-backed `CheckEquivocation`. `transparency-dev/merkle` is now a real,
    imported dep (no longer prose-only).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (incl. `hub_keys`).
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin,keyid,checkpointkey}.go` —
    transport-only `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`, pure `VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`/`Origin()`, pure
    `KeyIDFromVerifier` + pure `KeyIDFromCheckpoint`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 26 (`logclient`, +3 for `CheckEquivocation`) + 31 (`store`) + 8
    (`follower`: 5 `follower_test.go` + 3 `loop_test.go`) + 3 (`registry`) + 4 (`config`) + 1
    (`cmd/iscc-monitor`) = **82 `func Test`**.

- Missing (the remaining M1 connective tissue):
  - **Equivocation trigger end-to-end** — the pure `CheckEquivocation` verifier exists, but the
    **follower wiring is unbuilt**: no third branch in `follower.checkConsistency`, and no source for
    the consistency-proof hashes. That source is the **tile-fetch / `SQLiteFetcher` / `ProofBuilder`**
    slice (M2 territory), which needs `transparency-dev/tessera` + tile fixtures (**both still absent**)
    and will trip the conformance/oracle gate (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion
    cross-check, golden-vector parity, `notecheck` in CI). This is the largest open M1 gap and needs
    decomposing into several slices.
  - **Structured logs** — no `slog` anywhere in `internal`/`cmd` (confirmed). Two stderr placeholders
    remain: `alert` in `cmd/iscc-monitor/main.go` and the per-tick swallowed error in `loop.go`
    (commented "surfaces via the store/metrics later").
  - **`/metrics`** — no metrics impl, no `expvar`/prometheus, no http server in `internal`/`cmd` (the
    only `metric` grep hit is a comment word in `loop.go:131`).
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder; production delivery
    (email/webhook) is unbuilt.

- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed for
  the equivocation follower wiring + M2). **Known stale-fixture drift, still not acted on:** the
  `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py`
  still carry sb1's PRE-rotation key (`z6MkiNW…` / `22b08f3e`); the live sb1 signer of the captured
  checkpoint is `069d0f14`. `verify_test.go` documents both keys and tests the stale-key mismatch, so the
  drift is *captured in prose/tests* (not green-but-wrong) — but the did.json fixtures themselves remain
  stale. The refresh remains its own trust-root step (re-arms the oracle gate).

- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`,
  `logclient/checkpointkey.go`), `modernc.org/sqlite` (`store/sqlite.go`), and **now**
  `github.com/transparency-dev/merkle` v0.0.2 (`logclient/consistency.go` — `proof` + `rfc6962`).
  **Not yet wired** (confirmed zero non-comment import statements + zero go.mod entries):
  `transparency-dev/tessera` (`client`/`api`/`fsck`), `transparency-dev/formats`,
  `nbd-wtf/opentimestamps`.

- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected +
  evidence-survives-restart, through the poll loop). Coverage (`monitored_since`) — **tracked** and
  asserted set-once. The **third trigger, equivocation, has a verified pure verifier but is NOT wired
  into the follower** (no `checkConsistency` branch, no proof source), so the M1 Verify line ("synthetic
  …equivocation → correct `violations.kind` + `frozen=1` + …") is **not satisfied**.

## M2 — Aggregator
**Status**: not started. (Note: the tile-fetch / `SQLiteFetcher` / `ProofBuilder` slice that M2 needs is
also a prerequisite for wiring the M1 equivocation trigger, so it may be pulled forward.)

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `merkle v0.0.2` + `x/mod v0.33.0` +
  `sqlite v1.46.1`); `mise run check` runnable. Latest `review` handoff (2026-06-20, "Pure RFC-6962
  consistency-proof verifier (`CheckEquivocation`) + `transparency-dev/merkle` dep", verdict
  **PASS_WITH_NOTES / CONTINUE**) records the gate green at HEAD `8a8279d`: `mise run check` green
  (build + vet + test, all 7 packages ok, re-run uncached), `gofmt -l .` empty, `go mod tidy` no-op,
  `go mod verify` passes. **Oracle/conformance gate APPLIES this step (RFC-6962/merkle crypto) and is
  satisfied**: golden is ground truth from `testonly.New(rfc6962.DefaultHasher)`, prover and verifier
  are independent merkle paths, reviewer re-ran two mutations that each fail the suite. `notecheck`/
  `derive_vkey.py`/`fsck` correctly N/A (no signature/did:web/tile path this slice).
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop` (0/0); tree clean at HEAD `8a8279d`. **No `.github/workflows/` — no CI configured.**
  When CI is wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and
  shell out the future `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. The pure equivocation verifier is **done**; the immediate gap is sourcing its proof hashes
and wiring it into the follower. Remaining candidate slices, in rough order:
1. **Tile-fetch / `SQLiteFetcher` / `ProofBuilder`** (M2 infrastructure, pulled forward): the only way
   to source the RFC-6962 consistency-proof hashes `CheckEquivocation` consumes. Needs
   `transparency-dev/tessera` (new dep) + tile fixtures (both absent) and **trips the conformance/oracle
   gate** (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check, golden-vector parity,
   `notecheck` in CI). Needs decomposing into multiple slices.
2. **Wire `CheckEquivocation` into `follower.checkConsistency`** as the third branch (map
   `FollowState.LastSize → prevSize`, the stored root at that size → `prevRoot`, `info.TreeSize →
   nextSize`, `info.Root → nextRoot`, fetched consistency proof), returning `ViolationEquivocation` →
   `RecordViolation` + evidence `RecordCheckpoint` + `Freeze` + alert-once, no advance — closing the M1
   Verify line. Depends on (1) for the proof source.
3. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`
   HUBS) — its own trust-root step that re-arms the oracle gate.
4. The lighter remaining M1 gaps — structured logs (`slog`, replacing the two stderr placeholders),
   `/metrics`, real alert transport — then complete M1's Verify criteria.
No CI is configured: flag for whoever sets up the workflow (it becomes load-bearing now that the merkle
crypto path has landed and the tile-fetch / fixture-refresh slices arm the external `notecheck` oracle).
