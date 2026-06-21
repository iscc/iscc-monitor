<!-- assessed-at: f8247cabe9202b8f1232d3d044e51e7797142358 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — the certificate path. The realm-wide
certificate-of-inclusion page (`internal/certificate`, `GET /inclusion/{iscc_id}`) skeleton has landed
(§1 Subject + decode→resolve→store-lookup chain, mounted in `main.go`) but the latest `review` verdict
is **NEEDS_WORK**: two `critical` correctness bugs in the page's headline behavior block PASS. M1/M2/M3
remain fully met. WASM and OTS not started.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: badge,
DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, paginated record list,
single record, the ISCC-IDv1 decoder, and the Hub-List resolver are all built + verified. The
certificate skeleton is the first slice of the last M-UI Verify criterion but is **not yet correct** —
it certifies leaves outside the accepted tree and looks up a bare id that never matches the stored
`ISCC:`-prefixed key.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): 1 still open, now partially attacked but not closed.**
    **Met:** five-status `HubStatusBadge` (`internal/badge`); the DS v2 shared shell (`/_ds/` tokens +
    self-hosted webfonts, CDN-free); the `/` realm-index grid; the `/<domain>/log/` log-browser; the
    **hub dossier** (`GET /<domain>`, `internal/dossier`); the **frozen Exhibit**; the **paginated
    record list** (`GET /<domain>/log/records`); the **single-record page** (`GET /record?index=<seq>`);
    the **ISCC-IDv1 decoder** (`internal/index.Decode`); the `(realm, hub_id) → domain` **Hub-List
    resolver** (`internal/registry`, fail-closed on path/query/fragment urls + missing `hub_id`).
    **In progress (NOT closed):** the **certificate of inclusion** HTML page at `/inclusion/{iscc_id}`
    — a skeleton (`internal/certificate.Handler`, route mounted in `cmd/iscc-monitor/main.go`) renders
    §1 Subject + the documented honesty 200 states and wires the decode→resolve→`ListHubs`→
    `SeqsForISCCID` chain, but its §1 inclusion claim is **unsound** (2 critical bugs, below). Clauses
    §2–§6 + the downloadable **proof-bundle assembler** are gated empty placeholders — not built. The
    separate **Bitcoin-anchor vs comparison-anchor** panels are not built.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js` in source,
    re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~6 milestone-Verify-advancing / ~4 refactor·polish·hardening.** The arc closed
  all four M3 criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid → log
  browser → dossier → frozen Exhibit → record list → single record → ISCC-IDv1 decoder → Hub-List
  resolver → resolver fail-open hardening → certificate skeleton). The certificate skeleton (this
  iteration's advance) is the first attempt at the one remaining M-UI Verify criterion — but it landed
  NEEDS_WORK with two critical correctness defects, so the criterion has not advanced from open to met.
  **Not drift**, but **note**: the loop has now spent three of the last four iterations
  (resolver → resolver-hardening → cert-skeleton-with-2-bugs) without closing a Verify criterion. The
  next iteration MUST fix the two critical certificate bugs (they belong in one slice with the §2
  Checkpoint sub-step) before any new surface, or the cert page stalls.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `f6bf809..HEAD` diff touched only `internal/certificate`
(new), `cmd/iscc-monitor/main.go` + `main_test.go` (route mount), `CLAUDE.md`, and context/learnings
docs — **no M1 source touched**. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all
three triggers (fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart
survival, coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **310 `func Test`** across **59** `_test.go` files (up from 300/58 — the
  certificate skeleton added `internal/certificate/handler_test.go` and `main_test.go` cases). Package
  count **19 internal + 2 cmd = 21** (up from 18 internal; `internal/certificate` is the new package).
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
  tests run against in-process fixtures. Stale `sb1.amlet.id` did.json drift captured in tests; not
  refreshed.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (direct, Hub-List parser).
  **Not wired:** `nbd-wtf/opentimestamps`, `github.com/iscc/iscc-lib/packages/go` (ADR-0011, not yet
  adopted — see Quality gates).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched in the `f6bf809..HEAD` diff. Both Verify
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
**Status**: **in progress — certificate skeleton landed but NEEDS_WORK (2 critical bugs); certificate
page not yet correct.** Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen
Exhibit, record list, single-record page, the ISCC-IDv1 decoder, and the Hub-List resolver are all
built and verified.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` log-browser; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`GET /record?index=<seq>`); the ISCC-IDv1 `internal/index.Decode`
  (`GOOS=js GOARCH=wasm`-buildable); the fail-closed `(realm, hub_id) → domain` Hub-List resolver in
  `internal/registry`.
