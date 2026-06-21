## 2026-06-21 — Review of: Paginated record list at `GET /<domain>/log/records?from=…[&n=…]`

**Verdict:** NEEDS_WORK
**Loop:** CONTINUE

**Summary:** The slice lands a clean, well-tested no-JS newest-first record list backed by a leaf
`store.ListRecords`, with the DS shell, CDN-free body, overlay-status badge, and empty state all
correct and non-vacuously tested (`DESC→ASC` mutation FAILS store + proofserve tests, reverted). But
the independent review (corroborated by Codex) found **three real defects** in pagination/coverage
that defeat stated controls: an uncapped listing that shows unaccepted leaves on a frozen/violation
hub, a page-size clamp that a huge `n` bypasses (negative `LIMIT` → unlimited), and an oldest-record
(seq 0) that is unreachable via the older-link chain. `mise run check` is green and the milestone
progresses, so the loop continues; the next iteration fixes the root causes.

**Verification:**
- [x] `mise run check` green — build + vet + all 20 packages `ok`; `gofmt -l .` empty.
- [x] `go test -run TestListRecords ./internal/store` — PASS (newest-first, hub-scoped, page window,
  total, empty hub → empty slice + nil err). Non-vacuous: `DESC→ASC` mutation FAILS, reverted.
- [x] `go test -run TestRecords ./internal/proofserve` — PASS (200 text/html, newest-first, per-row
  `entries?index=<seq>`, working `?from=…&n=…` link, no CDN URL, overlay status, empty state).
- [x] `go test -run TestRecordsEmpty ./internal/proofserve` — PASS (empty index → 200 empty state).
- [x] `go list -deps ./internal/store | grep '^net/http$'` empty — store stays a leaf.
- [x] `go test ./cmd/iscc-monitor` — PASS (`/records` mount does not regress the route table).
- [x] go.mod / go.sum / internal/store/schema.sql byte-identical to HEAD~1.
- [x] Gate-integrity scan over all unpushed commits (`origin/develop..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag/swallowed-error/deleted-test markers in code; the only matches are in overwritten
  `handoff.md` prose. No gate weakened.
- [x] Oracle/conformance gate correctly N/A — pure HTML render of persisted `iscc_index` rows via a
  leaf read; no signature/RFC-6962/Merkle/did:web/fsck/proof path.
- [ ] **Coverage honesty (ADR-0001)** — FAIL: `/records` lists `iscc_index` rows with NO `LastSize`
  cap, so a frozen/violation hub (where ingest wrote projections beyond `LastSize` before the freeze)
  shows unaccepted leaves as accepted, with `entries?index=` links `serveEntries` then 404s.
- [ ] **Anti-DoS page-size clamp** — FAIL: a huge `n` wraps `int(n)` negative BEFORE the
  `> maxPageSize` check; modernc SQLite reads a negative `LIMIT` as unlimited → whole-index render.
- [ ] **No-JS pagination reaches every record** — FAIL: the oldest record (seq 0) is unreachable via
  the older-link chain because `from=0` is overloaded as the "start at newest" sentinel.

**Issues found:** Three confirmed defects, all filed in `issues.md` as `normal` `[review]`:
1. `/records` lists unaccepted leaves (no `LastSize` cap) — coverage-honesty defect on the
   frozen/violation case (`serveRecords`/`ListRecords`). Root: `PollHub` writes projections for all
   `info.TreeSize` leaves (`ingestTiles`→`RecordProjections`) BEFORE `checkConsistency`/`AdvanceAccepted`,
   so a freeze or fault leaves `iscc_index` rows with `seq >= LastSize`. Every other record route caps
   at `>= LastSize`; this one omits it. Fix: pass `fs.LastSize` as a ceiling into `ListRecords`.
2 & 3. `/records` pagination overloads `from=0` and skips the clamp on overflow (two bugs, one root).
   (a) seq 0 unreachable via older link; (b) page-size clamp bypassed by a negative `int(n)`. Fix:
   clamp while still `uint64` before `int()`; stop overloading 0 (separate has-cursor bool or 1-based
   cursor). `parseUint` also wraps silently on overflow — bound it.

None of these block milestone progress, so all are `normal` (not `critical`); the verdict is
NEEDS_WORK because confirmed defects defeat stated controls, and PASS may never approve that.

**Codex second opinion:** Available; produced a verdict after ~3 min. **All three of its findings
were reviewer-confirmed** against the code (not taken on faith):
- **[P1] clamp page size before int conversion** — CONFIRMED. Reproduced: `int(MaxInt64+1)` wraps
  negative, the `> 200` check misses it, and a negative `LIMIT` returns all rows in modernc SQLite
  (test returned 10 rows for `n=-5`). Filed (issue 2/3b).
- **[P1] filter records to the accepted tree size** — CONFIRMED. Verified the `PollHub` ingest-before-
  advance ordering leaves projections above `LastSize` on freeze; `serveRecords` is the only record
  route without the `>= LastSize` cap. Filed (issue 1).
- **[P2] stop emitting from=0 as an older cursor** — CONFIRMED. Reproduced: page `from=1&n=1` emits
  older link `from=0&n=1`; following it shows seq 299 (newest), so seq 0 is unreachable. Filed (issue
  2/3a).
No findings dismissed. The oracle gate was N/A this slice, so no oracle-vs-Codex conflict arose.

**Next:** A single fix slice for `serveRecords`/`ListRecords` covering all three (they cluster in two
files): (1) thread `fs.LastSize` into `ListRecords` as a `seq < LastSize` ceiling and cap the total;
(2) clamp page size while `uint64` before `int()` and bound `parseUint`'s overflow; (3) un-overload
`from=0` (a `hasFrom`/1-based cursor) so the older chain reaches seq 0. Add tests: a frozen-hub-with-
projections-above-LastSize fixture asserting only accepted rows list; `n=9223372036854775808` →
≤ maxPageSize rows; an older-link walk down to seq 0. THEN resume the planned M-UI order: the
single-record page (declaration/deletion/unknown schema), re-pointing each row from `entries?index=`
to it, then the certificate of inclusion (`/inclusion/{iscc_id}`, which re-engages the oracle gate).

**Notes:**
- **Pre-existing uncommitted human edits left untouched (NOT mine, NOT committed):** the working tree
  carries human-authored edits to `.claude/adr/0007-*.md` (the "why network-level not hub-level"
  expansion) and `.claude/context/issues.md` (the matching scaling trip-wire `low` issue). They are
  well-formed and self-consistent. I staged ONLY the specific context files I touched
  (`handoff.md`, `learnings/http-surface.md`, my new `issues.md` entries) so the human can review/commit
  the ADR + their issue edit deliberately. My `issues.md` additions sit alongside theirs.
- **`learnings/http-surface.md` rotated** 165→153 lines: collapsed the settled
  `/entries`/`/verify`/`/consistency` verbose blocks into `settled:` one-liners (keeping only the
  durable traps) to make room for the record-list `from=0`-overload trap; index unchanged.
- **The root enabler is shared:** `from=0` is both a cursor value and the "newest" sentinel, and
  `parseUint`/clamp lack overflow guards. Fixing the overload + bounding the parse closes two of three
  defects together. The third (LastSize cap) is independent but in the same two files.
- CI green at the last pushed commit (`452d23c`); these three commits are unpushed. No push this cycle
  (NEEDS_WORK).
- Test totals at HEAD: **266 `func Test`** across **55** `_test.go` files.
