<!-- assessed-at: 27f4804c34d5c7d878714824a5aa66b270c54d79 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — certificate clause-by-clause, BLOCKED on a critical.
The certificate of inclusion (`internal/certificate`, `GET /inclusion/{iscc_id}`) now renders four of
its six numbered clauses' code: §1 SUBJECT + §2 CHECKPOINT are sound and PASS-verified; **§3 INCLUSION
PROOF landed this cycle but is UNSOUND** — the latest `review` verdict is **NEEDS_WORK** with a new
**critical** issue (a frozen-after-fork hub can render a self-contradictory proof). M1/M2/M3 remain
fully met. §4–§6 + the downloadable proof bundle, WASM, and OTS remain.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): 1 still open.** Met: five-status `HubStatusBadge`
    (`internal/badge`); DS v2 shared shell (`/_ds/` tokens + self-hosted webfonts, CDN-free); `/`
    realm-index grid; `/<domain>/log/` log browser; hub dossier (`GET /<domain>`, `internal/dossier`);
    frozen Exhibit; paginated record list (`GET /<domain>/log/records`); single-record page
    (`GET /record?index=<seq>`); ISCC-IDv1 decoder (`internal/index.Decode`); `(realm, hub_id) → domain`
    Hub-List resolver (`internal/registry`); certificate **§1 SUBJECT + §2 CHECKPOINT** (sound,
    PASS-verified). **Open / unsound:** certificate **§3 INCLUSION PROOF code present but
    NEEDS_WORK** (critical — built ≠ verified against the accepted root; ignores `hub.Frozen`); clauses
    **§4–§6** (signing key, Bitcoin anchor, record history) not started; the downloadable **proof-bundle
    assembler** not started; the separate **Bitcoin-anchor vs comparison-anchor** panels.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js` in source).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum`/source).
- **Last ~10 iterations: ~7 milestone-Verify-advancing / ~3 refactor·polish·hardening.** The arc is
  building the certificate clause-by-clause (badge → DS tokens → webfonts → `/` grid → log browser →
  dossier → frozen Exhibit → record list → single record → ISCC-IDv1 decoder → Hub-List resolver →
  cert §1 → §2 → §3). §1/§2 each PASS-verified + mutation-proven; §3 advanced but bounced to NEEDS_WORK
  on a real trust-root defect. No drift — the loop is converging on real Verify criteria, and the gate
  correctly caught (via review + Codex) a defect the automated checks miss.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `17c4957..HEAD` diff touched ONLY `internal/certificate`
