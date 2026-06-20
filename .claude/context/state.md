<!-- assessed-at: 8715370321c5a77f2c83bfe374aa4b4523543320 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — all four pure verification primitives complete and golden-tested (did:web chain + signed-note verify + validity window + composed four-way `AcceptCheckpoint`); no store/follower/binary yet

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`; one dep:
`golang.org/x/mod v0.33.0` for `sumdb/note`). M1's *pure verification primitives* are now complete:
the did:web trust-root chain, the networked `ResolveVerifierKey`, the signed-note `VerifyCheckpoint`,
`DIDKey.ValidAt`, and — new this iteration — `AcceptCheckpoint`, which composes all three into the
ADR-0009 four-way hub-status verdict. What remains of M1 is the entire *stateful* layer: the per-hub
follower poll loop, the per-network SQLite store, the three-trigger RFC-6962 consistency check, freeze
+ alert, coverage, structured logs, `/metrics`, restart survival — and there is still no binary
entrypoint. Last `review` verdict (HEAD `8715370`) is **PASS** with the gate recorded green; branch
`develop` in sync with `origin/develop` (0 ahead / 0 behind); `issues.md` is empty.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; since `bd9030d` only `internal/logclient/accept.go` +
  `accept_test.go` were added and `internal/didweb/resolve.go` + `resolve_test.go` changed for the
  `parseTime` fail-closed fix — all other sections carried forward unchanged):
  - `internal/logclient/accept.go` (NEW since last assessment) — pure `Status` enum
    (`StatusVerified/Unverified/Unresolvable/Rotated` + `String()`) and
    `AcceptCheckpoint(ctx, fetcher, baseURL, raw, observedAt)`. Composes the three primitives in
    load-bearing order: `ResolveVerifierKey` (`ErrUnresolvable → StatusUnresolvable`) → `VerifyCheckpoint`
    (`ErrUnverified → StatusUnverified`; a verified-but-garbled body is returned as a **non-nil error**,
    not folded into a status — callers must check `err` first) → `DIDKey.ValidAt(observedAt)`
    (out-of-window valid signature → `StatusRotated`, in-window → `StatusVerified` with `CheckpointInfo`).
    `CheckpointInfo{Origin, TreeSize, Root}` is the zero value on every non-verified outcome. Pure,
    dependency-injected through the `Fetcher` seam, never reads the clock. Tested by
    `TestAcceptCheckpoint` (6 subcases: verified, unverified, unresolvable×2 [not-found + malformed-revoked],
    rotated×2 [validUntil + revoked in the past]).
  - `internal/didweb/resolve.go` — `parseTime` now **fails closed**: a non-empty-but-unparseable RFC-3339
    validity timestamp returns a wrapped error so `ParseDIDDocument` → `ErrUnresolvable` (never silently
    treated as absent → never a false `verified`). Empty string = "no constraint" (zero time, no error).
    Closes the previously-open `normal` fail-open issue; covered by `TestParseDIDDocument` malformed
    subcases.
  - `internal/didweb/validity.go` — pure `func (k DIDKey) ValidAt(now time.Time) bool`: half-open
    `[ValidFrom, ValidUntil)`, revoked at/after `Revoked`, zero field = "no constraint". Takes `now` as an
    argument (deterministic); imports only `time` (WASM-pure). Tested by `TestValidAt` + `TestValidAtZeroKeyNow`.
  - `internal/logclient/verify.go` — pure `VerifyCheckpoint(vkey, raw) -> (origin, treeSize, root, err)`:
    `note.NewVerifier` + `note.Open` + the oracle's exact reject, then `parseCheckpointBody`. Exported
    `ErrUnverified`. Tested by `TestVerifyCheckpoint` (sb0/sb1 real fixtures), `…Unverified`,
    `TestParseCheckpointBody`. Carried forward unchanged.
  - `internal/logclient/didresolve.go` — networked did:web resolver: `Fetcher` 1-method seam,
    `NewHTTPFetcher`, `ErrUnresolvable`, `ResolveVerifierKey`. Tested by `TestResolveVerifierKey`,
    `…Unresolvable`, `…OverHTTP`, `TestHTTPFetcherNotFound`. Carried forward unchanged.
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; `TestOrigin` + `TestOriginErrors`.
  - `internal/didweb/{url,resolve,vkey}.go` — pure `DocumentURL`, `ParseDIDDocument`, `VerifierKey`/`DIDKey`
    (byte-exact port of `derive_vkey.py`). Golden-tested.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed C2SP checkpoints.
  - Test totals: 9 `func Test` in `internal/didweb`, 10 in `internal/logclient` (19 total).
  - Gate-dodge scan clean (no `//nolint` / `t.Skip` / `//go:build ignore` in `internal`).
- Missing (the stateful majority of M1 — nothing consumes `AcceptCheckpoint`'s verdict yet):
  - **Per-hub follower poll loop**: call `AcceptCheckpoint` and persist the verdict (only
    `StatusVerified` advances accepted state; the other three are recorded findings while mirroring
    continues); run the three-trigger RFC-6962 consistency check (fork/shrink/equivocation via
    `transparency-dev/merkle`), persist both raw checkpoints + proof, set `frozen=1`, alert once,
    no auto-unfreeze, other hubs unaffected.
  - **Per-network SQLite store** (`modernc.org/sqlite`): `hub_keys` cache, `violations`, coverage
    (`monitored_since`), restart survival.
  - config + realm registry (domains only), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet**.
- Fixtures: did:web golden fixtures under `internal/{didweb,logclient}/testdata/`; `testdata/live/`
  holds the two checkpoints. Still **no tiles or entry bundles** in `testdata/live/` (needed for the
  consistency check + M2 aggregator). Known stale-fixture drift (not yet acted on): `derive_vkey.py`
  `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation key
  (`22b08f3e`); refresh to `069d0f14` lands with the follower/`hub_keys` step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (in `verify.go`). Not yet wired:
  `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`, `modernc.org/sqlite`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (real checkpoints verify end-to-end under the resolved key, and
  `AcceptCheckpoint` returns `StatusVerified` with `TreeSize == 10183` for sb0). The synthetic
  fork/shrink/equivocation → `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected
  + restart-survival half of M1 is **not started** (no follower or store).

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, requires `x/mod v0.33.0`); `mise run check` runnable
  (`go build ./... && go vet ./... && go test ./...`). Latest `review` handoff records the gate green
  at HEAD `8715370` (build/vet/test exit 0, `gofmt -l .` empty,
  `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds; trust-root oracle `derive_vkey.py`
  byte-exact and live checkpoint fixtures verify under `note.Open`).
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop` (0/0). **No `.github/workflows/` and `gh run list` returns empty — no CI configured.**
  When CI is wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and
  shell out the future `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): the **SQLite `hub_keys` cache + the stateful
follower poll loop** that calls `AcceptCheckpoint` and persists the verdict — only `StatusVerified`
advances accepted state; the other three are recorded findings while mirroring continues. The follower
must check `AcceptCheckpoint`'s returned `err` before the status (a verified-but-garbled body is a fault,
not a four-way verdict). Refresh the stale sb1 did.json fixture + `derive_vkey.py` `HUBS` to `069d0f14`
at this step. The three-trigger RFC-6962 consistency check (needs tiles in `testdata/live/` +
`transparency-dev/merkle`) is the step after; then freeze/alert + coverage + restart-survival complete
M1's Verify criteria. No CI is configured — flag for whoever sets up the GitHub workflow.
