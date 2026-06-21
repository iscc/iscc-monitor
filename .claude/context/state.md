<!-- assessed-at: b9ee0e15c4bda4d7f37303e2f579a4d819c5a240 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green — M2 (Aggregator) actively under construction; its Verify bar is
now HALF met. The `fsck` root-rebuild over `SQLiteFetcher` (the FIRST half of M2's Verify) has landed:
`fsckMirror` rebuilds each accepted root from the freshly-mirrored local tiles and cross-checks it
against the signed checkpoint root on every verified poll. The SECOND half (inclusion cross-check vs the
hub's own `IsccLogInclusionProof`), the `iscc_index` projection, and store-served proofs are still
not-started, so M2 is not yet met.

Since the prior assessment (`4a42b97`) the source diff touches **only `internal/follower/follower.go`**
(+~55 lines: the new `fsckMirror` helper plus its wiring into `PollHub` after `ingestTiles`, before
`recordVerdict`). `git diff 4a42b97..HEAD -- go.mod go.sum internal/store/schema.sql` is **empty** (no
dependency/schema change — `tessera/fsck`, `note`, the `SQLiteFetcher` were already in the closure). This
is the **first production caller of `logclient.RunFsck`** — that trust-root seam is no longer unwired.
Latest `review` handoff (2026-06-21, this slice) is **PASS_WITH_NOTES / CONTINUE**, mutation-proven
non-vacuous (early-`return nil` in `fsckMirror` makes `RejectsCorruptedMirror` FAIL), with CI green at
HEAD. The project stays off DONE because M2's Verify bar is only half met and M3 → OTS are not built.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP and collected by the binary). **CI-gated & green.** The one remaining M1
connective-tissue gap (a real, non-`slog` alert transport) is outside M1's Verify bar.

- **Verified incrementally** from `4a42b97`. The diff `4a42b97..HEAD` touches no Go/`go.mod`/`go.sum`/
  `schema.sql` byte outside `internal/follower/follower.go` (production) plus follower `_test.go` files
  (the rest of the diff is `.claude/context/*` docs). Every `cmd/iscc-monitor` / `cmd/notecheck` /
  logclient / store / didweb / tiles / metrics / metricshttp / registry / config source + `schema.sql` is
  carried forward byte-unchanged.
- **Test totals re-grepped at HEAD**: **147 `func Test`** across the production tree (`cmd/` +
  `internal/`), **36** `_test.go` files (11 production packages). Up from 146 — the +1 is the new
  `internal/follower/fsck_test.go::TestPollHubFsck` (two subtests: `RebuildsSignedRoot` +
  `RejectsCorruptedMirror`). Per the review handoff this slice also CONVERTED several real-sb0
  follower tests to the in-process `buildVerifiedMirror(t, mirrorLeaves=300)` fixture (net assertions up:
  +32/−16), since the live sb0 log's leaf preimages were never captured and would fail fsck.

- **equivocation trigger WIRED + live-fed** (unchanged posture): `checkConsistency` evaluates shrink →
  fork → equivocation; the growing-pair branch builds the RFC-6962 consistency proof from the LOCAL
  mirror and on `true` → freeze (ADR-0006 narrow proof-build-error skip). Golden-tested + mutation-proven
  across the 256-leaf tile boundary. Since `PollHub` mirrors real tiles via `ingestTiles`,
  `SQLiteFetcher.ReadTile` returns real BLOBs, so the live equivocation path no longer always hits the
  missing-tile skip (the proof for an equivocation observation is built from tiles mirrored on *prior*
  growing polls — the intended design).

- **`cmd/notecheck` — fully-independent signature-parity oracle (CI-gated)** (unchanged): reads `--vkey`
  + checkpoint text on stdin, runs `transparency-dev/formats/note.NewVerifier` +
  `golang.org/x/mod/sumdb/note.Open`, strict reject (`len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0`).
  EXTERNAL parity oracle for the monitor's crypto path; CI shells the built binary against
  `testdata/live/sb0.iscc.id_checkpoint` (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).

- **`/metrics` served + wired**, **`internal/metrics` leaf**, **structured logging (`slog`)**,
  **`SQLiteFetcher` + partial-tile mirror CRUD**, **hub_keys cache + key readers**, **coverage tracking**
  (set-once, ADR-0001), **`logclient.Origin`** + golden `TestOrigin`, **config loader**, **realm-registry
  parser** (domains-only), **poll-loop cadence** (single-writer), **freeze + alert-once** (gated on
  `!wasFrozen`), the pure `ConsistencyProofFromTiles` + `InclusionProofFromTiles` builders, `LeafHashes`,
  `RunFsck`, the `internal/tiles` seam, and the two `Fetch{Tile,EntryBundle}` transport primitives — all
  carried forward / unchanged.

- `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
  single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
  `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
  `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a leaf
  (`internal/follower` → `{logclient, store, tiles, metrics}`, never the reverse; no `net/http` in the
  store closure). **Byte-unchanged this slice.**

- **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
  `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
  possible. The window IS re-checked every poll via `AcceptCheckpoint`'s `ValidAt`, gating
  `StatusVerified` before the cache fast path. Documented limitation, not a green-but-wrong path.

- **Missing (remaining M1 connective tissue, outside the Verify bar):**
  - **Real alert transport** — `alertFunc` is a WARN `slog` emit; real delivery (email/webhook) is a
    later step. The `AlertFunc func(int64,string)` seam is unchanged.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — still **no tiles or entry bundles** (verified `ls`). The fsck wiring this
  slice rebuilds from an **in-process** `buildVerifiedMirror` fixture (per-run keypair, byte-accurate
  tiles/bundles across the 256-leaf boundary), NOT captured real tiles — so this slice did not add real
  on-disk fixtures. Real tile/entry-bundle + `IsccLogInclusionProof` fixtures remain a hard prerequisite
  for the M2 **inclusion cross-check**. **Known stale did.json drift, still not acted on:**
  `sb1.amlet.id_did.json` (both `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py` carry
  sb1's PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`. Captured in `verify_test.go`
  prose/tests (not green-but-wrong), but the fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2` (`rfc6962`, `proof` — `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera
  v1.0.2` (`api/layout` in `internal/tiles`; `BundleCoords`/`TileCoords`, `Fetch{Tile,EntryBundle}`; both
  proof builders; `leafhasher.go`; `tessera/fsck` in `logclient/fsck.go` — **now with a production
  caller via `fsckMirror`**; `tessera/client` re-export), `transparency-dev/formats` (DIRECT —
  `cmd/notecheck`), stdlib `log/slog` + `net/http` + `os`. **Not yet wired**: `nbd-wtf/opentimestamps`
  (grep → no hits outside `cauldron/`).

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: partially met — **HALF of the Verify bar landed**. The mirror is populated from `PollHub`
(via `ingestTiles`, prior slice) and the **`fsck` root-rebuild over `SQLiteFetcher` now runs on every
verified poll** (this slice):
- `internal/follower/follower.go` — new `fsckMirror(ctx, st, fetcher, hubID, baseURL)` resolves the vkey
  (`ResolveVerifierKey`) + origin (`Origin`), builds a read-only `store.SQLiteFetcher{Store, HubID}` over
  the just-ingested tiles, and calls `logclient.RunFsck` — **the first production caller of `RunFsck`** —
  which re-hashes each entry bundle, re-derives the lower hash tiles, and compares the rebuilt RFC-6962
  root to the checkpoint's signed root. **Wired into `PollHub`** AFTER `ingestTiles`, BEFORE
  `recordVerdict`. A rebuild mismatch / mirror fault is a **genuine fault returned to the caller — NOT a
  self-consistency violation** (no freeze; the checkpoint is already recorded/advanced, so a transient
  fault re-attempts next poll — ADR-0006).
- Golden-tested by `TestPollHubFsck` (`RebuildsSignedRoot` + `RejectsCorruptedMirror`), mutation-proven
  by review (early-`return nil` → `RejectsCorruptedMirror` FAILS). RFC-6962 root-rebuild oracle gate
  re-armed: review confirms `derive_vkey.py` reproduces both golden vectors and `notecheck` accepts the
  real sb0 checkpoint. Honest framing (review + code doc): `RunFsck` is an **in-process structural
  self-check** (shares the monitor's own `LeafHashes`/RFC-6962 code), not the fully-independent oracle.
- All prior pure prerequisites remain in place (carried forward): `internal/tiles` layout re-export +
  `IsFull`; `BundleCoords` + `TileCoords`; `logclient.FetchTile`/`FetchEntryBundle`;
  `store/{tiles,fetcher}.go` partial-tile CRUD + `SQLiteFetcher`; `ConsistencyProofFromTiles` +
  `InclusionProofFromTiles`; `LeafHashes`; `ingestTiles`.

**What remains for M2's Verify bar (all not-started):**
1. **Inclusion cross-check** (SECOND half of M2 Verify): assert `InclusionProofFromTiles` byte-equals the
   hub's own `evidence.IsccLogInclusionProof` for sampled `iscc_id`s. `InclusionProofFromTiles` still has
   **no production caller** (verified: only its own definition in `proofbuilder.go` + the
   `inclusionproof_test.go` unit golden). Needs captured `IsccLogInclusionProof` fixtures + tile/bundle
   fixtures (real or in-process).
2. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many; stores raw
   `note.$schema`) — not started (no `iscc_index`/`RecordIndex` writer in production; the table exists in
   `schema.sql` but is unwritten).
3. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`, never
   re-hitting the hub — not started.

## M3 — Trust API + dashboard
**Status**: not started. The binary has a `net/http` mux serving only `/metrics`; `/`, `/healthz`, and
the REST surface are out of scope until M3.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`); no `internal/proof` package exists (verified absent) and no WASM build target — the
WASM-shareable purity invariant currently rides on `internal/didweb` (review runs `GOOS=js GOARCH=wasm go
build ./internal/didweb` as the guard; `internal/tiles` is also WASM-green).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `4a42b97` (this slice
  adds no dependency — `tessera/fsck`+`note`+`SQLiteFetcher` were already in the closure).
- **CI is configured and passing.** `.github/workflows/ci.yml` runs one `ubuntu-latest` /
  `CGO_ENABLED=0` job on push + PR to `develop`/`main`: the inlined `mise run check` gate (`go build
  ./...`, `go vet ./...`, `go test ./...`) plus the `cmd/notecheck` oracle shell-out. **Latest run on
  `develop` for HEAD `b9ee0e1`: `conclusion: success`** (run 27892774944). Remote `origin` configured
  (`github.com/iscc/iscc-monitor`); working branch `develop`, tree clean at HEAD.
- Latest `review` handoff (2026-06-21, "Wire RunFsck into PollHub over the live SQLiteFetcher mirror",
  verdict **PASS_WITH_NOTES / CONTINUE**) records `mise run check` green (all 11 packages `ok`, `go
  vet`/`gofmt -l .` clean) and mutation evidence: `fsckMirror` early-`return nil` → `RejectsCorruptedMirror`
  FAILS (non-vacuous). Oracle gate **APPLIES** (RFC-6962 root-rebuild) and is satisfied; `derive_vkey.py`
  reproduces both vectors, `notecheck` accepts the real sb0 checkpoint. The advance's HUMAN REVIEW
  REQUESTED (converting 6 real-sb0 tests to the in-process mirror) was independently cleared by review as
  a sound equivalent with no trust-root coverage loss (real-sb0 parity still covered at the verification
  layer by `TestAcceptCheckpoint/"verified"` + `notecheck`).
- **No open `critical`/`normal`-blocking issue against DONE.** Two **`normal`** issues are filed by
  review (both surfaced by this slice, neither a gate failure): (1) `CheckpointAt` unordered `LIMIT 1` →
  fork re-detection compares against an undefined post-freeze row (forced `TestPollHubFork` to drive
  re-detection via `freeze`); fix belongs to the store-touching equivocation/serving slice. (2)
  `fsckMirror` re-resolves the did:web key every verified poll (the +1 did.json fetch the cache-hit test
  now pins at cold=3/warm=2) — efficiency-only, correctness unaffected. One open **`low`** issue
  (loop-skipped): `cmd/notecheck`'s `run` has a vestigial `out io.Writer` param.
- **CI footnotes** (carried forward): CI never checks out gitignored `cauldron/`, so a fresh `go build
  ./...` is clean. The inlined CI commands duplicate `mise.toml [tasks.check]` byte-for-byte — keep them
  in lockstep.

## Next Milestone
**M2 — Aggregator (SECOND half of the Verify bar).** With the `fsck` root-rebuild now live on every
verified poll, the immediate next goal is the **inclusion cross-check**: wire `InclusionProofFromTiles`
(still an unwired seam — verified no production caller) so the monitor's computed inclusion proof
byte-equals the hub's own `evidence.IsccLogInclusionProof` for sampled `iscc_id`s. This is the final
half of M2's Verify and the second true *external* oracle (the hub-computed proof). It needs captured
`IsccLogInclusionProof` fixtures + tile/bundle fixtures (real or in-process, per the review handoff's
fsck precedent).

Candidate order:
1. **Inclusion cross-check** — wire `InclusionProofFromTiles`; assert byte-equality vs the hub's
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s. SECOND half of M2's Verify (completes M2's
   Verify bar once landed).
2. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many) + serving
   `inclusion`/`consistency`/`entries` from the local store via a full `ProofBuilder`.
3. **`CheckpointAt ORDER BY` fix** (`normal`, review-filed) — unblocks deterministic fork re-detection
   via `PollHub`; a store-touching equivocation/serving step.
4. **`fsckMirror` redundant did:web resolve** (`normal`, review-filed) — thread the already-resolved
   vkey through to drop the +1 did.json fetch per verified poll.
5. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
6. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to close
   M1's alert path.
