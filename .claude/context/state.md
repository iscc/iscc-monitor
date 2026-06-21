<!-- assessed-at: 890edf8c41f93638a4f3ebccdf7008525d6a0c8b -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — the tlog-tiles mirror HTTP-plumbing arc is complete; current
work drains the ADR-0006 trust-path / follower-locality backlog. verify-for-me, the server-rendered
dashboard, the log browser, the WASM verifier, and OTS anchoring all remain unstarted.

The monitor follows + mirrors + verifies hubs (M1 met) and serves all three computed proofs per hub
from the local mirror (M2 met: `/inclusion`, `/consistency`, `/entries`). M3 has the raw tlog-tiles
mirror fully plumbed (CORS → Cache-Control → conditional GET). The bulk of M3 (verify-for-me,
dashboard, log browser), the WASM verifier, and OTS anchoring remain.

Incremental review of `be6ccd3..HEAD` (HEAD `890edf8` on `develop`). The diff touches **zero
production files** — the only `.go` change is `internal/follower/follower_test.go`; the rest is context
docs, `CLAUDE.md`, and `.devcontainer/devcontainer.json` (dev-port mapping). The slice rewrote
`TestPollHubFork`'s re-detection block to drive a real second `PollHub` (later `observedAt=time.Unix
(2,0)`) instead of calling `freeze(...)` directly, and removed the stale "unordered LIMIT 1 …
non-deterministic" comment. Verified from source: on the second poll `checkConsistency` runs BEFORE the
`fs.Frozen` short-circuit and reads the prior accepted root via `CheckpointAt(hubID, m.size)`, whose
explicit `ORDER BY rowid LIMIT 1` returns the lowest-rowid seed root (not the contradicting evidence
the first freeze persisted at a higher rowid) — so `CheckFork` re-fires and `freeze(wasFrozen=true)`
re-records evidence without re-alerting. The block now asserts `violations==2`, `alerts==1`, hub stays
`Frozen`, `LastSize==m.size`, and the cumulative `iscc_monitor_violations_total{...kind="fork"} 2`
metric. **No `go.mod`/`go.sum`/`schema.sql` change.** Latest `review` handoff (2026-06-21, "Drive
`TestPollHubFork` re-detection through a second `PollHub`") is **PASS / CONTINUE**: `mise run check`
green (15 packages `ok`, `gofmt -l .` empty), the re-detection drive reviewer-mutation-proven
non-vacuous (asserting `alerts==2` FAILS; removing the second `PollHub` FAILS three assertions), oracle
gate correctly N/A (test-only — both golden vkeys `40b74463`/`22b08f3e` still reproduce), gate-integrity
scan clean across the 3 commits, scope-clean (0 production files). **CI green at HEAD `890edf8`**
(develop run 27897829003, `conclusion: success`).

**Branch note:** active work happens on `develop` (HEAD `890edf8`, tree clean, in sync with
`origin/develop`); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward; this slice strengthened the *test* coverage of the already-landed
frozen-hub-evidence-only behavior (re-detection now exercised end-to-end through `PollHub`, not a direct
`freeze`). All Verify criteria remain satisfied: `origin`/`vkey` golden, all three triggers golden-tested
end-to-end with freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics`
served over HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **206 `func Test`** across `cmd/` + `internal/`, **47** `_test.go` files
  (unchanged vs the prior assessment — the slice added assertions to the existing `TestPollHubFork`,
  no net new test function or file).
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,corsmw,didweb,follower,
  healthz,logclient,metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}` (13 internal
  packages). Module path `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation in `checkConsistency`. The
  growing-pair equivocation builds the RFC-6962 consistency proof from the LOCAL mirror → freeze on
  `true` (ADR-0006). **Frozen hubs are evidence-only** on clean re-polls; fork re-detection on a frozen
  hub now re-records evidence without re-alerting, proven through a real second `PollHub`.
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
  later.
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
touch proof or mirror logic.
- **fsck root-rebuild (WIRED, runs every verified non-frozen poll):** `fsckMirror` (`follower.go`, after
  `ingestTiles`, before `recordVerdict`) builds a read-only `store.SQLiteFetcher` and calls
  `logclient.RunFsck`, comparing the rebuilt RFC-6962 root to the signed checkpoint root. A frozen hub's
  clean re-poll skips `fsckMirror` (already-accepted state needs no re-verify; tiles still ingest).
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
in source). The WASM-shared verifier seam continues to ride on `internal/didweb` (untouched this slice).

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable. `go.mod`/`go.sum`/`schema.sql` byte-unchanged this slice (zero production files
  changed; the only `.go` change is a test file).
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for
  HEAD `890edf8`: `conclusion: success`** (run 27897829003).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (15 packages
  `ok`, `go vet`/`gofmt -l .` clean), the re-detection-via-second-`PollHub` drive reviewer-mutation-proven
  non-vacuous, oracle gate correctly N/A (test-only; both golden vkeys reproduce), trust root untouched +
  green, gate-integrity scan clean across the 3 commits, scope-clean (0 production files).
- **No open `critical` issue.**
- **Open `normal` issues (block DONE, do not block this slice's PASS)** — 4 in `issues.md` (was 5; the
  "`TestPollHubFork` re-detection still bypasses `PollHub`" issue is verified fixed and removed):
  `AcceptCheckpoint` discards resolved context → verified polls re-fetch did.json; tile writers require
  `width` (duplicated `p`-translation in follower); accepted-checkpoint advancement is three
  caller-sequenced store writes (locality); self-consistency policy split across follower + logclient
  (refactor). **`low` (loop-skipped):** `cmd/notecheck`'s vestigial `out io.Writer` param.

## Next Milestone
**M1/M2 met; M3 in progress — the tlog-tiles mirror HTTP arc is complete and the ADR-0006 frozen-hub
trust-path is now end-to-end tested. Continue draining the remaining ADR-0006 / follower-locality
`normal` backlog, or begin the verify-for-me / proof-surface cache arc.** No `critical` open, so feature
work proceeds; the 4 open `normal` issues block DONE and are weighed against the state→target gap.

Candidate order:
1. **`AcceptCheckpoint` resolved-context reuse** (highest-value follower slice): verified polls
   currently re-fetch `did.json` two/three times per poll — widen the verified result to carry the
   resolved per-poll context while preserving ADR-0009's per-poll validity check.
2. **Proof-surface cache/conditional-GET** — extend ETag/`Cache-Control` to the size-varying
   `/inclusion`/`/consistency`/`/entries` (`internal/proofserve`), tied to `LastSize`, needing a
   derived/weak validator.
3. **`normal` backlog remainder**: tile-writer `p`-vocabulary unification (push `widthForP` into the
   store); deep store-owned `AdvanceAccepted`; collapse the self-consistency decision into a pure
   `logclient.CheckConsistency`.
4. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
5. **M3 verify-for-me / dashboard / log browser → WASM → OTS** remain the bulk of the v1 work.
