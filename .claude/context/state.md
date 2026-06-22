<!-- assessed-at: 62d5a2f18c505a184854569b4d75ac821302753e -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone in progress — the open front is the Surface-C (`monitor.iscc.codes` Independent Verification app) GitHub-Pages deploy. The reproducible BUILD command for that deploy now exists (`cmd/verifier-site` renders the full static tree), but no Pages PUBLISH workflow exists yet, so the artifact is built-but-undeployed. Remaining WASM Verify work: the deploy workflow, the dossier tier-2 WASM caller, and the verifier-scope signature/id gap. OTS still needs a real Bitcoin-confirmed transit.

Incremental review against assessed-at `47f4605`. The ONLY source change since is the NEW package
`cmd/verifier-site` (`main.go` + `main_test.go`) plus the one allowed `CLAUDE.md` doc edit and
context/docs. `cmd/verifier-site` is a thin, build-time static-site GENERATOR: `main` owns the single
`os.Exit`; the testable `generate` drives the REAL `verifier.Handler` (→ `index.html`) and
`web.Handler` (→ every `/_ds/` asset, fonts enumerated from the served `fonts.css`) over `httptest`
and writes the byte-pinned tree to an out dir — one source of truth, no template re-embed, no
network/DB/`GOOS=js` build. It fails closed (a non-200 from any handler aborts) and copies the
byte-pinned `verify.wasm`. It is a leaf (only `internal/verifier` + `internal/web`) and does NOT mount
a server. M1/M2/M3/M-UI/WASM-core/OTS source were untouched — all carry forward met/open as before.
The latest `review` verdict is PASS_WITH_NOTES (loop CONTINUE), CI is `success` at current HEAD
`62d5a2f` (== `origin/develop`), and there is no `critical` issue — but the WASM and OTS Verify
criteria are still open and 10 `normal` issues remain, so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Built/byte-pinned `.wasm` at `/_ds/verify.wasm` (`WasmVerifyHash`,
    `TestWasmVerifyHashPinned`); first SSR tier-2 caller on the **certificate** (live-verified);
    Surface-C `internal/verifier` resolves its target **client-side** as a single static artifact;
    and now the reproducible **build command** for the deploy exists (`cmd/verifier-site` renders the
    full static tree — `index.html` + every `/_ds/` asset — into an out dir, fail-closed, wasm-hash
    verified). `internal/verifier` remains **deliberately unmounted** in `cmd/iscc-monitor`
    (grep-confirmed). Still OPEN on the milestone Verify: **the GitHub-Pages / `monitor.iscc.codes`
    PUBLISH workflow** (only `.github/workflows/ci.yml` exists — no Pages deploy workflow; the BUILD
    command that workflow will invoke has landed, the workflow itself has not); **NO dossier tier-2
    WASM caller** (grep-confirmed: no `verify.wasm`/`wasm_exec`/`isccVerify` in `internal/dossier` —
    the dossier's `verify ↗ monitor.iscc.codes` is a static link, not a WASM island); and the
    **cross-origin verifier-scope gap** (the WASM core verifies inclusion math only — no
    checkpoint-signature / id-binding — so a malicious monitor can render a green `verified`).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots`
    serve route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 chrome·plumbing·hardening.** Recent
  arc: Surface-C skeleton → Surface-C LIVE wiring (advances WASM) → dossier tier-2 chrome (M-UI
  nav-closure) → Surface-C client-side target gating (closed the static-deploy `.HasTarget` blocker)
  → **the Surface-C static-site generator `cmd/verifier-site` (this iteration — the reproducible
  BUILD command the Pages deploy will invoke; advances the WASM milestone toward a deployable
  artifact).** **DRIFT WATCH (amber, holding):** the last two increments both targeted the WASM
  Surface-C deploy chain — good, they are on the front-of-queue open Verify — but neither has yet
  CLOSED the milestone Verify criterion (the deploy is not live; the artifact is built but
  unpublished). The build command is in place and the only remaining step to a live artifact is the
  Pages publish workflow. The next increment should land that workflow (or the dossier WASM island)
  and actually close a Verify criterion, not add further build plumbing around the still-unpublished
  artifact.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `47f4605..HEAD` diff (confined to
the new `cmd/verifier-site` + context/docs). All M1 Verify criteria remain satisfied: `origin`/`vkey`
golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart
survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+ `cmd/wasm/verifyadapter`);
  23 internal packages — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower,
  healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry,
  store, tiles, tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
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
(`internal/verifier`) resolves `?monitor=`/`?id=` CLIENT-side as a single static artifact; and the
reproducible BUILD command for the deploy now exists (`cmd/verifier-site`, fail-closed, wasm-hash
verified) — but the Pages PUBLISH workflow is still absent, so the artifact is built-but-undeployed;
NO dossier WASM caller (static verify-link only); the deploy workflow and verifier-scope signature/id
gaps remain. OTS — both observable HTTP halves landed; only a real Bitcoin confirmation remains
(offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  pure `cmd/wasm/verifyadapter.VerifyJSON` base64-decodes the bundle into `verify.VerifyInclusion`.
  `cert.html` embeds the tier-2 proof island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` loader and
  shows the WASM verdict only when §3 passed. `internal/verifier` is a SINGLE STATIC artifact (the
  `Handler` execs `nil` data; the always-emitted loader reads `?monitor=&id=`, fetches
  `<monitor>/inclusion/<id>.bundle`, runs `isccVerifyInclusion`, and gates the verdict on a real
  re-verification). **NEW this iteration:** `cmd/verifier-site` is the reproducible BUILD command —
  `generate(outDir)` drives `verifier.Handler` (→ `index.html`) + `web.Handler` (→ `/_ds/` tree,
  fonts enumerated from `fonts.css`) over `httptest`, writes a 14-file tree, copies the byte-pinned
  `verify.wasm` (hash == `web.WasmVerifyHash`, review-verified), and fails closed on any non-200
  (mutation-proven by `TestGenerate`). It is a leaf (only `internal/verifier` + `internal/web`; no
  store/DB/follower; `net/http` only via `httptest`) and is NOT a server — it does not mount
  `verifier.Handler` in `cmd/iscc-monitor` (grep-confirmed). **Still 1/1 open on the milestone
  Verify:** the GitHub-Pages / `monitor.iscc.codes` PUBLISH workflow does not exist (only `ci.yml`;
  no Pages deploy step — `cmd/verifier-site` is its build command, not the deploy); NO dossier WASM
  tier-2 caller (grep-confirmed — the dossier's verify-link is static); identical-verdict (WASM vs
  server) parity exercised end-to-end live (golden-tested as markup only).
  **Carried `normal` defects (NOT fixed):** (a) `safeIndex` is a pure `float64→(uint64,string)` fn
  trapped in the tagged `main.go` with NO executable test — move it into the untagged `verifyadapter`
  and table-test the reject branches; (b) the WASM verifier core proves inclusion math ONLY (no
  checkpoint-signature check, no id-binding), so a malicious cross-origin monitor can render a green
  `verified` — same scope the certificate tier-2 already ships, more acute on Surface C; the success
  copy overstates it; (c) Surface-C `readTarget` (`verifier.html:550-553`) accepts opaque-scheme
  monitor forms (`https:example.com`) the Go `parseTarget` rejected — NOT a trust defect (the browser
  resolves to the same host, WASM re-verifies the bundle), fix = return `u.href` not the raw
  `monitor`.
  **New `low` (this iteration):** `cmd/verifier-site` `generate` writes non-atomically — `index.html`
  is written before the asset loop, so a mid-run error leaves a partial tree in a reused out dir; the
  run-level fail-closed (errors → `os.Exit(1)`) is intact, only the output dir is half-written. Filed
  `low` (Codex P3); fix = stage to a temp dir + rename, or buffer all responses before the first write.
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
  Review reported `mise run check` green at HEAD (all 27 packages ok incl. `cmd/verifier-site`;
  `gofmt -l .` empty); `GOOS=js GOARCH=wasm go build ./cmd/wasm` OK.
- **Latest `review` verdict: PASS_WITH_NOTES (loop CONTINUE)** for the `cmd/verifier-site` generator.
  Tight diff (1 non-test/doc production file + 1 test + the one allowed `CLAUDE.md` + docs), all gates
  green, fail-closed contract mutation-proven (handler-404 → `generate` aborts), generated tree =
  14 files, `verify.wasm` SHA-256 == `web.WasmVerifyHash`, no-CDN re-asserted on the live output,
  dep closure verified leaf, `verifier.Handler` still NOT mounted in `buildMux`. Visual check N/A (no
  SSR template touched).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle
  on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`, in sync with `origin/develop` (both at `62d5a2f`). **Latest CI run is `success` at
  current HEAD** (run 27940905908). **No GitHub-Pages publish workflow exists yet** — only `ci.yml`.
- **Open issues: 0 `critical`, 10 `normal`, 10 `low`.** (The lone "critical" grep hit is the issues.md
  format-legend line, not a real issue.) DONE requires 0 critical AND 0 normal, so the loop stays
  CONTINUE. The 10 normal span: certificate §5 digest binding, OTS `safeStamp` guard, `hubDomain`
  ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the `safeIndex` test
  gap, the certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region parity deltas, the
  WASM verifier-scope signature/id-binding gap, and the Surface-C `readTarget` opaque-URL permissiveness.

## Next Milestone
**Continue the WASM milestone — it is the front-of-queue open Verify.** The reproducible BUILD command
(`cmd/verifier-site`) now exists, so the natural next closer is the **GitHub-Pages / `monitor.iscc.codes`
PUBLISH workflow** itself: add a `.github/workflows/*` that runs `go run ./cmd/verifier-site -out <dir>`
and deploys the tree to Pages (no such workflow exists today). The alternative next sub-step is wiring
the tier-2 WASM caller into the **hub dossier** (no WASM island in `internal/dossier` today; its chrome
shell exists) — mirror the certificate/verifier data-island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm`
loader. At a WASM-verifier-scope touch, expand the core to verify the checkpoint signature against the
hub's did:web key + bind the record to the requested id (the cross-origin trust-path gap), move
`safeIndex` into the untagged `verifyadapter` with table-tested reject branches, fold in the
`readTarget` `u.href` normalization, and (when `cmd/verifier-site` is next edited) stage-and-rename for
atomic output.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are
next edited; carry the named-region + `←` back-link parity pass across the remaining SSR surfaces
(log browser / single record / certificate); the M-UI exit visual-pass + human sign-off (ADR-0012).
