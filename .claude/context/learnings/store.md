<!-- area: internal/store (store.go, tiles.go, fetcher.go, iscc_index.go, checkpoints.go) -->
<!-- indexed-as: store.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/store` — SQLite leaf, mirror read-back, projections

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## SQLiteFetcher / mirror read-back (`internal/store/tiles.go` + `fetcher.go`)

- **settled (landed):** `SQLiteFetcher` satisfies `fsck.Fetcher` via a *byte-identical copied*
  `fsckFetcher` interface (pinned `var _ fsckFetcher = SQLiteFetcher{}`), never by importing `fsck`
  (that pulls `net/http`/`otel`/`klog` and breaks leaf purity). `widthForP` is the SINGLE `p→width`
  authority (`fetcher.go`): `p==0 → tiles.TileWidth (256)`, else `int(p)`; the write API speaks `p` and
  **full coords MUST be written as the literal `0`, never `uint8(256)`** (wraps to 0 only by luck) —
  runtime-`width` test sites guard `if width == tiles.TileWidth { p = 0 }`. `is_full=1` only at
  width==256; partials overwrite in place via composite-PK `ON CONFLICT … DO UPDATE`. Both invariants
  are mutation-proven and covered by `TestIngestTilesWidthMapping`/`TestRecordTilePartialOverwrite`.
- **The partial→full fallback wraps `os.ErrNotExist` on BOTH legs, so `errors.Is` survives a double
  miss.** `ReadTile`/`ReadEntryBundle` retry at width 256 only when `p>0 && errors.Is(err,
  os.ErrNotExist)`; if the full leg also misses it returns *that* wrapped `os.ErrNotExist`
  (TestFetcherReadTilePartialNoFallbackNoFull). `readTileAt`/`readEntryBundleAt` are the no-fallback
  inner reads; a `p==0` miss returns the wrapped sentinel directly. Matches tessera's
  `PartialOrFullResource` (which is `internal/` and not importable) exactly.
- **Oracle/conformance gate correctly N/A for this slice** — plain CRUD + BLOB round-trip with
  synthetic in-test bytes; no signature/RFC-6962/Merkle/did:web/`fsck`-rebuild path. The actual
  `fsck.New(...).Check(...)` root-rebuild + inclusion cross-check is the *next* conformance slice
  (needs real tile fixtures) and is the point where this seam first faces the trust-root oracle.

## iscc_index writer/reader (`internal/store/iscc_index.go`)

- **settled:** `RecordProjections` writer + the two readers (`SeqsForISCCID` one-to-many BLOB lookup,
  `RecordAt` single-row `(hub_id, seq)` PK read) are the store half of the M2 projection (ADR-0008),
  store stays a leaf (no `internal/logclient`/`net/http`), schema/go.mod/go.sum byte-unchanged. Writer
  is per-row `ON CONFLICT(seq) DO UPDATE` (second write wins; mutation `DO NOTHING` → idempotency test
  FAILS on stale fields). `iscc_id` is bound as BOTH the BLOB and TEXT column; readers query the BLOB
  so the index drives it (no ISCC-ID codec). `RecordAt` maps `sql.ErrNoRows → (RecordRow{}, false, nil)`
  (absent projection is a plain miss, not an error — the leaf's mirrored BYTES are the source of truth);
  reads `iscc_id_str`/`note_schema` through `sql.NullString` (NULL→""); scans `seq` int64→uint64.
  Hub-scoping is load-bearing for both readers (mutation dropping `hub_id` → scope test FAILS). (Detail
  in git history at-2026-06-21.) One durable trap below.
- **`RecentRecords(ctx, n) ([]RecordRow, error)` is the realm-wide recent reader** (added 2026-06-23 for
  the dashboard "Recently declared" row): newest-first by the global `seq`, `JOIN follow_state f ON
  f.hub_id = i.hub_id WHERE i.seq < f.last_size` so the accepted-tree ceiling is applied per hub and a
  hub with no `follow_state` row (or NULL/zero `last_size`) contributes nothing. It is SCHEMA-AGNOSTIC
  (ADR-0008): NO `note_schema` filter — rows carry the verbatim schema for the caller (the dashboard view
  layer) to interpret which are declarations. Tests: `TestRecentRecords` (realm-wide order + cap + no-
  follow_state exclusion + limit + schema passthrough) / `TestRecentRecordsEmpty`.
- **Two PRE-EXISTING facts confirmed while wiring `RecentRecords` (NOT introduced by it, NOT fixed here):**
  (a) The `iscc_index.go` package-docstring's "RecordProjections has no production caller yet" is STALE —
  `follower/ingest.go:118` calls it on every verified poll, so the index IS populated in production. (b)
  `seq` is a SINGLE global `INTEGER PRIMARY KEY`, but ingest writes the per-hub ABSOLUTE leaf index
  (`bundleIndex*TileWidth + offset`) — so two hubs whose logs share a leaf index (e.g. both seq 0) COLLIDE
  on the PK and the later write clobbers the earlier (`ON CONFLICT(seq) DO UPDATE` overwrites `hub_id`).
  For a multi-hub realm this means `iscc_index` cannot faithfully hold both hubs' low leaves; it likely
  should be a composite PK `(hub_id, seq)`. `RecentRecords` ordering-by-`seq` is the only monotonic signal
  available (there is no observed-at/declared-at column to order by) and is consistent with `ListRecords`'
  newest-first convention — but the cross-hub recency is only as honest as the clobber allows. Filed as a
  finding for the loop (see issues.md), out of scope for the out-of-loop UI tweak that surfaced it.
- **The stored `note.$schema` is the VERBATIM wire value — a full URI, NOT a short name.** Production
  records carry `http://purl.org/iscc/schema/iscc-note-0.8.0.json` (declaration) /
  `…iscc-note-delete-0.8.0.json` (deletion); the golden `projection_test.go` and `follower/fsck_test.go`
  pin exactly these, and every realistic store fixture uses the `…-0.8.0.json` form. Any consumer that
  matches `note_schema` against a literal (e.g. a kind-label map) MUST use the full URI, and its test
  MUST seed the URI, not a glossary short form — CLAUDE.md's `iscc-note-0.8.0` is prose shorthand, never
  the wire value (the open `proofserve` kind-label issue is exactly this trap shipped).
