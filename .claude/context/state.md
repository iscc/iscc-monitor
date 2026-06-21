<!-- assessed-at: 4a42b97bfd4b63dfc0fe2d68b61c74a06c10468b -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green — M2 (Aggregator) actively under construction. The live
tile-ingestion writer (the first M2 production code) has now landed and is wired into `PollHub`, so the
local mirror is populated from the live path for the first time. What remains for M2's **Verify** bar —
the `fsck` root-rebuild over `SQLiteFetcher`, the inclusion cross-check vs the hub's own
`IsccLogInclusionProof`, `iscc_index`, and store-served proofs — is still not-started.

Since the prior assessment (`2d7b057`) the source diff touches **only `internal/follower`**: a new
`ingest.go` (`ingestTiles` + `ingestHashTiles` + `ingestEntryBundles` + the re-derived `widthForP`) plus
a 3-test `ingest_test.go`, and a 21-line wiring change in `follower.go` that calls `ingestTiles` from the
verified, non-violation path of `PollHub` (after `cacheHubKey`, before `recordVerdict` — i.e. AFTER
`RecordCheckpoint`/`AdvanceFollowState`). `git diff 2d7b057..HEAD -- go.mod go.sum schema.sql` is empty
(no dependency change — `tiles`/`logclient`/`store` seams were already in the closure). This is the
**first production caller** of `tiles.TileCoords`/`BundleCoords` → `logclient.FetchTile`/
`FetchEntryBundle` → `store.RecordTile`/`RecordEntryBundle`; those four seams are no longer unwired. Latest
`review` handoff (2026-06-21, this slice) is **PASS / CONTINUE**, mutation-proven non-vacuous, with CI
green at HEAD. The project stays off DONE because M2's Verify criteria are not met and M3 → OTS are not
built.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs,
`/metrics` served over HTTP and collected by the binary). **CI-gated & green.** The one remaining M1
connective-tissue gap (a real, non-`slog` alert transport) is outside M1's Verify bar.

- **Verified incrementally** from `2d7b057`. The diff `2d7b057..HEAD` touches no
  Go/`go.mod`/`go.sum`/`schema.sql` byte outside `internal/follower/{follower.go, ingest.go,
  ingest_test.go}`. Every cmd/iscc-monitor / logclient / store / didweb / tiles / metrics / metricshttp
  / registry / config source + `cmd/notecheck` + `schema.sql` is carried forward byte-unchanged.
- **Test totals re-grepped at HEAD**: **146 `func Test`** across the project, **35** `_test.go` files
  (the production packages are the 11 under `cmd/` + `internal/`; `find` also lists the gitignored
  `cauldron/` reference copies, which are not module deps). Up from 143 — the +3 are
  `internal/follower/ingest_test.go` (tile/bundle mirror round-trip + `widthForP` mapping + a
  `PollHub`-mirrors-tiles end-to-end check).

- **equivocation trigger WIRED and now LIVE-FED** (changed posture): `checkConsistency` evaluates shrink
  → fork → equivocation; the growing-pair branch builds the RFC-6962 consistency proof from the LOCAL
  mirror and on `true` → freeze (ADR-0006 narrow proof-build-error skip). Golden-tested + mutation-proven
  across the 256-leaf tile boundary. **Previously dormant on the live path; now that `PollHub` mirrors
  real tiles via `ingestTiles`, `SQLiteFetcher.ReadTile` returns real BLOBs**, so the live equivocation
  path no longer always hits the missing-tile skip. (Note: `ingestTiles` runs AFTER the consistency check
  on the same poll, so the proof for an equivocation observation is built from the tiles mirrored on the
  *prior* growing polls — the intended design.)

- **`cmd/notecheck` — fully-independent signature-parity oracle (CI-gated)** (unchanged): reads `--vkey`
  + checkpoint text on stdin, runs `transparency-dev/formats/note.NewVerifier` +
  `golang.org/x/mod/sumdb/note.Open`, strict reject (`len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0`),
  prints `OK <name>`/exit 0 or exits 1/2. EXTERNAL parity oracle for the monitor's crypto path; CI shells
  the built binary against `testdata/live/sb0.iscc.id_checkpoint` (accept `OK sb0.iscc.id/log` + reject a
  one-char-flipped signature).

