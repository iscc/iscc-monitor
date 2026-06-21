## 2026-06-21 — Review of: Fix the `/records` list — cap at `LastSize`, clamp overflow, reach seq 0

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance closes all three record-list defects (the two open `normal` `[review]` issues)
in exactly the two source files `next.md` named: `ListRecords` gained an accepted-tree ceiling applied to
BOTH the `COUNT(*)` total and the windowed `SELECT`, plus an explicit `hasFrom bool` so the seq cursor is
no longer overloaded as the start-at-newest sentinel; `serveRecords` threads `fs.LastSize` + `hasFrom`,
moved the page-size clamp before the `int()` conversion, and `parseUint` now rejects uint64 overflow. The
diff is tight, well-documented, all gates are green, and every new assertion is mutation-proven
non-vacuous (I re-ran all three mutations independently).

**Verification:**
- [x] `mise run check` — green (build + vet + all 20 packages `ok`).
- [x] `gofmt -l .` — empty (no formatting failures).
- [x] `go test -run TestListRecords ./internal/store` — PASS incl. new `TestListRecordsCeiling`; drop-ceiling mutation → FAILS (`total = 6, want 4`; seq 4/5 leak), reverted.
- [x] `go test -run TestRecords ./internal/proofserve` — PASS incl. `TestRecordsClampsHostilePageSize`, `TestRecordsOlderLinkReachesSeq0`, `TestRecordsCeilingHidesUnacceptedLeaves`, `TestParseUintOverflow`.
- [x] Mutation: post-`int()` clamp → `TestRecordsClampsHostilePageSize` FAILS (whole 300-row index renders), reverted.
- [x] Mutation: re-overload `from` (`hasFrom`→`from > 0`) → `TestListRecords` + `TestRecordsOlderLinkReachesSeq0` FAIL (seq 0 unreachable), reverted.
- [x] `go list -deps ./internal/store | grep '^net/http$'` empty AND no `internal/logclient` — store stays a leaf; `.Imports` unchanged.
- [x] `git diff --stat HEAD~1..HEAD -- internal/store/schema.sql go.mod go.sum` empty (byte-identical).
- [x] Scope: 2 non-test/doc `.go` files (handler.go, iscc_index.go) + 2 test files — within the ≤3 budget; nothing in `## Not In Scope` touched.
- [x] Gate-integrity scan over all unpushed commits (`@{upstream}..HEAD`) — no `//nolint`/`t.Skip`/build-tag/swallowed-error/removed-assertion patterns.
- [x] Oracle gate — correctly N/A (pure HTML render of persisted `iscc_index` rows via leaf reads; no signature / RFC-6962 / Merkle / did:web / fsck / proof path). Not skipped; does not apply.

**Issues found:** (none) — both open `normal` `[review]` issues verified fixed and deleted from `issues.md`.

**Codex second opinion:** Codex ran long (broad codebase grep well past the advance scope) but produced
a clean verdict before completing: "The changes correctly apply the accepted-size ceiling to record
listing, preserve from=0 as a real cursor, clamp page size before int conversion, and reject uint64 parse
overflow. The updated call sites and tests are consistent, and the test suite passes." No `[P1]`–`[P3]`
findings. Concurs with my independent assessment and mutation testing; nothing to triage.

**Next:** Resume the planned M-UI order — the single-record page (declaration / deletion / unknown
schema) and re-point each `/records` row from `entries?index=<seq>` to it, then the certificate of
inclusion (`/inclusion/{iscc_id}`, which re-engages the oracle gate). The record-list Verify criterion is
now on its bar.

**Notes:**
- The record-list section of `learnings/http-surface.md` was net-tightened: the two stale "open issue"
  pointers were collapsed into a single settled bullet carrying the durable seq-cursor-pagination lessons
  (never overload `0` as both cursor + sentinel; clamp page size while still `uint64`; cap COUNT and the
  windowed SELECT together; `parseUint` overflow reject). File is 157 lines — at the soft cap; a future
  record-list review should net-reduce by collapsing more settled traps rather than appending.
- Three remaining `low` issues only (the `overlayStatus`/`hubStatus` 3x duplication, the Mirror-seam
  leak, the proofserve `writeReadError` lift, the scaling trip-wire metrics, and the `notecheck` `out`
  param) — all skipped by the loop; reserved for human-directed work.
- The `olderHref` test helper correctly distinguishes the live anchor (`older &rarr;</a>`) from the dead
  `<span>` and grabs the older link's href via `LastIndex` past the newer link in source order — a
  faithful walk of the chain the page actually emits, not a hand-built URL.
