<!-- assessed-at: 47cc8c2fa53ac5e97075684a445fb5729c1a809b -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress — the five-status badge now renders honestly on BOTH `/` and the per-hub log browser.
This iteration threaded the `StatusSource` interface + `overlayStatus` precedence into
`proofserve.serveBrowser`, so `GET /<domain>/log/` now renders its hub status through the same
five-status `hubStatusBadge` partial overlaid with the in-memory live verdict — the taxonomy is now
consistent across both SSR surfaces (`/` and `/<domain>/log/`). M1/M2/M3 remain fully met. Remaining
v1 work: the rest of M-UI (DS tokens/fonts, dossier, record list, certificate + proof-bundle, anchor
panels), then the WASM verifier and OTS anchoring.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~8 still open** (in progress). **Landed:** the
    `HubStatusBadge` leaf (`internal/badge`, all five statuses, distinct label + inline-SVG
    silhouette, golden + mutation + fail-closed) AND its **five-status overlay wiring on BOTH `/` and
    `/<domain>/log/`** — `overlayStatus` overlays the in-memory verdict (`unresolvable`/`unverified`)
    on top of the store-provable subset, golden- + mutation-tested at the HTTP seam on each surface.
    The per-surface five-status render criterion is now **met for `/` and the log browser**. **Still
    open:** DS v2 tokens + self-hosted fonts embedded (`go:embed`, no CDN); badge wired into the
    dossier / certificate; hub dossier (+ categorically-distinct frozen **Exhibit**); paginated
    record list (`?from=…[&n=…]`, no-JS, newest-first) + single-record page (declaration / deletion /
    unknown schema); certificate of inclusion at `/inclusion/{iscc_id}` + **downloadable proof-bundle
    assembler** (`{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` — re-engages
    the oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes + resolved hub key
    the bundle needs); separate Bitcoin-anchor vs comparison-anchor panels; tier-1/tier-2 affordance.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported, re-verified).
- **Last ~10 iterations: ~6 milestone-Verify / ~4 refactor·polish·infra.** Healthy and on the Verify
  bar. The recent arc closed all four M3 criteria back-to-back (verify-for-me → `/` dashboard → log
  browser), opened M-UI leaf-first (`HubStatusBadge`), wired it into `/`, made the full five-status
  taxonomy render honestly on `/` via the in-memory overlay, and this iteration extended that same
  overlay into the log browser for cross-surface consistency. **No polish-streak drift** — each leaf
  is wired into an observable HTTP-seam assertion in the same or next iteration. Watch-item: M-UI is
  the largest remaining slice; keep converting screens into wired no-JS HTTP-seam Verify criteria
  (dossier, record list, certificate) rather than accumulating unwired primitives.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change since the last assessment. The
`6cb9880..HEAD` diff touched only `internal/proofserve/*` (log-browser overlay + tests),
`cmd/iscc-monitor/main.go` (passes `StatusSource` into the proofserve handler), `CLAUDE.md`, and
context docs. All Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **231 `func Test`** across `cmd/` + `internal/`, **52** `_test.go` files
  (up from 230 — the +1 is the new log-browser in-memory-status overlay test in `browser_test.go`).
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **15 internal packages** —
  `badge, config, corsmw, dashboard, didweb, follower, healthz, logclient, metrics, metricshttp,
  proofserve, registry, store, tiles, tilesserve`. Module `github.com/iscc/iscc-monitor`, `go 1.24.0`
  (no `toolchain` line).
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
  `store.ListHubs` is a pure read LEFT JOINing `hubs` with `follow_state`; store stays a leaf (no
  `net/http`/`logclient`/`metrics` — re-verified `go list -deps` clean).
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; `AlertFunc func(int64,string)` seam unchanged. The warm-path's second `did.json`
  resolve is a larger design change, not on the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**.
  All proof/dashboard/browser/badge tests run against in-process fixtures. Stale `sb1.amlet.id`
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
  `overlayStatus` overlay. Golden + mutation-tested.
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing accepted checkpoint `(size, root)` +
  relative links into `entries`/`inclusion`/`consistency`/`verify`/`checkpoint`. Its status cell now
  also renders through `hubStatusBadge` overlaid with the in-memory verdict (`serveBrowser` →
  `overlayStatus`), so the five-status taxonomy is consistent with `/`. `POST /` → 405; unpolled hub
  → 200. Golden + mutation + e2e-proven.

