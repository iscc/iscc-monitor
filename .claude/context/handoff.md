# Handoff

## 2026-06-20 — Review of: Add the canonical tlog-tiles layout layer (`internal/tiles`) over `tessera/api/layout`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/tiles`, a thin golden-tested re-export of tessera's tlog-tiles
layout primitives (`TilePath`/`EntriesPath`/`PartialTileSize`, `TileWidth=256`/`TileHeight=8`) plus the
one ADR-0005 predicate `IsFull(width int) bool`, with `tessera v1.0.2` as a clean direct dep. Scope is
tight (1 production file + 1 test + the authorized go.mod/go.sum dep), all gates are green, and every
golden assertion is independently confirmed from tessera ground truth. The package stays a pure leaf
(`net/http`/`database/sql`-free, WASM-green), so it is safely shareable by the SQLite store keys and the
WASM proof path downstream.

**Verification:**
- [x] `mise run check` green — build + vet + test, all 8 packages ok.
- [x] `go test -count=1 ./internal/tiles` PASS (uncached).
- [x] `go list -m …/tessera` → `v1.0.2`; `go.mod` directive still `go 1.24.0`, no `toolchain` line;
  `go mod tidy` is a verified no-op (go.mod + go.sum byte-unchanged); `go mod verify` → all verified.
- [x] `go list -deps ./internal/tiles | grep -E '^net/http$|^database/sql$'` empty (leaf purity); the
  only `.Imports` entry is `api/layout`, whose own closure is stdlib-only; `GOOS=js GOARCH=wasm go build
  ./internal/tiles` green.
- [x] `gofmt -l .` empty.
- [x] Golden assertions hold — independently re-ran the real `layout` funcs: `TilePath(0,0,255)==
  "tile/0/000.p/255"`, `TilePath(1,0,0)=="tile/1/000"`, `TilePath(15,455667,0)=="tile/15/x455/667"`,
  `EntriesPath(0,8)=="tile/entries/000.p/8"`, `EntriesPath(255,0)=="tile/entries/255"`,
  `PartialTileSize(0,0,256)==0`, `IsFull(256)==true`, `IsFull(255)==false`. All four path strings are
  verbatim from tessera's `api/layout/paths_test.go`.
- [x] **`next.md`'s `PartialTileSize(0,0,300)==44` golden was factually WRONG; advance corrected it.**
  Verified against tessera `tile.go` and by re-running the real func: `(0,0,300)=0` (first 256-leaf tile
  is full), `(0,1,300)=44` (partial-44 spills into index 1), `(0,0,44)=44`. The corrected vectors are
  pinned in the test. This is a justified, well-documented deviation that makes the golden match the
  reference's real behavior — not a gate dodge.
- [x] **Oracle/conformance gate: N/A** (pure path strings — no signature/RFC-6962/Merkle/did:web/fsck
  path). Confirmed; it re-arms at the SQLiteFetcher + `fsck` slice. No `go run` of `cauldron/` done.
- [x] Gate-integrity scan over the 3 unpushed commits: no `//nolint`, `t.Skip`, build-tag exclusions,
  swallowed errors, or deleted tests/assertions. No existing file modified or deleted (only 2 new files
  + the authorized dep add).

**Issues found:** (none)

**Next:** Build the `SQLiteFetcher` — implement the tessera `client.Fetcher` seam (`{ReadCheckpoint,
ReadTile, ReadEntryBundle}`, see `cauldron/tessera/client/fetcher.go`) over the `tiles`/`entry_bundles`
store tables, keyed by `tiles.TilePath`/`EntriesPath` and gated by `IsFull`. This is the prerequisite
for sourcing the consistency-proof hashes the follower's deferred equivocation branch needs, and for
the M3 canonical-path mirror. The oracle/conformance gate re-arms there (`fsck` root-rebuild over the
`SQLiteFetcher`, inclusion cross-check) — it will need tile fixtures (still absent).

**Notes:**
- The `PartialTileSize` correction has a downstream consequence the SQLiteFetcher slice must honor: the
  44-leaf partial of a 300-leaf tree is at **index 1**, never index 0. Captured in learnings.
- `internal/tiles` (and `IsFull`) are an intentional unused-until-wired export seam — `go vet` clean,
  not dead code; the SQLiteFetcher store-key slice is its first caller.
- tessera v1.0.2's heavy deps (otel/klog/formats/`x/crypto`/backoff) are module-graph-only (never
  compiled), so they stay out of `go.mod`'s indirect block and the package closure; `x/sys` bumped
  0.37→0.41 from the require graph (benign, pure-Go).
- Working branch is `develop`, 3 commits ahead of `origin/develop` (now 4 incl. this review). Remote
  configured (github.com/iscc/iscc-monitor); pushing `develop` on this PASS verdict. Still no
  `.github/workflows/` — no CI; flag for whoever wires it (the external `notecheck` oracle becomes
  load-bearing once the tile-fetch / fixture slices arm the conformance gate).
