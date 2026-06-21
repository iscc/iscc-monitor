<!-- assessed-at: f6a5d7261697a804f0e77da3d94fbe1195d8cc9d -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — cross-cutting HTTP plumbing on the tlog-tiles mirror is now
complete (CORS → Cache-Control → conditional GET). verify-for-me, the server-rendered dashboard, and
the log browser remain open. WASM verifier + OTS anchoring not started.

The monitor follows + mirrors + verifies hubs (M1 met) and serves all three computed proofs per hub
from the local mirror (M2 met: `/inclusion`, `/consistency`, `/entries`). M3 has advanced through three
cross-cutting HTTP slices on the raw tlog-tiles mirror — CORS, then per-route `Cache-Control`, now
strong-ETag + `If-None-Match`→`304` conditional GET. The bulk of M3 (conditional-GET / cache policy on
the size-varying proof surfaces, verify-for-me, dashboard, log browser), the WASM verifier, and OTS
anchoring remain unstarted.

Incremental review of `0ef8acf..HEAD` (HEAD `f6a5d72` on `develop`). Three commits: define-next /
advance / review of the conditional-GET slice. The diff touches **one production file** —
`internal/tilesserve/handler.go` — plus its `_test.go` and context docs. **No `go.mod`, `go.sum`, or
`schema.sql` change** (verified `git diff --quiet 0ef8acf..HEAD -- go.mod go.sum
internal/store/schema.sql` exit 0). The change adds, inside `writeBlob`, a strong content ETag
(`fmt.Sprintf("\"%x\"", sha256.Sum256(data))`, no `W/` prefix) to every tlog-tiles 200 plus an
`If-None-Match` short-circuit (`inm == "*" || inm == etag` → `304` Not Modified, empty body, ETag +
Cache-Control still echoed per RFC 7232 §4.1). `writeBlob` now takes `r *http.Request`; all three
callers (`serveCheckpoint`/`serveTile`/`serveEntries`) were updated. Verified from source: ETag and
both cache headers are set before the conditional branch (the 200/304 freezes the header map on first
write); the discriminant `width == 0` immutable plumbing from the prior slice is unchanged. Latest
`review` handoff (2026-06-21, "Conditional GET … on the tlog-tiles mirror") is **PASS / CONTINUE**:
`mise run check` green (15 packages `ok`, `gofmt -l .` empty), the load-bearing re-GET→304 assertion
and full-tile strong-ETag value reviewer-re-derived (`sha256(0x11*8192)`), no header conflict with the
`corsmw` `Access-Control-*` wrap, oracle gate correctly N/A (pure HTTP header wiring on opaque BLOBs —
no signature/RFC-6962/Merkle/did:web/fsck path), gate-integrity scan clean (no `//nolint`/`t.Skip`/
build-tag/swallowed-error across the 3 commits), scope-clean (1 production file). **CI green at HEAD
`f6a5d72`** (develop run 27897191153, `conclusion: success`).

