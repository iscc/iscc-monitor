<!-- assessed-at: 6220744d2d243185ab22bf5bba0ab53d6c8b6145 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI parity regression surfaced — a steer reopened a `critical` navigation/log-browser gap.
The §3 dossier observed-time honesty fix is now CLOSED end-to-end (equivocation/higher-size + same-size
fork, both mutation-pinned, review PASS `c102fc3`, CI fully green). But the latest steer (`6220744`)
filed — and this assessment VERIFIED in code — a `critical`: the hub dossier's "Browse the log →" lands
on a no-JS dead-end and the record-list log browser is off-mockup. That reopens an M-UI Verify criterion
(navigation closure + log-browser named-region parity) the prior state had carried as met. DONE is
blocked on this critical first, then the residual design/human-blocked `normal`s + the M-UI exit sign-off.

This window (`b080409..6220744`, 6 commits): update-state `d56e00f` → define-next `6bb9d46` → advance
`244d450` → review `c102fc3` → steer `3e51de1` (M-API to target.md — already landed in code) → steer
`6220744` (file the log-browser critical + sharpen M-UI nav/parity criteria). The only SOURCE diff was
`internal/store/hubs.go` (§3 subselect `ORDER BY c.id DESC` → `ASC`) + `hubs_test.go` (same-size-fork
regression test) — verified present (`hubs.go:86`). The two steer commits touched `target.md` +
`issues.md` only; no other source. Working tree clean; HEAD `6220744` level with `origin/develop`.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 4/4 MET. M-Deploy: 0 open.** Carried forward —
    none of their source touched this window.
  - **M-UI: 1 Verify criterion REOPENED (navigation closure + log-browser record-list named-region
    parity) — now `critical`.** Verified in code: `dossier.html:573` links "Browse the log →" to
    `/{{.Origin}}/` (the `/<domain>/log/` checkpoint-summary page, which carries no link to `records`, a
    no-JS dead end), NOT the record list the mockup points at; `records.html:303` head reads "Records"
    (not eyebrow "Log browser" + hub + "N records mirrored"), has no `← dossier` breadcrumb, omits the
    instance-identity + `verify ↗` chrome, and uses a bottom-only "showing N of M" pager instead of the
    mockup's top+bottom "seq X–Y of Z" pagers. Also still open (all `normal`/design- or human-blocked):
    §1 unconditional "resolved"-vs-unresolvable wording, the per-hub-vs-per-checkpoint realm-index Anchor
    honesty question, instance identity on only THREE of six SSR mastheads (proofserve trio placeholder),
    and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED.** Still open: cross-origin **signature half** (no
    checkpoint-signature / did:web check), `normal`, design-first.
  - **OTS anchoring: 1/1 open.** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing / ~4 honesty/data-model.** This window
  CLOSED the §3 same-size-fork honesty `normal` (the sole code-closable convergence move at the time);
  the steer then surfaced a genuine M-UI parity regression. **No drift** — the loop is working real
  named-region/honesty gaps to completion. The log-browser `critical` is the clear next code-closable
  move (a [human] host-machine review found it), exactly the convergence work the steer intends.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `b080409..6220744` source diff touched NO M1 path (config,
registry, follower, didweb, metrics all untouched). `origin`/`vkey` golden; fork/shrink/equivocation
golden-tested with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages**: `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal — `badge, certificate,
  config, corsmw, dashboard, didweb, docs, dossier, follower, healthz, index, logclient, metrics,
  metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve, verifier,
  version, web`. Module `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / `iscc_index` path touched this
