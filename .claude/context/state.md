<!-- assessed-at: dfb7f377cae07ba33e61b40dfacc6bf0e750a2df -->

# Project State

## Status: IN_PROGRESS

## Phase: OTS milestone + M-UI tail. The OTS stamp→upgrade transit is now wired end-to-end **in code**:
`OTSTick` calls an injected `Stamper` for any not-yet-stamped pending row (empty `OTSBytes` sentinel),
persists the calendar proof via `store.MarkOTSStamped`, then upgrades on a later tick; `main.go`'s
`runOTSLoop` wires the real `otsclient.Stamp` closure (`stampFunc()` → `DefaultCalendarURL`). This
**closes the long-standing drift line** (the prior "stampRoot writes EMPTY OTSBytes, OTSTick is a
structural no-op" is now stale). But the OTS milestone **Verify** ("a stamped root upgrades to
Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client") is **still 1/1
open**: there is no `.ots` HTTP route, certificate §5 is still a placeholder, and no root has actually
reached Bitcoin (the upgrade depends on a live calendar + real Bitcoin confirmation, not provable in
the offline suite). Remaining v1 work: the `.ots` route, certificate §5, the WASM verifier, the
anchor panels, and the mandatory M-UI visual-exit gate.

This is the `cid(review)` PASS_WITH_NOTES commit for the OTS stamp seam (HEAD `dfb7f37`). Incremental
review from `a7555ed`: the `a7555ed..HEAD` diff touched ONLY `internal/follower/otsloop.go` (+test),
`cmd/iscc-monitor/main.go`, `internal/store/ots.go` (+test, the new `MarkOTSStamped` mutator), and
`.claude/*` context — no M1/M2/M3/M-UI production source, so those sections are carried forward
verified. **HEAD is in sync with `origin/develop`** (0 ahead / 0 behind) and **CI at HEAD `dfb7f37` =
success** (run 27926658875).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met.** Met: five-status `HubStatusBadge` (`internal/badge`); DS v2 shared shell
    (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/` realm-index grid; `/<domain>/log/` browser;
    hub dossier (`GET /<domain>`, `internal/dossier`); frozen Exhibit; paginated record list;
    single-record page; ISCC-IDv1 decoder (`internal/index.Decode`); `(realm, hub_id) → domain`
    Hub-List resolver (`internal/registry`); certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION
    PROOF + §4 SIGNING KEY + §6 RECORD HISTORY**; the proof-bundle endpoint (`.bundle`) + download
    link. **Open:** certificate **§5 BITCOIN ANCHOR** (`HasClause5` still a gated placeholder, never set
    true — re-verified at `internal/certificate/handler.go:286`; blocked on confirmed OTS rows
    existing); the separate **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI
    exit visual-pass + human sign-off** (ADR-0012; the agent-browser tooling is on `develop` via the
    merged `5fb2454`, but the pass + sign-off has not run).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file under `internal/`/`cmd/`; both re-verified empty this iteration).
  - **OTS anchoring: 1/1 open — drift line now CLOSED (stamp→upgrade wired end-to-end in code), but
    still not Verify-met.** What exists and is tested: (1) `ots`-table CRUD seam (`internal/store/ots.go`,
    now incl. `MarkOTSStamped`); (2) the poll-path pending-row writer (`stampRoot` writes the empty
    sentinel — deliberately, NOT a calendar round-trip, so polling never blocks); (3) the
    stamp-then-upgrade control core (`OTSTick`, `internal/follower/otsloop.go`) — now stamps empty rows
    via the `Stamper` seam before upgrading; (4) the `internal/ots` classifier (`Confirmed`, with the
    `>math.MaxInt64` height guard); (5) the real `internal/otsclient` Upgrader + `Stamp` + `safeUpgrade`
    + the `runOTSLoop` goroutine in `main.go`, now wiring the real `stampFunc()`. The milestone Verify
    still needs: the **`.ots` HTTP route** (re-verified absent — no `.ots`/`/ots` handler in
    `proofserve`/`tilesserve`/`cmd`), **certificate §5** (`HasClause5` never true), and a root that
    actually transits to **Bitcoin-confirmed** (depends on a live calendar + real BTC confirmation;
    `OTSTick`'s upgrade is exercised in tests via an injected Upgrader, not against the chain). Still
    1/1 open.
- **Last ~10 iterations: ~3–4 milestone-Verify-advancing / ~6–7 foundational·plumbing·hardening.**
  Recent arc: proof-bundle endpoint → ADR-0011 stack bump → OTS store CRUD → OTS stamp pass → OTS
  upgrade-loop core → OTS adapter → real Upgrader + main.go wiring → otsclient upgrade-path hardening
  (safeUpgrade) → **OTS stamp seam (Stamper in OTSTick + stampFunc wiring + MarkOTSStamped)**.
  **DRIFT WATCH (easing but not clear):** this increment finally made the stamp→upgrade transit live
  in code (the Stamper closes the wiring the criterion's "a stamped root upgrades" half needs), so the
  ~7-step OTS sub-step streak is no longer pure de-risk — it advanced the path materially. But the OTS
  **Verify** is still 1/1 open because its observable bar (served `.ots` verifying with the standard
  `ots` client, plus §5) is not yet served. **The next increment must close the served-`.ots`/§5 bar,
  not add another internal OTS seam.** The single best next step is now the `.ots` route + certificate
  §5 reading `OTSForRoot` + `ots.Confirmed` — the first observable, HTTP-seam-testable OTS surface.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since last assessment. The
`a7555ed..HEAD` diff touched ONLY `internal/follower/otsloop.go` (+test), `cmd/iscc-monitor/main.go`,
`internal/store/ots.go` (+test) and `.claude/*`; all M1 sources (`follower.go`, `store/sqlite.go`,
`schema.sql`, the consistency-trigger code) are byte-unchanged. All M1 Verify criteria remain
satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze +
alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **21 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` v0.4.0 with real
  production callers: imported by `internal/ots` (parse/classify) AND `internal/otsclient` (calendar
  transport: `Stamp`/`UpgradeSequence` via `safeUpgrade`); `cmd/iscc-monitor/main.go` constructs
  `otsclient.NewUpgrader()` + `stampFunc()` and runs both in `runOTSLoop`.

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
  via `ots.Confirmed` — now that real stamped rows can be written this is unblocked, but no
  Bitcoin-confirmed row exists yet); the separate **Bitcoin-anchor vs comparison-anchor** panels; the
  **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012; agent-browser tooling merged to
  `develop` via `5fb2454`, pass + sign-off not yet run).
- **Residual notes (filed `normal`/`low`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — stamp→upgrade transit now wired end-to-end in code (drift line closed); the served
`.ots` route + §5 + a real Bitcoin confirmation are the remaining Verify bar. WASM — not started.**
- **OTS:** (1) `internal/store/ots.go` — typed `ots`-table CRUD, now incl. `MarkOTSStamped` (persists
  `OTSBytes`/`CalendarURLs` on a previously-empty pending row; status-untouching). (2) `stampRoot`
  (`internal/follower/follower.go`) writes the accepted root as a `pending` row with the **empty
  `OTSBytes` sentinel** — deliberate (a synchronous calendar round-trip on the poll path would violate
  "OTS never blocks"); the calendar submit happens off the poll path. (3) `OTSTick`
  (`internal/follower/otsloop.go`) — for a not-yet-stamped row (`len(r.OTSBytes)==0`) it now calls the
  injected `Stamper`, persists via `MarkOTSStamped`, and continues; an already-stamped row is asked of
  the `Upgrader` (confirmed → `MarkOTSUpgraded`; else backed-off retry). (4) `internal/ots.Confirmed`
  — the pure offline classifier, fail-closed, with the `>math.MaxInt64` height guard. (5)
  `internal/otsclient` — the real Upgrader (`safeUpgrade`: per-request `WithTimeout(ctx,30s)` +
  panic-recover) and `Stamp`/`DefaultCalendarURL`; `cmd/iscc-monitor/main.go` `runOTSLoop` now wires
  BOTH the real `stampFunc()` Stamper and `otsclient.NewUpgrader()`. The transit is mutation-proven
  non-vacuous (review reverted the stamp branch + the `MarkOTSStamped` UPDATE; both made follower/store
  tests FAIL).
  **Not yet built (the actual Verify-closer):** the **`.ots` HTTP route** (re-verified absent — no
  `.ots`/`/ots` handler in `proofserve`/`tilesserve`/`cmd`) and **certificate §5**; and no root has
  actually reached **Bitcoin-confirmed** (the upgrade is only exercised against an injected Upgrader in
  tests, not the live chain). The milestone Verify is still 1/1 open: **the served `.ots` does not yet
  exist and cannot be verified with the standard `ots` client.**
  **NEW `normal` defect (filed, NOT fixed):** the production `Stamp` path (`internal/otsclient/client.go:121`,
  now live via `stampFunc`) has NEITHER a panic-recover NOR a per-request timeout — the symmetric guards
  the upgrade path got via `safeUpgrade`. A malformed calendar response crashes the monitor; a stalled
  one hangs the OTS goroutine (Codex P1/P2, reviewer-confirmed against `opentimestamps@v0.4.0`). The
  next stamp-path touch should add a `safeStamp` wrapper. (The `Stamp` docstring is also stale — it says
  "called by stampRoot in a follow-up sub-step; the follower does not call it yet", but `stampFunc` now
  wires it live — cosmetic.)
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The verifier app does not exist.

## Quality gates
**Status**: **GREEN.** HEAD (`dfb7f37`) is itself the `cid(review)` PASS_WITH_NOTES commit; the branch
is in sync with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 23 packages; `gofmt -l .` clean;
  `go mod tidy -diff` clean) at HEAD.
- The latest **`review` verdict is PASS_WITH_NOTES at HEAD `dfb7f37`** (OTS stamp seam; transit
  mutation-proven; 3 Codex-confirmed defects filed as issues, none blocking the increment's goal).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **CI at HEAD `dfb7f37` = `success`** (run
  27926658875). No CI failure.
- **Open issues: 0 `critical`, 3 `normal`, 6 `low`.** The 3 `normal`: (a) **NEW** — the production OTS
  `Stamp` path has neither panic-recover nor timeout (crash/hang on a malformed/stalled calendar, now a
  live path); (b) Hub-List `hubDomain` `ForceQuery` fail-open; (c) §4 AND bundle `did:web:` + raw
  domain mis-render a `host:port` hub's DID (2 surfaces, fix together). (The prior §6 timestamp item is
  also `normal` — making it effectively 4 `normal` if counted; issues.md lists it. The nil-Stamper
  fall-through is `low`.)

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. Gate green at HEAD, CI green — no CI hygiene
needed.** The OTS stamp→upgrade transit is now wired end-to-end in code; the OTS Verify criterion is
still 1/1 open and the next step is the first **observable** OTS surface:

1. **The `.ots` HTTP route + certificate §5 BITCOIN ANCHOR (the Verify-closer).** Serve the stamped/
   upgraded proof at a canonical `.ots` path (reads `OTSForRoot`) so the served `.ots` verifies with the
   standard `ots` client, and light up certificate **§5** (`HasClause5`, reading `OTSForRoot` +
   classifying via `ots.Confirmed`) + the Bitcoin-anchor panel. This closes the last open certificate
   clause and the OTS milestone Verify's observable half. **A step that adds another internal OTS seam
   instead of an observable HTTP surface is drift.** Fold in the `safeStamp` guard (the new `normal`
   crash/hang defect) when the stamp path is next touched.
2. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` → `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** run the agent-browser visual pass (tooling on `develop` via `5fb2454`)
   on every SSR surface, file deviations as issues, and obtain human sign-off — M-UI does not reach DONE
   until this clears.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched.
