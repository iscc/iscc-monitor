<!-- assessed-at: 64b333993579dd10f612ee4dd0ae2f2f5523e62f -->

# Project State

## Status: IN_PROGRESS

## Phase: M1 complete + gated in CI — M2 (Aggregator) is the active milestone. CI is now wired and green, closing the project's last gate-relevant gap; six M2 prerequisite slices have landed, but M2's live-ingestion / inclusion-cross-check / index / served-proofs bar remains not-started.

Since the prior assessment (`8b61cda`) the **only source-relevant change is the new
`.github/workflows/ci.yml`** (`64b3339`/`beacc98`): zero Go, `go.mod`/`go.sum`, or `schema.sql` bytes
changed (verified `git diff 8b61cda..HEAD --stat -- '*.go' go.mod go.sum '*.sql'` is empty). CI now
runs the `mise run check` gate (`go build`/`vet`/`test` under `CGO_ENABLED=0`) plus the
fully-independent `cmd/notecheck` signature oracle (accept the real checkpoint + reject a corrupted
one) on every push to `develop`/`main`. The latest run on `develop` concluded **success**
(`gh run list` → `conclusion: success`), the sole open `normal` issue (no CI) is **closed**, and only
one loop-skipped `low` issue remains. The project stays off DONE because M2 → OTS are not yet built.

## M1 — Read-only Monitor
**Status**: met (all Verify criteria satisfied — `origin`/`vkey` golden, all three triggers
golden-tested end-to-end with freeze + alert-once + restart survival, coverage tracked, structured
logs, `/metrics` served over HTTP and collected by the binary). **Now also gated in CI.** A real
(non-log) alert transport and live tile ingestion (the latter an M2 dependency) remain connective
tissue outside M1's Verify bar.

- **Verified incrementally** from `8b61cda`. The diff `8b61cda..HEAD` touches no Go/`go.mod`/`go.sum`/
  `schema.sql` byte — only `.github/workflows/ci.yml` (new) and context md. Every cmd/iscc-monitor/
  follower/logclient/store/didweb/tiles/metrics/metricshttp/registry/config source + `cmd/notecheck` +
  `schema.sql` is carried forward byte-unchanged. **Test totals re-grepped**: 9 (didweb) + 33
  (logclient) + 48 (store) + 14 (follower) + 3 (registry) + 4 (config) + 5 (tiles) + 6 (metrics) + 1
  (metricshttp) + 1 (cmd/iscc-monitor) + 3 (cmd/notecheck) = **127 `func Test`** across **11 packages**
  (30 `_test.go` files).

- **`cmd/notecheck` — fully-independent signature-parity oracle (now CI-gated)**: reads `--vkey` +
  checkpoint text on stdin, runs `transparency-dev/formats/note.NewVerifier` + `golang.org/x/mod/
  sumdb/note.Open`, applies the strict reject (`len(n.Sigs)==0 || len(n.UnverifiedSigs)!=0`) that
  mirrors `internal/logclient/verify.go`, prints `OK <name>`/exit 0 or exits 1 (verify fail) / 2 (bad
  vkey or stdin read). It is the EXTERNAL parity check for the monitor's own crypto path — distinct
  from the in-process `RunFsck` self-check, which shares the monitor's code. Three tests (golden accept
  + corrupted-body reject + bad-vkey exit-2). **CI shells the built binary out** against
  `testdata/live/sb0.iscc.id_checkpoint`, asserting the exact `OK sb0.iscc.id/log` accept AND a
  one-char-flipped-signature reject (exit 1) — review mutation-proved the reject guard catches a
  green-but-wrong always-accept stub.

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
  `LeafHashes`, `RunFsck`, and the `notecheck` golden/CI tests all use these checkpoints or synthesize
  inputs in-process; real on-disk tile/entry-bundle fixtures are still needed for the inclusion
  cross-check vs the hub's `IsccLogInclusionProof` and a 256-crossing live `fsck` case (M2).
  **Known stale-fixture drift, still not acted on:** the `sb1.amlet.id_did.json` fixtures (both
  `internal/didweb/` and `internal/logclient/`) and `derive_vkey.py` carry sb1's PRE-rotation key
  (`22b08f3e`); the live sb1 signer is `069d0f14`. Captured in `verify_test.go` prose/tests (not
  green-but-wrong), but the did.json fixtures remain stale.

- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle
  v0.0.2`, `transparency-dev/tessera v1.0.2` (`tiles/layout.go`; `proofbuilder.go`; `leafhasher.go`;
  `tessera/fsck` in `logclient/fsck.go`; `tessera/client` re-export under `proofbuilder.go`/
  `store/fetcher.go`), `transparency-dev/formats` (DIRECT — `cmd/notecheck` imports `formats/note`),
  stdlib `log/slog` + `net/http` + `os`/`flag`/`io`. `internal/metrics` is stdlib-only. **Not yet
  wired**: the full tessera `client` proof-builder for inclusion, `nbd-wtf/opentimestamps`.

- **Verify criteria status — ALL MET**: `origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `VerifierKey` byte-match — met. All three triggers met end-to-end in golden tests (synthetic shrink,
  fork, AND equivocation each → correct `violations.kind` + `frozen=1` + exactly-one-alert +
  other-hubs-unaffected + evidence-survives-restart). Coverage (`monitored_since`) — tracked + asserted
  set-once. Structured logs — met. `/metrics` — served over HTTP and collected by the binary.

