# Next Work Package

## Step: Poll-loop wrapper over PollHub (single-writer cadence + frozen back-off)

## Goal
Turn the follower from "one observation per explicit call" into a per-network driver that polls every
registered hub on a cadence from a single goroutine (the single writer per DB, ADR-0005/0007), with a
frozen hub re-polled on a backed-off evidence-only cadence (ADR-0006). This closes the named M1
"per-hub follower" loop and the "keep polling evidence-only at a backed-off cadence" correctness rule
without touching any crypto/merkle/oracle path.

## Goal-fit (state → target gap)
M1's first Verify half (`origin`/`verifierKey`/single-poll) and two of three triggers (shrink+fork
freeze) are met end-to-end. The remaining M1 majority is connective tissue: a poll loop, the did:web
key cache, the equivocation trigger, coverage, structured logs, `/metrics`, and a `cmd/` binary. The
equivocation trigger is the highest *value* but the highest *risk* — it needs `transparency-dev/merkle`
(NOT vendored under `cauldron/`, so a brand-new dependency) plus tile fixtures that do not exist yet,
making it a multi-concern step that also trips the oracle gate. The `hub_keys` cache write couples to a
stale-fixture + `derive_vkey.py` refresh that re-triggers the crypto parity oracle. The **poll loop**
is the cleanest unblocked slice the review handoff names: pure-Go orchestration over the already-tested
`PollHub`, no new deps, no fixtures, no crypto path, fully deterministic with an injected clock. It
builds directly on what exists and unblocks the eventual `cmd/` binary.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/follower/loop.go` — the cadence wrapper: a
  `HubTarget{HubID int64, BaseURL string}` value; a pure `due(...)` predicate deciding whether a hub is
  due this tick (normal vs. frozen back-off interval); a `Tick(ctx, ...)` that makes one pass over the
  targets, polling each *due* hub through the existing `PollHub`; and a thin `Run(ctx, ...)` that calls
  `Tick` on a `time.Ticker` until `ctx.Done()`.
- **Create**: `/workspace/iscc-monitor/internal/follower/loop_test.go` — seam tests (no wall-clock
  sleeps; drive `Tick`/`due` directly with an injected `now`, reusing the existing offline helpers).
- **Modify**: (none expected). `PollHub`, `store.FollowState`, and the `logclient.Fetcher` seam are
  already sufficient. Do **not** edit `follower.go` unless a tiny exported-helper extraction is
  genuinely unavoidable; if so, keep it to that ONE production file and within the ≤3-file cap.
- **Reference**:
  - `/workspace/iscc-monitor/internal/follower/follower.go` — `PollHub` signature
    `(ctx, *store.Store, logclient.Fetcher, hubID int64, baseURL string, observedAt time.Time,
    alert AlertFunc) (logclient.Status, error)`, the `AlertFunc` seam, and the freeze-on-frozen-hub
    re-detection behavior the loop relies on (a frozen hub re-polled records evidence + never re-alerts).
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` — the established offline test pattern
    to reuse verbatim: `compositeFetcher`, `openTemp`, `countRows`, `assertViolation`, `sb0ObservedAt`,
    `noopAlert`, `sb0VerifiedFetcher`, and the shrink/fork seed pattern (`RecordCheckpoint` +
    `AdvanceFollowState`).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `FollowState{LastSize, Frozen, LastError}`
    (the `Frozen` flag the back-off decision reads) and `AdvanceFollowState` (the cursor `PollHub`
    advances).
  - `/workspace/iscc-monitor/.claude/context/learnings.md` — "freeze, never crash" / "no auto-unfreeze"
    / "re-detection is itself evidence" / "single writer per DB (ADR-0005/0007)".

## Not In Scope
- The merkle-backed **equivocation** trigger — needs `transparency-dev/merkle` (a new dep, not in
  `cauldron/`) plus tile fixtures that do not exist; its own later step, and it trips the oracle gate.
- The `hub_keys` did:web cache write and the stale `sb1.amlet.id_did.json` / `derive_vkey.py` HUBS
  refresh (couples to the crypto parity oracle — separate step).
- A `cmd/` binary entrypoint, config loading, and the realm registry — the loop stays a pure library
  function with injected `targets`, `now`, `fetcher`, and `alert`; wiring it to a binary is later.
- `/metrics`, structured logging, and a real alert transport.
- Coverage tracking — do **not** write `hubs.monitored_since_*` here.
- Any schema change — do **not** add a `last_poll` column; track last-poll times in-memory in the loop
  state for v1.
- Adding any dependency — `go.mod`/`go.sum` must stay byte-identical.
- Spawning a goroutine per hub — the loop is single-goroutine to preserve the single-writer discipline.
- Calling `time.Now()` inside `Tick`/`due` — inject `now time.Time` so tests stay deterministic and
  never sleep; `time.Now()` may appear only inside `Run`'s ticker plumbing.

