<!-- assessed-at: 06c77494d28436167fae66b648edff7a331d576c -->

# Project State

## Status: IN_PROGRESS

## Phase: Hub-Dossier critical rework — increment 2a landed and review-verified. The §5 honest
observation log (size-transition / anchor-confirmation / freeze-pointer lines, derived from the new
`store.ListCheckpoints` leaf, with NO synthesized per-poll "consistent" verdict) now SERVES at
`GET /<domain>`. The dossier critical was re-scoped: increment 2 → 2a (§5 log, DONE) + 2b (richer
frozen Exhibit + the §3/§1/§5 honesty `normal`s, the lone remaining `critical`). DONE stays blocked.

This window (`c45fe48..06c7749`, 4 commits: update-state `a504f65` → define-next `626df2f` → advance
`96e9600` → review `06c7749`) is a clean single in-loop increment. The diff touched ONLY
`internal/dossier/{dossier.html,handler.go,handler_test.go}` + `internal/store/checkpoints{.go,_test.go}`
(and context files). No M1/M2/M3/WASM/OTS/M-Deploy source was touched — those sections carry forward
verified. HEAD `06c7749` is level with `origin/develop`; CI `success`, Pages `success`, Publish
`in_progress` (the push-to-develop run was still running at assessment — not a failure).

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no M1–M3 trust-path
    source touched this window.
  - **M-UI: behavioral + named-region Verify met on five surfaces; the hub-dossier surface is now
    further closed — increment 1 (§1–§4 doc) + increment 2a (§5 observation log) DONE, increment 2b
    (richer frozen Exhibit + the §3/§1/§5 honesty nits) is the lone remaining `critical`.** Verified in
    the served template: the `§5 · OBSERVATION LOG` region renders recorded events only (transition /
    anchor / freeze lines), honest empty state for ≤1 checkpoint. The mandatory M-UI exit visual-pass +
    human sign-off (ADR-0012) still pending; instance identity config-driven on THREE of six SSR
    mastheads (proofserve trio still placeholder).
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** No open Verify item remains.
- **Last ~10 iterations: this window was 1 milestone-Verify-advancing increment (dossier §5, the 2a
  half of the lone `critical`). Over the broader window ~7 milestone-Verify-or-gate-advancing / ~3
  context-prune.** No drift: the loop has concrete code-closable critical work (the dossier 2b rework)
  and is steadily peeling it off (increment 1 → 2a → 2b). The new §5 same-size pseudo-transition
  `normal` filed this window is a natural co-resident of 2b, not new scope creep.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `c45fe48..06c7749` diff touched NO M1 source (config, registry,
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
window. The new `store.ListCheckpoints` (added in 2a for the §5 log — additive, newest-first leaf read over
`checkpoints` joining `ots` for confirmations) is a read-only leaf; the M2 mirror/fsck contract is unchanged.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged. The
dossier 2a work extends the existing `GET /<domain>` surface (an M-UI surface) without touching the
verify-for-me / CORS / log-browser / realm-index Verify bars.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on five surfaces; the hub dossier is now FURTHER met —
increment 1 (§1–§4 doc) + increment 2a (§5 observation log) LANDED + review-verified; increment 2b (richer
frozen Exhibit + the §3/§1/§5 honesty nits) is the lone remaining `critical`.** Verified by reading the
served template (`internal/dossier/dossier.html`), the 17 dossier handler tests + the new `TestListCheckpoints`
pair in `internal/store/checkpoints_test.go`:
- **Hub dossier — increments 1 + 2a MET, increment 2b OPEN (`critical`).** `dossier.html` renders the
  numbered trust document: the §1–§4 grid (honesty-gated anchor), and now the `§5 · OBSERVATION LOG` region
  (`dossier.html:530`) listing recorded events only — size-transition lines (newest-first), an
  "anchored · block N" line for a confirmed anchor with a real height, and a "froze hub (split view)"
  freeze pointer — with an honest empty state for ≤1 checkpoint and NO synthesized per-poll "consistent"
  line (the recurring SSR-honesty trap, tested-against). Derived from the new `store.ListCheckpoints`
  (`checkpoints.go:356`, `ORDER BY observed_at DESC`, hub-scoped, NULL-safe), wired in `handler.go:230`.
  **Increment 2b (`critical`, NOW PICKABLE)** fills the richer frozen Exhibit ("size before → presented" +
  a stable evidence ref derived from `Violation.RawA`/`RawB`, reusing the existing checkpoint parser) and
  folds in three honesty `normal`s: §3 frozen size/time decouple, §1 "resolved"-vs-unresolvable, and the
  §5 same-size/shrink pseudo-transition (`size N → N` on a fork — filed this window by 2a's review).
- **Other surfaces met + carried:** `/` realm index (with a "Recently declared" hero row), log browser,
  single record, certificate, badge, DS shell, proof-bundle all render and pass the behavioral +
  named-region HTTP-seam Verify.
- **Still open (carried, NOT critical):** instance identity config-driven on THREE of six mastheads
  (proofserve trio still placeholder); the M-UI exit visual-pass + human sign-off (ADR-0012) not executed
  across all surfaces (review ran a per-increment visual pass on the §5 region and filed no delta); the
  per-hub-vs-per-checkpoint realm-index Anchor honesty question (`normal`).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`
  (`TestWasmVerifyHashPinned` green), publicly served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm`
  (Pages run on HEAD `06c7749` green). **Still open:** the cross-origin **SIGNATURE-half gap** (no
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
  window touched Go source (`internal/dossier`, `internal/store/checkpoints`). `gofmt -l .` is **empty
  (verified clean)** outside the gitignored `cauldron/`.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` + `publish.yml`. **HEAD `06c7749` (level with `origin/develop`): CI `success`,
  Pages `success`, Publish `in_progress`** (the push-to-develop Publish run was still running at
  assessment — not a failure; confirmed via `gh run list`).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `06c7749`) — covers the dossier §5 (2a)
  advance: gate-green (all 28 pkgs `ok`, `gofmt` empty), mutation-proven (drop the size-transition append
  FAILS; `ORDER BY … DESC → ASC` FAILS both store + dossier), with a live `agent-browser` visual pass
  confirming the §5 region matches the document style (no visual delta filed). One Codex-confirmed honesty
  nit filed `normal` (§5 `size N → N` pseudo-transition on a fork/shrink frozen hub), folded into 2b;
  does not block.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 8 normal, 17 low** (the third `## … critical` match is the format-example
  block, not a real issue). The lone `critical` is the Hub-Dossier increment-2b rework. The `normal` count
  rose 7 → 8: the new §5 fork pseudo-transition nit filed this window. DONE requires 0 critical AND 0 normal.

## Next Milestone
**Close the lone remaining `critical` — Hub Dossier increment 2b — the immediate, code-closable DONE
blocker.** Now that §5 (2a) serves, finish the frozen-evidence detail + honesty nits:
1. **Richer frozen Exhibit:** reuse the existing unexported checkpoint/note parser to read the two
   contradictory checkpoints' tree sizes from `Violation.RawA`/`RawB` → render "size before → presented" +
   a stable, honest evidence ref (e.g. a short hash over the pair, never a fabricated id), keeping the
   kind + detected-at + "do not trust new state".
2. **§5 fork/shrink honesty:** skip non-increasing (equal/shrunk) consecutive pairs in the transition loop
   (`handler.go:430-433`) so a fork (same `tree_size`, different root) or a shrink contradictory checkpoint
   never renders a `size N → N` / `larger → smaller` pseudo-transition; only emit the oldest singleton when
   at least one REAL (increasing) transition was emitted.
3. **Fold in the §3/§1 honesty `normal`s** while reworking §3: select `observed_at` for the `f.last_size`
   row (so size + time describe one checkpoint); neutral §1 source wording or gate "resolved" off the
   `unresolvable` overlay (a design call).

After the critical: the 8 `normal`s remain (DB migration + the `iscc_index.seq` multi-hub PK collision —
paired design decisions; the WASM signature-half gap; the per-hub-Anchor honesty question; the three
dossier edge-state honesty nits folded into 2b; and the `/` "recent declarers" entry — all four of its
sub-items are CLOSED, so `update-state`/`review` may prune it). The OTS Bitcoin-confirmed half + the M-UI
exit visual-pass across all surfaces remain offline/human-blocked.
