# Next Work Package

## Step: Typed store persistence helpers for the checkpoint verdict (`hubs` / `checkpoints` / `follow_state`)

## Goal
Give `internal/store` the typed insert/query methods the follower needs to persist one observed
checkpoint verdict: register a hub, record an observed checkpoint, and read/advance the per-hub
follow cursor. This is the persistence half the follower will drive next — landed now, with its own
driving tests, so the follower step that follows is mostly wiring rather than wiring + schema
discovery.

## Goal-fit (state → target gap)
`state.md` records the schema + `Open`/`Close` exist but "no typed insert/query methods yet (they
land with their real caller, the follower)" and "nothing yet reads or writes store rows." The
handoff's `**Next:**` bundles store CRUD + the follower loop + the `hub_keys` did:web cache + the sb1
fixture refresh into a single item — that is far more than 3 files and several distinct verifiable
behaviors. This step takes the store-CRUD slice only: the smallest coherent unit that compiles, runs,
and is mechanically testable against a temp SQLite DB on observable row state. The follower loop, the
`hub_keys` cache, and the sb1 fixture refresh are explicitly deferred to the next step.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — typed methods on `*Store`:
    - `UpsertHub(ctx, domain, origin, baseURL string) (hubID int64, err error)` — insert-or-get the
      `hubs` row for a hub, returning its `hub_id`. Idempotent on re-register (same domain → same id).
    - `RecordCheckpoint(ctx, c CheckpointRecord) (id int64, inserted bool, err error)` — insert one
      observed checkpoint into `checkpoints`; dedupe on the `UNIQUE(hub_id, tree_size, root)` key
      (a re-observed `(size, root)` returns the existing id with `inserted=false`, never an error).
    - `FollowState(ctx, hubID int64) (FollowState, error)` — read the per-hub cursor + freeze flag;
      a hub with no row yet returns the zero `FollowState{}` (last_size 0, frozen false), not an error.
    - `AdvanceFollowState(ctx, hubID int64, lastSize uint64) error` — upsert `follow_state`, setting
      `last_size` to the newly-accepted size. Must NOT clear `frozen` (no auto-unfreeze, ADR-0006).
    - Plain store-owned structs `CheckpointRecord{HubID int64; Status string; TreeSize uint64;
      Root []byte; Raw []byte; ObservedAt time.Time}` and `FollowState{LastSize uint64; Frozen bool;
      LastError string}`. Do NOT import `internal/logclient` for these.
  - `/workspace/iscc-monitor/internal/store/checkpoints_test.go` — table/scenario tests driving the
    new methods against a `t.TempDir()` DB, asserting on observable rows only (test file, not counted).
- **Modify**: (none required) — the methods attach to the existing `*Store` in the new file. Touch
  `/workspace/iscc-monitor/internal/store/sqlite.go` only if a tiny shared helper is genuinely needed;
  prefer not to.
- **Reference**:
  - `/workspace/iscc-monitor/internal/store/schema.sql` — exact columns/types for `hubs`,
    `checkpoints`, `follow_state` (timestamps INTEGER unix-seconds; booleans 0/1; `root`/`raw` BLOB;
    `checkpoints` UNIQUE is `(hub_id, tree_size, root)`; `follow_state` PK is `hub_id`).
  - `/workspace/iscc-monitor/internal/store/sqlite.go` — the `*Store` type, the `_ = db.Close()`
    error-preserving idiom, and the file-docstring / time-convention style to match.
  - `/workspace/iscc-monitor/internal/store/sqlite_test.go` — the established test style
    (`t.TempDir()` path, raw `s.db` reads for assertions, independently-pinned expectations).
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — the `Status.String()` values
    (`"verified"`/`"unverified"`/`"unresolvable"`/`"rotated"`) and `CheckpointInfo` shape the follower
    will later map into a `CheckpointRecord`. Read for the contract only; do not import the package.

## Not In Scope
- The follower poll loop itself (calling `AcceptCheckpoint`, the `err`-before-status contract,
  network polling, backoff) — that is the very next step and consumes these methods.
- The `hub_keys` did:web key cache and the stale sb1 did.json / `derive_vkey.py` `HUBS` fixture
  refresh to `069d0f14` — deferred with the follower step that actually writes `hub_keys`.
- The three-trigger RFC-6962 consistency check, `violations` inserts, and the `frozen=1` write path
  (this step only *reads* `frozen` and refuses to clear it; it never sets it).
