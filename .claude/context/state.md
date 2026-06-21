<!-- assessed-at: 71903e2bbc9b2b96db2728e91f63b96a8afa7484 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green. M2 (Aggregator) under construction — its iscc_index
projection plumbing is growing but still unwired. The first half of M2's Verify bar (`fsck` root-rebuild)
runs every verified poll; the second half (inclusion cross-check) is built-but-unwired, and the
`iscc_index` projection writer / store-served proofs remain not-started. M3 / WASM / OTS not started.

Since the prior assessment (`4edec1d`) the source diff `4edec1d..HEAD` touches **only two new additive
files** — `internal/logclient/projection.go` (+86) and `internal/logclient/projection_test.go` (+139),
plus context files. This slice adds the pure, schema-agnostic `BundleProjections(bundle, baseSeq)
([]Projection, error)` decoder (ADR-0008): folds one tlog-tiles entry bundle into per-leaf
`{Seq, IsccID, NoteSchema, RecordSHA256}` records by reading the top-level `iscc_id` and the INNER
`note.$schema`, interpreting nothing. It is the first building block of the `iscc_index` projection writer.
`git diff 4edec1d..HEAD -- go.mod go.sum internal/store/schema.sql` is **empty** (no dep/schema change).
Latest `review` handoff (2026-06-21, "Pure entry-bundle → iscc_index projection decoder") is **PASS /
CONTINUE**, mutation-proven non-vacuous, WASM-shareable, oracle N/A. **CI green at HEAD `71903e2`**
(develop run 27893716778, `success`).

**Branch note:** active work happens on `develop` (HEAD `71903e2`, tree clean); a human merges
`develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP and collected by the binary). **CI-gated & green.**

- **Test totals re-grepped at HEAD**: **159 `func Test`** across the production tree (`cmd/` + `internal/`),
  **38** `_test.go` files (11 production packages). +5 funcs and +1 file over the prior 154/37 — entirely the
  new `projection_test.go` (M2 plumbing), no M1 regression.
- **Packages present** (verified `ls`): `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,
  logclient,metrics,metricshttp,registry,store,tiles}`. Module path `github.com/iscc/iscc-monitor`.
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). The prior `critical` (growing split view could advance before candidate tiles were mirrored)
  was CLOSED in the prior slice (`ingestTiles` moved ahead of `checkConsistency`); no regression here.
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics` served + wired (sole `mux.Handle`, `cmd/iscc-monitor/main.go:108`), `internal/metrics` leaf,
  `slog` structured logging, `SQLiteFetcher` + partial-tile mirror CRUD, hub_keys cache, coverage tracking
  (set-once, ADR-0001), `logclient.Origin` + golden `TestOrigin`, config loader, realm-registry parser
  (domains-only), poll-loop cadence (single-writer), freeze + alert-once.
- `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer
  discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table
  `schema.sql` (`hubs`, `hub_keys`, `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`,
  `follow_state`, `ots`). Store stays a leaf. Byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — still **no tiles or entry bundles** (verified `ls`). All tile/fsck/inclusion/
  projection tests run against in-process `testonly.Tree` / `buildVerifiedMirror` fixtures. Real
  tile/entry-bundle + `IsccLogInclusionProof` fixtures remain a soft prerequisite for wiring the inclusion
  cross-check into `PollHub`. Known stale `sb1.amlet.id_did.json` drift (pre-rotation key `22b08f3e`; live
  signer `069d0f14`) captured in tests; fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`,
  `api/layout`, both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib
  `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (grep → no hits in `cmd/`+`internal/`).

## M2 — Aggregator
**Status**: partially met — **the first half of the Verify bar is WIRED; the second half is BUILT but
unwired; the `iscc_index` projection writer is now PARTIALLY BUILT (decoder only) but unwired.**
- **First half (WIRED, runs every verified poll):** `fsckMirror` (`follower.go`, after `ingestTiles`,
  before `recordVerdict`) builds a read-only `store.SQLiteFetcher` over the just-ingested tiles and calls
  `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash tiles, comparing the rebuilt
  RFC-6962 root to the signed checkpoint root. A mismatch is a mirror fault (no freeze), not a
  self-consistency violation. Mutation-proven by review.
