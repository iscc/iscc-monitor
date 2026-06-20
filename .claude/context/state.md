<!-- assessed-at: 23f26bc95b9a44114d14a4dfaead1e1d4f591ad4 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verify core + cadence + parser + config + `cmd/` binary + coverage tracking landed; equivocation trigger + did:web cache + logs/metrics/alert still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), the wired `cmd/iscc-monitor` binary, and now set-once **coverage tracking**
(`monitored_since_{size,time}`, ADR-0001) have all landed. What still blocks M1: the merkle-backed
equivocation trigger (the third of three self-consistency triggers), the `hub_keys` did:web cache
write, structured logs, `/metrics`, and real alert transport (currently a stderr placeholder).

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary + coverage tracking done; equivocation trigger and the
did:web-cache/logs/metrics/alert tissue still missing)
- Verified present (incremental re-check from prior assessment at `d5db9f3`; the only Go changes since
  are the coverage-tracking slice — `internal/store/checkpoints.go` (`SetCoverage`/`Coverage`/
  `CoverageInfo`) and one wiring block in `internal/follower/follower.go`. All other sections
  re-confirmed unchanged on disk):
  - **NEW — coverage tracking** (`internal/store/checkpoints.go` + `internal/follower/follower.go`):
    set-once `monitored_since_{size,time}` per ADR-0001. `SetCoverage(ctx, hubID, size, observedAt)`
    is a guarded `UPDATE hubs SET monitored_since_size=?, monitored_since_time=? WHERE hub_id=? AND
    monitored_since_size IS NULL` — immutable start (re-call is a silent no-op; `RowsAffected` ignored).
    `Coverage(ctx, hubID) (CoverageInfo, error)` reads both columns through `sql.NullInt64` (absent/
    un-started hub → zero value, nil err, mirroring `FollowState`). Schema confirmed: `monitored_since_size`
    + `monitored_since_time` INTEGER columns on `hubs` (schema.sql lines 24-25). Wired in `PollHub` on
    the verified, non-violation path **between** `RecordCheckpoint` and `AdvanceFollowState` (line 114),
    explicitly **outside** `freeze` (the `violated` branch returns early at line 103), so a contradictory
    observation never starts coverage. **3 new `func Test`** (`TestCoverageSetOnce`, `TestCoverageUnset`,
    `TestCoverageZeroObservedAtNull`) + follower assertions (verified poll sets coverage to fixture size
    `10183` + injected `observedAt`; 24h-later re-poll does not move it; fork-freeze + unverified both
    leave coverage unset). No new dependency (go.mod/go.sum byte-identical); store stays a leaf.
  - **`cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): the wired M1 entrypoint.
    `run()` does `config.Load(os.LookupEnv)` → `os.ReadFile(cfg.RealmPath)` → `registry.Parse` →
    `store.Open(cfg.DBPath)` → `registerHubs` → `follower.Loop{…}.Run(ctx)` under a
    `signal.NotifyContext(os.Interrupt)`. `registerHubs` derives origin via exported `logclient.Origin`
    (`<domain>/log`) and `UpsertHub`s idempotently. `alert(hubID, kind)` is a documented stderr
    placeholder. **1 `func Test`** (`TestRegisterHubs`). `Loop.Run` correctly untested (blocking ticker).
  - **exported `logclient.Origin`** (`internal/logclient/origin.go`): one-line wrapper delegating to
    private `origin`; golden `TestOrigin` vectors cover it.
  - **config loader leaf** (`internal/config/config.go`): pure `Load(get) (Config, error)` →
    `Config{DBPath, RealmPath, Normal, Frozen}`. Required `ISCC_MONITOR_{DB,REALM}`; optional
    `{NORMAL,FROZEN}` default 5m/1h; load-bearing `Frozen >= Normal` cross-check. Imports `{fmt time}`.
    **4 `func Test`.**
  - **Realm-registry parser** (`internal/registry/registry.go`): pure `Parse([]byte) ([]Entry, error)`
    → ordered `Entry{Domain, BaseURL="https://"+Domain}`, drops blank/`#`, fails closed on URL-shaped
    lines, no key field (ADR-0009). Imports `{bufio bytes fmt strings}`. **3 `func Test`.**
  - **Poll-loop cadence wrapper** (`internal/follower/loop.go`): `Loop` drives `PollHub` over a fixed
    `[]HubTarget` from one goroutine (single writer, ADR-0005/0007). Pure `due(...)`; frozen hubs
    re-poll at the longer `Frozen` interval; `lastPoll` lazily inited in `Tick`.
  - **Freeze + alert-once wiring** (`internal/follower/follower.go`): on `StatusVerified`, `PollHub`
    runs `checkConsistency` (shrink-then-fork vs prior accepted checkpoint) BEFORE any record/advance;
    on a true verdict `freeze(...)` does `RecordViolation` + `RecordCheckpoint` (evidence, no advance) +
    `Freeze`, then fires `AlertFunc` once iff `!wasFrozen`. Tested by `TestPollHubFork`/`TestPollHubShrink`.
  - `internal/store/checkpoints.go` — `store.CheckpointAt` recovers prior accepted root via `LIMIT 1`
    (rowid order). Typed CRUD: `UpsertHub`/`RecordCheckpoint`/`FollowState`/`AdvanceFollowState`/
    `RecordViolation`/`Freeze`/`SetCoverage`/`Coverage`.
  - `internal/logclient/consistency.go` — dep-free `CheckShrink` + `CheckFork` + `ViolationKind`
    strings; `transparency-dev/merkle` is prose-only (single grep hit is a comment on line 21, not in
    go.mod, not imported — confirmed).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite v1.46.1` with ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`, typed CRUD.
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin}.go` — transport-only
    `FetchCheckpoint`, pure 4-way `AcceptCheckpoint` (clock-injected), pure `VerifyCheckpoint`,
    networked `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`/`Origin()`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    per-package `testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 19 (`logclient`) + 22 (`store`) + 7 (`follower`) + 3 (`registry`)
    + 4 (`config`) + 1 (`cmd/iscc-monitor`) = **65 `func Test`** (up from 62; store +3 for coverage).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — the third of three triggers (RFC-6962 consistency-proof failure across
    *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and will
    trip the conformance/oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - structured logs (replacing the two stderr placeholders: `alert` + the documented per-tick swallowed
    error), `/metrics`.
  - **Real alert transport** — `alert`/`AlertFunc` is a stderr-only placeholder; production delivery
    (email/webhook) is unbuilt.
- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (confirmed:
  find for `*tile*`/`*entries*` returns nothing; needed for the equivocation trigger + M2). Known
  stale-fixture drift (not yet acted on): `derive_vkey.py` HUBS + both `sb1.amlet.id_did.json` fixtures
  carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer is now `069d0f14`; refresh lands with
  the `hub_keys` cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero import statements + zero
  go.mod entries): `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink
  AND fork each → correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected +
  evidence-survives-restart, exercised through the poll loop). Coverage (`monitored_since`) — **now
  tracked** and asserted set-once through `PollHub`. The **third trigger, equivocation, is not met**
  (absent), so the M1 Verify line is not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "coverage tracking — monitored_since")
  records the gate green at HEAD `23f26bc`: `go build`/`go vet`/`go test ./...` all `ok` (7 packages),
  `gofmt -l .` empty, `go.mod`/`go.sum` byte-identical (confirmed here: no dep added since `d5db9f3`),
  no gate-dodging (no `//nolint`/`t.Skip`/build-tag/swallowed-error in the diff). Oracle/conformance
  gate correctly N/A this step (plain `hubs`-column CRUD + a set-once write on the already-verified
  path; no proof/verify/merkle/fsck path touched). The merkle-backed equivocation slice that follows
  *will* trip the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `23f26bc`. **No `.github/workflows/` — no CI configured.** When CI is wired it must avoid
  `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future `notecheck`
  oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. The cleanest high-value remaining slice is the **merkle-backed equivocation trigger**
(RFC-6962 consistency-proof failure across growing sizes): a third branch in the same `checkConsistency`
seam returning `ViolationEquivocation` + real `ProofJSON`. It needs `transparency-dev/merkle` (new dep)
+ tile fixtures and **trips the conformance/oracle gate** (`fsck` root-rebuild over a `SQLiteFetcher`,
inclusion cross-check vs `IsccLogInclusionProof`, golden-vector parity, `notecheck` parity in CI), and
must preserve "compare against the prior *accepted* root, not the contradicting evidence" (`CheckpointAt`
`LIMIT 1` rowid-order learning). The lighter remaining M1 gaps — `hub_keys` did:web cache write (which
MUST also refresh the stale `sb1.amlet.id_did.json` + `derive_vkey.py` HUBS to signer `069d0f14`),
structured logs, `/metrics`, and real alert transport — then complete M1's Verify criteria. No CI is
configured — flag for whoever sets up the workflow (it becomes load-bearing once the merkle slice lands).
