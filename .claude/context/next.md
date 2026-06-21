# Next Work Package

## Step: Cache-Control on the tlog-tiles mirror (immutable full tiles/bundles vs. revalidating partials + checkpoint)

## Goal
Make the raw tlog-tiles mirror cacheable correctly: serve content-addressed **full**
tiles/entry-bundles with a long-lived immutable `Cache-Control` and serve **partials** and the
size-varying **checkpoint** with a revalidating (`no-cache`) policy. This is the next M3 cross-cutting
HTTP slice (review handoff `**Next:**`) and it respects the partial-tile discipline (ADR-0005): only a
full resource is immutable, partials are overwritten every poll and must never be cached as immutable.

## Scope
- **Modify**: `internal/tilesserve/handler.go` (the only production file — set `Cache-Control` per
  route, using the cacheability signal already in hand at each handler).
- **Reference**:
  - `/workspace/iscc-monitor/internal/tilesserve/handler.go` — current routes; `serveCheckpoint` /
    `serveTile` / `serveEntries` / `writeBlob`; the parsed `width` (tessera path-API: `0 == full`,
    `>0 == partial` leaf count) is available in `serveTile`/`serveEntries` before `writeBlob`.
  - `/workspace/iscc-monitor/internal/store/fetcher.go` — confirms the `p`/`width` convention
    (`p == 0 → full width 256`) and the partial→full promotion semantics this policy must honor.
  - `/workspace/iscc-monitor/internal/tiles/layout.go` — `IsFull(width int) bool == width == TileWidth`
    and `TileWidth == 256`. NOTE the two vocabularies: in the tessera *path-API* a full resource is
    `width 0`, while the *store* uses `width 256` for full. This slice works in the path-API vocabulary
    (the parsed `width` from `layout.ParseTile*`), so "full" here is `width == 0`. Do not confuse them.
  - `/workspace/iscc-monitor/internal/corsmw/corsmw.go` — the existing single CORS leaf, to confirm the
    `buildMux` wrap only sets `Access-Control-*` (never `Cache-Control`), so per-route `Cache-Control`
    here neither duplicates nor conflicts with it.

## Not In Scope
- **Conditional GET** — no `ETag` / `If-None-Match` / `Last-Modified` / `If-Modified-Since` and no 304
  responses this slice. That is a deliberately separate follow-up slice (the review handoff lists both;
  keep this step to `Cache-Control` only so it stays one small, single-file change).
- The computed-proof surfaces (`proofserve`: `/inclusion`, `/consistency`, `/entries`) — their
  cacheability is size-dependent and tied to `LastSize`; leave them uncached here (a later slice).
- `/metrics`, `/healthz`, and any change to `corsmw` or `cmd/iscc-monitor/main.go` wiring.
- Draining any `normal` issue (`TestPollHubFork` cleanup, frozen-advance, `AcceptCheckpoint` context
  reuse, tile-writer `p` vocab, `AdvanceAccepted`, `CheckConsistency` collapse) — weighed and deferred
  to finish the coherent M3 HTTP-plumbing arc first.

## Implementation Notes
- **Cacheability is known at the route — no new store lookup needed.** The checkpoint is always
  size-varying; tiles/bundles carry the parsed partial `width` already (`layout.ParseTileLevelIndexPartial`
  returns `width`; `layout.ParseTileIndexPartial` returns `width`). In the tessera path-API vocabulary a
  **full** resource parses to `width == 0` and a **partial** parses to its actual leaf count (`> 0`) —
  the same convention `store.SQLiteFetcher` documents (`p == 0 → full`). So the predicate is simply
  `full := width == 0`.
- **Plumb a cacheability bool into `writeBlob`**, e.g. `writeBlob(w, data, immutable bool)`:
  - `serveCheckpoint` → `immutable == false` (the checkpoint is overwritten as the tree grows).
  - `serveTile` / `serveEntries` → `immutable := (width == 0)` (full = immutable; partial = revalidate).
  Keep the change minimal: add the one parameter and set the header before the existing `w.Write`.
- **Header values** (define fixed, documented `const`s in the file):
  - Immutable: `Cache-Control: public, max-age=31536000, immutable` (one year; content-addressed full
    tiles/bundles never change — the RFC-8246 `immutable` directive lets browsers skip revalidation).
  - Revalidate: `Cache-Control: no-cache` (cache may store but must revalidate before reuse; correct for
    partials that are overwritten in place and for the size-varying checkpoint). Do **not** use
    `no-store` — we *want* the response cacheable-with-revalidation, just not reusable-without-checking.
- **Header-write order**: set `Content-Type` and `Cache-Control` BEFORE the first `w.Write` (the 200 is
  sent on first write and freezes the header map) — mirror the existing `writeBlob` comment about the
  status being sent on first write. Error paths (`http.Error` 400/404/405/500) need no `Cache-Control`.
- **Correctness rule (learnings.md → "Partial-tile discipline", ADR-0005):** "Mark a tile/bundle BLOB
  `is_full` (immutable) only at `width == 256`; re-fetch partials every poll and overwrite. Never
  promote a partial." The HTTP cache policy must mirror that exactly: a partial response must NOT carry
  the `immutable` directive, or a client would cache a soon-overwritten partial forever. A wrong
  predicate here (marking partials immutable) is the load-bearing bug this step guards against — pin it
  with an explicit partial-tile assertion.
- **Oracle/conformance gate is N/A** for this slice: pure HTTP header wiring on opaque BLOBs, no
  signature / RFC-6962 / Merkle / did:web / fsck path touched. Do not add `//nolint`, `t.Skip`, or weaken
  any check. No new import (`net/http` + `strings` + `tessera/api/layout` are already in the closure), so
  `go.mod` / `go.sum` / `schema.sql` must stay byte-unchanged.
- **Tests** (`internal/tilesserve/handler_test.go` already seeds a full tile, a 44-leaf partial tile, a
  full bundle, and a checkpoint over `httptest`): extend the existing `200` subtests to read
  `resp.Header.Get("Cache-Control")` and assert the per-route policy. Reuse `newServer`; assert on HTTP
  output only (per PRD testing decisions), never on handler internals. Keep the body-equality assertions
  intact. Note `get()` currently discards the response header — either widen it to return the header or
  add a small sibling that does; do not assert headers on the existing body-only path by guessing.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .`
  empty).
- `go test -count=1 ./internal/tilesserve` passes.
- Full tile `GET tile/0/000` response carries `Cache-Control: public, max-age=31536000, immutable`
  (asserted in the test).
- Full entry bundle `GET tile/entries/000` response carries
  `Cache-Control: public, max-age=31536000, immutable`.
- Partial tile `GET tile/0/001.p/44` response carries `Cache-Control: no-cache` (NOT `immutable`) — the
  load-bearing partial-tile assertion.
- Checkpoint `GET checkpoint` response carries `Cache-Control: no-cache`.
- `git diff --quiet -- go.mod go.sum internal/store/schema.sql` exits 0 (no dependency/schema change).
- `grep -rn "immutable" internal/tilesserve/handler.go` shows the directive in exactly one place (the
  immutable branch), and `grep -rn "no-store" internal/tilesserve/handler.go` is empty.

## Done When
`mise run check` is green and the four route-level `Cache-Control` assertions (full tile + full bundle =
immutable; partial tile + checkpoint = `no-cache`) pass in `go test ./internal/tilesserve`, with
`go.mod` / `go.sum` / `schema.sql` byte-unchanged.
