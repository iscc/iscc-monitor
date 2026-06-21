# Next Work Package

## Step: Fix the `/records` list — cap at `LastSize`, clamp overflow, reach seq 0

## Advances
M-UI (Evidence Ledger frontend, ADR-0010) Verify criterion: *"the log-browser record list paginates
via plain links (`?from=…[&n=…]`, no-JS), newest-first, each row links to its single-record page, and
an empty log renders the informative empty state (200)"* and the umbrella *"a pre-coverage state never
renders as a guarantee (ADR-0001)"*. The paginated record list landed but the latest `review` verdict
is **NEEDS_WORK** with two open `normal` issues — both record-list defects that defeat stated controls,
so the criterion is NOT yet met. These `normal` issues preempt the next M-UI screen: per the handoff
`**Next:**`, this is the planned single fix slice before the single-record page.

Closes both `normal` `[review]` issues in `issues.md`:
- "`/records` lists unaccepted leaves (no `LastSize` cap), unlike every other record-facing route"
- "`/records` pagination overloads `from=0` and skips the page-size clamp on overflow (two bugs, one root)"

## Goal
Make the record list honest and bounded: it must show only accepted leaves (`seq < LastSize`), it must
never render the whole index when handed a hostile `n`, and its older-link chain must reach the oldest
record (seq 0). This brings the record-list Verify criterion onto its bar so M-UI can resume with the
single-record-page slice.

