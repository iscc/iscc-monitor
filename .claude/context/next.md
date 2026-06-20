# Next Work Package

## Step: Pure fork-trigger detection (`CheckFork`) in `internal/logclient`

## Goal
Land the second of M1's three RFC-6962 self-consistency triggers — fork (same `tree_size`,
*different* `root`) — as a pure, golden-testable verdict, the dep-free sibling of `CheckShrink`.
This brings M1 one trigger closer to the freeze behavior while keeping `transparency-dev/merkle`
and tile fixtures deferred to the single equivocation slice that genuinely needs them.

## Goal-fit (state → target gap)
M1's first Verify half (`origin` / `verifierKey` / single-poll) is met; the dominant remaining half
is *synthetic fork/shrink/equivocation → correct `violations.kind` + `frozen=1` + exactly one alert
+ other hubs unaffected + evidence survives restart*. Shrink landed last iteration; fork is the next
trigger detectable with **zero new deps and no new fixtures** — a same-size, different-root array
comparison. Only equivocation (RFC-6962 consistency-proof failure) genuinely needs
`transparency-dev/merkle` + tile fixtures, so it stays deferred to its own slice. This builds on the
existing `CheckShrink` slice and the `CheckpointInfo.Root [rootBytes]byte` type — not skipping ahead
into Merkle math.

## Scope
- **Create**: (none — extend the existing consistency files)
- **Modify** (0 of ≤3 non-test/doc files; the doc comment update rides on the same file):
  - `/workspace/iscc-monitor/internal/logclient/consistency.go` — add `ViolationFork ViolationKind =
    "fork"` const and the pure `CheckFork` function; update the file/package doc prose so fork is no
    longer listed among the deferred merkle-backed triggers (only equivocation remains deferred).
  - `/workspace/iscc-monitor/internal/logclient/consistency_test.go` — add table-driven `TestCheckFork`
    plus a `ViolationFork` kind-string assertion (test file, not counted against the budget).
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/consistency.go` — the existing `CheckShrink` /
    `ViolationShrink` slice to mirror exactly (the same shape this step extends).
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — `CheckpointInfo.Root [rootBytes]byte`
    (line 65); the verified root the follower will eventually feed `CheckFork`.
  - `/workspace/iscc-monitor/internal/logclient/verify.go` — `const rootBytes = 32` (line 35); the
    root array type.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `Violation.Kind` doc (line 157) and
    `RecordViolation` / `Freeze` confirm `"fork"` is the exact `violations.kind` string this const
    must equal (also asserted in `store/checkpoints_test.go`). Do not call them here.
  - `/workspace/iscc-monitor/.claude/context/handoff.md` — the review `**Next:**` block specifying
    this exact step.
  - `/workspace/iscc-monitor/.claude/context/learnings.md` — "A self-consistency violation freezes,
    never crashes (ADR-0006). Three triggers — fork / shrink / equivocation."

## Not In Scope
- **Do NOT wire `CheckFork` / `CheckShrink` into `follower.PollHub`** (mapping `LastSize`/last-root →
  prev, `CheckpointInfo` → next, then `RecordViolation` + `Freeze` + alert-once). That is the
  explicitly separate composition slice that follows, exactly as `CheckShrink` was left unwired.
- **Do NOT persist the last-observed root** in `follow_state` / `FollowState` — no `LastRoot` column
  or struct field. The follower wiring slice decides how the prior root is sourced (likely a
  `checkpoints` lookup at the prior size); this pure unit just takes two roots as parameters.
- **Do NOT add the `transparency-dev/merkle` dependency** or any new dep — `go.mod`/`go.sum` must stay
  byte-identical. Fork needs no Merkle math.
- **Do NOT implement the equivocation trigger**, add tile / entry-bundle fixtures, the alert-once
  mechanism, coverage, the poll loop, `/metrics`, structured logs, or any `cmd/` binary — each is its
  own later step.
- **Do NOT touch the store, `accept.go`, `verify.go`, or any did:web / verify code** — the change is
  confined to `consistency.go` + its test.

## Implementation Notes
- Mirror `CheckShrink` precisely. Suggested signature: `func CheckFork(prevSize uint64, prevRoot
  [rootBytes]byte, nextSize uint64, nextRoot [rootBytes]byte) bool`. A fork is
  `prevSize > 0 && nextSize == prevSize && nextRoot != prevRoot`.
- The `[rootBytes]byte` (i.e. `[32]byte`) array is directly comparable with `==`/`!=` in Go — do
  **not** reach for `bytes.Equal`; that would force a `bytes` import and break the slice's whole point
  of staying import-free of any new dep (the `CheckShrink` file imports nothing).
- The `prevSize > 0` guard is load-bearing for the *same* reason as in `CheckShrink`: a fresh-store
  `FollowState{}.LastSize == 0` (and its zero `[32]byte` root) must never read as a fork. With
  `prevSize == 0`, return false regardless of roots.
- Equal size is the fork trigger's concern (the existing `consistency.go` doc already says
  "next == prev is … the fork trigger's concern, not a shrink"); a strict decrease stays shrink's
  concern and growth (`nextSize > prevSize`) is neither — `CheckFork` returns false for both, even
  when the roots differ.
- Identical roots at equal size is a benign re-observation (`store.RecordCheckpoint` dedupes it on
  `UNIQUE(hub_id, tree_size, root)`), so `nextRoot == prevRoot` → false. Only a *differing* root at
  the *same* committed size is the split-of-history a monotonic append-only log can never legitimately
  present.
- `CheckFork` never panics or errors on any input — a violation freezes, never crashes (ADR-0006); it
  is a pure boolean verdict like `CheckShrink`.
- Add `const ViolationFork ViolationKind = "fork"` next to `ViolationShrink`; it must equal the string
  `"fork"` already asserted in `store/checkpoints_test.go` and named in the `Violation.Kind` doc. Keep
  the `ViolationKind` string type so `store` stays import-free of `logclient`.
- Update the package/file doc comment so it no longer lists fork among the deferred merkle-backed
  triggers: state that fork has landed dep-free (same-size, different-root array compare) and only
  equivocation (RFC-6962 consistency-proof failure) still awaits `transparency-dev/merkle` + tiles.
- **Relevant Correctness rule (learnings.md, ADR-0006):** a self-consistency violation *freezes,
  never crashes*. This slice adds the *fork* verdict only; it remains an intentional
  unused-until-wired export seam — do not flag `CheckFork` / `ViolationFork` as dead code (`go vet` is
  clean and the follower wiring is a later slice). The conformance/oracle gate is correctly N/A here:
  this is a pure array/size comparison touching no signature, RFC-6962 proof, didweb, or merkle code
  (that gate trips only when the equivocation slice lands).
- Tests: a non-vacuous table with both true and false cases. Build distinct `[rootBytes]byte` literals
  (e.g. one all-`0x01`, one all-`0x02`, and the zero array) so "different root" is genuinely different
  and no case is satisfied vacuously by the zero value. Style: evergreen docstrings, short
  single-purpose functions, no `t.Skip` / `//nolint` / swallowed errors / build tags.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass on go1.24).
