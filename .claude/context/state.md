<!-- assessed-at: c73c5b78de1099878a90bae3402104cf476975b0 -->

# Project State

## Status: IN_PROGRESS

## Phase: Closing the `critical` log-browser navigation/parity gap — the no-JS navigation chain is now UNBROKEN end-to-end; only cosmetic record-list pager parity (part-2b) + the human M-UI exit sign-off remain.
The single-record page (`record.html`) — the last broken back-leg flagged in the prior assessment —
now carries the full navigation set: chrome identity + `verify ↗`, the `← Log browser` breadcrumb,
the no-JS older/newer stepper disabled at the ends, and the honesty-gated "Prove this record's
inclusion →" / "Back to list" actions. Threaded `domain, instance, operator` into `serveRecord` with
pure-derived view-model fields (no new store read). With this slice the no-JS chain
`/` → dossier → record list → single record → (cert / back) is traversable **forward and back**
end-to-end. The `critical` stays OPEN for the cosmetic part-2b pager parity (record list still has a
bottom-only "showing N of M" span + a Status-badge row the mockup omits) and the human M-UI exit
sign-off — neither is a dead end.

This window (`a671e97..c73c5b7`, 4 commits): update-state `c3518a9` → define-next `1208145`
(single-record back-leg) → advance `14d6bc9` (chrome+breadcrumb+stepper+actions) → review `c73c5b7`
(PASS / CONTINUE, mutation re-proven, chain traced unbroken). Code touched:
`internal/proofserve/{handler.go, record.html, record_test.go}` (+ context files). All other
milestones carry forward unchanged. Branch `develop`, HEAD `c73c5b7` level with `origin/develop`;
working tree clean.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 4/4 MET. M-Deploy: 0 open.** Carried forward —
    none of their source touched this window.
  - **M-UI: 1 Verify criterion still REOPENED (log-browser record-list named-region parity) —
    `critical`, navigation-closure clause now MET, only cosmetic parity + human sign-off remain.**
    Closed this window: the **single-record navigation closure** — `record.html` now carries the
    `← Log browser` breadcrumb (`record.html:376`), the older/newer stepper disabled at the ends
    (`:382-392`), the chrome identity + `verify ↗` (`:366-371`), and the honesty-gated
    "Prove this record's inclusion →" / "Back to list" actions (`:445-448`), mutation-proven by
    `TestRecordBreadcrumbAndChromeIdentity` / `TestRecordStepperEnds` / `TestRecordProveInclusionHonestyGate`
    (reviewer re-mutated the honesty gate + stepper guards → both FAIL). The dossier→list→record→back
    chain is now unbroken (every href reviewer-traced). **STILL OPEN (cosmetic, NOT a dead-end):**
    (a) **part-2b pager parity** — `records.html:433` is still a bottom-only "showing N of M" span,
    not the mockup's top+bottom "seq X–Y of Z" pager disabled at the ends; (b) the **Status-badge row**
    the mockup's log browser omits is still present (`records.html:404`); (c) type-badge tint /
    append-only footnote copy deltas. Also still open (all `normal`/design- or human-blocked): §1
    "resolved"-vs-unresolvable wording; per-hub-vs-per-checkpoint realm-index Anchor honesty; the
    mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED.** Still open: cross-origin **signature half** (no
    checkpoint-signature / did:web check), `normal`, design-first.
  - **OTS anchoring: 1/1 open.** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing / ~4 honesty/data-model.** This window
  closed the load-bearing remaining half of the `critical` (the single-record back-leg), turning the
  navigation-closure clause from "broken" to "met". **No drift** — every iteration this window worked
  the human-found `critical` toward closure or removed a recurring process hazard. The arc is genuinely
  converging on the named-region/navigation gap; what's left of the `critical` is cosmetic pager parity
  + the human exit sign-off, not a behavioral defect.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 path (config, registry, follower, didweb, metrics) touched
this window beyond `serveRecord` gaining the masthead-identity threading in `proofserve/handler.go`.
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze + alert-once + restart
survival; structured logs; `/metrics`.
- **Packages**: `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal — `badge, certificate,
  config, corsmw, dashboard, didweb, docs, dossier, follower, healthz, index, logclient, metrics,
  metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, verifier,
  version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / `iscc_index` path touched this