- **`/metrics` served + wired**, **`internal/metrics` leaf**, **structured logging (`slog`)**,
  **`SQLiteFetcher` + partial-tile mirror CRUD**, **hub_keys cache + key readers**, **coverage tracking**
  (set-once, ADR-0001), **`logclient.Origin`** + golden `TestOrigin`, **config loader**, **realm-registry
  parser** (domains-only), **poll-loop cadence** (single-writer), **freeze + alert-once** (gated on
  `!wasFrozen`), the pure `ConsistencyProofFromTiles` + `InclusionProofFromTiles` builders, the
  `internal/tiles` seam, and the two `Fetch{Tile,EntryBundle}` transport primitives — all carried forward
  / unchanged.

- `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
  single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
  `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
  `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a leaf
  (`internal/follower` → `{logclient, store, tiles, metrics}`, never the reverse; no `net/http` in the
  store closure). **Byte-unchanged this slice** (the follower re-derives a one-line `widthForP` rather
  than exporting the store's private copy).

- **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
  `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
  possible. The window IS re-checked every poll via `AcceptCheckpoint`'s `ValidAt`, gating
  `StatusVerified` before the cache fast path. Documented limitation, not a green-but-wrong path.

- **Missing (remaining M1 connective tissue, outside the Verify bar):**
  - **Real alert transport** — `alertFunc` is a WARN `slog` emit; real delivery (email/webhook) is a
    later step. The `AlertFunc func(int64,string)` seam is unchanged.

- **Fixtures**: `testdata/live/` holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — still no tiles or entry bundles (verified `ls`). The new `ingest_test.go`
  uses **synthetic per-URL bytes** (a transport+CRUD writer needs no real tile bytes), so this slice did
  NOT add real fixtures. Real on-disk tile/entry-bundle fixtures remain a hard prerequisite for the M2
  `fsck` root-rebuild and the inclusion cross-check vs the hub's `IsccLogInclusionProof`. **Known stale
  did.json drift, still not acted on:** `sb1.amlet.id_did.json` (both `internal/didweb/` and
  `internal/logclient/`) and `derive_vkey.py` carry sb1's PRE-rotation key (`22b08f3e`); the live sb1
  signer is `069d0f14`. Captured in `verify_test.go` prose/tests (not green-but-wrong), but the fixtures
  remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2` (`rfc6962`, `proof` — `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera
  v1.0.2` (`api/layout` in `internal/tiles` — `BundleCoords`/`TileCoords`, `Fetch{Tile,EntryBundle}`; both
  proof builders; `leafhasher.go`; `tessera/fsck` in `logclient/fsck.go`; `tessera/client` re-export),
  `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib `log/slog` + `net/http` + `os`. **Not yet
  wired**: `nbd-wtf/opentimestamps` (grep → no hits outside `cauldron/`).

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: partially met (first build step landed; Verify bar NOT met). The **live tile-ingestion
writer** now exists and is wired:
- `internal/follower/ingest.go` — `ingestTiles(ctx, st, fetcher, hubID, baseURL, treeSize, observedAt)`
  walks `tiles.TileCoords(treeSize)` then `tiles.BundleCoords(treeSize)`, fetches each over
  `logclient.FetchTile`/`FetchEntryBundle`, and writes via `store.RecordTile`/`RecordEntryBundle` keyed by
  the `widthForP`-translated width (full tile `Partial==0` → 256, not 0 — the load-bearing translation,
  triple-pinned + mutation-proven per the review handoff). Re-fetched/overwritten every verified growing
  poll (ADR-0005 idempotent upsert). A fetch/store fault is a genuine transport error returned up to
  `PollHub` (NOT a violation, ADR-0006). **Wired into `PollHub`** on the verified, non-violation path.
- All prior pure prerequisites remain in place (carried forward): `internal/tiles` layout re-export +
  `IsFull`; `BundleCoords` + `TileCoords` (oracle-confirmed); `logclient.FetchTile`/`FetchEntryBundle`
  (now with a production caller); `store/{tiles,fetcher}.go` partial-tile CRUD + `SQLiteFetcher`;
  `ConsistencyProofFromTiles` + `InclusionProofFromTiles`; `LeafHashes`; `RunFsck`.

**What remains for M2's Verify bar (all not-started):**
1. **`fsck` root-rebuild over `SQLiteFetcher`** (FIRST half of M2 Verify): wire `RunFsck` (still has **no
   production caller** — verified) so `fsck.New(...).Check(ctx)` rebuilds each accepted root from the now-
   populated local mirror and cross-checks it against the signed checkpoint root. This re-arms the
   trust-root oracle gate (RFC-6962 root-rebuild) and is the first place the mirrored tiles face
   conformance — it needs **byte-accurate live tile/entry-bundle fixtures** captured into
   `testdata/live/` (the review handoff explicitly flags: do NOT let it reuse the writer's synthetic
   bytes).
