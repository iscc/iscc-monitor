<!-- assessed-at: 2bc2ae91fed06bc59f9a1d8749eb9bd9f410f79c -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone in progress — Surface-C (`monitor.iscc.codes` Independent Verification app) now has its live in-browser re-verification wiring (`internal/verifier`: parse `?monitor=`/`?id=` → WASM data-island + loader → genuine verdict-gated mismatch alert). Still open on the milestone: the dossier tier-2 caller, the static GitHub-Pages deploy (with client-side target gating), and a real Bitcoin-confirmed OTS transit.

Incremental review against assessed-at `5266efc`. The ONLY source change since is the `internal/verifier`
leaf (Surface-C live wiring: `handler.go` + `verifier.html` + `handler_test.go`, +741/-360 across those 3
source files plus context/docs). M1/M2/M3/M-UI/OTS source untouched and carry forward met/open as before.
The latest `review` verdict is PASS_WITH_NOTES (loop CONTINUE), CI is `success` at current HEAD `2bc2ae9`
(== `origin/develop`), and no `critical` issue exists — but the WASM and OTS Verify criteria are still
open and 10 `normal` issues remain, so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open — but materially advanced.** Built/byte-pinned `.wasm` at
    `/_ds/verify.wasm` (`WasmVerifyHash`, `TestWasmVerifyHashPinned`); first SSR `<script>` tier-2 caller
    on the **certificate** (live-verified). Surface-C is no longer a static skeleton: `internal/verifier`
    now has the **live wiring** (verified) — `parseTarget` validates `?monitor=&id=` with `net/url`
    (fails closed to baseline), and under `{{if .HasTarget}}` the template emits a JSON data-island +
    `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader that fetches `<monitor>/inclusion/<id>.bundle`, runs
    `isccVerifyInclusion`, and gates the verdict/mismatch alert on a GENUINE re-verification (the
    mismatch alert is illustrative-by-default, revealed only on a real `failed` verdict — this closed the
    prior unconditional-alert honesty issue). It remains **deliberately unmounted** in `buildMux`
    (grep-confirmed: only `cmd/notecheck` + `cmd/wasm` reference "verifier", no `internal/verifier` in
    `cmd/`). Still OPEN on the milestone Verify: NO **dossier** tier-2 caller (grep-confirmed: no
    `verify.wasm`/`wasm_exec`/`isccVerify` in `internal/dossier`); the GitHub-Pages /
    `monitor.iscc.codes` deploy — which must reconcile the NEW Codex-confirmed gap that `.HasTarget` is
    server-side and freezes in a static artifact (read the target client-side); and the cross-origin
    verifier-scope gap (the WASM core verifies inclusion math only — no checkpoint-signature / id-binding
    check — so a malicious monitor can produce a green `verified`). Verify not met.
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots` serve
    route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~6 milestone-Verify-advancing / ~4 foundational·plumbing·hardening.** Recent
  arc: self-hosted ISCC logo on all six mastheads (closed the human `critical`) → Surface-C static
  skeleton → **Surface-C LIVE wiring (this iteration)**. **DRIFT WATCH (clear):** increments are
  building the open WASM Surface-C app toward its Verify, not polish; this one also closed a `normal`
  honesty issue. The loop is correctly pointed at the WASM gap; the natural next sub-step (dossier
  tier-2 caller / the Pages deploy) keeps closing the milestone.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `5266efc..HEAD` diff (confined to the
`internal/verifier` leaf + context/docs). All M1 Verify criteria remain satisfied: `origin`/`vkey`
golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart survival;
structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}` (+ `cmd/wasm/verifyadapter`); 23 internal
  packages — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve,
  verifier, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
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
Carried forward — no M-UI source (`dashboard`/`dossier`/`proofserve`/`certificate`/`badge`/`web`) touched
since assessed-at. All six certificate clauses (§1–§6) + both anchor panels + badge + DS shell + `/` index
+ log browser + hub dossier + frozen Exhibit + record list + single-record page + ISCC-IDv1 decoder +
Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify, and every SSR
masthead carries the shared chrome (logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass has not been
  carried to the remaining SSR surfaces (dossier / log browser / single record / certificate lack the full
  masthead instance-identity block + back-link chain); `/` sub-region deltas (config-driven instance
  identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`; the
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this browser
  re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest to §2's
  root; `did:web:` + raw `data.Domain` rides §4 AND the proof bundle (mis-renders a `host:port` DID);
  `hubDomain` fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN but advanced: certificate tier-2 caller live-verified; Surface-C now
