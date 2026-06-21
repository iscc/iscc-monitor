## 2026-06-21 — Collapse accepted-checkpoint advancement into one store-owned transaction (`AdvanceAccepted`)

**Done:** Added `Store.AdvanceAccepted(ctx, CheckpointRecord)` which performs the checkpoint
dedupe-insert, the set-once coverage UPDATE, and the follow-cursor upsert against one `*sql.Tx` and
commits (rolls back on any error), then replaced the follower's three caller-sequenced writes
(`RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState`) in the verified non-violation path with a
single `AdvanceAccepted(ctx, rec)`. This is the first `BeginTx`/`*sql.Tx` use in the repo. Closes the
`normal` ADR-0005 issue "Accepted checkpoint advancement is three caller-sequenced store writes".

**Files changed:**
- `internal/store/checkpoints.go`: added `AdvanceAccepted` (the three SQL statements ported
  byte-for-byte from `RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState`, `s.db.ExecContext` →
  `tx.ExecContext`; `defer func() { _ = tx.Rollback() }()` guard; `return tx.Commit()` last; all errors
  `%w`-wrapped with a `store.AdvanceAccepted:` prefix). The three original methods are unchanged and
  still public.
- `internal/follower/follower.go`: the verified-advance block now calls `st.AdvanceAccepted(ctx, rec)`
  once (error wrapped `follower.PollHub: hub %d: advance accepted: %w`); the `rec`,
  `cacheHubKey`/`fsckMirror`/`recordVerdict` calls, the freeze path's `RecordCheckpoint`, and
  `ingestTiles` are untouched.
- `internal/store/checkpoints_test.go`: added `TestAdvanceAccepted` covering the three `next.md`
  criteria.

**Verification:** `mise run check` → green (build + vet + all 15 packages `ok`); `gofmt -l .` empty.
- [x] `go test -count=1 -run TestAdvanceAccepted ./internal/store` — PASS: (1) one call records the
  checkpoint row + `monitored_since_size` + `follow_state.last_size`; (2) a second call with the same
  `(hub, size, root)` is idempotent (exactly one checkpoints row, `last_size` unchanged); (3) a later
  call at size 500 advances `last_size` to 500 but leaves coverage at size 100 / `t0` (set-once,
  asserted via both `Coverage` and a raw `monitored_since_{size,time}` read).
- [x] `go test -count=1 -run TestPollHub ./internal/follower` — PASS (verified-advance path now runs
  through `AdvanceAccepted` end-to-end; freeze / fork / equivocation / inclusion / fsck paths green).
- [x] `grep -n "st.RecordCheckpoint(ctx, rec)" internal/follower/follower.go` — empty (exit 1).
- [x] Conformance-bearing packages re-run uncached (`internal/follower`, `internal/store`,
  `internal/logclient`, `cmd/notecheck`) — all `ok` (the fsck root-rebuild, inclusion cross-check, and
  golden-vector tests still pass after the verified-advance path changed).

**Next:** Drain the remaining `normal` issue — the tile-writer `p`-vocabulary unification: make
`RecordTile`/`RecordEntryBundle` take `p uint8` and delete the follower's `widthForP` copy (keeps the
M3 mirror arc moving without opening a new milestone).

**Notes:**
- **Oracle/conformance gate is correctly N/A for this slice.** `AdvanceAccepted` is plain
  transactional SQL (dedupe-insert + guarded UPDATE + upsert) with no signature / RFC-6962 / Merkle /
  did:web / fsck-rebuild path. The golden-vector and fsck conformance tests live in
  `logclient`/`follower`/`notecheck` and were re-run uncached above to prove the verified-advance path
  change didn't regress them. `derive_vkey.py` not run (no key-derivation path touched).
- **Idempotent re-poll comes free from `ON CONFLICT … DO NOTHING`** (checkpoint) + `IS NULL` guard
  (coverage) — `AdvanceAccepted` discards every `Result`, never inspects `RowsAffected`, and drops
  `RecordCheckpoint`'s read-back-on-conflict branch (the follower's advance path needs neither the id
  nor the inserted-bool). Set-once coverage and no-auto-unfreeze (`frozen` omitted from the cursor
  upsert) are preserved exactly per `next.md`.
- **`Rollback` after a successful `Commit` returns `sql.ErrTxDone`**, which the `defer _ = tx.Rollback()`
  ignores by design — this is the standard `database/sql` tx pattern, not a swallowed-error gate dodge
  (the real commit error is returned and `%w`-wrapped). The transaction runs on the store's single
  capped connection (`SetMaxOpenConns(1)`) and opens no second connection (ADR-0005/0007).
- **`RecordCheckpoint`/`SetCoverage`/`AdvanceFollowState` stay public** — still used as seed helpers by
  `internal/follower/*_test.go`, `cmd/iscc-monitor/main_test.go`, and the freeze-path
  `RecordCheckpoint` at follower.go:434 (all out of scope, untouched).
- `go.mod`/`go.sum`/`schema.sql` byte-unchanged; diff is exactly the three in-scope files.
