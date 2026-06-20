# Next Work Package

## Step: Add the canonical tlog-tiles layout layer (`internal/tiles`) over `transparency-dev/tessera/api/layout`

## Goal
Introduce the pure tlog-tiles path/coordinate math the monitor needs everywhere downstream — the
canonical tile/entry-bundle paths, the partial-tile width (`is_full` only at `width==256`), and the
`PartialTileSize` discipline — by **reusing** `transparency-dev/tessera/api/layout` (target mandate)
behind a thin, golden-tested `internal/tiles` package. This is the smallest pure first slice of the
tile-fetch / `SQLiteFetcher` / `ProofBuilder` infrastructure that the state, handoff, and learnings all
name as the prerequisite for both wiring the M1 equivocation trigger and starting M2. No I/O, no
follower wiring — pure path math first.

## Scope
- **Create**: `internal/tiles/layout.go` — a thin wrapper exposing the tlog-tiles layout primitives the
  monitor will key its SQLite mirror on. Re-export (do not reimplement) tessera's `layout.TilePath`,
  `layout.EntriesPath`, `layout.PartialTileSize`, and the `TileWidth = 256` / `TileHeight = 8`
  constants, plus one project-specific helper `IsFull(width uint8) bool` encoding the ADR-0005
  partial-tile rule. Keep the package import-clean (the `api/layout` closure is stdlib-only, so this
  package must stay free of `net`/`net/http`/`database/sql` — it is the shared layout leaf, like
  `didweb`).
- **Create**: `internal/tiles/layout_test.go` — table-driven golden tests (this is the test file; it
  does not count against the 3-file budget).
- **Modify**: `go.mod`, `go.sum` — add `github.com/transparency-dev/tessera v1.0.2` as a direct require
  (the authorized dep add; the only non-test, non-created change).
- **Reference**:
  - `/workspace/iscc-monitor/cauldron/tessera/client/fetcher.go` — shows the canonical
    `{ReadCheckpoint, ReadTile, ReadEntryBundle}` Fetcher seam and how `layout.TilePath`/
    `layout.EntriesPath` feed it (the next slice builds the SQLite-backed version of this).
  - Module-cache reference (read-only; do not import beyond `api/layout`):
    `$(go env GOMODCACHE)/github.com/transparency-dev/tessera@v1.0.2/api/layout/paths.go` and
    `.../api/layout/tile.go` — the exact `TilePath`/`EntriesPath`/`PartialTileSize`/`TileWidth`
    semantics.
  - `.../api/layout/paths_test.go` — the reference's own golden vectors (ground-truth, not
    author-asserted): `TilePath(0,0,255) == "tile/0/000.p/255"`, `TilePath(1,0,0) == "tile/1/000"`,
    `TilePath(15,455667,0) == "tile/15/x455/667"`, `EntriesPath(0,8) == "tile/entries/000.p/8"`,
    `EntriesPath(255,0) == "tile/entries/255"`.
  - `/workspace/iscc-monitor/internal/store/schema.sql` (the `tiles` / `entry_bundles` tables:
    `(level, tile_index, width)` / `(bundle_index, width)` PKs, `is_full` default 0) — the SQLite
    columns the layout keys map onto in the *next* slice.
  - Learnings: ADR-0005 partial-tile discipline (`is_full` only at `width==256`, never promote a
    partial) and the `merkle v0.0.2` / `sqlite v1.46.1` go-directive pitfall (verify the new dep keeps
    `go 1.24.0`, no `toolchain` line).

## Not In Scope
- **No `SQLiteFetcher`, no HTTP tile fetcher, no `ProofBuilder`, no `fsck`** — this slice is path/width
  math only. Implementing the `client.Fetcher` interface over the store is the very next slice and needs
  this layer first.
- **No follower wiring** — do not touch `internal/follower`, do not add the third `checkConsistency`
  equivocation branch, do not source consistency-proof hashes. That waits on the fetcher slice.
- **No store changes** — do not add tile/entry-bundle CRUD to `internal/store/checkpoints.go`; the
  schema columns already exist and stay byte-identical here.
- **No tile fixtures** — do not capture real hash tiles from the live hubs; the golden vectors are pure
  path strings, not tile bytes.
- **Do not import any tessera package other than `api/layout`** (e.g. `client`, `fsck`, `api`): those
  pull `net/http`/otel/klog and would dirty this leaf and the WASM-shareable purity target.

