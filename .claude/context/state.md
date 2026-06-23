<!-- assessed-at: adb70f62dc96aeb38f33c18901808bc18826b3a8 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-API near-complete — `GET /docs` + self-hosted Stoplight Elements LANDED this window.
The OpenAPI 3.1 contract is served byte-verbatim at `/openapi.json` + `/openapi.yaml` (slices 1+2), and the
interactive `<elements-api>` reference now mounts at `GET /docs` against same-origin byte-pinned assets
(slice 3). **3/4 M-API Verify criteria met.** The remaining M-API work is the **contract-accuracy doc-fix
(slice 4)**: the path-only drift test is structurally blind to the three filed param/media-type/response
mismatches, and the vendored Elements bundle carries a latent mermaid-from-unpkg no-CDN gap. None blocks
the loop; all are concrete, code-closable doc-touches.

This window (`058d943..adb70f6`, 4 commits: update-state `f4fd571` → define-next `08a7adb` → advance
`2250d53` → review `adb70f6`) is a clean single in-loop increment. The diff touched ONLY the new
`internal/docs/{docs.go,docs.html,docs_test.go}`, `internal/web/{web.go,web_test.go,elements.min.js,
elements.min.css}`, `cmd/iscc-monitor/main.go` (+1 `/docs` mount), the now-tracked `.claude/adr/0014`,
CLAUDE.md (one doc line), and context files. No M1/M2/M3/M-UI/WASM/OTS/M-Deploy SOURCE was touched — those
sections carry forward verified. HEAD `adb70f6` is level with `origin/develop`; CI / Pages / Publish all
`success`. (The working tree carries one uncommitted `target.md` edit from a prior `cid(steer)` — review
correctly left it for the next steer/update-state; not state-assessor's to commit.)

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no M1–M3 trust-path
    source touched this window.
  - **M-UI: behavioral + named-region Verify met on all five SSR surfaces; hub dossier closed at the code
    level.** No M-UI `critical` remains. Still open (all `normal`/human-blocked, NOT critical): the §3
    frozen size/time decouple, the §1 "resolved"-vs-unresolvable wording (a design call), the
    per-hub-vs-per-checkpoint realm-index Anchor honesty question, instance identity config-driven on THREE
    of six SSR mastheads (proofserve trio still placeholder), the `/` "recent declarers" hero sub-item
    (CLOSED, prunable), and the mandatory **M-UI exit visual-pass + human sign-off** (ADR-0012) across all
    surfaces.
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200 byte-pinned).
    **Still open:** the cross-origin **signature half** (no checkpoint-signature / did:web check), `normal`.
  - **OTS anchoring: 1/1 open (carried).** Only a real Bitcoin confirmation remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** No open Verify item remains.
  - **M-API: 3/4 met — slices 1+2+3 LANDED, slice 4 (contract accuracy) OPEN.** Slices 1+2 (`/openapi.json`
    + `/openapi.yaml` byte-verbatim serve + the route↔spec drift test) and slice 3 (`GET /docs` + the two
    self-hosted byte-pinned Stoplight Elements assets, no CDN, no `tryItCorsProxy`) are all met +
    review-verified. **Open (slice 4):** the contract-accuracy fixes the path-only drift test cannot catch
    — 2 `normal` (phantom `verify` `index` param + `checkpoint` media type) + 1 `low` (`healthz` 503) +
    the latent Elements-mermaid-from-unpkg `normal` no-CDN gap.
