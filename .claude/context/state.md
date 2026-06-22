<!-- assessed-at: bfcbd98e27fe06ce4f546e589a5097773ba92360 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone in progress — the open front is Surface-C (`monitor.iscc.codes` Independent Verification app, live `?monitor=`/`?id=` → WASM re-verification wiring landed but unmounted/undeployed) plus the dossier tier-2 WASM caller and a real Bitcoin-confirmed OTS transit. The latest increment was M-UI navigation-closure chrome on the hub dossier (back-link + instance-identity + tier-2 link), not a WASM-Verify closer.

Incremental review against assessed-at `2bc2ae9`. The ONLY source change since is the dossier
masthead chrome leaf — `internal/dossier/dossier.html` + `internal/dossier/handler_test.go`
(+87/-3 in those two source files, the rest is context/docs). It verbatim-ports the certificate's
shared chrome masthead (`monitor instance` identity label + `verify ↗ monitor.iscc.codes` tier-2
link) and the `← Realm index` back-link into the dossier, making `/` → dossier → log browser fully
no-JS traversable. M1/M2/M3/WASM/OTS source untouched; all carry forward met/open as before. The
latest `review` verdict is PASS (loop CONTINUE), CI is `success` at current HEAD `bfcbd98`
(== `origin/develop`), and there is no `critical` issue — but the WASM and OTS Verify criteria are
still open and 10 `normal` issues remain, so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Built/byte-pinned `.wasm` at `/_ds/verify.wasm` (`WasmVerifyHash`,
    `TestWasmVerifyHashPinned`); first SSR tier-2 caller on the **certificate** (live-verified) and
    Surface-C `internal/verifier` has its **live wiring** (verified) — `parseTarget` validates
    `?monitor=&id=` with `net/url` (fails closed to baseline), and under `{{if .HasTarget}}` the
    template emits a JSON data-island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader that fetches
    `<monitor>/inclusion/<id>.bundle`, runs `isccVerifyInclusion`, and gates the verdict/mismatch
    alert on a GENUINE re-verification. `internal/verifier` remains **deliberately unmounted** in
    `cmd/` (grep-confirmed: no `internal/verifier` in `cmd/`). Still OPEN on the milestone Verify:
    **NO dossier tier-2 WASM caller** (grep-confirmed: no `verify.wasm`/`wasm_exec`/`isccVerify` in
    `internal/dossier` — the dossier's `verify ↗ monitor.iscc.codes` is a static link, not a WASM
    island); the GitHub-Pages / `monitor.iscc.codes` deploy (must read the target client-side, not
    via server-side `.HasTarget`, which freezes in a static artifact — Codex-confirmed); and the
    cross-origin verifier-scope gap (the WASM core verifies inclusion math only — no
    checkpoint-signature / id-binding — so a malicious monitor can produce a green `verified`).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots`
    serve route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 chrome·plumbing·hardening.** Recent
  arc: certificate tier-2 WASM caller (advances WASM) → self-hosted logo on all six mastheads
  (closed human `critical`) → Surface-C skeleton → Surface-C LIVE wiring (advances WASM) → **dossier
  tier-2 CHROME (this iteration — M-UI nav-closure, NOT a WASM-Verify closer).** **DRIFT WATCH
  (amber):** this last increment was masthead chrome (a verbatim cert port), not a WASM Verify
  closer, while the WASM milestone has three concrete open sub-steps (dossier WASM island, the Pages
  deploy with client-side gating, the verifier-scope sig/id). One chrome iteration is fine, but the
  next increment should re-point at a WASM Verify criterion (the dossier WASM data-island is the
  natural next sub-step now that its chrome shell is in place) rather than another polish pass.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `2bc2ae9..HEAD` diff (confined to
the dossier chrome leaf + context/docs). All M1 Verify criteria remain satisfied: `origin`/`vkey`
golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart
survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}` (+ `cmd/wasm/verifyadapter`); 23 internal
  packages — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz,
  index, logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.
  `transparency-dev/merkle` backs `internal/proof/verify`, which `cmd/wasm` reuses.

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me
at `GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core);
`GET /` realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API;
no ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong
ETag + `no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + chrome complete; the M-UI exit visual-pass + human sign-off is
pending.** This iteration advanced M-UI navigation closure: the hub dossier now carries the shared
chrome masthead (`monitor instance` identity label + `verify ↗ monitor.iscc.codes` tier-2 link) and
the `← Realm index` back-link, byte-identical to the certificate's masthead — so `/` → dossier → log
browser is fully no-JS traversable. All six certificate clauses (§1–§6) + both anchor panels + badge
+ DS shell + `/` index + log browser + hub dossier + frozen Exhibit + record list + single-record
page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral
HTTP-seam Verify; every SSR masthead carries the shared chrome (logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is now
  carried on the dossier but NOT yet on the remaining SSR surfaces (log browser / single record /
  certificate still lack the full instance-identity block + back-link chain on every surface);
  `/` sub-region deltas (config-driven instance identity/realm, Checkpoint/Bitcoin-anchor columns,
  recent-declarers footer) filed `normal`; the mandatory **M-UI exit visual-pass + human sign-off**
  (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this
  browser re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest
  to §2's root; `did:web:` + raw `data.Domain` rides §4 AND the proof bundle (mis-renders a
  `host:port` DID); `hubDomain` fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the
  per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN: certificate tier-2 caller live-verified; Surface-C
(`internal/verifier`) has its live `?monitor=`/`?id=` + WASM re-verification wiring (verified, honest
mismatch alert) but is still unmounted and not deployed; NO dossier WASM caller (only a static
verify-link landed this iteration); the static-deploy gating + verifier-scope signature/id gaps
remain. OTS — both observable HTTP halves landed; only a real Bitcoin confirmation remains
(offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  pure `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`.
  `cert.html` embeds the tier-2 proof island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader and
  shows the WASM verdict only when §3 passed. `internal/verifier` (`handler.go` + `verifier.html` +
  `handler_test.go`, 8 tests) is the live re-verifier — `parseTarget` validates `?monitor=&id=`
  (http/https scheme + non-empty host + no fragment, fails closed to baseline), and under
  `{{if .HasTarget}}` emits the JSON data-island + WASM loader that fetches
  `<monitor>/inclusion/<id>.bundle`, runs `isccVerifyInclusion`, and gates the verdict (and the
  honest illustrative-by-default mismatch alert) on a real verdict. It is **deliberately NOT mounted**
  in `cmd/` (Surface C ships on a different origin). **Still 1/1 open on the milestone Verify:** NO
  dossier WASM tier-2 caller (grep-confirmed — the dossier's `verify ↗ monitor.iscc.codes` is a
  static link, not a WASM island); the GitHub-Pages / `monitor.iscc.codes` deploy; identical-verdict
  (WASM vs server) parity exercised end-to-end live (golden-tested as markup only). **Carried/new
  `normal` defects (NOT fixed):** (a) `safeIndex` is a pure `float64→(uint64,string)` fn trapped in
  the tagged `main.go` with NO executable test — move it into the untagged `verifyadapter` and
  table-test the reject branches; (b) Surface-C live wiring is gated on SERVER-side `.HasTarget`,
  which freezes in the documented STATIC GitHub-Pages artifact (`?monitor=&id=` unreachable in
  production) — the deploy step must read the target client-side; (c) the WASM verifier core proves
  inclusion math ONLY (no checkpoint-signature check, no id-binding), so a malicious cross-origin
  monitor can render a green `verified` — same scope the certificate tier-2 already ships, more acute
  on Surface C; the success copy overstates it.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the
  calendar transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. The
  Verify-closer not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real
  BTC confirmation. Still 1/1 open. **Open `normal` defect (NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the
  symmetric guards the upgrade path got via `safeUpgrade`); the next stamp-path touch should add
  `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS (loop CONTINUE), CI green at
current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (dossier included; `gofmt -l .` excl. `cauldron/`
  clean).
- **Latest `review` verdict: PASS (loop CONTINUE)** for the dossier tier-2 chrome + back-link.
  Template-only production change (1 production + 1 test file), mutation-proven, visual pass
  (ADR-0012 `agent-browser`) and Codex both clean.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle
  on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`, in sync with `origin/develop` (both at `bfcbd98`). **Latest CI run is `success` at
  current HEAD** (run 27938396360).
- **Open issues: 0 `critical`, 10 `normal`, 9 `low`.** DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE. The 10 normal span: certificate §5 digest binding, OTS `safeStamp` guard,
  `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the
  `safeIndex` test gap, the certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region
  parity deltas, the Surface-C static-deploy `.HasTarget` gating gap, and the WASM verifier-scope
  signature/id-binding gap.

## Next Milestone
**Continue the WASM milestone — it is the front-of-queue open Verify, and the last increment was
M-UI chrome (re-point at a WASM Verify closer).** Wire the tier-2 WASM caller into the **hub
dossier** (no WASM island in `internal/dossier` today; its chrome shell now exists, so this is the
clean next sub-step) — mirror the certificate/verifier data-island + `/_ds/wasm_exec.js` +
`/_ds/verify.wasm` loader pattern. Then land the GitHub-Pages / `monitor.iscc.codes` deploy (final
Surface-C piece), which MUST read the target **client-side** (`location.search` / `URLSearchParams`),
not via server-side `.HasTarget`, to fix the static-deploy gating issue. At a WASM-verifier-scope
touch, expand the core to verify the checkpoint signature against the hub's did:web key + bind the
record to the requested id (the cross-origin trust-path gap), and move `safeIndex` into the untagged
`verifyadapter` with table-tested reject branches.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are
next edited; carry the named-region + `←` back-link parity pass across the remaining SSR surfaces
(log browser / single record / certificate); the M-UI exit visual-pass + human sign-off (ADR-0012).
