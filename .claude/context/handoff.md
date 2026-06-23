## 2026-06-23 — Record-list pager parity (part-2b): top+bottom "seq X–Y of Z" pager + drop the off-mockup Status row

**Done:** Reworked the `/records` log-browser record list to the Log-Browser mockup's pager region — a
top pager (above the ledger card) and a bottom pager (below it), each a three-slot row (Newer left,
center label, Older right) with the end affordances rendered as non-interactive `<span class="pager-disabled">`
(the no-JS equivalent of the mockup's opacity-disabled button). The top pager's center is the pure-derived
`seq <top> – <bottom> of <total>` range label (newest-first, so top is the LARGER seq); the bottom pager's
center is the append-only footnote. Dropped the off-mockup Status-badge row (and its now-dead CSS); the
overlaid status now reaches only the `.ledger` element's `data-status` frozen-tint attribute. Closes the
last code-closable half of the lone `critical` — only the human M-UI exit sign-off remains.

**Files changed:**
- `internal/proofserve/handler.go`: added `RangeTop`/`RangeBottom uint64` to `recordsData` and set them
  inside the existing `if len(records) > 0 {` block in `serveRecords` from the already-fetched page slice
  (`records[0].Seq` / `records[len-1].Seq`) — zero new store reads, no new query.
- `internal/proofserve/records.html`: replaced the single bottom-only "showing N of M" pager + the
  `.ledger-status` Status row with the top+bottom pager region (`.pager-top` seamed into the ledger card,
  `.pager-bottom` below); removed the dead `.ledger-status`/`.row-label` CSS AND the now-dead
  `.hub-status-badge` CSS block (the badge partial is no longer invoked on this page); kept the standalone
  coverage-honesty footnote; updated the `.ledger[data-status=frozen]` comment to reflect the dropped badge.
- `internal/proofserve/records_test.go` (test-only, off the ≤3 budget): reworked
  `TestRecordsRendersInMemoryStatus` to assert the overlay reaches the `.ledger` `data-status` attribute
  while the dropped Status-badge row markers (`row-label">Status`, `class="hub-status-badge"`,
  `>Unresolvable<`) are ABSENT; added `TestRecordsPagerRangeAndTopBottom` (hardcoded `seq 4 – 0 of 5` /
  `seq 4 – 3 of 5` range + top/bottom pager bracketing) and `TestRecordsPagerEndsDisabled` (disabled-span
  ends). The existing `TestRecordsPagination` / `TestRecordsOlderLinkReachesSeq0` / `olderHref` markers
  needed NO change — the live older anchor still renders `older &rarr;</a>` and the `?from=…&amp;n=…` chain
  is unchanged (the rework only moved the markup, not the cursor math).

**Verification:** `mise run check` → GREEN (build + vet + test across all 30 packages; `gofmt -l .` empty).
- [x] `go test -count=1 -run TestRecords ./internal/proofserve` → all 15 PASS (existing pagination-chain
  tests green under the reworked pager).
- [x] NEW pager-parity test (`TestRecordsPagerRangeAndTopBottom`) asserts the hardcoded `seq 4 &ndash; 0 of 5`
  range + a top pager preceding and a bottom pager following the ledger; append-only + coverage-honesty
  clauses both present.
- [x] NEW dropped-Status-row test (in `TestRecordsRendersInMemoryStatus`) asserts the Status-row markers are
  absent while `class="ledger" data-status="unresolvable"` (frozen-tint attribute) survives.
- [x] `go list -deps ./internal/proofserve | grep -qx internal/metrics` → ABSENT (no new dep).
- [x] `go.mod` / `go.sum` byte-identical (template/view-model-only change).
- [x] **Mutations re-proven** (restored after each, suite green): swap `RangeTop`/`RangeBottom` → range test
  FAILS; rename `pager pager-bottom` class → "body missing the bottom pager" FAILS; re-add the Status-badge
  row → "still carries the dropped Status-badge row marker" FAILS; force top-pager older always-live → ends
  test FAILS.
- Oracle/conformance gate: **N/A** — pure SSR template + view-model rework, no signature, RFC-6962, Merkle,
  did:web, fsck, or proof path touched; store stays a leaf, no new read.

**Next:** Request the **human M-UI exit sign-off** — this was the final code slice gating it; the no-JS
navigation chain `/` → dossier → record list → single record → cert/back is traversable both ways and the
record-list pager region now matches the Log Browser mockup. After sign-off, the masthead-identity const
consolidation (the tracked `low`, now 4× duplicated across dashboard/dossier/certificate/proofserve) is the
next milestone-fresh candidate.

**Notes:**
- **Dead partial association kept (harmless):** `recordsTmpl` still parses `badge.Source` into its set even
  though the body no longer invokes `{{template "hubStatusBadge" .}}`. Left as-is to stay in scope (no
  dep/behavior change — `internal/badge` is still imported for `badge.Label`); removing it is a trivial
  cleanup if review prefers. Likewise `serveRecords` still computes the `.Label` field (per `next.md`: "KEEP
  the `Status`/`Label` fields") though the template only reads `.Status` now.
- **Removed the `.hub-status-badge` CSS block** (a small deviation from `next.md`, which scoped only
  `.ledger-status`/`.row-label`): that block solely styled the badge partial this slice stops rendering, so
  it is genuinely dead code. Leaving it would be misleading; it uses the unquoted `[data-status=…]` selector
  so removing it cannot affect the negative `data-status="verified"` assert either way.
- **Empty-state case renders no pagers** — both pagers are gated on `{{if .Records}}`, so an empty index
  shows just the `.ledger` empty state + the coverage footnote (`TestRecordsEmpty` green). The `.pager-top +
  .ledger` CSS seam (squared corners, no top border) only applies when the top pager is present.
- **No `&ndash;` escaping surprise:** the en-dash entity is literal template text, so `html/template`
  passes it through verbatim (`seq 4 &ndash; 0 of 5`); the tests assert that exact byte form.
- The `from=1&n=2` window (seqs 1..0) is the oldest-page fixture for the disabled-Older end; a 5-record
  mirror at full `n=50` disables BOTH ends (the whole index is one page), so the ends test deliberately uses
  `n=2` for the newest-page case.
