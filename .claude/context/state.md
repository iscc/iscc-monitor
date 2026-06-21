<!-- assessed-at: 3fc5d96fbe557698f4d0f611451fe96fa5eb8160 -->

# Project State

## Status: IN_PROGRESS

## Phase: M-UI (Evidence Ledger frontend, ADR-0010) in progress — the first wired seam landed.
The `HubStatusBadge` partial is now rendered through the `/` dashboard (icon + label + silhouette) for
the three store-provable statuses, but no full M-UI Verify criterion is met yet (DS tokens/fonts, the
five-status taxonomy, dossier, record list, and certificate are all still open). M1/M2/M3 remain fully
met. Remaining v1 work: the rest of M-UI, then the WASM verifier and OTS anchoring.

The monitor follows + mirrors + verifies hubs (M1 met), serves all three computed proofs per hub from
the local mirror plus fsck root-rebuild (M2 met), and the full M3 Trust-API surface is functional
(CORS on every GET, verify-for-me verdict, `/` hub-list dashboard, per-hub HTML log browser). M-UI is
in flight: the WASM-shareable `HubStatusBadge` leaf now renders on `/` for verified/frozen/inactive.

## Convergence
- **Remaining Verify criteria:**
  - **M1: 0 open (met). M2: 0 open (met). M3: 0 open (met, 4/4).**
  - **M-UI (Evidence Ledger frontend): ~9 still open** (in progress, no criterion closed). Landed so
    far: the `HubStatusBadge` leaf (`internal/badge`, all five statuses with distinct label +
    inline-SVG silhouette, golden + mutation + fail-closed proven) AND its **first wiring** — the `/`
    dashboard status cell now renders `{{template "hubStatusBadge" .}}` backed by `badge.Label`
    (golden + mutation-proven). But only 3/5 statuses (verified/frozen/inactive) appear because the
    full five-status taxonomy is not yet store-provable, so the per-status HTTP-seam Verify criterion
    (all five rendered honestly) is NOT met. Still entirely open: DS v2 tokens + self-hosted fonts
    embedded (`go:embed`); the full five-status taxonomy made store-provable
    (`unresolvable`/`unverified` are in-memory only, not threaded through `dashboard`/`proofserve`);
    hub dossier (+ frozen Exhibit); paginated record list + single-record page; certificate-of-
    inclusion + **downloadable proof-bundle assembler** (re-engages the oracle/conformance gate;
    `serveVerify` discards the raw checkpoint bytes + resolved hub key the bundle needs); separate
    Bitcoin-anchor vs comparison-anchor panels.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`, re-verified).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported, re-verified).
- **Last ~10 iterations: ~5 milestone-Verify / ~5 refactor·polish·test·infra.** Healthy and on the
  Verify bar. The recent arc closed all four M3 criteria back-to-back (verify-for-me → `/` dashboard →
  log browser), then opened M-UI leaf-first (`HubStatusBadge`) and immediately wired it into `/`. **No
  polish-streak drift** — the badge did not sit unwired. Watch-item: M-UI is the largest remaining
  slice; the next iterations must keep converting leaves into wired HTTP-seam Verify criteria (the
  five-status store-provability that unlocks the per-status golden) rather than accumulate unwired
  primitives.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change since the last assessment (the
`b418270..HEAD` diff touched only `internal/dashboard/*`, `internal/badge/*` (one new `Label`
accessor + test), and context/spec files). All Verify criteria remain satisfied: `origin`/`vkey`
golden, all three triggers (fork/shrink/equivocation) golden-tested end-to-end with freeze +
alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over HTTP.
**CI-gated & green.**

- **Test totals at HEAD**: **227 `func Test`** across `cmd/` + `internal/`, **52** `_test.go` files
  (up from 226/52 — the +1 test is `badge.Label`'s coverage in `internal/badge/badge_test.go`).
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **15 internal packages** —
  `badge, config, corsmw, dashboard, didweb, follower, healthz, logclient, metrics, metricshttp,
  proofserve, registry, store, tiles, tilesserve`. Module `github.com/iscc/iscc-monitor`, `go 1.24.0`
  (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation, evaluated inside
  `logclient.CheckConsistency`; growing-pair equivocation builds the RFC-6962 consistency proof from
  the LOCAL mirror → freeze on `true` (ADR-0006). Frozen hubs are evidence-only on clean re-polls.
- **`AcceptCheckpoint` 4-way verdict** (`logclient/accept.go`) returns `(Status, CheckpointInfo,
  VerifiedContext{VKey,Key}, error)`, context populated only on `StatusVerified`; `PollHub` threads
  it into the hub-key cache upsert + `fsckMirror`. Reuse is per-poll only (ADR-0009).
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real
  sb0 checkpoint.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL,
  `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`.
  `store.ListHubs` is a pure read LEFT JOINing `hubs` with `follow_state`, returning plain Go types so
  store stays a leaf (no `net/http`/`logclient`).
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; `AlertFunc func(int64,string)` seam unchanged. The warm-path's second `did.json`
  resolve is a larger design change, not on the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**.
  All proof/dashboard/browser/badge tests run against in-process fixtures. Stale `sb1.amlet.id`
  did.json drift (pre-rotation key) captured in tests; not refreshed.
- **Reuse imports wired** (carried forward): `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion`+`proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `fsck` via
  `fsckMirror`, `client` re-export), `transparency-dev/formats` (`cmd/notecheck`). **Not wired:**
  `nbd-wtf/opentimestamps` (re-verified: no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — carried forward; no production change. Both Verify criteria are exercised (fsck
root-rebuild WIRED on every verified non-frozen poll via `fsckMirror` → `logclient.RunFsck` over the
read-only `store.SQLiteFetcher`; inclusion cross-check conformance-tested over the real verified
mirror in `internal/follower/inclusion_test.go`). The served proof surface is complete: all three
computed proofs — `inclusion`, `consistency`, `entries` — served from the local mirror, never
re-hitting the hub. Mirror write API single-source on `p` (`RecordTile`/`RecordEntryBundle`, store the
only `p→width` authority via `widthForP`). **Nothing remains on the M2 Verify bar.**

## M3 — Trust API + dashboard
**Status**: **met (4/4 Verify criteria)** — carried forward.
- **CORS** — `Access-Control-Allow-Origin: *` on every public GET via the single `corsmw.Handler` wrap.
- **verify-for-me** — `GET /<domain>/log/verify?iscc_id=<id>` returns store-provable `hub_status`,
  accepted `(size, root)`, and a REAL RFC-6962 inclusion result recomputed from the mirror and
  Merkle-verified against the accepted root. Every id-shaped fault → 200 non-verified. Golden +
  mutation non-vacuous.
- **`GET /` dashboard** — `200 text/html` listing **every** realm hub with its store-provable status +
  ADR-0001 coverage window (`internal/dashboard` + `store.ListHubs`). The status cell now renders
  through the `hubStatusBadge` partial (icon + label + silhouette). Golden + mutation-tested.
- **`GET /<domain>/log/` log browser** — `200 text/html` exposing accepted checkpoint `(size, root)` +
  relative links into `entries`/`inclusion`/`consistency`/`verify`/`checkpoint`. `POST /` → 405;
  unpolled hub → 200 "no accepted checkpoint yet". Golden + mutation + e2e-proven.

**Known limitations (carried forward, off the M3 Verify bar — these become M-UI work):**
- `proofserve.hubStatus` / `dashboard.hubStatus` resolve only the store-provable subset
  (`frozen`/`verified`, plus `inactive` in dashboard); `unverified`/`unresolvable`/`rotated` are not
  threaded through. Making the full five-status taxonomy store-provable is on the M-UI Verify bar
  (per the review handoff, the thread-through belongs at the **registry boundary**, not in
  `store.ListHubs` or `dashboard.hubStatus`).
- `inactive` is currently unreachable through the public store API (no `SetActive`/deactivation
  writer), so it is covered by a white-box `hubStatus` table test, not the HTTP-seam golden.
- No ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces or on `/`/`/verify` —
  not a Verify criterion.

## M-UI — Evidence Ledger frontend
**Status**: **in progress — first wired seam landed, no Verify criterion closed.**
- **Landed (`internal/badge` + `internal/dashboard`, review PASS `2fefcf5`):** a pure, stdlib-only,
  WASM-shareable `HubStatusBadge` partial (`Render(w, status)`, `Label(status) (string, bool)`,
  embedded `badge.html`, `PartialName = "hubStatusBadge"`, `Source` for association). Renders all five
  statuses each with a distinct text label (fixed `labels` table, caller status never trusted) AND a
  distinct inline-SVG silhouette (check/question/triangle/octagon-x/pause). Fails closed on
  unknown/empty status. The badge is now **wired into `/`**: `dashboard.html` renders
  `{{template "hubStatusBadge" .}}` per row; `buildRows` precomputes `.Label` via `badge.Label`; the
  page golden test asserts `class="hub-status-badge"` plus per-status SVG markers, mutation-proven
  (reverting to `{{.Status}}` fails). Leaf purity confirmed (`internal/store` and `internal/badge`
  have no `net/http`); `GOOS=js GOARCH=wasm build` OK.
- **Honest scope on `/`:** only 3/5 statuses (verified/frozen/inactive) can appear, because the full
  five-status taxonomy is not yet store-provable — so the per-status M-UI golden Verify criterion is
  NOT met.
- **Still open on the M-UI Verify bar:** DS v2 tokens + self-hosted Readex Pro / JetBrains Mono fonts
  embedded (`go:embed`, no CDN); the full **five-status taxonomy made store-provable** (precondition
  for honest five-badge rendering + the per-status golden); badge wired into the log browser /
  dossier / certificate; hub dossier (+ frozen Exhibit); paginated record list + single-record page
  (declaration/deletion/unknown-schema); certificate of inclusion + **downloadable proof-bundle
  assembler** (`{checkpoint, inclusion/consistency proof, record bytes, hub key, ots?}` —
  re-engages the oracle/conformance gate; `serveVerify` discards the raw checkpoint bytes + resolved
  hub key the bundle needs); separate Bitcoin-anchor vs comparison-anchor panels.
- Build source of truth: `.claude/design/ISCC Monitor - Developer Handoff.dc.html` + the `_ds/` token
  bundle (subordinate to ADR/PRD).

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM build
target (`syscall/js` not in source — grep clean). The `internal/badge` leaf is a WASM-shareable
primitive (`GOOS=js GOARCH=wasm` builds) that the verifier app will reuse, but the verifier itself
does not exist.

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** at `2fefcf5` (run 27909102927). Local HEAD is `3fc5d96`, one **docs-only**
  commit (M-UI target definition) ahead of `origin/develop` — not yet pushed, touches no Go.
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**, for wiring `HubStatusBadge` into `/`)
  records `mise run check` green (all 17 packages `ok`, `go vet`/`gofmt -l .` clean), leaf purity
  confirmed, WASM build OK, golden + mutation-proven, no new deps / schema change, gate-integrity scan
  clean, oracle gate correctly N/A (pure HTML composition). Codex second opinion clean, no findings.
- **No open `critical` or `normal` issue.** **Open `low` issue (loop-skipped, not a DONE blocker):** 1
  in `issues.md` — `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2/M3 all met. The next v1 milestone is M-UI (Evidence Ledger frontend, ADR-0010), now in
progress.** CI green, no `critical`/`normal` open, so feature work proceeds.

Convergence-driven order (turn the wired badge into a closed per-status Verify criterion):
1. **Make the full five-status taxonomy store-provable** (`verified`/`unresolvable`/`unverified`/
   `frozen`/`inactive`) by threading the in-memory `metrics.Registry` statuses through at the
   **registry boundary** (NOT `store.ListHubs` or `dashboard.hubStatus` — the store cannot prove
   `unresolvable`/`unverified`; they live in memory; `follower.glossaryStatus` folds `rotated →
   unverified`). Then add an end-to-end render assertion for at least one in-memory-only status and
   extend the golden markers — this closes the per-status M-UI Verify criterion (all five rendered
   honestly across `/`).
2. **DS v2 tokens + self-hosted fonts embedded** (`go:embed`, no CDN), then **hub dossier** (frozen
   Exhibit), **paginated record list + single record**, and the **certificate + downloadable
   proof-bundle assembler** (the slice that re-engages the oracle/conformance gate).
3. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
4. **Off the Verify bar:** sb1 fixture refresh (stale did.json key), real alert transport, and an
   end-to-end `inactive`-render assertion once a registry-deactivation writer lands.