window (the store touch was the `checkpoints` read-order in `ListHubs` §3 only). The `SQLiteFetcher` /
`ProofBuilder` read side is unchanged; the M2 mirror/fsck contract holds.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 handler source touched (only the
store query feeding §3 changed shape, observably correct).
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/openapi.*` routes
  DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **NOT fully met — one Verify criterion REOPENED as `critical` this window.** The §3 store
honesty fix is CLOSED (renderer was already correct; the query now picks the accepted row), but the
dossier→log-browser navigation leg is broken and the record-list surface is off-mockup.
- **`critical` (steer `6220744`, verified in code this assessment):** (1) the dossier's "Browse the log
  →" (`internal/dossier/dossier.html:573`) links to `/{{.Origin}}/` — the `/<domain>/log/`
  checkpoint-summary page (`internal/proofserve/browser.html`), which has no link to `records`, so with
  JS disabled the record-list browser is unreachable from the dossier (navigation-closure dead end). (2)
  the record list (`internal/proofserve/records.html`) does not render `ISCC Monitor - Log Browser.dc.html`
  landmark regions: head reads "Records" (line 303) not "Log browser"/hub/"N records mirrored"; no
  `← <hub> dossier` breadcrumb; missing instance-identity + `verify ↗` chrome; one bottom-only "showing
  N of M" pager (line 338) instead of the mockup's top+bottom "seq X–Y of Z" pagers disabled at the ends.
- **§3 honesty (CLOSED this window, advance `244d450`, review PASS `c102fc3`):** `ListHubs` §3
  `observed_at` subselect is `AND c.tree_size = f.last_size ORDER BY c.id ASC LIMIT 1` (`hubs.go:86`),
  mutation-proven by `TestListHubsFrozenObservedTracksAcceptedSameSizeFork` (reverting to `id DESC`
  fails it). Both the equivocation/higher-size and same-size-fork cases are now pinned.
- **Other open (carried, NOT critical):** §1 "resolved"-vs-unresolvable wording (`normal`); per-hub-vs-
  per-checkpoint realm-index Anchor honesty (`normal`, design question); instance identity on THREE of
  six mastheads (proofserve trio placeholder); the **M-UI exit visual-pass + human sign-off** (ADR-0012)
  not yet executed. The `/` realm-index hero/logo/Checkpoint/Anchor sub-items are CLOSED (prunable).

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
- **Carried `low` traps:** (1) out-of-range `user_version` guard runs AFTER `db.Exec(schemaSQL)`
  (downgrade re-applies idempotent baseline DDL before the reject — hoist when `Open` is next touched);
  (2) composite-PK rebuild dropped `seq`'s standalone ordering path (`RecentRecords` sorts instead of
  index-walking; perf-only, negligible at 2-hub testnet). Both correctness-neutral, latent.
- **Carried `low`:** `docker/login-action@v3` + `docker/build-push-action@v6` still Node-20;
  `.dockerignore` slashless globs; §Footprint qualitative disk-growth; `cmd/verifier-site` non-atomic
  write; `schemaDeclaration/Deletion` URI triplication; masthead-fallback consts 3x; dossier overlay 3x.

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **4/4 Verify MET** — re-verified this window by reading the served
`internal/openapi/openapi.yaml` against the handlers.
- **Slice 1 (serve):** OpenAPI **3.1** (`openapi.yaml` + JSON twin, `go:embed`-ed) served byte-verbatim
  at `GET /openapi.json` + `/openapi.yaml` (no-cache + strong ETag + 304) under CORS `*`.
- **Slice 2 (drift test):** `cmd/iscc-monitor/openapi_drift_test.go` present, asserts path↔mux alignment.
- **Slice 3 (`/docs` + Stoplight Elements):** `/docs` mounts `<elements-api>` against same-origin
  byte-pinned `/_ds/elements.min.{js,css}` (SHA-256 `web.ElementsJSHash`/`ElementsCSSHash`). No CDN body.
- **Slice 4 (contract accuracy):** RE-VERIFIED in the YAML this window: `/{domain}/log/verify` (line 220)
  has ONLY `Domain` + `iscc_id` — **no phantom `index`**; `/{domain}/log/checkpoint` (line 252) 200 is
  **`application/octet-stream`** (line 267). Pinned by per-operation golden (`contract_test.go`).
  **Bookkeeping lag (NOT a code gap):** `issues.md` STILL carries 3 now-RESOLVED M-API `normal` entries
  (phantom `index`; `/checkpoint` media-type; "No machine-readable API contract" umbrella) — all
  verified fixed in the served doc this assessment, awaiting a `review`/`update-state` prune.

## Quality gates
**Status**: **GREEN (review-confirmed + CI green on the last code commit).**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. The
  one source commit this window (`244d450`) touched `internal/store/hubs.go` + `hubs_test.go`. Review
  `c102fc3` recorded `mise run check` GREEN over 30 packages, `gofmt -l .` empty, `go.mod`/`go.sum`
  byte-unchanged, store leaf-purity intact, the new fork test mutation-proven load-bearing.
- **CI**: `.github/workflows/` = `ci.yml` + `pages.yml` + `publish.yml`. On the last code commit
  `c102fc3` (HEAD `6220744` is a docs-only steer): **CI, Publish, Pages all `success`** (`gh run list`).
- **Latest `review` verdict: PASS / CONTINUE** (commit `c102fc3`) — §3 `id ASC` fork fix: gate-green
  (30 pkgs), `gofmt` empty, new same-size-fork test mutation-proven (reverting to `id DESC` FAILS it),
  store leaf purity intact, scope = exactly the 2 asked files, oracle gate N/A (pure `checkpoints`
  read-order change), Codex concurs clean. The §3 honesty `normal` was deleted from `issues.md`.
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 7 normal, 21 low** (the bare `## <short title>` grep matches are the
  format-example lines, not real issues). The **1 critical** is the dossier→log-browser dead-end +
  off-mockup record list (verified real). Of the 7 normals: 4 are stale-but-resolved (the realm-index
  hero entry — all 4 sub-items closed — plus 3 M-API entries re-verified fixed in the served doc this
  assessment) awaiting a prune; the real open normals are the WASM signature half, the realm-index Anchor
  honesty, the §1 wording. DONE requires 0 critical AND 0 normal.

