# Next Work Package

## Step: Pure shrink-trigger detection (`CheckShrink`) in `internal/logclient`

## Goal
Land the first of M1's three RFC-6962 consistency triggers as a pure, golden-testable unit: detect
when a hub presents a verified checkpoint whose `tree_size` is **smaller** than a previously-accepted
size for that hub (a "shrink" — the hub rewrote history backwards against this monitor). This is the
one trigger detectable with pure size arithmetic — no `transparency-dev/merkle` dep and no tile
fixtures — so it is the highest-leverage next increment toward the consistency check, and it composes
directly with the existing `CheckpointInfo` verdict type and the just-landed `RecordViolation` /
`Freeze` store seam (wiring into the follower is a deliberately separate later slice).

## Goal-fit (state → target gap)
M1's first Verify half (`origin` / `verifierKey` / single-poll) is met; the dominant remaining half is
*synthetic fork/shrink/equivocation → correct `violations.kind` + `frozen=1` + exactly one alert +
other hubs unaffected + evidence survives restart*. The full three-trigger check needs
`transparency-dev/merkle` (a network `go get`, not vendored in `cauldron/`) plus tile fixtures (absent
from `testdata/live/`) — too large and too dependency-heavy for one step. Following the handoff's
"detection-only over fixtures first" guidance, the smallest coherent slice that advances it *now* with
zero new deps and no new fixtures is the **shrink** trigger: a strict tree-size decrease, detectable
by pure `uint64` comparison. Fork (same size, different root) and equivocation (RFC-6962
consistency-proof failure) are the merkle-backed slices that follow once that dep and tile fixtures
land. This builds on the existing `CheckpointInfo` / `Status` types — not skipping ahead into Merkle
math.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/logclient/consistency.go` — the pure detection function.
- **Create**: `/workspace/iscc-monitor/internal/logclient/consistency_test.go` — table-driven tests
  (test file, not counted against the budget).
- **Modify**: (none — additive new file only; 0 of ≤3 non-test/doc files modified.)
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — `CheckpointInfo{Origin, TreeSize, Root}`
    and the `Status` taxonomy this composes with.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — the `Violation{HubID, Kind, RawA, RawB,
    ProofJSON, DetectedAt}` shape and `RecordViolation` / `Freeze` seam the *later* wiring step drives.
    Do not call them here; this step only classifies and supplies the `"shrink"` kind string.
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/log_tree.py` — confirms `tree_size` is the
    monotonic committed-record count a hub must never decrease (`current_tree_size`; the
    `old_size >= size` monotonic guard in `build_checkpoint`). A decrease against this monitor is the
    shrink violation.
  - `/workspace/iscc-monitor/.claude/context/learnings.md` — "A self-consistency violation freezes,
    never crashes (ADR-0006). Three triggers — fork / shrink / equivocation."

## Not In Scope
- **Do NOT add the `transparency-dev/merkle` dependency** (`go get`) — it is not in `cauldron/` and is
  not needed for a size-only shrink check. Fork and equivocation are separate later slices that land
  *with* the merkle dep and tile fixtures.
- **Do NOT wire detection into `follower.PollHub`** — no calls to `RecordViolation`, `Freeze`, or
  `FollowState` here. This step is a pure unit; the freeze/alert wiring slice follows separately.
- **Do NOT add tile or entry-bundle fixtures** to `testdata/live/` — unneeded for a size comparison.
- **Do NOT implement the alert-once mechanism, coverage, the poll loop, or any `cmd/` binary** — each
  is its own later step.
- **Do NOT touch the store, `accept.go`, or any did:web / verify code** — additive new file only.
- **Do NOT pre-build fork/equivocation kind constants or detectors** (YAGNI) — add only `"shrink"`.

## Implementation Notes
- Keep the function **pure**: `logclient` today imports only stdlib + `golang.org/x/mod/sumdb/note`;
  add **no new imports** (the shrink check needs none). Mirror the package's "pure verdict,
  dependency-injected" style from `accept.go`.
- Signature suggestion: `func CheckShrink(prev, next uint64) bool`, returning `true` only when
  `next < prev`. Take the two sizes (not whole `CheckpointInfo` structs) so the unit stays trivially
  table-testable; the future caller maps `FollowState.LastSize` → `prev` and `CheckpointInfo.TreeSize`
  → `next` at the call site, exactly as `PollHub` maps verdicts into store structs.
- If a named result reads clearer than a bare bool, a small `ViolationKind` string type with a single
  exported `ViolationShrink ViolationKind = "shrink"` constant is acceptable — but the literal kind
  string this slice owns is exactly `"shrink"`, matching the `violations.kind` value the M1 Verify
  criteria expect and the `store.Violation.Kind` field the persistence step will populate. Do not add
  fork/equivocation constants that have no detector yet.
- **Boundary discipline (the load-bearing edge cases, all must be pinned by tests):**
  - `next == prev` is **not** a shrink — it is a re-observation / candidate-fork, which is the
    fork-trigger's concern (out of scope here). Return `false`.
  - `next > prev` is normal growth → `false`.
  - `prev == 0` means "no size accepted yet" so **any** `next` is growth, never a shrink → `false`.
    The fresh-store zero (`FollowState{}.LastSize == 0`) must never be misread as a violation.
  - Only strict `next < prev` **with** `prev > 0` is a shrink → `true`.
- Correctness rule from `learnings.md` (ADR-0006): a violation *freezes, never crashes* — so this
  function returns a verdict for the caller to act on and must never panic or error on any `uint64`
  pair. Document in the file docstring that fork and equivocation are deferred to the merkle-backed
  slices, so the next scoper sees the seam.
- Style: evergreen docstring on the file and the function; short and single-purpose; no `t.Skip` /
  `//nolint` / swallowed errors / build tags.

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...` all exit 0 on go1.24).
- `gofmt -l internal/logclient/consistency.go internal/logclient/consistency_test.go` prints nothing.
- `go test -count=1 -run TestCheckShrink ./internal/logclient` passes with non-vacuous table cases.
- Asserted by the table tests (each a distinct case so none is vacuous):
  - `CheckShrink(10183, 10182) == true` (strict shrink).
  - `CheckShrink(10183, 10183) == false` (equal size — re-observation, not a shrink).
  - `CheckShrink(10183, 10184) == false` (growth).
  - `CheckShrink(0, 5) == false` (fresh store — no prior accepted size).
  - `CheckShrink(0, 0) == false`.
- `go list -deps ./internal/logclient | grep -c '^net/http'` is unchanged by the additive file, and
  `go list -m github.com/transparency-dev/merkle` still reports "not a known dependency" (no dep added;
  `go.mod` untouched).

## Done When
`internal/logclient/consistency.go` exposes a pure `CheckShrink` that classifies a strict tree-size
decrease (and only that, with `prev > 0`) as a shrink violation, its table tests pass the five
boundary assertions above, and `mise run check` is green with no new dependency added.
