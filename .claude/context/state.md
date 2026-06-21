<!-- assessed-at: fcea631dad459bfd7a27ae63dafd8f7acbbbe0a5 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green. M2 (Aggregator) under construction — the `iscc_index`
projection now has BOTH halves built at the unit level (the pure `BundleProjections` decoder + the
store-side `RecordProjections` writer / `SeqsForISCCID` reader) but neither is wired into `PollHub`. The
first half of M2's Verify bar (`fsck` root-rebuild) runs every verified poll; the second half (inclusion
cross-check) and the projection writer are built-but-unwired. M3 / WASM / OTS not started.

Incremental review of `71903e2..HEAD` (HEAD `fcea631`). The source diff touches **exactly one new
production file** — `internal/store/iscc_index.go` (+100) — plus its test `internal/store/iscc_index_test.go`
(+258) and context files. `git diff 71903e2..HEAD --stat -- go.mod go.sum internal/store/schema.sql` is
**empty** (no dep/schema change — the writer's `iscc_id_str`/`record_sha256` columns pre-existed). This
slice adds the store half of the ADR-0008 `iscc_index` projection: `RecordProjections` (idempotent
per-row `ON CONFLICT(seq) DO UPDATE` upsert) + `SeqsForISCCID` (one-to-many `iscc_id → []seq` reader over
the existing `iscc_index_by_iscc_id` BLOB index, hub-scoped, `ORDER BY seq`), with a store-owned
`ProjectionRecord` value struct so the follower copies `logclient.Projection → ProjectionRecord` at the
call site and the store stays a leaf. Latest `review` handoff (2026-06-21, "iscc_index store writer +
iscc_id → []seq read-back") is **PASS / CONTINUE**, mutation-proven (hub-scope filter + idempotent upsert
both fail when reverted), oracle correctly N/A (plain CRUD + BLOB round-trip, no crypto path).
**CI green at HEAD `fcea631`** (develop run 27893952177, `success`).

**Branch note:** active work happens on `develop` (HEAD `fcea631`, tree clean); a human merges
`develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward unchanged (this slice touched only `internal/store/iscc_index.go`; no M1
production source changed). All Verify criteria satisfied: `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP and collected by the binary. **CI-gated & green.**

- **Test totals re-grepped at HEAD**: **167 `func Test`** across the production tree (`cmd/` + `internal/`),
  **39** `_test.go` files. +8 funcs and +1 file over the prior 159/38 — entirely the new
  `iscc_index_test.go` (8 store tests: round-trip, empty-slice, one-to-many, descending-input ordering,
  absent, idempotent, schema-agnostic, hub-scoped). No M1 regression.
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,logclient,metrics,
  metricshttp,registry,store,tiles}`. Module path `github.com/iscc/iscc-monitor`.
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). No regression here.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics` served + wired (sole `mux.Handle`, `cmd/iscc-monitor/main.go:108`), `internal/metrics` leaf,
  `slog` structured logging, `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking
  (set-once, ADR-0001), `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser
  (domains-only), poll-loop cadence (single-writer), freeze + alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`,
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a
  leaf — `go list` imports show only `internal/tiles` (a pure coordinate package); no `internal/logclient`,
  no `net/http`. `schema.sql` byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles**. All tile/fsck/inclusion/projection tests run
  against in-process `testonly.Tree` / `buildVerifiedMirror` fixtures. Real tile/entry-bundle +
  `IsccLogInclusionProof` fixtures remain a soft prerequisite for wiring the inclusion cross-check into
  `PollHub`. Known stale `sb1.amlet.id_did.json` drift (pre-rotation key `22b08f3e`; live signer
  `069d0f14`) captured in tests; fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`,
  both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib
  `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (grep → no hits in `cmd/`+`internal/`).

## M2 — Aggregator
**Status**: partially met — **the first half of the Verify bar is WIRED; the second half is BUILT but
unwired; the `iscc_index` projection now has BOTH unit halves built (decoder + store writer/reader) but is
still unwired into `PollHub`.**
- **First half (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after `ingestTiles`,
  before `recordVerdict`) builds a read-only `store.SQLiteFetcher` over the just-ingested tiles and calls
  `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash tiles, comparing the rebuilt
  RFC-6962 root to the signed checkpoint root. A mismatch is a mirror fault (no freeze). Mutation-proven.
- **Second half (BUILT, pure, NOT YET WIRED):** `internal/logclient/inclusioncheck.go` —
  `ParseInclusionEvidence` decodes a hub's `IsccLogInclusionProof` VC evidence, `VerifyInclusionEvidence`
  recomputes the proof via `InclusionProofFromTiles` and `bytes.Equal`-compares each hash. **Verified
  unwired** (grep): no non-test production caller.