window. The `SQLiteFetcher` / `ProofBuilder` read side is unchanged; the M2 mirror/fsck contract holds.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the proofserve `serveRecord` signature
gained the hub `domain` + `instance`/`operator` masthead threading, but the M3 functional contract
(CORS, verify-for-me JSON, `/` index, `/<domain>/log/`) is unchanged.
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/openapi.*` routes
  DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **NOT fully met — the `critical` log-browser gap's navigation-closure clause is now MET; only
cosmetic record-list parity (part-2b) + the human M-UI exit sign-off remain.** The full no-JS chain is
traversable forward AND back end-to-end.
- **`critical` part-1 — forward repoint (CLOSED, committed `de9ed3c`):** `dossier.html:573` links
  "Browse the log →" to `/{{.Origin}}/records` (live record-list browser, `serveRecords`).
- **`critical` part-2a — record-list chrome/breadcrumb/head (CLOSED, committed `7ea7fef`):** `records.html`
  carries the shared masthead chrome (instance identity + `verify ↗`), the absolute-site-root
  `← {{.Domain}} dossier` breadcrumb, the eyebrow "Log browser" / hub / "<domain> · N records" head.
  Mutation-proven.
- **`critical` part-2 — single-record navigation closure (CLOSED this window, committed `14d6bc9`):**
  `record.html` now carries the chrome identity + `verify ↗` (`:366-371`), the `← Log browser`
  breadcrumb (relative `href="records"`, `:376`), the no-JS older/newer stepper disabled at the ends
  (`:382-392`), and the honesty-gated "Prove this record's inclusion →" (`{{if .ProveInclusionID}}`,
  `:445-446`) / "Back to list" (`:448`) actions. `serveRecord` threads `domain, instance, operator`
  (`handler.go:1103`) with pure-derived `HasOlder/HasNewer/OlderIndex/NewerIndex/ProveInclusionID`
  view-model fields (no new store read). Mutation-proven by the three new `TestRecord*` tests; review
  `c73c5b7` PASS, reviewer re-mutated the honesty gate + stepper guards → both FAIL. The chain
  `/` → dossier → record list → single record → (cert / back) is now traversable forward AND back, no-JS.
- **`critical` part-2b — record-list pager parity (OPEN, cosmetic):** `records.html:433` still renders a
  single bottom-only "showing {{len .Records}} of {{.Total}}" span, NOT the mockup's top+bottom
  "seq X–Y of Z" pager disabled at the ends; the Status-badge row (`records.html:404`) the Log-Browser
  mockup omits is still present; type-badge tint / append-only footnote copy diverge. NOT a dead-end —
  the navigation closure is satisfied; this is the remaining mockup-region parity.
- **§3 honesty (CLOSED, carried):** `ListHubs` §3 `observed_at` subselect is `ORDER BY c.id ASC LIMIT 1`,
  mutation-proven.
- **Other open (carried, NOT critical):** §1 "resolved"-vs-unresolvable wording (`normal`); per-hub-vs-
  per-checkpoint realm-index Anchor honesty (`normal`, design question); the **M-UI exit visual-pass +
  human sign-off** (ADR-0012) not yet executed (the headless pass cannot launch — Chrome/Chromium absent
  in the devcontainer; degrades gracefully per protocol). The `/` realm-index sub-items are CLOSED (prunable).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live, byte-pinned); signature half design-blocked.
OTS — observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`, served
  byte-pinned at `monitor.iscc.codes/_ds/verify.wasm`. **Still open:** the cross-origin **SIGNATURE-half
  gap** (no checkpoint-signature / did:web check; success copy overstates an unrun key check) — `normal`,
  design-first.
