<!-- assessed-at: 7b0d037e7c7319c8adb97b741fc002ca9c974ccf -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 complete (Trust API + functional SSR) — all four M3 Verify criteria met (CORS, verify-for-me
verdict, `GET /` hub-list dashboard, `GET /<domain>/log/` HTML log browser). The next arc is **M-UI**
(Evidence Ledger frontend, ADR-0010), then the WASM verifier and OTS anchoring.

The monitor follows + mirrors + verifies hubs (M1 met), serves all three computed proofs per hub from
the local mirror plus fsck root-rebuild (M2 met), and the full M3 Trust-API surface is functional:
CORS on every GET, the verify-for-me JSON verdict, the server-rendered `/` dashboard, and the per-hub
HTML log browser. What remains for v1: dressing/extending those functional SSR surfaces into the
Evidence Ledger design (M-UI), the WASM verifier, and OTS/Bitcoin anchoring.

## Convergence
- **Remaining Verify criteria:**
  - **M3: 0 open (met).** Closed in the assessed range (`c940248..7b0d037`): the `GET /<domain>/log/`
    HTML log browser (`200 text/html`, exposes accepted checkpoint `(size, root)` + relative links into
    `entries`/proof routes), golden + mutation + e2e-proven at the HTTP seam (`internal/proofserve`,
    review PASS at `7b0d037`). The other three M3 criteria (CORS on every GET; verify-for-me verdict;
    `/` hub-list dashboard) were already met.
  - **M-UI (Evidence Ledger frontend): ~9/9 open** (not started). The functional SSR surfaces exist
    (`/` realm index via `internal/dashboard`, per-hub log browser via `internal/proofserve`), but none
    of the M-UI Verify bar is met: no ISCC Design System v2 tokens / self-hosted fonts embedded, no
    five-status `HubStatusBadge` partial (icon+label+silhouette), no hub dossier / frozen Exhibit, no
    paginated record list or single-record page, no certificate-of-inclusion + downloadable proof
    bundle, and the full five-status taxonomy is not yet store-provable (`proofserve.hubStatus` /
    `dashboard` resolve only `frozen`/`verified`/`inactive`).
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported).
  - M1: 0 open (met). M2: 0 open (met).
- **Last ~10 iterations: ~4 milestone-Verify / ~6 refactor·polish·test·infra.** Convergence is healthy
  and on the Verify bar: the assessed range (`7cc8cbf..7b0d037`) closed M3's fourth and final criterion
  — `ab50cfe` (define-next: log browser) → `b46f25f` (advance) → `7b0d037` (review PASS). Combined with
  the dashboard (`9638abf`/`c940248`) and verify-for-me (`562a4d8`/`8f0aa9d`) slices, the loop has
  attacked four distinct M3 criteria back-to-back with no polish-streak drift. **One note:** `target.md`
  now carries a new **M-UI** milestone (Evidence Ledger, ADR-0010) that lands between M3 and the WASM
  verifier — a substantial scope addition since the prior assessment, expanding the remaining v1 work.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change in the assessed range
(`c940248..7b0d037` touched only `internal/proofserve` (browser), `cmd/iscc-monitor/main.go`
(hubHandler rewiring), `CLAUDE.md`, and loop context). All Verify criteria remain satisfied:
`origin`/`vkey` golden, all three triggers (fork/shrink/equivocation) golden-tested end-to-end with
freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over
HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **222 `func Test`** across `cmd/` + `internal/`, **51** `_test.go` files
  (up from 219/50 — the +3 tests / +1 file are `internal/proofserve/browser_test.go`).
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **14 internal packages** —
  `config, corsmw, dashboard, didweb, follower, healthz, logclient, metrics, metricshttp,
  proofserve, registry, store, tiles, tilesserve`. Module `github.com/iscc/iscc-monitor`,
  `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation, evaluated inside
  `logclient.CheckConsistency`; growing-pair equivocation builds the RFC-6962 consistency proof from
  the LOCAL mirror → freeze on `true` (ADR-0006). Frozen hubs are evidence-only on clean re-polls.
- **`AcceptCheckpoint` 4-way verdict** (`logclient/accept.go`) returns `(Status, CheckpointInfo,
  VerifiedContext{VKey,Key}, error)`, context populated only on `StatusVerified`; `PollHub` threads
  it into the hub-key cache upsert + `fsckMirror`. Reuse is per-poll only (ADR-0009).
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real
  sb0 checkpoint.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL,
  `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`.
  `store.ListHubs` (`internal/store/hubs.go`) is a pure read LEFT JOINing `hubs` with `follow_state`,
  returning plain Go types so store stays a leaf (no `net/http`/`logclient`).
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; `AlertFunc func(int64,string)` seam unchanged. The warm-path's second `did.json`
  resolve is a larger design change, not on the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**.
  Tile/fsck/inclusion/consistency/entries/verify/dashboard/browser tests run against in-process
  fixtures. Stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests; not refreshed.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck` via
  `fsckMirror`, `client` re-export), `transparency-dev/formats` (`cmd/notecheck`). **Not wired:**
  `nbd-wtf/opentimestamps` (re-verified: no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — carried forward; no production change. Both Verify criteria are exercised (fsck
root-rebuild WIRED on every verified non-frozen poll via `fsckMirror` → `logclient.RunFsck` over the
read-only `store.SQLiteFetcher`; inclusion cross-check conformance-tested over the real verified
mirror in `internal/follower/inclusion_test.go`). The served proof surface is complete: all three
computed proofs — `inclusion`, `consistency`, `entries` — served from the local mirror, never
re-hitting the hub. The mirror write API is single-source on `p` — `RecordTile(…, p uint8, …)` /
`RecordEntryBundle(…, p uint8, …)` with the store the only `p→width` authority (`widthForP` at
`internal/store/fetcher.go`). **Nothing remains on the M2 Verify bar.**

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria).** All four HTTP-seam criteria are satisfied:
- **CORS** — `Access-Control-Allow-Origin: *` on every public GET via the single `corsmw.Handler` wrap
  (`cmd/iscc-monitor/main.go:160`).
