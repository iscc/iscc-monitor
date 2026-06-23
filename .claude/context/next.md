# Next Work Package

## Step: Tie the dossier §3 observed-time to the accepted `last_size` checkpoint row (frozen-hub honesty fix)

## Advances
Closes the open `normal` issue "Dossier §3 latest-checkpoint size and observed-time can come from
different rows on a FROZEN hub" and advances the **M-UI** Verify criterion:

> "the coverage window (`monitored_since` size + RFC-3339 time, or an explicit … 'no coverage yet')
> shows for **every** hub on the index + dossier and a pre-coverage state never renders as a guarantee
> (ADR-0001)"

…and the M-UI honesty bar more broadly ("never imply a guarantee the data does not support"). Right now
§3 renders the ACCEPTED size (`.LastSize`) paired with the time of a DIFFERENT (rejected, higher-tree-size)
checkpoint on a frozen hub — an accepted size with a rejected timestamp. This is the recurring SSR-honesty
gap the always-loaded learnings flag, and it is a concrete, code-closable correctness fix — NOT a design
pass. (Contrast: the sibling §1 "resolved"-wording `normal` and the realm-index Anchor `normal` both
explicitly need a design decision first, per their issues + `learnings/dossier.md`.)

**Why this and not the handoff's suggestion:** the handoff's `**Next:**` points at "M-API contract-fidelity
slice 4 (the phantom `verify` `index` param + the `checkpoint` media-type doc fixes)". That work is ALREADY
DONE in the served doc — I verified `internal/openapi/openapi.{yaml,json}`: the `verify` op carries only
`Domain` + `iscc_id` (no `index`), `/checkpoint`'s `200` is `application/octet-stream`, and `/healthz`
documents both `200` and `503`. Those four M-API entries in `issues.md` are stale bookkeeping awaiting a
prune (state.md §M-API confirms this), not open code work. So the genuinely-open, code-closable `normal` is
this §3 honesty fix.

## Goal
Make the dossier §3 "latest checkpoint" SIZE and TIME always describe the SAME checkpoint row, so a frozen
hub no longer shows the accepted size next to a rejected checkpoint's timestamp.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/store/hubs.go` — change the §3 `observed_at` correlated subselect in `ListHubs` so it reads
    the `observed_at` of the checkpoint row whose `tree_size = f.last_size` (the accepted size §3 renders),
    not the newest-by-`tree_size` row. Update the `CheckpointObserved` doc comment (lines ~27-32) and the
    in-query comment block (lines ~56-62) so they describe the new "tied to the accepted size" semantics
    (evergreen — describe the current state).
  - `internal/store/hubs_test.go` *(test — does not count against the ≤3 budget)* — add a frozen-hub
    regression test asserting §3 time tracks the accepted-size checkpoint, mutation-pinned.
