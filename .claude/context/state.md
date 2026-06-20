<!-- assessed-at: bd9030d498f9101faf0f03fe122785e853b5efc1 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verification primitives complete (did:web chain + signed-note checkpoint verify + pure validity predicate), all golden-tested; no follower/store/binary yet

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`; one dep:
`golang.org/x/mod v0.33.0` for `sumdb/note`). M1's *pure verification primitives* are now complete and
golden-tested: the did:web trust-root chain, the networked `ResolveVerifierKey`, the signed-note
`VerifyCheckpoint`, and — new this iteration — `DIDKey.ValidAt(now)`, the CID 1.0 validity-window
predicate. What remains of M1 is the entire *stateful* layer: the per-hub follower (three-trigger
consistency check + freeze + alert), the per-network SQLite store, coverage, structured logs,
`/metrics`, restart survival — and there is still no binary entrypoint. Last `review` verdict (HEAD
`bd9030d`) is **PASS** with the gate recorded green; branch `develop` in sync with `origin/develop`;
one open `normal` issue (`parseTime` fail-open, deferred to the `hub_keys` step).

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; only `internal/didweb/validity.go` + its test and a doc-only
  comment in `resolve.go` changed since the last assessment):
  - `internal/didweb/validity.go` (NEW since last assessment) — pure
    `func (k DIDKey) ValidAt(now time.Time) bool`: half-open `[ValidFrom, ValidUntil)`, revoked
    at/after `Revoked`, each zero field = "no constraint" so a zero-value `DIDKey` is always valid.
    Boundaries compare with `Before` only, guarded by `!IsZero()`; takes `now` as an argument
    (deterministic, never reads the clock); imports only `time` (WASM-pure). Tested by `TestValidAt`
    (12-case boundary table) + `TestValidAtZeroKeyNow`. **Pure predicate only — nothing consumes it
    yet**; the follower must call it and map out-of-window → not-`verified` (distinct from
    `ErrUnverified`/`ErrUnresolvable`).
  - `internal/logclient/verify.go` — pure `VerifyCheckpoint(vkey, raw) -> (origin, treeSize, root,
    err)`: `note.NewVerifier` + `note.Open`, the oracle's exact `len(Sigs)==0 || len(UnverifiedSigs)!=0`
    reject, then `parseCheckpointBody` (private) parsing the verified 3-line body and asserting
    signer-name == origin. Exported sentinel `ErrUnverified`. Imports only `sumdb/note` + stdlib.
    Tested by `TestVerifyCheckpoint` (sb0/sb1 real fixtures), `TestVerifyCheckpointUnverified`,
    `TestParseCheckpointBody`. Carried forward unchanged.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed C2SP
    checkpoints (sb0 size 10183, sb1 size 61; sb1 sig keyhash is the CURRENT rotated key `069d0f14`).
    Carried forward unchanged.
  - `internal/logclient/didresolve.go` — networked did:web resolver: `Fetcher` 1-method seam,
    `NewHTTPFetcher`, `ErrUnresolvable`, `ResolveVerifierKey(ctx, Fetcher, baseURL)`. Tested by
    `TestResolveVerifierKey`, `…Unresolvable`, `…OverHTTP` (httptest TLS), `TestHTTPFetcherNotFound`.
    Carried forward unchanged.
  - `internal/logclient/origin.go` — `origin(baseURL)` → `<domain>/log`; `TestOrigin` + `TestOriginErrors`.
    Carried forward unchanged.
  - `internal/didweb/{url,resolve,vkey}.go` — pure `DocumentURL`, `ParseDIDDocument` (polymorphic
    `assertionMethod`; parses CID 1.0 `ValidFrom`/`ValidUntil`/`Revoked`), `VerifierKey`/`DIDKey`
    (byte-exact port of `derive_vkey.py`). Golden-tested. Carried forward unchanged.
  - Test totals: 9 `func Test` in `internal/didweb`, 9 in `internal/logclient` (18 total).
  - Gate-dodge scan clean (no `//nolint` / `t.Skip` / swallowed err / build-tag exclusion in `internal`).
- Missing (the stateful majority of M1 — nothing consumes the verified key/checkpoint/validity yet):
  - **Per-hub follower**: wire `ResolveVerifierKey → VerifyCheckpoint → DIDKey.ValidAt` into a
    checkpoint-acceptance path mapping `ErrUnresolvable → unresolvable`, `ErrUnverified → unverified`,
    and out-of-window key → not-`verified`; run the three-trigger RFC-6962 consistency check
    (fork/shrink/equivocation via `transparency-dev/merkle`), persist both raw checkpoints + proof,
    set `frozen=1`, alert once, poll evidence-only at a backed-off cadence, no auto-unfreeze, other
    hubs unaffected.
  - **Per-network SQLite store** (`modernc.org/sqlite`): `hub_keys` cache, `violations`, coverage
    (`monitored_since`), restart survival.
  - config + realm registry (domains only), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet**.
- Fixtures: did:web golden fixtures under `internal/{didweb,logclient}/testdata/`; `testdata/live/`
  holds the two checkpoints. Still **no tiles or entry bundles** in `testdata/live/` (needed for the
  consistency check + M2 aggregator). Known stale-fixture drift (not yet acted on): `derive_vkey.py`
  `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation key
  (`22b08f3e`); refresh to `069d0f14` lands with the `hub_keys`/validity-enforcement step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (in `verify.go`). Not yet wired:
  `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`, `modernc.org/sqlite`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (exercised end-to-end: real checkpoints verify under the resolved
  key). The synthetic fork/shrink/equivocation → `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + restart-survival half of M1 is **not started** (no follower or store).

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
  at HEAD `bd9030d` (build/vet/test exit 0 on a cleared cache, `gofmt -l .` empty,
  `GOOS=js GOARCH=wasm go build ./internal/didweb` succeeds — `validity.go` is `time`-only WASM-pure).
  Trust-root oracle re-confirmed: `derive_vkey.py` byte-exact AND the live checkpoint fixtures verify
  under `note.Open`.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop` in sync with
  `origin/develop`. **No `.github/workflows/` and `gh run list` returns empty — no CI configured.**
  When CI is wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees
  (module-less, break a fresh build) and shell out the future `notecheck` oracle rather than `go run`
  from `cauldron/` — see learnings.

## Next Milestone
Continue M1. Immediate next unit (per the PASS handoff): the **SQLite `hub_keys` cache + follower
checkpoint-acceptance path** — wire `ResolveVerifierKey → VerifyCheckpoint`, calling
`DIDKey.ValidAt(observedAt)`; map `ErrUnresolvable → unresolvable`, `ErrUnverified → unverified`, and
out-of-window matching key → not-`verified` (rotation/revocation, a distinct outcome). Refresh the
stale sb1 fixture + `derive_vkey.py` `HUBS` to `069d0f14` here, and resolve the open `normal`
`parseTime` fail-open issue at this step (malformed validity timestamp must not yield `verified`). The
three-trigger RFC-6962 consistency check (needs tiles in `testdata/live/`) is the step after; then
freeze/alert + per-network store + coverage + restart-survival complete M1's Verify criteria. No CI is
configured — flag for whoever sets up the GitHub workflow.
