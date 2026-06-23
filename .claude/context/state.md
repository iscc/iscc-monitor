<!-- assessed-at: 0673ff0e0fe607d6c39fd4a7c0dfc2e3b417071d -->

# Project State

## Status: IN_PROGRESS

## Phase: The lone `critical` log-browser gap is now FULLY code-closed — every code-closable half (navigation closure + record-list named-region + pager parity) has landed and is reviewer-verified. Only the human M-UI exit sign-off remains on that critical; the loop continues on the open `normal` M-API contract-accuracy fixes.
This window closed **part-2b** — the record-list pager parity. `records.html` now renders a top
pager seamed into the ledger card plus a bottom pager, each a three-slot `← newer` /
`seq RangeTop – RangeBottom of Total` / `older →` row with disabled-`<span>` ends, and the
off-mockup Status-badge row + its dead CSS are gone (the overlaid status reaches only the
`.ledger[data-status=frozen]` tint). With this slice the dossier→record-list→single-record→cert/back
no-JS chain matches the Log-Browser mockup's named regions end-to-end, forward AND back.

This window (`c73c5b7..0673ff0`, 4 commits): update-state `42741b9` → define-next `85c6d28`
(part-2b pager parity) → advance `b6eee37` (top+bottom pager + drop Status row) → review `0673ff0`
(PASS / CONTINUE, mutation-proven, Codex-clean, agent-browser visual-verified). Code touched:
`internal/proofserve/{handler.go, records.html, records_test.go}` (+ context files). All other
milestones carry forward unchanged. Branch `develop`, HEAD `0673ff0` level with `origin/develop`;
working tree clean.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 4/4 MET. M-Deploy: 0 open.** Carried forward —
    none of their source touched this window.
  - **M-UI: 1 Verify criterion still REOPENED (log-browser record-list named-region parity) —
    `critical`, now FULLY code-closed; only the human M-UI exit sign-off remains.** Closed this window:
    the **record-list pager parity (part-2b)** — `records.html:394` (`.pager-top`) + `:438`
    (`.pager-bottom`) each render the three-slot `← newer` / `seq {{.RangeTop}} &ndash; {{.RangeBottom}}
    of {{.Total}}` (`:400`) / `older →` row with disabled-`<span>` ends; the off-mockup Status-badge row
    + its dead CSS are removed (`grep -c ledger-status/hubStatusBadge/row-label records.html` → 0), with
    only the `.ledger[data-status=frozen]` tint overlay retained (`:409`). Mutation-proven by
    `TestRecordsPagerRangeAndTopBottom` / `TestRecordsPagerEndsDisabled` (+ the dropped-Status assert in
    `TestRecordsRendersInMemoryStatus`); review `0673ff0` PASS, reviewer re-mutated (swap RangeTop/Bottom,
    drop the top pager, re-inject the Status row → each FAILS), each reverted → PASS; an `agent-browser`
    visual pass vs the Log-Browser mockup confirmed parity. **REMAINING (human-only, NOT a code slice):**
    the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012). Also still open (all
    `normal`/design- or human-blocked): §1 "resolved"-vs-unresolvable wording; per-hub-vs-per-checkpoint
    realm-index Anchor honesty.
  - **WASM verifier: published half CLOSED.** Still open: cross-origin **signature half** (no
    checkpoint-signature / did:web check), `normal`, design-first.
  - **OTS anchoring: 1/1 open.** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing / ~4 honesty/data-model.** This window
  closed the LAST code-closable half of the `critical` (part-2b pager parity). **No drift** — every
  iteration this window worked the human-found `critical` toward closure. The `critical` is now fully
  code-closed; what remains there is the human exit sign-off ONLY. The reviewer correctly kept the loop
  CONTINUE (not spinning on cosmetics) by steering `define-next` to the real open `normal` M-API
  contract-accuracy fixes, which are NOT human-blocked.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 path (config, registry, follower, didweb, metrics, proof)
touched this window. `origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze +
alert-once + restart survival; structured logs; `/metrics`.
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
**Status**: **met (4/4 Verify criteria)** — carried forward. The proofserve `serveRecords` view-model
gained pager-range fields (`RangeTop/RangeBottom`), but the M3 functional contract (CORS, verify-for-me
JSON, `/` index, `/<domain>/log/`) is unchanged.
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/openapi.*` routes
  DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **NOT fully met — the `critical` log-browser gap is now FULLY code-closed; only the human
M-UI exit sign-off remains.** The full no-JS chain is traversable forward AND back end-to-end and the
record-list browser now matches the Log-Browser mockup's named regions.
- **`critical` part-1 — forward repoint (CLOSED, committed `de9ed3c`):** `dossier.html:573` links
  "Browse the log →" to `/{{.Origin}}/records` (live record-list browser, `serveRecords`).
- **`critical` part-2a — record-list chrome/breadcrumb/head (CLOSED, committed `7ea7fef`):** `records.html`
  carries the shared masthead chrome (instance identity + `verify ↗`), the absolute-site-root
  `← {{.Domain}} dossier` breadcrumb, the eyebrow "Log browser" / hub / "<domain> · N records" head.
- **`critical` part-2 — single-record navigation closure (CLOSED, committed `14d6bc9`):** `record.html`
  carries the chrome identity + `verify ↗`, the `← Log browser` breadcrumb, the no-JS older/newer stepper
  disabled at the ends, and the honesty-gated "Prove this record's inclusion →" / "Back to list" actions.