- **Last ~10 iterations: ~7-8 milestone-Verify-or-gate-advancing / ~2-3 context-prune.** This window
  closed the 3rd M-API Verify criterion (a real milestone advance: `/docs` + Stoplight Elements went from
  grep-confirmed-absent to live + visually verified). **No drift:** the loop retired its only `critical`,
  the steer reloaded with the code-closable M-API milestone, and the last two windows advanced it
  concretely (serve → drift test → `/docs`). The next code-closable bar is M-API slice 4 (the contract-
  accuracy doc-touch + mermaid guard).

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `058d943..adb70f6` diff touched NO M1 source (config, registry,
follower, didweb, metrics all untouched). All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (27 internal + 4 cmd): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`; internal —
  `badge, certificate, config, corsmw, dashboard, didweb, docs, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, openapi, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, version, web` (+ the new `docs` leaf this window). Module
  `github.com/iscc/iscc-monitor` (`go 1.26.1`).
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`, `github.com/iscc/iscc-lib/packages/go`,
  `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest / store path
touched this window. The M2 mirror/fsck contract is unchanged.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the M3 HTTP-seam contract is unchanged. The
OpenAPI machine routes (`/openapi.json`, `/openapi.yaml`) and the new `/docs` reference are additive
discovery surface, not part of the M3 verify-for-me / CORS / log-browser / realm-index Verify bars (which
stay green).

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets — now including the two
Elements assets — and the `/openapi.*` routes DO carry strong ETag + no-cache + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete on all five SSR surfaces; hub dossier closed at the code
level — no `critical` remains.** Carried forward — no `internal/dashboard|dossier|certificate|web|
proofserve` template or `.html` source touched this window (the diff is the new `internal/docs` leaf + the
`internal/web` Elements assets + `cmd/iscc-monitor`; `/docs` is a third-party Elements component with no DS
mockup, so it renders only the shared masthead chrome — review's agent-browser visual pass confirmed it
matches the sibling mastheads, no delta to file).
- **Still open (carried, NOT critical):** the §3 frozen size/time decouple (`normal`); the §1
  "resolved"-vs-unresolvable wording (`normal`, a design call — the mockup specifies the static phrasing);
  the per-hub-vs-per-checkpoint realm-index Anchor honesty question (`normal`); instance identity
  config-driven on THREE of six mastheads (proofserve trio still placeholder); the **M-UI exit visual-pass
  + human sign-off** (ADR-0012) not yet executed across all surfaces. The `/` "recent declarers" hero
  sub-item is fully CLOSED (prunable).

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** id-binding half closed in source + artifact, reproducible from `mise run build:wasm`
  (`TestWasmVerifyHashPinned` green, hash `2c91e61f…f48e` reverified against committed `verify.wasm`),
  publicly served byte-pinned at `monitor.iscc.codes/_ds/verify.wasm` (Pages run on HEAD `adb70f6` green).
  **Still open:** the cross-origin **SIGNATURE-half gap** (no checkpoint-signature / did:web check; the
  success copy overstates an unrun key check) — `normal`, design-first.
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
**Status**: **3/4 Verify met — slices 1+2+3 LANDED, slice 4 (contract accuracy) OPEN.** Verified this
window by reading the new `internal/docs` leaf, the `internal/web` Elements assets + pins, the `/docs`
mount in `buildMux`, and re-confirming the asset hashes against the committed bytes.
- **MET (slice 1 — serve):** A hand-authored OpenAPI **3.1** document lives in-repo
  (`internal/openapi/openapi.yaml`, `openapi: 3.1.0`, 13 machine paths, verify-for-me flagged weaker) with
  a byte-distinct `openapi.json` twin, both `go:embed`-ed; `internal/openapi.Handler()` serves them
  byte-verbatim at `GET /openapi.json` + `GET /openapi.yaml` (no-cache + strong content ETag + 304) under
  the outer CORS `*` wrap. (Carried verified from last window — no openapi source touched.)
- **MET (slice 2 — drift test):** `cmd/iscc-monitor/openapi_drift_test.go` asserts every doc-declared path
  is mounted AND every machine route the mux mounts is declared (HTML SSR + `/docs` on an explicit
  exclusion list). Review mutation-confirmed NON-VACUOUS both directions. Caveat (`learnings/openapi.md`):
  it gates **paths only** — `machineProbes()` is hand-maintained, not mux-derived — so it is structurally
  blind to param / media-type / response-code mismatches (exactly the slice-4 defects).
- **MET (slice 3 — `/docs` + Stoplight Elements):** LANDED this window. `mux.Handle("/docs", docs.Handler())`
  (`main.go:306`). `internal/docs.Handler()` (pure stdlib leaf) serves `200 text/html` mounting
  `<elements-api apiDescriptionUrl="/openapi.json">` against same-origin assets: the two `@stoplight/
  elements@9.0.23` files are vendored byte-verbatim into `internal/web` (`elements.min.js`,
  `elements.min.css`), served under `/_ds/elements.min.{js,css}` with the same no-cache + strong-ETag + 304
  policy as `verify.wasm`, each SHA-256 published as `ElementsJSHash` / `ElementsCSSHash` next to
  `WasmVerifyHash`. **Hashes reverified against committed bytes:** JS `46e5a044…d6938` ✓, CSS
  `a52002228…27b06` ✓. No external CDN in the body, no `tryItCorsProxy`. Tests:
  `docs.TestDocsRendersElementsShell / TestDocsNoTryItCorsProxy / TestDocsNoExternalCDN / TestDocsNonGET`,
  `web.TestElementsJSServed / TestElementsCSSServed / TestElementsAssetsHashPinned`. Review-verified live
  against the real binary + an agent-browser visual pass (Elements fully mounted against `/openapi.json`).
- **OPEN (slice 4 — contract accuracy):** the drift-test-blind fidelity fixes + the latent no-CDN gap.
  Filed: `normal` — `/{domain}/log/verify` advertises a phantom `index` query param `serveVerify` never
  reads; `normal` — `/{domain}/log/checkpoint` advertises `text/plain` but the handler serves
  `application/octet-stream`; `low` — `/healthz` omits its `503` store-down readiness response; `normal` —
  the vendored `elements.min.js` hardcodes `https://unpkg.com/mermaid@9.4.3/...` and lazy-loads it the
  moment a description renders a fenced ` ```mermaid ` block (NOT triggered today — the served OpenAPI doc
  has zero mermaid; close with a test banning ` ```mermaid ` in the served doc body). **This is the
  immediate standing code-closable milestone** — a single `openapi.yaml` doc-touch + regenerated JSON twin
  + a per-operation golden + the mermaid guard.

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`); `mise run check` runnable. This
  window touched Go source (new `internal/docs` leaf + `internal/web` Elements assets + `cmd/iscc-monitor`).
  `gofmt -l .` is **empty (verified clean)** outside the gitignored `cauldron/`.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` + `publish.yml`. **HEAD `adb70f6` (level with `origin/develop`): CI `success`,
  Pages `success`, Publish `success`** — all three completed/green (confirmed via `gh run list`).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (commit `adb70f6`) — covers M-API slice 3:
  gate-green (30 pkgs), `gofmt` empty, `/docs`→200 text/html with all four Elements/openapi markers and no
  external CDN, both assets served byte-equal with hash-matched strong ETags, `internal/docs` confirmed a
  pure leaf, `/docs` probed MOUNTED + excluded from the contract, live smoke + agent-browser visual pass
  both clean, gate-circumvention scan clean. PASS_WITH_NOTES (not PASS) only because Codex surfaced the
  latent Elements-mermaid-from-unpkg no-CDN gap (confirmed, not currently triggered, filed `normal`).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; fails on non-UTC dev hosts only (filed `low`). CI runs UTC and is green.
