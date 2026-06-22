## 2026-06-22 — OTS store seam — `ots`-table CRUD (`RecordOTS` / `OTSForRoot` / `PendingOTS` / `MarkOTSUpgraded`)

**Done:** Added the typed `ots`-table CRUD seam to `internal/store` on the already-present schema
table: `RecordOTS` (dedupe-insert on `UNIQUE(hub_id, tree_size, root)`), `OTSForRoot` (absent-is-a-miss
read), `PendingOTS` (oldest-first pending-only list), and `MarkOTSUpgraded` (flip pending→confirmed).
No new dependency, no follower wiring, no HTTP/certificate change — `ots_bytes` is an opaque BLOB and
`status` an opaque string, so the store stays a `net/http`-free leaf and the crypto/proof path is
untouched.

**Files changed:**
- `internal/store/ots.go` (new): `OTSRecord` struct mirroring `CheckpointRecord`'s plain-leaf shape;
  two status consts (`OTSStatusPending = "pending"`, `OTSStatusConfirmed = "confirmed"`) as the single
  source of truth for the literals (literal-drift trap); the four CRUD methods. Ports `RecordCheckpoint`'s
  `ON CONFLICT … DO NOTHING` + `RowsAffected` first-sighting dance, `CheckpointAt`'s
  `sql.ErrNoRows → (zero, false, nil)`, `ListViolations`'s `[]T` leaf read, and reuses
  `unixOrNil` / `nullStringOrNil`.
- `internal/store/ots_test.go` (new): round-trip + dedupe + zero-times-NULL + pending-filter/order +
  upgrade + idempotent-remark tests, with mutation-targeted assertions.

**Verification:** `mise run check` → green (build + vet + all 21 packages `ok`; `gofmt -l .` clean
excluding `cauldron/`). Per `next.md` criterion:
- [x] `go test -count=1 ./internal/store` passes (existing suite unaffected).
- [x] `go test -count=1 -run TestOTS ./internal/store` passes (round-trip / dedupe / pending / upgrade).
- [x] `go list -deps ./internal/store | grep '^net/http'` empty (store stays a leaf).
- [x] `git diff --stat HEAD -- internal/store/schema.sql go.mod go.sum` empty (no schema/dep change).
- [x] Mutation-proven non-vacuous (both reverted): `ASC → DESC` in `PendingOTS` FAILS `TestPendingOTS`'s
  oldest-first order check; dropping the `status = ?` WHERE filter FAILS both `TestPendingOTS` (3 rows
  not 2) and `TestMarkOTSUpgraded` (confirmed row not excluded).

**Next:** With the seam landed, the next OTS sub-step is the **daily stamp pass** that writes through
`RecordOTS` for each distinct accepted root without blocking the follower poll (Correctness rule "OTS
never blocks the follower"). After that: the background **upgrade loop** (reads `PendingOTS`, marks via
`MarkOTSUpgraded` once Bitcoin-confirmed) — this is the step that first pulls in the
`nbd-wtf/opentimestamps` dependency and the calendar HTTP. Then the `.ots` HTTP route and **certificate
§5 BITCOIN ANCHOR** (`HasClause5`), both reading `OTSForRoot` for a `(hub, tree_size, root)` and
rendering the honest pending/not-yet-anchored state on a miss.

**Notes:**
- Oracle/conformance gate is correctly N/A for this slice (plain CRUD + opaque BLOB round-trip; no
  signature / RFC-6962 / Merkle / did:web / `fsck`-rebuild path), matching the `store.md` precedent for
  the other store-leaf CRUD slices.
- `Attempts` and `next_retry` are persisted/read but no method increments `Attempts` or sets
  `next_retry` to a back-off yet — those columns belong to the upgrade loop's retry policy (the
  `attempts`/`next_retry` schema columns exist for it). Carried verbatim here; not exercised beyond
  round-trip. Not a debt for this step — the seam stores what callers give it.
- `MarkOTSUpgraded` intentionally ignores `RowsAffected` (idempotent re-mark / absent-row is a no-op,
  not an error), mirroring `SetCoverage`; `TestMarkOTSUpgradedIdempotent` covers both cases.
- During mutation testing my throwaway `sed` edits to `ots.go` were left in place because the bash cwd
  resets between calls and the `git checkout` to revert ran against the wrong (untracked) path. I
  restored the two correct lines (`WHERE status = ? ORDER BY stamped_at ASC, id ASC`) by hand and
  re-verified the file has no leftover `DESC` / `IS NOT NULL` mutation strings and the suite is green.
  The committed file is the correct version.
- The open `host:port` DID / `ForceQuery` / §6-timestamp issues were left untouched per Not-In-Scope.