- **Second half (BUILT, pure, NOT YET WIRED):** `internal/logclient/inclusioncheck.go` —
  `ParseInclusionEvidence` decodes a hub's `IsccLogInclusionProof` VC evidence, `VerifyInclusionEvidence`
  recomputes the proof via `InclusionProofFromTiles` and `bytes.Equal`-compares each hash. **Verified
  unwired** (grep): the only non-test references to `iscc_index`/`BundleProjections`/`VerifyInclusionEvidence`
  are in `inclusioncheck.go` and `projection.go` themselves — no production caller.
- **`iscc_index` projection writer — PARTIALLY BUILT (this slice):** `internal/logclient/projection.go`
  adds the pure `BundleProjections(bundle, baseSeq) ([]Projection, error)` fold (ADR-0008): decodes an
  entry bundle into `{Seq, IsccID, NoteSchema, RecordSHA256}` per leaf, reading the committed `iscc_id` and
  the verbatim inner `note.$schema`, interpreting nothing (only a JSON-parse failure is an error). This is
  the read/decode half. **Still missing**: the store writer that persists these into the `iscc_index` table,
  and any production caller — the decoder is an unwired export seam (verified: no non-test caller).

**What remains for M2's Verify bar (all not-started):**
1. **Store writer for `iscc_index`** — persist `BundleProjections` output (`iscc_id → seq` one-to-many,
   raw `note.$schema`) into the `iscc_index` table (which exists in `schema.sql` but is unwritten).
2. **Wire `VerifyInclusionEvidence` into `PollHub`** — the cross-check exists but never runs in production.
   Needs the `iscc_index` projection (to resolve `iscc_id → leafIndex`) + a sampled entry bundle. Until
   wired, M2's second Verify criterion ("computed inclusion proof matches the hub's
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s") is not exercised on real polls.
3. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`, never
   re-hitting the hub — not started.

## M3 — Trust API + dashboard
**Status**: not started. The `net/http` mux serves only `/metrics` (verified: sole `mux.Handle` is
`metricshttp.Handler` at `cmd/iscc-monitor/main.go:108`). No `/`, `/healthz`, REST surface,
`verify-for-me`, dashboard, log browser, or raw tlog-tiles mirror at canonical paths.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`); no `internal/proof` package exists; no WASM build target. The WASM-shareable purity
invariant rides on per-file import discipline — this slice's `projection.go` keeps it (review confirmed
`GOOS=js GOARCH=wasm go build ./internal/logclient` exits 0).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `4edec1d` (this slice is two
  pure additive files, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  **Latest run on `develop` for HEAD `71903e2`: `conclusion: success`** (run 27893716778).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (all 11 packages
  `ok`, `go vet`/`gofmt -l .` clean), the new golden mutation-proven non-vacuous, WASM build clean, scope
  discipline confirmed (exactly 2 additive files, no store import / no PollHub wiring), and oracle correctly
  N/A (pure JSON + content-SHA-256 fold).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `CheckpointAt` unordered `LIMIT 1` (fork re-detection compares an undefined row); `AcceptCheckpoint`
  discards resolved context → verified polls re-fetch did.json; "frozen hubs still advance accepted state
  on later clean polls" (ADR-0006); tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency
  policy split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s
  vestigial `out io.Writer` param.

## Next Milestone
**Complete M2.** No `critical` open, so feature work proceeds; the open `normal` issues also block DONE but
are weighed against the state→target gap. The natural next slice continues the `iscc_index` projection.

Candidate order:
1. **`iscc_index` store writer** — persist `BundleProjections` output into the `iscc_index` table
   (`iscc_id → seq` one-to-many, raw `note.$schema`); the decoder half landed this slice.
2. **Wire `VerifyInclusionEvidence` into `PollHub`** — completes M2's second Verify half. Needs the
   `iscc_index` projection (`iscc_id → leafIndex`) + a sampled entry bundle; `store.SQLiteFetcher.ReadTile`
   matches the `TileFetcher` signature.
3. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`.
4. **`normal` backlog**: "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt ORDER BY` fix;
   `AcceptCheckpoint` context reuse; tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store
   method; `CheckConsistency` collapse.
5. **sb1 fixture refresh** (`22b08f3e`→`069d0f14`) and **real alert transport** (close M1's alert path).
6. **M3 → WASM → OTS** remain after M2's Verify bar is met.