- **`note_timestamp` is the one RFC-3339-TEXT exception to the unix-seconds time convention** (verbatim
  optional `note.timestamp`, both note types — `schema.py:364/443` `Timestamp | None`). Written via
  `nullStringOrNil` (absent "" → SQL NULL, a true NULL distinct from a present empty string), read back
  "" through `sql.NullString` in `RecordAt`/`ListRecords`; never parsed (ADR-0008). It is the §6 `· at`
  store prerequisite — the certificate render is still the open follow-up; mutation-proven non-vacuous on
  both the present and the NULL→"" leaf (drop it from `DO UPDATE SET` → `…Idempotent` FAILS).
- **No on-disk migration story exists for ANY added column — codebase-wide, by design (not a defect of
  this slice).** `Open` applies `schema.sql` as one `db.Exec` of 9 `CREATE TABLE IF NOT EXISTS` (zero
  `ALTER TABLE`, no `PRAGMA user_version`, no migration framework), so a column added to an EXISTING
  table is a silent no-op on a pre-existing DB — an upgraded node would then hit `no such column: <col>`
  on the new INSERT/SELECT. This is intentional (`next.md` Not-In-Scope: dev DBs are ephemeral; every
  prior column landed this way). Adding an `ALTER`/migration is a deliberate, design-reviewed step — do
  NOT introduce the project's first migration mechanism as a side effect of a field slice. Filed `normal`.

## SQLite store (`internal/store`)

