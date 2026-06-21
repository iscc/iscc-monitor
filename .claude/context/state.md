<!-- assessed-at: f8c741d118e22c922abad6fece48fbe833157f10 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — building the realm-wide certificate path. The
first sub-step, a pure ISCC-IDv1 decoder (`internal/index`), has landed but the latest `review` verdict
is **NEEDS_WORK** (codec correct + mutation-proven, but the decoder does not validate the header
**Length nibble** — a fail-closed gap on the certificate's trust root, filed `normal`). M1/M2/M3 fully
met. The certificate of inclusion + proof-bundle assembler and the anchor panels are not yet built;
WASM and OTS not started.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: badge,
DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, paginated record list, and
single-record page are all dressed + verified. The certificate's trust-root decoder (`internal/index`)
is in place but has one open NEEDS_WORK defect; the certificate HTML page / proof-bundle assembler (which
re-engages the crypto/oracle gate) is the next large slice.

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
    `serveRecord` + `store.RecordAt`, kind-label constants byte-equal to golden, review PASS). **In
    progress (NEEDS_WORK):** the certificate's trust-root **ISCC-IDv1 decoder** (`internal/index.Decode`)
    — codec correct + golden-tested + mutation-proven non-vacuous, but does not validate the header Length
    nibble (open `normal` issue). **Still open:** the **certificate of inclusion** HTML page at
    `/inclusion/{iscc_id}` (numbered evidence clauses §1–§6 + tier-1/tier-2 affordance) + the
    **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate — `serveVerify`
    discards the raw checkpoint bytes + resolved hub key the bundle needs); separate **Bitcoin-anchor vs
    comparison-anchor** panels.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~7 milestone-Verify / ~3 refactor·polish·shell.** The arc closed all four M3
  criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid → log-browser redress →
  hub dossier → frozen Exhibit → record list → single record → single-record fix → ISCC-IDv1 decoder).
  Not drift: every step targets a named M-UI Verify criterion, and the detect→fix discipline is working —
  the record-list and single-record slices each landed NEEDS_WORK then PASSed; the just-landed decoder is
  NEEDS_WORK with a one-guard fix pending. The certificate / proof-bundle assembler (the screen that
  re-engages the crypto/oracle gate) is the milestone's last large slice and the decoder is its trust root.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `9ddee5b..HEAD` diff added only the new `internal/index`
