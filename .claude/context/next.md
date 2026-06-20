# Next Work Package

## Step: SQLiteFetcher — read mirrored tiles/bundles/checkpoint back as a `fsck.Fetcher`

## Goal
Give the store a write side for the empty `tiles` / `entry_bundles` tables (`RecordTile` /
`RecordEntryBundle`) and a `SQLiteFetcher` that reads tiles, entry bundles, and the latest checkpoint
back out — implementing tessera's three-method `Fetcher` interface (`ReadCheckpoint` / `ReadTile` /
`ReadEntryBundle`). This is the prerequisite seam for sourcing the RFC-6962 consistency-proof hashes the
deferred M1 equivocation branch needs, and the foundation for the M2 `fsck` root-rebuild and the M3
canonical-path mirror — all of which read mirror BLOBs from the local store, never re-hitting the hub.

## Scope
- **Create**:
  - `internal/store/tiles.go` — partial-tile-discipline write CRUD + read helpers over the `tiles` /
    `entry_bundles` tables:
    - `RecordTile(ctx, hubID int64, level, index uint64, width int, data []byte, observedAt time.Time) error`
    - `RecordEntryBundle(ctx, hubID int64, bundleIndex uint64, width int, data []byte, observedAt time.Time) error`
    - `ReadTileBlob(ctx, hubID int64, level, index uint64, width int) ([]byte, bool, error)`
    - `ReadEntryBundleBlob(ctx, hubID int64, bundleIndex uint64, width int) ([]byte, bool, error)`
    - `LatestCheckpointRaw(ctx, hubID int64) ([]byte, bool, error)`
  - `internal/store/fetcher.go` — `SQLiteFetcher` struct (`{Store *Store; HubID int64}`) with the three
    `fsck.Fetcher` methods, composing the helpers above and honoring the partial→full fallback +
    `os.ErrNotExist` contract.
  - `internal/store/tiles_test.go` + `internal/store/fetcher_test.go` (tests; do not count against the
    3-file budget).
