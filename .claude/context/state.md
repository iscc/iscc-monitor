<!-- assessed-at: 4df69fb2cbbd5fdf1e5181cc1637200d5b1dd3c8 -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — the tlog-tiles mirror HTTP arc is complete; current work
drains the ADR-0006 trust-path / follower-locality backlog. verify-for-me, the server-rendered
dashboard, the log browser, the WASM verifier, and OTS anchoring all remain unstarted.

The monitor follows + mirrors + verifies hubs (M1 met) and serves all three computed proofs per hub
from the local mirror (M2 met: `/inclusion`, `/consistency`, `/entries`). M3 has the raw tlog-tiles
mirror fully plumbed (CORS → Cache-Control → conditional GET). The bulk of M3 (verify-for-me,
dashboard, log browser), the WASM verifier, and OTS anchoring remain.

Incremental review of `86bc118..HEAD` (HEAD `4df69fb` on `develop`). The diff touches **two
production files** — the new `internal/logclient/checkconsistency.go` plus a slimmed
`internal/follower/follower.go` — and one new test (`checkconsistency_test.go`). Verified from source:
the shrink→fork→equivocation `switch`, the RFC-6962 consistency-proof build, and the narrow
missing-tile swallow now live in **one pure `logclient.CheckConsistency`** (`checkconsistency.go`,
imports **only** `context`+`fmt` — verified by reading the file; `go list -deps ./internal/logclient`
confirms **no `internal/store` edge**, so the dependency direction stays follower → logclient). The
follower's `checkConsistency` (follower.go:402) is reduced to the persistence concern: a
`CheckpointAt(prevSize)` store read, then `logclient.CheckConsistency(ctx, fetcher.ReadTile, …)`. The
equivocation proof is still sourced from the LOCAL mirror via the injected `SQLiteFetcher.ReadTile`
(never re-hits the hub); a proof-build/missing-tile fault is swallowed to `(false, "", nil)` inside
`CheckConsistency` (the ADR-0006 false-positive guard, with a dedicated `wantViol:false` test case),
while a genuine store failure surfaces via `CheckpointAt`. Latest `review` handoff (2026-06-21,
"Collapse the self-consistency decision into `logclient.CheckConsistency`") is **PASS / CONTINUE**:
`mise run check` green (15 packages `ok`, `gofmt -l .` empty), the new `TestCheckConsistency` table
(8 cases) reviewer-mutation-proven non-vacuous (suppress-equivocation → growing-split-view FAILS;
`fork:=false` → fork FAILS), oracle/conformance gate re-satisfied, gate-integrity scan clean,
scope-clean (exactly 2 production files + 1 test = `next.md`'s scope). **CI green at HEAD `4df69fb`**
(develop run 27898545770, `conclusion: success`).

**Branch note:** active work happens on `develop` (HEAD `4df69fb`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward; this slice refactored the self-consistency *verdict* path
(extracted the shrink/fork/equivocation decision into a pure `logclient.CheckConsistency`) without
changing any observable behavior: branch order, guards, error-vs-violation discipline, and the
LOCAL-mirror equivocation proof source are byte-faithful to the prior follower inline code. All
Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers golden-tested end-to-end
with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over
HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **207 `func Test`** across `cmd/` + `internal/` (+1), **48** `_test.go`
  files (+1) — the slice added `internal/logclient/checkconsistency_test.go` (the 8-case
  `TestCheckConsistency` table) and slimmed the follower's inline-consistency assertions in place.
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,corsmw,didweb,follower,
  healthz,logclient,metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}` (13 internal
  packages). Module path `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation, now evaluated inside
  `logclient.CheckConsistency`. The growing-pair equivocation builds the RFC-6962 consistency proof
  from the LOCAL mirror (`ConsistencyProofFromTiles` over the injected fetch) → freeze on `true`
  (ADR-0006). **Frozen hubs are evidence-only** on clean re-polls; fork re-detection on a frozen hub
  re-records evidence without re-alerting, proven through a real second `PollHub`.
- **`AcceptCheckpoint` 4-way verdict** (`logclient/accept.go`) returns `(Status, CheckpointInfo,
  VerifiedContext{VKey,Key}, error)`, the context populated **only** on `StatusVerified` (zero on
  `Unverified`/`Unresolvable`/`Rotated`). `PollHub` threads it into the hub-key cache upsert and
  `fsckMirror`; **no `ResolveVerifierKey` caller remains in production `follower.go`** (re-verified by
  grep). Reuse is per-poll only (ADR-0009).
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real
  sb0 checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics`, `/healthz`, the per-hub mirror, and the per-hub `/inclusion` + `/consistency` + `/entries`
  proof routes all ride one `*http.ServeMux` on a single listener, wrapped once in `corsmw.Handler`.
  `internal/metrics` leaf, `slog` structured logging, `SQLiteFetcher` + partial-tile mirror CRUD,
  hub_keys cache, coverage tracking (set-once, ADR-0001), `logclient.Origin` + golden `TestOrigin`,
  config loader, realm-registry parser (domains-only), poll-loop cadence (single-writer), freeze +
  alert-once.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL, `busy_timeout=5000`,
  `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql` (`hubs`, `hub_keys`,
  `checkpoints`, `violations`, `tiles`, `entry_bundles`, `iscc_index`, `follow_state`, `ots`).
  `schema.sql` byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery is
  later. The warm-path's irreducible second `did.json` resolve (`AcceptCheckpoint` always re-fetches
  even on a warm `hub_keys` cache) is a larger design change (needs a pre-resolved-key
  `AcceptCheckpoint` variant), not on the Verify bar.
- **Fixtures**: `testdata/live/` still holds **only the two checkpoints** (`sb0.iscc.id_checkpoint`,
  `sb1.amlet.id_checkpoint`) — **no tiles or entry bundles** (re-verified by `ls`). All tile/fsck/
  inclusion/consistency/entries/projection/serve tests run against in-process fixtures. Real tile/
  entry-bundle + `IsccLogInclusionProof` fixtures remain a soft prerequisite for an *inbound* hub-
  evidence transport. Known stale `sb1.amlet.id` did.json drift (pre-rotation key) captured in tests;
  fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`, `transparency-dev/merkle`
  (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`), `transparency-dev/tessera` (`api`, `api/layout`,
  both proof builders, `leafhasher`, `tessera/fsck` with a production caller via `fsckMirror`,
  `tessera/client` re-export, `api.EntryBundle.UnmarshalText`), `transparency-dev/formats` (DIRECT —
  `cmd/notecheck`), stdlib `slog`/`net/http`/`os`/`crypto/sha256`. **Not yet wired**:
  `nbd-wtf/opentimestamps` (re-verified: grep → no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — carried forward unchanged. Both Verify criteria are exercised (fsck root-rebuild
WIRED on every verified non-frozen poll; inclusion cross-check conformance-tested over the real verified
mirror), and the served proof surface is complete: all three computed proofs — `inclusion`,
`consistency`, `entries` — served from the local mirror, never re-hitting the hub. This slice did not
touch M2's proof or mirror logic; the equivocation consistency-proof build (`ConsistencyProofFromTiles`,
shared with M2) merely moved from inline follower code into `logclient.CheckConsistency`.
- **fsck root-rebuild (WIRED, runs every verified non-frozen poll):** `fsckMirror` (`follower.go`, after
  `ingestTiles`, before `recordVerdict`) builds a read-only `store.SQLiteFetcher` and calls
  `logclient.RunFsck`, comparing the rebuilt RFC-6962 root to the signed checkpoint root, reusing the
  verifier key `AcceptCheckpoint` resolved this poll. A frozen hub's clean re-poll skips `fsckMirror`.
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` folds the
  raw bundle via `logclient.BundleProjections` → `store.RecordProjections`.
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED):** `internal/logclient/inclusioncheck.go` closed
  against M2's Verify bar by `internal/follower/inclusion_test.go`.
- **raw tlog-tiles HTTP read surface (BUILT + WIRED):** `internal/tilesserve/handler.go` routes the
  three canonical iscc-log §9 paths to raw mirror BLOBs verbatim, per hub, with per-route `Cache-Control`
  and a strong content ETag + `If-None-Match`→`304` conditional GET on every 200.
- **computed inclusion/consistency/entries proofs (BUILT + WIRED):** `internal/proofserve/handler.go`
  `serveInclusion`/`serveConsistency`/`serveEntries` serve from the mirror against `LastSize` /
  `FollowState.LastSize`, shaped like the hub's `IsccLogInclusionProof` (inclusion) and
  `application/octet-stream` schema-agnostic record bytes (entries, ADR-0008).
- **`CheckpointAt` determinism (in place):** `internal/store/checkpoints.go` `CheckpointAt` is
  `… ORDER BY rowid LIMIT 1`, returning the lowest-rowid (prior accepted) root deterministically. The
  follower's `checkConsistency` now reads `prevRoot`/`prevFound`/`prevRaw` straight from this call.

**What remains for M2:** nothing on the Verify bar — M2 is met.

## M3 — Trust API + dashboard
**Status**: **in progress**. The raw tlog-tiles mirror is fully plumbed: (1) CORS on every public GET
via the single `corsmw.Handler` wrap; (2) per-route `Cache-Control`; (3) a strong content ETag +
wildcard/exact `If-None-Match`→`304` conditional GET on every 200. The binary's mux serves `/metrics`,
`/healthz`, the per-hub raw tlog-tiles mirror subtrees, and the per-hub `/inclusion` + `/consistency` +
`/entries` computed-proof endpoints — all behind CORS. **Still absent (re-verified):** conditional-GET /
cache policy on the size-dependent proof surfaces (`/inclusion`/`/consistency`/`/entries`, tied to
`LastSize` — `internal/proofserve` carries no ETag/Cache-Control); `verify-for-me` verdict surface (no
`html/template`/`text/template` import anywhere — grep clean); server-rendered dashboard (status/
coverage/lag/violations/OTS); `/` landing; and the log browser.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js`
in source — grep clean). The WASM-shared verifier seam continues to ride on `internal/didweb`
(`accept.go`'s `internal/didweb` import is the WASM-pure parser, not the networked resolver — purity
invariant holds; the new store-free `logclient.CheckConsistency` could share a WASM seam later).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. `schema.sql` byte-unchanged this slice.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for
  HEAD `4df69fb`: `conclusion: success`** (run 27898545770).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (15 packages
  `ok`, `go vet`/`gofmt -l .` clean), the `CheckConsistency` extraction reviewer-mutation-proven
  non-vacuous (suppress-equivocation + `fork:=false` both fail), oracle/conformance gate re-satisfied
  (independent prover/verifier paths over a `testonly.Tree` fixture; both golden vkeys
  `40b74463`/`22b08f3e` unmoved), `go list -deps ./internal/logclient` confirms no store edge,
  gate-integrity scan clean, scope-clean (2 production files + 1 test).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 2 in `issues.md` (was 3; the
  ADR-0006 "self-consistency policy split across follower + logclient" issue is verified fixed and
  removed): (1) tile writers require `width`, duplicating the tlog `p`-translation in the follower's
  `widthForP` copy; (2) accepted-checkpoint advancement is three caller-sequenced store writes
  (`RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState`, locality). **`low` (loop-skipped):**
  `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2 met; M3 in progress — the tlog-tiles mirror HTTP arc is complete and the self-consistency
verdict now lives in one pure, table-tested `logclient.CheckConsistency`. Continue draining the
remaining ADR-0005 / follower-locality `normal` backlog, or begin the verify-for-me / proof-surface
cache arc.** No `critical` open, so feature work proceeds; the 2 open `normal` issues block DONE and
are weighed against the state→target gap.

Candidate order:
1. **`normal` backlog (ADR-0005 / locality)** — the store-owned `AdvanceAccepted(hubID, info, raw,
   observedAt)` single-transaction write that collapses the three caller-sequenced
   `RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState` writes in `PollHub` into one store-boundary
   operation (idempotent re-poll + set-once coverage); OR the tile-writer `p`-vocabulary unification
   (make `RecordTile`/`RecordEntryBundle` take `p uint8`, delete the follower's `widthForP` copy).
2. **Proof-surface cache/conditional-GET** — extend ETag/`Cache-Control` to the size-varying
   `/inclusion`/`/consistency`/`/entries` (`internal/proofserve`), tied to `LastSize`, needing a
   derived/weak validator.
3. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
4. **M3 verify-for-me / dashboard / log browser → WASM → OTS** remain the bulk of the v1 work.
