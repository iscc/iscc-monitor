<!-- assessed-at: 56ebeadb00e2103e1eb2283d14fca33dd7f7e584 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress — the shared DS v2 shell (token CSS + self-hosted webfonts under `/_ds/`) is complete, and now the **first screen** is dressed: `GET /` is redressed from a `<table>` into the Evidence-Ledger CSS-grid realm index.
This iteration replaced the bare dashboard table with the Evidence-Ledger grid screen (document-chrome
masthead, bordered/shadowed ledger card, mono uppercase header row, per-hub grid rows with the
two-line Domain/Origin stack, coverage-since cell, observed-size cell, five-status badge, frozen-row
tint, coverage-honesty footnote), all styled through a page-scoped `<style>` over the embedded DS
`var(--*)` tokens — CDN-free, no-JS, with the Codex `min-width:0` ellipsis fix applied. M1/M2/M3
remain fully met. Remaining v1 work: the rest of M-UI (dossier, record list, certificate +
proof-bundle, anchor panels, log-browser redress), then the WASM verifier and OTS anchoring.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~6 still open** (in progress). **Landed:** (a) the
    `HubStatusBadge` leaf (`internal/badge`, all five statuses, distinct label + inline-SVG
    silhouette, golden + mutation + fail-closed) wired with five-status overlay on BOTH `/` and
    `/<domain>/log/`; (b) the DS v2 shared shell — token CSS (`/_ds/tokens.css`) + self-hosted Readex
    Pro / JetBrains Mono webfonts (`/_ds/fonts.css` + 8 woff2 subsets), CDN-free, `no-cache`+ETag+304;
    (c) **this iteration** — the `/` realm index redressed into the Evidence-Ledger CSS-grid screen
    (no `<table>`, `display: grid`, DS token classes, CDN-free, no-JS, coverage-honesty footnote).
    **Still open:** thread the same DS-token/font shell + ledger grid into the **log browser**
    (`/<domain>/log/` still links no token/font CSS); **hub dossier** (`/<domain>`, + categorically-
    distinct frozen **Exhibit**); **paginated record list** (`?from=…[&n=…]`, no-JS, newest-first) +
    **single-record page** (declaration / deletion / unknown schema); **certificate of inclusion** at
    `/inclusion/{iscc_id}` (numbered evidence clauses) + **downloadable proof-bundle assembler**
    (`{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` — re-engages the
    oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes + resolved hub key the
    bundle needs); **separate Bitcoin-anchor vs comparison-anchor panels** + tier-1/tier-2 affordance.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported, re-verified).
- **Last ~10 iterations: ~6 milestone-Verify / ~4 refactor·polish·shell.** Healthy and on the Verify
  bar. The recent arc closed all four M3 criteria (verify-for-me → `/` dashboard → log browser), then
  opened M-UI leaf-first: `HubStatusBadge` → wired into `/` → five-status overlay on `/` → same
  overlay into the log browser → DS token CSS → self-hosted webfonts → and this iteration the first
  *screen* (the `/` Evidence-Ledger grid redress). **No polish-streak drift** — each leaf/screen is
  wired into an observable HTTP-seam assertion and linked from `/` in the same iteration.
  Watch-item: M-UI is the largest remaining slice. The shared shell (tokens + fonts) is complete and
  the index screen is now dressed; the next iterations must convert the REMAINING *screens* into wired
  no-JS HTTP-seam Verify criteria (log-browser redress, dossier, record list, certificate) — the
  proof-bundle assembler being the one that re-engages the crypto/oracle gate.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change since the last assessment. The
