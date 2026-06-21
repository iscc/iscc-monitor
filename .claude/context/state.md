<!-- assessed-at: f6bf80920ab85043d1fe62043e30dbcf24a5db3f -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) — the certificate path. The `(realm, hub_id) →
domain` Hub-List resolver (`internal/registry.ParseHubList`/`Resolve`) is built and its two original
fail-opens are now hardened (path-bearing url + missing `hub_id`, both mutation-proven), but a third
`ForceQuery` fail-open in the same guard remains open (`normal`). M1/M2/M3 fully met. The
certificate-of-inclusion HTML page + proof-bundle assembler (re-engages the crypto/oracle gate) and the
anchor panels are still open. WASM and OTS not started.

The monitor's read-only/aggregator/trust-API core (M1, M2, M3) is fully met. M-UI is mid-flight: badge,
DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, paginated record list,
single record, the ISCC-IDv1 decoder, and the Hub-List resolver (now fail-closed on path-url and
missing-`hub_id`) are all built + verified. The remaining M-UI slices are the certificate HTML page +
proof-bundle assembler and the Bitcoin-anchor / comparison-anchor panels.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~1 still open.** **Met:** five-status `HubStatusBadge`
    (`internal/badge`); the DS v2 shared shell (`/_ds/` tokens + self-hosted webfonts, CDN-free); the
    `/` realm-index grid; the `/<domain>/log/` log-browser redress; the **hub dossier** (`GET
    /<domain>`, `internal/dossier.Handler`); the **frozen Exhibit**; the **paginated record list**
    (`GET /<domain>/log/records`); the **single-record page** (`GET /record?index=<seq>`); the
    certificate's trust-root **ISCC-IDv1 decoder** (`internal/index.Decode`). The `(realm, hub_id) →
    domain` **Hub-List resolver** (`internal/registry.ParseHubList`/`Hub`/`HubList`/`Resolve` +
    `testnet.yaml` golden) is built and now **fail-closed** on path/query/fragment urls and on a missing
    `hub_id` (`Hub.HubID` is `*uint16`; both guards mutation-proven this iteration). **Still open:** the
    **certificate of inclusion** HTML page at `/inclusion/{iscc_id}` (numbered evidence clauses §1–§6 +
    tier-1/tier-2 affordance) + the **downloadable proof-bundle assembler** (re-engages the
    oracle/conformance gate — `serveVerify` discards the raw checkpoint bytes + resolved hub key the
    bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels. The resolver is built but
    **not yet wired** into any caller (re-verified: no `ParseHubList`/`.Resolve(` reference outside
    `internal/registry`).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js` in source,
    re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or
    source, re-verified).
- **Last ~10 iterations: ~7 milestone-Verify / ~3 refactor·polish·hardening.** The arc closed all four
  M3 criteria, then opened M-UI leaf-first (badge → DS tokens → webfonts → `/` grid → log-browser
  redress → hub dossier → frozen Exhibit → record list → single record → ISCC-IDv1 decoder →
  Length-nibble guard → Hub-List resolver → resolver fail-open hardening). Not drift: every step targets
  a named M-UI Verify criterion, the resolver being the explicitly-named intermediate before the
  certificate page. **Note:** the last two iterations (resolver, then its hardening) were both
  trust-root-adjacent correctness work rather than a new Verify-closing surface — and the hardening
  iteration *itself* surfaced a third fail-open (`ForceQuery`). This is legitimate detect→fix
  discipline, but the certificate page (the next actual M-UI Verify criterion) has not advanced in two
  iterations; `define-next` should weigh closing the one cheap residual against pushing onto the cert
  page so the loop does not settle into an open-ended fail-open-polish streak on an unwired leaf.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `4a521ca..HEAD` diff touched only `internal/registry`
(`registry.go` +59, `hublist_test.go` +48), the ADR-0011 doc, and context/design docs — **no M1 source
touched**. All M1 Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP.
- **Test totals at HEAD**: **300 `func Test`** across **58** `_test.go` files (unchanged from last
  assessment — the hardening added test *cases* and adapted struct-literal comparisons, not net-new
  test functions). Package count **18 internal + 2 cmd** (unchanged).
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **18 internal packages** —
  `badge, config, corsmw, dashboard, didweb, dossier, follower, healthz, index, logclient, metrics,
  metricshttp, proofserve, registry, store, tiles, tilesserve, web`. Module
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- All three triggers WIRED + golden-tested inside `logclient.CheckConsistency`; `AcceptCheckpoint`
  4-way verdict threads `VerifiedContext` into the hub-key cache upsert + `fsckMirror`. `cmd/notecheck`
  is the fully-independent signature-parity oracle, shelled out in CI. `store/*.go` uses
  `modernc.org/sqlite` with ADR-0005/0007 single-writer discipline.
