<!-- assessed-at: fa8d62464aa9207d931b56069220ca011850c490 -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — tlog-tiles mirror HTTP arc + all three computed proofs
complete; verify-for-me, the server-rendered dashboard, the log browser, the WASM verifier, and OTS
anchoring all remain unstarted.

The monitor follows + mirrors + verifies hubs (M1 met) and serves all three computed proofs per hub
from the local mirror (M2 met: `/inclusion`, `/consistency`, `/entries`). M3 has the raw tlog-tiles
mirror fully plumbed (CORS → Cache-Control → conditional GET); the bulk of M3 (verify-for-me,
dashboard, log browser), the WASM verifier, and OTS anchoring remain.

**This assessment is a no-op on production code.** Incremental review of
`650f965..HEAD` (HEAD `fa8d624` on `develop`). The diff (`git diff 650f965..HEAD --stat -- cmd/
internal/ go.mod go.sum mise.toml .github/` is **empty**) touches **only loop infrastructure**: the
five commits since the last assessment are `.claude/agents/*` prompt edits, splitting `learnings.md`
into per-package detail files, the CID convergence-steering prompt change, a Codex second-opinion
review hook, and a `.devcontainer` memory/parallelism cap. **No `cmd/`, `internal/`, `go.mod`, or
`mise.toml` change.** Every milestone section below is carried forward from the `650f965` assessment
and re-confirmed by spot-check (see Quality gates). The last *production* commit remains `650f965`,
whose `review` handoff is **PASS / CONTINUE** with CI green.

**Branch note:** active work happens on `develop` (HEAD `fa8d624`, **working tree clean**, ahead of
`origin/develop` by 5 commits — the loop-infra commits are unpushed). A human merges
`develop`→`main`. `main` lags. The `.devcontainer` tuning is now committed (was uncommitted at the
last assessment).

## Convergence
- **Remaining Verify criteria:**
  - **M3: 3/4 open** — CORS-on-every-GET is met (the only closed M3 criterion); still open:
    `verify-for-me` JSON verdict (`GET /<domain>/log/verify`), `GET /` HTML dashboard (every realm
    hub + status + coverage), `GET /<domain>/log/` HTML log browser.
  - **WASM verifier: 1/1 open** (not started — no `internal/proof`, no `syscall/js`).
  - **OTS anchoring: 1/1 open** (not started — `nbd-wtf/opentimestamps` not imported).
  - M1: 0 open (met). M2: 0 open (met).
- **Last ~10 iterations: ~1 milestone-Verify / ~9 refactor·polish·test·infra — a drift signal.**
  Only the CORS slice closed an M3 Verify criterion. The rest were refactor/collapse
  (`RecordTile p uint8` single-source, `AdvanceAccepted` tx, `CheckConsistency` collapse,
  `AcceptCheckpoint` context reuse), mirror polish (Cache-Control, conditional GET), a test-only
  slice, an M1 correctness edge (frozen-evidence-only), and now a run of pure loop-tooling commits.
  **The five newest commits are 100% loop/context/devcontainer infra — zero production code.** The
  three biggest, clearly-reachable M3 Verify criteria (verify-for-me, dashboard, log browser) plus
  WASM and OTS have not been touched while the loop has spent its recent budget on internal cleanups
  and meta-tooling. **Flag:** the next iterations should attack an open Verify criterion (start the
  verify-for-me / dashboard arc) rather than further refactor or tooling.

## M1 — Read-only Monitor
**Status**: met — carried forward; no production change since `650f965`. All Verify criteria remain
satisfied: `origin`/`vkey` golden, all three triggers (fork/shrink/equivocation) golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP. **CI-gated & green at the last production HEAD.**

- **Test totals at HEAD**: **207 `func Test`** across `cmd/` + `internal/`, **48** `_test.go` files
  (both re-counted, unchanged from `650f965`).
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
  (re-verified by `ls`). Tile/fsck/inclusion/consistency/entries tests run against in-process
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
`internal/store/fetcher.go:113`, **re-verified as the single definition**; the follower's duplicate
was deleted). **Nothing remains on the M2 Verify bar.**

