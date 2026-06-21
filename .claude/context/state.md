<!-- assessed-at: 8b61cda5e65f50d3b99b9695ce8c4afb12c37f1a -->

# Project State

## Status: IN_PROGRESS

## Phase: M2 conformance scaffolding — the fully-independent `notecheck` signature-parity oracle now has an in-repo compile path (`cmd/notecheck`), but it is still not wired into CI (none exists), and M2's live-ingestion / inclusion-cross-check / index / served-proofs bar remains not-started

Since the prior assessment (`648158069`) the only source change is the **`cmd/notecheck` oracle port**
(`a24f109`): `cmd/notecheck/main.go` + `main_test.go` (+188 lines, 2 new files) plus a one-line
`go.mod` edit promoting `transparency-dev/formats` from indirect to direct (notecheck imports
`formats/note`). This gives the external signature-parity oracle a real in-module compile path for the
first time — but it has **no CI to run in yet** (`.github/workflows/` still absent), which keeps the
sole open `normal` issue open and the project off DONE. M1 still meets its full Verify bar; M2's
structural `RunFsck` self-check is in place but M2's live-ingestion / inclusion / index / served-proofs
bar is not-started.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary). A real (non-log) alert transport and
live tile ingestion (the latter an M2 dependency) remain as connective tissue but are outside M1's
Verify bar.

- **Verified incrementally** from the prior assessment at `648158069`. The diff `648158069..HEAD` on
  the source side is exactly two new files (`cmd/notecheck/main.go` + `_test.go`) plus a one-line
  `go.mod` indirect→direct promotion of `formats`; every cmd/iscc-monitor/follower/logclient/store/
  didweb/tiles/metrics/metricshttp/registry/config source and `schema.sql` is byte-unchanged and
  carried forward. **Test totals re-grepped**: 9 (didweb) + 33 (logclient) + 48 (store) + 14
  (follower) + 3 (registry) + 4 (config) + 5 (tiles) + 6 (metrics) + 1 (metricshttp) + 1
  (cmd/iscc-monitor) + **3 (cmd/notecheck, new)** = **127 `func Test`** across **11 packages**.

- **`cmd/notecheck` — fully-independent signature-parity oracle (NEW)**: reads `--vkey` + checkpoint
  text on stdin, runs `transparency-dev/formats/note.NewVerifier` + `golang.org/x/mod/sumdb/note.Open`,
  applies the strict reject (`len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0`) that mirrors
  `internal/logclient/verify.go`, prints `OK <name>`/exit 0 or exits 1 (verify fail) / 2 (bad vkey or
  stdin read). It is the EXTERNAL parity check for the monitor's own crypto path — distinct from the
  in-process `RunFsck` self-check, which shares the monitor's code. Three tests (golden accept +
  corrupted-body reject + bad-vkey exit-2), mutation-proven non-vacuous by `review`. Its CI consumer
  does not yet exist (see Quality gates).

