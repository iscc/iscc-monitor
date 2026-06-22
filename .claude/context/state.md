<!-- assessed-at: b8638603ee4cbb4c3726ff85d01e781df552c7a4 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy build phase — feature milestones M1–M3/M-UI behaviorally complete, gate green. The order-independent **M-Deploy** packaging milestone (ADR-0013) is the active runway. Five of ~6 M-Deploy Verify items closed; the operability doc was attempted this window but the review found it does NOT boot, so the persistence `critical` it was meant to close stays open.

This window added a single tracked operator doc — `deploy/OPERATING.md` (219 lines, doc-only; no Go /
test / Dockerfile / workflow source touched). The `review` verdict is **NEEDS_WORK**: the doc's
substance is accurate and well-sourced, but its headline "copy-pasteable" quick-start **does not boot**
— both the Compose and `docker run` snippets omit the REQUIRED `ISCC_MONITOR_REALM` (claiming it
"defaults to the baked realm", which is false: the Dockerfile `COPY`s the realm FILE but sets no `ENV`,
and `config.Load` calls `required(get, keyRealm)`), and they mount a fresh `root:root` named volume that
the non-root uid 65532 cannot write. So the operability doc cannot yet close the three iscc-infra
`critical`s, and a new `critical` (fix the non-booting quick-start) was filed. DONE stays out of reach.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — the
    `89660150..HEAD` diff touched NO feature Go source (only `deploy/OPERATING.md` + context files;
    confirmed by `git diff --stat`).
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
  - **M-Deploy: 5 of ~6 Verify items met, milestone PARTIALLY MET. Operability-doc item NOT yet met.**
    The doc exists but its quick-start does not boot (review NEEDS_WORK), so it does not yet satisfy the
    "an operator can deploy correctly" bar nor close the persistence `critical`. Root README still
    absent.
- **Last ~10 iterations: ~6 milestone-Verify-or-gate-advancing (canonical realm doc; GHCR publish
  workflow; Dockerfile + container smoke; version stamp; SIGTERM trap; Pages publish unblock) / ~4
  chrome·config-leaf·cert-TZ·doc-fix-needed.** **No drift:** the loop is steadily walking the M-Deploy
  runway — code/doc-closable work that depends on no feature milestone. This window's operability doc
  landed substantively correct but with a non-booting quick-start the review caught; the immediate next
  step is the two-line correction, then the root README — both code-unblocked.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `89660150..HEAD` diff touched NO Go source under any M1 path
(only `deploy/OPERATING.md` + context). All Verify criteria remain satisfied: `origin`/`vkey` golden;
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
pending.** No template-render change this window (the only change was `deploy/OPERATING.md`). All six
certificate clauses + both anchor panels + badge + DS shell + `/` index + log browser + hub dossier +
frozen Exhibit + single-record page + ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint
render and pass the behavioral HTTP-seam Verify.
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
  `monitor.iscc.codes` serves the byte-pinned `/_ds/verify.wasm`. **Still open:** the cross-origin
  **SIGNATURE-half gap** (the verifier core does NO checkpoint-signature / did:web check — the success
  copy overstates a key check that never runs) — design-first remainder / STOP-candidate, `normal`.
  **Carried `low`:** `cmd/verifier-site` writes non-atomically; `pages.yml` actions target deprecated
  Node 20.
- **OTS:** the `.ots` serve route, the §5 anchor clause (digest-bound via `ots.ConfirmedFor`), the store
  layer, the off-path stamp/upgrade loop (`OTSTick`), the offline classifier, and the calendar transport
  (both `safeUpgrade` + `safeStamp` guards) are wired. The Verify-closer not yet built: a root reaching
  Bitcoin-confirmed — needs a live calendar + real BTC confirmation. Still 1/1 open. **Carried `low`
  defect:** nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only
  path).

