# Next Work Package

## Step: Stamp pending OTS roots against the calendar in the off-path OTSTick loop

## Advances
The **OTS / Bitcoin anchoring** milestone Verify criterion (target.md §"OTS / Bitcoin anchoring"):

> "a stamped root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots`
> client."

This is the single open OTS Verify-closer's first unblocking half: it makes pending rows carry a **real
serialized OpenTimestamps proof** (`OTSBytes`) so `OTSTick`'s existing upgrade step stops being a
structural no-op and a root can actually transit `pending → Bitcoin-confirmed`. Per `state.md` "Next
Milestone" §1 and the `review` handoff `**Next:**`, this is *the* increment that finally advances the
end-to-end transit — `state.md`'s DRIFT WATCH names "an increment that does not make a root transit
pending → Bitcoin-confirmed end-to-end" as drift, so this closes the bar rather than de-risking it.

## Goal
Close the gap where pending OTS rows have EMPTY `OTSBytes`, so `OTSTick`'s upgrade closure fails to parse
the (empty) proof and the row backs off forever. After this step a pending row gets a real
calendar-submitted proof and `OTSTick` can upgrade it toward Bitcoin confirmation — without ever blocking
the follower poll path.

## Scope
- **Create**: (none)
- **Modify** (≤3 non-test/doc source files):
  - `/workspace/iscc-monitor/internal/follower/otsloop.go` — add a `Stamper` func seam and stamp
    not-yet-stamped pending rows in `OTSTick` (off the poll path) before the upgrade step.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — construct the real `Stamper` (a closure over
    `otsclient.Stamp` + `otsclient.DefaultCalendarURL`) and pass it through `runOTSLoop` → `OTSTick`.
- **Tests** (NOT counted toward the ≤3 budget):
  `/workspace/iscc-monitor/internal/follower/otsloop_test.go` — add a `TestOTSStamp…` exercising the new
  stamp-then-upgrade path with an injected fake `Stamper` + fake `Upgrader` (fully offline). The existing
  `TestOTSTick*` tests gain a fake `Stamper` argument.
