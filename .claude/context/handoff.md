## 2026-06-21 — Serve GET /entries (computed record bytes) from the local mirror

**Done:** Added the `/entries?index=<seq>` route that extracts a single accepted leaf's raw record
bytes from the hub's mirrored entry bundles and serves them verbatim as `application/octet-stream`,
never re-hitting the hub. This completes M2's three computed-proof surfaces (inclusion, consistency,
entries). Core is one pure decode-and-index function `logclient.RecordBytesFromBundle` plus an
`ErrLeafOutOfBundle` sentinel, wired into `proofserve.Handler` and the per-hub mux.

**Files changed:**
- `internal/logclient/entries.go` (new): `RecordBytesFromBundle(bundle, offset)` —
  `api.EntryBundle{}.UnmarshalText` then `eb.Entries[offset]` verbatim (raw JCS-canonical envelope,
  NOT re-hashed); `ErrLeafOutOfBundle` sentinel for an out-of-range offset; truncated frame `%w`-wrapped.
  Imports exactly `errors`+`fmt`+`tessera/api`, file-level WASM-pure (sibling of `leafhasher.go`).
- `internal/proofserve/handler.go`: added `case "/entries"` arm + `serveEntries` (mirrors
  `serveInclusion`'s accepted-tree guards, reuses `parseUint`), `writeRecord` helper, `octetStreamType`
  const, and the `tiles` import; package + Handler doc updated.
- `cmd/iscc-monitor/main.go`: one `mux.Handle("/entries", proofs)` line in `hubHandler` + doc note.
- Tests: `internal/logclient/entries_test.go`, `internal/proofserve/entries_test.go`,
  `cmd/iscc-monitor/main_test.go` (new `TestMirrorEntriesRoute` + `frameEntryBundle` helper).

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all pass, `gofmt -l .`
empty). Full suite re-run uncached `go test -count=1 ./...` → all 14 packages `ok`. Per-criterion:
- [x] `go test -run TestRecordBytesFromBundle ./internal/logclient` PASS — in-range byte-equality,
  `ErrLeafOutOfBundle` via `errors.Is` (offset == len and beyond, incl. empty bundle), `%w`-wrapped
  truncated frame; uses the `leafhasher_test.go` `encodeBundle` manual-uint16 third path.
- [x] `go test -run 'TestServeEntries|TestEntries' ./internal/proofserve` PASS — 200 + exact record
  bytes across the 256-leaf bundle boundary `{0,5,255,256,260,299}`, missing index → 400, non-numeric
  index → 400, `seq >= LastSize` → 404, no accepted checkpoint → 404, bundle not mirrored → 404,
  non-GET → 405, Content-Type `application/octet-stream`.
- [x] `go test -run TestMirror ./cmd/iscc-monitor` PASS — new `TestMirrorEntriesRoute` routes
  `/<origin>/log/entries?index=2` through the shared mux to the record bytes; existing inclusion +
  mirror + healthz sub-tests intact.
- [x] `entries.go` imports exactly `errors`+`fmt`+`tessera/api`; `GOOS=js GOARCH=wasm go build
  ./internal/logclient` exits 0.
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no dep/schema change;
  `tessera/api` + `internal/tiles` already in the closure).

**Next:** The M3 cross-cutting HTTP slice deferred across all mirror + proof surfaces at once: CORS,
caching, and conditional-GET (ETag/If-None-Match) headers on `/inclusion`, `/consistency`, `/entries`,
and the static mirror — now that all three computed surfaces exist, one slice can add them uniformly.
Alternatively the range-fetch additive slice (multiple leaves in one `/entries` response) or the
`TestPollHubFork` re-detection cleanup (still an open `normal` issue).

**Notes:**
- **Deviation from `next.md`'s implementation note (flagged, not a design change):** the note said
  "Request the full bundle (`p == 0`)". Passing `p == 0` unconditionally 404s the FINAL partial bundle
  of any non-multiple-of-256 tree (and every tree < 256 leaves) — the bundle is mirrored only at its
  partial width, and the `SQLiteFetcher` partial→full fallback only fires for `p > 0`. The two seeded
  failures (`TestServeEntries` seq 256 → 44-leaf partial; `TestMirrorEntriesRoute` → 5-leaf bundle)
  proved this. Fix: compute `p := tiles.PartialTileSize(0, bundleIndex, size)` (entry bundles are
  level-0) and pass it to `ReadEntryBundle` — the exact pattern the proof builder uses for partial
  tiles, so a partial later promoted to full still resolves via the fallback. No new dependency
  (`internal/tiles` was already imported). This is a correctness fix within the route, not an API or
  scope change.
- **Oracle / conformance gate correctly N/A for this slice** (matches the seeded pattern): the
  record-bytes extractor is a pure decode + index — it returns the leaf's bytes verbatim, never
  re-hashing, verifying, or interpreting (no signature / RFC-6962 / Merkle / did:web / fsck path is
  introduced; `serveInclusion`/`serveConsistency` already own the proof crypto). The test asserts
  byte-equality against the records the bundle was framed from via an independent encode path
  (`frameBundle`/`frameEntryBundle`, distinct from the `UnmarshalText` decode under test), so a
  green-but-wrong extractor returning the wrong leaf would fail. No gate was weakened.
- The `_, _ = w.Write(record)` drop in `writeRecord` is the documented post-status write-drop
  convention (matching `tilesserve.writeBlob` / `metricshttp`): the 200 is sent on the first byte and a
  mid-write fault on an opaque BLOB cannot un-send it. Not a swallowed-error gate dodge.
- `/entries` (computed single-leaf record_bytes) is distinct from `/tile/entries/<bundleindex>` (the
  raw whole-bundle BLOB served under `"/"` by `tilesserve`) — different route, different shape, not a
  duplicate (256 framed leaves vs exactly one leaf's bytes). `tilesserve` untouched as scoped.
- Scope is exactly 3 production `.go` files + 3 test files; nothing from `## Not In Scope` touched
  (no `tilesserve`, no CORS/caching/ETag header code, no range fetch, no proof-bundle JSON assembly,
  no open-issue cleanup). `go.mod`/`go.sum`/`schema.sql` byte-unchanged.
- Open `normal` issues remain for `define-next` to weigh (`TestPollHubFork` cleanup, frozen-advance,
  `AcceptCheckpoint` context reuse, tile-writer `p` vocab, `AdvanceAccepted`, `CheckConsistency`
  collapse). M2's Verify bar is now complete.
