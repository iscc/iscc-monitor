<!-- assessed-at: 0ef8acfbaeacdbe6790779784b90668d6c29cd4a -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — cross-cutting HTTP plumbing in progress. CORS landed; per-route
Cache-Control on the mirror landed. Conditional-GET, verify-for-me, dashboard, and the log browser
remain open. WASM verifier + OTS anchoring not started.

The monitor follows + mirrors + verifies hubs (M1 met) and serves all three computed-proof endpoints
per hub from the local mirror (M2 met: `/inclusion`, `/consistency`, `/entries`). M3 has begun with
two cross-cutting HTTP slices — a single CORS wrap on the public mux, then per-route `Cache-Control`
on the raw tlog-tiles mirror. The bulk of M3 (conditional-GET, verify-for-me, server-rendered
dashboard, log browser), the WASM verifier, and OTS anchoring remain unstarted.

Incremental review of `b5657e8..HEAD` (HEAD `0ef8acf` on `develop`). Three commits: define-next /
advance / review of the Cache-Control slice. The diff touches **one production file** —
`internal/tilesserve/handler.go` — plus its `_test.go` and context docs. **No `go.mod`, `go.sum`, or
`schema.sql` change** (verified `git diff --quiet b5657e8..HEAD -- go.mod go.sum
internal/store/schema.sql` exit 0). The change plumbs an `immutable bool` into `writeBlob` and sets a
per-route `Cache-Control` on every tlog-tiles 200: content-addressed FULL tiles/bundles (path-API
`width == 0`) get `public, max-age=31536000, immutable` (`cacheImmutable` const); partials
(`width > 0`) and the size-varying checkpoint get `no-cache` (`cacheRevalidate`). Verified from source:
the immutable directive lives in exactly one place; `serveTile`/`serveEntries` pass `width == 0` as the
immutable discriminant; `serveCheckpoint` passes `false`. Latest `review` handoff (2026-06-21,
"Cache-Control on the tlog-tiles mirror") is **PASS / CONTINUE**: `mise run check` green (15 packages
`ok`), the load-bearing partial-tile `no-cache` assertion reviewer-mutation-proven non-vacuous, no
header conflict with the `corsmw` `Access-Control-*` wrap, oracle gate correctly N/A (pure HTTP header
wiring on opaque BLOBs — no signature/RFC-6962/Merkle/did:web/fsck path), scope-clean (1 production
file). **CI green at HEAD `0ef8acf`** (develop run 27896955526, `conclusion: success`).

**Branch note:** active work happens on `develop` (HEAD `0ef8acf`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice only set a `Cache-Control` header on the
already-served mirror; no M1 verification logic touched). All Verify criteria satisfied:
`origin`/`vkey` golden, all three triggers golden-tested end-to-end with freeze + alert-once + restart
survival, coverage tracked, structured logs, `/metrics` served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **204 `func Test`** across `cmd/` + `internal/`, **47** `_test.go` files
  (unchanged over the prior assessment — this slice added/edited assertions in
  `internal/tilesserve/handler_test.go` without a net new test function or file).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,corsmw,didweb,follower,
  healthz,logclient,metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}` (14 internal
  packages). Module path `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on
  `true` (ADR-0006). No regression.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real
  sb0 checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics`, `/healthz`, the per-hub mirror, and the per-hub `/inclusion` + `/consistency` + `/entries`
  proof routes all ride one `*http.ServeMux` on a single listener, wrapped once in `corsmw.Handler`
  (`buildMux`/`mirrorHandler`/`hubHandler`, `cmd/iscc-monitor/main.go`). `/metrics` + `/healthz` mount
  as exact paths; per-hub `/<domain>/log/` subtrees use the trailing-slash subtree match, with
  `/inclusion`, `/consistency`, `/entries` mounted as exact paths inside each hub's mux and `/` falling
  through to the raw tlog-tiles mirror. `internal/metrics` leaf, `slog` structured logging,
  `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking (set-once, ADR-0001),
  `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser (domains-only),
  poll-loop cadence (single-writer), freeze + alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`,
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store
  stays a leaf; `schema.sql` byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is
  later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles** (re-verified by `ls`). All tile/fsck/
  inclusion/consistency/entries/projection/serve tests run against in-process `testonly.Tree` /
  `buildVerifiedMirror` / seeded-`SQLiteFetcher` fixtures. Real tile/entry-bundle +
  `IsccLogInclusionProof` fixtures remain a soft prerequisite for an *inbound* hub-evidence transport.
  Known stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests; fixtures not yet
  refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`,
  both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export, `api.EntryBundle.UnmarshalText`), `transparency-dev/formats` (DIRECT —
  `cmd/notecheck`), stdlib `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps`
  (re-verified: grep → no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — carried forward unchanged. Both original Verify criteria are exercised (fsck
root-rebuild WIRED on every verified poll; inclusion cross-check conformance-tested over the real
verified mirror), and the served proof surface is **complete**: all three computed proofs —
`inclusion`, `consistency`, `entries` — served from the local mirror, never re-hitting the hub. This
slice only added a `Cache-Control` header to the raw-mirror read surface; no proof/mirror logic
changed.
- **fsck root-rebuild (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after
  `ingestTiles`, before `recordVerdict`) builds a read-only `store.SQLiteFetcher` and calls
  `logclient.RunFsck`, comparing the rebuilt RFC-6962 root to the signed checkpoint root.
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` folds the
  raw bundle via `logclient.BundleProjections` → `store.RecordProjections`.
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED):** `internal/logclient/inclusioncheck.go` closed
  against M2's Verify bar by `internal/follower/inclusion_test.go`.