- **Missing (M1 connective tissue, off the Verify bar):** real alert transport (`alertFunc` is a WARN
  `slog` emit); warm-path second `did.json` resolve.
- **Fixtures**: `testdata/live/` (repo root) still holds only the two checkpoints — no tiles, entry
  bundles, or did.json. All proof/dashboard/browser/badge/web/dossier/records/registry tests run against
  in-process fixtures. Stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests; not
  refreshed.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck`, `client`),
  `transparency-dev/formats` (`cmd/notecheck`), `gopkg.in/yaml.v3` (direct, for the Hub-List parser).
  **Not wired:** `nbd-wtf/opentimestamps`, `github.com/iscc/iscc-lib/packages/go` (ADR-0011, not yet
  adopted — see Quality gates).

## M2 — Aggregator
**Status**: **met** — carried forward; no M2 source touched in the `4a521ca..HEAD` diff. Both Verify
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
**Status**: **in progress — Hub-List resolver built and hardened (PASS_WITH_NOTES), certificate page
still open.** Badge, DS shell, `/` index, `/<domain>/log/` browser, hub dossier, frozen Exhibit, record
list, single-record page, the ISCC-IDv1 decoder, and the `(realm, hub_id) → domain` resolver are all
built and verified. The certificate HTML page + proof-bundle assembler and the anchor panels are not
started.
- **Met (carried forward):** `internal/badge` five-status `Render`; `internal/web` `/_ds/` static-asset
  subtree; the `/` realm-index grid; the `/<domain>/log/` log-browser redress; `internal/dossier.Handler`
  (`GET /<domain>`); the frozen Exhibit; the paginated record list (`GET /<domain>/log/records`); the
  single-record page (`GET /record?index=<seq>`); the ISCC-IDv1 `internal/index.Decode` (fail-closed on
  all four header nibbles, `GOOS=js GOARCH=wasm`-buildable).
- **Hardened this iteration (review PASS_WITH_NOTES at f6bf809):** the `(realm, hub_id) → domain`
  Hub-List resolver in `internal/registry` is now **fail-closed** on the two original gaps —
  `hubDomain` rejects `u.Path`/`u.RawQuery`/`u.Fragment`-bearing urls (registry.go:188) and
  `Hub.HubID` is `*uint16` so an absent `hub_id` is rejected, not coerced to slot 0 (registry.go:129).
  Both reviewer-reproduced non-vacuous; golden fixture still resolves
  (`Resolve(0)=sb0.iscc.id`, `Resolve(1)=sb1.amlet.id`); resolver stays WASM-pure; `go.mod`/`go.sum`
  byte-unchanged. **Not yet wired** into config / dashboard / cert page (re-verified: no caller outside
  `internal/registry`).
- **Residual fail-open (filed `normal`, NOT yet fixed):** `hubDomain` (registry.go:188) does **not**
  check `u.ForceQuery`, so a bare trailing `?` (`https://sb0.iscc.id?`) slips the guard and round-trips
  the delimiter. Same class as the two just-closed; not exploitable yet (wiring deferred, fixture uses
  clean urls). The guard line still reads `if u.Path != "" || u.RawQuery != "" || u.Fragment != ""` —
  no `|| u.ForceQuery` — confirming the issue is open.
- **Residual (off the Verify bar, filed `low`):** the vacuous single-record label test
  (`record_test.go::schemaForSeq` returns the constants under test rather than HARDCODED literals).
- **Still open on the M-UI Verify bar:** the **certificate of inclusion** HTML page at
  `/inclusion/{iscc_id}` + the **downloadable proof-bundle assembler** (re-engages the oracle/conformance
  gate — `serveVerify` discards the raw checkpoint bytes + resolved hub key the bundle needs); separate
  **Bitcoin-anchor vs comparison-anchor** panels. Confirmed absent: no `/inclusion/{iscc_id}` HTML route
  in `cmd/iscc-monitor/main.go` (only the per-hub JSON `/<domain>/log/inclusion` proof route, and a
  comment in `proofserve/handler.go`); no proof-bundle assembler in source.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; never re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not in `go.mod`/`go.sum` or source;
no `internal/proof` package; no WASM build target (`syscall/js` not in source). The `internal/badge`,
`internal/web`, `internal/metrics`, `internal/index`, and `internal/registry` leaves are WASM-shareable
primitives the verifier app will reuse, but the verifier itself does not exist.

