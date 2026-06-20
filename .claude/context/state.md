<!-- assessed-at: 12f815561cade41e835ed00acf042b503edd376a -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 in progress — the shrink/fork freeze loop is now wired end-to-end

Verification primitives, the per-network SQLite store (incl. freeze/violations seam), transport-only
checkpoint fetch, and the `PollHub` caller are complete. Since the last assessment the two dep-free
RFC-6962 triggers (`CheckShrink`/`CheckFork`) have been **wired into `follower.PollHub`** with freeze +
alert-once behavior (ADR-0006), backed by a new `store.CheckpointAt` read. What remains of M1: the
merkle-backed equivocation trigger, the poll loop / single-writer goroutine wrapper, the `hub_keys`
did:web cache write, config + realm registry, coverage, structured logs, `/metrics`, and a `cmd/` binary.

## M1 — Read-only Monitor
**Status**: partially met (shrink+fork freeze loop now closed; equivocation + connective tissue + binary still missing)
- Verified present (incremental re-check; the only Go change since `259bcc4` is the freeze/alert wiring
  in `internal/follower/follower.go` + `internal/store/checkpoints.go` (+ their tests) — all other
  sections carried forward unchanged and reconfirmed on disk):
  - **NEW — freeze + alert-once wiring** (`internal/follower/follower.go`): on `StatusVerified`,
    `PollHub` reads `FollowState`, runs `checkConsistency` (shrink-then-fork against the prior accepted
    checkpoint) BEFORE any record/advance; on a true verdict it calls `freeze(...)` which does
    `RecordViolation` + `RecordCheckpoint`(contradictory evidence, no advance) + `Freeze`, then fires the
    injected `AlertFunc func(hubID int64, kind string)` exactly once iff `!wasFrozen`. A violation
    returns `(StatusVerified, nil)` — freezes, never crashes (ADR-0006), no auto-unfreeze. Production
    imports stay `{context, fmt, logclient, store, time}`; the `AlertFunc` is a func seam (not an
    interface, YAGNI). Tested by `TestPollHubFork` (size 10183 distinct root → `violations.kind=="fork"`,
    `frozen==1`, cursor held, one alert; re-detection records a 2nd row, alert stays 1; other hubs
    advance clean; restart survives) and `TestPollHubShrink` (10183 < prior 20000 → `kind=="shrink"`,
    frozen, cursor held, one alert).
  - **NEW — `store.CheckpointAt(ctx, hubID, treeSize)` → `(root, raw, found, err)`** (`internal/store/
    checkpoints.go`): the supporting read that recovers the prior accepted root (since `follow_state`
    deliberately does not persist `LastRoot`); `LIMIT 1`, no `ORDER BY`. Absent `(hubID, treeSize)` and
    absent hub → `found==false`, nil error. Tested by `TestCheckpointAt` +
    `TestCheckpointHelpersRestartSurvival`.
  - `internal/logclient/consistency.go` — the two dep-free RFC-6962 triggers: `CheckShrink(prev, next)
    == prev>0 && next<prev` and `CheckFork(prevSize, prevRoot, nextSize, nextRoot) == prevSize>0 &&
    nextSize==prevSize && nextRoot!=prevRoot` (array `!=`, no `bytes` import). `ViolationKind` string
    type + `ViolationShrink="shrink"` / `ViolationFork="fork"`. Now consumed by `follower.checkConsistency`
    (no longer an unused export seam). Tested by `TestCheckShrink` + `TestCheckFork` + `TestViolationForkKind`.
  - `internal/store/{sqlite,schema}.go` + `checkpoints.go` CRUD — `Open`/`Close` over `modernc.org/sqlite`
    with ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`,
    `synchronous=NORMAL`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`, typed
    `UpsertHub`/`RecordCheckpoint`/`FollowState`/`AdvanceFollowState`/`CheckpointAt`/`RecordViolation`/
    `Freeze`. `store` stays a **leaf** (no `logclient` import; no `net/http` in its closure).
  - `internal/logclient/checkpoint.go` — `FetchCheckpoint(...)`: transport-only, `%w`-wraps. Tested
    incl. a real TLS round-trip.
  - `internal/logclient/accept.go` — pure `AcceptCheckpoint` composing the 4-way `Status`; clock-injected.
  - `internal/didweb/{url,resolve,vkey,validity}.go` — `DocumentURL`, `ParseDIDDocument`,
    `VerifierKey`/`DIDKey` (byte-exact port of `derive_vkey.py`), pure `ValidAt`. WASM-pure.
  - `internal/logclient/{verify,didresolve,origin}.go` — pure `VerifyCheckpoint`, networked
    `ResolveVerifierKey` over the 1-method `Fetcher` seam, golden `origin()`.
  - `testdata/live/sb0.iscc.id_checkpoint` + `sb1.amlet.id_checkpoint` — real hub-signed checkpoints;
    `internal/{didweb,logclient}/testdata/{sb0,sb1}_did.json` — did:web fixtures.
  - Test totals: 9 (`didweb`) + 18 (`logclient`) + 19 (`store`, +1 for `CheckpointAt`) + 4 (`follower`,
    +2 for fork/shrink) = **50 `func Test`** (up from 47).