## M3 — Trust API + dashboard
**Status**: **in progress** (1/4 Verify criteria met). The raw tlog-tiles mirror is fully plumbed:
(1) CORS on every public GET via the single `corsmw.Handler` wrap — **closes the
`Access-Control-Allow-Origin: *` Verify criterion**; (2) per-route `Cache-Control`; (3) a strong
content ETag + `If-None-Match`→`304` conditional GET on every 200. The binary's mux serves
`/metrics`, `/healthz`, the per-hub raw tlog-tiles subtrees, and the per-hub `/inclusion` +
`/consistency` + `/entries` computed-proof endpoints, all behind CORS.

**Still absent (re-verified by grep at this assessment):**
- `verify-for-me` verdict surface — no `GET /<domain>/log/verify` route, no `html/template` /
  `text/template` import anywhere (grep clean). **M3 Verify open.**
- Server-rendered dashboard `GET /` (status/coverage/lag/violations/OTS) — absent (`/` returns 404).
  **M3 Verify open.**
- Log browser `GET /<domain>/log/` HTML — absent. **M3 Verify open.**
- ETag/Cache-Control/conditional-GET on the size-dependent proof surfaces (`internal/proofserve`
  carries none — tied to `LastSize`); not a Verify criterion but a noted gap.

## WASM verifier · OTS anchoring
**Status**: **not started** (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits
in `cmd/`+`internal/`+`go.mod`); no `internal/proof` package (`ls` → no such directory); no WASM
build target (`syscall/js` not in source — grep clean). The WASM-shared verifier seam continues to
ride on `internal/didweb`; `logclient.CheckConsistency` could share a WASM seam later.

## Quality gates
**Status**: **green** — enforced in CI.
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise
  run check` runnable. `cmd/`/`internal/`/`go.mod`/`go.sum`/`mise.toml` **byte-unchanged since the
  last assessment** (`git diff 650f965..HEAD --stat` over those paths is empty).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate
  (`go build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop`:
  `conclusion: success`** at HEAD `650f965` (run 27899275324) — the most recent *production* HEAD.
  The 5 commits since are loop/docs/infra only (no Go source), so no newer production CI run applies.
- Latest *production* `review` handoff (2026-06-21, **PASS / CONTINUE**, for `650f965`) records `mise
  run check` green (15 packages `ok`, `go vet`/`gofmt -l .` clean), the public-surface `widthForP`
  invariant reviewer-mutation-proven non-vacuous, oracle gate correctly N/A with the four
  mirror-consuming packages re-run uncached, store confirmed a leaf, gate-integrity scan clean.
- **No open `critical` or `normal` issue.** **Open `low` issue (loop-skipped, not a DONE blocker):** 1
  in `issues.md` — `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2 met; M3 in progress (1/4 Verify) — the tlog-tiles mirror HTTP arc + three computed proofs
are done and the `normal` backlog is drained. The last five iterations were loop/refactor/infra with
no Verify-criterion progress — break the polish streak and attack an open M3 Verify criterion.**
CI green, no `critical`/`normal` open, so feature work proceeds.

Convergence-driven order:
1. **verify-for-me REST surface** — `GET /<domain>/log/verify?iscc_id=<id>` returning the documented
   JSON verdict (hub status + checkpoint `(size, root)` + inclusion result; malformed/unknown id →
   documented non-verified verdict, never 5xx). Directly closes an M3 Verify criterion; reads the
   unified store/`SQLiteFetcher` seam. Pair with the shared proof-bundle assembler
   (`{checkpoint, inclusion proof, record bytes, hub key, ots?}`) the in-browser verifier also needs.
2. **`GET /` dashboard** + **`GET /<domain>/log/` log browser** — the remaining two M3 Verify
   criteria (server-rendered HTML, golden-tested at the HTTP seam on a fixture store).
3. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path)
   — connective tissue, off the Verify bar.
4. **WASM verifier → OTS anchoring** remain the last v1 milestones (each 1/1 Verify open).
