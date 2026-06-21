<!-- assessed-at: 0f0a2ec78fb16aa9d97448bf9fa379c06d9af8d8 -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 (Aggregator) — proof surface 2-of-3 served; cross-cutting HTTP plumbing (/healthz) landing

The monitor follows + mirrors + verifies hubs (M1 met) and now serves two of M2's three computed-proof
endpoints per hub from the local mirror (`/inclusion`, `/consistency`), never re-hitting the hub. This slice
added a `GET /healthz` liveness/store-readiness probe on the shared mux. The third proof surface (`entries`)
plus M3 (dashboard / verify-for-me / CORS) / WASM / OTS remain unstarted.

Incremental review of `a18a17c..HEAD` (HEAD `0f0a2ec` on `develop`). Four commits: define-next / advance /
review of the `/healthz` slice, plus the prior `update-state`. The diff touches **three production files** —
`internal/healthz/handler.go` (new leaf: `Handler(Pinger)` → 200 `{"status":"ok"}` / 503
`{"status":"unavailable"}` / 405 non-GET), `internal/store/sqlite.go` (+13, thin `Store.Ping(ctx)` →
`db.PingContext`), and `cmd/iscc-monitor/main.go` (+12, `buildMux` mounts `/healthz` as an exact path next to
`/metrics`). **No `go.mod`, `go.sum`, or `schema.sql` change.** Verified in code: `healthz.Handler` depends on
a local 1-method `Pinger` interface (no `store`/`database/sql` import) and `*store.Store` satisfies it
structurally; dep direction is binary → healthz, never reverse. Latest `review` handoff (2026-06-21, "Serve GET
/healthz") is **PASS / CONTINUE**: `mise run check` green (14 packages `ok`), leaf-purity verified both
directions, oracle gate correctly N/A (HTTP wiring + DB ping), scope-clean (3 production files). **CI green at
HEAD `0f0a2ec`** (develop run 27896063940, `success`).

**Branch note:** active work happens on `develop` (HEAD `0f0a2ec`, tree clean, in sync with `origin/develop`);
a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice added an HTTP `/healthz` probe + `Store.Ping`; no M1
verification logic touched). All Verify criteria satisfied: `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals at HEAD**: **191 `func Test`** across `cmd/` + `internal/`, **44** `_test.go` files (+4 funcs /
  +1 file over the prior 187/43 — the new `internal/healthz/handler_test.go` plus `internal/store/sqlite_test.go`
  Ping cases and a `main_test.go` healthz-on-mux sub-test).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,healthz,logclient,
  metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}`. Module path `github.com/iscc/iscc-monitor`,
  `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). No regression.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics`, `/healthz`, the per-hub mirror, and the per-hub `/inclusion` + `/consistency` proof routes all
  ride one `*http.ServeMux` on a single listener (`buildMux`/`mirrorHandler`/`hubHandler`,
  `cmd/iscc-monitor/main.go`). `/metrics` + `/healthz` mount as exact paths; per-hub `/<domain>/log/` subtrees
  use the trailing-slash subtree match. `internal/metrics` leaf, `slog` structured logging, `SQLiteFetcher` +
  partial-tile mirror CRUD, hub_keys cache, coverage tracking (set-once, ADR-0001), `logclient.Origin` + golden
  `TestOrigin`, config loader, realm-registry parser (domains-only), poll-loop cadence (single-writer), freeze +
  alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
  `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a leaf (verified
  `go list -deps ./internal/store` shows no `net/http`, no `proofserve`, no `healthz`). `Store.Ping` added this
  slice delegates to `db.PingContext`; `schema.sql` byte-unchanged.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles**. All tile/fsck/inclusion/consistency/projection/
  serve tests run against in-process `testonly.Tree` / `buildVerifiedMirror` / seeded-`SQLiteFetcher` fixtures.
  Real tile/entry-bundle + `IsccLogInclusionProof` fixtures remain a soft prerequisite for an *inbound*
  hub-evidence transport. Known stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests;
  fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`, both
  proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`, `tessera/client`
  re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib `slog`/`net/http`/`os`. **Not yet
  wired**: `nbd-wtf/opentimestamps` (re-verified: grep → no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: partially met — both original Verify criteria exercised (fsck root-rebuild WIRED on every poll;
inclusion cross-check conformance-tested over the real verified mirror), and **two of three** computed-proof
endpoints (`inclusion`, `consistency`) are now **served**. **Remaining**: serve `entries`.
- **fsck root-rebuild (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after `ingestTiles`,
  before `recordVerdict`) builds a read-only `store.SQLiteFetcher` and calls `logclient.RunFsck`, comparing the
  rebuilt RFC-6962 root to the signed checkpoint root. Mutation-proven. Carried forward (untouched).
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` folds the raw
  bundle via `logclient.BundleProjections` → `store.RecordProjections`. Carried forward (untouched).
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED):** `internal/logclient/inclusioncheck.go` closed against
  M2's Verify bar by `internal/follower/inclusion_test.go`. Carried forward.
