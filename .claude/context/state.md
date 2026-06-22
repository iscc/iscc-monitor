<!-- assessed-at: 5e4725cd0ef3fb65d293574c0fdb39ec35aad3b5 -->

# Project State

## Status: IN_PROGRESS

## Phase: masthead-identity arc continues — config-driven instance identity now on the hub-dossier masthead; gate green + pushed.
The hub-dossier masthead is now honest per-deployment: `dossier.Handler` takes the SAME
`dashboard.Identity` value the `/` page already receives, renders `{{.Instance}}`/`{{.Operator}}`
through a byte-identical port of the dashboard's `chrome-identity` block, and applies a dossier-local
`resolveIdentity` fail-safe so an unconfigured binary renders today's static masthead. Latest `review`
verdict is **PASS_WITH_NOTES (loop CONTINUE)**: mutation-proven on BOTH the template binding and the
wiring, Codex clean, agent-browser visual parity confirmed. DONE not reached: the WASM "published" +
signature halves and the OTS Bitcoin-confirmed criterion stay open, and 6 `normal` issues stand.

Incremental review against assessed-at `33fbae2`. The `33fbae2..HEAD` diff touched exactly TWO
production files — `internal/dossier/handler.go` (imports `dashboard`, adds the duplicated
`instanceFallback`/`operatorFallback` consts + `resolveIdentity` + `Instance`/`Operator` fields on
`dossierData`, `Handler`/`buildData` gain the identity args) and `internal/dossier/dossier.html` (the
`.chrome-identity`/`.chrome-operator` masthead block replaces the static `monitor instance` span) —
plus `cmd/iscc-monitor/main.go` (threads `id` through `mirrorHandler` → `dossier.Handler`), the test
file (`handler_test.go` adds `TestDossierRendersInstanceIdentity`), and `.claude/context/*` +
`learnings/dashboard.md`. Verified: no signature / RFC-6962 / Merkle / `proof` / `didweb` / `logclient`
/ `follower` / store / WASM / OTS source changed (pure HTML render of masthead strings + handler
plumbing). All M1/M2/M3/WASM/OTS sections carry forward met/open as before. `git status` clean; HEAD
(`5e4725c`) == `origin/develop` (0 ahead / 0 behind).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact (byte-pinned
    `internal/web/verify.wasm` carries the 6-arg id-binding shim; pin `WasmVerifyHash` matches;
    reproducible from `mise run build:wasm`). **Still OPEN on the milestone Verify** ("the verifier
    artifact … published value"): the Pages deploy **does not run** — the latest Pages run at HEAD
    (27959929167) is `failure` at `Configure Pages` (Pages not enabled / not set to "GitHub Actions"
    source), built-but-undeployed, a one-time human repo-Settings step. Also still open: **NO dossier
    tier-2 WASM caller** (no WASM island in `internal/dossier`); and the **cross-origin verifier-scope
    SIGNATURE gap** (the WASM core verifies inclusion math + id-binding only — no checkpoint-signature /
    did:web key resolution — a malicious monitor can still render a green `verified`; design-first
    remainder, STOP-candidate).
  - **OTS anchoring: 1/1 open (carried unchanged).** All three observable HTTP halves are closed (`.ots`
    serve route + the §5 anchor render, digest-bound via `ots.ConfirmedFor`) and BOTH calendar-transport
    guards (`safeUpgrade` + `safeStamp`) are in place. What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~0 milestone-Verify-advancing / ~10 chrome·plumbing·hardening.** Recent arc:
  log-browser `Type` column → **`/` masthead config-driven instance identity** → **hub-dossier masthead
  config-driven instance identity (this iteration, PASS_WITH_NOTES).** **DRIFT WATCH (amber):** no
  *milestone-level* Verify criterion has closed in ~20 increments (the last were the WASM id-binding
  slices ~15 iters back), but the loop is legitimately draining the code-closable M-UI masthead-identity
  arc rather than re-polishing met surfaces. The front-of-queue WASM "published" half stays
  **human-blocked, not code-blocked** (a workflow file cannot self-enable Pages), so the loop correctly
  pivots off it. The masthead-identity arc has a clear runway: the SAME `dashboard.Identity` value is now
  on TWO of six SSR mastheads (`/` + dossier); FOUR remain (certificate, the three proofserve surfaces),
  one ≤3-file sub-step each, with the env parsing scheduled to move into the `internal/config` leaf on the
  NEXT (certificate) surface — closing the config-move `normal`. After that arc, the remaining work is
  design-first (WASM signature half, per-hub Anchor honesty, DB migration) or human-blocked (Pages,
  custom domain) — flag a STOP/design candidate rather than re-polishing met surfaces.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `33fbae2..HEAD` diff touched no signature / RFC-6962 / Merkle
/ `proof` / `didweb` / `logclient` / `follower` file (the only production edits are the dossier masthead
+ `main.go` identity wiring). All M1 Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once + restart survival; structured
logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal packages
  — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, verifier,
  web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1` language directive in `go.mod`; build toolchain
  `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest / store path
touched this diff. fsck root-rebuild on every verified non-frozen poll; inclusion cross-check
conformance-tested over the real verified mirror; `inclusion`, `consistency`, `entries` all served from
the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 handler trust-path source touched. The
only M3-surface change this diff is the hub-dossier masthead identity (presentation, not a trust-path or
behavioral-Verify change). CORS on every public GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`
(routed through the shared `verify.VerifyInclusion` core); `GET /` realm-index dashboard; `GET
/<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + chrome complete; the M-UI exit visual-pass + human sign-off is pending.**
This diff threaded config-driven instance identity onto the hub-dossier masthead: `dossier.Handler` now
takes the SAME `dashboard.Identity` value the `/` page receives, renders `{{.Instance}}`/`{{.Operator}}`
through a byte-identical port of the dashboard `chrome-identity` block, with a dossier-local
`resolveIdentity` fail-safe so an unconfigured binary renders today's static masthead. Visual pass
confirmed the live dossier masthead matches the mockup's instance-identity region. All six certificate
clauses (§1–§6) + both anchor panels + badge + DS shell + `/` index (six-column ledger + claim-lookup
hero + per-row dossier link + config-driven masthead) + log browser (full 4-column record list) + hub
dossier (config-driven masthead + `← Realm index` back-link) + frozen Exhibit + single-record page +
ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam
Verify; every SSR masthead carries the shared chrome (self-hosted logo + text mark + divider).
- **Still open (NOT critical, carried):** instance identity is now config-driven on TWO of six SSR
  mastheads (`/` + dossier); the SAME `dashboard.Identity` is NOT yet threaded into the remaining FOUR
  (certificate + the three proofserve surfaces still render the static `monitor instance` placeholder) —
  the follow-on arc, one ≤3-file sub-step each. The named-region + `←` back-link parity pass is on `/` +
  dossier but NOT yet on every SSR surface (log browser / single record / certificate still lack the full
  instance-identity block). The remaining `/` sub-region delta is the "recent declarers checked" hero
  footer (needs a recent-lookup history the store does not track), filed `normal` (#214 sub-4). The
  mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes:** the realm-index Anchor cell is a per-HUB latest-stamped-root indicator NOT
  tied to the displayed Checkpoint — a design-honesty question for the M-UI exit (spec-faithful,
  non-blocking). The config-move `normal` STAYS OPEN (identity env keys still read inline in `main.go`
  via `os.Getenv`, not validated via `internal/config`; CLAUDE.md's env table lacks them) — deferred to
  the certificate (second-surface) slice. The §6 humanization is intentional ADR-0008-deferred visual
  polish.
- **NEW `low` filed this iteration (×2):** the masthead identity fallback consts are now duplicated
  across dashboard + dossier (cert next would be the third copy) — consolidate into one shared resolve
  leaf when the arc finishes; and the stale `.chrome-identity` CSS comment in `dashboard.html:75-76` still
  says "static copy in this skeleton" (inaccurate since `b30b84e`) — tidy when `dashboard.html` is next
  touched.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN (published half human-blocked, signature half design-blocked),
reproducibility critical CLOSED, HEAD is PASS. OTS — §5 render digest-bound + both calendar-transport
guards landed; only a real Bitcoin confirmation remains (offline-unprovable).** Neither the WASM core nor
OTS source was touched by this diff; both sections carry forward.
- **WASM (carried):** the id-binding half of the verifier-scope trust gap is closed IN SOURCE and
  ARTIFACT, and the artifact is reproducible from its documented command: the committed
  `internal/web/verify.wasm` carries the 6-arg/id-binding shim (`verifyadapter.RecordCommitsID`), its
  SHA-256 matches `WasmVerifyHash` (`internal/web/web.go`), and `mise run build:wasm` deterministically
  re-emits those bytes (`TestWasmVerifyHashPinned` green). `cmd/wasm/main.go` registers
  `isccVerifyInclusion`; `cert.html` embeds the tier-2 island; `internal/verifier` is a SINGLE STATIC
  cross-origin artifact; `.github/workflows/pages.yml` is the PUBLISH workflow; `verifier.Handler` is NOT
  mounted in `cmd/iscc-monitor`. **Still 1/1 OPEN on the milestone Verify** — the deploy is not live
  (Pages `failure` at `Configure Pages` at HEAD, run 27959929167), needs the one-time human repo-Settings
  step. **Carried `normal` defect:** the verifier core does NO checkpoint-signature / did:web check (a
  malicious cross-origin monitor can render green `verified`; success copy overstates a key check that
  never runs) — design-first remainder / STOP-candidate. **Carried `normal`:** the Pages custom-domain /
  Actions-source enablement gap. **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause (digest-bound via
  `ots.ConfirmedFor`), the store layer (`internal/store/ots.go`), the off-path stamp/upgrade loop
  (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier (`internal/ots`), and the
  calendar transport (`internal/otsclient`, with BOTH `safeUpgrade` + `safeStamp` guards) are all wired
  via `main.go`'s `runOTSLoop`. The Verify-closer not yet built: a root reaching **Bitcoin-confirmed** —
  needs a live calendar + real BTC confirmation. Still 1/1 open. **Carried `low` defect:** nil-Stamper +
  empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only nil path).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`5e4725c`) == `origin/develop` (run
27959929215); the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate
defect). Latest `review` verdict is PASS_WITH_NOTES (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build toolchain
  `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run check` green
  at HEAD (build + vet + `go test ./...`, all 27 packages ok; `gofmt -l .` empty); the dossier masthead
  identity is mutation-proven non-vacuous on BOTH the template binding ({{.Instance}}→literal → test
  fails) and the wiring (force `resolveIdentity` to overwrite the supplied value → test fails). The
  trust-root oracles are unaffected — no signature / RFC-6962 / Merkle / `proof` / `didweb` / store source
  changed (pure HTML render of masthead strings; oracle gate correctly N/A).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`5e4725c`) is
  pushed (0 ahead / 0 behind upstream); CI run 27959929215 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27959929167) at HEAD
  is **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions" source);
  deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 6 normal, 12 low** (the lone "critical" grep hit is the format-template
  legend on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
  Net +2 low this iteration (masthead-const dup; stale dashboard.html chrome comment), 6 normal unchanged.
  The 6 normal: the DB-migration hazard, the lone `/` "recent declarers checked" footer (#214 sub-4), the
  WASM verifier-scope SIGNATURE-half gap, the Pages custom-domain/enablement gap, the per-hub-Anchor
  design-honesty question, and the config-move/docs follow-up.

## Next Milestone
**CI green and pushed; the dossier masthead identity is closed — the masthead-identity arc continues to
the certificate (where the config-leaf move is scheduled), then the proofserve surfaces.** In order:
1. **Thread the SAME `dashboard.Identity` into `internal/certificate`** (`cert.html` carries the same
   static `monitor instance` placeholder — the lockstep twin), one ≤3-file sub-step. On THIS second
   masthead-arc surface, **move the env parsing into the `internal/config` `optional(get, key, fallback)`
   leaf** (ratifying the `ISCC_MONITOR_REALM_NAME` key name there) so all surfaces draw from one validated
   source — closing the config-move `normal` — and add the three keys to CLAUDE.md's env table. Folding
   `Identity` + the fallback consts + a single exported `Resolve` into one shared leaf at that point also
   closes the new const-dup `low`. After cert: the proofserve surfaces (`browser.html`, `records.html`,
   `record.html`). `internal/verifier` stays EXCLUDED (its `.codes` chrome is the verifier-app identity).
   This is the strongest next code-only candidate — prefer it over re-polishing met surfaces.
2. **Design-first pass on the signature half of the verifier-scope gap** (the WASM milestone's actual
   trust bar) — browser did:web resolution + checkpoint-note signature verify, gating `verified` on
   signature + id-binding + inclusion. `review` recommends a **design pass before building**; a good
   STOP-candidate. Until it lands, do NOT loosen `verifier.html`'s "hub-signed root" success copy. And/or
   wire the tier-2 WASM caller into the **hub dossier** (no WASM island today).
3. **Unblock + verify the Pages deploy** (one-time human repo-Settings step: Settings → Pages → source
   "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and confirm
   a `success` deploy — closes the "published" half of the WASM Verify criterion. A workflow file alone
   cannot self-enable Pages; flag for the human rather than re-polishing it.

Subsequent: the realm-index per-hub-vs-per-checkpoint Anchor honesty design pass; the OTS "upgrades to
Bitcoin-confirmed" half (offline-unprovable); the named-region + `←` back-link parity pass across the
remaining SSR surfaces (log browser / single record / certificate); the `/` "recent declarers checked"
footer (needs store history); the no-on-disk-DB-migration mechanism (the project's first migration
framework); the M-UI exit visual-pass + human sign-off (ADR-0012).
