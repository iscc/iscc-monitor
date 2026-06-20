<!-- assessed-at: ecf904a2e9ce3437cd3dd615724c18978e85dde0 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — pure verification primitives, the per-network SQLite store, and now the transport-only checkpoint fetch are complete and tested; still no follower or binary wires them end-to-end

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; deps:
`golang.org/x/mod v0.33.0` for `sumdb/note`, `modernc.org/sqlite v1.46.1` pure-Go store). M1's *pure
verification primitives* (did:web trust-root chain, networked `ResolveVerifierKey`, signed-note
`VerifyCheckpoint`, `DIDKey.ValidAt`, composed four-way `AcceptCheckpoint`), the *stateful foundation*
(per-network SQLite with single-writer discipline + nine-table schema + four typed CRUD methods), and
now the *networked fetch* (`FetchCheckpoint`) are all in place and tested. What remains of M1 is the
*connective tissue*: the per-hub follower poll loop that actually wires `FetchCheckpoint →
AcceptCheckpoint →` the store methods, the three-trigger RFC-6962 consistency check, freeze + alert,
coverage, structured logs, `/metrics`, config + realm registry, and a binary (`cmd/` is still absent).
Nothing yet *calls* `FetchCheckpoint`, `AcceptCheckpoint`, or the store CRUD methods. Last `review`
verdict (HEAD `ecf904a`) is **PASS_WITH_NOTES / CONTINUE** with the gate recorded green; branch
`develop`; `issues.md` empty.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; since `8b4efe3` the only Go change is the additive
  `internal/logclient/checkpoint.go` + `checkpoint_test.go` — all other sections carried forward
  unchanged and reconfirmed on disk):
  - `internal/logclient/checkpoint.go` (NEW since last assessment) — `FetchCheckpoint(ctx, Fetcher,
    baseURL) ([]byte, error)`: transport-only, imports only `context`+`fmt`. Reuses `origin()` to
    derive `https://<domain>/log/checkpoint` by concatenation (NOT a second `net/url` parse), fetches
    through the injected `Fetcher`, returns the body verbatim. Wraps both `origin()` and Fetcher errors
    with `%w` so a 404's `errors.Is(err, os.ErrNotExist)` survives; does NOT wrap in `ErrUnresolvable`
    (that sentinel is did:web-only). Tested by `TestFetchCheckpoint` (5 subtests: offline URL golden,
    bare-host input, 404 contract, empty-base error) + `TestFetchCheckpointOverHTTP` (real
    `httptest.NewTLSServer` round-trip → fetched bytes then verify to `StatusVerified`, `TreeSize ==
    10183`).
  - `internal/store/checkpoints.go` — four typed methods on `*Store` (`UpsertHub` /
    `RecordCheckpoint` / `FollowState` / `AdvanceFollowState`), plus `CheckpointRecord` / `FollowState`
    structs and the `unixOrNil` helper. `RecordCheckpoint` uses `ON CONFLICT(hub_id,tree_size,root) DO
    NOTHING` + `RowsAffected()` to tell first-sighting from re-observed; `AdvanceFollowState` omits
    `frozen` from the `DO UPDATE SET` (no auto-unfreeze, ADR-0006); zero `ObservedAt` → SQL NULL.
    `store` stays a **leaf** (`go list -deps` shows no internal deps; stdlib + `database/sql` only).
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
  - Test totals: 9 (`didweb`) + 14 (`logclient`) + 11 (`store`) = **34 `func Test`**.
- Missing (the connective majority of M1 — nothing yet *consumes* `FetchCheckpoint`/`AcceptCheckpoint`
  or *calls* the store CRUD methods):
  - **Per-hub follower poll loop** (the real caller): `FetchCheckpoint` → `AcceptCheckpoint` (check
    `err` before status), map `Status.String()` + `CheckpointInfo` into a `CheckpointRecord`,
    `RecordCheckpoint`, `AdvanceFollowState` **only** on `StatusVerified`, populate the `hub_keys`
    did:web cache; run the three-trigger RFC-6962 consistency check (fork/shrink/equivocation via
    `transparency-dev/merkle`), persist raw checkpoints + proof into `violations`, set `frozen=1`,
    alert once, no auto-unfreeze, other hubs unaffected. The single-writer goroutine wrapper is the
    follower's concern.
  - config + realm registry (domains only), coverage (`monitored_since`), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet**.
- Fixtures: `testdata/live/` still holds **only the two checkpoints — no tiles or entry bundles**
  (needed for the consistency check + M2). Known stale-fixture drift (not yet acted on): `derive_vkey.py`
  `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation key
  (`22b08f3e`); refresh to `069d0f14` lands with the follower/`hub_keys` step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`verify.go`), `modernc.org/sqlite`
  (`store/sqlite.go`). **Not yet wired**: `transparency-dev/*` (merkle/tessera/formats),
  `nbd-wtf/opentimestamps` (grep confirms zero references in `internal/` or `go.mod`).
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (real checkpoints verify end-to-end; `AcceptCheckpoint` returns
  `StatusVerified`, `TreeSize == 10183` for sb0; the new HTTP round-trip test exercises this over real
  transport). Restart survival demonstrated at the store layer. The synthetic
  fork/shrink/equivocation → `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected
  half of M1 is **not started** (no follower).

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
  handoff records the gate green at HEAD `ecf904a` (build + vet + test exit 0 on go1.24; `gofmt -l .`
  empty; `derive_vkey.py` both golden vectors print exactly; no `//nolint`/`t.Skip`/build-tag dodges in
  the diff). Conformance/oracle gate correctly N/A this step — the `FetchCheckpoint` diff is
  transport-only, touching no signature/RFC-6962/proof code.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`. **No `.github/workflows/`
  and no CI runs — no CI configured.** When CI is wired it must avoid `go build ./...` over the
  gitignored `cauldron/` reference trees and shell out the future `notecheck` oracle rather than
  `go run` from `cauldron/`, and re-assert WAL/FK pragmas on any read connections if a read-pool split
  lands at serving — see learnings.

## Next Milestone
Continue M1. Immediate next unit (per the PASS_WITH_NOTES handoff): the **stateful follower poll
loop** — the real caller of `FetchCheckpoint`, `AcceptCheckpoint`, and the four store methods. It calls
`FetchCheckpoint` then `AcceptCheckpoint` (check the returned `err` before the status — a
verified-but-garbled body is a fault, not a four-way verdict), maps `CheckpointInfo{Origin,TreeSize,Root}`
into a `CheckpointRecord` (`ObservedAt` injected, never `time.Now()` in the pure layer), persists via
`RecordCheckpoint`, calls `AdvanceFollowState` **only** on `StatusVerified`, and writes the `hub_keys`
did:web cache — that step also refreshes the stale sb1 `did.json` fixture + `derive_vkey.py` `HUBS` to
the current key (`069d0f14`), re-applying the parity oracle gate. The three-trigger RFC-6962
consistency check (needs tiles in `testdata/live/` + `transparency-dev/merkle`) is the step after; then
freeze/alert + coverage + structured logs + `/metrics` complete M1's Verify criteria. No CI is
configured — flag for whoever sets up the GitHub workflow.