## M-Deploy — Packaged & operable instance
**Status**: **PARTIALLY MET (5 of ~6 Verify items) — the front-of-queue code-closable work (ADR-0013).
The operability-doc item was ATTEMPTED this window but does not yet meet its bar (review NEEDS_WORK).**
Verified by exploration:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, draining the store via
  the deferred `st.Close()`. Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at `/version`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED.** Multi-stage build
  (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`, uid 65532, CA roots) producing a
  static stripped `cmd/iscc-monitor` binary, an empty-`VERSION` fail-fast `RUN` guard, and the baked
  realm. The `ci.yml` `docker` job builds it, runs the container, and polls `GET /healthz` → 200.
- **GHCR publish workflow — CLOSED.** `.github/workflows/publish.yml` triggers on push to `develop` +
  `workflow_dispatch`, logs into GHCR, and pushes BOTH `:develop` AND `:sha-<short>`.
- **Canonical realm document — CLOSED.** `deploy/realm-testnet.txt` at the fixed repo-root path, baked
  at `/etc/iscc-monitor/realm.txt` (`Dockerfile:46`), `registry.Parse`-accepted by the non-vacuous
  golden `TestParseCanonicalDeployRealm`. CLAUDE.md's env table names the three masthead identity keys.
- **Operability doc — ATTEMPTED, NOT YET MET (review NEEDS_WORK this window).** `deploy/OPERATING.md`
  (219 lines) exists and its substance is accurate and well-sourced (uid 65532, bind `:9464`/no
  host-publish, baked realm path, OTS calendar host, WAL siblings, single-writer, `GET /version` shape,
  SIGTERM drain, `:develop`+`:sha-<short>` tags, the `/metrics` exposure decision, the
  "recreate-volume-on-schema-change" interim migration policy). **BUT the quick-start does not boot:**
  - **[critical, NEW]** both snippets (`deploy/OPERATING.md:192`, `:214`) OMIT the REQUIRED
    `ISCC_MONITOR_REALM`, claiming (`:67`) it "defaults to the baked realm". Verified false:
    `config.Load` calls `required(get, keyRealm)` (`internal/config/config.go:123`) and the Dockerfile
    sets ZERO `ENV` (only `COPY deploy/realm-testnet.txt …`, `Dockerfile:46`) — the baked FILE is not a
    set VAR, so a copy-paste exits with `config: required key "ISCC_MONITOR_REALM" is missing`.
  - **[normal, NEW]** the snippets mount a fresh `root:root` named volume (`monitor-data:/data`) without
    a chown/init, so uid 65532 cannot create `/data/monitor.db` — contradicting the doc's own
    uid-65532-writable requirement two sections earlier (`:56-59`).
  Because the doc cannot be deployed correctly as written, it does **not** yet close the three iscc-infra
  `critical`s (persistence / exposure / egress).
- **No root `README.md`** — the "Done When" requirement is still open (`normal`). The only README is the
  loop-internal `.claude/context/README.md`; `CLAUDE.md` is agent-facing.
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT
  EXISTS` only, no `PRAGMA user_version`) — the doc states the interim "recreate the volume on schema
  change" policy; `publish.yml`'s publish job has no ref guard (`workflow_dispatch` from a non-develop
  ref would move the floating `:develop` tag; immutable `:sha-<short>` unaffected). **Carried `low`:**
  the `.dockerignore` secret/sidecar globs are slashless (root-level only, not recursive).

## Quality gates
**Status**: **GREEN** (carried — no code touched this window, so the last green gate stands; review will
re-confirm green on the doc-fix increment).
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable. This window's change was doc-only (`deploy/OPERATING.md`); the review
  confirmed `mise run check` green (28 packages `ok`) and `gofmt -l .` clean.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` (the `.codes` verifier site) + `publish.yml` (GHCR image push). Last fully-CI'd
  commit `89660150`: **CI `success`, Pages `success`, Publish `success`** (the Publish run that the prior
  state recorded as `in_progress` has now completed green). **HEAD `b8638603` is 4 commits AHEAD of
  `origin/develop` (unpushed); these 4 are all doc/context-only, so no CI has run on them yet** — push
  to re-gate.
- **Latest `review` verdict: NEEDS_WORK / CONTINUE** (`deploy/OPERATING.md` — quick-start does not boot:
  omits required `ISCC_MONITOR_REALM`; fresh volume not uid-65532-writable). The doc substance is
  correct; two precise corrections are needed, not a rewrite. Gate stayed green (doc-only).
- **Open issues: 4 critical, 7 normal, 15 low.** The four `critical`: (1) the NEW non-booting
  `OPERATING.md` quick-start (the active step's own deliverable); (2-4) the three iscc-infra ops asks
  (DB-volume persistence contract; public-route `/metrics` exposure; egress/footprint sizing) — answered
  in substance by the doc but not confirmed closed while the quick-start is non-booting. The `normal`s
  include the DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half
  gap, the per-hub-Anchor design-honesty question, the missing root README, the `publish.yml`
  ref-guard, and the new uid-65532 volume-prep note. DONE requires 0 critical AND 0 normal, so the loop
  stays CONTINUE.

## Next Milestone
**M-Deploy is the gate. The operability doc is the active step but its quick-start does not boot — fix
that FIRST (it is the smallest correct change and a NEW `critical`).** In priority order:
1. **Fix `deploy/OPERATING.md`'s quick-start so it boots.** Smallest correct change: set
   `ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` explicitly in BOTH the Compose and `docker run`
   snippets and correct the "valid out of the box" sentence (the FILE is baked; the VAR is not), OR add
   `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` to the Dockerfile (makes the "out of the box"
   claim true, but that is a Dockerfile change). Also add a one-line volume-prep note (pre-`chown
   65532:65532` / a 65532-writable bind mount) so the named-volume mount is writable. Once the doc boots
   correctly it closes the three iscc-infra `critical`s (persistence, exposure, egress).
2. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Fold-in candidate whenever a workflow file is next touched: the `publish.yml` (and `pages.yml`)
`workflow_dispatch` ref-guard (`if: github.ref == 'refs/heads/develop'`).

Subsequent / parallel (all `normal`/`low`, none gate DONE once the doc-fix + README land except where
noted): the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the design-first
pass on the WASM signature half (`normal`); the realm-index per-hub-vs-per-checkpoint Anchor honesty pass
(`normal`); the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism (`normal`); the M-UI exit visual-pass + human sign-off (ADR-0012).
