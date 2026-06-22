# Next Work Package

## Step: OTS upgrade-loop core — deterministic `OTSTick` over an injected `Upgrader` seam, with `Attempts`/`NextRetry` back-off

## Advances
The **OTS / Bitcoin anchoring** milestone (target.md): *"stamp each distinct observed root daily
(`UNIQUE(hub, tree_size, root)`) + background upgrade loop (pending → Bitcoin-confirmed) + serve
`.ots`; never blocks the follower. **Verify:** a stamped root upgrades to Bitcoin-confirmed and the
served `.ots` verifies with the standard `ots` client."*

This is the **background upgrade loop** half of that criterion, and the directive the latest
`handoff.md` `**Next:**` names verbatim ("read `store.PendingOTS`, … flip rows to confirmed via
`MarkOTSUpgraded` … where the `Attempts`/`NextRetry` retry-policy columns finally get exercised … runs
in its own goroutine off the poll path"). The store learnings flag exactly this gap:
"`Attempts`/`NextRetry` are persisted+round-tripped but no method increments `Attempts` or sets a
back-off `NextRetry` yet — that retry policy belongs to the upgrade loop." No `critical`/`normal` issue
preempts milestone work; the 3 open `normal` issues are all "fix-on-next-touch" items in surfaces this
step does not touch.

**Skeleton-first (target.md rule):** the *full* upgrade loop (pull in `nbd-wtf/opentimestamps`,
calendar HTTP, Bitcoin-header confirmation, the `.ots` route, certificate §5) is far larger than one
≤3-file step and mixes an unverifiable live-network path with testable retry logic. This step lays the
**verifiable skeleton**: the pure, deterministic retry-policy core driven by an injected `Upgrader`
seam, golden-tested with no live network — closing the loop's *control* logic. The real
`opentimestamps`-backed `Upgrader` (the first `go.mod`/`go.sum` change), the `.ots` route, and cert §5
are the named sub-steps under `## Not In Scope`, so later iterations continue this arc.

## Goal
Give the monitor a deterministic, testable background upgrade loop that reads pending stamped roots,
asks an injected `Upgrader` whether each is Bitcoin-confirmed yet, and either marks it confirmed
(`MarkOTSUpgraded`) or records a backed-off retry (`Attempts++`, `NextRetry` pushed out) — without ever
touching the follower poll path or the network in tests. This is the control core every later OTS
sub-step (real calendar client, `.ots` route, cert §5) plugs into.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/follower/otsloop.go` — the `Upgrader` seam (a func
  type), an `UpgradeResult` (confirmed vs still-pending), and a deterministic
  `OTSTick(ctx, st, up Upgrader, now time.Time) error` that drives one pass over `store.PendingOTS`.
  (Tests: `/workspace/iscc-monitor/internal/follower/otsloop_test.go`.)
- **Modify**:
  - `/workspace/iscc-monitor/internal/store/ots.go` — add ONE leaf method
    `MarkOTSAttempted(ctx, hubID int64, treeSize uint64, root []byte, attempts int64, nextRetry
    time.Time) error` (plain `UPDATE … SET attempts=?, next_retry=?`, `RowsAffected`-ignored like
    `MarkOTSUpgraded`/`SetCoverage`); and narrow `PendingOTS` so the loop only sees rows whose back-off
    has elapsed — add a `now time.Time` arg and `AND (next_retry IS NULL OR next_retry <= ?)` to its
    `WHERE`.
  - `/workspace/iscc-monitor/internal/store/ots_test.go` (test, not counted) — `MarkOTSAttempted` +
    `PendingOTS` back-off-filter coverage.

  **Two production files** (`otsloop.go`, `ots.go`). The third allowable production slot is intentionally
  left unused — `cmd/iscc-monitor/main.go` wiring of a real `Run` goroutine is **Not In Scope** (the real
  `Upgrader` does not exist yet; wiring a no-op loop would be dead code).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — the OTS-CRUD idioms
    (`MarkOTSUpgraded` ignores `RowsAffected`; `OTSStatusPending`/`OTSStatusConfirmed` are the single
    source of the literals; the `Attempts`/`NextRetry` trap this step closes; `unixOrNil` zero-time→NULL).
  - `/workspace/iscc-monitor/.claude/context/learnings/follower.md` — the `loop.go` cadence pattern: a
    **pure, injected-`now`, deterministic** `Tick` with all wall-clock confined to `Run`'s ticker; tests
    never sleep; a flaky item logs-and-continues, the pass returns the first error, never aborts (mirror
    this for `OTSTick`); `AlertFunc` is a func seam, not an interface (YAGNI) — model `Upgrader` on it.
  - `/workspace/iscc-monitor/internal/store/ots.go` — `OTSRecord`, `PendingOTS`, `MarkOTSUpgraded`
    signatures to port the new method/arg against.
  - `/workspace/iscc-monitor/internal/follower/loop.go` — `Tick`/`due()`/`Run` shape to mirror.
  - `/workspace/iscc-monitor/.claude/plans/cosmic-baking-octopus.md` §"OpenTimestamps (ADR-0004)"
    (lines ~48, 81-83): "background upgrade loop (pending → Bitcoin-confirmed)", calendar-attested
    server-side, "OTS **never blocks** the follower".

## Not In Scope
- **Do NOT add `github.com/nbd-wtf/opentimestamps`** (or any calendar/Bitcoin dependency) to
  `go.mod`/`go.sum` this step. The `Upgrader` stays an injected seam; the *real* calendar-HTTP
  `Upgrader` implementation + the dependency add is the **next** sub-step.
- **Do NOT add the `.ots` HTTP route** or wire it into any serve handler — later sub-step (reads
  `OTSForRoot`).
- **Do NOT touch certificate §5 / `HasClause5`** (`internal/certificate/handler.go:286`) — it consumes
  confirmed `ots` rows; it lands after the real `Upgrader` actually produces them.
- **Do NOT wire a live `Run` goroutine into `cmd/iscc-monitor/main.go`** — there is no real `Upgrader`
  to drive it yet; a no-op background loop would be dead code. Wire it when the real `Upgrader` lands.
- **Do NOT change the follower poll path** (`PollHub`/`stampRoot`/`loop.go` `Tick`/`Run`) — the upgrade
  loop is a *separate* driver; OTS must never block the follower (the stamp already runs on the verified
  path; this loop only upgrades pending rows).
- Do NOT touch the three open `normal` issues (ForceQuery, `host:port` DID, §6 timestamp) — none lie on
  this path.

## Implementation Notes
- **Mirror `loop.go`'s shape, do not invent a new one.** `OTSTick(ctx, st, up, now)` is the pure
  injected-`now` analogue of `Tick`: read `st.PendingOTS(ctx, now)` (now back-off-filtered), iterate
  oldest-first, and for each row call `up(ctx, row)`; **all wall-clock stays out of `OTSTick`** (the
  caller injects `now`), exactly like `Tick`/`due()`. A `Run`-style ticker wrapper is optional and, if
  added, must mirror `loop.go`'s `Run` (own ticker, `defer Stop()`, log-and-continue, only ctx
  cancellation ends it, untested-by-design per the follower learnings) — but prefer to leave `Run` for
  the wiring sub-step and keep this step to the testable `OTSTick` + store method.
- **`Upgrader` seam = a func type, not an interface (YAGNI, matches `AlertFunc`).** Suggested:
  `type Upgrader func(ctx context.Context, r store.OTSRecord) (UpgradeResult, error)` returning
  `UpgradeResult{Confirmed bool; OTSBytes []byte; BTCHeight int64}`. The real calendar client becomes a
  closure of this type next step. Keeping it a func keeps `internal/follower` import-free of any
  anchoring package and the store a leaf.
- **Per-row outcomes:**
  - `Upgrader` returns `Confirmed==true` → `st.MarkOTSUpgraded(ctx, r.HubID, r.TreeSize, r.Root,
    res.OTSBytes, res.BTCHeight, now)` (the row drops out of future `PendingOTS`).
  - `Upgrader` returns `Confirmed==false`, `err==nil` (calendar says "not yet Bitcoin-confirmed") →
    record a back-off: `st.MarkOTSAttempted(ctx, …, r.Attempts+1, now.Add(backoff(r.Attempts+1)))`.
  - `Upgrader` returns a non-nil `err` (transport fault) → ALSO a back-off (`Attempts+1`, pushed
    `NextRetry`) and **log-and-continue** (this single hub/row fault must not abort the pass), then fold
    into `firstErr` and `return firstErr` at the end — the exact `loop.go` `Tick` error discipline. OTS
    faults NEVER freeze and NEVER surface to the follower (anchoring is best-effort, ADR-0004 +
    learnings "OTS never blocks").
- **`backoff(attempts)` is a tiny pure helper** (e.g. capped exponential: `base << min(attempts, cap)`
  clamped to a `max` — pick simple, document the cap). Make it a pure function so it is directly
  golden-testable; the OTS-never-blocks rule means exact cadence is not safety-critical, but the
  function must be deterministic and monotonic up to the cap.
- **`MarkOTSAttempted` ports `MarkOTSUpgraded` verbatim** minus the status/btc columns: plain
  `UPDATE ots SET attempts=?, next_retry=? WHERE hub_id=? AND tree_size=? AND root=?`,
  `RowsAffected`-ignored (idempotent/absent re-mark is a no-op, the `SetCoverage`/`MarkOTSUpgraded`
  idiom), `unixOrNil(nextRetry)` for the zero-time→NULL convention. Do NOT touch `status` here — a
  back-off keeps the row `pending` so `PendingOTS` re-surfaces it once `next_retry` elapses.
- **`PendingOTS` gains a `now time.Time` arg + `AND (next_retry IS NULL OR next_retry <= ?)`.** Bind
  `now.Unix()` directly (NOT `unixOrNil` — `now` is never the zero time here; a freshly-stamped row has
  `next_retry == NULL` so it is immediately due via the `IS NULL` leg). `grep -rn "PendingOTS"
  --include='*.go'` confirms `PendingOTS` has **no production caller yet** (only `ots_test.go`), so the
  only call-site edits are the test + the new loop — re-verify with that grep before editing.
- **Store stays a leaf** (learnings durable rule): `MarkOTSAttempted` adds no import; verify
  `go list -deps ./internal/store | grep '^net/http'` stays empty and the package `.Imports` stay
  `context database/sql embed errors fmt time` + the sqlite driver.
- **Oracle/conformance gate is N/A** for this slice (learnings + handoff precedent): it moves an opaque
  `pending`→`confirmed`/back-off over an already-fsck-verified accepted root; no
  signature/RFC-6962/Merkle/did:web/proof code added or changed. The `Upgrader` is injected, so tests
  use a fake confirming/declining/erroring upgrader — no `ots verify` crypto in this step (that lands
  with the real `Upgrader`). State this N/A explicitly in the handoff.
- **Tests (seam-based, observable store outputs only — never loop internals, per PRD Testing
  Decisions):** seed `ots` rows via `RecordOTS`, then drive `OTSTick` with fake `Upgrader`s and assert
  on `OTSForRoot`/`PendingOTS` read-back:
  - confirming upgrader → row becomes `OTSStatusConfirmed` with the returned `OTSBytes`/`BTCHeight`,
    drops out of `PendingOTS(ctx, later)`;
  - declining upgrader (`Confirmed==false, err==nil`) → row stays `pending` with `Attempts==1` and a
    `NextRetry` in the future; a second `OTSTick` at the same `now` does NOT re-process it
    (`PendingOTS(ctx, now)` excludes it); at `now+backoff` it re-surfaces and `Attempts==2`;
  - erroring upgrader → back-off recorded AND `OTSTick` returns the wrapped first error
    (log-and-continue, pass not aborted — assert a second seeded row still got processed);
  - `backoff()` golden table (monotonic up to the cap).
- **Mutation-prove non-vacuous** (revert after): no-op the `MarkOTSUpgraded` call → the confirm test
  FAILS (row stays pending); no-op `MarkOTSAttempted` → the back-off/`Attempts++` test FAILS; revert the
  `next_retry` `WHERE` clause → the "not re-processed before back-off elapses" assertion FAILS.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty,
  excluding gitignored `cauldron/`).
- `go test -count=1 -run TestOTS ./internal/store ./internal/follower` passes (name the new tests
  `TestOTS…` / `TestMarkOTSAttempted` / `TestPendingOTSBackoff` so this filter catches them all — the
  filter caveat the prior OTS review flagged).
- A confirming `Upgrader` over a seeded `pending` row leaves it `OTSStatusConfirmed` with the returned
  `OTSBytes`/`BTCHeight` (`OTSForRoot` read-back) and absent from `PendingOTS(ctx, now)`.
- A declining `Upgrader` leaves the row `pending` with `Attempts==1` and a future `NextRetry`; it is
  excluded from `PendingOTS(ctx, now)` and re-surfaces at `PendingOTS(ctx, now+backoff)` with `Attempts`
  incrementing.
- `go list -deps ./internal/store | grep '^net/http'` is empty; `go.mod`/`go.sum` are byte-unchanged
  (`git diff --name-only -- go.mod go.sum` empty) — no `opentimestamps` dependency added this step.
- Mutation check: no-op `MarkOTSUpgraded` FAILS the confirm test; reverting the `next_retry` `WHERE`
  clause FAILS the back-off-exclusion test (proves non-vacuous), then reverted.

## Done When
`OTSTick` deterministically upgrades a confirmed pending root via `MarkOTSUpgraded` and records a
backed-off `Attempts`/`NextRetry` retry (via the new `MarkOTSAttempted` + `next_retry`-filtered
`PendingOTS`) for a still-pending or erroring one — all golden-tested with an injected `Upgrader` and no
new dependency — with `mise run check` green and the mutation checks proving the tests non-vacuous.
