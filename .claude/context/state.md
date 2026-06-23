<!-- assessed-at: a671e97eb39c4b14b24f4246ef40be57e07005a5 -->

# Project State

## Status: IN_PROGRESS

## Phase: Closing the `critical` log-browser navigation/parity gap — forward chain now committed; the single-record back-leg + record-list pager parity remain open.
The dossier "Browse the log →" repoint that was thought lost to a "git race" was re-committed
(`de9ed3c`), and the "race" itself was diagnosed as a false alarm: the advance agent's own
`git stash` dep-probe stranded its working set and was misread as a concurrent reset (`a671e97`
fixed the agent configs so it never mutates git to inspect it). Part-2a (record-list chrome +
breadcrumb + head) also landed (`7ea7fef`, review PASS_WITH_NOTES). The `critical` stays OPEN: the
record-list pager parity (part-2b) is unbuilt, and — newly verified this window — the **single-record
page `record.html` carries no navigation links at all**, so the no-JS back-leg of the chain is broken.

This window (`3c7cb2d..a671e97`, 5 commits): define-next `8a9210a` (part-2a) → advance `7ea7fef`
(part-2a chrome) → review `daa002e` (PASS_WITH_NOTES) → fix `de9ed3c` (re-commit the part-1 repoint)
→ fix `a671e97` (agent-config: stop the stash-probe). Code touched: `internal/dossier/{dossier.html,
handler_test.go}`, `internal/proofserve/{handler.go,records.html,records_test.go + the 8 *_test.go
recompiled under the new Handler sig}`, `cmd/iscc-monitor/main.go`. All other milestones carry forward
unchanged. Branch `develop`, HEAD `a671e97` level with `origin/develop`; working tree clean.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open. M2: 0 open. M3: 0 open (4/4). M-API: 4/4 MET. M-Deploy: 0 open.** Carried forward —
    none of their source touched this window.
  - **M-UI: 1 Verify criterion still REOPENED (navigation closure + log-browser record-list named-region
    parity) — `critical`, PARTIALLY closed.** Closed this window: part-1 "navigation closure / forward
    link" (`dossier.html:573` → `/{{.Origin}}/records`, committed `de9ed3c`, mounted at
    `proofserve/handler.go:242 case "/records"`, no-JS-chain test `TestDossierBrowseLogLandsOnLiveRecordList`)
    and part-2a "record-list chrome/breadcrumb/head" (committed `7ea7fef`, mutation-proven). **STILL
    OPEN:** (a) **part-2b pager parity** — `records.html:433` is still a bottom-only "showing N of M"
    span, not the mockup's top+bottom "seq X–Y of Z" pager disabled at the ends; (b) the **Status-badge
    row** the mockup's log browser omits is still present (`records.html:404`); (c) **single-record
    navigation closure** — `record.html` has ZERO `<a>` nav links (no `← Log browser` breadcrumb, no
    older/newer stepper, no "Back to list"/"Prove this record's inclusion →" actions), so the
    dossier→record-list→single-record→**back** chain is broken at the single-record back-leg. The
    `critical`'s navigation-closure clause requires back-links present on every surface; it is not met.
    Also still open (all `normal`/design- or human-blocked): §1 "resolved"-vs-unresolvable wording;
    per-hub-vs-per-checkpoint realm-index Anchor honesty; instance identity reached the proofserve
    record-list this window but the proofserve trio's other surfaces still vary; the mandatory **M-UI
    exit visual-pass + human sign-off** (ADR-0012).
  - **WASM verifier: published half CLOSED.** Still open: cross-origin **signature half** (no
    checkpoint-signature / did:web check), `normal`, design-first.
  - **OTS anchoring: 1/1 open.** Only a real Bitcoin confirmation remains (offline-unprovable).
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing / ~4 honesty/data-model.** This window
  ran a multi-commit `critical`-closing arc (part-2a chrome → re-commit part-1 → agent-config hardening).
  **No drift** — every iteration works the human-found `critical` toward closure or removes a recurring
  process hazard (the phantom git-race). The arc is genuinely converging on a named-region/navigation gap.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no M1 path (config, registry, follower, didweb, metrics) touched
