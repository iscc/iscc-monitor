## 2026-06-21 — Fix the `/records` list — cap at `LastSize`, clamp overflow, reach seq 0

**Done:** Closed all three record-list defects (the two open `normal` `[review]` issues) in two
source files: `ListRecords` now takes an accepted-tree ceiling (`last`) applied to BOTH the
`COUNT(*)` total and the page query, and an explicit `hasFrom bool` so the seq cursor is no longer
overloaded as the start-at-newest sentinel; `serveRecords` threads `fs.LastSize`, clamps the page
size while still `uint64` BEFORE the `int()` conversion, and `parseUint` now rejects overflow.

**Files changed:**
- `internal/store/iscc_index.go`: `ListRecords` signature is now `(ctx, hubID, last uint64,
  hasFrom bool, from uint64, n int)`. Added `AND seq < ?` (bound `int64(last)`) to both the total
  count and the windowed SELECT; the cursor predicate is gated on `hasFrom` instead of `from > 0`,
  so `from=0` is a real cursor. Docstring updated.
- `internal/proofserve/handler.go`: `serveRecords` passes `fs.LastSize` and `hasFrom` (=`fromErr ==
  nil`) into `ListRecords`; the page-size clamp moved to `if n > maxPageSize { n = maxPageSize }`
  BEFORE `pageSize = int(n)`; `HasNewer` now keys on `hasFrom`. `parseUint` rejects values that would
  overflow `uint64` (returns its existing error type, leaving empty/non-numeric behaviour intact);
  added `math` import. Two docstrings updated.
- `internal/store/iscc_index_test.go`: updated all `ListRecords` callers to the new signature; added
  a `from=0`-cursor assertion to `TestListRecords` (reaches seq 0, not the newest page) and a new
  `TestListRecordsCeiling` (rows + total cap at `seq < last`).
- `internal/proofserve/records_test.go`: added `TestRecordsClampsHostilePageSize`
  (`n=9223372036854775808` → ≤ `maxPageSize` rows), `TestRecordsOlderLinkReachesSeq0` (walks the
  page-emitted older chain at `n=1` down to seq 0), `TestRecordsCeilingHidesUnacceptedLeaves`
  (projections above `LastSize` never list), and `TestParseUintOverflow` (overflow + contract).

**Verification:** `mise run check` → green (build + vet + all 20 packages `ok`; `gofmt -l .` empty).
Per-criterion:
- [x] `go test -run TestListRecords ./internal/store` PASS incl. `TestListRecordsCeiling`; ceiling
  drop mutation → FAILS (`total = 6, want 4`; seq 4/5 leak), reverted (non-vacuous).
- [x] `go test -run TestRecords ./internal/proofserve` PASS incl. the overflow-clamp and
  older-link-reaches-seq-0 cases.
- [x] `go list -deps ./internal/store | grep '^net/http$'` empty (store stays a leaf); package
  `.Imports` unchanged (no `logclient`/`net/http`).
- [x] `git diff --stat HEAD -- internal/store/schema.sql go.mod go.sum` empty (byte-identical).
- [x] Ceiling assertion: a hub with rows at `seq >= last` lists only `seq < last`, total counts only
  `seq < last` (store + proofserve).
- [x] Older-chain assertion: following the page-emitted older chain (`n=1`) from the page ending at
  seq 1 reaches a page containing seq 0; re-overload mutation → walk never reaches seq 0, reverted.
- [x] Hostile-`n` assertion: `?n=9223372036854775808` renders ≤ `maxPageSize` rows; post-`int()`
  clamp mutation → whole 300-row index renders, reverted.
- [x] `parseUint` overflow guard mutation-proven (`MaxUint64+1` rejected), reverted.

**Next:** Resume the planned M-UI order — the single-record page (declaration / deletion / unknown
schema) and re-point each `/records` row from `entries?index=<seq>` to it, then the certificate of
inclusion (`/inclusion/{iscc_id}`, which re-engages the oracle gate). The record-list Verify criterion
should now be on its bar.

**Notes:**
- **Oracle/conformance gate correctly N/A** — pure HTML render of persisted `iscc_index` rows via a
  leaf read; no signature / RFC-6962 / Merkle / did:web / fsck / proof path touched. Not skipped, it
  does not apply.
- **Cursor-encoding choice:** went with `hasFrom bool` threaded into `ListRecords` (not a `+1` /
  1-based cursor) — it keeps the URL semantics intact (`from=<seq>` is always an inclusive upper-bound
  seq, `from=0` == seq 0) and the template emits unchanged, so emit / parse / query all agree with no
  off-by-one. The template (`records.html`) needed no change: it already emits raw `OlderFrom` /
  `NewerFrom` seq values.
- **`parseUint` is shared** by `selectSeq`/`serveInclusion`/`serveEntries`/`serveRecords`; the overflow
  rejection only ADDS a guard returning the same `error` type, so every caller treats overflow exactly
  as it already treats a malformed value (→ 400 / silent default). Verified the reviewer's exact repro:
  `n=9223372036854775808` (= `1<<63`) parses fine and is caught by the `> maxPageSize` clamp, not by
  the overflow guard — so the clamp is the load-bearing anti-DoS guard, the overflow guard handles
  21+-digit values that previously wrapped silently.
- **`int64(last)` cast** follows the file's existing `int64(from)` / `int64(r.Seq)` cast symmetry (a
  tree size always fits in int64); no new lint or vet noise.
- Working tree carries only the 4 in-scope files; no scratch dir created; `go.mod`/`go.sum`/
  `schema.sql` byte-identical. The 3x `overlayStatus`/`hubStatus` consolidation (open `low` issue)
  was left untouched per scope.
