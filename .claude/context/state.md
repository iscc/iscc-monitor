<!-- assessed-at: ba1f4e5e063bf079019818f4f3ed89707b6a8b2b -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI certificate tail closing — §5 BITCOIN ANCHOR now observable; WASM not started

The last open certificate clause — **§5 BITCOIN ANCHOR** — is now rendered: `HasClause5` is set true
(`internal/certificate/handler.go:845`) from the mirrored OTS row of §2's accepted root
(`store.OTSForRoot`, classified via `ots.Confirmed`), showing `block <height>` when Bitcoin-confirmed
and the honest "pending — awaiting Bitcoin confirmation" otherwise, omitting §5 cleanly for an
un-anchored root. M1/M2/M3 are met; M-UI and OTS are the active milestones; WASM is not started.

This is the `cid(review)` PASS_WITH_NOTES commit for the §5 clause (HEAD `ba1f4e5`). Incremental
review from `7940959`: the `79409594..HEAD` prod diff touched ONLY `internal/certificate/`
(`handler.go`, `cert.html`, `handler_test.go`, two `.ots` fixtures) — no M1/M2/M3/follower/store/
proofserve/dossier/dashboard production source — so those sections are carried forward verified. **HEAD
is in sync with `origin/develop`** (0 ahead / 0 behind) and **CI at HEAD `ba1f4e5` = success** (run
27928057483).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met.** Met: five-status `HubStatusBadge` (`internal/badge`); DS v2 shared shell
    (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/` realm-index grid; `/<domain>/log/` browser;
    hub dossier (`GET /<domain>`, `internal/dossier`); frozen Exhibit; paginated record list;
    single-record page; ISCC-IDv1 decoder (`internal/index.Decode`); `(realm, hub_id) → domain`
    Hub-List resolver (`internal/registry`); certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION
    PROOF + §4 SIGNING KEY + §5 BITCOIN ANCHOR + §6 RECORD HISTORY** (all six numbered clauses now
    render); the proof-bundle endpoint (`.bundle`) + download link. **Open:** the separate
    **Bitcoin-anchor vs comparison-anchor** panels on the certificate (only the Bitcoin side, §5,
    exists — no distinct comparison-anchor element; re-verified absent this iteration: no
    `comparison` token anywhere in `internal/certificate/`); and the mandatory **M-UI exit
    visual-pass + human sign-off** (ADR-0012; agent-browser tooling on `develop` via `5fb2454`, the
    review step now runs per-surface screenshots, but the full M-UI exit pass + human sign-off is not
    run).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof` package, no `syscall/js` in any
    source file under `internal/`/`cmd/`; both re-verified empty this iteration).
  - **OTS anchoring: 1/1 open.** The observable HTTP halves are CLOSED: the served `.ots`
    (`GET /<domain>/log/checkpoint.ots`) parses with the standard reader (prior iteration), and §5
    now renders the anchor state on the certificate (this iteration). What remains for the full
    Verify: a root that actually transits to **Bitcoin-confirmed** — offline-unprovable; the upgrade
    is exercised only against an injected Upgrader in tests, never the live chain (needs a live
    calendar + real BTC confirmation). Still 1/1 open.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 foundational·plumbing·hardening.** Recent
  arc: ADR-0011 stack bump → OTS store CRUD → OTS stamp pass → OTS upgrade-loop core → OTS adapter →
  real Upgrader + main.go wiring → otsclient hardening (safeUpgrade) → OTS stamp seam → **`.ots` serve
  route** → **certificate §5**. **DRIFT WATCH (clear):** the OTS work has now surfaced TWO observable
  Verify-relevant clauses in a row (`.ots` route, then §5) — exactly what the prior assessment
  demanded; no internal-seam drift this iteration. With all six certificate clauses landed, the
  next observable Verify-closers are the **comparison-anchor panel** (M-UI) and the **WASM verifier**
  (1/1 untouched for the whole arc — the longest-standing open Verify criterion).

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since last assessment. The
`79409594..HEAD` prod diff is confined to `internal/certificate/`; all M1 sources (`follower.go`,
`store/sqlite.go`, `schema.sql`, the consistency-trigger code) are byte-unchanged. All M1 Verify
criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end
with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; 21 internal packages — `badge,
  certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  v0.5.0 (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` v0.4.0 with real
  production callers: `internal/ots` (parse/classify, now also read by `internal/certificate` §5) and
  `internal/otsclient` (calendar transport).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard;
`GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/checkpoint.ots`
DO carry strong ETag + `no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — §5 landed this iteration; all six certificate clauses now render.** §1
SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING KEY, **§5 BITCOIN ANCHOR (new)**, §6 RECORD
HISTORY all sound. Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit,
record list, single-record page, ISCC-IDv1 decoder, Hub-List resolver, proof-bundle endpoint +
download link all built and verified (carried forward).
- **§5 detail (this iteration):** `internal/certificate/handler.go:839-850` reads
  `st.OTSForRoot(ctx, hub.HubID, hub.LastSize, root)` for §2's accepted root and classifies via
  `ots.Confirmed`: confirmed → `HasClause5=true` + `BTCHeight` + RFC-3339 `BTCConfirmedAt`; pending →
  `HasClause5=true` + honest "awaiting Bitcoin confirmation"; un-anchored / empty-bytes sentinel /
  unparseable proof → §5 omitted (never 5xx; only a genuine `OTSForRoot` DB fault is a 500). Rendered
  at `cert.html:393-399`. Review-confirmed: 4 new §5 tests, 4 independent mutations each fail a test.
- **Still open on the M-UI Verify bar:** the separate **Bitcoin-anchor vs comparison-anchor** panels
  on the certificate — only the §5 Bitcoin side exists; no distinct comparison-anchor element
  (re-verified: no `comparison` token in `internal/certificate/`); and the **mandatory M-UI exit
  visual-pass + human sign-off** (ADR-0012; agent-browser tooling on `develop`, full exit pass + human
  sign-off not yet run).
- **Residual notes (filed `normal`/`low`, NOT fixed):**
  - **NEW `normal` (Codex P2, this iteration):** §5 does not bind the OTS proof's committed
    `File.Digest` to §2's accepted root — `ots.Confirmed` classifies attestations only, never compares
    the proof digest against the root. NOT exploitable: the production write path always stamps the
    row's own root; only a buggy `RecordOTS` or a fixture-mismatched test row produces a mismatch. Fix
    when §5/`ots.Confirmed` is next touched (`bytes.Equal(file.Digest, root)` before `HasClause5`).
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID. Not exploitable on the clean testnet realm. Fix BOTH sites together (`%3A`-encode).
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard. Not exploitable (resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows — `store.RecordRow` carries no
    timestamp column; a store/projection schema change, larger than the clause. Cosmetic.

## WASM verifier · OTS anchoring
**Status**: **OTS — both observable HTTP halves now landed (`.ots` route + certificate §5); only a
real Bitcoin confirmation remains (offline-unprovable). WASM — not started.**
- **OTS:** the `.ots` serve route (`GET /<domain>/log/checkpoint.ots`, `internal/proofserve`) and the
  certificate §5 anchor clause (this iteration) are both live and tested. The store layer
  (`internal/store/ots.go` typed CRUD), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the real
  calendar transport (`internal/otsclient`) are all wired in `main.go`'s `runOTSLoop`. **The actual
  Verify-closer not yet built:** a root that actually reaches **Bitcoin-confirmed** — the upgrade is
  exercised only against an injected Upgrader in tests, not the live chain; depends on a live calendar
  + real BTC confirmation, not provable in the offline suite. The milestone Verify is still 1/1 open.
  **Open `normal` defect (filed, NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout — the
  symmetric guards the upgrade path got via `safeUpgrade`. A malformed calendar response crashes the
  monitor; a stalled one hangs the OTS goroutine. The next stamp-path touch should add `safeStamp`.
- **WASM:** not started. No `internal/proof` package; no `syscall/js` in any source file (re-verified
  empty). The `monitor.iscc.codes` Independent Verification verifier app does not exist. This is the
  longest-standing open Verify criterion (untouched across the whole recent OTS arc).

## Quality gates
**Status**: **GREEN.** HEAD (`ba1f4e5`) is itself the `cid(review)` PASS_WITH_NOTES commit; the branch
is in sync with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 23 packages; `gofmt -l .` clean;
  `go mod tidy -diff` clean) at HEAD.
- The latest **`review` verdict is PASS_WITH_NOTES at HEAD `ba1f4e5`** (certificate §5; 4 mutations
  proven; dep-clean; WASM-purity guard clean; one Codex P2 digest-binding gap filed `normal`,
  unreachable in the production write path — does not block).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **CI at HEAD `ba1f4e5` = `success`** (run
  27928057483). No CI failure.
- **Open issues: 0 `critical`, 5 `normal`, 8 `low`.** The 5 `normal`: (a) §5 does not bind the OTS
  proof digest to §2's root (new, unreachable in prod); (b) the production OTS `Stamp` path has neither
  panic-recover nor timeout (crash/hang on a malformed/stalled calendar, a live path); (c) Hub-List
  `hubDomain` `ForceQuery` fail-open; (d) §4 AND bundle `did:web:` + raw domain mis-render a
  `host:port` hub's DID (2 surfaces, fix together); (e) §6 omits the per-record `· at` timestamp.

## Next Milestone
**M1/M2/M3 met; M-UI + OTS are the active milestones. Gate green at HEAD, CI green — no CI hygiene
needed.** All six certificate clauses now render. The next observable Verify-closers:

1. **Certificate comparison-anchor panel (M-UI Verify, observable).** target.md names the
   **Bitcoin-anchor** AND a separate **comparison-anchor** panel as distinct, distinctly-labelled
   elements; only the §5 Bitcoin side exists. Add the comparison-anchor panel (the monitor's
   "independently-signed record of what the hub showed it" affordance) — "anchoring" copy stays
   Bitcoin-only. This is the last open *observable* M-UI certificate element. Fold in the §5
   digest-binding fix (new `normal`) and the `host:port` DID `%3A`-encode (both §4 + bundle) when
   `handler.go` is next touched.
2. **WASM verifier** (1/1 Verify open, the longest-standing open criterion) — `internal/proof/verify`
   → `GOOS=js GOARCH=wasm` plus the `monitor.iscc.codes` Independent Verification app.
3. **M-UI exit gate (ADR-0012):** run the agent-browser visual pass on every SSR surface, file
   deviations as issues, and obtain human sign-off — M-UI does not reach DONE until this clears.

Fold in the `safeStamp` guard (the live-path crash/hang `normal` defect) when the stamp path is next
touched. The OTS "upgrades to Bitcoin-confirmed" half stays offline-unprovable (live calendar + chain).
