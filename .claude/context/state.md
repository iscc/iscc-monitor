<!-- assessed-at: 645b73f1066bea972bf6a756ff1b05473da56459 -->

# Project State

## Status: IN_PROGRESS

## Phase: WASM verifier — `.wasm` artifact built + served byte-pinned, but the reproducible-build half is broken; M-UI design-parity front-loaded ahead of the WASM caller by a human steer

The verifier `verify.wasm` is now built and served byte-pinned at `/_ds/verify.wasm` (route +
`application/wasm` + no-cache/strong-ETag/304/405/CORS + `//go:embed` + a mutation-proven hash-pin
test). But the latest **`review` verdict is NEEDS_WORK** (loop CONTINUE): the `mise run build:wasm`
task omits `-buildvcs=false`, so the committed artifact embeds a `+dirty` parent-revision VCS stamp
and its pinned `WasmVerifyHash` cannot be regenerated from the documented task — the reproducible-build
/ published-hash contract (the whole point of the step) is unmet. A **second critical** is open: a
human dev-instance review found `/` (and the SSR chain) far below its authoritative mockup, and
target.md now carries a human **sequencing steer** to front-load M-UI design-parity ahead of the WASM
`<script>` caller and OTS. M1/M2/M3 stay met; M-UI is downgraded from "observable surface complete" to
**partially met** — it is complete behaviorally but below the named-region parity bar.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI: partially met — behavioral Verify largely met, but the design-parity named-region bar is
    NOT.** Met behaviorally: five-status `HubStatusBadge`; DS v2 shared shell (CDN-free, self-hosted
    fonts); `/` realm-index grid; `/<domain>/log/` browser; hub dossier; frozen Exhibit; paginated
    record list; single-record page; ISCC-IDv1 decoder; Hub-List resolver; certificate **§1–§6** +
    proof-bundle endpoint + COMPARISON ANCHOR panel. **Open (now CRITICAL):** `/` lacks its mockup's
    three headline landmark regions — the claim-lookup hero, every-row→dossier links (served HTML has
    **0 `<a>` tags** → no-JS navigation dead-end), and the masthead/instance-identity chrome
    (`verify ↗ monitor.iscc.codes` tier-2 link). target.md's human steer extends the same named-region
    + `←` back-link parity pass across dossier / log browser / single record / certificate. Also still
    open: the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: 1/1 open — and the reproducible-build sub-goal regressed to a CRITICAL.** The
    `.wasm` artifact now exists and is served byte-pinned (SERVE half correct), but the build is
    **non-deterministic** (review measured 0/10 regenerations matching the pinned hash; missing
    `-buildvcs=false`). Still no SSR `<script>` caller (grep of the SSR packages finds no
    `isccVerifyInclusion`/`VerifyJSON` reference), no `monitor.iscc.codes` Independent Verification app.
    Verify ("identical vectors → identical verdicts WASM vs server in-browser; published artifact hash;
    split-view alert") not met.
  - **OTS anchoring: 1/1 open (carried forward, untouched this iteration).** Both observable HTTP
    halves CLOSED (`.ots` serve route + certificate §5 anchor render). What remains: a root that
    actually transits to **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected
    Upgrader. Still 1/1.
- **Last ~10 iterations: ~4-5 milestone-Verify-advancing / ~5-6 foundational·plumbing·hardening.** Arc:
  certificate §5 → COMPARISON ANCHOR panel → `internal/proof/verify` core → `cmd/wasm` entrypoint →
  `/_ds/wasm_exec.js` loader → CDN-free gate re-green → **build+serve `verify.wasm`** (NEEDS_WORK:
  unreproducible). **DRIFT WATCH (clear, but blocked):** the increments are genuinely targeting the
  WASM Verify criterion, not polish — but the last one regressed on its own reproducibility contract
  and a human steer has now repointed the loop at M-UI design-parity (two open criticals). The next
  step is unambiguous: re-do the artifact reproducibly OR honor the steer's `/`-parity item first.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 source touched by the `657ed12..HEAD` diff (confined to
`internal/web/{web.go,web_test.go,verify.wasm}`, `mise.toml`, `CLAUDE.md`, context). All M1 Verify
criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end
with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present**: `cmd/{iscc-monitor,notecheck,wasm}` (+ `cmd/wasm/verifyadapter`); **22 internal
  packages** — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz,
  index, logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`
  (test-only in the build closure), `github.com/nbd-wtf/opentimestamps` (production callers
  `internal/ots` + `internal/otsclient`). `transparency-dev/merkle` backs `internal/proof/verify`, which
  `cmd/wasm/verifyadapter` reuses (`syscall/js` is stdlib).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched. fsck root-rebuild on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror; `inclusion`,
`consistency`, `entries` all served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me at
`GET /<domain>/log/verify?iscc_id=<id>` (routed through the shared `verify.VerifyInclusion` core);
`GET /` realm-index dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong
ETag + `no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **partially met — behaviorally complete, but BELOW the design-parity named-region bar
(now CRITICAL).** All six numbered certificate clauses (§1–§6) + both anchor panels + badge + DS shell
+ `/` index + log browser + hub dossier + frozen Exhibit + record list + single-record page + ISCC-IDv1
decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify. The
prior state's "certificate-observable surface complete" over-claimed: target.md's design-parity bar
(named landmark regions per mockup) is a hard part of M-UI, and a human review found the served surfaces
far below it.
- **Open on the M-UI Verify bar (escalated to CRITICAL by Titusz, 2026-06-22):** `/` is missing its
  mockup's three headline landmark regions — the **claim-lookup hero** (no-JS `GET` form → `/inclusion/…`),
  **every-row→dossier links** (served `/` HTML has 0 `<a>` tags → no-JS navigation dead-end), and the
  **masthead + instance-identity chrome** (logo, instance domain/operator/realm, `verify ↗
  monitor.iscc.codes`). The human steer extends the same named-region + `←` back-link parity pass across
  dossier / log browser / single record / certificate.