- **Modify**: (none — keep `checkpoints.go`, `schema.sql`, `internal/tiles/layout.go` byte-identical)
- **Reference**:
  - `internal/tiles/layout.go` — `IsFull(width int) bool`, `TileWidth = 256`, `PartialTileSize`,
    `TilePath` / `EntriesPath` (the width math the fetcher's p-argument maps onto).
  - `internal/store/schema.sql` (lines 71–97) — the `tiles` PK `(hub_id, level, tile_index, width)` and
    `entry_bundles` PK `(hub_id, bundle_index, width)`, plus `is_full`, `sha256`, `updated_at` columns.
  - `internal/store/checkpoints.go` — port the conventions: the `unixOrNil` helper for `updated_at`, the
    `errors.Is(err, sql.ErrNoRows)` → `(…, false, nil)` "absent is not an error" pattern (from
    `CheckpointAt` / `LookupHubKey`), and the `uint→int64` column casts. Since `tiles` / `entry_bundles`
    have real composite PKs, prefer an `INSERT … ON CONFLICT(<pk cols>) DO UPDATE SET data=excluded.data,
    is_full=excluded.is_full, sha256=excluded.sha256, updated_at=excluded.updated_at` upsert (cleaner
    than the guarded-UPDATE-then-INSERT idiom `RecordHubKey` uses for its UNIQUE-less table).
  - `cauldron/tessera/client/fetcher.go` (lines 97–132, `HTTPFetcher`/`FileFetcher`) — the exact method
    shapes to mirror, including the `os.ErrNotExist` return on a 404 / missing file and the
    `fetcher.PartialOrFullResource` partial→full fallback wrapper.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/fsck/fsck.go` (lines 36–41) — the
    canonical `Fetcher` interface (`ReadCheckpoint` / `ReadTile(ctx, l, i uint64, p uint8)` /
    `ReadEntryBundle(ctx, i uint64, p uint8)`) the `SQLiteFetcher` must satisfy.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/internal/fetcher/fallback.go` — the
    `PartialOrFullResource(ctx, p, f)` semantics (p>0 + `os.ErrNotExist` → retry with p=0). It is an
    `internal/` package and **cannot be imported** — replicate its tiny logic inline.

## Not In Scope
- Wiring `CheckEquivocation` into `follower.checkConsistency` — that is the *next* slice and depends on
  this one; do not touch `internal/follower/`.
- Running `fsck.New(...).Check(...)` / the full root-rebuild oracle, capturing real tile fixtures, or the
  inclusion cross-check — those land in the dedicated conformance slice once fixtures exist. This slice
  only proves the read/write round-trip + the not-exist contract with synthetic in-test BLOBs.
- `ProofBuilder`, the `iscc_index` writer, the bundle-hasher, and any REST/serving surface.
- Changing `schema.sql` (the `tiles` / `entry_bundles` tables already exist with the right PKs) or any
  pure-`internal/tiles` path math.
- Importing `tessera/fsck` or `tessera/client` into **production** store code — `SQLiteFetcher` satisfies
  `fsck.Fetcher` structurally, so it must NOT pull `net/http`/otel/klog into the store closure. Pin
  conformance with a `var _ fsck.Fetcher = SQLiteFetcher{}` assertion in the **test** file only.

## Implementation Notes
- **Partial-tile discipline (ADR-0005 / learnings "Correctness rules").** A row is keyed by `width`. Set
  the `is_full` column to `1` **only** when `tiles.IsFull(width)` (i.e. `width == 256`); a partial is
  `width < 256` with `is_full = 0`. Never promote a partial to full. Re-fetched partials overwrite in
  place via the PK upsert; a full tile is immutable but an idempotent re-write of identical bytes is
  harmless. Store `sha256 = sha256.Sum256(data)[:]` (`crypto/sha256` is stdlib — does not dirty the leaf)
  and `updated_at` from the injected `observedAt time.Time` via `unixOrNil` (mirrors how `RecordHubKey`
  takes a caller-supplied time rather than calling `time.Now()` in the method).
- **The p-argument ↔ width mapping is the load-bearing translation.** The `fsck.Fetcher` methods take
  `p uint8` where `p == 0` means "full" (path-API convention, per `internal/tiles` docstrings); the
  SQLite `width` column stores the actual leaf count (`256` for full). So `ReadTile(ctx, l, i, p)` must
  query `width = 256` when `p == 0`, else `width = int(p)`; same for `ReadEntryBundle`. Get this backwards
  and a full-tile request would miss the stored row — pin it with the round-trip test below.
- **Partial→full fallback (replicate `PartialOrFullResource` inline).** `tessera/internal/fetcher` is not
  importable. In `ReadTile` / `ReadEntryBundle`: read at `p`; if `p > 0` and the inner read is
  `os.ErrNotExist`, retry the read at full width (`256`) and return that; if `p == 0` and not found,
  return the `os.ErrNotExist` (wrapped `%w`). This matches `fallback.go` exactly and the `cauldron`
  `HTTPFetcher`/`FileFetcher` behavior.
- **Not-exist contract.** A missing row makes the inner helper return `found=false`; the fetcher maps
  `!found` → `fmt.Errorf("…: %w", os.ErrNotExist)` so `errors.Is(err, os.ErrNotExist)` survives for the
  fallback and any future `fsck` consumer. `ReadCheckpoint` reads the most-recently observed raw
  checkpoint via `LatestCheckpointRaw`: `SELECT raw FROM checkpoints WHERE hub_id = ? ORDER BY tree_size
  DESC LIMIT 1`; no row → `os.ErrNotExist`. (The existing `CheckpointAt` is size-keyed; this new
  size-agnostic read is what the fetcher needs.)
- **store stays a leaf (learnings).** Do NOT import `internal/logclient` or anything pulling `net/http`
  into the store closure. The `SQLiteFetcher` reads `s.Store.db` directly via the helpers (same package,
  so the unexported `db` field is reachable). Verify `go list -deps ./internal/store | grep '^net/http$'`
  stays empty.
- **Oracle/conformance gate is N/A for this slice** — plain CRUD + BLOB round-trip; no signature /
  RFC-6962 / Merkle / did:web / `fsck`-rebuild path is exercised (the rebuild oracle re-arms in the
  dedicated fixtures+fsck slice). Keep `go.mod` / `go.sum` / `schema.sql` byte-identical; no new dep is
  needed (tessera is already required), so any `go get` is a red flag.

## Verification
- `mise run check` is green (build + vet + test across all packages; `gofmt -l .` empty).
- `go test -count=1 ./internal/store` passes (uncached).
- Round-trip assertion: after `UpsertHub` to get a `hubID`, `RecordTile(hub, level=0, index=0, width=256,
  data, t)` then `SQLiteFetcher{Store,HubID:hub}.ReadTile(ctx, 0, 0, 0)` returns the identical `data`
  bytes, and `ReadTileBlob(hub, 0, 0, 256)` confirms the stored row; a `width=44` partial write then
  `ReadTile(ctx, 0, 0, 44)` returns those bytes. Likewise for `RecordEntryBundle` / `ReadEntryBundle`.
- `is_full` assertion: a `width=256` write sets `is_full=1`; a `width=44` write sets `is_full=0`
  (asserted via a raw `SELECT is_full …` in the test).
- Not-exist assertion: `ReadTile` / `ReadEntryBundle` for an un-written (level,index) returns an error
  satisfying `errors.Is(err, os.ErrNotExist)`; `ReadCheckpoint` on a hub with no checkpoint row likewise.
- Partial→full fallback assertion: with only a `width=256` full tile stored, `ReadTile(ctx, l, i, p=200)`
  (a partial request) still returns the full tile's bytes (mirrors `PartialOrFullResource`).
- `ReadCheckpoint` returns the **highest-`tree_size`** raw checkpoint bytes when two sizes are stored.
- `var _ fsck.Fetcher = SQLiteFetcher{}` compiles in the test file (interface conformance pinned).
- `go list -deps ./internal/store | grep -E '^net/http$|^database/sql$'` shows `database/sql` (expected,
  already present) but **no** `net/http` (store stays a leaf).
- `git diff --quiet HEAD -- internal/store/checkpoints.go internal/store/schema.sql internal/tiles` (the
  reference files are untouched).

## Done When
`SQLiteFetcher` satisfies `fsck.Fetcher`, round-trips tile / entry-bundle / checkpoint BLOBs through the
store with the partial-tile-discipline width mapping and the `os.ErrNotExist` partial→full fallback, and
all Verification criteria pass with `mise run check` green.
