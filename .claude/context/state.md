<!-- assessed-at: b41512e1ab2def4176b2eb033646c3e694fb1d3f -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress. The **single-record page**
(`GET /record?index=<seq>`) has landed but is **review-NEEDS_WORK**: the kind-label constants use the
glossary short forms while production `note.$schema` is the full `http://purl.org/...json` URI, so every
real declaration/deletion renders "Unknown record type". One open `normal` issue. M1/M2/M3 remain fully
met. The four single-record commits are **unpushed**; CI is green at the last pushed commit.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: badge,
DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, and the paginated record
list are all dressed + verified. The single-record page is the active slice — landed with a correct
handler flow but a real correctness defect (see below). WASM and OTS are not started.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~2 still open.** **Met:** five-status `HubStatusBadge`
    (`internal/badge`, golden + mutation + fail-closed) wired into `/`, `/<domain>/log/`, dossier; the
    DS v2 shared shell (`/_ds/` tokens.css + self-hosted Readex Pro / JetBrains Mono webfonts, CDN-free,
    `no-cache`+ETag+304); the `/` realm-index grid redress; the `/<domain>/log/` log-browser redress; the
    **hub dossier** (`GET /<domain>`, `internal/dossier.Handler`); the **frozen Exhibit**
    (`store.ListViolations` → non-dismissable panel, gated on `status == "frozen"`); the **paginated
    record list** (`GET /<domain>/log/records`, `serveRecords` + `store.ListRecords`, accepted-tree
    `LastSize` cap + hostile-`n` clamp + seq-0 reachable, all non-vacuously tested). **In-flight /
    NEEDS_WORK:** the **single-record page** (`GET /record?index=<seq>`, `serveRecord` +
    `store.RecordAt`) — handler flow is correct (accepted-tree cap, bytes-as-source-of-truth so a missing
    projection renders rather than 404s, DS shell, badge overlay, 200 on unknown/empty schema) and rows
    in `/records` are re-pointed to it, BUT the kind-label is broken (see defect). **Still open:** the
    **certificate of inclusion** at `/inclusion/{iscc_id}` (numbered evidence clauses + tier-1/tier-2
    affordance) + the **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate —
    `serveVerify` discards the raw checkpoint bytes + resolved hub key the bundle needs); separate
    **Bitcoin-anchor vs comparison-anchor** panels.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~7 milestone-Verify / ~3 refactor·polish·shell.** The arc closed all four M3
  criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid → log-browser redress →
  hub dossier → frozen Exhibit → record list → single record). Not drift: every step targets a named
  M-UI Verify criterion, and the loop's detect→fix discipline is working — the record-list slice landed
  NEEDS_WORK (3 defects) then PASSed the next iteration; the single-record slice has now landed
  NEEDS_WORK (1 defect) and awaits its fix. M-UI remains the largest remaining slice; the proof-bundle
  assembler is the one screen that re-engages the crypto/oracle gate.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `83588b1..HEAD` diff touched only the single-record slice
