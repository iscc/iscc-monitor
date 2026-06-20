# Next Work Package

## Step: `hub_keys` did:web key cache — store-leaf `RecordHubKey` upsert

## Goal
Add the persistence half of the did:web key cache (ADR-0009): a single typed store-leaf
method that records a resolved hub key into the `hub_keys` table, deduped per
`(hub_id, key_id)` and refreshed each poll. This is the prerequisite for the follower to
cache resolved keys and the first concrete consumer of the empty `hub_keys` table.

## Scope
- **Create**: (none — extend an existing file)
- **Modify**:
  - `internal/store/checkpoints.go` — add a `HubKey` struct + `RecordHubKey(ctx, HubKey) error`
    (one new typed CRUD method, mirroring `RecordCheckpoint`/`RecordViolation`/`SetCoverage`).
  - `internal/store/checkpoints_test.go` — add focused `TestRecordHubKey*` tests (test file,
    does not count against the ≤3 non-test/doc budget).
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/schema.sql` lines 30-40 — the `hub_keys` columns
    (`hub_id, key_id, pubkey_raw, pubkey_z, revoked_at, resolved_at`; no PK/UNIQUE declared).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `RecordCheckpoint` (lines 95-123,
    the `RowsAffected`/`LastInsertId` dedupe idiom), `RecordViolation` (line 211), `SetCoverage`
    (line 251, the guarded-`UPDATE` set-once idiom), and `unixOrNil` (line 296, zero-time→NULL).
  - `/workspace/iscc-monitor/internal/didweb/resolve.go` lines 54-60 — the `DIDKey` fields
    (`Multibase`, `PublicKey`, `Revoked`) the follower will later map into `HubKey`.
  - `/workspace/iscc-monitor/internal/didweb/vkey.go` lines 75-95 — how `key_id` is the
    BE-uint32 `keyID(name, pub)` (private) and how the `<name>+<keyid:08x>+<base64>` verifier
    string encodes it; documents what the follower will supply as `HubKey.KeyID` later.

## Not In Scope
- **Follower wiring.** Do NOT call `RecordHubKey` from `PollHub`/`follower.go` this step. The
  follower→store wiring (mapping `ResolveVerifierKey`'s `DIDKey` + key_id into a `HubKey`) is a
  separate later slice; touching `follower.go` here would blow the seam open prematurely.
- **Refreshing the stale `sb1.amlet.id_did.json` fixture / `derive_vkey.py` HUBS** to signer
  `069d0f14`. That fixture drift is real but belongs with the follower-wiring step that actually
  exercises a live resolution into the cache; it is not needed to test a store insert.
- **The merkle-backed equivocation trigger.** Still the headline M1 gap, but it needs a new
  `transparency-dev/merkle` dependency + tile fixtures + a conformance/oracle test package +
  CI `notecheck` wiring — far more than 3 files and not one verifiable step. Deferred.
- **Schema changes / migrations.** Do not add a PK or UNIQUE index to `hub_keys`; dedupe in the
  method via check-then-insert/update so `schema.sql` stays byte-identical.
- Reading a key back out (a `HubKey` lookup method) — only the write is needed now; add the read
  when a consumer needs it.

## Implementation Notes
- Add a `HubKey` struct in `checkpoints.go` near the other record types:
  `HubKey{ HubID int64; KeyID uint32; PubkeyRaw []byte; PubkeyZ string; Revoked time.Time;
  ResolvedAt time.Time }`. Mirror the existing convention where strings/IDs ride on the struct so
  **store stays a leaf** — `go list -deps ./internal/store` must keep zero internal iscc-monitor
  deps (do NOT import `didweb`/`logclient`; the follower maps `DIDKey`→`HubKey` at the call site
  later).
- `key_id` is the signed-note BE-uint32 keyhash. Store it in the table's `INTEGER` via
  `int64(k.KeyID)` (consistent with how `RecordCheckpoint` casts `uint64`→`int64`). Document in
  the method docstring that the follower derives it from `ResolveVerifierKey`'s verifier string
  middle field (`+<hex>+`) or `keyID(name, pub)` — but this method just persists what it is given.
- **Dedupe per `(hub_id, key_id)`** (a hub's key is identified by its keyhash; the same key
  re-resolved each poll must refresh, not accumulate). The table has no UNIQUE index, so use the
  `SetCoverage`-style guarded pattern, NOT `ON CONFLICT`: run an `UPDATE hub_keys SET
  pubkey_raw=?, pubkey_z=?, revoked_at=?, resolved_at=? WHERE hub_id=? AND key_id=?`, check
  `RowsAffected()`; if zero rows matched, `INSERT` the full row. This refreshes `resolved_at`
  (and `revoked_at` if the DID doc now revokes the key) on every poll while keeping exactly one
  row per `(hub_id, key_id)`. A *different* key_id for the same hub (rotation) inserts a second
  row — that is correct (both the old and new keys stay cached; revocation is recorded via
  `revoked_at`, not by deleting the old row).
- `pubkey_z` (the `z6Mk…` multibase) is a nullable `TEXT` — an empty `PubkeyZ` should write SQL
  NULL, not `""`. Use a `sql.NullString` (Valid only when non-empty) or a tiny local helper
  mirroring `unixOrNil`, so "no multibase" stays distinct from empty.
- `revoked_at` and `resolved_at` are nullable `INTEGER` unix-seconds — use the existing
  `unixOrNil(t)` so a zero `time.Time` writes NULL (an un-revoked key has a zero `Revoked` →
  NULL `revoked_at`, distinct from epoch; an absent `ResolvedAt` likewise). This matches the
  zero-time→NULL convention already proven in `RecordCheckpoint`/`SetCoverage`.
- **Correctness rule (learnings — "did:web is the only key source", ADR-0009):** these rows are a
  *cache* of the DID document, never an independent key source. The method must overwrite on
  re-resolve (the DID doc is the source of truth) — so the `UPDATE`-then-`INSERT` refresh, NOT an
  insert-once. Do not add any logic that would let a cached row outvote a fresh resolution.
- Foreign key: `hub_id REFERENCES hubs(hub_id)` is enforced (`PRAGMA foreign_keys=ON` +
  `SetMaxOpenConns(1)` — see learnings), so a test inserting a `HubKey` for an unknown `hub_id`
  must get a FK error (SQLite 787). Seed a real hub via `UpsertHub` first in the happy-path tests.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass,
  `gofmt -l .` empty).
- `go test -run TestRecordHubKey ./internal/store` passes, covering at minimum:
  - happy-path insert: after `RecordHubKey` for a hub seeded via `UpsertHub`, a raw
    `SELECT count(*) FROM hub_keys WHERE hub_id=? AND key_id=?` returns exactly 1, and the
    `pubkey_raw`/`pubkey_z`/`resolved_at` columns read back equal the input.
  - refresh/dedupe: a second `RecordHubKey` with the **same** `(hub_id, key_id)` but a later
    `ResolvedAt` (and/or a set `Revoked`) leaves `count(*) == 1` and updates `resolved_at`
    (and `revoked_at`) in place — no duplicate row.
  - rotation: a `RecordHubKey` with a **different** `key_id` for the same hub yields
    `count(*) == 2` for that hub (both keys cached).
  - nullability: a `HubKey` with empty `PubkeyZ` and zero `Revoked` writes `pubkey_z IS NULL`
    and `revoked_at IS NULL` (assert via raw `SELECT … WHERE pubkey_z IS NULL`).
  - FK guard: `RecordHubKey` for a `hub_id` with no `hubs` row returns a non-nil error.
- `go list -deps ./internal/store | grep -E '^github.com/iscc/iscc-monitor'` is empty
  (store stays a leaf — no `didweb`/`logclient` import leaked in).
- `git diff -- internal/store/schema.sql` is empty (no schema/migration change).

## Done When
`RecordHubKey` upserts exactly one row per `(hub_id, key_id)` into `hub_keys` with correct null
handling and FK enforcement, all Verification checks pass, and the store still has zero internal
iscc-monitor dependencies.
