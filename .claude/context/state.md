<!-- assessed-at: a9c29faed3d8a6b0010093cb40ff2235a2cfb35d -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress. The paginated record list slice
landed but the latest `review` verdict is **NEEDS_WORK**: three confirmed defects in
pagination/coverage (no `LastSize` cap, page-size clamp bypassed on overflow, seq 0 unreachable). Two
new `normal` issues are open. M1/M2/M3 remain fully met and CI-green.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: the
badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, and the frozen Exhibit are dressed
and verified. The newest slice — the paginated `/<domain>/log/records` list — is functional and
tested but shipped three real defects (review NEEDS_WORK, loop CONTINUE); the next iteration is a fix
slice. WASM and OTS are not started.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~3 still open** (the record-list criterion is partially landed
    but NEEDS_WORK). **Met:** five-status `HubStatusBadge` (`internal/badge`, all five statuses,
    distinct label + inline-SVG silhouette, golden + mutation + fail-closed) wired into `/`,
    `/<domain>/log/`, and the dossier; the DS v2 shared shell (`/_ds/` tokens.css + self-hosted Readex
    Pro / JetBrains Mono webfonts, CDN-free, `no-cache`+ETag+304); the `/` realm index Evidence-Ledger
    CSS-grid redress; the `/<domain>/log/` log-browser Evidence-Ledger card redress; the **hub
    dossier** (`GET /<domain>`, `internal/dossier.Handler`); the **frozen Exhibit**
    (`store.ListViolations` → non-dismissable Exhibit panel, gated on `status == "frozen"`).
    **Landed-but-NEEDS_WORK:** the **paginated record list** (`GET /<domain>/log/records?from=…[&n=…]`,
    `serveRecords` + `store.ListRecords`) — no-JS newest-first list, DS shell, empty state, badge
    overlay all present and non-vacuously tested, BUT three defects defeat stated controls (see below);
    it does NOT yet meet its Verify criterion. **Still open:** the record-list fixes; the
    **single-record page** (declaration / deletion / unknown schema); the **certificate of inclusion**
    at `/inclusion/{iscc_id}` (numbered evidence clauses) + the **downloadable proof-bundle assembler**
    (re-engages the oracle/conformance gate); separate **Bitcoin-anchor vs comparison-anchor** panels +
    tier-1/tier-2 affordance.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source).
- **Last ~10 iterations: ~7 milestone-Verify / ~3 refactor·polish·shell + 2 defect-fix.** The arc
  closed all four M3 criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid →
  log-browser redress → hub dossier → frozen Exhibit → record list). The record-list slice is the
  second this arc to land NEEDS_WORK (after the dossier startup-panic regression, fixed the next
  advance). No polish-streak drift — each step targets a named M-UI Verify criterion. M-UI remains the
  largest remaining slice; the proof-bundle assembler is the one screen that re-engages the crypto/
  oracle gate.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `452d23c..HEAD` diff touched only `internal/proofserve/*`
