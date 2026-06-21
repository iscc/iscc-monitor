## 2026-06-21 — Transport primitives to fetch a tile / entry bundle over the Fetcher seam (`logclient.FetchTile` / `FetchEntryBundle`)

**Done:** Added the two missing pure transport primitives `FetchTile(ctx, fetcher, baseURL, level, index, p)` and `FetchEntryBundle(ctx, fetcher, baseURL, index, p)` to `internal/logclient`, ported verbatim-in-shape from `FetchCheckpoint` (origin() → `"https://"+name+"/"+path` → `fetcher.Fetch` → body verbatim), building the canonical tlog-tiles path via `tiles.TilePath`/`tiles.EntriesPath` and propagating a 404's `os.ErrNotExist` through the `%w` wrap. Purely additive — no existing non-test file changed.

**Files changed:**
- `internal/logclient/tilefetch.go` (new): `FetchTile` + `FetchEntryBundle`. Each derives `name` via the shared `origin()`, appends the path from `tiles.TilePath(level, index, p)` / `tiles.EntriesPath(index, p)` (never hand-built), fetches over the injected `Fetcher`, returns bytes verbatim, and `%w`-wraps both the `origin()` error and the Fetch error (Fetch wrap includes the constructed `url`, mirroring `fetch checkpoint %q`). Adds the `internal/tiles` import edge to the package (clean — pure `api/layout` leaf, no new closure entry).
- `internal/logclient/tilefetch_test.go` (new): offline table tests over the existing `fakeFetcher` (reused from `didresolve_test.go`, same package). Asserts exact URLs for full + partial tile/bundle and scheme-ful/bare base URLs, verbatim byte pass-through, `os.ErrNotExist` survival via `errors.Is`, and empty-base-URL origin error.
- `.claude/context/handoff.md`: this handoff.

**Verification:** Ran `mise run check` → green (build + vet + test; all 11 packages `ok`). Per-criterion:
- [x] `gofmt -l .` empty (ran `mise run fmt` first).
- [x] `go test -run 'TestFetchTile|TestFetchEntryBundle' -count=1 ./internal/logclient` passes.
- [x] `FetchTile(…, "https://sb0.iscc.id", 1, 0, 0)` → `https://sb0.iscc.id/log/tile/1/000`, bytes verbatim.
- [x] `FetchTile(…, "https://sb0.iscc.id", 0, 0, 255)` → `https://sb0.iscc.id/log/tile/0/000.p/255` (partial suffix via `TilePath`).
- [x] `FetchEntryBundle(…, "https://sb0.iscc.id", 255, 0)` → `https://sb0.iscc.id/log/tile/entries/255`; width-8 partial (`index 0, p 8`) → `https://sb0.iscc.id/log/tile/entries/000.p/8`.
- [x] Both primitives propagate a `%w`-wrapped `os.ErrNotExist` such that `errors.Is(err, os.ErrNotExist)` is true.
- [x] `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change — `tessera/api/layout` already in the closure).
- [x] `GOOS=js GOARCH=wasm go build ./internal/tiles` exits 0 (WASM-shared leaf untouched).
- [x] `internal/logclient` still imports `net/http` (1 occurrence) — package non-WASM status unchanged, as expected.

**Next:** Wire these into a `PollHub` tile-ingestion loop (its own ≤3-file slice, touching `follower.go`): on a verified growing checkpoint, walk `TileCoords(size)` + `BundleCoords(size)`, fetch each tile/bundle over `FetchTile`/`FetchEntryBundle`, and write via `store.RecordTile`/`RecordEntryBundle` — translating the path-API `Partial` (`p`) to the store's `width` via `widthForP` (full tile `p==0` → width 256), re-fetching partials every poll per ADR-0005. That writer is what finally arms the deferred `fsck` root-rebuild over `SQLiteFetcher` and the inclusion cross-check vs the hub's `IsccLogInclusionProof`, both of which want real tile/bundle fixtures in `testdata/live/`.

**Notes:**
- Oracle/conformance gate is correctly **N/A** for this slice — pure URL construction + transport, no signature/RFC-6962/Merkle/did:web/`fsck` path introduced (same as `FetchCheckpoint` and the `tiles` layout slice). The trust-root oracle (`notecheck`/`fsck`/`derive_vkey.py`) re-arms at the ingestion-writer + `fsck.Check` slice that needs captured fixtures.
- `FetchTile`/`FetchEntryBundle` are an intentional unused-until-wired export seam (like `TileCoords`/`BundleCoords`/the consistency triggers) — `go vet` clean, not dead code. First caller is the next ingestion-writer slice.
- Deliberately did **not** replicate tessera's `PartialOrFullResource` partial→full fallback (out of scope): these fetch exactly the `p` the coord enumeration names; the store's `SQLiteFetcher` already owns that fallback on the read side.
- The open `low` `cmd/notecheck` `out io.Writer` issue was left untouched (loop-skipped, out of scope).
