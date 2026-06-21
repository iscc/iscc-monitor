<!-- assessed-at: 4edec1df0ce6026360b2c16200806e01b00d0022 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green; the prior `critical` growing-split-view trust gap is now FIXED.
M2 (Aggregator) actively under construction: its Verify bar's two halves are both built, the first
(`fsck` root-rebuild over `SQLiteFetcher`) is wired and runs every verified poll, the second
(`VerifyInclusionEvidence` inclusion cross-check) remains pure-but-unwired. The `iscc_index` projection
writer and store-served proofs are still not-started, so M2 is not yet met. M3 / WASM / OTS not started.

Since the prior assessment (`e2ed264`) the source diff `e2ed264..HEAD` touches **only
`internal/follower/follower.go` (reordered, ~68 lines) and `internal/follower/equivocation_test.go`
(+92, three test funcs)** — a pure reorder that moves the single `ingestTiles(...info.TreeSize...)` call
**ahead of `checkConsistency`**, closing the open `critical`: a growing split view now has its
candidate-size tiles mirrored before the RFC-6962 consistency proof is built, so the proof can no longer
hit a missing-tile error that `checkConsistency` swallowed as a clean pass → silent advance.
`git diff e2ed264..HEAD -- go.mod go.sum internal/store/schema.sql` is **empty** (no dep/schema change).
Latest `review` handoff (2026-06-21, "Mirror candidate tiles before the consistency check") is **PASS /
CONTINUE**, mutation-proven non-vacuous, with the prior `critical` deleted from `issues.md` after
verifying the fix end-to-end. **CI green at HEAD `4edec1d`** (develop run 27893486635, `success`).

**Branch note:** active work happens on `develop` (HEAD `4edec1d`, tree clean, up to date with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP and collected by the binary). **CI-gated & green.**

- **Test totals re-grepped at HEAD**: **154 `func Test`** across the production tree (`cmd/` + `internal/`),
  **37** `_test.go` files (11 production packages). +1 func over the prior 153 — the three new
  `equivocation_test.go` funcs net +1 because they extend an existing file (file count unchanged at 37).
- **Packages present** (verified `ls`): `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,
  logclient,metrics,metricshttp,registry,store,tiles}`. Module path `github.com/iscc/iscc-monitor`.
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on `true`
  (ADR-0006). **The prior `critical` (growing split view could advance before candidate tiles were
  mirrored) is now CLOSED** — re-verified in code this assessment: PollHub order is `FollowState`(145) →
  `ingestTiles`(163, aborts the poll on a tile fault before any accepted-state write) → `checkConsistency`
  (169) → `freeze`(183) / `RecordCheckpoint`(194) + `AdvanceFollowState`(202). New end-to-end
  `TestPollHubGrowingSplitViewFreezes` exercises it.
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
  `sb1.amlet.id_checkpoint`) — still **no tiles or entry bundles** (verified `ls`). All tile/fsck/inclusion
  tests run against in-process `testonly.Tree` / `buildVerifiedMirror` fixtures. Real tile/entry-bundle +
  `IsccLogInclusionProof` fixtures remain a soft prerequisite for wiring the inclusion cross-check into
  `PollHub`. Known stale `sb1.amlet.id_did.json` drift (pre-rotation key `22b08f3e`; live signer
  `069d0f14`) captured in tests; fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api/layout`, both
  proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`, `tessera/client`
  re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib `slog`/`net/http`/`os`. **Not
  yet wired**: `nbd-wtf/opentimestamps` (grep → no hits in `cmd/`+`internal/`).

