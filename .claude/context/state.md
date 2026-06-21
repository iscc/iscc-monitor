<!-- assessed-at: 7f01cd5e7d387ee661d295ba30d524e2845b0221 -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 (Aggregator) — computed inclusion proofs now SERVED over HTTP from the local mirror

The first computed-proof endpoint is live: `GET /inclusion?iscc_id=<id>[&index=<n>]` is served per hub,
building an RFC-6962 inclusion proof from that hub's mirrored tiles (never re-hitting the hub) against the
monitor's accepted tree size. This closes the *inclusion* half of M2's computed-proof surface; the
`consistency` and `entries` proof endpoints (plus CORS / caching / conditional-GET / healthz) remain. M3 /
WASM / OTS not started.

Incremental review of `ae5238b..HEAD` (HEAD `7f01cd5` on `develop`). The diff touches **two production
files** — new `internal/proofserve/handler.go` (+201) and `cmd/iscc-monitor/main.go` (+43/-) — plus
`internal/proofserve/handler_test.go`, `cmd/iscc-monitor/main_test.go`, and context files. **No
`internal/store`, `internal/logclient`, `go.mod`, `go.sum`, or `schema.sql` change** (verified
byte-unchanged in the diff). `proofserve.Handler(st, hubID)` resolves the leaf seq via `SeqsForISCCID`
(schema-agnostic, ADR-0008), builds the proof via `logclient.InclusionProofFromTiles` over a read-only
`store.SQLiteFetcher`, and returns JSON shaped like the hub's `IsccLogInclusionProof`. It is mounted per
hub behind a nested `*http.ServeMux` (`hubHandler`: `/inclusion` → proofserve, `/` → tilesserve) next to
the existing static mirror routes and `/metrics` on the single HTTP server. Latest `review` handoff
(2026-06-21, "Serve computed inclusion proofs over HTTP from the local mirror") is **PASS / CONTINUE**:
`mise run check` green (13 packages `ok`), oracle gate (RFC-6962 inclusion crypto) reviewer-mutation-proven
non-vacuous (serve `proof=nil` and build-for-`leafIndex+1` both fail the served-proof verification test),
nested-mux routing verified, store/logclient leaf-purity reconfirmed, WASM purity untouched. **CI green at
HEAD `7f01cd5`** (develop run 27895477324, `success`).

