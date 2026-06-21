## 2026-06-21 — Review of: Cache-Control on the tlog-tiles mirror (immutable full tiles/bundles vs. revalidating partials + checkpoint)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` plumbed an `immutable bool` into `writeBlob` and set a per-route `Cache-Control`
on every tlog-tiles 200: content-addressed FULL tiles/bundles (path-API `width == 0`) get `public,
max-age=31536000, immutable`; partials (`width > 0`) and the size-varying checkpoint get `no-cache`. The
change is exactly what `next.md` asked — one production file, scope-clean, the load-bearing partial-tile
assertion present and reviewer-mutation-proven non-vacuous, trust root untouched, all gates green.

**Verification:**
- [x] `mise run check` — green (build + vet + test; all 15 packages `ok`).
- [x] `go test -count=1 ./internal/tilesserve` — ok; all 9 subtests PASS (verbose, uncached).
- [x] Full tile `GET tile/0/000` → `public, max-age=31536000, immutable` — asserted + passes.
- [x] Full bundle `GET tile/entries/000` → `public, max-age=31536000, immutable` — asserted + passes.
- [x] Partial tile `GET tile/0/001.p/44` → `no-cache`, NOT `immutable` — load-bearing assertion passes;
  reviewer mutated the predicate to always-immutable and this subtest FAILED (reverted), so non-vacuous.
- [x] Checkpoint `GET checkpoint` → `no-cache` — asserted + passes.
- [x] `gofmt -l .` empty.
- [x] `git diff --quiet -- go.mod go.sum internal/store/schema.sql` exits 0 (no dep/schema change).
- [x] `immutable` directive string in exactly one place (the `cacheImmutable` const, line 37);
  `grep no-store internal/tilesserve/handler.go` empty.
- [x] Width predicate verified from ground truth (throwaway test, removed): `ParseTileLevelIndexPartial`
  yields width 0 for `tile/0/000`, 44 for `tile/0/001.p/44`; `ParseTileIndexPartial` yields 0 for
  `tile/entries/000`. So `immutable := (width == 0)` is the correct full/partial discriminant.
- [x] No header conflict: `corsmw.Handler` sets only `Access-Control-*` (read from source); both
  `Content-Type` and `Cache-Control` are set before the first `w.Write`.
- [x] Scope: only `internal/tilesserve/handler.go` (1 prod file) + its `_test.go` + context docs changed
  across all unpushed commits. Nothing from `## Not In Scope` touched (no `proofserve`/`corsmw`/`main.go`,
  no conditional-GET/ETag, no issue-draining).
- [x] Gate integrity: scanned all 3 unpushed commits (`origin/develop..HEAD`); no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-assertion. The lone `_, _ = w.Write(data)` is the pre-existing,
  documented post-status-write idiom (matches `metricshttp`), not introduced here and not a dodge.

**Issues found:** (none) — clean, well-scoped, well-tested slice.

**Next:** The deliberately-deferred conditional-GET follow-up on the same routes: `ETag` / `If-None-Match`
(and/or `Last-Modified` / `If-Modified-Since`) with `304` responses. Content-addressed full tiles/bundles
have a strong-ETag source (the BLOB is keyed by its hashes); the checkpoint/partials need a weak/derived
validator. Alternatively, begin draining a `normal` issue — the ADR-0006 split-view / frozen-advance
items (`TestPollHubFork` re-detection via `PollHub`, frozen-hub evidence-only short-circuit,
`CheckConsistency` collapse) are the highest-value cluster — or start the proof-bundle JSON +
verify-for-me arc. Note: `proofserve`'s size-dependent surfaces (`/inclusion`/`/consistency`/`/entries`)
still have no cache policy (tied to `LastSize`), explicitly left out of this slice.

**Notes:**
- Oracle/conformance gate correctly N/A: pure HTTP header wiring on opaque BLOBs — no signature /
  RFC-6962 / Merkle / did:web / fsck / proof path touched. `derive_vkey.py` / `notecheck` / `fsck`
  golden vectors untouched and unaffected. No new import (`net/http` + `strings` + `tessera/api/layout`
  already in the closure), so go.mod/go.sum/schema are byte-identical and the WASM-shared `didweb` leaf
  is untouched.
- Two vocabularies remain a live trap for the next reviewer: path-API full = `width 0` (what this slice
  keys on, via the parsed `layout.ParseTile*` width) vs. store-column full = `256`. Both `serveTile`
  /`serveEntries` work in the path-API vocabulary, so `width == 0` is the right immutable predicate and
  mirrors `SQLiteFetcher`'s documented `p == 0 → full`. Do not confuse them.
- Housekeeping during review: a stale `.git/index.lock` (left by a sed-mutation experiment that ran
  concurrently with a checkout) was removed and `handler.go` restored clean — working tree is pristine,
  no behavioral residue.
