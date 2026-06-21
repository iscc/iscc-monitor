<!-- assessed-at: e2ed264becf496c6a0409475c99b13caaffc186b -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green. M2 (Aggregator) actively under construction; its Verify bar's
two halves are now BOTH built but only the first is wired. The `fsck` root-rebuild over `SQLiteFetcher`
(first half) runs on every verified poll via `fsckMirror`. The inclusion cross-check (second half) just
landed as `VerifyInclusionEvidence` — **pure, oracle-exact, mutation-proven, but with no production caller
yet**. The `iscc_index` projection and store-served proofs remain not-started, so M2 is not yet met.

Since the prior assessment (`b9ee0e1`) the source diff `b9ee0e1..HEAD` touches **only
`internal/logclient/inclusioncheck.go` (+110) and its `_test.go` (+183)** — strictly additive, in one
package. `git diff b9ee0e1..HEAD -- go.mod go.sum internal/store/schema.sql` is **empty** (no dependency
or schema change — `bytes`/`base64`/`json` are stdlib, `InclusionProofFromTiles` was already in the
package). The rest of the diff is `.claude/context/*` docs. Latest `review` handoff (2026-06-21,
"VerifyInclusionEvidence inclusion cross-check") is **PASS / CONTINUE**, mutation-proven non-vacuous
(neutering the length+`bytes.Equal` compares makes both corruption subtests FAIL), with **CI green at
HEAD `e2ed264`** (develop run 27893223960, `conclusion: success`).

**Branch note:** active work happens on `develop` (HEAD `e2ed264`, tree clean, up to date with
`origin/develop`); a human merges `develop`→`main` via CI. The session-start git snapshot showing `main`
at `c59d380` was stale — the real working branch is `develop`.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP and collected by the binary). **CI-gated & green.** Carried forward byte-unchanged since
`b9ee0e1` — the diff to HEAD touches no M1 source (only `internal/logclient/inclusioncheck.go` + docs).

- **Test totals re-grepped at HEAD**: **153 `func Test`** across the production tree (`cmd/` + `internal/`),
  **37** `_test.go` files (11 production packages). Up from 147/36 — the +6 funcs / +1 file is
  `internal/logclient/inclusioncheck_test.go` (`TestParseInclusionEvidence` + `TestVerifyInclusionEvidence`
  with their subtests).
- **Packages present** (verified `ls`): `cmd/{iscc-monitor,notecheck}`; `internal/{config,didweb,follower,
  logclient,metrics,metricshttp,registry,store,tiles}`. Module path `github.com/iscc/iscc-monitor`.
- **All three triggers WIRED + golden-tested** (carried forward): shrink → fork → equivocation in
  `checkConsistency`; growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror
  → freeze on `true` (ADR-0006). **Caveat — see the open `critical` below: the equivocation proof is built
  before `ingestTiles`, so a *growing* split view can still pass cleanly and advance.**
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real sb0
  checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig). Carried forward.
- `/metrics` served + wired, `internal/metrics` leaf, `slog` structured logging, `SQLiteFetcher` +
  partial-tile mirror CRUD, hub_keys cache, coverage tracking (set-once, ADR-0001), `logclient.Origin` +
  golden `TestOrigin`, config loader, realm-registry parser (domains-only), poll-loop cadence
  (single-writer), freeze + alert-once. All carried forward byte-unchanged.
- `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
  single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded
  nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`, `violations`, `tiles`, `entry_bundles`,
  `iscc_index`, `follow_state`, `ots`). Store stays a leaf. Byte-unchanged.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a WARN
  `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is later.
- **Fixtures**: `testdata/live/` holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — still **no tiles or entry bundles** (verified `ls`). All tile/fsck/inclusion
  tests run against in-process `testonly.Tree` / `buildVerifiedMirror` fixtures. Real tile/entry-bundle +
  `IsccLogInclusionProof` fixtures remain a soft prerequisite for wiring the inclusion cross-check into
  `PollHub`. Known stale `sb1.amlet.id_did.json` drift (pre-rotation key `22b08f3e`; live signer
  `069d0f14`) — captured in tests, fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2` (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera v1.0.2`
  (`api/layout`, both proof builders, `leafhasher`, `tessera/fsck` with a production caller via
  `fsckMirror`, `tessera/client` re-export), `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib
  `slog`/`net/http`/`os`. **Not yet wired**: `nbd-wtf/opentimestamps` (grep → no hits in `cmd/`+`internal/`).

## M2 — Aggregator
**Status**: partially met — **BOTH pure halves of the Verify bar are now BUILT; only the first is WIRED.**
- **First half (WIRED, runs every verified poll):** `fsckMirror` (in `follower.go`, after `ingestTiles`,
  before `recordVerdict`, lines 197-208) builds a read-only `store.SQLiteFetcher` over the just-ingested
  tiles and calls `logclient.RunFsck` — re-hashing each entry bundle, re-deriving lower hash tiles, and
  comparing the rebuilt RFC-6962 root to the signed checkpoint root. A mismatch is a genuine mirror fault
  (no freeze), not a self-consistency violation. Mutation-proven (early `return nil` → `RejectsCorruptedMirror`
  FAILS).
- **Second half (BUILT this slice, pure, NOT YET WIRED):** `internal/logclient/inclusioncheck.go` —
  `ParseInclusionEvidence` decodes a hub's `IsccLogInclusionProof` VC evidence (`{type, checkpoint, treeSize,
  leafIndex, inclusionProof[]}`, base64-Std proof), and `VerifyInclusionEvidence` recomputes the proof via
  `InclusionProofFromTiles` (its first production-shaped caller) and `bytes.Equal`-compares each hash,
  wrapping `ErrInclusionMismatch` on a hash mismatch while preserving `os.ErrNotExist` on a tile fault.
  Oracle-exact vs `iscc_hub`'s `inclusion_evidence`; mutation-proven load-bearing by review. **Verified
  unwired** (grep): no caller of `VerifyInclusionEvidence` / `InclusionProofFromTiles` outside its own
  package definition + tests; nothing in `PollHub`.

**What remains for M2's Verify bar (all not-started):**
1. **Wire `VerifyInclusionEvidence` into `PollHub`** — the cross-check exists but never runs in production.
   Needs the `iscc_index` projection (to resolve `iscc_id → leafIndex`) + a sampled entry bundle. Until
   wired, M2's second Verify criterion ("computed inclusion proof matches the hub's
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s") is not exercised on real polls.
2. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many, raw `note.$schema`) —
   not started (verified: no production `iscc_index`/`RecordIndex` writer; the table exists in `schema.sql`
   but is unwritten).
3. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`, never
   re-hitting the hub — not started.