- `gofmt -l internal/logclient/consistency.go internal/logclient/consistency_test.go` prints nothing.
- `go test -count=1 -run TestCheckFork ./internal/logclient` passes with both true and false subtests.
- `string(logclient.ViolationFork) == "fork"` (asserted by a test).
- Asserted by the table tests (each a distinct case so none is vacuous), with `rootA != rootB` and
  `zeroRoot` the zero `[rootBytes]byte`:
  - `CheckFork(10183, rootA, 10183, rootB) == true` (same size, different root).
  - `CheckFork(10183, rootA, 10183, rootA) == false` (identical root — re-observation).
  - `CheckFork(10183, rootA, 10182, rootB) == false` (shrink — shrink's concern, not fork).
  - `CheckFork(10183, rootA, 10184, rootB) == false` (growth, even with differing roots).
  - `CheckFork(0, zeroRoot, 5, rootB) == false` (fresh-store `prevSize == 0` guard).
- `git status --short go.mod go.sum` is empty (no dependency added) and
  `go list -m github.com/transparency-dev/merkle` still reports "not a known dependency".

## Done When
`internal/logclient` exports a pure `CheckFork` plus a `ViolationFork` const, the package doc reflects
fork-landed / equivocation-deferred, and every Verification criterion above passes with `mise run
check` green and `go.mod` / `go.sum` untouched.
