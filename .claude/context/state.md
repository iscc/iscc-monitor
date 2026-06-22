<!-- assessed-at: 95c2a399121dccc065dbe168254f48fc33e060cc -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM verifier — `.wasm` artifact now built reproducibly + served byte-pinned; one critical left (M-UI `/` design-parity, human-front-loaded ahead of the WASM `<script>` caller)

The verifier `verify.wasm` is built deterministically (`-buildvcs=false` added) and served byte-pinned
at `/_ds/verify.wasm`; the committed blob's SHA-256 equals the pinned `WasmVerifyHash`
(`f03b9b89…`) and carries zero VCS strings. The latest **`review` verdict is PASS** (loop CONTINUE),
CI is **green at current HEAD `95c2a39`**, and the branch is in sync with `origin/develop`. M1/M2/M3
stay met; one `critical` blocks DONE — `/` realm index is below its mockup's named-region bar
(no claim-lookup hero, rows not linked, masthead/instance-identity chrome absent).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met — behavioral Verify largely met, but the design-parity named-region bar is
    NOT (1 open CRITICAL).** Met behaviorally: five-status `HubStatusBadge`; DS v2 shared shell
    (CDN-free, self-hosted fonts); `/` realm-index grid; `/<domain>/log/` browser; hub dossier; frozen
    Exhibit; paginated record list; single-record page; ISCC-IDv1 decoder; Hub-List resolver;
    certificate **§1–§6** + proof-bundle endpoint + COMPARISON ANCHOR panel. **Open (CRITICAL):** `/`
    lacks its mockup's three headline landmark regions — verified at HEAD: `dashboard.html` has **0
    `<a>` tags** (no-JS dead-end), **0** claim-lookup hero / `GET`→`/inclusion` form, **0**
    instance-identity / `verify ↗ monitor.iscc.codes` chrome. target.md's human steer extends the same
    named-region + `←` back-link parity pass across dossier / log browser / single record / certificate.
    Also still open: the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: 1/1 open — but the reproducible-build sub-goal is now CLOSED (was CRITICAL).** The
    `.wasm` artifact exists, is served byte-pinned, AND builds reproducibly (review independently
    rebuilt across 4 tree states → byte-identical; `strings | grep vcs.` → 0). Still no SSR `<script>`
    caller (grep of the SSR packages finds no `isccVerifyInclusion`/`VerifyJSON` reference), no
    `monitor.iscc.codes` Independent Verification app, no in-browser identical-verdict parity, no
    split-view alert. Verify not met.
  - **OTS anchoring: 1/1 open (carried forward, untouched this iteration).** Both observable HTTP halves
    CLOSED (`.ots` serve route + certificate §5 anchor render). What remains: a root that actually
    transits to **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~5 milestone-Verify-advancing / ~5 foundational·plumbing·hardening.** Arc:
  certificate §5 → COMPARISON ANCHOR panel → `internal/proof/verify` core → `cmd/wasm` entrypoint →
  `/_ds/wasm_exec.js` loader → CDN-free gate re-green → build+serve `verify.wasm` → **fix reproducible
  build (`-buildvcs=false`, re-pin)**. **DRIFT WATCH (clear):** increments are genuinely targeting the
  WASM Verify criterion, not polish; the one regression (unreproducible artifact) was caught and closed
  the next iteration. The loop is now repointed at the M-UI `/`-parity critical per the human steer.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `645b73f..HEAD` diff (confined to
`internal/web/{web.go,verify.wasm}`, `mise.toml`, `learnings/web.md`, context). All M1 Verify criteria
remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze
+ alert-once + restart survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}`; **22 internal packages** — `badge,
  certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`. 68 `_test.go` files.
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
**Status**: **partially met — behaviorally complete, but BELOW the design-parity named-region bar (1
open CRITICAL).** All six numbered certificate clauses (§1–§6) + both anchor panels + badge + DS shell +
`/` index + log browser + hub dossier + frozen Exhibit + record list + single-record page + ISCC-IDv1
decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify. The
design-parity bar (named landmark regions per mockup) is a hard part of M-UI and `/` falls short of it.
- **Open on the M-UI Verify bar (CRITICAL, human-escalated 2026-06-22):** `/`
  (`internal/dashboard/dashboard.html` + `handler.go`) is missing its mockup's three headline landmark
  regions — verified at HEAD: the **claim-lookup hero** (no-JS `GET` form → `/inclusion/…`) absent,
  **every-row→dossier links** absent (served `/` HTML has **0 `<a>` tags** → no-JS navigation
  dead-end), and the **masthead + instance-identity chrome** (logo, instance domain/operator/realm,
  `verify ↗ monitor.iscc.codes`) absent. Also missing vs mockup: `#` row number, Checkpoint-size column,
  Bitcoin-anchor dot+label column, "N hubs followed & mirrored" count. The human steer extends the same
  named-region + `←` back-link parity pass across dossier / log browser / single record / certificate.
