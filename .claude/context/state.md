<!-- assessed-at: ba111e86aa08e66f2356ceee27b94aea0a01b1d5 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete, gate green. The order-independent **M-Deploy** packaging milestone (ADR-0013) is the active runway. The **last code/doc-closable `target.md` "Done When" gate — the public-facing root `README.md` — is now CLOSED** (HEAD `ba111e8`). DONE now turns ENTIRELY on the three open `critical` iscc-infra ops issues.

This window (`0753b65..ba111e8`) the advance added the tracked root `README.md` (verifiable-cache
overview, Go 1.26 / `CGO_ENABLED=0` stack, copy-pasteable testnet build/run snippet, `mise run check`
gate, GHCR/deploy pointer, spec links; env-var table LINKED to `CLAUDE.md`, not duplicated). The diff
touched **NO Go source** — only `README.md` + the context pack (handoff/issues/next/state). Review
verdict **PASS / CONTINUE**; the README `normal` is closed. HEAD `ba111e8` is pushed and level with
`origin/develop`.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `0753b65..ba111e8` diff touched NO Go source (only `README.md` + context pack; confirmed by
    `git diff --stat`).
  - **M-UI: behavioral + named-region Verify met; the mandatory M-UI exit visual-pass + human sign-off
    (ADR-0012) still pending.** Instance identity config-driven on THREE of six SSR mastheads
    (`/`, dossier, certificate); the proofserve trio still renders the static placeholder. No template
    touched this window.
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200
    `application/wasm`, byte-pinned). **Still open:** the cross-origin **signature half** (the WASM core
    verifies inclusion math + id-binding only — no checkpoint-signature / did:web resolution),
    design-first remainder, `normal`.
  - **OTS anchoring: 1/1 open (carried).** All observable HTTP halves + both calendar-transport guards in
    place; only a root actually transiting to **Bitcoin-confirmed** remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED, including the root `README.md` "Done When" gate.** The
    only residual is confirming/pruning the three iscc-infra `critical`s (answered in substance by
    `deploy/OPERATING.md` + `README.md` but still OPEN entries) plus the Compose volume-prefix chown
    `normal`.
- **Last ~10 iterations: ~7 milestone-Verify-or-gate-advancing (canonical realm doc; GHCR publish
  workflow; Dockerfile + container smoke; version stamp; SIGTERM trap; Pages publish unblock; realm-bake
  so the quick-start boots; root README) / ~3 chrome·config-leaf·cert-TZ.** **No drift:** the loop has
  steadily walked the M-Deploy runway to its end — every code/doc-closable item is now landed. The
  remaining DONE blockers (the three iscc-infra `critical`s) are doc-confirmation/pruning work this repo
  owns, not feature code; if any proves truly external with nothing left to close in-repo, that is a
  STOP/IDLE edge the loop should surface rather than spin on cosmetic chrome.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `0753b65..ba111e8` diff touched NO Go source under any M1
path. All Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation
golden-tested with freeze + alert-once + restart survival; structured logs; `/metrics`.
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
pending.** No template-render change this window. All six certificate clauses + both anchor panels +
badge + DS shell + `/` index + log browser + hub dossier + frozen Exhibit + single-record page +
ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral HTTP-seam
Verify.
- **Still open (NOT critical, carried):** instance identity is config-driven on THREE of six SSR
  mastheads; the proofserve trio (`browser.html`, `records.html`, `record.html`) still renders the
  static `monitor instance` placeholder. The `/` "recent declarers checked" hero footer is omitted
  (needs store lookup history; `normal`). The mandatory **M-UI exit visual-pass + human sign-off**
  (ADR-0012) is not executed.
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
  `monitor.iscc.codes` serves the byte-pinned `/_ds/verify.wasm` (Pages run on HEAD `ba111e8` green).
  **Still open:** the cross-origin **SIGNATURE-half gap** (the verifier core does NO
  checkpoint-signature / did:web check — the success copy overstates a key check that never runs) —
  design-first remainder / STOP-candidate, `normal`. **Carried `low`:** `cmd/verifier-site` writes
  non-atomically; `pages.yml` actions target deprecated Node 20.
- **OTS:** the `.ots` serve route, the §5 anchor clause (digest-bound via `ots.ConfirmedFor`), the store
  layer, the off-path stamp/upgrade loop (`OTSTick`), the offline classifier, and the calendar transport
  (both `safeUpgrade` + `safeStamp` guards) are wired. The Verify-closer not yet built: a root reaching
  Bitcoin-confirmed — needs a live calendar + real BTC confirmation. Still 1/1 open. **Carried `low`
  defect:** nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only
  path).

