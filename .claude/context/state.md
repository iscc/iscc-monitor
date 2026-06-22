<!-- assessed-at: a5d5d27ef8c75924ac380659341986cbf2b40610 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI certificate complete — comparison-anchor panel landed; WASM not started, OTS live-chain half open

The realm-wide Certificate of Inclusion now renders all six numbered clauses (§1–§6) **and** both
anchor panels — the §5 BITCOIN ANCHOR and the distinct, separately-labelled COMPARISON ANCHOR — so the
last open *observable* M-UI certificate Verify element is closed. M1/M2/M3 are met; M-UI is near exit
(only the WASM tier-2 result, dossier anchor regions, and the mandatory exit visual-pass + human
sign-off remain); the WASM verifier is not started and OTS keeps its offline-unprovable live-chain
half open.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met (certificate-observable surface now complete).** Met (verified): five-status
    `HubStatusBadge` (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted webfonts,
    CDN-free); `/` realm-index grid; `/<domain>/log/` browser; hub dossier (`internal/dossier`);
    frozen Exhibit; paginated record list; single-record page; ISCC-IDv1 decoder (`internal/index.Decode`);
    `(realm, hub_id) → domain` Hub-List resolver (`internal/registry`); certificate **§1 SUBJECT + §2
    CHECKPOINT + §3 INCLUSION PROOF + §4 SIGNING KEY + §5 BITCOIN ANCHOR + §6 RECORD HISTORY**, the
    proof-bundle endpoint (`.bundle`) + download link, **and the distinct COMPARISON ANCHOR panel
    (new this iteration** — `HasComparisonAnchor`, `internal/certificate/handler.go:920`, rendered at
    `cert.html:405`; verified separate from §5 with no Bitcoin/"anchoring" copy in the panel). **Open:**
    the **M-UI exit visual-pass + human sign-off** (ADR-0012; agent-browser tooling on `develop`, the
    review step now runs per-surface screenshots, but the full exit pass + human sign-off is not run);
    plus the dossier §4 Bitcoin-anchor region (named-region parity, not yet a separate filed gap).
  - **WASM verifier: 1/1 open** (not started — re-verified this iteration: no `internal/proof` package,
    no `syscall/js` in any `internal/`/`cmd/` source, no `GOOS=js` build target; the
    `monitor.iscc.codes` Independent Verification app does not exist).
  - **OTS anchoring: 1/1 open.** Both observable HTTP halves are CLOSED (the `.ots` serve route + the
    certificate §5 anchor render). What remains for the full Verify: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; the upgrade is exercised only against an injected
    Upgrader in tests, never the live chain (needs a live calendar + real BTC confirmation). Still 1/1.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 foundational·plumbing·hardening.** Recent
  arc: OTS upgrade-loop core → adapter → real Upgrader + main.go wiring → otsclient hardening → OTS
  stamp seam → **`.ots` serve route** → **certificate §5** → **certificate COMPARISON ANCHOR panel**.
  **DRIFT WATCH (clear):** the last THREE iterations each closed an observable Verify-relevant element
  (`.ots` route, §5, comparison anchor). No internal-seam drift. The next observable Verify-closer is
  the **WASM verifier** — the longest-standing open criterion, 1/1 untouched across the whole arc.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched since `ba1f4e5`. The
