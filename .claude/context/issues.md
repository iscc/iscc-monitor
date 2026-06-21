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

## Hub-status overlay precedence is duplicated across dashboard and proofserve
- **Priority:** low
- **Source:** [review] (architecture review)
- **What / where / how to verify:** `internal/dashboard/handler.go:161-190` (`overlayStatus` + `hubStatus`)
  and `internal/proofserve/handler.go:600-635` (`overlayStatus` + `hubStatus`) implement the same
  five-status overlay precedence verbatim — both docstrings confess it (dashboard: "mirrors
  internal/proofserve.hubStatus and extends it with the realm-registry inactive case"; proofserve:
  "mirrors dashboard.overlayStatus precedence verbatim"). The `internal/badge` package owns *rendering*
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
