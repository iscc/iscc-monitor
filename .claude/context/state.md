<!-- assessed-at: 89660150663e1da42e8123966d0eeb5bdb36d96f -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete, gate green on every host. The loop is executing the order-independent **M-Deploy** packaging milestone (ADR-0013). Five of ~6 M-Deploy Verify items are now closed (SIGTERM trap, version stamp, production Dockerfile + container `/healthz` smoke, GHCR publish workflow, and this window's canonical realm doc); the remaining gate is the operability doc + the root README.

This window's change was the **canonical realm document**: a tracked `deploy/realm-testnet.txt` (domains-only, ADR-0009), the Dockerfile bake `COPY` repointed from the Go testdata fixture to it, a non-vacuous golden test (`registry.Parse` accepts the real `deploy/` file as exactly the two ordered entries), and the canonical path documented in CLAUDE.md's env table. This closes M-Deploy's "canonical realm document" Verify item and the realm-doc half of the last code-closable `critical`. The remaining work — an operability/deployment doc and a public root `README.md` — is doc-only and code-unblocked, so DONE stays out of reach.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `b0b2fd0..HEAD` diff touched NO feature Go source (only `deploy/realm-testnet.txt`,
    `internal/registry/deploy_test.go`, `Dockerfile`, `CLAUDE.md`, and context files; confirmed by
    `git diff --stat`).
  - **M-UI: behavioral + named-region Verify met; the mandatory M-UI exit visual-pass + human sign-off
    (ADR-0012) still pending.** Instance identity is config-driven on THREE of six SSR mastheads
    (`/`, dossier, certificate); the proofserve trio (`browser.html`, `records.html`, `record.html`)
    still renders the static placeholder — the follow-on arc. No template touched this window.
  - **WASM verifier: published half CLOSED** (Pages live; `monitor.iscc.codes/_ds/verify.wasm` → 200
    `application/wasm`, byte-pinned). **Still open on the milestone Verify**: the cross-origin
    **signature half** (the WASM core verifies inclusion math + id-binding only — no
    checkpoint-signature / did:web resolution) — design-first remainder, `normal`. No dossier tier-2
    WASM caller.
  - **OTS anchoring: 1/1 open (carried).** All observable HTTP halves + both calendar-transport guards
    in place; only a root actually transiting to **Bitcoin-confirmed** remains (offline-unprovable).
  - **M-Deploy: 5 of ~6 Verify items met (SIGTERM trap, version stamp, Dockerfile + container
    `/healthz` CI smoke, GHCR publish workflow, canonical realm doc), milestone PARTIALLY MET.**
    **CLOSED this window:** `deploy/realm-testnet.txt` at a fixed non-testdata path, baked into the
    image at `/etc/iscc-monitor/realm.txt` (Dockerfile `COPY deploy/realm-testnet.txt …`), and
    `registry.Parse`-accepted by a non-vacuous golden test reading the real file; CLAUDE.md's env table
    already names the masthead identity keys. **Still open:** no operability/deployment doc; no root
    README.
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing (canonical realm doc; GHCR publish
  workflow; Dockerfile + container smoke; version stamp; SIGTERM trap; Pages publish unblock) / ~4
  chrome·config-leaf·cert-TZ.** **No drift:** the loop is steadily walking the M-Deploy runway —
  code-closable work that depends on no feature milestone — and closing one Verify criterion per
  iteration (SIGTERM → version stamp → Dockerfile → publish workflow → realm doc). The two remaining
  slices (operability doc, root README) are doc-closable with no feature dependency.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `b0b2fd0..HEAD` diff touched NO Go source under any M1 path
