<!-- assessed-at: 650f96506ab63c60d36d705ff1740e0cd24a1986 -->

# Project State

## Status: IN_PROGRESS

## Phase: M3 (Trust API + dashboard) — the tlog-tiles mirror HTTP arc plus all three computed proofs
are complete and the M2 store/follower mirror-write seam is now single-source. verify-for-me, the
server-rendered dashboard, the log browser, the WASM verifier, and OTS anchoring all remain unstarted.

The monitor follows + mirrors + verifies hubs (M1 met) and serves all three computed proofs per hub
from the local mirror (M2 met: `/inclusion`, `/consistency`, `/entries`). M3 has the raw tlog-tiles
mirror fully plumbed (CORS → Cache-Control → conditional GET). The bulk of M3 (verify-for-me,
dashboard, log browser), the WASM verifier, and OTS anchoring remain.

Incremental review of `be2ed3f..HEAD` (HEAD `650f965` on `develop`). The diff touches **three
production files**, all in the M2 mirror-write path: `internal/store/tiles.go` (`RecordTile`/
`RecordEntryBundle` now take `p uint8`), `internal/follower/ingest.go` (call sites pass `c.Partial`
straight through; the follower's duplicate `widthForP` deleted), and `internal/tiles/coords.go`
(comment-only doc fix). Verified from source: `grep -rn "func widthForP" internal/` returns exactly
**one** hit (`internal/store/fetcher.go:113`) — the store now owns the only `p→width` translation;
`grep -n "widthForP" internal/follower/ingest.go` is empty (exit 1). `RecordTile(…, p uint8, …)` and
`RecordEntryBundle(…, p uint8, …)` confirmed at `internal/store/tiles.go:46,68`. `go list -deps
./internal/store | grep -E "net/http|internal/logclient|internal/follower"` is empty (store stays a
leaf). `git diff be2ed3f..HEAD --stat -- go.mod go.sum internal/store/schema.sql` is empty
(byte-unchanged). Latest `review` handoff (2026-06-21, "Unify the tlog-tiles `p`-vocabulary — store
owns the only `p→width` translation") is **PASS / CONTINUE**: `mise run check` green (15 packages
`ok`, `gofmt -l .` clean), the public-surface invariant reviewer-mutation-proven non-vacuous
(`widthForP` full-mapping → `tiles.TileWidth-1` makes `TestIngestTilesWidthMapping` +
`TestRecordTileRoundTrip` FAIL), oracle gate correctly N/A (pure arithmetic / CRUD) but the four
mirror-consuming packages re-run uncached, gate-integrity scan clean, scope-clean (3 production files,
one comment-only). **Resolves the last open `normal` issue.** **CI green at HEAD `650f965`** (develop
run 27899275324, `conclusion: success`).

**Branch note:** active work happens on `develop` (HEAD `650f965`, tree has one uncommitted
non-source change — `.devcontainer/devcontainer.json`, pre-existing infra tuning left for the human;
no source drift); a human merges `develop`→`main`. The `main` branch lags at `c59d380`.

## M1 — Read-only Monitor
**Status**: met — carried forward; this slice did not touch the verify/consistency path. All Verify
criteria remain satisfied: `origin`/`vkey` golden, all three triggers golden-tested end-to-end with
freeze + alert-once + restart survival, coverage tracked, structured logs, `/metrics` served over
HTTP. **CI-gated & green.**

- **Test totals at HEAD**: **207 `func Test`** across `cmd/` + `internal/` (was 208), **48** `_test.go`
  files (unchanged). The net −1 is the deleted `TestWidthForP` (it tested the now-removed follower
  duplicate's arithmetic; superseded — more strongly — by `TestIngestTilesWidthMapping` over the
  public `RecordTile`/`ReadTileBlob` surface), partly offset by the test renames in this slice.
- **Packages present**: `cmd/{iscc-monitor,notecheck}`; `internal/{config,corsmw,didweb,follower,
  healthz,logclient,metrics,metricshttp,proofserve,registry,store,tiles,tilesserve}` (13 internal
  packages). Module path `github.com/iscc/iscc-monitor`, `go 1.24.0` (no `toolchain` line).
- **All three triggers WIRED + golden-tested**: shrink → fork → equivocation, evaluated inside
  `logclient.CheckConsistency`. The growing-pair equivocation builds the RFC-6962 consistency proof
  from the LOCAL mirror → freeze on `true` (ADR-0006). Frozen hubs are evidence-only on clean
  re-polls; fork re-detection on a frozen hub re-records evidence without re-alerting.
- **`AcceptCheckpoint` 4-way verdict** (`logclient/accept.go`) returns `(Status, CheckpointInfo,
  VerifiedContext{VKey,Key}, error)`, the context populated **only** on `StatusVerified`. `PollHub`
  threads it into the hub-key cache upsert and `fsckMirror`; no `ResolveVerifierKey` caller remains
  in production `follower.go`. Reuse is per-poll only (ADR-0009).
- **`cmd/notecheck`** — fully-independent signature-parity oracle, shelled out in CI against the real
  sb0 checkpoint (accept `OK sb0.iscc.id/log` + reject a one-char-flipped sig).
- `/metrics`, `/healthz`, the per-hub mirror, and the per-hub `/inclusion` + `/consistency` +
  `/entries` proof routes all ride one `*http.ServeMux` on a single listener, wrapped once in
  `corsmw.Handler`.
- `store/*.go` — `modernc.org/sqlite`, ADR-0005/0007 single-writer discipline (WAL,
  `busy_timeout=5000`, `foreign_keys=ON`, `SetMaxOpenConns(1)`), embedded nine-table `schema.sql`.
  `schema.sql` byte-unchanged this slice.
- **Missing (M1 connective tissue, outside the Verify bar):** real alert transport — `alertFunc` is a
  WARN `slog` emit; the `AlertFunc func(int64,string)` seam is unchanged. Real email/webhook delivery
  is later. The warm-path's irreducible second `did.json` resolve is a larger design change, not on
  the Verify bar.
- **Fixtures**: `testdata/live/` (repo root) still holds **only the two checkpoints**
  (`sb0.iscc.id_checkpoint`, `sb1.amlet.id_checkpoint`) — **no tiles, entry bundles, or did.json**
  (re-verified by `find`/`ls`). All tile/fsck/inclusion/consistency/entries/projection/serve tests run
  against in-process fixtures. Real tile/entry-bundle + `IsccLogInclusionProof` fixtures remain a soft
  prerequisite for an *inbound* hub-evidence transport. Known stale `sb1.amlet.id` did.json drift
  (pre-rotation key) captured in tests; fixtures not yet refreshed.
- **Reuse imports wired**: `golang.org/x/mod/sumdb/note`, `modernc.org/sqlite`,
  `transparency-dev/merkle` (`rfc6962`, `proof.Inclusion` AND `proof.Consistency`),
  `transparency-dev/tessera` (`api`, `api/layout`, both proof builders, `leafhasher`, `tessera/fsck`
  with a production caller via `fsckMirror`, `tessera/client` re-export, `api.EntryBundle.UnmarshalText`),
  `transparency-dev/formats` (DIRECT — `cmd/notecheck`), stdlib `slog`/`net/http`/`os`/`crypto/sha256`.
  **Not yet wired**: `nbd-wtf/opentimestamps` (re-verified: grep → no hits in `cmd/`+`internal/`+`go.mod`).

## M2 — Aggregator
**Status**: **met** — both Verify criteria are exercised (fsck root-rebuild WIRED on every verified
non-frozen poll; inclusion cross-check conformance-tested over the real verified mirror), and the
served proof surface is complete: all three computed proofs — `inclusion`, `consistency`, `entries` —
served from the local mirror, never re-hitting the hub. **This slice's change lands here:** the mirror
write API (`RecordTile`/`RecordEntryBundle`) now speaks the tlog-tiles partial qualifier `p uint8`
directly, with the store as the single `p→width` authority (`widthForP` at `internal/store/
fetcher.go:113`); the follower's duplicate translation is deleted (single source of truth, ADR-0005).
- **fsck root-rebuild (WIRED, runs every verified non-frozen poll):** `fsckMirror` (`follower.go`)
  builds a read-only `store.SQLiteFetcher` and calls `logclient.RunFsck`, comparing the rebuilt
  RFC-6962 root to the signed checkpoint root, reusing the verifier key `AcceptCheckpoint` resolved
  this poll. A frozen hub's clean re-poll skips `fsckMirror`.
- **`iscc_index` projection (WIRED, runs every verified poll):** `internal/follower/ingest.go` folds
  the raw bundle via `logclient.BundleProjections` → `store.RecordProjections`.
- **inclusion cross-check (BUILT + CONFORMANCE-TESTED):** `internal/logclient/inclusioncheck.go` closed
  against M2's Verify bar by `internal/follower/inclusion_test.go`.
- **raw tlog-tiles HTTP read surface (BUILT + WIRED):** `internal/tilesserve/handler.go` routes the
  three canonical iscc-log §9 paths to raw mirror BLOBs verbatim, per hub, with per-route
  `Cache-Control` and a strong content ETag + `If-None-Match`→`304` conditional GET on every 200.
- **computed inclusion/consistency/entries proofs (BUILT + WIRED):** `internal/proofserve/handler.go`
  `serveInclusion`/`serveConsistency`/`serveEntries` serve from the mirror against `LastSize`.
- **mirror write API is single-source on `p` (this slice):** `RecordTile(…, p uint8, …)` /
  `RecordEntryBundle(…, p uint8, …)` compute `width := widthForP(p)` internally; the follower passes
  `c.Partial` straight through. Full coords are written as the literal `0` (never `uint8(256)`).

**What remains for M2:** nothing on the Verify bar — M2 is met.

## M3 — Trust API + dashboard
**Status**: **in progress**. The raw tlog-tiles mirror is fully plumbed: (1) CORS on every public GET
via the single `corsmw.Handler` wrap; (2) per-route `Cache-Control`; (3) a strong content ETag +
wildcard/exact `If-None-Match`→`304` conditional GET on every 200. The binary's mux serves `/metrics`,
`/healthz`, the per-hub raw tlog-tiles mirror subtrees, and the per-hub `/inclusion` + `/consistency` +
`/entries` computed-proof endpoints — all behind CORS. **Still absent (re-verified):** conditional-GET /
cache policy on the size-dependent proof surfaces (`internal/proofserve` carries no ETag/Cache-Control
— grep clean, tied to `LastSize`); `verify-for-me` verdict surface (no `html/template`/`text/template`
import anywhere — grep clean); server-rendered dashboard (status/coverage/lag/violations/OTS); `/`
landing; and the log browser.

## WASM verifier · OTS anchoring
**Status**: not started (re-verified). `nbd-wtf/opentimestamps` is not imported (grep → no hits in
`cmd/`+`internal/`+`go.mod`); no `internal/proof` package exists (`ls` → no such directory); no WASM
build target (`syscall/js` not in source — grep clean). The WASM-shared verifier seam continues to
ride on `internal/didweb` (the WASM-pure parser); `logclient.CheckConsistency` could share a WASM seam
later.

## Quality gates
**Status**: green — **enforced in CI**
- `go.mod` present (`module github.com/iscc/iscc-monitor`, `go 1.24.0`, no `toolchain` line); `mise run
  check` runnable (`mise.toml` `[tasks.check]` present). `schema.sql`/`go.mod`/`go.sum` byte-unchanged
  this slice.
- **CI configured and passing.** `.github/workflows/ci.yml` runs the inlined `mise run check` gate (`go
  build`/`go vet`/`go test ./...`) + the `cmd/notecheck` oracle shell-out on push/PR to
  `develop`/`main`. Remote `origin` = `github.com/iscc/iscc-monitor.git`. **Latest run on `develop` for
  HEAD `650f965`: `conclusion: success`** (run 27899275324).
- Latest `review` handoff (2026-06-21, **PASS / CONTINUE**) records `mise run check` green (15 packages
  `ok`, `go vet`/`gofmt -l .` clean), the public-surface `widthForP` invariant reviewer-mutation-proven
  non-vacuous, oracle gate correctly N/A (pure arithmetic) with the four mirror-consuming packages
  re-run uncached, `go list -deps ./internal/store` confirms no `net/http`/logclient/follower edge
  (store stays a leaf — re-verified by assessor), gate-integrity scan over the unpushed commits clean,
  scope-clean (3 production files incl. 1 comment-only).
- **No open `critical` or `normal` issue.** The ADR-0005 tile-writer `p`-duplication issue is verified
  fixed by this slice and removed from `issues.md`.
- **Open `low` issue (loop-skipped, does not block DONE per loop semantics... but note: `low` is
  skipped by the loop, not a DONE blocker):** 1 in `issues.md` — `cmd/notecheck`'s vestigial `out
  io.Writer` param.

## Next Milestone
**M1/M2 met; M3 in progress — the tlog-tiles mirror HTTP arc + three computed proofs are complete, the
M2 mirror-write seam is now single-source on `p`, and the `normal` backlog is drained. Begin the
verify-for-me / proof-bundle arc.** No `critical`/`normal` open, so feature work proceeds.

Candidate order:
1. **Proof-bundle assembler / proof-surface cache** — compose `{checkpoint, inclusion proof, record
   bytes, hub key, ots?}` into one self-contained client-verifiable package, reading from the unified
   store/`SQLiteFetcher` seam (start in `internal/proofserve` or a sibling); and extend ETag/
   `Cache-Control`/conditional-GET to the size-varying `/inclusion`/`/consistency`/`/entries`
   (`internal/proofserve`), tied to `LastSize` (needs a derived/weak validator). This bundle is shared
   by both the in-browser verifier and verify-for-me.
2. **verify-for-me REST surface** — the non-authoritative verdict path over the proof bundle.
3. **sb1 fixture refresh** (stale did.json key) and **real alert transport** (close M1's alert path).
4. **M3 dashboard / log browser → WASM verifier → OTS anchoring** remain the bulk of the v1 work.
