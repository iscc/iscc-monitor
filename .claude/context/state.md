<!-- assessed-at: 2d7b0571e1992cf679ec066098f402dde3a55ecd -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green — M2 (Aggregator) is the active milestone. Every pure
prerequisite for the M2 live tile-ingestion writer now exists: both coordinate enumerations
(`tiles.BundleCoords` + `tiles.TileCoords`) AND, as of this iteration, the two transport primitives
(`logclient.FetchTile` + `logclient.FetchEntryBundle`) that fetch a tile/bundle over the `Fetcher`
seam. What remains for M2's Verify bar — wiring these into `PollHub` to actually mirror live
tiles/bundles, the `fsck` root-rebuild over `SQLiteFetcher`, the inclusion cross-check vs the hub's
own proof, `iscc_index`, and store-served proofs — is still not-started.

Since the prior assessment (`2a21f35`) the **only source-relevant change is one additive pure seam**:
`internal/logclient/tilefetch.go` gained `FetchTile(ctx, fetcher, baseURL, level, index uint64, p
uint8)` and `FetchEntryBundle(ctx, fetcher, baseURL, index uint64, p uint8)` — both port
`FetchCheckpoint` verbatim-in-shape (`origin()` → `"https://"+name+"/"+TilePath/EntriesPath` →
`fetcher.Fetch` → body verbatim, `%w`-wrapping so a 404's `os.ErrNotExist` survives) plus the golden
test `tilefetch_test.go` (6 `func Test`, reviewed PASS in `2d7b057`). `git diff 2a21f35..HEAD -- go.mod
go.sum schema.sql` is empty; the diff touches no other `.go` byte (only those two new `logclient` files
+ context md). It is an **unwired export seam** (`grep` confirms neither `FetchTile` nor
`FetchEntryBundle` has any production caller — they appear only in their definitions + tests). The
project stays off DONE because M2 → OTS are not yet built.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary). **CI-gated & green.** A real (non-log)
alert transport and live tile ingestion (the latter an M2 dependency) remain connective tissue outside
M1's Verify bar.

- **Verified incrementally** from `2a21f35`. The diff `2a21f35..HEAD` touches no
  Go/`go.mod`/`go.sum`/`schema.sql` byte outside the two new `internal/logclient/tilefetch{,_test}.go`
  files. Every cmd/iscc-monitor / follower / logclient (sans the two new files) / store / didweb /
  tiles / metrics / metricshttp / registry / config source + `cmd/notecheck` + `schema.sql` is carried
  forward byte-unchanged.
- **Test totals re-grepped at HEAD**: **143 `func Test`** across **11 packages** (**34** `_test.go`
  files), up from 137 — the +6 are the new `tilefetch_test.go` (`FetchTile`/`FetchEntryBundle` golden
  URLs + `os.ErrNotExist` propagation).

- **`cmd/notecheck` — fully-independent signature-parity oracle (CI-gated)** (unchanged): reads
  `--vkey` + checkpoint text on stdin, runs `transparency-dev/formats/note.NewVerifier` +
  `golang.org/x/mod/sumdb/note.Open`, applies the strict reject (`len(n.Sigs)==0 ||
  len(n.UnverifiedSigs)!=0`) mirroring `internal/logclient/verify.go`, prints `OK <name>`/exit 0 or
  exits 1 (verify fail) / 2 (bad vkey or stdin read). EXTERNAL parity check for the monitor's own
  crypto path — distinct from the in-process `RunFsck` self-check. **CI shells the built binary out**
  against `testdata/live/sb0.iscc.id_checkpoint`, asserting the exact `OK sb0.iscc.id/log` accept AND a
  one-char-flipped-signature reject (exit 1).

- **`/metrics` served + wired** (unchanged): `internal/metricshttp/handler.go` —
  `Handler(*metrics.Registry) http.Handler`; `cmd/iscc-monitor/main.go` serves it from a background
  `net/http.Server` on `ISCC_MONITOR_ADDR` (default `:9464`) and passes the registry as `Loop.Metrics`.

