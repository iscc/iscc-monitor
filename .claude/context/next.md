# Next Work Package

## Step: Serve GET /entries (computed record bytes) from the local mirror

## Goal
Complete M2's third and final computed-proof surface: serve a single log leaf's raw
record bytes (`?index=<seq>`) extracted from the hub's mirrored entry bundles, never
re-hitting the hub. This is the `record_bytes` member every self-contained proof bundle
needs (PRD §"API contracts" line 194, user stories 2 & 4) and the last item on M2's
Verify bar per `state.md` (inclusion + consistency are served; `entries` is unstarted).

## Scope
- **Create**: `internal/logclient/entries.go` — one pure function
  `RecordBytesFromBundle(bundle []byte, offset uint64) ([]byte, error)` that decodes a
  tlog-tiles entry bundle and returns the record bytes of the leaf at `offset` within it,
  plus a `var ErrLeafOutOfBundle = errors.New(...)` sentinel for an out-of-range offset.
- **Modify**: `internal/proofserve/handler.go` — add a `serveEntries` route and a
  `case "/entries"` arm in `Handler`'s switch (mirrors the `serveInclusion` flow).
- **Modify**: `cmd/iscc-monitor/main.go` — mount the `/entries` exact path on the per-hub
  mux in `hubHandler` (one `mux.Handle("/entries", proofs)` line next to `/inclusion`).
- **Reference**:
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/log_tree.py` (`get_entry_bundle`,
    lines 137-157) — the hub's bundle-by-(index,width) shape; ground the bundle-index math.
  - `/workspace/iscc-monitor/internal/logclient/leafhasher.go` and
    `/workspace/iscc-monitor/internal/logclient/projection.go` — the
    `api.EntryBundle{}.UnmarshalText` → `eb.Entries[i]` decode pattern to port verbatim,
    incl. the file-level WASM-purity comment to copy.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` (`serveInclusion`, `parseUint`,
    `writeEvidence`) — the existing route to mirror for status mapping and the `parseUint` reuse.
  - `/workspace/iscc-monitor/internal/store/fetcher.go` (`SQLiteFetcher.ReadEntryBundle`) —
    the mirror read with its partial→full fallback and `os.ErrNotExist` sentinel contract.
  - `/workspace/iscc-monitor/internal/tiles/layout.go` (`TileWidth = 256`, the entry-bundle width).

## Not In Scope
- **Do NOT touch `tilesserve`.** The raw `/tile/entries/<bundleindex>` *whole-bundle* BLOB
  stays as is; this new `/entries?index=<seq>` is the *computed single-leaf* `record_bytes`
  surface (a distinct route + distinct shape). The two are not duplicates — the raw BLOB is
  256 framed leaves a client must still parse; this returns exactly one leaf's bytes.
- CORS / caching / conditional-GET (ETag) headers — still deferred; a later cross-cutting
  slice owns all three across the mirror + proof surfaces at once.
- Range fetch (multiple leaves in one response) — PRD mentions "entry/range fetch" but the
  single-leaf `record_bytes` is what the proof bundle needs; range is a later additive slice.
- Assembling the full proof-bundle JSON (`{checkpoint, inclusionProof, record_bytes,
  hub_didweb_key, …}`) — that is M3 proof-bundle assembly / verify-for-me, not M2.
- The open `normal` issues (`TestPollHubFork` cleanup, frozen-advance, `AcceptCheckpoint`
  context reuse, tile-writer `p` vocab, `AdvanceAccepted`, `CheckConsistency` collapse) —
  weighed and deferred to finish M2's coherent proof surface first.

## Implementation Notes
- **`RecordBytesFromBundle` is the `leafhasher.go`/`projection.go` sibling — same decode,
  different projection.** Port the decode verbatim: `eb := &api.EntryBundle{}` then
  `eb.UnmarshalText(bundle)` (`%w`-wrap the error with a `logclient.RecordBytesFromBundle:`
  prefix). Then bound-check `offset >= uint64(len(eb.Entries))` → return `ErrLeafOutOfBundle`
  (a package sentinel, mirroring how `ErrInclusionMismatch` is a `logclient` sentinel the
  handler maps via `errors.Is`). On success return `eb.Entries[offset]` — the RAW record
  bytes (the JCS-canonical envelope), NOT re-hashed; this is the `record_bytes`, not a leaf
  hash. Keep the file WASM-pure: imports exactly `errors` + `fmt` + `tessera/api` (no
  `net`/`os`/`database/sql`), matching `leafhasher.go`'s purity comment. `go.mod`/`go.sum`
  must stay byte-identical (`tessera/api` is already in the closure).
