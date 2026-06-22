<!-- assessed-at: 66fd02ace8a31314423cbaec30c0fb2a308f4217 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI tail + OTS milestone. The OTS path has its plumbing in place — daily stamp pass + the upgrade-loop control core — but not yet its payload. `OTSTick` walks `PendingOTS`, asks an **injected** `Upgrader` whether each root is Bitcoin-confirmed, and applies confirm / back-off via the store seam; the **real** `Upgrader` (the `opentimestamps` calendar client), the live `Run` goroutine in `main.go`, the `.ots` route, and certificate §5 are all still absent. Remaining v1 work: those OTS payload pieces (which unblock §5 BITCOIN ANCHOR), the WASM verifier, the anchor panels, and the mandatory M-UI visual-exit gate.

The reviewed OTS upgrade-loop core is the latest landed increment (HEAD `66fd02a`, a `cid(review)` PASS
commit). It adds `internal/follower/otsloop.go` — `OTSTick(ctx, st, up, now, logger)` over the
`Upgrader func` seam, plus a capped-exponential `backoff` — and extends the store with
`MarkOTSAttempted` + a `now`-filtered `PendingOTS` back-off WHERE leg. No `go.mod`/`go.sum` change, no
`opentimestamps` dependency, not wired into `cmd/iscc-monitor/main.go`, no HTTP route, no §5 render.
M1/M2/M3 remain fully met. **HEAD is reviewed PASS and pushed** (branch is in sync with
`origin/develop`, 0 ahead / 0 behind) and **CI is green at HEAD** (run 27924202260, `66fd02a` =
success) — so the last confirmed-green gate is at HEAD this iteration, not behind it.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): partially met.** Met: five-status `HubStatusBadge`
    (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/`
    realm-index grid; `/<domain>/log/` browser; hub dossier (`GET /<domain>`, `internal/dossier`);
    frozen Exhibit; paginated record list; single-record page; ISCC-IDv1 decoder
    (`internal/index.Decode`); `(realm, hub_id) -> domain` Hub-List resolver (`internal/registry`);
    certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION PROOF + §4 SIGNING KEY + §6 RECORD
    HISTORY**; the proof-bundle endpoint (`.bundle`) + download link. **Open:** certificate **§5 BITCOIN
    ANCHOR** (`HasClause5` still hard-false — re-verified at `internal/certificate/handler.go:286`); the
    separate **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI exit visual-pass +
    human sign-off** (ADR-0012 agent-browser).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file; both re-verified empty this iteration).
  - **OTS anchoring: 1/1 open — third sub-step landed.** Three pieces now exist and are tested: (1) the
    `ots`-table CRUD seam (`internal/store/ots.go`); (2) the daily stamp pass (`stampRoot` in PollHub);
    (3) the **upgrade-loop control core** (`OTSTick` over the injected `Upgrader` seam + `backoff`). The
    milestone's Verify ("a stamped root upgrades to Bitcoin-confirmed and the served `.ots` verifies with
    the standard `ots` client") still needs the **real `Upgrader`** (the `nbd-wtf/opentimestamps`
    calendar-HTTP client — the first `go.mod`/`go.sum` change for this milestone, still absent), the live
    `Run` goroutine wired into `main.go` (OTSTick has **no production caller** today), the `.ots` route,
    and §5. The control flow is proven against a fake `Upgrader`; no root has actually been stamped to
    Bitcoin. Still 1/1 open.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 foundational·plumbing.** Recent arc: cert §3
  fail-close -> §6 -> proof-bundle endpoint -> #ZgotmplZ link fix -> ADR-0011 stack bump -> OTS store CRUD
  seam -> OTS stamp pass -> **OTS upgrade-loop core**. The last **four** increments (stack bump, OTS
  store seam, stamp pass, upgrade-loop core) are foundational/plumbing rather than direct Verify-closures.
  This is the fourth-consecutive-non-closing increment the prior state.md flagged as the drift watch-line
  — but it stays on the only path to the open OTS milestone and is NOT polish: each is a target-mandated
  prerequisite, mutation-proven non-vacuous, and the OTS-core was the named "increment that reaches toward
  the bar." **Watch sharpens:** the next increment should be the **real `Upgrader` + live wiring** that
  finally makes a stamped root reach Bitcoin and so closes the OTS Verify criterion. A *fifth* plumbing
  step that still does not call `OTSTick` from `main.go` with a real calendar client would read as drift.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since last assessment. The
`3b8db9f..HEAD` diff touched ONLY `internal/follower/otsloop.go` (new, OTS upgrade loop),
`internal/store/ots.go` (back-off seam), their `_test.go` files, and `.claude/*` context;
`go.mod`/`go.sum`/`schema.sql`/the consistency-trigger code all byte-unchanged. All M1 Verify criteria
remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze +
alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **19 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, proofserve, registry, store, tiles, tilesserve, web` (no new package this
  iteration; the OTS upgrade loop lives inside `internal/follower`). Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure; production `internal/index` leaf stays iscc-lib-free under the
  ADR-0011 carve-out). **Not wired:** `nbd-wtf/opentimestamps` — re-verified absent from `go.mod`/`go.sum`
  and from every `.go` source (the upgrade loop operates on the abstract `Upgrader` seam only).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard;
`GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry `no-cache` +
strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — carried forward (no M-UI source in the diff).** §1 SUBJECT, §2 CHECKPOINT,
§3 INCLUSION PROOF, §4 SIGNING KEY, §6 RECORD HISTORY all sound. Badge, DS shell, `/` index,
`/<domain>/log/` browser, hub dossier, frozen Exhibit, record list, single-record page, ISCC-IDv1
decoder, Hub-List resolver, proof-bundle endpoint + download link all built and verified.
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` hard-false at
  `handler.go:286`, re-verified; §5 will read OTS rows via `OTSForRoot` but the upgrade loop has no real
  Upgrader/wiring yet, so no confirmed anchor data exists to render); the separate **Bitcoin-anchor vs
  comparison-anchor** panels; the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — three sub-steps landed (store seam + stamp pass + upgrade-loop core); WASM — not
started.**
- **OTS:** (1) `internal/store/ots.go` — typed `ots`-table CRUD
  (`RecordOTS`/`OTSForRoot`/`PendingOTS`/`MarkOTSUpgraded`/`MarkOTSAttempted`). (2) The daily stamp pass
  (`stampRoot` in `internal/follower/follower.go`) writes the accepted root as a `pending` row on the
  verified non-violation advance path, idempotent via `UNIQUE(hub, tree_size, root)`. (3) **The
  upgrade-loop control core** — `OTSTick(ctx, st, up, now, logger)` in `internal/follower/otsloop.go`
  reads `now`-filtered `PendingOTS`, asks the injected `Upgrader func(ctx, OTSRecord) (UpgradeResult,
  error)` whether each root is Bitcoin-confirmed, and either marks confirmed (`MarkOTSUpgraded`) or backs
  off (`MarkOTSAttempted`, capped-exponential `backoff`: base 1h, cap 24h); errors never abort the pass
  (log-and-continue), the wrapped first error is returned. `review` confirmed PASS at HEAD with the
  back-off `next_retry` WHERE leg and the no-op `MarkOTSAttempted` both mutation-proven non-vacuous; Codex
  clean; CI green. **Not yet built (the actual Verify-closer):** the **real `Upgrader`** (the
  `nbd-wtf/opentimestamps` calendar-HTTP client — first `go.mod`/`go.sum` change, still absent), the live
  `Run` goroutine in `cmd/iscc-monitor/main.go` (`OTSTick` has **zero production callers** today — verified
  not referenced in `cmd/`), the `.ots` HTTP route, and certificate §5. The `ots verify` crypto-oracle
  gate first applies at the real-Upgrader step. The milestone Verify is still 1/1 open: no root has
  actually been anchored to Bitcoin.
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The verifier app does not exist.

## Quality gates
**Status**: **GREEN at HEAD.** HEAD (`66fd02a`) is itself the `cid(review)` PASS commit; the branch is in
sync with `origin/develop` (0 ahead / 0 behind), and CI ran green on HEAD.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (all 21 packages `ok`, `gofmt -l .` clean) at HEAD.
- The latest **`review` PASS verdict is at HEAD `66fd02a`** (OTS upgrade-loop core).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **Latest CI run is at HEAD `66fd02a` =
  `success`** (run 27924202260); `381764d` and `2bf66a8` also `success`.
- **Open issues: 0 `critical`, 3 `normal`, 7 `low`.** The 3 `normal`: (a) Hub-List `hubDomain`
  `ForceQuery` fail-open; (b) §4 AND bundle `did:web:` + raw domain mis-render a `host:port` hub's DID
  (2 surfaces); (c) §6 RECORD HISTORY omits the per-record `· at` timestamp. The 7 `low` (one new this
  iteration — `-run TestOTS` under-selects the OTS store tests, a filter-shorthand gap, not a coverage
  hole) are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. Gate is green at HEAD — no CI hygiene needed.**
Per the roadmap, the immediate next goal is the OTS payload that actually reaches the Verify bar:

1. **Real `Upgrader` + live wiring** — implement the `Upgrader` closure against
   `github.com/nbd-wtf/opentimestamps` (the first `go.mod`/`go.sum` change for this milestone), driving
   stamp/query/upgrade over `OTSRecord.CalendarURLs` and returning `UpgradeResult{Confirmed, OTSBytes,
   BTCHeight}` once Bitcoin-attested; wire the live `Run` goroutine into `cmd/iscc-monitor/main.go` (own
   ticker, `defer Stop()`, log-and-continue, off the poll path — `OTSTick` currently has no production
   caller). The `ots verify` crypto-oracle gate first applies here. Then the `.ots` HTTP route (reads
   `OTSForRoot`) and certificate **§5 BITCOIN ANCHOR** (`HasClause5`, reading `OTSForRoot`) — closing the
   last open certificate clause + the Bitcoin-anchor panel + the OTS milestone Verify. This is the
   increment that finally closes a Verify criterion after four plumbing steps.
2. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` -> `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched.
