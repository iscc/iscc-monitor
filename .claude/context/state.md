<!-- assessed-at: 12cd9bd2d5feca5968bf78d5107f8799782ca78e -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — pure realm-registry parser landed; binary + equivocation trigger still missing

The M1 crypto/verification core (did:web → vkey/origin → 4-way accept → shrink/fork freeze) and the
single-writer poll-loop cadence are complete. Since the last assessment a pure domains-only realm
registry parser (`internal/registry`) landed — the last building block before a `cmd/` binary can
wire the loop. What still blocks M1: the merkle-backed equivocation trigger, the `hub_keys` did:web
cache write, config + the registry→target wiring, coverage (`monitored_since`), structured logs,
`/metrics`, real alert transport, and a `cmd/` binary entrypoint.

## M1 — Read-only Monitor
**Status**: partially met (verify primitives + shrink/fork freeze + poll-loop cadence + realm-registry
parser done; equivocation trigger, the config/coverage/logs/metrics/binary tissue, and real alert
transport still missing)
- Verified present (incremental re-check; the only Go change since the prior assessment at `623d866` is
  the new `internal/registry` package — all other sections re-confirmed on disk unchanged):
  - **NEW — realm-registry parser** (`internal/registry/registry.go`): pure, dep-free
    `Parse([]byte) ([]Entry, error)` turning a line-based domains-only document into ordered
    `Entry{Domain, BaseURL="https://"+Domain}`. Drops blank/`#`-comment lines, trims, preserves input
    order (no sort/dedupe — reconciliation deferred to wiring), and fails closed (wrapped, line-naming
    error) on any URL-shaped line (`://` scheme or `/` path). No key field (ADR-0009: keys come from
    did:web). Imports exactly `{bufio bytes fmt strings}` — leaf, no `net`/`net/http` (the `os` in the
    dep closure is the known `fmt`-transitive case, not a leak). Tested by `TestParse`/`TestParseGolden`
    + table cases (single/whitespace/`host:port`/no-trailing-newline/all-comment/empty); golden asserts
    exactly the two real testnet hubs in input order. **3 `func Test`.**
  - **Poll-loop cadence wrapper** (`internal/follower/loop.go`): `Loop` drives `PollHub` over a fixed
    `[]HubTarget` from one goroutine (single writer per DB, ADR-0005/0007). Pure `due(...)` predicate
    (`>=` boundary, zero `lastPoll` → always due) is the only branching; `Tick(ctx, now)` reads the
    freeze flag fresh each pass, polls every due hub, records `lastPoll` only on nil-error poll, returns
    the first error without aborting the pass. `time.Now()` never appears in the file. Frozen hubs re-poll
    only at the longer `Frozen` interval. Tested by `TestDue`/`TestTick`/`TestTickFrozenUnaffected`.
  - **Freeze + alert-once wiring** (`internal/follower/follower.go`): on `StatusVerified`, `PollHub` reads
    `FollowState`, runs `checkConsistency` (shrink-then-fork against prior accepted checkpoint) BEFORE any
    record/advance; on a true verdict `freeze(...)` does `RecordViolation` + `RecordCheckpoint` (evidence,
    no advance) + `Freeze`, then fires the injected `AlertFunc` once iff `!wasFrozen`. A violation returns
    `(StatusVerified, nil)` — freezes, never crashes (ADR-0006), no auto-unfreeze. Tested by
    `TestPollHubFork`/`TestPollHubShrink`.
  - **`store.CheckpointAt`** (`internal/store/checkpoints.go`): recovers prior accepted root (`LIMIT 1`,
    no `ORDER BY` → rowid order = prior/seed row). Tested by `TestCheckpointAt` + restart-survival test.
  - `internal/logclient/consistency.go` — dep-free `CheckShrink` + `CheckFork` + `ViolationKind` strings;
    `transparency-dev/merkle` is prose-only (confirmed: not in go.mod, not imported).
  - `internal/store/{sqlite,schema,checkpoints}.go` — `modernc.org/sqlite` with ADR-0005/0007 single-writer
    discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`, `SetMaxOpenConns(1)`),
    embedded nine-table `schema.sql`, typed CRUD. `store` stays a leaf (no `logclient`/`net/http`).
  - `internal/logclient/{checkpoint,accept,verify,didresolve,origin}.go` — transport-only `FetchCheckpoint`,
    pure 4-way `AcceptCheckpoint` (clock-injected), pure `VerifyCheckpoint`, networked `ResolveVerifierKey`
    over the 1-method `Fetcher` seam, golden `origin()`.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/
    `DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    `internal/{didweb,logclient}/testdata/{sb0,sb1}_did.json` + `internal/registry/testdata/realm.txt`.
  - Test totals: 9 (`didweb`) + 18 (`logclient`) + 19 (`store`) + 7 (`follower`) + 3 (`registry`)
    = **56 `func Test`** (up from 53).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — the third of three triggers (RFC-6962 consistency-proof failure across
    *growing* sizes); needs `transparency-dev/merkle` + tile fixtures (**both still absent**) and will
    trip the conformance/oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
  - **`cmd/` binary entrypoint** — still absent (verified: `cmd/` does not exist). With `registry.Parse`
    and `Loop.Run` both now pure libraries, the binary slice is unblocked: read realm doc → `registry.Parse`
    → `store.UpsertHub` per `Entry` → build `[]follower.HubTarget` → wire DI + `Loop.Run`.
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - config + coverage (`monitored_since`), structured logs, `/metrics`.
  - **Real alert transport** — `AlertFunc` is a minimal func seam; production delivery is unbuilt.
