<!-- assessed-at: 3c7cb2d5dd2a9860a847851162fbeeb34e065711 -->

# Project State

## Status: IN_PROGRESS

## Phase: Mid-iteration on the `critical` log-browser navigation gap — part (1) (forward repoint) implemented, uncommitted/unreviewed; part (2) (record-list dressing) still open.
The §3 dossier observed-time honesty fix stayed CLOSED (review PASS `c102fc3`, CI green). The latest
steer's `critical` — the dossier's "Browse the log →" dead-ends and the record list is off-mockup — is
being worked in two parts. This window's `define-next` (`3c7cb2d`) scoped **only part (1)** (repoint the
forward href), and `advance` has implemented exactly that in the working tree (uncommitted, not yet
reviewed). Part (2) (record-list named-region dressing) is explicitly deferred, so the `critical` and the
reopened M-UI navigation-closure Verify criterion remain OPEN until both land and review PASSes.

This window (`6220744..3c7cb2d`, 1 commit): define-next `3c7cb2d` (next.md only — repoint scope). On top of
HEAD there is an **uncommitted advance**: `internal/dossier/dossier.html` (href `/{{.Origin}}/` →
`/{{.Origin}}/records`, line 573) + `internal/dossier/handler_test.go` (updated href assertion + a new
`TestDossierBrowseLogLandsOnLiveRecordList` no-JS-chain test). Verified present in the working tree. No
other source touched; all other milestones carry forward unchanged. Branch `develop`, HEAD `3c7cb2d` (a
define-next docs commit) level with `origin/develop`; the advance is unpushed working-tree state.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 4/4 MET. M-Deploy: 0 open.** Carried forward —
    none of their source touched this window.
  - **M-UI: 1 Verify criterion REOPENED (navigation closure + log-browser record-list named-region
    parity) — `critical`, partially addressed.** Part (1) "navigation closure / forward link" is now
    implemented in the working tree but uncommitted+unreviewed: `dossier.html:573` links "Browse the log →"
    to `/{{.Origin}}/records` (the live record list, `serveRecords` is mounted at `case "/records"`,
    `handler.go:202`), no longer the `/<domain>/log/` checkpoint-summary dead end. Part (2) "log-browser
    named-region parity" is STILL OPEN (explicitly Not-In-Scope this iteration): `records.html:303` head
    reads "Records" (not eyebrow "Log browser" + hub + "N records mirrored"), no `← <hub> dossier`
    breadcrumb, no instance-identity + `verify ↗` chrome, and a bottom-only "showing N of M" pager
    (`records.html:338`) instead of top+bottom "seq X–Y of Z" pagers. Also still open (all `normal`/design-
    or human-blocked): §1 "resolved"-vs-unresolvable wording; per-hub-vs-per-checkpoint realm-index Anchor
    honesty; instance identity on only THREE of six SSR mastheads (proofserve trio placeholder); the
    mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED.** Still open: cross-origin **signature half** (no
    checkpoint-signature / did:web check), `normal`, design-first.
  - **OTS anchoring: 1/1 open.** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing / ~4 honesty/data-model.** define-next
  titles this window run §5 observation log → frozen Exhibit → M-API slices 1/3/4 → migration runner →
  composite-PK → §3 honesty (×2) → log-browser repoint. **No drift** — every iteration works a real
  named-region/honesty/contract gap to completion or a code-closable convergence move. The current
  iteration is a clear, human-found `critical` (the log-browser dead end), split into a code-closable
  forward-repoint half (done) and a multi-file dressing follow-on.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 path (config, registry, follower, didweb, metrics) touched
this window. `origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze + alert-once +
restart survival; structured logs; `/metrics`.
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
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 handler source touched this window.
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/openapi.*` routes
  DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **NOT fully met — the `critical` log-browser gap is PARTIALLY addressed this window.** Part
