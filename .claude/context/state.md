<!-- assessed-at: 849657972dfaa1b68422f63f63bd51c162a5f075 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone — reproducibility critical CLOSED, gate green, work pushed. The committed
`internal/web/verify.wasm` (SHA-256 `2c91e61f…`) is now byte-reproducible from the documented `mise run
build:wasm` under the exact-patch-pinned toolchain (`mise.toml` `go = "1.26.4"`; the artifact embeds
`go1.26.4`), and `WasmVerifyHash` matches it. The 6-arg/id-binding shim is intact, so the deployed
verifier gates on id-binding. Latest `review` verdict is **PASS (loop CONTINUE)** and HEAD is **pushed —
`origin/develop` == HEAD**, CI **success** at HEAD. The WASM "published" deploy is still blocked at
`Configure Pages` (one-time human repo-Settings step); the WASM signature-half trust gap and the OTS
Bitcoin-confirmed transit remain open. DONE not reached: WASM + OTS Verify still open, 10 `normal` issues
stand.

Incremental review against assessed-at `193878c`. The `193878c..HEAD` diff touched exactly TWO production
files — `internal/web/web.go` (re-pin `WasmVerifyHash` `96b2a40d…`→`2c91e61f…`) and `mise.toml` (`go`
`"1.26"`→`"1.26.4"`) — plus a regenerated `internal/web/verify.wasm` and `.claude/context/*` docs. All
M1/M2/M3/M-UI/OTS production source is untouched — those sections carry forward met/open as before.
**Reproducibility critical CLOSED** (verified this assessment): `sha256sum internal/web/verify.wasm` →
`2c91e61f…` == the `web.go:93` pin; `strings … | grep -oE 'go1.26.[0-9]+'` → `go1.26.4` (was
`go1.26.1`); content markers survive (`expected 5 or 6 args` → 1, `expected 5 args` → 0, `decode record
envelope` → 1). The prior-cycle 11 commits pushed together on this PASS; `git status` clean,
`origin/develop` == HEAD.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Now CLOSED in source AND artifact: the byte-pinned `.wasm` carries the
    6-arg/id-binding shim, the pin matches the committed bytes, AND the bytes are reproducible from the
    documented `mise run build:wasm` (the audited-artifact contract is restored). Certificate SSR tier-2
    caller (live-verified); Surface-C `internal/verifier` single static artifact; the build command
    (`cmd/verifier-site`); the Pages publish workflow (`.github/workflows/pages.yml`). **Still OPEN on
    the milestone Verify** ("the verifier artifact … [is published at its] published value"): the deploy
    **does not run** — the Pages run at HEAD fails at `Configure Pages` (Pages not enabled / not set to
    "GitHub Actions" source); built-but-undeployed pending a one-time human repo-Settings step. Also
    still open: **NO dossier tier-2 WASM caller** (grep-confirmed: no WASM island in `internal/dossier`);
    and the **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies inclusion math +
    id-binding only — no checkpoint-signature / did:web key resolution — so a malicious monitor can still
    render a green `verified`; the id-binding half of that gap is closed in source AND artifact).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots` serve
    route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 chrome·plumbing·hardening.** Recent arc:
  `cmd/verifier-site` build command → Pages publish workflow → `safeIndex`→`SafeIndex` hardening → WASM
  id-binding bind (NEEDS_WORK, source-only) → rebuild + re-pin verify.wasm (NEEDS_WORK, content skew
  closed but toolchain-unreproducible) → **pin `go=1.26.4` + re-pin to `2c91e61f…` (this iteration,
  PASS — artifact now reproducible from `mise run build:wasm`).** **DRIFT WATCH (amber, easing):** the
  front-of-queue WASM "published" Verify has not formally closed for seven increments, but this is the
  honest tail of a real convergence — the closer kept shifting one layer deeper (source → artifact
  content → artifact reproducibility) and each layer is now genuinely landed. The remaining open half of
  the criterion (the live Pages deploy) is **human-blocked, not code-blocked** (a workflow file cannot
  self-enable Pages), so the loop cannot autonomously close it. The next code-closable WASM target is
  the signature-half trust gap (design-first) or the dossier WASM caller; otherwise the loop should
  pivot off WASM to OTS/§5/SSR-parity work rather than re-polishing a human-blocked criterion.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `193878c..HEAD` diff (confined to
`internal/web/web.go` + `mise.toml` + the wasm artifact + context/docs). All M1 Verify criteria remain
satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze +
alert-once + restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); 23 internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1` in `go.mod` (the language directive; the *build toolchain*
  is now `mise.toml` `go = "1.26.4"`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.
  `transparency-dev/merkle` backs `internal/proof/verify`, which `cmd/wasm`/`verifyadapter` reuse.

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core); `GET
/` realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + chrome complete; the M-UI exit visual-pass + human sign-off is pending.**
No M-UI surface (`internal/dashboard`/`dossier`/`certificate`/`proofserve`) was touched by this diff —
carried forward. All six certificate clauses (§1–§6) + both anchor panels + badge + DS shell + `/`
index + log browser + hub dossier (with the shared chrome masthead + `← Realm index` back-link) +
frozen Exhibit + record list + single-record page + ISCC-IDv1 decoder + Hub-List resolver +
proof-bundle endpoint render and pass the behavioral HTTP-seam Verify; every SSR masthead carries the
shared chrome (self-hosted logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the dossier
  but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still lack the
  full instance-identity block + back-link chain on every surface); `/` sub-region deltas
  (config-driven instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer)
  filed `normal`; the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this browser
  re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest to §2's
  root; `did:web:` + raw `data.Domain` rides §4 AND the proof bundle (mis-renders a `host:port` DID);
  `hubDomain` fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN, but the reproducibility critical is CLOSED and HEAD is PASS.** The