(record list), `internal/store/iscc_index*` (`ListRecords`), `cmd/iscc-monitor/main*` (the `/records`
mount), and context docs — no M1 source. All M1 Verify criteria remain satisfied: `origin`/`vkey`
golden, all three triggers (fork/shrink/equivocation) golden-tested end-to-end with freeze +
alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **266 `func Test`** across **55** `_test.go` files (was 256/54 — +10 from
  the record-list slice: `ListRecords` store tests + `serveRecords` HTTP-seam tests + `/records` mount
  test). Package count **17 internal + 2 cmd**.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **17 internal packages** —
  `badge, config, corsmw, dashboard, didweb, dossier, follower, healthz, logclient, metrics,
  metricshttp, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- All three triggers WIRED + golden-tested inside `logclient.CheckConsistency`; `AcceptCheckpoint`
  4-way verdict threads `VerifiedContext` into the hub-key cache upsert + `fsckMirror`. `cmd/notecheck`
  is the fully-independent signature-parity oracle, shelled out in CI. `store/*.go` uses
  `modernc.org/sqlite` with ADR-0005/0007 single-writer discipline.
- **Missing (M1 connective tissue, off the Verify bar):** real alert transport (`alertFunc` is a WARN
  `slog` emit); warm-path second `did.json` resolve (a larger design change).
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — no tiles, entry bundles, or did.json. All
  proof/dashboard/browser/badge/web/dossier/records tests run against in-process fixtures. Stale
  `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests; not refreshed.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`,
  `client`), `transparency-dev/formats` (`cmd/notecheck`). **Not wired:** `nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward; no production change to M2 source. Both Verify criteria
exercised: fsck root-rebuild WIRED on every verified non-frozen poll; inclusion cross-check
conformance-tested over the real verified mirror. All three computed proofs — `inclusion`,
`consistency`, `entries` — served from the local mirror, never re-hitting the hub.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward. CORS on every public GET; verify-for-me
at `GET /<domain>/log/verify?iscc_id=<id>` (store-provable status + accepted `(size, root)` +
Merkle-verified inclusion, every id-fault → 200 non-verified); `GET /` realm-index dashboard (all five
badge statuses, Evidence-Ledger grid, no CDN); `GET /<domain>/log/` log browser (accepted `(size,
root)` + proof links, badge overlay). All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` is unreachable through the public store API
(no `SetActive`/deactivation writer); no ETag/Cache-Control on the size-dependent proof surfaces (the
`/_ds/` static assets DO carry `no-cache` + strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress.** Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, and
the frozen Exhibit are all met and verified. The newest slice — the paginated record list — is
**landed but NEEDS_WORK** (latest `review` verdict; 2 open `normal` issues).
- **Met:** `internal/badge` five-status `Render` (golden + mutation, fail-closed); `internal/web`
  `/_ds/` static-asset subtree (token CSS + self-hosted woff2 + `@font-face`, CDN-free, `no-cache` +
  strong content-ETag + 304, traversal-guarded); the `/` realm-index Evidence-Ledger CSS-grid redress;
  the `/<domain>/log/` log-browser card redress; `internal/dossier.Handler` serving `GET /<domain>`
  (Evidence-Ledger per-hub page, coverage honesty, badge overlay, reserved/empty-domain mount guard);
  the **frozen Exhibit** (`store.ListViolations` newest-first read → non-dismissable Exhibit panel,
  gated on `status == "frozen"`).
- **Landed but NEEDS_WORK — paginated record list** (`GET /<domain>/log/records`,
  `serveRecords` at `internal/proofserve/handler.go:671`, `store.ListRecords` at
  `internal/store/iscc_index.go:98`): a no-JS newest-first list with DS shell, CDN-free body,
  overlay-status badge, and empty state, all non-vacuously tested (`DESC→ASC` mutation FAILS, reverted).
  But the latest `review` (Codex-corroborated, both reviewer-confirmed against the code) found **three
  real defects** that defeat stated controls and keep it off its Verify bar:
  1. **No `LastSize` cap** — `ListRecords` lists every `iscc_index` row with no `seq < LastSize`
     ceiling, so a frozen/violation hub (where ingest wrote projections above `LastSize` before the
     freeze) shows unaccepted leaves as accepted; their `entries?index=<seq>` links then 404. Every
     OTHER record route caps at `>= LastSize`; this one omits it. (`normal`, ADR-0001 coverage honesty.)
  2. **Page-size clamp bypassed on overflow** — `handler.go:694` does `pageSize = int(n)` BEFORE the
     `> maxPageSize` check; a huge `n` wraps `int(n)` negative, the `> 200` check misses it, and modernc
     SQLite reads a negative `LIMIT` as unlimited → whole-index render. `parseUint` also wraps silently.
     (`normal`, anti-DoS clamp defeated.)
  3. **seq 0 unreachable** — `from == 0` is overloaded as the "start at newest" sentinel
     (`iscc_index.go:108` `if from > 0`), so the older-link chain (`OlderFrom = oldest-1`, gated on
     `oldest > 0`) jumps back to the newest page instead of reaching seq 0. (`normal`, same root as #2.)
- **Still open on the M-UI Verify bar:** the record-list fix slice (above); the **single-record page**
  (declaration / deletion / unknown schema), with each `/records` row re-pointed from `entries?index=`
  to it; the **certificate of inclusion** at `/inclusion/{iscc_id}` (numbered evidence clauses) + the
  **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate — `serveVerify`
  discards the raw checkpoint bytes + resolved hub key the bundle needs); separate **Bitcoin-anchor vs
  comparison-anchor** panels + tier-1/tier-2 affordance.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
source; no `internal/proof` package; no WASM build target (`syscall/js` not in source). The
`internal/badge`, `internal/web`, and `internal/metrics` leaves are WASM-shareable primitives the
verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **mixed — `mise run check` reported green at HEAD by the latest `review`, but the latest
`review` VERDICT is NEEDS_WORK (3 confirmed defects), and the record-list commits are UNPUSHED so CI
has not confirmed HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. The latest `review` recorded `mise run check` green at HEAD (build + vet + all 20
  packages `ok`, `gofmt -l .` empty); the three defects are correctness/coverage-honesty failures, not
  build/vet/test failures.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest run on `develop`: `conclusion: success` at headSha `452d23c` (run 27913131414)** — but that
  is `origin/develop`, NOT HEAD. HEAD (`a9c29fa`) and the 3 record-list commits are **unpushed**, so CI
  has NOT exercised the record-list code. No push happened this cycle (review verdict was NEEDS_WORK).
- **Latest `review` verdict: NEEDS_WORK, loop CONTINUE** — the slice is well-tested and the milestone
  progresses, but three confirmed defects defeat stated controls; PASS may not approve that.
- **Open issues: 2 `normal` + 5 `low`.** The 2 `normal` are both the record-list defects above
  (`/records` no `LastSize` cap; `/records` from=0 overload + clamp overflow), filed `[review]`. The 5
  `low` are loop-skipped (notecheck `out` param; overlay precedence duplicated 3x; mirror write-path
  tile-coord leak; proofserve `os.ErrNotExist`→404 duplication; `[human]` scaling-trip-wire metrics).

## Next Milestone
**M1/M2/M3 met. The active milestone is M-UI (Evidence Ledger frontend, ADR-0010). The immediate next
step is the record-list fix slice — the latest `review` is NEEDS_WORK with 2 open `normal` defects, and
the record-list criterion is not met until they are fixed.**

1. **Record-list fix slice** (clusters in `serveRecords` + `store.ListRecords`, two files):
   (a) thread `fs.LastSize` into `ListRecords` as a `seq < LastSize` ceiling and cap the total, so only
   accepted leaves list (ADR-0001 coverage honesty); (b) clamp page size while still `uint64` BEFORE
   `int()` and bound `parseUint`'s overflow; (c) un-overload `from=0` (a has-cursor bool or 1-based
   cursor) so the older chain reaches seq 0. Tests: a frozen-hub-with-projections-above-`LastSize`
   fixture asserting only accepted rows; `n=9223372036854775808` → ≤ `maxPageSize` rows; an older-link
   walk down to seq 0. Push so CI confirms HEAD.
2. **Single-record page** — render `declaration`, `deletion` (a new record — original preserved), and an
   **unknown `note.$schema`** without erroring; re-point each `/records` row to it.
3. **Certificate of inclusion** at `/inclusion/{iscc_id}` (the slice that re-engages the oracle/
   conformance gate — `serveVerify` currently discards the raw checkpoint bytes + resolved hub key the
   bundle needs) + the **downloadable proof-bundle assembler**, plus the separate Bitcoin-anchor vs
   comparison-anchor panels + tier-1/tier-2 affordance.
4. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
5. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands, and consolidating
   the now-3x overlay precedence into `internal/badge`.
