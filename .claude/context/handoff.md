## 2026-06-21 — Live tile/bundle ingestion writer in PollHub

**Done:** Added `ingestTiles` (`internal/follower/ingest.go`) — the first production caller of
`tiles.TileCoords`/`BundleCoords`, `logclient.FetchTile`/`FetchEntryBundle`, and
`store.RecordTile`/`RecordEntryBundle`. On a verified, growing checkpoint `PollHub` now walks both
coordinate enumerations, fetches each named tile/bundle over the injected Fetcher, and mirrors the raw
BLOBs into the store at the `widthForP`-translated width. This un-dormants the equivocation branch on
the live path (it previously always hit the missing-tile skip) and feeds `SQLiteFetcher`.

**Files changed:**
- `internal/follower/ingest.go` (new): `ingestTiles` + `ingestHashTiles`/`ingestEntryBundles` helpers +
  the re-derived unexported `widthForP(p uint8) int` (`p==0 → tiles.TileWidth`, else `int(p)`).
- `internal/follower/follower.go`: call `ingestTiles(ctx, st, fetcher, hubID, baseURL, info.TreeSize,
  observedAt)` on the verified, non-violation path — AFTER `cacheHubKey`, BEFORE the final
  `recordVerdict`/`return`; error wrapped `follower.PollHub: hub %d: ingest tiles: %w`. Updated the
  package doc to mention tile mirroring. Not on the freeze path, not on non-verified verdicts.
- `internal/follower/ingest_test.go` (new): `TestIngestTilesWidthMapping` (table-driven coord→width over
  tree 300), `TestWidthForP`, `TestPollHubMirrorsTiles` (PollHub end-to-end vs sb0 fixture, size 10183).

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all pass, 11 pkgs
`ok`); `gofmt -l .` empty.
- [x] `go test -run 'TestIngest|TestPollHub' -count=1 ./internal/follower` → ok.
- [x] `git diff --quiet HEAD -- go.mod go.sum` → exit 0 (no dependency change).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` → exit 0 (trust-root WASM leaf unaffected).
- [x] Coord unit test: full tile (`Partial==0`) stored/read at width 256; 44-leaf partial at width 44;
  also asserts the full tile is NOT readable at width 0.
- [x] After a verified `PollHub` vs sb0: every enumerated `TileCoords`/`BundleCoords` coord returns
  `found==true` with the fetched bytes; the full level-0 tile round-trips through
  `SQLiteFetcher.ReadTile(0,0,p0)` at width 256.
- [x] follower non-test imports = `{context, fmt, logclient, metrics, store, tiles, time, log/slog}`;
  store closure has no `net/http` (still a leaf).

**Next:** Wire `RunFsck` over `SQLiteFetcher` (the M2 `fsck` root-rebuild) — now that PollHub mirrors
real tiles, `fsck.New(...).Check(ctx)` can rebuild the root from the local mirror and cross-check it
against the signed checkpoint root. That slice re-arms the trust-root oracle gate (RFC-6962
root-rebuild crypto) and wants byte-accurate live tile/entry-bundle fixtures captured into
`testdata/live/` (the inclusion cross-check vs the hub's `IsccLogInclusionProof` is the sibling slice
that needs the same fixtures).

**Notes:**
- **Oracle gate N/A for this slice** (per `next.md`): transport + CRUD only, no
  signature/RFC-6962/Merkle/did:web/fsck path introduced — the equivocation branch it un-dormants is
  already golden-tested and unchanged. The trust root re-arms at the `fsck`-over-`SQLiteFetcher` slice.
- **Synthetic tile bytes**, not byte-accurate fixtures: the writer is transport+CRUD (not crypto), so
  the unit test's recording fetcher returns `"body:"+url` and the integration `mirrorFetcher` returns
  `"tile:"+url` per tile/bundle URL — unique per coord, so a store read-back matches the exact coord
  fetched. Byte-accurate live tile fixtures arrive with the inclusion cross-check (out of scope here).
- **`mirrorFetcher` tile branch precedes the checkpoint fallback** (`strings.Contains(url, "/tile/")`).
  The existing `compositeFetcher` returns checkpoint bytes for any non-did.json URL, so the existing
  `TestPollHub{Fork,Shrink,VerifiedAdvances,...}` now also ingest tiles (recording the checkpoint bytes
  as synthetic tile BLOBs) — harmless; those tests still pass and assert only on their own outputs.
- **Idempotency / error contract (ADR-0005/0006):** every coord is re-fetched and overwritten in place
  each verified growing poll; a mid-ingest fetch/store fault is wrapped and returned (NOT a violation,
  NOT a freeze) — the checkpoint is already recorded/advanced above, so the next poll re-fetches the
  missing coords via the idempotent upsert. The "missing tile = skip equivocation" swallow in
  `checkConsistency` is a different concern and was left untouched.
- **Stale comment (out of scope, flagged for review):** `internal/follower/equivocation_test.go:4`
  says "the M2 tile-ingestion writer is not yet wired, so the test seeds them directly" — now slightly
  stale since this slice wires it. That test calls `checkConsistency`/`freeze` directly (not
  `PollHub`), so it is unaffected and still seeds its own tiles; I left it untouched per scope
  discipline. A one-line comment refresh there is the only cleanup, if desired.
- **`widthForP` is intentionally duplicated** between `internal/store/fetcher.go` (unexported) and the
  follower (re-derived one-liner) per `next.md` — the store's copy must NOT be exported and the store
  package must NOT be touched. The follower's copy is pinned by `TestWidthForP`.
