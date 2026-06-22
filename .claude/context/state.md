<!-- assessed-at: 0753b6547dafdb162d4b9e77bf7cb6210ed2f0b2 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete, gate green. The order-independent **M-Deploy** packaging milestone (ADR-0013) is the active runway. The deploy quick-start now BOOTS (realm-bake advance, CI-proven); the last `target.md` "Done When" gate — a public-facing root `README.md` — is still absent.

This window the realm-bake advance set `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` in the
Dockerfile, dropped the now-redundant `-e ISCC_MONITOR_REALM` from the CI `docker` smoke (so a
realm-less boot to `/healthz` 200 mechanically proves the bake), and corrected `deploy/OPERATING.md`'s
phantom-default wording + added a uid-65532 volume-prep note. Review verdict **PASS_WITH_NOTES /
CONTINUE**; the NEW non-booting-quickstart `critical` is closed. HEAD `0753b65` is pushed and all three
workflows (CI, Pages, Publish) are green.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `b8638603..HEAD` diff touched NO Go source (only `Dockerfile`, `ci.yml`, `deploy/OPERATING.md` +
    context/learnings; confirmed by `git diff --stat`).
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
  - **M-Deploy: Dockerfile + CI smoke + GHCR publish + SIGTERM + version + canonical realm + operability
    doc all CLOSED; only the root `README.md` "Done When" item remains.** The quick-start now boots
    (realm-bake, CI-proven) modulo the Compose volume-prefix chown caveat (`normal`).
- **Last ~10 iterations: ~7 milestone-Verify-or-gate-advancing (canonical realm doc; GHCR publish
  workflow; Dockerfile + container smoke; version stamp; SIGTERM trap; Pages publish unblock; realm-bake
  so the quick-start boots) / ~3 chrome·config-leaf·cert-TZ.** **No drift:** the loop is steadily walking
  the M-Deploy runway — code/doc-closable work that depends on no feature milestone. The remaining
  code-closable item is the root README, then the residual `normal`s.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `b8638603..HEAD` diff touched NO Go source under any M1 path
(only `Dockerfile`, `ci.yml`, `deploy/OPERATING.md` + context). All Verify criteria remain satisfied:
`origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze + alert-once + restart
survival; structured logs; `/metrics`.
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
  `monitor.iscc.codes` serves the byte-pinned `/_ds/verify.wasm` (Pages run on HEAD `0753b65` green).
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
**Status**: **NEARLY MET — every code/CI Verify item CLOSED and the operability doc now states a
correctly-booting deploy. The remaining "Done When" item is the root `README.md`.** Verified by
exploration + CI:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, draining the store via
  the deferred `st.Close()`. Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at `/version`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED + STRENGTHENED.** Multi-stage build
  (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`, uid 65532, CA roots), empty-`VERSION`
  fail-fast `RUN` guard, baked realm, AND now `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt`
  (`Dockerfile:51`). The `ci.yml` `docker` job builds it and runs the container with NO
  `-e ISCC_MONITOR_REALM` — so a realm-less `/healthz` → 200 mechanically proves the bake. CI run on HEAD
  `0753b65` green.
- **GHCR publish workflow — CLOSED.** `.github/workflows/publish.yml` triggers on push to `develop` +
  `workflow_dispatch`, logs into GHCR, and pushes BOTH `:develop` AND `:sha-<short>`. Publish run on HEAD
  `0753b65` green.
- **Canonical realm document — CLOSED.** `deploy/realm-testnet.txt` at the fixed repo-root path, baked
  at `/etc/iscc-monitor/realm.txt` (`Dockerfile:46`) and now ENV-pointed, `registry.Parse`-accepted by
  the non-vacuous golden `TestParseCanonicalDeployRealm`. CLAUDE.md's env table names the three masthead
  identity keys.
