<!-- assessed-at: 414b64cf77e32faf92ee60ae56845777758a718c -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — all four pure verification primitives complete and golden-tested, and the per-network SQLite store (schema + open/close) is now stood up; no follower/binary yet wires them together

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`; deps: `golang.org/x/mod
v0.33.0` for `sumdb/note`, `modernc.org/sqlite v1.46.1` for the pure-Go store). M1's *pure verification
primitives* are complete (did:web trust-root chain, networked `ResolveVerifierKey`, signed-note
`VerifyCheckpoint`, `DIDKey.ValidAt`, and the composed four-way `AcceptCheckpoint`), and — new this
iteration — the *stateful foundation* exists: `internal/store` opens one network's SQLite DB with the
single-writer discipline and applies the nine-table M1 schema idempotently. What remains of M1 is the
*connective tissue*: the per-hub follower poll loop that calls `AcceptCheckpoint` and persists verdicts,
the three-trigger RFC-6962 consistency check, freeze + alert, coverage, structured logs, `/metrics`,
config + realm registry, and a binary entrypoint (`cmd/` is still absent). Nothing yet reads or writes
store rows. Last `review` verdict (HEAD `414b64c`) is **PASS** with the gate recorded green; branch
`develop` in sync with `origin/develop` (0 ahead / 0 behind); `issues.md` is empty.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; since `8715370` only `internal/store/` was added —
  `sqlite.go` + `schema.sql` + `sqlite_test.go` — plus go.mod/go.sum dependency wiring; all other
  sections carried forward unchanged and reconfirmed on disk):
  - `internal/store/sqlite.go` (NEW since last assessment) — `Open(path)`/`(*Store).Close()` over the
    pure-Go `modernc.org/sqlite` driver. Applies the ADR-0005/0007 single-writer discipline in order:
    `PRAGMA journal_mode=WAL`, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`, plus
    `SetMaxOpenConns(1)` so a second writer can never open. Embeds `schema.sql` via `//go:embed` and
    applies it idempotently (`CREATE … IF NOT EXISTS`) on every `Open`. Error paths use the
    error-preserving `_ = db.Close()` idiom. No CRUD/fetcher/follower yet — schema + open/close only.
    Tested by `TestStoreOpenCreatesCoreTables`, `…EnablesWAL`, `TestStoreRestartSurvival`,
    `TestStoreReopenIdempotent`.
  - `internal/store/schema.sql` (NEW) — nine core M1 tables: `hubs`, `hub_keys`, `checkpoints`,
    `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`. No `network` column
    anywhere (ADR-0007 — file IS the namespace; grep confirms `network` appears only in comments). No
    `cosigs` table (M7-deferred). Timestamps stored as INTEGER unix-seconds.
  - `internal/logclient/accept.go` — pure `Status` enum (`StatusVerified/Unverified/Unresolvable/Rotated`
    + `String()`) and `AcceptCheckpoint(ctx, fetcher, baseURL, raw, observedAt)`. Composes the three
    primitives in load-bearing order: `ResolveVerifierKey` → `VerifyCheckpoint` → `DIDKey.ValidAt`. A
    verified-but-garbled body is a **non-nil error**, not a status — callers must check `err` first.
    `CheckpointInfo{Origin,TreeSize,Root}` is the zero value on every non-verified outcome. Pure,
    dependency-injected, never reads the clock. Tested by `TestAcceptCheckpoint` (6 subcases).
  - `internal/didweb/resolve.go` — `parseTime` **fails closed**: a non-empty-but-unparseable RFC-3339
    validity timestamp returns a wrapped error → `ParseDIDDocument` → `ErrUnresolvable` (never a false
    `verified`). Empty = "no constraint". Covered by `TestParseDIDDocument` malformed subcases.
  - `internal/didweb/validity.go` — pure `func (k DIDKey) ValidAt(now time.Time) bool`: half-open
    `[ValidFrom, ValidUntil)`, revoked at/after `Revoked`, zero field = "no constraint"; imports only
    `time` (WASM-pure). Tested by `TestValidAt` + `TestValidAtZeroKeyNow`.
  - `internal/logclient/verify.go` — pure `VerifyCheckpoint(vkey, raw)`: `note.NewVerifier` + `note.Open`
    + the oracle's exact reject, then `parseCheckpointBody`. Exported `ErrUnverified`. Tested against
    sb0/sb1 real fixtures.
  - `internal/logclient/didresolve.go` — networked did:web resolver: `Fetcher` 1-method seam,
    `NewHTTPFetcher`, `ErrUnresolvable`, `ResolveVerifierKey`. Tested offline (fake + httptest).
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; golden-tested.
  - `internal/didweb/{url,resolve,vkey}.go` — pure `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/
    `DIDKey` (byte-exact port of `derive_vkey.py`). Golden-tested.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed C2SP checkpoints.
  - Test totals: 9 `func Test` in `internal/didweb`, 10 in `internal/logclient`, 4 in `internal/store`
    (23 total).
  - Gate-dodge scan clean (no `//nolint` / `t.Skip` / `//go:build ignore` in `internal`).
