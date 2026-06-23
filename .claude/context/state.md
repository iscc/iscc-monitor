<!-- assessed-at: c213dd98ff80dcaee8171be8f8da2af1f2908a67 -->

# Project State

## Status: IN_PROGRESS

## Phase: Data-model hardening — the project's FIRST real migration landed (composite `(hub_id, seq)` PK).
All feature milestones (M1→M3, M-UI named-region, WASM published half, OTS observable halves) plus both
order-independent code-closable milestones (M-Deploy + M-API) are met in-repo. This window converted the
empty migration baseline into a real, single-entry migration list: `iscc_index` is re-keyed from the
single-global `seq PRIMARY KEY` to the composite `(hub_id, seq)` PK (fresh DB via `schema.sql`,
pre-existing DB via migration index 0), and the runner now bounds its read-back `user_version`
fail-closed. That closes the two paired data-model `normal`s plus the migration-empty `normal`. The
code-closable feature/milestone backlog is drained; what remains for DONE is the design/human-blocked
`normal`s + the M-UI human sign-off.

This window (`5846e97..c213dd9`, 4 commits: update-state `0627349` → define-next `cc293e6` → advance
`ed3206d` → review `c213dd9`) is a clean single in-loop increment. The diff touched ONLY
`internal/store/{iscc_index.go,schema.sql,sqlite.go}` + their two test files (plus context files). **No
M1/M2/M3/M-UI/WASM/OTS/M-Deploy/M-API milestone-criterion source was touched** — those sections carry
forward verified. HEAD `c213dd9` is level with `origin/develop`; the working tree is clean. **CI, Pages,
AND Publish are all `success` on HEAD `c213dd9`** (verified via `gh run list`).

