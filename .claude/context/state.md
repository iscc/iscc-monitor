<!-- assessed-at: 9ddee5b2ae49ce80cd5065e4da21be5e3d079db1 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress. The **single-record page**
(`GET /record?index=<seq>`) defect is **fixed and review-PASSed** — kind-label constants now hold the
full `note.$schema` wire URIs (byte-equal to the golden ground truth), so real declarations/deletions
render correctly. All 8 prior-unpushed commits are pushed; CI is green at HEAD. M1/M2/M3 fully met.
The remaining M-UI screens (certificate of inclusion + proof-bundle assembler, anchor panels) are not
started; WASM and OTS not started.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: badge,
DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, the paginated record list,
and the single-record page are all dressed + verified. The certificate of inclusion / proof-bundle
assembler is the next slice and re-engages the crypto/oracle gate.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~1 still open.** **Met:** five-status `HubStatusBadge`
    (`internal/badge`, golden + mutation + fail-closed); the DS v2 shared shell (`/_ds/` tokens.css +
    self-hosted Readex Pro / JetBrains Mono webfonts, CDN-free, `no-cache`+ETag+304); the `/` realm-index
    grid; the `/<domain>/log/` log-browser redress; the **hub dossier** (`GET /<domain>`,
    `internal/dossier.Handler`); the **frozen Exhibit** (`store.ListViolations` → non-dismissable panel,
    gated on `status == "frozen"`); the **paginated record list** (`GET /<domain>/log/records`,
    `serveRecords` + `store.ListRecords`, accepted-tree `LastSize` cap + hostile-`n` clamp + seq-0
    reachable); the **single-record page** (`GET /record?index=<seq>`, `serveRecord` + `store.RecordAt`)
    — kind-label constants now hold the full wire URIs
    (`http://purl.org/iscc/schema/iscc-note(-delete)-0.8.0.json`), byte-equal to the golden
    `internal/logclient/projection_test.go:19-20`, so declaration/deletion/unknown all render correctly
    (review PASS_WITH_NOTES). **Still open:** the **certificate of inclusion** at `/inclusion/{iscc_id}`
    (numbered evidence clauses §1–§6 + tier-1/tier-2 affordance) + the **downloadable proof-bundle
    assembler** (re-engages the oracle/conformance gate — `serveVerify` discards the raw checkpoint bytes
    + resolved hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~7 milestone-Verify / ~3 refactor·polish·shell.** The arc closed all four M3
  criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid → log-browser redress →
  hub dossier → frozen Exhibit → record list → single record → single-record fix). Not drift: every step
  targets a named M-UI Verify criterion, and the detect→fix discipline is working — the record-list slice
  landed NEEDS_WORK (3 defects) then PASSed; the single-record slice landed NEEDS_WORK (1 defect) and was
  fixed + PASSed the next iteration. M-UI's last large slice is the certificate / proof-bundle assembler,
  the one screen that re-engages the crypto/oracle gate.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `b41512e..HEAD` diff touched only the single-record
kind-label fix (`internal/proofserve/{handler.go,record_test.go}`) + context docs + `target.md` — no M1
source. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **286 `func Test`** across **56** `_test.go` files (unchanged from the prior
  assessment — the fix touched existing test bodies, not the count). Package count **17 internal + 2 cmd**
  (unchanged).
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
**Status**: **in progress — single-record slice now met (review PASS).** Badge, DS shell, `/` index,
`/<domain>/log/` browser, hub dossier, frozen Exhibit, the paginated record list, and the single-record
page are all met and verified. The remaining M-UI screens (certificate of inclusion + proof-bundle
assembler, anchor panels) are not started.
- **Met (carried forward + newly closed):** `internal/badge` five-status `Render`; `internal/web`
  `/_ds/` static-asset subtree; the `/` realm-index grid; the `/<domain>/log/` log-browser redress;
  `internal/dossier.Handler` (`GET /<domain>`); the frozen Exhibit; the paginated record list
  (`GET /<domain>/log/records`); and now the **single-record page** (`serveRecord` at
  `internal/proofserve/handler.go` + `store.RecordAt`, mounted at `GET /record?index=<seq>`). The
  blocking kind-label defect is **fixed**: `handler.go:111-112` constants
  (`schemaDeclaration`/`schemaDeletion`) now hold the full wire URIs
  `http://purl.org/iscc/schema/iscc-note(-delete)-0.8.0.json`, byte-equal to the golden ground truth
  `internal/logclient/projection_test.go:19-20`, so real declarations render "Declaration", deletions
  "Deletion", unknown/empty → "Unknown record type" (all 200). The no-CDN scheme ban was correctly
  re-scoped to the document head (up to `</style>`) so it no longer false-fails on the verbatim
  `http://` schema URI in the body.
