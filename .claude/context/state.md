<!-- assessed-at: 395be65dda11758922236c24718817dd2d55e78b -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete; gate green on every host. The loop is executing the order-independent **M-Deploy** packaging milestone (ADR-0013). Three of ~6 M-Deploy Verify items now closed (SIGTERM trap, version stamp, and this window's production Dockerfile + CI container `/healthz` smoke job); the remaining blockers are the GHCR publish workflow, the canonical `deploy/` realm doc, the operability doc, and the root README.

The feature surface is built; the loop is on the M-Deploy runway. This window's production change was
the deployment artifact: a tracked multi-stage `CGO_ENABLED=0` distroless-nonroot `Dockerfile` plus a
sibling CI `docker` job that builds the image, runs the container, and asserts `GET /healthz` → 200.
M-Deploy is now met on 3 of ~6 Verify items, but no GHCR publish workflow, no `deploy/` realm doc, no
operability doc, and no root `README.md` exist, so DONE stays far out of reach.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `5f069d5..HEAD` diff touched NO Go source at all (only `Dockerfile`, `.dockerignore`,
    `.github/workflows/ci.yml`, and context files).
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
  - **M-Deploy: 3 of ~6 Verify items met (SIGTERM trap, version stamp, Dockerfile + container
    `/healthz` CI smoke), milestone PARTIALLY MET.** **CLOSED this window:** the tracked root
    `Dockerfile` (multi-stage `golang:1.26.4` build → `distroless/static-debian12:nonroot`, static
    stripped binary, CA roots, baked interim realm) plus the `docker` CI job that builds it, runs the
    container, and asserts `GET /healthz` → 200 — review-proven host-equivalent (static/stripped ELF,
    boots, serves `/healthz` 200 + `/version` stamp) plus the empty-`VERSION` fail-fast guard.
    **Still open:** no GHCR publish workflow (push `ghcr.io/iscc/iscc-monitor` tagged `develop` +
    `sha-<short>`); no canonical `deploy/` realm doc; no operability/deployment doc; no root README.
- **Last ~10 iterations: ~4 milestone-Verify-or-gate-advancing (Pages publish unblock; SIGTERM trap;
  version stamp; this window's Dockerfile + container smoke) / ~6 chrome·plumbing·config-leaf.** **No
  drift:** the loop is on the M-Deploy runway — code-closable work that depends on no feature milestone,
  and it is steadily closing M-Deploy Verify criteria (SIGTERM → version stamp → Dockerfile). The next
  slice is the GHCR publish workflow (which publishes the image just built).

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `5f069d5..HEAD` diff touched NO Go source (only `Dockerfile`,
`.dockerignore`, `ci.yml`, context); no signature / RFC-6962 / Merkle / `proof` / `didweb` / `logclient`
/ `follower` / store source. All Verify criteria remain satisfied: `origin`/`vkey` golden;
fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured logs;
`/metrics`.
- **Packages present** (28 source pkgs incl. `internal/version`): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`;
  internal — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles,
  tilesserve, verifier, version, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1` language
  directive; toolchain `mise.toml` `go = "1.26.4"`.
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
pending.** No template-render change this window (the only prod change was the `Dockerfile` +
`.dockerignore` + CI config — no Go source, no templates touched). All six certificate clauses + both
anchor panels + badge + DS shell + `/` index + log browser + hub dossier + frozen Exhibit +
single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the
behavioral HTTP-seam Verify.
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
**Status**: **PARTIALLY MET (3 of ~6 Verify items) — the front-of-queue code-closable work (ADR-0013).**
Verified by exploration:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go:115-116`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, so `docker stop` /
  orchestrator SIGTERM cancels the run context, draining the store via the deferred `st.Close()`.
  Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at the exact reserved path `/version`
  (`main.go:293`) and serves `{"version":"<Version>"}`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED this window.** The tracked root
  `Dockerfile` is a multi-stage build (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`,
  uid 65532, CA roots) producing a static stripped `cmd/iscc-monitor` binary with `-trimpath` +
  `-ldflags "-s -w -X …version.Version=${VERSION}"`, an empty-`VERSION` fail-fast `RUN` guard, and the
  baked interim realm at `/etc/iscc-monitor/realm.txt`. The sibling `ci.yml` `docker` job builds it
  (`--build-arg VERSION="$(git rev-parse --short HEAD)"`), runs the container, and polls `GET /healthz`
  → 200 within 15s. Review-proven host-equivalent (static/stripped ELF boots, serves `/healthz` 200 +
  `/version` stamp); Docker itself runs only in CI (absent on the build host).
- **No GHCR publish workflow** — `.github/workflows/` is `ci.yml` (check + docker smoke) + `pages.yml`
  (the `.codes` verifier site) only; nothing pushes `ghcr.io/iscc/iscc-monitor` with `develop` +
  `sha-<short>` tags. This is the **open half of the GHCR `critical`** (image-build half now closed).
- **No canonical realm doc** — the only realm file is `internal/registry/testdata/realm.txt` (a
  testdata path, baked into the image as the interim source); no `deploy/` directory exists. CLAUDE.md's
  env table DOES list the three masthead identity keys (verified), satisfying that sub-item.
- **No operability/deployment doc** — nothing states the volume path / backup unit / non-root uid /
  migration policy / egress endpoints / reverse-proxy contract / `/metrics` exposure decision.
- The **on-disk DB migration hazard** (`store.Open` = `CREATE TABLE IF NOT EXISTS` only, no
  `PRAGMA user_version`) remains an open `normal`, which M-Deploy's interim "recreate the volume on
  schema change" policy must document.
- **Carried `normal` traps to fold into the GHCR-publish slice:** (1) `.dockerignore` does not mirror
  the gitignored secret patterns (`.env`/`.env.*`/`**/auth.json`) or the WAL/SHM DB sidecars
  (`*.db-wal`/`*.db-shm`) — build-stage-layer-only, never reaches the published image (final stage only
  `COPY --from=build`s the binary) nor a clean CI checkout, so it does not block PASS. (2) the host
  `mise.toml build:monitor`'s `$(git rev-parse …)` still empty-expands on git failure, silently stamping
  an empty `/version` (the *image* path is already guarded by the Dockerfile fail-fast `RUN`).

## Quality gates
**Status**: **GREEN on every host.** Carried forward — no Go source changed this window (Dockerfile + CI
config only). Review confirmed `mise run check` green, gofmt clean, `go mod verify` OK (no new
dependency), `go.mod`/`go.sum`/`schema.sql` byte-identical.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable.
- **CI**: `.github/workflows/ci.yml` runs `mise run check` + the `cmd/notecheck` oracle, plus the new
  sibling `docker` job (image build + container `/healthz` smoke), on push/PR. Latest CI run at HEAD
  `395be65` is **success**; the matching Pages run is also **success** (publishes the separate `.codes`
  verifier site — feature-irrelevant to the server gate). HEAD == `origin/develop` (nothing unpushed).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (production Dockerfile + docker smoke job),
  gate confirmed green by `review`.
- **Open issues: 5 critical, 7 normal, 13 low.** The five `critical` are the remaining M-Deploy ops asks
  (GHCR image **publish** workflow — image-build half now closed; canonical mountable realm doc + identity
  env; DB-volume persistence contract; public-route exposure decision; egress/footprint docs). The seven
  `normal`: the DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half
  gap, the per-hub-Anchor design-honesty question, the missing root README, the `build:monitor`
  empty-stamp-on-git-failure trap, and the new `.dockerignore` secret/sidecar hygiene gap. DONE requires
  0 critical AND 0 normal, so the loop stays CONTINUE.

## Next Milestone
**M-Deploy is the gate — five `critical` ops issues remain (SIGTERM + version stamp + Dockerfile/CI smoke
now closed). The gate is green, so the code-closable in-repo Verify items can be verified cleanly.** In
priority order:
1. **Add the GHCR publish workflow** — M-Deploy's second Verify bullet and the open half of the GHCR
   `critical`. On push to `develop`, build + push `ghcr.io/iscc/iscc-monitor` tagged BOTH `develop`
   (floating) AND `sha-<short>` (immutable, for pin/rollback) — `docker/login-action` +
   `docker/build-push-action` (or a plain `docker push` with `GITHUB_TOKEN` + `packages: write`), as a
   new `.github/workflows/publish.yml` or a `ci.yml` job; it publishes the Dockerfile just landed.
   **Fold in** the two carried `normal` traps that this slice naturally touches: the `.dockerignore`
   secret/sidecar hardening (`.env`/`.env.*`/`**/auth.json`/`*.db-wal`/`*.db-shm`) and the host
   `build:monitor` empty-SHA fail-fast fix.
2. **Ship a canonical realm doc** under `deploy/` (e.g. `deploy/realm-testnet.txt`,
   `registry.Parse`-tested) and a tracked **deployment/operability doc** (volume path + backup unit +
   non-root uid + interim "recreate volume on schema change" migration policy + egress endpoints +
   reverse-proxy contract + `/metrics` exposure decision). Point the Dockerfile's baked realm at the
   canonical `deploy/` doc once it exists.
3. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Subsequent / parallel: the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the
design-first pass on the WASM signature half; the realm-index per-hub-vs-per-checkpoint Anchor honesty
pass; the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism; the M-UI exit visual-pass + human sign-off (ADR-0012).
