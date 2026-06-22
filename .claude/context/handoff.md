## 2026-06-22 — Stamp pending OTS roots against the calendar in the off-path OTSTick loop

**Done:** `OTSTick` now stamps not-yet-stamped pending rows (the empty-`OTSBytes` sentinel the poll
path writes) against the calendar via a new injected `Stamper` func seam BEFORE the upgrade step, off
the poll path — so a pending row can finally carry a real serialized OTS proof and the existing upgrade
closure stops being a structural no-op. `main.go` wires the production `Stamper` as a closure over
`otsclient.Stamp` + `otsclient.DefaultCalendarURL`, threaded `runOTSLoop → OTSTick`. A new minimal,
leaf-pure `store.MarkOTSStamped` mutator persists `OTSBytes`/`CalendarURLs` onto the existing row
without touching status (a fresh seam was required — `RecordOTS`'s `ON CONFLICT DO NOTHING` cannot
update an existing row's bytes, and `MarkOTSUpgraded`/`MarkOTSAttempted` change status/attempts).

**Files changed:**
- `internal/follower/otsloop.go`: added the `Stamper` func-seam type; added `stamper` param to
  `OTSTick`; added the `len(r.OTSBytes) == 0` stamp branch (build root key → call `Stamper` → persist
  via `MarkOTSStamped` → `continue`; on a stamp transport fault → `MarkOTSAttempted` back-off +
  log-and-continue, never abort/freeze; nil-`Stamper` tolerant). Updated package + `OTSTick` docstrings.
- `cmd/iscc-monitor/main.go`: added `stampFunc()` (the real `otsclient.Stamp` closure); threaded
  `follower.Stamper` through `runOTSLoop`'s signature + its `OTSTick` call + the `go runOTSLoop(...)`
  call site.
- `internal/store/ots.go`: added `MarkOTSStamped(ctx, hubID, treeSize, root, otsBytes, calendarURLs)`
  — plain UPDATE keyed on `(hub_id, tree_size, root)`, `RowsAffected`-ignored (absent = no-op),
  status untouched, empty `calendarURLs` → NULL via `nullStringOrNil`. Leaf-pure, no anchoring import.
- `internal/follower/otsloop_test.go` (test): `seedPendingOTS` now seeds pre-stamped rows (non-empty
  `OTSBytes`) so the upgrade-path tests skip the stamp branch + assert it via `noStamper(t)`; added
  `seedNotStampedOTS`, `fakeStamper`, `noStamper` helpers; added `TestOTSStampThenUpgrade` (full
  stamp→upgrade arc, the pending→confirmed transit) and `TestOTSStampBacksOff` (stamp fault → back-off,
  never aborts/confirms, Upgrader not reached). Existing `TestOTSTick*` calls gained the stamper arg.
- `internal/store/ots_test.go` (test): added `TestMarkOTSStamped` + `TestMarkOTSStampedAbsentAndEmptyCalendars`.

**Verification:** `mise run check` → GREEN (build + vet + test, all 23 packages). Per criterion:
- [x] `mise run check` green; `gofmt -l .` (excl. `cauldron/`) clean.
- [x] `go test -count=1 -run TestOTS ./internal/follower` passes (new stamp tests + existing
  `TestOTSTick*`/`TestOTSBackoff`).
- [x] follower anchoring deps == 0 (`go list -deps ./internal/follower | grep -c -e internal/otsclient
  -e internal/ots -e nbd-wtf/opentimestamps` → 0; `Stamper` is the boundary, like `Upgrader`).
- [x] store coupling deps == 0 (`go list -deps ./internal/store | grep -c -e internal/follower
  -e internal/otsclient -e ^net/http$` → 0; store stays a leaf).
- [x] Mutation (reverted): disabling the `len(r.OTSBytes) == 0` stamp branch → `TestOTSStampThenUpgrade`
  + `TestOTSStampBacksOff` FAIL (row never stamped → never confirms; stamp fault never surfaces);
  restored → green, file byte-identical.
- [x] Mutation (reverted): no-op'ing `MarkOTSStamped`'s UPDATE → `store.TestMarkOTSStamped` AND
  follower `TestOTSStampThenUpgrade` FAIL (OTSBytes stays empty, never confirms); restored → green.
- [x] `go.mod`/`go.sum` untouched; `go mod tidy -diff` clean.
- [x] `internal/follower/follower.go` byte-unchanged (the deliberate deviation below).

**Next:** The `.ots` HTTP route (serve `OTSForRoot`'s confirmed proof bytes at the canonical path so a
client can `ots verify` the served `.ots`) — this plus certificate §5 BITCOIN ANCHOR (`HasClause5`,
gated on a confirmed row) are the remaining halves of the OTS Verify-closer; both now have a real path
to confirmed rows. Note the end-to-end pending→confirmed transit is exercised in tests with injected
seams; the *production* live-calendar transit still needs a real calendar round-trip (not run in
`go test`, by design) — that is an integration/CI concern, not a code gap.

**Notes:**
- **Deliberate, rule-driven deviation from the prior `review` handoff `**Next:**`** (which said "wire
  `otsclient.Stamp` into `follower.stampRoot`"): stamping was placed in the off-path `OTSTick`, NOT in
  `stampRoot`/`PollHub`. A synchronous calendar HTTP round-trip on the poll path violates the
  always-loaded Correctness rule "OTS never blocks the follower" (ADR-0004). `next.md` explicitly
  scoped it this way; `follower.go`/`stampRoot` is byte-unchanged. The poll path still writes pending
  rows with the empty-`OTSBytes` sentinel; the off-path loop fills them in.
- **A new store mutator was genuinely required** (not riding `RecordOTS`): `RecordOTS` is
  `ON CONFLICT DO NOTHING`, so re-calling it never updates an existing pending row's bytes;
  `MarkOTSUpgraded` flips status to confirmed; `MarkOTSAttempted` only touches attempts/next_retry.
  `MarkOTSStamped` is the minimal seam that persists the stamp without changing status. This makes 3
  non-test source files changed (otsloop.go, main.go, ots.go) — at the ≤3 budget, not over.
- **`continue`-after-stamp choice** (not fall-through to upgrade): the row is stamped this tick and
  upgraded the NEXT tick, mirroring the back-off flow's "re-surface next tick". Simpler and matches the
  loop's existing per-row discipline; documented in the `OTSTick` docstring. (A fresh-stamped pending
  sequence is never Bitcoin-confirmed immediately, so falling through to upgrade the same tick would
  just back it off anyway.)
- **nil-`Stamper` tolerated** (skip stamping, leave the row empty), mirroring the nil-`Logger`/`Metrics`
  discipline — a bare `OTSTick` call path stays well-defined; documented in the docstring. The
  no-pending test passes a nil stamper to exercise this.
- Oracle gate correctly N/A this slice: opaque pending→confirmed over an already-fsck-verified root; no
  signature/RFC-6962/Merkle/did:web/proof code touched. The `ots verify` oracle (bundled `examples/*.ots`
  + `internal/ots`) is unchanged. All tests run fully offline (injected `Stamper`/`Upgrader`).