**Branch note:** active work happens on `develop` (HEAD `f6a5d72`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice only added an ETag + conditional-GET branch to
the already-served mirror; no M1 verification logic touched). All Verify criteria satisfied:
`origin`/`vkey` golden, all three triggers golden-tested end-to-end with freeze + alert-once + restart
survival, coverage tracked, structured logs, `/metrics` served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **205 `func Test`** across `cmd/` + `internal/`, **47** `_test.go` files
  (net **+1** test function vs the prior assessment — the slice added `TestHandlerConditionalGet` to
  `internal/tilesserve/handler_test.go`; no net new test file).
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
  `cmd/notecheck`), stdlib `slog`/`net/http`/`os`/`crypto/sha256` (now also for the mirror ETag). **Not
  yet wired**: `nbd-wtf/opentimestamps` (re-verified: grep → no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — carried forward unchanged. Both original Verify criteria are exercised (fsck
root-rebuild WIRED on every verified poll; inclusion cross-check conformance-tested over the real
verified mirror), and the served proof surface is **complete**: all three computed proofs —
`inclusion`, `consistency`, `entries` — served from the local mirror, never re-hitting the hub. This
slice only added an ETag + conditional-GET branch to the raw-mirror read surface; no proof/mirror logic
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
  BLOBs verbatim, per hub via `hubHandler`'s `/` fall-through, with a per-route `Cache-Control` policy
  (immutable full vs no-cache partial/checkpoint) and now a strong content ETag + `If-None-Match`→`304`
  conditional GET on every 200.
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
**Status**: **in progress**. Three cross-cutting HTTP slices have landed on the raw tlog-tiles mirror,
completing its cache/validation plumbing: (1) CORS applied uniformly to every public GET via the single
`corsmw.Handler` wrap in `buildMux` (`Access-Control-Allow-Origin: *`; `OPTIONS` → `204`); (2) per-route
`Cache-Control` (immutable full, no-cache partial/checkpoint); (3) a strong content ETag + wildcard/
exact `If-None-Match`→`304` conditional GET on every 200. The binary's mux serves `/metrics`,
`/healthz`, the per-hub raw tlog-tiles mirror subtrees, and the per-hub `/inclusion` + `/consistency` +
`/entries` computed-proof endpoints — all behind CORS. **Still absent (verified):** conditional-GET and
cache policy on the size-dependent proof surfaces (`/inclusion`/`/consistency`/`/entries`, tied to
`LastSize` — `internal/proofserve` carries no ETag/Cache-Control); `verify-for-me` verdict surface (no
`html/template` or `text/template` import anywhere); server-rendered dashboard (status/coverage/lag/
violations/OTS); `/` landing; and the log browser.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js`
in source). This slice's code change (`tilesserve/handler.go`) imports only `crypto/sha256`/`errors`/
`fmt`/`net/http`/`os`/`strings`/`tessera/api/layout`/`internal/store`; the WASM-shared verifier seam
continues to ride on `internal/didweb` (untouched).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (one production file
  changed, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for
  HEAD `f6a5d72`: `conclusion: success`** (run 27897191153).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (15 packages
  `ok`, `go vet`/`gofmt -l .` clean), the re-GET→304 and strong-ETag-value assertions reviewer-
  re-derived non-vacuous, no `corsmw`/`Cache-Control`/ETag header conflict, oracle gate correctly N/A
  (pure HTTP header wiring), trust root untouched + green (both golden vkeys reproduce), gate-integrity
  scan clean across the 3 commits, scope-clean (1 production file).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `TestPollHubFork` re-detection still bypasses `PollHub` (store `CheckpointAt` root cause RESOLVED, but
  the follower test + a stale comment remain); `AcceptCheckpoint` discards resolved context → verified
  polls re-fetch did.json; "frozen hubs still advance accepted state on later clean polls" (ADR-0006);
  tile writers require `width` (duplicated `p`-translation in follower); accepted-checkpoint advancement
  is three caller-sequenced store writes (locality); self-consistency policy split across follower +
  logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M2 met; M3 in progress — the tlog-tiles mirror HTTP-plumbing arc (CORS → Cache-Control → conditional
GET) is now complete. Shift to either draining the ADR-0006 trust-path backlog or beginning the
verify-for-me / proof-surface cache arc.** No `critical` open, so feature work proceeds; the 6 open
`normal` issues block DONE and are weighed against the state→target gap.

Candidate order:
1. **Drain a `normal` ADR-0006 issue** (highest-value gap, re-arms the consistency goldens): the
   frozen-hub evidence-only short-circuit that stops `AdvanceFollowState` advancing accepted state when
   `fs.Frozen`; the `TestPollHubFork` re-detection-via-second-`PollHub` cleanup (quick win now that
   `CheckpointAt` is deterministic — delete the stale "unordered LIMIT 1 … non-deterministic" comment);
   collapsing the self-consistency decision into a pure `logclient.CheckConsistency`.
2. **Proof-surface cache/conditional-GET** — extend ETag/`Cache-Control` to the size-varying
   `/inclusion`/`/consistency`/`/entries` (`internal/proofserve`), tied to `LastSize`, needing a
   derived/weak validator.
3. **`normal` backlog remainder**: `AcceptCheckpoint` context reuse; tile-writer `p`-vocabulary
   unification; deep `AdvanceAccepted` store method.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 verify-for-me / dashboard / log browser → WASM → OTS** remain the bulk of the v1 work.
