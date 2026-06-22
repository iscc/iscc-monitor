<!-- assessed-at: 24544bfe249704951e5b18de1f9e61386d43ec2d -->

# Project State

## Status: IN_PROGRESS

## Phase: M-Deploy opening — feature milestones M1–M3/M-UI behaviorally complete; the WASM "published" half just closed (Pages live); the gate is the new **M-Deploy** packaging milestone (6 critical ops issues, 0% built) plus a root README.

The feature surface is built and the loop has pivoted to packaging: target.md gained an
order-independent **M-Deploy** milestone (ADR-0013) and a root-README requirement in "Done When".
The WASM verifier artifact is now publicly served from `monitor.iscc.codes` (Pages deploy live), so
that milestone's "published value" half is finally closed — but **six new `critical` ops issues**
(Dockerfile/GHCR, SIGTERM, canonical realm doc, DB-volume contract, route-exposure decision, footprint
docs) and the missing README put DONE far out of reach.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).** Carried forward — no trust-path
    source touched since the last full pass.
  - **M-UI: behavioral + named-region Verify met; the mandatory M-UI exit visual-pass + human
    sign-off (ADR-0012) still pending.** The masthead-identity arc is partway: instance identity is
    config-driven on THREE of six SSR mastheads (`/`, dossier, certificate); the proofserve trio
    (`browser.html`, `records.html`, `record.html`) still renders the static placeholder.
  - **WASM verifier: published half now CLOSED** (live-verified: `https://monitor.iscc.codes/` → 200
    html, `/_ds/verify.wasm` → 200 `application/wasm`, 3,514,750 bytes, from a public commit). **Still
    open on the milestone Verify**: the cross-origin **signature half** (the WASM core verifies inclusion
    math + id-binding only — no checkpoint-signature / did:web resolution; a malicious monitor can still
    render green `verified`) — design-first remainder, `normal`. No dossier tier-2 WASM caller.
  - **OTS anchoring: 1/1 open (carried).** All observable HTTP halves closed + both calendar-transport
    guards in place; only a root actually transiting to **Bitcoin-confirmed** remains (offline-unprovable).
  - **M-Deploy: 0 of its Verify bar met (milestone NOT STARTED, verified by exploration).** No root
    `Dockerfile` (only `.devcontainer/Dockerfile`); no GHCR publish workflow (`.github/workflows/` =
    `ci.yml` + `pages.yml` only); `run()` traps **SIGINT only** (`main.go:133`
    `signal.NotifyContext(context.Background(), os.Interrupt)` — no `syscall.SIGTERM`); no version
    stamp (no `-ldflags`/`GET /version`/`/healthz` version field); no `deploy/` canonical realm doc; no
    operability/deployment doc.
  - **Root README: missing (blocks "Done When").** Repo root has no `README.md`.
- **Last ~10 iterations: ~1 milestone-Verify-advancing (the Pages publish unblock) / ~9
  chrome·plumbing·config-leaf.** The recent arc drained the code-closable masthead-identity slices
  (`/` → dossier → certificate mastheads → env-keys-into-config-leaf, this iteration). **DRIFT now
  resolving:** the standing M-Deploy milestone (newly ratified in target.md) is a large body of
  code-closable work that depends on no feature milestone, so the loop has a clear non-cosmetic runway
  again. The masthead polish streak is ending; M-Deploy is the front of the queue.

## M1 — Read-only Monitor
**Status**: **met** — carried forward. The `7a32458..HEAD` diff touched no signature / RFC-6962 /
Merkle / `proof` / `didweb` / `logclient` / `follower` / store source (production edits were only
`cmd/iscc-monitor/main.go` identity wiring + `internal/config` field add). All Verify criteria remain
satisfied: `origin`/`vkey` golden; fork/shrink/equivocation golden-tested with freeze + alert-once +
restart survival; structured logs; `/metrics`.
- **Packages present** (unchanged, 28 source pkgs): `cmd/{iscc-monitor,notecheck,verifier-site,wasm}`;
  internal — `badge, certificate, config, corsmw, dashboard, didweb, dossier, follower, healthz, index,
  logclient, metrics, metricshttp, ots, otsclient, proof, proofserve, registry, store, tiles, tilesserve,
  verifier, web`. Module `github.com/iscc/iscc-monitor`, `go 1.26.1` language directive; toolchain
  `mise.toml` `go = "1.26.4"`. 408 `func Test` across the tree.
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
pending.** This window's only M-UI-adjacent change is the config-leaf move of the masthead-identity env
keys (`ISCC_MONITOR_{INSTANCE,OPERATOR,REALM_NAME}` now parsed via `internal/config`'s no-validation
`optional` helper into typed `Config` fields; `identity(cfg)` builds `dashboard.Identity` from them) —
presentation-neutral plumbing, no template render change. All six certificate clauses + both anchor
panels + badge + DS shell + `/` index + log browser + hub dossier + frozen Exhibit + single-record page
+ ISCC-IDv1 decoder + Hub-List resolver + proof-bundle endpoint render and pass the behavioral
HTTP-seam Verify.
- **Still open (NOT critical, carried):** instance identity is config-driven on THREE of six SSR
  mastheads; the proofserve trio (`browser.html`, `records.html`, `record.html`) still renders the
  static `monitor instance` placeholder — the follow-on arc. The `←` back-link + named-region parity
  pass is on `/` + dossier but not yet on log browser / single record. The `/` "recent declarers
  checked" hero footer is omitted (needs store lookup history; `normal` #214 sub-4). The mandatory
  **M-UI exit visual-pass + human sign-off** (ADR-0012) is not executed.