- **`/metrics` served + wired** (carried forward, unchanged): `internal/metricshttp/handler.go` —
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
  `!wasFrozen`), the pure `ConsistencyProofFromTiles` builder, and the `internal/tiles` seam — all
  unchanged and carried forward.

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
  `LeafHashes`, `RunFsck`, AND the new `notecheck` golden test all use these checkpoints or synthesize
  inputs in-process; real on-disk tile/entry-bundle fixtures are still needed for the inclusion
  cross-check vs the hub's `IsccLogInclusionProof` and a 256-crossing live `fsck` case (M2).
  **Known stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json` fixtures (both
  `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py` carry sb1's PRE-rotation key
  (`22b08f3e`); the live sb1 signer is `069d0f14`. Captured in `verify_test.go` prose/tests (not
  green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2`, `transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`; `leafhasher.go`;
  `tessera/fsck` in `logclient/fsck.go`), **`transparency-dev/formats` (now a DIRECT require —
  `cmd/notecheck` imports `formats/note`)**, stdlib `log/slog` + `net/http` + `os`/`flag`/`io`.
  `internal/metrics` is stdlib-only. **Not yet wired**: the rest of tessera (`client` proof-builder),
  `nbd-wtf/opentimestamps`.

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: not started — but **six prerequisite slices have now landed** (five carried forward + one
new this iteration):
- `internal/tiles` re-exports tessera's tlog-tiles layout math + the `IsFull` predicate;
- `internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
  (structural `client.Fetcher`/`fsck.Fetcher`);
- `internal/logclient/proofbuilder.go` provides a pure `ConsistencyProofFromTiles`;
- `internal/logclient/leafhasher.go` — `LeafHashes(bundle []byte) ([][]byte, error)`;
- `internal/logclient/fsck.go` — `RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error`, the
  root-rebuild conformance gate over the mirror path (in-process **structural self-check**, shares the
  monitor's `LeafHashes`/RFC-6962 code; mutation-proven). No production caller yet — its first caller
  needs the live tile-ingestion writer;
- **NEW** `cmd/notecheck` — the **fully-independent** external signature-parity oracle now has an
  in-module compile path (was only a module-less `main.go` under gitignored `cauldron/`). It is the
  truly-external counterpart to `RunFsck`; it just needs a CI consumer to actually gate the trust root
  on every push.

What remains for M2's Verify bar (all not-started): the **live tile-ingestion writer** (make `PollHub`
mirror real tiles/bundles — also un-dormants the equivocation branch and enables a *production*
`RunFsck` caller); the **inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof`
(needs real captured `IsccLogInclusionProof` fixtures + an inclusion `ProofBuilder` — the SECOND half
of M2's Verify); the `iscc_index` projection writer (schema-agnostic, `iscc_id → seq` one-to-many);
and `inclusion`/`consistency`/`entries` served via a full `ProofBuilder` from the local store.

## M3 — Trust API + dashboard
**Status**: not started. (The binary has a `net/http` mux serving only `/metrics`; `/`, `/healthz`, and
the REST surface are explicitly out of scope until M3.)

## WASM verifier · OTS anchoring
**Status**: not started.

## Quality gates
**Status**: green (as recorded by `review`; not re-run here) — but **no CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `formats` (now DIRECT) + `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable. Latest `review` handoff (2026-06-21, "Port `notecheck` into the monitor
  module as `cmd/notecheck`", verdict **PASS / CONTINUE**) records the gate green at HEAD `8b61cda`:
  `mise run check` green (build + vet + test, all 11 packages `ok`, incl. `cmd/notecheck`); `gofmt -l
  .` clean outside gitignored `cauldron/`; `go test -run TestNotecheck ./cmd/notecheck` PASS uncached
  (golden accept + corrupted-body reject + bad-vkey exit-2); `go mod tidy && git diff --exit-code --
  go.mod go.sum` clean (idempotent after the `formats` direct-require promotion); `go mod verify` → all
  verified. Oracle gate **APPLIED** (this slice IS the external signature oracle): `derive_vkey.py`
  reproduces the embedded `sb0VKey` byte-for-byte (`…40b74463…`); the built binary verifies
  `sb0.iscc.id_checkpoint` (`OK sb0.iscc.id/log`, exit 0) and rejects a one-byte-corrupted checkpoint
  (exit 1). **Mutation-proven**: short-circuiting `run` to always-accept fails both negative tests.
- **One open `normal` issue**: no `.github/workflows/` (verified `ls`) — so the now-compilable
  `notecheck` oracle and any build/test/format/tidy gate still run only locally, never in CI. This is
  the natural next slice and is now fully UNBLOCKED (CI need only `go build ./cmd/notecheck` + shell
  the binary out against a captured checkpoint, plus a `mise run check` job).
- **One open `low` issue** (loop-skipped): `cmd/notecheck`'s `run(vkey, in, out)` has a vestigial
  `out io.Writer` param never written to. Harmless, `go vet`-clean; fix when `run` is next touched.
- Remote `origin` configured (`github.com/iscc/iscc-monitor`); working branch is **`develop`**; tree
  clean at HEAD `8b61cda` (review pushed on PASS). **No `.github/workflows/` — no CI configured**;
  `gh run list --branch develop` returns `[]` (no runs), confirming no CI applies. When CI is wired it
  must avoid `go build ./...` over the gitignored `cauldron/` reference trees — `./cmd/notecheck`
  sidesteps that cleanly.

## Next Milestone
**M2 — Aggregator.** M1 meets its full Verify bar, the `RunFsck` root-rebuild self-check is landed, and
the fully-independent `notecheck` oracle now compiles in-repo. The only gate-relevant gap is that
**none of it runs in CI** — wiring CI takes priority before further M2 feature work.

Candidate order:
1. **Wire CI + `notecheck`** (the sole open `normal` issue, the only gate-relevant gap) —
   `.github/workflows/` running `mise run check` + `go build ./cmd/notecheck` and shelling the binary
   out against a captured checkpoint (assert `OK <name>` + exit 0, exit 1 on a corrupted one). The
   tree is tidy-clean, so a `go mod tidy && git diff --exit-code` CI step will pass.
2. **Inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof` — the SECOND half of
   M2's Verify bar; needs real captured `IsccLogInclusionProof` fixtures + an inclusion `ProofBuilder`.
3. **Live tile-ingestion writer** — make `PollHub` mirror real tiles/bundles, which un-dormants the
   wired equivocation branch on the live path AND gives `RunFsck` its first production caller.
4. **`iscc_index` projection writer** + serving `inclusion`/`consistency`/`entries` from the local
   store via a full `ProofBuilder`.
5. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
6. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
