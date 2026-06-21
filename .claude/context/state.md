<!-- assessed-at: 679d4fda5a3a0722886db74529577f3233a6a0d1 -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + CI-gated & green — M2 (Aggregator) is the active milestone. Eight M2 prerequisite slices have now landed (the pure `tiles.BundleCoords` entry-bundle coordinate enumeration is the newest), but M2's Verify bar — live tile/bundle ingestion, the inclusion cross-check vs the hub's own proof, `fsck` root-rebuild over the store, `iscc_index`, and store-served proofs — remains not-started.

Since the prior assessment (`ed4668a`) the **only source-relevant change is one additive pure seam**:
`internal/tiles/coords.go` gained `BundleCoord{Index uint64, Partial uint8}` + `BundleCoords(treeSize
uint64) []BundleCoord` (delegating to `layout.Range(0, treeSize, treeSize)`) plus its golden test
`coords_test.go` (`8de24c1`, reviewed PASS in `679d4fd`). `git diff ed4668a..HEAD -- go.mod go.sum
schema.sql` is empty; the diff touches no other `.go` byte (only those two `tiles` files + context
md). It is an **unwired export seam** (`grep` confirms `BundleCoords` has no production caller — it
appears only in its own definition + doc comment + test). The project stays off DONE because M2 → OTS
are not yet built.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary). **CI-gated & green.** A real (non-log)
alert transport and live tile ingestion (the latter an M2 dependency) remain connective tissue outside
M1's Verify bar.

- **Verified incrementally** from `ed4668a`. The diff `ed4668a..HEAD` touches no
  Go/`go.mod`/`go.sum`/`schema.sql` byte outside `internal/tiles/coords.go` (+`coords_test.go`). Every
  cmd/iscc-monitor / follower / logclient / store / didweb / tiles (sans the one new file) / metrics /
  metricshttp / registry / config source + `cmd/notecheck` + `schema.sql` is carried forward
  byte-unchanged.
- **Test totals re-grepped at HEAD**: 9 (didweb) + 36 (logclient) + 48 (store) + 14 (follower) + 3
  (registry) + 4 (config) + 7 (tiles, +2 from `BundleCoords`) + 6 (metrics) + 1 (metricshttp) + 1
  (cmd/iscc-monitor) + 3 (cmd/notecheck) = **132 `func Test`** across **11 packages** (**32** `_test.go`
  files).

- **`cmd/notecheck` — fully-independent signature-parity oracle (CI-gated)** (unchanged): reads
  `--vkey` + checkpoint text on stdin, runs `transparency-dev/formats/note.NewVerifier` +
  `golang.org/x/mod/sumdb/note.Open`, applies the strict reject (`len(n.Sigs)==0 ||
  len(n.UnverifiedSigs)!=0`) mirroring `internal/logclient/verify.go`, prints `OK <name>`/exit 0 or
  exits 1 (verify fail) / 2 (bad vkey or stdin read). It is the EXTERNAL parity check for the monitor's
  own crypto path — distinct from the in-process `RunFsck` self-check. Three tests. **CI shells the
  built binary out** against `testdata/live/sb0.iscc.id_checkpoint`, asserting the exact
  `OK sb0.iscc.id/log` accept AND a one-char-flipped-signature reject (exit 1).

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
  `!wasFrozen`), the pure `ConsistencyProofFromTiles` + `InclusionProofFromTiles` builders, and the
  `internal/tiles` seam — all unchanged and carried forward.

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
  v1.0.2` (`api/layout` in `internal/tiles` — now also `layout.Range` via `coords.go`; both proof
  builders; `leafhasher.go`; `tessera/fsck` in `logclient/fsck.go`; `tessera/client` re-export under
  `proofbuilder.go`/`store/fetcher.go`), `transparency-dev/formats` (DIRECT — `cmd/notecheck` imports
  `formats/note`), stdlib `log/slog` + `net/http` + `os`/`flag`/`io`. `internal/metrics` is stdlib-only.
  **Not yet wired**: the full tessera `client` proof-builder as a production caller,
  `nbd-wtf/opentimestamps`.

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: not started — but **eight prerequisite slices have landed** (all carried forward):
- `internal/tiles` re-exports tessera's tlog-tiles layout math (`TilePath`/`EntriesPath`/
  `PartialTileSize`) + the `IsFull` predicate;
- `internal/tiles/coords.go` — **`BundleCoords(treeSize uint64) []BundleCoord` (NEW this iteration)**:
  pure entry-bundle coordinate enumeration naming which bundles a complete mirror of a tree of size N
  must hold, delegating boundary math to `layout.Range(0, treeSize, treeSize)`. Table-driven golden over
  7 boundary sizes (review re-ran the real `layout.Range` in a throwaway module → byte-equal: `0→[]`,
  `1→[{0,1}]`, `255→[{0,255}]`, `256→[{0,0}]`, `257→[{0,0},{1,1}]`, `300→[{0,0},{1,44}]`,
  `513→[{0,0},{1,0},{2,1}]`). **Unwired export seam** — first caller is the M2 fetch loop;
- `internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
  (structural `client.Fetcher`/`fsck.Fetcher`);
