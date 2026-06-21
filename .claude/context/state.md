<!-- assessed-at: dbc14499362fa54cc3cb3a4260e508e1bdb46d97 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress. The shared DS v2 shell (token CSS + self-hosted webfonts under `/_ds/`) and two dressed screens (`/` realm index, `/<domain>/log/` log browser) are met. The hub-dossier screen (`GET /<domain>`) has LANDED but is NOT verified — the latest `review` verdict is **NEEDS_WORK**: the new bare-domain mount introduces a startup-panic crash surface (open `normal` issue).

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met and CI-green. M-UI is mid-flight: the badge, DS shell, index, and log browser are dressed and verified; the hub dossier was added this iteration but its advance introduced a `normal`-priority defect (reserved-domain mount panic) and was returned NEEDS_WORK — it must be fixed before the dossier counts as met. The dossier commits are NOT pushed, so CI still reflects the previous PASS at `173f718`.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~5 still open.** **Met:** five-status `HubStatusBadge`
    (`internal/badge`, all five statuses, distinct label + inline-SVG silhouette, golden + mutation +
    fail-closed) wired into `/` and `/<domain>/log/`; the DS v2 shared shell (`/_ds/tokens.css` +
    self-hosted Readex Pro / JetBrains Mono webfonts, CDN-free, `no-cache`+ETag+304); the `/` realm
    index Evidence-Ledger CSS-grid redress; the `/<domain>/log/` log-browser Evidence-Ledger card
    redress. **Landed-but-NOT-met (NEEDS_WORK):** the **hub dossier** (`GET /<domain>`,
    `internal/dossier`) — the page itself passed all its Verify criteria (coverage honesty, badge
    overlay, no-JS/no-CDN DS shell), but the advance introduced a startup-panic crash surface (a realm
    `Domain` colliding with `/metrics`/`/healthz` panics `buildMux`); review returned NEEDS_WORK and
    filed it as a `normal` issue. The dossier's frozen **Exhibit** sub-step (needs a new
    `store.ListViolations` read) is not yet built. **Still open:** dossier fix + Exhibit; **paginated
    record list** (`?from=…[&n=…]`, no-JS, newest-first) + **single-record page** (declaration /
    deletion / unknown schema); **certificate of inclusion** at `/inclusion/{iscc_id}` (numbered
    evidence clauses) + **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate;
    `serveVerify` discards the raw checkpoint bytes + resolved hub key the bundle needs); separate
    **Bitcoin-anchor vs comparison-anchor** panels + tier-1/tier-2 affordance.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod` or source,
    re-verified: grep rc=1 across `go.mod` + `cmd/` + `internal/`).
- **Last ~10 iterations: ~6 milestone-Verify / ~4 refactor·polish·shell, then the most recent advance
  returned NEEDS_WORK.** The arc closed all four M3 criteria, then opened M-UI leaf-first (badge → DS
  token CSS → webfonts → `/` grid redress → log-browser redress → hub dossier). No polish-streak drift;
  each step targets a named M-UI Verify criterion. **Watch-item:** the dossier advance shipped a
  startup-panic regression (caught by review, not by the gate — it is a misconfig crash, not a test
  failure), so the immediate next advance is the fix, not new screen work. M-UI remains the largest
  remaining slice; the proof-bundle assembler is the one screen that re-engages the crypto/oracle gate.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; no production change since the last assessment. The
`173f718..HEAD` diff touched only the new `internal/dossier` package + `cmd/iscc-monitor/main.go`
(dossier mount) + `main_test.go` + context/learnings docs — no M1 source. All M1 Verify criteria remain
satisfied: `origin`/`vkey` golden, all three triggers (fork/shrink/equivocation) golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP.
- **Test totals at HEAD**: **250 `func Test`** across `cmd/` + `internal/`, **54** `_test.go` files
  (+7 in the new `internal/dossier/handler_test.go`). Package count now **17 internal + 2 cmd**.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **17 internal packages** —
  `badge, config, corsmw, dashboard, didweb, dossier, follower, healthz, logclient, metrics,
  metricshttp, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- All three triggers WIRED + golden-tested inside `logclient.CheckConsistency`; `AcceptCheckpoint`
  4-way verdict (`logclient/accept.go`) threads `VerifiedContext` into the hub-key cache upsert +
  `fsckMirror`. `cmd/notecheck` is the fully-independent signature-parity oracle, shelled out in CI.
  `store/*.go` uses `modernc.org/sqlite` with ADR-0005/0007 single-writer discipline.
