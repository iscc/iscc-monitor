<!-- assessed-at: f8e95eadacce5d02e2935a5bb166bab8925fe87a -->

# Project State

## Status: IN_PROGRESS

## Phase: Post-M-Deploy convergence — every v1 feature milestone (M1–M3/M-UI) is behaviorally
complete and the M-Deploy in-repo Verify bar is met. **This window the last two `critical`s
(iscc-infra ops asks) were pruned**, so the open-`critical` count is now **0**. DONE turns solely on
the **5 remaining `normal`s** (DONE requires 0 critical AND 0 normal), most of which are
design-honesty / human-blocked questions, not code-closable edits — the loop is at the
human-blocked-DONE edge the memory warns about.

This window (`1d2eb90..f8e95ea`) was **context-only**: the diff touched ONLY `.claude/context/*`
(handoff, issues, next, state) — **zero Go source, zero `Dockerfile`/workflow/doc** (confirmed by
`git diff --stat`). The advance pruned the two answered iscc-infra `critical`s (route-exposure/`/metrics`;
egress + footprint) after confirming each clause maps to `deploy/OPERATING.md`; review verdict
**PASS_WITH_NOTES / CONTINUE** (one Codex note — the §Footprint disk-growth answer is qualitative, not a
measured rate — refiled `low`, not a re-block, because the number needs live testnet measurement the loop
cannot perform). HEAD `f8e95ea` is level with `origin/develop`; CI / Pages / Publish all `success` on it.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no Go source touched
    this window.
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
  - **M-Deploy: ALL in-repo Verify items CLOSED, and the two iscc-infra ops `critical`s are now pruned
    (confirmed answered by `deploy/OPERATING.md`).** No open Verify item remains for this milestone.
- **Last ~10 iterations: ~7 milestone-Verify-or-gate-advancing** (GHCR publish workflow; Dockerfile +
  container smoke; version stamp; SIGTERM trap; Pages publish unblock; realm-bake; root README; Compose
  volume-name pin closing the persistence critical) **/ ~3 context-prune·config-leaf·cert-TZ**. **Drift
  watch:** the M-Deploy runway is now fully walked — every code/doc-closable item has landed AND the
  iscc-infra criticals are pruned. The 5 remaining `normal`s are mostly NOT code-closable (DB-migration is
  a deliberate design decision; the WASM signature half + per-hub-Anchor + `/` declarers-footer are
  design/store-history questions). Only the `publish.yml`/`pages.yml` ref-guard `normal` is a small
  self-contained edit. **This is the human-blocked-DONE edge:** if `define-next` finds no code-closable
  `normal`, it should surface a STOP/IDLE decision (route the design `normal`s to a human/design pass)
  rather than spin on cosmetic chrome.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `1d2eb90..f8e95ea` diff touched NO Go source (context-only).
All Verify criteria remain satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested with
freeze + alert-once + restart survival; structured logs; `/metrics`.
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
  `monitor.iscc.codes` serves the byte-pinned `/_ds/verify.wasm` (Pages run on HEAD `f8e95ea` green).
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
**Status**: **ALL in-repo Verify items CLOSED; the two iscc-infra ops `critical`s are now PRUNED
(confirmed answered by `deploy/OPERATING.md`). No open Verify item remains.** Verified by exploration + CI:
- **SIGTERM trapped — CLOSED.** `notifyShutdown()` (`cmd/iscc-monitor/main.go`) registers
  `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`, draining the store via
  the deferred `st.Close()`. Mutation-proven by the `//go:build unix` `shutdown_test.go`.
- **Version-stamped binary — CLOSED.** `internal/version.Version` (`var Version = "dev"`) is the
  `-ldflags -X` injection target; `version.Handler()` is mounted at `/version`.
- **Production `Dockerfile` + container `/healthz` CI smoke — CLOSED.** Multi-stage build
  (`golang:1.26.4` → `gcr.io/distroless/static-debian12:nonroot`, uid 65532, CA roots), empty-`VERSION`
  fail-fast `RUN` guard, baked realm, AND `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt`. The
  `ci.yml` `docker` job builds it and runs the container with NO `-e ISCC_MONITOR_REALM` — so a
  realm-less `/healthz` → 200 mechanically proves the bake. CI run on HEAD `f8e95ea` green.
- **GHCR publish workflow — CLOSED.** `.github/workflows/publish.yml` triggers on push to `develop` +
  `workflow_dispatch`, logs into GHCR, and pushes BOTH `:develop` AND `:sha-<short>`. **Publish run on
  HEAD `f8e95ea` is `success`.**
- **Canonical realm document — CLOSED.** `deploy/realm-testnet.txt` at the fixed repo-root path, baked
  at `/etc/iscc-monitor/realm.txt` and ENV-pointed, `registry.Parse`-accepted by the non-vacuous golden
  `TestParseCanonicalDeployRealm`. CLAUDE.md's env table names the three masthead identity keys.
