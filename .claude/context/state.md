<!-- assessed-at: 058d9430a713ffe5b8e061080e804a6638eb234b -->

# Project State

## Status: IN_PROGRESS

## Phase: M-API in flight — OpenAPI contract slices 1+2 LANDED. The OpenAPI 3.1 document is hand-authored
in-repo and served byte-verbatim at `GET /openapi.json` + `GET /openapi.yaml` (CORS `*`, no-cache + strong
ETag + 304), guarded by a non-vacuous route↔spec drift test. **2/4 M-API Verify criteria met.** What
remains for M-API is **slice 3: `GET /docs` + self-hosted byte-pinned Stoplight Elements** (grep-confirmed
absent — zero `/docs` / `elements-api` / `Stoplight` references in any `.go`). This is the immediate
standing code-closable work.

This window (`7a3aa92..058d943`, 4 commits: update-state `b47dbb0` → define-next `6c611e7` → advance
`e2de5e6` → review `058d943`) is a clean single in-loop increment. The diff touched ONLY the new
`internal/openapi/{openapi.go,openapi.yaml,openapi.json,openapi_test.go}`, `cmd/iscc-monitor/{main.go,
openapi_drift_test.go}`, plus CLAUDE.md (one doc line) and context files. No M1/M2/M3/M-UI/WASM/OTS/M-Deploy
SOURCE was touched — those sections carry forward verified. HEAD `058d943` is level with `origin/develop`;
CI / Pages / Publish all `success`.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no M1–M3 trust-path
    source touched this window.
  - **M-UI: behavioral + named-region Verify met on all five SSR surfaces; hub dossier FULLY closed at the
    code level.** No M-UI `critical` remains. Still open (all `normal`/human-blocked, NOT critical): the §3
    frozen size/time decouple, the §1 "resolved"-vs-unresolvable wording (a design call), the
    per-hub-vs-per-checkpoint realm-index Anchor honesty question, instance identity config-driven on THREE
    of six SSR mastheads (proofserve trio still placeholder), and the mandatory **M-UI exit visual-pass +
    human sign-off** (ADR-0012) across all surfaces.
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** No open Verify item remains.
  - **M-API: 2/4 met — slices 1+2 LANDED, slice 3 OPEN.** The `/openapi.json` + `/openapi.yaml` serve
    (byte-verbatim, CORS `*`, valid 3.1, 13 machine paths, SSR excluded, verify-for-me flagged weaker) and
    the route↔spec **drift test** are both met + review-verified. **Open:** `GET /docs` + the self-hosted
    byte-pinned Stoplight Elements assets under `/_ds/` (grep-confirmed not started). 3 contract-accuracy
    defects the PATH-only drift test cannot catch are filed (2 `normal`: phantom `verify` `index` param +
    `checkpoint` media type; 1 `low`: `healthz` 503) — to fold into the slice-3 doc-touch.