- **Missing (M1 connective tissue, off the Verify bar):** real alert transport (`alertFunc` is a WARN
  `slog` emit); warm-path second `did.json` resolve (a larger design change).
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — no tiles, entry bundles, or did.json. All
  proof/dashboard/browser/badge/web/dossier tests run against in-process fixtures. Stale `sb1.amlet.id`
  did.json drift (pre-rotation key) captured in tests; not refreshed.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`,
  `client`), `transparency-dev/formats` (`cmd/notecheck`). **Not wired:** `nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward; no production change. Both Verify criteria exercised: fsck
root-rebuild WIRED on every verified non-frozen poll (`fsckMirror` → `logclient.RunFsck` over the
read-only `store.SQLiteFetcher`); inclusion cross-check conformance-tested over the real verified mirror
(`internal/follower/inclusion_test.go`). All three computed proofs — `inclusion`, `consistency`,
`entries` — served from the local mirror, never re-hitting the hub. Nothing remains on the M2 Verify bar.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward.
- **CORS** — `Access-Control-Allow-Origin: *` on every public GET via the single `corsmw.Handler` wrap.
- **verify-for-me** — `GET /<domain>/log/verify?iscc_id=<id>` returns store-provable `hub_status`,
  accepted `(size, root)`, and a real RFC-6962 inclusion result recomputed from the mirror and
  Merkle-verified against the accepted root. Every id-shaped fault → 200 non-verified. Golden + mutation.
- **`GET /` dashboard** — `200 text/html` listing every realm hub with status + coverage window, all
  five glossary statuses via `hubStatusBadge`, Evidence-Ledger CSS-grid (no `<table>`), links both
  `/_ds/tokens.css` and `/_ds/fonts.css`, no CDN URL. Golden + mutation.
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing accepted `(size, root)` + relative
  links into `entries`/`inclusion`/`consistency`/`verify`/`checkpoint`, status via badge overlay,
  Evidence-Ledger card redress. Golden + mutation + e2e.

