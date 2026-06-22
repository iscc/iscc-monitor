<!-- assessed-at: 5266efce568732333a0c988b17f46a22f962d405 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone in progress — Surface-C (`monitor.iscc.codes` Independent Verification app) has landed as a static no-JS skeleton (`internal/verifier`); the live `?monitor=` + WASM wiring and the dossier tier-2 caller remain the open WASM Verify work.

Incremental review against assessed-at `09cb17e`: the only source change since is the new
`internal/verifier` leaf (Surface-C skeleton — handler + template + test, deliberately unmounted, no
`go.mod` change). M1/M2/M3/M-UI source untouched and carry forward met. The latest `review` verdict is
PASS_WITH_NOTES (loop CONTINUE), CI is green at current HEAD `5266efc` (matches `origin/develop`), and
no `critical`/`normal`-blocking gate failure exists — but the WASM and OTS Verify criteria are still
open and 9 `normal` issues remain, so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Built/byte-pinned `.wasm` at `/_ds/verify.wasm`
    (`WasmVerifyHash`, `TestWasmVerifyHashPinned`); first SSR `<script>` tier-2 caller on the
    **certificate** (live-verified). Surface-C **skeleton** landed (`internal/verifier`: static no-JS
    page rendering the Independent Verification named regions, golden-tested) but is a SKELETON — it is
    **deliberately unmounted** in `buildMux` (grep-confirmed: no `verifier` reference in `cmd/`),
    parses **no `?monitor=<url>`**, runs **no WASM**, and renders the split-view alert
    UNCONDITIONALLY (the filed `normal` honesty gap). Still open: NO **dossier** tier-2 caller
    (grep-confirmed: no `verify.wasm`/`wasm_exec` in `internal/dossier`); the Surface-C **live wiring**
    (`?monitor=` parse + WASM run + conditional mismatch alert + identical-verdict parity); the GitHub
    Pages / `monitor.iscc.codes` deploy. Verify not met.
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots` serve
    route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~6 milestone-Verify-advancing / ~4 foundational·plumbing·hardening.** Recent
  arc: `/` named-region parity → first SSR WASM `<script>` caller on the certificate → self-hosted
  ISCC logo rendered on all six mastheads (closed the human `critical`) → **Surface-C static skeleton
  (`internal/verifier`)**. **DRIFT WATCH (clear):** increments are closing milestone Verify criteria /
  building toward the open WASM Surface-C app, not polish. The loop is correctly pointed at the WASM
  gap; the natural next sub-step (Surface-C live wiring) also closes the honesty gap just filed.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `09cb17e..HEAD` diff (confined to the
new `internal/verifier` package + context/docs). All M1 Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once +
restart survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}` (+ `cmd/wasm/verifyadapter`); 23 internal
  packages — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz,
  index, logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`. `transparency-dev/merkle` backs `internal/proof/verify`, which
  `cmd/wasm` reuses (`syscall/js` is stdlib).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core);
`GET /` realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + chrome complete; the M-UI exit visual-pass + human sign-off is pending.**
Carried forward — no M-UI source (`dashboard`/`dossier`/`proofserve`/`certificate`/`badge`/`web`)
touched since assessed-at. All six certificate clauses (§1–§6) + both anchor panels + badge + DS shell
+ `/` index + log browser + hub dossier + frozen Exhibit + record list + single-record page +
ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam
Verify, and every SSR masthead carries the shared chrome (logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass has not been
  carried to the remaining SSR surfaces (dossier / log browser / single record / certificate lack the
  full masthead instance-identity block + back-link chain); `/` sub-region deltas (config-driven
  instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`;
  the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this browser
  re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest to §2's
  root; `did:web:` + raw `data.Domain` rides §4 AND the proof bundle (mis-renders a `host:port` DID);
  `hubDomain` fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN: certificate tier-2 caller live-verified; Surface-C app exists only