- **Operability doc — CLOSED (substance + booting quick-start).** `deploy/OPERATING.md` exists, its
  substance is accurate and well-sourced (uid 65532, bind `:9464`/no host-publish, baked realm path, OTS
  calendar host, WAL siblings, single-writer, `GET /version` shape, SIGTERM drain, `:develop`+
  `:sha-<short>` tags, the `/metrics` exposure decision, the "recreate-volume-on-schema-change" interim
  migration policy). The phantom-default wording is corrected — the doc now says the image **sets** the
  var via `ENV` — and a uid-65532 volume-prep note was added. **Residual `normal`:** the volume-prep
  `chown` literal-names `monitor-data`, but the headline Compose fragment declares the volume with no
  `name:`/`external:`, so `docker compose up` mounts a project-prefixed volume the chown never touched —
  the Compose path still fails permission-denied at `store.Open`. Doc-correctness gap, not a code defect;
  does not block progress.
- **No root `README.md`** — the "Done When" requirement is still open (`normal`). The only README is the
  loop-internal `.claude/context/README.md`; `CLAUDE.md` is agent-facing. **This is the last code/doc-
  closable "Done When" gate.**
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT
  EXISTS` only, no `PRAGMA user_version`) — the doc states the interim "recreate the volume on schema
  change" policy; `publish.yml`'s publish job has no ref guard (`workflow_dispatch` from a non-develop
  ref would move the floating `:develop` tag; immutable `:sha-<short>` unaffected). **Carried `low`:**
  the `.dockerignore` secret/sidecar globs are slashless (root-level only, not recursive).
- **The 3 iscc-infra `critical`s** (persistence / `/metrics` exposure / egress footprint) are answered
  in substance by `deploy/OPERATING.md`'s now-booting deploy, but remain OPEN entries in `issues.md`.
  They must be confirmed closed (or pruned) before DONE; the Compose chown `normal` is the one residual
  doc gap on the persistence ask.

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable. This window's change was Dockerfile + CI workflow + deploy doc (no Go
  source); the review confirmed `mise run check` green (28 packages `ok`) and `gofmt -l .` clean.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` (the `.codes` verifier site) + `publish.yml` (GHCR image push). **HEAD `0753b65`
  is pushed (level with `origin/develop`) and ALL THREE workflows are `success`: CI `success`, Pages
  `success`, Publish `success`.**
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (realm-bake — the quick-start now boots,
  CI-proven by the realm-less docker smoke). The NEW non-booting-quickstart `critical` is closed; one
  residual `normal` (Compose volume-prefix chown mismatch).
- **Open issues: 3 critical, 7 normal, 15 low.** The three `critical`s are the iscc-infra ops asks
  (DB-volume persistence contract; public-route `/metrics` exposure; egress/footprint sizing) — answered
  in substance by the now-booting operability doc but not yet confirmed closed/pruned. The `normal`s
  include the DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half
  gap, the per-hub-Anchor design-honesty question, the missing root README, the `publish.yml` ref-guard,
  and the Compose volume-prep chown mismatch. DONE requires 0 critical AND 0 normal, so the loop stays
  CONTINUE.

## Next Milestone
**M-Deploy is the gate. The quick-start now boots; the immediate next code/doc-closable work is the root
`README.md` (the last `target.md` "Done When" requirement) and closing/pruning the three iscc-infra
`critical`s.** In priority order:
1. **Confirm/prune the three iscc-infra `critical`s.** The now-booting `deploy/OPERATING.md` states the
   persistence contract (DB path + volume + uid-65532 + interim migration policy), the `/metrics`
   exposure decision, and the egress/footprint — verify each ask is satisfied and prune it (the
   persistence ask still has the Compose volume-prep chown `normal` caveat to settle first).
2. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Fold-in candidates whenever the relevant file is next touched: fix the Compose volume-prep `chown`
(`OPERATING.md` — pin `name: monitor-data` or a Compose-native prep); the `publish.yml`/`pages.yml`
`workflow_dispatch` ref-guard (`if: github.ref == 'refs/heads/develop'`); the `pages.yml` Node-20 action
bumps.

Subsequent / parallel (all `normal`/`low`, none gate DONE once the README + criticals land except where
noted): the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the design-first
pass on the WASM signature half (`normal`); the realm-index per-hub-vs-per-checkpoint Anchor honesty pass
(`normal`); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism (`normal`); the M-UI exit visual-pass + human sign-off (ADR-0012).