this window beyond `cmd/iscc-monitor/main.go` threading `dashboard.Identity` into `proofserve.Handler`.
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze + alert-once + restart
survival; structured logs; `/metrics`.
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
**Status**: **met (4/4 Verify criteria)** — carried forward; the proofserve `serveRecords` signature
gained the hub `domain` + `dashboard.Identity` view-model threading, but the M3 functional contract
(CORS, verify-for-me JSON, `/` index, `/<domain>/log/`) is unchanged.
- **Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
  ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets and `/openapi.*` routes
  DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **NOT fully met — the `critical` log-browser gap is PARTIALLY closed.** Part-1 (forward
repoint) + part-2a (chrome/breadcrumb/head) landed and are committed; part-2b (pager) + the
single-record navigation closure are open.
- **`critical` part-1 — forward repoint (CLOSED, committed `de9ed3c`):** `dossier.html:573` links
  "Browse the log →" to `/{{.Origin}}/records` (live record-list browser, `serveRecords` at
  `handler.go:242`), no longer the `/<domain>/` checkpoint-summary dead end. `TestDossierBrowseLogLandsOnLiveRecordList`
  drives the no-JS forward leg through both handlers.
- **`critical` part-2a — record-list chrome/breadcrumb/head (CLOSED, committed `7ea7fef`):** `records.html`
  carries the shared masthead chrome (instance identity + `verify ↗ monitor.iscc.codes`, line 383), the
  absolute-site-root `← {{.Domain}} dossier` breadcrumb (line 389), and the eyebrow "Log browser" / hub /
  "<domain> · N records mirrored" head (lines 393-395). Mutation-proven by `TestRecordsHeadAndBreadcrumb`
  + `TestRecordsChromeInstanceIdentity`.
- **`critical` part-2b — record-list pager parity (OPEN):** `records.html:433` still renders a single
  bottom-only "showing {{len .Records}} of {{.Total}}" span (with newer/older plain links at 429/435),
  NOT the mockup's top+bottom "seq X–Y of Z" pager disabled at the ends; the Status-badge row
  (`records.html:404`) the Log-Browser mockup omits is still present.
- **`critical` — single-record navigation closure (OPEN, newly verified):** `internal/proofserve/record.html`
  has NO `<a>` navigation links — only the masthead chrome + `/_ds/` stylesheet refs. No `← Log browser`
  breadcrumb, no older/newer link stepper, no "Back to list" / "Prove this record's inclusion →" actions
  (the `ISCC Monitor - Single Record.dc.html` named regions). So the no-JS
  dossier→record-list→single-record→**back** chain is broken at the single-record back-leg — the
  navigation-closure clause (back-links present on every surface) is NOT met. The `critical` stays OPEN.
- **§3 honesty (CLOSED, carried):** `ListHubs` §3 `observed_at` subselect is `ORDER BY c.id ASC LIMIT 1`,
  mutation-proven.
- **Other open (carried, NOT critical):** §1 "resolved"-vs-unresolvable wording (`normal`); per-hub-vs-
  per-checkpoint realm-index Anchor honesty (`normal`, design question); the **M-UI exit visual-pass +
  human sign-off** (ADR-0012) not yet executed (the headless pass cannot launch — Chrome/Chromium absent
  in the devcontainer; degrades gracefully per protocol). The `/` realm-index sub-items are CLOSED (prunable).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live, byte-pinned); signature half design-blocked.
