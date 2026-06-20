# Next Work Package

## Step: Add `store.LookupHubKey` — the read side of the `hub_keys` did:web key cache

## Goal
The follower now *writes* `hub_keys` via `RecordHubKey`/`cacheHubKey`, but nothing can read it back, so
no consumer (offline verification, future serving) can consult the cache. Add a typed reader keyed on
`(hub_id, key_id)` so the write-only cache becomes usable. This is the cheapest open M1 slice (the
handoff `Next:` and `state.md`'s candidate ordering both put it first) and unblocks consulting the
cache without touching the crypto/merkle path.

## Scope
- **Modify**: `internal/store/checkpoints.go` — add one method `LookupHubKey(ctx, hubID int64, keyID
  uint32) (HubKey, bool, error)` (the only non-test/doc file changed).
- **Create**: tests for the reader — add `func Test…` cases to the existing
  `internal/store/checkpoints_test.go` (the package keeps one test file; do not create a new one).
  (Tests do not count against the ≤3 non-test/doc budget.)
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — port the shape from the sibling readers
    `FollowState` (lines ~172–192) and `Coverage` (lines ~286–310): the "absent row → zero value +
    `found=false` + nil error" convention, `sql.NullInt64`/`sql.NullString` scanning, and the
    `errors.Is(err, sql.ErrNoRows)` switch.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — the `HubKey` struct (lines ~71–78) is the
    return type; the `RecordHubKey` writer (lines ~324–352) plus `unixOrNil` (line ~356) and
    `nullStringOrNil` (line ~365) are the exact column/nullability contract to invert on read.
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — `TestRecordHubKeyInsert`/`Refresh`/
    `Rotation`/`Nullable` (lines ~719–921) for the `openTemp(t)` + `UpsertHub` + column-scan test idiom
    and the live key ids (`0x40b74463`, `0x069d0f14`, `0x22b08f3e`).
  - `/workspace/iscc-monitor/internal/store/schema.sql` — the `hub_keys` table (lines 33–40): no UNIQUE,
    `pubkey_z`/`revoked_at`/`resolved_at` nullable, `pubkey_raw` BLOB NOT NULL.

## Not In Scope
- Wiring `LookupHubKey` into the follower or any verification path — this step only adds the read method
  and its tests; the consumer is a later slice.
- The merkle-backed **equivocation** trigger, `transparency-dev/merkle`, or tile fixtures (the large
  remaining M1 gap — needs its own decomposition and trips the oracle gate; not this slice).
- The sb1 fixture refresh (`22b08f3e`→`069d0f14` in the two `sb1.amlet.id_did.json` + `derive_vkey.py`)
  — a separate trust-root step.
- Any list-all-keys-for-a-hub variant or "latest/active key" selection logic — keep the reader a
  single-row `(hub_id, key_id)` lookup symmetric with the `(hub_id, key_id)`-keyed write. A multi-row
  reader is a later slice only if a consumer needs it (YAGNI).
- Schema changes, new columns, structured logs, `/metrics`, or any `go.mod`/`go.sum`/`schema.sql` edit.

## Implementation Notes
- Signature: `func (s *Store) LookupHubKey(ctx context.Context, hubID int64, keyID uint32) (HubKey,
  bool, error)`. Return `(HubKey{}, false, nil)` for an absent `(hub_id, key_id)` — mirror `FollowState`
  /`Coverage`'s "absent row is not an error" convention (keep store's readers total; absent ≠ error).
  Return `(HubKey{}, false, err)` only on a real query fault.
- Query `SELECT pubkey_raw, pubkey_z, revoked_at, resolved_at FROM hub_keys WHERE hub_id = ? AND
  key_id = ? LIMIT 1` (cast `keyID` to `int64` for the parameter, mirroring `RecordHubKey`'s
  `int64(k.KeyID)` bind). `LIMIT 1` is defensive: `hub_keys` has no UNIQUE, but the write path keeps at
  most one row per `(hub_id, key_id)` (UPDATE-then-INSERT), so a match is single by construction.
- Scan the nullable columns through `sql.NullString` (`pubkey_z`) and `sql.NullInt64` (`revoked_at`,
  `resolved_at`); map an invalid `pubkey_z` back to `""` and an invalid time back to the zero
  `time.Time` (`time.Unix(n, 0)` only when `.Valid`) — the exact inverse of `nullStringOrNil`/
  `unixOrNil`, so a `RecordHubKey` → `LookupHubKey` round-trip is lossless for the empty/zero cases.
- Set the returned `HubKey.HubID`/`KeyID` from the in-args (not re-scanned) so the struct is fully
  populated; `pubkey_raw` is BLOB NOT NULL, scan straight into `[]byte`.
- Write a fresh evergreen docstring describing current behavior (the "absent → not an error" contract).
  Do **not** add an import — `context`/`database/sql`/`errors`/`fmt`/`time` are already imported.
- Oracle/conformance gate is correctly **N/A**: plain CRUD, no proof/verify/didweb/merkle/fsck path and
  no `go.mod`/`go.sum`/`schema.sql` change.

## Verification
- `mise run check` is green (build + vet + test) and `gofmt -l .` is empty.
- `go test -run TestLookupHubKey ./internal/store` passes, covering at minimum:
  - **round-trip**: `RecordHubKey` then `LookupHubKey(hubID, 0x40b74463)` returns `found=true` with
    `PubkeyRaw`/`PubkeyZ`/`Revoked`/`ResolvedAt` byte/field-equal to what was written (reuse the
    `TestRecordHubKeyInsert` fixture: 32-byte `pubkey_raw`, set `PubkeyZ`, set `ResolvedAt`).
  - **nullable round-trip**: a key written with empty `PubkeyZ` and zero `Revoked` reads back
    `PubkeyZ == ""` and `Revoked.IsZero() == true` (proves the NULL→zero inverse of `nullStringOrNil`/
    `unixOrNil`).
  - **absent**: `LookupHubKey` for a `(hubID, keyID)` with no row returns `(HubKey{}, false, nil)` — no
    error.
  - **key-id discrimination**: after a rotation (two rows, `0x22b08f3e` and `0x069d0f14`, written via
    distinct `RecordHubKey` calls per `TestRecordHubKeyRotation`), `LookupHubKey` for each key id
    returns that key's own `pubkey_raw`, not the other's (non-vacuous: the two `PubkeyRaw` values
    differ, so assert each lookup returns the matching one).
- `go list -deps ./internal/store | grep '^net/http'` is empty (store stays a leaf; no new import).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).

## Done When
`store.LookupHubKey` is the round-trip-tested, leaf-clean read side of the `hub_keys` cache and all
Verification criteria pass.