- **`internal/metrics` leaf** (unchanged): `*Registry` over `iscc_monitor_violations_total` /
  `_poll_failures_total` counters and `_hub_status` / `_last_observed_at` gauges, Prometheus text,
  stdlib-only. Verdict→glossary remap lives in the follower.

- **equivocation trigger WIRED, production-dormant** (unchanged): `checkConsistency` evaluates shrink →
  fork → equivocation; the growing-pair branch builds the RFC-6962 consistency proof from the LOCAL
  mirror and on `true` → freeze, with a narrow proof-build-error skip (ADR-0006 guard). Golden-tested +
  mutation-proven across the 256-leaf tile boundary. **Dormant on the live path until M2's
  tile-ingestion writer lands** — `PollHub` never writes tiles.

- **structured logging (`slog`)**, **`SQLiteFetcher` + partial-tile mirror CRUD**, **hub_keys cache +
  key readers**, **coverage tracking** (set-once `monitored_since_{size,time}`, ADR-0001),
  **`logclient.Origin`** + golden `TestOrigin`, **config loader**, **realm-registry parser**
  (domains-only), **poll-loop cadence** (single-writer), **freeze + alert-once** (gated on
  `!wasFrozen`), the pure `ConsistencyProofFromTiles` + `InclusionProofFromTiles` builders, the
  `internal/tiles` seam, and now the two `Fetch{Tile,EntryBundle}` transport primitives — all carried
  forward / unchanged.

- `store/{sqlite,schema,checkpoints,tiles,fetcher}.go` — `modernc.org/sqlite v1.46.1`, ADR-0005/0007
  single-writer discipline (WAL, `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL`,
  `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`, `checkpoints`,
  `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`). Store stays a leaf.

- **Schema caveat carried forward**: the cached `HubKey` row carries only `revoked_at` (no
  `valid_from`/`valid_until`), so a full CID-1.0 validity-window re-check from the cache alone is not
  possible. The window IS re-checked every poll via `AcceptCheckpoint`'s `ValidAt`, which gates
  `StatusVerified` before the cache fast path. Documented limitation, not a green-but-wrong path.

- **Missing (remaining M1 connective tissue, outside the Verify bar):**
  - **Real alert transport** — `alertFunc` is a WARN `slog` emit; real delivery (email/webhook) is a
    later step. The `AlertFunc func(int64,string)` seam is unchanged.
  - **Live tile ingestion** (M2 dependency) — `PollHub` never writes tiles, so the wired equivocation
    branch is dormant on the live path. Not an M1 Verify gap (the trigger is golden-tested end-to-end
    against seeded tiles).

- **Fixtures**: `testdata/live/` holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — no tiles or entry bundles (verified `ls`, unchanged). The equivocation,
  inclusion, `LeafHashes`, `RunFsck`, `notecheck`, and both proof-builder golden tests use these
  checkpoints or synthesize in-process `testonly.Tree` inputs; real on-disk tile/entry-bundle fixtures
  are still needed for the inclusion cross-check vs the hub's `IsccLogInclusionProof` and a 256-crossing
  live `fsck` case (M2). **Known stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json`
  fixtures (both `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py` carry sb1's
  PRE-rotation key (`22b08f3e`); the live sb1 signer is `069d0f14`. Captured in `verify_test.go`
  prose/tests (not green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2` (`rfc6962`, `proof` — `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera
  v1.0.2` (`api/layout` in `internal/tiles` — `layout.Range` via `BundleCoords`, plus
  `PartialTileSize`/`TilePath`/`EntriesPath` via the rest of the package incl. `TileCoords` and now the
  `Fetch{Tile,EntryBundle}` primitives; both proof builders; `leafhasher.go`; `tessera/fsck` in
  `logclient/fsck.go`; `tessera/client` re-export under `proofbuilder.go`/`store/fetcher.go`),
  `transparency-dev/formats` (DIRECT — `cmd/notecheck` imports `formats/note`), stdlib `log/slog` +
  `net/http` + `os`/`flag`/`io`. `internal/metrics` is stdlib-only. **Not yet wired**: the full tessera
  `client` proof-builder as a production caller, `nbd-wtf/opentimestamps`.

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: not started — but **every pure prerequisite for the live tile-ingestion writer has now
landed** (all carried forward):
- `internal/tiles` re-exports tessera's tlog-tiles layout math (`TilePath`/`EntriesPath`/
  `PartialTileSize`) + the `IsFull` predicate;
