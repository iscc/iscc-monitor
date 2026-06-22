# Next Work Package

## Step: OTS daily stamp pass — record each distinct accepted root through `RecordOTS` in PollHub

## Advances
The **OTS / Bitcoin anchoring** milestone (`target.md`):

> stamp each distinct observed root daily (`UNIQUE(hub, tree_size, root)`) + background upgrade loop
> (pending → Bitcoin-confirmed) + serve `.ots`; **never blocks the follower**. **Verify:** a stamped
> root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client.

This step builds the **stamp half** of that criterion — the first production caller of the
already-landed `internal/store/ots.go` CRUD seam. It does not close the full milestone Verify on its
own (that needs the upgrade loop + `.ots` route + the `opentimestamps` dependency, listed under Not In
Scope), but it is the necessary, dependency-free next sub-step on the same arc that the `review`
handoff explicitly named as `**Next:**`. No `critical`/`normal` issue preempts milestone work; the 3
open `normal` issues are all "fix-on-next-touch" items in surfaces this step does not touch.

## Goal
On the verified, non-violation `PollHub` path, write each newly-accepted checkpoint root through
`store.RecordOTS` (status `pending`), so every distinct observed root is recorded for later
OpenTimestamps stamping. The write is a local SQLite insert with no calendar/Bitcoin I/O, so it never
blocks the follower; `RecordOTS`'s `UNIQUE(hub, tree_size, root)` dedupe makes re-polls of the same
root a silent no-op (the "stamp each distinct root once" semantics).

