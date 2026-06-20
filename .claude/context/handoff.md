# Handoff

## 2026-06-20 — Review of: SQLiteFetcher — read mirrored tiles/bundles/checkpoint back as a `fsck.Fetcher`

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** `internal/store/tiles.go` adds the partial-tile-discipline write/read CRUD
(`RecordTile`/`RecordEntryBundle`/`ReadTileBlob`/`ReadEntryBundleBlob`/`LatestCheckpointRaw`) and
`fetcher.go` adds `SQLiteFetcher` satisfying tessera's three-method Fetcher shape with the load-bearing
p↔width mapping and an inline `PartialOrFullResource` partial→full fallback honoring the `os.ErrNotExist`
contract. Correct, well-tested at the public seam (17 new tests, all uncached-green), scope-disciplined
(exactly 2 production files, both new + in scope), and store stays a leaf with go.mod/go.sum
byte-identical. The single note is a documented, justified test-conformance technique (see Notes).

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 8 packages ok.
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -count=1 ./internal/store` — PASS uncached; all 17 new tests pass verbosely.
- [x] Tile round-trip: `RecordTile(0,0,256,…)` → `ReadTile(p=0)` returns identical bytes; `width=44`
      partial → `ReadTile(p=44)` returns its bytes. sha256 column = `sha256.Sum256(data)`. (verified)
- [x] `is_full`: raw `SELECT is_full` = 1 for width 256, 0 for width 44 (tiles **and** entry_bundles).
- [x] Partial overwrite: second `RecordTile` at same `(hub,level,index,width=100)` overwrites in place
      → `tiles` row count stays 1 (composite-PK upsert).
- [x] Not-exist: `ReadTile`/`ReadEntryBundle`/`ReadCheckpoint` on un-written keys satisfy
      `errors.Is(err, os.ErrNotExist)`.
- [x] Partial→full fallback: only-full stored + `ReadTile(p=200)` returns the full bytes; both-missing
      `ReadTile(p=100)` still wrapped `os.ErrNotExist`.
- [x] `ReadCheckpoint` / `LatestCheckpointRaw` return the highest-`tree_size` raw bytes (two sizes
      stored, asserts the higher).
- [x] `go list -deps ./internal/store | grep '^net/http$'` empty; `database/sql` present (store is a
      leaf). `.Imports` = `context crypto/sha256 database/sql embed errors fmt internal/tiles
      modernc.org/sqlite os time`.
- [x] Reference files untouched (`git diff --quiet HEAD~1..HEAD -- checkpoints.go schema.sql
      internal/tiles` → clean); `go.mod`/`go.sum` byte-identical; `go mod tidy` is a verified no-op.
- [x] Conformance assertion compiles: local `fsckFetcher` interface is byte-for-byte identical to the
      real `fsck.Fetcher@v1.0.2` (reviewer diffed against `fsck/fsck.go`), pinned by `var _ fsckFetcher
      = SQLiteFetcher{}`.
- [x] Gate-integrity scan over the 3 unpushed commits: no `//nolint`, `t.Skip`, build-tag exclusions,
      swallowed errors, deleted assertions, or loosened gates (the only `[x]` match was prior handoff
      prose, not code).
- [x] Oracle/conformance gate correctly **N/A** (no signature/RFC-6962/Merkle/did:web/`fsck`-rebuild
      path; plain CRUD + synthetic BLOB round-trip).

**Issues found:** (none) — no open issues; nothing filed this iteration.

**Next:** Either of the two seams this slice unblocks:
1. Wire `CheckEquivocation` into `follower.checkConsistency` — source the RFC-6962 consistency-proof
   hashes from `SQLiteFetcher`. Touches `internal/follower/`; needs tile fixtures for an end-to-end test.
2. The dedicated `fsck` root-rebuild conformance slice — real tile fixtures + `fsck.New(...).Check(...)`
   over `SQLiteFetcher` + the inclusion cross-check vs the hub's `IsccLogInclusionProof`. This is the
   first slice where the trust-root **oracle gate re-arms** for the mirror path; it lives in `cmd/` or a
   future conformance package that *can* take the `fsck` dep (with its otel/klog closure).
   Recommend (1) first — it advances M1 to completion (the last unwired self-consistency trigger).

**Notes:**
- **NOTE (accepted, not a blocker): `var _ fsck.Fetcher` assertion uses a local interface copy, not the
  real `fsck` import.** `next.md` asked for `var _ fsck.Fetcher = SQLiteFetcher{}` AND byte-identical
  go.mod with "no new dep needed" — these are mutually exclusive. Reviewer confirmed from source: `go
  list -deps github.com/transparency-dev/tessera/fsck` pulls `net/http`, `otel`, and `klog`, so
  importing `fsck` even in a `_test.go` breaks store leaf purity and forces `go mod tidy` to add
  indirect requires. The author's resolution (a local `fsckFetcher` interface byte-for-byte identical to
  the real one, asserted with `var _ fsckFetcher = SQLiteFetcher{}`) gives the **equivalent**
  drift-detection guarantee without the dep leak, keeps go.mod/go.sum byte-identical, and keeps store a
  leaf — all explicitly load-bearing constraints. Sound engineering judgment, transparently flagged.
  Not HUMAN-REVIEW material: it is test-internal, fully reversible, and the production `SQLiteFetcher`
  is unaffected. Recorded in learnings.
- The actual `fsck.New(...).Check(...)` root-rebuild is correctly out of scope (no fixtures) and
  unaffected; this slice only proves the read/write round-trip + not-exist contract with synthetic BLOBs.
- `boolToInt` is a small new private helper in `tiles.go` beside the existing `unixOrNil`/
  `nullStringOrNil` helpers — same package, same style. No dead code; `go vet` clean.
- CI/`notecheck`: still no `.github/workflows/` in the repo — the external signature-parity oracle
  remains unwired. Pre-existing (flagged in earlier handoffs), not a regression of this slice, and N/A
  to the CRUD path here. Worth wiring before the `fsck`-rebuild conformance slice so the trust-root
  oracle has CI coverage when the mirror path first faces it.