## M2 — Aggregator
**Status**: not started — but **six prerequisite slices have landed** (all carried forward):
- `internal/tiles` re-exports tessera's tlog-tiles layout math + the `IsFull` predicate;
- `internal/store/{tiles,fetcher}.go` provides the partial-tile mirror CRUD + `SQLiteFetcher`
  (structural `client.Fetcher`/`fsck.Fetcher`);
- `internal/logclient/proofbuilder.go` provides a pure `ConsistencyProofFromTiles`;
- `internal/logclient/leafhasher.go` — `LeafHashes(bundle []byte) ([][]byte, error)`;
- `internal/logclient/fsck.go` — `RunFsck(ctx, vkey, origin string, f fsck.Fetcher) error`, the
  root-rebuild conformance gate over the mirror path (in-process **structural self-check**; shares the
  monitor's `LeafHashes`/RFC-6962 code; mutation-proven). No production caller yet — its first caller
  needs the live tile-ingestion writer;
- `cmd/notecheck` — the **fully-independent** external signature-parity oracle, now compiling in-repo
  AND shelled out in CI on every push, so the trust root is gated, not just locally reproducible.

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
**Status**: not started. (`nbd-wtf/opentimestamps` not imported; no `internal/proof` package / WASM
build target yet — the WASM-shareable purity invariant currently rides on `internal/didweb`.)

## Quality gates
**Status**: green — **and now enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line; requires
  `formats` (DIRECT) + `merkle v0.0.2` + `tessera v1.0.2` + `x/mod v0.33.0` + `sqlite v1.46.1`);
  `mise run check` runnable.
- **CI is configured and passing.** `.github/workflows/ci.yml` (new this iteration) runs one
  `ubuntu-latest` / `CGO_ENABLED=0` job on push + PR to `develop`/`main`: the inlined `mise run check`
  gate (`go build ./...`, `go vet ./...`, `go test ./...`) plus the `cmd/notecheck` oracle shell-out
  (accept `OK sb0.iscc.id/log` exit 0 + reject a one-char-corrupted signature exit 1). **Latest run on
  `develop`: `conclusion: success`** (`gh run list --branch develop` → success). Remote `origin`
  configured (`github.com/iscc/iscc-monitor`); working branch `develop`; tree clean at HEAD `64b3339`.
- Latest `review` handoff (2026-06-21, "Wire CI — `mise run check` + the `notecheck` signature-parity
  oracle shell-out", verdict **PASS / CONTINUE**) records the gate green at HEAD `64b3339`, with the
  reject guard mutation-proven (a stdin-draining always-accept stub makes CI exit 1). The single
  pre-push-unmeetable criterion (live CI run) is now satisfiable and is **confirmed success** post-push.
- **No open `critical`/`normal` issue** (the no-CI issue is closed). **One open `low` issue**
  (loop-skipped): `cmd/notecheck`'s `run(vkey, in, out)` has a vestigial `out io.Writer` param never
  written to — harmless, `go vet`-clean, fix when `run` is next touched.
- **CI footnotes** (from learnings/handoff): CI never checks out gitignored `cauldron/`, so a fresh
  `go build ./...` is clean. The inlined CI commands duplicate `mise.toml [tasks.check]` byte-for-byte —
  a KISS choice; keep them in lockstep if `[tasks.check]` changes. The `notecheck` build artifact at
  repo root is not gitignored but only ever materializes on the ephemeral runner.

## Next Milestone
**M2 — Aggregator.** M1 meets its full Verify bar and is now CI-gated; the last gate-relevant gap (no
CI) is closed and CI is green. The active work is M2's Verify bar.

Candidate order:
1. **Inclusion cross-check** vs the hub's own `evidence.IsccLogInclusionProof` — the SECOND half of
   M2's Verify bar and the first independent conformance check beyond signature parity. Needs real
   captured `IsccLogInclusionProof` fixtures + `proof.VerifyInclusion` over the mirrored tiles.
2. **Live tile-ingestion writer** — make `PollHub` mirror real tiles/bundles, which un-dormants the
   wired equivocation branch on the live path AND gives `RunFsck` its first production caller (pairing
   the inclusion cross-check with the `fsck` root-rebuild over `SQLiteFetcher`).
3. **`iscc_index` projection writer** (schema-agnostic, `iscc_id → seq` one-to-many) + serving
   `inclusion`/`consistency`/`entries` from the local store via a full `ProofBuilder`.
4. **sb1 fixture refresh** (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
   — its own trust-root step that re-arms the oracle gate.
5. **Real alert transport** — replace the WARN `slog` placeholder with email/webhook delivery to fully
   close M1's alert path.
6. Optional CI hardening: an additive `go mod tidy && git diff --exit-code` drift gate.
