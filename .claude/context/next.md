# Next Work Package

## Step: Unify the tlog-tiles `p`-vocabulary — store owns the only `p→width` translation

## Goal
Eliminate the duplicated, load-bearing `p→width` translation by making `RecordTile` /
`RecordEntryBundle` accept the tlog-tiles `p uint8` qualifier (not a pre-translated `width int`),
keeping the single `widthForP` authority private to `internal/store`, and deleting the follower's
copy. This closes the last open `normal` issue (ADR-0005 / single source of truth for the mirror
coordinate mapping) — a real bug surface, since a follower-side copy that ever stored a full tile at
width 0 instead of 256 would make the `SQLiteFetcher` unable to read the mirror back.

## Scope
- **Create**: (none)
- **Modify** (production files — this is 2, ≤3):
  - `internal/store/tiles.go` — change the signatures to
    `RecordTile(ctx, hubID int64, level, index uint64, p uint8, data []byte, observedAt time.Time)`
    and `RecordEntryBundle(ctx, hubID int64, bundleIndex uint64, p uint8, data []byte, observedAt
    time.Time)`. At the top of each method compute `width := widthForP(p)` (the package-private helper
    already defined in `fetcher.go`) and feed that `width` to the unchanged INSERT/UPDATE,
    `tiles.IsFull(width)`, and error-format args. SQL, schema, and the `is_full` discipline do not
    change — only the public parameter and the internal translation move into the store.
  - `internal/follower/ingest.go` — at the two write call sites (`ingestHashTiles`,
    `ingestEntryBundles`) pass `c.Partial` directly instead of `widthForP(c.Partial)`, and **delete**
    the duplicate `func widthForP` (lines 119-130). Update the file/function docstrings that say
    "widthForP-translated width" to state the store now owns the `p→width` translation.
- **Test files to update** (not counted against the 3-file limit; required to keep `mise run check`
  green) — every call site swaps its width argument for the `p` qualifier (`256`/`tiles.TileWidth` →
  `0`; a partial width `W` → `uint8(W)`):
  - `internal/store/tiles_test.go`, `internal/store/fetcher_test.go`
  - `internal/follower/equivocation_test.go` (its `width` is `len(nodes)`, always ≤256; the full
    level-0 tile at width 256 → `0`, the trailing partial → `uint8(width)`), `internal/follower/fsck_test.go`
  - `internal/logclient/fsck_test.go`
  - `internal/proofserve/entries_test.go`, `internal/proofserve/handler_test.go`
  - `internal/tilesserve/handler_test.go`
  - `cmd/iscc-monitor/main_test.go`

## Reference (read before editing — exact paths)
- `/workspace/iscc-monitor/internal/store/fetcher.go:109-118` — the canonical private `widthForP`
  (the read side already owns it; the write side adopts the same authority — do NOT add a second copy).
- `/workspace/iscc-monitor/internal/store/tiles.go:31-70` — the two methods to re-parameterize and
  their INSERT/UPDATE + `tiles.IsFull(width)` + error-format args.
- `/workspace/iscc-monitor/internal/follower/ingest.go:54-130` — the two call sites and the duplicate
  `widthForP` to delete.
- `/workspace/iscc-monitor/internal/tiles/layout.go:15-58` — `IsFull(width)` and `TileWidth == 256`
  (the full sentinel that cannot fit in `uint8`, which is exactly why the public API must speak `p`).
- `/workspace/iscc-monitor/internal/tiles/coords.go` — `TileCoords`/`BundleCoords` emit `c.Partial
  uint8` (the `p` the follower already holds and should pass straight through).
- `/workspace/iscc-monitor/.claude/context/issues.md` — the `normal` issue this closes (its verify
  recipe: move full/partial width tests to the store API, round-trip full + partial via SQLiteFetcher).

## Not In Scope
- Do NOT change `fetcher.go`'s `widthForP` body, the `SQLiteFetcher` read methods, or the
  `ReadTileBlob`/`ReadEntryBundleBlob` read API — they keep their `width int` parameter (they are
  internal store reads keyed by the stored column, not the public ingest API).
