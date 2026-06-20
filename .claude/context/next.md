# Next Work Package

## Step: Wire CheckShrink + CheckFork into follower.PollHub (freeze + alert-once)

## Goal
Turn the two landed pure verdicts (`CheckShrink`/`CheckFork`) into M1's freeze behavior: when a
verified observation contradicts the prior accepted checkpoint for the same hub, record the violation,
freeze the hub, and alert exactly once — without advancing the cursor. This converts detection into the
ADR-0006 freeze, the largest remaining piece of M1's Verify criteria.

## Goal-fit (state → target gap)
M1's first Verify half (`origin` / `verifierKey` / single-poll) is met; the dominant remaining half is
*synthetic fork/shrink/equivocation → correct `violations.kind` + `frozen=1` + exactly one alert +
other hubs unaffected + evidence survives restart*. Shrink and fork are both landed pure verdicts but
unwired; `RecordViolation`/`Freeze` are landed persistence seams but uncalled. This slice composes them
in `follower.PollHub` — the exact step the review handoff `**Next:**` names. It builds directly on the
existing single-observation `PollHub` and the two pure triggers; the merkle-backed equivocation trigger
stays deferred to its own slice (no skipping ahead into Merkle math).

## Scope
- **Create**: (none)
- **Modify** (2 of ≤3 non-test/doc files):
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — add ONE read method returning the
    persisted `(root, raw)` for a given `(hubID, treeSize)`. `follow_state` deliberately does not
    persist `LastRoot`, so the fork check needs the prior root and the violation evidence needs the
    prior raw bytes; both come from a `checkpoints` lookup at the prior accepted size.
  - `/workspace/iscc-monitor/internal/follower/follower.go` — wire the consistency check + freeze +
    alert-once into `PollHub` and update the file/package doc prose (the freeze/alert step it currently
    lists as "a later step" lands here).
