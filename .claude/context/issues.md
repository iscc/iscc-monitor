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

## `fsckMirror` re-resolves the did:web key every verified poll (a redundant did.json fetch)
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/follower/follower.go` `fsckMirror` calls
  `ResolveVerifierKey` (a did.json fetch) on every verified, non-violation poll purely to obtain the
  vkey *string* for `RunFsck` — even though `AcceptCheckpoint` already resolved the vkey and
  `cacheHubKey` caches the key bytes. Observable: `TestPollHubCacheHitSkipsDidFetch` now asserts
  cold=3 / warm=2 did.json fetches (was 2/1), the +1 each being fsck's resolve. An efficiency slice
  could thread the already-resolved vkey from `AcceptCheckpoint`/`cacheHubKey` through to `fsckMirror`
  (e.g. `AcceptCheckpoint` returning the vkey, or recovering it from the cached `hub_keys` row +
  `VerifierKey` formatting) to drop the extra fetch. Verify fixed: cold drops to 2, warm to 1, and the
  cache-hit delta test still pins "warm = cold − 1". Correctness is unaffected today (the resolve
  yields the same key); this is purely a fetch-count efficiency item.
- **Spec:** KISS / no spec contract; the second-resolve elimination mirrors the `cacheHubKeyFast`
  optimization already landed.

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