- Fixtures: `testdata/live/` holds **only the two checkpoints — no tiles or entry bundles** (needed for
  the equivocation trigger + M2). Known stale-fixture drift (not yet acted on): `derive_vkey.py` HUBS +
  both `sb1.amlet.id_did.json` fixtures carry sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer is
  now `069d0f14`; refresh lands with the `hub_keys` cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero import statements + zero
  go.mod entries): `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey` byte-match
  for both hubs — **met**. **Two of three triggers fully met end-to-end** (synthetic shrink AND fork each →
  correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected + evidence-survives-
  restart, exercised through the poll loop). The **third trigger, equivocation, is not met** (absent), so
  the M1 Verify line is not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-20, "Pure realm-registry parser") records
  the gate green at HEAD `12cd9bd`: `go build/vet/test ./...` all `ok`, `gofmt -l .` empty,
  `go.mod`/`go.sum` byte-identical (no dep added), `internal/registry` imports = `{bufio bytes fmt
  strings}` with no `net`/`net/http`, no gate-dodging. Conformance/oracle gate correctly N/A this step
  (pure string parser, no proof/verify/merkle/signature path touched). The merkle-backed equivocation
  slice that follows *will* trip the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `12cd9bd`. **No `.github/workflows/` and `gh run list` returns `[]` — no CI configured.** When CI is
  wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out the
  future `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. The realm-registry parser unblocks the **`cmd/iscc-monitor` binary + `internal/config`** —
the cleanest no-crypto slice that makes the monitor actually run: read the realm doc from disk →
`registry.Parse` → `store.UpsertHub(domain, baseURL, …)` per `Entry` → build `[]follower.HubTarget` →
wire DI + `Loop.Run`. The other high-value slice is the **merkle-backed equivocation trigger** (RFC-6962
consistency-proof failure across growing sizes): plug a third branch into the same `checkConsistency`
seam returning `ViolationEquivocation` + real `ProofJSON`; it needs `transparency-dev/merkle` (a new dep)
+ tile fixtures and **trips the conformance/oracle gate** (`fsck` root-rebuild over a `SQLiteFetcher`,
inclusion cross-check vs the hub's `IsccLogInclusionProof`, golden-vector parity), so it is the heavier
step. It MUST preserve "compare against the prior *accepted* root, not the contradicting evidence" (the
`CheckpointAt` `LIMIT 1` rowid-order learning). The **`hub_keys` did:web cache write** (which MUST also
refresh the stale `sb1.amlet.id_did.json` + `derive_vkey.py` HUBS to signer `069d0f14`), coverage
(`monitored_since`), structured logs, `/metrics`, and real alert transport then complete M1's Verify
criteria. No CI is configured — flag for whoever sets up the workflow.
