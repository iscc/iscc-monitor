## 2026-06-21 — Review of: Live tile/bundle ingestion writer in PollHub

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `ingestTiles` (`internal/follower/ingest.go`), the first production caller
binding `tiles.TileCoords`/`BundleCoords` → `logclient.FetchTile`/`FetchEntryBundle` →
`store.RecordTile`/`RecordEntryBundle`, and wired it into `PollHub` on the verified, non-violation path
(after `cacheHubKey`, before `recordVerdict`). The diff is tightly scoped (2 non-test files + 1 test
file, store untouched), every Verification criterion passes, and independent mutation testing confirms
both the writer and the load-bearing `widthForP` translation are non-vacuously pinned.

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 11 packages `ok`.
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -run 'TestIngest|TestPollHub|TestWidthForP' -count=1 ./internal/follower` — all pass
  (incl. existing `TestPollHub{Fork,Shrink,Equivocation,VerifiedAdvances,...}` which now also ingest).
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` exits 0 — no dependency change.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 — trust-root WASM leaf unaffected.
- [x] Coord unit test: full tile (`Partial==0`) stored/read at width 256, 44-leaf partial at width 44,
  and the full tile is NOT readable at width 0 — all asserted and passing.
- [x] After verified `PollHub` vs sb0 (tree 10183): every enumerated `TileCoords`/`BundleCoords` coord
  returns `found==true` with the fetched bytes; full level-0 tile round-trips through
  `SQLiteFetcher.ReadTile(0,0,p0)` at width 256.
- [x] Follower prod imports = `{context, fmt, logclient, metrics, store, tiles, log/slog}` (verified via
  `go list`); store closure has no `net/http` and no reverse dep — still a leaf, direction
  follower → {logclient, store, tiles, metrics}.
- [x] Scope discipline — exactly 3 changed files (`follower.go` modify, `ingest.go` + `ingest_test.go`
  new), all `internal/follower`; store byte-identical (`git diff --quiet -- internal/store/`); nothing
  from `## Not In Scope` done; `store.widthForP` stays unexported.
- [x] Gate-circumvention scan across all 3 unpushed commits (`@{upstream}..HEAD`, `.go` only) — no
  `//nolint`, `t.Skip`, build-tag exclusion, swallowed error, or deleted assertion. (The two `_ =`
  matches are the test's `data`-slice discard while checking `err` — legitimate.)
- [x] Oracle/conformance gate — correctly **N/A** for this slice (transport + CRUD, no
  signature/RFC-6962/Merkle/did:web/fsck path introduced). Re-confirmed `derive_vkey.py` reproduces
  both golden vectors (`40b74463` / `22b08f3e`); trust-root packages (`didweb`/`logclient`/`follower`/
  `notecheck`) re-run uncached → all `ok`. The CI `notecheck` parity oracle cannot regress from this
  diff (no crypto touched).

**Mutation evidence (reverted):**
- Neuter `ingestTiles` → no-op: `TestIngestTilesWidthMapping` + `TestPollHubMirrorsTiles` FAIL (no
  mirrored rows, `SQLiteFetcher.ReadTile(0,0,p0)` round-trip fails). Tests are not existence-vacuous.
- Break `widthForP` (`p==0 → 256` removed): `TestWidthForP` + both ingest tests FAIL — full tile
  invisible at width 256, readable at width 0, SQLiteFetcher round-trip fails. The highest-risk bug
  surface (per learnings) is genuinely caught.

**Issues found:** (none) — the only open issue (`cmd/notecheck` vestigial `out io.Writer`) is
`low`/loop-skipped and correctly untouched.

**Next:** Wire `RunFsck` over `SQLiteFetcher` (the M2 `fsck` root-rebuild) — now that `PollHub` mirrors
real tiles, `fsck.New(...).Check(ctx)` can rebuild the root from the local mirror and cross-check it
against the signed checkpoint root. This is the slice that **re-arms the trust-root oracle gate**
(RFC-6962 root-rebuild crypto) and is the first place the mirrored tiles face conformance — so it wants
byte-accurate live tile/entry-bundle fixtures captured into `testdata/live/` (the inclusion cross-check
vs the hub's `IsccLogInclusionProof` is the sibling slice needing the same fixtures). `RunFsck`,
`LeafHashes`, and the proof builders already exist as unwired pure seams ready for this caller.

**Notes:**
- **Synthetic, not byte-accurate, fixtures here.** The writer is transport + CRUD (not crypto), so the
  test fetchers return per-URL synthetic bytes (`"body:"+url` / `"tile:"+url`); a store read-back
  matches the exact coord fetched. The `RunFsck` slice is the one that needs real tile bytes — flag for
  define-next: do not let it reuse synthetic bytes (the root-rebuild would not be meaningful).
- **Existing `TestPollHub{Fork,Shrink,...}` now also ingest tiles** (their `compositeFetcher` returns
  checkpoint bytes for any non-did.json URL, so tile URLs get the checkpoint as synthetic BLOBs) —
  harmless, those tests assert only on their own outputs and still pass.
- **Stale comment (cosmetic, not blocking):** `internal/follower/equivocation_test.go` near line 4 says
  "the M2 tile-ingestion writer is not yet wired, so the test seeds them directly" — now slightly stale
  since this slice wires it. That test calls `checkConsistency`/`freeze` directly (not `PollHub`) so it
  is functionally correct and unaffected; `advance` left it untouched per scope discipline. A one-line
  refresh when that file is next touched is the only cleanup — not worth a dedicated issue.
- **`widthForP` is intentionally duplicated** between `internal/store/fetcher.go` (unexported) and the
  follower (re-derived one-liner, pinned by `TestWidthForP`) per `next.md` — the store's copy stays
  private and the store package is byte-untouched. Confirmed both bodies identical.
- **No remote push obstacles anticipated** — branch is `develop`, 3 commits ahead of `origin/develop`;
  pushing on PASS.
