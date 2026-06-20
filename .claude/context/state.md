<!-- assessed-at: fbdde9bf234cfa47764a790181224ff34391a896 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verify core + cadence + parser + config leaves landed; binary + equivocation trigger still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze) and the
single-writer poll-loop cadence are complete, and the two pure no-crypto leaves that feed a future
binary — the realm-registry parser (`internal/registry`) and the typed config loader
(`internal/config`) — have both landed. What still blocks M1: the merkle-backed equivocation trigger,
the `cmd/` binary that wires config → parser → store → loop, the `hub_keys` did:web cache write,
coverage (`monitored_since`), structured logs, `/metrics`, and real alert transport.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser + config loader done; equivocation trigger, the `cmd/` binary, and the coverage/logs/metrics/
alert tissue still missing)
- Verified present (incremental re-check; the only Go change since the prior assessment at `12cd9bd` is
  the new `internal/config` package — all other sections re-confirmed on disk unchanged):
  - **NEW — config loader leaf** (`internal/config/config.go`): pure, dep-free
    `Load(get func(key string) (string, bool)) (Config, error)` turning a flat injected key/value lookup
    into a validated, typed `Config{DBPath, RealmPath string; Normal, Frozen time.Duration}`. Required
    `ISCC_MONITOR_{DB,REALM}` (absent/empty → wrapped, key-named error; whitespace not trimmed —
    binary owns I/O); optional `ISCC_MONITOR_{NORMAL,FROZEN}` default 5m / 1h, parsed with
    `time.ParseDuration`, each must be positive; load-bearing cross-check `Frozen >= Normal` (ADR-0006,
    matches `loop.go` `due()` back-off). Fails closed (zero `Config` alongside every error). Imports
    exactly `{fmt time}` — leaf, no `net`/`net/http`/`os`. **4 `func Test`** (`TestLoad`,
    `TestLoadGolden`, `TestLoadDefaults`, `TestLoadErrors`); `review` recorded 100% statement coverage.
  - **Realm-registry parser** (`internal/registry/registry.go`): pure `Parse([]byte) ([]Entry, error)`
    turning a domains-only line document into ordered `Entry{Domain, BaseURL="https://"+Domain}`. Drops
    blank/`#`-comment lines, preserves input order, fails closed on any URL-shaped line. No key field
    (ADR-0009). Imports `{bufio bytes fmt strings}`. **3 `func Test`.**
  - **Poll-loop cadence wrapper** (`internal/follower/loop.go`): `Loop` drives `PollHub` over a fixed
    `[]HubTarget` from one goroutine (single writer per DB, ADR-0005/0007). Pure `due(...)` predicate
    (`>=` boundary, zero `lastPoll` → always due); frozen hubs re-poll only at the longer `Frozen`
    interval; `time.Now()` never in the file. Tested by `TestDue`/`TestTick`/`TestTickFrozenUnaffected`.
  - **Freeze + alert-once wiring** (`internal/follower/follower.go`): on `StatusVerified`, `PollHub`
    runs `checkConsistency` (shrink-then-fork vs prior accepted checkpoint) BEFORE any record/advance;
    on a true verdict `freeze(...)` does `RecordViolation` + `RecordCheckpoint` (evidence, no advance) +
    `Freeze`, then fires the injected `AlertFunc` once iff `!wasFrozen`. No crash, no auto-unfreeze
    (ADR-0006). Tested by `TestPollHubFork`/`TestPollHubShrink`.
  - **`store.CheckpointAt`** (`internal/store/checkpoints.go`): recovers prior accepted root via
    `LIMIT 1` (rowid order = prior/seed row). Tested by `TestCheckpointAt` + restart-survival test.
  - `internal/logclient/consistency.go` — dep-free `CheckShrink` + `CheckFork` + `ViolationKind`
    strings; `transparency-dev/merkle` is prose-only (confirmed: not in go.mod, not imported).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite` with ADR-0005/0007
    single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
    `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`, typed CRUD. Leaf (no `logclient`/net).
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin}.go` — transport-only
    `FetchCheckpoint`, pure 4-way `AcceptCheckpoint` (clock-injected), pure `VerifyCheckpoint`,
    networked `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    `internal/{didweb,logclient}/testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 18 (`logclient`) + 19 (`store`) + 7 (`follower`) + 3 (`registry`)
    + 4 (`config`) = **60 `func Test`** (up from 56).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — the third of three triggers (RFC-6962 consistency-proof failure across
    *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and will
    trip the conformance/oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
  - **`cmd/` binary entrypoint** — still absent (verified: `cmd/` does not exist). Now fully unblocked:
    config's only consumer. Slice: `os.LookupEnv`-backed `get` → `config.Load` →
    `os.ReadFile(cfg.RealmPath)` → `registry.Parse` → `store.Open(cfg.DBPath)` → per `Entry`
    `store.UpsertHub(domain, origin, baseURL)` → `[]follower.HubTarget` → `Loop{Normal,Frozen,…}.Run`.
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - coverage (`monitored_since`), structured logs, `/metrics`.
  - **Real alert transport** — `AlertFunc` is a minimal func seam; production delivery is unbuilt.
- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed for
  the equivocation trigger + M2). Known stale-fixture drift (not yet acted on): `derive_vkey.py` HUBS +
  both `sb1.amlet.id_did.json` fixtures carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer
  is now `069d0f14`; refresh lands with the `hub_keys` cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero import statements + zero
  go.mod entries): `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink
  AND fork each → correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected +
  evidence-survives-restart, exercised through the poll loop). The **third trigger, equivocation, is
  not met** (absent), so the M1 Verify line is not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "internal/config pure config loader
  leaf") records the gate green at HEAD `fbdde9b`: `go build`/`go vet`/`go test ./...` all `ok`,
  `gofmt -l .` empty, `go.mod`/`go.sum` byte-identical (confirmed here: no dep added since `12cd9bd`),
  `internal/config` imports = `{fmt time}` with no `net`/`net/http`, 100% statement coverage, no
  gate-dodging (confirmed here: no `//nolint`/`t.Skip`/build-tag in `internal/config`). Conformance/
  oracle gate correctly N/A this step (pure no-crypto value parser). The merkle-backed equivocation
  slice that follows *will* trip the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `fbdde9b`. **No `.github/workflows/` — no CI configured.** When CI is wired it must avoid
  `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future `notecheck`
  oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. With both pure leaves (`registry.Parse`, `config.Load`) landed, the **`cmd/iscc-monitor`
binary** is the cleanest now-fully-unblocked slice and config's only consumer: build the
`os.LookupEnv`-backed `get` closure → `config.Load` → `os.ReadFile(cfg.RealmPath)` → `registry.Parse`
→ `store.Open(cfg.DBPath)` → per `Entry` `store.UpsertHub(domain, origin, baseURL)` →
`[]follower.HubTarget` → `Loop{Normal: cfg.Normal, Frozen: cfg.Frozen, …}.Run(ctx)`. It must resolve
the still-open **origin-export decision** (`UpsertHub` needs `<domain>/log` and `logclient.origin` is
private — export `logclient.Origin` OR carry origin on `registry.Entry`, never a second deriver). The
heavier high-value alternative is the **merkle-backed equivocation trigger** (RFC-6962 consistency-proof
failure across growing sizes): a third branch in the same `checkConsistency` seam returning
`ViolationEquivocation` + real `ProofJSON`; it needs `transparency-dev/merkle` (new dep) + tile fixtures
and **trips the conformance/oracle gate** (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion
cross-check vs `IsccLogInclusionProof`, golden-vector parity), and must preserve "compare against the
prior *accepted* root, not the contradicting evidence" (`CheckpointAt` `LIMIT 1` rowid-order learning).
The `hub_keys` did:web cache write (which MUST also refresh the stale `sb1.amlet.id_did.json` +
`derive_vkey.py` HUBS to signer `069d0f14`), coverage (`monitored_since`), structured logs, `/metrics`,
and real alert transport then complete M1's Verify criteria. No CI is configured — flag for whoever sets
up the workflow.
