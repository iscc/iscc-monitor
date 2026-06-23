<!-- assessed-at: b0804098e71773e9f785befb76db03710b400b9e -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI honesty polish — the dossier §3 observed-time is now tied to the accepted checkpoint row.
All feature milestones (M1→M3, M-UI named-region, WASM published half, OTS observable halves) plus both
order-independent code-closable milestones (M-Deploy + M-API) meet their `target.md` Verify bars in-repo.
This window re-tied the hub-dossier §3 "latest checkpoint observed-time" to the accepted `last_size`
checkpoint row, closing the equivocation / higher-tree-size decouple. What remains for DONE is the
design/human-blocked `normal`s (incl. the SAME-SIZE fork remainder of this very §3 fix) plus the M-UI
exit visual-pass + human sign-off.

This window (`c213dd9..b080409`, 4 commits: update-state `b565f4e` → define-next `6d52014` → advance
`820a831` → review `b080409`) is a clean single in-loop increment. The source diff touched ONLY
`internal/store/hubs.go` (the §3 `observed_at` subselect) + `internal/store/hubs_test.go` (plus context
files). **No M1/M2/M3/M-UI-template/WASM/OTS/M-Deploy/M-API milestone-criterion source was touched** —
those sections carry forward verified. HEAD `b080409` is level with `origin/develop`; the working tree
carries only an uncommitted `target.md` steer mod (`d2f259e`), not this role's to commit. **Pages is
`success` on HEAD; CI + Publish are still `in_progress` on HEAD `b080409`** (the previous HEAD `c213dd9`
was fully green across all three).