- **OTS:** `.ots` route, §4 anchor clause, store layer, off-path stamp/upgrade loop, offline classifier,
  and both calendar-transport guards wired. Not-yet-built: a root reaching Bitcoin-confirmed (needs a live
  calendar + real BTC confirmation). 1/1 open. **Carried `low` defect:** nil-Stamper + empty-OTSBytes row
  falls through to the Upgrader (`otsloop.go:144`; test-only path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED.** Carried forward — no M-Deploy source touched. Verified
previously: SIGTERM trap, version-stamped binary + `/version`, production `Dockerfile` + CI `/healthz`
smoke, GHCR `publish.yml` (`:develop` + `:sha-<short>`), canonical `deploy/realm-testnet.txt`,
`deploy/OPERATING.md`, root `README.md`, on-disk migration mechanism + its first real entry (composite-PK).
- **Carried `low` traps:** out-of-range `user_version` guard runs AFTER `db.Exec(schemaSQL)`; composite-PK
  rebuild dropped `seq`'s standalone ordering path; `docker/login-action@v3` + `build-push-action@v6`
  still Node-20; `.dockerignore` slashless globs; §Footprint qualitative disk-growth; `cmd/verifier-site`
  non-atomic write; `schemaDeclaration/Deletion` URI triplication; masthead-fallback consts 3x; dossier
  overlay 3x. All latent.

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **4/4 Verify MET** — carried forward (no M-API source touched this window).
- **Slice 1 (serve):** OpenAPI 3.1 served byte-verbatim at `GET /openapi.json` + `/openapi.yaml`.
- **Slice 2 (drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts path↔mux alignment.
- **Slice 3 (`/docs` + Stoplight Elements):** `/docs` mounts `<elements-api>` against same-origin
  byte-pinned `/_ds/elements.min.{js,css}`. No CDN body.
- **Slice 4 (contract accuracy):** verify has no phantom `index`; checkpoint 200 is octet-stream — both
  MET in the served doc. **Bookkeeping lag (NOT a code gap):** `issues.md` still carries 2 `normal`
  contract-accuracy entries (the phantom `verify` `index` param + the `checkpoint` media type) plus the
  M-API umbrella `normal` — the served doc reflects the fixes; these await a `review`/`update-state` prune.

## Quality gates
**Status**: **GREEN on HEAD (`c73c5b7`): CI + Pages both `success`; Publish in progress (non-blocking).**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Code changes this window are pure SSR-template + view-model threading (no crypto/Merkle/did:web path);
  oracle/conformance gate N/A for them.
- **CI**: `.github/workflows/` = `ci.yml` + `pages.yml` + `publish.yml`. On HEAD `c73c5b7`:
  **CI `success`, Pages `success`, Publish `in_progress`** (`gh run list`). Publish runs on push-to-develop
  and is not a gate of the code result.
- **Latest `review` verdict: PASS / CONTINUE** (commit `c73c5b7`, recorded in handoff.md) — single-record
  back-leg gate-green (30 pkgs), `gofmt` empty, three new tests mutation-proven (reviewer independently
  re-mutated the honesty gate + stepper guards → both FAIL), import-direction clean, no new module dep.
  Codex denied this iteration (stale verdict, not relied on); visual pass skipped (no Chrome) — both
  degraded gracefully per protocol.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 7 normal, ~22 low.** The **1 critical** is the dossier→log-browser
  navigation + off-mockup record list — its navigation-closure clause is now MET (part-1 + part-2a +
  the single-record back-leg all closed), so it stays OPEN only for part-2b (cosmetic pager parity +
  Status-row drop) and the human M-UI exit sign-off. Of the 7 normals: ~3 are stale-but-resolved
  (the realm-index hero entry — all 4 sub-items closed — plus the 2 M-API contract-accuracy entries
  verified fixed in the served doc + the M-API umbrella) awaiting a prune; the real open normals are the
  WASM signature half, the realm-index Anchor honesty, and the §1 wording. DONE requires 0 critical AND
  0 normal.

## Next Milestone
**Finish closing the `critical` — build part-2b (record-list pager parity) — then request the human M-UI
exit sign-off, so the navigable chain matches the mockup's named regions end-to-end.**
1. **Record-list pager parity (part-2b):** replace `records.html`'s bottom-only "showing N of M" span
   with the mockup's top+bottom "seq X–Y of Z" pager disabled at the ends; drop the Status-badge row the
   Log-Browser mockup omits; reconcile the type-badge tint / append-only footnote copy. Region/golden
   tests so removing a region FAILS. This is the last code-closable half of the `critical`.
2. **Request the human M-UI exit visual-pass + sign-off** (ADR-0012) once part-2b lands — the mandatory
   M-UI exit gate; the headless `agent-browser` pass cannot launch (no Chrome in the devcontainer).
3. **Prune the stale-but-resolved `normal`s** (next `update-state`/`review`): the fully-closed realm-index
   hero entry + the resolved M-API contract-accuracy entries — dropping the real open normals to ~3.
4. **Design/human-blocked `normal`s:** the WASM cross-origin signature half; the realm-index per-hub-vs-
   per-checkpoint Anchor honesty; the §1 "resolved" wording. The OTS Bitcoin-confirmed half remains
   offline-unprovable.
