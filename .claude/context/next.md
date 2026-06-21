# Next Work Package

## Step: Transport primitives to fetch a tile / entry bundle over the Fetcher seam (`logclient.FetchTile` / `FetchEntryBundle`)

## Goal
Add the two missing pure transport primitives that fetch one hash tile and one entry bundle from a
hub at their canonical tlog-tiles paths — the seam between the existing `tiles.TileCoords` /
`tiles.BundleCoords` enumerations and the existing `store.RecordTile` / `RecordEntryBundle` writers.
This is the smallest unblocking slice of the M2 live tile-ingestion writer: with it, the next step
can wire a `PollHub` fetch loop that walks the coords, fetches via these, and writes to the mirror.

## Scope
- **Create**: `internal/logclient/tilefetch.go` — `FetchTile(ctx, fetcher Fetcher, baseURL string,
  level, index uint64, p uint8) ([]byte, error)` and `FetchEntryBundle(ctx, fetcher Fetcher,
  baseURL string, index uint64, p uint8) ([]byte, error)`.
- **Create**: `internal/logclient/tilefetch_test.go` — offline table tests with a fake `Fetcher`.
- **Modify**: (none — purely additive; no existing non-test file changes)
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/checkpoint.go` — the primitive to port from (same
    `origin()` → `"https://"+name+"/<path>"` → `fetcher.Fetch` → verbatim-bytes shape, same `%w` wrapping).
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` — the `Fetcher` 1-method seam
    (`Fetch(ctx, url) ([]byte, error)`) and its 404→`os.ErrNotExist` contract these must preserve.
  - `/workspace/iscc-monitor/internal/tiles/layout.go` — `tiles.TilePath(level, index, p)` /
    `tiles.EntriesPath(index, p)`, the canonical path producers to call (do not hand-build paths).
  - `/workspace/iscc-monitor/internal/tiles/layout_test.go` — golden path strings (`tile/1/000`,
    `tile/0/000.p/255`, `tile/entries/000.p/8`, `tile/entries/255`) to anchor the expected URLs.
  - `/workspace/iscc-monitor/cauldron/tessera/client/fetcher.go` (lines 97-111) — the reference
    `HTTPFetcher.ReadTile`/`ReadEntryBundle`: each is `fetch(ctx, layout.TilePath(...))` /
    `fetch(ctx, layout.EntriesPath(...))`. Port the URL-construction shape (root + path), NOT the
    `PartialOrFullResource` fallback.

## Not In Scope
- Wiring these into `PollHub` / a tile-ingestion loop that walks `TileCoords`/`BundleCoords` and
  writes via `RecordTile`/`RecordEntryBundle` — that is the **next** step (it touches `follower.go`
  and is its own ≤3-file unit). This step delivers only the fetch primitives + their tests.
