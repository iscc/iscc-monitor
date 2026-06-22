<!-- assessed-at: 657ed124d72c6ec3a2f3dde23b5fc7a170535a19 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM verifier — loader skeleton landed and the CDN-free gate is re-greened; ready to wire the first real caller

The Go WASM runtime loader (`wasm_exec.js`) is served byte-verbatim at `/_ds/wasm_exec.js`, and the
gate hole that blocked the prior step is **closed**: the `noExternalCDN` `stripLineComments` helper now
treats `//` as a comment only at line-start or after whitespace, so a protocol-relative `src="//cdn..."`
trips the ban again (mutation-proven, review PASS_WITH_NOTES). The 4 previously-unpushed commits are now
pushed and **CI is green at HEAD `657ed12`**. M1/M2/M3 stay met and M-UI's certificate-observable
surface stays complete; the WASM milestone still has no SSR caller, no `.wasm` artifact, and no published
hash — that is the next step.

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
    `develop`, per-surface screenshots run in review, full exit pass + sign-off not yet executed); plus
    the dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.
  - **WASM verifier: 1/1 open — core + entrypoint + loader asset landed, but no caller yet.** Confirmed
    by direct probe: no SSR caller (grep of `internal/dashboard|dossier|certificate|proofserve` finds
    zero references to `isccVerifyInclusion`/`VerifyJSON`/`wasm_exec.js`), no `.wasm` artifact built or
    committed, no `mise run build:wasm` task (none in `mise.toml`), no reproducible build + published
    hash + SRI pin, and the standalone `monitor.iscc.codes` Independent Verification app does not exist.
    The Verify ("identical vectors → identical verdicts WASM vs server in-browser; published artifact
    hash; split-view alert") is not yet met.
  - **OTS anchoring: 1/1 open (carried forward, untouched this iteration).** Both observable HTTP halves
    are CLOSED (the `.ots` serve route + the certificate §5 anchor render). What remains: a root that
    actually transits to **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected
    Upgrader in tests, never the live chain. Still 1/1.
- **Last ~10 iterations: ~4-5 milestone-Verify-advancing / ~5-6 foundational·plumbing·hardening.** Arc:
  certificate §5 → certificate COMPARISON ANCHOR panel → `internal/proof/verify` shareable core →
  `cmd/wasm` callable entrypoint → `/_ds/wasm_exec.js` loader asset → **CDN-free gate re-green
  (test-only)**. **DRIFT WATCH (clear):** the last increment was a one-file test-only gate fix, not a
  Verify-closer, but it was the prescribed re-green that unblocks the WASM caller progression — not
  polish-for-polish's-sake. The next step (build + serve the verifier `.wasm` + the `<script>` caller)
  is the direct tier-2 Verify-closer; a second consecutive non-Verify step would be drift.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `24d715d..HEAD` diff (confined to
`internal/web/web_test.go` + context). All M1 Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart survival;
structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}` (+ `cmd/wasm/verifyadapter`); **22 internal
  packages** — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz,
  index, logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` (production callers
  `internal/ots` + `internal/otsclient`). `transparency-dev/merkle` backs `internal/proof/verify`, which
  `cmd/wasm/verifyadapter` reuses (no new dependency — `syscall/js` is stdlib).

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
- **Residual notes (filed `normal`/`low`, NOT fixed):**
  - §5 does not bind the OTS proof's committed `File.Digest` to §2's accepted root (`normal`;
    unreachable in the production write path).
  - `did:web:` + raw `data.Domain` rides TWO surfaces (§4 AND the bundle), mis-rendering a `host:port`
    hub's DID (`normal`; `%3A`-encode both sites together; not exploitable on the clean testnet realm).
  - `internal/registry` `hubDomain` does not check `u.ForceQuery`, so a bare trailing `?` slips the
    guard (`normal`; resolver not yet wired into a live caller).
  - §6 rows omit the per-record `· at` timestamp the mockup shows (`normal`; needs a store schema change).

