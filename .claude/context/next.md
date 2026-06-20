# Next Work Package

## Step: Store-side freeze + violations seam (RecordViolation, Freeze)

## Goal
Add the typed `internal/store` CRUD the freeze path needs — persist a self-consistency
violation (`violations`) and set `frozen=1` (`follow_state`) — so the next step (the
three-trigger RFC-6962 consistency check) has a tested, leaf-only persistence seam to drive.
This directly advances the dominant unmet M1 Verify criterion (`violations.kind` + `frozen=1`
+ evidence survives restart) at the persistence layer, without yet pulling in merkle math,
new fixtures, or a network `go get`.

## Goal-fit (state → target gap)
M1's first Verify half (`origin`/`verifierKey`/single-poll) is met; the dominant remaining
half is *synthetic fork/shrink/equivocation → correct `violations.kind` + `frozen=1` +
exactly one alert + other hubs unaffected + evidence survives restart*. That full check needs
`transparency-dev/merkle` (a network `go get`) plus tiles fixtures — too large and too
dependency-heavy for one step. The smallest coherent slice that advances it *now*, with zero
new deps and no new fixtures, is the **store-side persistence seam** the consistency check
will drive: `RecordViolation` (writes `violations`) and `Freeze` (sets `frozen=1`). Both build
directly on the existing `violations`/`follow_state` tables and the established
`AdvanceFollowState` no-auto-unfreeze idiom — building on what exists, not skipping ahead into
merkle math.

## Scope
- **Create**: (none — extend the existing file)
- **Modify**:
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — add a `Violation` struct +
    `RecordViolation(ctx, Violation) (int64, error)` and `Freeze(ctx, hubID int64) error`.
    (This is the only non-test/doc file changed — 1 of ≤3.)
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — add the tests below
    (test file, not counted against the budget).
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/schema.sql` — `violations` columns
    (`hub_id, kind, detected_at, raw_a, raw_b, proof_json`) and `follow_state`
    (`hub_id, last_size, frozen, last_error`).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — existing idioms to match:
    `unixOrNil`, `%w`-wrapped errors, single open connection (`s.db.ExecContext` /
    `QueryRowContext`), `LastInsertId`, the `AdvanceFollowState` upsert.
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — the `openTemp(t)` /
    `countRows` helpers and the raw-`db.QueryRow` column-assertion style to mirror.
  - `/workspace/iscc-monitor/.claude/context/learnings.md` — "AdvanceFollowState upsert omits
    frozen … no auto-unfreeze (ADR-0006)" and "Zero time.Time → NULL convention (unixOrNil)".

## Not In Scope
- The three-trigger consistency check itself (fork/shrink/equivocation detection over
  `transparency-dev/merkle`) — that is the next step and needs the merkle dep + tiles fixtures.
- Wiring `RecordViolation`/`Freeze` into `follower.PollHub` or any caller — this step only
  adds and tests the store seam; no caller files change.
- The alert ("exactly one alert") mechanism — separate concern, a later step.
- Any `transparency-dev/*` dependency or `go get` (none is needed here; `go.mod` is untouched).
- An `Unfreeze` method — there is deliberately no auto-unfreeze (ADR-0006); do not add one.
- The `hub_keys` did:web cache write and the stale sb1-fixture refresh — independent step.
- The poll loop / single-writer goroutine wrapper — independent step.

## Implementation Notes
- `Freeze(ctx, hubID)` sets `frozen=1` on the hub's `follow_state` row and **must work whether
  or not a row already exists** (a hub can be frozen before its first verified advance). Use an
  upsert on the `frozen` column, mirroring `AdvanceFollowState`:
  `INSERT INTO follow_state (hub_id, frozen) VALUES (?, 1) ON CONFLICT(hub_id) DO UPDATE SET frozen = 1`.
  This complements `AdvanceFollowState` (which omits `frozen` from its `DO UPDATE SET`): an
  advance after a freeze keeps `frozen=1` (ADR-0006, no auto-unfreeze).
- `Violation` struct (mirror `CheckpointRecord`'s field/doc style): `HubID int64`,
  `Kind string` (one of `fork`/`shrink`/`equivocation`), `RawA []byte`, `RawB []byte`,
  `ProofJSON string`, `DetectedAt time.Time`. `RecordViolation` does a **plain** INSERT (no
  `ON CONFLICT` — `violations` has no UNIQUE constraint; re-detection is itself evidence) and
  returns the new `id` via `res.LastInsertId()`. Map `DetectedAt` through `unixOrNil` (zero →
  SQL NULL, matching the existing convention). `proof_json` is a TEXT column — store
  `ProofJSON` as a plain string (an empty string is fine; do not coerce to NULL).
- Keep `store` a **leaf**: add no imports beyond what `checkpoints.go` already uses
  (`context`, `database/sql`, `errors`, `fmt`, `time`). Do **not** import `internal/logclient`;
  `Kind` is a plain string the future caller supplies, exactly as `Status` rides on
  `CheckpointRecord` today.
- Relevant Correctness rule (learnings.md): "A self-consistency violation freezes, never
  crashes (ADR-0006) … persist both raw checkpoints + proof permanently, set `frozen=1` …
  no auto-unfreeze, survive restart, other hubs unaffected." Tests must pin the
  no-auto-unfreeze and other-hubs-unaffected properties at this layer; `raw_a`/`raw_b`/
  `proof_json` persistence is the "permanent evidence" part.
- Style: evergreen docstrings on the new struct + both methods; short, single-purpose methods;
  wrap every error with `%w`; no `t.Skip` / `//nolint` / swallowed errors / build tags.

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...` exit 0;
  `gofmt -l .` empty).
- `go test -count=1 -run 'TestRecordViolation|TestFreeze' ./internal/store` passes.
- `RecordViolation` test: register a hub via `UpsertHub`, insert
  `Violation{Kind:"equivocation", RawA:…, RawB:…, ProofJSON:…, DetectedAt:…}`; assert the
  returned `id > 0` and that a raw `SELECT kind, raw_a, raw_b, proof_json FROM violations
  WHERE id = ?` round-trips the exact `kind`, both raw blobs, and the proof JSON.
- `Freeze` no-prior-row: `Freeze(ctx, hubID)` on a hub with no `follow_state` row, then
  `FollowState(ctx, hubID).Frozen == true`.
- No-auto-unfreeze: `Freeze(ctx, hubID)` then `AdvanceFollowState(ctx, hubID, 99)`, then
  `FollowState(ctx, hubID)` has `Frozen == true` AND `LastSize == 99` (the advance updated the
  cursor but did not clear the freeze).
- Other-hubs-unaffected: freeze hub A, then `FollowState(ctx, hubB).Frozen == false`.
- `go list -deps ./internal/store` shows zero internal `iscc-monitor` deps (store stays a leaf).

## Done When
`mise run check` is green and the `TestRecordViolation`/`TestFreeze` tests pass, proving the
store can persist a violation (`kind` + both raw checkpoints + proof) and set/keep `frozen=1`
(no auto-unfreeze, other hubs unaffected) over `t.TempDir()` databases — the leaf-only
persistence seam the consistency-check step will drive.
