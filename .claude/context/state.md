<!-- assessed-at: 47f4605423defc36d0eedb10b7052f280a999707 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone in progress — the open front is Surface-C (`monitor.iscc.codes` Independent Verification app): its `?monitor=`/`?id=` target now resolves CLIENT-side so the static artifact works, but the page is still unmounted and not deployed. Remaining WASM Verify work: the GitHub-Pages deploy, the dossier tier-2 WASM caller, and the verifier-scope signature/id gap. OTS still needs a real Bitcoin-confirmed transit.

Incremental review against assessed-at `bfcbd98`. The ONLY source change since is `internal/verifier`
(Surface-C client-side target resolution): `handler.go` (-/+, now `tmpl.Execute(&buf, nil)` — pure
static artifact), `verifier.html`, and `handler_test.go` (+context/learnings/handoff/next docs). The
advance moved `?monitor=&id=` resolution OUT of the Go handler (removed `net/url`/`parseTarget`/
`pageData`/`HasTarget`, grep-confirmed) and INTO the always-emitted browser loader
(`new URLSearchParams(location.search)`), so the page is now rendered identically for every request —
the fix that makes the documented GitHub-Pages deployment functional. M1/M2/M3/M-UI/OTS source
untouched; all carry forward met/open as before. The latest `review` verdict is PASS_WITH_NOTES
(loop CONTINUE), CI is `success` at current HEAD `47f4605` (== `origin/develop`), and there is no
`critical` issue — but the WASM and OTS Verify criteria are still open and 10 `normal` issues remain,
so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Built/byte-pinned `.wasm` at `/_ds/verify.wasm` (`WasmVerifyHash`,
    `TestWasmVerifyHashPinned`); first SSR tier-2 caller on the **certificate** (live-verified);
    Surface-C `internal/verifier` now resolves its target **client-side** — the page is a single
    static artifact (`Handler` execs `nil` data), and the always-emitted end-of-body loader reads
    `?monitor=&id=` via `URLSearchParams(location.search)`, fetches `<monitor>/inclusion/<id>.bundle`,
    runs `isccVerifyInclusion`, and gates the verdict/mismatch alert on a GENUINE re-verification.
    `internal/verifier` remains **deliberately unmounted** in `cmd/` (grep-confirmed: no
    `internal/verifier` in `cmd/`). Still OPEN on the milestone Verify: **the GitHub-Pages /
    `monitor.iscc.codes` deploy** (only `.github/workflows/ci.yml` exists — no Pages publish workflow;
    the client-side gating that unblocks it has landed, the deploy step itself has not); **NO dossier
    tier-2 WASM caller** (grep-confirmed: no `verify.wasm`/`wasm_exec`/`isccVerify` in
    `internal/dossier` — the dossier's `verify ↗ monitor.iscc.codes` is a static link, not a WASM
    island); and the **cross-origin verifier-scope gap** (the WASM core verifies inclusion math only —
    no checkpoint-signature / id-binding — so a malicious monitor can render a green `verified`).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots`
    serve route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 chrome·plumbing·hardening.** Recent
  arc: self-hosted logo on all six mastheads (closed human `critical`) → Surface-C skeleton →
  Surface-C LIVE wiring (advances WASM) → dossier tier-2 chrome (M-UI nav-closure) → **Surface-C
  client-side target gating (this iteration — closes the static-deploy `.HasTarget` blocker, advances
  the WASM milestone toward a deployable artifact).** **DRIFT WATCH (clear→amber):** this increment
  re-pointed at a WASM concern (good — it closed the static-deploy `normal` exactly as the prior state
  flagged), but it was a refactor of an unmounted page, not a new Verify criterion closed; the WASM
  milestone still has three concrete open sub-steps (the Pages deploy, the dossier WASM island, the
  verifier-scope sig/id). The deploy step is now unblocked and is the natural next closer — the next
  increment should land it (or the dossier WASM island) rather than further refining the unmounted
  verifier markup.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `bfcbd98..HEAD` diff (confined to
`internal/verifier` + context/docs). All M1 Verify criteria remain satisfied: `origin`/`vkey`
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
pending.** No M-UI surface (`internal/dashboard`/`dossier`/`certificate`/`proofserve`) was touched by
this diff — carried forward. All six certificate clauses (§1–§6) + both anchor panels + badge + DS
shell + `/` index + log browser + hub dossier (with the shared chrome masthead + `← Realm index`
back-link) + frozen Exhibit + record list + single-record page + ISCC-IDv1 decoder + Hub-List
resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify; every SSR masthead
carries the shared chrome (logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the
  dossier but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still
  lack the full instance-identity block + back-link chain on every surface); `/` sub-region deltas
  (config-driven instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer)
  filed `normal`; the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this
  browser re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest
  to §2's root; `did:web:` + raw `data.Domain` rides §4 AND the proof bundle (mis-renders a
  `host:port` DID); `hubDomain` fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the
  per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN: certificate tier-2 caller live-verified; Surface-C
(`internal/verifier`) now resolves `?monitor=`/`?id=` CLIENT-side as a single static artifact
(verified, honest mismatch alert) — unblocking the static deploy — but it is still unmounted and not
deployed; NO dossier WASM caller (static verify-link only); the Pages deploy step and the
verifier-scope signature/id gaps remain. OTS — both observable HTTP halves landed; only a real
Bitcoin confirmation remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  pure `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`.
  `cert.html` embeds the tier-2 proof island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader and
  shows the WASM verdict only when §3 passed. `internal/verifier` (`handler.go` + `verifier.html` +
  `handler_test.go`) is now a SINGLE STATIC artifact: `Handler` execs `tmpl.Execute(&buf, nil)` with
  no per-request data (`net/url`/`parseTarget`/`pageData`/`HasTarget` removed, grep-confirmed), and
  the always-emitted end-of-body loader reads `?monitor=&id=` via `URLSearchParams(location.search)`,
  fetches `<monitor>/inclusion/<id>.bundle`, runs `isccVerifyInclusion`, and gates the verdict (and
  the honest illustrative-by-default mismatch alert) on a real verdict. With no target / no JS it
  renders the honest baseline asserting no un-run verdict. It is **deliberately NOT mounted** in
  `cmd/` (Surface C ships on a different origin / static host). **Still 1/1 open on the milestone
  Verify:** the GitHub-Pages / `monitor.iscc.codes` deploy (only `ci.yml` exists — no Pages publish
  workflow); NO dossier WASM tier-2 caller (grep-confirmed — the dossier's verify-link is static);
  identical-verdict (WASM vs server) parity exercised end-to-end live (golden-tested as markup only).
  **Carried/new `normal` defects (NOT fixed):** (a) `safeIndex` is a pure `float64→(uint64,string)`
  fn trapped in the tagged `main.go` with NO executable test — move it into the untagged
  `verifyadapter` and table-test the reject branches; (b) the WASM verifier core proves inclusion
  math ONLY (no checkpoint-signature check, no id-binding), so a malicious cross-origin monitor can
  render a green `verified` — same scope the certificate tier-2 already ships, more acute on Surface
  C; the success copy overstates it; (c) NEW — Surface-C `readTarget` (`verifier.html:550-553`)
  accepts opaque-scheme monitor forms (`https:example.com`) the Go `parseTarget` rejected (the JS
  WHATWG port is strictly more permissive) — NOT a trust defect (the browser resolves to the same
  host, WASM re-verifies the bundle, worst case is an honest `error`), fix = return `u.href` not the
  raw `monitor`. (The prior "Surface-C gated on SERVER-side `.HasTarget`" `normal` was CLOSED by this
  advance — its exact prescribed fix.)
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
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS_WITH_NOTES (loop CONTINUE), CI
green at current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (all 27 packages ok; `gofmt -l .` empty);
  `GOOS=js GOARCH=wasm go build ./cmd/wasm` OK.
- **Latest `review` verdict: PASS_WITH_NOTES (loop CONTINUE)** for the Surface-C client-side target.
  Tight diff (2 non-test/doc + 1 test + docs), all gates green, both claimed mutations reproduce,
  closes the `.HasTarget` static-deploy `normal`, visual pass (ADR-0012 `agent-browser`) and Codex
  both run (Codex P2 `readTarget` opaque-URL permissiveness confirmed non-trust, filed `normal`).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle
  on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`, in sync with `origin/develop` (both at `47f4605`). **Latest CI run is `success` at
  current HEAD** (run 27939899605). No GitHub-Pages publish workflow exists yet.