`e0d8e71..HEAD` diff touched only `internal/dashboard/dashboard.html` + `handler_test.go` (the `/`
redress) and context/loop docs — zero Go source files, no new package. All M1 Verify criteria remain
satisfied: `origin`/`vkey` golden, all three triggers (fork/shrink/equivocation) golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **242 `func Test`** across `cmd/` + `internal/`, **53** `_test.go` files
  (unchanged — this iteration adjusted assertions within the existing `internal/dashboard/handler_test.go`
  rather than adding a package or test function). Package count unchanged at 16 internal + 2 cmd.
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **16 internal packages** —
  `badge, config, corsmw, dashboard, didweb, follower, healthz, logclient, metrics, metricshttp,
  proofserve, registry, store, tiles, tilesserve, web`. Module `github.com/iscc/iscc-monitor`,
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
  `store.ListHubs` is a pure read LEFT JOINing `hubs` with `follow_state`; store stays a leaf.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; `AlertFunc func(int64,string)` seam unchanged. The warm-path's second `did.json`
  resolve is a larger design change, not on the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**.
  All proof/dashboard/browser/badge/web tests run against in-process fixtures. Stale `sb1.amlet.id`
  did.json drift (pre-rotation key) captured in tests; not refreshed.
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
re-hitting the hub. **Nothing remains on the M2 Verify bar.**

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; the `/` criterion was re-verified this
iteration against the redressed grid screen.
- **CORS** — `Access-Control-Allow-Origin: *` on every public GET via the single `corsmw.Handler` wrap.
- **verify-for-me** — `GET /<domain>/log/verify?iscc_id=<id>` returns store-provable `hub_status`,
  accepted `(size, root)`, and a REAL RFC-6962 inclusion result recomputed from the mirror and
  Merkle-verified against the accepted root. Every id-shaped fault → 200 non-verified. Golden +
  mutation non-vacuous.
- **`GET /` dashboard** — `200 text/html` listing **every** realm hub with its status + ADR-0001
  coverage window, status cell rendering all five glossary statuses through `hubStatusBadge` via the
  `overlayStatus` overlay. **This iteration**: now an Evidence-Ledger CSS-grid (no `<table>`,
  `display: grid`, DS token classes), still links BOTH `/_ds/tokens.css` and `/_ds/fonts.css`, with
  **no `https://` in the body**. Golden + mutation-tested (`display:grid`→`block` and
  `no coverage yet`→`size 0` both proven to fail their assertions).
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing accepted checkpoint `(size, root)` +
  relative links into `entries`/`inclusion`/`consistency`/`verify`/`checkpoint`. Its status cell
  renders through `hubStatusBadge` overlaid with the in-memory verdict. `POST /` → 405; unpolled hub
  → 200. Golden + mutation + e2e-proven. (Not yet redressed into the ledger grid — see M-UI.)

**Known limitations (carried forward, off the M3 Verify bar — these become M-UI work):**
- The dossier / certificate / record surfaces don't exist yet (M-UI); the log browser is not yet
  redressed into the Evidence-Ledger grid (it links no token/font CSS — only `/` does).
- `inactive` is unreachable through the public store API (no `SetActive`/deactivation writer), so the
  `/` golden covers it via a fixture-deactivated hub at the store seam, but no registry-deactivation
  end-to-end path exists yet.
- No ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces or on `/`/`/verify` —
  not a Verify criterion. (The `/_ds/` static assets DO carry `no-cache` + strong ETag + 304.)

## M-UI — Evidence Ledger frontend
**Status**: **in progress — the five-status badge render is met on `/` and the log browser, the DS v2
shared shell (token CSS + self-hosted webfonts under `/_ds/`) is complete, and the first screen (`/`
realm index) is now dressed into the Evidence-Ledger grid. The remaining screens are open.**
- **Landed so far:**
  - `internal/badge` — a pure, stdlib-only, WASM-shareable `HubStatusBadge` partial (`Render`,
    `Label`, embedded `badge.html`, `PartialName = "hubStatusBadge"`, `Source`) rendering all five
    statuses each with a distinct text label + inline-SVG silhouette, failing closed on unknown/empty
    status. Wired into both `/` and `/<domain>/log/`, each with its own local `StatusSource` interface
    + `overlayStatus` precedence (store `inactive`/`frozen` win; the in-memory verdict
    `unresolvable`/`unverified` overlays a store-`verified` hub). Both surfaces golden-assert
    `data-status`/labels/per-status SVG markers at the HTTP seam, mutation-proven.
  - `internal/web` — a pure stdlib leaf (WASM-green) serving the whole `/_ds/` static-asset subtree
    via one `web.Handler` at `web.Prefix`: token CSS (`/_ds/tokens.css`), self-hosted webfonts
    (8 latin woff2 subsets + `@font-face` `/_ds/fonts.css`, traversal-guarded), all CDN-free with
    `Cache-Control: no-cache` + strong content-ETag + 304; non-GET → 405; unknown `/_ds/` path → 404.
  - **`/` realm index redress (this iteration, review `56ebead` PASS_WITH_NOTES)** — the dashboard
    `<table>` is replaced by the Evidence-Ledger CSS-grid screen (masthead, bordered/shadowed ledger
    card, mono uppercase column-header row, per-hub grid rows with two-line Domain/Origin, coverage-
    since + observed-size cells, five-status badge, frozen-row tint, coverage-honesty footnote),
    styled via a page-scoped `<style>` over the embedded DS `var(--*)` tokens. No `<table>`,
    `display: grid` present, `var(--font-sans)`/`--font-mono` resolve to the embedded fonts, body is
    CDN-free, `.hub-cell { min-width: 0 }` ellipsis fix applied (Codex P2). Two new HTTP-seam
    assertions, both mutation-confirmed non-vacuous.