## Implementation Notes
- **Reuse, do not reimplement (target Stack rule).** `internal/tiles/layout.go` should be a thin
  re-export, e.g. `func TilePath(level, index uint64, width uint8) string { return
  layout.TilePath(level, index, width) }`, a matching `EntriesPath(index uint64, width uint8) string`,
  `PartialTileSize(level, index, logSize uint64) uint8`, and `const TileWidth = layout.TileWidth` /
  `TileHeight = layout.TileHeight`. Wrapping (not aliasing) gives the monitor a stable seam plus a
  docstring per the project's "start each file with a docstring" rule; do not copy the body of tessera's
  path math.
- **`IsFull` is the one project-specific predicate** and encodes ADR-0005. Mind the two encodings of
  "full": tessera's *path* API represents a full tile as **width `0`** (the `p > 0` partial suffix),
  while the `tiles.width` SQLite **column** stores the actual leaf count (`256` for full, per
  schema/learnings). `IsFull(width uint8) bool` must adopt the **column** convention — `width ==
  TileWidth` → full (`true`), anything `< TileWidth` → partial (`false`). Document in the docstring that
  this package owns that translation so callers never confuse path-API "0 == full" with column "256 ==
  full". (`uint8` cannot hold 256, so define `TileWidth`/the comparison against a wider int, or document
  that the column width caps at 255-as-partial vs a full sentinel — pick one and pin it in the test;
  prefer comparing against `layout.TileWidth` as an untyped/int constant so `IsFull` reads `int(width)
  == TileWidth` if you keep the `uint8` arg, OR take `width int`.)
- **Dep-add hygiene (learnings pitfall, verified during scoping).** Run `go get
  github.com/transparency-dev/tessera@v1.0.2` then `go mod tidy`. tessera v1.0.2's go directive is
  `go 1.24.0`, and `api/layout`'s compiled closure is **stdlib-only** (`cmp`, `slices`), so the heavy
  otel/klog/formats deps land in `go.sum` as module-graph requirements only (never compiled), exactly
  like `go-cmp` did for merkle. After tidy, confirm `go.mod`'s directive is still `go 1.24.0` with **no
  `toolchain` line** (drop it if auto-injected). If `go mod tidy` tries to bump the directive past
  `1.24.0`, stop — that is a gate regression, not this step.
- **Golden vectors are ground truth from the reference, not author-asserted.** Use the exact strings
  from `api/layout/paths_test.go` (listed under Reference). Because `internal/tiles` only delegates, the
  test proves the delegation wiring and pins the canonical paths so a future refactor cannot silently
  diverge. The oracle/conformance gate is **N/A** here (no signature/RFC-6962/Merkle/did:web path; pure
  path strings) — say so in the handoff; it re-arms at the fetcher + `fsck` slice.
- **Purity is load-bearing** (learnings `internal/didweb` nuance): verify with `go list -deps
  ./internal/tiles | grep -E '^net/http$|^database/sql$'` returning empty, not by grepping `os` out (it
  rides in transitively via `fmt`). This package is destined to be shared by the SQLite store keys and
  the M3 canonical-path mirror, so keep it a leaf.

## Verification
- `mise run check` is green (build + vet + test all packages, `gofmt -l .` empty).
- `go test -count=1 ./internal/tiles` passes.
- `go list -m github.com/transparency-dev/tessera` prints `v1.0.2`; `go.mod`'s `go` directive is still
  `go 1.24.0` with no `toolchain` line; `go mod tidy` is a no-op afterward and `go mod verify` passes.
- `go list -deps ./internal/tiles | grep -E '^net/http$|^database/sql$'` is empty (leaf purity).
- Golden assertions hold (ground truth from `api/layout/paths_test.go` / `tile.go`):
  - `tiles.TilePath(0, 0, 255) == "tile/0/000.p/255"`
  - `tiles.TilePath(1, 0, 0) == "tile/1/000"`
  - `tiles.TilePath(15, 455667, 0) == "tile/15/x455/667"`
  - `tiles.EntriesPath(0, 8) == "tile/entries/000.p/8"`
  - `tiles.EntriesPath(255, 0) == "tile/entries/255"`
  - `tiles.PartialTileSize(0, 0, 300) == 44` (300 % 256 = 44; the first tile of a 300-leaf tree is
    partial-44 — confirm against `tile.go`'s `PartialTileSize`)
  - `tiles.PartialTileSize(0, 0, 256) == 0` (an exactly-full first tile reports 0 = full)
  - `tiles.IsFull(256) == true` and `tiles.IsFull(255) == false` (ADR-0005 width-256 rule)

## Done When
`advance` is done when `internal/tiles` re-exports the tessera tlog-tiles layout primitives behind a
golden-tested seam (with the `IsFull` ADR-0005 predicate), tessera v1.0.2 is a clean direct dep that
keeps the `go 1.24.0` directive, the package's dep closure is `net/http`/`database/sql`-free, and every
Verification criterion passes.
