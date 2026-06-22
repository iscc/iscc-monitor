<!-- assessed-at: c0d8273175cba0f205a8a605facb7cc54cd54fcf -->

# Project State

## Status: IN_PROGRESS

## Phase: Surface-C readTarget normalization landed (last pure-code-closable normal drained) — gate green, pushed.
The Surface-C verifier `readTarget` now returns the WHATWG-normalized parsed URL (`u.href`) instead
of the raw `?monitor=` query string (`internal/verifier/verifier.html:556`), so an opaque-scheme form
(`https:example.com`) flows downstream as `https://example.com/`. Latest `review` verdict is
**PASS (loop CONTINUE)** and HEAD (`c0d8273`) is **pushed** — `origin/develop` == HEAD, CI **success**
at HEAD. DONE not reached: the WASM "published" + signature halves and the OTS Bitcoin-confirmed Verify
criteria stay open, and 4 `normal` issues stand.

Incremental review against assessed-at `ca2b05a`. The `ca2b05a..HEAD` diff touched exactly ONE
production file — `internal/verifier/verifier.html` (one JS return line + its docstring) — plus its
test (`handler_test.go`, +19 lines: a mutation-provable positive+negative assertion pair) and
`.claude/context/*` + `learnings/verifier.md` docs. All M1/M2/M3/M-UI/WASM/OTS production source
outside that one template line is byte-unchanged — those sections carry forward met/open as before.
**The "Surface-C `readTarget` accepts opaque-scheme monitor forms" `normal` is CLOSED** (verified this
assessment): the rendered body now carries `return { monitor: u.href, id: id };` and the old raw-string
return is gone; the close is mutation-proven by `TestVerifierReadTargetReturnsNormalizedURL` (per
review: reverting the return trips both the positive and negative assertion). `git status` clean,
`origin/develop` == HEAD, CI run 27950901831 `success` at HEAD.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact: the byte-pinned
    `internal/web/verify.wasm` carries the 6-arg/id-binding shim, the pin (`WasmVerifyHash`) matches the
    committed bytes, and the bytes are reproducible from `mise run build:wasm` under `go=1.26.4`.
    **Still OPEN on the milestone Verify** ("the verifier artifact … [published at its] published value"):
    the Pages deploy **does not run** — the Pages run at HEAD (27950901768) fails at `Configure Pages`
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
  pin `go=1.26.4` + re-pin verify.wasm → `safeStamp` OTS guard → §4/bundle `host:port` did:web encode →
  ForceQuery hubDomain reject → §5 OTS digest-binding (PASS) → certificate Tier-2 no-JS honesty copy
  (PASS) → **Surface-C readTarget URL normalization (this iteration, PASS)**. **DRIFT WATCH (amber→red
  candidate):** the front-of-queue WASM "published" Verify has not closed for ~13 increments, but its
  remaining open half (the live Pages deploy) is **human-blocked, not code-blocked** — a workflow file
  cannot self-enable Pages, so the loop cannot autonomously close it. The loop has correctly pivoted off
  the human-blocked WASM criterion to drain code-closable `normal`s — and this iteration drained the
  **last pure copy/code-closable one** (Surface-C readTarget). The remaining 4 `normal`s are now ALL
  design-first or store-schema changes (WASM signature half = design pass / STOP-candidate; §6 `· at`
  timestamp + the `/` Checkpoint/Anchor columns = store-projection + follower-ingest change; Pages
  custom-domain = human Settings step). The loop has run out of cheap code-closable work while the
  milestone-Verify criteria stay human/design-blocked — the next increment must either pick a
  store-schema `normal` (larger, self-contained) or run a design pass on the WASM signature half. This is
  the convergence inflection: surface it loudly so define-next does not re-polish.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `ca2b05a..HEAD` diff touched only
