# Next Work Package

## Step: Finish the dossier §3 honesty fix — `ORDER BY c.id ASC` so a same-size fork reads the accepted row's time

## Advances
M-UI Verify (coverage / checkpoint honesty): "the coverage window (`monitored_since` size + RFC-3339
time …) shows for **every** hub on the index + dossier and a pre-coverage state never renders as a
guarantee (ADR-0001)". Concretely it closes the still-open `normal` **"Dossier §3 observed-time still
tracks the REJECTED row for a same-size FORK violation (equivocation/higher-size case CLOSED)"**
(`issues.md`) — the remainder the prior increment (`820a831`) opened against itself: it closed the
equivocation / higher-tree-size decouple but left the same-size fork case, where §3 pairs the accepted
`last_size` with the *rejected* fork checkpoint's later timestamp. This is the review handoff's recorded
`**Next:**` and the sole code-closable convergence move; every other open `normal` is design- or
human-blocked.

## Goal
Make the hub-dossier "§3 Latest checkpoint" observed-time read the ACCEPTED checkpoint's time on a frozen
hub even when the violation is a same-size fork, so the trust document never pairs the accepted `(size)`
with a rejected checkpoint's timestamp. This completes the §3 size/time-honesty `normal` end-to-end.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/store/hubs.go` — in the §3 `observed_at` correlated subselect (currently
    `AND c.tree_size = f.last_size ORDER BY c.id DESC LIMIT 1`, line ~77), change `ORDER BY c.id DESC` →
    `ORDER BY c.id ASC`. Update the `HubSummary.CheckpointObserved` field doc (lines ~27-34) and the
    `ListHubs` doc comment (lines ~62-69) so they state the tiebreak rationale (evergreen — the accepted
    row at a size is the EARLIEST `id`; a later same-size row is a fork's rejected evidence). One source file.
  - `internal/store/hubs_test.go` *(test — does not count against the ≤3 budget)* — add a same-size-fork
    regression test, mutation-pinned.
- **Reference**:
  - `.claude/context/learnings/store.md` — the TRAP bullet (lines ~173-179: "`id DESC` is BACKWARDS for a
    same-size FORK … the fix is `id ASC`"), the `RecordCheckpoint` dedupe bullet (lines ~106-109), and the
    coverage set-once / leaf-purity notes.
  - `.claude/context/learnings/dossier.md` — the §3 size/time-decouple note (Read before touching §3 honesty;
    confirms the §1-resolved + realm-Anchor siblings are DESIGN-BLOCKED, NOT to touch).
  - `internal/store/hubs_test.go:158-226` — the existing `TestListHubsFrozenObservedTracksAcceptedSize`
    (higher-size case) to mirror its shape (`UpsertHub` → `AdvanceAccepted` → `RecordCheckpoint` → `Freeze`).
  - `internal/store/checkpoints.go:108-141` (`RecordCheckpoint` `ON CONFLICT(hub_id,tree_size,root) DO
    NOTHING`) and `:143-166` (`CheckpointAt` `ORDER BY rowid LIMIT 1` — the "earliest row is the accepted
    one" precedent this change matches).

## Not In Scope
- Do NOT touch the §3 renderer (`internal/dossier`), the `ListCheckpoints` §5 observation-log query, or the
  `Anchor` / `AnchorHeight` subselects in `ListHubs` — only the `observed_at` subselect's `ORDER BY` changes.
- Do NOT touch the design-blocked `normal`s: the §1 "Key resolved from did:web:…" wording on the
  `unresolvable` path, the realm-index `/` per-hub-vs-per-checkpoint Anchor honesty, the WASM cross-origin
  signature half, the proofserve-trio masthead identity.
- Do NOT consolidate the 3x hub-status overlay precedence, the masthead-identity fallback consts, or any
  other `low` locality issue — skipped by the loop.
- Do NOT add an index, column, migration, or `user_version` change — this is a query-shape change inside the
  existing `ListHubs` statement, no DDL.
- Pruning the stale resolved issue entries (the realm-index hero, the 4 M-API entries) is
  `update-state`/`review` bookkeeping, not an advance code change.

## Implementation Notes
- **The fix is one token.** In `ListHubs` (`internal/store/hubs.go:76-77`) the §3 subselect today reads
  ```sql
  (SELECT c.observed_at FROM checkpoints c WHERE c.hub_id = h.hub_id
   AND c.tree_size = f.last_size ORDER BY c.id DESC LIMIT 1)
  ```
  Change `ORDER BY c.id DESC` → `ORDER BY c.id ASC`. Nothing else in the query, the scan path
  (`sql.NullInt64 observedAt` → `time.Unix`), or the NULL-safe fallback changes.
- **Why `id ASC` is correct (taxonomy, `learnings/store.md`):** a re-observed ACCEPTED checkpoint (same
  root) is deduped by `RecordCheckpoint`'s `ON CONFLICT(hub_id, tree_size, root) DO NOTHING`, so the ONLY
  way two rows share `tree_size = f.last_size` is a FORK (different root). The accepted row was recorded
  FIRST → it has the EARLIEST `id`; the fork's contradictory row has a LATER `id`. So `ORDER BY c.id ASC
  LIMIT 1` deterministically selects the accepted row — the same selection `store.CheckpointAt` already
  makes with `ORDER BY rowid LIMIT 1`. `id DESC` is exactly backwards: it picks the later fork row.
- **No regression on the existing higher-size test.** `TestListHubsFrozenObservedTracksAcceptedSize`
  records its rejected checkpoint at `tree_size = 200 ≠ last_size = 100`, so the `AND c.tree_size =
  f.last_size` clause already excludes it — only one row matches the subselect and ordering is irrelevant
  there. It must stay green; run it to confirm. The verified-hub and bare-hub assertions in
  `TestListHubsCheckpointAndAnchorHeight` likewise see a single matching row (or none → NULL) and are
  unaffected.
- **New regression test** (e.g. `TestListHubsFrozenObservedTracksAcceptedSameSizeFork`): mirror the
  existing frozen test but make the contradictory checkpoint a SAME-SIZE fork —
  `UpsertHub` → `AdvanceAccepted{TreeSize: 100, Root: acceptedRoot, ObservedAt: tAccepted}` →
  `RecordCheckpoint{TreeSize: 100, Root: forkedRoot (DIFFERENT bytes), ObservedAt: tForkRejected}` with
  `tForkRejected > tAccepted` (so the assertion is sharp) → `Freeze`. The two roots MUST differ, else
  `ON CONFLICT(hub_id, tree_size, root)` dedupes and no fork row exists. Assert `frozen.Frozen`,
  `frozen.LastSize == 100`, and `frozen.CheckpointObserved.Equal(tAccepted)` (NOT `tForkRejected`). Use
  distinct 32-ish-byte root literals exactly as the existing test does.
- **Mutation-proof the test (state it in the test docstring + the advance).** With `id ASC` it passes;
  reverting the subselect to `ORDER BY c.id DESC` makes the new test report `tForkRejected` and FAIL while
  the existing higher-size test stays green; restoring `ASC` makes both pass.
- **Store-leaf discipline (always-loaded rule):** stays a pure read in `internal/store` — no new import, no
  `net/http`, plain Go types out. The renderer (`internal/dossier`) is correct; the query was picking the
  wrong row.
- **Relevant Correctness rule:** learnings.md always-loaded **Coverage honesty (ADR-0001)** — "state
  guarantees from coverage start; never imply a guarantee the data does not support". Pairing the accepted
  size with a rejected checkpoint's timestamp is exactly that dishonest pairing; `id ASC` removes it. The
  frozen Exhibit still shouts "do not trust new state" above §3, but §3 must not assert a false specific time.
- **Oracle/conformance gate: N/A** — touches no signature / RFC-6962 / Merkle / did:web / proof path; it is
  a pure `checkpoints` read-order change. State the N/A in the advance.

## Verification
- `mise run check` is green (build + vet + test across all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestListHubs ./internal/store` passes — the new same-size-fork test plus the
  existing `TestListHubsFrozenObservedTracksAcceptedSize`, `TestListHubsCheckpointAndAnchorHeight`,
  `TestListHubsAnchorStatus` all green.
- Mutation check: reverting `internal/store/hubs.go`'s §3 subselect `ORDER BY c.id ASC` → `ORDER BY c.id
  DESC` makes the new same-size-fork test FAIL (reports the rejected time) while the higher-size test stays
  green; restoring `ASC` makes both pass.
- Store leaf purity intact: `go list -deps ./internal/store | grep '^net/http$'` is empty.
- Assertion: for a frozen hub with accepted `tree_size = last_size = 100 @ tAccepted` and a recorded
  SAME-SIZE fork `tree_size = 100` (different root) `@ tForkRejected`,
  `ListHubs(...)[hub].CheckpointObserved == tAccepted`.

## Done When
`mise run check` is green, the new mutation-proven same-size-fork test passes asserting §3 observed-time
tracks the accepted row, the existing higher-size / verified-hub / bare-hub assertions still hold, and
reverting the `id ASC` tiebreak makes the new test fail — closing the §3 size/time-decouple `normal`.
