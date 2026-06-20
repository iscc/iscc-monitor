<!-- assessed-at: 259bcc468d31681738a600d431db65352f3c4d8e -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — verification primitives, the per-network SQLite store (incl. freeze/violations seam), transport-only checkpoint fetch, the single-observation `PollHub` caller, and now BOTH dep-free pure consistency triggers (`CheckShrink` + `CheckFork`) are complete and tested; still no trigger→freeze wiring, no equivocation trigger, no poll loop, no alert mechanism, and no binary

The Go module is live (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; deps:
`golang.org/x/mod v0.33.0` for `sumdb/note`, `modernc.org/sqlite v1.46.1` pure-Go store). M1's *pure
verification primitives* (did:web trust-root chain, networked `ResolveVerifierKey`, signed-note
`VerifyCheckpoint`, `DIDKey.ValidAt`, composed four-way `AcceptCheckpoint`), the *stateful foundation*
(per-network SQLite with single-writer discipline + nine-table schema + typed CRUD + freeze/violations
seam), the *networked fetch* (`FetchCheckpoint`), the *first real caller* (`internal/follower.PollHub`,
one observation), and now BOTH *dep-free pure consistency triggers* (`CheckShrink`/`ViolationShrink` and
`CheckFork`/`ViolationFork`) are all in place and tested. What remains of M1 is the rest of the
*connective tissue*: the trigger→freeze wiring into `PollHub`, the alert-once mechanism, the third
(merkle-backed) equivocation trigger, the per-hub poll loop / single-writer goroutine wrapper, coverage,
structured logs, `/metrics`, config + realm registry, and a binary (`cmd/` is still absent).

