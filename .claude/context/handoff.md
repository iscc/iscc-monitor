## 2026-06-21 — Review of: Transport primitives to fetch a tile / entry bundle over the Fetcher seam (`logclient.FetchTile` / `FetchEntryBundle`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the two pure transport primitives `FetchTile` and `FetchEntryBundle`
(`internal/logclient/tilefetch.go`), ported verbatim-in-shape from `FetchCheckpoint`: `origin()` →
`"https://"+name+"/"+tiles.TilePath/EntriesPath(...)` → `fetcher.Fetch` → body verbatim, with both
`origin()` and Fetch errors `%w`-wrapped so a 404's `os.ErrNotExist` survives. The diff is purely
additive (2 new files, zero existing non-test files touched), matches `next.md` exactly, and every
Verification criterion passes. Independent inspection confirms the golden URLs anchor on tessera's own
layout golden strings, not author assertion.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 11 packages `ok`.
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -run 'TestFetchTile|TestFetchEntryBundle' -count=1 ./internal/logclient` — passes (all 10 subtests/cases, verbose).
- [x] `FetchTile(…, "https://sb0.iscc.id", 1, 0, 0)` → `https://sb0.iscc.id/log/tile/1/000`, bytes verbatim — asserted + passing.
- [x] `FetchTile(…, "https://sb0.iscc.id", 0, 0, 255)` → `https://sb0.iscc.id/log/tile/0/000.p/255` — asserted + passing.
- [x] `FetchEntryBundle(…, 255, 0)` → `…/log/tile/entries/255`; width-8 partial (`index 0, p 8`) → `…/log/tile/entries/000.p/8` — asserted + passing.
- [x] Both primitives propagate a `%w`-wrapped `os.ErrNotExist` such that `errors.Is(err, os.ErrNotExist)` is true — `TestFetch{Tile,EntryBundle}NotFound` pass.
- [x] `git diff --quiet HEAD -- go.mod go.sum` exits 0 — no dependency change (verified vs HEAD~1 and working tree).
- [x] `GOOS=js GOARCH=wasm go build ./internal/tiles` exits 0 — and `./internal/didweb` (trust-root WASM-shared leaf) also exits 0.
- [x] Gate-circumvention scan across all 3 unpushed commits (`@{upstream}..HEAD`) — no `//nolint`, `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion in code (only matches are handoff prose).
- [x] Scope discipline — exactly 2 non-context files changed (`tilefetch.go`, `tilefetch_test.go`), both new, both `internal/logclient`; nothing from `## Not In Scope` done.
- [x] Oracle/conformance gate — correctly **N/A** for this slice; verified `tilefetch.go` imports only `context`+`fmt`+`internal/tiles` (no signature/Merkle/RFC-6962/did:web/fsck path). Trust-root packages (`logclient`/`didweb`/`follower`/`notecheck`) re-run uncached → all `ok`. CI `notecheck` signature-parity oracle present in `.github/workflows/ci.yml` and remains green by construction (this diff cannot regress it).

**Issues found:** (none)

**Next:** Wire `FetchTile`/`FetchEntryBundle` into a `PollHub` tile-ingestion loop (its own ≤3-file
slice, touching `follower.go`): on a verified growing checkpoint, walk `TileCoords(size)` +
`BundleCoords(size)`, fetch each tile/bundle over these primitives, and write via
`store.RecordTile`/`RecordEntryBundle` — translating the path-API `Partial` (`p`) to the store's
`width` (full tile `p==0` → width 256, via `widthForP`), re-fetching partials every poll (ADR-0005).
That writer is what finally arms the deferred `fsck` root-rebuild over `SQLiteFetcher` and the
inclusion cross-check vs the hub's `IsccLogInclusionProof`, both of which want real tile/bundle
fixtures captured in `testdata/live/`.

**Notes:**
- `FetchTile`/`FetchEntryBundle` are an intentional unwired export seam (no production caller yet) —
  consistent with the established project pattern (`TileCoords`/`BundleCoords`/consistency triggers/
  `LeafHashes`/`RunFsck`). `go vet` clean; not dead code. First caller is the ingestion-writer slice.
- The package-level `merkle`/`note`/`fsck` imports `go list` shows for `internal/logclient` come from
  sibling files (`proofbuilder.go`/`verify.go`/`leafhasher.go`/`fsck.go`), NOT from this slice's
  `tilefetch.go` — do not attribute them to this diff.
- The only open issue is `low`/loop-skipped (`cmd/notecheck` vestigial `out io.Writer` param), untouched
  by this slice and correctly left alone.
- The `PartialOrFullResource` partial→full fallback is deliberately NOT replicated here (out of scope);
  these fetch exactly the `p` the coord enumeration names — the store's `SQLiteFetcher` owns that
  fallback on the read side.
