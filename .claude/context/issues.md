# Issues

Lightweight backlog `define-next` can prioritize. Append entries; `review` deletes resolved ones.

**Format** — one entry per issue:

```
## <short title>
- **Priority:** critical | normal | low
- **Source:** [human] | [review] | [advance]
- **What / where / how to verify:** <the problem, its location, and the check that proves it fixed>
- **Spec:** <optional — target.md or an ADR section this is rooted in>
```

**Priority semantics:** `critical` preempts everything; `normal` is weighed against the state→target
gap; **`low` is skipped by the loop** (reserved for human-directed work). The `Source` tag records who
filed it and does **not** affect priority.

---

## `CheckpointAt` uses an unordered `LIMIT 1` — fork re-detection compares against an undefined row
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/store/checkpoints.go:155` `CheckpointAt` is
  `SELECT root, raw FROM checkpoints WHERE hub_id=? AND tree_size=? LIMIT 1` with **no `ORDER BY`**.
  After a fork freeze two checkpoint rows share the same `tree_size` (the prior/seed root + the
  contradictory evidence root), so a re-poll's `checkConsistency` compares `info.Root` against a
  data-dependent/undefined row — fork re-detection may or may not re-fire depending on which row the
  unordered `LIMIT 1` returns. This already forced `TestPollHubFork` to drive fork *re-detection*
  through `freeze` directly instead of a second `PollHub` (the first detection still goes through
  `PollHub`; shrink re-detection via `PollHub`/`Tick` stays covered by `TestTickFrozenUnaffected`,
  which is size-only so it is immune). The equivocation/serving slice that revisits prior-root
  selection should add `ORDER BY rowid` (or, better, select the *prior accepted* root explicitly —
  never the contradicting-evidence row). Verify fixed: `CheckpointAt` deterministically returns the
  prior accepted row, and `TestPollHubFork` can drive re-detection through `PollHub` again. Touches
  `internal/store`, so it was correctly out of scope for the fsck slice.
- **Spec:** ADR-0006 (freeze evidence discipline — re-detection must compare against the prior
  accepted root, not the contradicting evidence).

## `AcceptCheckpoint` discards resolved context, so verified polls re-fetch did.json
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/logclient/accept.go:84-105`
  `AcceptCheckpoint` resolves the did:web verifier key and origin, then returns only
  `(Status, CheckpointInfo, error)`. The verified path in `internal/follower/follower.go:187` and
  `:206` therefore re-fetches the same `did.json`: cold polls resolve in `AcceptCheckpoint`,
  `cacheHubKeyResolve`, and `fsckMirror`; warm polls still resolve in `AcceptCheckpoint` and
  `fsckMirror`. Widen the verified result to include the resolved per-poll context needed by cache
  refresh and fsck (`vkey`, origin, and DID key metadata), while preserving ADR-0009's per-poll
  validity-window check. Verify fixed by updating `TestPollHubCacheHitSkipsDidFetch`: cold and warm
  verified polls should each require one did.json fetch, and cache rows / fsck should still use the
  same verified key context.
- **Spec:** ADR-0009 (DID document remains the source of truth; reuse only within one verified poll).

## `cmd/notecheck`'s `run` has a vestigial `out io.Writer` parameter
- **Priority:** low
- **Source:** [review]
- **What / where / how to verify:** `cmd/notecheck/main.go` `run(vkey string, in io.Reader, out
  io.Writer) (string, error)` never writes to `out` — it returns the signer name and `main` prints
  `OK %s` to `os.Stdout` itself. The param matches the literal signature `next.md` specified and is
  harmless (tests pass a throwaway buffer; `go vet` does not flag unused params), but the signature
  is misleading. Fix when `run` is next touched: drop `out`, OR have `run` print `OK %s` to `out` and
  let the test assert on it. Verify fixed: `out` is either gone or written to. Low — skipped by the loop.
- **Spec:** KISS / YAGNI (CLAUDE.md code standards); no spec contract.

