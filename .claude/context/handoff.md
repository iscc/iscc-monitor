## 2026-06-22 — OTS upgrade-loop core: deterministic `OTSTick` over an injected `Upgrader` seam, with `Attempts`/`NextRetry` back-off

**Done:** Added the deterministic, network-free control core of the OpenTimestamps background upgrade
loop. `OTSTick(ctx, st, up, now, logger)` reads the store's due pending stamped roots, asks an injected
`Upgrader` whether each is Bitcoin-confirmed, and either marks it confirmed (`MarkOTSUpgraded`) or
records a backed-off retry (new `MarkOTSAttempted`: `Attempts++`, `NextRetry = now+backoff`). `PendingOTS`
now takes a `now` arg and filters on `next_retry IS NULL OR next_retry <= now` so a backed-off row stays
hidden until its retry elapses. No real calendar/Bitcoin dependency, no `Run` goroutine wiring, no `.ots`
route, no cert §5 — all of those are the named later sub-steps.

**Files changed:**
- `internal/follower/otsloop.go` (new): the `Upgrader` func-seam, `UpgradeResult` struct, pure
  `backoff(attempts)` capped-exponential helper (base 1h, doubling, shift cap 5, max 24h), and the
  injected-`now` `OTSTick` that mirrors `loop.go`'s `Tick` error discipline (log-and-continue, fold into
  `firstErr`, never abort the pass, OTS never blocks the follower).
- `internal/store/ots.go`: added `MarkOTSAttempted(ctx, hubID, treeSize, root, attempts, nextRetry)`
  (plain `UPDATE ots SET attempts=?, next_retry=?`, `RowsAffected`-ignored, `unixOrNil` for the zero
  time, `status` untouched so the row stays pending); narrowed `PendingOTS` to take a `now time.Time`
  arg and `AND (next_retry IS NULL OR next_retry <= ?)` (bound `now.Unix()` directly, NOT `unixOrNil`).
- `internal/follower/otsloop_test.go` (new, test): confirming / declining-backs-off / erroring-continues
  OTSTick cases + no-pending no-op + `backoff()` golden table — all over the public store seam, asserting
  on `OTSForRoot`/`PendingOTS` read-back, never on loop internals; fake injected `Upgrader`s, no network.
- `internal/store/ots_test.go` (test): updated the three `PendingOTS` call-sites to the new `now` arg;
  added `TestMarkOTSAttempted` (Attempts/NextRetry round-trip + back-off exclusion + re-surface +
  second-attempt bump), `TestMarkOTSAttemptedAbsent` (absent row = no-op), and
  `TestMarkOTSAttemptedZeroNextRetryNull` (zero `nextRetry` → NULL → immediately due).

**Verification:** `mise run check` → green (all 21 packages `ok`; `gofmt -l .` clean excl. `cauldron/`).
- [x] `go test -count=1 -run TestOTS ./internal/store ./internal/follower` passes; the new store tests
  named `TestMarkOTSAttempted*` are also confirmed passing via the broader
  `-run 'TestOTS|TestMarkOTSAttempted|TestPendingOTS|TestMarkOTSUpgraded|TestRecordOTS'` run (see Notes
  on the filter caveat).
- [x] Confirming `Upgrader` over a seeded pending row → `OTSStatusConfirmed` with the returned
  `OTSBytes`/`BTCHeight`, `UpgradedAt == now`, absent from `PendingOTS(ctx, now+1h)`.
- [x] Declining `Upgrader` → row stays `pending`, `Attempts==1`, future `NextRetry`; excluded from
  `PendingOTS(ctx, now)` (a re-tick at the same `now` is a no-op, `Attempts` stays 1); re-surfaces at
  `now+backoff` and `Attempts` increments to 2.
- [x] Erroring `Upgrader` → erroring row backed off (`Attempts==1`, still pending) AND the second seeded
  row still processed (confirmed) AND `OTSTick` returns the wrapped first error (`errors.Is`).
- [x] `go list -deps ./internal/store | grep '^net/http'` empty; store own `.Imports` unchanged by this
  edit (only SQL strings changed in `ots.go`, no import lines). `git diff --name-only -- go.mod go.sum`
  empty — no `opentimestamps` dependency added. Follower production imports unchanged (`log/slog`,
  `context`, `fmt`, `time`, `store` already present; no anchoring package).
