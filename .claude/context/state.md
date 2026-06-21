<!-- assessed-at: 83588b194ecb589d021ae0e7c2a911e940cc56a1 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress. The paginated record-list slice's
three defects are **fixed and review-PASSed** (LastSize cap, pre-`int()` overflow clamp, seq-0
reachable); both `normal` issues are closed. CI is green at HEAD. M1/M2/M3 remain fully met.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: the
badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, **and now the
paginated record list** are dressed, fixed, and verified. WASM and OTS are not started.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~2 still open.** **Met:** five-status `HubStatusBadge`
    (`internal/badge`, all five statuses, distinct label + inline-SVG silhouette, golden + mutation +
    fail-closed) wired into `/`, `/<domain>/log/`, and the dossier; the DS v2 shared shell (`/_ds/`
    tokens.css + self-hosted Readex Pro / JetBrains Mono webfonts, CDN-free, `no-cache`+ETag+304); the
    `/` realm index Evidence-Ledger CSS-grid redress; the `/<domain>/log/` log-browser Evidence-Ledger
    card redress; the **hub dossier** (`GET /<domain>`, `internal/dossier.Handler`); the **frozen
    Exhibit** (`store.ListViolations` → non-dismissable Exhibit panel, gated on `status == "frozen"`);
    and the **paginated record list** (`GET /<domain>/log/records?from=…[&n=…]`, `serveRecords` +
    `store.ListRecords`) — no-JS newest-first list, DS shell, empty state, badge overlay, accepted-tree
    `LastSize` ceiling, hostile-`n` clamp, and a seq-0-reachable older chain, all non-vacuously tested
    (three mutations re-run, all FAIL + revert). **Still open:** the **single-record page**
    (declaration / deletion / unknown schema), with each `/records` row re-pointed from
    `entries?index=<seq>` to it; the **certificate of inclusion** at `/inclusion/{iscc_id}` (numbered
    evidence clauses + tier-1/tier-2 affordance) + the **downloadable proof-bundle assembler**
    (re-engages the oracle/conformance gate — `serveVerify` discards the raw checkpoint bytes +
    resolved hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~7 milestone-Verify / ~3 refactor·polish·shell, with 2 of those a defect-fix
  pair.** The arc closed all four M3 criteria, then opened M-UI leaf-first (badge → DS tokens →
  webfonts → `/` grid → log-browser redress → hub dossier → frozen Exhibit → record list). The
  record-list slice landed NEEDS_WORK (3 defects), then the very next iteration fixed all three and
  PASSed — a tight detect→fix loop, not drift. Each step targets a named M-UI Verify criterion. M-UI
  remains the largest remaining slice; the proof-bundle assembler is the one screen that re-engages
  the crypto/oracle gate.

## M1 — Read-only Monitor
**Status**: **met** — carried forward; the `a9c29fa..HEAD` diff touched only `internal/proofserve/*`
(record-list fix), `internal/store/iscc_index*` (`ListRecords` ceiling + `hasFrom` cursor), an ADR-0007
note, and context docs — no M1 source. All M1 Verify criteria remain satisfied: `origin`/`vkey`
golden, all three triggers (fork/shrink/equivocation) golden-tested end-to-end with freeze +
alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **271 `func Test`** across **55** `_test.go` files (was 266/55 — +5 from the
  record-list fix: `TestListRecordsCeiling`, `TestRecordsClampsHostilePageSize`,
  `TestRecordsOlderLinkReachesSeq0`, `TestRecordsCeilingHidesUnacceptedLeaves`, `TestParseUintOverflow`).
  Package count **17 internal + 2 cmd**.
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
**Status**: **in progress.** Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, the
frozen Exhibit, **and the paginated record list** are all met and verified (latest `review` verdict
**PASS**, both `normal` record-list issues closed). The remaining M-UI screens (single record,
certificate of inclusion + proof-bundle assembler, anchor panels) are not started.
- **Met:** `internal/badge` five-status `Render` (golden + mutation, fail-closed); `internal/web`
  `/_ds/` static-asset subtree (token CSS + self-hosted woff2 + `@font-face`, CDN-free, `no-cache` +
  strong content-ETag + 304, traversal-guarded); the `/` realm-index Evidence-Ledger CSS-grid redress;
  the `/<domain>/log/` log-browser card redress; `internal/dossier.Handler` serving `GET /<domain>`
  (Evidence-Ledger per-hub page, coverage honesty, badge overlay, reserved/empty-domain mount guard);
  the **frozen Exhibit** (`store.ListViolations` newest-first read → non-dismissable Exhibit panel,
  gated on `status == "frozen"`); the **paginated record list** (`GET /<domain>/log/records`,
  `serveRecords` at `internal/proofserve/handler.go`, `store.ListRecords` at
  `internal/store/iscc_index.go:110`) — no-JS newest-first list, DS shell, CDN-free body,
  overlay-status badge, empty state, all non-vacuously tested. The three prior defects are fixed and
  verified by `review`:
  1. **`LastSize` cap** — `ListRecords` now takes `last uint64` and applies a `seq < last` ceiling to
     BOTH the `COUNT(*)` total and the windowed `SELECT`; only accepted leaves list (ADR-0001 coverage
     honesty). `TestListRecordsCeiling` + `TestRecordsCeilingHidesUnacceptedLeaves` cover it.
  2. **Page-size clamp before `int()`** — `serveRecords` clamps `n > maxPageSize` while still `uint64`,
     BEFORE the `int()` conversion (`handler.go:711-714`); `parseUint` rejects uint64 overflow.
     `TestRecordsClampsHostilePageSize` + `TestParseUintOverflow` cover it.
  3. **seq 0 reachable** — `from` is no longer overloaded as the start-at-newest sentinel; an explicit
     `hasFrom bool` carries the present/absent distinction so the older chain walks down to seq 0.
     `TestRecordsOlderLinkReachesSeq0` covers it.
- **Still open on the M-UI Verify bar:** the **single-record page** (declaration / deletion / unknown
  schema), with each `/records` row re-pointed from `entries?index=` to it; the **certificate of
  inclusion** at `/inclusion/{iscc_id}` (numbered evidence clauses + tier-1/tier-2 affordance) + the
  **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate — `serveVerify`
  discards the raw checkpoint bytes + resolved hub key the bundle needs); separate **Bitcoin-anchor vs
  comparison-anchor** panels.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
