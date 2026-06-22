<!-- assessed-at: 25c12af1d468a91e911de4fa7369df34359aa869 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI `/` realm-index named-region parity LANDED (lone critical closed) — the WASM `<script>` verifier caller is now the front of the queue

The `/` realm index reached its mockup's three headline landmark regions — a no-JS claim-lookup hero
(`<form method="get" action="/inclusion/">`), every hub row wrapped in an `<a href="/{{.Domain}}">`
dossier link (navigation closure restored), and the masthead instance-identity block +
`verify ↗ monitor.iscc.codes` tier-2 link — with a lockstep `?iscc_id=` query fallback on the
certificate handler so the no-JS form resolves. The latest **`review` verdict is PASS** (loop
CONTINUE), CI is **green at current HEAD `25c12af`**, and the branch is in sync with `origin/develop`.
M1/M2/M3 stay met; **zero `critical`** issues remain, but the WASM and OTS milestone Verify criteria
are still open and 7 `normal` issues are filed — so DONE is not reached.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: behavioral Verify met; design-parity headline-region bar for `/` NOW MET (lone critical
    closed).** Met: five-status `HubStatusBadge`; DS v2 shared shell (CDN-free, self-hosted fonts);
    `/` realm-index grid + claim-lookup hero + per-row dossier links + instance-identity masthead;
    `/<domain>/log/` browser; hub dossier; frozen Exhibit; paginated record list; single-record page;
    ISCC-IDv1 decoder; Hub-List resolver; certificate **§1–§6** + proof-bundle endpoint + COMPARISON
    ANCHOR panel. **Still open (NOT critical):** the named-region + `←` back-link parity pass has not
    yet been carried to the remaining SSR surfaces (dossier / log browser / single record /
    certificate lack the masthead instance-identity block + full back-link chain), `/` sub-region
    deltas (logo asset, config-driven instance identity, Checkpoint/Anchor data columns) filed
    `normal`, and the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
  - **WASM verifier: 1/1 open.** The `.wasm` artifact exists, is served byte-pinned at `/_ds/verify.wasm`,
    AND builds reproducibly (closed last iteration). Still NO SSR `<script>` caller (grep of
    dashboard/dossier/certificate/proofserve finds no `isccVerifyInclusion`/`verify.wasm`/`wasm_exec`
    reference), no standalone `monitor.iscc.codes` Independent Verification app, no in-browser
    identical-verdict parity, no split-view alert. Verify not met.
  - **OTS anchoring: 1/1 open (carried forward, untouched this iteration).** Both observable HTTP
    halves CLOSED (`.ots` serve route + certificate §5 anchor render). What remains: a root that
    actually transits to **Bitcoin-confirmed** — offline-unprovable; exercised only against an
    injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 foundational·plumbing·hardening.** Arc:
  COMPARISON ANCHOR panel → `internal/proof/verify` core → `cmd/wasm` entrypoint → `/_ds/wasm_exec.js`
  loader → CDN-free gate re-green → build+serve `verify.wasm` → fix reproducible build → **`/`
  realm-index named-region parity (hero + row links + masthead)**. **DRIFT WATCH (clear):** increments
  are genuinely closing milestone Verify criteria, not polish — this iteration closed the lone open
  M-UI critical. The loop is now repointed at the WASM `<script>` caller per the review/handoff steer.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `95c2a39..HEAD` diff (confined to
`internal/dashboard/{dashboard.html,handler.go,handler_test.go}`, `internal/certificate/{handler.go,
handler_test.go}`, and context/learnings). All M1 Verify criteria remain satisfied: `origin`/`vkey`
golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart survival;
structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}`; **22 internal packages** — `badge,
  certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, web`.
  Module `github.com/iscc/iscc-monitor`, `go 1.26.1`. 68 `_test.go` files.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`. `transparency-dev/merkle` backs `internal/proof/verify`, which
  `cmd/wasm` reuses (`syscall/js` is stdlib).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core);
