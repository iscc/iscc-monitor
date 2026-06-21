<!-- assessed-at: e0d8e7175c44d48ee0a7e8d5fe436d1bbcb9778a -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress — the shared DS v2 shell is now complete: token CSS **and** self-hosted Readex Pro / JetBrains Mono webfonts ship CDN-free under `/_ds/`, served by one `web.Handler` with revalidating caches.
This iteration extended `internal/web`: the eight latin woff2 subsets (Readex Pro 300/400/500/600/700 +
JetBrains Mono 400/500/700) + a same-origin `@font-face` stylesheet are now `go:embed`ed and served at
`/_ds/fonts/...` / `/_ds/fonts.css`, the mount moved from an exact path to a `/_ds/` subtree, and every
`/_ds/` asset moved off `immutable` to `no-cache` + strong content-ETag + 304 (the `tilesserve.writeBlob`
shape) — which also closed the open `normal` cache issue. M1/M2/M3 remain fully met. Remaining v1 work:
the rest of M-UI (realm-index redress, dossier, record list, certificate + proof-bundle, anchor panels),
then the WASM verifier and OTS anchoring.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~7 still open** (in progress). **Landed:** (a) the
    `HubStatusBadge` leaf (`internal/badge`, all five statuses, distinct label + inline-SVG
    silhouette, golden + mutation + fail-closed) wired with five-status overlay on BOTH `/` and
    `/<domain>/log/`; (b) the DS v2 token CSS shared shell (`internal/web`, `go:embed`, CDN-free, served
    at `/_ds/tokens.css`, linked from `/`); (c) **this iteration** — self-hosted Readex Pro / JetBrains
    Mono webfonts (`go:embed`, 8 latin woff2 subsets + `@font-face` `fonts.css`, served CDN-free at
    `/_ds/fonts/...`, linked from `/`). **Still open:** the realm-index redress (table → Evidence-Ledger
    grid + token classes); badge wired into the dossier / certificate (those surfaces don't exist yet);
    hub dossier (+ categorically-distinct frozen **Exhibit**); paginated record list (`?from=…[&n=…]`,
    no-JS, newest-first) + single-record page (declaration / deletion / unknown schema); certificate of
    inclusion at `/inclusion/{iscc_id}` + **downloadable proof-bundle assembler**
    (`{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` — re-engages the
    oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes + resolved hub key the
    bundle needs); separate Bitcoin-anchor vs comparison-anchor panels; tier-1/tier-2 affordance.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported, re-verified).
- **Last ~12 iterations: ~7 milestone-Verify / ~5 refactor·polish·infra.** Healthy and on the Verify
  bar. The recent arc closed all four M3 criteria (verify-for-me → `/` dashboard → log browser), then
  opened M-UI leaf-first: `HubStatusBadge` → wired into `/` → full five-status overlay on `/` → same
  overlay into the log browser → DS token CSS shared shell → and this iteration the self-hosted webfonts
  (which also resolved an open `normal` cache issue in the same touch). **No polish-streak drift** —
  each leaf is wired into an observable HTTP-seam assertion and linked from `/` in the same iteration.
  Watch-item: M-UI is the largest remaining slice; with the shared shell (tokens + fonts) now complete,
  the next iterations must convert *screens* into wired no-JS HTTP-seam Verify criteria (realm-index
  redress, dossier, record list, certificate) rather than accumulating more shared primitives.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change since the last assessment. The `18160a8..HEAD`