## M1 — Read-only Monitor
**Status**: partially met
- Verified present (incremental re-check; since `f379919` the only Go change is the additive
  `CheckFork` trigger in `internal/logclient/consistency.go` + `consistency_test.go` — all other
  sections carried forward unchanged and reconfirmed on disk):
  - `internal/logclient/consistency.go` — TWO dep-free RFC-6962 triggers now landed:
    - `CheckShrink(prev, next uint64) bool == prev > 0 && next < prev` — strict tree-size decrease.
    - `CheckFork(prevSize uint64, prevRoot [rootBytes]byte, nextSize uint64, nextRoot [rootBytes]byte)
      bool == prevSize > 0 && nextSize == prevSize && nextRoot != prevRoot` (NEW since last assessment)
      — same committed size, differing root, via Go elementwise `[rootBytes]byte` `!=` (no `bytes`
      import). Both `prevSize > 0` guards are load-bearing (keep fresh-store `FollowState{}.LastSize ==
      0` + its zero root from misreading as a violation). `ViolationKind` string type +
      `ViolationShrink = "shrink"` + `ViolationFork = "fork"` consts. No new dependency — the
      `transparency-dev/merkle` mention in the file's doc comment is **prose only** (confirmed: zero
      import statements in `internal/`/`cmd/`; `go.mod` carries no `transparency-dev`/`opentimestamps`
      line). **No callers wired** — `CheckShrink`/`CheckFork`/`ViolationShrink`/`ViolationFork` are
      referenced only by their own tests (an intentional export seam; `go vet` clean). Tested by
      `TestCheckShrink` (6 subtests) + `TestCheckFork` (5 subtests, with a `rootA != rootB` setup guard
      making the differing-root case non-vacuous) + `TestViolationForkKind`.
  - `internal/store/checkpoints.go` — freeze/violations seam: `Violation` struct, plain-INSERT
    `RecordViolation` (no `ON CONFLICT` — re-detection records distinct rows as evidence), and
    `Freeze(ctx, hubID) error` (upsert; **sole writer of `frozen`**; `AdvanceFollowState` omits it so
    advance-after-freeze keeps `frozen=1`, ADR-0006 no auto-unfreeze). Tested round-trip + restart.
    **Still a persistence seam only — no callers** (confirmed: `RecordViolation`/`Freeze` referenced
    only by their own tests).
  - `internal/follower/follower.go` — `PollHub(...)`: the first real caller composing
    `FetchCheckpoint → AcceptCheckpoint` with `RecordCheckpoint → AdvanceFollowState` for one hub, one
    observation. Imports only `logclient` + `store` (+ stdlib); persists + advances **only** on
    `StatusVerified`. Tested by `TestPollHubVerifiedAdvances` (sb0 → `LastSize == 10183`) +
    `TestPollHubUnverifiedDoesNotAdvance`. **Does not yet call CheckShrink / CheckFork /
    RecordViolation / Freeze** (confirmed: zero trigger references in `follower.go`).
  - `internal/logclient/checkpoint.go` — `FetchCheckpoint(...)`: transport-only, reuses `origin()`,
    `%w`-wraps. Tested incl. a real TLS round-trip.
  - `internal/store/{sqlite,schema}.go` + `checkpoints.go` CRUD — `Open`/`Close` over `modernc.org/sqlite`
    with ADR-0005/0007 single-writer discipline (`WAL`, `busy_timeout=5000`, `foreign_keys=ON`,
    `synchronous=NORMAL`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`, typed
    `UpsertHub`/`RecordCheckpoint`/`FollowState`/`AdvanceFollowState`. `store` stays a **leaf**.
  - `internal/logclient/accept.go` — pure `AcceptCheckpoint` composing the 4-way `Status`; clock-injected.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `internal/logclient/{verify,didresolve,origin}.go` — pure `VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints.
  - Test totals: 9 (`didweb`) + 18 (`logclient`) + 18 (`store`) + 2 (`follower`) = **47 `func Test`**
    (logclient +2 since last assessment for the CheckFork tests).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — shrink and fork are both pure verdicts and have landed; equivocation
    (RFC-6962 consistency-proof failure across growing sizes) is the only remaining trigger and needs
    `transparency-dev/merkle` + tiles fixtures — **both still absent**.
  - **Trigger → freeze wiring** — `CheckShrink`/`CheckFork` (verdicts) + `RecordViolation`/`Freeze`
    (persistence) all exist but nothing composes them in `follower.PollHub`, and there is **no
    exactly-one-alert mechanism**. This is the step that turns the pure verdicts into M1 freeze behavior.
  - **Poll loop / single-writer goroutine wrapper** driving `PollHub` on a cadence (ADR-0005/0007).
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - config + realm registry (domains only), coverage (`monitored_since`), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet** (verified `cmd/` does not exist).
- Fixtures: `testdata/live/` still holds **only the two checkpoints — no tiles or entry bundles**
  (needed for the equivocation trigger + M2). Known stale-fixture drift (not yet acted on):
  `derive_vkey.py` `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation
  key (`22b08f3e`); the live sb1 signer is now `069d0f14`; refresh lands with the `hub_keys` cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero import statements + zero
  `go.mod` entries): `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met** (real checkpoints verify end-to-end; `AcceptCheckpoint` →
  `StatusVerified`, `TreeSize == 10183` for sb0; `PollHub` flows that verdict to a persisted cursor).
  Restart survival demonstrated at the store layer (incl. freeze persistence). The synthetic
  fork/shrink/equivocation → `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-unaffected
  half of M1 is **not met**: shrink AND fork *detection* are now tested pure verdicts, but they are not
  wired into the follower, equivocation detection is absent, no freeze trigger fires, and no alert path
  yet populates the persistence destination from a real comparison.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable (`mise.toml` present at repo root: build + vet + test). Latest `review`
  handoff records the gate green at HEAD `259bcc4` (build + vet + test `ok` on go1.24; `gofmt -l .`
  empty; `TestCheckFork` PASS all 5 subtests + `TestViolationForkKind`; no new dep — `go.mod`/`go.sum`
  unchanged; no `//nolint`/`t.Skip`/build-tag/swallowed-error dodges in the unpushed commits).
  Conformance/oracle gate correctly N/A this step — `CheckFork` is a pure size/root array comparison,
  touching no signature/RFC-6962-proof/didweb/merkle code. The merkle-backed equivocation slice that
  follows *will* trip the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `259bcc4`. **No `.github/workflows/` and no CI runs — no CI configured.** When CI is wired it must
  avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out the future
  `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. Both dep-free pure triggers (shrink + fork) have landed; the natural follow-on is the
**wiring slice into `follower.PollHub`** that turns those verdicts into M1's freeze behavior: map
`FollowState.LastSize → prevSize` and the stored root at that size (a `checkpoints` lookup, since
`LastRoot` is intentionally NOT persisted in `follow_state`) → prevRoot; map `CheckpointInfo.TreeSize/
Root → nextSize/nextRoot`; on a true `CheckShrink`/`CheckFork` verdict call `RecordViolation` + `Freeze`;
add the exactly-one-alert mechanism. Test through the outbound-fetch boundary with synthetic
same-size-different-root / shrink fixtures, asserting `violations.kind` + `frozen=1` + exactly one alert
+ other hubs unaffected + evidence surviving restart. The merkle-backed **equivocation** trigger
(consistency-proof failure; needs `transparency-dev/merkle` + tile fixtures, will trip the oracle gate)
remains the third trigger. Independent parallel ≤3-file slices remain available: (a) the **poll loop /
single-writer goroutine wrapper**; and (b) the **`hub_keys` did:web cache write** (which MUST also
refresh the stale `sb1.amlet.id_did.json` fixture + `derive_vkey.py` HUBS to signer `069d0f14`,
re-triggering the parity oracle gate). Coverage + structured logs + `/metrics` + a `cmd/` binary then
complete M1's Verify criteria. No CI is configured — flag for whoever sets up the workflow.