- **Still open (carried):** the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012;
  agent-browser tooling on `develop`, per-surface screenshots run in review, full exit pass not executed);
  dossier §4 Bitcoin-anchor / comparison-anchor named regions for full parity.
- **Residual notes (filed `normal`, NOT fixed):** §5 does not bind the OTS proof digest to §2's root;
  `did:web:` + raw `data.Domain` rides §4 AND the bundle (mis-renders a `host:port` DID); `hubDomain`
  fail-opens on a trailing `?` (`ForceQuery`); §6 rows omit the per-record `· at` timestamp.

## WASM verifier · OTS anchoring
**Status**: **WASM — `.wasm` artifact built + served byte-pinned (SERVE half correct), but the
reproducible BUILD half is BROKEN (open CRITICAL); no SSR caller / no `monitor.iscc.codes` app yet.
OTS — both observable HTTP halves landed; only a real Bitcoin confirmation remains (offline-unprovable).**
- **WASM:** `cmd/wasm/main.go` (tagged `//go:build js && wasm`) registers `isccVerifyInclusion`; the pure
  `cmd/wasm/verifyadapter.VerifyJSON` base64-Std-decodes the proof bundle into `verify.VerifyInclusion`.
  `internal/web` now serves the loader `/_ds/wasm_exec.js` (`WasmExecPath`) AND the verifier
  `/_ds/verify.wasm` (`WasmVerifyPath`, `//go:embed verify.wasm`, `application/wasm`,
  no-cache/strong-ETag/304/405/CORS) with a pinned `WasmVerifyHash` =
  `17b0f4f81a0952c3…` (mutation-proven by `TestWasmVerifyHashPinned`).
  **CRITICAL regression (just filed):** `mise.toml`'s `build:wasm` task (line 42) is
  `go build -trimpath -ldflags=-buildid= -o internal/web/verify.wasm ./cmd/wasm` — it OMITS
  `-buildvcs=false`, so Go stamps VCS metadata; the committed blob was built `+dirty` at the PARENT
  revision (`vcs.revision=b2667f86…`, `vcs.modified=true`). Review measured 10× regenerations → the
  pinned hash reproduced **0 times**; `-buildvcs=false` yields a stable `f03b9b89…`. The pin test stays
  green only because it never rebuilds. Fix = add `-buildvcs=false`, rebuild, re-pin, re-commit.
  **Still 1/1 open on the milestone Verify:** no SSR `<script>` caller (grep finds none in
  dashboard/dossier/certificate/proofserve), no standalone `monitor.iscc.codes` Independent Verification
  app, no in-browser identical-verdict parity, no split-view alert.
  **Carried `normal` defect (NOT fixed):** untagged `cmd/wasm/main.go:39-40` reads `index`/`size` via
  `js.Value.Int()` (= `int(v.Float())`), truncating a non-integer JS Number. Not exploitable (no caller;
  the tested `verifyadapter.VerifyJSON` takes `uint64`). Land safe-integer validation when the caller is
  wired.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause, the store layer
  (`internal/store/ots.go`), the off-path stamp/upgrade loop (`OTSTick` in `internal/follower/otsloop.go`),
  the offline classifier (`internal/ots.Confirmed`), and the real calendar transport (`internal/otsclient`)
  are all wired via `main.go`'s `runOTSLoop`. The Verify-closer not yet built: a root reaching
  **Bitcoin-confirmed** — needs a live calendar + real BTC confirmation. Still 1/1 open. **Open `normal`
  defect (filed, NOT fixed):** the production `Stamp` path (`internal/otsclient/client.go:121`) has NEITHER
  a panic-recover NOR a per-request timeout (the symmetric guards the upgrade path got via `safeUpgrade`);
  the next stamp-path touch should add `safeStamp`.

