<!-- assessed-at: a18a17cd37a0b0cbb0bb1eb9cccce383db231522 -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 (Aggregator) — computed consistency proofs now SERVED over HTTP, alongside inclusion

Two of M2's three computed-proof endpoints are now live per hub from the local mirror, never re-hitting the
hub: `GET /inclusion?iscc_id=<id>[&index=<n>]` and `GET /consistency?from=<n>`. This slice added the
consistency route and landed the deterministic `CheckpointAt … ORDER BY rowid` prior-root fix. The third
proof surface (`entries`) plus the M3-deferred cross-cutting concerns (CORS / caching / conditional-GET /
healthz) remain. M3 / WASM / OTS not started.

Incremental review of `7f01cd5..HEAD` (HEAD `a18a17c` on `develop`). The diff touches **three production
files** — `internal/proofserve/handler.go` (+178/-, adds `serveConsistency` + `ConsistencyEvidence` +
`writeConsistency`), `internal/store/checkpoints.go` (+12/-, `CheckpointAt` now `ORDER BY rowid LIMIT 1`),
and `cmd/iscc-monitor/main.go` (+17, mounts `/consistency` on the per-hub mux) — plus
`internal/proofserve/consistency_test.go`, `internal/store/checkpoints_test.go`, and context files. **No
`go.mod`, `go.sum`, or `schema.sql` change.** Verified in code: `Handler` switches `/inclusion` →
`serveInclusion` and `/consistency` → `serveConsistency`; both proofs computed against `FollowState.LastSize`
over the read-only `store.SQLiteFetcher`. Latest `review` handoff (2026-06-21, "Serve computed consistency
proofs over HTTP") is **PASS / CONTINUE**: `mise run check` green (13 packages `ok`), RFC-6962 consistency
oracle gate reviewer-mutation-proven non-vacuous (corrupt served proof → fails byte-equal + `VerifyConsistency`;
reverse `rowid` order → fork-determinism test fails), store stays a leaf, notecheck oracle green. **CI green
at HEAD `a18a17c`** (develop run 27895825621, `success`).

**Branch note:** active work happens on `develop` (HEAD `a18a17c`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice added a consistency proof route + the `CheckpointAt`
ordering fix; no M1 verification logic touched). All Verify criteria satisfied: `origin`/`vkey` golden, all
three triggers golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked,
structured logs, `/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals at HEAD**: **187 `func Test`** across `cmd/` + `internal/`, **43** `_test.go` files (+9
  funcs / +1 file over the prior 178/42 — the new `internal/proofserve/consistency_test.go` plus added
  `internal/store/checkpoints_test.go` cases incl. `TestCheckpointAtDeterministicOnFork`).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,logclient,
  metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}`. Module path
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). No regression.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics` served + wired, alongside the per-hub mirror + `/inclusion` + `/consistency` routes on the same
  mux (`buildMux`/`mirrorHandler`/`hubHandler`, `cmd/iscc-monitor/main.go`), `internal/metrics` leaf, `slog`
  structured logging, `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking
  (set-once, ADR-0001), `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser
  (domains-only), poll-loop cadence (single-writer), freeze + alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`,
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a
  leaf (review re-confirmed `go list -deps ./internal/store` shows no `proofserve`, no `net/http`).
  `schema.sql` byte-unchanged this slice; only `checkpoints.go`'s `CheckpointAt` query changed.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles**. All tile/fsck/inclusion/consistency/projection/
  serve tests run against in-process `testonly.Tree` / `buildVerifiedMirror` / seeded-`SQLiteFetcher`
  fixtures. Real tile/entry-bundle + `IsccLogInclusionProof` fixtures remain a soft prerequisite for an
  *inbound* hub-evidence transport. Known stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in
  tests; fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`,
  both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib
  `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (re-verified: grep → no hits in
  `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: partially met — both original Verify criteria exercised (fsck root-rebuild WIRED on every poll;
inclusion cross-check conformance-tested over the real verified mirror), and **two of three** computed-proof
endpoints (`inclusion`, `consistency`) are now **served**. **Remaining**: serve `entries`.
- **fsck root-rebuild (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after `ingestTiles`,
  before `recordVerdict`) builds a read-only `store.SQLiteFetcher` and calls `logclient.RunFsck`, comparing
  the rebuilt RFC-6962 root to the signed checkpoint root. Mutation-proven. Carried forward (untouched).
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` folds the raw
  bundle via `logclient.BundleProjections` → `store.RecordProjections`. Carried forward (untouched).
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED):** `internal/logclient/inclusioncheck.go` closed
  against M2's Verify bar by `internal/follower/inclusion_test.go`. Carried forward.
- **raw tlog-tiles HTTP read surface (BUILT + WIRED):** `internal/tilesserve/handler.go` routes the three
  canonical iscc-log §9 paths to raw mirror BLOBs verbatim, per hub via `hubHandler`'s `/` fall-through.
  Carried forward.
