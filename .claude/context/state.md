<!-- assessed-at: d5db9f3ef0a3f775b8f066c0d88474455805917d -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verify core + cadence + parser + config + `cmd/` binary landed; equivocation trigger + coverage/logs/metrics still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze), the
single-writer poll-loop cadence, both pure no-crypto leaves (realm-registry parser + typed config
loader), and now the `cmd/iscc-monitor` binary that wires them together (config → realm parse →
store → poll loop) have all landed. What still blocks M1: the merkle-backed equivocation trigger,
the `hub_keys` did:web cache write, coverage (`monitored_since`), structured logs, `/metrics`, and
real alert transport (currently a stderr placeholder).

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader + wired `cmd/` binary done; equivocation trigger and the coverage/logs/metrics/
alert tissue still missing)
- Verified present (incremental re-check; the only Go changes since the prior assessment at `fbdde9b`
  are the new `cmd/iscc-monitor` package and the exported `logclient.Origin` wrapper — all other
  sections re-confirmed unchanged on disk):
  - **NEW — `cmd/iscc-monitor` binary** (`cmd/iscc-monitor/main.go`): the wired M1 entrypoint.
    `run()` does `config.Load(os.LookupEnv)` → `os.ReadFile(cfg.RealmPath)` → `registry.Parse` →
    `store.Open(cfg.DBPath)` → `registerHubs` → `follower.Loop{Store,Fetcher:NewHTTPFetcher(nil),
    Targets,Normal,Frozen,Alert}.Run(ctx)` under a `signal.NotifyContext(os.Interrupt)` so a clean
    SIGINT returns `ctx.Err()` (reported as nil). `main` owns the single `os.Exit(1)` + stderr print.
    The registry→store wiring is extracted into the testable `registerHubs` helper, which derives the
    origin via the exported `logclient.Origin` (`<domain>/log`, never bare) and `UpsertHub`s each
    entry (idempotent on domain). `alert(hubID, kind)` is a documented stderr placeholder (real
    transport is a later step). **1 `func Test`** (`TestRegisterHubs`: 2 targets, `HubID>0`,
    `BaseURL=="https://"+domain`, idempotent re-run returns identical `HubID`s). `Loop.Run` itself
    remains correctly untested (blocking ticker `select`); the network path is exercised only by the
    review's manual smoke run, not a committed test.
  - **NEW — exported `logclient.Origin`** (`internal/logclient/origin.go`): a one-line wrapper that
    delegates to the private `origin` so the binary can feed `UpsertHub` its origin argument — no
    second deriver, so the golden `TestOrigin` vectors cover it too.
  - **config loader leaf** (`internal/config/config.go`): pure, dep-free
    `Load(get func(key string) (string, bool)) (Config, error)` → validated, typed
    `Config{DBPath, RealmPath string; Normal, Frozen time.Duration}`. Required `ISCC_MONITOR_{DB,REALM}`;
    optional `ISCC_MONITOR_{NORMAL,FROZEN}` default 5m / 1h; load-bearing `Frozen >= Normal` cross-check
    (ADR-0006). Fails closed. Imports exactly `{fmt time}`. **4 `func Test`.**
  - **Realm-registry parser** (`internal/registry/registry.go`): pure `Parse([]byte) ([]Entry, error)`
    → ordered `Entry{Domain, BaseURL="https://"+Domain}`, drops blank/`#` lines, fails closed on any
    URL-shaped line, no key field (ADR-0009). Imports `{bufio bytes fmt strings}`. **3 `func Test`.**
  - **Poll-loop cadence wrapper** (`internal/follower/loop.go`): `Loop` drives `PollHub` over a fixed
    `[]HubTarget` from one goroutine (single writer per DB, ADR-0005/0007). Pure `due(...)` predicate;
    frozen hubs re-poll only at the longer `Frozen` interval; `lastPoll` lazily inited in `Tick`
    (panic-safe for the binary building `&Loop{…}`). Tested by `TestDue`/`TestTick`/
    `TestTickFrozenUnaffected`.
  - **Freeze + alert-once wiring** (`internal/follower/follower.go`): on `StatusVerified`, `PollHub`
    runs `checkConsistency` (shrink-then-fork vs prior accepted checkpoint) BEFORE any record/advance;
    on a true verdict `freeze(...)` does `RecordViolation` + `RecordCheckpoint` (evidence, no advance) +
    `Freeze`, then fires the injected `AlertFunc` once iff `!wasFrozen`. Tested by
    `TestPollHubFork`/`TestPollHubShrink`.
  - **`store.CheckpointAt`** (`internal/store/checkpoints.go`): recovers prior accepted root via
    `LIMIT 1` (rowid order). Tested by `TestCheckpointAt` + restart-survival test.
  - `internal/logclient/consistency.go` — dep-free `CheckShrink` + `CheckFork` + `ViolationKind`
    strings; `transparency-dev/merkle` is prose-only (confirmed: the single grep hit is a comment on
    line 21, not in go.mod, not imported).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite` with ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`, typed CRUD.
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin}.go` — transport-only
    `FetchCheckpoint`, pure 4-way `AcceptCheckpoint` (clock-injected), pure `VerifyCheckpoint`,
    networked `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`/`Origin()`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    `internal/{didweb,logclient}/testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 19 (`logclient`) + 19 (`store`) + 7 (`follower`) + 3 (`registry`)
    + 4 (`config`) + 1 (`cmd/iscc-monitor`) = **62 `func Test`** (up from 60).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — the third of three triggers (RFC-6962 consistency-proof failure across
    *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and will
    trip the conformance/oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - coverage (`monitored_since`), structured logs (replacing the two stderr placeholders: `alert` +
    the documented per-tick swallowed error), `/metrics`.
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
  byte-match for both hubs — **met** (the binary's offline smoke run persisted `sb0.iscc.id/log` /
  `sb1.amlet.id/log`, confirming origin is `<domain>/log` never bare). **Two of three triggers fully
  met end-to-end** (synthetic shrink AND fork each → correct `violations.kind` + `frozen=1` +
  exactly-one-alert + other-hubs-unaffected + evidence-survives-restart, exercised through the poll
  loop). The **third trigger, equivocation, is not met** (absent), so the M1 Verify line is not fully
  satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "cmd/iscc-monitor entrypoint")
  records the gate green at HEAD `d5db9f3`: `go build`/`go vet`/`go test ./...` all `ok` (7 packages),
  `gofmt -l .` empty, `go.mod`/`go.sum` byte-identical (confirmed here: no dep added since `fbdde9b`),
  no gate-dodging (no `//nolint`/`t.Skip`/build-tag/swallowed-error in the diff; the stderr `alert` +
  per-tick swallowed error are documented placeholders explicitly in `next.md`'s Not-In-Scope).
  Oracle/conformance gate correctly N/A this step (`Origin` only re-exports the already-golden
  `origin`; no proof/verify/merkle/fsck path touched). The merkle-backed equivocation slice that
  follows *will* trip the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `d5db9f3`. **No `.github/workflows/` — no CI configured.** When CI is wired it must avoid
  `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future `notecheck`
  oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. With the `cmd/iscc-monitor` binary landed, the cleanest high-value slice is the
**merkle-backed equivocation trigger** (RFC-6962 consistency-proof failure across growing sizes): a
third branch in the same `checkConsistency` seam returning `ViolationEquivocation` + real `ProofJSON`.
It needs `transparency-dev/merkle` (new dep) + tile fixtures and **trips the conformance/oracle gate**
(`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check vs `IsccLogInclusionProof`,
golden-vector parity, `notecheck` parity in CI), and must preserve "compare against the prior
*accepted* root, not the contradicting evidence" (`CheckpointAt` `LIMIT 1` rowid-order learning). The
lighter remaining M1 gaps this wiring surfaced — `hub_keys` did:web cache write (which MUST also refresh
the stale `sb1.amlet.id_did.json` + `derive_vkey.py` HUBS to signer `069d0f14`), coverage
(`monitored_since`), structured logs, `/metrics`, and real alert transport — then complete M1's Verify
criteria. No CI is configured — flag for whoever sets up the workflow.
