<!-- assessed-at: 46e690adda83ba526ffd4a04fce4555eec12caca -->

# Project State

## Status: IN_PROGRESS

## Phase: Closing the open `normal` honesty + contract-accuracy backlog while the lone `critical` waits on human M-UI exit sign-off.
This window code-closed the **dossier §1 unresolvable-path honesty** `normal`: on the `unresolvable`
overlay path the §1 Identity line no longer asserts "Key resolved from did:web:<domain>" (which
contradicted the same page's "signing key is unresolved" caution) — it renders neutral "Key source:"
wording instead, gated by a new pure view-model flag `KeyUnresolved` set at the same site as
`ShowCaution`. All feature milestones (M1–M3, M-Deploy, M-API serve/drift/docs) carry forward
unchanged. The lone `critical` (dossier→log-browser navigation + record-list parity) stays fully
code-closed; only the human M-UI exit sign-off remains on it.

This window (`0673ff0..46e690a`, 4 commits): update-state `093ff56` → define-next `6edbc90`
(gate §1 off the unresolvable path) → advance `988d48d` (neutral "Key source:" wording) → review
`46e690a` (PASS / CONTINUE, both mutations reproduced, Codex denied/visual-skipped). Code touched:
`internal/dossier/{handler.go, dossier.html, handler_test.go}` (3 files + context). Branch `develop`,
HEAD `46e690a` level with `origin/develop`; working tree clean; CI + Pages + Publish all `success`.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 4/4 MET. M-Deploy: 0 open.** Carried forward —
    none of their source touched this window.
  - **M-UI: 1 Verify criterion still REOPENED (log-browser record-list named-region parity) —
    `critical`, FULLY code-closed; only the human M-UI exit sign-off remains.** No M-UI named-region
    code work is left on the critical. The dossier §1 honesty `normal` (not a named-region criterion,
    a CLAUDE.md "evergreen comment / honest copy" fix) is CLOSED this window. Still open (all
    `normal`/design- or human-blocked): per-hub-vs-per-checkpoint realm-index Anchor honesty; the
    `/` realm-index hero-footer `normal` is now fully resolved (all 4 sub-items CLOSED) and prunable.
  - **WASM verifier: published half CLOSED.** Still open: cross-origin **signature half** (no
    checkpoint-signature / did:web check), `normal`, design-first.
  - **OTS anchoring: 1/1 open.** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~5 milestone-Verify-or-gate-advancing / ~5 honesty/contract-accuracy.** This
  window closed an open `normal` (dossier §1 honesty) — genuine backlog burn-down, not cosmetic. **No
  drift** — the reviewer correctly kept the loop CONTINUE on real open `normal`s (the dossier §1 fix
  here; M-API contract-accuracy next) rather than re-attempting the human-blocked `critical`. Watch for
  drift only if the loop starts polishing already-closed surfaces; right now every iteration is closing
  a tracked open issue.

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
**Status**: **met (4/4 Verify criteria)** — carried forward. No proofserve / dashboard functional
contract touched this window (CORS, verify-for-me JSON, `/` index, `/<domain>/log/`).
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/openapi.*` routes
  DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **NOT fully met — the `critical` log-browser gap is FULLY code-closed; only the human
M-UI exit sign-off remains.** The full no-JS chain is traversable forward AND back end-to-end and the
record-list browser matches the Log-Browser mockup's named regions. This window code-closed an
honesty `normal` (dossier §1).
- **`critical` part-1 — forward repoint (CLOSED, committed `de9ed3c`):** `dossier.html:573` links
  "Browse the log →" to `/{{.Origin}}/records` (live record-list browser, `serveRecords`).
- **`critical` part-2a — record-list chrome/breadcrumb/head (CLOSED, committed `7ea7fef`):** `records.html`
  carries the shared masthead chrome (instance identity + `verify ↗`), the absolute-site-root
  `← {{.Domain}} dossier` breadcrumb, the eyebrow "Log browser" / hub / "<domain> · N records" head.
- **`critical` part-2 — single-record navigation closure (CLOSED, committed `14d6bc9`):** `record.html`
  carries the chrome identity + `verify ↗`, the `← Log browser` breadcrumb, the no-JS older/newer stepper
  disabled at the ends, and the honesty-gated "Prove this record's inclusion →" / "Back to list" actions.
- **`critical` part-2b — record-list pager parity (CLOSED, committed `b6eee37`):** `records.html`
  renders a `.pager-top` (seamed into the ledger card) + `.pager-bottom`, each a three-slot
  `← newer` / `seq RangeTop – RangeBottom of Total` / `older →` row with disabled-`<span>` ends; the
  off-mockup Status-badge row + dead CSS removed (only the `.ledger[data-status=frozen]` tint retained).
- **§1 honesty — unresolvable-path wording (CLOSED this window, committed `988d48d`):** on the
  `unresolvable` overlay path `dossier.html` §1 renders "Key source:" instead of "Key resolved from"
  (it could not resolve a key), gated by the pure view-model flag `KeyUnresolved` (`handler.go:317`,
  `status == "unresolvable"`) set at the same site as `ShowCaution`; every other status keeps the
  mockup's "Key resolved from" copy. Mutation-proven by `TestDossierUnresolvedKeyWordingHonesty`
  (reviewer reproduced both mutations: revert the template gate → unresolvable assert FAILS;
  over-gate `KeyUnresolved:true` → verified-sibling assert FAILS). Review `46e690a` PASS.
- **§3 honesty (CLOSED, carried):** `ListHubs` §3 `observed_at` subselect is `ORDER BY c.id ASC LIMIT 1`.
- **Other open (carried):** per-hub-vs-per-checkpoint realm-index Anchor honesty (`normal`, design
  question); the **M-UI exit visual-pass + human sign-off** (ADR-0012) not yet executed (the in-loop
  headless `agent-browser` pass ran; the residual aesthetic judgement is a human-only gate). The
  `/` realm-index hero-footer `normal` is fully resolved (4/4 sub-items CLOSED) and prunable.

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
  Two genuine open `normal` doc fixes remain (NOT human-blocked, oracle-N/A — the immediate code-closable
  work): (a) remove the phantom `index` query param from `/{domain}/log/verify` in `openapi.yaml` + the
  JSON twin (the handler never reads it); (b) fix the `/{domain}/log/checkpoint` `200` media type from
  `text/plain` → `application/octet-stream` (the live mux serves octet-stream). Plus two `low` residuals
  (the omitted `/healthz` 503; the Elements-mermaid-from-unpkg substring ban) and the M-API umbrella
  entry (slices 1–3 landed, prunable once slice 4 closes).

## Quality gates
**Status**: **GREEN on HEAD (`46e690a`): CI + Pages + Publish all `success`.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Code change this window is a pure SSR-template + view-model flag (`KeyUnresolved`); no
  crypto/Merkle/did:web-resolution/proof path — oracle/conformance gate N/A for it. `gofmt -l .` reported
  empty by the review handoff.
- **CI**: `.github/workflows/` = `ci.yml` + `pages.yml` + `publish.yml`. On HEAD `46e690a` all three
  `success` (`gh run list --branch develop`).
- **Latest `review` verdict: PASS / CONTINUE** (commit `46e690a`, recorded in handoff.md) — `mise run
  check` green across 30 packages, `gofmt` empty, `TestDossierUnresolvedKeyWordingHonesty` mutation-proven
  (reviewer reproduced both mutations). Codex second opinion unavailable (sandbox-denied — treated as a
  note, not NEEDS_WORK). Visual check skipped (pure text-content swap on a failed-resolution path).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 7 normal, ~24 low.** The **1 critical** (dossier→log-browser navigation +
  off-mockup record list) is FULLY code-closed — OPEN only for the human M-UI exit sign-off, not any code
  work. Of the 7 normals: ~3 are stale-but-resolved awaiting a prune (the `/` realm-index hero-footer
  entry — all 4 sub-items closed — and the M-API umbrella — slices 1–3 landed); the genuinely actionable
  open normals are the 2 M-API contract-accuracy doc fixes (NOT human-blocked), the WASM signature half,
  and the realm-index Anchor honesty (the last two design/human-blocked). DONE requires 0 critical AND
  0 normal.

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
3. **Prune the stale-but-resolved `normal`s** (next `update-state`/`review`): the fully-closed `/`
   realm-index hero-footer entry + the resolved M-API umbrella, once the two doc fixes land.
4. **Design/human-blocked `normal`s:** the WASM cross-origin signature half; the realm-index per-hub-vs-
   per-checkpoint Anchor honesty. The OTS Bitcoin-confirmed half remains offline-unprovable.