## M-Deploy — Packaged & operable instance
**Status**: **ALL in-repo Verify items CLOSED, including the root `README.md` "Done When" gate. DONE now
turns solely on confirming/pruning the three iscc-infra `critical`s.** Verified by exploration + CI:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, draining the store via
  the deferred `st.Close()`. Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at `/version`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED + STRENGTHENED.** Multi-stage build
  (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`, uid 65532, CA roots), empty-`VERSION`
  fail-fast `RUN` guard, baked realm, AND `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt`
  (`Dockerfile:51`). The `ci.yml` `docker` job builds it and runs the container with NO
  `-e ISCC_MONITOR_REALM` — so a realm-less `/healthz` → 200 mechanically proves the bake. CI run on HEAD
  `ba111e8` green.
- **GHCR publish workflow — CLOSED.** `.github/workflows/publish.yml` triggers on push to `develop` +
  `workflow_dispatch`, logs into GHCR, and pushes BOTH `:develop` AND `:sha-<short>`. (Publish run on
  HEAD `ba111e8` is in-progress at assessment time, just kicked off by the README push; the prior run on
  `0753b65` was green and this window changed no Dockerfile/workflow.)
- **Canonical realm document — CLOSED.** `deploy/realm-testnet.txt` at the fixed repo-root path, baked
  at `/etc/iscc-monitor/realm.txt` (`Dockerfile:46`) and ENV-pointed, `registry.Parse`-accepted by
  the non-vacuous golden `TestParseCanonicalDeployRealm`. CLAUDE.md's env table names the three masthead
  identity keys.
- **Operability doc — CLOSED (substance + booting quick-start).** `deploy/OPERATING.md` exists, accurate
  and well-sourced (uid 65532, bind `:9464`/no host-publish, baked realm path, OTS calendar host, WAL
  siblings, single-writer, `GET /version` shape, SIGTERM drain, `:develop`+`:sha-<short>` tags, the
  `/metrics` exposure decision, the "recreate-volume-on-schema-change" interim migration policy).
  **Residual `normal`:** the volume-prep `chown` literal-names `monitor-data`, but the headline Compose
  fragment declares the volume with no `name:`/`external:`, so `docker compose up` mounts a
  project-prefixed volume the chown never touched — the Compose path still fails permission-denied at
  `store.Open`. Doc-correctness gap, not a code defect; does not block progress.
- **Root `README.md` — CLOSED (this window).** The public-facing front door now exists at repo root
  (verifiable-cache overview, Go 1.26 / `CGO_ENABLED=0` stack, build/run snippet, `mise run check` gate,
  GHCR/deploy pointer, spec links; env table LINKED to `CLAUDE.md`). Review verified every factual claim
  and no dead relative links. **This was the last code/doc-closable "Done When" gate.**
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT
  EXISTS` only, no `PRAGMA user_version`) — the doc states the interim "recreate the volume on schema
  change" policy; `publish.yml`'s publish job has no ref guard (`workflow_dispatch` from a non-develop
  ref would move the floating `:develop` tag; immutable `:sha-<short>` unaffected). **Carried `low`:**
  the `.dockerignore` secret/sidecar globs are slashless (root-level only, not recursive).
- **The 3 iscc-infra `critical`s** (persistence / `/metrics` exposure / egress footprint) are answered
  in substance by `deploy/OPERATING.md` + the new `README.md`, but remain OPEN entries in `issues.md`.
  They must be confirmed closed (or pruned) before DONE; the Compose chown `normal` is the one residual
  doc gap on the persistence ask. **These are now the SOLE DONE blockers.**

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable. This window's change was `README.md` + context pack (no Go source); the
  review re-confirmed `mise run check` green (28 packages `ok`, cached) and `gofmt -l .` clean.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` (the `.codes` verifier site) + `publish.yml` (GHCR image push). **HEAD `ba111e8`
  is pushed (level with `origin/develop`): CI `success`, Pages `success`, Publish `in_progress`** (just
  triggered by the README push; no Dockerfile/workflow changed this window).
- **Latest `review` verdict: PASS / CONTINUE** (root README — scope-clean, one doc, zero Go, every claim
  verified, env-table linked not duplicated; Codex clean). The README `normal` is closed.
- **Open issues: 3 critical, 6 normal, 15 low** (the `## <short title>` in the issues.md header is the
  format template, not a real issue). The three `critical`s are the iscc-infra ops asks (DB-volume
  persistence contract; public-route `/metrics` exposure; egress/footprint sizing) — answered in
  substance by `deploy/OPERATING.md` + `README.md` but not yet confirmed closed/pruned. The `normal`s are
  the DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half gap, the
  per-hub-Anchor design-honesty question, the `publish.yml` ref-guard, and the Compose volume-prep chown
  mismatch. DONE requires 0 critical AND 0 normal, so the loop stays CONTINUE.

## Next Milestone
**M-Deploy is the gate, and every code/doc-closable item is now landed (README included). The immediate
next work is to confirm/prune the three iscc-infra `critical`s** — the now-booting `deploy/OPERATING.md`
+ root `README.md` answer the persistence contract, the `/metrics` exposure decision, and the
egress/footprint sizing in substance:
1. **Confirm/prune the three iscc-infra `critical`s.** Verify each ask is satisfied by the deployed docs
   and prune it. The persistence ask still has the Compose volume-prep chown `normal` caveat to settle
   first. If `define-next` judges any critical truly external (no doc/code closeable here), surface it as
   a STOP/IDLE edge rather than spinning on cosmetic chrome.

Fold-in candidates whenever the relevant file is next touched: fix the Compose volume-prep `chown`
(`OPERATING.md` — pin `name: monitor-data` or a Compose-native prep); the `publish.yml`/`pages.yml`
`workflow_dispatch` ref-guard (`if: github.ref == 'refs/heads/develop'`); the `pages.yml` Node-20 action
bumps.

Subsequent / parallel (all `normal`/`low`, none gate DONE once the three criticals land except where
noted): the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the design-first
pass on the WASM signature half (`normal`); the realm-index per-hub-vs-per-checkpoint Anchor honesty pass
(`normal`); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism (`normal`); the M-UI exit visual-pass + human sign-off (ADR-0012).
