<!-- assessed-at: 340303b9cfc974659d0ab3d3ea4fbd4fe22d8313 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — the certificate path.
The certificate-of-inclusion page (`internal/certificate`, `GET /inclusion/{iscc_id}`) §1 SUBJECT clause
is now **sound and PASS-verified**: both prior `critical` bugs (no accepted-tree cap; bare-id vs stored
`ISCC:`-prefixed key) are fixed, mutation-proven non-vacuous, and the issues are closed. M1/M2/M3 remain
fully met. WASM and OTS not started.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: badge,
DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, paginated record list,
single record, the ISCC-IDv1 decoder, and the Hub-List resolver are all built + verified. The certificate
§1 Subject clause (the first slice of the last M-UI Verify criterion) is now correct; clauses §2–§6 and
the downloadable proof bundle remain to be built.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): 1 still open, materially advanced this iteration.**
    **Met:** five-status `HubStatusBadge` (`internal/badge`); the DS v2 shared shell (`/_ds/` tokens +
    self-hosted webfonts, CDN-free); the `/` realm-index grid; the `/<domain>/log/` log-browser; the
    **hub dossier** (`GET /<domain>`, `internal/dossier`); the **frozen Exhibit**; the **paginated
    record list** (`GET /<domain>/log/records`); the **single-record page** (`GET /record?index=<seq>`);
    the **ISCC-IDv1 decoder** (`internal/index.Decode`); the `(realm, hub_id) → domain` **Hub-List
    resolver** (`internal/registry`); and now the **certificate §1 SUBJECT clause** — the
    `/inclusion/{iscc_id}` page decodes the id, resolves the hub, canonicalizes to the stored
    `ISCC:`-prefixed key, gates the affirmative inclusion claim on the accepted-tree cap
    (`seqs[0] < LastSize`), and renders honest cannot-certify states otherwise.
    **Still open (NOT closed):** certificate clauses **§2–§6** (checkpoint, inclusion proof, signing
    key, Bitcoin anchor, record history) + the downloadable **proof-bundle assembler** (re-engages the
    oracle/conformance gate at §3); the separate **Bitcoin-anchor vs comparison-anchor** panels.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js` in source,
    re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~7 milestone-Verify-advancing / ~3 refactor·polish·hardening.** The arc closed
  all four M3 criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid → log
  browser → dossier → frozen Exhibit → record list → single record → ISCC-IDv1 decoder → Hub-List
  resolver → cert skeleton → **cert §1 soundness fix**). The cert skeleton landed NEEDS_WORK with two
  critical defects last iteration; **this iteration's advance closed both** (PASS, mutation-proven,
  Codex clean) — the prior polish-streak concern is cleared. The certificate is now correctly building
  out clause-by-clause; no drift.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `f8247ca..HEAD` diff touched ONLY `internal/certificate`
(handler + tests), `cmd/iscc-monitor/main_test.go`, and context/learnings docs — **no M1 source
touched.** All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **312 `func Test`** across **59** `_test.go` files (up from 310/59 — the
  certificate §1 fix added test cases to `handler_test.go` + `main_test.go`). Package count
  **19 internal + 2 cmd = 21**.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **19 internal packages** —
  `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient,
  metrics, metricshttp, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- All three triggers WIRED + golden-tested inside `logclient.CheckConsistency`; `AcceptCheckpoint`
  4-way verdict threads `VerifiedContext` into the hub-key cache upsert + `fsckMirror`. `cmd/notecheck`
  is the fully-independent signature-parity oracle, shelled out in CI. `store/*.go` uses
  `modernc.org/sqlite` with ADR-0005/0007 single-writer discipline.
- **Missing (M1 connective tissue, off the Verify bar):** real alert transport (`alertFunc` is a WARN
  `slog` emit); warm-path second `did.json` resolve.
- **Fixtures**: `testdata/live/` (repo root) still holds only the two checkpoints — no tiles, entry
  bundles, or did.json. All proof/dashboard/browser/badge/web/dossier/records/registry/certificate
  tests run against in-process fixtures. Stale `sb1.amlet.id` did.json drift captured in tests.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (direct, Hub-List parser).
  **Not wired:** `nbd-wtf/opentimestamps`, `github.com/iscc/iscc-lib/packages/go` (ADR-0011, not yet
  adopted — see Quality gates).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched in the `f8247ca..HEAD` diff. Both Verify
criteria exercised: fsck root-rebuild WIRED on every verified non-frozen poll; inclusion cross-check
conformance-tested over the real verified mirror. All three computed proofs — `inclusion`,
`consistency`, `entries` — served from the local mirror, never re-hitting the hub.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard; `GET
/<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry `no-cache` +
strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — certificate §1 SUBJECT clause now sound and PASS-verified; §2–§6 + proof
bundle remain.** Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit,
record list, single-record page, the ISCC-IDv1 decoder, and the Hub-List resolver are all built and
verified.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` log-browser; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`GET /record?index=<seq>`); the ISCC-IDv1 `internal/index.Decode`
  (`GOOS=js GOARCH=wasm`-buildable); the fail-closed `(realm, hub_id) → domain` Hub-List resolver in
  `internal/registry`.
