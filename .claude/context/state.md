<!-- assessed-at: ae5238bc6faee23fc7bd0345cc0288cc4f756bc6 -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 (Aggregator) — raw tlog-tiles mirror now SERVED over HTTP, wired into the binary

Both M2 Verify criteria are exercised (fsck root-rebuild WIRED on every verified poll; inclusion
cross-check conformance-tested over the real verified mirror). This slice **wired the already-built
`tilesserve.Handler` into the monitor binary**: each followed hub's mirrored tlog-tiles artifacts are now
served at its canonical `/<origin>/...` prefix on the same single HTTP server as `/metrics`. What remains
for M2's Verify bar is **serving computed `inclusion`/`consistency`/`entries` proofs** from the local
store (the served surface today is raw mirror BLOBs only). M3 / WASM / OTS not started.

Incremental review of `52174271..HEAD` (HEAD `ae5238b` on `develop`). The diff touches **exactly one
production file** — `cmd/iscc-monitor/main.go` (+97/-) — plus `main_test.go` and context files. **No
`internal/` package, `go.mod`, `go.sum`, or `schema.sql` change** (verified byte-unchanged in the diff).
`mirrorHandler(st, routes)` mounts a `tilesserve.Handler` per hub at `"/" + Origin + "/"` over a read-only
`store.SQLiteFetcher`, on the same mux as `/metrics`; routes are index-aligned with the poll targets,
derived from the same `registerHubs` upsert, so the poll set and served mirror set never diverge. Latest
`review` handoff (2026-06-21, "Wire tilesserve.Handler into the binary with a per-hub mirror router") is
**PASS / CONTINUE**: `mise run check` green (12 packages `ok`), routing mutation-proven non-vacuous
(dropping the trailing-slash subtree match collapses the 200-byte-equal test to 404), store leaf-purity
reconfirmed (`go list -deps ./internal/store` → no `tilesserve`/`net/http`), oracle gate correctly N/A
(opaque BLOB transport, no crypto path), WASM purity untouched. **CI green at HEAD `ae5238b`** (develop
run 27895072865, `success`).