- Tiles / entry_bundles / iscc_index / ots CRUD (M2 and OTS milestones).
- A `cmd/` binary entrypoint, config, realm registry, `/metrics`, structured logs.
- Importing `internal/logclient` into `internal/store` (would couple the store to net/http-bearing
  deps and risk a future cycle; pass the status as a plain string instead).
- Any `schema.sql` change (e.g. adding a UNIQUE on `hubs.domain`) — a schema change is a separate,
  reviewable decision.

## Implementation Notes
- **No import of `internal/logclient`.** Keep `store` a leaf depending only on `database/sql` +
  stdlib (plus the already-wired driver). The follower (next step) maps `logclient.Status.String()`
  and `CheckpointInfo` into the plain `CheckpointRecord` / status-string at the call site. This honors
  the learnings' purity discipline and avoids dragging net/http into the store closure.
- **Single-writer discipline already holds** at this layer (`SetMaxOpenConns(1)`), so these methods
  use `db.ExecContext` / `db.QueryRowContext` directly; do not open new connections or pools. The
  goroutine-ownership write wrapper remains the follower's concern.
- **Timestamps:** store `ObservedAt` as `t.Unix()` INTEGER (the schema's convention). Decide how a
  zero `time.Time` is written (NULL or 0) and assert that choice in a test so it is explicit.
- **`UpsertHub` idempotency:** `hubs` has no UNIQUE on `domain`, so implement as "SELECT hub_id WHERE
  domain=?; if none, INSERT and return LastInsertId". One method, no schema change. The follower will
  also want `origin`/`base_url` set on first insert; on a re-register, returning the existing id
  (without rewriting columns) is acceptable for this step.
- **`RecordCheckpoint` dedupe:** rely on the existing `UNIQUE(hub_id, tree_size, root)`. Prefer
  `INSERT ... ON CONFLICT(hub_id, tree_size, root) DO NOTHING` then read back the id, or detect the
  constraint and SELECT the existing id; either way return `inserted=false` with a nil error on a
  repeat. Leave `consistent` / `root_rebuilt` NULL — they are the consistency-check step's job. There
  is no `status` column on `checkpoints`; carry `CheckpointRecord.Status` for the follower's later use
  (and for the verified-only-advances assertion at the call site), but persist only the columns the
  schema has this step.
- **`AdvanceFollowState` must not auto-unfreeze (Correctness rule, ADR-0006).** Upsert with
  `ON CONFLICT(hub_id) DO UPDATE SET last_size=excluded.last_size`, *omitting* `frozen` from the SET
  list so a frozen hub stays frozen. A test must prove: set `frozen=1` via raw SQL, call
  `AdvanceFollowState`, then read `frozen` is still 1 and `last_size` is the new value.
- Match existing file conventions: leading package-purpose docstring continuation, evergreen
  per-function docstrings, `_ = rows.Close()` / error-preserving idioms, no `t.Skip` / `//nolint`.

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...` all exit 0).
- `gofmt -l internal/store` prints nothing (no listed files).
- `go test -run TestStore ./internal/store` passes (the existing 4 subtests stay green).
- `go test -run 'TestUpsertHub|TestRecordCheckpoint|TestFollowState|TestAdvanceFollowState' ./internal/store`
  passes, with these mechanical assertions:
  - `UpsertHub` called twice with the same `domain` returns the **same** `hub_id` and leaves exactly
    one `hubs` row (`SELECT count(*) FROM hubs == 1`).
  - `RecordCheckpoint` of a `(hub_id, tree_size, root)` then a second identical call returns the same
    id with `inserted == false` and leaves exactly one `checkpoints` row.
  - `FollowState` on an unknown `hubID` returns the zero `FollowState{}` and a nil error.
  - After raw `UPDATE follow_state SET frozen=1`, `AdvanceFollowState(hubID, N)` leaves
    `SELECT frozen FROM follow_state == 1` and `SELECT last_size FROM follow_state == N`.
  - Restart survival: write via the methods, `Close`, reopen the same path, and the rows are still
    readable (mirrors `TestStoreRestartSurvival`).

## Done When
`advance` has added the typed `UpsertHub` / `RecordCheckpoint` / `FollowState` / `AdvanceFollowState`
methods (with their plain store-owned structs) plus driving tests, every Verification criterion passes
with `mise run check` green, and `internal/store` still imports no `internal/logclient`.