- [x] Mutation-proven non-vacuous (all reverted, tree restored): (1) no-op `MarkOTSUpgraded` → FAILS
  `TestOTSTickConfirms` (row stays pending); (2) no-op `MarkOTSAttempted` → FAILS
  `TestOTSTickDeclinesBacksOff` (`Attempts` stays 0); (3) drop the `next_retry` WHERE filter → FAILS
  `TestMarkOTSAttempted` (back-off-exclusion) and `TestOTSTickDeclinesBacksOff` (row re-processed at the
  same `now`, `Attempts` jumps to 2).

**Next:** The **real `Upgrader`** — pull in `github.com/nbd-wtf/opentimestamps` (the first
`go.mod`/`go.sum` change), implement the OpenTimestamps calendar-HTTP client as a closure of the
`Upgrader` type (stamp/query/upgrade against the calendars in `OTSRecord.CalendarURLs`, return
`UpgradeResult{Confirmed, OTSBytes, BTCHeight}` once Bitcoin-attested), and wire the live `Run` goroutine
into `cmd/iscc-monitor/main.go` (own ticker, `defer Stop()`, log-and-continue, off the poll path). Then
the `.ots` HTTP route (reads `OTSForRoot`) and certificate §5 BITCOIN ANCHOR (`HasClause5`, consumes
confirmed rows). The `ots verify` crypto oracle gate first applies at the real-`Upgrader` step (this
slice's `Upgrader` is injected, so no `ots` crypto runs yet — see Notes).

**Notes:**
- **Oracle/conformance gate correctly N/A for this slice** (as `next.md` instructed me to state
  explicitly): `OTSTick` moves an opaque `pending`→`confirmed`/back-off over an already-fsck-verified
  accepted root. No signature/RFC-6962/Merkle/did:web/proof code is added or changed; the `Upgrader` is
  injected so tests use fake confirming/declining/erroring closures — no `ots verify` crypto. The
  verified-advance fsck/inclusion conformance tests re-ran under `mise run check` and stayed green.
- **`backoff()` policy chosen simple and documented:** capped exponential, `base=1h`, doubling, shift
  cap 5 (32h pre-clamp), max 24h. Exact cadence is not safety-critical (OTS is best-effort, ADR-0004);
  the function is pure, deterministic, monotonic-up-to-the-cap, and golden-tabled (`TestOTSBackoff`).
- **`-run TestOTS` filter caveat (the prior OTS review flagged it):** the follower tests all match
  `TestOTS*`, but my new *store* tests are `TestMarkOTSAttempted*` (the names `next.md` named), which do
  NOT match `-run TestOTS`. I verified them via the broader `-run` union above and the full `mise run
  check`. If a reviewer relies on `-run TestOTS` alone they will miss the three new store tests — use the
  union pattern or the package run.
- **The third production-file slot was intentionally left unused** (per `next.md`): no `Run` goroutine or
  `main.go` wiring, because the real `Upgrader` does not exist yet and a no-op background loop would be
  dead code. `OTSTick` carries an optional `logger *slog.Logger` arg (nil → `slog.Default()`) so the
  later `Run` wrapper can thread the binary's logger through exactly like `Loop.logger()`.
- **`OTSTick` log-and-continue mirrors `loop.go`'s `Tick` exactly:** a per-row Upgrader transport fault
  OR a store-write fault is logged with `hub_id`/`tree_size`, folded into `firstErr`, and the pass
  continues; the first error is returned only for observability (it never freezes a hub or aborts the
  loop — OTS never blocks the follower). On a transport-fault row the back-off is still recorded so the
  row re-surfaces later.
- **Store-leaf invariant held:** `MarkOTSAttempted` adds no import; `go list -deps ./internal/store |
  grep '^net/http'` is empty; the `crypto/sha256`/`internal/tiles`/`os` lines in the dep list predate
  this step (via `iscc_index.go` and transitively `fmt`) and are unchanged.
- Open `normal` issues (ForceQuery fail-open, `host:port` DID, §6 timestamp) untouched per Not-In-Scope —
  none lie on this path.