## Scope
- **Create**: (none)
- **Modify** (2 non-test/doc `.go` files):
  - `/workspace/iscc-monitor/internal/store/iscc_index.go` — `ListRecords`: add an accepted-tree
    ceiling param (apply it to BOTH the `COUNT(*)` total and the page query) and un-overload the
    `from == 0` "start at newest" sentinel.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — `serveRecords`: thread `fs.LastSize`
    into `ListRecords`; clamp page size while still `uint64` BEFORE the `int()` conversion; bound
    `parseUint`'s silent overflow; carry the cursor so the older chain reaches seq 0.
  - **Tests (uncounted)**: `/workspace/iscc-monitor/internal/store/iscc_index_test.go`,
    `/workspace/iscc-monitor/internal/proofserve/records_test.go`.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — the `## HTML record list at
    /records` section (the `from == 0` overload trap + the three durable cursor-pagination lessons) and
    the `serveEntries` `seq >= LastSize` contract under `## Computed record-bytes HTTP surface`.
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — `## iscc_index writer/reader` (the
    `int64(seq)` cast symmetry; store-stays-a-leaf invariant; the `COUNT(*)` total convention).
  - `/workspace/iscc-monitor/internal/proofserve/handler.go:349-364` — the exact `serveEntries`
    `LastSize == 0 → 404` / `seq >= size → 404` cap to mirror as the list ceiling.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go:671-744` — `serveRecords` (where #2 and #3
    live: `parseUint`/`int(n)` clamp at 691-698; the `OlderFrom = oldest - 1` chain at 713-733).
  - `/workspace/iscc-monitor/internal/store/iscc_index.go:98-141` — `ListRecords` (where #1 and the
    `if from > 0` overload at line 108 live).

## Not In Scope
- The **single-record page** (declaration / deletion / unknown schema) and re-pointing each `/records`
  row from `entries?index=` to it — that is the NEXT slice, not this one.
- The **certificate of inclusion** / proof-bundle assembler (re-engages the oracle gate) — later.
- Consolidating the 3x `overlayStatus`/`hubStatus` precedence into `internal/badge` (open `low` issue;
  the loop skips `low`). Do not refactor the overlay here.
- Changing `defaultPageSize` (50) / `maxPageSize` (200) values, or switching the seq cursor to OFFSET.
- Any change to `RecordProjections` / the `PollHub` ingest write path — the fix is read-side only. The
  projections legitimately sit above `LastSize` on a freeze; the LIST must filter, not the writer.
- Any schema change — keep `internal/store/schema.sql` byte-identical.

## Implementation Notes
Three defects, two files. The `from == 0` overload (#3) and the overflow clamp (#2) share a root in
`serveRecords` + `parseUint`; the `LastSize` ceiling (#1) is independent but in the same two files.

1. **`LastSize` ceiling (coverage honesty, ADR-0001).** `PollHub` writes projections for ALL
   `info.TreeSize` leaves via `ingestTiles → RecordProjections` BEFORE `checkConsistency` /
   `AdvanceAccepted`, so on a freeze/fault `iscc_index` holds rows with `seq >= LastSize`. Every other
   record route caps at `seq >= LastSize → 404` (`serveEntries` handler.go:361, plus `serveInclusion`,
   `serveVerify`); `ListRecords` is the only one that omits it. Add a ceiling param to `ListRecords`
   (e.g. `last uint64`) and apply `AND seq < last` to BOTH the windowed `SELECT` AND the `COUNT(*)`
   total (the total drives "showing N of TOTAL", so an uncapped total still lies). `serveRecords` passes
   `fs.LastSize`. The guard is exclusive — `seq < last` — because seq is 0-based and `LastSize` is a
   count, matching the `seq >= size` form. When `LastSize == 0` (followed-but-unpolled) the list is
   simply empty → the EXISTING 200 empty-state path renders, NEVER a 404 (preserve `serveBrowser`'s
   `LastSize == 0 → 200` honesty).

2. **Clamp page size before `int()` (anti-DoS).** handler.go:693-698 does `pageSize = int(n)` THEN
   `if pageSize > maxPageSize`. A huge `n` (e.g. `9223372036854775808`) wraps `int(n)` NEGATIVE, the
   `> 200` check misses it, and modernc SQLite reads a negative `LIMIT` as UNLIMITED → whole index
   (reviewer-confirmed: 10 rows for `n=-5`). Clamp while still `uint64`: `if n > maxPageSize { n =
   maxPageSize }` BEFORE the `int()` conversion, then `pageSize = int(n)`. ALSO bound `parseUint`
   (handler.go:807-819): `n = n*10 + uint64(c-'0')` wraps silently on overflow with no guard. Add an
   overflow rejection (return its existing error type once another digit would overflow `uint64`).
   Keep `parseUint`'s contract intact — empty / non-numeric → error (→ default), so bare `/records` and
   a bad value still never 400. `parseUint` is shared by `selectSeq` / `serveInclusion` / `serveEntries`,
   so ONLY ADD the overflow rejection; do not change its reject-non-digit / empty behaviour.

3. **Reach seq 0 (un-overload `from == 0`).** `from == 0` is both a cursor value and the "start at
   newest" sentinel (`iscc_index.go:108 if from > 0`). The older link emits `OlderFrom = oldest - 1`
   when a page ends at seq 1, so it emits `from=0`, which `ListRecords` reads as "newest" → the chain
   jumps back to the top and seq 0 is unreachable (reviewer-confirmed: `from=1&n=1`'s older link
   `from=0&n=1` shows seq 299). Un-overload it: prefer a 1-based cursor at the HTTP seam — carry the
   cursor as `from+1` so `0`/absent means "no cursor (start newest)" and `1` means "start at seq 0" —
   OR thread a separate `hasFrom bool` into `ListRecords`. Choose the option that keeps `ListRecords` a
   clean leaf signature; whichever you pick, the older-link emit, the newer-link emit, and the
   query-param parse must agree on the encoding. Verify the round trip: a page ending at seq 1 emits an
   older link that, when followed, shows seq 0 (not the newest page).

**Correctness rules in play:** coverage honesty (ADR-0001) — never imply a leaf is accepted that is
not; the M2 `>= LastSize` accepted-tree contract (the invariant EVERY record route follows). **Store
stays a leaf** — `ListRecords` must not gain `net/http` or `internal/logclient` deps (`go list -deps
./internal/store | grep '^net/http$'` stays empty; the package's own `.Imports` stay `context
database/sql embed errors fmt time` + the sqlite driver). Keep `go.mod` / `go.sum` /
`internal/store/schema.sql` byte-identical. Oracle gate is correctly N/A (pure HTML render of persisted
rows; no signature / RFC-6962 / Merkle / proof path) — do NOT skip the gate, just note it does not apply.

**Tests (make each new assertion mutation-proven non-vacuous, then revert the mutation):**
- Store: a fixture with projections at `seq >= last` (a frozen-hub-past-violation case) →
  `ListRecords(..., last)` returns ONLY `seq < last` rows AND `total` counts only `seq < last`.
  Mutation: drop the `AND seq < last` ceiling → the test FAILS, then reverted.
- proofserve: `?n=9223372036854775808` → at most `maxPageSize` rows render. Mutation: revert the
  pre-`int()` clamp → the test FAILS (whole index), then reverted.
- proofserve: an older-link walk that follows the older link from a small page down to seq 0 and
  asserts seq 0 appears. Mutation: revert the `from == 0` un-overload → the walk never reaches seq 0.
- Update the existing `TestListRecords*` (store) and `TestRecords*` (proofserve) callers to pass the
  new ceiling/cursor args; keep them green and their `DESC`-order / no-CDN / overlay asserts intact.

## Verification
- `mise run check` is green (build + vet + all packages `ok`; `gofmt -l .` empty).
- `go test -run TestListRecords ./internal/store` passes, including the new `seq < last` ceiling case;
  the ceiling-drop mutation FAILS that case (non-vacuous), then reverted.
- `go test -run TestRecords ./internal/proofserve` passes, including the new overflow-clamp case and
  the older-link-reaches-seq-0 case.
- `go list -deps ./internal/store | grep '^net/http$'` is empty (store stays a leaf).
- `git diff --stat` shows `internal/store/schema.sql`, `go.mod`, `go.sum` unchanged.
- Assertion: with a hub holding rows at `seq >= last`, `ListRecords` returns rows only where
  `seq < last`, and `total` counts only `seq < last`.
- Assertion: following the older link from a page that ends at seq 1 reaches a page containing seq 0.
- Assertion: `?n=9223372036854775808` renders at most `maxPageSize` rows, never the whole index.

## Done When
`mise run check` is green and all three defect-targeting assertions hold (only accepted leaves list,
the older chain reaches seq 0, and a hostile `n` is clamped to `maxPageSize`), each new assertion
mutation-proven non-vacuous — closing both open `normal` `[review]` record-list issues.
