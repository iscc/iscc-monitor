<!-- assessed-at: 193878c36a264da1460f8e2fa68aa4004d0d8788 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone — content skew CLOSED, but a NEW critical reproducibility gap is the NEEDS_WORK gate. The committed `internal/web/verify.wasm` now carries the correct 6-arg/id-binding shim and its SHA-256 (`96b2a40d…`) matches the re-pinned `web.WasmVerifyHash`, so the deployed verifier would no longer render all-`error`. BUT the pin is the output of a **bare `go build` (go1.26.1)**, while the **documented** regeneration command `mise run build:wasm` runs under mise's **go1.26.4** and deterministically emits a different artifact (`2c91e61f…`). The audited-artifact pin is therefore NOT reproducible from its own documented command — `TestWasmVerifyHashPinned` would FAIL on a documented rebuild. Latest `review` verdict is **NEEDS_WORK (loop CONTINUE)**; HEAD is **8 commits ahead of `origin/develop` and unpushed**. Pages deploy still blocked at `Configure Pages`; OTS still needs a real Bitcoin-confirmed transit.

Incremental review against assessed-at `f5e8f59`. The `f5e8f59..HEAD` diff touched exactly ONE
production Go change — `internal/web/web.go` re-pins `WasmVerifyHash` `7d57ab1b…`→`96b2a40d…` — plus a
regenerated `internal/web/verify.wasm` (now the 6-arg shim) and `.claude/context/*` docs. All
M1/M2/M3/M-UI/OTS production source is untouched — those sections carry forward met/open as before.
**Content skew now CLOSED** (verified): `strings internal/web/verify.wasm | grep -c "expected 5 or 6
args"` → 1, `"expected 5 args"` → 0, `"decode record envelope"` → 1; `sha256sum` → `96b2a40d…` ==
the pin. **But the wasm embeds `go1.26.1`** (a bare-go build), not the mise toolchain — the new
critical. CI/Pages reflect `origin/develop` (`8efb714`), not HEAD — the 8 HEAD commits are unpushed
(correct for a NEEDS_WORK verdict): CI `success`, Pages `failure` at `Configure Pages`. WASM + OTS
Verify still open → DONE not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The built/byte-pinned `.wasm` now carries the 6-arg/id-binding shim
    and the pin matches the committed bytes; certificate SSR tier-2 caller (live-verified); Surface-C
    `internal/verifier` single static artifact; the build command (`cmd/verifier-site`); the Pages
    publish workflow (`.github/workflows/pages.yml`). **Still OPEN on the milestone Verify** ("the
    verifier artifact … [is published at its] published value"): (a) the deploy **does not run** — the
    Pages run at `origin/develop` fails at `Configure Pages` (Pages not enabled / not set to "GitHub
    Actions" source); built-but-undeployed pending a one-time human repo-Settings step; (b) **NEW
    reproducibility gap (the NEEDS_WORK gate)** — the pin is reproducible only from a bare `go build`
    (go1.26.1), NOT from the documented `mise run build:wasm` (go1.26.4 → `2c91e61f…`), so the
    audited-artifact contract is broken. Also still open: **NO dossier tier-2 WASM caller**
    (grep-confirmed: no WASM island in `internal/dossier`); and the **cross-origin verifier-scope
    SIGNATURE gap** (the WASM core verifies inclusion math + id-binding only — no checkpoint-signature
    / did:web key resolution — so a malicious monitor can still render a green `verified`; the
    id-binding half of that gap is now closed in source AND artifact).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots`
    serve route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 chrome·plumbing·hardening.** Recent arc:
  `cmd/verifier-site` build command → Pages publish workflow → `safeIndex`→`SafeIndex` hardening →
  WASM id-binding bind (NEEDS_WORK, source-only) → **rebuild + re-pin verify.wasm (this iteration,
  NEEDS_WORK).** **DRIFT WATCH (amber, holding):** the front-of-queue WASM "published" Verify has not
  closed for six increments. This iteration closed the content skew (real trust-bar progress: the
  deployed verifier would now actually gate on id-binding) but surfaced the toolchain-reproducibility
  gap, so it again did not land a closer. The pattern is converging in small correct steps but the
  closer keeps shifting one layer deeper (source → artifact content → artifact reproducibility). Next
  increment should close the reproducibility gap (a one-decision toolchain pin in `mise.toml` + a
  re-pin) — that finally lets the artifact half of the Verify criterion stand; then either unblock +
  verify the Pages deploy (needs the human step) or pivot to a code-closable WASM criterion.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `f5e8f59..HEAD` diff (confined to
`internal/web/web.go` + the wasm artifact + context/docs). All M1 Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once +
restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); 23 internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.
  `transparency-dev/merkle` backs `internal/proof/verify`, which `cmd/wasm`/`verifyadapter` reuse.

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
carries the shared chrome (self-hosted logo + text mark + divider).
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
**Status**: **WASM — milestone OPEN, NEEDS_WORK at HEAD on a NEW reproducibility gap.** The
id-binding half of the verifier-scope trust gap is now closed IN SOURCE **and ARTIFACT**: the
committed `internal/web/verify.wasm` carries the 6-arg/id-binding shim
(`verifyadapter.RecordCommitsID`), its SHA-256 (`96b2a40d…`) matches the re-pinned `WasmVerifyHash`,
and the live verifier would no longer render all-`error` (the prior content-skew NEEDS_WORK is
genuinely CLOSED). **But the pin is unreproducible from its own documented command** — see the new
critical below. The Pages deploy also still FAILS at `Configure Pages` (human repo-Settings step). NO
dossier WASM caller; the SIGNATURE half of the verifier-scope gap remains (design-first). OTS — both
observable HTTP halves landed; only a real Bitcoin confirmation remains (offline-unprovable).
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  shim accepts 5-or-6 args (`if len(args) != 5 && len(args) != 6`), calls `verifyadapter.SafeIndex`
  at its leaf-index/size sites, and when 6 args are passed gates `verified` on
  `verifyadapter.RecordCommitsID(record, args[5].String())`. The pure `cmd/wasm/verifyadapter`
  exposes `SafeIndex`, `VerifyJSON`, and `RecordCommitsID` — golden-tested by `TestVerifyJSON` +
  `TestSafeIndex` + `TestRecordCommitsID` (mutation-proven across all three RecordCommitsID
  branches). The committed `verify.wasm` is now byte-correct for these (content verified this
  assessment).
  **NEW CRITICAL (the NEEDS_WORK gate):** the pin `WasmVerifyHash = "96b2a40d…"`
  (`internal/web/web.go:93`) is the SHA-256 of a **bare `go build` under go1.26.1** (the committed
  artifact embeds `go1.26.1` — verified: `strings … | grep -oE 'go1.26.[0-9]+'` → `go1.26.1`). The
  **documented** regeneration command `mise run build:wasm` runs under mise's resolved toolchain
  (`mise.toml` pins `go = "1.26"`, a floating minor → mise's installed **go1.26.4** — both versions
  verified this assessment: bare `go version` → `go1.26.1`, `mise exec -- go version` → `go1.26.4`)
  and deterministically emits a DIFFERENT artifact (`2c91e61f…`, per the review). So the
  audited-artifact reproducibility contract — the entire point of the pin, and the `web.go` doc claim
  "this value tracks `mise run build:wasm`" — is broken: a documented rebuild yields `2c91e61f…`,
  failing `TestWasmVerifyHashPinned` against the `96b2a40d…` pin. The two artifacts are BEHAVIORALLY
  IDENTICAL (same 6-arg shim + `RecordCommitsID`); CI is not red today (CI runs `mise run check`,
  which only checks committed bytes vs the const; `pages.yml` copies, never rebuilds). Fix is a
  one-decision toolchain pick: (a) pin `mise.toml` to the exact patch `go = "1.26.4"`, rebuild via
  `mise run build:wasm`, re-pin to `2c91e61f…` (preferred — mise is the gate runner); OR (b) pin
  `mise.toml` to `go = "1.26.1"` so the documented command reproduces the committed `96b2a40d…`
  bytes. Either way: artifact == pin == output of `mise run build:wasm`.
  `cert.html` embeds the tier-2 island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` (5-arg caller).
  `internal/verifier` is a SINGLE STATIC artifact (cross-origin `?monitor=&id=` loader).
  `cmd/verifier-site` is the reproducible BUILD command (copy-not-rebuild of the pinned wasm).
  `.github/workflows/pages.yml` is the PUBLISH workflow. `verifier.Handler` is NOT mounted in
  `cmd/iscc-monitor` (grep-confirmed). **Still 1/1 OPEN on the milestone Verify** — the deploy is not
  live (Pages `failure` at `Configure Pages` on `origin/develop`) AND the new reproducibility gap
  defeats the "published value" half of the criterion until artifact == pin == documented-command
  output.
  **Carried `normal` defect (NOT fixed):** the WASM verifier core verifies inclusion math + id-binding
  only — NO checkpoint-signature check, NO did:web key resolution — so a malicious cross-origin monitor
  can still render a green `verified`; the success copy (`verifier.html:449,631`) overstates a
  signature/key check that never runs. Design-first remainder.
  **Carried `normal` (Surface-C):** `readTarget` (`verifier.html:550-553`) accepts opaque-scheme
  monitor forms (`https:example.com`) the Go `parseTarget` rejected — fix = return `u.href` not raw.
  **Carried `normal`:** the Pages custom-domain / GitHub-Actions-source enablement gap (one-time human
  repo-Settings step; artifact CNAME is a no-op under Actions).
  **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically.
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
**Status**: **Code gate GREEN on `origin/develop` (CI `success` at 8efb714); the Pages PUBLISH
workflow is `failure` (Configure Pages, human-step). HEAD is NEEDS_WORK + 8 commits UNPUSHED — the
HEAD increment closed the content skew but is blocked by a new artifact-reproducibility critical.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Latest `review` reported `mise run check` green at HEAD (all 27 packages build + vet + test; `gofmt
  -l .` empty); `TestWasmVerifyHashPinned` green (committed bytes == committed const). **But `mise run
  check` green does NOT prove the increment correct** — it checks only committed-bytes vs const, so it
  CANNOT catch the toolchain-reproducibility gap (a documented `mise run build:wasm` rebuild emits a
  different artifact than the pin).
- **Latest `review` verdict: NEEDS_WORK (loop CONTINUE)** for the rebuild + re-pin. The content skew
  is genuinely closed and the verifier would now work, but the pin is a bare-go-1.26.1 build while the
  documented regeneration command (`mise run build:wasm`, go1.26.4) emits `2c91e61f…` — so the
  audited-artifact pin is not reproducible from its own documented command, the STOP-on-divergence
  condition next.md called out. Codex P2 confirmed (reviewer-reproduced both toolchains + read the
  embedded version stamp). One new critical filed; the prior content-skew normal resolved/deleted.
  Not pushed (NEEDS_WORK).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle
  on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`. **HEAD (`193878c`) is 8 commits ahead of `origin/develop` (`8efb714`) and unpushed**, so
  the latest CI run (27942869775, `success`) is at `8efb714`, NOT HEAD — the HEAD commits are
  CI-untested by design (NEEDS_WORK held back).
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27942869797) at
  `8efb714` is **`failure`** — `Configure Pages` errors "verify that the repository has Pages enabled
  and configured to build using GitHub Actions"; `upload` + `deploy` skipped. One-time human
  repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 1 critical, 9 normal, 11 low** (the lone legend-line "critical" is excluded). DONE
  requires 0 critical AND 0 normal, so the loop stays CONTINUE. The 1 critical: the
  verify.wasm-not-reproducible-from-`mise run build:wasm` gap. The 9 normal: the Pages
  custom-domain/enablement gap, certificate §5 digest binding, OTS `safeStamp` guard, `hubDomain`
  ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the certificate tier-2
  no-JS honesty-copy overstatement, the `/` sub-region parity deltas, the WASM verifier-scope
  SIGNATURE-half gap, and the Surface-C `readTarget` opaque-URL permissiveness. (Tally moved from
  "0 critical / 11 normal" last assessment to "1 critical / 9 normal" — the content-skew normal closed
  and the id-binding-half signature normal narrowed, while the reproducibility critical was filed.)

## Next Milestone
**Close the NEW reproducibility critical first — it is the NEEDS_WORK gate at HEAD.** The content skew
is closed and the verifier would now gate on id-binding, but the pin (`96b2a40d…`, a bare-go-1.26.1
build) is not reproducible from the documented `mise run build:wasm` (go1.26.4 → `2c91e61f…`). The fix
is one toolchain decision + a re-pin:
1. **Pin the toolchain + re-pin the hash (the NEEDS_WORK fix).** Pick ONE (preferred (a), since mise
   is the gate runner): (a) set `mise.toml` `go = "1.26.4"` (exact patch), run `mise run build:wasm`,
   re-pin `web.WasmVerifyHash` to `2c91e61f…`; OR (b) set `mise.toml` `go = "1.26.1"` so the
   documented command reproduces the committed `96b2a40d…`. Verify: `mise run build:wasm && sha256sum
   internal/web/verify.wasm` == `web.WasmVerifyHash`, then `go test -run TestWasmVerifyHashPinned
   ./internal/web` green. This makes artifact == pin == documented-command output and satisfies the
   audited-artifact reproducibility contract.

Then, the front-of-queue WASM Verify work, in order:
2. **Unblock + verify the Pages deploy** (needs the one-time human repo-Settings step: Settings →
   Pages → source "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the
   workflow and confirm a `success` deploy — that closes the "published" half of the WASM Verify
   criterion. A workflow file alone cannot self-enable Pages, so the loop cannot fully close this
   autonomously.
3. **The signature half of the verifier-scope gap** (browser did:web resolution + checkpoint-note
   signature verify, gating `verified` on signature + id-binding + inclusion) — the milestone's actual
   trust bar; `review` recommends a design-first pass before building. And/or wire the tier-2 WASM
   caller into the **hub dossier** (no WASM island today).

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are
next edited; the named-region + `←` back-link parity pass across the remaining SSR surfaces; the M-UI
exit visual-pass + human sign-off (ADR-0012).
