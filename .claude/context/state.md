<!-- assessed-at: 0f2ed1a2da1ccec04c9e8b37f3a0cf08d4a36b2a -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — building the realm-wide certificate path. The
certificate's trust-root decoder (`internal/index`) is now complete and **PASS** (fail-closed on all
four header nibbles), CI green at HEAD, no open `normal`/`critical` issue. M1/M2/M3 fully met. The next
slice is the 12-bit `hub_id` → issuing-hub resolver, then the certificate HTML page + proof-bundle
assembler (which re-engages the crypto/oracle gate). WASM and OTS not started.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: badge,
DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, paginated record list,
single record, and the certificate's ISCC-IDv1 decoder are all dressed + verified. The remaining M-UI
slices are the certificate-of-inclusion HTML page + downloadable proof-bundle assembler and the
Bitcoin-anchor / comparison-anchor panels.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~1 still open.** **Met:** five-status `HubStatusBadge`
    (`internal/badge`, golden + mutation + fail-closed); the DS v2 shared shell (`/_ds/` tokens.css +
    self-hosted Readex Pro / JetBrains Mono webfonts, CDN-free, `no-cache`+ETag+304); the `/` realm-index
    grid; the `/<domain>/log/` log-browser redress; the **hub dossier** (`GET /<domain>`,
    `internal/dossier.Handler`); the **frozen Exhibit** (`store.ListViolations` → non-dismissable panel,
    gated on `status == "frozen"`); the **paginated record list** (`GET /<domain>/log/records`,
    `serveRecords` + `store.ListRecords`); the **single-record page** (`GET /record?index=<seq>`,
    `serveRecord` + `store.RecordAt`); the certificate's trust-root **ISCC-IDv1 decoder**
    (`internal/index.Decode`) — codec correct, golden-tested, mutation-proven, now **fail-closed on all
    four header nibbles** (MainType, Version, Length validated; SubType read as realm), review PASS.
    **Still open:** the **certificate of inclusion** HTML page at `/inclusion/{iscc_id}` (numbered
    evidence clauses §1–§6 + tier-1/tier-2 affordance) + the **downloadable proof-bundle assembler**
    (re-engages the oracle/conformance gate — `serveVerify` discards the raw checkpoint bytes + resolved
    hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels. The intermediate
    step before the page is the 12-bit-`hub_id` → issuing-hub resolver (registry / ADR-0010 Hub-List).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~7 milestone-Verify / ~3 refactor·polish·shell.** The arc closed all four M3
  criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid → log-browser redress →
  hub dossier → frozen Exhibit → record list → single record → single-record fix → ISCC-IDv1 decoder →
  Length-nibble guard). Not drift: every step targets a named M-UI Verify criterion, and the
  detect→fix discipline is working — the decoder landed NEEDS_WORK (Length-nibble gap) and the very next
  iteration closed it to PASS with a mutation-proven regression test. The certificate / proof-bundle
  assembler (the screen that re-engages the crypto/oracle gate) is the milestone's last large slice and
  the now-complete decoder is its trust root.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `f8c741d..HEAD` diff touched only `internal/index`
(`iscc.go` +8 lines: the Length guard; `iscc_test.go` +23: the regression test) and context docs — no
M1 source touched. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **293 `func Test`** across **57** `_test.go` files (was 292/57 — the
  Length-guard slice added `TestDecodeRejectsNonzeroLength`, 1 test, in the existing index test file).
  Package count **18 internal + 2 cmd** (unchanged).
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **18 internal packages** —
  `badge, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient, metrics,
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
**Status**: **met** — carried forward; no change to M2 source. Both Verify criteria exercised: fsck
root-rebuild WIRED on every verified non-frozen poll; inclusion cross-check conformance-tested over the
real verified mirror. All three computed proofs — `inclusion`, `consistency`, `entries` — served from
the local mirror, never re-hitting the hub.

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
**Status**: **in progress — certificate trust-root decoder complete (PASS).** Badge, DS shell, `/`
index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, paginated record list, single-record
page, and the ISCC-IDv1 decoder are all met and verified. The certificate HTML page + proof-bundle
assembler and the anchor panels are not started.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` log-browser redress; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`serveRecord` at `internal/proofserve/handler.go` + `store.RecordAt`, mounted at
  `GET /record?index=<seq>`).
- **Met (newly closed — review PASS at 0f2ed1a):** the certificate's trust-root
  **`internal/index.Decode`** — a pure, WASM-shareable ISCC-IDv1 decoder parsing `{Realm, HubID,
  Timestamp}` (SubType nibble realm, low-12-bit hub_id, high-52-bit µs timestamp). Codec arithmetic is
  golden-tested against the hub's own example, independently re-decoded in Python; mutation-proven
  non-vacuous. The prior NEEDS_WORK Length-nibble gap is **closed**: `internal/index/iscc.go:99-102` now
  rejects any header whose Length nibble (`raw[1] & 0xF`) is nonzero, guarded by
  `TestDecodeRejectsNonzeroLength`; the reviewer re-ran two independent mutations and re-derived both
  golden vectors' header bytes (Length 0) via Python. The decoder is fail-closed on all four header
  nibbles. `GOOS=js GOARCH=wasm go build ./internal/index` succeeds (stdlib-only closure).
