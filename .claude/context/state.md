<!-- assessed-at: 706eb1e4ae42a33258efe6e0a9af61d877b845ce -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — tlog-tiles mirror HTTP arc + all three computed proofs +
verify-for-me JSON verdict complete; the server-rendered dashboard, the HTML log browser, the WASM
verifier, and OTS anchoring remain.

The monitor follows + mirrors + verifies hubs (M1 met), serves all three computed proofs per hub from
the local mirror (M2 met), and now answers the verify-for-me verdict route
(`GET /<domain>/log/verify`). M3 is at 2/4 Verify criteria; the two HTML surfaces (dashboard + log
browser), the WASM verifier, and OTS anchoring are the remaining v1 work.

## Convergence
- **Remaining Verify criteria:**
  - **M3: 2/4 open** — closed: `Access-Control-Allow-Origin: *` on every GET, and the verify-for-me
    JSON verdict (`GET /<domain>/log/verify?iscc_id=<id>`, hub status + checkpoint `(size, root)` +
    Merkle-verified inclusion result, id-faults → 200 non-verified, never 5xx). Still open: `GET /`
    HTML dashboard (every realm hub + status + coverage), `GET /<domain>/log/` HTML log browser.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported).
  - M1: 0 open (met). M2: 0 open (met).
- **Last ~10 iterations: ~2 milestone-Verify / ~8 refactor·polish·test·infra.** The polish-streak
  drift flagged last assessment broke: the loop landed the verify-for-me slice
  (`562a4d8`/`b537abe`/`8f0aa9d`), closing a second M3 Verify criterion after a run of
  refactor/infra commits (`RecordTile p uint8`, learnings split, CID-steering prompt, Codex hook,
  devcontainer cap). HEAD `706eb1e` is a one-line loop-infra fix (Codex launch command) with zero
  production code. Convergence is back on track; the next two M3 criteria (dashboard + log browser)
  are the same HTML arc and clearly reachable — keep attacking Verify criteria, not tooling.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change in the assessed range
(`fa8d624..706eb1e` touched only `proofserve`, `cmd/iscc-monitor/main.go`, and loop context). All
Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers
(fork/shrink/equivocation) golden-tested end-to-end with freeze + alert-once + restart survival,
coverage tracked, structured logs, `/metrics` served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **215 `func Test`** across `cmd/` + `internal/`, **49** `_test.go` files
  (up from 207/48 — the +8 tests / +1 file are the new `proofserve/verify_test.go`).
- **Packages present (re-verified)**: `cmd/{iscc-monitor,notecheck}`; **13 internal packages** —
  `config, corsmw, didweb, follower, healthz, logclient, metrics, metricshttp, proofserve,
  registry, store, tiles, tilesserve`. Module `github.com/iscc/iscc-monitor`, `go 1.24.0` (no
  `toolchain` line).
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
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; `AlertFunc func(int64,string)` seam unchanged. The warm-path's second `did.json`
  resolve is a larger design change, not on the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**
  (re-verified by `ls`). Tile/fsck/inclusion/consistency/entries/verify tests run against in-process
  fixtures. Stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests; not refreshed.
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
**Status**: **in progress** (2/4 Verify criteria met). The raw tlog-tiles mirror is fully plumbed:
CORS on every public GET via the single `corsmw.Handler` wrap (closes the
`Access-Control-Allow-Origin: *` criterion); per-route `Cache-Control`; a strong content ETag +
`If-None-Match`→`304` conditional GET on the static-mirror 200s.

**Verify-for-me landed** (`internal/proofserve/handler.go`, `serveVerify` + `VerifyVerdict`, mounted
at `/verify` in `cmd/iscc-monitor/main.go:205`): `GET /<domain>/log/verify?iscc_id=<id>` returns the
store-provable `hub_status`, the accepted checkpoint `(size, root)`, and a REAL RFC-6962 inclusion
result recomputed from the local mirror and Merkle-verified against the accepted root
(`proof.VerifyInclusion`). Every id-shaped fault (missing/unknown id, leaf not covered, tile not
mirrored, no accepted checkpoint) is a 200 non-verified verdict; non-200 is reserved for genuine infra
faults. Golden-tested across the 256-leaf boundary; the inclusion check was reviewer-mutation-proven
non-vacuous through the HTTP seam (handoff PASS). **Closes the second M3 Verify criterion.**

**Still absent (re-verified by grep at this assessment):**
- Server-rendered dashboard `GET /` (status/coverage/lag/violations/OTS) — absent (`/` returns 404).
  No `html/template`/`text/template` import anywhere (grep clean). **M3 Verify open.**
- Log browser `GET /<domain>/log/` HTML — absent. **M3 Verify open.**
- ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces (`internal/proofserve`
  carries none — tied to `LastSize`; `/verify` has no caching either); not a Verify criterion.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits
in `cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM
build target (`syscall/js` not in source — grep clean). The WASM-shared verifier seam continues to
ride on `internal/didweb`; `serveVerify` now composes most of the proof-bundle pieces the in-browser
verifier will share (it discards the raw checkpoint bytes and the resolved hub key, which the bundle
path will need).

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** at HEAD `8f0aa9d` (run 27906233830) — the verify-for-me production HEAD.
  HEAD `706eb1e` is a one-line loop-infra fix (no Go source), so no newer production CI run applies.
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**, for the `/verify` route) records `mise run
  check` green (15 packages `ok`, `go vet`/`gofmt -l .` clean), the RFC-6962 inclusion path
  reviewer-mutation-proven non-vacuous through the HTTP seam, store confirmed a leaf, gate-integrity
  scan clean. The Codex second opinion was unavailable (env denied the bypass flag) — recorded as a
  graceful-degradation note, not a blocker.
- **No open `critical` or `normal` issue.** **Open `low` issue (loop-skipped, not a DONE blocker):** 1
  in `issues.md` — `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2 met; M3 in progress (2/4 Verify). The verify-for-me slice broke the polish streak and closed a
Verify criterion — continue the same HTML arc to finish M3.** CI green, no `critical`/`normal` open,
so feature work proceeds.

Convergence-driven order:
1. **`GET /` dashboard** — `200 text/html` listing **every** realm hub with its glossary status +
   coverage window (lag, violations, OTS), golden-tested at the HTTP seam on a fixture store. Closes
   the third M3 Verify criterion.
2. **`GET /<domain>/log/` log browser** — `200 text/html` exposing the mirrored checkpoint
   `(size, root)` with links into `entries`/proofs. Closes the fourth (final) M3 Verify criterion.
3. **Proof-bundle assembler** — `{checkpoint, inclusion proof, record bytes, hub key, ots?}` as one
   downloadable client-verifiable artifact (the authoritative path the in-browser verifier shares);
   `serveVerify` already composes most pieces. Connective tissue toward the WASM milestone.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** — off the Verify bar.
5. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
