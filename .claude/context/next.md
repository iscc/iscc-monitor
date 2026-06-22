# Next Work Package

## Step: OTS store seam — `ots`-table CRUD (`RecordOTS` / `OTSForRoot` / `PendingOTS` / `MarkOTSUpgraded`)

## Advances
The **OTS / Bitcoin anchoring** milestone (`target.md`):

> stamp each distinct observed root daily (`UNIQUE(hub, tree_size, root)`) + background upgrade loop
> (pending → Bitcoin-confirmed) + serve `.ots`; **never blocks the follower**. **Verify:** a stamped
> root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client.

This milestone is too large for one ≤3-file step (it spans the `nbd-wtf/opentimestamps` dependency, a
stamp loop, a background upgrade loop, an `.ots` HTTP route, and certificate §5). Per the skeleton-first
rule, this step lays the **foundational, fully store-testable seam every later sub-step reads/writes
through**: the `ots`-table CRUD on the *already-present* schema table (`internal/store/schema.sql:128-142`).
It also unblocks the M-UI tail's last open clause — **certificate §5 BITCOIN ANCHOR** (`HasClause5`,
today hard-false because "the OTS store seam does not exist", `internal/certificate/handler.go:187-188`)
reads its `(size, root) → ots row` from exactly this seam.

## Goal
Add typed `ots`-table CRUD methods to `internal/store` so the (later) stamp loop can persist a stamped
root, the upgrade loop can read pending rows and mark them Bitcoin-confirmed, and the `.ots` route +
certificate §5 can read an anchor state for a given `(hub, tree_size, root)`. Pure SQLite leaf work,
golden-tested in isolation, with **no new dependency** — `ots_bytes` is an opaque BLOB and `status` an
opaque string, so this step does not touch the crypto/proof path or the `opentimestamps` library.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/store/ots.go` — the OTS-table CRUD methods + the
  `OTSRecord` struct + the two status consts.
- **Create**: `/workspace/iscc-monitor/internal/store/ots_test.go` — golden round-trip + dedupe +
  pending-filter + upgrade tests (test file, package `store`).
- **Modify**: *(none — `schema.sql` already defines the `ots` table; do NOT re-edit it).*
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — port the exact idioms: `RecordCheckpoint`
    (`ON CONFLICT(hub_id,tree_size,root) DO NOTHING` + `RowsAffected()` first-sighting dance,
    lines 113-141), `CheckpointAt` (`(…, found bool, err error)` absent-is-not-an-error read,
    lines 157-169), `ListViolations` (newest-first `[]T` leaf read with `sql.NullInt64` time scans,
    lines 303-332), and `unixOrNil` / `nullStringOrNil` (lines 487-501).
  - `/workspace/iscc-monitor/internal/store/schema.sql` lines 128-142 — the `ots` table columns this
    seam reads/writes.
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — how the existing suite opens a
    test store (`t.TempDir()`) and seeds a hub via `UpsertHub`; reuse that helper shape.
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — the store-leaf rules (single writer,
    no `net/http` in the closure, `unixOrNil` zero-time→NULL, `RowsAffected`-after-`DO NOTHING` dedupe,
    `uint64→int64` casts, the literal-drift trap that motivates the status consts).
  - `/workspace/iscc-monitor/.claude/context/learnings.md` — Correctness rule: **"OTS never blocks the
    follower"** (this seam is a plain leaf write; it imposes no follower-loop coupling and opens no
    second connection).

## Not In Scope
- **No `nbd-wtf/opentimestamps` dependency, no stamping, no calendar HTTP, no upgrade-from-Bitcoin
  logic.** `ots_bytes` is stored/returned verbatim as an opaque `[]byte`; `status` is an opaque string.
  Real OTS serialization + calendar round-trips are a later sub-step.
- **No follower wiring / stamp loop.** Do not call these methods from `internal/follower` or
  `cmd/iscc-monitor` in this step; that is the next sub-step (a daily stamp pass that never blocks poll).
- **No `.ots` HTTP route** (`internal/tilesserve`/`proofserve`) and **no certificate §5 rendering**
  (`internal/certificate`). Those consume this seam in later sub-steps.
- **Do not touch the production decoder, crypto, or proof code** — the oracle/conformance gate is N/A
  for this slice (plain CRUD + BLOB round-trip, no signature/RFC-6962/Merkle/did:web path).
- Do not edit `schema.sql` — the `ots` table already exists with the right columns and `UNIQUE` key.
- Do not fold in the open `host:port` DID / ForceQuery / §6-timestamp `normal` issues; they wait.

## Implementation Notes
- **Struct.** Add `OTSRecord` mirroring `CheckpointRecord`'s plain-leaf shape: `HubID int64`,
  `TreeSize uint64`, `Root []byte`, `Status string`, `OTSBytes []byte`, `CalendarURLs string`,
  `StampedAt time.Time`, `UpgradedAt time.Time`, `BTCHeight int64`, `Attempts int64`,
  `NextRetry time.Time`. Keep `Status` an opaque string carried on the struct (same pattern as
  `CheckpointRecord.Status` / `Violation.Kind`) so `store` stays import-free of any anchoring package.
- **`RecordOTS(ctx, OTSRecord) (id int64, inserted bool, err error)`** — port `RecordCheckpoint`
  verbatim: `INSERT … ON CONFLICT(hub_id, tree_size, root) DO NOTHING`, then `RowsAffected()>0` →
  `LastInsertId()`/`inserted=true`, else SELECT the id back/`inserted=false`. This is the dedupe the
  milestone's `UNIQUE(hub, tree_size, root)` "stamp each distinct root once" criterion needs. Cast
  `TreeSize uint64 → int64` like `RecordCheckpoint`; write zero times via `unixOrNil`, empty
  `CalendarURLs` via `nullStringOrNil`.
- **`OTSForRoot(ctx, hubID int64, treeSize uint64, root []byte) (OTSRecord, bool, error)`** — port
  `CheckpointAt`'s shape: `sql.ErrNoRows → (OTSRecord{}, false, nil)` (an un-anchored root is a plain
  miss, **not** an error — certificate §5 must render the honest "pending / not-yet-anchored" state, not
  a 5xx). Read all nullable columns through `sql.NullInt64`/`sql.NullString` (NULL→zero), inverse of
  the write.
- **`PendingOTS(ctx) ([]OTSRecord, error)`** — the upgrade loop's read side: `SELECT … WHERE status = ?`
  (pending) `ORDER BY stamped_at ASC, id ASC` (oldest-first, fair upgrade order), a leaf `[]OTSRecord`
  read like `ListViolations`. A network with no pending rows returns an empty slice + nil err. Define a
  small package const for the pending status string (e.g. `OTSStatusPending = "pending"`) so the writer,
  this filter, and `MarkOTSUpgraded` share one source of truth — do **not** spell the literal at three
  sites (the store.md `note.$schema` literal-drift trap, applied to status strings).
- **`MarkOTSUpgraded(ctx, hubID int64, treeSize uint64, root []byte, otsBytes []byte, btcHeight int64,
  upgradedAt time.Time) error`** — a plain `UPDATE ots SET status = <confirmed>, ots_bytes = ?,
  btc_height = ?, upgraded_at = ? WHERE hub_id = ? AND tree_size = ? AND root = ?`. Use a second const
  (e.g. `OTSStatusConfirmed = "confirmed"`). Like `SetCoverage`, ignore `RowsAffected` (an idempotent
  re-mark is not an error). Write `upgradedAt` via `unixOrNil`.
- **Leaf purity.** No `internal/*` import, no `net/http`. Use only `context database/sql errors fmt
  time`. Reuse `unixOrNil`/`nullStringOrNil` from `checkpoints.go`; do **not** duplicate them.
- **Tests** (`ots_test.go`): open a store via the same `t.TempDir()` + `Open` helper
  `checkpoints_test.go` uses, then `UpsertHub` to get a `hub_id`. Assert: (1) round-trip `RecordOTS` →
  `OTSForRoot` returns the stored fields byte-equal (`Status`, `OTSBytes`, `CalendarURLs`, `BTCHeight`,
  and the times back as set); (2) absent `(hub,size,root)` → `OTSForRoot` `found=false`, nil err;
  (3) re-`RecordOTS` of the same `(hub,size,root)` → `inserted=false`, same id (dedupe); (4)
  `PendingOTS` returns only `pending`-status rows oldest-first, excluding an upgraded one; (5)
  `MarkOTSUpgraded` flips status + sets `ots_bytes`/`btc_height`/`upgraded_at` and the row then drops out
  of `PendingOTS`. Mutation-target each assertion (swapping `DO NOTHING`→plain insert breaks the dedupe
  id check; `ASC`→`DESC` breaks the pending order; dropping the `status` WHERE in `PendingOTS` includes
  the upgraded row).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty,
  excluding gitignored `cauldron/`).
- `go test -count=1 ./internal/store` passes (existing suite unaffected).
- `go test -count=1 -run TestOTS ./internal/store` passes (the new round-trip / dedupe / pending /
  upgrade cases).
- `go list -deps ./internal/store | grep '^net/http'` is empty (store stays a leaf; no `net/http` in
  the closure).
- `git diff --stat HEAD -- internal/store/schema.sql go.mod go.sum` is empty (no schema/dependency
  change — the `ots` table already existed and no new module was added).

## Done When
`internal/store` exposes golden-tested `RecordOTS` / `OTSForRoot` / `PendingOTS` / `MarkOTSUpgraded`
over the existing `ots` table, `mise run check` and `go test -run TestOTS ./internal/store` are green,
and the store remains a `net/http`-free leaf with `schema.sql`/`go.mod`/`go.sum` byte-unchanged.