## Convergence
- **Remaining Verify criteria (all milestone Verify bars MET; what's left is `normal`/`low` issues, not
  milestone criteria):**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 0 open (4/4). M-Deploy: 0 open.** Carried forward —
    none of their source was touched this window (the diff is the §3 `ListHubs` subselect only).
  - **M-UI: behavioral + named-region Verify met on all five SSR surfaces; no `critical`.** Still open
    (all `normal`/design- or human-blocked): the dossier **§3 SAME-SIZE FORK** remainder (this window
    closed the equivocation/higher-size case; the fork case still pairs the accepted size with the
    rejected fork checkpoint's time — `id ASC` is the recorded fix), the §1 unconditional
    "resolved"-vs-unresolvable wording, the per-hub-vs-per-checkpoint realm-index Anchor honesty question,
    instance identity config-driven on only THREE of six SSR mastheads (proofserve trio still placeholder),
    and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing / ~4 honesty/data-model.** Recent windows
  closed M-API (slices 1→4), landed the migration mechanism + the FIRST real migration (composite PK,
  closing 3 data-model `normal`s), and this window advanced the §3 honesty `normal` (equivocation case
  closed, fork remainder narrowed). **No drift:** the loop is working the remaining honesty `normal`s to
  completion rather than spinning on cosmetic chrome — the §3 fork remainder (`id DESC` → `id ASC`) is the
  natural code-closable follow-up and SHOULD be the next advance, exactly the convergence move the
  loop-stalls-on-blocked-DONE memory prescribes.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `c213dd9..b080409` diff touched NO M1 source (config, registry,
follower, didweb, metrics all untouched). All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (26 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, docs, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest / `iscc_index`
path touched this window (the store touch was the `checkpoints` read-shape in `ListHubs` §3 only). The
`SQLiteFetcher` / `ProofBuilder` read side is unchanged. The M2 mirror/fsck contract holds.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged (no
handler source touched — only the store query feeding §3 changed shape, observably correct).
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and the `/openapi.*`
  routes DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on all five SSR surfaces; no `critical` remains.**
Carried forward — no `internal/dashboard|dossier|certificate|web|proofserve` template or `.html` source
touched this window (the §3 renderer `internal/dossier` is untouched; the change is store-side, feeding it
honest data).
- **§3 honesty (this window, advance `820a831`):** the `ListHubs` §3 `observed_at` subselect is re-tied to
  `AND c.tree_size = f.last_size ORDER BY c.id DESC LIMIT 1`, mutation-proven by
  `TestListHubsFrozenObservedTracksAcceptedSize`. This CLOSES the equivocation/higher-tree-size decouple (a
  rejected LARGER checkpoint no longer supplies §3's time). It does **NOT** close the SAME-SIZE FORK case
  (a fork records a contradictory row at `tree_size == last_size`, later `id`, so `id DESC` still picks the
  rejected row) — review confirmed this via reproduction and kept the §3 `normal` open, narrowed to the
  fork remainder. Recorded fix: `ORDER BY c.id ASC` (the accepted row at a size is the earliest `id`).
- **Still open (carried, NOT critical):** the §3 same-size-fork remainder (`normal`); the §1
  "resolved"-vs-unresolvable wording (`normal`); the per-hub-vs-per-checkpoint realm-index Anchor honesty
  question (`normal`); instance identity config-driven on THREE of six mastheads (proofserve trio still
  placeholder); the **M-UI exit visual-pass + human sign-off** (ADR-0012) not yet executed across all
  surfaces. The `/` realm-index hero/logo/Checkpoint/Anchor sub-items are all CLOSED — that entry is fully
  resolved and prunable.

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`, publicly
  served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm` (Pages run on HEAD `b080409` green). **Still
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
canonical `deploy/realm-testnet.txt` (golden-accepted), `deploy/OPERATING.md`, root `README.md`. The
on-disk migration mechanism + its FIRST real entry (composite-PK rebuild, `len(migrations)==1`) are live,
backing the `OPERATING.md` §Migration-policy.
- **Carried `low` traps:** (1) the out-of-range `user_version` guard runs AFTER `db.Exec(schemaSQL)`
  (downgrade-from-newer re-applies the idempotent baseline DDL before the reject — hoist the guard ahead of
  the schema pass when `Open` is next touched); (2) the composite-PK rebuild dropped `seq`'s standalone
  ordering path, so `RecentRecords`' `ORDER BY i.seq DESC` sorts instead of index-walking (perf-only,
  negligible at 2-hub testnet; add a `seq` index WITH the next `iscc_index` schema edit). Both
  correctness-neutral, latent, skipped by the loop.
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
  advertises ONLY `Domain` + `iscc_id` — **no phantom `index`** (the `index` param appears only on
  `inclusion` (line 141) and `entries` (line 201), both legitimate); `/{domain}/log/checkpoint` (line 252)
  `200` is **`application/octet-stream`** (line 267); `/healthz` (line 37) documents its **`503`** (line 52).
  Pinned by per-operation golden (`contract_test.go`) + a `TestNoMermaidInContract` fence ban.
  **Bookkeeping lag (NOT a code gap):** `issues.md` STILL carries 4 now-RESOLVED M-API entries (the phantom
  `index` `normal`, the `checkpoint` media-type `normal`, the `healthz` 503 `low`, and the umbrella "No
  machine-readable API contract" `normal`) — all FIXED + re-verified-in-doc this assessment, awaiting a
  `review` prune. **Carried `low`:** the mermaid ban is substring-only (misses `~~~mermaid` forms).

## Quality gates
**Status**: **GREEN (review-confirmed; CI re-running on HEAD, previous HEAD fully green).**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched only `internal/store/hubs.go` (1 source file) + `hubs_test.go`. Review recorded `mise run
  check` GREEN over 30 packages and `gofmt -l .` empty; `go.mod`/`go.sum` byte-unchanged; store leaf-purity
  intact (`go list -deps ./internal/store | grep net/http` empty per review).
- **CI**: `.github/workflows/` = `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz` smoke)
  + `pages.yml` + `publish.yml`. On HEAD `b080409` (via `gh run list`): **Pages `success`; CI + Publish
  `in_progress`** (not yet concluded at assessment time). The previous HEAD `c213dd9` was `success` on all
  three.
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `b080409`) — §3 observed-time tie:
  gate-green (30 pkgs), `gofmt` empty, the new frozen test mutation-proven load-bearing (reverting the
  subselect to `tree_size DESC, id DESC` FAILS it; the verified-hub test stays green), store leaf purity
  intact, scope = exactly the asked 2 files. Oracle gate N/A (pure `checkpoints` read-shape change, no
  crypto/RFC-6962/did:web path). One Codex `[P2]` finding CONFIRMED by reviewer reproduction (the §3
  same-size-fork remainder) — does NOT block; the §3 `normal` is kept open and narrowed to the fork case
  rather than deleted.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in local
  TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 0 critical, 8 normal, 21 low** (the single `## … critical` grep match is the
  format-example line, not a real issue). Of the 8 normals: 4 are stale-but-resolved (the realm-index hero
  entry — all 4 sub-items closed — plus 3 M-API entries re-verified fixed in the served doc this
  assessment) awaiting a prune; the remaining 4 real normals (the WASM signature half, the realm-index
  Anchor honesty, the §3 same-size-fork remainder, the §1 wording) plus the M-UI human/design gate keep it
  `IN_PROGRESS`. DONE requires 0 critical AND 0 normal.

## Next Milestone
**The §3 same-size-fork remainder is the next code-closable advance; the other open `normal`s are design-
or human-blocked.**
1. **Finish the §3 honesty fix** (review's recorded next step, code-closable): change the `ListHubs` §3
   subselect `ORDER BY c.id DESC` → `ORDER BY c.id ASC` (the accepted checkpoint at a given size is the
   EARLIEST `id` — `store.CheckpointAt` already uses `ORDER BY rowid LIMIT 1` for this exact selection); add
   a same-size-fork regression test (accepted root @ t1, forked root @ t2>t1, no `last_size` advance, Freeze)
   asserting §3 time == t1; keep `TestListHubsFrozenObservedTracksAcceptedSize` green. One-file + test, ≤3
   budget, oracle N/A.
2. **Prune the 5 resolved-but-unpruned issues** (next `update-state`/`review`): the fully-closed realm-index
   hero entry (all 4 sub-items closed) + the 3 resolved M-API entries (re-verified fixed in the served
   `openapi.yaml` this assessment: `/verify` has no `index`, `/checkpoint` is `application/octet-stream`,
   `/healthz` documents `503`) + the "No machine-readable API contract" umbrella — so the normal count
   reflects reality (would drop the real open-normal count to 4, three after the §3 fix lands).
3. **Design/human-blocked `normal`s (need a design pass or human sign-off):** the WASM cross-origin
   signature half (browser did:web resolution + note-signature verify); the realm-index
   per-hub-vs-per-checkpoint Anchor honesty (a design semantics decision); the §1 "resolved" wording on the
   `unresolvable` path; the proofserve-trio masthead identity; and the **M-UI exit visual-pass + human
   sign-off** (ADR-0012). The OTS Bitcoin-confirmed half remains offline-unprovable.
