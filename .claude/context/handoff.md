## 2026-06-21 — Unify the tlog-tiles `p`-vocabulary — store owns the only `p→width` translation

**Done:** Re-parameterized `Store.RecordTile` / `Store.RecordEntryBundle` to take the tlog-tiles
partial qualifier `p uint8` (not a pre-translated `width int`); each method now computes
`width := widthForP(p)` at the top via the package-private helper in `fetcher.go` (the single
`p→width` authority, shared with the `SQLiteFetcher` read side). Deleted the follower's duplicate
`func widthForP` and switched both ingest call sites to pass `c.Partial` straight through. Closes the
last open `normal` issue (ADR-0005 single-source-of-truth for the mirror coordinate mapping).

**Files changed:**
- `internal/store/tiles.go`: `RecordTile`/`RecordEntryBundle` signatures now end `…, p uint8, data
  []byte, observedAt time.Time)`; each computes `width := widthForP(p)` then feeds the unchanged
  INSERT/UPDATE, `tiles.IsFull(width)`, and `width %d` error args (so a full-tile error still reads
  `width 256`). SQL/schema/`is_full` discipline unchanged; file + method docstrings updated.
- `internal/follower/ingest.go`: both write call sites pass `c.Partial` directly; deleted the
  duplicate `func widthForP` (was lines 119-130); `ingestTiles`/`ingestHashTiles`/`ingestEntryBundles`
  docstrings now state the store owns the `p→width` translation.
- `internal/tiles/coords.go`: two `BundleCoord`/`TileCoord` doc comments that said the translation is
  "translated downstream by the ingestion writer" updated to "the store translates Partial to that
  width on write" (the only honest description after the move; pure comment edit, no code change).
- Tests (full `256`/`tiles.TileWidth` → `0`; partial width `W` → `uint8(W)`):
  `internal/store/tiles_test.go`, `internal/store/fetcher_test.go`,
  `internal/follower/equivocation_test.go` (translate `len(nodes)` → `p`),
  `internal/follower/fsck_test.go`, `internal/follower/ingest_test.go` (see Notes),
  `internal/logclient/fsck_test.go`, `internal/proofserve/entries_test.go`,
  `internal/proofserve/handler_test.go` (pass `c.Partial` straight from `TileCoords`),
  `internal/tilesserve/handler_test.go`, `cmd/iscc-monitor/main_test.go`.

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all `ok`, 15 packages);
`gofmt -l .` empty. Per-criterion:
- `grep -rn "func widthForP" internal/` → exactly one hit (`internal/store/fetcher.go`). PASS.
- `grep -n "widthForP" internal/follower/ingest.go` → empty (exit 1). PASS.
- `go test -count=1 -run 'TestRecordTile|TestRecordEntryBundle|TestFetcher' ./internal/store` → ok
  uncached (full + partial round-trip via `RecordTile`/`RecordEntryBundle` → `SQLiteFetcher`). PASS.
  Named regression guards `TestRecordTilePartialOverwrite` (p=100), `TestRecordTileRoundTrip`,
  `TestRecordEntryBundleRoundTrip`, `TestRecordTilePartialIsNotFull` all `--- PASS`.
- `go test -count=1 ./internal/follower ./internal/logclient ./internal/proofserve ./internal/tilesserve`
  → all `ok` uncached (full/partial mirror round-trips, fsck root-rebuild, equivocation
  consistency-proof all read the mirror back correctly). PASS.
- `go list -deps ./internal/store | grep -E "net/http|internal/logclient|internal/follower"` → empty
  (store stays a leaf). PASS. `git diff --stat -- internal/store/schema.sql go.mod go.sum` → empty.

**Next:** The last `normal` issue is now drained. Remaining `issues.md` entry is the `low`
`cmd/notecheck` vestigial `out io.Writer` param (loop-skipped). With the M3 mirror arc + three computed
proofs + HTTP serving + this single-source cleanup landed, the natural next arc is the proof-surface
cache / verify-for-me REST surface (deferred by this step's Not-In-Scope) — start by scoping the
proof-bundle assembly (`{checkpoint, inclusion proof, record bytes, hub key, ots?}`) that both the
in-browser verifier and verify-for-me share, reading from the now-unified store/SQLiteFetcher seam.

**Notes:**
- **One unlisted test file (`internal/follower/ingest_test.go`) referenced the deleted follower
  `widthForP` and had to be fixed to keep the gate green.** `next.md`'s test-file list omitted it.
  Two changes: (1) deleted `TestWidthForP` — it directly tested the now-deleted follower duplicate's
  arithmetic; the load-bearing "full coord (p=0) must be readable at width 256, NOT 0" invariant is
  still pinned by `TestIngestTilesWidthMapping` over the public `RecordTile`/`ReadTileBlob` surface
  (it asserts the full tile is found at width 256 AND absent at width 0). (2) The two `PollHub`
  integration tests used `widthForP(c.Partial)` as a local read-back helper; replaced with a
  test-local `readWidth(p) int` (deliberately NOT named `widthForP`, so the single-authority grep
  stays one hit) — needed because the read-side helpers (`ReadTileBlob`/`ReadEntryBundleBlob`) keep
  their `width int` param per Not-In-Scope. This is not a gate dodge: the deleted test targeted code
  that no longer exists, and the invariant has stronger coverage elsewhere.
- **`internal/tiles/coords.go` was touched (3rd production file) for comment accuracy only** — its
  `BundleCoord`/`TileCoord` docs claimed the writer does the translation, now false. Edit is
  comment-only (no code, closure/go.mod/go.sum byte-unchanged); keeps total production files at 3
  (the scope ceiling). Flagging since `next.md` listed only 2 modify targets.
- **`uint8(256)` was never written** — every full coord is the literal `0` per the lossy-guard rule.
  Test sites with a runtime `width` variable (`equivocation_test.go`'s `len(nodes)`,
  `entries_test.go`'s `last-first`) translate with an explicit `if width == tiles.TileWidth { p = 0 }`
  guard so a future full bundle/tile in those loops maps to `0`, never a wrapped `uint8(256)`.
- **Oracle/conformance gate correctly N/A for this slice** (plain CRUD + the existing pure `p→width`
  arithmetic; no signature/RFC-6962/Merkle/did:web/fsck path changed) — same posture the reviewer
  recorded for the original tile-writer slice. The four mirror-consuming packages
  (`follower`/`logclient`/`proofserve`/`tilesserve`) were still re-run **uncached** because the
  full/partial round-trips, fsck root-rebuild, and equivocation consistency-proof read the mirror back
  via `SQLiteFetcher` — all green, no conformance regression.
- **Unrelated `.devcontainer/devcontainer.json` change is present in the working tree but NOT committed
  by me** (pre-existing infra tuning: `--memory` runArgs + `GOFLAGS=-p=2`). Left for the human/infra
  owner.