has its live `?monitor=`/`?id=` + WASM re-verification wiring (verified, honest mismatch alert) but is
still unmounted and not deployed; no dossier caller; the static-deploy gating + verifier-scope
signature/id gaps remain. OTS — both observable HTTP halves landed; only a real Bitcoin confirmation
remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the pure
  `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`. `cert.html`
  embeds the tier-2 proof island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader and shows the WASM
  verdict only when §3 passed. **Surface-C this iteration:** `internal/verifier` (`handler.go` +
  `verifier.html` + `handler_test.go`, 8 tests) is now the live re-verifier — `parseTarget` validates
  `?monitor=&id=` (http/https scheme + non-empty host + no fragment, fails closed to baseline), and under
  `{{if .HasTarget}}` the template emits the JSON data-island + WASM loader that fetches
  `<monitor>/inclusion/<id>.bundle`, runs `isccVerifyInclusion`, and gates the verdict (and the now-honest
  illustrative-by-default mismatch alert) on a real verdict. It is **deliberately NOT mounted** in
  `buildMux` (Surface C ships on a different origin). **Still 1/1 open on the milestone Verify:** no
  dossier tier-2 caller (grep-confirmed); the GitHub-Pages / `monitor.iscc.codes` deploy; identical-verdict
  (WASM vs server) parity exercised end-to-end live (golden-tested as markup only — the review could not
  drive a live `verified`/`failed` transition from the sandbox). **Carried/new `normal` defects (NOT
  fixed):** (a) `safeIndex` is a pure `float64→(uint64,string)` fn trapped in the tagged `main.go` with NO
  executable test — move it into the untagged `verifyadapter` and table-test the reject branches; (b) NEW —
  Surface-C live wiring is gated on SERVER-side `.HasTarget`, which freezes in the documented STATIC
  GitHub-Pages artifact (`?monitor=&id=` unreachable in production) — the deploy step must read the target
  client-side; (c) NEW — the WASM verifier core proves inclusion math ONLY (no checkpoint-signature check,
  no id-binding), so a malicious cross-origin monitor can render a green `verified` — same scope the
  certificate tier-2 already ships, more acute on Surface C; the success copy overstates it.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in `internal/follower/otsloop.go`),
  the offline classifier (`internal/ots.Confirmed`), and the calendar transport (`internal/otsclient`) are
  all wired via `main.go`'s `runOTSLoop`. The Verify-closer not yet built: a root reaching
  **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation. Still 1/1 open. **Open `normal`
  defect (NOT fixed):** the production `Stamp` path (`internal/otsclient/client.go:121`) has NEITHER a
  panic-recover NOR a per-request timeout (the symmetric guards the upgrade path got via `safeUpgrade`);
  the next stamp-path touch should add `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS_WITH_NOTES (loop CONTINUE), CI green
at current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. Review
  reported `mise run check` green at HEAD (`internal/verifier` included; `gofmt -l .` excl. `cauldron/`
  clean); `mise run build:wasm` reproduces `verify.wasm` byte-identical to `WasmVerifyHash`.
- **Latest `review` verdict: PASS_WITH_NOTES (loop CONTINUE)** for the Surface-C live wiring. Gates green,
  the honesty fix mutation-proven + visually confirmed (ADR-0012 `agent-browser`), Codex surfaced 2
  reviewer-confirmed P1 design gaps filed `normal` (static-deploy gating + verifier-scope signature/id).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`,
  in sync with `origin/develop` (both at `2bc2ae9`). **Latest CI run is `success` at current HEAD
  `2bc2ae9`** (run 27937713838).
- **Open issues: 0 `critical`, 10 `normal`, 9 `low`.** DONE requires 0 critical AND 0 normal, so the loop
  stays CONTINUE. The 10 normal span: certificate §5 digest binding, OTS `safeStamp` guard, `hubDomain`
  ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the `safeIndex` test gap, the
  certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region parity deltas, the NEW Surface-C
  static-deploy `.HasTarget` gating gap, and the NEW WASM verifier-scope signature/id-binding gap.

## Next Milestone
**Continue the WASM milestone — it is the front-of-queue open Verify.** With Surface-C live wiring landed,
wire the same tier-2 caller into the **hub dossier** (no caller in `internal/dossier` today) — mirror the
certificate/verifier data-island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader pattern. Then land the
GitHub-Pages / `monitor.iscc.codes` deploy (final Surface-C piece), which MUST read the target
**client-side** (`location.search` / `URLSearchParams`), not via server-side `.HasTarget`, to fix the new
static-deploy gating issue. At a WASM-verifier-scope touch, expand the core to verify the checkpoint
signature against the hub's did:web key + bind the record to the requested id (the new cross-origin
trust-path gap), and move `safeIndex` into the untagged `verifyadapter` with table-tested reject branches.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in `safeStamp`,
the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are next edited; carry
the named-region + `←` back-link parity pass across the deferred SSR surfaces; the M-UI exit visual-pass +
human sign-off (ADR-0012).
