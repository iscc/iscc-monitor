<!-- assessed-at: d7f3e0aa3511ecb2eda0e3cd81baf5856ba807e9 -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 (Aggregator) under construction — both Verify criteria now exercised. The `fsck`
root-rebuild and the `iscc_index` projection both run on every verified poll; the inclusion cross-check
is now **conformance-tested over a real verified-poll mirror** (test-only, deliberately — there is no
inbound hub-evidence transport on the follow path yet). What remains for M2 is **proof-serving**
(`inclusion`/`consistency`/`entries` over HTTP from the local store). M3 / WASM / OTS not started.

Incremental review of `cbc58d5..HEAD` (HEAD `d7f3e0a` on `develop`). The source diff touches **exactly one
production-adjacent file** — the new test `internal/follower/inclusion_test.go` (+147) — plus context
files. **Zero production lines changed; no `go.mod`/`go.sum`/`schema.sql` change** (verified: not in the
diff stat). This slice closes M2's second Verify criterion ("computed inclusion proof matches the hub's
`evidence.IsccLogInclusionProof` for sampled `iscc_id`s") with a conformance test: it drives a verified
`PollHub` over the byte-accurate 300-leaf in-process mirror, resolves sampled leaves' `iscc_id → leafIndex`
via the production `SeqsForISCCID`, builds each leaf's hub-side `IsccLogInclusionProof` from the fixture
`testonly.Tree`, and asserts `logclient.VerifyInclusionEvidence` (recomputing over the mirrored
`SQLiteFetcher` tiles) byte-matches it — with wrong-leaf + corrupted-proof negatives. Latest `review`
handoff (2026-06-21, "Close M2's inclusion cross-check Verify bar with a conformance test over the real
mirror") is **PASS / CONTINUE**, mutation-proven non-vacuous (short-circuiting `VerifyInclusionEvidence`
to `return nil` makes both negatives fail), oracle gate re-run green (RFC-6962 inclusion crypto applies:
`logclient`/`follower`/`didweb` pass, `derive_vkey.py` golden vectors reproduce, `notecheck` accept/reject).
**CI green at HEAD `d7f3e0a`** (develop run 27894536536, `success`).