## Implementation Notes
- **Keep `PollHub` the single source of poll behavior.** `Tick` must call the existing
  `PollHub(ctx, st, fetcher, hubID, baseURL, now, alert)` for each due hub and must not duplicate the
  fetch/verify/freeze logic. A frozen hub re-polled through `PollHub` already re-records the violation
  as evidence and never re-alerts (`wasFrozen` gates the alert), which is exactly the evidence-only
  re-poll behavior. The loop's only added responsibility is *when* (cadence), never *what*.
- **`due` is a pure predicate** — the one easily golden-testable unit. Suggested signature:
  `due(frozen bool, lastPoll, now time.Time, normal, frozenInterval time.Duration) bool` returning
  `now.Sub(lastPoll) >= interval`, where `interval = frozenInterval` when `frozen` else `normal`. A
  zero `lastPoll` (never polled) is always due. `frozenInterval >= normal` encodes the back-off. Read
  `frozen` from `store.FollowState(ctx, hubID).Frozen` inside `Tick` (it is not carried on the target).
- **Single goroutine, single writer (ADR-0005/0007).** `Run` owns one `time.Ticker`; each tick calls
  `Tick`, which iterates the targets sequentially in the same goroutine so all writes serialize. Use
  `select { case <-ctx.Done(): return ctx.Err(); case <-ticker.C: ... }` with `defer ticker.Stop()`.
  `Run` returns `ctx.Err()` on cancellation and never panics. The loop should hold its own
  `map[int64]time.Time` of last-poll times (keyed by hubID), updated after a successful `PollHub`.
- **Errors don't kill the loop.** A per-hub `PollHub` error (transport / garbled body / store fault)
  must not abort the pass over the other hubs — a flaky single hub never stalls the network's loop.
  Pick ONE explicit policy and document it: e.g. `Tick` attempts every hub, then returns the first
  error encountered (or `nil`); `Run` logs/ignores a `Tick` error and continues to the next tick. Do
  not swallow the error silently inside `Tick` without surfacing it to the caller.
- **Correctness rule (learnings / ADR-0006):** "A self-consistency violation freezes, never crashes …
  keep polling evidence-only at a backed-off cadence, no auto-unfreeze, survive restart, other hubs
  unaffected." The loop must (a) re-poll a frozen hub only at the longer `frozenInterval`, (b) never
  clear `frozen` (it already cannot — `PollHub`/`AdvanceFollowState` never unfreeze), and (c) keep
  advancing the other unfrozen hubs in the same pass.
- **Style:** short single-purpose functions, evergreen docstrings, file-level docstring explaining the
  loop's purpose. No `t.Skip` / `//nolint` / swallowed errors / build tags. Keep the package import set
  free of any `net/http` beyond what `PollHub` already pulls through the `logclient.Fetcher` seam, and
  reuse the test helpers rather than re-declaring fetchers.
- **Conformance/oracle gate:** N/A for this slice — it is pure orchestration over `PollHub` + store
  reads, touching no signature, RFC-6962 proof, didweb, or merkle code. That gate trips only when the
  merkle-backed equivocation slice lands.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .`
  empty).
- `go test -count=1 -run TestDue ./internal/follower` passes — the pure `due` table covers: a zero
  `lastPoll` (never polled) is always due; an unfrozen hub polled `< normal` ago is NOT due; a frozen
  hub polled past `normal` but `< frozenInterval` ago is NOT due (proves back-off); each is due at/past
  its relevant interval.
- `go test -count=1 -run TestTick ./internal/follower` passes — driving `Tick` with an injected `now`
  and `sb0VerifiedFetcher` over two registered clean hubs advances each due hub's
  `FollowState.LastSize` to `10183` (asserted via `store.FollowState`); a second `Tick` at the *same*
  `now` does not re-poll (cursor unchanged and `countRows(path, "checkpoints")` unchanged).
- `go test -count=1 -run TestTickFrozenUnaffected ./internal/follower` passes — with one hub seeded to
  freeze on the next poll (a shrink/fork seed like `TestPollHubShrink`) and one clean hub, a `Tick`
  re-polls the frozen hub only at `frozenInterval`, never clears its `Frozen` flag, records the
  violation again as evidence (`countRows(path, "violations")` increments on the back-off re-poll),
  fires no new alert, and still advances the clean hub to `10183` in the same pass.
- `go test -count=1 ./internal/follower` passes — the existing `TestPollHub*` tests stay green
  (`PollHub` unchanged).
- `git status --short go.mod go.sum` is empty (no dependency added; `go.mod`/`go.sum` byte-identical).

## Done When
`internal/follower/loop.go` drives `PollHub` over multiple hubs from one goroutine on a cadence with a
frozen-hub back-off, every listed `go test -run` check passes, and `mise run check` is green with no
new dependency added.
