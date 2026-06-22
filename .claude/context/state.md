<!-- assessed-at: ca2b05aa7392f1ec00748a78d0448af2b02eb0f9 -->

# Project State

## Status: IN_PROGRESS

## Phase: Certificate Tier-2 no-JS honesty copy landed (code-closable normal drained) — gate green, pushed.
The certificate `{{if .HasBundle}}` honesty header no longer asserts a present-tense "This browser
re-verifies the proof below" on the no-JS baseline; it now states the always-true offline path
unconditionally and gates the browser re-check on "with JavaScript enabled" (`cert.html:484`), matching
the register of the `#tier2-result` panel. Latest `review` verdict is **PASS (loop CONTINUE)** and HEAD
(`ca2b05a`) is **pushed** — `origin/develop` == HEAD, CI **success** at HEAD. DONE not reached: the WASM
"published"/signature halves and the OTS Bitcoin-confirmed Verify criteria are still open, and 5 `normal`
issues stand.

Incremental review against assessed-at `97ddd57`. The `97ddd57..HEAD` diff touched exactly ONE production
file — `internal/certificate/cert.html` (one honesty-header sentence) — plus its test
(`handler_test.go`, +17 lines: a non-vacuous negative+positive no-JS assertion pair) and
`.claude/context/*` + `learnings/certificate.md` docs. All M1/M2/M3/M-UI/WASM/OTS production source
outside that one template line is byte-unchanged — those sections carry forward met/open as before. **The
"certificate tier-2 honesty header overstates 'This browser re-verifies' on the no-JS baseline" `normal`
is CLOSED** (verified this assessment): the old string is gone (`grep "This browser re-verifies the proof
below" cert.html` → 0), the header now reads "with JavaScript enabled, this browser also re-checks the
proof below", and the close is mutation-proven by `TestCertificateRendersWasmVerifier` (per review:
reverting the sentence trips the new negative assertion). `git status` clean, `origin/develop` == HEAD,
CI run 27950022314 `success` at HEAD.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact: the byte-pinned
    `internal/web/verify.wasm` carries the 6-arg/id-binding shim, the pin (`WasmVerifyHash`) matches the
    committed bytes, and the bytes are reproducible from `mise run build:wasm` under `go=1.26.4`.
    **Still OPEN on the milestone Verify** ("the verifier artifact … [published at its] published value"):
    the Pages deploy **does not run** — the Pages run at HEAD (27950022300) fails at `Configure Pages`
    (Pages not enabled / not set to "GitHub Actions" source); built-but-undeployed, a one-time human
    repo-Settings step. Also still open: **NO dossier tier-2 WASM caller** (no WASM island in
    `internal/dossier`); and the **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies
    inclusion math + id-binding only — no checkpoint-signature / did:web key resolution — so a malicious
    monitor can still render a green `verified`; design-first remainder, STOP-candidate).
  - **OTS anchoring: 1/1 open (carried unchanged).** All three observable HTTP halves are closed (`.ots`
    serve route + the §5 anchor render, digest-bound via `ots.ConfirmedFor`) and BOTH calendar-transport
    guards (`safeUpgrade` + `safeStamp`) are in place. What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~2 milestone-Verify-advancing / ~8 chrome·plumbing·hardening.** Recent arc:
  rebuild+re-pin verify.wasm → pin `go=1.26.4` + re-pin → `safeStamp` OTS guard → §4/bundle `host:port`
  did:web encode → ForceQuery hubDomain reject → §5 OTS digest-binding (PASS) → **certificate Tier-2 no-JS
  honesty copy (this iteration, PASS)**. **DRIFT WATCH (amber):** the front-of-queue WASM "published"
  Verify has not closed for ~12 increments, but its remaining open half (the live Pages deploy) is
  **human-blocked, not code-blocked** — a workflow file cannot self-enable Pages, so the loop cannot
  autonomously close it. The loop continues to correctly *pivot off* the human-blocked WASM criterion to
  drain code-closable `normal`s (the right move per guidance) — this iteration closed a copy-only one. The
  remaining code-closable normals are thinning: the only pure copy/code-closable one left is the
  Surface-C `readTarget` normalization; the rest (WASM signature half, §6 `· at` timestamp, `/` Checkpoint/
  Anchor columns) are design-first or store-schema changes. Watch for the loop running out of cheap
  code-closable work while the milestone-Verify criteria stay human/design-blocked.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `97ddd57..HEAD` diff touched only `internal/certificate/cert.html`;
no M1 source touched. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/
equivocation golden-tested end-to-end with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); 23 internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`,
  `go 1.26.1` language directive in `go.mod`; build toolchain `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core); `GET /`
realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + chrome complete; the M-UI exit visual-pass + human sign-off is pending.**
The only M-UI change this diff was one sentence of certificate honesty-header copy (`cert.html:484`) — it
makes the no-JS Tier-2 claim honest, no layout/region change; every other template is byte-unchanged. All
six certificate clauses (§1–§6) + both anchor panels + badge + DS shell + `/` index + log browser + hub
dossier (with the shared chrome masthead + `← Realm index` back-link) + frozen Exhibit + record list +
single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the
behavioral HTTP-seam Verify; every SSR masthead carries the shared chrome (self-hosted logo + text mark
+ divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the dossier
  but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still lack the full
  instance-identity block + back-link chain on every surface); `/` sub-region deltas (config-driven
  instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`; the
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate §6 rows omit the per-record `· at` timestamp
  (store-schema change); the `/` sub-region parity deltas. (The certificate tier-2 no-JS honesty-copy
  `normal` is now CLOSED this iteration; the "§5 digest-binding" `normal` closed previously.)

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN (published half human-blocked, signature half design-blocked),
reproducibility critical CLOSED, HEAD is PASS. OTS — §5 render digest-bound + both calendar-transport
guards landed; only a real Bitcoin confirmation remains (offline-unprovable).** Neither WASM nor OTS
source was touched by this diff; both sections carry forward.
- **WASM (carried):** the id-binding half of the verifier-scope trust gap is closed IN SOURCE and ARTIFACT,
  and the artifact is reproducible from its documented command: the committed `internal/web/verify.wasm`
  carries the 6-arg/id-binding shim (`verifyadapter.RecordCommitsID`), its SHA-256 matches `WasmVerifyHash`
  (`internal/web/web.go`), and `mise run build:wasm` under `mise.toml` `go = "1.26.4"` deterministically
  re-emits those bytes (`TestWasmVerifyHashPinned` green). `cmd/wasm/main.go` (tagged `//go:build js &&
  wasm`) registers `isccVerifyInclusion`; `verifyadapter` exposes `SafeIndex`, `VerifyJSON`,
  `RecordCommitsID` (golden + mutation-tested); `cert.html` embeds the tier-2 island; `internal/verifier`
  is a SINGLE STATIC cross-origin artifact; `cmd/verifier-site` is the reproducible build command;
  `.github/workflows/pages.yml` is the PUBLISH workflow; `verifier.Handler` is NOT mounted in
  `cmd/iscc-monitor`. **Still 1/1 OPEN on the milestone Verify** — the deploy is not live (Pages
  `failure` at `Configure Pages` at HEAD, run 27950022300), needs the one-time human repo-Settings step.
  **Carried `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a malicious
  cross-origin monitor can render green `verified`; success copy overstates a key check that never runs) —
  design-first remainder / STOP-candidate; Surface-C `readTarget` accepts opaque-scheme monitor forms; the
  Pages custom-domain / Actions-source enablement gap. **Carried `low`:** `cmd/verifier-site` `generate`
  writes non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause (digest-bound via
  `ots.ConfirmedFor`), the store layer (`internal/store/ots.go`), the off-path stamp/upgrade loop
  (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier (`internal/ots.{Confirmed,
  ConfirmedFor}` sharing one private `classify`), and the calendar transport (`internal/otsclient`, with
  BOTH `safeUpgrade` + `safeStamp` guards) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer
  not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation.
  Still 1/1 open. **Carried `low` defect (NOT fixed):** nil-Stamper + empty-OTSBytes row falls through to
  the Upgrader instead of being left untouched (`otsloop.go:144`; docstring-vs-code mismatch on the
  test-only nil path; production always wires a non-nil Stamper).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`ca2b05a`) == `origin/develop` (run 27950022314);
the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate defect). Latest
`review` verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build toolchain
  `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run check` green
  at HEAD (build + vet + `go test ./...`, all 28 packages ok; `gofmt -l .` empty); the honesty-copy close
  is mutation-proven (reverting the `cert.html:484` sentence fails `TestCertificateRendersWasmVerifier`),
  and the trust-root oracles are unaffected (this diff touched no signature / RFC-6962 / Merkle / `proof`
  / `didweb` / `logclient` file — confirmed by review's name-only globs).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`ca2b05a`) is
  pushed; CI run 27950022314 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27950022300) at HEAD is
  **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions" source);
  deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 5 normal, 10 low** (the lone "critical" grep hit is the format-template legend
  on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE. Tally moved
  from "0 critical / 6 normal" last assessment to "0 critical / 5 normal" — the certificate tier-2 no-JS
  honesty-copy `normal` closed; no new normal filed. The 5 normal: certificate §6 `· at` timestamp, the
  `/` sub-region parity deltas, the WASM verifier-scope SIGNATURE-half gap, the Surface-C `readTarget`
  opaque-URL permissiveness, and the Pages custom-domain/enablement gap.

## Next Milestone
**CI green and pushed; the code-closable certificate tier-2 honesty `normal` just closed. Resume the
front-of-queue WASM-verifier upgrade milestone — but its remaining "published" half is human-blocked, so
prioritize a code-closable target.** In order:
1. **The signature half of the verifier-scope gap** (the milestone's actual trust bar) — browser did:web
   resolution + checkpoint-note signature verify, gating `verified` on signature + id-binding + inclusion.
   `review` recommends a **design-first pass** before building (browser-side did:web resolution is
   non-trivial); a good STOP-candidate if the design is unclear. Until it lands, do NOT loosen
   `verifier.html`'s "hub-signed root" success copy. And/or wire the tier-2 WASM caller into the **hub
   dossier** (no WASM island today).
2. **Remaining code-closable `normal`s** — the Surface-C `readTarget` opaque-URL normalization is the only
   pure code-closable one left (return the parsed `u.href` / reject the opaque form). (The §6 `· at`
   timestamp + the `/` Checkpoint/Anchor columns need a store schema/projection change — larger.)
3. **Unblock + verify the Pages deploy** (needs the one-time human repo-Settings step: Settings → Pages →
   source "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and
   confirm a `success` deploy — that closes the "published" half of the WASM Verify criterion. A workflow
   file alone cannot self-enable Pages, so flag for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the §6 `· at` timestamp and
the `/` Checkpoint/Anchor columns when the projection gains the needed columns; the named-region + `←`
back-link parity pass across the remaining SSR surfaces; the M-UI exit visual-pass + human sign-off
(ADR-0012).