source; no `internal/proof` package; no WASM build target (`syscall/js` not in source). The
`internal/badge`, `internal/web`, and `internal/metrics` leaves are WASM-shareable primitives the
verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **green.** Latest `review` verdict is **PASS / loop CONTINUE**; CI is green at HEAD.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. The latest `review` recorded `mise run check` green at HEAD (build + vet + all 20
  packages `ok`, `gofmt -l .` empty) and re-ran all three record-list mutations (each FAILS without the
  fix, reverted).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest run on `develop`: `conclusion: success` at headSha `83588b1` (run 27914382741) = HEAD.**
  Working tree clean, `develop` in sync with `origin/develop` — HEAD is pushed and CI-confirmed.
- **Open issues: 0 `critical`, 0 `normal`, 5 `low`.** The 5 `low` are loop-skipped (notecheck `out`
  param; overlay precedence duplicated 3x; mirror write-path tile-coord leak; proofserve
  `os.ErrNotExist`→404 duplication; `[human]` scaling-trip-wire metrics).

## Next Milestone
**M1/M2/M3 met; the M-UI record-list criterion is now met and CI-confirmed. The active milestone is
M-UI (Evidence Ledger frontend, ADR-0010). Resume the planned M-UI order:**

1. **Single-record page** — render `declaration`, `deletion` (a new record — original preserved), and an
   **unknown `note.$schema`** without erroring; re-point each `/records` row from `entries?index=<seq>`
   to it.
2. **Certificate of inclusion** at `/inclusion/{iscc_id}` (the slice that re-engages the oracle/
   conformance gate — `serveVerify` currently discards the raw checkpoint bytes + resolved hub key the
   bundle needs) + the **downloadable proof-bundle assembler**, plus the separate Bitcoin-anchor vs
   comparison-anchor panels + tier-1/tier-2 affordance.
3. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands, and consolidating
   the now-3x overlay precedence into `internal/badge`.
