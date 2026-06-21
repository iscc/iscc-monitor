<!-- assessed-at: 1d328c7243dd62a8bc6dfcfce10e59d4d3764789 -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 (Aggregator) — proof surface COMPLETE (3-of-3 served); next is M3 cross-cutting HTTP plumbing / dashboard

The monitor follows + mirrors + verifies hubs (M1 met) and now serves **all three** computed-proof
endpoints per hub from the local mirror — `/inclusion`, `/consistency`, and (this slice) `/entries`
single-leaf record bytes — never re-hitting the hub. M2's Verify bar (fsck root-rebuild + inclusion
cross-check) was already met; this completes M2's served proof surface. M3 (dashboard / verify-for-me /
CORS / landing), WASM, and OTS remain entirely unstarted.

Incremental review of `0f0a2ec..HEAD` (HEAD `1d328c7` on `develop`). Three commits: define-next / advance /
review of the `/entries` slice. The diff touches **three production files** — `internal/logclient/entries.go`
(new leaf: `RecordBytesFromBundle(bundle, offset)` → raw record bytes via `api.EntryBundle.UnmarshalText` +
`Entries[offset]`, plus `ErrLeafOutOfBundle` sentinel; imports only `errors`+`fmt`+`tessera/api`, WASM-pure),
`internal/proofserve/handler.go` (+`serveEntries`, switch arm `/entries`), and `cmd/iscc-monitor/main.go`
(`hubHandler` mounts `/entries` as an exact path alongside `/inclusion` + `/consistency`). **No `go.mod`,
`go.sum`, or `schema.sql` change** (verified `git diff --quiet 0f0a2ec..HEAD -- go.mod go.sum schema.sql`
exit 0). Latest `review` handoff (2026-06-21, "Serve GET /entries") is **PASS / CONTINUE**: `mise run check`
green (14 packages `ok`), the wrong-leaf extractor mutation FAILED the golden across the bundle boundary
(then reverted), oracle gate correctly N/A (no signature/RFC-6962/Merkle/did:web/fsck path — bytes returned
verbatim, never re-hashed), trust-root conformance reconfirmed (both `derive_vkey.py` vectors + `notecheck`),
scope-clean (3 production files). **CI green at HEAD `1d328c7`** (develop run 27896457576, `success`).

