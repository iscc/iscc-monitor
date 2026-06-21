<!-- assessed-at: c940248189b718b9692e7e4fccaff51aea96d84d -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — tlog-tiles mirror HTTP arc + all three computed proofs +
verify-for-me JSON verdict + server-rendered `GET /` dashboard complete; the HTML log browser, the
WASM verifier, and OTS anchoring remain.

The monitor follows + mirrors + verifies hubs (M1 met), serves all three computed proofs per hub from
the local mirror (M2 met), answers the verify-for-me verdict route (`GET /<domain>/log/verify`), and
now renders the `GET /` hub-list dashboard. M3 is at 3/4 Verify criteria; the one remaining M3 surface
(the `GET /<domain>/log/` HTML log browser), the WASM verifier, and OTS anchoring are the last v1 work.

## Convergence
- **Remaining Verify criteria:**
  - **M3: 1/4 open** — closed: `Access-Control-Allow-Origin: *` on every GET; the verify-for-me JSON
    verdict (`GET /<domain>/log/verify?iscc_id=<id>`, hub status + checkpoint `(size, root)` +
    Merkle-verified inclusion, id-faults → 200 non-verified); and now **`GET /` HTML dashboard** —
    `200 text/html` listing every realm hub with its glossary status + coverage window, golden +
    mutation-tested at the HTTP seam (`internal/dashboard`, review PASS at `c940248`). Still open:
    `GET /<domain>/log/` HTML log browser.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported).
  - M1: 0 open (met). M2: 0 open (met).
- **Last ~10 iterations: ~3 milestone-Verify / ~7 refactor·polish·test·infra.** Convergence is on
  track: the verify-for-me slice (`562a4d8`/`b537abe`/`8f0aa9d`) closed M3's second criterion, and the
  assessed range (`706eb1e..c940248`) closed the third — `b553734` (define-next) → `9638abf` (advance:
  `internal/dashboard` + `store.ListHubs` + `/` mount) → `c940248` (review PASS). No polish-streak
  drift: the loop is attacking M3 Verify criteria directly. The next criterion (log browser) is the
  same HTML arc and clearly reachable.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change in the assessed range
(`706eb1e..c940248` touched only `internal/dashboard`, `internal/store/hubs.go`,
`cmd/iscc-monitor/main.go` mount, and loop context). All Verify criteria remain satisfied:
`origin`/`vkey` golden, all three triggers (fork/shrink/equivocation) golden-tested end-to-end with
freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over
HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **219 `func Test`** across `cmd/` + `internal/`, **50** `_test.go` files
  (up from 215/49 — the +4 tests / +1 file are `internal/dashboard/handler_test.go`).
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **14 internal packages** —
  `config, corsmw, dashboard, didweb, follower, healthz, logclient, metrics, metricshttp,
  proofserve, registry, store, tiles, tilesserve` (added `dashboard`). Module
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
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
  The new `store.ListHubs` (`internal/store/hubs.go`) is a pure read LEFT JOINing `hubs` with
  `follow_state`, returning plain Go types so store stays a leaf (no `net/http`/`logclient`).
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; `AlertFunc func(int64,string)` seam unchanged. The warm-path's second `did.json`
  resolve is a larger design change, not on the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**
  (re-verified by `ls`). Tile/fsck/inclusion/consistency/entries/verify/dashboard tests run against
  in-process fixtures. Stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests; not
  refreshed.
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
re-hitting the hub. The mirror write API is single-source on `p` — `RecordTile(…, p uint8, …)` /
`RecordEntryBundle(…, p uint8, …)` with the store the only `p→width` authority (`widthForP` at
`internal/store/fetcher.go`). **Nothing remains on the M2 Verify bar.**

## M3 — Trust API + dashboard
**Status**: **in progress** (3/4 Verify criteria met). The raw tlog-tiles mirror is fully plumbed:
CORS on every public GET via the single `corsmw.Handler` wrap (closes the
`Access-Control-Allow-Origin: *` criterion); per-route `Cache-Control`; a strong content ETag +
`If-None-Match`→`304` conditional GET on the static-mirror 200s.

