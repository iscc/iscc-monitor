## 2026-06-21 — Review of: Unify the tlog-tiles `p`-vocabulary — store owns the only `p→width` translation

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` re-parameterized `Store.RecordTile`/`RecordEntryBundle` to take the tlog-tiles
partial qualifier `p uint8` (computing `width := widthForP(p)` internally via the package-private
single authority in `fetcher.go`), deleted the follower's duplicate `widthForP`, and switched both
ingest call sites to pass `c.Partial` straight through. The diff is scope-clean (3 production files,
one of them a comment-only doc fix), store stays a leaf, and the load-bearing invariant is
reviewer-mutation-proven non-vacuous through the public surface. Resolves the last open `normal`
issue (ADR-0005 single-source-of-truth for the mirror coordinate mapping).

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 15 packages `ok`.
- [x] `gofmt -l .` — empty (clean).
- [x] `grep -rn "func widthForP" internal/` — exactly **one** hit (`internal/store/fetcher.go:113`).
- [x] `grep -n "widthForP" internal/follower/ingest.go` — empty (exit 1); both call sites pass
  `c.Partial` directly.
- [x] `go test -count=1 -run 'TestRecordTile|TestRecordEntryBundle|TestFetcher' ./internal/store` — PASS
  uncached. Named regression guards `TestRecordTilePartialOverwrite` (p=100), `TestRecordTileRoundTrip`,
  `TestRecordEntryBundleRoundTrip`, `TestRecordTilePartialIsNotFull`, `TestFetcherReadTile*`,
  `TestFetcherReadEntryBundle*` all `--- PASS`.
- [x] `go test -count=1 ./internal/follower ./internal/logclient ./internal/proofserve
  ./internal/tilesserve` — all `ok` **uncached** (full/partial mirror round-trips, fsck root-rebuild,
  equivocation/inclusion consistency-proof all read the mirror back correctly; no conformance regression).
- [x] `go list -deps ./internal/store | grep -E "net/http|internal/logclient|internal/follower"` —
  empty (store stays a leaf). `git diff --stat HEAD~1..HEAD -- internal/store/schema.sql go.mod go.sum`
  — empty.
- [x] **Mutation testing (reviewer, reverted) — the public-surface invariant is non-vacuous.** Forcing
  `widthForP`'s full mapping to `tiles.TileWidth - 1` makes `TestIngestTilesWidthMapping` ("tile L0 I0
  not found at width 256") AND `TestRecordTileRoundTrip` ("full tile not found") FAIL. A green-but-wrong
  `widthForP` cannot ship through the public `RecordTile`/`ReadTileBlob` seam; reverting restores green.
- [x] **Deleted `TestWidthForP` is not a gate dodge.** It tested the now-deleted follower-private
  `widthForP` arithmetic directly; the load-bearing "full coord (p=0) readable at width 256, NOT 0"
  invariant is now pinned *more strongly* by `TestIngestTilesWidthMapping` over the public surface (both
  the positive read-at-256 and the negative not-readable-at-0).
- [x] **Gate-integrity scan over the 3 unpushed commits (`@{upstream}..HEAD`)** — no `//nolint` /
  `t.Skip` / build-tag exclusion / swallowed-error / loosened gate in code (the single grep hit is a
  prose line in the prior `handoff.md`).
- [x] **Not-In-Scope respected** — `fetcher.go` (`widthForP` body + read methods), `schema.sql`, and
  `cmd/notecheck` all byte-untouched; `ReadTileBlob`/`ReadEntryBundleBlob` keep their `width int` param.
- [x] **Oracle/conformance gate correctly N/A** for this slice (plain CRUD + the existing pure `p→width`
  arithmetic; no signature/RFC-6962/Merkle/did:web/fsck *path* changed) — but the four
  mirror-consuming packages were re-run uncached anyway because their fsck root-rebuild and
  equivocation/inclusion proofs read the mirror back via `SQLiteFetcher`.

**Issues found:** (none). Resolved + deleted the ADR-0005 `normal` issue "Tile writers require `width`,
duplicating the tlog `p` translation in the follower" (verified the fix in production + test). One `low`
issue remains in `issues.md`: the `cmd/notecheck` vestigial `out io.Writer` param (loop-skipped).

**Next:** The `normal` backlog is drained. With the M3 mirror arc + three computed proofs + HTTP
serving + this single-source cleanup landed, the natural next arc is the proof-surface cache /
verify-for-me REST surface. Scope the proof-bundle assembly (`{checkpoint, inclusion proof, record
bytes, hub key, ots?}`) that both the in-browser verifier and verify-for-me share, reading from the
now-unified store/SQLiteFetcher seam — start with the bundle assembler in `internal/proofserve` (or a
sibling) that composes `LatestCheckpointRaw` + `InclusionProofFromTiles` + the record bytes + the
resolved hub key into one self-contained, client-verifiable package.

**Notes:**
- **Third production file `internal/tiles/coords.go` is a comment-only doc fix** (confirmed via diff
  filter: every changed line is a comment) — its `BundleCoord`/`TileCoord` docs claimed the *writer*
  does the `p→width` translation, now false after the move. Within the ≤3 scope ceiling; flagged
  transparently by the advance author. `next.md` listed only 2 modify targets, so this is a benign
  over-list, not a scope violation.
- **`uint8(256)` is never written** — every full coord is the literal `0` per the lossy-guard rule; the
  test sites carrying a runtime `width` variable (`equivocation_test.go`'s `len(nodes)`,
  `entries_test.go`'s `last-first`) translate with an explicit `if width == tiles.TileWidth { p = 0 }`
  guard so a future full bundle/tile in those loops maps to `0`, never a wrapped `uint8(256)`.
- **`.devcontainer/devcontainer.json` is modified in the working tree but NOT committed** (pre-existing
  infra tuning: `--memory` runArgs + `GOFLAGS=-p=2`) — left for the human/infra owner, correctly not
  swept into this loop's commits.
- **Loop is CONTINUE:** M1/M2 met, M3 in progress (mirror arc + three computed proofs + HTTP serving +
  this cleanup done); verify-for-me REST, dashboard, log browser, WASM verifier, and OTS anchoring
  remain the bulk of v1 — not DONE. No human-only decision open — not STOP.