## Growing equivocations can be accepted before candidate tiles are mirrored
- **Priority:** critical
- **Source:** [review]
- **What / where / how to verify:** `internal/follower/follower.go:411-415` builds the growing
  checkpoint consistency proof before `ingestTiles`, so a normal mirror only has tiles for the
  previously accepted size. `ConsistencyProofFromTiles` then misses the candidate-size tiles, and
  `checkConsistency` treats that missing-tile error as a clean non-violation. The caller records and
  advances `last_size` to the inconsistent root; later polls compare against that new accepted root,
  so the split view is never frozen. Fix by ensuring the candidate tiles needed for the consistency
  proof are available before accepting the growing checkpoint, or by making missing proof tiles a
  retry/error path rather than a clean consistency pass. Verify fixed with a growing split-view poll
  where the prior checkpoint is accepted and the candidate root is inconsistent: the hub must freeze
  and must not advance accepted state to the candidate root.
- **Spec:** ADR-0006 (self-consistency violations freeze and preserve evidence); ADR-0005 (mirrored
  tiles back consistency proofs/root rebuilds).

## Frozen hubs still advance accepted state on later clean-looking polls
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/follower/follower.go:181` still reaches
  `AdvanceFollowState` when `fs.Frozen` is already true and the newly verified checkpoint is not a
  fresh violation relative to `LastSize`. That path also records checkpoints, coverage/key-cache
  state, tile mirrors, and fsck work even though ADR-0006 treats frozen hubs as evidence-only until a
  manual unfreeze. Fix by short-circuiting already-frozen hubs out of the accepted-state path after
  verification/re-detection evidence handling. Verify fixed by polling an already-frozen hub with a
  clean-looking larger checkpoint and asserting `last_size` and accepted-state side effects do not
  advance while re-detected violations can still be recorded as evidence.
- **Spec:** ADR-0006 (frozen hubs are evidence-only until manual unfreeze).

## Tile writers require `width`, duplicating the tlog `p` translation in the follower
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/store/fetcher.go:61-115` reads tiles and entry bundles
  through the tlog-tiles `p uint8` vocabulary and privately translates `p == 0` to width 256, while
  `internal/store/tiles.go:37` and `:56` require callers to pass the already-translated `width`.
  `internal/follower/ingest.go:57`, `:73`, and `:86-91` therefore carry a second load-bearing
  `widthForP` copy; if that copy ever stores a full tile at width 0 instead of 256, the
  `SQLiteFetcher` cannot read the mirror back. Make `RecordTile` and `RecordEntryBundle` accept
  `p uint8`, keep the `p`→`width` translation private to the store, and delete the follower copy.
  Verify fixed by moving the full/partial width tests to the store API and ensuring ingest passes
  `c.Partial` directly while `SQLiteFetcher` round-trips full and partial mirrors.
- **Spec:** ADR-0005 (mirror tile discipline); KISS / single source of truth for coordinate mapping.

## Accepted checkpoint advancement is three caller-sequenced store writes
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/follower/follower.go:165-183` owns the invariant
  "advance accepted state" by sequencing `RecordCheckpoint`, `SetCoverage`, and
  `AdvanceFollowState` directly. Those writes are separate store calls rather than one store-owned
  operation, so ordering and partial-write behavior live in the orchestrator instead of the storage
  boundary. Add a deep store method such as `AdvanceAccepted(hubID, info, raw, observedAt)` that
  records the checkpoint, sets coverage once, and advances the follow cursor in one transaction; use
  it from the verified, non-violation path. Verify fixed with a store-level test for the combined
  operation, including idempotent re-poll behavior and coverage staying set-once.
- **Spec:** ADR-0005 (single-writer mirror/follow-state discipline); KISS / locality.

## Self-consistency policy is split across follower orchestration and logclient helpers
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/follower/follower.go:355-425` owns the branch order,
  prior-checkpoint lookup, proof construction, missing-tile handling, and calls into
  `logclient.CheckShrink`, `CheckFork`, `ConsistencyProofFromTiles`, and `CheckEquivocation`.
  That leaves the load-bearing self-consistency decision spread across `follower`, `logclient`, and
  the tile fetch seam, making the indeterminate/missing-tile semantics harder to table-test in one
  place. After or alongside the critical growing split-view fix, collapse the pure decision into a
  deep `logclient.CheckConsistency` entry point that accepts prior/next checkpoint data plus an
  injected tile fetcher and returns `(violated, kind, err)` or an explicit indeterminate result. Verify
  fixed with logclient table tests for shrink, same-size split view, growing split view, clean growth,
  and missing-tile/error behavior; follower should just look up prior accepted evidence and act on the
  returned verdict.
- **Spec:** ADR-0006 (self-consistency violations freeze and preserve evidence).