- **Operability doc — CLOSED.** `deploy/OPERATING.md` (12.6 KB) is accurate and well-sourced (uid 65532,
  bind `:9464`/no host-publish, baked realm path, OTS calendar host, WAL siblings, single-writer,
  `GET /version` shape, SIGTERM drain, `:develop`+`:sha-<short>` tags, the `/metrics` exposure decision,
  the recreate-volume migration policy, pinned Compose `name: monitor-data`). The §"Route exposure & the
  `/metrics` decision", §Egress, and §Footprint sections **answer both iscc-infra `critical`s clause-by-
  clause** — review confirmed each maps before pruning the entries.
- **Root `README.md` — CLOSED.** Public front door at repo root (verifiable-cache overview, Go 1.26 /
  `CGO_ENABLED=0` stack, build/run snippet, `mise run check` gate, GHCR/deploy pointer, spec links; env
  table LINKED to `CLAUDE.md`).
- **iscc-infra ops `critical`s — PRUNED this window.** The two pre-deploy asks (public-route `/metrics`
  exposure decision; egress + resource footprint sizing) were the prior sole DONE blockers; the advance
  confirmed each clause is answered by `deploy/OPERATING.md` and deleted the entries (review independently
  re-mapped each clause, then PASS). The Codex disk-growth-rate sub-note was refiled `low` (needs live
  testnet measurement; not loop-closeable). **No open M-Deploy `critical`/Verify item remains.**
- **Carried `normal` traps:** the on-disk DB migration hazard (`store.Open` = `CREATE TABLE IF NOT
  EXISTS` only, no `PRAGMA user_version`) — the doc states the interim "recreate the volume on schema
  change" policy; `publish.yml`'s publish job has no ref guard (`workflow_dispatch` from a non-develop
  ref would move the floating `:develop` tag; immutable `:sha-<short>` unaffected). **Carried `low`:**
  the `.dockerignore` secret/sidecar globs are slashless (root-level only, not recursive); the
  `OPERATING.md` §Footprint disk-growth answer is qualitative not a measured rate.

## Quality gates
**Status**: **GREEN.**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable (`go build ./... && go vet ./... && go test ./...`). This window's change was
  `.claude/context/*` only (no Go source); review re-confirmed `mise run check` green (all cached, 28
  packages `ok`) and `gofmt -l .` clean.
- **CI**: `.github/workflows/` is `ci.yml` (`mise run check` + `notecheck` oracle + docker `/healthz`
  smoke) + `pages.yml` (the `.codes` verifier site) + `publish.yml` (GHCR image push). **HEAD `f8e95ea`
  is pushed (level with `origin/develop`): CI `success`, Pages `success`, Publish `success`** (confirmed
  via `gh run list`).
- **Latest `review` verdict: PASS_WITH_NOTES / CONTINUE** (the iscc-infra-criticals prune — surgical,
  gate-green, each deleted Verify bar re-mapped to a concrete `OPERATING.md` line; one Codex note refiled
  `low`).
- **Known non-CI flake (off the gate):** a certificate masthead test asserts an RFC-3339 timestamp in
  local TZ; it fails on non-UTC dev hosts only (filed `low` at commit `6cab58f`). CI runs UTC and is green
  — does not block. (`review` owns the gate verdict; not re-run here.)
- **Open issues: 0 critical, 5 normal, 16 low.** The `## <short title>` template header in the issues.md
  format block is NOT a real issue (the anchored `^- **Priority:** critical$` grep returns 0). The five
  `normal`s are: the DB-migration hazard, the `/` "recent declarers checked" footer, the WASM
  signature-half gap, the per-hub-Anchor design-honesty question, and the `publish.yml` ref-guard. DONE
  requires 0 critical AND 0 normal, so the loop stays CONTINUE.

## Next Milestone
**M-Deploy is fully met (criticals pruned). DONE is now blocked only by the 5 open `normal`s** — and
most are NOT code-closable. The immediate work is to triage those 5 against the DONE bar:
1. **The one code-closable `normal`:** the `publish.yml`/`pages.yml` `workflow_dispatch` ref-guard
   (`if: github.ref == 'refs/heads/develop'`) — a small self-contained workflow edit; carry the
   `pages.yml` Node-20 action bumps (`low`) along when that file is touched.
2. **The DB-migration `normal`** is a deliberate first-migration *design* decision (ADR-0007) — wants a
   grilling/design pass, not a quick edit.
3. **The three design-honesty `normal`s** (WASM signature-half gap → browser did:web design pass; the
   per-hub-vs-per-checkpoint Anchor honesty question; the `/` "recent declarers" footer needing store
   lookup history) are human/design-blocked. Per the standing "loop stalls on human-blocked DONE"
   memory, if `define-next` finds no further code-closable `normal` after the ref-guard lands, it should
   **surface a STOP/IDLE decision** (route these to a human/design pass) rather than spin on cosmetic
   chrome.

Parallel `low`/deferred (none gate DONE): the proofserve-trio masthead slice + the shared `Resolve` leaf
consolidation; the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the M-UI exit
visual-pass + human sign-off (ADR-0012); the §Footprint measured disk-growth rate (needs live data).
