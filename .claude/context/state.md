<!-- assessed-at: 5846e976c36546a5de188b5b7790967a50ecc2f2 -->

# Project State

## Status: IN_PROGRESS

## Phase: Data-model hardening — migration-runner mechanism landed (no-op baseline).
All feature milestones (M1→M3, M-UI named-region, WASM published half, OTS observable halves) plus
both order-independent code-closable milestones (M-Deploy + M-API) are met in-repo. This window added
the project's FIRST on-disk migration mechanism — a `PRAGMA user_version`-gated, fail-closed,
idempotent runner in `store.Open` — but its production migration list ships EMPTY, so it is the
enabling step before the real data-model fix (the `iscc_index` PK rebuild), not a milestone-criterion
close. What remains for DONE is the standing data-model + design/human-blocked `normal`s.

This window (`476743f..5846e97`, 4 commits: update-state `1163019` → define-next `4626397` → advance
`94a5f7a` → review `5846e97`) is a clean single in-loop increment. The diff touched ONLY
`internal/store/{sqlite.go,sqlite_test.go}` + `deploy/OPERATING.md` (§Migration-policy) plus context
files. **No M1/M2/M3/M-UI/WASM/OTS/M-Deploy/M-API milestone-criterion source was touched** — those
sections carry forward verified. HEAD `5846e97` is level with `origin/develop`. Pages `success`; CI +
Publish were still `in_progress` at assessment (same SHA, no failure observed; review recorded gate-green
locally over 30 packages). The working tree carries one uncommitted `target.md` edit (the prior
`cid(steer)` artifact) — review correctly left it for the next steer; not state-assessor's to commit.

