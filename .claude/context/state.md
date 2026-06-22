<!-- assessed-at: f5e8f59b9e248e231b57daff3cd7e5a309137eb8 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM milestone in progress — the id-binding half of the verifier-scope trust gap is now closed **in source** (`verifyadapter.RecordCommitsID` + a 6-arg `isccVerifyInclusion` shim), but the committed byte-pinned `internal/web/verify.wasm` was **NOT rebuilt** — it still carries the old 5-arg shim. The source and the shipped artifact contradict each other: `verifier.html:628` now calls with 6 args against a wasm that rejects anything but 5, so the live Surface-C verifier would render `error` for **every** target. The latest `review` verdict is **NEEDS_WORK (loop CONTINUE)**; HEAD is 4 commits ahead of `origin/develop` and **unpushed**. The Surface-C Pages deploy is still blocked at `Configure Pages`, and OTS still needs a real Bitcoin-confirmed transit.

Incremental review against assessed-at `8efb714`. The only production source the
`8efb714..HEAD` diff touched is the WASM verifier id-binding work: `cmd/wasm/verifyadapter`
gained `RecordCommitsID` (+ a 6-case `TestRecordCommitsID`), `cmd/wasm/main.go`'s shim now accepts
5-or-6 args and gates `verified` on the id binding, and `verifier.html` passes `target.id` as the 6th
arg. Everything else in the diff is `.claude/context/*` docs. All M1/M2/M3/M-UI/OTS production source
is untouched — those sections carry forward met/open as before. **The artifact was not rebuilt**:
`strings internal/web/verify.wasm | grep -c "expected 5 or 6 args"` → 0 (old shim: `"expected 5
args"` → 1; no `"decode record envelope"`), last touched at `196c1e8`. **CI/Pages reflect
`origin/develop` (8efb714), not HEAD** — the 4 HEAD commits are unpushed (correct for a NEEDS_WORK
verdict): CI `success`, Pages `failure` at `Configure Pages`. WASM + OTS Verify still open → DONE not
reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Built/byte-pinned `.wasm` at `/_ds/verify.wasm` (`WasmVerifyHash`,
    `TestWasmVerifyHashPinned`); certificate SSR tier-2 caller (live-verified); Surface-C
    `internal/verifier` resolves its target **client-side** as a single static artifact; the
    reproducible **build command** (`cmd/verifier-site`); the **Pages publish workflow**
    (`.github/workflows/pages.yml`). **Still OPEN on the milestone Verify** ("the verifier artifact …
    [is published at its] published value"): (a) the deploy **does not run** — the Pages run at
    `origin/develop` fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions"
    source); built-but-undeployed pending a one-time human repo-Settings step; (b) **source/artifact
    skew at HEAD** — the committed `verify.wasm` is stale (old 5-arg shim) while the source + the page
    call 6 args, so even once deployed the verifier renders `error` for every target until `mise run
    build:wasm` + re-pin `WasmVerifyHash`. Also still open: **NO dossier tier-2 WASM caller**
    (grep-confirmed: no WASM island in `internal/dossier`); and the **cross-origin verifier-scope
    SIGNATURE gap** (the WASM core verifies inclusion math + id-binding only — no checkpoint-signature
    / did:web key resolution — so a malicious monitor can still render a green `verified`; the
    id-binding half of that gap is now closed in source).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots`
    serve route + certificate §5 anchor render). What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 chrome·plumbing·hardening.** Recent arc:
  Surface-C client-side gating → `cmd/verifier-site` build command → Pages publish workflow →
  `safeIndex`→`SafeIndex` test-hardening → **the WASM id-binding bind (this iteration, NEEDS_WORK).**
  **DRIFT WATCH (amber, holding):** the front-of-queue WASM "published" Verify has not closed for
  five increments. This iteration was an actual milestone-Verify move (it narrows the verifier-scope
  trust gap by closing its id-binding half — real trust-bar progress, not plumbing), but it shipped
  the source/artifact skew below, so it did not land a closer. The next increment should close the
  skew first (one-command rebuild + re-pin — the NEEDS_WORK gate), then either unblock+verify the
  Pages deploy (needs the human step) or pivot to a code-closable WASM criterion (the dossier WASM
  island, or the signature half of the verifier-scope gap — design-first per `review`).

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `8efb714..HEAD` diff (confined to
`cmd/wasm` + `internal/verifier` + context/docs). All M1 Verify criteria remain satisfied:
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
**Status**: **WASM — milestone OPEN, NEEDS_WORK at HEAD: the id-binding half of the verifier-scope
trust gap is now closed IN SOURCE (`verifyadapter.RecordCommitsID` + 6-arg `isccVerifyInclusion`
shim, `verifier.html:628` passes `target.id`), but the committed `internal/web/verify.wasm` was NOT
rebuilt — it still carries the old 5-arg shim, so against the new 6-arg call the live Surface-C page
renders `error` for EVERY target until `mise run build:wasm` + re-pin. The Pages deploy also still
FAILS at `Configure Pages` (human repo-Settings step). NO dossier WASM caller; the SIGNATURE half of
the verifier-scope gap remains (design-first). OTS — both observable HTTP halves landed; only a real
Bitcoin confirmation remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  shim now accepts 5-or-6 args (`if len(args) != 5 && len(args) != 6`), calls
  `verifyadapter.SafeIndex` at its leaf-index/size sites, and when 6 args are passed gates `verified`
  on `verifyadapter.RecordCommitsID(record, args[5].String())`. The pure
  `cmd/wasm/verifyadapter` exposes `SafeIndex` (JS→Go integer guard), `VerifyJSON` (base64-decode →
  `verify.VerifyInclusion`), and the new `RecordCommitsID` (canonicalizes + byte-compares the
  record's committed `iscc_id` to the requested id; three-way verdict: mismatch → `failed`, parse
  fault → `error`) — golden-tested by `TestVerifyJSON` + `TestSafeIndex` + `TestRecordCommitsID`
  (mutation-proven in the latest review across all three RecordCommitsID branches). **The 5-or-6-arg
  design is a sound, flagged deviation** — a hard `!= 6` would have regressed the certificate's live
  5-arg tier-2 caller (`cert.html:565`); keep the optional-arg posture until a later increment pairs
  a strict guard with a cert.html 6th-id edit in the SAME step.
  **CRITICAL SKEW (NEEDS_WORK gate, new `normal`):** the committed `internal/web/verify.wasm` is
  STALE — `strings … | grep -c "expected 5 or 6 args"` → 0, `"expected 5 args"` → 1, no `"decode
  record envelope"`; last touched at `196c1e8`, NOT this commit. The source + the page call 6 args;
  the deployed module rejects > 5 → every live Surface-C verification returns `error`. Fix is one
  reproducible command (`mise run build:wasm`) + re-pin `web.WasmVerifyHash` (currently
  `7d57ab1b…`; review verified a clean rebuild yields `96b2a40d…` carrying the new shim).
  `mise run check` CANNOT catch this skew (it builds to `/tmp`; `TestWasmVerifyHashPinned` only
  confirms the committed bytes match the committed hash — both stale → green).
  `cert.html` embeds the tier-2 island + `/_ds/wasm_exec.js` + `/_ds/verify.wasm` (5-arg caller,
  unaffected by the skew today). `internal/verifier` is a SINGLE STATIC artifact (cross-origin
  `?monitor=&id=` loader). `cmd/verifier-site` is the reproducible BUILD command (copy-not-rebuild of
  the pinned wasm). `.github/workflows/pages.yml` is the PUBLISH workflow. `verifier.Handler` is NOT
  mounted in `cmd/iscc-monitor` (grep-confirmed). **Still 1/1 OPEN on the milestone Verify** — the
  deploy is not live (Pages `failure` at `Configure Pages` on `origin/develop`) AND, separately, the
  artifact skew would break the deployed page even once Pages is enabled.
  **Carried `normal` defect (NOT fixed):** the WASM verifier core verifies inclusion math + id-binding
  only — NO checkpoint-signature check, NO did:web key resolution — so a malicious cross-origin monitor
  can still render a green `verified`; the success copy (`verifier.html:449,631`) overstates a
  signature/key check that never runs. Design-first remainder.
  **Carried `normal` (Surface-C):** `readTarget` (`verifier.html:550-553`) accepts opaque-scheme
  monitor forms (`https:example.com`) the Go `parseTarget` rejected — NOT a trust defect, fix = return
  `u.href` not the raw `monitor`.
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
workflow is `failure` (Configure Pages, human-step). HEAD is NEEDS_WORK + 4 commits UNPUSHED — the
HEAD increment is internally inconsistent (stale wasm vs 6-arg source/page).**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Latest `review` reported `mise run check` green at HEAD (all 27 packages ok; `gofmt -l .` empty),
  plus `GOOS=js GOARCH=wasm go build ./cmd/wasm` and `./cmd/wasm/verifyadapter` both exit 0, and
  `TestRecordCommitsID` (6 cases) + `TestSafeIndex` + `TestVerifyJSON` green. **But `mise run check`
  green does NOT prove the increment correct** — it cannot see the stale-wasm skew (see WASM section).