- **Landed this iteration but NEEDS_WORK (review verdict at f8247ca):** the realm-wide **certificate of
  inclusion** skeleton — `internal/certificate.Handler`, a `GET /inclusion/{iscc_id}` subtree route
  mounted in `cmd/iscc-monitor/main.go`. It wires the decode→`hubList.Resolve`→`followedHub`/`ListHubs`
  →`SeqsForISCCID` chain, renders §1 Subject + the subject banner + all documented honesty 200 states
  (no-id / malformed / not-in-realm / not-followed / not-found), is buffer-then-200, 405 on non-GET,
  and links `/_ds/` with no third-party CDN. Clauses §2–§6 + the downloadable proof bundle are gated
  empty placeholders. The package's oracle gate is correctly N/A for the skeleton (pure HTML render of
  a decode + resolve + store lookup).
- **TWO `critical` bugs in the skeleton's headline §1 inclusion claim (both reviewer-confirmed against
  code; block PASS):**
  1. **No accepted-tree cap (`handler.go:200`):** `buildData` sets `Certifiable` on `len(seqs) > 0`
     alone, with NO `LastSize` cap. `PollHub` writes `iscc_index` projections BEFORE the
     consistency/freeze check and `AdvanceAccepted`, so a frozen/failed-poll leaf can be certified.
     The golden test even certifies with `LastSize == 0`. Fix: gate on `seqs[0] < FollowState.LastSize`
     like every sibling record route, else render the cannot-certify state.
  2. **Bare-id vs stored `ISCC:`-prefixed key (`handler.go:196`):** the handler passes the bare path
     suffix `rawID` to `SeqsForISCCID`, but production stores `iscc_id` verbatim and `ISCC:`-prefixed
     (`logclient/projection.go`), so a real declaration reports "not found in log". Tests pass only
     because the fixture seeds the BARE form (fixture matched to code, not ground truth). Fix:
     canonicalize to the stored prefixed form after decode.