- **Reference** (read before writing):
  - `/workspace/iscc-monitor/.claude/context/learnings/otsclient.md` — the `Stamp`/`Upgrader` contract;
    "OTS never blocks"; `OTSBytes` = serialized initial pending sequence(s); fixtures = `examples/*.ots`
    ground truth (never a live calendar in `go test`).
  - `/workspace/iscc-monitor/.claude/context/learnings/follower.md` §"OTS upgrade-loop control core" —
    `OTSTick` error discipline (log-and-continue, never abort, never freeze); `Upgrader` is a func seam
    (YAGNI, matches `AlertFunc`) — match that shape for `Stamper`.
  - `/workspace/iscc-monitor/internal/otsclient/client.go` — `Stamp(ctx, calendarURL, digest [32]byte)
    ([]byte, error)`, `DefaultCalendarURL`, and `buildUpgrader`'s `recoverRead(r.OTSBytes)` (empty bytes
    → parse error today, which is why the upgrade is a no-op).
  - `/workspace/iscc-monitor/internal/store/ots.go` — `OTSRecord` fields (`OTSBytes`, `CalendarURLs`),
    `PendingOTS`, `MarkOTSAttempted`, `RecordOTS`. **Confirm whether a store mutator exists to persist
    `OTSBytes`/`CalendarURLs` onto an existing pending row** (an update-by-`(hub,tree_size,root)` key); if
    none, the stamped bytes can ride `RecordOTS`'s idempotent upsert or a minimal new store-leaf mutator —
    decide which before coding, preferring an existing seam.
  - `/workspace/iscc-monitor/internal/follower/otsloop.go` (current `OTSTick`) — the per-row loop to extend.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go:151,208` — `runOTSLoop` construction + signature.

## Not In Scope
- **Do NOT call `otsclient.Stamp` inside `PollHub`/`stampRoot`** (the literal handoff wording said "wire
  `otsclient.Stamp` into `follower.stampRoot`"). `stampRoot` runs ON the poll path; a synchronous calendar
  HTTP round-trip there VIOLATES the always-loaded Correctness rule **"OTS never blocks the follower"**
  (learnings.md + ADR-0004). Stamping belongs in the already-off-path `OTSTick`/`runOTSLoop` goroutine.
  `stampRoot` stays a pure local insert (empty `OTSBytes` = "not yet stamped" sentinel) — leave
  `internal/follower/follower.go` byte-unchanged. **This is a deliberate, rule-driven deviation from the
  handoff `**Next:**`; record it in the advance handoff.**
- The `.ots` HTTP route (reads `OTSForRoot`) — a later sub-step, after confirmed rows can exist.
- Certificate **§5 BITCOIN ANCHOR** (`HasClause5`) — a later sub-step, after confirmed rows exist.
- A schema migration / new `ots` column — reuse the existing `OTSRecord`/`ots` table.
- The deferred `host:port` DID-encoding fix, §6 timestamp, and `hubDomain` ForceQuery issues — none of
  those surfaces are touched here.

## Implementation Notes
- **Seam shape (match `Upgrader`/`AlertFunc` — a func, not an interface):**
  `type Stamper func(ctx context.Context, root [32]byte) (otsBytes []byte, calendars string, err error)`.
  Production wires it in `main.go` as a closure: `func(ctx, root) { b, err := otsclient.Stamp(ctx,
  otsclient.DefaultCalendarURL, root); return b, otsclient.DefaultCalendarURL, err }`. Keep
  `internal/follower` import-free of `internal/otsclient`/`internal/ots`/`opentimestamps` (the anchoring
  import-isolation invariant) — `Stamper` IS the boundary, exactly like `Upgrader`.
- **Where to stamp:** in `OTSTick`'s per-row loop, BEFORE calling the `Upgrader`, branch on
  `len(r.OTSBytes) == 0` (the not-yet-stamped pending row `stampRoot` wrote):
  - build the root key (`var root [32]byte; copy(root[:], r.Root)`), call the `Stamper`,
  - on success persist `OTSBytes`/`CalendarURLs` onto the row via the chosen store mutator, then EITHER
    update `r` in-memory and fall through to the upgrade step, OR `continue` and let the next tick upgrade
    it — pick the simpler (likely `continue`, mirroring the back-off flow), and document the choice in the
    `OTSTick` docstring,
  - on a Stamp transport fault: `MarkOTSAttempted(attempts+1, now.Add(backoff(attempts)))` + log-and-
    continue, EXACTLY like the existing upgrade-fault branch. A stamp fault NEVER aborts the pass and
    NEVER freezes a hub (ADR-0004 / ADR-0006).
- **Thread the seam:** `OTSTick(ctx, st, stamper, up, now, logger)` — add `Stamper` next to `Upgrader`.
  Update `runOTSLoop`'s signature + its single `OTSTick` call; `runOTSLoop` gains a `stamper
  follower.Stamper` arg passed from `run()` (built next to `otsclient.NewUpgrader()` at main.go:151).
- **Nil-`Stamper` handling:** decide and document — either tolerate a nil `Stamper` (skip stamping, leave
  the row empty — keeps a bare call path well-defined and old behavior intact) or require it. Prefer
  nil-tolerant, mirroring the Loop's nil-`Logger`/`Metrics` discipline; state it in the docstring.
- **Governing Correctness rule (learnings.md, always-loaded):** "**OTS never blocks** the follower loop;
  calendars are best-effort with backoff + redundancy." All calendar I/O stays in the off-poll-path
  `OTSTick`/`runOTSLoop` goroutine; a stamp failure is a back-off, never a freeze, never an abort. Also:
  `internal/otsclient`/`internal/ots` are NOT WASM-pure — keep them out of `internal/follower`'s import
  closure (the `Stamper` seam preserves this).
- **Oracle gate:** N/A for the seam wiring (opaque pending→confirmed over an already-fsck-verified root;
  no signature/RFC-6962/Merkle/did:web/proof code touched). The `ots verify` oracle applies to the
  `internal/ots` classifier + bundled `examples/*.ots`, unchanged here. Tests MUST run fully offline —
  inject fake `Stamper`/`Upgrader`, never hit a live calendar in `go test`.
- **Test through the public store seam:** seed a pending row with empty `OTSBytes` (via `RecordOTS`); run
  `OTSTick` with a fake `Stamper` returning fixture bytes and a fake `Upgrader` that confirms; assert the
  row reaches `OTSStatusConfirmed` (and/or carries `OTSBytes` after the stamp pass) via
  `OTSForRoot`/`PendingOTS` read-back — never `OTSTick` internals.

## Verification
- `mise run check` is green (build + vet + test, all packages; `gofmt -l .` excl. `cauldron/` empty).
- `go test -count=1 -run TestOTS ./internal/follower` passes (includes the new stamp-path test plus the
  existing `TestOTSTick*` / `TestOTSBackoff`).
- `go list -deps ./internal/follower` contains NO `internal/otsclient`, `internal/ots`, or
  `github.com/nbd-wtf/opentimestamps` (import-isolation invariant held — `Stamper` is the boundary):
  `go list -deps ./internal/follower | grep -c -e 'internal/otsclient' -e 'internal/ots' -e 'nbd-wtf/opentimestamps'` == 0.
- `go list -deps ./internal/store` is still a leaf: `go list -deps ./internal/store | grep -c -e 'internal/follower' -e 'internal/otsclient' -e '^net/http$'` == 0.
- Mutation (reverted): removing the new `len(r.OTSBytes) == 0` stamp branch from `OTSTick` makes the new
  `TestOTSStamp…` test FAIL (the pending row never gets `OTSBytes`, so it can never confirm); restoring
  → green.

## Done When
`OTSTick` stamps not-yet-stamped pending rows against the calendar via an injected `Stamper` (off the poll
path), `main.go` wires the real `otsclient.Stamp` closure, all Verification checks pass, and the
follower's import closure still excludes every anchoring package.