(no signature / RFC-6962 / Merkle / `proof` / `didweb` / `logclient` / `follower` / store source — only
a new `deploy/` realm file, a new registry parse test, the Dockerfile bake line, and CLAUDE.md). All
Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested with
freeze + alert-once + restart survival; structured logs; `/metrics`.
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
pending.** No template-render change this window (the only changes were the `deploy/` realm doc, its
test, the Dockerfile bake line, and CLAUDE.md — no Go handler/template source touched). All six
certificate clauses + both anchor panels + badge + DS shell + `/` index + log browser + hub dossier +
frozen Exhibit + single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint
render and pass the behavioral HTTP-seam Verify.
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
**Status**: **PARTIALLY MET (5 of ~6 Verify items) — the front-of-queue code-closable work (ADR-0013).**
Verified by exploration:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, so `docker stop` /
  orchestrator SIGTERM cancels the run context, draining the store via the deferred `st.Close()`.
  Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at `/version` and serves
  `{"version":"<Version>"}`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED.** Multi-stage build
  (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`, uid 65532, CA roots) producing a
  static stripped `cmd/iscc-monitor` binary with `-trimpath` + `-ldflags "-s -w -X …version.Version=…"`,
  an empty-`VERSION` fail-fast `RUN` guard, and the baked realm. The `ci.yml` `docker` job builds it,
  runs the container, and polls `GET /healthz` → 200.
- **GHCR publish workflow — CLOSED.** `.github/workflows/publish.yml` triggers on push to `develop` +
  `workflow_dispatch`, `permissions: {contents: read, packages: write}`, logs into GHCR, and
  (`docker/build-push-action@v6`) builds the root Dockerfile and pushes BOTH
  `ghcr.io/iscc/iscc-monitor:develop` AND `:sha-<short>` with a non-empty `VERSION` build-arg.
- **Canonical realm document — CLOSED this window.** `deploy/realm-testnet.txt` exists at the fixed
  repo-root `deploy/` (non-testdata) path, is domains-only (ADR-0009), and is baked into the image at
  the documented `/etc/iscc-monitor/realm.txt` (`Dockerfile:46` `COPY deploy/realm-testnet.txt …`; the
  testdata reference is gone from the bake line). `registry.Parse` accepts it as exactly
  `sb0.iscc.id` + `sb1.amlet.id` via the non-vacuous golden `TestParseCanonicalDeployRealm`
  (`internal/registry/deploy_test.go`, reads the real file). CLAUDE.md's env table names the canonical
  doc and the three masthead identity keys (`ISCC_MONITOR_INSTANCE` / `_OPERATOR` / `_REALM_NAME`).
- **No operability/deployment doc** — nothing states the volume path / single-file backup unit /
  non-root uid / migration policy / egress endpoints / reverse-proxy contract / `/metrics` exposure
  decision. This single doc folds in all THREE remaining `critical` infra asks.
- **No root `README.md`** — the "Done When" requirement. The only README is the loop-internal
  `.claude/context/README.md`; `CLAUDE.md` is agent-facing.
- The **on-disk DB migration hazard** (`store.Open` = `CREATE TABLE IF NOT EXISTS` only, no
  `PRAGMA user_version`) remains an open `normal`, which the operability doc must document as the
  interim "recreate the volume on schema change" policy.
- **Carried `normal` trap:** `publish.yml`'s publish job has no ref guard, so a `workflow_dispatch` from
  a non-develop ref would move the floating `:develop` tag (immutable `:sha-<short>` unaffected). Mirrors
  the `pages.yml` convention; maintainer-only; fold-in when a workflow file is next touched. **Carried
  `low`:** the `.dockerignore` secret/sidecar globs are slashless (root-level only, not recursive).

## Quality gates
**Status**: **GREEN.** No feature Go source changed this window. The one new test
(`internal/registry/deploy_test.go`) and the source-path-swap Dockerfile change were reviewed PASS with
the gate confirmed green; `go.mod`/`go.sum` byte-identical.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` (the `.codes` verifier site) + `publish.yml` (GHCR image push). At HEAD
  `8966015`: **CI `success`, Pages `success`, Publish `in_progress`** (the image build/push is still
  running — expected to take longer than the source gates; not a failure). HEAD == `origin/develop`
  (nothing unpushed).
- **Latest `review` verdict: PASS / CONTINUE** (canonical `deploy/realm-testnet.txt` + bake repoint +
  golden test + CLAUDE.md doc; mutation-proven non-vacuous, Codex-clean), gate confirmed green by
  `review`.
- **Open issues: 3 critical, 7 normal, 19 low.** The three `critical` are the remaining M-Deploy
  iscc-infra ops asks (DB-volume persistence contract; public-route `/metrics` exposure decision;
  egress/footprint sizing) — all closable by the single operability doc. The `normal`s include the
  DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half gap, the
  per-hub-Anchor design-honesty question, the missing root README, the `publish.yml` `workflow_dispatch`
  ref-guard, and the `.dockerignore` slashless-globs note. DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE.

## Next Milestone
**M-Deploy is the gate — three `critical` ops issues remain (SIGTERM + version stamp + Dockerfile/CI
smoke + GHCR publish workflow + canonical realm doc now all closed). The gate is green, so the remaining
doc-closable Verify items can be landed cleanly.** In priority order:
1. **Ship a tracked deployment/operability doc** — SQLite volume path + single-file backup unit +
   non-root uid + the interim "recreate the volume on schema change" migration policy + egress endpoints
   (hub `/log`, did:web `/.well-known/did.json`, the OTS calendar) + reverse-proxy contract (binds
   `:9464`, publishes no host port) + the `/metrics` exposure decision. This single doc folds in all
   THREE remaining `critical` infra asks (persistence contract, public-route exposure, egress/footprint).
2. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Fold-in candidate whenever a workflow file is next touched: the `publish.yml` (and `pages.yml`)
`workflow_dispatch` ref-guard (`if: github.ref == 'refs/heads/develop'`).

Subsequent / parallel (all `normal`/`low`, none gate DONE once the two docs land except where noted):
the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the design-first pass on
the WASM signature half (`normal`); the realm-index per-hub-vs-per-checkpoint Anchor honesty pass
(`normal`); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism (`normal`); the M-UI exit visual-pass + human sign-off (ADR-0012).