- **Residual (off the Verify bar, filed `low`):** the guarding `record_test.go::schemaForSeq` returns
  the constants under test rather than HARDCODED literals, so reverting both constants leaves the
  proofserve record suite green — a vacuous regression gate. Production code is correct (the live
  protection is that `projection_test.go` is itself a non-vacuous golden test the constants match);
  harden when `record_test.go` is next touched.
- **Still open on the M-UI Verify bar:** the **certificate of inclusion** at `/inclusion/{iscc_id}`
  (numbered evidence clauses §1–§6 + tier-1/tier-2 affordance) + the **downloadable proof-bundle
  assembler** (re-engages the oracle/conformance gate — `serveVerify` discards the raw checkpoint bytes
  + resolved hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels.
  Confirmed absent: `cmd/iscc-monitor/main.go:296` mounts `/inclusion` only as the per-hub JSON proof
  route (`/<domain>/log/inclusion`); there is no realm-wide `/inclusion/{iscc_id}` HTML certificate page
  or proof-bundle assembler in source.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
source; no `internal/proof` package; no WASM build target (`syscall/js` not in source). The
`internal/badge`, `internal/web`, and `internal/metrics` leaves are WASM-shareable primitives the
verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **green at HEAD.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. The latest `review` verdict (`9ddee5b`) is **PASS_WITH_NOTES** with `mise run check`
  recorded green (build + vet + all packages, `gofmt -l .` empty); the lone residual is a `low`
  test-hardening item, not a gate failure.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest run on `develop`: `conclusion: success` at headSha `9ddee5b` (run 27915327998)** — that is
  HEAD. `develop` is in sync with `origin/develop`; all 8 previously-unpushed commits are pushed and CI
  has seen the single-record slice. Working tree clean.
- **Open issues: 0 `critical`, 0 `normal`, 6 `low`.** The single-record kind-label `normal` issue is
  resolved (production fix landed + PASSed). The 6 `low` are loop-skipped (vacuous single-record label
  test; notecheck `out` param; overlay precedence duplicated 3x; mirror write-path tile-coord leak;
  proofserve `os.ErrNotExist`→404 duplication; `[human]` scaling trip-wire metrics).

## Next Milestone
**M1/M2/M3 met and the single-record M-UI slice is now met (review PASS). The active milestone is M-UI
(Evidence Ledger frontend, ADR-0010); CI is green.**

1. **Certificate of inclusion** at `/inclusion/{iscc_id}` (HTML page with numbered evidence clauses
   §1 Subject · §2 Checkpoint (size, root) · §3 Inclusion proof · §4 Signing key (did:web) · §5 Bitcoin
   anchor · §6 Record history, plus the tier-1/tier-2 honesty panel) + the **downloadable proof-bundle
   assembler** `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}`. This slice
   **re-engages the oracle/conformance gate** — `serveVerify` currently discards the raw checkpoint bytes
   + resolved hub key the bundle needs, so the reviewer must mutation-prove the served bundle's inclusion
   proof non-vacuous and confirm `notecheck`/golden-vector parity for the hub-signed material. Plus the
   separate Bitcoin-anchor vs comparison-anchor panels.
2. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
3. **Off the Verify bar:** harden the vacuous single-record label test (filed `low`); sb1 fixture refresh
   (stale did.json key); real alert transport; an end-to-end registry-deactivation `inactive` path once a
   public `SetActive` lands; consolidating the now-3x overlay precedence into `internal/badge`.