- **Residual `normal` notes:** the realm-index Anchor cell is a per-HUB latest-stamped-root indicator
  not tied to the displayed Checkpoint (design-honesty question for the M-UI exit; spec-faithful,
  non-blocking). The config-move `normal` (#360) is now **CLOSED** by this iteration's advance.
- **`low` carried (const-dup):** the masthead fallback consts + `resolveIdentity` are duplicated across
  dashboard + dossier + certificate (3x, byte-identical); the proofserve slice would be the 4th copy,
  the natural trigger to consolidate into one shared exported `Resolve`. The stale `.chrome-identity` CSS
  comment in `dashboard.html:75-76` is still inaccurate.

## WASM verifier · OTS anchoring
**Status**: **WASM — published half now CLOSED (Pages live); signature half design-blocked. OTS —
observable halves + both transport guards landed; only a real Bitcoin confirmation remains
(offline-unprovable).** Neither core was touched this window.
- **WASM:** the id-binding half is closed in source + artifact, the artifact is reproducible from
  `mise run build:wasm` (`TestWasmVerifyHashPinned` green), and the artifact is now **publicly
  published**: the repo is public, Pages source = "GitHub Actions", `monitor.iscc.codes` custom domain
  bound, and the `github-pages` environment allows the `develop` branch — dispatched run `27965858770`
  went fully green and the live apex serves the byte-pinned `/_ds/verify.wasm` (200, 3,514,750 bytes).
  This closes the "verifier artifact hash matches the published value" half. **Still open:** the
  cross-origin **SIGNATURE-half gap** (the verifier core does NO checkpoint-signature / did:web check —
  the success copy overstates a key check that never runs) — design-first remainder / STOP-candidate,
  `normal`. No tier-2 WASM caller in the hub dossier. **Carried `low`:** `cmd/verifier-site` writes
  non-atomically; `pages.yml` actions target deprecated Node 20.
- **OTS:** the `.ots` serve route, the §5 anchor clause (digest-bound via `ots.ConfirmedFor`), the store
  layer, the off-path stamp/upgrade loop (`OTSTick`), the offline classifier, and the calendar transport
  (both `safeUpgrade` + `safeStamp` guards) are wired. The Verify-closer not yet built: a root reaching
  Bitcoin-confirmed — needs a live calendar + real BTC confirmation. Still 1/1 open. **Carried `low`
  defect:** nil-Stamper + empty-OTSBytes row falls through to the Upgrader (`otsloop.go:144`; test-only
  path).

## M-Deploy — Packaged & operable instance
**Status**: **NOT STARTED (0 of the Verify bar met) — newly ratified milestone (ADR-0013); now the
front-of-queue code-closable work.** Verified by exploration:
- **No production `Dockerfile`** — only `.devcontainer/Dockerfile`. The Verify bar wants a tracked
  multi-stage `CGO_ENABLED=0` non-root `scratch`/distroless image + a CI job that runs the container and
  asserts `GET /healthz` → 200.
- **No GHCR publish workflow** — `.github/workflows/` is `ci.yml` + `pages.yml` (the `.codes` verifier
  site) only; nothing pushes `ghcr.io/iscc/iscc-monitor` with `develop` + `sha-<short>` tags.
- **SIGTERM not trapped** — `main.go:133` is `signal.NotifyContext(context.Background(), os.Interrupt)`
  (SIGINT only); `docker stop`/orchestrators send SIGTERM, which would kill the process before the
  deferred `store.Close()` runs. This is one of the six `critical` ops issues.
- **No version stamp** — no `-ldflags` git-SHA injection and no `GET /version` / `/healthz` version
  field (grep-confirmed empty).
- **No canonical realm doc** — the only realm file is `internal/registry/testdata/realm.txt` (a
  testdata path); no `deploy/realm-testnet.txt`. CLAUDE.md's env table now lists the three masthead
  identity keys (verified — added this window), satisfying that sub-item.
- **No operability/deployment doc** — nothing states the volume path / backup unit / non-root uid /
  migration policy / egress endpoints / reverse-proxy contract / `/metrics` exposure decision.
- The **on-disk DB migration hazard** (`store.Open` = `CREATE TABLE IF NOT EXISTS` only, no
  `PRAGMA user_version`) remains an open `normal`, which M-Deploy's interim "recreate the volume on
  schema change" policy must document.

## Quality gates
**Status**: **GREEN in CI (UTC) / RED locally on non-UTC hosts.** CI `success` at `6cab58f`
(`origin/develop`); HEAD `24544bf` is a docs-only commit (ADR-0013 + target.md + issues.md), 1 ahead of
origin, no production Go change, so the CI-green verdict carries forward. Live Pages deploy verified.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.26.1`; toolchain `go = "1.26.4"`);
  `mise run check` runnable. **`mise run check` is RED on any non-UTC host** — confirmed this assessment:
  `TZ=America/New_York go test ./internal/certificate` FAILS (`TestCertificateComparisonAnchor` +
  `TestCertificateBitcoinAnchorConfirmed`) because `internal/certificate/handler.go` renders coverage +
  §5-confirmation timestamps via `.Format(time.RFC3339)` WITHOUT `.UTC()` (lines 1013, 1040), so a CET
  box renders `+01:00` where the test expects `Z`. Pre-existing, environmental (CI runners are UTC), and
  now filed as a `normal` issue. Every other surface renders UTC `Z`; this is the cert handler diverging.
- **CI**: `.github/workflows/ci.yml` runs `mise run check` + the `cmd/notecheck` oracle on push/PR.
  Latest CI run 27965347334 at `6cab58f` is `success`. Latest Pages run 27965858770 at `6cab58f` is
  `success` (live-verified).
- **Open issues: 6 critical, 7 normal, 9 low.** The six `critical` are the M-Deploy ops asks (GHCR
  image + push workflow, SIGTERM trap, canonical mountable realm doc, DB-volume persistence contract,
  public-route exposure decision, egress/footprint docs) — promoted to `critical` in an unstaged
  `issues.md` edit by the human/ops side (verified in the working tree). The seven `normal`: the
  DB-migration hazard, the `/` "recent declarers checked" footer, the WASM signature-half gap, the
  per-hub-Anchor design-honesty question, the **cert local-TZ timestamp bug**, the missing root README,
  and (the now-RESOLVED Pages-domain issue is prunable). DONE requires 0 critical AND 0 normal, so the
  loop stays CONTINUE.

## Next Milestone
**M-Deploy is the gate — six `critical` ops issues, none built. Start with the code-closable, in-repo
Verify items; the cert local-TZ fix should ride along first so subsequent local `mise run check` runs
aren't masked red.** In priority order:
1. **Fix the cert local-TZ timestamp bug** (`.UTC().Format(time.RFC3339)` on the coverage-since + §5
   confirmation chips in `internal/certificate/handler.go`) — the only thing keeping `mise run check`
   RED on a non-UTC host; a small, well-scoped gate-greening slice that matters before the M-Deploy
   CI/container work, since those jobs will run `mise run check`.
2. **Trap SIGTERM** — add `syscall.SIGTERM` to `signal.NotifyContext` in `cmd/iscc-monitor/main.go:133`
   with a test (send SIGTERM → context cancels → `store.Close()` runs → exit 0); reverting the
   registration must FAIL the test. Smallest critical, pure-code, high-leverage.
3. **Add the production `Dockerfile` + GHCR publish workflow** — multi-stage `CGO_ENABLED=0` non-root
   `scratch`/distroless image; a CI job that builds it, runs the container, asserts `GET /healthz` → 200;
   a workflow pushing `ghcr.io/iscc/iscc-monitor` on push to `develop` tagged `develop` + `sha-<short>`.
4. **Version-stamp the binary** (git SHA via `-ldflags`, default `dev`) and report it on `/healthz` JSON
   or `GET /version` (HTTP-seam test).
5. **Ship a canonical realm doc** under `deploy/` (e.g. `deploy/realm-testnet.txt`, `registry.Parse`-tested)
   and a tracked **deployment/operability doc** (volume path + backup unit + non-root uid + interim
   "recreate volume on schema change" migration policy + egress endpoints + reverse-proxy contract +
   `/metrics` exposure decision).
6. **Add the public-facing root `README.md`** ("Done When" requirement) — what it is (verifiable cache),
   the stack, a build/run snippet, spec pointers; link CLAUDE.md as the authoritative env source.

Subsequent / parallel: the proofserve-trio masthead slice + the shared `Resolve` leaf consolidation; the
design-first pass on the WASM signature half; the realm-index per-hub-vs-per-checkpoint Anchor honesty
pass; the OTS "upgrades to Bitcoin-confirmed" half (offline-unprovable); the on-disk DB migration
mechanism; the M-UI exit visual-pass + human sign-off (ADR-0012).
