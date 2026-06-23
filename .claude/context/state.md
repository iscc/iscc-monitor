<!-- assessed-at: d2f259e10c24f35d4b2aafb1eccad656a1ca2dab -->

# Project State

## Status: IN_PROGRESS

## Phase: Hub-Dossier design-parity rework. The prior "human-blocked-DONE edge" is OVER: a `steer`
commit (`d2f259e`) filed **two `critical`** Hub Dossier increments after a human design-parity review,
so there is now concrete, code-closable critical work. The open-issue count is back to **2 critical, 5
normal, 17 low** — DONE is firmly blocked again.

This window (`f8e95ea..d2f259e`, 6 commits) is a mix: the loop's ref-guard increment (define→advance→
review, reviewed PASS_WITH_NOTES), then **two out-of-loop human UI commits** (`f28f57e` logo height;
`0bb1963` coral lookup input + "Recently declared" row + hero copy — which landed `store.RecentRecords`
+ dashboard wiring with tests but carry **no `cid(review)` verdict**), then the `steer` filing the two
dossier criticals. HEAD `d2f259e` is level with `origin/develop`; CI / Pages / Publish all `success`.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no M1–M3 trust-path
    source touched.
  - **M-UI: behavioral + named-region Verify met, BUT the hub-dossier surface is now re-opened as a
    `critical` design-parity gap** (the served dossier is a flat 5-row key/value ledger, verified, not the
    numbered §1–§5 trust document its mockup + the target.md named-region bar require). The mandatory M-UI
    exit visual-pass + human sign-off (ADR-0012) also still pending. Instance identity is config-driven on
    THREE of six SSR mastheads; proofserve trio still placeholder.
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** No open Verify item remains.
- **Last ~10 iterations: ~7 milestone-Verify-or-gate-advancing / ~3 context-prune.** **Drift note,
  inverted from last window:** the human intervened exactly at the human-blocked-DONE edge the memory
  warned about — rather than letting the loop spin on cosmetic chrome, a `steer` re-scoped the hub
  dossier into two concrete `critical` increments. So the loop now HAS clear code-closable critical work
  and is no longer at the stall edge. Watch instead: the two out-of-loop UI commits (`RecentRecords`,
  logo) bypassed the review gate — `review` should confirm they are gate-clean retroactively.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `f8e95ea..d2f259e` diff touched NO M1 source (config, registry,
follower, didweb, metrics all untouched). All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (24 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, verifier,
  version, web`. Module `github.com/iscc/iscc-monitor`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest path touched.
The store gained one realm-wide read (`RecentRecords` in `internal/store/iscc_index.go:229`) for the
dashboard "Recently declared" row — additive, schema-agnostic, accepted-tree-bounded (`seq < f.last_size`),
store stays a leaf (stdlib only). It does not change the M2 mirror/fsck contract.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged. The
dashboard handler gained the "Recently declared" view feature (out-of-loop UI commit `0bb1963`,
`internal/dashboard/handler.go` + tests `TestDashboardRecentlyDeclared` / `TestDashboardNoRecentRowWhenIndexEmpty`),
which is additive to the realm-index page, not a change to the verify-for-me / CORS / log-browser bars.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete EXCEPT the hub dossier, which a human design-parity
review has re-opened as a `critical` two-increment rework.** Verified by reading the served template:
- **Hub dossier — `critical`, NOT met to the named-region bar.** `internal/dossier/dossier.html` renders
  ONE hub as a flat key/value ledger (Coverage row + a single "mirrored log" link) plus the frozen Exhibit
  + masthead chrome — verified present. It does NOT render the mockup's numbered trust document: there is
  no §1 Identity / §2 Coverage / §3 Latest checkpoint / §4 Bitcoin anchor numbered grid, no §5 observation
  log, no "Prove an ISCC-ID / Browse the log" action pair. The two filed criticals fix this:
  - **Increment 1 (`critical`):** numbered-document layout + §1–§4 (additive `store.HubSummary` fields for
    §3 observed-time + §4 btc-height via NULL-safe correlated subselects, mirroring the existing `Anchor`
    subselect); §5 ships as an honest minimal placeholder; `fork → "split view"` vocabulary map at render.
  - **Increment 2 (`critical`, GATED on increment 1):** the §5 honest observation log (only recorded
    events — size transitions, freezes, anchor confirmations — never a synthesized per-poll "consistent"
    verdict, the recurring SSR-honesty trap) + the richer frozen Exhibit ("size before → presented" +
    evidence ref, derived from `Violation.RawA`/`RawB`).
- **Other surfaces met + carried:** `/` realm index (now with a "Recently declared" hero row, out-of-loop),
  log browser, single record, certificate, badge, DS shell, proof-bundle all render and pass the
  behavioral + named-region HTTP-seam Verify.
- **Still open (carried, NOT critical):** instance identity config-driven on THREE of six mastheads
  (proofserve trio still placeholder); the M-UI exit visual-pass + human sign-off (ADR-0012) not executed;
  the per-hub-vs-per-checkpoint realm-index Anchor honesty question (`normal`).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`
  (`TestWasmVerifyHashPinned` green), publicly served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm`
  (Pages run on HEAD `d2f259e` green). **Still open:** the cross-origin **SIGNATURE-half gap** (no
  checkpoint-signature / did:web check; the success copy overstates an unrun key check) — `normal`,
  design-first.
- **OTS:** `.ots` route, §5 anchor clause (`ots.ConfirmedFor`), store layer, off-path stamp/upgrade loop,
  offline classifier, and both calendar-transport guards are wired. Not-yet-built: a root reaching
  Bitcoin-confirmed (needs a live calendar + real BTC confirmation). 1/1 open. **Carried `low` defect:**
  nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED.** No open Verify item remains. Carried forward — no
M-Deploy source touched this window beyond the ref-guard workflow edits (below). Verified previously:
SIGTERM trap, version-stamped binary + `/version`, production `Dockerfile` + CI `/healthz` smoke, GHCR
`publish.yml` (`:develop` + `:sha-<short>`), canonical `deploy/realm-testnet.txt` (golden-accepted),
`deploy/OPERATING.md`, root `README.md`. The prior iscc-infra ops `critical`s stay pruned/demoted to
`low` (the Dockerfile/publish residual is human/infra repo-settings work, out of loop scope).
- **This-window workflow edit (reviewed PASS_WITH_NOTES):** `publish.yml` + `pages.yml` `build` jobs gained
  `if: github.ref == 'refs/heads/develop'` ref-guards on their `workflow_dispatch` path, and the `actions/*`
  pins were bumped off Node-20 to the live latest majors. This closed the ref-guard `normal` and the
  Pages-annotation Node-20 `low`.
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT EXISTS`
  only, no `PRAGMA user_version` — design decision, ADR-0007); the **`iscc_index.seq` single-global-PK
  multi-hub collision** (new this window from the out-of-loop UI work — `RecordProjections`'
  `ON CONFLICT(seq)` clobbers when two hubs share a leaf index; pairs with the migration `normal`).
- **Carried `low`:** `docker/login-action@v3` + `docker/build-push-action@v6` still Node-20 (review
  corrected the false "container actions" claim); `.dockerignore` slashless globs; §Footprint qualitative
  disk-growth answer; `cmd/verifier-site` non-atomic write; `schemaDeclaration/Deletion` URI triplication.

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26`; toolchain `go = "1.26.4"`);
  `mise run check` runnable. This window touched Go source (`internal/dashboard`, `internal/store`,
  workflows, the six SSR templates' `.dc.html` ref bumps). `gofmt -l .` is **empty (verified clean)**.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` + `publish.yml`. **HEAD `d2f259e` (level with `origin/develop`): CI `success`,
  Pages `success`, Publish `success`** (confirmed via `gh run list`).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** — but it covers the **ref-guard** increment
  (`4909dd2`), NOT the two later out-of-loop UI commits (`f28f57e`, `0bb1963`) or the `steer`. The
  `RecentRecords` + dashboard "Recently declared" + logo changes shipped WITHOUT a `cid(review)` verdict;
  they have tests and the gate is green, but `review` has not independently signed off on them.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 2 critical, 5 normal, 17 low** (grep-verified on the anchored `Priority:` lines). The two
  `critical`s are the Hub Dossier rework increments. DONE requires 0 critical AND 0 normal.

## Next Milestone
**Close the two `critical` Hub Dossier increments — this is the immediate, code-closable DONE blocker.**
1. **Increment 1 (`critical`, pick first):** rebuild `internal/dossier/dossier.html` into the numbered
   trust-document layout with §1 Identity / §2 Coverage / §3 Latest checkpoint / §4 Bitcoin anchor, adding
   the NULL-safe `store.HubSummary` §3-observed-time + §4-btc-height correlated subselects (mirror the
   existing `Anchor` subselect; store stays a leaf), the `fork → "split view"` render map, the two action
   links, and §5 as an honest placeholder. HTTP-seam golden + mutation; no fabricated timestamps/anchor
   states (the SSR-verdict honesty trap).
2. **Increment 2 (`critical`, GATED on #1):** the §5 honest observation log (recorded events only — never a
   synthesized per-poll "consistent" line) via a new `ListCheckpoints`-style leaf read, plus the richer
   frozen Exhibit ("size before → presented" + evidence ref derived from `RawA`/`RawB`, reusing the
   existing note parser).

After the criticals: the 5 `normal`s remain (DB migration + the new `iscc_index.seq` multi-hub PK
collision — paired design decisions; the WASM signature-half gap; the per-hub-Anchor honesty question; and
re-confirm the `/` "recent declarers" item — the out-of-loop UI work shipped a "Recently declared" row that
may already close it, so `review`/`update-state` should re-check whether that `normal` is now prunable).
The OTS Bitcoin-confirmed half + the M-UI exit visual-pass remain offline/human-blocked.
