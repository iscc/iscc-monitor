<!-- assessed-at: b0b2fd04b4fc93b7785988eb1355eb21d9c4dff1 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete, gate green on every host. The loop is executing the order-independent **M-Deploy** packaging milestone (ADR-0013). Four of ~6 M-Deploy Verify items now closed (SIGTERM trap, version stamp, production Dockerfile + container `/healthz` smoke, and this window's GHCR publish workflow); the remaining blockers are the canonical `deploy/` realm doc, the operability doc, and the root README.

This window's production change was the deployment-publish artifact: a tracked `.github/workflows/publish.yml` that builds the root Dockerfile and pushes `ghcr.io/iscc/iscc-monitor` tagged BOTH `:develop` and `:sha-<short>` on push to `develop`, plus a `.dockerignore` secret/sidecar fold-in and a `mise.toml build:monitor` empty-SHA fail-fast. The `Publish` workflow ran green at HEAD, so the GHCR image-build/publish half is now CI-proven live. M-Deploy is met on 4 of ~6 Verify items, but no `deploy/` realm doc, no operability doc, and no root `README.md` exist, so DONE stays out of reach.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `395be65..HEAD` diff touched NO Go source at all (only `.dockerignore`, `publish.yml`, `mise.toml`,
    and context files; confirmed by `git diff --name-only`).
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
  - **M-Deploy: 4 of ~6 Verify items met (SIGTERM trap, version stamp, Dockerfile + container
    `/healthz` CI smoke, GHCR publish workflow), milestone PARTIALLY MET.** **CLOSED this window:** the
    tracked `.github/workflows/publish.yml` — push-to-`develop` + `workflow_dispatch`, `packages: write`,
    `docker/login-action@v3` + `docker/build-push-action@v6` building the root Dockelfile and pushing
    BOTH `ghcr.io/iscc/iscc-monitor:develop` (floating) AND `:sha-<short>` (immutable) with a non-empty
    `VERSION=${{ github.sha }}` build-arg. The `Publish` run is GREEN at HEAD, so the image actually
    builds+pushes. **Still open:** no canonical `deploy/` realm doc; no operability/deployment doc; no
    root README.
