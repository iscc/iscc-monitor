# Next Work Package

## Step: Harden the otsclient upgrade path — panic-recover + per-request timeout

## Advances
Primarily closes the open `normal` issue **"otsclient Upgrader can crash the monitor on a library
panic AND hang `OTSTick` on a stalled calendar"** (`internal/otsclient/client.go:86`), which preempts
the milestone-skeleton submit step. Both defects are **unreachable today** ONLY because `stampRoot`
writes pending rows with EMPTY `OTSBytes`. The very next planned step populates `OTSBytes` (the
calendar-submit transit), which makes the already-wired `runOTSLoop` reach `upgrade()` in production —
turning both latent defects live. Per the seeded correctness rules **"A self-consistency violation
freezes, never crashes (ADR-0006)"** / **"OTS never blocks the follower (ADR-0004)"** and the `review`
handoff (`**Next:**` "Fold the two Codex-confirmed otsclient defects into that step ... since that step
is what first makes `upgrade()` reachable"), the safe ordering is to harden the upgrade path BEFORE any
real `OTSBytes` flows. This keeps the arc on the **OTS / Bitcoin anchoring** milestone Verify ("a
stamped root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots`
client") by de-risking its next sub-step — not a detour.

## Goal
Make the production `follower.Upgrader` closure fail-closed against the OpenTimestamps library's two
known hazards on the calendar-upgrade call: a `panic` from an uncomputable/unimplemented op
(`sha1`/`reverse`/`hexlify`/`keccak256`/invalid-instruction) must become a wrapped error (the loop
backs off, never crashes the process); and a stalled calendar GET must be bounded by a per-request
timeout (one `OTSTick` pass can never hang indefinitely, starving later pending rows). This removes the
last reachability hazard before the submit step lights up the path.

## Scope
- **Create**: (none)
- **Modify**:
  - `/workspace/iscc-monitor/internal/otsclient/client.go` — wrap the `upgrade(ctx, seq, file.Digest)`
    loop body (the `seq.Compute`-bearing call) in a panic-recover that returns a wrapped fail-closed
    error; derive a bounded `context.WithTimeout` per `upgrade()` call; add an `upgradeTimeout` const.
    (1 non-test file.)
  - `/workspace/iscc-monitor/internal/otsclient/client_test.go` — tests proving both fixes (test file;
    NOT counted toward the ≤3 non-test budget).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/otsclient.md` — the two open-issue traps (panic
    at `client.go:86`; `http.DefaultClient` no-timeout) + the `recoverRead` precedent; the
    pending-only-upgrade and store-seam-count-1 / WASM-keep-out invariants to preserve.
  - `/workspace/iscc-monitor/.claude/context/learnings/ots.md` — the `recoverParse` FFI-boundary
    panic-recover pattern to mirror (a documented `recover()` that surfaces the panic as a returned
    `err`, NOT a gate-dodge).
  - `/workspace/iscc-monitor/.claude/context/issues.md` — the exact "Verify fixed" criteria for both
    defects (an unimplemented-op sequence returns a wrapped error not a panic; the upgrade call carries
    a bounded context observable via the injected `seqUpgrade` seam).
  - `/workspace/iscc-monitor/internal/otsclient/client.go` (current `buildUpgrader` + `recoverRead`
    shape) and `/workspace/iscc-monitor/internal/otsclient/client_test.go` (the `seqUpgrade` fake /
    `readFixture` harness + the offline convention — never call a live calendar in `go test`).
  - `/workspace/iscc-monitor/internal/follower/otsloop.go` — the `Upgrader` contract `OTSTick` drives
    (a wrapped error → the documented back-off + log-and-continue; the closure must keep returning
    `(follower.UpgradeResult{}, err)` on fault).

## Not In Scope
- The `stampRoot` calendar-submit transit (populating real `OTSBytes`/`CalendarURLs`) — the immediate
  FOLLOW-UP step this one de-risks. Do NOT add a submit branch to `OTSTick`, a `Stamper` seam, or a
  `store.MarkOTSSubmitted` method here.
- Touching `/workspace/iscc-monitor/internal/follower/follower.go` `stampRoot` (stays a pure local
  SQLite insert; OTS off the poll path) or `cmd/iscc-monitor/main.go` (`runOTSLoop` wiring unchanged).
- The `.ots` HTTP route and certificate §5 BITCOIN ANCHOR (both depend on confirmed rows existing —
  later steps).
- Changing `Stamp`'s body beyond what the timeout/recover refactor naturally shares; do not wire
  `Stamp` into any caller.
- The other `normal` issues (`hubDomain` ForceQuery; §4/bundle `host:port` DID; §6 timestamp) — this
  diff does not touch those surfaces.

## Implementation Notes
- **Panic-recover (defect 1).** Mirror the existing `recoverRead` pattern verbatim — a documented
  FFI-boundary `recover()` that converts the library panic into a wrapped fail-closed `error`, never a
  `//nolint`/`t.Skip`/swallow. `opentimestamps.UpgradeSequence` → `seq.Compute(initial)` →
  `inst.Operation.Apply(...)` panics on unimplemented ops (`opentimestamps@v0.4.0/ots.go:46-54`) and on
  invalid instruction/attestation (`ots.go:175/254/259`). The only place `seq.Compute` runs is the
  per-sequence `upgrade(...)` call, so wrap THAT. Prefer a tiny helper
  `func safeUpgrade(upgrade seqUpgrade, ctx context.Context, seq opentimestamps.Sequence, digest []byte)
  (opentimestamps.Sequence, error)` with the `defer recover()` so `buildUpgrader`'s loop stays readable
  and the recover is unit-targetable; the helper takes the injected `seqUpgrade`, so a fake that
  `panic(...)`s exercises the recover offline. On a recovered panic return
  `(nil, fmt.Errorf("otsclient.Upgrade: upgrade sequence panicked: %v", r))`; the loop's existing
  `if err != nil { return follower.UpgradeResult{}, fmt.Errorf("otsclient.Upgrade: upgrade sequence:
  %w", err) }` then wraps it — `OTSTick` records a back-off, never a crash.
- **Per-request timeout (defect 2).** Derive `ctx2, cancel := context.WithTimeout(ctx, upgradeTimeout)`
  immediately before each `upgrade(ctx2, seq, file.Digest)` and call `cancel()` per loop iteration —
  use an explicit `cancel()` (NOT a bare `defer` inside the loop: the classic defer-in-loop leak; if
  routed through `safeUpgrade`, `defer cancel()` inside that helper is clean). Add an `upgradeTimeout`
  const (e.g. `30 * time.Second`) with an evergreen docstring noting it is a best-effort transport bound
  (the value is a comment, not a gate). Production `UpgradeSequence` honors `ctx` deadlines on its
  calendar GET, so the bounded context is what makes a stalled GET return rather than hang.
- **Preserve the load-bearing invariants** (`learnings/otsclient.md`): upgrade ONLY
  `file.GetPendingSequences()` and keep `file.GetBitcoinAttestedSequences()` verbatim (do not
  reorder/regress — confirmed fixtures break otherwise); the closure still takes `store.OTSRecord` by
  value and opens no DB handle; `internal/otsclient` `database/sql` dep count stays 1 (the
  `store.OTSRecord` seam) and it stays NON-WASM-pure (no WASM-shared leaf imports it).
- **Tests (offline, mutation-targeted), added to `client_test.go`:**
  1. A `seqUpgrade` fake that `panic("op not implemented")` plus a fixture WITH a pending sequence (so
     `GetPendingSequences()` is non-empty and the closure reaches `safeUpgrade`). Assert
     `buildUpgrader(panicSeq)(ctx, rec)` returns a non-nil error whose message contains the
     upgrade-sequence wrap AND does NOT panic the test binary. (Pick a fixture with a pending sequence —
     the bundled calendar-only / `merkle1` vectors; if none parse with a pending sequence, construct one
     by reading a pending fixture or via `opentimestamps.Stamp` against the fake — confirm by inspecting
     the `client_test.go` fixtures first.)
  2. A `seqUpgrade` fake that captures the `ctx` it is called with; after driving the closure over a
     pending-sequence fixture, assert the captured `ctx.Deadline()` returns `ok == true` (a bounded
     deadline was applied). Reverting to the bare `ctx` makes this FAIL.
- **Gate discipline.** The recover is an FFI-boundary guard with a comment that re-surfaces the panic as
  a returned error — keep it that way; do not weaken any gate. The store-uncoupling / WASM / pending-only
  invariants must be re-verified, not assumed.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty
  excluding `cauldron/`).
- `go test -count=1 ./internal/otsclient` passes, including the two new tests.
- Mutation (reverted after): removing the `safeUpgrade` `recover()` makes the unimplemented-op test
  PANIC/FAIL; restoring → green.
- Mutation (reverted after): replacing `context.WithTimeout(ctx, upgradeTimeout)` with the bare `ctx`
  makes the deadline-assertion test FAIL; restoring → green.
- The already-confirmed fixtures still classify confirmed at their exact heights
  (`hello-world`→358391, `empty`→129405): `go test -count=1 -run TestUpgradeAlreadyConfirmed
  ./internal/otsclient` passes (the pending-only-upgrade invariant held; `failUpgrade` never fires).
- Isolation holds: `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge`
  succeeds and `go list -deps ./internal/badge | grep -c -e 'internal/ots' -e 'internal/otsclient'`
  == 0; `go list -deps ./internal/otsclient | grep -c '^database/sql$'` == 1 (the `store.OTSRecord`
  seam, unchanged).

## Done When
`internal/otsclient`'s production upgrade path returns a wrapped fail-closed error on a library panic
and bounds each calendar request with a timeout, both mutation-proven, with `mise run check` green and
the confirmed-fixture verdicts unchanged.
