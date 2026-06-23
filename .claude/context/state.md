<!-- assessed-at: 476743f3cf471e050ac199fe03597d4b45893243 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-API COMPLETE at the code level — slice 4 (contract accuracy) LANDED this window.
All four M-API Verify criteria are now met: the OpenAPI 3.1 contract is served byte-verbatim
(`/openapi.json` + `/openapi.yaml`), the path drift test gates route↔spec alignment, `GET /docs` mounts
self-hosted byte-pinned Stoplight Elements (no CDN), and the served doc now matches the real handlers
exactly (per-operation golden + mermaid-fence ban). M-Deploy + M-API — the two order-independent,
code-closable milestones — are both done in-repo; what remains for DONE is the standing data-model and
design-blocked `normal`s.

This window (`adb70f6..476743f`, 4 commits: update-state `26f3d39` → define-next `7b8c110` →
advance `53ee328` → review `476743f`) is a clean single in-loop increment. The diff touched ONLY
`internal/openapi/{openapi.yaml,openapi.json,contract_test.go}` (1 doc + its regenerated JSON twin + 1
new test file) plus context files. **No M1/M2/M3/M-UI/WASM/OTS/M-Deploy source was touched** — those
sections carry forward verified. HEAD `476743f` is level with `origin/develop`. Pages `success`; CI +
Publish were still `in_progress` at assessment (HEAD is the same SHA the latest green run covered up to
the new commit; review recorded gate-green locally). The working tree carries one uncommitted `target.md`
edit (the prior `cid(steer)` ADR-0014/M-API addition) — review correctly left it for the next steer; not
state-assessor's to commit.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no trust-path source touched.
  - **M-API: 0 open — 4/4 MET.** Slice 4 closed the last criterion: the served doc now matches the handlers
    (phantom `verify.index` removed, `checkpoint` media type `text/plain`→`application/octet-stream`,
    `healthz` `503` added), with a per-operation golden (`contract_test.go`) + a mermaid-fence ban, all
    mutation-proven by review.
  - **M-UI: behavioral + named-region Verify met on all five SSR surfaces; no `critical` open.** Still open
    (all `normal`/design- or human-blocked): the §3 frozen size/time decouple, the §1 unconditional
    "resolved"-vs-unresolvable wording, the per-hub-vs-per-checkpoint realm-index Anchor honesty question,
    instance identity config-driven on only THREE of six SSR mastheads (proofserve trio still placeholder,
    a `normal` sub-item), and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** No open Verify item remains.
- **Last ~10 iterations: ~7-8 milestone-Verify-or-gate-advancing / ~2-3 context-prune.** This window
  closed the **4th and final M-API Verify criterion** — a real milestone advance (M-API is now code-complete).
  **No drift:** the loop retired its only `critical`, reloaded with the code-closable M-API milestone, and
  the last three windows advanced it concretely (serve → drift test → `/docs` → contract accuracy). With both
  order-independent milestones (M-Deploy, M-API) now done in-repo, the remaining DONE blockers are
  design/human-blocked `normal`s — the standing-code-closable backlog is **drained**, which is the next
  steer's signal to either schedule the data-model `normal`s deliberately or surface the design/human gate.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `adb70f6..476743f` diff touched NO M1 source (config, registry,
