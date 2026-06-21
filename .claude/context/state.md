<!-- assessed-at: 5fb245435f44160df949dcadaf904e26be12b377 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — certificate clause-by-clause; §3 critical CLOSED.
The certificate of inclusion (`internal/certificate`, `GET /inclusion/{iscc_id}`) now has three of its
six numbered clauses sound and PASS-verified: §1 SUBJECT, §2 CHECKPOINT, and **§3 INCLUSION PROOF —
which the last review (`96f6ed9`) closed the open `critical` on by replacing the racy `!hub.Frozen`
flag with a fail-closed `proof.VerifyInclusion` against the §2 accepted root** (mutation-proven, Codex
clean, pushed to `origin/develop`, CI green). M1/M2/M3 remain fully met. §4–§6 + the downloadable proof
bundle, the WASM verifier, and OTS anchoring remain.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): 1 still open** (certificate §4–§6 + proof-bundle + the separate
    Bitcoin-anchor/comparison-anchor panels + the M-UI exit visual-pass gate). Met: five-status
    `HubStatusBadge` (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted webfonts,
    CDN-free); `/` realm-index grid; `/<domain>/log/` log browser; hub dossier (`GET /<domain>`,
    `internal/dossier`); frozen Exhibit; paginated record list (`GET /<domain>/log/records`);
    single-record page (`GET /<domain>/log/record?index=<seq>`); ISCC-IDv1 decoder
    (`internal/index.Decode`); `(realm, hub_id) → domain` Hub-List resolver (`internal/registry`);
    certificate **§1 SUBJECT + §2 CHECKPOINT + §3 INCLUSION PROOF** (all sound, PASS-verified). **Open:**
    certificate **§4–§6** (signing key, Bitcoin anchor, record history; `HasClause4..6` are all still
    `false`, never set); the downloadable **proof-bundle assembler**; the separate Bitcoin-anchor vs
    comparison-anchor panels; the mandatory M-UI exit visual-pass + human sign-off (ADR-0012).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js` in source).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum`/source).
- **Last ~10 iterations: ~7 milestone-Verify-advancing / ~3 refactor·polish·tooling.** The arc is
  building the certificate clause-by-clause (badge → DS tokens → webfonts → `/` grid → log browser →
  dossier → frozen Exhibit → record list → single record → ISCC-IDv1 decoder → Hub-List resolver →
  cert §1 → §2 → §3). §1/§2/§3 each PASS-verified + mutation-proven; §3 took three cycles (render →
  freeze-gate → fail-closed re-verify) to close a real trust-root defect the automated gate could not
  catch (green-but-wrong proof render). No drift — the loop is converging on real Verify criteria, and
  the gate (review + Codex) correctly held §3 at NEEDS_WORK until the proof was verified, not just built.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `87950569..HEAD` diff touched ONLY `internal/certificate`
(handler.go + handler_test.go) and `.claude/*` context/tooling/docs — **no M1 source touched.** All M1
Verify criteria remain satisfied: `origin`/`vkey` golden; all three triggers (fork/shrink/equivocation)
golden-tested end-to-end with freeze + alert-once + restart survival; coverage tracked; structured logs;
`/metrics` served over HTTP.
- **Test totals at HEAD**: **315 `func Test`** across **59** `_test.go` files. Package count
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
  tests run against in-process fixtures.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`+`proof.VerifyInclusion`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (Hub-List parser).
  **Not wired:** `nbd-wtf/opentimestamps`, `github.com/iscc/iscc-lib/packages/go` (ADR-0011, not yet
  adopted — see Quality gates).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched in the diff. Both Verify criteria exercised:
fsck root-rebuild WIRED on every verified non-frozen poll; inclusion cross-check conformance-tested over
the real verified mirror. All three computed proofs — `inclusion`, `consistency`, `entries` — served
from the local mirror, never re-hitting the hub.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 source touched. CORS on every public
GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index dashboard;
`GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry `no-cache` +
strong ETag + 304).

