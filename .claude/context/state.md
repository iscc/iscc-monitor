<!-- assessed-at: a7555ed8b41fe2f9a93ede378c11d5bb64f6cc95 -->

# Project State

## Status: IN_PROGRESS

## Phase: OTS milestone + M-UI tail. The OTS upgrade path is now structurally crash- and hang-proof
(`internal/otsclient.safeUpgrade` wraps every calendar upgrade in a panic-recover + a per-request
`context.WithTimeout(ctx, 30s)`), which closed the last open `normal` otsclient issue. But OTS is
still NOT end-to-end: `stampRoot` writes pending rows with EMPTY `OTSBytes` (no calendar submit), so
`OTSTick`'s upgrade remains a structural no-op and no root has reached Bitcoin. Remaining v1 work is
unchanged: the `stampRoot` calendar-submit (the actual Verify-closer), the `.ots` route, certificate
§5, the WASM verifier, the anchor panels, and the mandatory M-UI visual-exit gate.

This is the `cid(review)` PASS commit for the otsclient hardening (HEAD `a7555ed`). Incremental review
from `b03c1ca`: the `b03c1ca..HEAD` diff touched ONLY `internal/otsclient/client.go` (+ test;
`safeUpgrade`) and `.claude/*` context — no M1/M2/M3/M-UI production source, so those sections are
carried forward verified. **HEAD is in sync with `origin/develop`** (0 ahead / 0 behind) and **CI at
HEAD `a7555ed` = success** (run 27926003590). The earlier side-branch `feat(visual-verify)`
agent-browser commit (`5fb2454`, ADR-0012) is now an ancestor of `develop`'s HEAD (merged), so the
M-UI visual-exit tooling is on `develop`; the exit gate itself (visual pass + human sign-off) is still
pending.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met.** Met: five-status `HubStatusBadge` (`internal/badge`); DS v2 shared shell
    (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/` realm-index grid; `/<domain>/log/` browser;
    hub dossier (`GET /<domain>`, `internal/dossier`); frozen Exhibit; paginated record list;
    single-record page; ISCC-IDv1 decoder (`internal/index.Decode`); `(realm, hub_id) -> domain`
    Hub-List resolver (`internal/registry`); certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION
    PROOF + §4 SIGNING KEY + §6 RECORD HISTORY**; the proof-bundle endpoint (`.bundle`) + download
    link. **Open:** certificate **§5 BITCOIN ANCHOR** (`HasClause5` still a gated placeholder, never set
    true — re-verified at `internal/certificate/handler.go:286`; blocked on confirmed OTS rows
    existing); the separate **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI
    exit visual-pass + human sign-off** (ADR-0012; the agent-browser tooling is now on `develop` via the
    merged `5fb2454`, but the pass + sign-off has not run).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file; both re-verified empty this iteration. The only `syscall/js` match in the tree is prose
    in `state.md`).
  - **OTS anchoring: 1/1 open — drift line closed and now hang/crash-proof, but still not end-to-end.**
    Six pieces exist and are tested: (1) `ots`-table CRUD seam (`internal/store/ots.go`); (2) the daily
    stamp pass (`stampRoot` in PollHub); (3) the upgrade-loop control core (`OTSTick`); (4) the
    `internal/ots` classifier (`Confirmed`, with the `>math.MaxInt64` height guard); (5) the real
    `internal/otsclient` Upgrader + `Stamp` + the `runOTSLoop` goroutine in `main.go` (first production
    caller of `OTSTick`/`ots.Confirmed`); (6) **NEW — `safeUpgrade`** (`internal/otsclient/client.go`):
    every calendar upgrade now runs under a per-request 30s timeout + panic-recover, closing the last
    `normal` otsclient defect. The milestone Verify ("a stamped root upgrades to Bitcoin-confirmed and
    the served `.ots` verifies with the standard `ots` client") still needs: **`stampRoot` to call
    `otsclient.Stamp`** so pending rows carry real `OTSBytes`/`CalendarURLs` (re-verified `stampRoot` at
    `follower.go:410-421` writes NO `OTSBytes`), the `.ots` HTTP route (re-verified absent), and
    certificate §5. **No root has actually been stamped to Bitcoin; `OTSTick`'s upgrade is a structural
    no-op until `OTSBytes` is populated.** Still 1/1 open.
- **Last ~10 iterations: ~3 milestone-Verify-advancing / ~7 foundational·plumbing·hardening.** Recent
  arc: proof-bundle endpoint -> ADR-0011 stack bump -> OTS store CRUD -> OTS stamp pass -> OTS
  upgrade-loop core -> OTS adapter -> real Upgrader + main.go wiring -> **otsclient upgrade-path
  hardening (safeUpgrade)**. **DRIFT WATCH (sharpening):** the last increment hardened the OTS upgrade
  path but did NOT advance the OTS Verify criterion — it de-risks the *next* step rather than closing
  the bar. That is legitimate de-risking (the panic/hang defects had to be fixed before `upgrade()`
  goes live in production), but it extends an OTS sub-step streak where the Verify criterion has stayed
  1/1 open across ~6 consecutive OTS increments. The single clear Verify-closer is now unblocked and
  explicitly named: `stampRoot` -> `otsclient.Stamp`. **An increment that does not make a root transit
  pending -> Bitcoin-confirmed end-to-end (or instead detours to polish) while §5 + the `.ots` route +
  WASM stay open and reachable is drift — the next step must close the bar, not de-risk it further.**

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since last assessment. The
`b03c1ca..HEAD` diff touched ONLY `internal/otsclient/client.go` (+test) and `.claude/*`; all M1
sources (`follower`, `store/sqlite.go`, `schema.sql`, the consistency-trigger code) are byte-unchanged.
All M1 Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested
end-to-end with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (carried forward)**: `cmd/{iscc-monitor,notecheck}`; **21 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` v0.4.0 with a real
  production caller: `opentimestamps` is imported by `internal/ots` (parse/classify) AND
  `internal/otsclient` (calendar transport: `UpgradeSequence`/`Stamp`, now routed through
  `safeUpgrade`); `cmd/iscc-monitor/main.go` constructs `otsclient.NewUpgrader()` and runs it in
  `runOTSLoop`.

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
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` a gated placeholder
  at `handler.go:286`, never set true, re-verified; §5 will read OTS rows via `OTSForRoot` + classify
  via `ots.Confirmed`, but no confirmed anchor data exists yet — pending rows carry empty `OTSBytes`);
  the separate **Bitcoin-anchor vs comparison-anchor** panels; the **mandatory M-UI exit visual-pass +
  human sign-off** (ADR-0012; the agent-browser tooling is now merged to `develop` via `5fb2454`, but
  the pass + sign-off has not been run).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — six sub-steps landed (store seam + stamp pass + upgrade-loop core + classifier +
real Upgrader/ticker + safeUpgrade hardening); drift line closed and now hang/crash-proof, but NOT
end-to-end. WASM — not started.**
- **OTS:** (1) `internal/store/ots.go` — typed `ots`-table CRUD. (2) `stampRoot`
  (`internal/follower/follower.go:410`) writes the accepted root as a `pending` row (idempotent via
  `UNIQUE(hub, tree_size, root)`) — but with **EMPTY `OTSBytes`** (no calendar submit yet, re-verified
  at `follower.go:410-421`). (3) `OTSTick` (`internal/follower/otsloop.go`) — the upgrade-loop control
  core over the injected `Upgrader func` seam + capped-exponential backoff. (4)
  `internal/ots.Confirmed(otsBytes)` — the pure offline classifier over `nbd-wtf/opentimestamps` v0.4.0,
  fail-closed, with the `>math.MaxInt64` height guard. (5) `internal/otsclient` — the real
  `follower.Upgrader` closure (`NewUpgrader`/`buildUpgrader` wrapping `opentimestamps.UpgradeSequence`
  -> `ots.Confirmed`) + a `Stamp` helper + `recoverRead` panic-guard; wired into
  `cmd/iscc-monitor/main.go` as a background `runOTSLoop` goroutine, the first production caller of
  `OTSTick`/`ots.Confirmed`. (6) **NEW — `safeUpgrade`** (`internal/otsclient/client.go:157`): every
  calendar sequence-upgrade now runs under a per-request `context.WithTimeout(ctx, 30s)` (one
  `defer cancel()` per call, no defer-in-loop leak) + a `recover()` that re-surfaces a library panic as
  a fail-closed error. `review` confirmed PASS at HEAD; both fixes mutation-proven non-vacuous; Codex
  clean. This closed the last open `normal` otsclient issue.
  **Not yet built (the actual Verify-closer):** `stampRoot` does NOT call `otsclient.Stamp`, so pending
  rows have no `OTSBytes` and `OTSTick`'s upgrade is a no-op (the closure short-circuits at
  `recoverRead`; `upgrade()` is unreachable in production today). Also absent: the `.ots` HTTP route
  (re-verified) and certificate §5. The milestone Verify is still 1/1 open: **no root has actually been
  anchored to Bitcoin.** With the upgrade path now crash/hang-proof, the next step (`stampRoot` ->
  `otsclient.Stamp`) can safely make `upgrade()` go live.
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The verifier app does not exist.

## Quality gates
**Status**: **GREEN.** HEAD (`a7555ed`) is itself the `cid(review)` PASS commit; the branch is in sync
with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 23 packages incl.
  `internal/otsclient`; `gofmt -l .` clean; `go mod verify` + `go mod tidy -diff` clean) at HEAD.
- The latest **`review` verdict is PASS at HEAD `a7555ed`** (otsclient upgrade-path hardening).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **CI at HEAD `a7555ed` = `success`** (run
  27926003590). No CI failure.
- **Open issues: 0 `critical`, 3 `normal`, 7 `low`.** The 3 `normal`: (a) Hub-List `hubDomain`
  `ForceQuery` fail-open; (b) §4 AND bundle `did:web:` + raw domain mis-render a `host:port` hub's DID
  (2 surfaces, fix together); (c) §6 RECORD HISTORY omits the per-record `· at` timestamp. The prior
  otsclient panic/timeout `normal` was resolved + deleted this iteration. The 7 `low` are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. Gate green at HEAD, CI green — no CI hygiene
needed.** The OTS upgrade path is now crash/hang-proof; the OTS Verify criterion is still 1/1 open and
the next step is exactly what closes it:

1. **`stampRoot` calendar-submit -> first end-to-end transit (the Verify-closer).** Wire
   `otsclient.Stamp` into `follower.stampRoot` (`follower.go:410-421`) so a stamped pending row carries
   real `OTSBytes`/`CalendarURLs`, letting `OTSTick` actually upgrade pending -> Bitcoin-confirmed
   (`upgrade()` is now safe to make reachable in production — the panic/hang hazards are closed). Then
   the `.ots` HTTP route (reads `OTSForRoot`) and certificate **§5 BITCOIN ANCHOR** (`HasClause5`,
   reading `OTSForRoot` + classifying via `ots.Confirmed`) — closing the last open certificate clause +
   the Bitcoin-anchor panel + the OTS milestone Verify. **This is the increment that finally closes a
   Verify criterion; a step that does not advance the end-to-end transit is drift.**
2. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` -> `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** the agent-browser tooling is now on `develop` (`5fb2454` merged); run
   the agent-browser visual pass on every SSR surface, file deviations as issues, and obtain human
   sign-off — M-UI does not reach DONE until this clears.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched.
