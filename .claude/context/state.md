<!-- assessed-at: c45fe48677fe0cca60b24bf8e1cc2a8e95b71a9c -->

# Project State

## Status: IN_PROGRESS

## Phase: Hub-Dossier critical rework — increment 1 of 2 landed and review-verified. The numbered
trust-document layout (§1 Identity · §2 Coverage · §3 Latest checkpoint · §4 Bitcoin anchor + §5
placeholder + two action links) now SERVES at `GET /<domain>`, closing the first of the two `critical`
dossier increments. One `critical` remains (increment 2 — the §5 honest observation log + richer frozen
Exhibit). DONE stays blocked.

This window (`d2f259e..c45fe48`, 4 commits: update-state `afed6c2` → define-next `e6a380f` → advance
`a15a213` → review `c45fe48`) is a clean single in-loop increment. The diff touched ONLY
`internal/dossier/{dossier.html,handler.go,handler_test.go}` + `internal/store/hubs{.go,_test.go}` (and
context files). No M1/M2/M3/WASM/OTS/M-Deploy source was touched — those sections carry forward verified.
HEAD `c45fe48` is level with `origin/develop`; CI `success`, Pages `success`, Publish `in_progress` (the
push-to-develop run was still running at assessment).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no M1–M3 trust-path
    source touched this window.
  - **M-UI: behavioral + named-region Verify met on five surfaces; the hub-dossier surface is now
    PARTIALLY closed — increment 1 (numbered §1–§4 doc) DONE, increment 2 (§5 observation log + richer
    frozen Exhibit) is the lone remaining `critical`.** Verified in the served template: §1–§4 grid, §5
    honest placeholder, the two action links all render. The mandatory M-UI exit visual-pass + human
    sign-off (ADR-0012) still pending; instance identity config-driven on THREE of six SSR mastheads
    (proofserve trio still placeholder).
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** No open Verify item remains.
- **Last ~10 iterations: this window was 1 milestone-Verify-closing increment (dossier increment 1, a
  `critical` closer). Over the broader window ~7 milestone-Verify-or-gate-advancing / ~3 context-prune.**
  No drift: the loop has concrete code-closable critical work (the steer re-scoped the dossier into two
  criticals) and just closed the first. The next `update-state` should re-check whether the `/`
  "recent declarers checked" `normal` is prunable — the out-of-loop "Recently declared" row may already
  close all four of its sub-items.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `d2f259e..c45fe48` diff touched NO M1 source (config, registry,
follower, didweb, metrics all untouched). All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (24 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, verifier,
  version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest path touched this
window. `store.RecentRecords` (added in the prior out-of-loop UI window for the dashboard "Recently
declared" row — additive, schema-agnostic, accepted-tree-bounded) remains a leaf read; the M2 mirror/fsck
contract is unchanged.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged. The
dossier increment-1 work extends the existing `GET /<domain>` surface (an M-UI surface) without touching
the verify-for-me / CORS / log-browser / realm-index Verify bars.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on five surfaces; the hub dossier is now PARTIALLY met
— increment 1 (numbered §1–§4 trust document) LANDED + review-verified; increment 2 (§5 observation log +
richer frozen Exhibit) is the lone remaining `critical`.** Verified by reading the served template
(`internal/dossier/dossier.html`) and the 14 dossier handler tests + 2 new store tests:
- **Hub dossier — increment 1 MET, increment 2 OPEN (`critical`).** `dossier.html` now renders the
  numbered trust document: a 2×2 §1 Identity / §2 Coverage / §3 Latest checkpoint / §4 Bitcoin anchor grid
  (honesty-gated — confirmed-anchor fixture renders the block height; pending/absent renders honest
  "pending"/"not anchored", no fabricated `block 0`), a §5 honest minimal placeholder, the two action
  links (`Prove an ISCC-ID in this hub →`, `Browse the log →`), the `fork → "split view"` vocabulary map,
  with the frozen Exhibit + masthead chrome intact. Store gained NULL-safe `CheckpointObserved` (§3 time)
  + `AnchorHeight` (§4 height, scoped to `OTSStatusConfirmed`) subselects on `HubSummary`, mirroring the
  existing `Anchor` pattern (store stays a leaf). **Increment 2 (`critical`, NOW PICKABLE)** fills §5 with
  the honest observation log (recorded events only — size transitions, freezes, anchor confirmations,
  never a synthesized per-poll "consistent" line) via a new `ListCheckpoints`-style leaf read, plus the
  richer frozen Exhibit ("size before → presented" + a stable evidence ref derived from
  `Violation.RawA`/`RawB`). The two §3/§1 honesty `normal`s (frozen size/time decouple; §1
  "resolved"-vs-unresolvable) are folded into increment 2's §3 rework.
- **Other surfaces met + carried:** `/` realm index (with a "Recently declared" hero row), log browser,
  single record, certificate, badge, DS shell, proof-bundle all render and pass the behavioral +
  named-region HTTP-seam Verify.
- **Still open (carried, NOT critical):** instance identity config-driven on THREE of six mastheads
  (proofserve trio still placeholder); the M-UI exit visual-pass + human sign-off (ADR-0012) not executed
  across all surfaces (review ran a per-increment visual pass on the dossier and filed no delta); the
  per-hub-vs-per-checkpoint realm-index Anchor honesty question (`normal`).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`
  (`TestWasmVerifyHashPinned` green), publicly served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm`
  (Pages run on HEAD `c45fe48` green). **Still open:** the cross-origin **SIGNATURE-half gap** (no
  checkpoint-signature / did:web check; the success copy overstates an unrun key check) — `normal`,
  design-first.
- **OTS:** `.ots` route, §5 anchor clause (`ots.ConfirmedFor`), store layer, off-path stamp/upgrade loop,
  offline classifier, and both calendar-transport guards are wired. Not-yet-built: a root reaching
  Bitcoin-confirmed (needs a live calendar + real BTC confirmation). 1/1 open. **Carried `low` defect:**
  nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED.** No open Verify item remains. Carried forward — no
M-Deploy source touched this window. Verified previously: SIGTERM trap, version-stamped binary +
`/version`, production `Dockerfile` + CI `/healthz` smoke, GHCR `publish.yml` (`:develop` +
`:sha-<short>`), canonical `deploy/realm-testnet.txt` (golden-accepted), `deploy/OPERATING.md`, root
`README.md`. The iscc-infra ops residual (GHCR-package visibility) stays demoted to `low` (human/infra
repo-settings work, out of loop scope).
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT EXISTS`
  only, no `PRAGMA user_version` — design decision, ADR-0007); the **`iscc_index.seq` single-global-PK
  multi-hub collision** (`RecordProjections`' `ON CONFLICT(seq)` clobbers when two hubs share a leaf
  index; pairs with the migration `normal`).
- **Carried `low`:** `docker/login-action@v3` + `docker/build-push-action@v6` still Node-20;
  `.dockerignore` slashless globs; §Footprint qualitative disk-growth answer; `cmd/verifier-site`
  non-atomic write; `schemaDeclaration/Deletion` URI triplication; the masthead-identity fallback consts
  3x; the dossier overlay-precedence 3x duplication.

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched Go source (`internal/dossier`, `internal/store/hubs`). `gofmt -l .` is **empty (verified
  clean)** outside the gitignored `cauldron/`.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` + `publish.yml`. **HEAD `c45fe48` (level with `origin/develop`): CI `success`,
  Pages `success`, Publish `in_progress`** (the push-to-develop Publish run was still running at
  assessment — not a failure; confirmed via `gh run list`).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `c45fe48`) — covers the dossier
  increment-1 advance: gate-green (all 28 pkgs `ok`, `gofmt` empty), mutation-proven, with a live
  `agent-browser` visual pass confirming full design-parity with the Hub-Dossier mockup (no visual delta
  filed). Two Codex-confirmed edge-state honesty nits filed `normal` (frozen §3 size/time decouple; §1
  resolved-vs-unresolvable), both folded into increment 2; neither blocks.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 7 normal, 17 low** (awk-verified on the anchored `Priority:` lines, excluding
  the format-example block). The lone `critical` is the Hub-Dossier increment-2 rework. DONE requires
  0 critical AND 0 normal.

## Next Milestone
**Close the lone remaining `critical` — Hub Dossier increment 2 — the immediate, code-closable DONE
blocker.** Fill §5 + the Exhibit detail into the structure increment 1 created:
1. **§5 honest observation log:** add a `ListCheckpoints(hubID, n)`-style leaf read (newest-first over
   `checkpoints`, carrying `tree_size` + `observed_at`, join `ots` for confirmations) and derive the log
   lines in the dossier view layer — rendering ONLY recorded events (size transitions, freezes from
   `violations`, anchor confirmations from `ots`), NEVER a synthesized per-poll "consistent" line (the
   recurring SSR-honesty trap).
2. **Richer frozen Exhibit:** reuse the existing checkpoint/note parser to read the two contradictory
   checkpoints' tree sizes from `Violation.RawA`/`RawB` → render "size before → presented" + a stable,
   honest evidence ref (e.g. a short hash over the pair, never a fabricated id), keeping the kind +
   detected-at + "do not trust new state".
3. **Fold in the two §3/§1 honesty `normal`s** while reworking §3 (select `observed_at` for the
   `f.last_size` row; neutral §1 source wording or gate "resolved" off `unresolvable`).

After the critical: the 7 `normal`s remain (DB migration + the `iscc_index.seq` multi-hub PK collision —
paired design decisions; the WASM signature-half gap; the per-hub-Anchor honesty question; the two dossier
edge-state honesty nits folded into increment 2; and re-confirm the `/` "recent declarers" `normal` — the
out-of-loop "Recently declared" row may already close all four of its sub-items, so `update-state` should
re-check whether that entry is now prunable). The OTS Bitcoin-confirmed half + the M-UI exit visual-pass
across all surfaces remain offline/human-blocked.
