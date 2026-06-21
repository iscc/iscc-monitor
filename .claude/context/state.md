<!-- assessed-at: 52174271ef8413ab0142ab73d9981651171d37f0 -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 (Aggregator) under construction — both Verify criteria exercised; building the inbound
HTTP transport. The `fsck` root-rebuild and the `iscc_index` projection run on every verified poll, and
the inclusion cross-check is conformance-tested over a real verified-poll mirror. This slice adds the
**raw tlog-tiles HTTP read surface** (`internal/tilesserve`) that serves the mirror BLOBs verbatim — a
clean, built-but-**unwired** package. What remains for M2's Verify bar is **serving
`inclusion`/`consistency`/`entries` proofs** over HTTP from the local store (and wiring the binary). M3 /
WASM / OTS not started.

Incremental review of `d7f3e0a..HEAD` (HEAD `52174271` on `develop`). The source diff touches **exactly
one new package** — `internal/tilesserve/{handler.go (+133), handler_test.go (+185)}` — plus context
files. **Zero existing-production lines changed; no `go.mod`/`go.sum`/`schema.sql` change** (verified: not
in the diff stat). `tilesserve.Handler(f store.SQLiteFetcher) http.Handler` routes the three canonical
tlog-tiles paths (`checkpoint`, `tile/entries/...`, `tile/...`) to `ReadCheckpoint`/`ReadEntryBundle`/
`ReadTile` and serves the raw mirror BLOBs, with `entries`-before-`tile` ordering and 400/404/405/500
mapping. Latest `review` handoff (2026-06-21, "Serve the raw tlog-tiles mirror … over HTTP from one hub's
SQLiteFetcher") is **PASS / CONTINUE**: `mise run check` green (12 packages `ok`), body-equality + status
mutation-proven non-vacuous, oracle gate correctly N/A (opaque BLOB transport — `go list -deps
./internal/tilesserve` pulls in no `merkle`/`note`/`logclient`/`proof`), store leaf-purity reconfirmed,
WASM purity untouched. **CI green at HEAD `52174271`** (develop run 27894822975, `success`).

**Branch note:** active work happens on `develop` (HEAD `52174271`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice added a standalone `tilesserve` package; no M1
verification logic touched). All Verify criteria satisfied: `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals re-grepped at HEAD**: **170 `func Test`** across `cmd/` + `internal/`, **41** `_test.go`
  files (+1 func, +1 file over the prior 169/40 — the new `internal/tilesserve/handler_test.go`, whose
  single `TestHandlerServesSeededBytes` carries 9 sub-cases).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,logclient,
  metrics,metricshttp,registry,store,tiles,tilesserve}`. Module path `github.com/iscc/iscc-monitor`,
  `go 1.24.0` (no `toolchain` line).
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
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays
  a leaf — re-verified at HEAD: `go list -deps ./internal/store` shows no `internal/tilesserve`, no
  `net/http`, no `internal/logclient`. `schema.sql` byte-unchanged this slice.
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
**under construction**: this slice adds the raw tlog-tiles read surface; **proof-serving + binary wiring
remain** the outstanding M2 work items.
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
- **raw tlog-tiles HTTP read surface (BUILT this slice, NOT yet wired into the binary):**
  `internal/tilesserve/handler.go` — `Handler(f store.SQLiteFetcher) http.Handler` routes the three
  canonical iscc-log §9 paths (`GET /checkpoint`, `GET /tile/<L>/<index...>`, `GET /tile/entries/<index>`
  incl. `.p/<W>` partial suffix) to `ReadCheckpoint`/`ReadTile`/`ReadEntryBundle`, serving raw mirror BLOBs
  verbatim. Correct 400 (malformed path)/404 (not mirrored)/405 (non-GET)/500 mapping; `tile/entries/`
  matched before `tile/`. **No production caller** — `grep tilesserve cmd/` is empty; the only route in the
  binary remains `/metrics`. Binary wiring (per-hub routing, hub→origin resolution) is the next slice.

**What remains for M2's Verify bar:**
1. **Serve `inclusion`/`consistency`/`entries`** from the local store via the existing
   `logclient.ProofBuilder` / `ConsistencyProofFromTiles`, never re-hitting the hub — **not started** (the
   `tilesserve` surface added here serves raw static mirror files only, not computed proofs). This also
   gives the inclusion cross-check its natural *inbound* hub-evidence transport / `verify-for-me` surface,
   at which point a non-circular production caller becomes possible.
2. **Wire `tilesserve.Handler` into `cmd/iscc-monitor/main.go`** — a per-hub route prefix + hub→origin
   router resolving the request's `HubID` + a read-only connection. Built but unserved today.

## M3 — Trust API + dashboard
**Status**: not started. The `net/http` mux in the binary serves only `/metrics` (verified: sole
`mux.Handle` is `metricshttp.Handler` at `cmd/iscc-monitor/main.go:108`; `tilesserve` is not referenced in
`cmd/`). No `/`, `/healthz`, REST surface, `verify-for-me`, dashboard, log browser, or *served* raw
tlog-tiles mirror — though `internal/tilesserve` now provides the raw-mirror handler M3 will mount.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js`,
`GOOS=js`/`GOARCH=wasm` in source). This slice does not touch the WASM-shareable purity invariant
(`tilesserve` is an opaque BLOB transport over the store, no crypto path).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (one additive
  package, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for HEAD `52174271`:
  `conclusion: success`** (run 27894822975).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (12 packages
  `ok`, `go vet`/`gofmt -l .` clean), the 9 handler sub-cases passing, body+status mutation-proven
  non-vacuous (constant-byte + collapsed-404 mutations both fail), store leaf-purity reconfirmed
  (`tilesserve → store`, never the reverse), scope discipline confirmed (exactly 2 new files, 0
  existing-production lines, no `schema.sql`/dep change), oracle gate correctly N/A by dep-closure, WASM
  purity invariant green (`GOOS=js GOARCH=wasm go build ./internal/didweb`).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `CheckpointAt` unordered `LIMIT 1` (fork re-detection compares an undefined row); `AcceptCheckpoint`
  discards resolved context → verified polls re-fetch did.json; "frozen hubs still advance accepted state
  on later clean polls" (ADR-0006); tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency
  policy split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s
  vestigial `out io.Writer` param.

## Next Milestone
**Complete M2 → serve proofs over HTTP + wire the binary.** No `critical` open, so feature work proceeds;
the open `normal` issues also block DONE but are weighed against the state→target gap. Both M2 Verify
criteria are exercised and the raw-mirror read surface is built; the remaining M2 work is the computed
proof surface and binary wiring.

Candidate order:
1. **Wire `tilesserve.Handler` into `cmd/iscc-monitor/main.go`** — per-hub route prefix + hub→origin
   router + read-only connection. Built-but-unserved today; the natural inbound-transport foundation.
2. **Serve `inclusion`/`consistency`/`entries`** from the local store via `logclient.ProofBuilder` /
   `ConsistencyProofFromTiles`, never re-hitting the hub. Completes M2's Verify bar and gives the
   inclusion cross-check / `verify-for-me` an inbound consumer, enabling a non-circular production caller.
3. **`normal` backlog** (the wiring + proof-serving slices touch the verified path and the store — the
   natural moment to weigh these): "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt
   ORDER BY` fix; `AcceptCheckpoint` context reuse; tile-writer `p`-vocabulary unification; deep
   `AdvanceAccepted` store method; `CheckConsistency` collapse.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 → WASM → OTS** remain after M2's Verify bar is fully met.