**Branch note:** active work happens on `develop` (HEAD `1d328c7`, tree clean, in sync with `origin/develop`);
a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice added a single-leaf record-bytes proof route; no M1
verification logic touched). All Verify criteria satisfied: `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals at HEAD**: **203 `func Test`** across `cmd/` + `internal/`, **46** `_test.go` files (+12 funcs /
  +2 files over the prior 191/44 — `internal/logclient/entries_test.go`, `internal/proofserve/entries_test.go`,
  and `cmd/iscc-monitor/main_test.go`'s `TestMirrorEntriesRoute`).
- **Packages present** (unchanged): `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,healthz,
  logclient,metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}`. Module path
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). No regression.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics`, `/healthz`, the per-hub mirror, and the per-hub `/inclusion` + `/consistency` + `/entries` proof
  routes all ride one `*http.ServeMux` on a single listener (`buildMux`/`mirrorHandler`/`hubHandler`,
  `cmd/iscc-monitor/main.go`). `/metrics` + `/healthz` mount as exact paths; per-hub `/<domain>/log/` subtrees
  use the trailing-slash subtree match, with `/inclusion`, `/consistency`, `/entries` mounted as exact paths
  inside each hub's mux and `/` falling through to the raw tlog-tiles mirror. `internal/metrics` leaf, `slog`
  structured logging, `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking (set-once,
  ADR-0001), `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser (domains-only),
  poll-loop cadence (single-writer), freeze + alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
  `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a leaf; `schema.sql`
  byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles**. All tile/fsck/inclusion/consistency/entries/
  projection/serve tests run against in-process `testonly.Tree` / `buildVerifiedMirror` / seeded-`SQLiteFetcher`
  fixtures. Real tile/entry-bundle + `IsccLogInclusionProof` fixtures remain a soft prerequisite for an
  *inbound* hub-evidence transport. Known stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in
  tests; fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`, both
  proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`, `tessera/client`
  re-export, and now `api.EntryBundle.UnmarshalText` in `entries.go`), `transparency-dev/formats` (DIRECT —
  `cmd/notecheck`), stdlib `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (re-verified:
  grep → no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — both original Verify criteria exercised (fsck root-rebuild WIRED on every poll; inclusion
cross-check conformance-tested over the real verified mirror), and the served proof surface is now **complete**:
all three computed proofs — `inclusion`, `consistency`, `entries` — are served from the local mirror, never
re-hitting the hub.
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
  Carried forward.
- **computed entries record bytes (BUILT + WIRED — this slice):** `serveEntries` serves `GET /entries?index=
  <seq>` from the mirror. Core is the pure `logclient.RecordBytesFromBundle(bundle, offset)`
  (`api.EntryBundle.UnmarshalText` then `Entries[offset]`, never re-hashed/interpreted — schema-agnostic,
  ADR-0008), served as `application/octet-stream`. Guards mirror `serveInclusion`: 400 missing/non-numeric
  index; 404 `LastSize==0` / `seq>=LastSize` / bundle-not-mirrored (`os.ErrNotExist`) / partial-bundle-not-yet-
  covering (`ErrLeafOutOfBundle`); 500 otherwise. Uses `p := tiles.PartialTileSize(0, bundleIndex, size)`
  (NOT a literal `p==0`) so a partial final bundle resolves via the `SQLiteFetcher` fallback — reviewer-
  validated correctness fix over `next.md`'s literal note. Reviewer mutation (wrong-leaf extractor) FAILED the
  boundary golden.
- **`CheckpointAt` determinism (in place):** `internal/store/checkpoints.go` `CheckpointAt` is `… ORDER BY rowid
  LIMIT 1`, returning the lowest-rowid (prior accepted) root deterministically. Carried forward.

**What remains for M2:** nothing on the Verify bar — M2 is met. The natural next slice is the M3-deferred
cross-cutting HTTP plumbing (CORS / caching / conditional-GET headers) now that all three computed surfaces
plus the raw mirror exist.

## M3 — Trust API + dashboard
**Status**: not started. The binary's mux serves `/metrics`, `/healthz`, the per-hub raw tlog-tiles mirror
subtrees, and the per-hub `/inclusion` + `/consistency` + `/entries` computed-proof endpoints. Everything M3
requires is absent (verified): no `/` landing, no `verify-for-me` verdict surface, no server-rendered dashboard,
no log browser, **no CORS** (grep → no `Access-Control` header code anywhere in `cmd/`+`internal/`). The
raw-mirror + proof handlers that M3's log browser / canonical tlog-tiles paths build on are mounted.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js` in
source). This slice's new code (`entries.go`) is WASM-pure (imports only `errors`+`fmt`+`tessera/api`); the
WASM-shared verifier seam continues to ride on `internal/didweb` (untouched; review re-confirmed `GOOS=js
GOARCH=wasm go build ./internal/logclient` exits 0).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run check`
  runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (three production files changed, no
  dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `1d328c7`:
  `conclusion: success`** (run 27896457576).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (14 packages `ok`,
  `go vet`/`gofmt -l .` clean), `/entries` tested (200 + exact bytes across the 256-leaf boundary; 400/404/405),
  wrong-leaf mutation proven non-vacuous, oracle gate correctly N/A (no crypto path; bytes returned verbatim),
  trust root reconfirmed (`derive_vkey.py` + `notecheck`), WASM purity invariant green, scope-clean (3
  production files).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`: `TestPollHubFork`
  re-detection still bypasses `PollHub` (store `CheckpointAt` root cause RESOLVED, but the follower test + a
  stale comment remain); `AcceptCheckpoint` discards resolved context → verified polls re-fetch did.json;
  "frozen hubs still advance accepted state on later clean polls" (ADR-0006); tile writers require `width`
  (duplicated `p`-translation in follower); accepted-checkpoint advancement is three caller-sequenced store
  writes (locality); self-consistency policy split across follower + logclient (refactor). **`low`
  (loop-skipped):** `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M2 is met (proof surface 3-of-3 served + Verify bar closed). Next is M3 — start with the cross-cutting HTTP
plumbing, then the dashboard / verify-for-me arc.** No `critical` open, so feature work proceeds; the 6 open
`normal` issues block DONE and are weighed against the state→target gap.

Candidate order:
1. **M3 cross-cutting HTTP slice** — apply CORS + caching + conditional-GET (ETag/If-None-Match) headers
   uniformly across `/inclusion`, `/consistency`, `/entries`, and the static mirror, now that all three
   computed surfaces plus the raw mirror exist. (target.md M3: "CORS on every public GET".)
2. **`TestPollHubFork` cleanup** (quick win, now that `CheckpointAt` is deterministic): re-detect via a second
   `PollHub` and delete the stale "unordered LIMIT 1 … non-deterministic" comment — closes the retargeted issue.
3. **`normal` backlog**: "frozen hubs still advance" (ADR-0006 evidence-only); `AcceptCheckpoint` context reuse;
   tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store method; `CheckConsistency` collapse.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 dashboard / verify-for-me / log browser → WASM → OTS** remain the bulk of the v1 work after the HTTP
   plumbing lands.
