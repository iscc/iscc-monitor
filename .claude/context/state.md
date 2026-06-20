<!-- assessed-at: 8b4efe3f7e425ced8fe1f2c4373cff74c134465a -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — pure verification primitives + the per-network SQLite store (schema, open/close, and now typed CRUD for the checkpoint verdict) are complete and tested; still no follower or binary wires them together

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; deps:
`golang.org/x/mod v0.33.0` for `sumdb/note`, `modernc.org/sqlite v1.46.1` pure-Go store). M1's *pure
verification primitives* (did:web trust-root chain, networked `ResolveVerifierKey`, signed-note
`VerifyCheckpoint`, `DIDKey.ValidAt`, composed four-way `AcceptCheckpoint`) and the *stateful
foundation* (per-network SQLite with single-writer discipline + nine-table schema) were already in
place; new this iteration are the four **typed store CRUD methods** the follower will drive
(`UpsertHub` / `RecordCheckpoint` / `FollowState` / `AdvanceFollowState`). What remains of M1 is the
*connective tissue*: the per-hub follower poll loop that actually calls `AcceptCheckpoint` and these
store methods, the three-trigger RFC-6962 consistency check, freeze + alert, coverage, structured
logs, `/metrics`, config + realm registry, and a binary (`cmd/` is still absent). Nothing yet *calls*
the new store methods. Last `review` verdict (HEAD `8b4efe3`) is **PASS** with the gate recorded
green; branch `develop` in sync with `origin/develop` (0 ahead / 0 behind); `issues.md` empty.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; since `414b64c` only `internal/store/checkpoints.go` +
  `checkpoints_test.go` were added — all other sections carried forward unchanged and reconfirmed on
  disk):
  - `internal/store/checkpoints.go` (NEW since last assessment) — four typed methods on `*Store`,
    plus the store-owned `CheckpointRecord` / `FollowState` structs and the `unixOrNil` helper:
    - `UpsertHub` — select-then-insert keyed on `domain` (no UNIQUE on `domain`; sound under
      `SetMaxOpenConns(1)`); idempotent, re-register returns the existing id without rewriting
      `origin`/`base_url`.
    - `RecordCheckpoint` — `INSERT … ON CONFLICT(hub_id, tree_size, root) DO NOTHING` then branch on
      `RowsAffected()`: real insert → `inserted=true` via `LastInsertId`; conflict (`n==0`) → SELECT
      the existing id back, `inserted=false`, nil err. `consistent`/`root_rebuilt` left NULL for the
      consistency-check step; zero `ObservedAt` stored as NULL via `unixOrNil`.
    - `FollowState` — reads `last_size`/`frozen`/`last_error` through `sql.NullInt64`/`NullString`;
      unknown hub → zero `FollowState{}` + nil err ("never polled").
    - `AdvanceFollowState` — upsert that **omits `frozen` from the `DO UPDATE SET`**, so a frozen hub
      stays frozen across an advance (ADR-0006, no auto-unfreeze). Nothing in this layer ever sets or
      clears `frozen`.
    - `store` stays a **leaf**: `go list -deps ./internal/store` shows no internal iscc-monitor deps
      (only the package itself); imports are stdlib + `database/sql` only, no `logclient` coupling, so
      net/http never enters the closure. Status is carried on `CheckpointRecord` but intentionally not
      persisted (`checkpoints` has no status column).
    - Tested by 7 new tests (`TestUpsertHub`, `TestRecordCheckpoint`, `…ZeroObservedAtNull`,
      `TestFollowState`, `TestAdvanceFollowState`, `TestCheckpointHelpers`, restart-survival), bringing
      `internal/store` to 11 `func Test`.
  - `internal/store/sqlite.go` — `Open(path)`/`(*Store).Close()` over `modernc.org/sqlite`. Applies
    ADR-0005/0007 single-writer discipline in order (`journal_mode=WAL`, `busy_timeout=5000`,
    `foreign_keys=ON`, `synchronous=NORMAL`, `SetMaxOpenConns(1)`), embeds + idempotently applies
    `schema.sql` (`//go:embed`, `CREATE … IF NOT EXISTS`). Unchanged this iteration.
  - `internal/store/schema.sql` — nine core M1 tables (`hubs`, `hub_keys`, `checkpoints`,
    `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). No `network` column
    (ADR-0007), no `cosigs` (M7-deferred), timestamps INTEGER unix-seconds. Unchanged this iteration.
  - `internal/logclient/accept.go` — pure `AcceptCheckpoint` composing `ResolveVerifierKey →
    VerifyCheckpoint → DIDKey.ValidAt` into the 4-way `Status`; a verified-but-garbled body returns a
    **non-nil error** alongside `StatusUnverified`'s zero (callers check `err` first). `CheckpointInfo`
    is the zero value on every non-verified outcome. Pure, clock-injected.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt` half-open window, and
    `parseTime` that **fails closed** (non-empty-unparseable validity timestamp → `ErrUnresolvable`,
    never a false `verified`). 9 `func Test`; WASM-pure (no net imports).
  - `internal/logclient/{verify,didresolve,origin}.go` — pure `VerifyCheckpoint` (oracle-exact
    `note.Open` reject), networked `ResolveVerifierKey` over the 1-method `Fetcher` seam (tested
    offline), and golden `origin()`. 10 `func Test`.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed C2SP
    checkpoints. did:web golden fixtures under `internal/{didweb,logclient}/testdata/`.
  - Test totals: 9 (`didweb`) + 10 (`logclient`) + 11 (`store`) = **30 `func Test`**.
- Missing (the connective majority of M1 — nothing yet *consumes* `AcceptCheckpoint` or *calls* the
  store CRUD methods):
  - **Per-hub follower poll loop** (the real caller of the new methods): call `AcceptCheckpoint`
    (check `err` before status), map `Status.String()` + `CheckpointInfo` into a `CheckpointRecord`,
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
  `StatusVerified`, `TreeSize == 10183` for sb0). Restart survival demonstrated at the store layer
  (`TestStoreRestartSurvival` + the CRUD restart test). The synthetic fork/shrink/equivocation →
  `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected half of M1 is **not
  started** (no follower).

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
  handoff records the gate green at HEAD `8b4efe3` (build + vet + test exit 0 on go1.24; `gofmt -l .`
  empty; `go mod tidy` zero-diff; leaf purity, dedupe, no-auto-unfreeze, NULL-not-0, and restart
  survival each independently re-verified).
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop` (0/0). **No `.github/workflows/` and no CI runs — no CI configured.** When CI is
  wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out
  the future `notecheck` oracle rather than `go run` from `cauldron/`, and re-assert WAL/FK pragmas on
  any read connections if a read-pool split lands at serving — see learnings.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): the **stateful follower poll loop** — the
real caller of `AcceptCheckpoint` and the four new store methods. It checks the returned `err` before
the status (a verified-but-garbled body is a fault, not a four-way verdict), persists via
`RecordCheckpoint`, calls `AdvanceFollowState` **only** on `StatusVerified`, and writes the `hub_keys`
did:web cache — that step also refreshes the stale sb1 `did.json` fixture + `derive_vkey.py` `HUBS` to
the current key (`069d0f14`). The three-trigger RFC-6962 consistency check (needs tiles in
`testdata/live/` + `transparency-dev/merkle`) is the step after; then freeze/alert + coverage +
structured logs + `/metrics` complete M1's Verify criteria. No CI is configured — flag for whoever
sets up the GitHub workflow.
