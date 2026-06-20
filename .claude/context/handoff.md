# Handoff

## 2026-06-20 — Add the canonical tlog-tiles layout layer (`internal/tiles`) over `tessera/api/layout`

**Done:** Added `internal/tiles`, a thin, golden-tested re-export of tessera's tlog-tiles layout
primitives (`TilePath`, `EntriesPath`, `PartialTileSize`, `TileWidth=256`, `TileHeight=8`) plus the one
project-specific ADR-0005 predicate `IsFull(width int) bool` (column convention: `width == 256` → full).
`github.com/transparency-dev/tessera v1.0.2` is now a clean direct dep that keeps the `go 1.24.0`
directive, and the package's dep closure is `net/http`/`database/sql`-free (WASM build green).

**Files changed:**
- `internal/tiles/layout.go` (created): thin wrappers delegating to `layout.TilePath`/`EntriesPath`/
  `PartialTileSize`, the two spec constants, and `IsFull`. The docstring documents that this package
  owns the path-API "0 == full" vs column "256 == full" translation; `IsFull` takes `int` because the
  full sentinel 256 cannot fit in `uint8`.
- `internal/tiles/layout_test.go` (created): table-driven golden tests pinning the canonical path
  strings (ground truth from tessera's `api/layout/paths_test.go`), `PartialTileSize`, `IsFull`, and the
  constants.
- `go.mod`, `go.sum`: added `tessera v1.0.2` as a direct require via `go get` + `go mod tidy`.

**Verification:** `mise run check` → green (build + vet + test, all 8 packages ok; `gofmt -l .` empty).
Per-criterion:
- [x] `go test -count=1 ./internal/tiles` PASS.
- [x] `go list -m …/tessera` → `v1.0.2`; `go.mod` directive still `go 1.24.0`, no `toolchain` line;
  `go mod tidy` is a no-op; `go mod verify` → all modules verified.
- [x] `go list -deps ./internal/tiles | grep -E '^net/http$|^database/sql$'` empty (leaf purity);
  `GOOS=js GOARCH=wasm go build ./internal/tiles` green; the only `.Imports` entry is `api/layout`.
- [x] Golden path strings all hold: `TilePath(0,0,255)=="tile/0/000.p/255"`, `TilePath(1,0,0)=="tile/1/000"`,
  `TilePath(15,455667,0)=="tile/15/x455/667"`, `EntriesPath(0,8)=="tile/entries/000.p/8"`,
  `EntriesPath(255,0)=="tile/entries/255"`, `PartialTileSize(0,0,256)==0`, `IsFull(256)==true`,
  `IsFull(255)==false`.
- **Oracle/conformance gate: N/A** (no signature/RFC-6962/Merkle/did:web/fsck path — pure path strings).
  It re-arms at the SQLiteFetcher + `fsck` slice. No `go run` of `cauldron/` was needed or done.

**Next:** Build the `SQLiteFetcher` — implement the tessera `client.Fetcher` seam
(`{ReadCheckpoint, ReadTile, ReadEntryBundle}`, see `cauldron/tessera/client/fetcher.go`) over the
`tiles`/`entry_bundles` store tables, keyed by `tiles.TilePath`/`EntriesPath` and gated by `IsFull`.
That slice sources the consistency-proof hashes the follower's deferred equivocation branch needs, and
is the prerequisite for both the M1 equivocation trigger wiring and the M3 canonical-path mirror.

**Notes:**
- **One `next.md` golden vector was factually wrong and I corrected it to genuine tessera ground truth.**
  `next.md` asserted `PartialTileSize(0, 0, 300) == 44` ("the first tile of a 300-leaf tree is
  partial-44"). That is incorrect: per tessera's `tile.go`, `sizeAtLevel=300`, `fullTiles=300/256=1`,
  and `index 0 < fullTiles` → returns **0** (the first 256-leaf tile is *full*); the leftover 44 spill
  into the *second* tile, so the partial-44 is at **index 1**. I verified this directly against
  `layout.PartialTileSize` (`(0,0,300)=0`, `(0,1,300)=44`, `(0,0,44)=44`) and pinned the corrected
  vectors. `PartialTileSize(0,0,256)==0` (from `next.md`) was correct and is kept. This is a documented
  deviation from `next.md`'s literal Verification text, but it makes the test match the reference's real
  behavior — pinning the wrong coordinate would have been a false golden. Flagging for `review`'s
  awareness; not a `**HUMAN REVIEW REQUESTED:**` item (no public-API or ADR change, just a test vector
  fix). This also matters downstream: the SQLiteFetcher slice must use index-1 (not index-0) for the
  44-leaf partial of a 300-leaf tree.
- tessera v1.0.2's heavy deps (otel, klog, `formats`, `x/crypto`, `cenkalti/backoff`) land in `go.sum`
  as module-graph requirements only — they are never compiled, so they do **not** appear in `go.mod`'s
  indirect block and do not enter the `internal/tiles` closure (verified). `golang.org/x/sys` and
  `x/sync` got minor version bumps from the tessera require graph (`sys 0.37→0.41`); benign, still pure-Go.
- `IsFull` is an intentional unused-until-wired export seam (like the consistency triggers) — `go vet`
  clean, not dead code. The SQLiteFetcher/store-key slice is its first caller.