(`internal/proofserve/{handler.go,record.html,record_test.go,records.html,records_test.go}`,
`internal/store/iscc_index.go` + test, `cmd/iscc-monitor/main.go` + test, `CLAUDE.md`, context docs) —
no M1 source. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **286 `func Test`** across **56** `_test.go` files (was 271/55 — +15 from
  the single-record slice's new `record_test.go` + store/`main` additions). Package count **17 internal
  + 2 cmd** (unchanged).
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
**Status**: **in progress — single-record slice NEEDS_WORK.** Badge, DS shell, `/` index,
`/<domain>/log/` browser, hub dossier, frozen Exhibit, and the paginated record list are all met and
verified. The single-record page has landed but the latest `review` verdict is **NEEDS_WORK** with one
open `normal` issue (see defect). The remaining M-UI screens (certificate of inclusion + proof-bundle
assembler, anchor panels) are not started.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid redress; the `/<domain>/log/` log-browser redress;
  `internal/dossier.Handler` (`GET /<domain>`); the frozen Exhibit; the paginated record list
  (`GET /<domain>/log/records`, `serveRecords`, `store.ListRecords` with the LastSize cap + hostile-`n`
  clamp + seq-0-reachable older chain, all review-PASSed).
- **In-flight (NEEDS_WORK):** the **single-record page** — `serveRecord` at
  `internal/proofserve/handler.go` + `store.RecordAt`, mounted at `GET /record?index=<seq>` in
  `cmd/iscc-monitor/main.go`, with each `/records` row re-pointed from `entries?index=` to
  `record?index=`. The handler flow is correct (accepted-tree cap, partial-bundle `p`,
  bytes-as-source-of-truth so a missing projection renders rather than 404s, buffer-then-200, DS shell,
  no-CDN, badge overlay, 200 on unknown/empty schema). **The blocking defect:**
  - **Kind-label constants miss the real `note.$schema` URIs** (open `normal` issue). `handler.go:108-109`
    sets `schemaDeclaration = "iscc-note-0.8.0"` / `schemaDeletion = "iscc-note-delete-0.8.0"`, but the
    projection fold stores the verbatim wire value — the full URI
    `http://purl.org/iscc/schema/iscc-note-0.8.0.json` (and `…delete…`). Ground truth confirms this:
    `internal/logclient/projection_test.go:19-20` and `internal/follower/fsck_test.go:119` pin the full
    URIs. So `recordKind` falls through to `kindUnknown` for every real declaration/deletion, defeating
    the M-UI Verify criterion. The new `record_test.go::schemaForSeq` (lines 40-50) returns the bare
    `schemaDeclaration`/`schemaDeletion` constants — a self-consistent fixture that masks the bug. Fix:
    set both constants to the full URIs and reseed the test with the production URIs (and confirm the
    no-CDN `http://` body ban targets the template/CDN region, not verbatim record fields).
- **Still open on the M-UI Verify bar:** the **certificate of inclusion** at `/inclusion/{iscc_id}`
  (numbered evidence clauses + tier-1/tier-2 affordance) + the **downloadable proof-bundle assembler**
  (re-engages the oracle/conformance gate — `serveVerify` discards the raw checkpoint bytes + resolved
  hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
source; no `internal/proof` package; no WASM build target (`syscall/js` not in source). The
`internal/badge`, `internal/web`, and `internal/metrics` leaves are WASM-shareable primitives the
verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **green at last pushed commit; the unpushed single-record slice has a known defect.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. The single-record commits compile and `mise run check` passes per the review (the
  defect is a correctness bug masked by a self-consistent synthetic test, not a build/gate failure).
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest run on `develop`: `conclusion: success` at headSha `83588b1` (run 27914382741).** That is
  the last **pushed** commit — `develop` is **ahead 4** (the single-record `define-next`/`advance`/
  `update-state`-equivalent + NEEDS_WORK `review`), which are correctly NOT pushed because the slice is
  NEEDS_WORK. CI has therefore not seen the single-record code; per protocol the next cycle fixes the
  defect, then pushes. Working tree has one unstaged change to `.claude/context/target.md` (review-noted,
  not state-owned — left untouched).
- **Open issues: 0 `critical`, 1 `normal`, 5 `low`.** The 1 `normal` is the single-record kind-label
  defect above. The 5 `low` are loop-skipped (notecheck `out` param; overlay precedence duplicated 3x;
  mirror write-path tile-coord leak; proofserve `os.ErrNotExist`→404 duplication; `[human]` scaling
  trip-wire metrics).

## Next Milestone
**M1/M2/M3 met. The active milestone is M-UI (Evidence Ledger frontend, ADR-0010); the single-record
slice is NEEDS_WORK and must be fixed first.**

1. **Fix the single-record kind-label defect** (the open `normal` issue): set
   `schemaDeclaration`/`schemaDeletion` to the full `http://purl.org/iscc/schema/iscc-note(-delete)-0.8.0.json`
   URIs and reseed `record_test.go::schemaForSeq` with the production URIs (confirm the no-CDN ban does
   not false-fail on the now-realistic schema string). A ~2-line constant change + fixture correction
   scoped to `internal/proofserve`. This closes the single-record M-UI Verify criterion. Then push so CI
   sees the slice.
2. **Certificate of inclusion** at `/inclusion/{iscc_id}` (the slice that re-engages the oracle/
   conformance gate — `serveVerify` currently discards the raw checkpoint bytes + resolved hub key the
   bundle needs) + the **downloadable proof-bundle assembler**, plus the separate Bitcoin-anchor vs
   comparison-anchor panels + tier-1/tier-2 affordance.
3. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands, and consolidating
   the now-3x overlay precedence into `internal/badge`.