- Missing (still the connective majority of M1):
  - **Equivocation trigger** — shrink + fork freeze loops are closed; equivocation (RFC-6962
    consistency-proof failure across *growing* sizes) is the only remaining trigger and needs
    `transparency-dev/merkle` + tile fixtures — **both still absent**. It will trip the conformance/
    oracle gate (`fsck` root-rebuild, inclusion cross-check, golden-vector parity).
  - **Poll loop / single-writer goroutine wrapper** driving `PollHub` on a cadence (incl. the backed-off
    evidence-only re-poll of a frozen hub) — `PollHub` is still one observation per call.
  - **`hub_keys` did:web cache write** — persists resolved keys; MUST also refresh the stale sb1 fixture.
  - config + realm registry (domains only), coverage (`monitored_since`), structured logs, `/metrics`.
  - `cmd/` is absent — **no binary entrypoint yet** (verified `cmd/` does not exist).
  - **Real alert transport** — `AlertFunc` is a minimal func seam; production delivery
    (email/webhook/log sink) is unbuilt.
- Fixtures: `testdata/live/` still holds **only the two checkpoints — no tiles or entry bundles**
  (needed for the equivocation trigger + M2). Known stale-fixture drift (not yet acted on):
  `derive_vkey.py` `HUBS` + both `sb1.amlet.id_did.json` did:web fixtures still carry sb1's PRE-rotation
  key (`22b08f3e`); the live sb1 signer is now `069d0f14`; refresh lands with the `hub_keys` cache step.
- Reuse imports wired: `golang.org/x/mod/sumdb/note` (`didweb/vkey.go`, `logclient/verify.go`),
  `modernc.org/sqlite` (`store/sqlite.go`). **Not yet wired** (confirmed zero import statements + zero
  `go.mod` entries): `transparency-dev/*` (merkle/tessera/formats), `nbd-wtf/opentimestamps`. The
  `transparency-dev/merkle` mention in `consistency.go` is prose-only.
- Verify criteria status: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and `VerifierKey`
  byte-match for both hubs — **met**. **Two of three triggers now fully met end-to-end**: synthetic
  shrink AND fork each → correct `violations.kind` + `frozen=1` + exactly-one-alert + other-hubs-
  unaffected + evidence-survives-restart (all proven by `TestPollHubFork`/`TestPollHubShrink` through the
  outbound-fetch seam, asserting on observable store rows via an independent connection). The **third
  trigger, equivocation, is not met** (absent), so the M1 Verify line is not fully satisfied.

## M2 — Aggregator
**Status**: not started.

## M3 — Trust API + dashboard
**Status**: not started.

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here)
- `go.mod` present (`go 1.24.0`, no `toolchain` line; requires `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable (`mise.toml` at repo root: build + vet + test). Latest `review` handoff
  (2026-06-20, "Wire CheckShrink + CheckFork into follower.PollHub") records the gate green at HEAD
  `12f8155`: `go build/vet/test ./...` all `ok` (re-confirmed uncached), `gofmt -l .` empty, all
  fork/shrink/CheckpointAt tests PASS (alert count stable over `-count=20`), `go.mod`/`go.sum`
  byte-identical (no dep added), no `//nolint`/`t.Skip`/build-tag/swallowed-error dodges.
  Conformance/oracle gate correctly N/A this step (no signature/RFC-6962-proof/didweb/merkle code
  touched; `internal/proof` does not exist yet). The merkle-backed equivocation slice that follows
  *will* trip the oracle gate.
- Remote `origin` configured (github.com/iscc/iscc-monitor); branch `develop`; tree clean at HEAD
  `12f8155`. **No `.github/workflows/` and `gh run list` returns `[]` — no CI configured.** When CI is
  wired it must avoid `go build ./...` over the gitignored `cauldron/` reference trees and shell out the
  future `notecheck` oracle rather than `go run` from `cauldron/` — see learnings.

## Next Milestone
Continue M1. The shrink + fork freeze loops are now closed end-to-end, so the highest-value remaining
slice is the **merkle-backed equivocation trigger** (RFC-6962 consistency-proof failure across growing
sizes): plug it into the same `checkConsistency` seam as a third branch returning `ViolationEquivocation`
+ a real `ProofJSON`; it needs `transparency-dev/merkle` + tile fixtures and **trips the conformance/
oracle gate** (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check vs the hub's
`IsccLogInclusionProof`, golden-vector parity). It MUST preserve "compare against the prior *accepted*
root, not the contradicting evidence" (see the `CheckpointAt` `LIMIT 1` rowid-order learning).
Independent parallel ≤3-file slices remain available: (a) the **poll loop / single-writer goroutine
wrapper** (backed-off evidence-only re-poll of a frozen hub); and (b) the **`hub_keys` did:web cache
write** (which MUST also refresh the stale `sb1.amlet.id_did.json` fixture + `derive_vkey.py` HUBS to
signer `069d0f14`, re-triggering the parity oracle gate). Coverage (`monitored_since`) + structured logs
+ `/metrics` + a `cmd/` binary + real alert transport then complete M1's Verify criteria. No CI is
configured — flag for whoever sets up the workflow.