as a static unmounted skeleton; no dossier caller / no `?monitor=` live wiring / no Pages deploy yet.
OTS — both observable HTTP halves landed; only a real Bitcoin confirmation remains
(offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  pure `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`.
  `cert.html` embeds the tier-2 proof island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader and
  shows the WASM verdict only when §3 passed. **Surface-C this iteration:** new `internal/verifier`
  leaf (`handler.go` + `verifier.html` + `handler_test.go`, 5 tests) renders the Independent
  Verification mockup's named regions (chrome, breadcrumb, independence statement naming
  `monitor.iscc.codes`/`monitor.iscc.id`, five verification-record steps, split-view `(size, root)`
  form, guided mismatch alert), no-CDN, self-hosted, golden-tested. It is a **skeleton**: deliberately
  NOT mounted in `buildMux`, parses NO `?monitor=`, runs NO WASM. **Still 1/1 open on the milestone
  Verify:** no dossier tier-2 caller (grep-confirmed); Surface-C live `?monitor=` + WASM run +
  conditional-on-real-verdict mismatch alert; identical-verdict (WASM vs server) parity surface; the
  Pages / `monitor.iscc.codes` deploy. **Carried `normal` defects (NOT fixed):** (a) `safeIndex` is a
  pure `float64→(uint64,string)` fn trapped in the tagged `main.go` with NO executable test — move it
  into the untagged `verifyadapter` and table-test the reject branches; (b) NEW this iteration — the
  Surface-C skeleton renders the split-view mismatch alert unconditionally in the present tense (an
  un-run negative verdict); the live-wiring sub-step is its natural fix point.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the calendar
  transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer not
  yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation.
  Still 1/1 open. **Open `normal` defect (NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the
  symmetric guards the upgrade path got via `safeUpgrade`); the next stamp-path touch should add
  `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS_WITH_NOTES (loop CONTINUE), CI
green at current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (all 26 packages `ok`, `internal/verifier` included;
  `gofmt -l .` excl. `cauldron/` clean); `mise run build:wasm` reproduces `verify.wasm` byte-identical
  to `WasmVerifyHash`.
- **Latest `review` verdict: PASS_WITH_NOTES (loop CONTINUE)** for the Surface-C skeleton. Gates green,
  mutation-proof non-vacuous, Codex clean, visual pass strong parity; one `normal` honesty gap filed
  (the unconditional mismatch alert), not progress-blocking.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`, in sync with `origin/develop` (both at `5266efc`). **Latest CI run is `success` at current
  HEAD `5266efc`** (run 27936631410).
- **Open issues: 0 `critical`, 9 `normal`, 9 `low`.** DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE. The 9 normal span: certificate §5 digest binding, OTS `safeStamp` guard,
  `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the
  `safeIndex` test gap, the certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region
  parity deltas, and the NEW Surface-C unconditional-mismatch-alert honesty gap.

## Next Milestone
**Continue the WASM milestone — it is the front-of-queue open Verify.** Wire the Surface-C **live
verification** sub-step onto `internal/verifier`: mount the handler in `buildMux`, parse `?monitor=<url>`
(+ `?id=`), embed the `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader + JSON data-island (mirror the
certificate tier-2 pattern), fetch the target instance's proof bundle client-side, run
`isccVerifyInclusion`, and gate the rendered ✓ / mismatch-alert on the GENUINE re-verification — which
also closes the new `normal` honesty gap (the alert becomes conditional on a real verdict). Then wire
the same tier-2 caller into the **hub dossier** (no caller today). At a WASM touch, move `safeIndex`
into the untagged `verifyadapter` and table-test its reject branches.

Subsequent: the GitHub Pages / `monitor.iscc.codes` deploy (final Surface-C piece); the OTS "upgrades
to Bitcoin-confirmed" half (offline-unprovable) plus folding in `safeStamp`, the §5 digest-binding, and
the `host:port` DID `%3A`-encode when those exact lines are next edited; carry the named-region + `←`
back-link parity pass across the deferred SSR surfaces; the M-UI exit visual-pass + human sign-off
(ADR-0012).