diff touched only `internal/web/*` (the new webfont subtree + tests), `internal/dashboard/dashboard.html`
(the `fonts.css` `<link>`), `cmd/iscc-monitor/main.go` (mounts `web.Handler` at the `web.Prefix`
subtree), and context/loop docs. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all
three triggers (fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart
survival, coverage tracked, structured logs, `/metrics` served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **242 `func Test`** across `cmd/` + `internal/`, **53** `_test.go` files
  (up from 235; the +7 tests are the new webfont assertions in `internal/web/web_test.go` — no new
  package this iteration). Package count unchanged at 16 internal + 2 cmd.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **16 internal packages** —
  `badge, config, corsmw, dashboard, didweb, follower, healthz, logclient, metrics, metricshttp,
  proofserve, registry, store, tiles, tilesserve, web`. Module `github.com/iscc/iscc-monitor`,
  `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation, evaluated inside
  `logclient.CheckConsistency`; growing-pair equivocation builds the RFC-6962 consistency proof from
  the LOCAL mirror → freeze on `true` (ADR-0006). Frozen hubs are evidence-only on clean re-polls.
- **`AcceptCheckpoint` 4-way verdict** (`logclient/accept.go`) returns `(Status, CheckpointInfo,
  VerifiedContext{VKey,Key}, error)`, context populated only on `StatusVerified`; `PollHub` threads
  it into the hub-key cache upsert + `fsckMirror`. Reuse is per-poll only (ADR-0009).
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real
  sb0 checkpoint.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL,
  `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`.
  `store.ListHubs` is a pure read LEFT JOINing `hubs` with `follow_state`; store stays a leaf.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; `AlertFunc func(int64,string)` seam unchanged. The warm-path's second `did.json`
  resolve is a larger design change, not on the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**.
  All proof/dashboard/browser/badge/web tests run against in-process fixtures. Stale `sb1.amlet.id`
  did.json drift (pre-rotation key) captured in tests; not refreshed.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck` via
  `fsckMirror`, `client` re-export), `transparency-dev/formats` (`cmd/notecheck`). **Not wired:**
  `nbd-wtf/opentimestamps` (re-verified: no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — carried forward; no production change. Both Verify criteria are exercised (fsck
root-rebuild WIRED on every verified non-frozen poll via `fsckMirror` → `logclient.RunFsck` over the
read-only `store.SQLiteFetcher`; inclusion cross-check conformance-tested over the real verified
mirror in `internal/follower/inclusion_test.go`). The served proof surface is complete: all three
computed proofs — `inclusion`, `consistency`, `entries` — served from the local mirror, never
re-hitting the hub. **Nothing remains on the M2 Verify bar.**

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward.
- **CORS** — `Access-Control-Allow-Origin: *` on every public GET via the single `corsmw.Handler` wrap.
- **verify-for-me** — `GET /<domain>/log/verify?iscc_id=<id>` returns store-provable `hub_status`,
  accepted `(size, root)`, and a REAL RFC-6962 inclusion result recomputed from the mirror and
  Merkle-verified against the accepted root. Every id-shaped fault → 200 non-verified. Golden +
  mutation non-vacuous.
- **`GET /` dashboard** — `200 text/html` listing **every** realm hub with its status + ADR-0001
  coverage window, status cell rendering all five glossary statuses through `hubStatusBadge` via the
  `overlayStatus` overlay; now links BOTH the shared DS token stylesheet (`/_ds/tokens.css`) and the
  self-hosted webfont stylesheet (`/_ds/fonts.css`) in its `<head>`, with no external CDN URL in the
  body. Golden + mutation-tested.
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing accepted checkpoint `(size, root)` +
  relative links into `entries`/`inclusion`/`consistency`/`verify`/`checkpoint`. Its status cell
  renders through `hubStatusBadge` overlaid with the in-memory verdict (`serveBrowser` →
  `overlayStatus`), consistent with `/`. `POST /` → 405; unpolled hub → 200. Golden + mutation +
  e2e-proven.

**Known limitations (carried forward, off the M3 Verify bar — these become M-UI work):**
- The dossier / certificate / record surfaces don't exist yet (M-UI); the log browser is not yet
  redressed into the Evidence-Ledger grid (it links no token/font CSS yet — only `/` does).
- `inactive` is unreachable through the public store API (no `SetActive`/deactivation writer), so the
  `/` golden covers it via a fixture-deactivated hub at the store seam, but no registry-deactivation
  end-to-end path exists yet.
- No ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces or on `/`/`/verify` —
  not a Verify criterion. (The `/_ds/` static assets DO carry `no-cache` + strong ETag + 304.)

## M-UI — Evidence Ledger frontend
**Status**: **in progress — the five-status badge render is met on `/` and the log browser, and the DS v2 shared shell is now complete (token CSS + self-hosted webfonts, both CDN-free under `/_ds/` and linked from `/`); the remaining screens are open.**
- **Landed so far:**
  - `internal/badge` — a pure, stdlib-only, WASM-shareable `HubStatusBadge` partial (`Render`,
    `Label`, embedded `badge.html`, `PartialName = "hubStatusBadge"`, `Source`) rendering all five
    statuses each with a distinct text label + inline-SVG silhouette, failing closed on unknown/empty
    status. Wired into both `/` and `/<domain>/log/`, each with its own local `StatusSource` interface
    + `overlayStatus` precedence (store `inactive`/`frozen` win; the in-memory verdict
    `unresolvable`/`unverified` overlays a store-`verified` hub). Both surfaces golden-assert
    `data-status`/labels/per-status SVG markers at the HTTP seam, mutation-proven.
  - `internal/web` — a pure stdlib leaf (`bytes`/`embed`/`net/http`/`strings` only; WASM-green) that
    now serves the **whole `/_ds/` static-asset subtree** via one `web.Handler` mounted at `web.Prefix`:
    - **Token CSS** (`tokens.css`, `go:embed`, 0 external CDN refs) at `/_ds/tokens.css`.
    - **Self-hosted webfonts** (**this iteration**, review PASS `e0d8e71`) — 8 embedded latin woff2
      subsets (Readex Pro 300/400/500/600/700 + JetBrains Mono 400/500/700) + the SIL OFL, served at
      `/_ds/fonts/<name>.woff2` (`font/woff2`, traversal-guarded) behind a same-origin `@font-face`
      stylesheet `fonts.css` at `/_ds/fonts.css`. `fonts.css` references exactly the 8 committed files
      (1:1, seam-enforced by `TestFontsCSSReferencesEmbeddedSubsets`).
    - **Cache discipline**: every `/_ds/` asset (CSS + woff2) serves `Cache-Control: no-cache` + strong
      content-ETag + `If-None-Match`→304 (the `tilesserve.writeBlob` shape); non-GET → 405; unknown
      `/_ds/` path → 404. This moved off `immutable` and **closed the open `normal` cache issue**.
    - Both `tokens.css` and `fonts.css` are `<link>`ed from `internal/dashboard/dashboard.html` `<head>`;
      CDN-free + dashboard-link mutation-proven non-vacuous at the seam.
- **Still open on the M-UI Verify bar:** realm-index redress (table → Evidence-Ledger grid + token
  classes — the `var(--font-sans)`/`--font-mono` tokens now resolve to the embedded webfonts);
  threading the token/font CSS into the log browser + future surfaces; badge wired into the dossier /
  certificate (those surfaces don't exist yet); hub dossier (+ categorically-distinct frozen
  **Exhibit**); paginated record list (no-JS, newest-first) + single-record page (declaration /
  deletion / unknown schema); certificate of inclusion at `/inclusion/{iscc_id}` (numbered evidence
  clauses) + **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate;
  `serveVerify` discards the raw checkpoint bytes + resolved hub key the bundle needs); separate
  Bitcoin-anchor vs comparison-anchor panels; tier-1/tier-2 affordance.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned (fetched once at authoring
  time: readex-pro@5.2.11, jetbrains-mono@5.2.8); served bytes are never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM build
