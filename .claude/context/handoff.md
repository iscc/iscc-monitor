# Handoff

## 2026-06-20 — `hub_keys` did:web key cache — store-leaf `RecordHubKey` upsert

**Done:** Added the persistence half of the did:web key cache (ADR-0009): a `HubKey` struct and a
`RecordHubKey(ctx, HubKey) error` method on `*Store` that caches a resolved hub key into `hub_keys`,
deduped per `(hub_id, key_id)` and refreshed each poll. It is a guarded `UPDATE … WHERE hub_id=? AND
key_id=?` (mirroring `SetCoverage`) followed by an `INSERT` only when no row matched — no `ON CONFLICT`
(the table has no UNIQUE index). The follower→store wiring is deliberately NOT touched (separate later
slice).

**Files changed:**
- `internal/store/checkpoints.go`: added the `HubKey` struct (near `CoverageInfo`), the
  `RecordHubKey` method (after `Coverage`), and a small `nullStringOrNil` helper next to `unixOrNil`
  (empty string → SQL NULL for the nullable `pubkey_z` TEXT column).
- `internal/store/checkpoints_test.go`: added `TestRecordHubKeyInsert` / `Refresh` / `Rotation` /
  `Nullable` / `ForeignKey` — one per Verification criterion.

**Verification:** `mise run check` → green (`go build` / `go vet` / `go test ./...` all `ok`, 7
packages). Per-criterion:
- [x] `go test -run TestRecordHubKey ./internal/store` → PASS (all 5).
- [x] happy-path insert (`TestRecordHubKeyInsert`): after one `RecordHubKey` for an `UpsertHub`-seeded
  hub, `count(*) WHERE hub_id=? AND key_id=?` == 1; `pubkey_raw`/`pubkey_z`/`resolved_at` read back
  equal to input.
- [x] refresh/dedupe (`TestRecordHubKeyRefresh`): a second call, same `(hub_id, key_id)`, later
  `ResolvedAt` + set `Revoked` → `count` stays 1, `pubkey_raw`/`resolved_at`/`revoked_at` updated in
  place.
- [x] rotation (`TestRecordHubKeyRotation`): a different `key_id` for the same hub → `count(*) WHERE
  hub_id=?` == 2 (both keys cached).
- [x] nullability (`TestRecordHubKeyNullable`): empty `PubkeyZ` + zero `Revoked` → row selectable via
  `WHERE pubkey_z IS NULL AND revoked_at IS NULL` (proves NULL, not `""`/epoch).
- [x] FK guard (`TestRecordHubKeyForeignKey`): a key for an unknown `hub_id` returns a non-nil error
  (independently probed: genuine SQLite `FOREIGN KEY constraint failed (787)` wrapped through the
  insert path — non-vacuous).
- [x] `go list -deps ./internal/store | grep '^github.com/iscc/iscc-monitor'` → only the self line
  (store stays a leaf; no `didweb`/`logclient` import leaked).
- [x] `go list -deps ./internal/store | grep '^net/http$'` → empty.
- [x] `git diff -- internal/store/schema.sql` → empty (no schema/migration change).
- [x] `git diff -- go.mod go.sum` → empty (no new dependency).
- [x] `gofmt -l .` → empty.

**Next:** The follower→store wiring slice — call `RecordHubKey` from `PollHub` after a successful
`ResolveVerifierKey`, mapping its `DIDKey` (`PublicKey`→`PubkeyRaw`, `Multibase`→`PubkeyZ`,
`Revoked`→`Revoked`) plus the derived key_id (the verifier-string middle `+<hex>+` field) and the
injected `observedAt`→`ResolvedAt` into a `HubKey`. That step is where the stale `sb1.amlet.id_did.json`
fixture + `derive_vkey.py` HUBS must finally be refreshed to signer `069d0f14`, because it is the first
to exercise a live resolution into the cache. The heavier merkle-backed equivocation trigger remains the
headline M1 gap (needs `transparency-dev/merkle` + tile fixtures + a conformance/oracle package + CI
`notecheck` — more than one verifiable slice).

**Notes:**
- The reader (a `HubKey` lookup) was intentionally NOT added — only the write is needed now, per
  `next.md`. The follower-wiring step (or whoever first needs to read a cached key for verification)
  should add the read.
- `key_id` is stored via `int64(k.KeyID)` (uint32→int64), matching how `RecordCheckpoint` casts
  `uint64`→`int64` for SQLite's signed INTEGER. The two reference signer keyhashes (`40b74463`,
  `069d0f14`, plus sb1's prior `22b08f3e`) appear in the tests as realistic key_ids — they are not
  load-bearing fixtures, just plausible values, so no oracle/`derive_vkey.py` parity is asserted here
  (this method only persists what it is given).
- Oracle/conformance gate is correctly **N/A** for this slice: it is plain `hub_keys`-column CRUD with
  null handling and FK enforcement — no proof/verify/didweb/merkle/consistency/fsck/notecheck/signature
  path touched; go.mod/go.sum byte-identical.
- Scope: 1 production file (`checkpoints.go`) + its `_test.go`, within the ≤3 budget; nothing from
  `## Not In Scope` (follower wiring, fixture refresh, merkle trigger, schema/migration) was touched.
- The `(hub_id, key_id)` dedupe is a check-then-write under the store's single-writer connection
  (`SetMaxOpenConns(1)`), so the UPDATE-then-INSERT is not subject to a concurrent-writer race in this
  layer.
