<!-- assessed-at: 173f718e9d8429ab3d97cc5bf3bb8efb817aacb0 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress — the shared DS v2 shell (token CSS + self-hosted webfonts under `/_ds/`) is complete, and the first TWO screens are now dressed: `GET /` (realm index, CSS-grid) and `GET /<domain>/log/` (log browser, Evidence-Ledger card).

This iteration redressed the **log browser** (`GET /<domain>/log/`) from a bare `<table>` into the
Evidence-Ledger card screen — `/_ds/tokens.css` + `/_ds/fonts.css` `<link>`s, a page-scoped `<style>`
over the embedded DS `var(--*)` tokens, the masthead/ledger-card definition rows + proof-surface link
list — with a new `TestBrowserLinksTokensNoCDN` HTTP-seam assert. Zero Go source files changed. M1/M2/M3
remain fully met. Remaining v1 work: the rest of M-UI (hub dossier, record list, single record,
certificate + proof-bundle, anchor panels), then the WASM verifier and OTS anchoring.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~5 still open** (in progress). **Landed:** (a) the
    `HubStatusBadge` leaf (`internal/badge`, all five statuses, distinct label + inline-SVG
    silhouette, golden + mutation + fail-closed) wired with five-status overlay on BOTH `/` and
    `/<domain>/log/`; (b) the DS v2 shared shell — token CSS (`/_ds/tokens.css`) + self-hosted Readex
    Pro / JetBrains Mono webfonts (`/_ds/fonts.css` + 8 woff2 subsets), CDN-free, `no-cache`+ETag+304;
    (c) the `/` realm index redressed into the Evidence-Ledger CSS-grid screen (no `<table>`,
    `display: grid`, DS token classes, CDN-free, no-JS, coverage-honesty footnote); (d) **this
    iteration** — the **log browser** (`/<domain>/log/`) redressed into the Evidence-Ledger card
    screen (token/font CSS linked, no `<table>`, no CDN, no-JS, coverage-honesty no-checkpoint state).
    **Still open:** **hub dossier** (`/<domain>`, + categorically-distinct frozen **Exhibit**);
    **paginated record list** (`?from=…[&n=…]`, no-JS, newest-first) + **single-record page**
    (declaration / deletion / unknown schema); **certificate of inclusion** at `/inclusion/{iscc_id}`
    (numbered evidence clauses) + **downloadable proof-bundle assembler** (`{checkpoint,
    inclusion/consistency proof, record bytes, hub key, ots?}` — re-engages the oracle/conformance
    gate; `serveVerify` discards the raw checkpoint bytes + resolved hub key the bundle needs);
    **separate Bitcoin-anchor vs comparison-anchor panels** + tier-1/tier-2 affordance.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported, re-verified).
- **Last ~10 iterations: ~6 milestone-Verify / ~4 refactor·polish·shell.** Healthy and on the Verify
  bar. The recent arc closed all four M3 criteria (verify-for-me → `/` dashboard → log browser), then
  opened M-UI leaf-first: `HubStatusBadge` → wired into `/` → five-status overlay on `/` → same
  overlay into the log browser → DS token CSS → self-hosted webfonts → the `/` Evidence-Ledger grid
  redress → and this iteration the **log-browser** Evidence-Ledger redress (the second dressed screen).
  **No polish-streak drift** — each leaf/screen is wired into an observable HTTP-seam assertion.
  Watch-item: M-UI is the largest remaining slice. The shared shell (tokens + fonts) is complete and
  two screens (index + log browser) are now dressed; the next iterations must build the REMAINING
  *screens* as wired no-JS HTTP-seam Verify criteria (dossier, record list, single record,
  certificate) — the proof-bundle assembler being the one that re-engages the crypto/oracle gate.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change since the last assessment. The
