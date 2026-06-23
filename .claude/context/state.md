<!-- assessed-at: 7a3aa929df2a9236240c963ea1aad63f6d149d35 -->

# Project State

## Status: IN_PROGRESS

## Phase: Hub-Dossier critical CLOSED + new M-API milestone opened. The lone code-closable `critical`
(Hub-Dossier increment 2b) is now DONE and review-deleted: the frozen Exhibit renders each contradictory
checkpoint's tree size ("size before → presented") read back unverified via the new pure
`logclient.CheckpointSizeFromRaw`, plus a content-derived evidence ref, and the §5 observation log skips
non-increasing pairs so a fork/shrink never renders a phantom `size N → N` transition. With that, **0
`critical` issues remain.** Concurrently a steer added a new v1 milestone — **M-API** (OpenAPI 3.1 +
hosted Stoplight Elements docs, ADR-0014) — to `target.md` and `Done When`; it is **fully unmet** and is
now the immediate standing code-closable work.

This window (`06c7749..7a3aa92`, 4 commits: update-state `3057661` → define-next `f313be5` → advance
`872ab8b` → review `7a3aa92`) is a clean single in-loop increment. The Go diff touched ONLY
`internal/dossier/{dossier.html,handler.go,handler_test.go}`, the new `internal/logclient/checkpointsize{.go,_test.go}`,
and `internal/store/checkpoints{.go,_test.go}` (the 4th-file `ListViolations` raw read-back). No
M1/M2/M3/WASM/OTS/M-Deploy source was touched — those sections carry forward verified. HEAD `7a3aa92` is
level with `origin/develop`; CI / Pages / Publish all `success`.

**Working-tree note:** `target.md` carries the unstaged M-API steer (the version I assess against) and a
new untracked `.claude/adr/0014-…md` — left for the steer/next cycle to commit; not my files to stage.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no M1–M3 trust-path
    source touched this window.
  - **M-UI: behavioral + named-region Verify met on all five SSR surfaces; the hub dossier is now FULLY
    closed at the code level** — §1–§4 doc + §5 observation log (skip-non-increasing) + the richer frozen
    Exhibit (size before→presented + evidence ref, fail-closed) all LANDED + review-verified. No M-UI
    `critical` remains. Still open (all `normal`/human-blocked, NOT critical): the §3 frozen size/time
    decouple, the §1 "resolved"-vs-unresolvable wording (a design call), the per-hub-vs-per-checkpoint
    realm-index Anchor honesty question, instance identity config-driven on THREE of six SSR mastheads
    (proofserve trio still placeholder), and the mandatory **M-UI exit visual-pass + human sign-off**
    (ADR-0012) across all surfaces.
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** No open Verify item remains.
  - **M-API (NEW — ADR-0014): 0/4 met — fully unmet.** No `/openapi.json` / `/openapi.yaml`, no `/docs`,
    no OpenAPI document, no Stoplight Elements assets in `internal/web` (grep-confirmed: zero `openapi`/
    `/docs`/`elements-api` references in the mux or any `.go`). ADR-0014 exists. This is the immediate
    standing code-closable milestone, with no human/infra dependency.