(handler.go + cert.html + handler_test.go) and context/learnings docs — **no M1 source touched.** All
M1 Verify criteria remain satisfied: `origin`/`vkey` golden; all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival;
coverage tracked; structured logs; `/metrics` served over HTTP.
- **Test totals at HEAD**: **314 `func Test`** across **59** `_test.go` files (the §3 advance added two
  new cert test funcs — `TestCertificateInclusionProof`, `TestCertificateInclusionProofTileGap`).
  Package count **19 internal + 2 cmd = 21**.
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
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (Hub-List parser).
  **Not wired:** `nbd-wtf/opentimestamps`, `github.com/iscc/iscc-lib/packages/go` (ADR-0011, not yet
  adopted — see Quality gates).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched in the `17c4957..HEAD` diff. Both Verify
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
**Status**: **in progress — BLOCKED on a critical.** §1 SUBJECT + §2 CHECKPOINT are sound and
PASS-verified; **§3 INCLUSION PROOF is landed-but-unsound (NEEDS_WORK)**; §4–§6 + the proof bundle are
not started. Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, record
list, single-record page, the ISCC-IDv1 decoder, and the Hub-List resolver are all built and verified.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` browser; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`GET /record?index=<seq>`); the ISCC-IDv1 `internal/index.Decode`
  (`GOOS=js GOARCH=wasm`-buildable); the fail-closed `(realm, hub_id) → domain` Hub-List resolver.
- **§1 SUBJECT (PASS, carried forward):** `internal/certificate.Handler` decodes the id → `hubList.Resolve`
  → `followedHub`/`ListHubs` → `SeqsForISCCID`, with the accepted-tree cap (`seqs[0] < LastSize`) and the
  canonicalized `"ISCC:" + TrimPrefix` lookup key.
- **§2 CHECKPOINT (PASS at 17c4957, carried forward):** for a certifiable id, `buildData` reads the
  accepted root via `CheckpointAt(LastSize)`, sets `HasClause2` + `CheckpointSize`/`CheckpointRoot`
  (base64-Std); `cert.html` renders `§2 CHECKPOINT — size N · root <b64>` only when `HasClause2`.
  Mutation-proven non-vacuous (verified §1/§2 in prior reviews).
- **§3 INCLUSION PROOF (LANDED but NEEDS_WORK at HEAD `27f4804`):** `handler.go:325` builds the RFC-6962
  proof of `data.Position` against `hub.LastSize` via `logclient.InclusionProofFromTiles` over a
  `SQLiteFetcher`, base64-Std encodes each sibling, and `cert.html:345-355` renders the leaf→siblings→root
  chain. The test is non-vacuous (review reproduced both mutations). **BUT** `handler.go:336-343` sets
  `HasClause3 = true` whenever the build succeeds and §2 holds — it NEVER verifies the proof rebuilds the
  accepted root and IGNORES `hub.Frozen` (in hand). For a frozen-after-fork hub the mirror can hold the
  contradictory tree's tiles while the accepted root is the old one, so §3 renders a sibling chain under a
  `root … ✓` the siblings do not rebuild — a self-contradictory certificate on the trust-root surface.
  This is the **open critical issue**; review + Codex both confirmed it against the follower
  freeze/ingest ordering.
- **Still open on the M-UI Verify bar:** **fix §3 first** (verify the proof rebuilds `CheckpointRoot` via
  `proof.VerifyInclusion` before `HasClause3 = true`, or gate §3 on `!hub.Frozen`; add a
  contradictory-tile frozen-hub fixture test, mutation-proven). Then clauses **§4–§6** (signing key,
  Bitcoin anchor, record history) + the **downloadable proof-bundle assembler**
  `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}`; the separate Bitcoin-anchor
  vs comparison-anchor panels. `HasClause4..6` are all still `false`.
- **Residual fail-open (filed `normal`, NOT fixed):** `internal/registry` `hubDomain` (registry.go:188)
  does not check `u.ForceQuery`, so a bare trailing `?` slips the guard. Not exploitable (resolver not
  yet wired into a live caller). Fold in when `hubDomain` is next touched.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or source;
no `internal/proof` package; no WASM build target (`syscall/js` not in source). The `internal/badge`,
`internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves are WASM-shareable
primitives the verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **gate green on the §3 commit, but an open `critical` BLOCKS the milestone.** The automated
gate (`mise run check`) passes on the §3 advance — the review recorded build + vet + all 21 packages
`ok`, `gofmt -l .` empty, `go.mod`/`go.sum` byte-unchanged, and the oracle/conformance gate (notecheck +
`derive_vkey.py`) green. **But the latest `review` verdict (HEAD `27f4804`) is NEEDS_WORK / loop
CONTINUE** — the §3 honesty defect is a class the automated gate cannot catch (a green-but-wrong proof
render), which is exactly why the oracle/LLM review exists. **Not DONE-eligible: one open `critical`.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.24.0`**, no `toolchain` line);
  `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`, branch `develop`. **The §3
  cycle is UNPUSHED (correct — NEEDS_WORK does not push):** `origin/develop` is at `17c4957`, and the four
  local commits (§2 update-state + the §3 define-next/advance/review) are ahead of it. **Latest concluded
  CI run: `success` at `17c4957`** — i.e. CI is green at the last PASSed state, NOT at HEAD's unsound §3.
- **TARGET/CODE GAP (ADR-0011, filed `normal`):** `target.md` "Stack (locked)" mandates **Go 1.26** + the
  **`iscc-lib/packages/go` v0.5.0** codec, but `go.mod` is still `go 1.24.0` with no iscc-lib require.
  Deliberate, sequenced, not-yet-started increment — target and code disagree on the stack until it lands.
- **Open issues: 1 `critical`, 2 `normal`, 6 `low`.** The 1 `critical`: §3 self-contradictory proof for a
  frozen-after-fork hub (preempts everything). The 2 `normal`: (a) [human] adopt iscc-lib codec + bump to
  Go 1.26 (ADR-0011, foundational); (b) [review/Codex] Hub-List `hubDomain` `ForceQuery` fail-open. The 6
  `low` are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone and is BLOCKED by one open `critical`.** The next
`define-next`/`advance` MUST:

1. **Fix the §3 critical FIRST (preempts all feature work).** In `buildData`'s §3 branch, before
   `HasClause3 = true`, verify the built proof rebuilds `data.CheckpointRoot` via `proof.VerifyInclusion`
   (leaf hash from the mirrored entry bundle + the decoded accepted root) — the fail-closed choice for a
   self-verifiable artifact — OR gate §3 on `!hub.Frozen`. Add a frozen-hub / contradictory-tile fixture
   test that asserts §1+§2 render but NO §3 `✓`, mutation-proven (reverting the guard makes it FAIL).
   Then push so CI re-greens at the real HEAD.
2. **Resume the certificate clauses** — §4 SIGNING KEY (did:web key via `hub_keys`/`LookupHubKey`), §5
   Bitcoin anchor, §6 record history + the proof-bundle assembler (re-engages the oracle gate) + the
   separate Bitcoin-anchor vs comparison-anchor panels. Fold in the deferred `ForceQuery` registry fix
   when `hubDomain` is next touched.
3. **The human-filed ADR-0011 stack bump** (Go 1.24 → 1.26 + adopt `iscc-lib` v0.5.0 + the `internal/index`
   tripwire/parity test) — flagged foundational; confirm mise can provision Go 1.26 before flipping
   `go.mod`'s `go` directive.
4. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