- Missing (the connective majority of M1 — nothing consumes `AcceptCheckpoint` or writes the store yet):
  - **Per-hub follower poll loop**: call `AcceptCheckpoint` (check `err` before status), persist the
    verdict into `checkpoints`/`follow_state` (only `StatusVerified` advances accepted state; the other
    three are recorded findings while mirroring continues), populate the `hub_keys` did:web cache; run
    the three-trigger RFC-6962 consistency check (fork/shrink/equivocation via `transparency-dev/merkle`),
    persist both raw checkpoints + proof into `violations`, set `frozen=1`, alert once, no auto-unfreeze,
    other hubs unaffected.
  - **Store CRUD/query helpers** — schema + open/close exist, but no typed insert/query methods yet
    (they land with their real caller, the follower).
  - config + realm registry (domains only), coverage (`monitored_since`), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet**.
- Fixtures: did:web golden fixtures under `internal/{didweb,logclient}/testdata/`; `testdata/live/`
  holds the two checkpoints. Still **no tiles or entry bundles** in `testdata/live/` (needed for the
  consistency check + M2 aggregator). Known stale-fixture drift (not yet acted on): `derive_vkey.py`
  `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation key
  (`22b08f3e`); refresh to `069d0f14` lands with the follower/`hub_keys` step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (in `verify.go`), `modernc.org/sqlite` (in
  `store/sqlite.go`). Not yet wired: `transparency-dev/*` (merkle/tessera/formats),
  `nbd-wtf/opentimestamps`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (real checkpoints verify end-to-end; `AcceptCheckpoint` returns
  `StatusVerified` with `TreeSize == 10183` for sb0). Restart survival is demonstrated at the store
  layer (`TestStoreRestartSurvival`). The synthetic fork/shrink/equivocation → `violations.kind` +
  `frozen=1` + exactly-one-alert + other-hubs-unaffected half of M1 is **not started** (no follower).

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable (`go build ./... && go vet ./... && go test ./...`). Latest `review` handoff
  records the gate green at HEAD `414b64c` (build + vet + test exit 0 on go1.24.13, `gofmt -l .` empty;
  store tables/pragmas/FK enforcement independently verified, `go mod tidy` zero-diff).
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop` (0/0). **No `.github/workflows/` and no CI runs — no CI configured.** When CI is
  wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out the
  future `notecheck` oracle rather than `go run` from `cauldron/`, and re-assert WAL/FK pragmas on any
  read connections if a read-pool split lands at serving — see learnings.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): the **stateful follower poll loop** plus the
typed store insert/query helpers it drives. The follower calls `AcceptCheckpoint`, checks the returned
`err` before the status (a verified-but-garbled body is a fault, not a four-way verdict), persists the
verdict into `checkpoints`/`follow_state` (only `StatusVerified` advances accepted state), and writes
the `hub_keys` did:web cache — that step also refreshes the stale sb1 did.json fixture +
`derive_vkey.py` `HUBS` to the current key (`069d0f14`). The three-trigger RFC-6962 consistency check
(needs tiles in `testdata/live/` + `transparency-dev/merkle`) is the step after; then freeze/alert +
coverage + structured logs + `/metrics` complete M1's Verify criteria. No CI is configured — flag for
whoever sets up the GitHub workflow.
