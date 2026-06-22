<!-- assessed-at: 1d2eb9093c14977ff56e938deb5a199cc0f6eccb -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete, gate green, every code/doc-closable `target.md` "Done When" gate (root `README.md` included) CLOSED. The order-independent **M-Deploy** packaging milestone (ADR-0013) is the active runway, and this window CLOSED the persistence `critical`. DONE now turns on confirming/pruning the **two remaining** iscc-infra ops `critical`s (route-exposure / `/metrics`; egress + footprint), both answered in substance by `deploy/OPERATING.md`.

This window (`ba111e8..1d2eb90`) the advance pinned the Compose volume name in `deploy/OPERATING.md`
(`name: monitor-data`) so the documented `chown` volume-prep operates on the SAME engine volume
`docker compose up` mounts. The diff touched **NO Go source** — only `deploy/OPERATING.md` (8 lines)
+ the context pack. Review verdict **PASS / CONTINUE**; this closed both the persistence-contract
`critical` (Verify bar now met end-to-end) and its Compose-chown `normal`. HEAD `1d2eb90` is level
with `origin/develop`; CI / Pages / Publish all green on it.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `ba111e8..1d2eb90` diff touched NO Go source (only `deploy/OPERATING.md` + context pack; confirmed
    by `git diff --stat`).
  - **M-UI: behavioral + named-region Verify met; the mandatory M-UI exit visual-pass + human sign-off
    (ADR-0012) still pending.** Instance identity config-driven on THREE of six SSR mastheads
    (`/`, dossier, certificate); the proofserve trio still renders the static placeholder. No template
    touched this window.
  - **WASM verifier: published half CLOSED** (`monitor.iscc.codes/_ds/verify.wasm` → 200
    `application/wasm`, byte-pinned). **Still open:** the cross-origin **signature half** (the WASM core
    verifies inclusion math + id-binding only — no checkpoint-signature / did:web resolution),
    design-first remainder, `normal`.
  - **OTS anchoring: 1/1 open (carried).** All observable HTTP halves + both calendar-transport guards
    in place; only a root actually transiting to **Bitcoin-confirmed** remains (offline-unprovable).
  - **M-Deploy: ALL in-repo Verify items CLOSED.** The persistence `critical` closed this window (the
    Compose volume-name pin was its last doc caveat). The residual is confirming/pruning the **two**
    remaining iscc-infra ops `critical`s, both answered in substance by `deploy/OPERATING.md`.
- **Last ~10 iterations: ~7 milestone-Verify-or-gate-advancing (GHCR publish workflow; Dockerfile +
  container smoke; version stamp; SIGTERM trap; Pages publish unblock; realm-bake; root README; Compose
  volume-name pin closing the persistence critical) / ~3 chrome·config-leaf·cert-TZ.** **No drift:** the
  loop has walked the M-Deploy runway to its end — every code/doc-closable item has landed and the
  persistence critical is now closed. The remaining DONE blockers are the two iscc-infra ops `critical`s,
  doc-confirmation/pruning work this repo owns. If `define-next` judges either truly external with
  nothing left to close in-repo, that is a STOP/IDLE edge the loop should surface rather than spin on
  cosmetic chrome.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `ba111e8..1d2eb90` diff touched NO Go source under any M1
path. All Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation
golden-tested with freeze + alert-once + restart survival; structured logs; `/metrics`.
- **Packages present** (29 source pkgs incl. `internal/version`): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`;
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
  `monitor.iscc.codes` serves the byte-pinned `/_ds/verify.wasm` (Pages run on HEAD `1d2eb90` green).
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
**Status**: **ALL in-repo Verify items CLOSED; the persistence `critical` closed this window. DONE now
turns solely on confirming/pruning the TWO remaining iscc-infra ops `critical`s.** Verified by
exploration + CI:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, draining the store via
  the deferred `st.Close()`. Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at `/version`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED.** Multi-stage build
  (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`, uid 65532, CA roots), empty-`VERSION`
  fail-fast `RUN` guard, baked realm, AND `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt`. The
  `ci.yml` `docker` job builds it and runs the container with NO `-e ISCC_MONITOR_REALM` — so a
  realm-less `/healthz` → 200 mechanically proves the bake. CI run on HEAD `1d2eb90` green.
- **GHCR publish workflow — CLOSED.** `.github/workflows/publish.yml` triggers on push to `develop` +
  `workflow_dispatch`, logs into GHCR, and pushes BOTH `:develop` AND `:sha-<short>`. **Publish run on
  HEAD `1d2eb90` is `success`** (the prior in-progress run from the README push completed green).
