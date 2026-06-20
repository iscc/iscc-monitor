# Handoff

## 2026-06-20 — SQLiteFetcher — read mirrored tiles/bundles/checkpoint back as a `fsck.Fetcher`

**Done:** Added the write/read CRUD for the `tiles` / `entry_bundles` tables (`RecordTile`,
`RecordEntryBundle`, `ReadTileBlob`, `ReadEntryBundleBlob`) plus a size-agnostic `LatestCheckpointRaw`,
and a `SQLiteFetcher{Store, HubID}` that satisfies tessera's three-method Fetcher shape
(`ReadCheckpoint` / `ReadTile` / `ReadEntryBundle`) with the p↔width mapping and an inline
`PartialOrFullResource` partial→full fallback honoring the `os.ErrNotExist` contract. Store stays a leaf.

**Files changed:**
- `internal/store/tiles.go` (new): partial-tile-discipline upsert CRUD + read helpers + `LatestCheckpointRaw`. `is_full=1` only at `width==256` (via `tiles.IsFull`); `sha256`+`updated_at` recorded from injected `observedAt`; composite-PK `ON CONFLICT … DO UPDATE` upsert overwrites partials in place. Absent row → `(nil, false, nil)`.
- `internal/store/fetcher.go` (new): `SQLiteFetcher`. `widthForP(p)` maps `p==0→256`, else `int(p)`. `ReadTile`/`ReadEntryBundle` replicate `PartialOrFullResource` inline (p>0 + `os.ErrNotExist` → retry at full). Misses wrap `os.ErrNotExist` with `%w`. No `fsck`/`net/http` import in production code.
- `internal/store/tiles_test.go` (new): round-trip, `is_full` column assertion, partial overwrite (1 row), absent-is-not-error, `LatestCheckpointRaw` highest-size + absent.
- `internal/store/fetcher_test.go` (new): full/partial round-trip via `ReadTile`/`ReadEntryBundle`, `os.ErrNotExist` on miss, partial→full fallback, both-missing still `os.ErrNotExist`, `ReadCheckpoint` highest-size + absent. Conformance pinned via a local `fsckFetcher` interface (see Notes).

**Verification:** `mise run check` → green (build + vet + test, all 8 packages ok). `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/store` PASS (uncached); all 22 store tests pass verbosely.
- [x] Tile round-trip: `RecordTile(0,0,256,…)` → `ReadTile(0,0,p=0)` returns identical bytes; `ReadTileBlob(0,0,256)` confirms the row; `width=44` partial → `ReadTile(0,1,44)` returns its bytes.
- [x] `is_full`: raw `SELECT is_full` = 1 for width 256, 0 for width 44 (tiles **and** entry_bundles).
- [x] Not-exist: `ReadTile`/`ReadEntryBundle`/`ReadCheckpoint` on un-written keys satisfy `errors.Is(err, os.ErrNotExist)`.
- [x] Partial→full fallback: only-full stored + `ReadTile(p=200)` returns the full bytes; both-missing `ReadTile(p=100)` still `os.ErrNotExist`.
- [x] `ReadCheckpoint` returns highest-`tree_size` raw bytes (two sizes stored, asserts the higher).
- [x] Conformance assertion compiles (`var _ fsckFetcher = SQLiteFetcher{}`).
- [x] `go list -deps ./internal/store` shows `database/sql`, **no** `net/http` (store stays a leaf).
- [x] `git diff --quiet HEAD -- internal/store/checkpoints.go internal/store/schema.sql internal/tiles` → clean (reference files untouched).
- [x] `go.mod` / `go.sum` byte-identical to pre-change; `go mod tidy` is a verified no-op.

**Next:** Wire `CheckEquivocation` into `follower.checkConsistency` — source the RFC-6962 consistency-proof
hashes from the `SQLiteFetcher` (the prerequisite this slice built). That is the deferred equivocation
branch; it touches `internal/follower/` and needs tile fixtures for an end-to-end test. The dedicated
`fsck` root-rebuild conformance slice (real tile fixtures + `fsck.New(...).Check(...)` over the
`SQLiteFetcher` + inclusion cross-check vs the hub's `IsccLogInclusionProof`) is still pending fixtures.

**Notes:**
- **DESIGN DEVIATION (needs review sign-off): `var _ fsck.Fetcher` assertion does NOT import the real
  `tessera/fsck`.** `next.md` required pinning conformance with `var _ fsck.Fetcher = SQLiteFetcher{}`
  in the test AND keeping `go.mod`/`go.sum` byte-identical with "no new dep needed (tessera already
  required)". These two are **mutually exclusive**: `fsck`'s package closure imports
  otel/klog/errgroup/transparency-dev-formats (verified — `client/otel.go`, `fetcher.go`,
  `client.go`), so importing it even in a `_test.go` makes `go mod tidy` add ~9 indirect requires and
  ~31 `go.sum` lines, and `go build`/`vet`/`test` fail under `-mod=readonly` with "updates to go.mod
  needed". The `next.md` premise was wrong about `fsck`'s closure (it was true only for `internal/tiles`
  → `api/layout`, which is stdlib-only). Resolution: I copied `fsck.Fetcher`'s interface verbatim into
  the test as `fsckFetcher` and asserted against it. I diffed it against the real interface
  (`fsck@v1.0.2/fsck.go:37-41`) — method names, param names, types, returns all identical — so any
  signature drift still breaks the build; the conformance guarantee is equivalent, the dep closure is
  not pulled in, go.mod/go.sum stay byte-identical, and store stays a leaf. The production
  `SQLiteFetcher` is unaffected. If `review` prefers the literal `fsck` import, that requires accepting
  the indirect-dep additions (`go mod tidy`) and re-checking store leaf purity — I judged byte-identical
  go.mod + leaf purity to be the higher-priority, explicitly-load-bearing constraint. **HUMAN REVIEW
  REQUESTED** only if the literal `fsck` import is considered mandatory over byte-identical go.mod.
- The actual `fsck.New(...).Check(...)` root-rebuild is out of scope here (no fixtures) and unaffected:
  it lives in `cmd/` or a future conformance package that *can* take the `fsck` dep; this slice only
  proves the read/write round-trip + not-exist contract with synthetic in-test BLOBs, as scoped.
- `boolToInt` is a small new private helper in `tiles.go` (schema's INTEGER 0/1 from a Go bool), beside
  the existing `unixOrNil`/`nullStringOrNil` helpers in `checkpoints.go` — same package, same style.
- Oracle/conformance gate correctly **N/A** for this slice (plain CRUD + BLOB round-trip; no
  signature / RFC-6962 / Merkle / did:web / `fsck`-rebuild path exercised). No `go run` of `cauldron/`.
  Still no `.github/workflows/` in the repo — CI/`notecheck` remains unwired (flagged in prior handoff).