## M-UI — Evidence Ledger frontend
**Status**: **in progress — §3 critical CLOSED; §4–§6 + proof bundle + visual-exit gate remain.** §1
SUBJECT, §2 CHECKPOINT, and §3 INCLUSION PROOF are all sound and PASS-verified. Badge, DS shell, `/`
index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, record list, single-record page, the
ISCC-IDv1 decoder, and the Hub-List resolver are all built and verified.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` browser; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`GET /<domain>/log/record?index=<seq>`); the ISCC-IDv1 `internal/index.Decode`
  (`GOOS=js GOARCH=wasm`-buildable); the fail-closed `(realm, hub_id) → domain` Hub-List resolver.
- **§1 SUBJECT (PASS):** `internal/certificate.Handler` decodes the id → `hubList.Resolve` →
  `followedHub`/`ListHubs` → `SeqsForISCCID`, with the accepted-tree cap (`seqs[0] < LastSize`) and the
  canonicalized `"ISCC:" + TrimPrefix` lookup key.
- **§2 CHECKPOINT (PASS):** for a certifiable id, `buildData` reads the accepted root via
  `CheckpointAt(LastSize)`, sets `HasClause2` + `CheckpointSize`/`CheckpointRoot` (base64-Std); `cert.html`
  renders §2 only when `HasClause2`. Mutation-proven non-vacuous.
- **§3 INCLUSION PROOF (PASS — critical CLOSED at `96f6ed9`):** `handler.go:335-420` builds the RFC-6962
  proof of `data.Position` against `hub.LastSize` via `logclient.InclusionProofFromTiles` over a
  `SQLiteFetcher`, then reads the subject leaf's raw bytes from the mirrored entry bundle
  (`ReadEntryBundle` → `RecordBytesFromBundle` → `HashLeaf`) and gates `HasClause3 = true` on
  `proof.VerifyInclusion(...) == nil` against the §2 accepted root (`handler.go:410`). The rendered ✓ is
  now true by construction — it fails closed against ANY tile↔root divergence (steady-state frozen AND
  the fork-poll TOCTOU race the previous freeze-flag gate could not), with no status-flag read. Honest
  gaps (tile/bundle not yet mirrored = `os.ErrNotExist`, `ErrLeafOutOfBundle`, or a non-rebuilding proof)
  silently decline §3 at a 200; only a genuine read fault is a 500. Mutation-proven by
  `TestCertificateInclusionProofContradictory` (mirror tree A, accept tree B's root → §1+§2 render, no §3
  ✓; reverting the verify guard makes it FAIL). Codex independently found no bugs.
- **Still open on the M-UI Verify bar:** clauses **§4–§6** (§4 signing key via did:web `hub_keys`/
  `LookupHubKey`; §5 Bitcoin anchor; §6 record history incl. any deletion) + the **downloadable
  proof-bundle assembler** `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}`
  (re-engages the oracle/conformance gate — shares the §3 crypto path); the separate **Bitcoin-anchor vs
  comparison-anchor** panels; and the **mandatory M-UI exit visual-pass + human sign-off** (ADR-0012
  agent-browser visual verification, now tooled — `5fb2454`). `HasClause4..6` are all still `false`.
- **Residual fail-open (filed `normal`, NOT fixed):** `internal/registry` `hubDomain` (registry.go:188)
  does not check `u.ForceQuery`, so a bare trailing `?` slips the guard. Not exploitable (resolver not
  yet wired into a live caller). Fold in when `hubDomain` is next touched.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or source;
no `internal/proof` package; no WASM build target (`syscall/js` not in source). The `internal/badge`,
`internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves are WASM-shareable
primitives the verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **green at the source HEAD; no open `critical`.** The §3 fail-closed fix passed `review`
(`96f6ed9`, PASS / CONTINUE) and is pushed; CI is **green at `96f6ed9`** (`gh run list`: latest concluded
run `success` at that SHA). The only commit ahead of `origin/develop` is `5fb2454` (ADR-0012 agent-browser
tooling/docs — `.claude/*` + `.devcontainer/Dockerfile`, **no Go source**), so CI parity holds for the
build.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.24.0`**, no `toolchain` line);
  `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`.
  `origin/develop` = `96f6ed9`; CI concluded `success` there. The trailing `5fb2454` doc/tooling commit is
  unpushed but touches no code path the gate runs.
- **TARGET/CODE GAP (ADR-0011, filed `normal`):** `target.md` "Stack (locked)" mandates **Go 1.26** + the
  **`iscc-lib/packages/go` v0.5.0** codec, but `go.mod` is still `go 1.24.0` with no iscc-lib require.
  Deliberate, sequenced, not-yet-started increment — target and code disagree on the stack until it lands.
- **Open issues: 0 `critical`, 2 `normal`, 6 `low`.** The §3 TOCTOU `critical` was deleted as
  verified-fixed by the last review. The 2 `normal`: (a) [human] adopt iscc-lib codec + bump to Go 1.26
  (ADR-0011, foundational); (b) [review/Codex] Hub-List `hubDomain` `ForceQuery` fail-open. The 6 `low`
  are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone with no blocker.** The next `define-next`/`advance` SHOULD:

1. **Resume the certificate clauses** — §4 SIGNING KEY (did:web key via `hub_keys`/`LookupHubKey`), §5
   Bitcoin anchor, §6 record history (incl. any deletion), then the **downloadable proof-bundle assembler**
   `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` (re-engages the oracle/
   conformance gate — shares the §3 build+verify crypto path), plus the separate Bitcoin-anchor vs
   comparison-anchor panels. Fold in the deferred `ForceQuery` registry fix when `hubDomain` is next
   touched. Push `5fb2454` opportunistically (docs/tooling, no gate impact).
2. **The human-filed ADR-0011 stack bump** (Go 1.24 → 1.26 + adopt `iscc-lib` v0.5.0 + the `internal/index`
   tripwire/parity test) — flagged foundational; confirm mise can provision Go 1.26 before flipping
   `go.mod`'s `go` directive.
3. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
4. **M-UI exit gate (ADR-0012):** before M-UI is DONE, every SSR surface must pass the agent-browser
   visual pass with deviations filed and a human sign-off.