- **Open issues: 0 `critical`, 10 `normal`, 9 `low`.** DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE. The 10 normal span: certificate §5 digest binding, OTS `safeStamp` guard,
  `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the
  `safeIndex` test gap, the certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region
  parity deltas, the WASM verifier-scope signature/id-binding gap, and the new Surface-C `readTarget`
  opaque-URL permissiveness.

## Next Milestone
**Continue the WASM milestone — it is the front-of-queue open Verify.** The Surface-C static-deploy
blocker is now closed (client-side target), so the natural next closer is the **GitHub-Pages /
`monitor.iscc.codes` deploy** itself: render `verifier.Handler` once to a static `index.html` + the
`/_ds/` assets and publish via a `.github/workflows/*` (no such workflow exists today). The
alternative next sub-step is wiring the tier-2 WASM caller into the **hub dossier** (no WASM island in
`internal/dossier` today; its chrome shell exists) — mirror the certificate/verifier data-island +
`/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader. At a WASM-verifier-scope touch, expand the core to
verify the checkpoint signature against the hub's did:web key + bind the record to the requested id
(the cross-origin trust-path gap), move `safeIndex` into the untagged `verifyadapter` with
table-tested reject branches, and fold in the `readTarget` `u.href` normalization.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are
next edited; carry the named-region + `←` back-link parity pass across the remaining SSR surfaces
(log browser / single record / certificate); the M-UI exit visual-pass + human sign-off (ADR-0012).