- `internal/tiles/coords.go` — `BundleCoords(treeSize uint64) []BundleCoord`: pure entry-bundle
  coordinate enumeration, delegating boundary math to `layout.Range`. Golden over 7 boundary sizes,
  oracle-confirmed byte-equal. **Unwired export seam**;
- `internal/tiles/coords.go` — `TileCoords(treeSize uint64) []TileCoord`: pure multi-level hash-tile
  coordinate enumeration (climbs tile-levels itself, `Partial` from `PartialTileSize`, stops at the
  root tile). Table-driven golden + oracle cross-check, mutation-proved non-vacuousness. **Unwired
  export seam**;
- `internal/logclient/tilefetch.go` — **`FetchTile` + `FetchEntryBundle` (NEW this iteration)**: the
  transport-only seam between the coordinate enumerations and the mirror writers. Port `FetchCheckpoint`
  verbatim-in-shape (`origin()` → canonical `TilePath`/`EntriesPath` URL → `fetcher.Fetch` → body
  verbatim), `%w`-wrapping the Fetcher's error so a 404's `os.ErrNotExist` survives for the ingestion
  loop to distinguish "tile not served yet" from a hard fault. 6 golden tests (URLs anchored on
  tessera's own layout goldens, not author assertion; `os.ErrNotExist` propagation). **Unwired export
  seam** — first caller is the ingestion writer;
- `internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
  (structural `client.Fetcher`/`fsck.Fetcher`);
- `internal/logclient/proofbuilder.go` — pure `ConsistencyProofFromTiles` + `InclusionProofFromTiles`
  (tile-sourced RFC-6962 inclusion proof, golden over a 300-leaf `testonly.Tree`, oracle
  mutation-proven). **Unwired export seams**;
- `internal/logclient/leafhasher.go` — `LeafHashes(bundle []byte) ([][]byte, error)`;
- `internal/logclient/fsck.go` — `RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error`, the
  root-rebuild conformance gate over the mirror path (in-process **structural self-check**;
  mutation-proven). No production caller yet — needs the live tile-ingestion writer;
- `cmd/notecheck` — the **fully-independent** external signature-parity oracle, compiling in-repo AND
  shelled out in CI on every push.

What remains for M2's Verify bar (all not-started): the **live tile-ingestion writer** (make `PollHub`,
on a verified growing checkpoint, walk `TileCoords(size)` + `BundleCoords(size)`, fetch each
tile/bundle over the new `Fetch{Tile,EntryBundle}` primitives, and write via
`RecordTile`/`RecordEntryBundle` with the path-`Partial`→store-`width` translation — full tile `p==0` →
width 256 via `widthForP`, re-fetching partials every poll per ADR-0005); the **`fsck` root-rebuild
over `SQLiteFetcher`** (first half of M2 Verify, gives `RunFsck` its first production caller); the
**inclusion cross-check** — assert `InclusionProofFromTiles` byte-equals a hub's real
`evidence.IsccLogInclusionProof` for sampled `iscc_id`s (needs captured `IsccLogInclusionProof` + tile
+ entry-bundle fixtures in `testdata/live/`); the `iscc_index` projection writer (schema-agnostic,
`iscc_id → seq` one-to-many); and `inclusion`/`consistency`/`entries` served via a full `ProofBuilder`
from the local store.

## M3 — Trust API + dashboard
**Status**: not started. (The binary has a `net/http` mux serving only `/metrics`; `/`, `/healthz`, and
the REST surface are explicitly out of scope until M3.)

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits); no
`internal/proof` package exists and no WASM build target — the WASM-shareable purity invariant
currently rides on `internal/didweb` (review still runs `GOOS=js GOARCH=wasm go build ./internal/didweb`
as the guard; `internal/tiles` is also WASM-green).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `formats` (DIRECT) + `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged since `2a21f35` (this slice
  adds no dependency — `TilePath`/`EntriesPath` already in the closure via `internal/tiles`).
- **CI is configured and passing.** `.github/workflows/ci.yml` runs one `ubuntu-latest` /
  `CGO_ENABLED=0` job on push + PR to `develop`/`main`: the inlined `mise run check` gate (`go build
  ./...`, `go vet ./...`, `go test ./...`) plus the `cmd/notecheck` oracle shell-out (accept
  `OK sb0.iscc.id/log` exit 0 + reject a one-char-corrupted signature exit 1). **Latest run on
  `develop`: `conclusion: success`** (`gh run list --branch develop` → success, run 27891778400).
  Remote `origin` configured (`github.com/iscc/iscc-monitor`); working branch `develop`; tree clean at
  HEAD `2d7b057`.
- Latest `review` handoff (2026-06-21, "Transport primitives to fetch a tile / entry bundle over the
  Fetcher seam (`logclient.FetchTile` / `FetchEntryBundle`)", verdict **PASS / CONTINUE**) records
  `mise run check` green (all 11 packages `ok`, `go vet`/`gofmt -l .` clean). Oracle gate correctly
  N/A for this transport-only slice (imports only `context`+`fmt`+`internal/tiles`; no
  signature/Merkle/RFC-6962/did:web/fsck path — trust-root packages re-ran uncached, all `ok`). Golden
  URLs anchored on tessera's own layout golden strings. Gate-circumvention + scope-discipline scans
  clean (2 new files, both `internal/logclient`; no caller wired).
- **No open `critical`/`normal` issue.** **One open `low` issue** (loop-skipped): `cmd/notecheck`'s
  `run(vkey, in, out)` has a vestigial `out io.Writer` param never written to — harmless, `go vet`-clean,
  fix when `run` is next touched.
- **CI footnotes** (from learnings/handoff): CI never checks out gitignored `cauldron/`, so a fresh
  `go build ./...` is clean. The inlined CI commands duplicate `mise.toml [tasks.check]` byte-for-byte —
  keep them in lockstep if `[tasks.check]` changes. The `notecheck` build artifact at repo root only
  ever materializes on the ephemeral runner.

## Next Milestone
**M2 — Aggregator.** M1 meets its full Verify bar and is CI-gated & green; the active work is M2's
Verify bar. With both coordinate enumerations (`BundleCoords` + `TileCoords`) AND both transport
primitives (`FetchTile` + `FetchEntryBundle`) now landed and oracle-/golden-proven, every pure
prerequisite for the live tile-ingestion writer exists — that writer is the next slice.

Candidate order:
1. **Live tile-ingestion writer** — make `PollHub`, on a verified growing checkpoint, walk
   `TileCoords(size)` + `BundleCoords(size)`, fetch each tile/bundle via the new
   `FetchTile`/`FetchEntryBundle` primitives, and write via `RecordTile`/`RecordEntryBundle`
   (translating the path-`Partial` to the store's `width` via `widthForP` — full tile `p==0` → 256, not
   0 — re-fetching partials every poll per ADR-0005). This un-dormants the wired equivocation branch on
   the live path AND feeds `SQLiteFetcher.ReadTile`/`ReadEntryBundle` — the unblocker for both crypto
   cross-checks below; both want real tile + entry-bundle fixtures in `testdata/live/`.
2. **`fsck` root-rebuild over `SQLiteFetcher`** — wire `RunFsck` against the mirrored tiles (its first
   production caller), the FIRST half of M2's Verify.
3. **Inclusion cross-check** — assert `InclusionProofFromTiles` byte-equals the hub's own
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s (SECOND half of M2's Verify; needs captured
   `IsccLogInclusionProof` fixtures).
4. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many) + serving
   `inclusion`/`consistency`/`entries` from the local store via a full `ProofBuilder`.
5. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
6. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
