<!-- area: internal/store (store.go, tiles.go, fetcher.go, iscc_index.go, checkpoints.go) -->
<!-- indexed-as: store.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/store` — SQLite leaf, mirror read-back, projections

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## SQLiteFetcher / mirror read-back (`internal/store/tiles.go` + `fetcher.go`)

- **settled (landed, detail in git history at-2026-06-21):** `SQLiteFetcher` satisfies `fsck.Fetcher` via a
  *byte-identical copied* `fsckFetcher` interface (pinned `var _`), never importing `fsck` (pulls
  `net/http`/`otel`/`klog` → breaks leaf purity). `widthForP` is the SINGLE `p→width` authority: `p==0 →
  tiles.TileWidth (256)`, else `int(p)`; full coords MUST be written as the literal `0`, never `uint8(256)`
  (runtime sites guard `if width == tiles.TileWidth { p = 0 }`); `is_full=1` only at width 256, partials
  overwrite via composite-PK `ON CONFLICT … DO UPDATE`. The partial→full fallback wraps `os.ErrNotExist` on
  BOTH legs so `errors.Is` survives a double miss (retry at 256 only when `p>0 && errors.Is(…ErrNotExist)`).
  All mutation-proven (`TestIngestTilesWidthMapping`/`TestRecordTilePartialOverwrite`/`TestFetcherReadTilePartialNoFallbackNoFull`).
- **Oracle/conformance gate correctly N/A for this slice** — plain CRUD + BLOB round-trip with
  synthetic in-test bytes; no signature/RFC-6962/Merkle/did:web/`fsck`-rebuild path. The actual
  `fsck.New(...).Check(...)` root-rebuild + inclusion cross-check is the *next* conformance slice
  (needs real tile fixtures) and is the point where this seam first faces the trust-root oracle.

## iscc_index writer/reader (`internal/store/iscc_index.go`)

- **settled:** `RecordProjections` writer + the readers (`SeqsForISCCID` one-to-many BLOB lookup,
  `RecordAt`/`ListRecords`/`RecentRecords`) are the store half of the M2 projection (ADR-0008), store
  stays a leaf (no `internal/logclient`/`net/http`), schema/go.mod/go.sum byte-unchanged. Writer is
  per-row `ON CONFLICT(hub_id, seq) DO UPDATE` (composite PK, second write wins; mutation `DO NOTHING`
  → idempotency test FAILS on stale fields). `iscc_id` is bound as BOTH the BLOB and TEXT column;
  readers query the BLOB so the index drives it (no ISCC-ID codec). `RecordAt` maps `sql.ErrNoRows →
  (RecordRow{}, false, nil)` (absent projection is a plain miss, not an error — the leaf's mirrored
  BYTES are the source of truth); reads `iscc_id_str`/`note_schema` through `sql.NullString` (NULL→"");
  scans `seq` int64→uint64. Hub-scoping is load-bearing for every reader (mutation dropping `hub_id` →
  scope test FAILS). (Detail in git history at-2026-06-21.) One durable trap below.
- **`RecentRecords(ctx, n) ([]RecordRow, error)` is the realm-wide recent reader** (added 2026-06-23 for
  the dashboard "Recently declared" row): newest-first by the global `seq`, `JOIN follow_state f ON
  f.hub_id = i.hub_id WHERE i.seq < f.last_size` so the accepted-tree ceiling is applied per hub and a
  hub with no `follow_state` row (or NULL/zero `last_size`) contributes nothing. It is SCHEMA-AGNOSTIC
  (ADR-0008): NO `note_schema` filter — rows carry the verbatim schema for the caller (the dashboard view
  layer) to interpret which are declarations. Tests: `TestRecentRecords` (realm-wide order + cap + no-
  follow_state exclusion + limit + schema passthrough) / `TestRecentRecordsEmpty`.
- **settled (landed, migration 0): `iscc_index` is keyed on the composite `(hub_id, seq)` PK.** `seq` is
  each hub's per-hub ABSOLUTE leaf index (`bundleIndex*TileWidth + offset`, restarts at 0 per hub), so the
  composite PK lets two hubs both index low leaves (seq 0, 1, …) without colliding — the original single
  global `seq PRIMARY KEY` clobbered the earlier hub's row on `ON CONFLICT(seq)`. Both the `schema.sql`
  fresh-DB shape and migration 0's rebuilt table carry `PRIMARY KEY (hub_id, seq)`; the two DDLs MUST stay
  identical (if either is edited, edit both). `RecentRecords` orders by the global `seq` (the only
  monotonic signal — no observed-at column), now honest across hubs since the clobber is gone.
  Mutation-proven (revert to single-PK → `TestRecordProjectionsMultiHubSeqZero` FAILS on the `ON CONFLICT`
  mismatch). Also confirmed (NOT a defect): the `iscc_index.go` package-docstring's "no production caller
  yet" is STALE — `follower/ingest.go` calls `RecordProjections` on every verified poll.
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
- **Migration runner lives in `Open` after the `schemaSQL` exec: a `PRAGMA user_version`-gated,
  append-only `migrations []func(*sql.Tx) error`.** `len(migrations)` IS the code's current schema
  version; index `i` lifts the version `i→i+1` inside its OWN `*sql.Tx` (the `AdvanceAccepted` `defer
  Rollback` idiom; the bump is `fmt.Sprintf("PRAGMA user_version = %d", v+1)` because SQLite won't BIND
  that pragma — in-code `int`, no injection). Fail-closed + idempotent; `applyMigrations` takes the slice
  explicitly so it's unit-testable with a synthetic list. **Append migrations, never edit/reorder a
  released entry** (`migrateISCCIndexCompositePK` is index 0; a fresh DB skips it — `schemaSQL` already
  builds the composite shape and Open jumps straight to `len(migrations)`; only a pre-existing single-PK
  DB runs it). A migration body does NOT bump `user_version` — `applyMigration` does that on the same tx.
- **settled (landed): the runner now bounds the read-back `user_version` BEFORE the loop** (`if version <
  0 || version > len(migs)` → wrapped `errUnsupportedSchemaVersion`), so a DB from a NEWER binary
  (version > len) is rejected fail-closed instead of opening silently, and a corrupt NEGATIVE version
  returns an error instead of panicking on `migs[-1]`. Match the reject with `errors.Is(err,
  errUnsupportedSchemaVersion)`, not the message. Mutation-proven (remove the guard →
  `TestMigrationOutOfRangeVersion` FAILS: future returns nil, negative panics). This is the durable
  fail-closed posture every future migration index inherits — the guard is the reason a non-empty
  `migrations` slice is safe.

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
- **settled (landed, detail in git history): the freeze + evidence seams are leaf, struct-carried
  (`Kind`/`Status` ride structs, not columns, so store stays `logclient`-free).** `RecordViolation`
  (plain INSERT, no `ON CONFLICT` → distinct re-detection rows = evidence) + `Freeze` (upsert
  `frozen=1`); `AdvanceFollowState`/`AdvanceAccepted` omit `frozen` from their `DO UPDATE` so
  advance-after-freeze keeps `frozen=1` while moving `last_size` (no auto-unfreeze, ADR-0006).
  `ListViolations` reads the evidence newest-first (`detected_at DESC, id DESC`; NULL `detected_at`
  sorts LAST); the dossier Exhibit reads each raw checkpoint's size via `logclient.CheckpointSizeFromRaw`.
  All mutation-proven, oracle N/A.
- **`ListCheckpoints(ctx, hubID, n) ([]CheckpointSummary, error)` is the §5-observation-log read** (added
  2026-06-23, advance `96e9600`): a leaf read of `tree_size, observed_at FROM checkpoints WHERE hub_id=?
  ORDER BY observed_at DESC, id DESC LIMIT ?`, mirroring `ListViolations`'s shape. `observed_at` reads
  through `sql.NullInt64` (the `unixOrNil` inverse) → zero `time.Time` on NULL; absent hub → empty slice +
  nil err. Deliberately `observed_at DESC` (chronological log), DISTINCT from `ListHubs`'s §3 `tree_size
  DESC` subselect — note a frozen hub can have a same-size or smaller contradictory checkpoint at a LATER
  `observed_at`, so the size is NOT monotonic across this ordering (the view layer must not assume it — see
  `learnings/dossier.md`). Mutation-proven (`DESC → ASC` flips `TestListCheckpoints` + the dossier order
  assertion). Store stays a leaf (stdlib-only imports; no `net/http`).
- **The `net`/`net/netip`/`net/url` in `go list -deps ./internal/store` are from `modernc.org/sqlite`,
  NOT iscc-monitor code.** The load-bearing invariant is "no `net/http` in the store closure" — verify
  with `go list -deps ./internal/store | grep '^net/http'` (empty) and that the package's own `.Imports`
  are exactly `context database/sql embed errors fmt time` + the sqlite driver. Do not flag the bare
  `net` lines as a leak.
- **settled (landed, detail in git history): `RecordHubKey` is the did:web key cache write** — guarded
  `UPDATE` then `INSERT` on zero `RowsAffected` (the `SetCoverage` idiom; `hub_keys` has no UNIQUE);
  UPDATE rewrites ALL mutable columns so a re-resolve tracks the DID doc (clears `pubkey_z`→NULL when
  multibase drops, not append-only). `nullStringOrNil`+`unixOrNil` keep absent distinct from `""`/epoch;
  FK enforced (787). Leaf-pure, oracle N/A.
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
- **settled (landed, detail in git history): the verified-advance + OTS CRUD seams are leaf,
  struct-carried, mutation-proven, oracle N/A.** `AdvanceAccepted(ctx, CheckpointRecord)` is the repo's
  first `*sql.Tx`, collapsing the verified-advance triad (`RecordCheckpoint` `DO NOTHING`, `SetCoverage`
  set-once, `AdvanceFollowState` `frozen`-omitting upsert) in one tx (`defer Rollback` after `Commit` →
  benign `sql.ErrTxDone`, NOT a dodge). The OTS seam (`ots.go`) is idiom-identical: `RecordOTS` (`DO
  NOTHING` + `RowsAffected`), `OTSForRoot` (`ErrNoRows → (zero,false,nil)`, never 5xx), `PendingOTS`
  (`next_retry` filter + `stamped_at ASC`), `MarkOTS*` no-op-on-absent. `OTSStatus{Pending,Confirmed}`
  consts are the single literal source; `Status`/`ots_bytes` carried opaque so store stays leaf-pure.
- **settled (landed): the realm-index `Anchor` projection is a read-only column on `ListHubs`.** A
  correlated subselect `(SELECT o.status FROM ots o WHERE o.hub_id=h.hub_id ORDER BY o.stamped_at DESC,
  o.id DESC LIMIT 1)` read through `sql.NullString` (NULL→"") gives the hub's latest-stamped-root OTS
  status. It is per-HUB (newest by `stamped_at`), DECOUPLED from `f.last_size`, so it is NOT a
  per-checkpoint attestation — see `dashboard.md` for the honesty rationale and the open design `normal`.
  Store stays a leaf (no new import; `go list -deps … | grep '^net/http$'` empty); schema/go.mod/go.sum
  byte-unchanged; mutation-proven (subselect→`''` FAILS `TestListHubsAnchorStatus`).
- **TRAP (advance `820a831`): the §3 observed-time subselect now ties to the accepted size
  (`AND c.tree_size = f.last_size ORDER BY c.id DESC LIMIT 1`) — `id DESC` is BACKWARDS for a same-size
  FORK.** It closes the equivocation/higher-size decouple but a fork records a contradictory row at
  `tree_size == last_size` (later `id`), so `id DESC` still picks the rejected row. The accepted row at a
  size is the EARLIEST (`store.CheckpointAt` uses `ORDER BY rowid`; two rows at `last_size` ⇒ a fork, since
  same-root re-observation deduped by `ON CONFLICT(hub_id,tree_size,root)`); the fix is `id ASC`. See
  `dossier.md` + the open `normal`. [Codex P2, reviewer-reproduced.]