## Convergence
- **Remaining Verify criteria (all milestone Verify bars MET; what's left is `normal`/`low` issues, not
  milestone criteria):**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 0 open (4/4). M-Deploy: 0 open.** All carried
    forward — none of their source was touched this window (the diff is store-migration-only).
  - **M-UI: behavioral + named-region Verify met on all five SSR surfaces; no `critical`.** Still open
    (all `normal`/design- or human-blocked): the dossier §3 frozen size/time decouple, the §1
    unconditional "resolved"-vs-unresolvable wording, the per-hub-vs-per-checkpoint realm-index Anchor
    honesty question, instance identity config-driven on only THREE of six SSR mastheads (proofserve trio
    still placeholder), and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~7 milestone-Verify-or-gate-advancing / ~3 data-model-or-prune.** The last
  three windows closed M-API (slices 1→4), and this window landed the migration MECHANISM — a deliberate
  enabling step for the scheduled `iscc_index` PK rebuild, NOT a milestone-criterion close. **No drift:**
  the code-closable feature/milestone backlog is drained, so the loop has correctly pivoted to the
  data-model `normal`s (migration runner now, PK rebuild next) rather than spinning on cosmetic chrome.
  This is the right convergence move per the loop-stalls-on-blocked-DONE memory.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `476743f..5846e97` diff touched NO M1 source (config,
registry, follower, didweb, metrics all untouched). All Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival;
structured logs; `/metrics`.
- **Packages present** (26 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, docs, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest path touched
this window. The store touch was the migration runner (`store.Open`); the `SQLiteFetcher` /
`iscc_index` / `ProofBuilder` contract is unchanged. The M2 mirror/fsck contract holds.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged.
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and the `/openapi.*`
  routes DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on all five SSR surfaces; hub dossier closed at the
code level — no `critical` remains.** Carried forward — no `internal/dashboard|dossier|certificate|web|
proofserve` template or `.html` source touched this window (the diff is store-only).
- **Still open (carried, NOT critical):** the dossier §3 frozen size/time decouple (`normal`); the §1
  "resolved"-vs-unresolvable wording (`normal`); the per-hub-vs-per-checkpoint realm-index Anchor honesty
  question (`normal`); instance identity config-driven on THREE of six mastheads (proofserve trio still
  placeholder — its own `normal` sub-item); the **M-UI exit visual-pass + human sign-off** (ADR-0012) not
  yet executed across all surfaces. The `/` realm-index hero/logo/Checkpoint/Anchor sub-items are all
  CLOSED (that `normal` entry is now prunable).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`, publicly
  served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm` (Pages run on HEAD `5846e97` green). **Still
  open:** the cross-origin **SIGNATURE-half gap** (no checkpoint-signature / did:web check; the success
  copy overstates an unrun key check) — `normal`, design-first.
- **OTS:** `.ots` route, §4 anchor clause (`ots.ConfirmedFor`), store layer, off-path stamp/upgrade loop,
  offline classifier, and both calendar-transport guards are wired. Not-yet-built: a root reaching
  Bitcoin-confirmed (needs a live calendar + real BTC confirmation). 1/1 open. **Carried `low` defect:**
  nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED.** No open Verify item remains. Carried forward — no
M-Deploy source touched this window. Verified previously: SIGTERM trap, version-stamped binary +
`/version`, production `Dockerfile` + CI `/healthz` smoke, GHCR `publish.yml` (`:develop` + `:sha-<short>`),
canonical `deploy/realm-testnet.txt` (golden-accepted), `deploy/OPERATING.md`, root `README.md`.
- **Migration story PROGRESSED this window:** the on-disk migration HAZARD (`store.Open` = baseline
  `CREATE TABLE IF NOT EXISTS` + now a `PRAGMA user_version`-gated runner) is partially addressed — the
  MECHANISM exists (`internal/store/sqlite.go:48,95,119-152`: append-only `migrations []func(*sql.Tx)
  error`, fail-closed per-step `*sql.Tx`, idempotent; 5 mutation-proven `TestMigration*` tests) but the
  production list is **EMPTY** (no-op baseline), so the "recreate the volume on a schema change" interim
  in `OPERATING.md` §Migration-policy still holds until a real migration lands. `OPERATING.md` accurately
  documents the new mechanism + keeps the honest caveat.
- **Carried `normal` traps:** the **`iscc_index.seq` single-global-PK multi-hub collision**
  (`RecordProjections`' `ON CONFLICT(seq)` clobbers when two hubs share a leaf index — the scheduled next
  step, landing as migration index 0); and the NEW (this-window) **out-of-range `user_version`** edge —
  the runner does not bound the read-back version, so a future `user_version > len(migrations)` opens
  silently and a `-1` would panic `migs[-1]` (latent today with the empty slice; both go live the instant
  migration index 0 lands, so fix WITH or BEFORE it).
- **Carried `low`:** `docker/login-action@v3` + `docker/build-push-action@v6` still Node-20;
  `.dockerignore` slashless globs; §Footprint qualitative disk-growth answer; `cmd/verifier-site`
  non-atomic write; `schemaDeclaration/Deletion` URI triplication; the masthead-identity fallback consts
  3x; the dossier overlay-precedence 3x duplication.

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **4/4 Verify MET** — re-verified this window by reading the served `internal/openapi/openapi.yaml`
against the handlers (the doc was NOT touched this window; spot-checked it still matches).
- **Slice 1 (serve):** OpenAPI **3.1** (`openapi.yaml` + JSON twin, `go:embed`-ed) served byte-verbatim at
  `GET /openapi.json` + `/openapi.yaml` (no-cache + strong ETag + 304) under CORS `*`.
- **Slice 2 (drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts path↔mux alignment both ways.
- **Slice 3 (`/docs` + Stoplight Elements):** `/docs` mounts `<elements-api apiDescriptionUrl="/openapi.json">`
  against same-origin byte-pinned `/_ds/elements.min.{js,css}` (SHA-256 pinned). No external CDN body.
- **Slice 4 (contract accuracy):** RE-VERIFIED in the YAML this window: `/{domain}/log/verify` (line 220)
  advertises ONLY `Domain` + `iscc_id` (no phantom `index`); `/{domain}/log/checkpoint` (line 252) `200` is
  `application/octet-stream` (line 92 `text/plain` is `/metrics`, correct); `/healthz` documents its `503`
  (line 52). Pinned by per-operation golden (`contract_test.go`) + a `TestNoMermaidInContract` fence ban.
  **Bookkeeping lag (NOT a code gap):** `issues.md` still carries 4 now-RESOLVED M-API entries (the phantom
  `index` `normal`, the `checkpoint` media-type `normal`, the `healthz` 503 `low`, and the umbrella "No
  machine-readable API contract" `normal`) — all FIXED + verified-in-doc this assessment, awaiting a
  `review` prune. **Carried `low`:** the mermaid ban is substring-only (misses `~~~mermaid` forms).

## Quality gates
**Status**: **GREEN (review-confirmed; CI re-running on HEAD).**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched only `internal/store` (1 non-test file + 1 test file) + `deploy/OPERATING.md`. Review
  recorded `mise run check` GREEN over 30 packages and `gofmt -l .` empty (clean).
- **CI**: `.github/workflows/` = `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz` smoke)
  + `pages.yml` + `publish.yml`. On HEAD `5846e97`: **Pages `success`**; **CI + Publish `in_progress`** at
  assessment (they were `success` on the immediate parent `476743f`; review recorded `mise run check` green
  locally on the migration commits). No CI failure observed — re-verify on next assessment if needed.
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `5846e97`) — migration runner: gate-green
  (30 pkgs), `gofmt` empty, all 5 `TestMigration*` PASS, both mutations reproduced (user_version-bump revert
  → idempotent+upgrade tests FAIL; error-swallow revert → fail-closed test FAILS), store leaf purity intact
  (`go list -deps ./internal/store | grep net/http` empty), gate-circumvention scan clean. PASS_WITH_NOTES
  (not PASS) only for the Codex-confirmed out-of-range `user_version` edge (filed `normal`, latent today).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in local
  TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 0 critical, 11 normal, 19 low** (the single `## … critical` grep match is the
  format-example line, not a real issue). Of the 11 normals: 4 are stale-but-resolved M-API entries
  (re-verified fixed this assessment) awaiting a prune; the remaining 7 real normals (2 data-model
  migration items, the WASM signature half, the realm-index Anchor honesty, the dossier §3 + §1, the
  proofserve-masthead identity) + the M-UI human/design gate keep it `IN_PROGRESS`. DONE requires 0
  critical AND 0 normal.

## Next Milestone
**The code-closable feature/milestone backlog is drained; the next deliberate code work is the paired
data-model `normal`s, for which this window's migration runner is the enabling step.**
1. **Prune the 4 resolved M-API issues** (next `update-state`/`review` can delete them — re-verified fixed
   in the served `openapi.yaml` this assessment) so the normal count reflects reality.
2. **Data-model `normal`s (code-closable, now unblocked by the migration runner):** land the
   `iscc_index.seq` single-global-PK → composite `(hub_id, seq)` rebuild as **migration index 0** (re-keys
   `iscc_index`, copies existing rows), plus the `iscc_index.go` writer/reader updates — AND fold in the
   **out-of-range `user_version` guard** (`version < 0 || version > len(migs)` → error), which goes live
   the instant the migration slice becomes non-empty. Both data-model normals + the runner guard close
   together.
3. **Design/human-blocked `normal`s (need a design pass or human sign-off, not autonomous loop work):** the
   WASM cross-origin signature half; the realm-index per-hub-vs-per-checkpoint Anchor honesty; the dossier
   §3 size/time decouple + §1 "resolved" wording; the proofserve-trio masthead identity; and the **M-UI exit
   visual-pass + human sign-off** (ADR-0012). The OTS Bitcoin-confirmed half remains offline-unprovable.