(1) is implemented in the working tree (uncommitted, unreviewed); part (2) is open.
- **`critical` part (1) — forward repoint (IMPLEMENTED, uncommitted):** `dossier.html:573` now links
  "Browse the log →" to `/{{.Origin}}/records` (the live record-list browser) instead of `/{{.Origin}}/`
  (the checkpoint-summary dead end). `serveRecords` is mounted at `case "/records"` (`proofserve/handler.go:202`),
  so the repointed href resolves to a live `200 text/html` record list. Tests updated:
  `handler_test.go` asserts the new href and adds `TestDossierBrowseLogLandsOnLiveRecordList` (drives the
  same fixture store through both the dossier and proofserve handlers, mutation-described). **Not yet
  committed or reviewed** — the `critical` is not closed until this lands with a review PASS.
- **`critical` part (2) — record-list named-region parity (OPEN, Not-In-Scope this iteration):**
  `records.html` still does not render `ISCC Monitor - Log Browser.dc.html` regions: head reads "Records"
  (line 303), no `← <hub> dossier` breadcrumb, no instance-identity + `verify ↗` chrome, bottom-only
  "showing N of M" pager (line 338) vs the mockup's top+bottom "seq X–Y of Z" pagers. next.md defers this
  as the follow-on (needs the hub domain + `dashboard.Identity` threaded through `proofserve.Handler`).
- **§3 honesty (CLOSED, carried):** `ListHubs` §3 `observed_at` subselect is `ORDER BY c.id ASC LIMIT 1`
  (`hubs.go:86`), mutation-proven; both the equivocation/higher-size and same-size-fork cases pinned.
- **Other open (carried, NOT critical):** §1 "resolved"-vs-unresolvable wording (`normal`); per-hub-vs-
  per-checkpoint realm-index Anchor honesty (`normal`, design question); instance identity on THREE of six
  mastheads (proofserve trio placeholder); the **M-UI exit visual-pass + human sign-off** (ADR-0012) not
  yet executed. The `/` realm-index hero/logo/Checkpoint/Anchor sub-items are CLOSED (prunable).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live, byte-pinned); signature half design-blocked.
