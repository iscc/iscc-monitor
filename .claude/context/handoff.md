# Handoff

## 2026-06-20 — Bootstrap the per-network SQLite store (`internal/store`): WAL + single-writer open + embedded schema

**Done:** Stood up `internal/store` — `Open(path)`/`Close()` over the pure-Go `modernc.org/sqlite`
driver (`CGO_ENABLED=0`), applying the single-writer discipline (WAL + `busy_timeout=5000` +
`foreign_keys=ON` + `synchronous=NORMAL`, `SetMaxOpenConns(1)`) and an embedded `schema.sql` of the
nine core M1 tables idempotently on every open. No insert/query/CRUD methods, no fetcher, no follower
— schema + open/close only, as scoped.

**Files changed:**
- `internal/store/sqlite.go` (new): `Open`/`Close`, `//go:embed schema.sql`, ordered pragmas,
  `SetMaxOpenConns(1)`, `database/sql` with driver name `"sqlite"`.
- `internal/store/schema.sql` (new): `CREATE TABLE IF NOT EXISTS` DDL for `hubs`, `hub_keys`,
  `checkpoints` (`UNIQUE(hub_id,tree_size,root)`), `violations`, `tiles` (`PK(hub_id,level,tile_index,
  width)`), `entry_bundles` (`PK(hub_id,bundle_index,width)`), `iscc_index` (`seq` PK +
  `INDEX(iscc_id)`), `follow_state` (`hub_id` PK), `ots` (`UNIQUE(hub_id,tree_size,root)`). No
  `network` column (ADR-0007). Times = INTEGER unix-seconds (documented in the file docstring).
- `internal/store/sqlite_test.go` (new): fresh-open table-set, WAL pragma, restart-survival,
  idempotent-reopen — all assert on observable DB state via raw SQL, never on `Store` internals.
- `go.mod` + `go.sum`: added `modernc.org/sqlite v1.46.1` (direct) + indirect deps.

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test` exit 0). Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -run TestStore ./internal/store` → PASS (4 subtests: CreatesCoreTables, EnablesWAL,
  RestartSurvival, ReopenIdempotent).
- [x] `SELECT name FROM sqlite_master WHERE type='table'` contains all nine core tables (asserted).
- [x] `PRAGMA journal_mode` returns `"wal"`.
- [x] Sentinel `hubs` row inserted before `Close()` is readable after reopening the same path.
- [x] `grep "network" internal/store/schema.sql` matches only comment lines (no column).
- [x] `grep -E '^(go|toolchain)' go.mod` → `go 1.24.0`, no `toolchain` line. Gate ran on go1.24.13.

**Next:** Wire the stateful follower poll loop + the typed insert/query helpers whose shape the
follower drives (per Not-In-Scope, row-access methods land with a real caller). The follower calls
`AcceptCheckpoint`, persists the verdict into `checkpoints`/`follow_state` (only `StatusVerified`
advances accepted state; the other three are recorded findings while mirroring continues), and writes
the `hub_keys` did:web cache — that step also refreshes the sb1 did.json fixture + `derive_vkey.py`
`HUBS` to the current key (`069d0f14`). Carry the `AcceptCheckpoint` caller contract forward (check
`err` before the status — a verified-but-garbled body is a non-nil error alongside
`StatusUnverified`'s zero).

**Notes:**
- **Driver version pin (the go-directive footgun, as `next.md` predicted).** `modernc.org/sqlite@latest`
  is `v1.52.0` and requires `go >= 1.25.0`, which rewrites the directive to `go 1.25.0` and would fail
  the gate on the pinned 1.24 toolchain. The newest version that requires only `go 1.24.0` is
  **`v1.46.1`** (`v1.46.2`..`v1.52.0` all require ≥1.25; `v1.46.1` and below require ≤1.24). Pinned
  `v1.46.1`; `go.mod` directive stays `go 1.24.0` with no `toolchain` line, verified green on go1.24.13.
  The simple reset-the-directive route was NOT viable because `v1.52.0` genuinely will not build on
  1.24, so the `next.md` fallback ("pin an older version") was the correct path.
- Reads serialize behind the single writer for now (`SetMaxOpenConns(1)`), which is intentional and
  fine pre-serving (avoids `SQLITE_BUSY` flakes). A read-pool split is deferred to when serving lands,
  per `next.md`.
- No signature/consistency/proof code touched in this step, so no conformance-oracle obligations apply
  here (still no `internal/proof/` package, no `cauldron/` CI compile path — expected pre-M1-store).
- The restart-survival test reuses `s.db` (the package-internal handle) for its raw INSERT/SELECT
  because the store exposes no public row API yet; it still proves on-disk persistence by closing the
  first handle entirely and reopening the file path before reading back. No public-API assertion is
  faked.