- **Last ~10 iterations: ~7-8 milestone-Verify-or-gate-advancing / ~2-3 context-prune.** This window
  closed 2/4 M-API Verify criteria (a real milestone advance). **No drift:** the loop retired its only
  `critical` last window, the steer reloaded with the fully code-closable M-API milestone, and this window
  advanced it concretely. There is concrete, non-cosmetic, non-blocked work to advance next (M-API slice 3).

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `7a3aa92..058d943` diff touched NO M1 source (config, registry,
follower, didweb, metrics all untouched). All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (26 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve,
  verifier, version, web` (+ the new `openapi` leaf this window). Module `github.com/iscc/iscc-monitor`
  (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest / store path
touched this window. The M2 mirror/fsck contract is unchanged.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged. The
new OpenAPI routes (`/openapi.json`, `/openapi.yaml`) are additive machine surface, not part of the M3
verify-for-me / CORS / log-browser / realm-index Verify bars (which stay green).

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets + the new `/openapi.*`
routes DO carry strong ETag).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on all five SSR surfaces; hub dossier FULLY closed at the
code level — no `critical` remains.** Carried forward — no `internal/dashboard|dossier|certificate|web|
proofserve` template or `.html` source touched this window (the diff is the new `internal/openapi` leaf +
`cmd/iscc-monitor`, which render no DS shell).
- **Still open (carried, NOT critical):** the §3 frozen size/time decouple (`normal`); the §1
  "resolved"-vs-unresolvable wording (`normal`, a design call — the mockup specifies the static phrasing);
  the per-hub-vs-per-checkpoint realm-index Anchor honesty question (`normal`); instance identity
  config-driven on THREE of six mastheads (proofserve trio still placeholder); the **M-UI exit visual-pass +
  human sign-off** (ADR-0012) not yet executed across all surfaces.

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`
  (`TestWasmVerifyHashPinned` green), publicly served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm`
  (Pages run on HEAD `058d943` green). **Still open:** the cross-origin **SIGNATURE-half gap** (no
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

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **2/4 Verify met — slices 1+2 LANDED, slice 3 OPEN.** Verified this window by reading the new
`internal/openapi` leaf, the served routes in `cmd/iscc-monitor/main.go buildMux`, and the drift test.
- **MET (slice 1 — serve):** A hand-authored OpenAPI **3.1** document lives in-repo
  (`internal/openapi/openapi.yaml`, `openapi: 3.1.0`, 13 machine paths, info/description flagging
  verify-for-me as the weaker tier-1 path) with a byte-distinct `openapi.json` twin, both `go:embed`-ed.
  `internal/openapi.Handler()` serves them byte-verbatim at `GET /openapi.json` (`application/json`) +
  `GET /openapi.yaml` (`application/yaml`) with the `internal/web` revalidating-cache shape (no-cache +
  strong content ETag + If-None-Match → 304); CORS `*` rides the outer `corsmw` wrap. Both routes mounted
  in `buildMux` (`main.go:299-300`) and reserved as exact mux-mount names. The leaf is pure (stdlib +
  `yaml.v3`, no store/metrics/logclient import — review-confirmed via `go list -deps`).
- **MET (slice 2 — drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts every doc-declared path
  is mounted AND every machine route the mux mounts is declared (HTML SSR routes on an explicit exclusion
  list). Review mutation-confirmed NON-VACUOUS both directions (drop a path → set-equality FAILS; declare a
  ghost path → "has no probe" FAILS). Caveat (recorded `learnings/openapi.md`): the drift test gates
  **paths only** — `machineProbes()` is hand-maintained, not mux-derived — so it is structurally blind to
  param / media-type / response-code mismatches; the 3 filed contract-accuracy defects are exactly that
  class.
- **OPEN (slice 3 — `/docs` + Stoplight Elements):** NOT STARTED. Grep over all `.go` finds ZERO `/docs` /
  `elements-api` / `Stoplight` references; `buildMux` mounts no `/docs` route; no Stoplight Elements asset
  or hash constant exists in `internal/web` next to `WasmVerifyHash`. Required: `GET /docs` → `200
  text/html` loading the self-hosted byte-pinned `<elements-api>` JS + CSS from `/_ds/` (each with a
  published `internal/web` SHA, same strong-ETag + no-cache + 304), `apiDescriptionUrl="/openapi.json"`, NO
  external CDN host, no `tryItCorsProxy`. **This is the immediate standing code-closable milestone.**
- **Filed contract-accuracy defects (drift-test-blind, fold into slice 3):** `normal` — `/{domain}/log/verify`
  advertises a phantom `index` query param `serveVerify` never reads (client silently gets a verdict for a
  different leaf); `normal` — `/{domain}/log/checkpoint` advertises `text/plain` but the handler serves
  `application/octet-stream`; `low` — `/healthz` omits its `503` store-down readiness response.

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched Go source (new `internal/openapi` leaf + `cmd/iscc-monitor`). `gofmt -l .` is **empty
  (verified clean)** outside the gitignored `cauldron/` (`main.go`'s function-call map keys are correctly
  left unaligned by gofmt).
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` + `publish.yml`. **HEAD `058d943` (level with `origin/develop`): CI `success`,
  Pages `success`, Publish `success`** — all three completed/green (confirmed via `gh run list`).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `058d943`) — covers M-API slice 1+2:
  gate-green (28 pkgs `ok`, `gofmt` empty), the drift test mutation-confirmed non-vacuous both directions,
  byte-verbatim + CORS `*` re-probed on the real mux, the JSON twin confirmed a faithful full-body render of
  the YAML, leaf purity confirmed, scope clean (2 prod files), gate-circumvention scan clean. Codex ran
  clean and surfaced 3 reviewer-confirmed contract-vs-handler mismatches (the param / media-type / response
  defects above) that the PATH-only drift test cannot catch — all filed, none blocking.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 0 critical, 10 normal, 18 low** (the single `## … critical` grep match is the
  format-example block, not a real issue — verified). The 10 `normal`s (now including 2 new OpenAPI
  contract-accuracy defects + the M-API umbrella with slice 3 remaining) + the unmet M-API slice 3 keep it
  `IN_PROGRESS`. DONE requires 0 critical AND 0 normal.

## Next Milestone
**Advance M-API slice 3 (ADR-0014) — the last 2 M-API Verify criteria, the only `critical`-free,
human-unblocked, oracle-gate-N/A work that closes a Done-When bar.**
1. **`GET /docs` + pinned assets:** self-host the Stoplight Elements `<elements-api>` JS + CSS under
   `/_ds/`, each byte-pinned with a published `internal/web` SHA constant next to `WasmVerifyHash` (same
   strong-ETag + no-cache + 304 policy), `apiDescriptionUrl="/openapi.json"`, NO external CDN host, no
   `tryItCorsProxy` — "try it" goes browser→this-instance directly on the existing CORS `*`. `GET /docs` →
   `200 text/html` with no external host in the body or any runtime call.
2. **Fold in the 2 `normal` contract-accuracy fixes** (this is the natural next doc-touch): remove the
   `verify` `index` param + fix the `checkpoint` media type to `application/octet-stream`, regenerate the
   JSON twin deterministically from the YAML (`TestOpenAPIDocsAgree` gates only the path set, so a one-sided
   body edit silently rots the twin). Consider adding a per-operation golden pinning each documented
   operation's params + `200` media type against the handler's real behavior, since the path-only drift
   test cannot.

In parallel, the remaining `normal`s stay DONE blockers: the DB migration + `iscc_index.seq` multi-hub PK
collision (paired design decisions); the WASM signature-half gap; the per-hub-Anchor honesty question; the
dossier §3 size/time decouple + §1 wording. The OTS Bitcoin-confirmed half + the M-UI exit visual-pass
across all surfaces remain offline/human-blocked.
