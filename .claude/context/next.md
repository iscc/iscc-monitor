# Next Work Package

## Step: Collapse accepted-checkpoint advancement into one store-owned transaction (`AdvanceAccepted`)

## Goal
Move the "advance accepted state" invariant from three caller-sequenced follower writes
(`RecordCheckpoint` → `SetCoverage` → `AdvanceFollowState`) into one transactional store method
`AdvanceAccepted`, so ordering and partial-write atomicity live at the storage boundary (ADR-0005
single-writer locality). Closes the open `normal` issue "Accepted checkpoint advancement is three
caller-sequenced store writes".

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/store/checkpoints.go` — add `func (s *Store) AdvanceAccepted(ctx context.Context, c
    CheckpointRecord) error` that opens one `s.db.BeginTx`, performs the three writes (record-checkpoint
    dedupe-insert, set-once coverage UPDATE, advance follow cursor upsert) against the `*sql.Tx`, and
    commits; rolls back on any error.
  - `internal/follower/follower.go` — replace the three sequential calls at lines 217-227
    (`RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState`) with a single `st.AdvanceAccepted(ctx, rec)`;
    the `rec store.CheckpointRecord` already built at follower.go:209-216 carries
    `HubID`/`TreeSize`/`Root`/`Raw`/`ObservedAt`.
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — the existing `RecordCheckpoint`
    (ON CONFLICT dedupe), `SetCoverage` (guarded set-once UPDATE), `AdvanceFollowState` (upsert), and the
    `unixOrNil` helper to reuse verbatim inside the tx.
  - `/workspace/iscc-monitor/internal/store/sqlite.go` — `Store{db *sql.DB}`, `SetMaxOpenConns(1)`,
    WAL + `busy_timeout=5000` (ADR-0005/0007 single-writer discipline the tx rides on).
  - `/workspace/iscc-monitor/internal/store/schema.sql` — `checkpoints UNIQUE(hub_id, tree_size, root)`,
    `hubs.monitored_since_{size,time}`, `follow_state(hub_id PRIMARY KEY, last_size, frozen)`.
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — the `openTemp(t)` helper + the
    assert-on-observable-rows style to mirror for the new test.
  - `/workspace/iscc-monitor/internal/follower/follower.go` — the verified-advance block (lines 209-249)
    is the call site; only lines 217-227 change.

## Not In Scope
- Do NOT delete or change the signatures of `RecordCheckpoint`, `SetCoverage`, or `AdvanceFollowState`:
  they stay public and are still used directly as seed helpers by `internal/follower/*_test.go`,
  `cmd/iscc-monitor/main_test.go`, and the follower's frozen-path / equivocation seeding.
- Do NOT touch the freeze path, the `cacheHubKey`/`fsckMirror` calls (follower.go:233-247), `ingestTiles`,
  or the second `RecordCheckpoint` caller in the test-only seeding code.
- Do NOT fold the second `normal` issue (the `widthForP` / tile-writer `p`-vocabulary unification) into
  this step — that is a separate slice.
- Do NOT alter `schema.sql`, the consistency-check logic, or the `logclient` package.

## Implementation Notes
- This is the **first** `BeginTx`/`*sql.Tx` use in the repo (verified: `grep -rn BeginTx internal/` is
  empty). Pattern: `tx, err := s.db.BeginTx(ctx, nil)`; guard with `defer func() { _ = tx.Rollback() }()`
  (after a successful `Commit`, `Rollback` returns `sql.ErrTxDone`, which is safe to ignore); return
  `tx.Commit()` last. Wrap every error with `%w` and a `store.AdvanceAccepted:` prefix matching the
  file's convention.
- Port the three SQL statements **byte-for-byte** from the existing methods, swapping `s.db.ExecContext`
  → `tx.ExecContext`:
  - checkpoint: `INSERT INTO checkpoints (hub_id, tree_size, root, raw, observed_at) VALUES (?,?,?,?,?)
    ON CONFLICT(hub_id, tree_size, root) DO NOTHING` with `c.HubID, int64(c.TreeSize), c.Root, c.Raw,
    unixOrNil(c.ObservedAt)`. Idempotent re-poll comes for free from `ON CONFLICT … DO NOTHING`;
    `AdvanceAccepted` needs neither the inserted id nor the inserted-bool, so drop `RecordCheckpoint`'s
    read-back-on-conflict branch and discard the `Result`.
  - coverage: `UPDATE hubs SET monitored_since_size = ?, monitored_since_time = ? WHERE hub_id = ? AND
    monitored_since_size IS NULL` with `int64(c.TreeSize), unixOrNil(c.ObservedAt), c.HubID`. The
    `IS NULL` guard keeps coverage **set-once** (ADR-0001); a re-poll is a silent no-op, never an error
    — do NOT inspect `RowsAffected`.
  - follow cursor: `INSERT INTO follow_state (hub_id, last_size) VALUES (?, ?) ON CONFLICT(hub_id) DO
    UPDATE SET last_size = excluded.last_size` with `c.HubID, int64(c.TreeSize)`. Omitting `frozen` from
    the conflict update preserves ADR-0006 no-auto-unfreeze (the follower already short-circuits a frozen
    hub before this path, but keep the SQL identical).
- `c.TreeSize` is the single source for both the coverage size and the follow cursor — in the old
  three-call sequence both used `info.TreeSize`, which equals `rec.TreeSize`.
- Correctness rules from `learnings.md` this step must respect: **Coverage honesty (ADR-0001)** —
  coverage start is immutable / set-once (the `IS NULL` guard); **SQLite single writer (ADR-0005/0007)** —
  the tx runs on the one capped connection, opens no second connection; **freeze no-auto-unfreeze
  (ADR-0006)** — never write `frozen` here.
- The follower edit is purely mechanical: keep the `rec store.CheckpointRecord{...}` already at
  follower.go:209-216, then `if err := st.AdvanceAccepted(ctx, rec); err != nil { return status,
  fmt.Errorf("follower.PollHub: hub %d: advance accepted: %w", hubID, err) }`. Leave the following
  `cacheHubKey`/`fsckMirror`/`recordVerdict` calls unchanged.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestAdvanceAccepted ./internal/store` passes — a new store-level test that
  asserts: (1) one `AdvanceAccepted` records the checkpoint row, sets `monitored_since_size`, and sets
  `follow_state.last_size`; (2) a **second** `AdvanceAccepted` with the same `(hub, size, root)` is
  idempotent (exactly one checkpoints row, `last_size` unchanged); (3) a later `AdvanceAccepted` at a
  larger size advances `last_size` but does **not** move `monitored_since_size` (set-once coverage).
- `go test -count=1 -run TestPollHub ./internal/follower` passes (the verified-advance path now runs
  through `AdvanceAccepted` end-to-end: checkpoint recorded, coverage set, cursor advanced).
- `grep -n "st.RecordCheckpoint(ctx, rec)" internal/follower/follower.go` is empty (exit 1) — the
  verified-advance path no longer calls the three writes directly; it calls `AdvanceAccepted` instead.

## Done When
`AdvanceAccepted` performs record + set-once coverage + cursor-advance as one transaction, the follower's
verified non-violation path calls it instead of three sequential writes, the new store test proves
atomicity / idempotent re-poll / set-once coverage, and `mise run check` plus the named follower test are
green.