- **Latest `review` verdict: NEEDS_WORK (loop CONTINUE)** for the id-binding bind. The Go work is
  excellent + mutation-proven, but the committed pinned `verify.wasm` was not rebuilt, so the
  step's user-visible goal (the browser gates `verified` on id-binding) is NOT achieved by the
  committed tree — the deployed verifier would render `error` for every target. Codex P1 confirmed.
  One new `normal` filed (the artifact skew); the verifier-scope signature issue narrowed (id-binding
  half closed). Not pushed (NEEDS_WORK).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle
  on push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`. **HEAD (`f5e8f59`) is 4 commits ahead of `origin/develop` (`8efb714`) and unpushed**, so
  the latest CI run (27942869775, `success`) is at `8efb714`, NOT HEAD — the HEAD commits are
  CI-untested by design (NEEDS_WORK held back).
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27942869797) at
  `8efb714` is **`failure`** — `Configure Pages` errors "Get Pages site failed … verify that the
  repository has Pages enabled and configured to build using GitHub Actions"; `upload` + `deploy`
  skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 real `critical`, 11 `normal`, 10 `low`** (the lone "critical" grep hit is the
  issues.md format-legend line 9, not a real issue). DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE. The 11 normal: the NEW stale-`verify.wasm` skew, the Pages
  custom-domain/enablement gap, certificate §5 digest binding, OTS `safeStamp` guard, `hubDomain`
  ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, the certificate tier-2
  no-JS honesty-copy overstatement, the `/` sub-region parity deltas, the WASM verifier-scope
  SIGNATURE-half gap, and the Surface-C `readTarget` opaque-URL permissiveness.

## Next Milestone
**Close the WASM stale-artifact skew first — it is the NEEDS_WORK gate at HEAD.** The id-binding
source is correct and mutation-proven, but the pinned `internal/web/verify.wasm` was not rebuilt, so
the committed tree is self-contradictory (6-arg source/page vs 5-arg artifact). The fix is one
reproducible command + re-pin:
1. **Rebuild + re-pin (the NEEDS_WORK fix).** `mise run build:wasm` to regenerate
   `internal/web/verify.wasm` with the 6-arg/id-binding shim, then re-pin `web.WasmVerifyHash` to the
   emitted SHA-256 (`TestWasmVerifyHashPinned` gates it; review verified the rebuild yields
   `96b2a40d…`). Confirm `strings internal/web/verify.wasm | grep -c "expected 5 or 6 args"` → 1.
   This makes the live verifier actually gate on id-binding and completes the increment's goal.

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