**Verify-for-me landed** (`internal/proofserve/handler.go`, `serveVerify` + `VerifyVerdict`, mounted
at `/verify` in `cmd/iscc-monitor/main.go:211`): `GET /<domain>/log/verify?iscc_id=<id>` returns the
store-provable `hub_status`, the accepted checkpoint `(size, root)`, and a REAL RFC-6962 inclusion
result recomputed from the local mirror and Merkle-verified against the accepted root
(`proof.VerifyInclusion`). Every id-shaped fault is a 200 non-verified verdict; non-200 is reserved
for genuine infra faults. Golden-tested across the 256-leaf boundary; reviewer-mutation-proven
non-vacuous through the HTTP seam.

**Dashboard landed** (`internal/dashboard/handler.go` + `dashboard.html` + `store.ListHubs`, mounted
at `/` in `cmd/iscc-monitor/main.go:157`): `GET /` returns `200 text/html; charset=utf-8` listing
**every** followed realm hub with its store-provable glossary status (`inactive` > `frozen` >
`verified`, mirroring `proofserve.hubStatus`) and ADR-0001 coverage window (`monitored_since` size +
RFC-3339 time, or "no coverage yet" — never implying a pre-coverage guarantee). Rendered into a buffer
then copied so a client never sees a half-rendered 200; `POST /` → 405, `GET /unknown` → 404. Golden +
mutation-tested at the HTTP seam on a fixture store (review PASS, 3 reverted mutations). **Closes the
third M3 Verify criterion.**

**Still absent (re-verified by grep at this assessment):**
- Log browser `GET /<domain>/log/` HTML — absent. No `html/template`/`text/template` import anywhere
  outside `internal/dashboard` (grep clean). It must mount under the per-hub subtree (via `hubHandler`,
  a different mount than `/`) — proofserve or a new per-hub HTML leaf, not `internal/dashboard`.
  **M3 Verify open.**
- ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces and on `/`/`/verify` — none;
  not a Verify criterion.

**Known limitation (not a defect, from review handoff):** `inactive` status is currently unreachable
through the public store API (no `SetActive`/registry-deactivation writer; `UpsertHub` inserts schema
default `active=1`), so it is covered by a white-box `hubStatus` table test, not the HTTP-seam golden.
Add an end-to-end inactive-render assertion when a deactivation writer lands. The richer in-memory
statuses (`unverified`/`unresolvable`/`rotated`) are deliberately not threaded into the dashboard.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits
in `cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM
build target (`syscall/js` not in source — grep clean). The WASM-shared verifier seam continues to
ride on `internal/didweb`; `serveVerify` already composes most of the proof-bundle pieces the
in-browser verifier will share (it discards the raw checkpoint bytes and the resolved hub key, which
the bundle path will need).

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** at HEAD `c940248` (run 27907753513) — the current HEAD, dashboard included.
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**, for the `GET /` dashboard) records `mise
  run check` green (16 packages `ok`, `go vet`/`gofmt -l .` clean), store kept a leaf
  (`go list -deps ./internal/store` has no `net/http`/`internal/dashboard`), HTTP-seam golden + 3
  reverted mutations proving the page non-vacuous, `GOOS=js GOARCH=wasm go build ./internal/didweb`
  OK, gate-integrity scan clean. The Codex second opinion produced one P3 ordering nit (405-before-404
  on `POST /unknown`), correctly dismissed as a non-defect style nit consistent with sibling leaves.
- **No open `critical` or `normal` issue.** **Open `low` issue (loop-skipped, not a DONE blocker):** 1
  in `issues.md` — `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2 met; M3 in progress (3/4 Verify). The dashboard closed the third M3 criterion — finish the
same HTML arc with the log browser to close M3.** CI green at HEAD, no `critical`/`normal` open, so
feature work proceeds.

Convergence-driven order:
1. **`GET /<domain>/log/` log browser** — `200 text/html` exposing the mirrored checkpoint
   `(size, root)` with links into `entries`/proofs. Mounts under the per-hub subtree (`hubHandler`),
   so it belongs in `proofserve` or a new per-hub HTML leaf, NOT `internal/dashboard`. Closes the
   fourth (final) M3 Verify criterion.
2. **Proof-bundle assembler** — `{checkpoint, inclusion proof, record bytes, hub key, ots?}` as one
   downloadable client-verifiable artifact (the authoritative path the in-browser verifier shares);
   `serveVerify` already composes most pieces. Connective tissue toward the WASM milestone.
3. **sb1 fixture refresh** (stale did.json key), **real alert transport**, and an end-to-end
   `inactive`-render assertion once a registry-deactivation writer lands — all off the Verify bar.
4. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
