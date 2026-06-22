<!-- assessed-at: 39f04036d6c9545a06e27b6f3541f1b32eb0d6ab -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete; gate green on every host. The loop is executing the order-independent **M-Deploy** packaging milestone (ADR-0013). First M-Deploy Verify item just closed (SIGTERM trap); the remaining blockers are the Dockerfile/GHCR image, version stamp, canonical realm doc + operability doc, and the root README.

The feature surface is built; the loop is on the M-Deploy runway. This window's only production change
was the SIGTERM trap in `cmd/iscc-monitor/main.go` — the first M-Deploy Verify criterion to close
(graceful shutdown under `docker stop`). M-Deploy is now partially met (1 of ~6 Verify items), but no
Dockerfile, no GHCR workflow, no version stamp, no `deploy/` realm doc, no operability doc, and no root
`README.md` exist, so DONE stays far out of reach.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `a9f088a7..HEAD` diff touched no trust-path source (only `cmd/iscc-monitor` process-lifecycle wiring).
  - **M-UI: behavioral + named-region Verify met; the mandatory M-UI exit visual-pass + human sign-off
    (ADR-0012) still pending.** Instance identity is config-driven on THREE of six SSR mastheads
    (`/`, dossier, certificate); the proofserve trio (`browser.html`, `records.html`, `record.html`)
    still renders the static placeholder — the follow-on arc.
  - **WASM verifier: published half CLOSED** (Pages live; `monitor.iscc.codes/_ds/verify.wasm` → 200
    `application/wasm`, byte-pinned). **Still open on the milestone Verify**: the cross-origin
    **signature half** (the WASM core verifies inclusion math + id-binding only — no
    checkpoint-signature / did:web resolution) — design-first remainder, `normal`. No dossier tier-2
    WASM caller.
  - **OTS anchoring: 1/1 open (carried).** All observable HTTP halves + both calendar-transport guards
    in place; only a root actually transiting to **Bitcoin-confirmed** remains (offline-unprovable).
  - **M-Deploy: 1 of ~6 Verify items met (SIGTERM trap), milestone PARTIALLY MET.** **CLOSED this
    window:** SIGTERM cancels the run context (verified — `notifyShutdown` at `main.go:113-114` registers
    `os.Interrupt, syscall.SIGTERM`, mutation-proven test in `shutdown_test.go`). **Still open:** no root
    `Dockerfile` + CI-runs-container `/healthz` job; no GHCR publish workflow; no `-ldflags` version stamp
    / `/version` field; no canonical `deploy/` realm doc; no operability/deployment doc.
  - **Root README: missing (blocks "Done When").** Repo root has no `README.md`.
