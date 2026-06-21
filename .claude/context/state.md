<!-- assessed-at: cbc58d5a126e19fe634526c9b1a35efa86a91f84 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green. M2 (Aggregator) under construction — the `iscc_index`
projection is now **fully wired into `PollHub`**: every verified, growing poll mirrors each entry bundle
and folds it into `iscc_index`. The first half of M2's Verify bar (`fsck` root-rebuild) runs every
verified poll; the projection writer is now live; the second half (inclusion cross-check) is built but
still unwired. Proof-serving (`inclusion`/`consistency`/`entries`) and M3 / WASM / OTS not started.

Incremental review of `fcea631..HEAD` (HEAD `cbc58d5` on `develop`). The source diff touches **exactly one
production file** — `internal/follower/ingest.go` (+43) — plus two test files (`ingest_test.go`,
`fsck_test.go`) and context files. `go.mod`/`go.sum`/`schema.sql` are byte-unchanged (verified: nothing in
the diff stat). This slice wires the two previously built-but-unwired `iscc_index` halves: a new
`projectEntryBundle` helper folds every mirrored entry bundle via `logclient.BundleProjections(raw,
c.Index*tiles.TileWidth)` and persists it via `store.RecordProjections`, called from `ingestEntryBundles`
right after `RecordEntryBundle`. `logclient.Projection` is copied field-by-field into
`store.ProjectionRecord` at the call site so the store stays a leaf. Latest `review` handoff (2026-06-21,
"Wire the iscc_index projection into PollHub's entry-bundle ingestion") is **PASS / CONTINUE**,
mutation-proven (correct `baseSeq` math + record-or-fail both fail when reverted), oracle re-run green
(fsck root-rebuild, golden vectors, `notecheck` accept/reject) since the slice touches the verified path.
**CI green at HEAD `cbc58d5`** (develop run 27894260332, `success`).