- **Last ~10 iterations: ~5 milestone-Verify-or-gate-advancing (Pages publish unblock; SIGTERM trap;
  version stamp; Dockerfile + container smoke; this window's GHCR publish workflow) / ~5
  chrome·plumbing·config-leaf.** **No drift:** the loop is on the M-Deploy runway — code-closable work
  that depends on no feature milestone, and it is steadily closing M-Deploy Verify criteria
  (SIGTERM → version stamp → Dockerfile → publish workflow). The next slices are the canonical
  `deploy/` realm doc and the operability/deployment doc (both code/doc-closable, no feature dep).

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `395be65..HEAD` diff touched NO Go source (only
`.dockerignore`, `publish.yml`, `mise.toml`, context); no signature / RFC-6962 / Merkle / `proof` /
`didweb` / `logclient` / `follower` / store source. All Verify criteria remain satisfied: `origin`/`vkey`
golden; fork/shrink/equivocation golden-tested with freeze + alert-once + restart survival; structured
logs; `/metrics`.
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
pending.** No template-render change this window (the only prod change was `publish.yml` + `.dockerignore`
+ `mise.toml` — no Go source, no templates touched). All six certificate clauses + both anchor panels +
badge + DS shell + `/` index + log browser + hub dossier + frozen Exhibit + single-record page +
ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam
Verify.
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
**Status**: **PARTIALLY MET (4 of ~6 Verify items) — the front-of-queue code-closable work (ADR-0013).**
Verified by exploration:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, so `docker stop` /
  orchestrator SIGTERM cancels the run context, draining the store via the deferred `st.Close()`.
  Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at the reserved path `/version` and
  serves `{"version":"<Version>"}`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED (prior window).** The tracked root
  `Dockerfile` is a multi-stage build (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`,
  uid 65532, CA roots) producing a static stripped `cmd/iscc-monitor` binary with `-trimpath` +
  `-ldflags "-s -w -X …version.Version=${VERSION}"`, an empty-`VERSION` fail-fast `RUN` guard, and the
  baked interim realm at `/etc/iscc-monitor/realm.txt`. The `ci.yml` `docker` job builds it, runs the
  container, and polls `GET /healthz` → 200. The `docker` job retains NO registry login (publish lives in
  its own file).
- **GHCR publish workflow — CLOSED this window.** `.github/workflows/publish.yml` triggers on push to
  `develop` + `workflow_dispatch`, `permissions: {contents: read, packages: write}`, logs into GHCR with
  `GITHUB_TOKEN`, and (`docker/build-push-action@v6`) builds the root Dockerfile and pushes BOTH
  `ghcr.io/iscc/iscc-monitor:develop` AND `ghcr.io/iscc/iscc-monitor:sha-${{short SHA}}` with a non-empty
  `VERSION=${{ github.sha }}` build-arg. The **`Publish` run is GREEN at HEAD `b0b2fd0`** — the image
  actually builds and pushes (not just statically inspected). The remaining GHCR work (make the package
  public / issue a `read:packages` token) is iscc-infra repo-settings, explicitly out of the loop's scope
  (the matching issue was correctly demoted `critical → low`).
- **No canonical realm doc** — the only realm file is `internal/registry/testdata/realm.txt` (a
  testdata path, baked into the image as the interim source); no `deploy/` directory exists. CLAUDE.md's
  env table DOES list the three masthead identity keys (verified), satisfying that sub-item.
- **No operability/deployment doc** — nothing states the volume path / backup unit / non-root uid /
  migration policy / egress endpoints / reverse-proxy contract / `/metrics` exposure decision.
- The **on-disk DB migration hazard** (`store.Open` = `CREATE TABLE IF NOT EXISTS` only, no
  `PRAGMA user_version`) remains an open `normal`, which M-Deploy's interim "recreate the volume on
  schema change" policy must document.
- **Carried `normal` trap:** `publish.yml`'s `publish` job has no ref guard, so a `workflow_dispatch`
  from a non-develop ref would move the floating `:develop` tag (the immutable `:sha-<short>` is
  unaffected). Mirrors the pre-existing `pages.yml` convention; maintainer-only; fold-in when a workflow
  file is next touched. **Carried `low`:** the `.dockerignore` secret/sidecar globs are slashless (root-
  level only, not recursive) — never reaches the shipped image or a clean CI checkout.

## Quality gates
**Status**: **GREEN on every host.** Carried forward — no Go source changed this window (`publish.yml` +
`.dockerignore` + `mise.toml` task-table only). Review confirmed `mise run check` green, gofmt clean
outside `cauldron/`, no new dependency, `go.mod`/`go.sum`/`schema.sql`/Dockerfile/`ci.yml` byte-identical.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable. The `mise.toml build:monitor` task now fail-fasts on an empty git SHA.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` (the `.codes` verifier site) + `publish.yml` (GHCR image push). At HEAD
  `b0b2fd0` **all three workflows are `success`** (CI, Pages, Publish). HEAD == `origin/develop`
  (nothing unpushed).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (GHCR publish workflow + `.dockerignore` +
  `build:monitor` fail-fast), gate confirmed green by `review`.
- **Open issues: 4 critical, 6 normal, 15 low.** The four `critical` are the remaining M-Deploy ops
  asks (canonical mountable realm doc + identity env; DB-volume persistence contract; public-route
  exposure decision; egress/footprint docs) — all closable by the `deploy/` realm doc + the operability
  doc. The GHCR image-publish `critical` was demoted to `low` (code-complete). The six `normal`: the
  DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half gap, the
  per-hub-Anchor design-honesty question, the missing root README, and the `publish.yml`
  `workflow_dispatch` ref-guard. DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.

## Next Milestone
**M-Deploy is the gate — four `critical` ops issues remain (SIGTERM + version stamp + Dockerfile/CI smoke
+ GHCR publish workflow now closed). The gate is green, so the remaining code/doc-closable Verify items
can be landed cleanly.** In priority order:
1. **Ship a canonical realm doc** under `deploy/` (e.g. `deploy/realm-testnet.txt`, `registry.Parse`-tested),
   then point the Dockerfile's baked realm at it. This is M-Deploy's "canonical realm document" Verify
   bullet and closes the realm-doc half of the "mountable realm file + identity env" `critical`
   (CLAUDE.md already lists the identity keys).
2. **Ship a tracked deployment/operability doc** — volume path + single-file backup unit + non-root uid +
   the interim "recreate the volume on schema change" migration policy + egress endpoints (hub `/log`,
   did:web, OTS calendar) + reverse-proxy contract (binds `:9464`, publishes no host port) + the
   `/metrics` exposure decision. This single doc folds in the remaining THREE `critical` infra asks
   (persistence contract, public-route exposure, egress/footprint).
3. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Fold-in candidate whenever a workflow file is next touched: the `publish.yml` (and `pages.yml`)
`workflow_dispatch` ref-guard (`if: github.ref == 'refs/heads/develop'`).

Subsequent / parallel: the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the
design-first pass on the WASM signature half; the realm-index per-hub-vs-per-checkpoint Anchor honesty
pass; the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism; the M-UI exit visual-pass + human sign-off (ADR-0012).
