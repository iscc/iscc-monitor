<!-- assessed-at: c6c392ab5c46312738a13f2f6ff7ef6ff1119e43 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM verifier kicked off — pure `internal/proof/verify` core landed (GOOS=js-shareable); the `syscall/js` entrypoint + in-browser app not yet built

The shared, WASM-shareable RFC-6962 inclusion-verifier core (`internal/proof/verify`) now exists and
both production call sites (verify-for-me + certificate §3) route through it — the first WASM sub-step,
verified compiling under `GOOS=js GOARCH=wasm`. M1/M2/M3 are met; M-UI's certificate observable surface
is complete (only the exit visual-pass + human sign-off and dossier anchor regions remain). WASM is now
*started* (core only) and OTS keeps its offline-unprovable live-chain half open.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met (certificate-observable surface complete).** Met (verified, carried forward):
    five-status `HubStatusBadge` (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted
    webfonts, CDN-free); `/` realm-index grid; `/<domain>/log/` browser; hub dossier
    (`internal/dossier`); frozen Exhibit; paginated record list; single-record page; ISCC-IDv1 decoder
    (`internal/index.Decode`); `(realm, hub_id) → domain` Hub-List resolver (`internal/registry`);
    certificate **§1–§6** + the proof-bundle endpoint (`.bundle`) + the distinct COMPARISON ANCHOR
    panel. **Open:** the **M-UI exit visual-pass + human sign-off** (ADR-0012; agent-browser tooling on
    `develop`, review runs per-surface screenshots, but the full exit pass + human sign-off is not run);
    plus the dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.
  - **WASM verifier: 1/1 open — but no longer untouched.** The pure shareable core landed this iteration
    (`internal/proof/verify`, builds under `GOOS=js GOARCH=wasm`). Still open: no `syscall/js`/`cmd/wasm`
    entrypoint, no `GOOS=js` build target wired, no lazy tier-2 enhancement on the certificate/dossier,
    and the standalone `monitor.iscc.codes` Independent Verification app does not exist. The Verify
    ("identical vectors → identical verdicts WASM vs server; published artifact hash; split-view alert")
    is unblocked by the core but not yet met.
  - **OTS anchoring: 1/1 open.** Both observable HTTP halves are CLOSED (the `.ots` serve route + the
    certificate §5 anchor render). What remains: a root that actually transits to **Bitcoin-confirmed** —
    offline-unprovable; exercised only against an injected Upgrader in tests, never the live chain (needs
    a live calendar + real BTC confirmation). Still 1/1.
- **Last ~10 iterations: ~4 milestone-Verify-advancing / ~6 foundational·plumbing·hardening.** Recent
  arc: OTS adapter → real Upgrader + main.go wiring → otsclient hardening → OTS stamp seam →
  `.ots` serve route → certificate §5 → certificate COMPARISON ANCHOR panel → **`internal/proof/verify`
  shareable core (WASM sub-step)**. **DRIFT WATCH (clear):** the last FOUR iterations each closed or
  began an observable Verify-relevant element (`.ots` route, §5, comparison anchor, the WASM-shareable
  core). The next observable Verify-closer is the WASM **`syscall/js` entrypoint + tier-2 enhancement** —
  the natural continuation of the core just landed; this is the only un-started v1 milestone with
  offline-provable Verify criteria.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched by the `a5d5d27..HEAD` diff
(confined to `internal/{certificate,proofserve}` + the new `internal/proof/verify`). All M1 sources
(`follower.go`, `store/sqlite.go`, `schema.sql`, the consistency-trigger code) unchanged. All M1 Verify
criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end
with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; **22 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, web`
  (the new `proof` package holds `proof/verify`). Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` (production callers `internal/ots`
  + `internal/otsclient`). `transparency-dev/merkle` (`proof`, `rfc6962`) now also backs the new
  `internal/proof/verify` core.

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (now routed through the shared `verify.VerifyInclusion` core,
behavior-preserving per the review verdict); `GET /` realm-index dashboard; `GET /<domain>/log/` log
browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/checkpoint.ots`
DO carry strong ETag + `no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — certificate observable surface complete (carried forward).** All six numbered
clauses (§1 SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING KEY, §5 BITCOIN ANCHOR, §6 RECORD
HISTORY) plus BOTH anchor panels render. Badge, DS shell, `/` index, `/<domain>/log/` browser, hub
dossier, frozen Exhibit, record list, single-record page, ISCC-IDv1 decoder, Hub-List resolver,
proof-bundle endpoint + download link all built and verified.
- **This iteration's M-UI touch:** the certificate §3 clause's inline leaf-hash-and-verify primitive was
  swapped to call the shared `verify.VerifyInclusion` core (`handler.go:803`) — a behavior-preserving
  refactor (rendered output byte-identical; the `TestCertificate` suite passed unchanged per review). No
  observable certificate Verify element opened or closed.
