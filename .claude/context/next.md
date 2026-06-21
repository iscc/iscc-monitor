# Next Work Package

## Step: Serve the raw tlog-tiles mirror (checkpoint / tile / entries) over HTTP from one hub's SQLiteFetcher

## Goal
Stand up the canonical static read surface M2 requires — `GET /checkpoint`,
`GET /tile/<L>/<K>`, `GET /tile/entries/<K>` (including `.p/<W>` partials) — served verbatim
from the local mirror via the existing `store.SQLiteFetcher`, never re-hitting the hub. This is
the spec-correct foundation of "serve proofs from the local store" (iscc-log §9: there are no
proof-computing endpoints; a verifier computes proofs locally from the served tiles) and the
inbound transport the later `consistency`/`inclusion` `verify-for-me` slices build on.

## Scope
- **Create**: `internal/tilesserve/handler.go` — a new package exporting
  `Handler(f store.SQLiteFetcher) http.Handler` that routes the three canonical tlog-tiles paths
  to `f.ReadCheckpoint` / `f.ReadTile` / `f.ReadEntryBundle` and writes the raw BLOB bytes.
- **Create**: `internal/tilesserve/handler_test.go` — table-driven `httptest` test over a real
  `store.Open(tmp)` seeded with one full tile, one partial tile, one entry bundle, and one
  checkpoint via the existing store writers.
- **Modify**: (none — binary wiring is a separate slice; see Not In Scope)
- **Reference**:
  - `/workspace/iscc-monitor/internal/metricshttp/handler.go` — the `Handler(...) http.Handler`
    leaf-wrapping pattern and the mid-write `_ = w.Write(...)` swallow idiom to mirror.
  - `/workspace/iscc-monitor/internal/store/fetcher.go` — `SQLiteFetcher.ReadCheckpoint` /
    `ReadTile(ctx,l,i uint64,p uint8)` / `ReadEntryBundle(ctx,i uint64,p uint8)`; note the
    `os.ErrNotExist` wrap on a missing row and the partial→full fallback already handled inside.
  - `/workspace/iscc-monitor/internal/tiles/layout.go` — the build-side `TilePath`/`EntriesPath`
    re-exports the test can use to construct request URLs without hand-building chunked paths.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/api/layout/paths.go`
    (lines 149–208) — `ParseTileLevelIndexPartial(level, index string) (uint64,uint64,uint8,error)`
    and `ParseTileIndexPartial(index string) (uint64,uint8,error)`: the canonical parsers for the
    `x###/###` chunked index and the `.p/<W>` partial suffix. Use these — do NOT hand-roll path math.
  - `/workspace/iscc-monitor/internal/store/tiles.go` — `RecordTile`/`RecordEntryBundle` (the test's
    seed writers) and the `width` they take (256 = full, else the partial leaf count).

## Not In Scope
- Binary wiring in `cmd/iscc-monitor/main.go` (mounting a per-hub route prefix, a hub→origin
  router, a read-only connection pool). That needs a multi-hub routing design and is its own slice.
- The `consistency` / `inclusion` proof-computing or `verify-for-me` REST surface — those are
  later M2/M3 slices that consume `ConsistencyProofFromTiles` / `VerifyInclusionEvidence`; this
  slice only serves the raw static bytes a verifier (or those later handlers) reads from.
- CORS headers, `Cache-Control`, ETag, conditional GET, and any non-GET method handling beyond a
  405/404 default — defer to the M3 REST-surface slice (target.md M3: "CORS on every public GET").
- Any change to `store`, `logclient`, `tiles`, `schema.sql`, `go.mod`, or `go.sum`. If a tile-path
  parse re-export feels cleaner in `internal/tiles`, resist it — import `api/layout` directly in the
  new handler this slice (no new exported surface on the `tiles` leaf).
- Resolving `iscc_id → seq` or serving an entry bundle by record index — `EntriesPath` indexes by
  bundle, and that resolution belongs to the inclusion slice.

