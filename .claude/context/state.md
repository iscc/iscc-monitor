<!-- assessed-at: a9a700e14309cfa42a38b173a83a9c81bdf05113 -->

# Project State

## Status: IN_PROGRESS

## Phase: Log-browser record-list `Type` column landed (completes the 4-column `Seq · Type · ISCC-ID · Logged` head); gate green + pushed.
The record-list (`GET /<domain>/log/records`) now renders the mockup's per-row Type badge
(declaration / deletion / unknown) projected from the verbatim `note.$schema` (ADR-0008), completing
the mockup's full `Seq · Type · ISCC-ID · Logged` head. Latest `review` verdict is **PASS (loop
CONTINUE)**: mutation-proven non-vacuous, Codex clean, agent-browser visual pass matches the mockup's
Type named region with no new delta. DONE not reached: the WASM "published" + signature halves and the
OTS Bitcoin-confirmed criterion stay open, and 5 `normal` issues stand.

Incremental review against assessed-at `64b34f9`. The `64b34f9..HEAD` diff touched exactly ONE
production file (`internal/proofserve/handler.go` — a handler-local `recordRowVM` view-model +
`recordKindKey` CSS-token mapping) plus its template (`records.html` — the Type badge cell + grid) and
test (`records_test.go` — new `TestRecordsRendersTypeColumn`); the rest is `.claude/context/*`.
Verified: the name-only diff over every trust-root glob (`internal/proof/`, `logclient`, `didweb`,
`follower`, `index`, `cmd/wasm`, `verifier`, `internal/web/`, `ots`, `certificate`, `dossier`, `store`,
`go.mod`, `go.sum`, `schema.sql`) is **empty** — no signature / RFC-6962 / Merkle / store / WASM / OTS
source changed. All M1/M2/M3/WASM/OTS sections carry forward met/open as before. `git status` clean;
HEAD (`a9a700e`) == `origin/develop` (0 ahead / 0 behind).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact (byte-pinned
    `internal/web/verify.wasm` carries the 6-arg id-binding shim; pin `WasmVerifyHash` matches;
    reproducible from `mise run build:wasm`). **Still OPEN on the milestone Verify** ("the verifier
    artifact … published value"): the Pages deploy **does not run** — the latest Pages run at HEAD
    (27957180974) is `failure` at `Configure Pages` (Pages not enabled / not set to "GitHub Actions"
    source), built-but-undeployed, a one-time human repo-Settings step. Also still open: **NO dossier
    tier-2 WASM caller** (no WASM island in `internal/dossier`); and the **cross-origin verifier-scope
    SIGNATURE gap** (the WASM core verifies inclusion math + id-binding only — no checkpoint-signature /
    did:web key resolution — a malicious monitor can still render a green `verified`; design-first
    remainder, STOP-candidate).
  - **OTS anchoring: 1/1 open (carried unchanged).** All three observable HTTP halves are closed (`.ots`
    serve route + the §5 anchor render, digest-bound via `ots.ConfirmedFor`) and BOTH calendar-transport
    guards (`safeUpgrade` + `safeStamp`) are in place. What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~1 milestone-Verify-advancing / ~9 chrome·plumbing·hardening.** Recent arc:
  `/` realm-index Checkpoint/Anchor columns → log-browser `Logged` column → **log-browser `Type` column
  (this iteration, PASS — completes the 4-column record-list head).** **DRIFT WATCH (amber):** no
  *milestone-level* Verify criterion has closed in ~18 increments (the last were the WASM id-binding
  slices ~13 iters back), but the loop is legitimately draining the code-closable M-UI named-region
  backlog rather than re-polishing met surfaces. The front-of-queue WASM "published" half stays
  **human-blocked, not code-blocked** (a workflow file cannot self-enable Pages), so the loop correctly
  pivots off it. **The log-browser surface is now exhausted of pure-code named-region slices** — `review`
  flags this was the LAST one on that surface; the remaining log-browser deltas (pager buttons,
  jump-to-sequence JS input, `← <hub> dossier` back-link) are design-first or human-blocked. The next
  pickable code-only slice shifts surface or to an open `normal`: the `/` realm-index config-driven
  instance-identity copy (issue #214 sub-2 — code-only, unblocks all six SSR mastheads, most-referenced
  remaining `normal`) is the strongest candidate. After it, the remaining work is design-first (WASM
  signature half, per-hub Anchor honesty, DB migration) or human-blocked (Pages, custom domain) — flag a
  STOP/design candidate rather than re-polishing met surfaces.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `64b34f9..HEAD` diff touched no signature / RFC-6962 / Merkle
/ `proof` / `didweb` / `logclient` / `follower` file (the only production edit is `handler.go`'s
record-row Type view-model + `records.html`/`records_test.go`). All M1 Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with freeze + alert-once +
restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`,
  `go 1.26.1` language directive in `go.mod`; build toolchain `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest / store path
touched this diff. fsck root-rebuild on every verified non-frozen poll; inclusion cross-check
conformance-tested over the real verified mirror; `inclusion`, `consistency`, `entries` all served from
the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 handler trust-path source touched.
CORS on every public GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>` (routed through the
shared `verify.VerifyInclusion` core); `GET /` realm-index dashboard; `GET /<domain>/log/` log browser.
All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + chrome complete; the M-UI exit visual-pass + human sign-off is pending.**
This diff landed the log-browser record-list `Type` column — a per-row badge (Declaration / Deletion /
Unknown record type) keyed via `data-kind` and projected from the verbatim `note.$schema` through
`recordKind`/`recordKindKey` (the single mapping site) — completing the mockup's 4-column
`Seq · Type · ISCC-ID · Logged` head. All six certificate clauses (§1–§6) + both anchor panels + badge +
DS shell + `/` index (six-column ledger) + log browser (now full 4-column record list) + hub dossier
(with shared chrome masthead + `← Realm index` back-link) + frozen Exhibit + single-record page +
ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam
Verify; every SSR masthead carries the shared chrome (self-hosted logo + text mark + divider).
- **Still open (NOT critical, carried):** the log-browser surface is now exhausted of pure-code
  named-region slices; the remaining log-browser deltas (pager prev/next buttons, jump-to-sequence JS
  input, `← <hub> dossier` back-link) are design-first or human-blocked. The named-region + `←` back-link
  parity pass is on the dossier but NOT yet on the remaining SSR surfaces (log browser / single record /
  certificate still lack the full instance-identity block + back-link chain on every surface); the `/`
  sub-region deltas — config-driven instance identity/realm name (needs `internal/config` env wiring) and
  "recent declarers checked" footer (needs a recent-lookup history the store does not track) — filed
  `normal`; the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes:** the realm-index Anchor cell is a per-HUB latest-stamped-root indicator NOT
  tied to the displayed Checkpoint (issue:326) — a design-honesty question for the M-UI exit
  (spec-faithful, non-blocking). The §6 humanization is intentional ADR-0008-deferred visual polish.

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
  (Pages `failure` at `Configure Pages` at HEAD, run 27957180974), needs the one-time human repo-Settings
  step. **Carried `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a
  malicious cross-origin monitor can render green `verified`; success copy overstates a key check that
  never runs) — design-first remainder / STOP-candidate; the Pages custom-domain / Actions-source
  enablement gap. **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause (digest-bound via
  `ots.ConfirmedFor`), the store layer (`internal/store/ots.go`), the off-path stamp/upgrade loop
  (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier (`internal/ots`), and the
  calendar transport (`internal/otsclient`, with BOTH `safeUpgrade` + `safeStamp` guards) are all wired
  via `main.go`'s `runOTSLoop`. The Verify-closer not yet built: a root reaching **Bitcoin-confirmed** —
  needs a live calendar + real BTC confirmation. Still 1/1 open. **Carried `low` defect:** nil-Stamper +
  empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only nil path).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`a9a700e`) == `origin/develop` (run
27957180896); the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate
defect). Latest `review` verdict is PASS (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build toolchain
  `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run check` green
  at HEAD (build + vet + `go test ./...`, all 28 packages ok; `gofmt -l .` empty); the `Type` column is
  mutation-proven non-vacuous (deleting the `.record-type` badge cell from `records.html` AND reverting
  the `schemaDeclaration` constant to the short form each fail `TestRecordsRendersTypeColumn` — the
  constant-vs-constant vacuity trap is avoided via a HARDCODED literal schema URI). The trust-root oracles
  are unaffected — name-only diff over the signature / RFC-6962 / Merkle / `proof` / `didweb` / store
  globs is empty (this is a pure HTML render of a persisted, schema-agnostic `note.$schema`; oracle gate
  correctly N/A).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`a9a700e`) is
  pushed (0 ahead / 0 behind upstream); CI run 27957180896 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27957180974) at HEAD
  is **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions" source);
  deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 5 normal, 10 low** (the lone "critical" grep hit is the format-template
  legend on issues.md:8, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
  Unchanged from last assessment (this iteration filed no new issue and closed none). The 5 normal: the
  two remaining `/` sub-region deltas (config-driven identity + recent-declarers footer), the per-hub-Anchor
  design-honesty question (issue:326), the WASM verifier-scope SIGNATURE-half gap, the Pages
  custom-domain/enablement gap, and the no-on-disk-DB-migration hazard.

## Next Milestone
**CI green and pushed; the record-list `Type` column is closed — the log-browser surface is now exhausted
of pure-code named-region slices. The next cheapest code-only slice shifts surface; after that the backlog
is design-first or human-blocked.** In order:
1. **Render config-driven instance identity + realm name across the SSR mastheads (`internal/config` env
   wiring → the six SSR surfaces)** — issue #214 sub-2, the most-referenced remaining `normal`. The
   mockup shows `monitor.iscc.id` / "instance operated by ISCC Foundation · ISCC mainnet" + a
   "REALM REGISTER · ISCC MAINNET" subtitle; the live pages render generic static copy because the
   identity is not env-configurable. Code-only, unblocks honest per-deployment chrome on ALL SIX mastheads
   (handoff invariant 9), recurs everywhere. `review` flags it as the strongest next candidate. Prefer it
   over re-polishing met surfaces.
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
remaining SSR surfaces (log browser / single record / certificate); the no-on-disk-DB-migration mechanism
(the project's first migration framework); the M-UI exit visual-pass + human sign-off (ADR-0012).