## M2 — Aggregator
**Status**: partially met — **BOTH pure halves of the Verify bar are BUILT; only the first is WIRED.**
- **First half (WIRED, runs every verified poll):** `fsckMirror` (`follower.go:352`, called at line 218
  after `ingestTiles`, before `recordVerdict`) builds a read-only `store.SQLiteFetcher` over the
  just-ingested tiles and calls `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash
  tiles, comparing the rebuilt RFC-6962 root to the signed checkpoint root. A mismatch is a mirror fault
  (no freeze), not a self-consistency violation. Mutation-proven by review.
- **Second half (BUILT, pure, NOT YET WIRED):** `internal/logclient/inclusioncheck.go` —
  `ParseInclusionEvidence` decodes a hub's `IsccLogInclusionProof` VC evidence, `VerifyInclusionEvidence`
  recomputes the proof via `InclusionProofFromTiles` and `bytes.Equal`-compares each hash, wrapping
  `ErrInclusionMismatch` on mismatch while preserving `os.ErrNotExist` on a tile fault. Oracle-exact vs
  `iscc_hub`'s `inclusion_evidence`; mutation-proven load-bearing. **Verified unwired** (grep): no
  production caller of `VerifyInclusionEvidence` / `InclusionProofFromTiles` outside its package def + tests.

**What remains for M2's Verify bar (all not-started):**
1. **Wire `VerifyInclusionEvidence` into `PollHub`** — the cross-check exists but never runs in production.
   Needs the `iscc_index` projection (to resolve `iscc_id → leafIndex`) + a sampled entry bundle. Until
   wired, M2's second Verify criterion ("computed inclusion proof matches the hub's
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s") is not exercised on real polls.
2. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many, raw `note.$schema`) —
   not started (verified: the only `iscc_index` mention in non-test Go is a comment in `inclusioncheck.go`;
   the table exists in `schema.sql` but is unwritten).
3. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`, never
   re-hitting the hub — not started.

## M3 — Trust API + dashboard
**Status**: not started. The `net/http` mux serves only `/metrics` (verified: sole `mux.Handle` is
`metricshttp.Handler` at `cmd/iscc-monitor/main.go:108`). No `/`, `/healthz`, REST surface,
`verify-for-me`, dashboard, log browser, or raw tlog-tiles mirror at canonical paths.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`); no `internal/proof` package exists (verified absent); no WASM build target. The
WASM-shareable purity invariant currently rides on `internal/didweb` (review's `GOOS=js GOARCH=wasm go
build ./internal/didweb` guard).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `e2ed264` (this slice is a
  pure reorder + test, no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  **Latest run on `develop` for HEAD `4edec1d`: `conclusion: success`** (run 27893486635).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (all 11 packages
  `ok`, `go vet`/`gofmt -l .` clean), the oracle gate satisfied (`derive_vkey.py` reproduces both vectors
  `40b74463`/`22b08f3e`; `notecheck` accepts real sb0), a clean gate-integrity scan of all unpushed
  commits, and the growing-split-view freeze fix verified end-to-end + mutation-proven non-vacuous.
- **No open `critical` issue** (the prior growing-split-view `critical` was resolved + deleted this slice).
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 6 in `issues.md`:
  `CheckpointAt` unordered `LIMIT 1` (fork re-detection compares an undefined row); `AcceptCheckpoint`
  discards resolved context → verified polls re-fetch did.json; "frozen hubs still advance accepted state
  on later clean polls" (ADR-0006); tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency
  policy split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s
  vestigial `out io.Writer` param.

## Next Milestone
**Complete M2** (the `critical` is cleared, so feature work proceeds). Per the gate, the open `normal`
issues also block DONE but are weighed against the state→target gap; the natural next slice is M2's
second half.

Candidate order:
1. **Wire `VerifyInclusionEvidence` into `PollHub`** — completes M2's second Verify half. Needs the
   `iscc_index` projection (`iscc_id → leafIndex`) + a sampled entry bundle; `store.SQLiteFetcher.ReadTile`
   matches the `TileFetcher` signature and passes straight in.
2. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many) + serving
   `inclusion`/`consistency`/`entries` from the local store via a full `ProofBuilder`.
3. **`normal` backlog**: "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt ORDER BY` fix
   (deterministic fork re-detection); `AcceptCheckpoint` context reuse (skip redundant did.json fetch);
   tile-writer `p`-vocabulary unification; deep `AdvanceAccepted` store method; `CheckConsistency` collapse.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14`) and **real alert transport** (close M1's alert path).
5. **M3 → WASM → OTS** remain after M2's Verify bar is met.