`GET /` realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally complete AND the `/` headline-region design-parity bar now MET — the lone
open critical is CLOSED.** All six numbered certificate clauses (§1–§6) + both anchor panels + badge +
DS shell + `/` index (now with hero + per-row dossier links + instance-identity masthead) + log
browser + hub dossier + frozen Exhibit + record list + single-record page + ISCC-IDv1 decoder +
Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify.
- **Landed this iteration (verified at HEAD):** `internal/dashboard/dashboard.html` now carries
  `<a class="chrome-verify" href="https://monitor.iscc.codes/">`, a `<form class="hero-form"
  method="get" action="/inclusion/">` claim-lookup hero, and `<a class="ledger-row"
  href="/{{.Domain}}">` per-row dossier links — restoring no-JS navigation closure. The certificate
  handler accepts the lockstep `?iscc_id=` query fallback (`handler.go:472`). New tests
  `TestDashboardRendersHeroAndNavigation` and `TestCertificateQueryFallback` are present and
  mutation-proven per the review.
- **Still open (NOT critical, carried):** the same named-region + `←` back-link parity pass has not
  been carried to the remaining SSR surfaces (dossier / log browser / single record / certificate
  lack the masthead instance-identity block + full back-link chain); `/` sub-region deltas (no
  self-hosted logo asset, static rather than config-driven instance identity/realm, absent
  Checkpoint/Bitcoin-anchor data columns, omitted recent-declarers footer) filed `normal`; the
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed (the per-surface
  visual pass runs in review; the full exit pass has not).
- **Residual notes (filed `normal`, NOT fixed):** §5 does not bind the OTS proof digest to §2's root;
  `did:web:` + raw `data.Domain` rides §4 AND the bundle (mis-renders a `host:port` DID); `hubDomain`
  fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — `.wasm` artifact built reproducibly + served byte-pinned; no SSR caller / no
`monitor.iscc.codes` app yet (this is now the next milestone). OTS — both observable HTTP halves
landed; only a real Bitcoin confirmation remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  pure `cmd/wasm/verifyadapter.VerifyJSON` base64-Std-decodes the proof bundle into
  `verify.VerifyInclusion`. `internal/web` serves the loader `/_ds/wasm_exec.js` AND the verifier
  `/_ds/verify.wasm` (pinned `WasmVerifyHash`, mutation-proven; built with `-buildvcs=false`,
  byte-identical rebuild confirmed in review). **Still 1/1 open on the milestone Verify:** no SSR
  `<script>` caller (grep of dashboard/dossier/certificate/proofserve finds none), no standalone
  `monitor.iscc.codes` Independent Verification app, no in-browser identical-verdict parity, no
  split-view alert. **Carried `normal` defect (NOT fixed):** untagged `cmd/wasm/main.go:39-40` reads
  `index`/`size` via `js.Value.Int()` (= `int(v.Float())`), truncating a non-integer JS Number — land
  safe-integer validation when the caller is wired.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the real
  calendar transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. The
  Verify-closer not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real
  BTC confirmation. Still 1/1 open. **Open `normal` defect (filed, NOT fixed):** the production `Stamp`
  path (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the
  symmetric guards the upgrade path got via `safeUpgrade`); the next stamp-path touch should add
  `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS, and CI is green at current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (all 25 packages `ok`; `gofmt -l .` clean).
- **Latest `review` verdict: PASS (loop CONTINUE)** at HEAD `25c12af`. The `/` named-region parity
  critical was independently verified (mutation-proven, visual-pass faithful, Codex clean) and closed.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`. **Branch is in sync with `origin/develop`; the latest CI run is `success` at current HEAD
  `25c12af`** (run 27933128579).
- **Open issues: 0 `critical`, 7 `normal`, 9 `low`.** DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE. The 7 normal issues span: certificate §5 digest binding, OTS `safeStamp` guard,
  `hubDomain` ForceQuery gap, §4/bundle `host:port` DID encode, certificate §6 timestamp, WASM shim
  `Int()` truncation, and the `/` sub-region parity deltas.

## Next Milestone
**WASM verifier `<script>` caller — the front of the queue now the `/` critical is closed.** Wire the
served `/_ds/verify.wasm` + `/_ds/wasm_exec.js` into a real SSR caller so the certificate/dossier
tier-2 ("your browser verified…") result lights up against the proof bundle, and land the
`cmd/wasm/main.go:39-40` `js.Value.Int()` safe-integer validation at that first caller (open `normal`).
Then the standalone `monitor.iscc.codes` Independent Verification app (in-browser identical-verdict
parity + the guided split-view alert).

Subsequent: carry the named-region + `←` back-link parity pass across dossier / log browser / single
record / certificate (the deferred SSR surfaces); the M-UI exit visual-pass + human sign-off
(ADR-0012); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in
`safeStamp`, the §5 digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are
next edited.