## M3 — Trust API + dashboard
**Status**: not started. The `net/http` mux serves only `/metrics` (verified: the sole `mux.Handle` is
`metricshttp.Handler`). No `/`, `/healthz`, REST surface, `verify-for-me`, dashboard, or raw tiles mirror.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`); no `internal/proof` package exists (verified absent); no WASM build target. The
WASM-shareable purity invariant currently rides on `internal/didweb` (review's `GOOS=js GOARCH=wasm go
build ./internal/didweb` guard; `internal/logclient` also builds js/wasm).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `b9ee0e1` (this slice adds
  no dependency).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to `develop`/`main`.
  **Latest run on `develop` for HEAD `e2ed264`: `conclusion: success`** (run 27893223960).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (all 11 packages
  `ok`, `go vet`/`gofmt -l .` clean), the oracle gate satisfied (`derive_vkey.py` reproduces both vectors,
  `notecheck` accepts the real sb0 checkpoint), and the byte-comparison mutation-proven load-bearing.
- **One open `critical` issue blocks DONE** (filed by review, in `issues.md`): "Growing equivocations can be
  accepted before candidate tiles are mirrored" — `internal/follower/follower.go` builds the growing
  consistency proof in `checkConsistency` (before `RecordCheckpoint`/`AdvanceFollowState` at line 181) while
  `ingestTiles` only runs afterward at line 197, so the mirror holds tiles only up to the *previously*
  accepted size. A growing split view hits a missing-candidate-tile error that `checkConsistency` treats as
  a clean pass, then advances `last_size` to the inconsistent root and never freezes. **Confirmed in code**
  this assessment (PollHub ordering re-read). This is a genuine trust-root correctness gap, not an
  efficiency item.
- **Other open issues** (do not block this slice's PASS but `normal` ones block DONE): `normal` —
  "Frozen hubs still advance accepted state on later clean-looking polls"; "`CheckpointAt` unordered `LIMIT
  1` → fork re-detection compares against an undefined row"; "`fsckMirror` re-resolves the did:web key every
  verified poll" (efficiency). `low` (loop-skipped) — `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**Fix the open `critical` first, then complete M2.** Per the gate rules, a `critical` correctness gap in
the equivocation freeze path preempts feature work — a growing split view must freeze and must not advance
accepted state.

Candidate order:
1. **Fix the `critical` growing-equivocation gap** (`internal/follower/follower.go`): ensure the
   candidate-size tiles backing the consistency proof are mirrored before the growing checkpoint is
   accepted, OR make a missing proof tile a retry/error rather than a clean consistency pass. Verify with a
   growing split-view poll: the hub freezes and does not advance `last_size` to the candidate root.
2. **Wire `VerifyInclusionEvidence` into `PollHub`** (this slice's natural successor) — completes M2's
   second Verify half. Needs the `iscc_index` projection (`iscc_id → leafIndex`) + a sampled entry bundle;
   `store.SQLiteFetcher.ReadTile` matches the `TileFetcher` signature and passes straight in.
3. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many) + serving
   `inclusion`/`consistency`/`entries` from the local store via a full `ProofBuilder`.
4. **`normal` backlog**: "frozen hubs still advance" (ADR-0006 evidence-only); `CheckpointAt ORDER BY` fix
   (deterministic fork re-detection); `fsckMirror` redundant did:web resolve (efficiency).
5. **sb1 fixture refresh** (`22b08f3e`→`069d0f14`) and **real alert transport** (close M1's alert path).