- **Still open on the M-UI Verify bar:** thread the DS-token/font shell + ledger grid into the **log
  browser** (`proofserve.serveBrowser` still links no token/font CSS); **hub dossier** (`/<domain>`,
  + categorically-distinct frozen **Exhibit** — non-dismissable violation kind + detected-at);
  **paginated record list** (`?from=…[&n=…]`, no-JS, newest-first) + **single-record page**
  (declaration / deletion / unknown schema); **certificate of inclusion** at `/inclusion/{iscc_id}`
  (numbered evidence clauses) + **downloadable proof-bundle assembler** (re-engages the
  oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes + resolved hub key the
  bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels; tier-1/tier-2 affordance.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD). woff2 binaries are committed/build-pinned; served bytes are never
  re-fetched at runtime.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM build
target (`syscall/js` not in source — grep clean). The `internal/badge`, `internal/web`, and
`internal/metrics` leaves are WASM-shareable primitives the verifier app will reuse, but the verifier
itself does not exist.

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** (run 27911277271, headSha `56ebead` = HEAD).
- Latest `review` handoff (2026-06-21, **PASS_WITH_NOTES / CONTINUE**, for the `/` Evidence-Ledger
  redress) records `mise run check` green (all 18 packages `ok`, `go vet`/`gofmt -l .` clean), the
  scope held to the HTML asset + its golden test (zero Go source files changed), both new assertions
  mutation-confirmed non-vacuous, `GOOS=js GOARCH=wasm` build of the shared leaves green, token
  resolution checked, gate-integrity scan clean. Oracle gate correctly N/A (pure HTML rendering of
  persisted store rows — no crypto path touched). Codex second opinion: one P2 (grid ellipsis
  `min-width:0`) confirmed and fixed in-review.
- **No open `critical` or `normal` issue.** **4 open `low`** remain (all loop-skipped, none block
  DONE): `cmd/notecheck` vestigial `out io.Writer` param; hub-status overlay precedence duplicated
  across dashboard/proofserve; mirror write-path leaks tile coordinates into the follower; proofserve
  repeats the `os.ErrNotExist`→404 mapping `tilesserve` already centralised.

## Next Milestone
**M1/M2/M3 all met. The next v1 milestone is M-UI (Evidence Ledger frontend, ADR-0010), in progress.**
CI green, no `critical`/`normal` open, so feature work proceeds.

Convergence-driven order (the shared DS shell is complete and the `/` index is now dressed; the next
iterations must build the REMAINING screens, not more shared primitives):
1. **Log-browser redress** — thread the same DS-token/font shell + Evidence-Ledger card/grid pattern
   into `proofserve.serveBrowser` (`/<domain>/log/`), which already reuses the dashboard's
   `StatusSource`/`overlayStatus` shape and is the lowest-risk next screen; carry the CSS-Grid
   `min-width:0` ellipsis rule into any new truncating cell.
2. **Hub dossier** (`/<domain>`, frozen **Exhibit** — non-dismissable violation kind + detected-at),
   **paginated record list + single record** (declaration / deletion / unknown schema, no-JS), and
   the **certificate of inclusion** at `/inclusion/{iscc_id}` + **downloadable proof-bundle
   assembler** (the slice that re-engages the oracle/conformance gate — `serveVerify` currently
   discards the raw checkpoint bytes + resolved hub key the bundle needs). Add the separate
   Bitcoin-anchor vs comparison-anchor panels + tier-1/tier-2 affordance.
3. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands.