## Quality gates
**Status**: **AMBER — gate runnable and reported green by review at HEAD, BUT the latest review verdict is
NEEDS_WORK with an open critical, and CI is green only at the PRIOR HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. Review
  reported `mise run check` green at HEAD `645b73f` (all 25 packages `ok`; `gofmt -l .` clean). NOTE:
  `mise run check` passing does NOT mean the step is correct — the pinned `WasmVerifyHash` test never
  rebuilds the artifact, so the unreproducible-build critical is latent under the gate (exactly the
  "green-but-wrong" trap target.md warns about for hash-pinned artifacts).
- **Latest `review` verdict: NEEDS_WORK (loop CONTINUE) at HEAD `645b73f`.** This is NOT a PASS — DONE is
  out of reach until the artifact is reproducible AND the open criticals close. NEEDS_WORK was correctly
  not pushed.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR (`go-version: "1.26"`). Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`.
  **The branch is now 4 commits AHEAD of `origin/develop` (0 behind); the latest CI run is `success` only
  at `657ed124` (the PRIOR HEAD), NOT at current HEAD `645b73f`.** Current HEAD is unpushed/un-CI'd — but
  this is expected: the trailing 4 commits are define-next + the NEEDS_WORK advance + reviews, which the
  loop does not push until a PASS.
- **Open issues: 2 `critical`, 6 `normal`, 9 `low`.** Criticals: (1) `verify.wasm` unreproducible
  (`-buildvcs=false` missing); (2) `/` realm index below its mockup's named-region bar (no hero, no row
  links, no masthead chrome — human-escalated). DONE additionally requires 0 critical AND 0 normal, so the
  loop stays CONTINUE.

## Next Milestone
**Two open criticals gate everything; honor the human sequencing steer (target.md, 2026-06-22 — Titusz).**

1. **Re-do the WASM artifact reproducibly (CRITICAL, fastest unblock).** Add `-buildvcs=false` to
   `mise.toml`'s `build:wasm` task, `mise run build:wasm`, re-pin `WasmVerifyHash` to the emitted hash,
   re-commit `verify.wasm`; confirm two builds from different tree states are byte-identical and
   `TestWasmVerifyHashPinned` passes against the rebuild.
2. **M-UI design-parity, front-loaded ahead of the WASM caller (CRITICAL — human steer).** Bring `/` to
   its mockup's named regions: the claim-lookup hero as a no-JS `GET` form → `/inclusion/…`, every row a
   dossier `<a>` link, the masthead logo + instance-identity + `verify ↗` chrome, plus the `#`/checkpoint/
   Bitcoin-anchor columns and "N hubs followed & mirrored" count; assert the landmark regions in the
   dashboard golden test. Then the same named-region + `←` back-link parity pass across dossier / log
   browser / single record / certificate.
3. **Resume the WASM `<script>` caller** (the tier-2 certificate/dossier enhancement; land the
   `js.Value.Int()` safe-integer validation here), then the standalone `monitor.iscc.codes` app.
4. **M-UI exit gate (ADR-0012):** agent-browser visual pass on every SSR surface + human sign-off.
5. **OTS:** the "upgrades to Bitcoin-confirmed" half stays offline-unprovable; fold in `safeStamp`, the §5
   digest-binding, and the `host:port` DID `%3A`-encode when those exact lines are next edited.
