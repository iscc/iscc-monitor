<!-- assessed-at: 79409594c63e09b65c47c833e8588350d7939232 -->

# Project State

## Status: IN_PROGRESS

## Phase: OTS milestone + M-UI tail — first observable OTS surface landed

The mirrored OpenTimestamps proof is now served at `GET /<domain>/log/checkpoint.ots`
(`internal/proofserve` `serveOTS`/`writeOTS`, mounted in `main.go`), closing the **observable HTTP
half** of the OTS Verify criterion: the served `.ots` parses as a valid OpenTimestamps proof. The
remaining OTS bar is certificate **§5 BITCOIN ANCHOR** (`HasClause5` still never set true) plus a root
that actually transits to **Bitcoin-confirmed** (offline-unprovable; needs a live calendar + chain
confirmation). M1/M2/M3 are met; M-UI and OTS are the active milestones; WASM is not started.

This is the `cid(review)` PASS commit for the `.ots` serve route (HEAD `7940959`). Incremental review
from `dfb7f37`: the `dfb7f37..HEAD` diff touched ONLY `internal/proofserve/handler.go` (+`ots_test.go`),
`cmd/iscc-monitor/main.go` (+`main_test.go`), `CLAUDE.md`, and `.claude/*` context — no
M1/M2/M3/M-UI/certificate/follower/store production source, so those sections are carried forward
verified. **HEAD is in sync with `origin/develop`** (0 ahead / 0 behind) and **CI at HEAD `7940959` =
success** (run 27927375066).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met.** Met: five-status `HubStatusBadge` (`internal/badge`); DS v2 shared shell
    (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/` realm-index grid; `/<domain>/log/` browser;
    hub dossier (`GET /<domain>`, `internal/dossier`); frozen Exhibit; paginated record list;
    single-record page; ISCC-IDv1 decoder (`internal/index.Decode`); `(realm, hub_id) → domain`
    Hub-List resolver (`internal/registry`); certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION
    PROOF + §4 SIGNING KEY + §6 RECORD HISTORY**; the proof-bundle endpoint (`.bundle`) + download
    link. **Open:** certificate **§5 BITCOIN ANCHOR** (`HasClause5` declared at
    `internal/certificate/handler.go:286`, read only by `cert.html:375`, **never assigned `true`** —
    re-verified this iteration; blocked on confirmed OTS rows existing); the separate
    **Bitcoin-anchor vs comparison-anchor** panels; and the mandatory **M-UI exit visual-pass + human
    sign-off** (ADR-0012; agent-browser tooling on `develop` via `5fb2454`, pass + sign-off not run).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file under `internal/`/`cmd/`; both re-verified empty this iteration).
  - **OTS anchoring: 1/1 open — observable `.ots` HTTP half now CLOSED.** The served `.ots`
    (`GET /<domain>/log/checkpoint.ots`) serves the mirrored proof verbatim and parses with the
    standard OpenTimestamps reader (review-confirmed). What remains for the full Verify: certificate
    **§5** (`HasClause5` never true) and a root that actually transits to **Bitcoin-confirmed** (the
    upgrade is only exercised against an injected Upgrader in tests, not the live chain — depends on a
    live calendar + real BTC confirmation, not provable in the offline suite). Still 1/1 open.
- **Last ~10 iterations: ~4 milestone-Verify-advancing / ~6 foundational·plumbing·hardening.**
  Recent arc: proof-bundle endpoint → ADR-0011 stack bump → OTS store CRUD → OTS stamp pass → OTS
  upgrade-loop core → OTS adapter → real Upgrader + main.go wiring → otsclient hardening (safeUpgrade)
  → OTS stamp seam (Stamper in OTSTick) → **`.ots` serve route**. **DRIFT WATCH (clearing):** the long
  OTS sub-step streak finally produced its first *observable* surface this iteration (`.ots` HTTP
  route), which is exactly the Verify-closer the prior assessment demanded — not another internal seam.
  The OTS Verify is still 1/1 open, but its closure is now bounded by certificate §5 (a small,
  HTTP-seam-testable clause reading `OTSForRoot` + `ots.Confirmed`) and an offline-unprovable live-chain
  confirmation. **The next increment should land §5 (observable) — anything that adds another internal
  OTS seam without surfacing §5 is drift.**

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since last assessment. The
`dfb7f37..HEAD` diff touched ONLY `internal/proofserve/handler.go` (+`ots_test.go`),
`cmd/iscc-monitor/main.go` (+`main_test.go`), `CLAUDE.md`, and `.claude/*`; all M1 sources
(`follower.go`, `store/sqlite.go`, `schema.sql`, the consistency-trigger code) are byte-unchanged. All
M1 Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested
end-to-end with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **21 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` v0.4.0 with real
  production callers: `internal/ots` (parse/classify) and `internal/otsclient` (calendar transport).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard;
`GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and the new
`/checkpoint.ots` DO carry strong ETag + `no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — carried forward (no M-UI source in the diff).** §1 SUBJECT, §2 CHECKPOINT,
§3 INCLUSION PROOF, §4 SIGNING KEY, §6 RECORD HISTORY all sound. Badge, DS shell, `/` index,
`/<domain>/log/` browser, hub dossier, frozen Exhibit, record list, single-record page, ISCC-IDv1
decoder, Hub-List resolver, proof-bundle endpoint + download link all built and verified.
- **Still open on the M-UI Verify bar:** clause **§5 BITCOIN ANCHOR** (`HasClause5` declared at
  `handler.go:286`, read only by `cert.html:375`, never set `true`, re-verified; §5 will read OTS rows
  via `OTSForRoot` + classify via `ots.Confirmed` and can now link the live `.ots` route — unblocked,
  but no Bitcoin-confirmed row exists yet); the separate **Bitcoin-anchor vs comparison-anchor**
  panels; the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012; agent-browser tooling
  merged to `develop` via `5fb2454`, pass + sign-off not yet run).
- **Residual notes (filed `normal`/`low`, NOT fixed):**
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode the
    port) when handler.go is next touched.
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — observable `.ots` HTTP serve now landed; certificate §5 + a real Bitcoin
confirmation are the remaining Verify bar. WASM — not started.**
- **OTS:** (1) **`.ots` route (NEW this iteration):** `GET /<domain>/log/checkpoint.ots`
  (`internal/proofserve` `serveOTS`/`writeOTS`, mounted `main.go:397`) resolves the accepted
  `(size, root)`, reads `store.OTSForRoot`, and serves `OTSBytes` verbatim as
  `application/octet-stream` with a strong content ETag + `If-None-Match`→304 + `Cache-Control:
  no-cache`; an un-anchored root (OTSForRoot miss OR the empty-OTSBytes sentinel) → 404, never 5xx.
  Review-confirmed the served body parses as a valid `.ots` and is mutation-proven (4 reverts each fail
  a test). The production proofserve closure stays off `internal/ots`/`internal/otsclient`. (2)
  `internal/store/ots.go` — typed `ots`-table CRUD incl. `MarkOTSStamped`/`MarkOTSUpgraded`. (3)
  `stampRoot` writes the accepted root as a `pending` row with the empty `OTSBytes` sentinel
  (deliberate — no synchronous calendar round-trip on the poll path). (4) `OTSTick`
  (`internal/follower/otsloop.go`) stamps empty rows via the injected `Stamper`, persists via
  `MarkOTSStamped`, then upgrades on a later tick. (5) `internal/ots.Confirmed` — the offline
  fail-closed classifier with the `>math.MaxInt64` height guard. (6) `internal/otsclient` — real
  Upgrader (`safeUpgrade`) + `Stamp`; `main.go` `runOTSLoop` wires both.
  **Not yet built (the actual Verify-closers):** **certificate §5** (`HasClause5` never true) and a
  root that actually reaches **Bitcoin-confirmed** (upgrade only exercised against an injected Upgrader
  in tests, not the live chain). The milestone Verify is still 1/1 open.
  **Open `normal` defect (filed, NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`, live via `stampFunc`) has NEITHER a panic-recover NOR a
  per-request timeout — the symmetric guards the upgrade path got via `safeUpgrade`. A malformed
  calendar response crashes the monitor; a stalled one hangs the OTS goroutine. The next stamp-path
  touch should add a `safeStamp` wrapper.
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The verifier app does not exist.

## Quality gates
**Status**: **GREEN.** HEAD (`7940959`) is itself the `cid(review)` PASS commit; the branch is in sync
with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 23 packages; `gofmt -l .` clean;
  `go mod tidy -diff` clean) at HEAD.
- The latest **`review` verdict is PASS at HEAD `7940959`** (`.ots` serve route; 4 mutations proven;
  dep-clean; Codex clean; one behavior-neutral doc-fix applied).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **CI at HEAD `7940959` = `success`** (run
  27927375066). No CI failure.
- **Open issues: 0 `critical`, 3 `normal`, 6 `low`.** The 3 `normal`: (a) the production OTS `Stamp`
  path has neither panic-recover nor timeout (crash/hang on a malformed/stalled calendar, a live path);
  (b) Hub-List `hubDomain` `ForceQuery` fail-open; (c) §4 AND bundle `did:web:` + raw domain mis-render
  a `host:port` hub's DID (2 surfaces, fix together). The §6-timestamp item is also filed `normal` in
  issues.md (so 4 if counted there); the nil-Stamper fall-through is `low`.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. Gate green at HEAD, CI green — no CI hygiene
needed.** The `.ots` HTTP serve closed the observable OTS half this iteration; the next step is the
remaining **observable** OTS clause:

1. **Certificate §5 BITCOIN ANCHOR (the next Verify-closer).** Light up `HasClause5` in
   `internal/certificate/handler.go` (currently declared but never set true) by reading `OTSForRoot` +
   classifying via `ots.Confirmed`, linking the new `.ots` route, plus the separate Bitcoin-anchor
   panel. This is the last open certificate clause. **A step that adds another internal OTS seam
   instead of surfacing §5 is drift.** Fold in the `safeStamp` guard (the live-path crash/hang
   `normal` defect) when the stamp path is next touched. The "upgrades to Bitcoin-confirmed" half stays
   offline-unprovable (live calendar + chain confirmation).
2. **WASM verifier** (1/1 Verify open) — `internal/proof/verify` → `GOOS=js GOARCH=wasm` plus the
   `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** run the agent-browser visual pass (tooling on `develop` via
   `5fb2454`) on every SSR surface, file deviations as issues, and obtain human sign-off — M-UI does
   not reach DONE until this clears.

Fold in the deferred `host:port` DID-encoding fix (both §4 and bundle sites) when handler.go is next
touched.