- **raw tlog-tiles HTTP read surface (BUILT + WIRED):** `internal/tilesserve/handler.go` routes the three
  canonical iscc-log §9 paths (`/checkpoint`, `/tile/...`, `/tile/entries/...`) to raw mirror BLOBs verbatim,
  per hub via `hubHandler`'s `/` fall-through. Carried forward.
- **computed inclusion proof (BUILT + WIRED):** `serveInclusion` serves `GET /inclusion?iscc_id=<id>[&index=
  <n>]` from the mirror against `LastSize`, shaped like the hub's `IsccLogInclusionProof`. Carried forward.
- **computed consistency proof (BUILT + WIRED):** `internal/proofserve/handler.go` `serveConsistency` serves
  `GET /consistency?from=<n>` from the mirror against `FollowState.LastSize`, JSON
  `ConsistencyEvidence{type:"IsccLogConsistencyProof", firstSize, secondSize, consistencyProof:[base64-Std]}`.
  RFC-6962 oracle gate reviewer-mutation-proven non-vacuous. Carried forward.
- **`CheckpointAt` determinism (in place):** `internal/store/checkpoints.go` `CheckpointAt` is `… ORDER BY rowid
  LIMIT 1`, returning the lowest-rowid (prior accepted) root deterministically over a later
  contradicting-evidence row at the same size. Carried forward; verified by `TestCheckpointAtDeterministicOnFork`.

**What remains for M2's Verify bar:**
1. **Serve `entries`** computed/served from the local mirror to complete the M2 proof surface — **not started**
   (verified: no `/entries` route in `proofserve` or `hubHandler`'s switch, which mounts only `/`, `/inclusion`,
   `/consistency`). This is the natural slice to also start owning the M3-deferred CORS / caching /
   conditional-GET. (`/healthz` is now done.)

## M3 — Trust API + dashboard
**Status**: not started. The binary's mux now serves `/metrics`, `/healthz`, the per-hub raw tlog-tiles mirror
subtrees, and the per-hub `/inclusion` + `/consistency` computed-proof endpoints. Everything else is absent
(verified): no `/` landing, no `entries` proof route, no `verify-for-me` verdict surface, no server-rendered
dashboard, no log browser, no CORS (grep → no `Access-Control` header code; only "out of scope" comments). The
raw-mirror + proof handlers that M3's log browser / canonical tlog-tiles paths build on are mounted.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js` in
source). This slice does not touch the WASM-shareable purity invariant; the WASM-shared verifier seam rides on
`internal/didweb` (untouched; review re-confirmed `GOOS=js GOARCH=wasm go build ./internal/didweb` OK).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run check`
  runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (three production files changed, no
  dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `0f0a2ec`:
  `conclusion: success`** (run 27896063940).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (14 packages `ok`
  uncached incl. `healthz`, `go vet`/`gofmt -l .` clean), healthz tested (200/503/405), leaf-purity verified
  both directions (`go list -deps ./internal/healthz` has no `store`/`database/sql`; `./internal/store` has no
  `net/http`/`healthz`), oracle gate correctly N/A, scope-clean (3 production files), WASM purity invariant green.
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`: `TestPollHubFork`
  re-detection still bypasses `PollHub` (store `CheckpointAt` root cause RESOLVED, but the follower test + a
  stale comment remain); `AcceptCheckpoint` discards resolved context → verified polls re-fetch did.json;
  "frozen hubs still advance accepted state on later clean polls" (ADR-0006); tile writers require `width`
  (duplicated `p`-translation in follower); accepted-checkpoint advancement is three caller-sequenced store
  writes (locality); self-consistency policy split across follower + logclient (refactor). **`low`
  (loop-skipped):** `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**Complete M2 → serve the remaining computed proof (`entries`) over HTTP.** No `critical` open, so feature work
proceeds; the open `normal` issues also block DONE but are weighed against the state→target gap. Inclusion and
consistency are now served; `entries` is the last M2 proof surface.

Candidate order:
1. **Serve `entries`** computed/served from the local mirror, reusing the `/inclusion` + `/consistency` routing
   pattern in `hubHandler`'s switch — the last M2 Verify-bar item. Start owning the M3-deferred CORS / caching /
   conditional-GET here (`/healthz` is done).
2. **`TestPollHubFork` cleanup** (quick win, now that `CheckpointAt` is deterministic): re-detect via a second
   `PollHub` and delete the stale "unordered LIMIT 1 … non-deterministic" comment — closes the retargeted issue.
3. **`normal` backlog**: "frozen hubs still advance" (ADR-0006 evidence-only); `AcceptCheckpoint` context reuse;
   tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store method; `CheckConsistency` collapse.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 → WASM → OTS** remain after M2's Verify bar is fully met.