- **`critical` part-2b — record-list pager parity (CLOSED this window, committed `b6eee37`):**
  `records.html` renders a `.pager-top` (seamed into the ledger card, `:394`) + a `.pager-bottom`
  (`:438`), each a three-slot `← newer` / `seq {{.RangeTop}} &ndash; {{.RangeBottom}} of {{.Total}}`
  (`:400`) / `older →` row with disabled-`<span>` ends; the off-mockup Status-badge row + its dead CSS
  are removed (only the `.ledger[data-status=frozen]` tint overlay retained, `:409`). `serveRecords`
  threads the pure-derived `RangeTop/RangeBottom` (`handler.go:862-863, :991-992`). Mutation-proven by
  `TestRecordsPagerRangeAndTopBottom` / `TestRecordsPagerEndsDisabled` + the dropped-Status assert in
  `TestRecordsRendersInMemoryStatus`; review `0673ff0` PASS, reviewer re-mutated each new assert → FAIL.
  An `agent-browser` visual pass confirmed parity (top+bottom pager + range label + no Status row; only
  constraint-win residuals: plain-link pager vs mockup buttons, no JS "Jump to sequence" input).
- **§3 honesty (CLOSED, carried):** `ListHubs` §3 `observed_at` subselect is `ORDER BY c.id ASC LIMIT 1`,
  mutation-proven.
- **Other open (carried, NOT critical):** §1 "resolved"-vs-unresolvable wording (`normal`, design-rooted);
  per-hub-vs-per-checkpoint realm-index Anchor honesty (`normal`, design question); the **M-UI exit
  visual-pass + human sign-off** (ADR-0012) not yet executed (the in-loop headless pass uses
  `agent-browser`'s bundled browser; the residual aesthetic judgement is a human-only gate). The `/`
  realm-index sub-items are all CLOSED (prunable).

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
- **Slice 4 (contract accuracy):** **OPEN as a bookkeeping/contract-fidelity gap, not a milestone block.**
  The latest `review` (handoff) steered `define-next` to two genuine open `normal` doc fixes: (a) remove
  the phantom `index` query param from `/{domain}/log/verify` in `openapi.yaml` + the JSON twin (the
  handler never reads it); (b) fix the `/{domain}/log/checkpoint` `200` media type from `text/plain` →
  `application/octet-stream`. Both are small, oracle-N/A doc fixes — the immediate non-human-blocked work.

## Quality gates
**Status**: **GREEN on HEAD (`0673ff0`): CI + Pages + Publish all `success`.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Code changes this window are pure SSR-template + view-model (pager range fields, dropped Status row);
  no crypto/Merkle/did:web path — oracle/conformance gate N/A for them. `gofmt -l internal/proofserve/`
  empty.
- **CI**: `.github/workflows/` = `ci.yml` + `pages.yml` + `publish.yml`. On HEAD `0673ff0` all three
  `success` (`gh run list --branch develop`).
- **Latest `review` verdict: PASS / CONTINUE** (commit `0673ff0`, recorded in handoff.md) — part-2b
  gate-green (30 pkgs), `gofmt` empty, new pager tests mutation-proven (reviewer re-mutated each → FAIL),
  Codex clean, `agent-browser` visual pass confirmed parity. One minor stale doc-comment fixed in-review.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 7 normal, ~21 low.** The **1 critical** (dossier→log-browser navigation +
  off-mockup record list) is now FULLY code-closed (part-1 + 2a + 2b + single-record back-leg all closed)
  — it stays OPEN only for the human M-UI exit sign-off, not any code work. Of the 7 normals: ~4 are
  stale-but-resolved (the realm-index hero entry — all 4 sub-items closed — plus 2 M-API
  contract-accuracy entries now flagged as the immediate work + the M-API umbrella) awaiting a prune; the
  genuinely actionable open normals are the 2 M-API contract-accuracy doc fixes (NOT human-blocked), the
  WASM signature half, the realm-index Anchor honesty, and the §1 wording (the last three design/human-
  blocked). DONE requires 0 critical AND 0 normal.

## Next Milestone
**The lone `critical` is fully code-closed and human-blocked — do NOT re-attempt it in code. Steer
`define-next` to the open `normal` M-API contract-accuracy fixes (NOT human-blocked, genuine progress),
then request the human M-UI exit sign-off.**
1. **M-API contract-accuracy doc fixes (immediate, non-human-blocked):** (a) remove the phantom `index`
   query param from `/{domain}/log/verify` in `openapi.yaml` + the JSON twin (the handler never reads it);
   (b) fix the `/{domain}/log/checkpoint` `200` media type `text/plain` → `application/octet-stream`. One
   slice or two; oracle-N/A; closes the M-API contract-fidelity criterion + prunes the stale normals.
2. **Request the human M-UI exit visual-pass + sign-off** (ADR-0012) — the only remaining gate on the
   lone `critical`; every code-closable half has landed and is reviewer-verified.
3. **Prune the stale-but-resolved `normal`s** (next `update-state`/`review`): the fully-closed realm-index
   hero entry + the resolved M-API umbrella, once the two doc fixes land.
4. **Design/human-blocked `normal`s:** the WASM cross-origin signature half; the realm-index per-hub-vs-
   per-checkpoint Anchor honesty; the §1 "resolved" wording. The OTS Bitcoin-confirmed half remains
   offline-unprovable.
