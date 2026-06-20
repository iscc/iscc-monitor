# Next Work Package

## Step: Coverage tracking — persist `monitored_since` (size + time), set-once, on first verified observation

## Goal
Record per-hub coverage start (`monitored_since_size` + `monitored_since_time`) the first time a
hub yields a verified checkpoint, and never overwrite it thereafter. This is the ADR-0001 "coverage
honesty" rule and an unmet M1 deliverable: the `hubs.monitored_since_*` columns already exist in the
schema but nothing reads or writes them.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/store/checkpoints.go` — add `SetCoverage(ctx, hubID, size, observedAt)` (set-once
    upsert of `monitored_since_size`/`monitored_since_time`) and `Coverage(ctx, hubID)` reader.
  - `internal/follower/follower.go` — in `PollHub`'s verified, non-violation path, call
    `SetCoverage` once before `AdvanceFollowState`, and update the package/function doc comment to
    drop "coverage" from the deferred "later steps" list.
- **Tests/docs (not counted against the 3 non-test/doc file limit)**:
  - `internal/store/checkpoints_test.go` — coverage CRUD + set-once tests.
  - `internal/follower/follower_test.go` — extend the verified-advance assertions to confirm coverage
    is set once and not moved by a second poll.
- **Reference**:
  - `/workspace/iscc-monitor/.claude/adr/0001-v1-trust-guarantee.md` — "Coverage and cold start"
    section: `monitored_since` is `(size + time)`, immutable, guarantees hold "from coverage start
    onward".
  - `/workspace/iscc-monitor/internal/store/schema.sql` — `hubs` table lines 24-25
    (`monitored_since_size`, `monitored_since_time`) and the unix-seconds / NULL time convention.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — mirror the existing method idioms
    (`ExecContext`, `unixOrNil`, `sql.NullInt64`, "absent row is not an error" reads, `int64(size)`
    column writes).
  - `/workspace/iscc-monitor/internal/follower/follower.go` lines 80-114 — the verified,
    non-violation path where `RecordCheckpoint`/`AdvanceFollowState` already live.

## Not In Scope
- The merkle-backed **equivocation** trigger, `transparency-dev/merkle`, tile fixtures, `fsck`, the
  `SQLiteFetcher`, or anything that trips the oracle/conformance gate — that is its own later step.
- `hub_keys` did:web cache write and the stale sb1 fixture refresh (`22b08f3e`→`069d0f14`).
- Structured logging, `/metrics`, real alert transport — separate later M1 steps.
- Backfill / cold-start tile recovery and any dashboard "coverage window" rendering (M2/M3).
- Surfacing coverage through the binary (`cmd/iscc-monitor`) or any REST/projection path.
- Recording coverage for non-verified or violating verdicts — only a clean `StatusVerified`,
  non-violation observation starts coverage.

## Implementation Notes
- **Set-once is the load-bearing semantic (ADR-0001): coverage start is immutable.** Implement
  `SetCoverage` as a conditional write so a later, larger observation never moves the start. On the
  single capped connection, prefer the explicit guarded UPDATE — it is the most obvious "write only
  if unset" and matches this file's style:
  `UPDATE hubs SET monitored_since_size = ?, monitored_since_time = ? WHERE hub_id = ? AND
  monitored_since_size IS NULL`. The `IS NULL` guard makes a re-call after the start is set a silent
  no-op. A re-call must return a nil error and must **not** depend on `RowsAffected` to signal
  success (zero rows affected after the start is set is the correct, non-error case).
- `monitored_since_time` is unix-seconds via the existing `unixOrNil(observedAt)` helper, mirroring
  `RecordCheckpoint`; write `int64(size)` like the other size columns. In practice `PollHub` always
  injects a real `observedAt`, so size and time are set together on the first verified poll.
- `Coverage(ctx, hubID)` reads `monitored_since_size`/`monitored_since_time` back through
  `sql.NullInt64` and returns `(size uint64, since time.Time, set bool, err error)` (or a tiny
  struct — match whatever reads cleanest). An unset/absent hub returns `set=false` + nil error,
  mirroring `FollowState`'s "absent row is not an error" convention. This reader lets the tests
  assert on observable state without poking raw SQL (a raw `QueryRow` in the test is also acceptable
  — match the existing store tests).
- **Wiring in `PollHub`** (`follower.go`): the call belongs on the verified, *non-violation* path
  only — after the `if violated { … }` branch returns, alongside `RecordCheckpoint` /
  `AdvanceFollowState` (lines 100-114). Place it before `AdvanceFollowState`; wrap its error like the
  siblings: `fmt.Errorf("follower.PollHub: hub %d: set coverage: %w", hubID, err)`. A frozen/violating
  hub must **not** start coverage from the contradictory observation — keep `SetCoverage` out of the
  `freeze` helper. `store` stays a leaf (no new imports; `logclient` must not enter its closure).
- Relevant Correctness rules / learnings: **"Coverage honesty (ADR-0001) — record `monitored_since`,
  state guarantees from coverage start"**; the store **"zero `time.Time` → NULL (`unixOrNil`)"** and
  **"absent row is not an error"** conventions; and the **`AdvanceFollowState` upsert omits `frozen`**
  discipline — apply that same "never clobber an immutable field" rule to `monitored_since_*`.
- Update the `follower.go` doc comment (line 12) that currently lists "coverage" among the deferred
  "later steps" so the prose stays evergreen and accurate once coverage lands.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass, `gofmt -l .`
  empty).
- `go test -run TestCoverage ./internal/store` passes: after `SetCoverage(hubID, 100, t0)` then
  `SetCoverage(hubID, 500, t1)`, `Coverage(hubID)` reports `size==100` and `since==t0` (set-once; the
  larger/later values are ignored), and an unset hub reports `set==false`.
- `go test -run TestPollHub ./internal/follower` passes: after a verified `PollHub`, the hub's
  coverage is set to the fixture size (`10183`) and the observed time; a second verified poll does
  not move it.
- `git diff HEAD -- go.mod go.sum` is empty (no new dependency).
- `go list -deps ./internal/store | grep '^net/http$'` is empty (store stays a leaf; no `net/http`).

## Done When
`SetCoverage`/`Coverage` exist with set-once semantics, `PollHub` records coverage once on the first
verified observation (not on violation/non-verified verdicts), and all Verification criteria pass.