- **Canonical realm document — CLOSED.** `deploy/realm-testnet.txt` at the fixed repo-root path, baked
  at `/etc/iscc-monitor/realm.txt` and ENV-pointed, `registry.Parse`-accepted by the non-vacuous golden
  `TestParseCanonicalDeployRealm`. CLAUDE.md's env table names the three masthead identity keys.
- **Operability doc — CLOSED (substance + booting quick-start + persistence contract).**
  `deploy/OPERATING.md` is accurate and well-sourced (uid 65532, bind `:9464`/no host-publish, baked
  realm path, OTS calendar host, WAL siblings, single-writer, `GET /version` shape, SIGTERM drain,
  `:develop`+`:sha-<short>` tags, the `/metrics` exposure decision, the recreate-volume migration
  policy). **This window** the Compose `volumes:` block was pinned `name: monitor-data` so the
  documented `chown` prep targets the same volume `docker compose up` mounts — closing the last
  persistence doc caveat. The §10 sections cover Route exposure / `/metrics`, Egress, and Footprint.
- **Root `README.md` — CLOSED.** The public-facing front door exists at repo root (verifiable-cache
  overview, Go 1.26 / `CGO_ENABLED=0` stack, build/run snippet, `mise run check` gate, GHCR/deploy
  pointer, spec links; env table LINKED to `CLAUDE.md`). The last code/doc-closable "Done When" gate.
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT
  EXISTS` only, no `PRAGMA user_version`) — the doc states the interim "recreate the volume on schema
  change" policy; `publish.yml`'s publish job has no ref guard (`workflow_dispatch` from a non-develop
  ref would move the floating `:develop` tag; immutable `:sha-<short>` unaffected). **Carried `low`:**
  the `.dockerignore` secret/sidecar globs are slashless (root-level only, not recursive).
- **The 2 remaining iscc-infra `critical`s** (public-route `/metrics` exposure decision; egress +
  resource footprint sizing) are answered in substance by `deploy/OPERATING.md` (§"Route exposure & the
  `/metrics` decision", §Egress, §Footprint) but remain OPEN entries in `issues.md`. They must be
  confirmed closed (or pruned) before DONE. **These are now the SOLE DONE blockers.**

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable (`go build ./... && go vet ./... && go test ./...`). This window's change was
  `deploy/OPERATING.md` + context pack (no Go source); the review re-confirmed `mise run check` green
  (all cached) and `gofmt -l .` clean.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` (the `.codes` verifier site) + `publish.yml` (GHCR image push). **HEAD `1d2eb90`
  is pushed (level with `origin/develop`): CI `success`, Pages `success`, Publish `success`.**
- **Latest `review` verdict: PASS / CONTINUE** (Compose volume-name pin — scope-clean one-doc fix, gates
  green, persistence Verify bar met end-to-end, Codex clean).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; it fails on non-UTC dev hosts only (filed as a pre-existing issue at commit `6cab58f`). CI
  runs UTC and is green — does not block. (`review` owns the gate verdict; not re-run here.)
- **Open issues: 2 critical, 5 normal, 15 low** (the `## <short title>` template header in the issues.md
  format block is NOT a real issue; a naive grep over-counts it as a 3rd critical). The two `critical`s
  are the iscc-infra ops asks — public-route `/metrics` exposure decision; egress/footprint sizing —
  answered in substance by `deploy/OPERATING.md` but not yet confirmed closed/pruned. The `normal`s are
  the DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half gap, the
  per-hub-Anchor design-honesty question, and the `publish.yml` ref-guard. DONE requires 0 critical AND
  0 normal, so the loop stays CONTINUE.

## Next Milestone
**M-Deploy is the gate, and every code/doc-closable item is now landed (the persistence critical closed
this window). The immediate next work is to confirm/prune the two remaining iscc-infra `critical`s** —
`deploy/OPERATING.md` answers the `/metrics` exposure decision (§"Route exposure & the `/metrics`
decision") and the egress/footprint sizing (§Egress + §Footprint) in substance:
1. **Confirm/prune the two iscc-infra `critical`s.** Verify each ask is satisfied by the deployed docs
   and prune it. If `define-next` judges either truly external (no doc/code closeable here), surface it
   as a STOP/IDLE edge rather than spinning on cosmetic chrome.

Fold-in candidates whenever the relevant file is next touched: the `publish.yml`/`pages.yml`
`workflow_dispatch` ref-guard (`if: github.ref == 'refs/heads/develop'`); the `pages.yml` Node-20
action bumps.

Subsequent / parallel (all `normal`/`low`, none gate DONE once the two criticals land except where
noted): the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the design-first
pass on the WASM signature half (`normal`); the realm-index per-hub-vs-per-checkpoint Anchor honesty pass
(`normal`); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism (`normal`); the M-UI exit visual-pass + human sign-off (ADR-0012).
