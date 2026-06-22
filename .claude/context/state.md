<!-- assessed-at: b03c1cad858fb817aaf53e0e48f8dcc5657cc182 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI tail + OTS milestone. OTS now has its first production caller — a real calendar Upgrader
closure (`internal/otsclient`) driven by a background `runOTSLoop` goroutine in `main.go` — which closes
the long-standing "no production caller" drift watch-line. But the path is still NOT end-to-end: the
daily `stampRoot` writes pending rows with EMPTY `OTSBytes` (no calendar submit yet), so `OTSTick`'s
upgrade is a structural no-op and no root has actually reached Bitcoin. Remaining v1 work: the
`stampRoot` calendar-submit (which finally lets a root transit pending → confirmed), the `.ots` route,
certificate §5, the WASM verifier, the anchor panels, and the mandatory M-UI visual-exit gate.

This is the `cid(review)` PASS_WITH_NOTES commit for the real OTS Upgrader + ticker (HEAD `b03c1ca`).
Incremental review from `655d004`: the `655d004..HEAD` diff touched ONLY `internal/ots/ots.go` (+ test;
the `>math.MaxInt64` height guard), the new `internal/otsclient/` (the real `follower.Upgrader` closure
+ `Stamp` helper + 5 tests + 3 `.ots` fixtures), `cmd/iscc-monitor/main.go` (the `runOTSLoop`
goroutine), and `.claude/*` context. No M1/M2/M3/M-UI production source changed — those sections are
carried forward. **HEAD is in sync with `origin/develop`** (0 ahead / 0 behind) and **CI at HEAD
`b03c1ca` = success** (run 27925579420). NOTE: a separate `feat(visual-verify)` agent-browser commit
(ADR-0012, `5fb2454`) exists on another branch but is NOT in `develop`'s linear history; the M-UI exit
visual gate it tools is still pending on develop.

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
    ANCHOR** (`HasClause5` still hard-false — re-verified at `internal/certificate/handler.go:286`,
    blocked on confirmed OTS rows existing); the separate **Bitcoin-anchor vs comparison-anchor**
    panels; and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012 agent-browser; the
    tooling is not yet on `develop`).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file; both re-verified empty this iteration).
  - **OTS anchoring: 1/1 open — drift line closed, but still not end-to-end.** Five pieces now exist and
    are tested: (1) `ots`-table CRUD seam (`internal/store/ots.go`); (2) the daily stamp pass
    (`stampRoot` in PollHub); (3) the upgrade-loop control core (`OTSTick`); (4) the `internal/ots`
    classifier (`Confirmed`, now with the `>math.MaxInt64` height guard); (5) **the real
    `internal/otsclient` Upgrader + `Stamp` + the `runOTSLoop` goroutine in `main.go`** — the first
    production caller of `OTSTick`/`ots.Confirmed`. The milestone Verify ("a stamped root upgrades to
    Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client") still needs:
    **`stampRoot` to call `otsclient.Stamp`** so pending rows carry real `OTSBytes`/`CalendarURLs`
    (re-verified `stampRoot` at `follower.go:410-421` writes NO `OTSBytes` — calendar submit deferred),
    the `.ots` HTTP route (re-verified absent), and certificate §5. **No root has actually been stamped
    to Bitcoin; `OTSTick`'s upgrade is a structural no-op until `OTSBytes` is populated.** Still 1/1 open.
- **Last ~10 iterations: ~4 milestone-Verify-advancing / ~6 foundational·plumbing.** Recent arc: cert §6
  -> proof-bundle endpoint -> #ZgotmplZ fix -> ADR-0011 stack bump -> OTS store CRUD -> OTS stamp pass
  -> OTS upgrade-loop core -> OTS adapter -> **real Upgrader + main.go wiring**. **The drift watch-line
  the prior states sharpened — "another seam-only step that does not call a real calendar from `main.go`
  would be drift" — is now CLOSED:** this increment is the real calendar transport wired into a live
  `main.go` goroutine, the genuine production caller, not a sixth pure seam. But the OTS *Verify
  criterion remains open* — the very next step (`stampRoot` → `otsclient.Stamp`) is what finally makes a
  root transit pending → Bitcoin-confirmed end-to-end and closes the bar. A step that does not advance
  that end-to-end transit (or instead detours to polish) while §5 + the `.ots` route + WASM stay open
  and reachable would re-open the drift concern.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since last assessment. The
`655d004..HEAD` diff touched ONLY `internal/ots/ots.go` (+test), new `internal/otsclient/*`,
`cmd/iscc-monitor/main.go`, and `.claude/*`; all M1 sources (`follower`, `store/sqlite.go`,
`schema.sql`, the consistency-trigger code) are byte-unchanged. All M1 Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once +
restart survival; structured logs; `/metrics`.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **21 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proofserve, registry, store, tiles, tilesserve, web` (one new
  this iteration: `internal/otsclient`). Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` v0.4.0. **Now a
  production caller exists:** `opentimestamps` is imported by `internal/ots` (parse/classify) AND
  `internal/otsclient` (calendar transport: `UpgradeSequence`/`Stamp`); `cmd/iscc-monitor/main.go`
  constructs `otsclient.NewUpgrader()` and runs it in `runOTSLoop`.

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
  `handler.go:286`, re-verified; §5 will read OTS rows via `OTSForRoot` + classify via `ots.Confirmed`,
  but no confirmed anchor data exists yet — pending rows carry empty `OTSBytes`); the separate
  **Bitcoin-anchor vs comparison-anchor** panels; the **mandatory M-UI exit visual-pass + human
  sign-off** (ADR-0012; the agent-browser tooling is on a side branch, not yet merged to `develop`).
