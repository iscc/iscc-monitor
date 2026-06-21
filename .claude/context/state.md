<!-- assessed-at: 18160a8331c414c6b59a8d98e1d6b6604acd5a0d -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress — the shared DS v2 token CSS shell now lands as a pure `internal/web` leaf, served CDN-free and linked from `GET /`.
This iteration stood up `internal/web`: a stdlib-only leaf that `go:embed`s a concatenated, CDN-free
ISCC Design System v2 token stylesheet and serves it at the exact path `/_ds/tokens.css`
(GET→200 `text/css`, non-GET→405), mounted in `buildMux` and linked from the dashboard `<head>`. This
is the first piece of the M-UI shared shell. M1/M2/M3 remain fully met. Remaining v1 work: the rest of
M-UI (self-hosted fonts, dossier, record list, certificate + proof-bundle, anchor panels), then the
WASM verifier and OTS anchoring.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~8 still open** (in progress). **Landed:** (a) the
    `HubStatusBadge` leaf (`internal/badge`, all five statuses, distinct label + inline-SVG
    silhouette, golden + mutation + fail-closed) wired with five-status overlay on BOTH `/` and
    `/<domain>/log/`; (b) **this iteration** — the DS v2 token CSS shared shell (`internal/web`,
    `go:embed`, CDN-free, served at `/_ds/tokens.css`, linked from `/`). **Still open:** self-hosted
    Readex Pro / JetBrains Mono fonts embedded (`go:embed`, no CDN — token CSS lists Arial/Consolas
    fallbacks for now); the realm-index redress (table → Evidence-Ledger grid + token classes); badge
    wired into the dossier / certificate (those surfaces don't exist yet); hub dossier (+
    categorically-distinct frozen **Exhibit**); paginated record list (`?from=…[&n=…]`, no-JS,
    newest-first) + single-record page (declaration / deletion / unknown schema); certificate of
    inclusion at `/inclusion/{iscc_id}` + **downloadable proof-bundle assembler**
    (`{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` — re-engages the
    oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes + resolved hub key the
    bundle needs); separate Bitcoin-anchor vs comparison-anchor panels; tier-1/tier-2 affordance.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported, re-verified).
- **Last ~12 iterations: ~7 milestone-Verify / ~5 refactor·polish·infra.** Healthy and on the Verify
  bar. The recent arc closed all four M3 criteria (verify-for-me → `/` dashboard → log browser), then
  opened M-UI leaf-first: `HubStatusBadge` → wired into `/` → full five-status overlay on `/` → same
  overlay into the log browser → and this iteration the DS token CSS shared shell. **No polish-streak
  drift** — each leaf is wired into an observable HTTP-seam assertion in the same or next iteration.
  Watch-item: M-UI is the largest remaining slice; keep converting screens into wired no-JS HTTP-seam
  Verify criteria (fonts, dossier, record list, certificate) rather than accumulating unwired
  primitives.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change since the last assessment. The `47cc8c2..HEAD`
diff touched only `internal/web/*` (new DS-token leaf + tests), `internal/dashboard/*` (the `<link>` +
its test), `cmd/iscc-monitor/main.go` (mounts `web.Handler` at `web.TokensPath`), `CLAUDE.md`, and
context/loop docs. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **235 `func Test`** across `cmd/` + `internal/`, **53** `_test.go` files
  (up from 231 / 52 — the +1 file + tests are the new `internal/web` package + the dashboard-link test).
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
  `overlayStatus` overlay; now also links the shared DS token stylesheet (`/_ds/tokens.css`) in its
  `<head>` with no external CDN URL in the body. Golden + mutation-tested.
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing accepted checkpoint `(size, root)` +
  relative links into `entries`/`inclusion`/`consistency`/`verify`/`checkpoint`. Its status cell
  renders through `hubStatusBadge` overlaid with the in-memory verdict (`serveBrowser` →
  `overlayStatus`), consistent with `/`. `POST /` → 405; unpolled hub → 200. Golden + mutation +
  e2e-proven.

**Known limitations (carried forward, off the M3 Verify bar — these become M-UI work):**
- The dossier / certificate / record surfaces don't exist yet (M-UI); the log browser is not yet
  redressed into the Evidence-Ledger grid (it links no token CSS yet — only `/` does).
- `inactive` is unreachable through the public store API (no `SetActive`/deactivation writer), so the
  `/` golden covers it via a fixture-deactivated hub at the store seam, but no registry-deactivation
  end-to-end path exists yet.
- No ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces or on `/`/`/verify` —
  not a Verify criterion.