- **Modify (tests/docs, not counted against the ≤3 budget)**:
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` — add synthetic shrink + fork tests.
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — test the new read method
    (round-trip + absent-row).
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/consistency.go` — `CheckShrink(prev, next)`,
    `CheckFork(prevSize, prevRoot, nextSize, nextRoot)`, `ViolationShrink`/`ViolationFork`, and the
    `prevSize > 0` guard rationale.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `RecordViolation`, `Freeze`,
    `FollowState` (carries `LastSize` + `Frozen`), `RecordCheckpoint`, `AdvanceFollowState`, and
    `Violation{HubID, Kind, RawA, RawB, ProofJSON, DetectedAt}`.
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — `AcceptCheckpoint` returns
    `(Status, CheckpointInfo, error)`; `CheckpointInfo{TreeSize, Root [rootBytes]byte}` is the zero
    value on every non-verified verdict (lines 57–66, 84–105).
  - `/workspace/iscc-monitor/internal/follower/follower.go` — current `PollHub`
    (record-only-on-verified, err-before-status; lines 45–79).
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` — the `compositeFetcher` pattern and
    the existing two tests whose `PollHub` call signature this slice changes.
  - `/workspace/iscc-monitor/internal/store/schema.sql` — `checkpoints` columns (`root`, `raw`,
    `tree_size`) and `follow_state.frozen` (lines 42–56, 115–123).
  - `/workspace/iscc-monitor/.claude/context/learnings.md` — the ADR-0006 freeze rule and the
    "intentional unused-until-wired export seam" notes for the two triggers and the freeze path.

## Not In Scope
- The **equivocation** trigger (RFC-6962 consistency-proof failure across growing sizes) — needs
  `transparency-dev/merkle` + tile fixtures and trips the oracle gate; keep it deferred to its own slice.
- The **poll loop / single-writer goroutine wrapper** — `PollHub` stays one observation per call.
- The **`hub_keys` did:web cache write** and the stale-sb1-fixture refresh — a separate slice.
- A real alerting transport (email/webhook/log sink) — inject a minimal alert seam (a func field or a
  1-method interface) so the test counts invocations; do NOT build delivery here.
- Backed-off evidence-only re-polling cadence of a frozen hub — record the freeze now; cadence is the
  poll-loop slice.
- Persisting `consistent`/`root_rebuilt` on `checkpoints` — those wait for the merkle/fsck path.
- Adding any dependency — `go.mod`/`go.sum` must stay byte-identical.

## Implementation Notes
- **New store method** (suggested `CheckpointAt(ctx, hubID int64, treeSize uint64) (root []byte, raw
  []byte, found bool, err error)`): `SELECT root, raw FROM checkpoints WHERE hub_id=? AND tree_size=?
  LIMIT 1`. On `sql.ErrNoRows` return `found=false` with a **nil** error — mirror `FollowState`'s
  "absent row is not an error" convention. Keep `store` a leaf: return `[]byte`, never a logclient
  type; the follower copies the `[]byte` root into a `[rootBytes]byte` at the call site.
- **PollHub ordering** — on a `StatusVerified` observation, run the consistency check BEFORE
  `RecordCheckpoint`/`AdvanceFollowState`:
  1. `fs, _ := st.FollowState(ctx, hubID)` → `prevSize := fs.LastSize`, `wasFrozen := fs.Frozen`.
  2. If `prevSize > 0`, call `CheckpointAt(ctx, hubID, prevSize)` → `prevRoot`/`prevRaw`/`prevFound`.
     Copy `prevRoot` into a local `[rootBytes]byte` (via `copy`) for `CheckFork`. If `prevFound == false`
     (a hub advanced before this slice existed, leaving no stored root at that size), skip the fork
     check but still run the size-only shrink check.
  3. `shrink := logclient.CheckShrink(prevSize, info.TreeSize)`;
     `fork := prevFound && logclient.CheckFork(prevSize, prevRootArr, info.TreeSize, info.Root)`.
  4. On a true verdict, pick `kind := logclient.ViolationShrink` when `shrink`, else
     `logclient.ViolationFork`. Call `st.RecordViolation(ctx, store.Violation{HubID: hubID, Kind:
     string(kind), RawA: prevRaw, RawB: raw, DetectedAt: observedAt})`, then `st.Freeze(ctx, hubID)`.
     Persist the contradictory checkpoint as evidence (`RecordCheckpoint` for `raw`) but **do NOT**
     `AdvanceFollowState` — a frozen hub does not advance accepted state. Return the verdict status
     (still `StatusVerified` — the signature was valid; the violation is a separate axis) with a **nil**
     error: a violation freezes, never crashes (ADR-0006).
  5. **Alert-once**: fire the injected alert only when `!wasFrozen` (the not-frozen → frozen
     transition). A later poll of an already-frozen hub that re-detects MUST record the violation again
     (re-detection is itself evidence — `RecordViolation` has no `ON CONFLICT`) but MUST NOT re-alert.
     This is the load-bearing exactly-one-alert rule from `target.md` and the ADR-0006 learning.
  6. No violation → keep the existing `RecordCheckpoint` → `AdvanceFollowState` path unchanged.
- **Alert seam**: add a minimal injected sink to `PollHub`'s signature — prefer a func field/parameter
  (e.g. `alert func(hubID int64, kind string)`) over an interface for YAGNI. Both existing tests must be
  updated to pass a no-op or counter; that signature change is the reason `follower_test.go` is touched.
- **Correctness rules in play (learnings.md / ADR-0006)**: the `prevSize > 0` guard already lives in
  both `CheckShrink`/`CheckFork` — rely on it (do not duplicate) so a fresh-store `LastSize == 0` is
  never misread. A violation **freezes, never crashes** — every branch returns `(status, nil)` on a
  true verdict, never a panic/error. `store` stays import-free of `logclient` (return `[]byte`, convert
  in the follower). No auto-unfreeze: `AdvanceFollowState` already omits `frozen` from its conflict
  update, and this slice never clears it.
- **Shrink/fork are mutually exclusive by size** (`next < prev` vs `next == prev`), so order is
  immaterial — evaluate shrink first and use its kind when true to keep the mapping obvious.
- Style: short single-purpose functions, evergreen docstrings, no `t.Skip` / `//nolint` / swallowed
  errors / build tags. The `err`-before-status contract on `AcceptCheckpoint`/`FetchCheckpoint` stays
  intact (the existing fault paths are untouched).
- **Conformance/oracle gate**: N/A for this slice — it composes pure size/root verdicts and store CRUD,
  touching no signature, RFC-6962 proof, didweb, or merkle code. That gate trips only when the
  merkle-backed equivocation slice lands.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 -run TestPollHub ./internal/follower` passes (existing verified-advances +
  unverified-no-advance tests still green after the signature change).
- New `go test -count=1 -run TestPollHubFork ./internal/follower`: a second verified observation at the
  same `tree_size` with a different root yields a `violations` row with `kind == "fork"`, `frozen == 1`,
  the cursor does NOT advance past the prior size, and the alert fired exactly once.
- New `go test -count=1 -run TestPollHubShrink ./internal/follower`: a verified observation at a
  strictly smaller `tree_size` than the prior accepted size yields a `violations` row with
  `kind == "shrink"`, `frozen == 1`, and exactly one alert.
- Assertion: a third poll of the already-frozen hub records another `violations` row (re-detection is
  evidence) but the alert count stays at 1 (exactly-one-alert across re-detection).
- Assertion: a second registered hub polled with a clean verified checkpoint advances normally and
  stays `frozen == 0` (other hubs unaffected).
- Assertion: after re-opening the store from the same path, `FollowState(...).Frozen == true` for the
  frozen hub (freeze survives restart) and the violation row is still present.
- New `go test -count=1 -run TestCheckpointAt ./internal/store`: round-trips a recorded `(root, raw)`
  and returns `found == false` with a nil error for an absent `(hubID, treeSize)`.
- `git status --short go.mod go.sum` is empty (no dependency added).

## Done When
`PollHub` records the violation + freezes + alerts exactly once on a true shrink/fork verdict (without
advancing, surviving restart, other hubs unaffected) and every Verification criterion above passes with
`mise run check` green.