- **Still open (carried):** the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012;
  agent-browser tooling on `develop`, per-surface screenshots run in review, full exit pass not
  executed); dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.
- **Residual notes (filed `normal`, NOT fixed):** §5 does not bind the OTS proof digest to §2's root;
  `did:web:` + raw `data.Domain` rides §4 AND the bundle (mis-renders a `host:port` DID); `hubDomain`
  fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — `.wasm` artifact built reproducibly + served byte-pinned (SERVE + reproducible
BUILD both correct); no SSR caller / no `monitor.iscc.codes` app yet. OTS — both observable HTTP halves
landed; only a real Bitcoin confirmation remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the
  pure `cmd/wasm/verifyadapter.VerifyJSON` base64-Std-decodes the proof bundle into
  `verify.VerifyInclusion`. `internal/web` serves the loader `/_ds/wasm_exec.js` AND the verifier
  `/_ds/verify.wasm` (`WasmVerifyPath`, `//go:embed verify.wasm`, `application/wasm`,
  no-cache/strong-ETag/304/405/CORS) with a pinned `WasmVerifyHash =
  f03b9b8973e308be12dd7d0c210e612d0aa388e6e8a7cdce4ea8515cd823047e` (mutation-proven by
  `TestWasmVerifyHashPinned`). **Reproducibility CLOSED:** `mise.toml`'s `build:wasm` now sets
  `-buildvcs=false`; committed blob sha256 == pinned const, zero `vcs.` strings, review rebuilt
  byte-identical across clean/untracked-dirty/tracked-dirty/`go clean -cache`.
  **Still 1/1 open on the milestone Verify:** no SSR `<script>` caller (grep finds none in
  dashboard/dossier/certificate/proofserve), no standalone `monitor.iscc.codes` Independent Verification
  app, no in-browser identical-verdict parity, no split-view alert.
  **Carried `normal` defect (NOT fixed):** untagged `cmd/wasm/main.go:39-40` reads `index`/`size` via
  `js.Value.Int()` (= `int(v.Float())`), truncating a non-integer JS Number. Not exploitable (no caller;
  the tested `verifyadapter.VerifyJSON` takes `uint64`). Land safe-integer validation when the caller is
  wired.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in
  `internal/follower/otsloop.go`), the offline classifier (`internal/ots.Confirmed`), and the real
  calendar transport (`internal/otsclient`) are all wired via `main.go`'s `runOTSLoop`. The Verify-closer
  not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation.
  Still 1/1 open. **Open `normal` defect (filed, NOT fixed):** the production `Stamp` path
  (`internal/otsclient/client.go:121`) has NEITHER a panic-recover NOR a per-request timeout (the
  symmetric guards the upgrade path got via `safeUpgrade`); the next stamp-path touch should add
  `safeStamp`.

## Quality gates
**Status**: **GREEN — gate runnable, latest `review` verdict is PASS, and CI is green at current HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Review reported `mise run check` green at HEAD (all 25 packages `ok`; `gofmt -l .` clean).
- **Latest `review` verdict: PASS (loop CONTINUE)** at HEAD `95c2a39`. The reproducible-build critical
  was independently re-verified and closed.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch
  `develop`. **Branch is in sync with `origin/develop`; the latest CI run is `success` at current HEAD
  `95c2a39`** (run 27932310164).
- **Open issues: 1 `critical`, 6 `normal`, 9 `low`.** The lone critical is the `/` realm-index
  named-region parity gap (no hero, no row links, no masthead chrome — human-escalated). DONE requires
  0 critical AND 0 normal, so the loop stays CONTINUE.

## Next Milestone
**M-UI `/` design-parity — the one open critical, human-front-loaded ahead of the WASM `<script>`
caller (target.md steer, Titusz 2026-06-22).** Bring `/` to its mockup's named regions:
1. **claim-lookup hero** as a no-JS `GET` form → `/inclusion/…` (the primary call to action), above the
   register.
2. **every hub row wrapped in an `<a href>`** to its dossier (anchor count ≥ hub count) — restore the
   realm-index→dossier traversal so no-JS navigation is not a dead end.
3. **masthead logo + instance-identity block + `verify ↗ monitor.iscc.codes`** chrome; plus the
   `#`/Checkpoint-size/Bitcoin-anchor columns and "N hubs followed & mirrored" count.
Assert those landmark regions in the `internal/dashboard` handler golden test; run the ADR-0012 visual
pass against the mockup. Then extend the same named-region + `←` back-link parity pass across dossier /
log browser / single record / certificate.

Subsequent: resume the WASM `<script>` caller (land the `js.Value.Int()` safe-integer validation there),
then the standalone `monitor.iscc.codes` app; the M-UI exit visual-pass + human sign-off (ADR-0012); and
the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable) plus folding in `safeStamp`, the §5
digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are next edited.