## Next Milestone
**Fix the `critical` log-browser navigation/parity gap first — it is the clear code-closable move and a
[human] host-machine finding; it reopens an M-UI Verify criterion.**
1. **Close the dossier→log-browser `critical`** (steer `6220744`): repoint the dossier's "Browse the log
   →" (`dossier.html:573`) at the record-list browser (`/<domain>/log/records`, not `/<domain>/log/`),
   and dress `records.html` to `ISCC Monitor - Log Browser.dc.html` named regions — `← <hub> dossier`
   breadcrumb; "Log browser"/hub/"N records mirrored" head; instance-identity + `verify ↗` chrome;
   top+bottom "seq X–Y of Z" pager disabled at the ends; `Seq·Type·ISCC-ID·Logged` rows each linking to
   its single record; append-only footnote. Keep the `/<domain>/log/` checkpoint-summary page (M3 +
   CLAUDE.md-documented) but it must not be the dossier's log-browser target. Add region/golden tests so
   removing a region FAILS; `mise run check` green; the no-JS dossier → record list → single record →
   back chain must be unbroken. Visual pass vs the mockup (ADR-0012) files residual deltas.
2. **Prune the 4 resolved-but-unpruned `normal`s** (next `update-state`/`review`): the fully-closed
   realm-index hero entry (all 4 sub-items closed) + the 3 resolved M-API entries (re-verified fixed in
   the served `openapi.yaml`: `/verify` has no `index`, `/checkpoint` is `application/octet-stream`, plus
   the "No machine-readable API contract" umbrella) — dropping the real open-normal count to 3.
3. **Design/human-blocked `normal`s (need a design pass or human sign-off):** the WASM cross-origin
   signature half (browser did:web resolution + note-signature verify); the realm-index per-hub-vs-per-
   checkpoint Anchor honesty; the §1 "resolved" wording on the `unresolvable` path; the proofserve-trio
   masthead identity; and the **M-UI exit visual-pass + human sign-off** (ADR-0012). The OTS
   Bitcoin-confirmed half remains offline-unprovable.