- Do NOT touch `schema.sql`, the `tiles`/`entry_bundles` table shape, or the `is_full` semantics.
- Do NOT add a `p`-typed read API or otherwise widen scope to the read path; this step is the write
  surface + the follower copy only.
- Do NOT resolve the `low` `cmd/notecheck` `out io.Writer` issue (loop-skipped).
- Do NOT begin the proof-surface cache / verify-for-me arc — that waits for a later step.

## Implementation Notes
- **Lossless direction, lossy guard.** `p uint8 → width int` is total (`0 → 256`, else `int(p)`); the
  reverse is not representable (256 doesn't fit `uint8`), which is the whole reason the public API
  should be `p`. Every test site currently passing `256` or `tiles.TileWidth` for a *full* tile MUST
  become the literal `0`, and a partial width `W (1..255)` becomes `uint8(W)`. Do NOT write
  `uint8(256)` (it wraps to 0 only by luck and reads as a bug) — write `0` for full.
- **Single authority.** After this step `grep -rn "func widthForP" internal/` returns exactly one hit
  (`internal/store/fetcher.go`). `RecordTile`/`RecordEntryBundle` call it; the follower passes
  `c.Partial` with no translation. This is the KISS / single-source-of-truth fix the issue demands.
- **Keep error messages honest.** The store error format strings already print `width %d` — keep them
  printing the *translated* `width` (compute `width := widthForP(p)` first, format with it), so a
  full-tile error still reads `width 256`, not `p 0`.
- **Correctness rule (learnings.md, "Partial-tile discipline" / "SQLiteFetcher").** `is_full` is set
  via `tiles.IsFull(width)` (true only at width 256); a full coord (`Partial == 0`) must land at
  width 256 so `SQLiteFetcher.ReadTile` (which looks a full tile up at width 256) can read it back.
  The whole point of this unification is to make that invariant impossible to break in two places.
  `TestRecordTilePartialOverwrite` (width-100 partial → `p=100`) and the `TestRecordTile*` /
  `TestRecordEntryBundle*` round-trips are the regression guards — they must still pass after the rename.
- **Oracle/conformance gate is N/A** for this slice (plain CRUD + the existing pure `p→width`
  arithmetic; no signature / RFC-6962 / Merkle / did:web / fsck path is changed) — same posture the
  reviewer recorded for the original tile-writer slice. Still, re-run `internal/follower`,
  `internal/logclient`, `internal/proofserve`, `internal/tilesserve` **uncached**, because the
  full/partial mirror round-trips through these (fsck root-rebuild and equivocation proof read the
  mirror back via `SQLiteFetcher`).
- **Store stays a leaf.** No new import in either production file (`uint8` is builtin; `widthForP` is
  same-package). `go list -deps ./internal/store` must show no `net/http` / `internal/logclient` /
  `internal/follower` edge.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass) and
  `gofmt -l .` is empty.
- `grep -rn "func widthForP" internal/` returns exactly **one** line (`internal/store/fetcher.go`) —
  the follower's copy is gone.
- `grep -n "widthForP" internal/follower/ingest.go` returns nothing (the two call sites pass
  `c.Partial` directly).
- `go test -count=1 -run 'TestRecordTile|TestRecordEntryBundle|TestFetcher' ./internal/store` passes
  uncached (full + partial round-trip through `RecordTile`/`RecordEntryBundle` → `SQLiteFetcher`).
- `go test -count=1 ./internal/follower ./internal/logclient ./internal/proofserve ./internal/tilesserve`
  passes uncached (the full/partial mirror round-trips, fsck root-rebuild, and equivocation
  consistency-proof all still read the mirror back correctly).
- `go list -deps ./internal/store` shows no `net/http`, `internal/logclient`, or `internal/follower`
  edge (store stays a leaf); `git diff --stat -- internal/store/schema.sql go.mod go.sum` is empty.

## Done When
`RecordTile`/`RecordEntryBundle` take `p uint8`, the only `widthForP` lives in `internal/store`, the
follower's copy is deleted, and all Verification criteria pass with `mise run check` green.