## Implementation Notes
- One package `tilesserve`, one exported `Handler(f store.SQLiteFetcher) http.Handler`. It is NOT a
  WASM leaf (it imports `net/http` + `store`), so there is no purity constraint here — unlike
  `internal/metrics`, the split exists only to keep routing out of `store` (store stays a leaf; the
  new package depends on store, never the reverse).
- Route on `r.URL.Path` with a small `switch`/prefix match. Canonical tlog-tiles paths (iscc-log §9,
  served under the hub's `/log` origin, but THIS handler is mounted at the hub root so it sees the
  suffix): `checkpoint`, `tile/<L>/<index...>`, `tile/entries/<index...>`. Trim a single leading
  `/`. Match `tile/entries/` BEFORE `tile/` (entries is a sub-prefix of tile and a bare `tile/<L>`
  parse of an `entries/...` path must not win).
- For `tile/<L>/<index...>`: split off the level segment, pass `(level, rest)` to
  `layout.ParseTileLevelIndexPartial`, then `f.ReadTile(ctx, level, index, width)`. tessera's
  returned `width` IS the fetcher's `p` (0 = full, else partial leaf count) — they share the exact
  convention (`store/fetcher.go` `widthForP`), so pass it straight through; do not re-map it.
- For `tile/entries/<index...>`: pass the index remainder to `layout.ParseTileIndexPartial`, then
  `f.ReadEntryBundle(ctx, index, width)`.
- For `checkpoint`: `f.ReadCheckpoint(ctx)` with no path args.
- Error mapping (the load-bearing contract): a parse failure → `400`; `errors.Is(err,
  os.ErrNotExist)` (the `SQLiteFetcher` missing-row sentinel — Correctness: the fetcher wraps
  `os.ErrNotExist`) → `404`; any other read error → `500`; method != GET → `405`; unmatched path →
  `404`. On success write the raw bytes verbatim (the BLOB IS the canonical tlog-tiles body — these
  are static files), `Content-Type: application/octet-stream`. Mirror metricshttp's deliberate
  mid-write `_ = w.Write(...)` swallow (the 200 is already on the wire), documented inline — NOT a
  gate dodge.
- Use `r.Context()` for the fetcher calls. Keep the package docstring evergreen (purpose first line).
- Test: `store.Open(t.TempDir()+"/x.db")`, `UpsertHub`, seed via `RecordTile`(full width 256 at
  `(0,0)` and a partial e.g. width 44 at `(0,1)`), `RecordEntryBundle`(one bundle), and
  `RecordCheckpoint`(raw bytes). Build request URLs with `tiles.TilePath`/`tiles.EntriesPath` so the
  paths are tessera-canonical, not author-asserted. Assert: 200 + exact seeded bytes for each
  served path (full, partial, entries, checkpoint), 404 for a never-mirrored tile, 400 for a
  malformed index, 405 for POST. Drive through `httptest.NewServer` or call the handler with
  `httptest.NewRecorder()` — either is fine; assert on observable HTTP outputs (status + body),
  never on handler internals (PRD seam-testing rule).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestHandler -count=1 ./internal/tilesserve` passes (all sub-cases).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).
- `go list -deps ./internal/store | grep -E 'internal/tilesserve|net/http'` is empty (store stays a
  leaf; the new package depends on store, never the reverse).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (WASM purity invariant untouched).
- A `GET` for a full tile returns HTTP 200 with bytes byte-equal to the seeded `RecordTile` BLOB; a
  `GET` for a never-mirrored tile index returns HTTP 404; a malformed tile index returns HTTP 400;
  a `POST` returns HTTP 405.

## Done When
`internal/tilesserve.Handler` serves the three canonical tlog-tiles paths (checkpoint / tile /
entries, including `.p/<W>` partials) verbatim from one hub's `SQLiteFetcher` with correct
400/404/405/500 status mapping, and all Verification criteria pass.
