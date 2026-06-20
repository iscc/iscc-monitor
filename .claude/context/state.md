<!-- assessed-at: 70493674f16cacd1561ca75fbeffb5bb126703d8 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verification primitives, the per-network SQLite store, transport-only checkpoint fetch, the single-observation `PollHub` caller, and now the store-side freeze/violations persistence seam are complete and tested; still no poll loop, no consistency-check *logic*, no freeze *trigger* wiring, and no binary

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; deps:
`golang.org/x/mod v0.33.0` for `sumdb/note`, `modernc.org/sqlite v1.46.1` pure-Go store). M1's *pure
verification primitives* (did:web trust-root chain, networked `ResolveVerifierKey`, signed-note
`VerifyCheckpoint`, `DIDKey.ValidAt`, composed four-way `AcceptCheckpoint`), the *stateful foundation*
(per-network SQLite with single-writer discipline + nine-table schema + typed CRUD), the *networked
fetch* (`FetchCheckpoint`), the *first real caller* (`internal/follower.PollHub`, one observation:
fetch → accept → record → advance for a single hub), and now the *freeze/violations store seam*
(`RecordViolation` + `Freeze`) are all in place and tested. What remains of M1 is the rest of the
*connective tissue*: the per-hub poll loop / single-writer goroutine wrapper, the three-trigger
RFC-6962 consistency-check **logic** (the seam to persist its output now exists but nothing computes
or triggers it), the alert-once mechanism, coverage, structured logs, `/metrics`, config + realm
registry, and a binary (`cmd/` is still absent). Last `review` verdict (HEAD `7049367`) is
**PASS / CONTINUE** with the gate recorded green; branch `develop`; `issues.md` empty.

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; since `055a70e` the only Go change is the additive
  freeze/violations seam in `internal/store/checkpoints.go` + `checkpoints_test.go` — all other
  sections carried forward unchanged and reconfirmed on disk):
  - `internal/store/checkpoints.go` — freeze/violations seam (NEW since last assessment): a `Violation`
    struct (`HubID`, `Kind` string, `RawA`/`RawB` bytes, `ProofJSON`, `DetectedAt`), a plain-INSERT
    `RecordViolation(ctx, Violation) (int64, error)` (no `ON CONFLICT` — re-detection records distinct
    rows, itself evidence; `LastInsertId`-only; zero `DetectedAt` → NULL via `unixOrNil`; empty
    `ProofJSON` stays empty string), and `Freeze(ctx, hubID int64) error` (upsert `INSERT … (hub_id,
    frozen) VALUES (?,1) ON CONFLICT(hub_id) DO UPDATE SET frozen=1`). `Freeze` is the **sole writer of
    `frozen`**; `AdvanceFollowState` omits it from its `DO UPDATE`, so advance-after-freeze keeps
    `frozen=1` (ADR-0006, no auto-unfreeze). Tested by `TestRecordViolation*` + `TestFreeze*` (round-trip,
    no-dedupe, zero-detected-at-NULL, freeze-no-prior-row, no-auto-unfreeze, other-hubs-unaffected,
    restart-survival — all non-vacuous). **No callers wired** — this is a persistence seam only.
  - `internal/follower/follower.go` — `PollHub(ctx, *store.Store, logclient.Fetcher, hubID, baseURL,
    observedAt) (logclient.Status, error)`: the first real caller composing the M1 verify chain
    (`FetchCheckpoint → AcceptCheckpoint`) with store CRUD (`RecordCheckpoint → AdvanceFollowState`) for
    one hub, one observation. Imports only `logclient` + `store` (+ stdlib) — direction
    follower → {logclient, store}, store stays a leaf. Honors err-before-status; persists + advances
    **only** on `StatusVerified`. Tested by `TestPollHubVerifiedAdvances` (sb0 → `LastSize == 10183`,
    non-vacuous) + `TestPollHubUnverifiedDoesNotAdvance` (`LastSize == 0`).
  - `internal/logclient/checkpoint.go` — `FetchCheckpoint(ctx, Fetcher, baseURL) ([]byte, error)`:
    transport-only (imports only `context`+`fmt`), reuses `origin()` to derive
    `https://<domain>/checkpoint`, `%w`-wraps errors, does NOT wrap in `ErrUnresolvable`. Tested by
    `TestFetchCheckpoint` (5 subtests) + `TestFetchCheckpointOverHTTP` (real TLS round-trip).
  - `internal/store/{sqlite,schema}.go` + earlier `checkpoints.go` CRUD — `Open`/`Close` over
    `modernc.org/sqlite` with ADR-0005/0007 single-writer discipline (`WAL`, `busy_timeout=5000`,
    `foreign_keys=ON`, `synchronous=NORMAL`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`
    (`hubs`, `hub_keys`, `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`,
    `follow_state`, `ots`; no `network` column per ADR-0007, no `cosigs` per M7-deferral), and the
    `UpsertHub`/`RecordCheckpoint`/`FollowState`/`AdvanceFollowState` typed methods. `store` stays a
    **leaf** (`go list -deps ./internal/store` shows only the self-line; `net/http` absent).
  - `internal/logclient/accept.go` — pure `AcceptCheckpoint` composing `ResolveVerifierKey →
    VerifyCheckpoint → DIDKey.ValidAt` into the 4-way `Status`; verified-but-garbled body returns a
    non-nil error alongside `StatusUnverified`'s zero (callers check `err` first); `CheckpointInfo` zero
    on every non-verified outcome. Clock-injected.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt` half-open window,
    fail-closed `parseTime`. WASM-pure (no net imports).
  - `internal/logclient/{verify,didresolve,origin}.go` — pure `VerifyCheckpoint` (oracle-exact
    `note.Open` reject), networked `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed C2SP
    checkpoints. did:web golden fixtures under `internal/{didweb,logclient}/testdata/`.
  - Test totals: 9 (`didweb`) + 14 (`logclient`) + 18 (`store`) + 2 (`follower`) = **43 `func Test`**.
- Missing (still the connective majority of M1):
  - **Three-trigger RFC-6962 consistency-check LOGIC** — the store seam to *persist* a violation +
    freeze now exists (`RecordViolation`/`Freeze`), but nothing yet *computes* fork/shrink/equivocation
    (needs `transparency-dev/merkle` — still unwired), *triggers* the freeze, or fires the
    **exactly-one alert**. Needs tiles in `testdata/live/`. Recommended scoping (per handoff):
    detection-only over fixtures first, then a separate slice wiring it into `follower.PollHub`.
  - **Poll loop / single-writer goroutine wrapper** that drives `PollHub` on a cadence and owns all
    writes per network DB (ADR-0005/0007). `PollHub` does one observation per call; nothing loops it.
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - config + realm registry (domains only), coverage (`monitored_since`), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet**.
- Fixtures: `testdata/live/` still holds **only the two checkpoints — no tiles or entry bundles**
  (needed for the consistency check + M2). Known stale-fixture drift (not yet acted on): `derive_vkey.py`
  `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation key
  (`22b08f3e`); the live sb1 signer is now `069d0f14`; refresh lands with the `hub_keys` cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired**: `transparency-dev/*` (merkle/tessera/
  formats), `nbd-wtf/opentimestamps` (grep confirms zero references in `internal/` or `go.mod`).
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (real checkpoints verify end-to-end; `AcceptCheckpoint` →
  `StatusVerified`, `TreeSize == 10183` for sb0; `PollHub` flows that verdict to a persisted cursor).
  Restart survival demonstrated at the store layer (incl. freeze persistence). The synthetic
  fork/shrink/equivocation → `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected
  half of M1 is **not met**: the persistence destination exists, but no detection logic, freeze trigger,
  or alert path yet exists to populate it from a real consistency comparison.

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
  handoff records the gate green at HEAD `7049367` (build + vet + test exit 0 on go1.24; `gofmt -l .`
  empty; `TestRecordViolation|TestFreeze` PASS all 7 subtests; store stays a leaf; no
  `//nolint`/`t.Skip`/build-tag/swallowed-error dodges in the 3 unpushed commits). Conformance/oracle
  gate correctly N/A this step — the freeze/violations seam is pure persistence, touching no
  signature/RFC-6962/proof/didweb code.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`. **No `.github/workflows/`
  and no CI runs — no CI configured.** When CI is wired it must avoid `go build ./...` over the
  gitignored `cauldron/` reference trees and shell out the future `notecheck` oracle rather than
  `go run` from `cauldron/`, and re-assert WAL/FK pragmas on any read connections if a read-pool split
  lands at serving — see learnings.

## Next Milestone
Continue M1. Per the PASS/CONTINUE handoff, the natural follow-on is the **three-trigger RFC-6962
consistency-check logic** — it now has a tested store seam (`RecordViolation` + `Freeze`) to drive.
That step is heavier: it needs the `transparency-dev/merkle` dep (`go get`, with the v1.24-compatible
pin caveat) plus tiles fixtures in `testdata/live/`, so scope it as detection-only over fixtures first,
then a separate slice wiring it into `follower.PollHub` + the exactly-one-alert mechanism. Independent
parallel ≤3-file slices remain available: (a) the **poll loop / single-writer goroutine wrapper**; and
(b) the **`hub_keys` did:web cache write** (which MUST also refresh the stale `sb1.amlet.id_did.json`
fixture + `derive_vkey.py` HUBS to signer `069d0f14`, re-triggering the parity oracle gate). Coverage +
structured logs + `/metrics` + a `cmd/` binary then complete M1's Verify criteria. No CI is configured —
flag for whoever sets up the GitHub workflow.