`internal/verifier/verifier.html`; no M1 source touched. All M1 Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once +
restart survival; structured logs; `/metrics`.
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
No M-UI surface changed this diff (the lone production edit was the cross-origin Surface-C verifier's
`readTarget` JS — a runtime value computed from `location.search`, never displayed in the no-JS body;
its named-region markup is byte-unchanged). All six certificate clauses (§1–§6) + both anchor panels +
badge + DS shell + `/` index + log browser + hub dossier (with the shared chrome masthead + `← Realm
index` back-link) + frozen Exhibit + record list + single-record page + ISCC-IDv1 decoder + Hub-List
resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify; every SSR masthead
carries the shared chrome (self-hosted logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the dossier
  but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still lack the full
  instance-identity block + back-link chain on every surface); `/` sub-region deltas (config-driven
  instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`; the
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate §6 rows omit the per-record `· at` timestamp
  (store-schema change); the `/` sub-region parity deltas. (The certificate tier-2 no-JS honesty-copy and
  the §5 digest-binding `normal`s closed in prior iterations.)

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN (published half human-blocked, signature half design-blocked),
reproducibility critical CLOSED, HEAD is PASS. OTS — §5 render digest-bound + both calendar-transport
guards landed; only a real Bitcoin confirmation remains (offline-unprovable).** Neither the WASM core
nor OTS source was touched by this diff; both sections carry forward. (This diff DID touch the
cross-origin Surface-C verifier page — `verifier.html` `readTarget` — but only its query-parse
normalization, not the WASM verdict path or its scope.)
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
  `failure` at `Configure Pages` at HEAD, run 27950901768), needs the one-time human repo-Settings step.
  **Carried `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a malicious
  cross-origin monitor can render green `verified`; success copy overstates a key check that never runs) —
  design-first remainder / STOP-candidate; the Pages custom-domain / Actions-source enablement gap.
  **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically. (The Surface-C `readTarget`
  opaque-URL `normal` is now CLOSED this iteration.)
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
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`c0d8273`) == `origin/develop` (run
27950901831); the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate
defect). Latest `review` verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build toolchain
  `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run check` green
  at HEAD (build + vet + `go test ./...`, all 28 packages ok; `gofmt -l .` empty); the readTarget close
  is mutation-proven (reverting the `verifier.html` return fails `TestVerifierReadTargetReturnsNormalizedURL`
  on both the positive and negative assertion), and the trust-root oracles are unaffected (this diff
  touched no signature / RFC-6962 / Merkle / `proof` / `didweb` / `logclient` file — confirmed by review's
  name-only globs; the edit is a usability guard on a static render, not a trust boundary).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`c0d8273`) is
  pushed; CI run 27950901831 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27950901768) at HEAD is
  **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions" source);
  deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 4 normal, 10 low** (the lone "critical" grep hit is the format-template legend
  on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE. Tally moved
  from "0 critical / 5 normal" last assessment to "0 critical / 4 normal" — the Surface-C `readTarget`
  opaque-URL `normal` closed; no new normal filed. The 4 normal: certificate §6 `· at` timestamp, the
  `/` sub-region parity deltas, the WASM verifier-scope SIGNATURE-half gap, and the Pages custom-domain/
  enablement gap. **All 4 are now design-first or store-schema/human-blocked — none is pure-code-closable
  in one package (the convergence inflection flagged above).**

## Next Milestone
**CI green and pushed; the last pure-code-closable `normal` (Surface-C readTarget) just closed. The
cheap code-closable backlog is now empty — every remaining `normal` is store-schema, design-first, or
human-blocked. Resume the front-of-queue WASM-verifier milestone via a design pass, OR pick a
self-contained store-schema `normal`.** In order:
1. **Design-first pass on the signature half of the verifier-scope gap** (the milestone's actual trust
   bar) — browser did:web resolution + checkpoint-note signature verify, gating `verified` on signature +
   id-binding + inclusion. `review` recommends a **design pass before building** (browser-side did:web
   resolution is non-trivial); a good STOP-candidate if the design is unclear. Until it lands, do NOT
   loosen `verifier.html`'s "hub-signed root" success copy. And/or wire the tier-2 WASM caller into the
   **hub dossier** (no WASM island today).
2. **A self-contained store-schema `normal`** — the certificate §6 per-record `· at` timestamp (add a
   timestamp column to the `iscc_index` projection written by `RecordProjections`, surface via `RecordAt`,
   render in §6), OR the `/` Checkpoint/Anchor data columns (add fields to `ListHubs`/`HubSummary` +
   render). Both are store-projection + follower-ingest changes — larger than a one-line copy edit but
   self-contained and code-closable. Prefer one of these over re-polishing already-met surfaces.
3. **Unblock + verify the Pages deploy** (needs the one-time human repo-Settings step: Settings → Pages →
   source "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and
   confirm a `success` deploy — that closes the "published" half of the WASM Verify criterion. A workflow
   file alone cannot self-enable Pages, so flag for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the named-region + `←`
back-link parity pass across the remaining SSR surfaces; the M-UI exit visual-pass + human sign-off
(ADR-0012).
