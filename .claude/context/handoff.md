## 2026-06-21 — Cache-Control on the tlog-tiles mirror (immutable full tiles/bundles vs. revalidating partials + checkpoint)

**Done:** Plumbed a cacheability bool into `writeBlob` and set a per-route `Cache-Control` on every
tlog-tiles mirror 200: content-addressed FULL tiles/entry-bundles (path-API `width == 0`) get
`public, max-age=31536000, immutable`; the size-varying checkpoint and any PARTIAL tile/bundle
(`width > 0`) get `no-cache`. No conditional GET (deliberately deferred); single production file.

**Files changed:**
- `internal/tilesserve/handler.go`: added `cacheImmutable` / `cacheRevalidate` consts; `writeBlob` now
  takes `immutable bool` and sets `Cache-Control` (before the first write, alongside `Content-Type`);
  `serveCheckpoint` passes `false`, `serveTile`/`serveEntries` pass `width == 0`. Updated the package +
  `Handler` doc comments to describe the cache policy and narrow the "out of scope" note to conditional
  GET only (CORS is set by the `corsmw` wrap; caching is now in scope here).
- `internal/tilesserve/handler_test.go`: added `getWithHeader` sibling (returns status + header + body;
  `get` now delegates to it, all existing body-only call sites unchanged); extended the four `200`
  subtests to assert the per-route `Cache-Control` while keeping body-equality intact. Partial-tile
  subtest pins the load-bearing assertion (`no-cache`, NOT `immutable`).

**Verification:** `mise run check` → green (build + vet + test; all 15 packages `ok`). Per criterion:
- [x] `go test -count=1 ./internal/tilesserve` → ok; all 9 subtests PASS.
- [x] Full tile `GET tile/0/000` → `Cache-Control: public, max-age=31536000, immutable` (asserted).
- [x] Full bundle `GET tile/entries/000` → `public, max-age=31536000, immutable` (asserted).
- [x] Partial tile `GET tile/0/001.p/44` → `no-cache`, not `immutable` (load-bearing assertion).
- [x] Checkpoint `GET checkpoint` → `no-cache` (asserted).
- [x] `gofmt -l .` empty.
- [x] `git diff --quiet -- go.mod go.sum internal/store/schema.sql` exits 0 (no dep/schema change).
- [x] The `max-age=31536000, immutable` directive string appears in exactly one place (the immutable
  branch, line 37); `grep no-store internal/tilesserve/handler.go` empty.

**Next:** The conditional-GET follow-up slice the review handoff and `next.md` both flag — `ETag` /
`If-None-Match` (and/or `Last-Modified` / `If-Modified-Since`) with `304` responses on the same mirror
routes. Content-addressed full tiles/bundles already have a strong-ETag source (the BLOB is keyed by
its hashes); the checkpoint/partials need a weak/derived validator. Alternatively, start draining a
`normal` issue (split-view / frozen-advance ADR-0006 items are highest-value) or begin the proof-bundle
JSON + verify-for-me arc. Note: the size-dependent `proofserve` surfaces (`/inclusion` /`/consistency` /
`/entries`) still have no cache policy — their cacheability is tied to `LastSize` and was explicitly
left out of this slice.

**Notes:**
- The `immutable bool` parameter name and the doc-comment uses of the word "immutable" naturally show up
  in `grep -rn immutable handler.go`; the `next.md` criterion is about the `immutable` *directive*
  (the header value), which is genuinely in one place — confirmed with `grep 'max-age=31536000,
  immutable'` (single hit).
- "full = `width == 0`" is the tessera path-API vocabulary (the parsed `width` from
  `layout.ParseTile*`), NOT the store column vocabulary where full = 256. Both `serveTile`/`serveEntries`
  work on the parsed path-API `width`, so `width == 0` is the correct immutable predicate and mirrors
  `store.SQLiteFetcher`'s documented `p == 0 → full` convention.
- Oracle/conformance gate is correctly N/A for this slice: pure HTTP header wiring on opaque BLOBs — no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path touched. No new import (`net/http` +
  `strings` + `tessera/api/layout` already in the closure), so go.mod/go.sum/schema.sql are byte-unchanged.
  No `//nolint` / `t.Skip` / swallowed error / weakened gate.
- Header-order is safe: `Content-Type` and `Cache-Control` are both set before the first `w.Write` (the
  200 freezes the header map on first write). Error paths (`http.Error` 400/404/405/500) intentionally
  carry no `Cache-Control`. The `corsmw` wrap only sets `Access-Control-*`, so no header conflict.
