# Handoff

## 2026-06-20 — Review of: Typed store persistence helpers for the checkpoint verdict (`hubs` / `checkpoints` / `follow_state`)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added the four typed methods the follower will drive —
`UpsertHub` / `RecordCheckpoint` / `FollowState` / `AdvanceFollowState` — plus the plain store-owned
`CheckpointRecord` / `FollowState` structs and a `unixOrNil` helper, in one new source file with a
dedicated driving test file. `internal/store` stays a leaf (no `internal/logclient` import). Scope is
tight (1 non-test source file), SQL matches `schema.sql` verbatim, edge cases are covered, and every
gate is green. Independent reviewer audit reconfirmed every claim.

**Verification:**
- [x] `mise run check` → green (exit 0; build + vet + test on go1.24).
- [x] `gofmt -l .` → empty (no listed files).
- [x] `go test -run TestStore ./internal/store` → PASS (existing 4 subtests stay green).
- [x] `go test -run 'TestUpsertHub|TestRecordCheckpoint|TestFollowState|TestAdvanceFollowState'`
  (+`TestCheckpointHelpers`) → PASS (7 new tests). Re-ran uncached (`-count=1`) → still PASS.
- [x] `UpsertHub` same domain → same `hub_id`, exactly 1 `hubs` row; distinct domain → 2nd row/id;
  first-insert `origin`/`base_url` preserved on re-register.
- [x] `RecordCheckpoint` re-observe → same id, `inserted=false`, 1 row; distinct `(size,root)` → new
  row; `observed_at` persisted as unix-seconds; `consistent`/`root_rebuilt` left NULL.
- [x] Zero `ObservedAt` stored as NULL (not 0) — `TestRecordCheckpointZeroObservedAtNull`.
- [x] `FollowState` unknown hub → zero `FollowState{}` + nil err.
- [x] No-auto-unfreeze: raw `UPDATE frozen=1`, then `AdvanceFollowState(99)` leaves `frozen==1` and
  `last_size==99` (asserted both via `FollowState` and a raw column read).
- [x] Restart survival: write via methods, Close, reopen → rows still readable.
- [x] Leaf purity: `go list -deps ./internal/store` shows no internal iscc-monitor deps; imports are
  stdlib + `modernc.org/sqlite` only (no `logclient` coupling).
- [x] `go mod tidy` → zero diff. No `schema.sql` / `sqlite.go` change (scope held).

**Conformance/oracle gate:** N/A this step. The diff touches only `internal/store` typed CRUD — no
signature verification, RFC-6962/Merkle, proof code, `internal/didweb`, or split-view logic.
`internal/proof` still does not exist (pre-M2). No oracle obligation; the purity gate has nothing to
regress.

**Quality-gate integrity:** Clean. Scanned all 3 unpushed commits (`@{upstream}..HEAD`) — no
`//nolint`, no `t.Skip`/`SkipNow`, no build-tag exclusions, no deleted assertions/tests, no loosened
gates. The grep hits for circumvention patterns are all markdown prose in the handoff/next describing
the rules; the only `_ =` usages in Go are `Close()` cleanup in deferred/`t.Cleanup` contexts — the
established error-preserving idiom in this package, not a swallowed error to dodge a check. Every
method body checks and wraps its errors.

**Issues found:** (none)

**Next:** The follower poll loop — the real caller of these methods. It calls
`logclient.AcceptCheckpoint` (**check `err` before the status** — a verified-but-garbled body returns
non-nil err alongside `StatusUnverified`'s zero), maps `Status.String()` +
`CheckpointInfo{Origin,TreeSize,Root}` into a `CheckpointRecord` (Root `[32]byte` → `[]byte`,
`ObservedAt` injected, never `time.Now()` in the pure layer), persists via `RecordCheckpoint`, and
calls `AdvanceFollowState` **only** on `StatusVerified`. It also writes the `hub_keys` did:web cache
and refreshes the stale sb1 `did.json` fixture + `derive_vkey.py` `HUBS` to the current key
`069d0f14`. The goroutine-ownership single-writer wrapper is the follower's concern. That step touches
the trust root indirectly (consumes `AcceptCheckpoint`) and refreshes a golden fixture — the
`derive_vkey.py` parity + `notecheck` oracle gates re-apply there.

**Notes:**
- `RecordCheckpoint` intentionally does **not** persist `CheckpointRecord.Status` — `checkpoints` has
  no status column; Status is carried for the follower's verified-only-advances decision at the call
  site. Correct per `next.md`; don't add a status column without a separate schema decision.
- `UpsertHub` re-register returns the existing id without rewriting `origin`/`base_url`. If a hub's
  `base_url` ever changes, the follower/registry step needs an explicit update path — future work, not
  a blocker (the realm-registry step is the natural home).
- `hubs` still has no UNIQUE on `domain`; `UpsertHub` is select-then-insert, sound under the
  single-writer pool (`SetMaxOpenConns(1)`). A UNIQUE-on-domain would let it become `ON CONFLICT`, but
  that is a separate reviewable schema change — fine to defer.
- Pushed to `origin/develop` on PASS (human merges develop→main via CI-gated PR; never push main).
