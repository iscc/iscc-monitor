<!-- assessed-at: a9f088a7e2cccb8b2e14e5eb9cdab232a975e0c6 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete; the gate is now **green on every host** (cert local-TZ bug fixed + pushed). The remaining gate is the **M-Deploy** packaging milestone (6 critical ops issues, 0% built) plus a root README.

The feature surface is built; the loop is executing the order-independent **M-Deploy** milestone
(ADR-0013). This window's only production change was the cert-timestamp UTC fix
(`internal/certificate/handler.go`), which **cleared the last gate-red condition** — `mise run check`
is now green on any host timezone, not just UTC CI runners. M-Deploy is still 0/Verify (no Dockerfile,
no GHCR workflow, no SIGTERM trap, no version stamp, no `deploy/` realm doc, no operability doc) and the
root `README.md` is still missing, so DONE stays far out of reach.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no trust-path
    source touched since the last full pass (this window's only prod edit was the cert handler's
    timestamp rendering).
  - **M-UI: behavioral + named-region Verify met; the mandatory M-UI exit visual-pass + human
    sign-off (ADR-0012) still pending.** Instance identity is config-driven on THREE of six SSR
    mastheads (`/`, dossier, certificate); the proofserve trio (`browser.html`, `records.html`,
    `record.html`) still renders the static placeholder — the follow-on arc.
  - **WASM verifier: published half CLOSED** (Pages live; `monitor.iscc.codes/_ds/verify.wasm` → 200
    `application/wasm`, byte-pinned). **Still open on the milestone Verify**: the cross-origin
    **signature half** (the WASM core verifies inclusion math + id-binding only — no
    checkpoint-signature / did:web resolution; a malicious monitor can still render green `verified`)
    — design-first remainder, `normal`. No dossier tier-2 WASM caller.
  - **OTS anchoring: 1/1 open (carried).** All observable HTTP halves + both calendar-transport guards
    in place; only a root actually transiting to **Bitcoin-confirmed** remains (offline-unprovable).
  - **M-Deploy: 0 of its Verify bar met (milestone NOT STARTED, verified by exploration).** No root
    `Dockerfile` (only `.devcontainer/Dockerfile`); no GHCR publish workflow (`.github/workflows/` =
    `ci.yml` + `pages.yml` only); `run()` traps **SIGINT only** (`main.go:133`
    `signal.NotifyContext(context.Background(), os.Interrupt)` — no `syscall.SIGTERM`); no version
    stamp (grep-confirmed: no `-ldflags`/`GET /version`/`/healthz` version field); no `deploy/`
    canonical realm doc (only `internal/registry/testdata/realm.txt`); no operability/deployment doc.
  - **Root README: missing (blocks "Done When").** Repo root has no `README.md`.
- **Last ~10 iterations: ~2 milestone-Verify-or-gate-advancing (Pages publish unblock; this window's
  cert UTC gate-greening) / ~8 chrome·plumbing·config-leaf.** **DRIFT resolved:** the masthead-polish
  streak has ended and the loop is now on the M-Deploy runway — a large body of code-closable work that
  depends on no feature milestone. The cert UTC fix was a clean gate-greening prerequisite (the
  M-Deploy CI/container jobs all run `mise run check`); the next slices are the SIGTERM trap and the
  Dockerfile/GHCR work.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `24544bf..HEAD` diff touched no signature / RFC-6962 /
Merkle / `proof` / `didweb` / `logclient` / `follower` / store source — the only production edit was
`internal/certificate/handler.go` (timestamp `.UTC()` normalization, a presentation fix). All Verify
criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze +
alert-once + restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged, 27 source pkgs): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`;
  internal — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1` language directive;
  toolchain `mise.toml` `go = "1.26.4"`. 409 `func Test` across the tree.
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
pending.** No template-render change this window — the cert-handler edit only zone-normalizes already-
rendered RFC-3339 timestamps. All six certificate clauses + both anchor panels + badge + DS shell + `/`
index + log browser + hub dossier + frozen Exhibit + single-record page + ISCC-IDv1 decoder + Hub-List
resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam Verify.
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
**Status**: **NOT STARTED (0 of the Verify bar met) — the front-of-queue code-closable work
(ADR-0013).** Verified by exploration:
- **No production `Dockerfile`** — only `.devcontainer/Dockerfile`. The Verify bar wants a tracked
  multi-stage `CGO_ENABLED=0` non-root `scratch`/distroless image + a CI job that runs the container and
  asserts `GET /healthz` → 200.
