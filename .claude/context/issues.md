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