- **raw tlog-tiles HTTP read surface (BUILT + WIRED):** `internal/tilesserve/handler.go` routes the
  three canonical iscc-log §9 paths (`/checkpoint`, `/tile/...`, `/tile/entries/...`) to raw mirror
  BLOBs verbatim, per hub via `hubHandler`'s `/` fall-through, now with a per-route `Cache-Control`
  policy (immutable full vs no-cache partial/checkpoint).
- **computed inclusion proof (BUILT + WIRED):** `serveInclusion` serves `GET /inclusion?iscc_id=<id>
  [&index=<n>]` from the mirror against `LastSize`, shaped like the hub's `IsccLogInclusionProof`.
- **computed consistency proof (BUILT + WIRED):** `serveConsistency` serves `GET /consistency?from=<n>`
  from the mirror against `FollowState.LastSize`.
- **computed entries record bytes (BUILT + WIRED):** `serveEntries` serves `GET /entries?index=<seq>`
  from the mirror via the pure `logclient.RecordBytesFromBundle(bundle, offset)`, schema-agnostic
  (ADR-0008), as `application/octet-stream`.
- **`CheckpointAt` determinism (in place):** `internal/store/checkpoints.go` `CheckpointAt` is
  `… ORDER BY rowid LIMIT 1`, returning the lowest-rowid (prior accepted) root deterministically.

**What remains for M2:** nothing on the Verify bar — M2 is met.

## M3 — Trust API + dashboard
**Status**: **in progress**. Two cross-cutting HTTP slices landed: (1) CORS applied uniformly to every
public GET via the single `corsmw.Handler` wrap in `buildMux` (`Access-Control-Allow-Origin: *`;
`OPTIONS` → `204`); (2) per-route `Cache-Control` on the raw tlog-tiles mirror (immutable full,
no-cache partial/checkpoint). The binary's mux serves `/metrics`, `/healthz`, the per-hub raw
tlog-tiles mirror subtrees, and the per-hub `/inclusion` + `/consistency` + `/entries` computed-proof
endpoints — all behind CORS. **Still absent (verified):** conditional-GET (`ETag`/`If-None-Match`/
`Last-Modified`/`304`) on any route; `Cache-Control` on the size-dependent proof surfaces
(`/inclusion`/`/consistency`/`/entries`, tied to `LastSize`); `verify-for-me` verdict surface (no
`html/template` or `text/template` import anywhere; only a comment mention); server-rendered dashboard
(status/coverage/lag/violations/OTS); `/` landing; and the log browser.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js`
in source). This slice's code change (`tilesserve/handler.go`) imports only `errors`/`net/http`/`os`/
`strings`/`tessera/api/layout`/`internal/store`; the WASM-shared verifier seam continues to ride on
`internal/didweb` (untouched).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (one production file
  changed, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for
  HEAD `0ef8acf`: `conclusion: success`** (run 27896955526).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (15 packages
  `ok`, `go vet`/`gofmt -l .` clean), the partial-tile `no-cache` assertion reviewer-mutation-proven
  non-vacuous, no `corsmw`/`Cache-Control` header conflict, oracle gate correctly N/A (pure HTTP header
  wiring), trust root untouched + green, scope-clean (1 production file).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `TestPollHubFork` re-detection still bypasses `PollHub` (store `CheckpointAt` root cause RESOLVED, but
  the follower test + a stale comment remain); `AcceptCheckpoint` discards resolved context → verified
  polls re-fetch did.json; "frozen hubs still advance accepted state on later clean polls" (ADR-0006);
  tile writers require `width` (duplicated `p`-translation in follower); accepted-checkpoint advancement
  is three caller-sequenced store writes (locality); self-consistency policy split across follower +
  logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M2 met; M3 in progress (CORS + mirror Cache-Control landed). Continue M3 cross-cutting HTTP plumbing,
then the dashboard / verify-for-me arc.** No `critical` open, so feature work proceeds; the 6 open
`normal` issues block DONE and are weighed against the state→target gap.

Candidate order:
1. **M3 conditional-GET slice** — add `ETag`/`If-None-Match` (and/or `Last-Modified`/`If-Modified-Since`)
   with `304` on the mirror routes. Content-addressed full tiles/bundles have a strong-ETag source (the
   BLOB is keyed by its hashes); the checkpoint/partials need a weak/derived validator. The size-varying
   proof surfaces (`/inclusion`/`/consistency`/`/entries`) still carry no cache policy — a related
   follow-up.
2. **`TestPollHubFork` cleanup** (quick win, now that `CheckpointAt` is deterministic): re-detect via a
   second `PollHub` and delete the stale "unordered LIMIT 1 … non-deterministic" comment — closes the
   retargeted issue.
3. **`normal` backlog**: "frozen hubs still advance" (ADR-0006 evidence-only); `AcceptCheckpoint`
   context reuse; tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store method;
   `CheckConsistency` collapse.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 verify-for-me / dashboard / log browser → WASM → OTS** remain the bulk of the v1 work.