- **`iscc_index` projection — BOTH unit halves now BUILT, still UNWIRED:**
  - *Decoder half* (prior slice): `internal/logclient/projection.go` — pure `BundleProjections(bundle,
    baseSeq) ([]Projection, error)` fold (ADR-0008): decodes an entry bundle into `{Seq, IsccID, NoteSchema,
    RecordSHA256}` per leaf, reading the committed `iscc_id` and verbatim inner `note.$schema`, interpreting
    nothing.
  - *Store half* (this slice): `internal/store/iscc_index.go` — `ProjectionRecord` value struct +
    `RecordProjections` (idempotent `ON CONFLICT(seq) DO UPDATE` batch upsert; `iscc_id` bound as both BLOB
    bytes and TEXT; empty id → empty BLOB+string, not NULL) + `SeqsForISCCID` (hub-scoped one-to-many
    `iscc_id → []seq` reader, `ORDER BY seq`, absent → nil/nil).
  - **Still missing**: any production caller. Verified — the only non-test references to `RecordProjections`/
    `SeqsForISCCID`/`BundleProjections`/`VerifyInclusionEvidence` are in their own defining files. The store
    writer persists nothing on real polls because `PollHub` never calls it.

**What remains for M2's Verify bar (all not-started at the wiring level):**
1. **Wire the `iscc_index` projection into `PollHub`** — decode each ingested entry bundle via
   `logclient.BundleProjections`, copy `Projection → ProjectionRecord` at the call site (store stays a leaf),
   `RecordProjections`. The decode + persist units exist; only the follower wiring is missing.
2. **Wire `VerifyInclusionEvidence` into `PollHub`** — resolve a sampled `iscc_id → leafIndex` via
   `SeqsForISCCID` + a sampled entry bundle, feed `VerifyInclusionEvidence` over the `SQLiteFetcher`. This
   re-arms the oracle/inclusion-cross-check gate (currently N/A for the pure CRUD slices) against the hub's
   own `IsccLogInclusionProof`. Until wired, M2's second Verify criterion is not exercised on real polls.
3. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`, never
   re-hitting the hub — not started.

## M3 — Trust API + dashboard
**Status**: not started. The `net/http` mux serves only `/metrics` (verified: sole `mux.Handle` is
`metricshttp.Handler` at `cmd/iscc-monitor/main.go:108`). No `/`, `/healthz`, REST surface, `verify-for-me`,
dashboard, log browser, or raw tlog-tiles mirror at canonical paths.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`); no `internal/proof` package exists; no WASM build target. This slice's
`internal/store/iscc_index.go` is `context`+`fmt` only — does not affect the WASM-shareable purity invariant,
which rides on `internal/logclient` per-file import discipline (the store is not a WASM target).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `71903e2` (this slice is one
  additive production file + its test, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  **Latest run on `develop` for HEAD `fcea631`: `conclusion: success`** (run 27893952177).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (all 11 packages
  `ok`, `go vet`/`gofmt -l .` clean), the new store surfaces mutation-proven non-vacuous (hub-scope filter +
  idempotent upsert both fail when reverted), store leaf-purity confirmed (`go list` imports show no
  `internal/logclient`/`net/http`), scope discipline confirmed (exactly 2 additive files, no `schema.sql`/dep
  change), and oracle correctly N/A (plain CRUD + BLOB round-trip, no crypto path).
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
are weighed against the state→target gap. With both `iscc_index` unit halves now built, the natural next
slice is the `PollHub` wiring that turns them into a running projection + re-arms the inclusion oracle gate.

Candidate order:
1. **Wire `iscc_index` into `PollHub`** — decode each ingested entry bundle via `BundleProjections`, copy to
   `ProjectionRecord` at the call site, `RecordProjections`. (Decode + persist units already landed.)
2. **Wire `VerifyInclusionEvidence` into `PollHub`** — resolve a sampled `iscc_id → leafIndex` via
   `SeqsForISCCID` + a sampled entry bundle over the `SQLiteFetcher`; completes M2's second Verify half and
   re-arms the inclusion cross-check oracle gate.
3. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`.
4. **`normal` backlog** (the `PollHub` wiring touches `follower`, the natural moment to weigh these):
   "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt ORDER BY` fix; `AcceptCheckpoint`
   context reuse; tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store method;
   `CheckConsistency` collapse.
5. **sb1 fixture refresh** (`22b08f3e`→`069d0f14`) and **real alert transport** (close M1's alert path).
6. **M3 → WASM → OTS** remain after M2's Verify bar is met.
