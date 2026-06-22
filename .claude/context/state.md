<!-- assessed-at: d9dd9eed3686284078cefb11fd5bd4cbc58ecf02 -->

# Project State

## Status: IN_PROGRESS

## Phase: `/` realm-index Checkpoint/Anchor ledger columns landed (sub-delta 3 of issue:214 closed); gate green + pushed.
The realm-index now renders the mockup's six-column ledger (`# | Hub · domain | Coverage since |
Checkpoint | Anchor | Status`): `store.HubSummary`/`ListHubs` gained a read-only `Anchor` projection
(latest-stamped-root OTS status via a correlated `ots` subselect), the dashboard view-model gained
`Checkpoint`/`Anchor`/`AnchorDot`, and `dashboard.html` renders the two new columns grayscale-safe.
Latest `review` verdict is **PASS_WITH_NOTES (loop CONTINUE)**, both mutations reproduced + visual-pass
confirmed + Codex clean. DONE not reached: WASM "published" + signature halves and OTS Bitcoin-confirmed
stay open, and 5 `normal` issues stand (net 4→5 — one closed, one new design-honesty `normal` filed).

Incremental review against assessed-at `e0cc5a3`. The `e0cc5a3..HEAD` diff touched exactly TWO
production Go files (`internal/store/hubs.go` — add `Anchor` projection + subselect; `internal/dashboard/
handler.go` — `anchorLabel` + 3 view-model fields) plus one template (`dashboard.html`) and two tests
(`hubs_test.go`, `handler_test.go`); the rest is `.claude/context/*` + learnings docs. All M1/M2/M3/
WASM/OTS production source outside that store-leaf projection + `/` render is byte-unchanged — those
sections carry forward met/open as before. Verified this assessment: store stays a leaf
(`go list -deps ./internal/store | grep '^net/http$'` empty); the name-only diff over the trust-root
globs (`internal/proof/`, `logclient`, `didweb`, `follower`, `cmd/wasm`, `verifier`, `web`, `ots`,
`otsclient`, `certificate`, `dossier`, `proofserve`) is empty. `git status` clean, HEAD == `origin/develop`.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4). M-UI: behavioral + design-chrome
    Verify met; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012) is still pending.**
  - **WASM verifier: 1/1 open.** The id-binding half is closed in source AND artifact (byte-pinned
    `internal/web/verify.wasm` carries the 6-arg id-binding shim; pin `WasmVerifyHash` matches;
    reproducible from `mise run build:wasm`). **Still OPEN on the milestone Verify** ("the verifier
    artifact … [published at its] published value"): the Pages deploy **does not run** — the latest
    Pages run at HEAD (27954709195) is `failure` at `Configure Pages` (Pages not enabled / not set to
    "GitHub Actions" source), built-but-undeployed, a one-time human repo-Settings step. Also still
    open: **NO dossier tier-2 WASM caller** (no WASM island in `internal/dossier`); and the
    **cross-origin verifier-scope SIGNATURE gap** (the WASM core verifies inclusion math + id-binding
    only — no checkpoint-signature / did:web key resolution — a malicious monitor can still render a
    green `verified`; design-first remainder, STOP-candidate).
  - **OTS anchoring: 1/1 open (carried unchanged).** All three observable HTTP halves are closed (`.ots`
    serve route + the §5 anchor render, digest-bound via `ots.ConfirmedFor`) and BOTH calendar-transport
    guards (`safeUpgrade` + `safeStamp`) are in place. What remains: a root that actually transits to
    **Bitcoin-confirmed** — offline-unprovable; exercised only against an injected Upgrader.
- **Last ~10 iterations: ~2 milestone-Verify-advancing / ~8 chrome·plumbing·hardening.** Recent arc:
  certificate Tier-2 no-JS honesty copy → §6 note.timestamp store layer → §6 `· at` render (closed a
  `normal`) → **`/` realm-index Checkpoint/Anchor columns (this iteration, PASS_WITH_NOTES — closed
  sub-delta 3 of issue:214, filed one new design-honesty `normal`).** **DRIFT WATCH (amber, easing):**
  no *milestone-level* Verify criterion has closed in ~16 increments (the last were the WASM id-binding
  slices ~11 iters back), but the loop is legitimately draining the code-closable `normal` backlog (§6
  render last iter, the Checkpoint/Anchor columns this iter) rather than re-polishing met surfaces. The
  front-of-queue WASM "published" half stays **human-blocked, not code-blocked** (a workflow file cannot
  self-enable Pages), so the loop correctly pivots off it. After this iteration the remaining
  code-closable `normal` slices are thin (the `/` config-driven identity sub-delta needs `internal/config`
  env wiring; the "recent declarers" footer needs a store history that does not exist); the rest is
  design-first (WASM signature half, per-hub Anchor honesty, no-migration story) or human-blocked (Pages,
  custom domain). Surface that inflection: the next pickable code-only slice is the log-browser record-list
  `Logged` column (reuses the landed `RecordRow.NoteTimestamp`, no new store read); after that the work
  turns design-first — flag a STOP/design candidate rather than re-polishing met surfaces.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `e0cc5a3..HEAD` diff touched no signature / RFC-6962 /
Merkle / `proof` / `didweb` / `logclient.verify` / `follower` file (the only production edits are the
store-leaf `Anchor` read-projection + the `/` dashboard view-model/template). All M1 Verify criteria
remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested end-to-end with
freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}` (+
  `cmd/wasm/verifyadapter`); 23 internal packages — `badge, certificate, config, corsmw, dashboard,
  didweb, dossier, follower, healthz, index, logclient, metrics, metricshttp, ots, otsclient, proof,
  proofserve, registry, store, tiles, tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`,
  `go 1.26.1` language directive in `go.mod`; build toolchain `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest path touched
this diff. The only store change is an additive read-only `Anchor` subselect on `ListHubs` (a leaf
projection, no write-path or schema impact). fsck root-rebuild on every verified non-frozen poll;
inclusion cross-check conformance-tested over the real verified mirror; `inclusion`, `consistency`,
`entries` all served from the local mirror.

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
This diff landed the `/` realm-index Checkpoint/Anchor ledger columns (the six-column mockup grid
`# | Hub · domain | Coverage since | Checkpoint | Anchor | Status`, the Anchor cell carrying a
grayscale-safe label beside a decorative dot), closing sub-delta (3) of the `/` parity `normal`
(issue:214). All six certificate clauses (§1–§6) + both anchor panels + badge + DS shell + `/` index
+ log browser + hub dossier (with shared chrome masthead + `← Realm index` back-link) + frozen Exhibit
+ record list + single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint
render and pass the behavioral HTTP-seam Verify; every SSR masthead carries the shared chrome
(self-hosted logo + text mark + divider).
- **Still open (NOT critical, carried):** the named-region + `←` back-link parity pass is on the
  dossier but NOT yet on the remaining SSR surfaces (log browser / single record / certificate still
  lack the full instance-identity block + back-link chain on every surface); the `/` remaining
  sub-region deltas — sub-delta (2) config-driven instance identity/realm name (needs `internal/config`
  env wiring) and sub-delta (4) "recent declarers checked" footer (needs a recent-lookup history the
  store does not track) — filed `normal`; the mandatory **M-UI exit visual-pass + human sign-off**
  (ADR-0012) is not executed.
- **Residual `normal` notes:** the `/` Checkpoint/Anchor sub-delta is now CLOSED. NEW this iteration:
  the realm-index Anchor cell is a per-HUB latest-stamped-root indicator NOT tied to the displayed
  Checkpoint (`store/hubs.go` subselect + `dashboard` anchorLabel) — a design-honesty question for the
  M-UI exit (the authoritative per-checkpoint claim lives in certificate §5); filed `normal` (issue:326),
  spec-faithful so non-blocking. Remaining M-UI `normal`s: the two `/` sub-deltas above + this Anchor
  honesty question. The §6 humanization (`2026-02-14 18:40 UTC` vs verbatim RFC-3339) is intentional
  ADR-0008-deferred visual polish, not a filed delta.

## WASM verifier · OTS anchoring
**Status**: **WASM — milestone OPEN (published half human-blocked, signature half design-blocked),
reproducibility critical CLOSED, HEAD is PASS. OTS — §5 render digest-bound + both calendar-transport
guards landed; only a real Bitcoin confirmation remains (offline-unprovable).** Neither the WASM core
nor OTS source was touched by this diff; both sections carry forward.
- **WASM (carried):** the id-binding half of the verifier-scope trust gap is closed IN SOURCE and
  ARTIFACT, and the artifact is reproducible from its documented command: the committed
  `internal/web/verify.wasm` carries the 6-arg/id-binding shim (`verifyadapter.RecordCommitsID`), its
  SHA-256 matches `WasmVerifyHash` (`internal/web/web.go`), and `mise run build:wasm` deterministically
  re-emits those bytes (`TestWasmVerifyHashPinned` green). `cmd/wasm/main.go` (tagged `//go:build js &&
  wasm`) registers `isccVerifyInclusion`; `verifyadapter` exposes `SafeIndex`, `VerifyJSON`,
  `RecordCommitsID`; `cert.html` embeds the tier-2 island; `internal/verifier` is a SINGLE STATIC
  cross-origin artifact; `cmd/verifier-site` is the reproducible build command;
  `.github/workflows/pages.yml` is the PUBLISH workflow; `verifier.Handler` is NOT mounted in
  `cmd/iscc-monitor`. **Still 1/1 OPEN on the milestone Verify** — the deploy is not live (Pages
  `failure` at `Configure Pages` at HEAD, run 27954709195), needs the one-time human repo-Settings step.
  **Carried `normal` defects:** the verifier core does NO checkpoint-signature / did:web check (a
  malicious cross-origin monitor can render green `verified`; success copy overstates a key check that
  never runs) — design-first remainder / STOP-candidate; the Pages custom-domain / Actions-source
  enablement gap. **Carried `low`:** `cmd/verifier-site` `generate` writes non-atomically.
- **OTS:** the `.ots` serve route (`internal/proofserve`), the §5 anchor clause (digest-bound via
  `ots.ConfirmedFor`), the store layer (`internal/store/ots.go`), the off-path stamp/upgrade loop
  (`OTSTick` in `internal/follower/otsloop.go`), the offline classifier (`internal/ots.{Confirmed,
  ConfirmedFor}` sharing one private `classify`), and the calendar transport (`internal/otsclient`, with
  BOTH `safeUpgrade` + `safeStamp` guards) are all wired via `main.go`'s `runOTSLoop`. The new `/`
  Anchor column READS the same `ots` table (latest-stamped row) but does not touch the loop. The
  Verify-closer not yet built: a root reaching **Bitcoin-confirmed** — needs a live calendar + real BTC
  confirmation. Still 1/1 open. **Carried `low` defect:** nil-Stamper + empty-OTSBytes row falls through
  to the Upgrader instead of being left untouched (`otsloop.go:144`; docstring-vs-code mismatch on the
  test-only nil path; production always wires a non-nil Stamper).

## Quality gates
**Status**: **GREEN and PUSHED.** CI `success` at HEAD (`d9dd9ee`) == `origin/develop` (run
27954709225); the Pages PUBLISH workflow is `failure` at `Configure Pages` (human-step, not a code/gate
defect). Latest `review` verdict is PASS_WITH_NOTES (CONTINUE).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; build
  toolchain `mise.toml` `go = "1.26.4"`); `mise run check` runnable. Latest `review` reported `mise run
  check` green at HEAD (build + vet + `go test ./...`, all 28 packages ok; `gofmt -l .` empty); the
  Anchor projection is mutation-proven (reverting the `ots` subselect to a literal `''` fails
  `TestListHubsAnchorStatus`; dropping the `.anchor-cell` data cell fails `TestDashboardRendersEveryHub`).
  The trust-root oracles are unaffected — name-only diff over the signature / RFC-6962 / Merkle /
  `proof` / `didweb` globs is empty; store stays a leaf (no `net/http`).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` + the `cmd/notecheck` oracle on
  push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. HEAD (`d9dd9ee`) is
  pushed (0 ahead / 0 behind upstream); CI run 27954709225 is `success` at HEAD.
- **Pages**: `.github/workflows/pages.yml` runs on push to develop; the latest run (27954709195) at HEAD
  is **`failure`** — fails at `Configure Pages` (Pages not enabled / not set to "GitHub Actions"
  source); deploy skipped. One-time human repo-Settings step (filed `normal`), not a code/gate defect.
- **Open issues: 0 critical, 5 normal, 10 low** (the lone "critical" grep hit is the format-template
  legend on issues.md:9, excluded). DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.
  Tally moved from "0 critical / 4 normal" last assessment to "0 critical / 5 normal" — the `/`
  Checkpoint/Anchor sub-delta CLOSED, but the review FILED one new design-honesty `normal` (per-hub vs
  per-checkpoint Anchor). The 5 normal: the two remaining `/` sub-region deltas (config-driven identity +
  recent-declarers footer), the per-hub-Anchor design-honesty question, the WASM verifier-scope
  SIGNATURE-half gap, the Pages custom-domain/enablement gap, and the no-on-disk-DB-migration hazard.
  (That is 6 named lines but two `/` sub-deltas share one issue entry → 5 distinct `normal` entries.)

## Next Milestone
**CI green and pushed; the `/` Checkpoint/Anchor sub-delta is closed. The remaining code-only slices are
thin — pick the log-browser `Logged` column next; the rest of the backlog is design-first or
human-blocked.** In order:
1. **Wire `RecordRow.NoteTimestamp` into the log-browser record-list `Logged` column
   (`internal/proofserve`).** A separate SSR surface reusing the already-landed store field — no new
   store read, code-only, advances the log-browser named-region parity. The cheapest remaining
   code-closable slice; prefer it over re-polishing met surfaces. (The other `/` sub-deltas are NOT pure
   code: config-driven instance identity needs `internal/config` env wiring; the recent-declarers footer
   needs a recent-lookup history the store does not track.)
2. **Design-first pass on the signature half of the verifier-scope gap** (the WASM milestone's actual
   trust bar) — browser did:web resolution + checkpoint-note signature verify, gating `verified` on
   signature + id-binding + inclusion. `review` recommends a **design pass before building**; a good
   STOP-candidate if the design is unclear. Until it lands, do NOT loosen `verifier.html`'s "hub-signed
   root" success copy. And/or wire the tier-2 WASM caller into the **hub dossier** (no WASM island today).
3. **Unblock + verify the Pages deploy** (one-time human repo-Settings step: Settings → Pages → source
   "GitHub Actions" + custom domain `monitor.iscc.codes` + DNS CNAME), then re-run the workflow and
   confirm a `success` deploy — closes the "published" half of the WASM Verify criterion. A workflow
   file alone cannot self-enable Pages; flag for the human rather than re-polishing it.

Subsequent: the realm-index per-hub-vs-per-checkpoint Anchor honesty design pass; the OTS "upgrades to
Bitcoin-confirmed" half (offline-unprovable); the named-region + `←` back-link parity pass across the
remaining SSR surfaces; the no-on-disk-DB-migration mechanism (a deliberate design step, the project's
first migration framework); the M-UI exit visual-pass + human sign-off (ADR-0012).