- **Last ~10 iterations: ~3 milestone-Verify-or-gate-advancing (Pages publish unblock; cert UTC
  gate-greening; this window's SIGTERM trap) / ~7 chrome·plumbing·config-leaf.** **No drift:** the loop
  is on the M-Deploy runway — code-closable work that depends on no feature milestone, and it has begun
  closing M-Deploy Verify criteria (SIGTERM done). The next slices are the Dockerfile/GHCR image + version
  stamp.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `a9f088a7..HEAD` diff touched only `cmd/iscc-monitor/main.go`
+ `shutdown_test.go` (signal-trap wiring); no signature / RFC-6962 / Merkle / `proof` / `didweb` /
`logclient` / `follower` / store source. All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (unchanged, 27 source pkgs): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`;
  internal — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1` language directive;
  toolchain `mise.toml` `go = "1.26.4"`.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/{merkle,tessera,formats}`, `gopkg.in/yaml.v3`,
  `github.com/iscc/iscc-lib/packages/go`, `github.com/nbd-wtf/opentimestamps`.

## M2 — Aggregator
**Status**: **met** — carried forward. No fsck / fetcher / mirror BLOB / follower-ingest / store path
touched. fsck root-rebuild on every verified non-frozen poll; inclusion cross-check conformance-tested
over the real verified mirror; `inclusion`/`consistency`/`entries` served from the local mirror.

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward; no M3 trust-path source touched. CORS on
every public GET; verify-for-me at `GET /<domain>/log/verify?iscc_id=<id>`; `GET /` realm-index
dashboard; `GET /<domain>/log/` log browser. All golden + mutation.

**Known limitations (off the M3 Verify bar):** `inactive` unreachable through the public store API; no
ETag/Cache-Control on size-dependent proof surfaces (the `/_ds/` static assets DO carry strong ETag +
`no-cache` + 304).

## M-UI — Evidence Ledger frontend
**Status**: **behaviorally + named-region complete; the M-UI exit visual-pass + human sign-off is
pending.** No template-render change this window (the only prod edit was `cmd/iscc-monitor` signal
wiring). All six certificate clauses + both anchor panels + badge + DS shell + `/` index + log browser +
hub dossier + frozen Exhibit + single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle
endpoint render and pass the behavioral HTTP-seam Verify.
- **Still open (NOT critical, carried):** instance identity is config-driven on THREE of six SSR
  mastheads; the proofserve trio (`browser.html`, `records.html`, `record.html`) still renders the
  static `monitor instance` placeholder — the follow-on arc. The `/` "recent declarers checked" hero
  footer is omitted (needs store lookup history; `normal`). The mandatory **M-UI exit visual-pass +
  human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes:** the realm-index Anchor cell is a per-HUB latest-stamped-root indicator
  not tied to the displayed Checkpoint (design-honesty question for the M-UI exit; spec-faithful,
  non-blocking).
- **`low` carried (const-dup):** the masthead fallback consts + `resolveIdentity` are duplicated across
  dashboard + dossier + certificate (3x, byte-identical); the proofserve slice would be the 4th copy,
  the natural trigger to consolidate into one shared exported `Resolve`. The stale `.chrome-identity` CSS
  comment in `dashboard.html:75-76` is still inaccurate.

## WASM verifier · OTS anchoring
**Status**: **WASM — published half CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** the id-binding half is closed in source + artifact, the artifact is reproducible from
  `mise run build:wasm` (`TestWasmVerifyHashPinned` green), and it is **publicly published** —
  `monitor.iscc.codes` serves the byte-pinned `/_ds/verify.wasm`. **Still open:** the cross-origin
  **SIGNATURE-half gap** (the verifier core does NO checkpoint-signature / did:web check — the success
  copy overstates a key check that never runs) — design-first remainder / STOP-candidate, `normal`. No
  tier-2 WASM caller in the hub dossier. **Carried `low`:** `cmd/verifier-site` writes non-atomically;
  `pages.yml` actions target deprecated Node 20.
- **OTS:** the `.ots` serve route, the §5 anchor clause (digest-bound via `ots.ConfirmedFor`), the store
  layer, the off-path stamp/upgrade loop (`OTSTick`), the offline classifier, and the calendar transport
  (both `safeUpgrade` + `safeStamp` guards) are wired. The Verify-closer not yet built: a root reaching
  Bitcoin-confirmed — needs a live calendar + real BTC confirmation. Still 1/1 open. **Carried `low`
  defect:** nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only
  path).

## M-Deploy — Packaged & operable instance
**Status**: **PARTIALLY MET (1 of ~6 Verify items) — the front-of-queue code-closable work (ADR-0013).**
Verified by exploration:
- **SIGTERM trapped — CLOSED this window.** `notifyShutdown()` (`cmd/iscc-monitor/main.go:113-114`)
  registers `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, so `docker
  stop`/orchestrator SIGTERM now cancels the same run context the follower loop drives, draining the
  store via the deferred `st.Close()` instead of SIGKILL over irreplaceable evidence. Verified
  non-vacuous by the `//go:build unix` `shutdown_test.go` (review mutation-proven). The resolved
  `critical` ops issue was correctly deleted.
- **No production `Dockerfile`** — only `.devcontainer/Dockerfile`. The Verify bar wants a tracked
  multi-stage `CGO_ENABLED=0` non-root `scratch`/distroless image + a CI job that runs the container and
  asserts `GET /healthz` → 200.
- **No GHCR publish workflow** — `.github/workflows/` is `ci.yml` + `pages.yml` (the `.codes` verifier
  site) only; nothing pushes `ghcr.io/iscc/iscc-monitor` with `develop` + `sha-<short>` tags.
- **No version stamp** — no `-ldflags` git-SHA injection and no `GET /version` / `/healthz` version
  field (grep-confirmed: the only `ldflags` reference is the WASM-reproducibility doc comment in
  `internal/web/web.go`).
- **No canonical realm doc** — the only realm file is `internal/registry/testdata/realm.txt` (a
  testdata path); no `deploy/` directory exists. CLAUDE.md's env table DOES list the three masthead
  identity keys (verified), satisfying that sub-item.
- **No operability/deployment doc** — nothing states the volume path / backup unit / non-root uid /
  migration policy / egress endpoints / reverse-proxy contract / `/metrics` exposure decision.
- The **on-disk DB migration hazard** (`store.Open` = `CREATE TABLE IF NOT EXISTS` only, no
  `PRAGMA user_version`) remains an open `normal`, which M-Deploy's interim "recreate the volume on
  schema change" policy must document.

## Quality gates
**Status**: **GREEN on every host.** Carried forward — no gate-relevant change this window beyond the
SIGTERM wiring (review confirmed `mise run check` green, gofmt clean, cross-platform-clean with the
Unix-only test correctly excluded on Windows).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs `mise run check` + the `cmd/notecheck` oracle on push/PR.
  Latest CI run `27983760942` at HEAD `39f0403` is **success**; the matching Pages run `27983760940` is
  `in_progress` (publishes the separate `.codes` verifier site — feature-irrelevant to the server gate).
  HEAD == `origin/develop` (nothing unpushed).
- **Latest `review` verdict: PASS / CONTINUE** (SIGTERM trap), gate confirmed green by `review`.
- **Open issues: 5 critical, 5 normal, 13 low.** The five `critical` are the remaining M-Deploy ops asks
  (GHCR image + push workflow, canonical mountable realm doc + identity env, DB-volume persistence
  contract, public-route exposure decision, egress/footprint docs). The five `normal`: the DB-migration
  hazard, the `/` "recent declarers checked" footer, the WASM signature-half gap, the per-hub-Anchor
  design-honesty question, and the missing root README. DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE.

## Next Milestone
**M-Deploy is the gate — five `critical` ops issues remain (SIGTERM just closed). The gate is green, so
the code-closable in-repo Verify items can be verified cleanly.** In priority order:
1. **Add the production `Dockerfile` + GHCR publish workflow** — multi-stage `CGO_ENABLED=0` non-root
   `scratch`/distroless image; a CI job that builds it, runs the container, asserts `GET /healthz` →
   200; a workflow pushing `ghcr.io/iscc/iscc-monitor` on push to `develop` tagged `develop` +
   `sha-<short>`. (Now cleanly unblocked — `docker stop`/SIGTERM drains the store.)
2. **Version-stamp the binary** (git SHA via `-ldflags`, default `dev`) and report it on `/healthz` JSON
   or `GET /version` (HTTP-seam test).
3. **Ship a canonical realm doc** under `deploy/` (e.g. `deploy/realm-testnet.txt`,
   `registry.Parse`-tested) and a tracked **deployment/operability doc** (volume path + backup unit +
   non-root uid + interim "recreate volume on schema change" migration policy + egress endpoints +
   reverse-proxy contract + `/metrics` exposure decision).
4. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Subsequent / parallel: the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the
design-first pass on the WASM signature half; the realm-index per-hub-vs-per-checkpoint Anchor honesty
pass; the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism; the M-UI exit visual-pass + human sign-off (ADR-0012).