OTS — observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`, served
  byte-pinned at `monitor.iscc.codes/_ds/verify.wasm` (Pages green on `c102fc3`). **Still open:** the
  cross-origin **SIGNATURE-half gap** (no checkpoint-signature / did:web check; success copy overstates
  an unrun key check) — `normal`, design-first.
- **OTS:** `.ots` route, §4 anchor clause (`ots.ConfirmedFor`), store layer, off-path stamp/upgrade loop,
  offline classifier, and both calendar-transport guards wired. Not-yet-built: a root reaching
  Bitcoin-confirmed (needs a live calendar + real BTC confirmation). 1/1 open. **Carried `low` defect:**
  nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED.** Carried forward — no M-Deploy source touched. Verified
previously: SIGTERM trap, version-stamped binary + `/version`, production `Dockerfile` + CI `/healthz`
smoke, GHCR `publish.yml` (`:develop` + `:sha-<short>`), canonical `deploy/realm-testnet.txt`,
`deploy/OPERATING.md`, root `README.md`, on-disk migration mechanism + its first real entry (composite-PK).
- **Carried `low` traps:** (1) out-of-range `user_version` guard runs AFTER `db.Exec(schemaSQL)` (hoist
  when `Open` is next touched); (2) composite-PK rebuild dropped `seq`'s standalone ordering path
  (`RecentRecords` sorts instead of index-walking; perf-only, negligible at 2-hub testnet). Both latent.
- **Carried `low`:** `docker/login-action@v3` + `docker/build-push-action@v6` still Node-20;
  `.dockerignore` slashless globs; §Footprint qualitative disk-growth; `cmd/verifier-site` non-atomic
  write; `schemaDeclaration/Deletion` URI triplication; masthead-fallback consts 3x; dossier overlay 3x.

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **4/4 Verify MET** — carried forward (no M-API source touched this window; re-verified last
window by reading the served `openapi.yaml` against the handlers).
- **Slice 1 (serve):** OpenAPI **3.1** (`openapi.yaml` + JSON twin, `go:embed`-ed) served byte-verbatim
  at `GET /openapi.json` + `/openapi.yaml` (no-cache + strong ETag + 304) under CORS `*`.
- **Slice 2 (drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts path↔mux alignment.
- **Slice 3 (`/docs` + Stoplight Elements):** `/docs` mounts `<elements-api>` against same-origin
  byte-pinned `/_ds/elements.min.{js,css}` (SHA-256 `web.ElementsJSHash`/`ElementsCSSHash`). No CDN body.
- **Slice 4 (contract accuracy):** `/{domain}/log/verify` has ONLY `Domain` + `iscc_id` (no phantom
  `index`); `/{domain}/log/checkpoint` 200 is `application/octet-stream`. Pinned by per-operation golden.
  **Bookkeeping lag (NOT a code gap):** `issues.md` STILL carries 3 now-RESOLVED M-API `normal` entries —
  all verified fixed in the served doc previously, awaiting a `review`/`update-state` prune.

## Quality gates
**Status**: **GREEN on the last reviewed code commit (`c102fc3`); the in-flight dossier advance is not yet
review-run.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. The
  only uncommitted change this window is the dossier href + test (`dossier.html`, `handler_test.go`) — a
  pure SSR-href + test-literal change; oracle/conformance gate is N/A (no crypto/Merkle/did:web path). Not
  yet gate-run by `review` (that is `review`'s job, next in the loop).
- **CI**: `.github/workflows/` = `ci.yml` + `pages.yml` + `publish.yml`. On the last code commit
  `c102fc3` (HEAD `3c7cb2d` is a docs-only define-next): **CI, Publish, Pages all `success`** (`gh run list`).
- **Latest `review` verdict: PASS / CONTINUE** (commit `c102fc3`) — §3 `id ASC` fork fix: gate-green
  (30 pkgs), `gofmt` empty, new same-size-fork test mutation-proven, store leaf purity intact, Codex clean.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 7 normal, 21 low.** The **1 critical** is the dossier→log-browser dead-end +
  off-mockup record list — part (1) implemented (uncommitted), part (2) still open, so it stays OPEN. Of
  the 7 normals: 4 are stale-but-resolved (the realm-index hero entry — all 4 sub-items closed — plus 3
  M-API entries verified fixed in the served doc) awaiting a prune; the real open normals are the WASM
  signature half, the realm-index Anchor honesty, the §1 wording. DONE requires 0 critical AND 0 normal.

## Next Milestone
**Commit + review the in-flight forward-repoint half of the `critical`, then close part (2) (record-list
dressing) — this is the active code-closable convergence arc.**
1. **Land + review part (1)** (uncommitted in the working tree): the dossier href repoint to
   `/{{.Origin}}/records` + the updated/added dossier tests. `review` runs `mise run check` and confirms
   the no-JS forward leg dossier → record list is live and mutation-proven.
2. **Close `critical` part (2) — dress `records.html` to `ISCC Monitor - Log Browser.dc.html`:** thread
   the hub domain + `dashboard.Identity` through `proofserve.Handler` → `serveRecords`; add the
   `← <hub> dossier` breadcrumb (and the record→list back-link), the "Log browser"/hub/"N records mirrored"
   head, the instance-identity + `verify ↗` chrome, and the top+bottom "seq X–Y of Z" pager (replacing the
   bottom-only "showing N of M"); add region/golden tests so removing a region FAILS. Only when BOTH parts
   land with a review PASS is the `critical` (and the reopened M-UI navigation-closure criterion) closed.
3. **Prune the 4 resolved-but-unpruned `normal`s** (next `update-state`/`review`): the fully-closed
   realm-index hero entry + the 3 resolved M-API entries (verify has no `index`, checkpoint is
   `application/octet-stream`, the "no machine-readable API contract" umbrella) — dropping real open normals to 3.
4. **Design/human-blocked `normal`s:** the WASM cross-origin signature half; the realm-index per-hub-vs-
   per-checkpoint Anchor honesty; the §1 "resolved" wording; and the **M-UI exit visual-pass + human
   sign-off** (ADR-0012). The OTS Bitcoin-confirmed half remains offline-unprovable.