- **Reference**:
  - `.claude/context/learnings/dossier.md` (§3 size/time-decouple rule — the durable fix is "select
    `observed_at` for the row whose `tree_size = f.last_size`"; the §1-resolved + realm-Anchor design-blocked
    siblings to NOT touch)
  - `.claude/context/learnings/store.md` (single-writer leaf; NULL-safe `sql.NullInt64` read-back; the
    §3 `tree_size DESC` subselect is deliberately distinct from `ListCheckpoints`' chronological order)
  - `internal/store/hubs.go` lines 63-124 (the `ListHubs` query + scan to edit)
  - `internal/store/checkpoints.go` — `RecordCheckpoint` (line 113), `AdvanceAccepted` (223), `Freeze`
    (391-399): the writers the test seeds a frozen-hub fixture with
  - `internal/store/hubs_test.go` lines 90-156 (`TestListHubsCheckpointAndAnchorHeight` — the existing §3
    test to keep green and the fixture pattern to copy)
  - `internal/dossier/handler.go` lines 290-320 + `observedTime` — confirms `CheckpointObserved` →
    `ObservedTime` is the only §3 consumer (the fix is store-side; the renderer is unchanged)

## Not In Scope
- The §1 "Key resolved from did:web:…" wording on the `unresolvable` path — a SEPARATE open `normal` that
  the issue + `learnings/dossier.md` explicitly say needs a DESIGN PASS (mockup-specified copy); do not touch.
- The realm-index `/` Anchor per-hub-vs-per-checkpoint honesty `normal` — also design-blocked; leave it.
- Pruning the 4 stale M-API `normal` issue entries — that is `update-state`/`review` bookkeeping, not an
  advance code change.
- The `seq`-index / `user_version`-guard-hoist `low`s — not triggered by this edit (no `iscc_index` schema
  change, no `Open` change).
- Any change to `ListCheckpoints` (§5) or the §5 observation-log loop — §5 already correctly skips
  non-increasing pairs; this step touches ONLY the §3 single-row subselect.
- Do NOT add a column or migration — this is a query-shape change INSIDE the existing `ListHubs` statement,
  no `schema.sql` / `user_version` edit.

## Implementation Notes
- **The fix (one subselect):** in `ListHubs` (`internal/store/hubs.go:69-70`) change
  ```sql
  (SELECT c.observed_at FROM checkpoints c WHERE c.hub_id = h.hub_id
   ORDER BY c.tree_size DESC, c.id DESC LIMIT 1)
  ```
  to tie the row to the accepted size:
  ```sql
  (SELECT c.observed_at FROM checkpoints c WHERE c.hub_id = h.hub_id
   AND c.tree_size = f.last_size
   ORDER BY c.id DESC LIMIT 1)
  ```
  `f` is the already-joined `follow_state` (LEFT JOIN, alias `f`). Keep `id DESC LIMIT 1` so a re-observed
  same-size row is deterministic. The scan path (`sql.NullInt64 observedAt` → `time.Unix`) is UNCHANGED —
  only the row the subselect selects changes.
- **Verified-hub path (no regression):** for a VERIFIED hub `f.last_size` equals the accepted tree size and
  the accepted checkpoint row carries that size, so the subselect picks the same row it picks today →
  `CheckpointObserved` is identical. The existing `TestListHubsCheckpointAndAnchorHeight` (accepted size 42,
  `last_size` 42 via `AdvanceAccepted`) must still pass unchanged — confirm it does.
- **NULL-safety (honest fallback):** when `f.last_size` is NULL (never-polled hub) OR no checkpoint row has
  `tree_size = last_size`, the subselect yields SQL NULL → `observedAt.Valid == false` → zero `time.Time` →
  the renderer's `observedTime` shows "observed time unknown". That is the honest state, consistent with the
  existing `bare`-hub assertion (`CheckpointObserved.IsZero()`), which must still hold.
- **Frozen-hub regression test (the load-bearing addition):** seed one hub, `AdvanceAccepted` at the
  accepted size (e.g. `TreeSize: 100`, `ObservedAt: tAccepted`), then `RecordCheckpoint` a rejected
  higher-size row (`TreeSize: 200`, `ObservedAt: tRejected` — a LATER, distinct instant) WITHOUT advancing
  `last_size` (this mirrors `follower.freeze`, which `RecordCheckpoint`s the contradictory checkpoint but
  never writes `last_size`); optionally `Freeze` the hub. Assert `ListHubs` reports
  `CheckpointObserved == tAccepted` (the accepted-size row's time), NOT `tRejected`. Use distinct
  `time.Unix` instants so the assertion is sharp.
- **Mutation check (state it in the advance):** reverting the query to
  `ORDER BY c.tree_size DESC, c.id DESC LIMIT 1` (dropping the `AND c.tree_size = f.last_size`) must make the
  new test FAIL — it would report `tRejected`.
- **Store-leaf discipline (always-loaded rule):** stays a pure read in `internal/store` — no new import, no
  `net/http`, plain Go types out. Do not move §3 logic into the dossier view; the renderer is correct, the
  query was picking the wrong row.
- **Relevant Correctness rule:** learnings.md always-loaded "Coverage honesty (ADR-0001) … state guarantees
  *from coverage start*" + the SSR-honesty rule (never render data the store cannot support). The frozen-hub
  Exhibit already shouts "do not trust new state", but §3 must still not pair an accepted size with a
  rejected time — that is a false specific claim.
- **Oracle/conformance gate: N/A** — touches no signature / RFC-6962 / Merkle / did:web / proof path; it is a
  pure `iscc_index`/`checkpoints` read-shape change. State the N/A in the advance.

## Verification
- `mise run check` is green (build + vet + test across all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestListHubs ./internal/store` passes — both the existing
  `TestListHubsCheckpointAndAnchorHeight` (verified-hub + bare-hub paths unchanged) and the new frozen-hub
  test.
- New frozen-hub test is mutation-proven load-bearing: reverting the §3 subselect to
  `ORDER BY c.tree_size DESC, c.id DESC LIMIT 1` (without `AND c.tree_size = f.last_size`) makes the new test
  FAIL (it reports the rejected checkpoint's time).
- Store leaf purity intact: `go list -deps ./internal/store | grep '^net/http$'` is empty.
- Assertion: for a frozen hub with accepted `tree_size = last_size = 100 @ tAccepted` and a recorded
  rejected `tree_size = 200 @ tRejected`, `ListHubs(...)[hub].CheckpointObserved == tAccepted`.

## Done When
`mise run check` is green, the new frozen-hub test passes and is mutation-proven, the existing verified-hub
and bare-hub assertions still hold, and the dossier §3 observed-time is read from the checkpoint row whose
`tree_size` equals the accepted `last_size` — closing the §3 size/time-decouple `normal`.