**Branch note:** active work happens on `develop` (HEAD `7f01cd5`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice added a leaf proof package and a per-hub mux mount;
no M1 verification logic touched). All Verify criteria satisfied: `origin`/`vkey` golden, all three
triggers golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals at HEAD**: **178 `func Test`** across `cmd/` + `internal/`, **42** `_test.go` files (+7
  funcs / +1 file over the prior 171/41 — the new `internal/proofserve/handler_test.go` plus added
  `cmd/iscc-monitor/main_test.go` cases).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,logclient,
  metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}`. Module path
  `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). No regression.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics` served + wired, alongside the per-hub mirror + `/inclusion` routes on the same mux
  (`buildMux`/`mirrorHandler`/`hubHandler`, `cmd/iscc-monitor/main.go`), `internal/metrics` leaf, `slog`
  structured logging, `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking
  (set-once, ADR-0001), `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser
  (domains-only), poll-loop cadence (single-writer), freeze + alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`,
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays
  a leaf — re-verified at HEAD: `go list -deps ./internal/store` shows no `internal/proofserve`, no
  `net/http`. `schema.sql` byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles**. All tile/fsck/inclusion/projection/serve
  tests run against in-process `testonly.Tree` / `buildVerifiedMirror` / seeded-`SQLiteFetcher` fixtures.
  Real tile/entry-bundle + `IsccLogInclusionProof` fixtures remain a soft prerequisite for an *inbound*
  hub-evidence transport. Known stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests;
  fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`,
  both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib
  `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (re-verified: grep → no hits in
  `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: partially met — both prior Verify criteria exercised (fsck root-rebuild WIRED on every poll;
inclusion cross-check conformance-tested over the real verified mirror), and the computed inclusion-proof
endpoint is now **served**. **Remaining**: serve `consistency` and `entries` as computed proofs.
- **fsck root-rebuild (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after
  `ingestTiles`, before `recordVerdict`) builds a read-only `store.SQLiteFetcher` over the just-ingested
  tiles and calls `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash tiles,
  comparing the rebuilt RFC-6962 root to the signed checkpoint root. A mismatch is a mirror fault (no
  freeze). Mutation-proven. Carried forward (untouched this slice).
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` —
  `ingestEntryBundles` calls `projectEntryBundle` after each `RecordEntryBundle`, folding the raw bundle
  via `logclient.BundleProjections` → `store.ProjectionRecord` → `RecordProjections`. A decode/store fault
  aborts the poll before accepted state advances and never freezes the hub (ADR-0008 + ADR-0006). Exercised
  on real polls (300-leaf mirror crossing the 256-leaf bundle boundary). Carried forward (untouched).
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED):** `internal/logclient/inclusioncheck.go` —
  `ParseInclusionEvidence` / `VerifyInclusionEvidence`. Closed against M2's Verify bar by
  `internal/follower/inclusion_test.go` (verified `PollHub` mirror → hub-built proof from `testonly.Tree` →
  `VerifyInclusionEvidence` over the `SQLiteFetcher` tiles, byte-matched, with negatives). Carried forward.
- **raw tlog-tiles HTTP read surface (BUILT + WIRED):** `internal/tilesserve/handler.go` routes the three
  canonical iscc-log §9 paths (`GET /checkpoint`, `GET /tile/<L>/<index...>`, `GET /tile/entries/<index>`
  incl. `.p/<W>` partial suffix) to raw mirror BLOBs verbatim, mounted per hub via `hubHandler`'s `/`
  fall-through. Carried forward.
- **computed inclusion proof (BUILT + WIRED — this slice):** `internal/proofserve/handler.go` serves `GET
  /inclusion?iscc_id=<id>[&index=<n>]`: `FollowState.LastSize` (accepted size) gate → `SeqsForISCCID`
  (schema-agnostic seq resolution, `seqs[0]` deterministic default, `&index=` must equal a committed seq) →
  `leafIndex < size` guard → `InclusionProofFromTiles` over the mirror → JSON shaped like the hub's
  `IsccLogInclusionProof` (base64-Std hashes, `checkpoint` omitted; client refetches `/checkpoint`). Status
  mapping: non-GET 405, unmatched path 404, missing `iscc_id` 400, uncommitted `index` 400, no accepted
  checkpoint / unknown id / uncovered leaf 404, unmirrored tile (wrapped `os.ErrNotExist`) 404, else 500.
  Mounted per hub at `/inclusion` on a nested mux that takes precedence over the tilesserve subtree
  (`cmd/iscc-monitor/main.go` `hubHandler`). Oracle gate (RFC-6962 inclusion) reviewer-mutation-proven
  non-vacuous. Leaf package — `net/http` stays out of store/logclient (verified).

**What remains for M2's Verify bar:**
1. **Serve `consistency` and `entries` as computed proofs** from the local store via
   `ConsistencyProofFromTiles(smaller=prev, larger=LastSize)` and the entry-bundle read path, never
   re-hitting the hub — **not started**. Per the M3 split, the consistency slice owns CORS, caching,
   conditional GET, and healthz, and is the natural place to fix the open `CheckpointAt` unordered-`LIMIT 1`
   prior-root selection (issues.md).

## M3 — Trust API + dashboard
**Status**: not started. The binary's mux now serves `/metrics`, the per-hub raw tlog-tiles mirror
subtrees, and the per-hub `/inclusion` computed-proof endpoint; everything else is absent: no `/`,
`/healthz`, `consistency`/`entries` proof routes, `verify-for-me` verdict surface, server-rendered
dashboard, log browser, or CORS. The raw-mirror + inclusion handlers that M3's log browser / canonical
tlog-tiles paths build on are now mounted.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js`,
`GOOS=js`/`GOARCH=wasm` in source). This slice does not touch the WASM-shareable purity invariant
(`proofserve` is a `net/http` leaf depending on store+logclient; the WASM-shared verifier seam rides on
`internal/didweb`, untouched, `GOOS=js GOARCH=wasm go build ./internal/didweb` re-confirmed OK by review).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (two production files
  changed, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `7f01cd5`:
  `conclusion: success`** (run 27895477324).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (13 packages
  `ok`, `go vet`/`gofmt -l .` clean), served proof verified via `proof.VerifyInclusion` for leaves
  `{0,5,255,256,260,299}`, oracle gate reviewer-mutation-proven non-vacuous, dep direction confirmed
  (`proofserve → {store, logclient}`, never reverse; `net/http` out of those closures), scope-clean (2
  production files, no `schema.sql`/dep change), WASM purity invariant green.
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `CheckpointAt` unordered `LIMIT 1` (fork re-detection compares an undefined row); `AcceptCheckpoint`
  discards resolved context → verified polls re-fetch did.json; "frozen hubs still advance accepted state
  on later clean polls" (ADR-0006); tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency
  policy split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s
  vestigial `out io.Writer` param.

## Next Milestone
**Complete M2 → serve the remaining computed proofs (`consistency`, `entries`) over HTTP.** No `critical`
open, so feature work proceeds; the open `normal` issues also block DONE but are weighed against the
state→target gap. Inclusion is now served; the remaining M2 work is the consistency + entries proof
surface.

Candidate order:
1. **Serve `consistency` as a computed proof** — `GET /consistency?from=<n>` over the same per-hub nested
   mux, via `ConsistencyProofFromTiles(smaller=prev, larger=LastSize)`, never re-hitting the hub. Fix the
   open `CheckpointAt` unordered-`LIMIT 1` prior-root selection here (this slice revisits prior-root
   selection), and start owning the M3-deferred CORS / caching / conditional-GET / healthz.
2. **Serve `entries`** computed/served from the local mirror to complete the M2 proof surface.
3. **`normal` backlog** (the proof-serving slices touch the verified path and the store — the natural
   moment to weigh these): "frozen hubs still advance" (ADR-0006 evidence-only); `AcceptCheckpoint` context
   reuse; tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store method; `CheckConsistency`
   collapse.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 → WASM → OTS** remain after M2's Verify bar is fully met.