**Known limitations (carried forward, off the M3 Verify bar — these become M-UI work):**
- The overlay is now threaded into both `/` and the log browser; the dossier / certificate surfaces
  don't exist yet (M-UI). The "log browser still on store-only subset" limitation from the prior
  assessment is **RESOLVED** this iteration.
- `inactive` is unreachable through the public store API (no `SetActive`/deactivation writer), so the
  `/` golden covers it via a fixture-deactivated hub at the store seam, but no registry-deactivation
  end-to-end path exists yet.
- No ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces or on `/`/`/verify` —
  not a Verify criterion.

## M-UI — Evidence Ledger frontend
**Status**: **in progress — the five-status badge render is met on `/` and the log browser; the remaining screens + DS are open.**
- **Landed (`internal/badge` + `internal/dashboard` + `internal/proofserve` + `internal/metrics`,
  review PASS `47cc8c2`):** a pure, stdlib-only, WASM-shareable `HubStatusBadge` partial (`Render`,
  `Label`, embedded `badge.html`, `PartialName = "hubStatusBadge"`, `Source`) rendering all five
  statuses each with a distinct text label + inline-SVG silhouette, failing closed on unknown/empty
  status. It is wired into both `/` (`internal/dashboard`) and `/<domain>/log/`
  (`proofserve.serveBrowser`), each defining its own tiny local `StatusSource` interface +
  `overlayStatus` precedence mirror — store `inactive`/`frozen` win; the in-memory verdict
  (`unresolvable`/`unverified`) overlays a store-`verified` hub. Both surfaces golden-assert
  `data-status`/labels/per-status SVG markers at the HTTP seam and are mutation-proven. `proofserve`
  imports neither `internal/metrics` nor `internal/dashboard` (re-verified `go list -deps` clean);
  `internal/store` stays a leaf.
- **Still open on the M-UI Verify bar:** DS v2 tokens + self-hosted Readex Pro / JetBrains Mono fonts
  embedded (`go:embed`, no CDN); badge wired into the dossier / certificate (those surfaces don't
  exist yet); hub dossier (+ categorically-distinct frozen **Exhibit**); paginated record list (no-JS,
  newest-first) + single-record page (declaration / deletion / unknown schema); certificate of
  inclusion at `/inclusion/{iscc_id}` (numbered evidence clauses) + **downloadable proof-bundle
  assembler** (re-engages the oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes
  + resolved hub key the bundle needs); separate Bitcoin-anchor vs comparison-anchor panels;
  tier-1/tier-2 affordance.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD).

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM build
target (`syscall/js` not in source — grep clean). The `internal/badge` and `internal/metrics` leaves
are WASM-shareable primitives that the verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** (run 27909898002). Local `develop` is in sync with `origin/develop` (both
  at `47cc8c2`).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**, for the log-browser five-status overlay)
  records `mise run check` green (all 17 packages `ok`, `go vet`/`gofmt -l .` clean), leaf purity
  confirmed (`go list -deps`), golden + mutation-proven (independent revert FAILs, restored PASS), no
  new deps / schema change, gate-integrity scan clean, oracle gate correctly N/A (pure HTML
  composition + in-memory status overlay). Codex second opinion clean (exit 0), no findings.
- **No open `critical` or `normal` issue.** **Open `low` issue (loop-skipped, not a DONE blocker):** 1
  in `issues.md` — `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2/M3 all met. The next v1 milestone is M-UI (Evidence Ledger frontend, ADR-0010), in progress.**
CI green, no `critical`/`normal` open, so feature work proceeds.

Convergence-driven order (the five-status render is now consistent across `/` and the log browser;
carry the pattern into the new screens):
1. **DS v2 tokens + self-hosted fonts embedded** (`go:embed`, no CDN) — the shared shell every
   remaining surface needs.
2. **Hub dossier** (frozen Exhibit), then **paginated record list + single record**, and the
   **certificate + downloadable proof-bundle assembler** (the slice that re-engages the
   oracle/conformance gate — `serveVerify` currently discards the raw checkpoint bytes + resolved hub
   key the bundle needs). Reuse the `StatusSource`/`overlayStatus`/`badge.Label`-precompute pattern.
3. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands.