**Branch note:** active work happens on `develop` (HEAD `d7f3e0a`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice added one follower-package test file; no M1
verification logic changed). All Verify criteria satisfied: `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals re-grepped at HEAD**: **169 `func Test`** across `cmd/` + `internal/`, **40** `_test.go`
  files (+1 func, +1 file over the prior 168/39 — the new `internal/follower/inclusion_test.go`).
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
  leaf — review confirmed `go list -deps ./internal/store` shows no `internal/logclient`, no `net/http`.
  `schema.sql` byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles**. All tile/fsck/inclusion/projection tests run
  against in-process `testonly.Tree` / `buildVerifiedMirror` fixtures. Real tile/entry-bundle +
  `IsccLogInclusionProof` fixtures remain a soft prerequisite for an *inbound* hub-evidence transport. Known
  stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests; fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`,
  both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib
  `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (grep → no hits in `cmd/`+`internal/`).

## M2 — Aggregator
**Status**: partially met — **both Verify criteria now exercised** (fsck root-rebuild WIRED on every poll;
inclusion cross-check conformance-tested over the real verified mirror). **Proof-serving over HTTP not
started** — this is the remaining M2 work item.
- **fsck root-rebuild (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after `ingestTiles`,
  before `recordVerdict`) builds a read-only `store.SQLiteFetcher` over the just-ingested tiles and calls
  `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash tiles, comparing the rebuilt
  RFC-6962 root to the signed checkpoint root. A mismatch is a mirror fault (no freeze). Mutation-proven.
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` —
  `ingestEntryBundles` calls `projectEntryBundle` after each `RecordEntryBundle`, folding the raw bundle via
  `logclient.BundleProjections(raw, bundleIndex*tiles.TileWidth)`, copying each `Projection` →
  `store.ProjectionRecord`, and `RecordProjections`. A decode/store fault aborts the poll before accepted
  state advances and never freezes the hub (ADR-0008 + ADR-0006). Exercised on real polls
  (`TestPollHubRecordsProjections`, 300-leaf mirror crossing the 256-leaf bundle boundary).
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED, deliberately NOT wired into `PollHub`):**
  `internal/logclient/inclusioncheck.go` — `ParseInclusionEvidence` decodes a hub's `IsccLogInclusionProof`
  VC evidence; `VerifyInclusionEvidence` recomputes the proof via the tile fetcher and `bytes.Equal`-compares
  each hash (`ErrInclusionMismatch` on mismatch). **Now closed against M2's Verify bar** by
  `internal/follower/inclusion_test.go` (`TestPollHubInclusion`): a verified `PollHub` mirror →
  `SeqsForISCCID` → hub-built proof from `testonly.Tree` → `VerifyInclusionEvidence` over the
  `SQLiteFetcher` tiles, byte-matched for leaves 5 + 260, with wrong-leaf and corrupted-proof negatives.
  **No production `PollHub` caller** (verified by grep: the only non-test refs are inside its defining file
  + a forward-looking comment in `iscc_index.go`). This is correct: `VerifyInclusionEvidence` is the
  *consumer* of a hub-supplied proof, and there is no inbound hub-evidence transport on the follow path yet
  — a self-checking `PollHub` caller would be circular (forbidden by target.md). The conformance test is the
  honest way to meet the Verify bar today.

**What remains for M2's Verify bar:**
1. **Serve `inclusion`/`consistency`/`entries`** from the local store via the existing
   `logclient.ProofBuilder`, never re-hitting the hub — **not started** (no HTTP route beyond `/metrics`).
   This also gives the inclusion cross-check its natural *inbound* hub-evidence transport / `verify-for-me`
   surface (a `FetchInclusionEvidence` consumer), at which point a non-circular production caller becomes
   possible.

## M3 — Trust API + dashboard
**Status**: not started. The `net/http` mux serves only `/metrics` (verified: sole `mux.Handle` is
`metricshttp.Handler` at `cmd/iscc-monitor/main.go:108`). No `/`, `/healthz`, REST surface, `verify-for-me`,
dashboard, log browser, or raw tlog-tiles mirror at canonical paths.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`); no `internal/proof` package exists; no WASM build target (no `syscall/js`,
`GOOS=js`/`GOARCH=wasm` in source). This test-only slice does not touch the WASM-shareable purity invariant,
which rides on `internal/logclient` per-file import discipline.

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (one additive test
  file, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `d7f3e0a`:
  `conclusion: success`** (run 27894536536).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (11 packages
  `ok`, `go vet`/`gofmt -l .` clean), the conformance test mutation-proven non-vacuous (short-circuit →
  both negatives fail), store leaf-purity reconfirmed, scope discipline confirmed (exactly 1 test file, 0
  production lines, no `schema.sql`/dep change), oracle gate re-run green (RFC-6962 inclusion crypto applies),
  and WASM purity invariant green (`GOOS=js GOARCH=wasm go build ./internal/didweb`).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `CheckpointAt` unordered `LIMIT 1` (fork re-detection compares an undefined row); `AcceptCheckpoint`
  discards resolved context → verified polls re-fetch did.json; "frozen hubs still advance accepted state on
  later clean polls" (ADR-0006); tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency policy
  split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s vestigial
  `out io.Writer` param.

## Next Milestone
**Complete M2 → serve proofs over HTTP.** No `critical` open, so feature work proceeds; the open `normal`
issues also block DONE but are weighed against the state→target gap. With both M2 Verify criteria now
exercised, the remaining M2 work item is the HTTP proof surface.

Candidate order:
1. **Serve `inclusion`/`consistency`/`entries`** from the local store via `logclient.ProofBuilder`, never
   re-hitting the hub — the next M2 slice. This also gives the inclusion cross-check an *inbound*
   hub-evidence transport / `verify-for-me` surface, enabling a non-circular production caller later.
2. **`normal` backlog** (the HTTP proof-serving slice reworks/touches the verified path and the store — the
   natural moment to weigh these): "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt
   ORDER BY` fix; `AcceptCheckpoint` context reuse; tile-writer `p`-vocabulary unification; deep
   `AdvanceAccepted` store method; `CheckConsistency` collapse.
3. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
4. **M3 → WASM → OTS** remain after M2's Verify bar is fully met (proof-serving done).