- `internal/logclient/proofbuilder.go` — pure `ConsistencyProofFromTiles`;
- `internal/logclient/proofbuilder.go` — `InclusionProofFromTiles(ctx, fetch, index, size)`: tile-sourced
  RFC-6962 inclusion proof, golden-tested over a 300-leaf `testonly.Tree` boundary, oracle mutation-proven.
  **Unwired export seam**;
- `internal/logclient/leafhasher.go` — `LeafHashes(bundle []byte) ([][]byte, error)`;
- `internal/logclient/fsck.go` — `RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error`, the
  root-rebuild conformance gate over the mirror path (in-process **structural self-check**;
  mutation-proven). No production caller yet — its first caller needs the live tile-ingestion writer;
- `cmd/notecheck` — the **fully-independent** external signature-parity oracle, compiling in-repo AND
  shelled out in CI on every push.

What remains for M2's Verify bar (all not-started): the **live tile-ingestion writer** (make `PollHub`
mirror real tiles/bundles — also un-dormants the equivocation branch and gives `RunFsck` its first
production `fsck` root-rebuild caller; the hash-tile multi-level coordinate enumeration — the sibling of
`BundleCoords` — is the immediate prerequisite); the **`fsck` root-rebuild over `SQLiteFetcher`** (first
half of M2 Verify); the **inclusion cross-check** — assert `InclusionProofFromTiles` byte-equals a hub's
real `evidence.IsccLogInclusionProof` for sampled `iscc_id`s (needs captured `IsccLogInclusionProof` +
tile + entry-bundle fixtures in `testdata/live/`); the `iscc_index` projection writer (schema-agnostic,
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
  `mise run check` runnable. `go.mod`/`go.sum` byte-unchanged since `ed4668a` (`layout.Range` already in
  the closure via `internal/tiles`).
- **CI is configured and passing.** `.github/workflows/ci.yml` runs one `ubuntu-latest` /
  `CGO_ENABLED=0` job on push + PR to `develop`/`main`: the inlined `mise run check` gate (`go build
  ./...`, `go vet ./...`, `go test ./...`) plus the `cmd/notecheck` oracle shell-out (accept
  `OK sb0.iscc.id/log` exit 0 + reject a one-char-corrupted signature exit 1). **Latest run on
  `develop`: `conclusion: success`** (`gh run list --branch develop` → success,
  run 27891292666). Remote `origin` configured (`github.com/iscc/iscc-monitor`); working branch
  `develop`; tree clean at HEAD `679d4fd`.
- Latest `review` handoff (2026-06-21, "Pure entry-bundle coordinate enumeration for a tree of size N
  (`tiles.BundleCoords`)", verdict **PASS / CONTINUE**) records `mise run check` green (all 11 packages
  `ok`, `go vet`/`gofmt -l .` clean). Oracle gate correctly N/A for this pure path-math slice; review
  re-ran `layout.Range(0,N,N)` externally over all 7 vectors → byte-equal goldens (tessera ground truth,
  not author-asserted). Gate-integrity + scope-discipline scans clean (1 prod file + 1 test file).
- **No open `critical`/`normal` issue.** **One open `low` issue** (loop-skipped): `cmd/notecheck`'s
  `run(vkey, in, out)` has a vestigial `out io.Writer` param never written to — harmless, `go vet`-clean,
  fix when `run` is next touched.
- **CI footnotes** (from learnings/handoff): CI never checks out gitignored `cauldron/`, so a fresh
  `go build ./...` is clean. The inlined CI commands duplicate `mise.toml [tasks.check]` byte-for-byte —
  keep them in lockstep if `[tasks.check]` changes. The `notecheck` build artifact at repo root only
  ever materializes on the ephemeral runner.

## Next Milestone
**M2 — Aggregator.** M1 meets its full Verify bar and is CI-gated & green; the active work is M2's
Verify bar. With both proof builders and the entry-bundle coordinate enumeration now landed and
oracle-proven, the next slice is the **hash-tile (multi-level) coordinate enumeration** (the sibling
`BundleCoords` deferred — needs its own per-level loop via `PartialTileSize(level, index, treeSize)`,
since no single `layout.Range` covers all levels), which together with `BundleCoords` unblocks the
**live tile-ingestion writer**.

Candidate order:
1. **Hash-tile coordinate enumeration** — the sibling of `BundleCoords`; per-level loop over tile-levels
   `0, 1, 2, …` (per-level full-tile counts + the top partial via `PartialTileSize`) until one root tile.
2. **Live tile-ingestion writer** — make `PollHub` mirror real tiles/bundles (consuming both coordinate
   enumerations), which un-dormants the wired equivocation branch on the live path AND feeds
   `SQLiteFetcher.ReadTile`/`ReadEntryBundle` for proof nodes. The unblocker for both crypto
   cross-checks below; both want real tile + entry-bundle fixtures in `testdata/live/`.
3. **`fsck` root-rebuild over `SQLiteFetcher`** — wire `RunFsck` against the mirrored tiles (its first
   production caller), the FIRST half of M2's Verify.
4. **Inclusion cross-check** — assert `InclusionProofFromTiles` byte-equals the hub's own
   `evidence.IsccLogInclusionProof` for sampled `iscc_id`s (SECOND half of M2's Verify; needs captured
   `IsccLogInclusionProof` fixtures).
5. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many) + serving
   `inclusion`/`consistency`/`entries` from the local store via a full `ProofBuilder`.
6. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
7. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