## WASM verifier · OTS anchoring
**Status**: **WASM — core + callable entrypoint + `/_ds/wasm_exec.js` loader landed; the CDN-free gate
hole is now CLOSED (review PASS_WITH_NOTES). No SSR caller / `.wasm` artifact / build task / published
hash yet. OTS — both observable HTTP halves landed; only a real Bitcoin confirmation remains
(offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion` on
  `globalThis`; the pure `cmd/wasm/verifyadapter.VerifyJSON` base64-Std-decodes the proof-bundle fields
  into `verify.VerifyInclusion`. `internal/web` serves the Go 1.26.1 `wasm_exec.js` byte-verbatim at
  `/_ds/wasm_exec.js` (`WasmExecPath`, `//go:embed`), reusing the no-cache/strong-ETag/304/405/CORS
  policy. **This iteration re-greened the gate:** `stripLineComments` (`internal/web/web_test.go:57`) now
  treats `//` as a comment only at line-start or after whitespace, so a protocol-relative
  `src="//cdn.jsdelivr.net/x.js"` survives the strip and trips `noExternalCDN` again — mutation-proven by
  the added `TestNoExternalCDNProtocolRelative`, byte-verbatim `wasm_exec.js` still passes. **Still 1/1
  open:** no SSR caller (grep finds none in the SSR packages), no `.wasm` built/committed, no
  `mise run build:wasm` task, no reproducible build + published hash + SRI pin, no standalone
  `monitor.iscc.codes` app.
  **Carried `normal` defect (NOT fixed):** the untagged `cmd/wasm/main.go:39-40` glue reads `index`/`size`
  via `js.Value.Int()` (= `int(v.Float())`), which truncates a non-integer JS Number. Not exploitable (no
  caller; the tested `verifyadapter.VerifyJSON` takes `uint64` and is correct). Land the safe-integer
  validation when the first real caller is wired.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the certificate §5 anchor clause, the store
  layer (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the real
  calendar transport (`internal/otsclient`) are all wired in `main.go`'s `runOTSLoop`. The Verify-closer
  not yet built: a root reaching **Bitcoin-confirmed** — depends on a live calendar + real BTC
  confirmation. Still 1/1 open. **Open `normal` defect (filed, NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the
  symmetric guards the upgrade path got via `safeUpgrade`). The next stamp-path touch should add
  `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest review verdict PASS_WITH_NOTES, CI green at HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (build + vet + test all 25 packages `ok`; `gofmt -l .`
  clean). The blocking CDN-free gate hole from the prior assessment is **closed** — the gate is now
  strictly stronger than before (protocol-relative `//cdn.` URLs trip the ban; mutation-proven).
- The latest **`review` verdict is PASS_WITH_NOTES at HEAD `657ed12`** (loop CONTINUE). DONE additionally
  requires no open `normal`/`critical` — 6 `normal` are still open, so the loop stays CONTINUE.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`,
  branch `develop`. **The branch is now in sync with `origin/develop` (0 ahead / 0 behind); the latest CI
  run is at the current HEAD `657ed12` = `success`.** The earlier "4 unpushed commits / CI behind HEAD"
  gap is resolved.
- **Open issues: 0 `critical`, 6 `normal`, 9 `low`.** The prior `normal` (holed `noExternalCDN`
  `stripLineComments`) was resolved and deleted; a narrower **`low`** residual was filed (whitespace-
  prefixed protocol-relative `//cdn.` URLs still strip — pre-existing, latent, no served asset triggers
  it). The 6 `normal`: (a) §5 does not bind the OTS proof digest to §2's root; (b) the production OTS
  `Stamp` path has neither panic-recover nor timeout; (c) Hub-List `hubDomain` `ForceQuery` fail-open;
  (d) §4 AND bundle `did:web:` + raw domain mis-render a `host:port` hub's DID; (e) §6 omits the
  per-record `· at` timestamp; (f) WASM shim `js.Value.Int()` truncates a non-integer JS `index`/`size`.

## Next Milestone
**WASM is the active milestone and the gate is green — wire the first real caller (1/1 Verify open).**

1. **WASM verifier — build + serve the verifier `.wasm` and the first real caller.** Add a
   `mise run build:wasm` task (ADR-0003 reproducible-build / published-hash / SRI pin), then the
   certificate/dossier `<script>` loader that calls `isccVerifyInclusion` with the base64-Std fields the
   surface already emits. This is the right place to land the `js.Value.Int()` integer/safe-integer
   validation the `normal` issue (f) tracks. After that: the standalone `monitor.iscc.codes` Independent
   Verification app (Surface C).
2. **M-UI exit gate (ADR-0012):** run the agent-browser visual pass on every SSR surface, file deviations
   as issues, obtain human sign-off; fold in the dossier §4 anchor named regions for full parity.
3. **OTS:** the "upgrades to Bitcoin-confirmed" half stays offline-unprovable (live calendar + chain);
   fold in `safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact
   lines are next edited.
