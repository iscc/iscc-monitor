<!-- assessed-at: fe4b9d7484398f6f3e464745425889c378514bd2 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verify core + cadence + parser + config + `cmd/` binary + coverage tracking + `hub_keys` cache-write landed; equivocation trigger + follower-wiring of the key cache + logs/metrics/alert still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, set-once coverage tracking (`monitored_since`,
ADR-0001), and now the store-leaf `hub_keys` did:web key-cache **write** (`RecordHubKey`) have landed.
What still blocks M1: the merkle-backed **equivocation** trigger (the third of three self-consistency
triggers), the follower→store **wiring** of `RecordHubKey` (plus a key *reader*) — which also owns the
stale-sb1 fixture refresh — structured logs, `/metrics`, and real alert transport (a stderr placeholder).

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking + `hub_keys` cache-write done;
equivocation trigger and the follower-wiring/reader + logs/metrics/alert tissue still missing)
- Verified present (incremental re-check from prior assessment at `23f26bc`; the only Go change since is
  the `hub_keys` cache-write slice — `internal/store/checkpoints.go` (`HubKey` struct + `RecordHubKey` +
  `nullStringOrNil`) and its `_test.go`. All other sections re-confirmed unchanged on disk: go.mod/go.sum
  and schema.sql byte-identical since `23f26bc`):
  - **NEW — `hub_keys` did:web key-cache write** (`internal/store/checkpoints.go` lines 71/324/365):
    `HubKey` struct + `RecordHubKey(ctx, HubKey) error` as a guarded `UPDATE … WHERE hub_id=? AND
    key_id=?` then `INSERT` on zero `RowsAffected` (the `SetCoverage` idiom; `hub_keys` has no UNIQUE,
    so no `ON CONFLICT`). Dedupe per `(hub_id, key_id)`; refresh-in-place (true cache of the DID doc, not
    append-only); rotation appends a row under a new `key_id`; FK to `hubs` enforced. `nullStringOrNil`
    maps empty `pubkey_z` → NULL (mirrors `unixOrNil` for zero `revoked_at`/`resolved_at`). No schema
    change, no new dependency, store stays a leaf. **5 new `func Test`** (`TestRecordHubKey{Insert,
    Refresh,Rotation,Nullable,ForeignKey}`). **Not yet wired into the follower** (confirmed: zero
    `RecordHubKey` refs in `internal/follower/` or `cmd/`) and **no reader added** (confirmed: the only
    `func …HubKey` in the store is `RecordHubKey` — no lookup) — both intentionally deferred to the
    wiring slice.
  - **coverage tracking** (`internal/store/checkpoints.go` + `internal/follower/follower.go`): set-once
    `monitored_since_{size,time}` per ADR-0001. `SetCoverage` is a guarded immutable-start UPDATE
    (re-call is a silent no-op); `Coverage` reads both columns via `sql.NullInt64`. Wired in `PollHub`
    on the verified, non-violation path between `RecordCheckpoint` and `AdvanceFollowState`, outside
    `freeze`. **3 `func Test`** (`TestCoverageSetOnce`/`TestCoverageUnset`/`TestCoverageZeroObservedAtNull`)
    + follower assertions.
  - **`cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): wired M1 entrypoint. `run()` does
    `config.Load` → `os.ReadFile(RealmPath)` → `registry.Parse` → `store.Open` → `registerHubs` →
    `follower.Loop{…}.Run(ctx)` under `signal.NotifyContext(os.Interrupt)`. `alert(hubID, kind)` is a
    documented stderr placeholder. **1 `func Test`** (`TestRegisterHubs`).
  - **exported `logclient.Origin`** (`internal/logclient/origin.go`): wrapper over private `origin`;
    golden `TestOrigin` vectors cover it.
  - **config loader leaf** (`internal/config/config.go`): pure `Load(get) (Config, error)`; required
    `ISCC_MONITOR_{DB,REALM}`, optional `{NORMAL,FROZEN}` (5m/1h), load-bearing `Frozen >= Normal`
    cross-check. **4 `func Test`.**
  - **realm-registry parser** (`internal/registry/registry.go`): pure `Parse([]byte) ([]Entry, error)`,
    domains-only (ADR-0009), fails closed on URL-shaped lines. **3 `func Test`.**
  - **poll-loop cadence** (`internal/follower/loop.go`): `Loop` drives `PollHub` from one goroutine
    (single writer, ADR-0005/0007); pure `due(...)`; frozen hubs re-poll at the longer `Frozen` interval.
  - **freeze + alert-once wiring** (`internal/follower/follower.go`): on `StatusVerified`, `PollHub` runs
    `checkConsistency` (shrink-then-fork vs prior accepted checkpoint) before record/advance; on a true
    verdict `freeze` does `RecordViolation` + `RecordCheckpoint` (evidence, no advance) + `Freeze`, then
    fires `AlertFunc` once iff `!wasFrozen`. (`TestPollHubFork`/`TestPollHubShrink`.)
  - `internal/store/checkpoints.go` — typed CRUD: `UpsertHub`/`RecordCheckpoint`/`FollowState`/
    `AdvanceFollowState`/`RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`/`RecordHubKey`;
    `CheckpointAt` recovers the prior accepted root via `LIMIT 1` (rowid order).
  - `internal/logclient/consistency.go` — dep-free `CheckShrink` + `CheckFork` + `ViolationKind`;
    `transparency-dev/merkle` is prose-only (single grep hit is a comment on line 21 — not in go.mod,
    not imported; confirmed).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (incl. `hub_keys`).
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin}.go` — transport-only
    `FetchCheckpoint`, pure 4-way `AcceptCheckpoint`, pure `VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`/`Origin()`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 19 (`logclient`) + 27 (`store`) + 7 (`follower`) + 3 (`registry`)
    + 4 (`config`) + 1 (`cmd/iscc-monitor`) = **70 `func Test`** (up from 65; store +5 for `RecordHubKey`).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — the third of three triggers (RFC-6962 consistency-proof failure across
    *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and will
    trip the conformance/oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
  - **Follower-wiring of `RecordHubKey` + a key reader** — `RecordHubKey` is built but never called from
    `PollHub`, and there is no `HubKey` lookup. This slice also MUST refresh the stale sb1 fixture
    (touches the trust-root → re-arms the oracle gate) and is a good place for a CI `notecheck` job.
  - structured logs (replacing the two stderr placeholders: `alert` + the per-tick swallowed error),
    `/metrics`.
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder; production delivery
    (email/webhook) is unbuilt.
- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (confirmed:
  `find testdata` returns just the two `_checkpoint` files + the `_did.json` vectors; needed for the
  equivocation trigger + M2). Known stale-fixture drift (still not acted on): `derive_vkey.py` HUBS +
  the `sb1.amlet.id_did.json` fixtures carry sb1's PRE-rotation key (`22b08f3e`, confirmed still present
  in five test files); the live sb1 signer is now `069d0f14`; refresh lands with the follower-wiring step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero non-comment import
  statements + zero go.mod entries): `transparency-dev/*` (merkle/tessera/formats),
  `nbd-wtf/opentimestamps`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink
  AND fork each → correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected +
  evidence-survives-restart, through the poll loop). Coverage (`monitored_since`) — **tracked** and
  asserted set-once through `PollHub`. The **third trigger, equivocation, is not met** (absent), so the
  M1 Verify line is not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "`hub_keys` did:web key cache —
  store-leaf `RecordHubKey` upsert") records the gate green at HEAD `fe4b9d7`: `go build` / `go vet` /
  `go test ./...` all `ok` (7 packages), `gofmt -l .` empty, `go.mod`/`go.sum` + `schema.sql`
  byte-identical (confirmed here: no dep added and no schema change since `23f26bc`), no gate-dodging.
  Oracle/conformance gate correctly **N/A** this step (plain `hub_keys`-column CRUD with NULL handling +
  FK; no proof/verify/didweb/merkle/consistency/fsck/signature path touched). The merkle-backed
  equivocation slice — and the follower-wiring slice that resolves a live key into the cache — *will*
  re-arm the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `fe4b9d7`. **No `.github/workflows/` — no CI configured.** When CI is wired it must avoid
  `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future `notecheck`
  oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. Two candidate slices, both still open:
1. **Merkle-backed equivocation trigger** (RFC-6962 consistency-proof failure across growing sizes): a
   third branch in the same `checkConsistency` seam returning `ViolationEquivocation` + real `ProofJSON`.
   Needs `transparency-dev/merkle` (new dep) + tile fixtures and **trips the conformance/oracle gate**
   (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check vs `IsccLogInclusionProof`,
   golden-vector parity, `notecheck` parity in CI). Must compare against the prior *accepted* root, not
   the contradicting evidence (`CheckpointAt` `LIMIT 1` rowid-order learning).
2. **Follower-wiring of the `hub_keys` cache + a key reader** — call `RecordHubKey` from `PollHub` after
   a successful `ResolveVerifierKey`, add a `HubKey` lookup, and finally refresh the stale
   `sb1.amlet.id_did.json` + `derive_vkey.py` HUBS to signer `069d0f14` (re-arms the oracle gate / parity).
The lighter remaining M1 gaps — structured logs, `/metrics`, real alert transport — then complete M1's
Verify criteria. No CI is configured: flag for whoever sets up the workflow (it becomes load-bearing the
moment the merkle path or the live-key resolution lands).