`ba1f4e5..HEAD` prod diff is confined to `internal/certificate/`; all M1 sources (`follower.go`,
`store/sqlite.go`, `schema.sql`, the consistency-trigger code) are byte-unchanged. All M1 Verify
criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end
with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present (carried forward)**: `cmd/{iscc-monitor,notecheck}`; 21 internal packages —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` with real production callers:
  `internal/ots` (parse/classify, read by `internal/certificate` §5) and `internal/otsclient`.

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
**Status**: **in progress — certificate observable surface complete this iteration.** All six numbered
clauses (§1 SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING KEY, §5 BITCOIN ANCHOR, §6 RECORD
HISTORY) plus BOTH anchor panels now render. Badge, DS shell, `/` index, `/<domain>/log/` browser, hub
dossier, frozen Exhibit, record list, single-record page, ISCC-IDv1 decoder, Hub-List resolver,
proof-bundle endpoint + download link all built and verified (carried forward).
- **Comparison-anchor detail (this iteration):** `internal/certificate/handler.go` sets
  `data.HasComparisonAnchor = true` (line 920) for any certifiable id with an accepted checkpoint,
  reframing §2's `(size, root)` as the monitor's independently-observed record bounded by the coverage
  window; rendered at `cert.html:405-…` with a distinct `COMPARISON ANCHOR` marker and NO
  Bitcoin/"anchoring" lexicon — verified separate from and decoupled from §5. Review-confirmed: 3 new
  tests (`TestCertificateComparisonAnchor`, `…IndependentOfOTS`, `…CoverageJustStarted`),
  mutation-proven non-vacuous, Codex clean, headless visual pass clean. Closes the last open
  *observable* M-UI certificate Verify element (the Bitcoin-anchor / comparison-anchor distinct panels).
- **Still open on the M-UI Verify bar:** the **mandatory M-UI exit visual-pass + human sign-off**
  (ADR-0012; agent-browser tooling present, per-surface screenshots run in review, but the full exit
  pass + human sign-off not yet executed); dossier §4 Bitcoin-anchor / comparison-anchor named regions
  for full parity.
- **Residual notes (filed `normal`/`low`, NOT fixed — fold in when the exact line is next edited):**
  - §5 does not bind the OTS proof's committed `File.Digest` to §2's accepted root (`normal`;
    unreachable in the production write path; fix `bytes.Equal(file.Digest, root)` before `HasClause5`).
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID (`normal`; `%3A`-encode both sites together; not exploitable on the clean testnet realm).
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard (`normal`; resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows (`normal`; `store.RecordRow` carries
    no timestamp column — a store/projection schema change, larger than the clause).

## WASM verifier · OTS anchoring
**Status**: **OTS — both observable HTTP halves landed (`.ots` route + certificate §5); only a real
Bitcoin confirmation remains (offline-unprovable). WASM — not started.**
- **OTS:** the `.ots` serve route (`GET /<domain>/log/checkpoint.ots`, `internal/proofserve`), the
  certificate §5 anchor clause, the store layer (`internal/store/ots.go` typed CRUD), the off-path
  stamp/upgrade loop (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier
  (`internal/ots.Confirmed`), and the real calendar transport (`internal/otsclient`) are all wired in
  `main.go`'s `runOTSLoop`. **The actual Verify-closer not yet built:** a root that actually reaches
  **Bitcoin-confirmed** — exercised only against an injected Upgrader in tests, not the live chain;
  depends on a live calendar + real BTC confirmation, not provable in the offline suite. Still 1/1 open.
  **Open `normal` defect (filed, NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout — the
  symmetric guards the upgrade path got via `safeUpgrade`. A malformed calendar response crashes the
  monitor; a stalled one hangs the OTS goroutine. The next stamp-path touch should add `safeStamp`.
- **WASM:** not started (re-verified this iteration). No `internal/proof` package; no `syscall/js` in
  any source file; no `GOOS=js` build target. The `monitor.iscc.codes` Independent Verification app
  does not exist. This is the longest-standing open Verify criterion (untouched across the OTS + cert arc).

## Quality gates
**Status**: **GREEN.** HEAD (`a5d5d27`) is itself the `cid(review)` PASS commit for the
comparison-anchor panel; the branch is in sync with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 23 packages; `gofmt -l .` clean;
  `go mod tidy -diff` clean) at HEAD — no new prod dependency added.
- The latest **`review` verdict is PASS at HEAD `a5d5d27`** (comparison-anchor panel; 3 tests,
  mutations proven non-vacuous; Codex clean; WASM-purity guard clean; visual pass clean; no new issue).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR, pinned to `go-version: "1.26"`. Remote `origin` =
  `github.com/iscc/iscc-monitor.git`, branch `develop`. **CI at HEAD `a5d5d27` = `success`** (run
  27928704092). No CI failure.
- **Open issues: 0 `critical`, 5 `normal`, 8 `low`.** The 5 `normal`: (a) §5 does not bind the OTS
  proof digest to §2's root (unreachable in prod); (b) the production OTS `Stamp` path has neither
  panic-recover nor timeout (crash/hang on a malformed/stalled calendar, a live path); (c) Hub-List
  `hubDomain` `ForceQuery` fail-open; (d) §4 AND bundle `did:web:` + raw domain mis-render a `host:port`
  hub's DID (2 surfaces, fix together); (e) §6 omits the per-record `· at` timestamp.

## Next Milestone
**M1/M2/M3 met; M-UI near exit + OTS are the active milestones. Gate green at HEAD, CI green — no CI
hygiene needed.** The certificate observable surface is now complete (all six clauses + both anchor
panels). The next observable Verify-closers:

1. **WASM verifier** (1/1 Verify open, the longest-standing open criterion) — `internal/proof/verify`
   → `GOOS=js GOARCH=wasm` lazy-loaded tier-2 result on the M-UI certificate/dossier, plus the
   standalone `monitor.iscc.codes` Independent Verification app (reproducible build + published hash +
   SRI pin). This is the only un-started v1 milestone with offline-provable Verify criteria.
2. **M-UI exit gate (ADR-0012):** run the agent-browser visual pass on every SSR surface, file
   deviations as issues, and obtain human sign-off — M-UI does not reach DONE until this clears. Fold in
   the dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.

Fold in the `safeStamp` guard (the live-path crash/hang `normal` defect) when the stamp path is next
touched, the §5 digest-binding fix and the `host:port` DID `%3A`-encode (§4 + bundle) when
`handler.go` is next touched. The OTS "upgrades to Bitcoin-confirmed" half stays offline-unprovable
(live calendar + chain).
