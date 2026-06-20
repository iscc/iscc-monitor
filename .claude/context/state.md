<!-- assessed-at: 055a70e0a25fbd07bf057c6b1756451bb49f92e6 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verification primitives, the per-network SQLite store, the transport-only checkpoint fetch, and now the first real caller (single-observation `PollHub`) are complete and tested; still no poll loop, no consistency check / freeze, and no binary

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; deps:
`golang.org/x/mod v0.33.0` for `sumdb/note`, `modernc.org/sqlite v1.46.1` pure-Go store). M1's *pure
verification primitives* (did:web trust-root chain, networked `ResolveVerifierKey`, signed-note
`VerifyCheckpoint`, `DIDKey.ValidAt`, composed four-way `AcceptCheckpoint`), the *stateful foundation*
(per-network SQLite with single-writer discipline + nine-table schema + four typed CRUD methods), the
*networked fetch* (`FetchCheckpoint`), and now the *first real caller* (`internal/follower.PollHub`,
one observation: fetch → accept → record → advance for a single hub) are all in place and tested. What
remains of M1 is the rest of the *connective tissue*: the per-hub poll loop / single-writer goroutine
wrapper that drives `PollHub` on a cadence, the three-trigger RFC-6962 consistency check, freeze +
alert, coverage, structured logs, `/metrics`, config + realm registry, and a binary (`cmd/` is still
absent). Last `review` verdict (HEAD `055a70e`) is **PASS / CONTINUE** with the gate recorded green;
branch `develop`; `issues.md` empty.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; since `ecf904a` the only Go change is the additive
  `internal/follower/follower.go` + `follower_test.go` — all other sections carried forward unchanged
  and reconfirmed on disk):
  - `internal/follower/follower.go` (NEW since last assessment) — `PollHub(ctx, *store.Store,
    logclient.Fetcher, hubID, baseURL, observedAt) (logclient.Status, error)`: the first real caller
    composing the M1 verify chain (`FetchCheckpoint → AcceptCheckpoint`) with the store CRUD
    (`RecordCheckpoint → AdvanceFollowState`) for one hub, one observation. Imports only
    `logclient` + `store` (+ `context`/`fmt`/`time`) — direction follower → {logclient, store}, store
    stays a leaf. Honors the err-before-status contract: a transport fault or a verified-but-garbled
    body returns the wrapped error and persists nothing. Persists + advances the cursor **only** on
    `StatusVerified` (non-verified verdicts carry a zero `CheckpointInfo` = no trustworthy
    `(size, root)`); maps `info.TreeSize`/`info.Root` into a `CheckpointRecord` with the injected
    `observedAt`. Tested by `TestPollHubVerifiedAdvances` (sb0 composite fetcher → `StatusVerified`,
    `FollowState.LastSize == 10183` — line 2 of the sb0 checkpoint fixture, non-vacuous) and
    `TestPollHubUnverifiedDoesNotAdvance` (mismatching key → `StatusUnverified`, `LastSize == 0`).
  - `internal/logclient/checkpoint.go` — `FetchCheckpoint(ctx, Fetcher, baseURL) ([]byte, error)`:
    transport-only (imports only `context`+`fmt`). Reuses `origin()` to derive
    `https://<domain>/checkpoint` by concatenation, fetches through the injected `Fetcher`, returns the
    body verbatim, `%w`-wraps both `origin()` and Fetcher errors (a 404's `errors.Is(err,
    os.ErrNotExist)` survives); does NOT wrap in `ErrUnresolvable` (did:web-only sentinel). Tested by
    `TestFetchCheckpoint` (5 subtests) + `TestFetchCheckpointOverHTTP` (real `httptest.NewTLSServer`
    round-trip → verify to `StatusVerified`, `TreeSize == 10183`).
  - `internal/store/checkpoints.go` — four typed methods on `*Store` (`UpsertHub` /
    `RecordCheckpoint` / `FollowState` / `AdvanceFollowState`), plus `CheckpointRecord` / `FollowState`
    structs and `unixOrNil`. `RecordCheckpoint` uses `ON CONFLICT(hub_id,tree_size,root) DO NOTHING` +
    `RowsAffected()` to tell first-sighting from re-observed; `AdvanceFollowState` omits `frozen` from
    `DO UPDATE SET` (no auto-unfreeze, ADR-0006); zero `ObservedAt` → SQL NULL. `store` stays a **leaf**
    (`go list -deps` shows no internal deps; stdlib + `database/sql` only).
  - `internal/store/sqlite.go` — `Open(path)`/`(*Store).Close()` over `modernc.org/sqlite`. Applies
    ADR-0005/0007 single-writer discipline (`journal_mode=WAL`, `busy_timeout=5000`, `foreign_keys=ON`,
    `synchronous=NORMAL`, `SetMaxOpenConns(1)`), embeds + idempotently applies `schema.sql`.
  - `internal/store/schema.sql` — nine core M1 tables (`hubs`, `hub_keys`, `checkpoints`,
    `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). No `network` column
    (ADR-0007), no `cosigs` (M7-deferred), timestamps INTEGER unix-seconds.
  - `internal/logclient/accept.go` — pure `AcceptCheckpoint` composing `ResolveVerifierKey →
    VerifyCheckpoint → DIDKey.ValidAt` into the 4-way `Status`; a verified-but-garbled body returns a
    **non-nil error** alongside `StatusUnverified`'s zero (callers check `err` first). `CheckpointInfo`
    is the zero value on every non-verified outcome. Pure, clock-injected.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt` half-open window, and
    `parseTime` that **fails closed** (non-empty-unparseable validity timestamp → `ErrUnresolvable`).
    WASM-pure (no net imports).
  - `internal/logclient/{verify,didresolve,origin}.go` — pure `VerifyCheckpoint` (oracle-exact
    `note.Open` reject), networked `ResolveVerifierKey` over the 1-method `Fetcher` seam (tested
    offline), and golden `origin()`.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed C2SP
    checkpoints. did:web golden fixtures under `internal/{didweb,logclient}/testdata/`.
  - Test totals: 9 (`didweb`) + 14 (`logclient`) + 11 (`store`) + 2 (`follower`) = **36 `func Test`**.
- Missing (still the connective majority of M1):
  - **Poll loop / single-writer goroutine wrapper** that calls `PollHub` on a cadence and owns all
    writes per network DB (ADR-0005/0007). `PollHub` does one observation per call; nothing yet loops it.
  - **`hub_keys` did:web cache write** — the step that persists resolved keys (and that MUST refresh the
    stale sb1 fixture, see below).
  - **Three-trigger RFC-6962 consistency check** (fork/shrink/equivocation via `transparency-dev/merkle`),
    persist raw checkpoints + proof into `violations`, set `frozen=1`, alert once, no auto-unfreeze,
    other hubs unaffected. Needs tiles in `testdata/live/`.
  - config + realm registry (domains only), coverage (`monitored_since`), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet**.
- Fixtures: `testdata/live/` still holds **only the two checkpoints — no tiles or entry bundles**
  (needed for the consistency check + M2). Known stale-fixture drift (not yet acted on): `derive_vkey.py`
  `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation key
  (`22b08f3e`); refresh to `069d0f14` lands with the `hub_keys` did:web cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`verify.go`), `modernc.org/sqlite`
  (`store/sqlite.go`). **Not yet wired**: `transparency-dev/*` (merkle/tessera/formats),
  `nbd-wtf/opentimestamps` (grep confirms zero references in `internal/` or `go.mod`).
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (real checkpoints verify end-to-end; `AcceptCheckpoint` returns
  `StatusVerified`, `TreeSize == 10183` for sb0; `PollHub` now flows that verdict through to a persisted
  cursor). Restart survival demonstrated at the store layer. The synthetic
  fork/shrink/equivocation → `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected
  half of M1 is **not started** (no consistency check, no freeze path).

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable (`go build ./... && go vet ./... && go test ./...`). Latest `review`
  handoff records the gate green at HEAD `055a70e` (build + vet + test exit 0 on go1.24; `gofmt -l .`
  empty; `TestPollHub*` PASS; store stays a leaf; no `//nolint`/`t.Skip`/build-tag dodges in the diff).
  Conformance/oracle gate correctly N/A this step — the follower diff touches no
  signature/RFC-6962/proof/didweb code.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`. **No `.github/workflows/`
  and no CI runs — no CI configured.** When CI is wired it must avoid `go build ./...` over the
  gitignored `cauldron/` reference trees and shell out the future `notecheck` oracle rather than
  `go run` from `cauldron/`, and re-assert WAL/FK pragmas on any read connections if a read-pool split
  lands at serving — see learnings.

## Next Milestone
Continue M1. Per the PASS/CONTINUE handoff, the natural follow-ons are independent ≤3-file slices:
(1) the **poll loop / single-writer goroutine wrapper** that drives `PollHub` on a cadence and owns all
writes per network DB (ADR-0005/0007); (2) the **`hub_keys` did:web cache write** — this step MUST also
refresh the stale `sb1.amlet.id_did.json` fixture + `derive_vkey.py` `HUBS` to the current sb1 signer
`069d0f14`, re-triggering the parity oracle gate; (3) the **three-trigger RFC-6962 consistency check**
(needs tiles in `testdata/live/` + `transparency-dev/merkle`) followed by freeze/alert. Then coverage +
structured logs + `/metrics` + a `cmd/` binary complete M1's Verify criteria. No CI is configured —
flag for whoever sets up the GitHub workflow.