follower, didweb, metrics all untouched). All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present** (26 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, docs, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest / store path
touched this window. The M2 mirror/fsck contract is unchanged.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged. The
OpenAPI machine routes (`/openapi.json`, `/openapi.yaml`) and the `/docs` reference are additive discovery
surface, not part of the M3 verify-for-me / CORS / log-browser / realm-index Verify bars (which stay green).

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and the `/openapi.*` routes
DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on all five SSR surfaces; hub dossier closed at the code
level — no `critical` remains.** Carried forward — no `internal/dashboard|dossier|certificate|web|
proofserve` template or `.html` source touched this window (the diff is openapi-only).
- **Still open (carried, NOT critical):** the §3 frozen size/time decouple (`normal`); the §1
  "resolved"-vs-unresolvable wording (`normal`, a design call — the mockup specifies the static phrasing);
  the per-hub-vs-per-checkpoint realm-index Anchor honesty question (`normal`); instance identity
  config-driven on THREE of six mastheads (proofserve trio still placeholder — its own `normal` sub-item);
  the **M-UI exit visual-pass + human sign-off** (ADR-0012) not yet executed across all surfaces. The `/`
  realm-index hero/logo/Checkpoint/Anchor sub-items are all CLOSED (that `normal` entry is now prunable).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`, publicly
  served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm` (Pages run on HEAD `476743f` green). **Still
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
iscc-infra ops residual (GHCR-package visibility) stays demoted to `low` (out of loop scope).
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT EXISTS`
  only, no `PRAGMA user_version`); the **`iscc_index.seq` single-global-PK multi-hub collision**
  (`RecordProjections`' `ON CONFLICT(seq)` clobbers when two hubs share a leaf index; pairs with migration).
- **Carried `low`:** `docker/login-action@v3` + `docker/build-push-action@v6` still Node-20;
  `.dockerignore` slashless globs; §Footprint qualitative disk-growth answer; `cmd/verifier-site`
  non-atomic write; `schemaDeclaration/Deletion` URI triplication; the masthead-identity fallback consts
  3x; the dossier overlay-precedence 3x duplication.

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **4/4 Verify MET — slice 4 (contract accuracy) LANDED this window.** Verified by reading the
served `internal/openapi/openapi.yaml` against the handlers, re-confirming the JSON twin, and the new
golden + mermaid ban; review (`476743f`) mutation-proved every assertion.
- **MET (slice 1 — serve):** hand-authored OpenAPI **3.1** (`openapi: 3.1.0`, machine surface, verify-for-me
  flagged weaker) with a byte-distinct JSON twin, both `go:embed`-ed; `internal/openapi.Handler()` serves
  them byte-verbatim at `GET /openapi.json` + `/openapi.yaml` (no-cache + strong ETag + 304) under CORS `*`.
- **MET (slice 2 — drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts path↔mux alignment both
  directions (HTML SSR + `/docs` excluded). Path-only by design — params/media/responses are the slice-4 golden's job.
- **MET (slice 3 — `/docs` + Stoplight Elements):** `mux.Handle("/docs", docs.Handler())` serves
  `200 text/html` mounting `<elements-api apiDescriptionUrl="/openapi.json">` against same-origin
  byte-pinned assets (`/_ds/elements.min.{js,css}`, SHA-256 published as `ElementsJSHash`/`ElementsCSSHash`).
  No external CDN body, no `tryItCorsProxy`.
- **MET (slice 4 — contract accuracy):** LANDED this window. The served doc now matches the handlers
  exactly — verified in the YAML: `/{domain}/log/verify` no longer advertises a phantom `index` param
  (`serveVerify` never reads it; `inclusion`/`entries` keep their legitimate `index`); `/{domain}/log/checkpoint`
  `200` is `application/octet-stream` (not `text/plain`; the `text/plain` at line 92 is `/metrics`, correct);
  `/healthz` now documents its `503`. Pinned by a per-operation golden (`contract_test.go`:
  `TestContractVerifyHasNoIndexParam / TestContractInclusionHasIndexParam / TestContractCheckpointMediaType /
  TestContractHealthzHas503`) and a `TestNoMermaidInContract` fence ban — review re-verified all 5 reverts
  FAIL. **Bookkeeping lag (NOT a code gap):** `issues.md` still carries the 3 now-resolved slice-4 issues
  (the phantom `index` `normal`, the `checkpoint` media-type `normal`, the `healthz` 503 `low`) + the umbrella
  "No machine-readable API contract" `normal` — all FIXED + verified-in-doc, awaiting a `review` prune.
  **Carried `low`:** the mermaid ban is substring-only (misses `~~~mermaid` / whitespace-fence forms).

## Quality gates
**Status**: **GREEN (review-confirmed; CI re-running on HEAD).**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched only `internal/openapi` (1 YAML doc + JSON twin + 1 test file). `gofmt -l .` is **empty
  (verified clean)** outside the gitignored `cauldron/`.
- **CI**: `.github/workflows/` = `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz` smoke)
  + `pages.yml` + `publish.yml`. On HEAD `476743f`: **Pages `success`**; **CI + Publish `in_progress`** at
  assessment (they were `success` on the immediate parent `adb70f6`, and review recorded `mise run check`
  green locally on the slice-4 commits). No CI failure observed — re-verify on next assessment if needed.
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `476743f`) — slice 4: gate-green (30 pkgs),
  `gofmt` empty, all 3 doc edits confirmed against ground-truth handlers (`serveVerify`/`serveCheckpoint`→
  `writeBlob`/`healthz.Handler`), JSON twin a faithful render (`reflect.DeepEqual`), golden mutation-non-vacuous
  (all 5 reverts FAIL), gate-circumvention scan clean. PASS_WITH_NOTES (not PASS) only for the Codex-confirmed
  substring-only mermaid-ban bypass (filed `low`, doc has zero mermaid today).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in local
  TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 0 critical, 10 normal, 19 low** (the single `## … critical` grep match is the format-example
  block, not a real issue). 4 of the 10 normals are the stale-but-resolved M-API entries awaiting a review
  prune; the remaining 6 real normals + the M-UI human/design gate keep it `IN_PROGRESS`. DONE requires 0
  critical AND 0 normal.

## Next Milestone
**Both order-independent code-closable milestones (M-Deploy + M-API) are now complete in-repo — the
standing-code-closable backlog is drained.** The remaining DONE blockers are all `normal`s that are either
data-model or design/human-blocked:
1. **Prune the 4 resolved M-API issues** (next `update-state`/`review` can delete them — fixes verified in
   the served doc) so the normal count reflects reality.
2. **Data-model `normal`s (code-closable, paired):** the on-disk DB migration story
   (`store.Open` `PRAGMA user_version`/idempotent `ALTER TABLE`) and the `iscc_index.seq` single-global-PK
   multi-hub collision (composite `(hub_id, seq)`) — these are the next deliberately-schedulable code work;
   a PK change needs the migration mechanism, so they fold together.
3. **Design/human-blocked `normal`s (need a design pass or human sign-off, not autonomous loop work):** the
   WASM cross-origin signature half; the realm-index per-hub-vs-per-checkpoint Anchor honesty; the dossier
   §3 size/time decouple + §1 "resolved" wording; the proofserve-trio masthead identity; and the **M-UI exit
   visual-pass + human sign-off** (ADR-0012). The OTS Bitcoin-confirmed half remains offline-unprovable.

With the autonomous code-closable backlog drained, the next steer should either schedule the data-model
`normal`s explicitly or surface that the residual is design/human-blocked (per the loop-stalls-on-blocked-DONE
memory) rather than spin on cosmetic chrome.