- **Advanced + PASS this iteration (review verdict at 340303b):** the **certificate §1 SUBJECT clause**.
  `internal/certificate.Handler` (`GET /inclusion/{iscc_id}` subtree, mounted in
  `cmd/iscc-monitor/main.go`) decodes the id → `hubList.Resolve` → `followedHub`/`ListHubs` →
  `SeqsForISCCID`, with both prior `critical` bugs FIXED:
  - **Accepted-tree cap (`handler.go:236-245`):** `Certifiable` now gates on
    `len(seqs) > 0 && seqs[0] < hub.LastSize`; `LastSize == 0` → "no accepted checkpoint yet";
    `seqs[0] >= LastSize` → "not in accepted tree". `LastSize` is carried out of the existing `ListHubs`
    scan via `followedHub` returning a `store.HubSummary` (no second round-trip).
  - **Prefixed-key lookup (`handler.go:218`):** the handler canonicalizes to the stored
    `"ISCC:" + TrimPrefix(rawID, "ISCC:")` form, matching what `logclient/projection.go` writes.
  Both fixes mutation-proven non-vacuous by review; fixtures re-grounded to production wire format
  (prefixed id + an accepted checkpoint). Oracle gate correctly N/A for this slice (pure HTML render of
  decode + resolve + store read).
- **Still open on the M-UI Verify bar:** certificate clauses **§2–§6** (checkpoint, inclusion proof,
  signing key, Bitcoin anchor, record history) + the **downloadable proof-bundle assembler**
  `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` (re-engages the
  oracle/conformance gate at §3 — `serveVerify` discards the raw checkpoint bytes + resolved hub key the
  bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels.
- **Residual fail-open (filed `normal`, NOT fixed):** `internal/registry` `hubDomain` (registry.go:188)
  does not check `u.ForceQuery`, so a bare trailing `?` slips the guard. Not exploitable (resolver not
  yet wired into a live caller; `cmd/iscc-monitor/main.go` notes production has no Hub-List source wired
  yet). Fold in when `hubDomain` is next touched.
- Build source of truth: `.claude/design/ISCC Monitor - Certificate.dc.html` + the `_ds/` token bundle
  (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or source;
no `internal/proof` package; no WASM build target (`syscall/js` not in source). The `internal/badge`,
`internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves are WASM-shareable
primitives the verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **green.** HEAD is PASS-verified, CI-confirmed, and has no open `critical`/`normal`-blocking
issue beyond the two deliberately-sequenced `normal`s below.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.24.0`**, no `toolchain` line);
  `mise run check` runnable. The latest `review` verdict (340303b) is **PASS / loop CONTINUE**: it
  records `mise run check` green (build + vet + all 21 packages `ok` uncached; `gofmt -l .` empty;
  `go.mod`/`go.sum` byte-unchanged), the cert package tests pass uncached (11 tests), and both critical
  fixes mutation-proven non-vacuous with a clean Codex second opinion.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck`
  oracle shell-out on push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`
  in sync with `origin/develop` (**nothing unpushed**). **Latest concluded run: `conclusion: success`
  at headSha `340303b` = HEAD.** The certificate fix IS CI-verified.
- **TARGET/CODE GAP (ADR-0011, filed `normal`):** `target.md` "Stack (locked)" mandates **Go 1.26** +
  the **`github.com/iscc/iscc-lib/packages/go` v0.5.0** codec, but `go.mod` is still `go 1.24.0` with no
  iscc-lib require. Deliberate, sequenced, not-yet-started increment — target and code disagree on the
  stack until it lands. The bump must run where mise can provision Go 1.26.
- **Open issues: 0 `critical`, 2 `normal`, 6 `low`.** Both prior `critical` certificate issues are
  CLOSED. The 2 `normal`: (a) [human] adopt iscc-lib codec + bump to Go 1.26 (ADR-0011); (b)
  [review/Codex] Hub-List `hubDomain` `ForceQuery` fail-open. The 6 `low` are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone and the gate is green with no critical issues.** The next
`define-next`/`advance` should:

1. **Continue the certificate path: §2 Checkpoint clause** — render the accepted `(size, root)` the §1
   cap already keys on, reusing the `HubSummary.LastSize` carry (read `FollowState` /
   `CheckpointAt(hubID, LastSize)`). Oracle gate stays N/A for §2 (pure store read + render).
2. **§3 Inclusion proof + the downloadable proof-bundle assembler** — this RE-ENGAGES the
   oracle/conformance gate: the served bundle's inclusion proof must be mutation-proven non-vacuous
   against the hub's `IsccLogInclusionProof` / `notecheck` / golden vectors before a PASS. Then §4–§6
   (signing key, Bitcoin anchor, record history) + the separate Bitcoin-anchor vs comparison-anchor
   panels. Fold in the deferred `ForceQuery` registry fix when `hubDomain` is next touched.
3. **The human-filed ADR-0011 stack bump** (Go 1.24 → 1.26 + adopt `iscc-lib` v0.5.0 + the
   `internal/index` tripwire/parity test) — flagged foundational and "before more M-UI feature work."
   Confirm the toolchain (mise must provision Go 1.26) before flipping `go.mod`'s `go` directive.
4. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
5. **Off the Verify bar (`low`):** harden the vacuous single-record label test; sb1 fixture refresh;
   real alert transport; `inactive` public path; consolidate the now-3x overlay precedence into
   `internal/badge`; `notecheck` `out` param; proofserve `os.ErrNotExist`→404 dedup; mirror write-path
   tile-coord leak; scaling trip-wire metrics.