- **Residual (off the Verify bar, filed `low`):** the vacuous single-record label test
  (`record_test.go::schemaForSeq` returns the constants under test rather than HARDCODED literals);
  harden when `record_test.go` is next touched.
- **Still open on the M-UI Verify bar:** the **certificate of inclusion** HTML page at
  `/inclusion/{iscc_id}` (numbered evidence clauses §1–§6 + tier-1/tier-2 affordance) + the
  **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate — `serveVerify`
  discards the raw checkpoint bytes + resolved hub key the bundle needs); separate **Bitcoin-anchor vs
  comparison-anchor** panels. Confirmed absent: `cmd/iscc-monitor/main.go` mounts `/inclusion` only as
  the per-hub JSON proof route (`/<domain>/log/inclusion`); no realm-wide `/inclusion/{iscc_id}` HTML
  certificate page or proof-bundle assembler in source. The intermediate step before the page is the
  12-bit-`hub_id` → issuing-hub resolver (registry / ADR-0010 Hub-List). **Note (review flag):** ADR-0010
  moves `internal/registry` from the domains-only realm file to the `hubs/<network>.yaml` Hub-List — a
  backward-incompatible registry-format change; the realm-file parser / config / dashboard callers must
  migrate together, and the format swap may warrant a STOP for human sign-off given the public-ish
  realm-file contract.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
source; no `internal/proof` package; no WASM build target (`syscall/js` not in source). The
`internal/badge`, `internal/web`, `internal/metrics`, and `internal/index` leaves are WASM-shareable
primitives the verifier app will reuse (`internal/index` is confirmed `GOOS=js GOARCH=wasm`-buildable),
but the verifier itself does not exist.

## Quality gates
**Status**: **green — CI passing at HEAD, working tree clean, nothing unpushed.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. The latest `review` verdict (0f2ed1a) records `mise run check` green (build + vet +
  all 20 packages incl. `internal/index` uncached, `gofmt -l .` empty) and is a clean **PASS**.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest run on `develop`: `conclusion: success` at headSha `0f2ed1a` (run 27916002703) — that IS
  HEAD.** `develop` is in sync with `origin/develop` (no unpushed commits). The green CI now covers the
  Length-nibble guard.
- **Open issues: 0 `critical`, 0 `normal`, 6 `low`.** The prior `normal` (ISCC-IDv1 Length-nibble
  fail-closed gap) is verified fixed and deleted from `issues.md`. The 6 `low` are loop-skipped
  (vacuous single-record label test; notecheck `out` param; overlay precedence duplicated 3x; mirror
  write-path tile-coord leak; proofserve `os.ErrNotExist`→404 duplication; `[human]` scaling trip-wire
  metrics).

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone. Latest review is a clean PASS, no `normal`/`critical`
issue open, CI green at HEAD — clear to proceed with the next M-UI slice.**

1. **The 12-bit `hub_id` → issuing-hub resolver.** Per ADR-0010, move `internal/registry` from the
   domains-only realm file to the `hubs/<network>.yaml` Hub-List so the decoded `(realm, hub_id)`
   resolves to a hub domain. This is a backward-incompatible registry-format change — migrate the
   realm-file parser / config / dashboard callers together, and weigh a STOP for human sign-off on the
   public-ish realm-file contract before building.
2. **Then the certificate path proper:** the **certificate of inclusion** HTML page at
   `/inclusion/{iscc_id}` (numbered evidence clauses §1 Subject · §2 Checkpoint (size, root) · §3
   Inclusion proof · §4 Signing key (did:web) · §5 Bitcoin anchor · §6 Record history, plus the
   tier-1/tier-2 honesty panel) + the **downloadable proof-bundle assembler** `{checkpoint,
   inclusion/consistency proof, record bytes, hub key, ots?}`. This slice **re-engages the
   oracle/conformance gate** — `serveVerify` currently discards the raw checkpoint bytes + resolved hub
   key the bundle needs, so the reviewer must mutation-prove the served bundle's inclusion proof
   non-vacuous and confirm `notecheck`/golden-vector parity. Plus the separate Bitcoin-anchor vs
   comparison-anchor panels.
3. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** harden the vacuous single-record label test (filed `low`); sb1 fixture
   refresh (stale did.json key); real alert transport; an end-to-end registry-deactivation `inactive`
   path once a public `SetActive` lands; consolidating the now-3x overlay precedence into `internal/badge`.
