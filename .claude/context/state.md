<!-- assessed-at: 2e83c6ecf32826fdb586703d620f68691dd04eae -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM verifier — callable `cmd/wasm` entrypoint landed (exports `globalThis.isccVerifyInclusion` under `GOOS=js GOARCH=wasm`); no SSR caller, no standalone app, no published-hash artifact yet

The shared RFC-6962 inclusion-verifier core (`internal/proof/verify`) now has a callable WASM
entrypoint: `cmd/wasm` (tagged `js && wasm`) wraps the pure, linux-testable `cmd/wasm/verifyadapter`
and exposes `isccVerifyInclusion` to JavaScript. M1/M2/M3 are met; M-UI's certificate-observable
surface is complete (exit visual-pass + human sign-off and dossier anchor regions remain). WASM is
mid-flight (core + entrypoint, no caller); OTS keeps its offline-unprovable live-chain half open.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met (certificate-observable surface complete, carried forward unchanged).** Met:
    five-status `HubStatusBadge` (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted
    fonts, CDN-free); `/` realm-index grid; `/<domain>/log/` browser; hub dossier (`internal/dossier`);
    frozen Exhibit; paginated record list; single-record page; ISCC-IDv1 decoder
    (`internal/index.Decode`); `(realm, hub_id) → domain` Hub-List resolver (`internal/registry`);
    certificate **§1–§6** + proof-bundle endpoint (`.bundle`) + the distinct COMPARISON ANCHOR panel.
    **Open:** the **M-UI exit visual-pass + human sign-off** (ADR-0012; agent-browser tooling on
    `develop`, per-surface screenshots run in review, full exit pass + sign-off not yet executed);
    plus the dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.
  - **WASM verifier: 1/1 open — core + callable entrypoint now both landed.** This iteration added
    `cmd/wasm` (the `syscall/js` shim exporting `isccVerifyInclusion`, verified to compile under
    `GOOS=js GOARCH=wasm`, 2.9 MB artifact, not committed) plus the pure linux-testable
    `cmd/wasm/verifyadapter.VerifyJSON` (base64-Std marshaling into the core; golden-vector parity test,
    mutation-proven per review). **Still open:** no SSR caller (no `wasm_exec.js` embed / `<script>`
    loader on certificate or dossier — `grep` of `internal/` finds zero references to
    `isccVerifyInclusion`/`VerifyJSON`/`wasm_exec.js`), no `mise run build:wasm` task, no reproducible
    build + published hash + SRI pin, and the standalone `monitor.iscc.codes` Independent Verification
    app does not exist. The Verify ("identical vectors → identical verdicts WASM vs server in-browser;
    published artifact hash; split-view alert") is not yet met.
  - **OTS anchoring: 1/1 open (carried forward).** Both observable HTTP halves are CLOSED (the `.ots`
    serve route + the certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader in tests,
    never the live chain (needs a live calendar + real BTC confirmation). Still 1/1.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 foundational·plumbing·hardening.** Recent
  arc: OTS adapter → real Upgrader + main.go wiring → otsclient hardening → OTS stamp seam →
  `.ots` serve route → certificate §5 → certificate COMPARISON ANCHOR panel → `internal/proof/verify`
  shareable core → **`cmd/wasm` callable WASM entrypoint**. **DRIFT WATCH (clear):** the last FIVE
  iterations each closed or began an observable Verify-relevant element. The natural next observable
  Verify-closer is the WASM **tier-2 progressive enhancement** (wire `isccVerifyInclusion` into the
  certificate/dossier with `wasm_exec.js` + a `<script>` loader) — the direct continuation of the
  entrypoint just landed, and the right place to fix the new `js.Value.Int()` truncation issue.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 production source touched by the `c6c392a..HEAD` diff
(confined to `cmd/wasm/` + context/issues). All M1 sources unchanged. All M1 Verify criteria remain
satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze +
alert-once + restart survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}` (+ `cmd/wasm/verifyadapter`); **23 internal
  packages** — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz,
  index, logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` (production callers `internal/ots`
  + `internal/otsclient`). `transparency-dev/merkle` (`proof`, `rfc6962`) backs `internal/proof/verify`,
  which `cmd/wasm/verifyadapter` now reuses (no new dependency — `syscall/js` is stdlib).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core);
`GET /` realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/checkpoint.ots`
DO carry strong ETag + `no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — certificate observable surface complete (carried forward; not touched this
iteration).** All six numbered clauses (§1 SUBJECT, §2 CHECKPOINT, §3 INCLUSION PROOF, §4 SIGNING KEY,
§5 BITCOIN ANCHOR, §6 RECORD HISTORY) plus BOTH anchor panels render. Badge, DS shell, `/` index,
`/<domain>/log/` browser, hub dossier, frozen Exhibit, record list, single-record page, ISCC-IDv1
decoder, Hub-List resolver, proof-bundle endpoint + download link all built and verified.
- **Still open on the M-UI Verify bar:** the **mandatory M-UI exit visual-pass + human sign-off**
  (ADR-0012; tooling present, per-surface screenshots run in review, but the full exit pass + sign-off
  not yet executed); dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.
