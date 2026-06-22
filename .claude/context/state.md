<!-- assessed-at: eebf9d6d8c5a274a9ee4b1fce9b8b7279748f0c2 -->

# Project State

## Status: IN_PROGRESS

## Phase: Hub-List `hubDomain` ForceQuery reject landed (registry hardening) — gate green, pushed.
The Hub-List bare-host guard now also rejects a bare trailing `?` (which `net/url` records as
`ForceQuery == true` with an empty `RawQuery`) so the empty-query delimiter can never round-trip into a
resolved hub domain. Latest `review` verdict is **PASS (loop CONTINUE)** and HEAD (`eebf9d6`) is
**pushed** — `origin/develop` == HEAD, CI **success** at HEAD. DONE not reached: the WASM
"published"/signature-half and OTS Bitcoin-confirmed Verify criteria are still open, and 7 `normal`
issues stand.

Incremental review against assessed-at `1d906d2`. The `1d906d2..HEAD` diff touched exactly ONE
production file — `internal/registry/registry.go` (added `|| u.ForceQuery` to the single `hubDomain`
bare-host reject at `:192`, with the docstring + in-line comment updated to match) — plus its test
(`hublist_test.go`, +1 `trailing question mark (ForceQuery)` case) and `.claude/context/*` +
`learnings/registry.md` docs. All M1/M2/M3/M-UI/WASM/OTS production source outside `internal/registry`
is untouched — those sections carry forward met/open as before. **ForceQuery fail-open `normal`
CLOSED** (verified this assessment): `registry.go:192` is now
`if u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" { … reject }`, mutation-proven
by `TestParseHubListErrors/trailing_question_mark_(ForceQuery)` (per the review handoff: reverting the
operand makes the case parse `URL:"https://sb0.iscc.id?"`), and the clean-domain golden
(`TestParseHubListGolden`) still parses `sb0.iscc.id` / `sb1.amlet.id` unchanged. `git status` clean,
`origin/develop` == HEAD, CI run 27947917147 `success` at HEAD.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** Source + artifact are closed for the id-binding half: the byte-pinned
    `internal/web/verify.wasm` carries the 6-arg/id-binding shim, the pin (`WasmVerifyHash`) matches the
    committed bytes, and the bytes are reproducible from `mise run build:wasm` under `go=1.26.4`.
    **Still OPEN on the milestone Verify** ("the verifier artifact … [published at its] published value"):
    the Pages deploy **does not run** — the Pages run at HEAD (27947917218) fails at `Configure Pages`
    (Pages not enabled / not set to "GitHub Actions" source); built-but-undeployed, a one-time human
    repo-Settings step. Also still open: **NO dossier tier-2 WASM caller** (no WASM island in
    `internal/dossier`); and the **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies
    inclusion math + id-binding only — no checkpoint-signature / did:web key resolution — so a malicious
    monitor can still render a green `verified`; design-first remainder).
  - **OTS anchoring: 1/1 open (carried unchanged).** Both observable HTTP halves closed (`.ots` serve
    route + certificate §5 anchor render) and BOTH calendar-transport guards (`safeUpgrade` + `safeStamp`)
    are in place. What remains: a root that actually transits to **Bitcoin-confirmed** — offline-unprovable;
    exercised only against an injected Upgrader.