`56ebead..HEAD` diff touched only `internal/proofserve/browser.html` + `browser_test.go` (the
log-browser redress) and context/loop docs — zero Go source files. All M1 Verify criteria remain
satisfied: `origin`/`vkey` golden, all three triggers (fork/shrink/equivocation) golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **243 `func Test`** across `cmd/` + `internal/`, **53** `_test.go` files
  (one new test function this iteration — `TestBrowserLinksTokensNoCDN` in
  `internal/proofserve/browser_test.go`; no new package, no new test file). Package count unchanged at
  16 internal + 2 cmd.
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
**Status**: **met (4/4 Verify criteria)** — carried forward; the `/<domain>/log/` log-browser
criterion was re-verified this iteration against the redressed Evidence-Ledger card (functional
content meaning-equivalent — accepted `(size, root)`, badge, all five relative proof links, no-checkpoint
state).
- **CORS** — `Access-Control-Allow-Origin: *` on every public GET via the single `corsmw.Handler` wrap.
- **verify-for-me** — `GET /<domain>/log/verify?iscc_id=<id>` returns store-provable `hub_status`,
  accepted `(size, root)`, and a REAL RFC-6962 inclusion result recomputed from the mirror and
  Merkle-verified against the accepted root. Every id-shaped fault → 200 non-verified. Golden +
  mutation non-vacuous.
- **`GET /` dashboard** — `200 text/html` listing **every** realm hub with its status + ADR-0001
  coverage window, status cell rendering all five glossary statuses through `hubStatusBadge` via the
  `overlayStatus` overlay. Now an Evidence-Ledger CSS-grid (no `<table>`, `display: grid`, DS token
  classes), links BOTH `/_ds/tokens.css` and `/_ds/fonts.css`, no `https://` in the body. Golden +
  mutation-tested.
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing accepted checkpoint `(size, root)` +
  relative links into `entries`/`inclusion`/`consistency`/`verify`/`checkpoint`. Its status cell
  renders through `hubStatusBadge` overlaid with the in-memory verdict. `POST /` → 405; unpolled hub
  → 200. **This iteration**: redressed into the Evidence-Ledger card (links both token/font CSS, no
  `<table>`, no CDN URL, page-scoped `<style>` over DS tokens). Golden + mutation + e2e-proven;
  `TestBrowserLinksTokensNoCDN` added, mutation-confirmed non-vacuous.

**Known limitations (carried forward, off the M3 Verify bar — these become M-UI work):**
- The dossier / certificate / record surfaces don't exist yet (M-UI).
- `inactive` is unreachable through the public store API (no `SetActive`/deactivation writer), so the
  `/` golden covers it via a fixture-deactivated hub at the store seam, but no registry-deactivation
  end-to-end path exists yet.
- No ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces or on `/`/`/verify` —
  not a Verify criterion. (The `/_ds/` static assets DO carry `no-cache` + strong ETag + 304.)

## M-UI — Evidence Ledger frontend
**Status**: **in progress — the five-status badge render is met on `/` and the log browser, the DS v2
shared shell (token CSS + self-hosted webfonts under `/_ds/`) is complete, and the first TWO screens
(`/` realm index + `/<domain>/log/` log browser) are now dressed into the Evidence-Ledger design. The
remaining screens are open.**
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
  - **`/` realm index redress** — the dashboard `<table>` replaced by the Evidence-Ledger CSS-grid
    screen (masthead, bordered/shadowed ledger card, mono uppercase column-header row, per-hub grid
    rows with two-line Domain/Origin, coverage-since + observed-size cells, five-status badge,
    frozen-row tint, coverage-honesty footnote), styled via a page-scoped `<style>` over embedded DS
    `var(--*)` tokens. No `<table>`, `display: grid`, CDN-free, `.hub-cell { min-width: 0 }` ellipsis
    fix applied (Codex P2). Mutation-confirmed non-vacuous.
  - **`/<domain>/log/` log-browser redress (this iteration, review `173f718` PASS / CONTINUE)** — the
    `<table>` in `internal/proofserve/browser.html` replaced by the Evidence-Ledger card screen: the
    two `/_ds/tokens.css` + `/_ds/fonts.css` `<link>`s, a page-scoped `<style>` over embedded DS
    `var(--*)` tokens, the `.chrome` masthead, a bordered/shadowed `.ledger` card with definition rows
    (Status / Accepted size / Accepted root) + the proof-surface link list. No `<table>`, no
    `http://`/`https://`/CDN in the body, no-JS. New `TestBrowserLinksTokensNoCDN` HTTP-seam assert,
    mutation-confirmed non-vacuous. Functional content meaning-equivalent (accepted `(size, root)`,
    five-status badge, all five relative proof links, coverage-honesty no-checkpoint state).
