<!-- assessed-at: 86bc1188a4fe80f12a30808ccc144eaf9d016e81 -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — the tlog-tiles mirror HTTP arc is complete; current work
drains the ADR-0006 trust-path / follower-locality backlog. verify-for-me, the server-rendered
dashboard, the log browser, the WASM verifier, and OTS anchoring all remain unstarted.

The monitor follows + mirrors + verifies hubs (M1 met) and serves all three computed proofs per hub
from the local mirror (M2 met: `/inclusion`, `/consistency`, `/entries`). M3 has the raw tlog-tiles
mirror fully plumbed (CORS → Cache-Control → conditional GET). The bulk of M3 (verify-for-me,
dashboard, log browser), the WASM verifier, and OTS anchoring remain.

Incremental review of `890edf8..HEAD` (HEAD `86bc118` on `develop`). The diff touches **two
production files** — `internal/logclient/accept.go` and `internal/follower/follower.go` — plus their
tests and context docs. Verified from source: `AcceptCheckpoint` now returns a 4-tuple
`(Status, CheckpointInfo, VerifiedContext, error)` where `VerifiedContext{VKey, Key}` is populated
**only on `StatusVerified`** and is the zero value on every non-verified verdict, including
`StatusRotated` (accept.go:125 returns the zero context even though it resolved a key — a rotated key
never leaks into follower reuse). `PollHub` threads `vctx` into `cacheHubKey`/`cacheHubKeyResolve`
(cache-miss upsert) and `fsckMirror`, both of which **dropped their own `ResolveVerifierKey` calls**
(grep confirms **zero** `ResolveVerifierKey` callers left in `follower.go`). A cold verified poll now
resolves `did.json` once instead of three; the reuse is in-bounds because the key was resolved +
`ValidAt`-checked **this poll** (ADR-0009), never cached across polls. Latest `review` handoff
(2026-06-21, "Carry the resolved did:web context out of `AcceptCheckpoint`") is
**PASS_WITH_NOTES / CONTINUE**: `mise run check` green (15 packages `ok`, `gofmt -l .` empty),
assertions net-strengthened + reviewer-mutation-proven, oracle gate re-run (both golden vkeys
`40b74463`/`22b08f3e` reproduce, conformance follower tests green with the threaded context),
gate-integrity scan clean, scope-clean (exactly 2 production files = `next.md`'s Modify set). The
single documented deviation is benign: the warm-path "0 additional did.json fetches" `next.md` asked
for is **physically impossible** (`AcceptCheckpoint` has no internal cache and unconditionally calls
`ResolveVerifierKey`), so the advance shipped the true value (warm window = 1); the load-bearing win
(removing the *extra* cold resolves, 3→1) is fully realized and pinned. **CI green at HEAD `86bc118`**
(develop run 27898208177, `conclusion: success`).

**Branch note:** active work happens on `develop` (HEAD `86bc118`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward; this slice strengthened the verified-poll did:web-resolution
path (one `did.json` fetch instead of three on a cold verified poll) by reusing the context
`AcceptCheckpoint` already resolved, with no change to the ADR-0009 per-poll validity check. All
Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers golden-tested end-to-end
with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over
HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **206 `func Test`** across `cmd/` + `internal/`, **47** `_test.go` files
  (unchanged vs the prior assessment — the slice modified existing assertions in
  `accept_test.go`/`follower_test.go`/`fsck_test.go`/`checkpoint_test.go`, no net new test function or file).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,corsmw,didweb,follower,
  healthz,logclient,metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}` (13 internal
  packages). Module path `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on
  `true` (ADR-0006). **Frozen hubs are evidence-only** on clean re-polls; fork re-detection on a frozen
  hub re-records evidence without re-alerting, proven through a real second `PollHub`.
- **`AcceptCheckpoint` 4-way verdict** (`logclient/accept.go`) now also returns `VerifiedContext{VKey,
  Key}`, populated **only** on `StatusVerified` (zero on `Unverified`/`Unresolvable`/`Rotated`).
  `PollHub` threads it into the hub-key cache upsert and `fsckMirror`; **no `ResolveVerifierKey`
  caller remains in production `follower.go`** (verified by grep). Reuse is per-poll only (ADR-0009).
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
`consistency`, `entries` — served from the local mirror, never re-hitting the hub. This slice touched
`fsckMirror`'s *caller signature* (now fed `vctx.VKey` instead of re-resolving) but not its proof or
mirror logic.
- **fsck root-rebuild (WIRED, runs every verified non-frozen poll):** `fsckMirror` (`follower.go`, after
  `ingestTiles`, before `recordVerdict`) builds a read-only `store.SQLiteFetcher` and calls
  `logclient.RunFsck`, comparing the rebuilt RFC-6962 root to the signed checkpoint root; it now reuses
  the verifier key `AcceptCheckpoint` resolved this poll rather than re-resolving `did.json`. A frozen
  hub's clean re-poll skips `fsckMirror` (already-accepted state needs no re-verify; tiles still ingest).
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
  `… ORDER BY rowid LIMIT 1`, returning the lowest-rowid (prior accepted) root deterministically.

**What remains for M2:** nothing on the Verify bar — M2 is met.

## M3 — Trust API + dashboard
**Status**: **in progress**. The raw tlog-tiles mirror is fully plumbed: (1) CORS on every public GET
via the single `corsmw.Handler` wrap; (2) per-route `Cache-Control`; (3) a strong content ETag +
wildcard/exact `If-None-Match`→`304` conditional GET on every 200. The binary's mux serves `/metrics`,
`/healthz`, the per-hub raw tlog-tiles mirror subtrees, and the per-hub `/inclusion` + `/consistency` +
`/entries` computed-proof endpoints — all behind CORS. **Still absent (re-verified):** conditional-GET /
cache policy on the size-dependent proof surfaces (`/inclusion`/`/consistency`/`/entries`, tied to
`LastSize` — `internal/proofserve` carries no ETag/Cache-Control); `verify-for-me` verdict surface (no
`html/template`/`text/template` import anywhere); server-rendered dashboard (status/coverage/lag/
violations/OTS); `/` landing; and the log browser.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists; no WASM build target (no `syscall/js`
in source). The WASM-shared verifier seam continues to ride on `internal/didweb` (`accept.go`'s
`internal/didweb` import is the WASM-pure parser, not the networked resolver — the purity invariant holds).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. `schema.sql` byte-unchanged this slice.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for
  HEAD `86bc118`: `conclusion: success`** (run 27898208177).
- Latest `review` handoff (2026-06-21, **PASS_WITH_NOTES / CONTINUE**) records `mise run check` green
  (15 packages `ok`, `go vet`/`gofmt -l .` clean), the context-reuse drive reviewer-mutation-proven
  non-vacuous (assertions net-strengthened), oracle gate re-run (both golden vkeys reproduce,
  conformance follower tests green), trust root untouched + green, gate-integrity scan clean,
  scope-clean (2 production files). The one PASS_WITH_NOTES flag is the physically-impossible
  warm-path "0" literal in `next.md` (shipped the true value `1`) — a benign deviation, not a gate dodge.
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 3 in `issues.md` (was 4; the
  "`AcceptCheckpoint` discards resolved context → verified polls re-fetch did.json" issue is verified
  fixed and removed): tile writers require `width` (duplicated `p`-translation in follower);
  accepted-checkpoint advancement is three caller-sequenced store writes (locality); self-consistency
  policy split across follower + logclient (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s
  vestigial `out io.Writer` param.

## Next Milestone
**M1/M2 met; M3 in progress — the tlog-tiles mirror HTTP arc is complete and the verified-poll
did:web-resolution path is now single-fetch. Continue draining the remaining ADR-0006 /
follower-locality `normal` backlog, or begin the verify-for-me / proof-surface cache arc.** No
`critical` open, so feature work proceeds; the 3 open `normal` issues block DONE and are weighed
against the state→target gap.

Candidate order:
1. **`normal` backlog (ADR-0006 / locality)** — collapse the self-consistency policy into a deep
   `logclient.CheckConsistency(prior, next, fetcher) → (violated, kind, err)` (closes the
   follower/logclient split, table-testable in one place); OR the store-owned `AdvanceAccepted`
   single-transaction write; OR the tile-writer `p`-vocabulary unification (deletes the follower's
   `widthForP` copy).
2. **Proof-surface cache/conditional-GET** — extend ETag/`Cache-Control` to the size-varying
   `/inclusion`/`/consistency`/`/entries` (`internal/proofserve`), tied to `LastSize`, needing a
   derived/weak validator.
3. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
4. **M3 verify-for-me / dashboard / log browser → WASM → OTS** remain the bulk of the v1 work.
