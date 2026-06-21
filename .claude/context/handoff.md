## 2026-06-21 — Review of: Pure hash-tile (multi-level) coordinate enumeration (`tiles.TileCoords`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the pure `TileCoord{Level, Index uint64; Partial uint8}` +
`TileCoords(treeSize uint64) []TileCoord` to `internal/tiles/coords.go` — the multi-level hash-tile
sibling of `BundleCoords` — with a golden table test plus power-of-256, zero-case, and oracle
cross-check tests. The diff is exactly what `next.md` asked (1 production file + 1 new test + handoff,
no caller wired), delegates every `Partial` to tessera's `PartialTileSize`, and the goldens are
independently confirmed as tessera ground truth. Clean, scoped, non-vacuous.

**Verification:**
- [x] `mise run check` — green (build + vet + test; all 11 packages `ok`, `go vet ./...` clean).
- [x] `gofmt -l internal/tiles/` — empty. Also `gofmt -l .` (whole tree) — empty.
- [x] `go test -run TestTileCoords -count=1 ./internal/tiles` — passes (7 table vectors + 2 power-of-256
      boundary tests + non-nil-empty zero case + oracle cross-check; 11 subtests PASS).
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` — exit 0 (no dependency change; `PartialTileSize`
      already in the closure via `layout.go`). Working tree also byte-identical to HEAD.
- [x] `GOOS=js GOARCH=wasm go build ./internal/tiles` — exit 0 (leaf-purity invariant holds).
- [x] Golden assertions independently re-derived against the tessera oracle (throwaway module, NOT the
      monitor's code): `0→[]`, `1→[{0,0,1}]`, `255→[{0,0,255}]`, `256→[{0,0,0}]`,
      `257→[{0,0,0},{0,1,1},{1,0,1}]`, `300→[{0,0,0},{0,1,44},{1,0,1}]`,
      `513→[{0,0,0},{0,1,0},{0,2,1},{1,0,2}]`, `65536→`257 entries ending `{1,0,0}`,
      `65537→`contains `{0,256,1}` and ends `{1,0,0}` (full root — extra leaf splits only level-0). All
      match byte-for-byte.
- [x] Mutation test (non-vacuousness, both reverted): (1) `Partial` from `index+1` → golden + oracle
      cross-check FAIL; (2) drop the stop-after-emit `break` → all three boundary tests FAIL. A
      green-but-wrong enumeration cannot ship.
- [x] Scope discipline — diff touches only `internal/tiles/{coords.go,tilecoords_test.go}` + handoff;
      no caller wired (`grep TileCoords` outside the package + its test = empty); nothing from
      `## Not In Scope` leaked; the open `low` notecheck issue correctly left untouched.
- [x] Gate-circumvention scan across all 3 unpushed commits (`@{upstream}..HEAD`) — no `//nolint`,
      `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion (10 live assertions in the
      new test).
- [x] Oracle/conformance gate correctly N/A (pure coordinate math; no signature/RFC-6962/Merkle/
      did:web/`fsck`-rebuild path). Sanity sweep ran anyway: didweb/logclient/follower conformance
      tests pass uncached, and `derive_vkey.py` reproduces both golden vectors (`40b74463`/`22b08f3e`).

**Issues found:** (none)

**Next:** Both pure coordinate sources now exist (`BundleCoords` + `TileCoords`), so the **M2 live
tile-ingestion writer** is the next slice: a `PollHub` fetch loop that, on a verified growing
checkpoint, walks `TileCoords(size)` + `BundleCoords(size)`, fetches each tile/bundle over the
`logclient` `Fetcher` seam at its `TilePath`/`EntriesPath`, and writes via `RecordTile`/
`RecordEntryBundle` (translating the path-API `Partial` to the store's `width` via `widthForP`,
re-fetching partials every poll per ADR-0005). That writer unblocks the still-deferred `fsck`
root-rebuild over `SQLiteFetcher` and the inclusion cross-check vs the hub's real
`IsccLogInclusionProof` — both want real tile/bundle fixtures in `testdata/live/`, the point where the
trust-root oracle (`notecheck`/`fsck`) re-arms.

**Notes:**
- The ingestion writer is the first real consumer that exercises the path-`p` → store-`width`
  translation end-to-end across the 256-leaf tile boundary; watch that a full-tile request (`p==0`)
  width-keys to 256, not 0 (the `widthForP` seam already pins this in the store layer).
- `internal/tiles` stays a pure stdlib-closure leaf (`api/layout`-only); keep it that way — it is
  WASM-shared territory.
- Branch is `develop`; `origin/develop` is the upstream. Pushing on PASS.