- **computed inclusion proof (BUILT + WIRED):** `serveInclusion` serves `GET /inclusion?iscc_id=<id>[&index=
  <n>]` from the mirror against `LastSize`, shaped like the hub's `IsccLogInclusionProof`. Carried forward.
- **computed consistency proof (BUILT + WIRED — this slice):** `internal/proofserve/handler.go`
  `serveConsistency` serves `GET /consistency?from=<n>`: `parseUint(from)` → `FollowState.LastSize` (=
  larger size) gate → `from > size` 400 (RFC-6962 M≤N) → `CheckpointAt(from)` must exist (404 if not; `from
  == 0` empty-tree prior is exempt) → `ConsistencyProofFromTiles(from, size)` over the mirror → JSON
  `ConsistencyEvidence{type:"IsccLogConsistencyProof", firstSize, secondSize, consistencyProof:[base64-Std]}`.
  Degenerate `from ∈ {0, LastSize}` → 200 empty proof; unmirrored tile (wrapped `os.ErrNotExist`) → 404;
  non-GET 405, unmatched path 404, else 500. Response is verifier-computed (iscc-log §10.2), so it is a local
  shape, NOT a hub-served member. Mounted per hub at `/consistency` on the nested mux (`hubHandler`,
  `cmd/iscc-monitor/main.go`). Oracle gate (RFC-6962 consistency) reviewer-mutation-proven non-vacuous.
- **`CheckpointAt` determinism fix (this slice):** `internal/store/checkpoints.go` `CheckpointAt` is now
  `… ORDER BY rowid LIMIT 1`, returning the lowest-rowid (prior accepted) root deterministically over a
  later contradicting-evidence row at the same size. Resolves the store-level half of the open `CheckpointAt`
  issue; verified by `TestCheckpointAtDeterministicOnFork`.

**What remains for M2's Verify bar:**
1. **Serve `entries`** computed/served from the local mirror to complete the M2 proof surface — **not
   started**. This is the natural slice to also start owning the M3-deferred CORS / caching / conditional-GET
   / healthz.

## M3 — Trust API + dashboard
**Status**: not started. The binary's mux now serves `/metrics`, the per-hub raw tlog-tiles mirror subtrees,
and the per-hub `/inclusion` + `/consistency` computed-proof endpoints; everything else is absent: no `/`,
`/healthz`, `entries` proof route, `verify-for-me` verdict surface, server-rendered dashboard, log browser,
or CORS. The raw-mirror + proof handlers that M3's log browser / canonical tlog-tiles paths build on are
mounted.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js` in
source). This slice does not touch the WASM-shareable purity invariant (`proofserve` is a `net/http` leaf
depending on store+logclient; the WASM-shared verifier seam rides on `internal/didweb`, untouched, review
re-confirmed `GOOS=js GOARCH=wasm go build ./internal/didweb` OK).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (three production files
  changed, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `a18a17c`:
  `conclusion: success`** (run 27895825621).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (13 packages
  `ok`, `go vet`/`gofmt -l .` clean), served consistency proof verified via `proof.VerifyConsistency` for
  `from ∈ {1,200,255,256,299}`, RFC-6962 oracle gate reviewer-mutation-proven non-vacuous, dep direction
  confirmed (store stays a leaf), scope-clean (3 production files, no `schema.sql`/dep change), WASM purity
  invariant green.
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `TestPollHubFork` re-detection still bypasses `PollHub` (store `CheckpointAt` root cause now RESOLVED, but
  the follower test + a stale comment remain); `AcceptCheckpoint` discards resolved context → verified polls
  re-fetch did.json; "frozen hubs still advance accepted state on later clean polls" (ADR-0006); tile writers
  require `width` (duplicated `p`-translation in follower); accepted-checkpoint advancement is three
  caller-sequenced store writes (locality); self-consistency policy split across follower + logclient
  (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**Complete M2 → serve the remaining computed proof (`entries`) over HTTP.** No `critical` open, so feature
work proceeds; the open `normal` issues also block DONE but are weighed against the state→target gap.
Inclusion and consistency are now served; `entries` is the last M2 proof surface.

Candidate order:
1. **Serve `entries`** computed/served from the local mirror, reusing the `/inclusion` + `/consistency`
   routing pattern in `hubHandler`'s switch — the last M2 Verify-bar item. Start owning the M3-deferred
   CORS / caching / conditional-GET / healthz here.
2. **`TestPollHubFork` cleanup** (quick win, now that `CheckpointAt` is deterministic): re-detect via a second
   `PollHub` and delete the stale "unordered LIMIT 1 … non-deterministic" comment — closes the retargeted
   issue.
3. **`normal` backlog**: "frozen hubs still advance" (ADR-0006 evidence-only); `AcceptCheckpoint` context
   reuse; tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store method; `CheckConsistency`
   collapse.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 → WASM → OTS** remain after M2's Verify bar is fully met.
