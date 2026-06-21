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

## Hub-status overlay precedence is duplicated across dashboard, proofserve, AND dossier (now 3x)
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** `internal/dashboard/handler.go:161-190`,
  `internal/proofserve/handler.go:600-635`, AND now `internal/dossier/handler.go:742-787` all implement
  the same five-status `overlayStatus` + `hubStatus` precedence verbatim — every docstring confesses it.
  The third copy landed with the hub dossier (deliberately, per its `next.md` Not-In-Scope), so the
  consolidation pressure is now 3x: a precedence fix is a three-site edit. Still `low` (no progress
  gate), but the move is more valuable now. The `internal/badge` package owns *rendering*
  the five statuses (silhouette + the single-source label table) but not *resolving* them, so the
  ADR-0010 visual-contract precedence (frozen/inactive are durable truths that win; only `verified`
  consults the live verdict, and only to adopt `unresolvable`/`unverified`) lives in two places keyed on
  two input types (`store.HubSummary`, which carries `.Active` → `inactive`, vs `store.FollowState`,
  which does not). A precedence fix in one silently diverges from the other; the log-browser overlay has
  no HTTP-seam test of its own. Deepen by moving resolution into `badge` (already the taxonomy owner) as
  one `Resolve(provable-status, live-verdict) → status` both handlers cross; the inactive case (only
  `HubSummary` has it) is decided before the seam so the resolver stays one function. Verify fixed: the
  overlay precedence exists in exactly one place, both handlers call it, and one test covers the
  five-status taxonomy. Pure locality deepening — contradicts no ADR.
- **Spec:** ADR-0010 five-status `HubStatusBadge` visual contract; CLAUDE.md "Hub status" glossary.

## Mirror write path leaks tile coordinates and the partial-`p` convention into the follower
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** `internal/follower/ingest.go:58-83` walks `tiles.TileCoords` /
  `tiles.BundleCoords` and passes each coord's `Partial` (the tlog-tiles p qualifier) straight to
  `store.RecordTile` / `store.RecordEntryBundle`, then projects via `logclient.BundleProjections` →
  `store.RecordProjections`. The "Mirror" (glossary: the complete copy of a hub's tiles + entries) has
  no single owner — coordinate enumeration, the p→width convention, projection, and BLOB writes are
  split between the follower's ingest path and the store's 22-method CRUD surface, so the follower must
  learn the tile layout to drive storage. Deepen by absorbing the walk + p + projection + writes behind
  one deep Mirror seam (e.g. `Sync(hubID, treeSize, fetcher)`); the follower stops referencing tile
  coordinates and the store's per-tile methods go private behind it. Verify fixed: `ingest.go` no longer
  references `tiles.*Coords` or a `Partial` qualifier, and the mirror round-trip is tested through the
  single Mirror interface. NOTE: this is **not** "add a store interface" — there is exactly one SQLite
  adapter (ADR-0005), so that would be a hypothetical seam with one adapter; Mirror still writes to the
  same SQLite store and `store.SQLiteFetcher` stays its read side. Larger move — wants a design/grilling
  pass before building.
- **Spec:** ADR-0005 single SQLite store; CLAUDE.md "Mirror" glossary.

## Add a scaling trip-wire: writer-wait time + per-network DB file size metrics
- **Priority:** low
- **Source:** [human]
- **What / where / how to verify:** The single-file-per-network store (ADR-0007) is right for
  10s–100s of hubs, but two axes can eventually bind: the single writer (`SetMaxOpenConns(1)`,
  `internal/store/sqlite.go`) serializing all hubs' poll-commits, and per-network file size (one
  high-traffic hub at millions/day bloating the shared file). Expose two Prometheus metrics via the
  existing `internal/metrics` registry so the bind is visible *before* it hurts, not discovered under
  load: (1) writer-wait / commit latency — how long a poll-commit waits on or holds the single
  connection (a rising p99 is the writer-contention signal); (2) per-network DB file size in bytes
  (e.g. `os.Stat` on the `.db` file, refreshed per poll cycle). Both feed the "revisit per-hub files
  or rebuildable-bulk tiering" decision recorded in ADR-0007. Verify fixed: `GET /metrics` exposes a
  writer-wait/commit-latency series and a DB-file-size gauge, both labelled per network, with a test
  asserting they appear. Low — skipped by the loop; reserved for when load planning resumes.
- **Spec:** ADR-0007 "Why network-level and not hub-level" (the trip-wire it names); CLAUDE.md
  `GET /metrics` surface.