- **Residual notes (filed `normal`/`low`, NOT fixed — fold in when the exact line is next edited):**
  - §5 does not bind the OTS proof's committed `File.Digest` to §2's accepted root (`normal`;
    unreachable in the production write path).
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID (`normal`; `%3A`-encode both sites together; not exploitable on the clean testnet realm).
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard (`normal`; resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows (`normal`; needs a store schema change).

## WASM verifier · OTS anchoring
**Status**: **WASM — core + callable entrypoint landed; no SSR caller / standalone app / published hash
yet. OTS — both observable HTTP halves landed; only a real Bitcoin confirmation remains
(offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion` on
  `globalThis` via `js.FuncOf` and blocks; the pure `cmd/wasm/verifyadapter.VerifyJSON` base64-Std-decodes
  the proof-bundle fields into `verify.VerifyInclusion` and folds the three-way verdict into
  `(verified, errMsg)`. **Re-verified this assessment:** `GOOS=js GOARCH=wasm go build ./cmd/wasm`
  exit 0 (2.9 MB, not committed); 1 `func Test` in `verify_adapter_test.go` (5 cases, golden-vector
  parity vs the core, mutation-proven per the review verdict). **Still 1/1 open:** no SSR caller (grep of
  `internal/` finds no `wasm_exec.js`/`isccVerifyInclusion`/`VerifyJSON`), no `mise` WASM build task, no
  reproducible build + published hash + SRI pin, and the standalone `monitor.iscc.codes` Independent
  Verification app (Surface C) does not exist. The "identical verdicts WASM vs server in-browser /
  published artifact hash / split-view alert" Verify is unblocked by the entrypoint but not met.
  **New `normal` defect filed (NOT fixed):** the untagged `cmd/wasm/main.go:39-40` glue reads
  `index`/`size` via `js.Value.Int()` (= `int(v.Float())`), which truncates a non-integer JS Number
  (`1.9 → 1`) and could report `verified` against a truncated leaf. Not exploitable (no caller; real
  callers emit server-computed integers; the tested `verifyadapter.VerifyJSON` takes `uint64` and is
  correct). Fix at the first real caller (the tier-2 wiring), where the JS→Go arg contract belongs.
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
**Status**: **GREEN.** HEAD (`2e83c6e`) is itself the `cid(review)` PASS_WITH_NOTES commit for the
`cmd/wasm` entrypoint; the branch is in sync with `origin/develop` (0 ahead / 0 behind), working tree
clean.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  `review` confirmed `mise run check` green (build + vet + test, all 24 packages incl.
  `cmd/wasm/verifyadapter`; `gofmt -l .` clean; `go mod tidy -diff` clean — no new dependency,
  `syscall/js` is stdlib). NOTE on the WASM package layout: `go vet ./cmd/wasm` alone returns
  "build constraints exclude all Go files" on linux (the package is platform-empty there — expected, NOT
  a compile error); the gate uses the `./...` wildcard, which skips it, so check stays green.
- The latest **`review` verdict is PASS_WITH_NOTES at HEAD `2e83c6e`** (WASM entrypoint;
  behavior-preserving, golden-vector parity green + mutation-proven non-vacuous, WASM-purity gates clean,
  one Codex P2 confirmed and filed `normal`, not blocking).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`,
  branch `develop`. **CI at HEAD `2e83c6e` = `success`** (run 27929899529, headSha matches HEAD). No CI
  failure.
- **Open issues: 0 `critical`, 6 `normal`, 8 `low`** (one `normal` opened this iteration: the WASM shim
  `js.Value.Int()` truncation). The 6 `normal`: (a) §5 does not bind the OTS proof digest to §2's root
  (unreachable in prod); (b) the production OTS `Stamp` path has neither panic-recover nor timeout (a
  live path); (c) Hub-List `hubDomain` `ForceQuery` fail-open; (d) §4 AND bundle `did:web:` + raw domain
  mis-render a `host:port` hub's DID; (e) §6 omits the per-record `· at` timestamp; (f) WASM shim
  `js.Value.Int()` truncates a non-integer JS `index`/`size` (untagged glue, no caller yet).

## Next Milestone
**M1/M2/M3 met; M-UI near exit, WASM (core + entrypoint landed) + OTS are the active milestones. Gate
green at HEAD, CI green — no CI hygiene needed.** The natural next observable Verify-closers:

1. **WASM verifier — wire the first real caller** (1/1 Verify open). The entrypoint is callable; the
   next sub-step is the **tier-2 progressive enhancement** on the M-UI certificate/dossier (embed
   `wasm_exec.js` from `$(go env GOROOT)/lib/wasm/wasm_exec.js`, a `<script>` that loads the `.wasm` and
   calls `isccVerifyInclusion` with the base64-Std fields the surface already emits) — and that is the
   right place to land the `js.Value.Int()` integer/safe-integer validation the new `normal` issue
   tracks. After that: the standalone `monitor.iscc.codes` Independent Verification app (Surface C) +
   reproducible build + published hash + SRI pin + a `mise run build:wasm` task. This is the only
   un-started-to-DONE v1 milestone with offline-provable Verify criteria.
2. **M-UI exit gate (ADR-0012):** run the agent-browser visual pass on every SSR surface, file
   deviations as issues, and obtain human sign-off — M-UI does not reach DONE until this clears. Fold in
   the dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.

Fold in the `safeStamp` guard (the live-path crash/hang `normal` defect) when the stamp path is next
touched; the §5 digest-binding fix and the `host:port` DID `%3A`-encode (§4 + bundle) when `handler.go`
is next touched. The OTS "upgrades to Bitcoin-confirmed" half stays offline-unprovable (live calendar +
chain).