- **verify-for-me** — `GET /<domain>/log/verify?iscc_id=<id>` (`proofserve.serveVerify` +
  `VerifyVerdict`) returns store-provable `hub_status`, accepted `(size, root)`, and a REAL RFC-6962
  inclusion result recomputed from the mirror and Merkle-verified against the accepted root. Every
  id-shaped fault → 200 non-verified; non-200 reserved for genuine infra faults. Golden + mutation
  non-vacuous.
- **`GET /` dashboard** — `200 text/html` listing **every** realm hub with its store-provable status +
  ADR-0001 coverage window (`internal/dashboard` + `store.ListHubs`). Golden + mutation-tested.
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing the accepted checkpoint `(size,
  root)` (read verbatim from `FollowState` + `CheckpointAt`, no crypto) + relative links into the
  `entries`/`inclusion`/`consistency`/`verify`/`checkpoint` routes
  (`proofserve.serveBrowser` + `browser.html`, mounted via the `hubHandler` `/`-dispatch func at
  `cmd/iscc-monitor/main.go:214`). `POST /` → 405; unpolled hub → 200 "no accepted checkpoint yet".
  Golden + mutation + full-`buildMux` e2e-proven at the HTTP seam (review PASS at `7b0d037`).

**Known limitations (carried forward, off the M3 Verify bar):**
- `proofserve.hubStatus` / `dashboard` resolve only the store-provable status subset
  (`frozen`/`verified`, plus `inactive` in dashboard); the richer in-memory statuses
  (`unverified`/`unresolvable`/`rotated`) are not threaded through. Making the full five-status
  taxonomy store-provable is M-UI work.
- `inactive` is currently unreachable through the public store API (no `SetActive`/deactivation
  writer), so it is covered by a white-box `hubStatus` table test, not the HTTP-seam golden.
- No ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces or on `/`/`/verify` —
  not a Verify criterion.

## M-UI — Evidence Ledger frontend
**Status**: **not started** (new milestone in `target.md`, ADR-0010, between M3 and WASM). The
functional SSR surfaces M3 produced (`internal/dashboard` `/` realm index; `internal/proofserve`
per-hub log browser) are the foundation, but none of the M-UI Verify bar is met:
- No ISCC Design System v2 tokens or self-hosted Readex Pro / JetBrains Mono fonts embedded
  (`go:embed`); current templates are bare HTML with no CSS/JS (grep clean — no `_ds/` bundle, no
  embedded fonts).
- No five-status `HubStatusBadge` template partial (icon + label + silhouette, grayscale-safe); the
  full taxonomy is not yet store-provable.
- Missing screens: hub dossier (incl. frozen Exhibit), paginated log-browser record list (over
  `iscc_index`), single-record page, certificate-of-inclusion + downloadable proof bundle.
- The proof-bundle assembler (`{checkpoint, inclusion/consistency proof, record bytes, hub key,
  ots?}`) — the authoritative client-verifies artifact — does not exist; `serveVerify` already
  composes most pieces but discards the raw checkpoint bytes and resolved hub key the bundle needs.
  This is the slice that re-engages the oracle/conformance gate (it re-asserts hub signature + proofs).
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD).

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits
in `cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM
build target (`syscall/js` not in source — grep clean). The WASM-shared verifier seam continues to
ride on `internal/didweb`; `serveVerify` already composes most of the proof-bundle pieces the
in-browser verifier will share.

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** at HEAD `7b0d037` (run 27908498428) — the current HEAD, log browser included.
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**, for the `GET /<domain>/log/` log browser)
  records `mise run check` green (16 packages `ok`, `go vet`/`gofmt -l .` clean), store kept a leaf
  (`go list -deps ./internal/store` has no `net/http`/`internal/proofserve`), HTTP-seam golden + 2
  reverted mutations + full-`buildMux` e2e dispatch, oracle gate correctly N/A (pure render of
  persisted rows), gate-integrity scan clean. Codex second opinion clean, no findings.
- **No open `critical` or `normal` issue.** **Open `low` issue (loop-skipped, not a DONE blocker):** 1
  in `issues.md` — `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2/M3 all met (M3 now 4/4 Verify). The next v1 milestone is M-UI (Evidence Ledger frontend,
ADR-0010).** CI green at HEAD, no `critical`/`normal` open, so feature work proceeds.

Convergence-driven order:
1. **M-UI — Evidence Ledger frontend.** Largest remaining slice. Natural decomposition, leaf-first:
   (a) make the full five-status taxonomy (`verified`/`unresolvable`/`unverified`/`frozen`/`inactive`)
   store-provable so the badge can render it honestly; (b) embed the DS v2 tokens + self-hosted fonts
   and the `HubStatusBadge` partial (icon+label+silhouette) into the existing `/` index + log browser;
   (c) add the hub dossier (incl. frozen Exhibit), paginated record list + single-record page; (d) the
   certificate-of-inclusion + **downloadable proof-bundle assembler** — the authoritative
   client-verifies path that re-engages the oracle/conformance gate (`serveVerify` already composes
   most pieces; it must additionally carry the raw checkpoint bytes + resolved hub key).
2. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
3. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end `inactive`-render assertion once a registry-deactivation writer lands.