- **Still open on the M-UI Verify bar:** **hub dossier** (`/<domain>`, + categorically-distinct frozen
  **Exhibit** — non-dismissable violation kind + detected-at); **paginated record list** (`?from=…[&n=…]`,
  no-JS, newest-first) + **single-record page** (declaration / deletion / unknown schema); **certificate
  of inclusion** at `/inclusion/{iscc_id}` (numbered evidence clauses) + **downloadable proof-bundle
  assembler** (re-engages the oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes +
  resolved hub key the bundle needs); separate **Bitcoin-anchor vs comparison-anchor** panels;
  tier-1/tier-2 affordance.
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
  `conclusion: success`** (run 27911659777, headSha `173f718` = HEAD).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**, for the log-browser Evidence-Ledger
  redress) records `mise run check` green (all 18 packages `ok`, `go vet`/`gofmt -l .` clean), the
  scope held to the HTML asset + its test (zero Go source files changed; go.mod/go.sum untouched), the
  new assertion mutation-confirmed non-vacuous, `GOOS=js GOARCH=wasm` build of the shared leaves green,
  token resolution checked, gate-integrity scan clean. Oracle gate correctly N/A (pure HTML rendering
  of persisted store rows — no crypto path touched). Codex second opinion: clean, no findings.
- **No open `critical` or `normal` issue.** **4 open `low`** remain (all loop-skipped, none block
  DONE): `cmd/notecheck` vestigial `out io.Writer` param; hub-status overlay precedence duplicated
  across dashboard/proofserve; mirror write-path leaks tile coordinates into the follower; proofserve
  repeats the `os.ErrNotExist`→404 mapping `tilesserve` already centralised.

## Next Milestone
**M1/M2/M3 all met. The next v1 milestone is M-UI (Evidence Ledger frontend, ADR-0010), in progress.**
CI green, no `critical`/`normal` open, so feature work proceeds.

Convergence-driven order (the shared DS shell is complete and the index + log-browser screens are now
dressed; the next iterations must build the REMAINING screens, which need NEW handlers + store reads):
1. **Hub dossier** (`/<domain>`) — the next lowest-risk new screen: per-hub page with the coverage
   window (ADR-0001 honesty), status badge, and the categorically-distinct frozen **Exhibit**
   (non-dismissable violation kind + detected-at), dressed in the same Evidence-Ledger card pattern.
   Carry the scoped-`<style>`-over-shared-tokens approach + the CSS-Grid `min-width:0` ellipsis rule.
2. **Paginated record list + single record** (`?from=…[&n=…]`, no-JS, newest-first over `iscc_index`;
   declaration / deletion / unknown schema) and the **certificate of inclusion** at
   `/inclusion/{iscc_id}` + **downloadable proof-bundle assembler** (the slice that re-engages the
   oracle/conformance gate — `serveVerify` currently discards the raw checkpoint bytes + resolved hub
   key the bundle needs). Add the separate Bitcoin-anchor vs comparison-anchor panels + tier-1/tier-2
   affordance.
3. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end registry-deactivation `inactive` path once a public `SetActive` lands.
