<!-- assessed-at: 97ddd572349ed1e198b0622d3ea52d855a27776c -->

# Project State

## Status: IN_PROGRESS

## Phase: Certificate §5 OTS digest-binding landed (anchor-render hardening) — gate green, pushed.
The §5 BITCOIN ANCHOR clause now renders against `ots.ConfirmedFor(rec.OTSBytes, root)`, which
fail-closes (wrapped error, same contract as a parse failure) unless the parsed OTS proof's committed
SHA-256 `File.Digest` byte-equals §2's accepted checkpoint root — so a mis-stamped (root, proof) row
silently declines §5 instead of fabricating "block N". Latest `review` verdict is **PASS (loop
CONTINUE)** and HEAD (`97ddd57`) is **pushed** — `origin/develop` == HEAD, CI **success** at HEAD.
DONE not reached: the WASM "published"/signature halves and the OTS Bitcoin-confirmed Verify criteria
are still open, and 6 `normal` issues stand.

Incremental review against assessed-at `eebf9d6`. The `eebf9d6..HEAD` diff touched exactly TWO
production files — `internal/ots/ots.go` (added `ConfirmedFor` + a shared private `classify`; `Confirmed`
and `ConfirmedFor` now share one parse + classify, with `ConfirmedFor` adding the
`bytes.Equal(file.Digest, root)` gate at `:104`) and `internal/certificate/handler.go` (§5 now calls
`ots.ConfirmedFor(rec.OTSBytes, root)` at `:939`) — plus their tests and `.claude/context/*` +
`learnings/{certificate,ots}.md` docs. All M1/M2/M3/M-UI/WASM/OTS production source outside
`internal/{ots,certificate}` is untouched — those sections carry forward met/open as before. **The
"§5 does not bind the OTS digest to §2's root" `normal` is CLOSED** (verified this assessment):
`ots.go:104` is `if !bytes.Equal(file.Digest, root) { … wrapped error }`, mutation-proven by
`TestOTSConfirmedForDigestBound/mismatch_declines` and `TestCertificateBitcoinAnchorDigestMismatch`
(per the review handoff: removing the gate / reverting the §5 call fails those tests). `git status`
clean, `origin/develop` == HEAD, CI run 27949293303 `success` at HEAD.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact: the byte-pinned
    `internal/web/verify.wasm` carries the 6-arg/id-binding shim, the pin (`WasmVerifyHash`) matches the
    committed bytes, and the bytes are reproducible from `mise run build:wasm` under `go=1.26.4`.
    **Still OPEN on the milestone Verify** ("the verifier artifact … [published at its] published value"):
    the Pages deploy **does not run** — the Pages run at HEAD (27949293297) fails at `Configure Pages`
    (Pages not enabled / not set to "GitHub Actions" source); built-but-undeployed, a one-time human
    repo-Settings step. Also still open: **NO dossier tier-2 WASM caller** (no WASM island in
    `internal/dossier`); and the **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies
    inclusion math + id-binding only — no checkpoint-signature / did:web key resolution — so a malicious
    monitor can still render a green `verified`; design-first remainder).
  - **OTS anchoring: 1/1 open (carried unchanged).** All three observable HTTP halves are closed (`.ots`
    serve route + the §5 anchor render, now digest-bound) and BOTH calendar-transport guards (`safeUpgrade`
    + `safeStamp`) are in place. What remains: a root that actually transits to **Bitcoin-confirmed** —
    offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~3 milestone-Verify-advancing / ~7 chrome·plumbing·hardening.** Recent arc:
  WASM id-binding bind → rebuild+re-pin verify.wasm → pin `go=1.26.4` + re-pin → `safeStamp` OTS guard →
  §4/bundle `host:port` did:web encode → ForceQuery hubDomain reject → **§5 OTS digest-binding (this
  iteration, PASS)**. **DRIFT WATCH (amber):** the front-of-queue WASM "published" Verify has not closed
  for ~11 increments, but its remaining open half (the live Pages deploy) is **human-blocked, not
  code-blocked** — a workflow file cannot self-enable Pages, so the loop cannot autonomously close it.
  The loop continues to correctly *pivot off* the human-blocked WASM criterion to drain code-closable
  `normal`s (the right move per guidance) — the §5 digest-binding closed one this iteration. Remaining
  code-closable productive targets: the WASM signature-half trust gap (design-first / STOP-candidate),
  the dossier WASM caller, and the remaining certificate `normal`s (tier-2 honesty copy, §6 timestamp,
  the `/` sub-region parity deltas).

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `eebf9d6..HEAD` diff touched only `internal/{ots,certificate}`;
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
The only M-UI-adjacent change this diff was the certificate §5 server-side gate (`handler.go` `buildData`
+ `ots.go`) — it decides WHICH already-defined §5 state (confirmed/pending/omitted) renders, not how it
looks; `cert.html` and every template are byte-unchanged (per the review's visual check). All six
certificate clauses (§1–§6) + both anchor panels + badge + DS shell + `/` index + log browser + hub
dossier (with the shared chrome masthead + `← Realm index` back-link) + frozen Exhibit + record list +
single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the
behavioral HTTP-seam Verify; every SSR masthead carries the shared chrome (self-hosted logo + text mark
+ divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the dossier
  but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still lack the full
  instance-identity block + back-link chain on every surface); `/` sub-region deltas (config-driven
  instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`; the
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this browser
  re-verifies" on the no-JS baseline (`cert.html:465`); §6 rows omit the per-record `· at` timestamp; the
  `/` sub-region parity deltas. (The "§5 digest-binding" `normal` is now CLOSED.)

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN, reproducibility critical CLOSED, HEAD is PASS. OTS — §5 render now
digest-bound + both calendar-transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** WASM source was untouched by this diff; OTS gained the `ConfirmedFor` classifier.
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
  `failure` at `Configure Pages` at HEAD, run 27949293297), needs the one-time human repo-Settings step.
  **Carried `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a malicious
  cross-origin monitor can render green `verified`; success copy overstates a key check that never runs) —
  design-first remainder; Surface-C `readTarget` accepts opaque-scheme monitor forms; the Pages
  custom-domain / Actions-source enablement gap. **Carried `low`:** `cmd/verifier-site` `generate` writes
  non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause (now digest-bound via
  `ots.ConfirmedFor`), the store layer (`internal/store/ots.go`), the off-path stamp/upgrade loop
  (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier (`internal/ots.{Confirmed,
  ConfirmedFor}` sharing one private `classify`), and the calendar transport (`internal/otsclient`, with
  BOTH `safeUpgrade` + `safeStamp` guards) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer
  not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation.
  Still 1/1 open. **Carried `low` defect (NOT fixed):** nil-Stamper + empty-OTSBytes row falls through to
  the Upgrader instead of being left untouched (`otsloop.go:144`; docstring-vs-code mismatch on the
  test-only nil path; production always wires a non-nil Stamper).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`97ddd57`) == `origin/develop` (run 27949293303);
the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate defect). Latest
`review` verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build toolchain
  `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run check` green
  at HEAD (all packages build + vet + test; `gofmt -l .` empty); the §5 digest-binding is mutation-proven
  (removing the `bytes.Equal` gate fails `TestOTSConfirmedForDigestBound/mismatch_declines`; reverting the
  §5 call fails `TestCertificateBitcoinAnchorDigestMismatch`), and the trust-root oracles still pass
  independently (`derive_vkey.py` reproduces both golden vectors; `go test ./internal/logclient
  ./internal/proof/verify` green).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`97ddd57`) is
  pushed; CI run 27949293303 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27949293297) at HEAD is
  **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions" source);
  deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 6 normal, 10 low** (the lone "critical" grep hit is the format-template legend
  on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE. Tally moved
  from "0 critical / 7 normal" last assessment to "0 critical / 6 normal" — the certificate §5
  digest-binding `normal` closed; no new normal filed. The 6 normal: certificate §6 `· at` timestamp, the
  certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region parity deltas, the WASM
  verifier-scope SIGNATURE-half gap, the Surface-C `readTarget` opaque-URL permissiveness, and the Pages
  custom-domain/enablement gap.

## Next Milestone
**CI green and pushed; the code-closable certificate §5 digest-binding `normal` just closed. Resume the
front-of-queue WASM-verifier upgrade milestone — but its remaining "published" half is human-blocked, so
prioritize a code-closable target.** In order:
1. **The signature half of the verifier-scope gap** (the milestone's actual trust bar) — browser did:web
   resolution + checkpoint-note signature verify, gating `verified` on signature + id-binding + inclusion.
   `review` recommends a **design-first pass** before building (browser-side did:web resolution is
   non-trivial); a good STOP-candidate if the design is unclear. Until it lands, do NOT loosen
   `verifier.html`'s "hub-signed root" success copy. And/or wire the tier-2 WASM caller into the **hub
   dossier** (no WASM island today).
2. **Remaining code-closable certificate `normal`s** (handoff-named order): the tier-2 honesty-copy
   overstatement (`cert.html:465` — copy-only, code-closable now). (The §6 `· at` timestamp + the `/`
   Checkpoint/Anchor columns need a store schema/projection change — larger.)
3. **Unblock + verify the Pages deploy** (needs the one-time human repo-Settings step: Settings → Pages →
   source "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and
   confirm a `success` deploy — that closes the "published" half of the WASM Verify criterion. A workflow
   file alone cannot self-enable Pages, so flag for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the §6 `· at` timestamp and
the `/` Checkpoint/Anchor columns when the projection gains the needed columns; the named-region + `←`
back-link parity pass across the remaining SSR surfaces; the M-UI exit visual-pass + human sign-off
(ADR-0012).