- **`serveEntries` mirrors `serveInclusion`'s flow.** Steps: (1) `parseUint(query "index")`
  → 400 on missing/non-numeric (REUSE the existing `parseUint`; do NOT hand-roll). (2) read
  `fs.LastSize` via `st.FollowState` — a `LastSize == 0` hub → 404 "no accepted checkpoint";
  a `seq >= LastSize` → 404 "leaf not covered by accepted checkpoint" (same guard shape as
  `serveInclusion`'s `leafIndex >= size`: the leaf is not yet in the monitor's accepted tree).
  (3) compute `bundleIndex := seq / tiles.TileWidth` and `offset := seq % tiles.TileWidth`.
  (4) `f.ReadEntryBundle(ctx, bundleIndex, 0)` (full-bundle request; `SQLiteFetcher` already
  does the partial→full fallback). (5) `errors.Is(err, os.ErrNotExist)` → 404 "bundle not
  mirrored"; any other read error → 500. (6) `logclient.RecordBytesFromBundle(bundle, offset)`
  → `errors.Is(err, logclient.ErrLeafOutOfBundle)` → 404 (bundle is mirrored but only partial
  and does not yet contain this leaf); any other → 500. (7) write the raw record bytes.
- **Content type:** serve the record bytes verbatim as `application/octet-stream` (a fresh
  local `const` or inline literal) — they are opaque canonical-envelope BLOBs, NOT the JSON
  proof shape; do NOT wrap them in a JSON struct. Keep the documented post-status write-drop
  convention (`_, _ = w.Write(data)`), matching `tilesserve.writeBlob`.
- **Correctness rule (learnings.md "iscc_id → seq is one-to-many, schema-agnostic, ADR-0008"):**
  this surface interprets nothing — it returns the leaf's raw bytes by absolute index. No
  ISCC-ID codec, no `note.$schema` parsing. The `index` here is the absolute leaf **seq**,
  consistent with `serveInclusion`'s `selectSeq` which resolves an `iscc_id` to a seq.
- **Routing (learnings.md "monitor binary"):** `/entries` mounts as an EXACT path on the
  per-hub inner `http.ServeMux` in `hubHandler`, so the most-specific match beats the `"/"`
  subtree (same posture as `/inclusion` and `/consistency`). `proofserve.Handler` switches on
  `r.URL.Path` internally, so the single `proofs` handler already in `hubHandler` serves it
  once the `case "/entries"` arm is added — just add `mux.Handle("/entries", proofs)`.
- **Oracle / conformance gate is correctly N/A for this slice** (learnings.md pattern): the
  record-bytes extractor is a pure decode + index — no signature / RFC-6962 / Merkle /
  did:web / fsck path is introduced (it does NOT re-hash or verify; `serveInclusion` already
  owns the proof crypto). State this in the advance notes; do NOT weaken any gate to pass.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass,
  `gofmt -l .` empty).
- `go test -run TestRecordBytesFromBundle ./internal/logclient` passes: a bundle of N known
  records returns `eb.Entries[offset]` byte-for-byte for in-range offsets, returns
  `ErrLeafOutOfBundle` for `offset >= len(Entries)` (assert via `errors.Is`), and `%w`-wraps a
  truncated-bundle frame error — using a `leafhasher_test.go`-style manual uint16 bundle
  encoder as the independent third path.
- `go test -run 'TestServeEntries|TestEntries' ./internal/proofserve` passes: an `httptest`
  request to `/entries?index=<seq>` over a seeded `SQLiteFetcher` returns 200 + the exact
  record bytes for an in-tree leaf; missing/non-numeric `index` → 400; `seq >= LastSize` →
  404; an un-mirrored bundle → 404; a non-GET method → 405.
- `go test -run TestMirror ./cmd/iscc-monitor` passes, with a new sub-test asserting
  `/<origin>/entries?index=<seq>` routes through the shared mux and returns the record bytes
  (the existing `/inclusion` route sub-test stays green).
- `entries.go`'s own imports are exactly `errors`+`fmt`+`tessera/api`, and
  `GOOS=js GOARCH=wasm go build ./internal/logclient` still exits 0.
- `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no dependency
  or schema change).

## Done When
`/entries?index=<seq>` serves a single accepted leaf's raw record bytes from the local mirror
(404 outside the accepted tree / un-mirrored bundle, 400 on a bad index, 405 non-GET), all
Verification checks pass, and M2's three computed-proof surfaces (inclusion, consistency,
entries) are complete.