## Scope
- **Create**: (none)
- **Modify**:
  - `/workspace/iscc-monitor/internal/follower/follower.go` — add a small `stampRoot` helper and call
    it on the verified, non-violation path of `PollHub` (after `AdvanceAccepted` / `cacheHubKey` /
    `fsckMirror`, before the final `recordVerdict`), mirroring the coverage/key-cache placement.
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` — extend
    `TestPollHubVerifiedAdvances` (or add a sibling `TestPollHubStampsRoot`) asserting the `ots` row is
    written once and de-duped on re-poll, plus assertions that the non-verified / frozen / fork paths
    write **zero** `ots` rows (use the existing `countRows(t, path, "ots")` inspector).
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/ots.go` — `RecordOTS(ctx, OTSRecord) (int64, bool, error)`
    signature, the `OTSStatusPending` const, and the `OTSRecord` field names (`HubID int64`,
    `TreeSize uint64`, `Root []byte`, `Status string`, `StampedAt time.Time`). `OTSForRoot` is the
    read-back used by the test assertion.
  - `/workspace/iscc-monitor/internal/follower/follower.go` `PollHub` (lines 209-245) — the
    verified-advance block (`AdvanceAccepted` → `cacheHubKey` → `fsckMirror` → `recordVerdict`) is
    where the stamp call slots in; `freeze` (lines 414-451) and the frozen-clean short-circuit
    (lines 204-207) are the paths that must NOT stamp. `fsckMirror` (lines 375-380) is the helper shape
    to copy.
  - `/workspace/iscc-monitor/internal/follower/follower_test.go` `TestPollHubVerifiedAdvances`
    (line ~563) — the model for the new assertions (`countRows`, the second-poll-dedupe pattern,
    `m.size`/`observedAt`, `noopAlert`, `metrics.New()`).
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` (the "OTS-table CRUD seam" bullet) and
    `/workspace/iscc-monitor/.claude/context/learnings/follower.md` (PollHub composition + placement
    discipline). Read both before writing.

## Not In Scope
- **No `nbd-wtf/opentimestamps` dependency, no calendar HTTP, no stamping of actual `.ots` bytes.**
  This step records the root as `pending` with empty `ots_bytes`; obtaining the OpenTimestamps proof
  is the *upgrade loop*, the next sub-step. Keep `go.mod`/`go.sum` byte-unchanged.
- **No background upgrade loop** (`PendingOTS` → calendar → `MarkOTSUpgraded`) — that pulls in the
  dependency and exercises `Attempts`/`NextRetry`; it is the step after this one.
- **No `.ots` HTTP route and no certificate §5 (`HasClause5`)** — both are downstream consumers of
  `OTSForRoot`, gated on the upgrade loop producing confirmed bytes.
- **No `Attempts`/`NextRetry` retry-policy logic** — those columns stay zero; their wiring belongs to
  the upgrade loop.
- **Do not touch `internal/store`** — the CRUD seam is complete; this step only *calls* it. Keep store
  a `net/http`-free leaf (no edit to `ots.go`, `schema.sql`, or any store file).
- **No "true daily cadence" timer.** `RecordOTS`'s `(hub, tree_size, root)` dedupe already makes a
  distinct root stamped exactly once regardless of poll frequency; a calendar-day scheduler is
  unnecessary plumbing (YAGNI) for the milestone Verify. Do not add a clock-based gate.
- Do not fold in the open `host:port` DID / ForceQuery / §6-timestamp `normal` issues; they wait for a
  step that touches their surfaces.

## Implementation Notes
- **Placement is the correctness story** (same discipline as coverage/key-cache, see
  `learnings/follower.md`): stamp ONLY on the verified, non-violation advance path. Add the call AFTER
  `fsckMirror` succeeds (so a mirror-rebuild fault aborts the poll before stamping) and BEFORE the
  final `recordVerdict(m, hubID, status, false, observedAt)` at follower.go:244. Do NOT stamp inside
  `freeze`, on the frozen-clean short-circuit (`if fs.Frozen { … return status, nil }`), or on the
  non-verified early return — the existing fork/shrink/unverified/frozen tests must continue to see
  `countRows(…, "ots") == 0`.
- **The helper is tiny** — model it on `fsckMirror`:
  ```
  func stampRoot(ctx context.Context, st *store.Store, hubID int64, info logclient.CheckpointInfo, observedAt time.Time) error
  ```
  building `store.OTSRecord{ HubID: hubID, TreeSize: info.TreeSize, Root: info.Root[:], Status:
  store.OTSStatusPending, StampedAt: observedAt }` and calling `st.RecordOTS`. Discard the
  `(id, inserted, err)` return except the error; wrap a non-nil error with
  `follower.PollHub: hub %d: stamp root: %w` (matching the sibling call sites) and surface it.
  `info.Root` is `[32]byte`, so pass `info.Root[:]` exactly as `CheckpointRecord.Root` does at
  follower.go:212. A stamp fault is a genuine store fault (NOT a self-consistency violation): surface
  it to the caller; the accepted state is already committed by `AdvanceAccepted`, so the next poll
  re-stamps via the idempotent dedupe.
- **"Never blocks the follower" (the always-loaded learnings rule + the milestone clause)** is
  satisfied structurally here because `RecordOTS` is a local SQLite insert — no network I/O. Keep it
  that way: do not add calendar/HTTP work to this path. (The future upgrade loop runs in its own
  goroutine off `PendingOTS`, never on the poll path.)
- **Idempotency / dedupe**: a re-poll of the same accepted `(hub, tree_size, root)` returns
  `inserted=false` from `RecordOTS` (its `ON CONFLICT … DO NOTHING` dance) — a clean no-op, so a
  second verified poll at the same size must NOT add a second `ots` row, and the first stamping's
  `StampedAt` is preserved (the conflict path never rewrites it). Assert this (one `ots` row after two
  polls), mirroring how `TestPollHubVerifiedAdvances` asserts `hub_keys == 1` after a second poll.
- **Store stays a leaf** — the follower already imports `store`; this adds no new import to either
  package. `RecordOTS` carries `Status`/`ots_bytes` as opaque values, so no anchoring/OTS package
  enters the closure (the seam was built exactly for this, per `learnings/store.md`).
- **Oracle/conformance gate is N/A** for this slice — it records an opaque pending row over an already
  fsck-verified accepted root; no signature/RFC-6962/Merkle/did:web/proof path is added or changed.
  Confirm the existing fsck/inclusion conformance tests stay green (they run under `mise run check`).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty,
  excluding gitignored `cauldron/`).
- `go test -count=1 -run TestPollHub ./internal/follower` passes (verified-advance, fork, shrink,
  unverified, frozen-clean, and equivocation tests all stay green).
- A verified `PollHub` over `buildVerifiedMirror` writes exactly one `ots` row:
  `countRows(t, path, "ots") == 1` after the first poll, and **still** `== 1` after a second verified
  poll at the same root (dedupe).
- The recorded row is `pending`: a `store.OTSForRoot(ctx, hubID, m.size, root)` lookup returns
  `found == true` with `Status == store.OTSStatusPending` (and empty `OTSBytes`).
- The non-verified, fork, shrink, and frozen-clean paths write zero `ots` rows
  (`countRows(t, path, "ots") == 0` — extend an existing test on each path or add a focused assertion).
- `git diff --stat` shows `go.mod` / `go.sum` / `internal/store/*` byte-unchanged (only
  `internal/follower/follower.go` + `follower_test.go` touched).

## Done When
`PollHub` records each distinct accepted root once as a `pending` `ots` row on the verified
non-violation path (and never on a freeze / non-verified / frozen-clean path), the dedupe and
zero-row assertions pass, and `mise run check` is green with `go.mod` / `go.sum` / `internal/store`
untouched.