## Convergence
- **Remaining Verify criteria (all milestone Verify bars MET; what's left is `normal`/`low` issues, not
  milestone criteria):**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 0 open (4/4). M-Deploy: 0 open.** All carried
    forward — none of their source was touched this window (the diff is `iscc_index` PK rework only).
  - **M-UI: behavioral + named-region Verify met on all five SSR surfaces; no `critical`.** Still open
    (all `normal`/design- or human-blocked): the dossier §3 frozen size/time decouple, the §1
    unconditional "resolved"-vs-unresolvable wording, the per-hub-vs-per-checkpoint realm-index Anchor
    honesty question, instance identity config-driven on only THREE of six SSR mastheads (proofserve trio
    still placeholder), and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~7 milestone-Verify-or-gate-advancing / ~3 data-model.** The last windows closed
  M-API (slices 1→4), then landed the migration MECHANISM, and this window landed the FIRST real migration
  (composite PK) — closing all three data-model `normal`s. **No drift:** the loop correctly worked the
  data-model `normal`s to completion (mechanism → migration index 0) rather than spinning on cosmetic
  chrome, exactly the convergence move the loop-stalls-on-blocked-DONE memory prescribes. The remaining
  `normal`s are now all design- or human-blocked; the code-closable backlog is genuinely drained.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `5846e97..c213dd9` diff touched NO M1 source (config, registry,
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
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest path touched.
The store touch this window was the `iscc_index` PK re-key (writer + schema + the migration), which keeps
the same `(iscc_id → seq)` one-to-many, schema-agnostic projection contract — now correctly composite so
two hubs sharing a leaf seq no longer collide. The `SQLiteFetcher` / `ProofBuilder` read side is unchanged
(the four reader queries were explicitly left out of scope and untouched). The M2 mirror/fsck contract holds.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged (no
handler source touched).
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
  served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm` (Pages run on HEAD `c213dd9` green). **Still
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
- **Migration story COMPLETE this window:** the on-disk migration HAZARD is now fully addressed — the
  mechanism (`internal/store/sqlite.go`: append-only `migrations []func(*sql.Tx) error`, fail-closed
  per-step `*sql.Tx`, idempotent, out-of-range `user_version` guard) is live AND the production list now
  carries its FIRST real entry (`len(migrations)==1`, the composite-PK rebuild). The interim "recreate the
  volume on a schema change" policy in `OPERATING.md` §Migration-policy now has a real migration backing it.
  Seven `TestMigration*` + `TestRecordProjectionsMultiHubSeqZero` exercise the runner, the migration body,
  and the guard (all reviewer mutation-proven load-bearing).
- **Carried `low` traps (this-window follow-ups, both filed by review):** (1) the out-of-range
  `user_version` guard runs AFTER `db.Exec(schemaSQL)`, so a downgrade-from-newer-binary re-applies the
  (idempotent, `CREATE … IF NOT EXISTS`-only — non-destructive) baseline DDL before the reject; hoist the
  guard ahead of the schema pass when `Open` is next touched. (2) the composite-PK rebuild dropped `seq`'s
  standalone ordering path, so `RecentRecords`' `ORDER BY i.seq DESC` now sorts instead of walking an index
  (a perf observation at scale, negligible at 2-hub testnet; add a `seq` index WITH the next `iscc_index`
  schema edit). Both are correctness-neutral, latent, and skipped by the loop.
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
  advertises ONLY `Domain` + `iscc_id` — **no phantom `index`** (the `index` param now appears only on
  `inclusion` (line 141) and `entries` (line 201), both legitimate); `/{domain}/log/checkpoint` (line 252)
  `200` is **`application/octet-stream`** (line 267); `/healthz` documents its **`503`** (line 52). Pinned
  by per-operation golden (`contract_test.go`) + a `TestNoMermaidInContract` fence ban.
  **Bookkeeping lag (NOT a code gap):** `issues.md` STILL carries 4 now-RESOLVED M-API entries (the phantom
  `index` `normal`, the `checkpoint` media-type `normal`, the `healthz` 503 `low`, and the umbrella "No
  machine-readable API contract" `normal`) — all FIXED + re-verified-in-doc this assessment, awaiting a
  `review` prune. **Carried `low`:** the mermaid ban is substring-only (misses `~~~mermaid` forms).

## Quality gates
**Status**: **GREEN (review-confirmed + CI green on HEAD).**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched only `internal/store` (3 non-test source files + 2 test files). Review recorded `mise run
  check` GREEN over 30 packages and `gofmt -l .` empty; `go.mod`/`go.sum` byte-unchanged; store leaf-purity
  intact (`go list -deps ./internal/store | grep net/http` empty — re-verified 0 this assessment).
- **CI**: `.github/workflows/` = `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz` smoke)
  + `pages.yml` + `publish.yml`. On HEAD `c213dd9` (verified via `gh run list`): **CI `success`, Pages
  `success`, Publish `success`** — all three green on the exact HEAD SHA.
- **Latest `review` verdict: PASS / CONTINUE** (commit `c213dd9`) — composite-PK migration: gate-green
  (30 pkgs), `gofmt` empty, all three new tests mutation-proven load-bearing (single-PK revert → multi-hub
  round-trip FAILS; guard removal → out-of-range FAILS; neutered migration body → upgrade FAILS), store
  leaf purity intact, scope = exactly the 3 asked source files. Oracle gate N/A (no crypto/RFC-6962/did:web
  path touched). Two Codex `[P2]` findings triaged DOWN to `low` and filed (the guard-ordering + the lost
  `seq` index path) — neither blocks; both strictly-narrower defense-in-depth / perf, not reachable today.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in local
  TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 0 critical, 8 normal, 21 low** (the single `## … critical` grep match is the
  format-example line, not a real issue). Of the 8 normals: 4 are stale-but-resolved M-API entries
  (re-verified fixed in the served doc this assessment) awaiting a prune; the remaining 4 real normals (the
  WASM signature half, the realm-index Anchor honesty, the dossier §3 + §1) plus the M-UI human/design gate
  keep it `IN_PROGRESS`. DONE requires 0 critical AND 0 normal.

## Next Milestone
**The code-closable feature/milestone backlog is drained (all data-model `normal`s now closed); the
remaining `normal`s are design- or human-blocked.**
1. **Prune the 4 resolved M-API issues** (next `update-state`/`review` can delete them — re-verified fixed
   in the served `openapi.yaml` this assessment: `/verify` has no `index`, `/checkpoint` is
   `application/octet-stream`, `/healthz` documents `503`) so the normal count reflects reality (would drop
   the real open-normal count to 4).
2. **Design/human-blocked `normal`s (need a design pass or human sign-off, not autonomous loop work):** the
   WASM cross-origin signature half (browser did:web resolution + note-signature verify); the realm-index
   per-hub-vs-per-checkpoint Anchor honesty (a design semantics decision); the dossier §3 size/time decouple
   on frozen hubs + the §1 "resolved" wording on the `unresolvable` path; the proofserve-trio masthead
   identity; and the **M-UI exit visual-pass + human sign-off** (ADR-0012). The OTS Bitcoin-confirmed half
   remains offline-unprovable.
3. **Optional `low` hardening when the touched files are next edited:** hoist the `user_version` guard ahead
   of `db.Exec(schemaSQL)`; add the `seq` index to `schema.sql` + migration 0 if a populated monitor shows
   `RecentRecords` hot.