**Known limitations (off the M3 Verify bar):** `inactive` is unreachable through the public store API
(no `SetActive`/deactivation writer); no ETag/Cache-Control on the size-dependent proof surfaces (the
`/_ds/` static assets DO carry `no-cache` + strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress.** The badge render is met on `/` and the log browser; the DS v2 shared shell
is complete; the `/` realm index and `/<domain>/log/` log browser are dressed and verified. The **hub
dossier landed this iteration but is NOT met** — review returned **NEEDS_WORK** for a startup-panic
regression in the new mount.
- **Met:**
  - `internal/badge` — pure stdlib-only `HubStatusBadge` partial (all five statuses, distinct label +
    inline-SVG silhouette, fail-closed), wired into `/` and `/<domain>/log/` with per-surface
    `overlayStatus` precedence; golden + mutation at the HTTP seam.
  - `internal/web` — `/_ds/` static-asset subtree (token CSS + 8 self-hosted woff2 subsets +
    `@font-face`), CDN-free, `no-cache` + strong content-ETag + 304, traversal-guarded; non-GET → 405,
    unknown `/_ds/` → 404.
  - `/` realm index — Evidence-Ledger CSS-grid redress (no `<table>`, DS tokens, coverage-honesty
    footnote, `min-width:0` ellipsis fix). Mutation-confirmed non-vacuous.
  - `/<domain>/log/` log browser — Evidence-Ledger card redress (token/font CSS linked, no `<table>`,
    no CDN, no-JS, no-checkpoint state). `TestBrowserLinksTokensNoCDN` added, mutation-confirmed.
- **Landed-but-NOT-met (`internal/dossier`, review verdict NEEDS_WORK):**
  - `internal/dossier` — `Handler(st, hubID, statuses)` serving `GET /<domain>`: a per-hub
    Evidence-Ledger page (masthead + ledger card, coverage window with ADR-0001 honesty, five-status
    badge overlay, no-JS/no-CDN DS shell). 7 tests; the page's own Verify criteria all pass and
    `mise run check` was green at the advance.
  - **BLOCKING DEFECT (open `normal` issue):** the dossier mounts `mux.Handle("/"+r.Domain, …)` inside
    `mirrorHandler` (`cmd/iscc-monitor/main.go:205`), which `buildMux` runs BEFORE registering the
    built-in exact routes `/metrics` (172) / `/healthz` (173) / `web.Prefix` (174). `registry.Parse`
    accepts any non-URL bare token as a `Domain`, so a realm line `metrics`/`healthz` makes the dossier
    register that exact path first → the later built-in `mux.Handle("/metrics", …)` panics
    (`pattern "/metrics" … conflicts`) and the monitor fails to start. Re-verified: NO reserved-name
    guard exists in `main.go` or `internal/registry/registry.go`. Root-cause fix (reject/reserve the
    name before mounting, covering the empty/`/`-colliding case too) is the immediate next advance.
- **Still open on the M-UI Verify bar:** the dossier fix + the frozen **Exhibit** (needs a new
  `store.ListViolations(hubID)` read over the `violations` table — only `RecordViolation` exists today —
  + non-dismissable Exhibit markup, ADR-0006); **paginated record list** (`?from=…[&n=…]`, no-JS,
  newest-first over `iscc_index`) + **single-record page** (declaration / deletion / unknown schema);
  **certificate of inclusion** at `/inclusion/{iscc_id}` (numbered evidence clauses) + **downloadable
  proof-bundle assembler** (re-engages the oracle/conformance gate — `serveVerify` discards the raw
  checkpoint bytes + resolved hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor**
  panels + tier-1/tier-2 affordance.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod` or source (grep
rc=1 across `go.mod` + `cmd/` + `internal/`); no `internal/proof` package (`ls` → no such directory); no
WASM build target (`syscall/js` not in source — grep rc=1). The `internal/badge`, `internal/web`, and
`internal/metrics` leaves are WASM-shareable primitives the verifier app will reuse, but the verifier
itself does not exist.

## Quality gates
**Status**: **green at the last PASS (`173f718`), but the dossier commits at HEAD are NOT pushed and NOT
CI-confirmed; the latest `review` verdict is NEEDS_WORK with one open `normal` issue.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest run on `develop`: `conclusion: success` at headSha `173f718`** (run 27911659777) — this is
  NOT HEAD (`dbc1449`). The four dossier-related commits (`e547d6d`, `a0da700`, `dbc1449`, and the
  prior `update-state`) are unpushed because the last review was NEEDS_WORK; CI has not seen them.
- The dossier *advance* reported `mise run check` green at the advance (review independently re-ran:
  all 19 packages `ok`, `go vet`/`gofmt -l .` clean, oracle gate correctly N/A for pure HTML render),
  but the **review verdict is NEEDS_WORK** because of the runtime startup-panic crash surface (a class
  of failure the gate does not catch — valid Go that crashes on a specific realm config).
- **Open issues: 1 `normal`, 4 `low`.** The `normal` (reserved-domain mount panic) BLOCKS DONE and is
  the next advance's target. The 4 `low` are loop-skipped (notecheck `out` param; overlay precedence
  now duplicated 3x across dashboard/proofserve/dossier; mirror write-path tile-coord leak; proofserve
  `os.ErrNotExist`→404 duplication).

## Next Milestone
**M1/M2/M3 met. The active milestone is M-UI (Evidence Ledger frontend, ADR-0010). The immediate next
step is the dossier fix — NOT new screen work.**

1. **Fix the reserved/empty-domain mount panic** (open `normal` issue, review-blocking). Reject or skip
   a `Domain` equal to a reserved mount name (`metrics`, `healthz`, the `web.Prefix` segment) and the
   empty/`/`-colliding case — prefer failing `registerHubs`/`registry.Parse` loudly over a silent skip —
   with a `buildMux` test driving a reserved name. This unblocks the dossier as a met M-UI screen.
2. **Frozen Exhibit** on the dossier — add `store.ListViolations(hubID)` (read over the `violations`
   table) + the categorically-distinct, non-dismissable Exhibit markup (violation kind + detected-at,
   ADR-0006).
3. **Remaining M-UI SSR screens** — paginated record list + single record (declaration / deletion /
   unknown schema), then the **certificate of inclusion** + **downloadable proof-bundle assembler** (the
   slice that re-engages the oracle/conformance gate), plus the separate Bitcoin-anchor vs
   comparison-anchor panels + tier-1/tier-2 affordance.
4. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
5. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands.