- **Last ~10 iterations: ~3 milestone-Verify-advancing / ~7 chrome·plumbing·hardening.** Recent arc:
  WASM id-binding bind → rebuild+re-pin verify.wasm → pin `go=1.26.4` + re-pin → `safeStamp` OTS guard →
  §4/bundle `host:port` did:web encode → **ForceQuery hubDomain reject (this iteration, PASS)**. **DRIFT
  WATCH (amber):** the front-of-queue WASM "published" Verify has not closed for ten increments, but its
  remaining open half (the live Pages deploy) is **human-blocked, not code-blocked** — a workflow file
  cannot self-enable Pages, so the loop cannot autonomously close it. The loop continues to correctly
  *pivot off* the human-blocked WASM criterion to close code-closable `normal`s (the right move per
  guidance). Remaining code-closable productive targets: the WASM signature-half trust gap (design-first /
  STOP-candidate), the dossier WASM caller, and the remaining certificate `normal`s (§5 digest-binding,
  tier-2 honesty copy, §6 timestamp, the `/` sub-region parity deltas).

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `1d906d2..HEAD` diff touched only the `internal/registry`
Hub-List `hubDomain` URL-shape guard (added the `ForceQuery` reject), which *hardens* an already-met M1
criterion without changing any Verify-relevant behavior; no other M1 source touched. All M1 Verify
criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end
with freeze + alert-once + restart survival; structured logs; `/metrics`.
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
No M-UI surface touched by this diff (the only production change was the `internal/registry` pure leaf —
no rendered output affected). All six certificate clauses (§1–§6) + both anchor panels + badge + DS shell
+ `/` index + log browser + hub dossier (with the shared chrome masthead + `← Realm index` back-link) +
frozen Exhibit + record list + single-record page + ISCC-IDv1 decoder + Hub-List resolver +
proof-bundle endpoint render and pass the behavioral HTTP-seam Verify; every SSR masthead carries the
shared chrome (self-hosted logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the dossier
  but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still lack the full
  instance-identity block + back-link chain on every surface); `/` sub-region deltas (config-driven
  instance identity/realm, Checkpoint/Bitcoin-anchor columns, recent-declarers footer) filed `normal`; the
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes (NOT fixed):** certificate tier-2 honesty header overstates "this browser
  re-verifies" on the no-JS baseline (`cert.html:465`); §5 does not bind the OTS proof digest to §2's root;
  §6 rows omit the per-record `· at` timestamp; the `/` sub-region parity deltas.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN, reproducibility critical CLOSED, HEAD is PASS. OTS — both
