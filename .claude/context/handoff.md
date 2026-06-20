# Handoff

## 2026-06-20 — Review of: Bootstrap the per-network SQLite store (`internal/store`): WAL + single-writer open + embedded schema

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` stood up `internal/store` — `Open(path)`/`Close()` over the pure-Go
`modernc.org/sqlite` driver with the ADR-0005/0007 single-writer discipline (WAL + `busy_timeout=5000`
+ `foreign_keys=ON` + `synchronous=NORMAL`, `SetMaxOpenConns(1)`) applying an embedded `schema.sql` of
the nine core M1 tables idempotently on every open. Schema + open/close only, no CRUD/fetcher/follower
— exactly as scoped. Clean, well-documented, well-tested; all gates green. Independent reviewer audit
reconfirmed every claim.

**Verification:**
- [x] `mise run check` → green (exit 0; build + vet + test on go1.24.13).
- [x] `gofmt -l .` → empty (clean).
- [x] `go test -run TestStore ./internal/store` → PASS, all 4 subtests (CreatesCoreTables, EnablesWAL,
  RestartSurvival, ReopenIdempotent).
- [x] Table set contains all nine core tables — **independently verified** via a throwaway reviewer
  test: `checkpoints entry_bundles follow_state hub_keys hubs iscc_index ots tiles violations`.
- [x] `PRAGMA journal_mode` returns `"wal"` — independently confirmed, plus `busy_timeout=5000`,
  `foreign_keys=1`, `synchronous=1(NORMAL)`, `MaxOpenConnections=1` all applied in order.
- [x] Restart survival — sentinel `hubs` row readable after close/reopen of the same path.
- [x] `grep -R network internal/store/schema.sql` → comment lines only, no column (ADR-0007).
- [x] `go.mod` directive is `go 1.24.0`, no `toolchain` line; `go mod verify` clean; `go mod tidy`
  produces zero diff (tidy-clean dependency wiring). Driver pinned `modernc.org/sqlite v1.46.1`.
- [x] Bonus: FK enforcement is genuinely live — orphan `hub_keys` insert rejected (SQLITE 787).
  `SetMaxOpenConns(1)` makes the per-connection `foreign_keys` pragma stick.

**Conformance/oracle gate:** N/A this step. The diff touches only `internal/store` + dependency
wiring; it does not touch signature verification, RFC-6962/Merkle, proof code, `internal/didweb`, or
fork/shrink/equivocation logic. `internal/proof` does not exist yet (expected pre-M1). No oracle
obligation, and the purity gate has nothing to regress. CGO_ENABLED=0 build clean.

**Quality-gate integrity:** Clean. Scanned all unpushed commits — no `//nolint`, no `t.Skip`, no
swallowed errors (the `_ = db.Close()` on `Open`'s error paths is the correct error-preserving idiom),
no build-tag exclusions, no deleted assertions. The lone "build" diff match is the word "builds" in a
docstring.

**Issues found:** (none)

**Next:** Wire the stateful follower poll loop + the typed insert/query helpers the follower drives
(row-access methods land with their real caller per Not-In-Scope/YAGNI). The follower calls
`AcceptCheckpoint`, persists the verdict into `checkpoints`/`follow_state` (only `StatusVerified`
advances accepted state; the other three are recorded findings while mirroring continues), and writes
the `hub_keys` did:web cache — that step also refreshes the sb1 did.json fixture + `derive_vkey.py`
`HUBS` to the current key (`069d0f14`). Carry the `AcceptCheckpoint` caller contract forward: a
verified-but-garbled body returns a non-nil `err` alongside `StatusUnverified`'s zero, so callers must
check `err` before the status.

**Notes:**
- Scope was tight and honest — 2 source files + 1 test file + dependency wiring, all within `next.md`.
- The goroutine-ownership write wrapper is correctly deferred to the follower; this layer only caps the
  pool so a second writer can't open. Reads serialize too for now (fine pre-serving).
- Watch at serving (M2+): when the read-pool split lands, `foreign_keys` + WAL pragmas are
  per-connection and must be re-asserted on the read connections — they will not carry across a larger
  pool automatically.
- Pushed to `origin/develop` on PASS (human merges develop→main via CI-gated PR; never push main).