2. **Inclusion cross-check** (SECOND half): assert `InclusionProofFromTiles` byte-equals the hub's own
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s (needs captured `IsccLogInclusionProof`
   fixtures + the same tile/bundle fixtures).
3. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many; stores raw
   `note.$schema`) — not started.
4. **Serve `inclusion`/`consistency`/`entries`** from the local store via a full `ProofBuilder`, never
   re-hitting the hub — not started.

## M3 — Trust API + dashboard
**Status**: not started. The binary has a `net/http` mux serving only `/metrics`; `/`, `/healthz`, and
the REST surface are out of scope until M3.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits outside
`cauldron/`); no `internal/proof` package exists (verified absent) and no WASM build target — the
WASM-shareable purity invariant currently rides on `internal/didweb` (review runs `GOOS=js GOARCH=wasm go
build ./internal/didweb` as the guard; `internal/tiles` is also WASM-green).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `2d7b057` (this slice
  adds no dependency — all seams already in the closure).
- **CI is configured and passing.** `.github/workflows/ci.yml` runs one `ubuntu-latest` /
  `CGO_ENABLED=0` job on push + PR to `develop`/`main`: the inlined `mise run check` gate (`go build
  ./...`, `go vet ./...`, `go test ./...`) plus the `cmd/notecheck` oracle shell-out. **Latest run on
  `develop` for HEAD `4a42b97`: `conclusion: success`** (run 27892094476, `gh run list --branch
  develop`). Remote `origin` configured (`github.com/iscc/iscc-monitor`); working branch `develop`, in
  sync with `origin/develop` (0/0); tree clean at HEAD.
- Latest `review` handoff (2026-06-21, "Live tile/bundle ingestion writer in PollHub", verdict **PASS /
  CONTINUE**) records `mise run check` green (all 11 packages `ok`, `go vet`/`gofmt -l .` clean) and
  mutation evidence: neutering `ingestTiles` or breaking `widthForP` makes the ingest/`PollHub` tests
  FAIL (non-vacuous). Oracle gate correctly **N/A** for this transport+CRUD slice (no
  signature/RFC-6962/Merkle/did:web/fsck path); trust-root packages re-ran uncached → all `ok`.
- **No open `critical`/`normal` issue.** **One open `low` issue** (loop-skipped): `cmd/notecheck`'s
  `run(vkey, in, out)` has a vestigial `out io.Writer` param never written to — harmless, `go vet`-clean,
  fix when `run` is next touched.
- **CI footnotes** (carried forward): CI never checks out gitignored `cauldron/`, so a fresh `go build
  ./...` is clean. The inlined CI commands duplicate `mise.toml [tasks.check]` byte-for-byte — keep them
  in lockstep. The `notecheck` build artifact at repo root only materializes on the ephemeral runner.

## Next Milestone
**M2 — Aggregator (Verify bar).** The live mirror is now populated from `PollHub`, so the next slice is
the FIRST half of M2's Verify: **wire `RunFsck` over `SQLiteFetcher`** so `fsck.New(...).Check(ctx)`
rebuilds each accepted root from the mirrored tiles and cross-checks it against the signed checkpoint
root. This is the slice that re-arms the trust-root oracle gate (RFC-6962 root-rebuild crypto) and the
first place the mirrored tiles face conformance — so it **requires byte-accurate live tile/entry-bundle
fixtures captured into `testdata/live/`** (must NOT reuse the writer's synthetic bytes — the review
handoff flags this explicitly). `RunFsck`, `LeafHashes`, and the proof builders already exist as unwired
pure seams ready for this caller.

Candidate order:
1. **`fsck` root-rebuild over `SQLiteFetcher`** — wire `RunFsck` against the mirrored tiles (its first
   production caller); capture real tile/entry-bundle fixtures. FIRST half of M2's Verify.
2. **Inclusion cross-check** — assert `InclusionProofFromTiles` byte-equals the hub's own
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s (SECOND half; same fixtures + captured
   `IsccLogInclusionProof`).
3. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many) + serving
   `inclusion`/`consistency`/`entries` from the local store via a full `ProofBuilder`.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
5. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to close
   M1's alert path.