- **Residual notes (filed `normal`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — five sub-steps landed (store seam + stamp pass + upgrade-loop core + classifier +
real Upgrader/ticker); drift line closed but NOT end-to-end. WASM — not started.**
- **OTS:** (1) `internal/store/ots.go` — typed `ots`-table CRUD. (2) `stampRoot`
  (`internal/follower/follower.go:410`) writes the accepted root as a `pending` row (idempotent via
  `UNIQUE(hub, tree_size, root)`) — but with **EMPTY `OTSBytes`** (no calendar submit yet, re-verified).
  (3) `OTSTick` (`internal/follower/otsloop.go`) — the upgrade-loop control core over the injected
  `Upgrader func` seam + capped-exponential backoff. (4) `internal/ots.Confirmed(otsBytes)` — the pure
  offline classifier over `nbd-wtf/opentimestamps` v0.4.0, fail-closed, **now with the `>math.MaxInt64`
  height guard** (`ots.go:74`, the prior `normal` issue — resolved, mutation-proven per review). (5)
  **NEW — `internal/otsclient`**: the real `follower.Upgrader` closure (`NewUpgrader`/`buildUpgrader`
  wrapping `opentimestamps.UpgradeSequence` → `ots.Confirmed`) + a `Stamp` helper + `recoverRead`
  panic-guard; wired into `cmd/iscc-monitor/main.go` as a background `runOTSLoop` goroutine
  (`main.go:151`), the **first production caller** of `OTSTick`/`ots.Confirmed`. `review` confirmed
  PASS_WITH_NOTES at HEAD; otsclient tests (5) + ots overflow test mutation-proven non-vacuous;
  WASM-purity + store-leaf invariants held.
  **Not yet built (the actual Verify-closer):** `stampRoot` does NOT call `otsclient.Stamp`, so pending
  rows have no `OTSBytes` and `OTSTick`'s upgrade is a no-op (the closure fails at `recoverRead`,
  `upgrade()` unreachable today). Also absent: the `.ots` HTTP route (re-verified) and certificate §5.
  The milestone Verify is still 1/1 open: **no root has actually been anchored to Bitcoin.**
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The verifier app does not exist.

## Quality gates
**Status**: **GREEN.** HEAD (`b03c1ca`) is itself the `cid(review)` PASS_WITH_NOTES commit; the branch
is in sync with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 23 packages incl. new
  `internal/otsclient`; `gofmt -l .` clean; `go mod verify` + `go mod tidy -diff` clean) at HEAD.
- The latest **`review` PASS_WITH_NOTES verdict is at HEAD `b03c1ca`** (real OTS Upgrader + ticker).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **CI at HEAD `b03c1ca` = `success`** (run
  27925579420). No CI failure.
- **Open issues: 0 `critical`, 4 `normal`, 7 `low`.** The 4 `normal`: (a) NEW this iteration (combined,
  Codex-confirmed) — the `otsclient` Upgrader can crash the monitor on a library panic AND hang
  `OTSTick` on a stalled calendar (`internal/otsclient/client.go:86`); both **unreachable today**
  (pending rows have empty `OTSBytes`), must be fixed with/before the `stampRoot` calendar-submit step.
  (b) Hub-List `hubDomain` `ForceQuery` fail-open. (c) §4 AND bundle `did:web:` + raw domain mis-render
  a `host:port` hub's DID (2 surfaces). (d) §6 RECORD HISTORY omits the per-record `· at` timestamp. The
  7 `low` are loop-skipped. The prior height-overflow `normal` was resolved this iteration.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. Gate green at HEAD, CI green — no CI hygiene
needed.** The OTS drift line is closed, but the OTS Verify criterion is still open and the next step is
exactly what closes it:

1. **`stampRoot` calendar-submit → first end-to-end transit.** Wire `otsclient.Stamp` into
   `follower.stampRoot` (`follower.go:410-421`) so a stamped pending row carries real
   `OTSBytes`/`CalendarURLs`, letting `OTSTick` actually upgrade pending → Bitcoin-confirmed.
   **Fold in BOTH Codex-confirmed otsclient defects** (`internal/otsclient/client.go:86`) — wrap the
   upgrade body in a panic-recover and add a per-request `context.WithTimeout` — since this step is what
   first makes `upgrade()` reachable in production. Then the `.ots` HTTP route (reads `OTSForRoot`) and
   certificate **§5 BITCOIN ANCHOR** (`HasClause5`, reading `OTSForRoot` + classifying via
   `ots.Confirmed`) — closing the last open certificate clause + the Bitcoin-anchor panel + the OTS
   milestone Verify. This is the increment that finally closes a Verify criterion; a step that does not
   advance the end-to-end transit would re-open the drift concern.
2. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` -> `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** merge the agent-browser tooling to `develop`, then every SSR surface
   must pass the agent-browser visual pass with deviations filed and a human sign-off.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched.
