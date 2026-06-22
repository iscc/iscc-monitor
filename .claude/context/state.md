<!-- assessed-at: 3cf18788bb629a6f74e2e30f464c86a821640687 -->

# Project State

## Status: IN_PROGRESS

## Phase: OTS transport hardening landed (both OTS calendar guards now symmetric) — gate green, pushed.
The live `otsclient.Stamp` calendar submit is now routed through `safeStamp` (per-request
`stampTimeout=30s` deadline + `recover()`-to-error), mirroring `safeUpgrade`, so the stamp path can no
longer freeze or panic the binary; `Stamp`'s exported signature is unchanged, so `cmd/iscc-monitor` is
untouched. Latest `review` verdict is **PASS (loop CONTINUE)** and HEAD is **pushed** — `origin/develop`
== HEAD, CI **success** at HEAD. DONE not reached: the WASM "published"/signature-half and OTS
Bitcoin-confirmed Verify criteria are still open, and 9 `normal` issues stand.

Incremental review against assessed-at `8496579`. The `8496579..HEAD` diff touched exactly TWO production
files — `internal/otsclient/client.go` (+`safeStamp`/`buildStamper`/`stampFn` seam) and its test
`client_test.go` (+`TestStamp*`) — plus `.claude/context/*` + `learnings/otsclient.md` docs. All
M1/M2/M3/M-UI/WASM production source is untouched — those sections carry forward met/open as before.
**OTS stamp-guard `normal` CLOSED** (verified this assessment): `safeStamp` exists at
`internal/otsclient/client.go:201` with `context.WithTimeout(ctx, stampTimeout)` (`:202`, `stampTimeout
= 30s` at `:52`) and a `recover()` guard (`:205`); the production wiring is intact —
`cmd/iscc-monitor/main.go:236` `stampFunc()` calls `otsclient.Stamp` inside `runOTSLoop`'s live
goroutine (`main.go:151`), so the guard is on a real path. New tests `TestStampPanicRecovered`,
`TestStampBoundsContext`, `TestStampSerializesRoundTrip` present. `git status` clean, `origin/develop`
== HEAD (`3cf1878`).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Source + artifact are closed for the id-binding half: the byte-pinned
    `internal/web/verify.wasm` carries the 6-arg/id-binding shim, the pin (`WasmVerifyHash`) matches the
    committed bytes, and the bytes are reproducible from `mise run build:wasm` under `go=1.26.4`.
    **Still OPEN on the milestone Verify** ("the verifier artifact … [is published at its] published
    value"): the Pages deploy **does not run** — the Pages run at HEAD fails at `Configure Pages` (Pages
    not enabled / not set to "GitHub Actions" source); built-but-undeployed, a one-time human
    repo-Settings step. Also still open: **NO dossier tier-2 WASM caller** (no WASM island in
    `internal/dossier`); and the **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies
    inclusion math + id-binding only — no checkpoint-signature / did:web key resolution — so a malicious
    monitor can still render a green `verified`; design-first remainder).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots` serve
    route + certificate §5 anchor render) and BOTH calendar-transport guards (`safeUpgrade` +
    `safeStamp`) are now in place. What remains: a root that actually transits to **Bitcoin-confirmed** —
    offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~4 milestone-Verify-advancing / ~6 chrome·plumbing·hardening.** Recent arc:
  Pages publish workflow → `safeIndex`→`SafeIndex` → WASM id-binding bind → rebuild+re-pin verify.wasm
  → pin `go=1.26.4` + re-pin (artifact reproducible, PASS) → **`safeStamp` OTS guard (this iteration,
  PASS)**. **DRIFT WATCH (amber):** the front-of-queue WASM "published" Verify has not closed for eight
  increments, but its remaining open half (the live Pages deploy) is **human-blocked, not
  code-blocked** — a workflow file cannot self-enable Pages, so the loop cannot autonomously close it.
  This iteration correctly *pivoted off* the human-blocked WASM criterion to close a code-closable OTS
  hardening `normal` (the right move per last assessment's guidance). The remaining code-closable WASM
  target is the signature-half trust gap (design-first) or the dossier WASM caller; OTS/§5/SSR-parity
  work is the other productive direction while WASM's published half stays human-blocked.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `8496579..HEAD` diff (confined to
`internal/otsclient/*` + context/learnings docs). All M1 Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once +
restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); 23 internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.26.1` language directive in `go.mod`; build toolchain
  `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

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
**Status**: **WASM — milestone OPEN, reproducibility critical CLOSED, HEAD is PASS. OTS — both
calendar-transport guards now landed; only a real Bitcoin confirmation remains (offline-unprovable).**
- **WASM (untouched this diff, carried):** the id-binding half of the verifier-scope trust gap is
  closed IN SOURCE and ARTIFACT, and the artifact is reproducible from its documented command: the
  committed `internal/web/verify.wasm` carries the 6-arg/id-binding shim
  (`verifyadapter.RecordCommitsID`), its SHA-256 (`2c91e61f…`) matches `WasmVerifyHash`
  (`internal/web/web.go:93`), and `mise run build:wasm` under `mise.toml` `go = "1.26.4"`
  deterministically re-emits those bytes (`TestWasmVerifyHashPinned` green). `cmd/wasm/main.go` (tagged
  `//go:build js && wasm`) registers `isccVerifyInclusion` (5-or-6 arg shim, `SafeIndex` at leaf/size
  sites, gates `verified` on `RecordCommitsID` at 6 args). `verifyadapter` exposes `SafeIndex`,
  `VerifyJSON`, `RecordCommitsID` (golden + mutation-tested). `cert.html` embeds the tier-2 island +
  `/_ds/wasm_exec.js` + `/_ds/verify.wasm` (5-arg caller). `internal/verifier` is a SINGLE STATIC
  cross-origin artifact; `cmd/verifier-site` is the reproducible build command;
  `.github/workflows/pages.yml` is the PUBLISH workflow; `verifier.Handler` is NOT mounted in
  `cmd/iscc-monitor`. **Still 1/1 OPEN on the milestone Verify** — the deploy is not live (Pages
  `failure` at `Configure Pages` at HEAD), needs the one-time human repo-Settings step. **Carried
  `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a malicious
  cross-origin monitor can render green `verified`; success copy overstates a key check that never runs)
  — design-first remainder; Surface-C `readTarget` accepts opaque-scheme monitor forms; the Pages
  custom-domain / Actions-source enablement gap. **Carried `low`:** `cmd/verifier-site` `generate`
  writes non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the calendar
  transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. **NEW this diff:** the
  live `Stamp` calendar submit is now guarded by `safeStamp` (`client.go:201`: `stampTimeout=30s`
  `context.WithTimeout` + `recover()`), symmetric with `safeUpgrade` — so BOTH OTS calendar guards are
  in place and the two OTS guard `normal`s are both closed. The Verify-closer not yet built: a root
  reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation. Still 1/1 open.
  **Carried `low` defect (NOT fixed):** nil-Stamper + empty-OTSBytes row falls through to the Upgrader
  instead of being left untouched (`otsloop.go:144`; docstring-vs-code mismatch on the test-only nil
  path; production always wires a non-nil Stamper).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`3cf1878`) == `origin/develop`; the Pages
PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate defect). Latest `review`
verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build
  toolchain `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run
  check` green at HEAD (all 27 packages build + vet + test; `gofmt -l .` empty); the stamp guards
  mutation-proven (`TestStampPanicRecovered` panics the binary when `recover()` is removed;
  `TestStampBoundsContext` fails when `context.WithTimeout` is dropped).
- **Latest `review` verdict: PASS (loop CONTINUE)** for the `safeStamp` guard. The reviewer confirmed
  the diff is exactly the two named files (+ docs), the guard is on a live path (`main.go:236` →
  `runOTSLoop`), the WASM purity guard holds (`internal/otsclient` not in didweb/index/badge/proof-verify
  dep trees), the conformance/oracle gate is N/A (no signature/Merkle/proof code touched), no
  gate-circumvention, and Codex corroborated clean.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`3cf1878`) is
  pushed; CI run 27946528901 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27946528871) at HEAD
  is **`failure`** — the `render verifier-site` job fails at `Configure Pages` (Pages not enabled / not
  set to "GitHub Actions" source); deploy skipped. One-time human repo-Settings step (filed `normal`),
  not a code/gate defect.
- **Open issues: 0 critical, 9 normal, 10 low** (the lone "critical" grep hit is the format-template
  legend on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
  Tally moved from "0 critical / 10 normal" last assessment to "0 critical / 9 normal" — the OTS
  stamp-guard `normal` closed; no new normal filed. The 9 normal: the Pages custom-domain/enablement
  gap, certificate §5 digest binding, `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode,
  certificate §6 timestamp, the certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region
  parity deltas, the WASM verifier-scope SIGNATURE-half gap, and the Surface-C `readTarget` opaque-URL
  permissiveness.

## Next Milestone
**CI green and pushed; the code-closable OTS hardening normal just closed. Resume the front-of-queue
WASM-verifier upgrade milestone — but its remaining "published" half is human-blocked, so prioritize a
code-closable target.** In order:
1. **The signature half of the verifier-scope gap** (the milestone's actual trust bar) — browser
   did:web resolution + checkpoint-note signature verify, gating `verified` on signature + id-binding +
   inclusion. `review` recommends a **design-first pass** before building (browser-side did:web
   resolution is non-trivial); a good STOP-candidate if the design is unclear. Until it lands, do NOT
   loosen `verifier.html`'s "hub-signed root" success copy. And/or wire the tier-2 WASM caller into the
   **hub dossier** (no WASM island today).
2. **Unblock + verify the Pages deploy** (needs the one-time human repo-Settings step: Settings → Pages
   → source "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow
   and confirm a `success` deploy — that closes the "published" half of the WASM Verify criterion. A
   workflow file alone cannot self-enable Pages, so flag for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the §5 digest-binding,
the `host:port` DID `%3A`-encode, and the `hubDomain` ForceQuery reject when those exact lines are next
edited; the named-region + `←` back-link parity pass across the remaining SSR surfaces; the M-UI exit
visual-pass + human sign-off (ADR-0012).
