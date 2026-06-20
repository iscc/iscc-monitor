<!-- assessed-at: c3e02c472b9f953b15b2965f461ce638fb1ff4af -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verify core + cadence + parser + config + `cmd/` binary + coverage + `hub_keys` cache-write wired into `PollHub` + **now a `hub_keys` key reader (`LookupHubKey`)**; equivocation trigger + logs/metrics/alert still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking, the follower wiring of the
`hub_keys` did:web key cache write, and now the **read side of that cache (`store.LookupHubKey`)** have
landed. What still blocks M1: the merkle-backed **equivocation** trigger (the third of three
self-consistency triggers), structured logs, `/metrics`, and real alert transport (a stderr
placeholder). The reader exists but is **not yet threaded into the follower** to skip the second
did.json fetch — that wiring is the next slice.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache write **and read**;
equivocation trigger + logs/metrics/alert tissue still missing)

- Verified present (incremental re-check from prior assessment at `4b000ca`; the only Go change since is
  the `LookupHubKey` slice — `internal/store/checkpoints.go` + `internal/store/checkpoints_test.go`. All
  other sections re-confirmed unchanged: `go.mod`/`go.sum` and `schema.sql` byte-identical
  (empty diff `4b000ca..HEAD`), fixtures unchanged, no new reuse imports):
  - **NEW — `hub_keys` key reader** (`internal/store/checkpoints.go` line 365): `LookupHubKey(ctx,
    hubID int64, keyID uint32) (HubKey, bool, error)` — the column-by-column inverse of `RecordHubKey`
    (line 324). Mirrors the sibling `FollowState`/`Coverage` "absent row → zero value + found=false +
    nil error" convention, maps NULL columns back through `sql.Null*`, reconstructs `HubID`/`KeyID`
    from the lookup args (so the `uint32` is never recovered from the signed `int64` column — no
    truncation/sign risk). **+4 `func Test`** (`TestLookupHubKey{RoundTrip,NullableRoundTrip,Absent,
    DiscriminatesKeyID}`). Store stays a leaf (no `net/http` in its closure, per the review handoff).
  - **Reader not yet wired into the follower** (confirmed: `cacheHubKey` in `internal/follower/` still
    re-resolves did.json on every verified poll; nothing calls `LookupHubKey` outside the store tests).
    So a verified poll still fetches did.json twice. Threading `LookupHubKey` into `PollHub` to skip the
    second fetch when the cached key is still in its CID-1.0 validity window is the obvious next slice.
  - **`hub_keys` cache write wired into `PollHub`** (`internal/follower/follower.go`): on the verified,
    non-violation path `cacheHubKey` re-runs `ResolveVerifierKey`, recovers the key id via pure
    `logclient.KeyIDFromVerifier` (`internal/logclient/keyid.go`), and upserts `store.HubKey`. Follower
    tests assert a verified poll writes exactly 1 `hub_keys` row (`key_id==0x40b74463`, 32-byte
    `pubkey_raw`), a second poll stays at 1 (refresh in place), and fork/shrink/unverified write 0.
  - **coverage tracking** (`internal/store/checkpoints.go` + `internal/follower/follower.go`): set-once
    `monitored_since_{size,time}` per ADR-0001. Wired in `PollHub` between `RecordCheckpoint` and
    `AdvanceFollowState`, outside `freeze`. Store tests + follower assertions.
  - **`cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): wired M1 entrypoint; `alert(hubID, kind)`
    is a documented stderr placeholder. 1 `func Test`.
  - **exported `logclient.Origin`** + golden `TestOrigin`; **config loader leaf** (4 tests, `Frozen >=
    Normal` cross-check); **realm-registry parser** (3 tests, domains-only, fails closed on URLs);
    **poll-loop cadence** (`loop.go`, single-writer, pure `due`, frozen hubs re-poll at `Frozen`);
    **freeze + alert-once** (`checkConsistency` shrink-then-fork; `freeze` → `RecordViolation` +
    evidence `RecordCheckpoint` + `Freeze` + one-shot `AlertFunc`).
  - `internal/store/checkpoints.go` — typed CRUD: `UpsertHub`/`RecordCheckpoint`/`FollowState`/
    `AdvanceFollowState`/`RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`/`RecordHubKey`/
    **`LookupHubKey`**; `CheckpointAt` recovers the prior accepted root via `LIMIT 1` (rowid order).
  - `internal/logclient/consistency.go` — dep-free `CheckShrink` + `CheckFork` + `ViolationKind`;
    `transparency-dev/merkle` is prose-only (not in go.mod, not imported).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (incl. `hub_keys`).
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin,keyid}.go` — transport-only
    `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`, pure `VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`/`Origin()`, pure
    `KeyIDFromVerifier`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 20 (`logclient`) + 31 (`store`) + 7 (`follower`) + 3 (`registry`)
    + 4 (`config`) + 1 (`cmd/iscc-monitor`) = **75 `func Test`** (store +4 for `LookupHubKey`; note the
    logclient count is 20, correcting a prior off-by-one in state — logclient was untouched this step).