- **No GHCR publish workflow** — `.github/workflows/` is `ci.yml` + `pages.yml` (the `.codes` verifier
  site) only; nothing pushes `ghcr.io/iscc/iscc-monitor` with `develop` + `sha-<short>` tags.
- **SIGTERM not trapped** — `main.go:133` is `signal.NotifyContext(context.Background(), os.Interrupt)`
  (SIGINT only); `docker stop`/orchestrators send SIGTERM, which would kill the process before the
  deferred `store.Close()` runs. One of the six `critical` ops issues.
- **No version stamp** — no `-ldflags` git-SHA injection and no `GET /version` / `/healthz` version
  field (grep-confirmed empty).
- **No canonical realm doc** — the only realm file is `internal/registry/testdata/realm.txt` (a
  testdata path); no `deploy/realm-testnet.txt`. CLAUDE.md's env table DOES list the three masthead
  identity keys (verified), satisfying that sub-item.
- **No operability/deployment doc** — nothing states the volume path / backup unit / non-root uid /
  migration policy / egress endpoints / reverse-proxy contract / `/metrics` exposure decision.
- The **on-disk DB migration hazard** (`store.Open` = `CREATE TABLE IF NOT EXISTS` only, no
  `PRAGMA user_version`) remains an open `normal`, which M-Deploy's interim "recreate the volume on
  schema change" policy must document.

## Quality gates
**Status**: **GREEN on every host.** The last gate-red condition — the cert handler rendering
coverage + §5 timestamps in host-local time on non-UTC boxes — is **fixed** (`24544bf..HEAD`):
all four `.Format(time.RFC3339)` sites in `internal/certificate/handler.go` (`:653`, `:962`, `:1013`,
`:1040`) are now `.UTC().Format(...)` (grep-confirmed: no bare `.Format(time.RFC3339)` remains). The
review handoff verifies green on UTC, America/New_York, and Asia/Kolkata, mutation-proven non-vacuous.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs `mise run check` + the `cmd/notecheck` oracle on push/PR.
  Latest CI run `27982826462` at HEAD `a9f088a` is `success`; latest Pages run `27982826414` at the
  same SHA is `success`. HEAD == `origin/develop` (nothing unpushed).
- **Latest `review` verdict: PASS / CONTINUE** (cert UTC fix), gate confirmed green by `review`.
- **Open issues: 6 critical, 5 normal, 13 low.** The six `critical` are the M-Deploy ops asks (GHCR
  image + push workflow, SIGTERM trap, canonical mountable realm doc, DB-volume persistence contract,
  public-route exposure decision, egress/footprint docs). The five `normal`: the DB-migration hazard,
  the `/` "recent declarers checked" footer, the WASM signature-half gap, the per-hub-Anchor
  design-honesty question, and the missing root README. (The cert local-TZ `normal` and one Pages-domain
  `normal` were pruned this window — resolved.) DONE requires 0 critical AND 0 normal, so the loop
  stays CONTINUE.

## Next Milestone
**M-Deploy is the gate — six `critical` ops issues, none built. The gate is now green, so the
code-closable in-repo Verify items can be verified cleanly.** In priority order:
1. **Trap SIGTERM** — add `syscall.SIGTERM` to `signal.NotifyContext` in `cmd/iscc-monitor/main.go:133`
   with a test (send SIGTERM → context cancels → `store.Close()` runs → exit 0); reverting the
   registration must FAIL the test. Smallest critical, pure-code, high-leverage — and the cleanest
   unblock before the Dockerfile work so `docker stop` drains the store cleanly.
2. **Add the production `Dockerfile` + GHCR publish workflow** — multi-stage `CGO_ENABLED=0` non-root
   `scratch`/distroless image; a CI job that builds it, runs the container, asserts `GET /healthz` →
   200; a workflow pushing `ghcr.io/iscc/iscc-monitor` on push to `develop` tagged `develop` +
   `sha-<short>`.
3. **Version-stamp the binary** (git SHA via `-ldflags`, default `dev`) and report it on `/healthz` JSON
   or `GET /version` (HTTP-seam test).
4. **Ship a canonical realm doc** under `deploy/` (e.g. `deploy/realm-testnet.txt`,
   `registry.Parse`-tested) and a tracked **deployment/operability doc** (volume path + backup unit +
   non-root uid + interim "recreate volume on schema change" migration policy + egress endpoints +
   reverse-proxy contract + `/metrics` exposure decision).
5. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Subsequent / parallel: the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the
design-first pass on the WASM signature half; the realm-index per-hub-vs-per-checkpoint Anchor honesty
pass; the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism; the M-UI exit visual-pass + human sign-off (ADR-0012).