## proofserve repeats the `os.ErrNotExist`→404 mapping that the sibling tilesserve already centralised
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** The three proof routes in `internal/proofserve/handler.go` —
  `serveInclusion` (198-202), `serveConsistency` (287-291), `serveEntries` (349-353) — each spell out the
  same `errors.Is(err, os.ErrNotExist) → 404 "… not mirrored", else → 500` mapping inline. The sibling
  `internal/tilesserve/handler.go:145-152` already lifts this into one `writeReadError(w, err)` reused by
  all its routes; a change to the not-mirrored mapping is a three-site edit in proofserve. Fix: lift one
  proofserve-local `writeReadError` and call it from the three proof routes. **Exclude `serveVerify`** —
  it deliberately maps a not-yet-mirrored tile/bundle (`os.ErrNotExist`) to a 200 verdict
  (handler.go:399-400), not a 404, so it must NOT share the helper. Verify fixed: the not-mirrored→404
  mapping for the three proof routes lives in one helper and `serveVerify`'s 200 behaviour is unchanged.
  Cosmetic locality only.
- **Spec:** `internal/tilesserve` `writeReadError` pattern; no spec contract.

## `/records` lists unaccepted leaves (no `LastSize` cap), unlike every other record-facing route
- **Priority:** normal
- **Source:** [review] (Codex P1, reviewer-confirmed)
- **What / where / how to verify:** `serveRecords` / `store.ListRecords`
  (`internal/proofserve/handler.go:700`, `internal/store/iscc_index.go:98-141`) list every `iscc_index`
  row for the hub with NO ceiling on `seq`, but `iscc_index` can hold projections ABOVE `LastSize`:
  `follower.PollHub` calls `ingestTiles`→`RecordProjections` (`internal/follower/follower.go:174`,
  `internal/follower/ingest.go:117`) for ALL `info.TreeSize` leaves BEFORE `checkConsistency` and
  `AdvanceAccepted` (lines 180, 221). On a violation (line 194) or an already-frozen short-circuit (line
  204) the cursor never advances, so projections persist for `seq >= LastSize`. The record list then shows
  unaccepted leaves as accepted and emits `entries?index=<seq>` links that `serveEntries` rejects with 404
  "leaf not covered by accepted checkpoint" (the `seq >= size` guard at handler.go:361 that EVERY other
  record route — `serveInclusion`:219, `serveEntries`:361, `serveVerify`:487 — applies but `serveRecords`
  omits). This is a coverage-honesty defect (ADR-0001) on exactly the frozen/violation case the monitor
  exists for. Fix: pass `fs.LastSize` into `ListRecords` as a ceiling (`AND seq < LastSize`, and cap the
  total count likewise) so only accepted leaves are listed. Verify fixed: a fixture with projections above
  `LastSize` (or a frozen hub past a violation) renders only `seq < LastSize` rows, and no listed row links
  to a leaf `serveEntries` would 404.
- **Spec:** ADR-0001 coverage honesty; CLAUDE.md "Coverage" glossary; M2 `>= LastSize` accepted-tree contract.

## `/records` pagination overloads `from=0` and skips the page-size clamp on overflow (two bugs, one root)
- **Priority:** normal
- **Source:** [review] (Codex P1+P2, reviewer-confirmed)
- **What / where / how to verify:** Both root in `internal/proofserve/handler.go` `serveRecords` +
  `store.ListRecords` treating `from == 0` as the "start at newest" sentinel.
  (1) **seq 0 unreachable** — the older link uses `OlderFrom = oldest-1` (handler.go:731) gated on
  `oldest > 0`, so a page ending at seq 1 emits `records?from=0&n=…`; `ListRecords` reads `from==0` as
  "newest" (iscc_index.go:108 `if from > 0`), so "older" jumps back to the newest page and the oldest
  record (seq 0) is never reachable by navigation. Reviewer-confirmed: `from=1&n=1` emits older link
  `from=0&n=1`; following it shows seq 299, never seq 0.
  (2) **page-size clamp bypass** — handler.go:693-697 does `pageSize = int(n)` BEFORE `if pageSize >
  maxPageSize`; a huge `n` (e.g. `9223372036854775808`) wraps `int(n)` NEGATIVE, the `> 200` check misses
  it, and modernc SQLite treats a negative `LIMIT` as UNLIMITED (reviewer-confirmed: `n=-5` returned all
  10 rows), so the intended anti-DoS clamp is defeated and the whole index renders. `parseUint`
  (handler.go:807-819) also wraps silently on overflow (`n = n*10 + …`, no guard), so the cap is the only
  defense. Fix: (a) clamp page size while still `uint64` BEFORE the `int()` conversion (and bound
  `parseUint` or reject overflow); (b) stop overloading 0 — carry a separate has-cursor bool or use a
  1-based/`+1` cursor so `from=0` can mean "start at seq 0". Verify fixed: a deep older-chain reaches seq
  0 (a test follows older links down to seq 0), and `n=9223372036854775808` yields at most `maxPageSize`
  rows.
- **Spec:** target.md M-UI record-list "paginates via plain links … each row links to its single-record
  page"; ADR-0001 (no fabricated/unreachable coverage); next.md line 93 (the clamp's stated purpose).