- **Last ~10 iterations: this window was 1 milestone-Verify-advancing increment** (closed the lone
  code-closable `critical`). Over the broader window ~7 milestone-Verify-or-gate-advancing / ~3
  context-prune. **No drift:** the loop just retired its only `critical`, and the steer immediately reloaded
  the queue with a fully code-closable milestone (M-API) — there is concrete, non-cosmetic, non-blocked
  work to advance next.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `06c7749..7a3aa92` diff touched NO M1 source (config, registry,
follower, didweb, metrics all untouched). All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (25 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, verifier,
  version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest path touched this
window. The store change this window (`ListViolations` now SELECTs+Scans the already-persisted `raw_a/raw_b`
for the Exhibit — a read-back-only change, schema byte-unchanged) is a read-only leaf; the M2 mirror/fsck
contract is unchanged.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged. The
dossier 2b work extends the existing `GET /<domain>` surface (an M-UI surface) without touching the
verify-for-me / CORS / log-browser / realm-index Verify bars.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on all five SSR surfaces; the hub dossier is now FULLY
closed at the code level — no `critical` remains.** Verified by reading the served template
(`internal/dossier/dossier.html`), the 18 dossier handler tests, and the new `TestCheckpointSizeFromRaw`
pair in `internal/logclient/checkpointsize_test.go`:
- **Hub dossier — CODE-COMPLETE.** The richer frozen **Exhibit** renders each contradictory checkpoint's
  tree size ("tree size <before> → then presented <presented>") read back UNVERIFIED via the new pure
  `logclient.CheckpointSizeFromRaw` (`checkpointsize.go:40`; `note.Open` empty-verifier-list →
  `*UnverifiedNoteError` → line-2 size parse mirroring `VerifyCheckpoint`, fail-closed on any malformed
  raw), plus a deterministic evidence ref over the raw pair; an unparseable raw renders "tree sizes
  unavailable" with NO fabricated `tree size 0 →`. The **§5 observation log** skips non-increasing
  consecutive pairs (`handler.go:482` `if newer.TreeSize <= older.TreeSize` skip), so a fork (same size,
  different root) or shrink never renders a phantom `size N → N` / `larger → smaller` transition; the
  freeze pointer + honest empty state are kept and the recurring synthesized per-poll "consistent" trap
  stays tested-against. `TestDossierFrozenExhibit` (real sb0(10183)/sb1(61) fork pair) +
  `TestDossierObservationLogFrozenNoPseudoTransition` cover both.
- **Other surfaces met + carried:** `/` realm index (with a "Recently declared" hero row), log browser,
  single record, certificate, badge, DS shell, proof-bundle all render and pass the behavioral +
  named-region HTTP-seam Verify.
- **Still open (carried, NOT critical):** the §3 frozen size/time decouple (`normal`); the §1
  "resolved"-vs-unresolvable wording (`normal`, a design call — the mockup specifies the static phrasing);
  the per-hub-vs-per-checkpoint realm-index Anchor honesty question (`normal`); instance identity
  config-driven on THREE of six mastheads (proofserve trio still placeholder); the **M-UI exit visual-pass
  + human sign-off** (ADR-0012) not yet executed across all surfaces (review ran a per-increment visual
  pass on the Exhibit + §5 and filed no delta).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`
  (`TestWasmVerifyHashPinned` green), publicly served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm`
  (Pages run on HEAD `7a3aa92` green). **Still open:** the cross-origin **SIGNATURE-half gap** (no
  checkpoint-signature / did:web check; the success copy overstates an unrun key check) — `normal`,
  design-first.
- **OTS:** `.ots` route, §4 anchor clause (`ots.ConfirmedFor`), store layer, off-path stamp/upgrade loop,
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

## M-API — OpenAPI contract + hosted interactive API docs  (NEW — ADR-0014)
**Status**: **NOT STARTED — 0/4 Verify met.** A steer added M-API to `target.md` + `Done When` this window;
`.claude/adr/0014-openapi-contract-and-hosted-api-docs.md` exists (untracked in the working tree). Nothing
is implemented: grep over all `.go` finds ZERO `openapi` / `/docs` / `elements-api` / `Stoplight`
references, and `cmd/iscc-monitor/main.go buildMux` mounts no `/openapi.json`, `/openapi.yaml`, or `/docs`
route. Required (all in-repo / CI-verifiable, no human step): a hand-authored OpenAPI 3.1 doc served
byte-verbatim at `/openapi.json` + `/openapi.yaml` (CORS `*`), a route-vs-spec **drift test**, and `GET
/docs` serving self-hosted byte-pinned Stoplight Elements (JS + CSS) under `/_ds/` with no external CDN.
**This is the immediate standing code-closable milestone** — order-independent, no oracle-gate path.

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched Go source (`internal/dossier`, the new `internal/logclient/checkpointsize`,
  `internal/store/checkpoints`). `gofmt -l .` is **empty (verified clean)** outside the gitignored
  `cauldron/`.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` + `publish.yml`. **HEAD `7a3aa92` (level with `origin/develop`): CI `success`,
  Pages `success`, Publish `success`** (all three completed/green — confirmed via `gh run list`).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `7a3aa92`) — covers the richer-Exhibit +
  §5 fork/shrink advance: gate-green (all 28 pkgs `ok`, `gofmt` empty), three mutations confirmed
  non-vacuous (revert the `<=` skip → §5 test FAILS; `CheckpointSizeFromRaw` off-by-one → ground-truth
  equality + Exhibit literal FAIL; revert `ListViolations` raw SELECT → round-trip FAILS), `CheckpointSizeFromRaw`'s
  unverified size byte-EQUALS `VerifyCheckpoint`'s independent parse (genuine ground truth), Codex clean,
  and a live `agent-browser` visual pass confirmed the Exhibit + §5 match the design intent (no delta filed).
  One accepted scope deviation (a 4th prod file, `ListViolations` raw read-back — read-only, schema
  byte-unchanged) and one cosmetic note (a redundant bare `mono` class); neither blocks.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 0 critical, 8 normal, 17 low** (the single `## … critical` grep match is the
  format-example block, not a real issue — verified). The code-closable Hub-Dossier `critical` was deleted
  by this window's review. DONE requires 0 critical AND 0 normal; the 8 `normal`s + the new unmet M-API
  milestone keep it `IN_PROGRESS`.

## Next Milestone
**Advance M-API (ADR-0014) — the new standing code-closable v1 milestone, now the only `critical`-free,
human-unblocked, oracle-gate-N/A work that closes a Done-When bar.** Build it in slices:
1. **OpenAPI 3.1 doc + serve:** hand-author the document covering the machine surface ONLY (`/healthz`,
   `/version`, `/metrics` as Prometheus text, the per-hub `inclusion`/`consistency`/`entries`/`checkpoint`/
   `checkpoint.ots`/`tile` routes, `verify-for-me` flagged as the weaker tier-1 path, `/inclusion/<id>.bundle`;
   HTML SSR surfaces excluded); serve it byte-verbatim via `go:embed` at `GET /openapi.json` + `/openapi.yaml`
   under `corsmw` (CORS `*`).
2. **Drift test:** assert every doc-declared path is mounted in the real mux AND every machine route the mux
   mounts is declared (HTML SSR routes on an explicit exclusion list) — reverting the alignment must FAIL.
3. **`GET /docs` + pinned assets:** self-host the Stoplight Elements `<elements-api>` JS + CSS under `/_ds/`,
   each byte-pinned with a published `internal/web` SHA constant next to `WasmVerifyHash` (same strong-ETag +
   no-cache + 304 policy), `apiDescriptionUrl="/openapi.json"`, NO external CDN host, no `tryItCorsProxy`.

In parallel, the 8 `normal`s remain DONE blockers: the DB migration + `iscc_index.seq` multi-hub PK
collision (paired design decisions); the WASM signature-half gap; the per-hub-Anchor honesty question; the
dossier §3 size/time decouple + §1 wording; and the OpenAPI `normal` (now superseded by the M-API milestone
itself). The OTS Bitcoin-confirmed half + the M-UI exit visual-pass across all surfaces remain
offline/human-blocked.