**Branch note:** active work happens on `develop` (HEAD `cbc58d5`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice touched only `internal/follower/ingest.go`'s
entry-bundle path; no M1 verification logic changed). All Verify criteria satisfied: `origin`/`vkey`
golden, all three triggers golden-tested end-to-end with freeze + alert-once + restart survival, coverage
tracked, structured logs, `/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals re-grepped at HEAD**: **168 `func Test`** across `cmd/` + `internal/`, **39** `_test.go`
  files. +1 func, same file count over the prior 167/39 — the new test is in the existing
  `internal/follower/ingest_test.go` (`TestPollHubRecordsProjections`, plus `fsck_test.go` fixture
  conversion). No M1 regression.
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,logclient,metrics,
  metricshttp,registry,store,tiles}`. Module path `github.com/iscc/iscc-monitor`, `go 1.24.0` (no
  `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). No regression.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics` served + wired (sole `mux.Handle`, `cmd/iscc-monitor/main.go:108`), `internal/metrics` leaf,
  `slog` structured logging, `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking
  (set-once, ADR-0001), `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser
  (domains-only), poll-loop cadence (single-writer), freeze + alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`,
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a
  leaf — review confirmed `go list` imports show no `internal/logclient`, no `net/http`. `schema.sql`
  byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles**. All tile/fsck/inclusion/projection tests run
  against in-process `testonly.Tree` / `buildVerifiedMirror` fixtures. Real tile/entry-bundle +
  `IsccLogInclusionProof` fixtures remain a soft prerequisite for wiring the inclusion cross-check into
  `PollHub`. Known stale `sb1.amlet.id_did.json` drift (pre-rotation key) captured in tests; fixtures not
  yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`,
  both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib
  `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (grep → no hits in `cmd/`+`internal/`).

## M2 — Aggregator
**Status**: partially met — **fsck root-rebuild WIRED, `iscc_index` projection now WIRED, inclusion
cross-check BUILT but unwired, proof-serving not started.**
- **fsck root-rebuild (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after `ingestTiles`,
  before `recordVerdict`) builds a read-only `store.SQLiteFetcher` over the just-ingested tiles and calls
  `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash tiles, comparing the rebuilt
  RFC-6962 root to the signed checkpoint root. A mismatch is a mirror fault (no freeze). Mutation-proven.
- **`iscc_index` projection (NOW WIRED, this slice):** `internal/follower/ingest.go:73-117` —
  `ingestEntryBundles` calls `projectEntryBundle` after each `RecordEntryBundle`, which folds the raw bundle
  via `logclient.BundleProjections(raw, bundleIndex*tiles.TileWidth)`, copies each `Projection` →
  `store.ProjectionRecord`, and `RecordProjections`. A verified `PollHub` now populates `iscc_index`; a
  decode/store fault aborts the poll before accepted state advances and never freezes the hub (ADR-0008 +
  ADR-0006). The decoder (`internal/logclient/projection.go` `BundleProjections`) and store
  writer/reader (`internal/store/iscc_index.go` `RecordProjections` + `SeqsForISCCID`) are both unit-tested
  and now exercised on real polls (`TestPollHubRecordsProjections`, 300-leaf mirror crossing the 256-leaf
  bundle boundary).
- **inclusion cross-check (BUILT, pure, NOT YET WIRED):** `internal/logclient/inclusioncheck.go` —
  `ParseInclusionEvidence` decodes a hub's `IsccLogInclusionProof` VC evidence, `VerifyInclusionEvidence`
  recomputes the proof via the tile fetcher and `bytes.Equal`-compares each hash. **Verified unwired**
  (grep): the only non-test references are inside its own defining file (and a forward-looking comment in
  `iscc_index.go`). No production caller in `PollHub`.

**What remains for M2's Verify bar:**
1. **Wire `VerifyInclusionEvidence` into `PollHub`** — resolve a sampled `iscc_id → leafIndex` via
   `SeqsForISCCID` + a sampled entry bundle, feed `VerifyInclusionEvidence` over the `SQLiteFetcher`. This
   re-arms the inclusion-cross-check oracle gate (currently N/A for the CRUD/wiring slices) against the
   hub's own `IsccLogInclusionProof`. Needs an `IsccLogInclusionProof` fixture (none captured yet). Until
   wired, M2's second Verify criterion is not exercised on real polls.
2. **Serve `inclusion`/`consistency`/`entries`** from the local store via the existing
   `logclient.ProofBuilder`, never re-hitting the hub — not started (no HTTP route beyond `/metrics`).

## M3 — Trust API + dashboard
**Status**: not started. The `net/http` mux serves only `/metrics` (verified: sole `mux.Handle` is
`metricshttp.Handler` at `cmd/iscc-monitor/main.go:108`). No `/`, `/healthz`, REST surface, `verify-for-me`,
dashboard, log browser, or raw tlog-tiles mirror at canonical paths.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`); no `internal/proof` package exists; no WASM build target (no `syscall/js`,
`GOOS=js`/`GOARCH=wasm` in source). This slice's change is confined to `internal/follower/ingest.go` and
does not touch the WASM-shareable purity invariant, which rides on `internal/logclient` per-file import
discipline.

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `fcea631` (this slice is
  one additive production file + two test files, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `cbc58d5`:
  `conclusion: success`** (run 27894260332).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (11 packages
  `ok`, `go vet`/`gofmt -l .` clean), the wiring mutation-proven non-vacuous (correct `baseSeq` math +
  record-or-fail both fail when reverted), store leaf-purity reconfirmed, scope discipline confirmed
  (exactly 1 production file, no `schema.sql`/dep change), oracle gate re-run green (fsck root-rebuild,
  `derive_vkey.py` golden vectors, `notecheck` accept/reject), and WASM purity invariant green.
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `CheckpointAt` unordered `LIMIT 1` (fork re-detection compares an undefined row); `AcceptCheckpoint`
  discards resolved context → verified polls re-fetch did.json; "frozen hubs still advance accepted state on
  later clean polls" (ADR-0006); tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency policy
  split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s vestigial
  `out io.Writer` param.

## Next Milestone
**Complete M2.** No `critical` open, so feature work proceeds; the open `normal` issues also block DONE but
are weighed against the state→target gap. With the `iscc_index` projection now wired and populating on every
verified poll, the natural next slice is wiring the inclusion cross-check — which deepens the verified path
and is the natural moment to weigh the open `normal` follower issues.

Candidate order:
1. **Wire `VerifyInclusionEvidence` into `PollHub`** — resolve a sampled `iscc_id → leafIndex` via
   `SeqsForISCCID` + a sampled entry bundle over the `SQLiteFetcher`; completes M2's second Verify half and
   re-arms the inclusion cross-check oracle gate. Needs an `IsccLogInclusionProof` fixture (none captured).
2. **Serve `inclusion`/`consistency`/`entries`** from the local store via `logclient.ProofBuilder`.
3. **`normal` backlog** (the inclusion-wiring slice reworks the verified path, the natural moment to weigh
   these): "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt ORDER BY` fix;
   `AcceptCheckpoint` context reuse; tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store
   method; `CheckConsistency` collapse.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 → WASM → OTS** remain after M2's Verify bar is met.