- The `PartialOrFullResource` partial→full fallback (tessera's `fetcher.PartialOrFullResource`). The
  store's `SQLiteFetcher` already owns that fallback on the read side; the ingestion fetch fetches
  exactly the `p` the coord enumeration names. Do not replicate it here.
- Any `p`↔`width` translation, `RecordTile`/`RecordEntryBundle` calls, `is_full` logic, or
  `widthForP` — all downstream of this step, in the store/follower layers.
- `fsck` root-rebuild wiring, the inclusion cross-check, `iscc_index`, or capturing real tile/bundle
  fixtures in `testdata/live/` — later M2 slices.
- Touching the open `low` `cmd/notecheck` `out io.Writer` issue (loop-skipped).

## Implementation Notes
- **Port `FetchCheckpoint` verbatim in shape.** It is the exact template: `origin(baseURL)` (the
  private helper, reused — NOT a second parser) → `"https://" + name + "/" + path` → `fetcher.Fetch(ctx,
  url)` → return body verbatim. `name` is already `<domain>/log` (e.g. `sb0.iscc.id/log`), so the URL
  is `https://sb0.iscc.id/log/tile/1/000`. Note `FetchCheckpoint` writes `"https://" + name +
  "/checkpoint"` (leading slash on a literal); here append `"/" + tiles.TilePath(...)` since
  `TilePath`/`EntriesPath` return slash-less paths like `tile/1/000`.
- **Path producers:** call `tiles.TilePath(level, index, p)` and `tiles.EntriesPath(index, p)` — never
  hand-format the `tile/<l>/<i>` / `tile/entries/<i>` / `.p/<w>` strings. This keeps the package
  delegating to tessera's layout math (target Stack rule: reuse, never reimplement).
- **Import note:** `internal/logclient` does not yet import `internal/tiles`. Adding that import edge
  is correct and clean (`tiles` is a pure `api/layout`-only leaf; no `net`/`sqlite` enters the
  closure — `logclient` already imports `net/http` via `didresolve.go`, so the package's non-WASM
  status is unchanged). `go.mod`/`go.sum` stay byte-identical (`tessera/api/layout` is already in the
  closure via `proofbuilder.go`).
- **Error wrapping (404-transparency rule):** wrap both the `origin()` error and the `fetcher.Fetch`
  error with `%w`, exactly as `FetchCheckpoint` does, so a 404's `errors.Is(err, os.ErrNotExist)`
  survives — the ingestion loop will need to tell "tile not served yet" from a hard transport fault.
  Include the constructed `url` in the Fetch-error wrap (mirror `fetch checkpoint %q`).
- **No `os.ErrNotExist` synthesis here.** Do NOT map a missing tile to a sentinel yourself; that is
  the `Fetcher` implementation's contract (`httpFetcher.Fetch` already maps 404→`os.ErrNotExist`).
  These primitives only propagate it via `%w`.
- **Pure-transport, no parsing.** Return the raw bytes verbatim — do not `UnmarshalText` the tile or
  bundle (that belongs to `LeafHashes` / the proof builders / fsck). Mirror `FetchCheckpoint`'s
  "body verbatim" doc.
- **Test seam:** use a fake `Fetcher` (a struct with a `Fetch` func field, or a `map[url]→bytes`, in
  the spirit of the existing `compositeFetcher`/`countingFetcher` in `follower_test.go`) that records
  the URL it was asked for and returns canned bytes. Assert the **exact URL** the primitive constructed
  for representative coords (full + partial, both tile and bundle), and assert a fetcher error /
  `os.ErrNotExist` propagates with `errors.Is`. The oracle/conformance gate is correctly **N/A** here
  (pure URL construction + transport; no signature/RFC-6962/Merkle/did:web/`fsck` path), same as
  `FetchCheckpoint` and the `tiles` layout slice.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .`
  empty).
- `go test -run 'TestFetchTile|TestFetchEntryBundle' -count=1 ./internal/logclient` passes.
- Asserted in the test: `FetchTile(ctx, fake, "https://sb0.iscc.id", 1, 0, 0, …)` requests URL
  `https://sb0.iscc.id/log/tile/1/000` and returns the fake's bytes verbatim.
- Asserted in the test: `FetchTile(ctx, fake, "https://sb0.iscc.id", 0, 0, 255, …)` requests
  `https://sb0.iscc.id/log/tile/0/000.p/255` (partial suffix via `TilePath`).
- Asserted in the test: `FetchEntryBundle(ctx, fake, "https://sb0.iscc.id", 255, 0, …)` requests
  `https://sb0.iscc.id/log/tile/entries/255`; and a width-8 partial requests
  `https://sb0.iscc.id/log/tile/entries/000.p/8`.
- Asserted in the test: a `Fetcher` returning a `%w`-wrapped `os.ErrNotExist` makes both primitives
  return an error for which `errors.Is(err, os.ErrNotExist)` is true.
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change).
- `GOOS=js GOARCH=wasm go build ./internal/tiles` still exits 0 (the WASM-shared leaf is untouched).

## Done When
`internal/logclient` exports `FetchTile` and `FetchEntryBundle` as pure transport primitives that
build the canonical tlog-tiles URL via `tiles.TilePath`/`tiles.EntriesPath` and propagate
`os.ErrNotExist`, with all Verification criteria passing.