- **Still open on the M-UI Verify bar (beyond the two bugs):** §2–§6 of the certificate (checkpoint,
  inclusion proof, signing key, Bitcoin anchor, record history) + the **downloadable proof-bundle
  assembler** (re-engages the oracle/conformance gate — `serveVerify` discards the raw checkpoint bytes
  + resolved hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels.
- **Residual fail-open (filed `normal`, NOT fixed):** `internal/registry` `hubDomain` (registry.go:188)
  does not check `u.ForceQuery`, so a bare trailing `?` slips the guard. Same class as the two
  already-closed resolver gaps; not exploitable (resolver not yet wired into a live caller). The cert
  skeleton uses `hubList.Resolve` but `cmd/iscc-monitor/main.go:131` notes production currently has no
  Hub-List wired (re-verify when the resolver's live source lands).
- Build source of truth: `.claude/design/ISCC Monitor - Certificate.dc.html` + the `_ds/` token bundle
  (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or source;
no `internal/proof` package; no WASM build target (`syscall/js` not in source). The `internal/badge`,
`internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves are WASM-shareable
primitives the verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **at-risk — HEAD has 2 open `critical` issues and is 4 commits ahead of the last CI run.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.24.0`**, no `toolchain` line);
  `mise run check` runnable. The latest `review` verdict (f8247ca) is **NEEDS_WORK / loop CONTINUE**:
  it records `mise run check` green (build + vet + all 22 packages incl. new `internal/certificate`;
  `gofmt -l .` empty; `go.mod`/`go.sum` byte-unchanged) and the cert package tests pass uncached — BUT
  it FILED two `critical` correctness bugs (the §1 inclusion claim is unsound). The gate is mechanically
  green; the milestone Verify is not met.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest concluded run:
  `conclusion: success` at headSha `f6bf809` — which is `origin/develop`, NOT HEAD.** HEAD (`f8247ca`)
  and its 3 predecessor commits (the cert define-next/advance/review + a prior update-state) are
  **unpushed**, so the certificate skeleton has NOT been CI-verified. (Expected: review withheld the
  push under NEEDS_WORK.)
- **TARGET/CODE GAP (ADR-0011, filed `normal`):** `target.md` "Stack (locked)" mandates **Go 1.26** +
  the **`github.com/iscc/iscc-lib/packages/go` v0.5.0** codec, but `go.mod` is still `go 1.24.0` with no
  iscc-lib require. Deliberate, sequenced, not-yet-started increment — target and code disagree on the
  stack until it lands. Local toolchain is go1.24.x; the bump must run where mise can provision Go 1.26.
- **Open issues: 2 `critical`, 2 `normal`, 5 `low`.** The 2 `critical` (both from this iteration's
  review): (1) certificate §1 certifies leaves outside the accepted tree (no `LastSize` cap); (2)
  certificate path-suffix id mismatched against the stored `ISCC:`-prefixed key. The 2 `normal`:
  (a) [human] adopt iscc-lib codec + bump to Go 1.26 (ADR-0011); (b) [review/Codex] Hub-List
  `hubDomain` `ForceQuery` fail-open. The 5 `low` are loop-skipped.

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone, but the latest review is NEEDS_WORK with 2 open
`critical` issues — these preempt everything.** The next `define-next`/`advance` MUST:

1. **Fix the two `critical` certificate defects in ONE slice** (`internal/certificate/handler.go`
   `buildData`): (a) add the accepted-tree cap — read `FollowState` and only set `Certifiable` when
   `len(seqs) > 0 && seqs[0] < fs.LastSize`, else render the cannot-certify "not in accepted tree" /
   "no accepted checkpoint yet" state; (b) canonicalize the lookup id to the stored `ISCC:`-prefixed
   form. **Re-ground the fixtures** so the golden test indexes the leaf under `"ISCC:MAIG…"` and seeds
   an accepted checkpoint covering it — proving the real production path, not a fixture matched to the
   code. The review recommends folding this with the §2 Checkpoint sub-step (it reads `FollowState`
   anyway). The reviewer must mutation-prove both fixes non-vacuous and confirm the gate stays green
   before any push.
2. **Then continue the certificate path:** §2–§6 clauses + the downloadable **proof-bundle assembler**
   `{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` — this re-engages the
   oracle/conformance gate (the served bundle's inclusion proof must be mutation-proven non-vacuous,
   `notecheck`/golden-vector parity confirmed). Plus the separate Bitcoin-anchor vs comparison-anchor
   panels. Fold in the deferred `ForceQuery` registry fix when `hubDomain` is next touched.
3. **The human-filed ADR-0011 stack bump** (Go 1.24 → 1.26 + adopt `iscc-lib` v0.5.0 + the
   `internal/index` tripwire/parity test) — flagged foundational and "before more M-UI feature work."
   Confirm the toolchain (mise must provision Go 1.26) before flipping `go.mod`'s `go` directive.
4. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
5. **Off the Verify bar (`low`):** harden the vacuous single-record label test; sb1 fixture refresh;
   real alert transport; `inactive` public path; consolidate the now-3x overlay precedence into
   `internal/badge`; `notecheck` `out` param; proofserve `os.ErrNotExist`→404 dedup; mirror write-path
   tile-coord leak; scaling trip-wire metrics.