- **Open issues: 0 critical, 12 normal, 19 low** (the single `## … critical` grep match is the
  format-example block, not a real issue — verified: `awk` over `## ` headings finds no critical-priority
  entry). The 12 `normal`s + the unmet M-API slice 4 keep it `IN_PROGRESS`. DONE requires 0 critical AND 0
  normal.

## Next Milestone
**Advance M-API slice 4 (ADR-0014) — the LAST M-API Verify criterion, the only `critical`-free,
human-unblocked, oracle-gate-N/A work that closes a Done-When bar.** A single `internal/openapi/openapi.yaml`
doc-touch + a regenerated, deterministic JSON twin + new guards:
1. **Fix the 2 `normal` contract-accuracy defects:** remove the phantom `verify` `index` param; change the
   `/{domain}/log/checkpoint` `200` media type `text/plain` → `application/octet-stream`. (Fold the `low`
   `healthz` 503 in the same touch.)
2. **Add a per-operation golden** pinning each documented operation's params + `200` media type against the
   handler's real behavior — the path-only drift test (`TestOpenAPIDocsAgree` / the drift test) is
   structurally blind to all of these, so a one-sided body edit silently rots the twin.
3. **Close the latent no-CDN gap:** add a durable test banning a fenced ` ```mermaid ` block in the served
   OpenAPI doc body (the only trigger for the vendored Elements bundle's unpkg-mermaid lazy-load).

That meets the 4th M-API Verify criterion fully and lets a later `update-state` close the umbrella OpenAPI
issue. After M-API, the remaining `normal`s stay DONE blockers: the DB migration + `iscc_index.seq`
multi-hub PK collision (paired design decisions); the WASM signature-half gap; the per-hub-Anchor honesty
question; the dossier §3 size/time decouple + §1 wording. The OTS Bitcoin-confirmed half + the M-UI exit
visual-pass across all surfaces remain offline/human-blocked.