## M-UI — Evidence Ledger frontend
**Status**: **in progress — the five-status badge render is met on `/` and the log browser, and the DS v2 token CSS shared shell now ships; the remaining screens + fonts are open.**
- **Landed so far:**
  - `internal/badge` — a pure, stdlib-only, WASM-shareable `HubStatusBadge` partial (`Render`,
    `Label`, embedded `badge.html`, `PartialName = "hubStatusBadge"`, `Source`) rendering all five
    statuses each with a distinct text label + inline-SVG silhouette, failing closed on unknown/empty
    status. Wired into both `/` and `/<domain>/log/`, each with its own local `StatusSource` interface
    + `overlayStatus` precedence (store `inactive`/`frozen` win; the in-memory verdict
    `unresolvable`/`unverified` overlays a store-`verified` hub). Both surfaces golden-assert
    `data-status`/labels/per-status SVG markers at the HTTP seam, mutation-proven.
  - `internal/web` (**this iteration**, review PASS_WITH_NOTES `18160a8`) — a pure stdlib leaf
    (`bytes`/`embed`/`net/http` only; `go list -deps` internal closure is just `internal/web`,
    `GOOS=js GOARCH=wasm build` OK) that `go:embed`s a concatenated CDN-free DS v2 token stylesheet
    (`tokens.css`, 0 external `https://`/CDN refs) and serves it at the exact path `/_ds/tokens.css`
    (GET→200 `text/css`; non-GET→405). Mounted via `mux.Handle(web.TokensPath, web.Handler())` and
    linked from `internal/dashboard/dashboard.html` `<head>`. CDN-free + dashboard-link both
    mutation-proven non-vacuous at the seam.
- **Still open on the M-UI Verify bar:** self-hosted Readex Pro / JetBrains Mono woff2 fonts embedded
  (`go:embed`, no CDN — token CSS uses Arial/Consolas fallbacks for now); realm-index redress (table →
  Evidence-Ledger grid + token classes); log browser + future surfaces linking the token CSS; badge
  wired into the dossier / certificate (those surfaces don't exist yet); hub dossier (+
  categorically-distinct frozen **Exhibit**); paginated record list (no-JS, newest-first) +
  single-record page (declaration / deletion / unknown schema); certificate of inclusion at
  `/inclusion/{iscc_id}` (numbered evidence clauses) + **downloadable proof-bundle assembler**
  (re-engages the oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes + resolved
  hub key the bundle needs); separate Bitcoin-anchor vs comparison-anchor panels; tier-1/tier-2
  affordance.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD).

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
  `conclusion: success`** (run 27910434338, headSha `18160a8`). Local `develop` is in sync with
  `origin/develop` (both at `18160a8`).
- Latest `review` handoff (2026-06-21, **PASS_WITH_NOTES / CONTINUE**, for the DS token CSS leaf)
  records `mise run check` green (all 18 packages `ok`, `go vet`/`gofmt -l .` clean), leaf purity +
  WASM-green confirmed (`go list -deps`, `GOOS=js GOARCH=wasm build`), CDN-free + dashboard-link
  mutation-proven (revert FAILs, restore PASS), no new deps/schema change, gate-integrity scan clean,
  oracle gate correctly N/A (pure static-asset transport). Codex second opinion clean on `ffa6049`.
- **No open `critical` or `normal` issue blocks progress, but one `normal` is now OPEN** (a DONE
  blocker once the rest converges; does not block current feature work): `/_ds/tokens.css` uses
  `immutable` Cache-Control on a stable overwrite-in-place URL (Codex [P2], reviewer-confirmed against
  the project's own `internal/tilesserve` cache discipline). Cosmetic (stale tokens after a redeploy;
  no correctness/security/trust-root impact). Also **1 open `low`** (loop-skipped): `cmd/notecheck`'s
  vestigial `out io.Writer` param.

## Next Milestone
**M1/M2/M3 all met. The next v1 milestone is M-UI (Evidence Ledger frontend, ADR-0010), in progress.**
CI green, no `critical` open, so feature work proceeds.

Convergence-driven order (the shared DS token shell now exists and is linked from `/`; extend it):
1. **Self-hosted fonts** (`go:embed` Readex Pro / JetBrains Mono woff2 under `/_ds/fonts/...` + a
   `@font-face` stylesheet, no CDN). **When that lands, fold in the open `normal` cache fix for BOTH
   `/_ds/...` assets** (switch to the `tilesserve` `no-cache` + strong-ETag + 304 pattern, or a
   build-fingerprinted path) since it touches `internal/web` anyway, and narrow the CDN-free `url(`
   ban to "no external `url(`" so self-hosted `src: url("/_ds/fonts/...")` passes.
2. **Realm-index redress** (table → Evidence-Ledger grid + token classes), then thread the token CSS
   into the log browser, and build the **hub dossier** (frozen Exhibit), **paginated record list +
   single record**, and the **certificate + downloadable proof-bundle assembler** (the slice that
   re-engages the oracle/conformance gate — `serveVerify` currently discards the raw checkpoint bytes
   + resolved hub key the bundle needs). Reuse the `StatusSource`/`overlayStatus`/`badge.Label`
   precompute pattern.
3. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands.
