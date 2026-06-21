# Next Work Package

## Step: Live tile/bundle ingestion writer in PollHub

## Goal
On a verified, growing checkpoint, make `PollHub` mirror the hub's hash tiles and entry bundles into
the local store — walking the pure coordinate enumerations, fetching each over the existing transport
primitives, and writing via the store CRUD. This is the first production caller of `TileCoords`,
`BundleCoords`, `FetchTile`, and `FetchEntryBundle`; it un-dormants the wired equivocation branch on
the live path (which today always hits the missing-tile skip) and feeds `SQLiteFetcher`, unblocking
the M2 `fsck` root-rebuild and inclusion cross-check.

## Scope
- **Create**: `internal/follower/ingest.go` — the `ingestTiles` helper (one new non-test file).
- **Modify**: `internal/follower/follower.go` — call `ingestTiles` from `PollHub` on the verified,
  non-violation path. (2 non-test files total.)
- **Create (test)**: `internal/follower/ingest_test.go` — table-driven coordinate + width-translation
  unit test plus a `PollHub`-level integration test that asserts mirrored rows in the store.
- **Reference**:
  - `/workspace/iscc-monitor/internal/tiles/coords.go` — `TileCoords`/`BundleCoords` return order +
    the `Partial` ("0 == full") convention the writer consumes.
  - `/workspace/iscc-monitor/internal/logclient/tilefetch.go` — `FetchTile(ctx, fetcher, baseURL,
    level, index uint64, p uint8)` / `FetchEntryBundle(ctx, fetcher, baseURL, index uint64, p uint8)`
    signatures + the `%w`-wrapped `os.ErrNotExist` contract.
  - `/workspace/iscc-monitor/internal/store/tiles.go` — `RecordTile(ctx, hubID, level, index uint64,
    width int, data, observedAt)` / `RecordEntryBundle(ctx, hubID, bundleIndex uint64, width int,
    data, observedAt)` signatures; `is_full` is set internally by `tiles.IsFull(width)`.
  - `/workspace/iscc-monitor/internal/store/fetcher.go` lines 109-118 — the canonical `widthForP`
    translation (`p==0 → tiles.TileWidth (256)`, else `int(p)`) that this writer must replicate (the
    store's copy is unexported; do NOT export it — re-derive the one-liner in the follower).
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` lines 32-88 — the `compositeFetcher`
    URL-routing pattern + fixture loaders the new test extends to serve tile/bundle URLs.

## Not In Scope
- Wiring `RunFsck` over `SQLiteFetcher` (the M2 `fsck` root-rebuild) — its own next slice; this step
  only makes the tiles exist in the store for it to read.
- The inclusion cross-check vs the hub's `evidence.IsccLogInclusionProof` — needs captured
  `IsccLogInclusionProof` + real tile/bundle fixtures, a later slice.
- Capturing real on-disk tile/entry-bundle fixtures into `testdata/live/` — the test synthesizes
  tile/bundle bytes in-process (the writer is transport+CRUD, not crypto, so synthetic BLOBs suffice;
  byte-accurate live fixtures arrive with the inclusion cross-check).
- The `iscc_index` projection writer (schema-aware fold) — a distinct M2 slice.
- Exporting `store.widthForP` or otherwise touching the store package.
- Re-fetch/backoff policy tuning or a partial-only optimization — fetch every named coord each verified
  growing poll (ADR-0005: re-fetch partials, overwrite in place; a full tile re-write is idempotent).

## Implementation Notes
- **Placement.** Call `ingestTiles(ctx, st, fetcher, hubID, baseURL, info.TreeSize, observedAt)` inside
  `PollHub` on the verified, non-violation path, AFTER `RecordCheckpoint`/`SetCoverage`/
  `AdvanceFollowState`/`cacheHubKey` succeed and BEFORE the final `recordVerdict`/`return`. Wrap its
  error as `fmt.Errorf("follower.PollHub: hub %d: ingest tiles: %w", hubID, err)`. Do NOT ingest on the
  freeze path (a frozen hub does not advance accepted state) nor on non-verified verdicts.
- **The writer (`ingest.go`).** `ingestTiles` walks `tiles.TileCoords(treeSize)` then
  `tiles.BundleCoords(treeSize)` in order. For each `TileCoord{Level, Index, Partial}`:
  `logclient.FetchTile(ctx, fetcher, baseURL, c.Level, c.Index, c.Partial)` →
  `st.RecordTile(ctx, hubID, c.Level, c.Index, widthForP(c.Partial), raw, observedAt)`. For each
  `BundleCoord{Index, Partial}`: `logclient.FetchEntryBundle(...)` → `st.RecordEntryBundle(ctx, hubID,
  c.Index, widthForP(c.Partial), raw, observedAt)`. Keep the two loops as short, separate, pure-ish
  helpers if it reads cleaner, but ≤3 non-test files total.
- **Width translation (load-bearing, learnings "p↔width translation is the load-bearing bug surface").**
  Define an unexported `widthForP(p uint8) int` in the follower (`p==0 → tiles.TileWidth`, else
  `int(p)`) — re-deriving the store's one-liner, NOT exporting the store's. A full coord has `Partial==0`
  and must be stored at width 256; storing width 0 would make it unreadable by `SQLiteFetcher`. The test
  must pin this: a full `TileCoord{Partial:0}` round-trips at width 256.
- **Imports.** `follower.go`/`ingest.go` add `internal/tiles` to the existing `{context, fmt, logclient,
  metrics, store, time}` set. `internal/tiles` is a pure leaf (stdlib-only closure) — store stays a leaf,
  direction stays follower → {logclient, store, tiles}. go.mod/go.sum stay byte-identical (`tessera/api/
  layout` already in the closure via `internal/tiles`). Confirm with `git diff --quiet HEAD -- go.mod
  go.sum`.
- **Error contract (ADR-0006 + learnings).** A tile/bundle fetch fault is a genuine transport error here
  (NOT a violation): return it up so `PollHub` surfaces it and accepted state for the NEXT poll is
  unaffected — but note the checkpoint was already recorded/advanced above, so a mid-ingest fault leaves
  a partial mirror that the next poll re-fetches (idempotent upsert). Do NOT swallow the fetch error and
  do NOT freeze on it. (The "missing tile = skip equivocation" swallow lives in `checkConsistency`, a
  different concern — leave it untouched.)
- **Oracle gate is N/A for this slice** — transport + CRUD only, no signature/RFC-6962/Merkle/did:web/
  fsck path is introduced (the equivocation branch it un-dormants is already golden-tested; this slice
  does not change crypto code). The trust-root re-arms at the `fsck`-over-`SQLiteFetcher` slice.
- **Test (`ingest_test.go`).** Two parts: (1) a table-driven unit test of the coordinate→width mapping
  over a boundary tree size (e.g. 300 → tile level 0 full at width 256 + a 44-leaf partial at index 1;
  bundle index 0 full + index 1 partial 44), asserting the exact `(level, index, width)` writes via a
  fake/recording fetcher + store read-back (`ReadTileBlob`/`ReadEntryBundleBlob`). (2) Extend the
  `compositeFetcher` to also serve synthetic tile/bundle bytes for `tile/` URLs, drive `PollHub`
  end-to-end against the sb0 checkpoint fixture (tree size 10183), and assert that after a verified poll
  the store holds mirrored tile + bundle rows for the enumerated coords (assert on
  `ReadTileBlob`/`ReadEntryBundleBlob` returning `found==true`, never on follower internals — PRD
  "assert on observable outputs"). Keep test functions small and focused (no test classes).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass) and
  `gofmt -l .` is empty.
- `go test -run 'TestIngest|TestPollHub' -count=1 ./internal/follower` passes.
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (trust-root WASM-shared leaf unaffected).
- The coordinate unit test asserts a full tile (`Partial==0`) is stored at width `256` and a 44-leaf
  partial at width `44` — read back via `ReadTileBlob` at the matching width.
- After a verified `PollHub` against the sb0 fixture, `ReadEntryBundleBlob`/`ReadTileBlob` for at least
  one enumerated coord return `found==true` with the fetched bytes (store-observable assertion).

## Done When
`PollHub`, on a verified growing checkpoint, mirrors every `TileCoords`/`BundleCoords`-named tile and
bundle into the store at the correct `widthForP`-translated width, and all Verification criteria pass.