- **`modernc.org/sqlite` pin is `v1.46.1` (last version requiring only `go 1.24.0`).** `v1.46.2`+ bump
  the `go.mod` directive to `≥1.25.0` and would fail the gate on the pinned 1.24 toolchain. After
  `go get`, the directive stayed `go 1.24.0` with no `toolchain` line; `go mod verify` + `go mod tidy`
  are both clean (reviewer reconfirmed — tidy produces zero diff). The reset-the-directive route is NOT
  viable for v1.52.0 (genuinely won't build on 1.24); the pin is the correct fix.
- **`PRAGMA foreign_keys=ON` is per-connection — `SetMaxOpenConns(1)` makes it stick.** Reviewer
  independently confirmed FK enforcement is live (orphan `hub_keys` insert rejected with SQLITE error
  787) and `journal_mode=wal`, `busy_timeout=5000`, `synchronous=1(NORMAL)`, `MaxOpenConnections=1` are
  all applied in order on `Open`. When the read-pool split lands at serving, FKs + WAL pragmas must be
  re-asserted on the read connections too (per-connection state does not carry across a larger pool).
- **Schema columns match the plan's "SQLite schema" block verbatim** (all 9 core tables, `iscc_id`
  non-unique indexed for the one-to-many `iscc_id→seq`, UNIQUE on checkpoints/ots `(hub_id,tree_size,
  root)`, PKs on tiles/entry_bundles). No `network` column (ADR-0007, grep confirms comment-only). No
  `cosigs` table (M7-deferred). The `_ = db.Close()` on `Open`'s error paths is the correct idiom
  (preserve the original error; don't mask it with the close error).
- **Typed CRUD seam for the follower is `UpsertHub`/`RecordCheckpoint`/`FollowState`/`AdvanceFollowState`
  (`checkpoints.go`), and `store` stays a leaf** — `go list -deps ./internal/store` shows zero internal
  iscc-monitor deps; the `Status` string is carried on `CheckpointRecord` (the `checkpoints` table has
  **no** status column, so it is intentionally not persisted) rather than importing `logclient`. Keep
  it that way so net/http never enters the store closure.
- **`RecordCheckpoint` dedupe relies on `ON CONFLICT(...) DO NOTHING` + `RowsAffected()`:** real insert
  → `n>0` → `LastInsertId`/`inserted=true`; conflict → `n==0` → SELECT the id back/`inserted=false`,
  nil err. The modernc driver returns `RowsAffected==0` on `DO NOTHING`, so this branch is load-bearing
  and is the correct way to tell "first sighting" from "re-observed" without an error.
- **`AdvanceFollowState` upsert omits `frozen` from the `DO UPDATE SET`** (`ON CONFLICT(hub_id) DO
  UPDATE SET last_size=excluded.last_size`), so a hub frozen via the freeze path stays frozen across an
  advance (ADR-0006, no auto-unfreeze) — proven by raw `UPDATE frozen=1` then asserting `frozen` still 1
  after advance. Nothing in this layer ever sets or clears `frozen`; only the (future) freeze path sets it.
- **Zero `time.Time` → NULL convention (`unixOrNil`):** a zero `ObservedAt` writes SQL NULL, not `0`, so
  "never observed" stays distinct from the unix epoch. `FollowState` reads `last_size`/`last_error`
  through `sql.NullInt64`/`sql.NullString` so a partial/absent row degrades to the zero value, never an
  error — an unknown `hubID` returns `FollowState{}` + nil err by design (follower treats it as "never polled").
- **Freeze path seam is `RecordViolation` (plain INSERT) + `Freeze` (upsert `frozen=1`).** Verified the
  two upserts compose: `Freeze` writes `INSERT … (hub_id, frozen) VALUES (?,1) ON CONFLICT(hub_id) DO
  UPDATE SET frozen=1`, `AdvanceFollowState` omits `frozen` from its `DO UPDATE`, so advance-after-freeze
  keeps `frozen=1` AND moves `last_size` (no auto-unfreeze, ADR-0006) — `TestFreezeNoAutoUnfreeze` is
  non-vacuous (asserts both flags). `RecordViolation` is `LastInsertId`-only (no `RowsAffected` dance —
  no `ON CONFLICT`), so re-detection yields distinct ids/rows (re-detection is itself evidence). The
  `Kind` string rides on the `Violation` struct exactly as `Status` rides on `CheckpointRecord`, keeping
  store import-free of `logclient`.
- **`ListViolations(ctx, hubID)` is the read side of the freeze evidence — a leaf read scoped to one
  hub, newest-first (`ORDER BY detected_at DESC, id DESC`), reading back only `hub_id, kind,
  detected_at` (raw_a/raw_b/proof_json deliberately left zero — they belong with the future
  proof-bundle surface).** `detected_at` reads through `sql.NullInt64` (the `unixOrNil` inverse) → zero
  `time.Time` on NULL, mirroring the other NULL-time reads; a hub with none returns an empty slice +
  nil err. SQLite quirk to know: a NULL `detected_at` sorts LAST under `DESC` (so a time-unknown
  violation lands at the bottom of the newest-first list) — acceptable for the Exhibit. Reviewer
  mutation-proved non-vacuous (reverted): `DESC → ASC` flips `TestListViolations`'s newest-first order.
- **The `net`/`net/netip`/`net/url` in `go list -deps ./internal/store` are from `modernc.org/sqlite`,
  NOT iscc-monitor code.** The load-bearing invariant is "no `net/http` in the store closure" — verify
  with `go list -deps ./internal/store | grep '^net/http'` (empty) and that the package's own `.Imports`
  are exactly `context database/sql embed errors fmt time` + the sqlite driver. Do not flag the bare
  `net` lines as a leak.
- **settled (landed): `RecordHubKey` is the did:web key cache write** — guarded `UPDATE … WHERE
  hub_id=? AND key_id=?` then `INSERT` on zero `RowsAffected` (the `SetCoverage` idiom; `hub_keys` has
  no UNIQUE). UPDATE rewrites *all* mutable columns so a re-resolve tracks the DID doc (clears
  `pubkey_z`→NULL when the multibase drops, not append-only); `nullStringOrNil`+`unixOrNil` keep "no
  multibase"/"not revoked" distinct from `""`/epoch; FK enforced (787 on orphan); `key_id uint32→int64`.
  Leaf-pure, oracle N/A. (Detail in git history.)
- **Coverage set-once is a guarded `UPDATE … WHERE monitored_since_size IS NULL` keyed on the SIZE
  column being NULL — and that guard is correct even for a size-0 start.** The first `SetCoverage`
  writes `int64(size)` (so the column is NOT NULL afterward, even when size==0), making every re-call a
  silent no-op regardless of size; the start is immutable at size 0 too (reviewer added a throwaway
  `TestCoverageZeroSizeStillSet` — PASS — then removed it; the committed suite does not cover the
  size-0 edge but the live `PollHub` only ever records `info.TreeSize` from a verified checkpoint).
  `SetCoverage` intentionally ignores `RowsAffected` (zero-rows-after-set is the correct non-error
  case), mirroring how `next.md` scoped it. `Coverage` reads both columns through `sql.NullInt64` and a
  zero `monitored_since_time` (NULL via `unixOrNil`) degrades to a zero `time.Time` with `Set` still
  true — so "coverage started, time unknown" is representable but unreachable from `PollHub` (which
  always injects a real `observedAt`). Wiring lives ONLY on the verified, non-violation `PollHub` path
  (between `RecordCheckpoint` and `AdvanceFollowState`), kept out of `freeze`, so a contradictory
  observation never starts coverage (ADR-0001) — the fork/unverified tests assert `cov.Set == false`.
- **settled (landed): `AdvanceAccepted(ctx, CheckpointRecord)` is the repo's first `*sql.Tx`** — it
  collapses the verified-advance triad into one transaction whose three `tx.ExecContext` statements are
  byte-for-byte the `RecordCheckpoint` insert (`DO NOTHING`), the `SetCoverage` guarded set-once UPDATE
  (`… IS NULL`), and the `AdvanceFollowState` upsert (`frozen` omitted, no auto-unfreeze). The
  `defer func(){ _ = tx.Rollback() }()` is the standard pattern (post-`Commit` rollback → benign
  `sql.ErrTxDone`; real commit error returned `%w`-wrapped), NOT a swallowed-error dodge. The three
  original methods stay public (still used as seed helpers + the freeze-path `RecordCheckpoint`).
  Mutation-proven non-vacuous; store stays a leaf; oracle N/A. Detail in git history.
- **settled (landed): OTS-table CRUD seam (`ots.go`) is a leaf, idiom-identical to the checkpoint
  family.** `RecordOTS` ports `RecordCheckpoint`'s `ON CONFLICT(hub_id,tree_size,root) DO NOTHING` +
  `RowsAffected` dance; `OTSForRoot` ports the `sql.ErrNoRows → (zero,false,nil)` absent-is-not-error
  miss (un-anchored root → honest pending, never 5xx); `PendingOTS(now)` is a status-scoped leaf read
  `WHERE status=? AND (next_retry IS NULL OR next_retry<=?) ORDER BY stamped_at ASC, id ASC` (the
  `next_retry` back-off leg binds `now.Unix()` directly, not `unixOrNil`); `MarkOTSUpgraded` /
  `MarkOTSAttempted` / `MarkOTSStamped` are plain UPDATEs that ignore `RowsAffected` (idempotent/absent
  re-mark is a no-op) and leave `status` untouched except the explicit upgrade. `OTSStatusPending`/
  `OTSStatusConfirmed` consts are the single source for the two literals (literal-drift trap); `Status`/
  `ots_bytes` carried opaque so store stays leaf-pure (no anchoring/OTS import). All mutation-proven;
  oracle N/A (opaque-BLOB round-trip, no merkle/proof path). Detail in git history at-2026-06-21.
- **settled (landed): the realm-index `Anchor` projection is a read-only column on `ListHubs`.** A
  correlated subselect `(SELECT o.status FROM ots o WHERE o.hub_id=h.hub_id ORDER BY o.stamped_at DESC,
  o.id DESC LIMIT 1)` read through `sql.NullString` (NULL→"") gives the hub's latest-stamped-root OTS
  status. It is per-HUB (newest by `stamped_at`), DECOUPLED from `f.last_size`, so it is NOT a
  per-checkpoint attestation — see `dashboard.md` for the honesty rationale and the open design `normal`.
  Store stays a leaf (no new import; `go list -deps … | grep '^net/http$'` empty); schema/go.mod/go.sum
  byte-unchanged; mutation-proven (subselect→`''` FAILS `TestListHubsAnchorStatus`).