- Missing (still the connective majority of M1):
  - **Equivocation trigger** — the third of three triggers (RFC-6962 consistency-proof failure across
    *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and will
    trip the conformance/oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
    More than one verifiable slice — needs decomposing.
  - **Reader→follower wiring** — `LookupHubKey` exists but `cacheHubKey` does not consult it, so the
    verified path still fetches did.json twice per poll.
  - structured logs (replacing the two stderr placeholders: `alert` + the per-tick swallowed error),
    `/metrics` (no `slog`, no metrics impl anywhere in `internal`/`cmd`).
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder; production delivery
    (email/webhook) is unbuilt.

- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed for
  the equivocation trigger + M2). **Known stale-fixture drift, still not acted on:** the
  `sb1.amlet.id_did.json` fixtures (both `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py`
  still carry sb1's PRE-rotation key (`z6MkiNW…` / `22b08f3e`); the live sb1 signer of the captured
  checkpoint is `069d0f14`. `verify_test.go` documents both keys and tests the stale-key mismatch, so the
  drift is *captured in prose/tests*, not a green-but-wrong test — but the did.json fixtures themselves
  remain stale. The refresh remains its own trust-root step (re-arms the oracle gate).

- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero non-comment import
  statements + zero go.mod entries): `transparency-dev/*` (merkle/tessera/formats),
  `nbd-wtf/opentimestamps`.

- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected +
  evidence-survives-restart, through the poll loop). Coverage (`monitored_since`) — **tracked** and
  asserted set-once. The **third trigger, equivocation, is not met** (absent), so the M1 Verify line is
  not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "Add `store.LookupHubKey` — the read
  side of the `hub_keys` did:web key cache", verdict **PASS / CONTINUE**) records the gate green at HEAD
  `c3e02c4`: `mise run check` green (build + vet + test, 7 packages ok), `gofmt -l .` empty,
  `go.mod`/`go.sum` + `schema.sql` byte-identical (confirmed here: empty diff `4b000ca..HEAD`), store
  stays a leaf (no `net/http` in its closure), no gate-dodging across unpushed commits. Oracle/conformance
  gate correctly **N/A** this step (pure CRUD diff — no `proof`/`verify`/`didweb`/`merkle`/`fsck` path).
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` (upstream `origin/develop`);
  tree clean at HEAD `c3e02c4`. **No `.github/workflows/` — no CI configured.** When CI is wired it must
  avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future
  `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. Candidate slices, in rough order:
1. **Thread `LookupHubKey` into the follower** — consult the cache in `cacheHubKey`/`PollHub` so a
   verified poll skips the second did.json fetch while the cached key is in its CID-1.0 validity window.
   Small, unblocked by the just-landed reader. Cheapest open slice.
2. **Merkle-backed equivocation trigger** (RFC-6962 consistency-proof failure across growing sizes): a
   third branch in the `checkConsistency` seam returning `ViolationEquivocation` + real `ProofJSON`.
   Needs `transparency-dev/merkle` (new dep) + tile fixtures and **trips the conformance/oracle gate**
   (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check, golden-vector parity, `notecheck`
   in CI). Must compare against the prior *accepted* root (`CheckpointAt` `LIMIT 1` rowid-order). Needs
   decomposing into multiple slices.
3. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`
   HUBS) — its own trust-root step that re-arms the oracle gate.
4. The lighter remaining M1 gaps — structured logs, `/metrics`, real alert transport — then complete
   M1's Verify criteria.
No CI is configured: flag for whoever sets up the workflow (it becomes load-bearing the moment the
merkle path or a fixture refresh lands).
