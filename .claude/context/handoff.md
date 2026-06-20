# Handoff

## 2026-06-20 — Typed store persistence helpers for the checkpoint verdict (`hubs` / `checkpoints` / `follow_state`)

**Done:** Added the four typed methods on `*Store` the follower will drive to persist one observed
checkpoint verdict — `UpsertHub`, `RecordCheckpoint`, `FollowState`, `AdvanceFollowState` — plus the
plain store-owned structs `CheckpointRecord` and `FollowState`, with driving tests against a
`t.TempDir()` DB asserting on observable rows only. `internal/store` stays a leaf: no
`internal/logclient` import (the follower maps the verdict to a plain status string at the call site).

**Files changed:**
- `internal/store/checkpoints.go` (new): the four methods + the `CheckpointRecord`/`FollowState`
  structs + a `unixOrNil` helper. `UpsertHub` is select-then-insert (idempotent on `domain`, no schema
  change). `RecordCheckpoint` uses `INSERT … ON CONFLICT(hub_id,tree_size,root) DO NOTHING` then reads
  the id back on conflict (`inserted=false`, nil err); leaves `consistent`/`root_rebuilt` NULL.
  `FollowState` returns the zero value (nil err) for a hub with no row. `AdvanceFollowState` upserts
  `last_size` and omits `frozen` from the `DO UPDATE SET` so a frozen hub stays frozen (ADR-0006).
- `internal/store/checkpoints_test.go` (new): idempotency, dedupe (+ distinct root), zero-ObservedAt→
  NULL, unknown-hub zero value, advance round-trip, no-auto-unfreeze, restart survival.

**Verification:** `mise run check` → green (build + vet + test all exit 0, go1.24).
- [x] `gofmt -l internal/store` → empty.
- [x] `go test -run TestStore ./internal/store` → PASS (existing 4 subtests stay green).
- [x] New tests PASS: UpsertHub same-domain → same id, exactly 1 `hubs` row (distinct domain → 2nd
  row/id); RecordCheckpoint re-observe → same id, `inserted=false`, exactly 1 `checkpoints` row;
  FollowState unknown hub → zero `FollowState{}` + nil err; after raw `UPDATE … frozen=1`,
  `AdvanceFollowState(N)` leaves `frozen==1` and `last_size==N`; restart survival round-trips rows.
- [x] `go list -deps …/internal/store` shows no internal `iscc-monitor` deps → no `logclient` coupling.
- [x] `go mod tidy` → zero diff (no dependency change; pure-stdlib + existing driver).

**Conformance/oracle gate:** N/A this step. The diff touches only `internal/store` typed CRUD —
no signature verification, RFC-6962/Merkle, proof code, `internal/didweb`, or fork/shrink/equivocation
logic. `internal/proof` still does not exist. No oracle obligation; purity gate has nothing to regress.

**Next:** The follower poll loop — the real caller of these methods. It calls
`logclient.AcceptCheckpoint` (check `err` before the status), maps `Status.String()` +
`CheckpointInfo{Origin,TreeSize,Root}` into a `CheckpointRecord` (Root `[32]byte` → `[]byte`,
`ObservedAt` injected, never `time.Now()` in the pure layer), persists the verdict via
`RecordCheckpoint`, and calls `AdvanceFollowState` **only** on `StatusVerified`. It also writes the
`hub_keys` did:web cache and refreshes the stale sb1 `did.json` fixture + `derive_vkey.py` `HUBS` to
the current key `069d0f14`. The goroutine-ownership single-writer wrapper is the follower's concern.

**Notes:**
- Decision (flagged explicitly per `next.md`): a zero `ObservedAt` is written as **NULL** (via
  `unixOrNil`), not `0`, so "never observed" stays distinct from the unix epoch; asserted in
  `TestRecordCheckpointZeroObservedAtNull`. If the follower always sets `ObservedAt`, this never fires
  in practice but the column semantics are now explicit.
- `UpsertHub` on re-register returns the existing id **without rewriting** `origin`/`base_url` (per
  `next.md` — acceptable this step). If a hub's `base_url` ever changes, the follower/registry step
  will need an explicit update path; noted as future work, not a blocker.
- No `schema.sql` change and no `sqlite.go` change — the methods attach to the existing `*Store`.
  `hubs` still has no UNIQUE on `domain` (a schema change is a separate reviewable decision), which is
  why `UpsertHub` is select-then-insert rather than `ON CONFLICT`.
- Single-writer discipline holds at the pool level (`SetMaxOpenConns(1)` from the prior step); these
  methods use `db.ExecContext`/`db.QueryRowContext` directly and open no new connections.