package (+ tests) and context docs — no M1 source touched. All M1 Verify criteria remain satisfied:
`origin`/`vkey` golden, all three triggers (fork/shrink/equivocation) golden-tested end-to-end with
freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **292 `func Test`** across **57** `_test.go` files (was 286/56 — the
  `internal/index` slice added 6 tests in 1 file). Package count **18 internal + 2 cmd** (was 17 internal;
  `internal/index` is the new leaf).
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
**Status**: **in progress — certificate trust-root decoder landed NEEDS_WORK.** Badge, DS shell, `/`
index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, paginated record list, and single-record
page are all met and verified. The certificate's ISCC-IDv1 decoder is in place but has one open defect;
the certificate HTML page + proof-bundle assembler and the anchor panels are not started.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` log-browser redress; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`serveRecord` at `internal/proofserve/handler.go` + `store.RecordAt`, mounted at
  `GET /record?index=<seq>`, kind-label constants byte-equal to the golden ground truth — review PASS).
- **In progress (NEEDS_WORK — latest review verdict f8c741d):** the certificate's trust-root
  **`internal/index.Decode`** — a pure, WASM-shareable ISCC-IDv1 decoder parsing `{Realm, HubID,
  Timestamp}` (SubType nibble realm, low-12-bit hub_id, high-52-bit µs timestamp). Codec arithmetic is
  correct and golden-tested against the hub's own `schema.py` example, independently re-decoded in
  Python; mutation-proven non-vacuous. **Open defect (filed `normal`):** `Decode` validates MainType
  (`raw[0]>>4 == 6`) and Version (`raw[1]>>4 == 1`) but never checks the **Length nibble**
  (`raw[1] & 0xF`) — `internal/index/iscc.go:91-94` reads the body straight after the Version check. A
  header like `MAIQAAAAAAAAAAAA` (byte1 = 0x11, Length nibble 1) is accepted and mis-read as a 64-bit
  body, routing a malformed id to a hub instead of rejecting it. Fix: guard `raw[1] & 0xF == 0` before
  reading the body + add a malformed golden case; both real golden vectors (Length 0) still decode.
- **Residual (off the Verify bar, filed `low`):** the vacuous single-record label test
  (`record_test.go::schemaForSeq` returns the constants under test rather than HARDCODED literals);
  harden when `record_test.go` is next touched.
- **Still open on the M-UI Verify bar:** the **certificate of inclusion** HTML page at
  `/inclusion/{iscc_id}` (numbered evidence clauses §1–§6 + tier-1/tier-2 affordance) + the
  **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate — `serveVerify`
  discards the raw checkpoint bytes + resolved hub key the bundle needs); separate **Bitcoin-anchor vs
  comparison-anchor** panels. Confirmed absent: `cmd/iscc-monitor/main.go` mounts `/inclusion` only as
  the per-hub JSON proof route (`/<domain>/log/inclusion`); no realm-wide `/inclusion/{iscc_id}` HTML
  certificate page or proof-bundle assembler in source. The decoder is the first piece of this path; the
  next step after the Length-guard fix is the 12-bit-`hub_id` → hub resolver (registry / Hub-List).
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
source; no `internal/proof` package; no WASM build target (`syscall/js` not in source). The
`internal/badge`, `internal/web`, `internal/metrics`, and now `internal/index` leaves are WASM-shareable
primitives the verifier app will reuse (`internal/index` is confirmed `GOOS=js GOARCH=wasm`-buildable per
the review), but the verifier itself does not exist.

## Quality gates
**Status**: **green at last review — but HEAD is unpushed and CI has NOT yet seen `internal/index`.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. The latest `review` verdict (f8c741d) records `mise run check` green (build + vet +
  all 20 packages incl. `internal/index` uncached, `gofmt -l .` empty) — but the verdict is
  **NEEDS_WORK** on a fail-closed contract gap (the Length-nibble guard), so the gate passing does not
  make the slice complete.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest run on `develop`: `conclusion: success` at headSha `9ddee5b` (run 27915327998) — that is NOT
  HEAD.** HEAD is `f8c741d`, and `develop` is **4 commits ahead of `origin/develop` (unpushed)**:
  define-next + advance + NEEDS_WORK review + the prior update-state. CI has not exercised
  `internal/index`. By loop convention NEEDS_WORK commits stay local until a clean PASS pushes — so this
  is expected, but it means the latest green CI does not cover HEAD. Working tree clean.
- **Open issues: 0 `critical`, 1 `normal`, 6 `low`.** The `normal` is the ISCC-IDv1 Length-nibble
  fail-closed gap (review-confirmed Codex P2, the active work item). The 6 `low` are loop-skipped
  (vacuous single-record label test; notecheck `out` param; overlay precedence duplicated 3x; mirror
  write-path tile-coord leak; proofserve `os.ErrNotExist`→404 duplication; `[human]` scaling trip-wire
  metrics).

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone. The latest review is NEEDS_WORK and one `normal` issue is
open — fix that before any new feature work.**

1. **Close the open `normal` issue first**: add the Length-nibble guard to `internal/index.Decode`
   (`raw[1] & 0xF == 0`, reject otherwise) before it reads the body, plus a malformed golden case
   (`MAIQAAAAAAAAAAAA`); confirm both existing golden vectors still decode and that flipping the guard
   makes a test FAIL. This is the trust root of the certificate path and must land + push (CI green over
   HEAD) before the decoder is consumed by the hub resolver.
2. **Then continue the certificate path**: the 12-bit-`hub_id` → issuing-hub resolver (registry /
   ADR-0010 Hub-List), then the **certificate of inclusion** HTML page at `/inclusion/{iscc_id}`
   (numbered evidence clauses §1 Subject · §2 Checkpoint (size, root) · §3 Inclusion proof · §4 Signing
   key (did:web) · §5 Bitcoin anchor · §6 Record history, plus the tier-1/tier-2 honesty panel) + the
   **downloadable proof-bundle assembler** `{checkpoint, inclusion/consistency proof, record bytes, hub
   key, ots?}`. This slice **re-engages the oracle/conformance gate** — `serveVerify` currently discards
   the raw checkpoint bytes + resolved hub key the bundle needs, so the reviewer must mutation-prove the
   served bundle's inclusion proof non-vacuous and confirm `notecheck`/golden-vector parity. Plus the
   separate Bitcoin-anchor vs comparison-anchor panels.
3. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** harden the vacuous single-record label test (filed `low`); sb1 fixture
   refresh (stale did.json key); real alert transport; an end-to-end registry-deactivation `inactive`
   path once a public `SetActive` lands; consolidating the now-3x overlay precedence into `internal/badge`.
