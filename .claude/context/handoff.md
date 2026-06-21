## 2026-06-21 — Serve the raw tlog-tiles mirror (checkpoint / tile / entries) over HTTP from one hub's SQLiteFetcher

**Done:** Added `internal/tilesserve`, a new package exporting
`Handler(f store.SQLiteFetcher) http.Handler` that routes the three canonical tlog-tiles paths
(`GET /checkpoint`, `GET /tile/<L>/<index...>`, `GET /tile/entries/<index...>`, including `.p/<W>`
partials) to `f.ReadCheckpoint` / `f.ReadTile` / `f.ReadEntryBundle` and writes the raw BLOB bytes
verbatim from the local mirror — never re-hitting the hub. Status mapping is 400 (parse) / 404
(missing row or unmatched path) / 405 (non-GET) / 500 (other read error).

**Files changed:**
- `internal/tilesserve/handler.go` (new): `Handler` + four unexported helpers. Routes on
  `strings.TrimPrefix(r.URL.Path, "/")`; matches `tile/entries/` BEFORE `tile/` (entries is a
  sub-prefix). Tile path uses `strings.Cut(rest, "/")` to split the single level segment from the
  chunked index remainder, then `layout.ParseTileLevelIndexPartial(level, index)`; entries uses
  `layout.ParseTileIndexPartial`. tessera's returned `width` is passed straight through as the
  fetcher's `p` (shared 0==full convention). `Content-Type: application/octet-stream`; mirrors
  metricshttp's deliberate `_, _ = w.Write(...)` mid-write swallow (documented inline).
- `internal/tilesserve/handler_test.go` (new): table/subtest `httptest.NewServer` test over a real
  `store.Open(t.TempDir())` seeded with a full tile (0,0,256), a partial (0,1,44), a full bundle
  (0,0,256), and a checkpoint. Asserts 200 + exact seeded bytes for all four served paths, 404 for a
  never-mirrored tile and an unmatched path, 400 for malformed tile + entries indices, 405 for POST.
  Request URLs are built from `tiles.TilePath`/`tiles.EntriesPath` (tessera-canonical, not asserted).

**Verification:** `mise run check` → GREEN (all 12 packages `ok`; build + vet + test). Per-criterion:
- [x] `mise run check` green (`go build`/`go vet`/`go test`/`gofmt -l .` empty).
- [x] `go test -run TestHandler -count=1 ./internal/tilesserve` PASS (all sub-cases: full, partial,
  entries, checkpoint, 404 missing, 400 malformed tile, 400 malformed entries, 404 unmatched, 405 POST).
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exit 0 (no schema/dep change).
- [x] `go list -deps ./internal/store | grep -E 'internal/tilesserve|net/http'` empty (store stays a
  leaf; `tilesserve → store`, never the reverse — confirmed the inverse `go list -deps ./internal/tilesserve`
  shows `internal/store` + `net/http`).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exit 0 (WASM purity untouched).
- [x] Full tile GET → 200 byte-equal to seeded BLOB; never-mirrored tile → 404; malformed index → 400;
  POST → 405.

**Next:** Wire `tilesserve.Handler` into `cmd/iscc-monitor/main.go` — this is the deferred binary-wiring
slice (`next.md` Not In Scope): mount a per-hub route prefix, a hub→origin router (resolve the request's
hub from a path prefix or host to its `HubID`), and a read-only connection pool. That needs the multi-hub
routing design called out in `next.md`. After that (or in parallel), the `consistency`/`inclusion`
`verify-for-me` REST surface (M2/M3) consumes `ConsistencyProofFromTiles` / `VerifyInclusionEvidence`
and builds on this inbound transport. The 6 open follower issues remain orthogonal.

**Notes:**
- **Oracle/conformance gate correctly N/A for this slice.** This handler is a pure static-byte read
  transport over the existing `SQLiteFetcher` (which itself was N/A per prior review) — no
  signature/RFC-6962/Merkle/did:web/`fsck`-rebuild path is introduced. The bytes served are opaque BLOBs;
  the verifier that reads them computes proofs locally (a later slice). `notecheck`/`derive_vkey.py`/`fsck`
  re-arm at the proof-serving / `verify-for-me` slice. The test asserts only observable HTTP outputs
  (status + body), per the PRD seam rule.
- **Path routing decision:** the level is a single segment but the chunked index can itself contain `/`
  and a `.p/<W>` suffix, so `strings.Cut(rest, "/")` (split on the FIRST slash only) is the correct split
  — `level=0`, `index=001.p/44` for `tile/0/001.p/44`. Verified the tessera path forms and parse
  round-trips before writing (`tile/0/000`, `tile/0/001.p/44`, `tile/entries/000`, `tile/entries/001.p/44`).
- **Intentional unwired export seam** (like the prior M2 seams): no production caller yet — the binary
  wiring is the deferred next slice. `go vet` clean, not dead code.
- **Partial→full fallback is the fetcher's job, not the handler's:** a request for `tile/0/001.p/44`
  whose partial was promoted to full transparently falls back inside `SQLiteFetcher.ReadTile` (already
  tested in `store/fetcher_test.go`); the handler passes `width` through untouched.
- Scope clean: exactly the 2 new files in `internal/tilesserve/` + this handoff; no change to `store`,
  `logclient`, `tiles`, `schema.sql`, `go.mod`, `go.sum`, or the binary.