target (`syscall/js` not in source — grep clean). The `internal/badge`, `internal/web`, and
`internal/metrics` leaves are WASM-shareable primitives the verifier app will reuse, but the verifier
itself does not exist.

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** (run 27910906519, headSha `e0d8e71`). Local `develop` is in sync with
  `origin/develop` (both at `e0d8e71`).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**, for the self-hosted webfonts + `/_ds`
  revalidate) records `mise run check` green (all 18 packages `ok`, `go vet`/`gofmt -l .` clean), leaf
  purity + WASM-green confirmed (`go list -deps`, `GOOS=js GOARCH=wasm build`), the 8 woff2 binaries
  validated as WOFF2 + 1:1 `fonts.css` reference check, `serveFont` traversal guard probed, CDN-free +
  dashboard-link + `no-cache`/strong-ETag/304 all E2E-confirmed through the real `buildMux`, no
  dep/schema change, gate-integrity scan clean, oracle gate correctly N/A (pure static-asset
  transport). Codex second opinion clean.
- **No open `critical` or `normal` issue.** The previously-open `normal` (`/_ds/tokens.css`
  `immutable`-on-stable-URL) was **resolved and deleted** this iteration (all `/_ds/` assets now
  `no-cache` + strong ETag + 304). **4 open `low`** remain (all loop-skipped, none block DONE):
  `cmd/notecheck` vestigial `out io.Writer` param; hub-status overlay precedence duplicated across
  dashboard/proofserve; mirror write-path leaks tile coordinates into the follower; proofserve repeats
  the `os.ErrNotExist`→404 mapping `tilesserve` already centralised.

## Next Milestone
**M1/M2/M3 all met. The next v1 milestone is M-UI (Evidence Ledger frontend, ADR-0010), in progress.**
CI green, no `critical`/`normal` open, so feature work proceeds.

Convergence-driven order (the shared DS shell — tokens + self-hosted webfonts — is now complete and
linked from `/`; the next iterations must build SCREENS, not more shared primitives):
1. **Realm-index redress** — replace the dashboard `<table>` with the Evidence-Ledger grid + DS token
   classes (`var(--font-sans)`/`--font-mono` now resolve to the embedded webfonts), then thread the
   token/font CSS into the log browser.
2. **Hub dossier** (frozen **Exhibit** — non-dismissable violation kind + detected-at), **paginated
   record list + single record** (declaration / deletion / unknown schema, no-JS), and the
   **certificate of inclusion** at `/inclusion/{iscc_id}` + **downloadable proof-bundle assembler**
   (the slice that re-engages the oracle/conformance gate — `serveVerify` currently discards the raw
   checkpoint bytes + resolved hub key the bundle needs). Add the separate Bitcoin-anchor vs
   comparison-anchor panels + tier-1/tier-2 affordance. Reuse the
   `StatusSource`/`overlayStatus`/`badge.Label` precompute pattern.
3. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands.