- **Still open on the M-UI Verify bar:** the **mandatory M-UI exit visual-pass + human sign-off**
  (ADR-0012; tooling present, per-surface screenshots run in review, but the full exit pass + sign-off
  not yet executed); dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.
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
**Status**: **WASM — STARTED this iteration (shareable core only). OTS — both observable HTTP halves
landed; only a real Bitcoin confirmation remains (offline-unprovable).**
- **WASM:** the pure, WASM-shareable RFC-6962 inclusion-verifier core landed —
  `internal/proof/verify.VerifyInclusion` (`verify.go`), a three-way `(bool, error)` verdict wrapping
  `rfc6962.DefaultHasher.HashLeaf` + `merkleproof.VerifyInclusion`, **verified to compile under
  `GOOS=js GOARCH=wasm` (re-run this assessment)** and import-clean (no net/os/sql/template/embed). Both
  production call sites route through it (`internal/proofserve/handler.go:679`,
  `internal/certificate/handler.go:803`); 3 `func Test` in `verify_test.go`, mutation-proven per review.
  **Still 1/1 open:** no `syscall/js` import anywhere, no `//go:build js` target, no `cmd/wasm`
  entrypoint, no lazy tier-2 enhancement on the certificate/dossier, and the standalone
  `monitor.iscc.codes` Independent Verification app does not exist. The "identical verdicts WASM vs
  server / published artifact hash / split-view alert" Verify is unblocked by the core but not met.
- **OTS:** the `.ots` serve route (`GET /<domain>/log/checkpoint.ots`, `internal/proofserve`), the
  certificate §5 anchor clause, the store layer (`internal/store/ots.go`), the off-path stamp/upgrade
  loop (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`),
  and the real calendar transport (`internal/otsclient`) are all wired in `main.go`'s `runOTSLoop`. The
  Verify-closer not yet built: a root reaching **Bitcoin-confirmed** — exercised only against an injected
  Upgrader in tests, depends on a live calendar + real BTC confirmation. Still 1/1 open. **Open `normal`
  defect (filed, NOT fixed):** the production `Stamp` path (`internal/otsclient/client.go:121`) has
  NEITHER a panic-recover NOR a per-request timeout (the symmetric guards the upgrade path got via
  `safeUpgrade`). The next stamp-path touch should add `safeStamp`.

## Quality gates
**Status**: **GREEN.** HEAD (`c6c392a`) is itself the `cid(review)` PASS commit for the `proof/verify`
extraction; the branch is in sync with `origin/develop` (0 ahead / 0 behind).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 24 packages incl. the new
  `internal/proof/verify`; `gofmt -l .` clean; `go mod tidy -diff` clean — no new prod dependency,
  the core reuses the existing `transparency-dev/merkle`).
- The latest **`review` verdict is PASS at HEAD `c6c392a`** (proof/verify extraction; behavior-preserving
  at both call sites, mutation-proven non-vacuous, WASM-shareability + purity gates clean, Codex clean,
  no new issue).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`,
  branch `develop`. **CI at HEAD `c6c392a` = `success`** (run 27929219539, headSha matches HEAD). No CI
  failure.
- **Open issues: 0 `critical`, 5 `normal`, 8 `low`** (unchanged this iteration; no issue opened or
  resolved). The 5 `normal`: (a) §5 does not bind the OTS proof digest to §2's root (unreachable in
  prod); (b) the production OTS `Stamp` path has neither panic-recover nor timeout (crash/hang on a
  malformed/stalled calendar, a live path); (c) Hub-List `hubDomain` `ForceQuery` fail-open; (d) §4 AND
  bundle `did:web:` + raw domain mis-render a `host:port` hub's DID; (e) §6 omits the per-record `· at`
  timestamp.

## Next Milestone
**M1/M2/M3 met; M-UI near exit, WASM (core landed) + OTS are the active milestones. Gate green at HEAD,
CI green — no CI hygiene needed.** The natural next observable Verify-closers:

1. **WASM verifier — continue from the just-landed core** (1/1 Verify open). Add the `syscall/js`
   entrypoint (`cmd/wasm`) that exports `verify.VerifyInclusion` to JS, then the lazy tier-2 enhancement
   on the M-UI certificate/dossier, then the standalone `monitor.iscc.codes` Independent Verification app
   (reproducible build + published hash + SRI pin). This is the only un-started v1 milestone with
   offline-provable Verify criteria, and the core seam is now in place.
2. **M-UI exit gate (ADR-0012):** run the agent-browser visual pass on every SSR surface, file deviations
   as issues, and obtain human sign-off — M-UI does not reach DONE until this clears. Fold in the dossier
   §4 Bitcoin-anchor / comparison-anchor named regions for full parity.

Fold in the `safeStamp` guard (the live-path crash/hang `normal` defect) when the stamp path is next
touched; the §5 digest-binding fix and the `host:port` DID `%3A`-encode (§4 + bundle) when `handler.go`
is next touched. The OTS "upgrades to Bitcoin-confirmed" half stays offline-unprovable (live calendar +
chain).