**Branch note:** active work happens on `develop` (HEAD `ae5238b`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice only wired an existing handler into `main.go`; no
M1 verification logic touched). All Verify criteria satisfied: `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals at HEAD**: **171 `func Test`** across `cmd/` + `internal/`, **41** `_test.go` files (+1
  func over the prior 170 — the new `TestMirrorRouter` + 4 subtests in `cmd/iscc-monitor/main_test.go`;
  file count unchanged, `main_test.go` already existed).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,logclient,
  metrics,metricshttp,registry,store,tiles,tilesserve}`. Module path `github.com/iscc/iscc-monitor`,
  `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). No regression.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics` served + wired, now alongside the per-hub mirror routes on the same mux
  (`buildMux`/`mirrorHandler`, `cmd/iscc-monitor/main.go`), `internal/metrics` leaf, `slog` structured
  logging, `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking (set-once,
  ADR-0001), `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser
  (domains-only), poll-loop cadence (single-writer), freeze + alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`,
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays
  a leaf — re-verified at HEAD: `go list -deps ./internal/store` shows no `internal/tilesserve`, no
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
**Status**: partially met — **both Verify criteria exercised** (fsck root-rebuild WIRED on every poll;
inclusion cross-check conformance-tested over the real verified mirror). The inbound HTTP transport is now
**served**: this slice wired the raw tlog-tiles read surface into the binary; **computed proof-serving
remains** the outstanding M2 work item.
- **fsck root-rebuild (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after
  `ingestTiles`, before `recordVerdict`) builds a read-only `store.SQLiteFetcher` over the just-ingested
  tiles and calls `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash tiles,
  comparing the rebuilt RFC-6962 root to the signed checkpoint root. A mismatch is a mirror fault (no
  freeze). Mutation-proven. Carried forward (untouched this slice).
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` —
  `ingestEntryBundles` calls `projectEntryBundle` after each `RecordEntryBundle`, folding the raw bundle
  via `logclient.BundleProjections`, copying each `Projection` → `store.ProjectionRecord`, and
  `RecordProjections`. A decode/store fault aborts the poll before accepted state advances and never
  freezes the hub (ADR-0008 + ADR-0006). Exercised on real polls (`TestPollHubRecordsProjections`,
  300-leaf mirror crossing the 256-leaf bundle boundary). Carried forward (untouched this slice).
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED, deliberately NOT wired into `PollHub`):**
  `internal/logclient/inclusioncheck.go` — `ParseInclusionEvidence` / `VerifyInclusionEvidence`. Closed
  against M2's Verify bar by `internal/follower/inclusion_test.go` (`TestPollHubInclusion`): a verified
  `PollHub` mirror → `SeqsForISCCID` → hub-built proof from `testonly.Tree` → `VerifyInclusionEvidence`
  over the `SQLiteFetcher` tiles, byte-matched for leaves 5 + 260, with wrong-leaf + corrupted-proof
  negatives. No production `PollHub` caller (correct: there is no inbound hub-evidence transport on the
  follow path yet; a self-checking caller would be circular). Carried forward (untouched this slice).
- **raw tlog-tiles HTTP read surface (BUILT + now WIRED into the binary):** `internal/tilesserve/
  handler.go` — `Handler(f store.SQLiteFetcher) http.Handler` routes the three canonical iscc-log §9 paths
  (`GET /checkpoint`, `GET /tile/<L>/<index...>`, `GET /tile/entries/<index>` incl. `.p/<W>` partial
  suffix) to `ReadCheckpoint`/`ReadTile`/`ReadEntryBundle`, serving raw mirror BLOBs verbatim. Correct 400
  (malformed)/404 (not mirrored)/405 (non-GET)/500 mapping; `tile/entries/` matched before `tile/`.
  **Now served by the binary** via `mirrorHandler` (`cmd/iscc-monitor/main.go`): one `tilesserve.Handler`
  per hub mounted at `"/" + Origin + "/"` (e.g. `/sb0.iscc.id/log/`) on the same single HTTP server as
  `/metrics`, each over a read-only `store.SQLiteFetcher` sharing the store's single connection. Routing
  mutation-proven (trailing-slash subtree match) and asserted in `TestMirrorRouter`.

**What remains for M2's Verify bar:**
1. **Serve `inclusion`/`consistency`/`entries` as computed proofs** from the local store via the existing
   `logclient.ProofBuilder` / `ConsistencyProofFromTiles` / `InclusionProofFromTiles`, never re-hitting
   the hub — **not started** (the served `tilesserve` surface returns raw static mirror BLOBs only, not
   computed proofs). This also gives the inclusion cross-check its natural *inbound* hub-evidence
   transport / `verify-for-me` consumer, at which point a non-circular production caller becomes possible.
   Per the M3 split, this proof slice owns CORS, caching, conditional GET, and healthz.

## M3 — Trust API + dashboard
**Status**: not started. The binary's mux now serves `/metrics` plus the per-hub raw tlog-tiles mirror
subtrees; everything else is absent: no `/`, `/healthz`, computed-proof REST surface, `verify-for-me`,
server-rendered dashboard, log browser, or CORS. The raw-mirror handler that M3's log browser / canonical
tlog-tiles paths build on is now mounted.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js`,
`GOOS=js`/`GOARCH=wasm` in source). This slice does not touch the WASM-shareable purity invariant
(`tilesserve` wiring is opaque BLOB transport, no crypto path; `GOOS=js GOARCH=wasm go build
./internal/didweb` re-confirmed OK by review).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (one production file
  changed, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `ae5238b`:
  `conclusion: success`** (run 27895072865).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (12 packages
  `ok`, `go vet`/`gofmt -l .` clean), `TestMirror`'s 4 subtests passing uncached (200 byte-equal,
  unmirrored→404, missing-`/log`→404, `/metrics`→200), routing mutation-proven non-vacuous, store
  leaf-purity reconfirmed, scope discipline confirmed (1 production file, no `schema.sql`/dep change),
  oracle gate correctly N/A by dep-closure, WASM purity invariant green.
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `CheckpointAt` unordered `LIMIT 1` (fork re-detection compares an undefined row); `AcceptCheckpoint`
  discards resolved context → verified polls re-fetch did.json; "frozen hubs still advance accepted state
  on later clean polls" (ADR-0006); tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency
  policy split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s
  vestigial `out io.Writer` param.

## Next Milestone
**Complete M2 → serve computed proofs over HTTP.** No `critical` open, so feature work proceeds; the open
`normal` issues also block DONE but are weighed against the state→target gap. Both M2 Verify criteria are
exercised and the raw-mirror read surface is now served; the remaining M2 work is the computed proof
surface.

Candidate order:
1. **Serve `inclusion`/`consistency`/`entries` as computed proofs** from the local store via
   `logclient.ProofBuilder` / `ConsistencyProofFromTiles` / `InclusionProofFromTiles` /
   `VerifyInclusionEvidence` over the same `SQLiteFetcher`, never re-hitting the hub. Completes M2's
   Verify bar and gives the inclusion cross-check / `verify-for-me` an inbound consumer, enabling a
   non-circular production caller. Owns CORS, caching, conditional GET, healthz (per the M3 split).
   Opportunistically fix the `CheckpointAt` unordered-`LIMIT 1` issue if this slice revisits prior-root
   selection.
2. **`normal` backlog** (the proof-serving slice touches the verified path and the store — the natural
   moment to weigh these): "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt ORDER BY`
   fix; `AcceptCheckpoint` context reuse; tile-writer `p`-vocabulary unification; deep `AdvanceAccepted`
   store method; `CheckConsistency` collapse.
3. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
4. **M3 → WASM → OTS** remain after M2's Verify bar is fully met.