id-binding half of the verifier-scope trust gap is closed IN SOURCE **and ARTIFACT**, and the artifact
is now **reproducible from its documented command**: the committed `internal/web/verify.wasm` carries
the 6-arg/id-binding shim (`verifyadapter.RecordCommitsID`), its SHA-256 (`2c91e61f…`) matches the
re-pinned `WasmVerifyHash`, and a documented `mise run build:wasm` under `mise.toml` `go = "1.26.4"`
deterministically re-emits those exact bytes (review ran it twice, tree clean). The Pages deploy still
FAILS at `Configure Pages` (human repo-Settings step). NO dossier WASM caller; the SIGNATURE half of
the verifier-scope gap remains (design-first). OTS — both observable HTTP halves landed; only a real
Bitcoin confirmation remains (offline-unprovable).
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  shim accepts 5-or-6 args (`if len(args) != 5 && len(args) != 6`), calls `verifyadapter.SafeIndex` at
  its leaf-index/size sites, and when 6 args are passed gates `verified` on
  `verifyadapter.RecordCommitsID(record, args[5].String())`. The pure `cmd/wasm/verifyadapter` exposes
  `SafeIndex`, `VerifyJSON`, and `RecordCommitsID` — golden-tested by `TestVerifyJSON` + `TestSafeIndex`
  + `TestRecordCommitsID` (mutation-proven across all three RecordCommitsID branches). The committed
  `verify.wasm` is byte-correct for these AND reproducible from `mise run build:wasm`.
  **Reproducibility contract (the closed critical, verified this assessment):**
  `WasmVerifyHash = "2c91e61f…"` (`internal/web/web.go:93`) is the SHA-256 of the artifact built by the
  documented `mise run build:wasm` under the exact-patch toolchain `mise.toml` `go = "1.26.4"`; the
  committed wasm embeds `go1.26.4` (verified: `strings … | grep -oE 'go1.26.[0-9]+'` → `go1.26.4`). The
  prior gap — pin = bare-`go-1.26.1` build vs documented command = go1.26.4 — is gone: option (a) was
  taken (pin the gate runner's toolchain forward, rebuild, re-pin), so artifact == pin ==
  documented-command output. `TestWasmVerifyHashPinned` green.
  `cert.html` embeds the tier-2 island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` (5-arg caller).
  `internal/verifier` is a SINGLE STATIC artifact (cross-origin `?monitor=&id=` loader).
  `cmd/verifier-site` is the reproducible BUILD command (copy-not-rebuild of the pinned wasm).
  `.github/workflows/pages.yml` is the PUBLISH workflow. `verifier.Handler` is NOT mounted in
  `cmd/iscc-monitor` (grep-confirmed). **Still 1/1 OPEN on the milestone Verify** — the deploy is not
  live (Pages `failure` at `Configure Pages` at HEAD); the "published value" half needs the human
  repo-Settings step, which a workflow cannot self-enable.
  **Carried `normal` defect (NOT fixed):** the WASM verifier core verifies inclusion math + id-binding
  only — NO checkpoint-signature check, NO did:web key resolution — so a malicious cross-origin monitor
  can still render a green `verified`; the success copy (`verifier.html:449,631`) overstates a
  signature/key check that never runs. Design-first remainder.
  **Carried `normal` (Surface-C):** `readTarget` (`verifier.html:550-553`) accepts opaque-scheme monitor
  forms (`https:example.com`) the Go `parseTarget` rejected — fix = return `u.href` not raw.
  **Carried `normal`:** the Pages custom-domain / GitHub-Actions-source enablement gap (one-time human
  repo-Settings step; artifact CNAME is a no-op under Actions).
  **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the calendar
  transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer not yet
  built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation. Still
  1/1 open. **Open `normal` defect (NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the
  symmetric guards the upgrade path got via `safeUpgrade`); the next stamp-path touch should add
  `safeStamp`.

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`8496579`) == `origin/develop`; the Pages
PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate defect). Latest `review`
verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build
  toolchain `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run
  check` green at HEAD (all 27 packages build + vet + test; `gofmt -l .` empty);
  `TestWasmVerifyHashPinned` green; `mise run build:wasm` byte-reproducible (reviewer ran it twice, tree
  clean).
- **Latest `review` verdict: PASS (loop CONTINUE)** for the toolchain pin + re-pin. The reviewer
  reproduced the artifact from `mise run build:wasm` (byte-equal, twice), confirmed the embedded
  `go1.26.4` stamp + intact 6-arg content markers, ran the full gate green, and Codex corroborated
  clean. The reproducibility `critical` was deleted from `issues.md` after the fix was verified.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`. HEAD (`8496579`) is pushed; CI run 27945760457 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27945760486) at HEAD
  is **`failure`** — the `render verifier-site` job fails at `Configure Pages` ("verify that the
  repository has Pages enabled and configured to build using GitHub Actions"); `deploy to
  monitor.iscc.codes` skipped. One-time human repo-Settings step (filed `normal`), not a code/gate
  defect.
- **Open issues: 0 critical, 10 normal, 10 low** (the lone "critical" grep hit is the format-template
  legend on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
  The 10 normal: the Pages custom-domain/enablement gap, certificate §5 digest binding, OTS `safeStamp`
  guard, `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the
  certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region parity deltas, the WASM
  verifier-scope SIGNATURE-half gap, and the Surface-C `readTarget` opaque-URL permissiveness. (Tally
  moved from "1 critical / 9 normal" last assessment to "0 critical / 10 normal" — the reproducibility
  critical closed; no normal closed this cycle.)

## Next Milestone
**Reproducibility critical is closed and pushed; CI green. Resume the front-of-queue WASM-verifier
upgrade milestone — but its remaining "published" half is human-blocked, so prioritize a code-closable
target.** In order:
1. **The signature half of the verifier-scope gap** (the milestone's actual trust bar) — browser
   did:web resolution + checkpoint-note signature verify, gating `verified` on signature + id-binding +
   inclusion. `review` recommends a **design-first pass** before building (browser-side did:web
   resolution is non-trivial); a good STOP-candidate if the design is unclear. Until it lands, do NOT
   loosen `verifier.html`'s "hub-signed root" success copy. And/or wire the tier-2 WASM caller into the
   **hub dossier** (no WASM island today).
2. **Unblock + verify the Pages deploy** (needs the one-time human repo-Settings step: Settings → Pages
   → source "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow
   and confirm a `success` deploy — that closes the "published" half of the WASM Verify criterion. A
   workflow file alone cannot self-enable Pages, so the loop cannot fully close this autonomously — flag
   for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are next
edited; the named-region + `←` back-link parity pass across the remaining SSR surfaces; the M-UI exit
visual-pass + human sign-off (ADR-0012).