## Quality gates
**Status**: **green — CI passing at HEAD, working tree clean, nothing unpushed.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, **`go 1.24.0`**, no `toolchain` line);
  `mise run check` runnable. The latest `review` verdict (f6bf809) is **PASS_WITH_NOTES / loop
  CONTINUE** and records `mise run check` green (build + vet + all 20 packages incl. uncached
  `internal/registry`, `gofmt -l .` empty, `go.mod`/`go.sum` byte-unchanged). The one new issue is a
  fail-open edge in the unwired resolver, filed `normal` — it does not red the gate.
- **CI**: `.github/workflows/ci.yml` runs the inlined `mise run check` gate + the `cmd/notecheck` oracle
  shell-out on push/PR to `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`.
  **Latest concluded run on `develop`: `conclusion: success` at headSha `f6bf809` (that IS HEAD).**
  `develop` is in sync with `origin/develop` (no unpushed commits).
- **TARGET/CODE GAP (ADR-0011, filed `normal`):** `target.md` "Stack (locked)" now mandates **Go 1.26**
  + the **`github.com/iscc/iscc-lib/packages/go` v0.5.0** codec (ADR-0011 committed at 19dfc90 in this
  diff range), but `go.mod` is still `go 1.24.0` with no iscc-lib require (local toolchain is go1.24.x).
  This is a deliberate, sequenced, not-yet-started increment (the human issue says "sequence it
  **before** more M-UI feature work"), not a regression — but target and code disagree on the stack
  until it lands.
- **Open issues: 0 `critical`, 2 `normal`, 6 `low`** (the "critical" line in the `issues.md` format
  legend is the template placeholder, not a real issue). The 2 `normal`: (1) [human] adopt iscc-lib
  codec + bump to Go 1.26 (ADR-0011); (2) [review/Codex] Hub-List `hubDomain` accepts a trailing `?`
  (`ForceQuery` fail-open). The two earlier resolver fail-opens (path-url, missing-`hub_id`) are
  verified-fixed and deleted from `issues.md`. The 6 `low` are loop-skipped (vacuous single-record label
  test; notecheck `out` param; overlay precedence duplicated 3x; mirror write-path tile-coord leak;
  proofserve `os.ErrNotExist`→404 duplication; `[human]` scaling trip-wire metrics).

## Next Milestone
**M1/M2/M3 met; M-UI is the active milestone. Latest review is PASS_WITH_NOTES (loop CONTINUE), CI green
at HEAD, no `critical` issue — clear to proceed.** The next `define-next` should weigh, in roughly this
order:

1. **Knock out the one residual `ForceQuery` fail-open** (registry.go:188) — the cheapest, most
   self-contained slice (one `|| u.ForceQuery` clause + one `https://host?` reject test). Fully closes
   the bare-host-base-url contract before the certificate page consumes `Resolve`. Could be folded into
   the certificate-page slice's prelude rather than spent as a standalone iteration, to avoid a third
   consecutive resolver-polish iteration without an M-UI Verify advance (see Convergence note).
2. **The human-filed ADR-0011 stack bump** (Go 1.24 → 1.26 + adopt `iscc-lib` v0.5.0 + the
   `internal/index` tripwire/parity test) — flagged foundational and "before more M-UI feature work."
   **Gating caveat:** local toolchain is go1.24.x; this increment must run where mise can provision Go
   1.26, and `go.mod`'s `go` directive must not flip without the 1.26 toolchain present (it reds the
   whole-module gate). Confirm the toolchain first.
3. **Then the certificate path proper:** the **certificate of inclusion** HTML page at
   `/inclusion/{iscc_id}` (numbered evidence clauses §1 Subject · §2 Checkpoint (size, root) · §3
   Inclusion proof · §4 Signing key (did:web) · §5 Bitcoin anchor · §6 Record history, plus the
   tier-1/tier-2 honesty panel) + the **downloadable proof-bundle assembler** `{checkpoint,
   inclusion/consistency proof, record bytes, hub key, ots?}` — wiring the now-built resolver + decoder.
   This slice **re-engages the oracle/conformance gate** (`serveVerify` currently discards the raw
   checkpoint bytes + resolved hub key the bundle needs), so the reviewer must mutation-prove the served
   bundle's inclusion proof non-vacuous and confirm `notecheck`/golden-vector parity. Plus the separate
   Bitcoin-anchor vs comparison-anchor panels.
4. **WASM verifier → OTS anchoring** remain the last two v1 milestones (each 1/1 Verify open).
5. **Off the Verify bar:** harden the vacuous single-record label test (`low`); sb1 fixture refresh
   (stale did.json key); real alert transport; an end-to-end registry-deactivation `inactive` path once
   a public `SetActive` lands; consolidating the now-3x overlay precedence into `internal/badge`.