OTS — observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`, served
  byte-pinned at `monitor.iscc.codes/_ds/verify.wasm`. **Still open:** the cross-origin **SIGNATURE-half
  gap** (no checkpoint-signature / did:web check; success copy overstates an unrun key check) — `normal`,
  design-first.
- **OTS:** `.ots` route, §4 anchor clause, store layer, off-path stamp/upgrade loop, offline classifier,
  and both calendar-transport guards wired. Not-yet-built: a root reaching Bitcoin-confirmed (needs a live
  calendar + real BTC confirmation). 1/1 open. **Carried `low` defect:** nil-Stamper + empty-OTSBytes row
  falls through to the Upgrader (`otsloop.go:144`; test-only path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED.** Carried forward — no M-Deploy source touched. Verified
previously: SIGTERM trap, version-stamped binary + `/version`, production `Dockerfile` + CI `/healthz`
smoke, GHCR `publish.yml` (`:develop` + `:sha-<short>`), canonical `deploy/realm-testnet.txt`,
`deploy/OPERATING.md`, root `README.md`, on-disk migration mechanism + its first real entry (composite-PK).
- **Carried `low` traps:** out-of-range `user_version` guard runs AFTER `db.Exec(schemaSQL)`; composite-PK
  rebuild dropped `seq`'s standalone ordering path; `docker/login-action@v3` + `build-push-action@v6`
  still Node-20; `.dockerignore` slashless globs; §Footprint qualitative disk-growth; `cmd/verifier-site`
  non-atomic write; `schemaDeclaration/Deletion` URI triplication; masthead-fallback consts 4x; dossier
  overlay 3x. All latent.

## M-API — OpenAPI contract + hosted interactive API docs  (ADR-0014)
**Status**: **4/4 Verify MET** — carried forward (no M-API source touched this window).
- **Slice 1 (serve):** OpenAPI 3.1 served byte-verbatim at `GET /openapi.json` + `/openapi.yaml`.
- **Slice 2 (drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts path↔mux alignment.
- **Slice 3 (`/docs` + Stoplight Elements):** `/docs` mounts `<elements-api>` against same-origin
  byte-pinned `/_ds/elements.min.{js,css}`. No CDN body.
- **Slice 4 (contract accuracy):** verify has no phantom `index`; checkpoint 200 is octet-stream.
  **Bookkeeping lag (NOT a code gap):** `issues.md` still carries the 2 RESOLVED M-API `normal` entries
  (the phantom `index` param + the `checkpoint` media type) plus the M-API umbrella `normal` — verified
  fixed in the served doc, awaiting a `review`/`update-state` prune.

## Quality gates
**Status**: **GREEN on the last reviewed code commit (`daa002e` CI run); HEAD `a671e97` is a docs/agent-config commit.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable.
  Code changes this window are pure SSR-template + view-model threading (no crypto/Merkle/did:web path);
  oracle/conformance gate N/A for them. The part-1 re-commit (`de9ed3c`) and agent-config fix (`a671e97`)
  followed the part-2a review PASS_WITH_NOTES.
- **CI**: `.github/workflows/` = `ci.yml` + `pages.yml` + `publish.yml`. On commit `daa002e`
  (HEAD's last code-bearing ancestor with a run): **CI, Publish, Pages all `success`** (`gh run list`).
  Commits `de9ed3c` + `a671e97` are unrun-by-CI as of this assessment (HEAD ahead of the last run SHA).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `7ea7fef`, recorded in handoff.md) —
  part-2a chrome/breadcrumb/head gate-green (30 pkgs), `gofmt` empty, two new tests mutation-proven,
  Codex clean; the `critical` correctly kept OPEN (part-1 was still missing AT THAT TIME, since
  re-committed in `de9ed3c`).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 1 critical, 7 normal, ~22 low.** The **1 critical** is the dossier→log-browser
  navigation + off-mockup record list — part-1 + part-2a closed, but part-2b (pager) + the single-record
  back-leg + Status-row drop remain, so it stays OPEN. Of the 7 normals: ~4 are stale-but-resolved
  (the realm-index hero entry — all 4 sub-items closed — plus 2-3 M-API entries verified fixed in the
  served doc) awaiting a prune; the real open normals are the WASM signature half, the realm-index Anchor
  honesty, the §1 wording. DONE requires 0 critical AND 0 normal.

## Next Milestone
**Finish closing the `critical` — build part-2b (record-list pager parity) and the single-record page's
navigation closure — so the no-JS dossier→list→record→back chain is unbroken end-to-end.**
1. **Single-record navigation closure (`record.html`):** add the `← Log browser` breadcrumb (back to
   `records?from=…`), the older/newer record link stepper, and the "Back to list" / "Prove this record's
   inclusion →" actions per `ISCC Monitor - Single Record.dc.html`; add region tests so removing a
   back-link FAILS. This is the load-bearing remaining half of the navigation-closure clause.
2. **Record-list pager parity (part-2b):** replace the bottom-only "showing N of M" span with the
   mockup's top+bottom "seq X–Y of Z" pager disabled at the ends; drop the Status-badge row the
   Log-Browser mockup omits; reconcile the type-badge tint / append-only footnote copy. Region/golden
   tests so removing a region FAILS. Only when the full chain is traversable (forward + back, no-JS) AND
   the record-list regions match the mockup does the `critical` close with a review PASS.
3. **Prune the stale-but-resolved `normal`s** (next `update-state`/`review`): the fully-closed realm-index
   hero entry + the resolved M-API entries — dropping the real open normals to ~3.
4. **Design/human-blocked `normal`s:** the WASM cross-origin signature half; the realm-index per-hub-vs-
   per-checkpoint Anchor honesty; the §1 "resolved" wording; and the **M-UI exit visual-pass + human
   sign-off** (ADR-0012). The OTS Bitcoin-confirmed half remains offline-unprovable.