calendar-transport guards landed; only a real Bitcoin confirmation remains (offline-unprovable).** Neither
WASM nor OTS source was touched by this diff — carried forward.
- **WASM (carried):** the id-binding half of the verifier-scope trust gap is closed IN SOURCE and ARTIFACT,
  and the artifact is reproducible from its documented command: the committed `internal/web/verify.wasm`
  carries the 6-arg/id-binding shim (`verifyadapter.RecordCommitsID`), its SHA-256 (`2c91e61f…`) matches
  `WasmVerifyHash` (`internal/web/web.go:93`), and `mise run build:wasm` under `mise.toml` `go = "1.26.4"`
  deterministically re-emits those bytes (`TestWasmVerifyHashPinned` green). `cmd/wasm/main.go` (tagged
  `//go:build js && wasm`) registers `isccVerifyInclusion`; `verifyadapter` exposes `SafeIndex`,
  `VerifyJSON`, `RecordCommitsID` (golden + mutation-tested); `cert.html` embeds the tier-2 island;
  `internal/verifier` is a SINGLE STATIC cross-origin artifact; `cmd/verifier-site` is the reproducible
  build command; `.github/workflows/pages.yml` is the PUBLISH workflow; `verifier.Handler` is NOT mounted
  in `cmd/iscc-monitor`. **Still 1/1 OPEN on the milestone Verify** — the deploy is not live (Pages
  `failure` at `Configure Pages` at HEAD, run 27947917218), needs the one-time human repo-Settings step.
  **Carried `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a malicious
  cross-origin monitor can render green `verified`; success copy overstates a key check that never runs) —
  design-first remainder; Surface-C `readTarget` accepts opaque-scheme monitor forms; the Pages
  custom-domain / Actions-source enablement gap. **Carried `low`:** `cmd/verifier-site` `generate` writes
  non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in `internal/follower/otsloop.go`),
  the offline classifier (`internal/ots.Confirmed`), and the calendar transport (`internal/otsclient`,
  with BOTH `safeUpgrade` + `safeStamp` guards) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer
  not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation.
  Still 1/1 open. **Carried `low` defect (NOT fixed):** nil-Stamper + empty-OTSBytes row falls through to
  the Upgrader instead of being left untouched (`otsloop.go:144`; docstring-vs-code mismatch on the
  test-only nil path; production always wires a non-nil Stamper).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`eebf9d6`) == `origin/develop` (run 27947917147);
the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate defect). Latest
`review` verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build toolchain
  `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run check` green
  at HEAD (all 27 packages build + vet + test; `gofmt -l .` empty); the `ForceQuery` reject is
  mutation-proven (reverting `|| u.ForceQuery` FAILS the `trailing_question_mark_(ForceQuery)` subtest).
- **Latest `review` verdict: PASS (loop CONTINUE)** for the ForceQuery hubDomain reject. The reviewer
  confirmed the diff is exactly `registry.go` + one test file (+ docs), the change is one operand on one
  condition + an evergreen-comment update + one table case, mutation-proven non-vacuous, the clean-domain
  golden intact, the leaf stays WASM-shareable (`GOOS=js GOARCH=wasm go build ./internal/registry`), no
  `go.mod`/`go.sum` drift, the oracle/conformance gate N/A (pure URL-shape leaf, no crypto path), no
  gate-circumvention, and Codex corroborated clean.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`eebf9d6`) is
  pushed; CI run 27947917147 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27947917218) at HEAD is
  **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions" source);
  deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 7 normal, 10 low** (the lone "critical" grep hit is the format-template legend
  on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE. Tally moved
  from "0 critical / 8 normal" last assessment to "0 critical / 7 normal" — the `hubDomain` ForceQuery
  fail-open `normal` closed; no new normal filed. The 7 normal: certificate §5 digest binding, certificate
  §6 timestamp, the certificate tier-2 no-JS honesty-copy overstatement, the `/` sub-region parity deltas,
  the WASM verifier-scope SIGNATURE-half gap, the Surface-C `readTarget` opaque-URL permissiveness, and the
  Pages custom-domain/enablement gap.

## Next Milestone
**CI green and pushed; the code-closable `hubDomain` ForceQuery `normal` just closed. Resume the
front-of-queue WASM-verifier upgrade milestone — but its remaining "published" half is human-blocked, so
prioritize a code-closable target.** In order:
1. **The signature half of the verifier-scope gap** (the milestone's actual trust bar) — browser did:web
   resolution + checkpoint-note signature verify, gating `verified` on signature + id-binding + inclusion.
   `review` recommends a **design-first pass** before building (browser-side did:web resolution is
   non-trivial); a good STOP-candidate if the design is unclear. Until it lands, do NOT loosen
   `verifier.html`'s "hub-signed root" success copy. And/or wire the tier-2 WASM caller into the **hub
   dossier** (no WASM island today).
2. **Remaining code-closable certificate `normal`s** (handoff-named order): the §5 OTS-digest-binding
   `bytes.Equal` gap (`internal/certificate/handler.go:843-845` + `internal/ots`); the tier-2 honesty-copy
   overstatement (`cert.html:465`). (The §6 `· at` timestamp + the `/` Checkpoint/Anchor columns need a
   store schema/projection change — larger.)
3. **Unblock + verify the Pages deploy** (needs the one-time human repo-Settings step: Settings → Pages →
   source "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and
   confirm a `success` deploy — that closes the "published" half of the WASM Verify criterion. A workflow
   file alone cannot self-enable Pages, so flag for the human rather than re-polishing it.

Subsequent: the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the §6 `· at` timestamp and
the `/` Checkpoint/Anchor columns when the projection gains the needed columns; the named-region + `←`
back-link parity pass across the remaining SSR surfaces; the M-UI exit visual-pass + human sign-off
(ADR-0012).
