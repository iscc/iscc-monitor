<!-- assessed-at: 8efb7141583588ff6b7530eeaabfcd48b516501a -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone in progress — Surface-C (`monitor.iscc.codes` Independent Verification app) deploy chain is code-complete (build command + Pages publish workflow both landed), but the Pages deploy at HEAD still **FAILS** at `Configure Pages` (repo Pages not enabled / not set to the GitHub-Actions source), so the artifact stays built-but-**undeployed**. Closing the WASM "published" Verify criterion now hinges on a one-time human repo-Settings step, not on more code. OTS still needs a real Bitcoin-confirmed transit.

Incremental review against assessed-at `af6ddf2`. The ONLY source change since is the WASM
marshaling-boundary relocation: `safeIndex`/`maxSafeInteger` moved byte-for-byte out of the
build-tagged `cmd/wasm/main.go` into the untagged, WASM-pure `cmd/wasm/verifyadapter` as exported
`SafeIndex`, with `main.go`'s two call sites re-pointed and a table-driven `TestSafeIndex` added that
now runs on every linux `go test` (`git diff af6ddf2..HEAD --stat`: 2 Go production files + 1 test
file + context/docs; NO other Go production file changed). All M1/M2/M3/M-UI/OTS source untouched —
those sections carry forward met/open as before. The latest `review` verdict is **PASS (loop
CONTINUE)** and it closed one `normal` issue (the "`safeIndex` trapped behind the build tag with no
executable test" gap), mutation-reverified across all four reject branches with Codex clean. **CI**
is `success` at HEAD (== `origin/develop`); the **Pages** workflow run at HEAD is still `failure` at
`Configure Pages`. With WASM + OTS Verify still open, DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Built/byte-pinned `.wasm` at `/_ds/verify.wasm` (`WasmVerifyHash`,
    `TestWasmVerifyHashPinned`); certificate SSR tier-2 caller (live-verified); Surface-C
    `internal/verifier` resolves its target **client-side** as a single static artifact; the
    reproducible **build command** (`cmd/verifier-site`); the **Pages publish workflow**
    (`.github/workflows/pages.yml`). **Still OPEN on the milestone Verify** ("the verifier artifact …
    [is published at its] published value"): the deploy **does not actually run** — the Pages run at
    HEAD fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions" source); the
    artifact is built-but-undeployed pending a one-time human repo-Settings step. Also still open:
    **NO dossier tier-2 WASM caller** (grep-confirmed: no `verify.wasm`/`wasm_exec`/`isccVerify` in
    `internal/dossier` — the dossier's `verify ↗` is a static link, not a WASM island); and the
    **cross-origin verifier-scope gap** (the WASM core verifies inclusion math only — no
    checkpoint-signature / id-binding — so a malicious monitor can render a green `verified`).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots`
    serve route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 chrome·plumbing·hardening.** Recent arc:
  Surface-C client-side target gating → `cmd/verifier-site` build command → Pages publish workflow →
  **the `safeIndex`→`SafeIndex` test-hardening relocation (this iteration).** **DRIFT WATCH (amber,
  holding):** the front-of-queue open Verify (the WASM "published" criterion) has NOT closed for four
  increments — the deploy chain is now code-complete, but the deploy still does not run (blocked on
  the human Pages-enable step), and this latest increment was a test-gap hardening fix (a closed
  `normal`), NOT a milestone-Verify closer. The deploy chain is code-complete; the remaining blocker
  is operational. The next increment should either (a) supply the human-step doc + verify the deploy
  succeeds, or (b) pivot to a code-closable WASM criterion (the dossier WASM island, or the
  verifier-scope signature/id check) — not add further plumbing around the still-undeployed artifact.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `af6ddf2..HEAD` diff (confined to
the `cmd/wasm` adapter relocation + context/docs). All M1 Verify criteria remain satisfied:
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
**Status**: **WASM — milestone OPEN: certificate tier-2 caller live-verified; Surface-C
(`internal/verifier`) resolves `?monitor=`/`?id=` CLIENT-side as a single static artifact; the
reproducible BUILD command (`cmd/verifier-site`) AND the Pages PUBLISH workflow
(`.github/workflows/pages.yml`) both exist — but the Pages deploy at HEAD FAILS at `Configure Pages`
(Pages not enabled / not set to GitHub-Actions source), so the artifact is still built-but-undeployed
(one-time human repo-Settings step needed); NO dossier WASM caller (static verify-link only); the
verifier-scope signature/id gap remains. OTS — both observable HTTP halves landed; only a real Bitcoin
confirmation remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion` and
  now calls `verifyadapter.SafeIndex` at its two leaf-index/size call sites; the pure
  `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`, and the
  newly-relocated, untagged `verifyadapter.SafeIndex` is the JS→Go integer guard (table-tested by
  `TestSafeIndex` on linux; mutation-proven across its NaN/Inf, fractional, lower-bound, and
  upper-bound reject branches in the latest review). `cert.html` embeds the tier-2 proof island +
  `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader and shows the WASM verdict only when §3 passed.
  `internal/verifier` is a SINGLE STATIC artifact (the `Handler` execs `nil` data; the always-emitted
  loader reads `?monitor=&id=`, fetches `<monitor>/inclusion/<id>.bundle`, runs `isccVerifyInclusion`,
  and gates the verdict on a real re-verification). `cmd/verifier-site` is the reproducible BUILD
  command — `generate(outDir)` drives `verifier.Handler` (→ `index.html`) + `web.Handler` (→ `/_ds/`
  tree), writes the 14-file tree, copies the byte-pinned `verify.wasm` (hash == `web.WasmVerifyHash`),
  and fails closed on any non-200. `.github/workflows/pages.yml` is the Pages PUBLISH workflow — a
  `build` job (checkout → setup-go 1.26 → `go run ./cmd/verifier-site -out dist` → `cp
  .github/pages/CNAME dist/CNAME` → configure-pages@v5 → upload-pages-artifact@v3) and a `deploy` job
  (`needs: build`, `environment: github-pages`, deploy-pages@v4), on push[develop] +
  workflow_dispatch. It does NOT rebuild the WASM (copy-not-rebuild). `verifier.Handler` remains NOT
  mounted in `cmd/iscc-monitor` (grep-confirmed). **Still 1/1 OPEN on the milestone Verify — the
  deploy is not live:** the Pages run at HEAD (`27942869797`) is `failure` at the `Configure Pages`
  step ("Get Pages site failed … verify that the repository has Pages enabled and configured to build
  using GitHub Actions"), so `upload`/`deploy` are skipped and nothing reaches `monitor.iscc.codes` —
  the one-time human Settings step the review flagged (filed `normal`), still blocking the deploy.
  Also still open: NO dossier WASM tier-2 caller (grep-confirmed); identical-verdict (WASM vs server)
  parity exercised end-to-end live but golden-tested as markup + the `verifyadapter` golden vector.
  **Carried `normal` defect (NOT fixed):** the WASM verifier core proves inclusion math ONLY (no
  checkpoint-signature check, no id-binding), so a malicious cross-origin monitor can render a green
  `verified` — same scope the certificate tier-2 already ships, more acute on Surface C; the success
  copy overstates it.
  **Carried `normal` (Surface-C):** `readTarget` (`verifier.html:550-553`) accepts opaque-scheme
  monitor forms (`https:example.com`) the Go `parseTarget` rejected — NOT a trust defect, fix = return
  `u.href` not the raw `monitor`.
  **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically; the Pages custom-domain
  doc gap (filed `normal` — the artifact CNAME is a no-op under Actions).
  **Closed this iteration:** the `safeIndex` "trapped behind the build tag, no executable test" gap —
  relocated to untagged `verifyadapter.SafeIndex` with a mutation-proven table test.
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
**Status**: **GREEN for the code gate (CI `success` at HEAD); the Pages PUBLISH workflow run is
`failure` — blocked on a one-time repo-Settings step, not a code/gate defect.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Latest `review` reported `mise run check` green at HEAD (all 27 packages ok; `gofmt -l .` empty),
  plus `GOOS=js GOARCH=wasm go build ./cmd/wasm` and `./cmd/wasm/verifyadapter` both exit 0
  (`verifyadapter` stays WASM-pure), and `TestSafeIndex` (8 subtests) + `TestVerifyJSON` green.
- **Latest `review` verdict: PASS (loop CONTINUE)** for the `safeIndex`→`SafeIndex` relocation.
  Tight diff (2 Go production files + 1 test + context), gates green, byte-for-byte port confirmed (no
  bound relaxed), mutation-reverified across all four reject branches, Codex clean, one `normal`
  closed. Visual check N/A (no SSR surface changed).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle
  on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`, in sync with `origin/develop` (both at `8efb714`). **CI run is `success` at HEAD**
  (run 27942869775).
- **Pages**: `.github/workflows/pages.yml` exists and runs on push to develop, but the run at HEAD
  (run 27942869797) is **`failure`** — `Configure Pages` errors "Get Pages site failed … verify that
  the repository has Pages enabled and configured to build using GitHub Actions"; `upload` + `deploy`
  are skipped. This is a one-time human repo-Settings step (enable Pages → "GitHub Actions" source +
  custom domain), already filed as a `normal` issue; it is NOT a code defect and NOT a `mise run
  check` / gate failure.
- **Open issues: 0 `critical`, 10 `normal`, 10 `low`** (the lone "critical" grep hit is the
  issues.md format-legend line, not a real issue). DONE requires 0 critical AND 0 normal, so the loop
  stays CONTINUE. The 10 normal span: the Pages custom-domain/enablement gap, certificate §5 digest
  binding, OTS `safeStamp` guard, `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode,
  certificate §6 timestamp, the certificate tier-2 no-JS honesty-copy overstatement, the `/`
  sub-region parity deltas, the WASM verifier-scope signature/id-binding gap, and the Surface-C
  `readTarget` opaque-URL permissiveness.

## Next Milestone
**Continue the WASM milestone — it is the front-of-queue open Verify.** The deploy chain is now
code-complete (build command + publish workflow), but the deploy **does not run**: the Pages workflow
fails at `Configure Pages` because the repo has not enabled Pages with the "GitHub Actions" source.
Two ways forward, pick one:
1. **Unblock + verify the deploy.** This is primarily the one-time human repo-Settings step (Settings
   → Pages → source "GitHub Actions" + custom domain `monitor.iscc.codes` + the DNS CNAME). The
   code-side follow-up is the deploy-setup doc note (the existing Pages custom-domain `normal`); once
   Pages is enabled, re-run the workflow and confirm a `success` deploy — that actually CLOSES the
   "published" half of the WASM Verify criterion. (A workflow file alone cannot self-enable Pages, so
   the loop cannot fully close this criterion autonomously without the human step.)
2. **Pivot to a code-closable WASM criterion** while the deploy is blocked: wire the tier-2 WASM
   caller into the **hub dossier** (no WASM island in `internal/dossier` today — mirror the
   certificate/verifier data-island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader), and/or expand
   the WASM core to verify the checkpoint signature against the hub's did:web key + bind the record to
   the requested id (the cross-origin trust-path gap), and fold in the `readTarget` `u.href`
   normalization. (The verifier-scope expansion is the milestone's actual trust bar — `review`
   suggests a design-first split before building.)

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are
next edited; carry the named-region + `←` back-link parity pass across the remaining SSR surfaces
(log browser / single record / certificate); the M-UI exit visual-pass + human sign-off (ADR-0012).
