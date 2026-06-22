## 2026-06-22 — Real OTS `Upgrader` + `main.go` wiring — first production caller for `OTSTick`/`ots.Confirmed`

**Done:** Added `internal/otsclient` — the calendar-HTTP transport adapter providing the real
`follower.Upgrader` closure (`UpgradeSequence` → serialize → `ots.Confirmed`) plus a `Stamp` helper —
and wired a background OTS upgrade ticker (`runOTSLoop`, sibling of `serveMetrics`, off the poll path,
log-and-continue) into `cmd/iscc-monitor/main.go`, giving `follower.OTSTick`/`ots.Confirmed` their
first production caller. Also folded in the `>math.MaxInt64` Bitcoin-height fail-closed guard in
`internal/ots` (the open `normal` issue), with a non-vacuous overflow test.

**Files changed:**
- `internal/otsclient/client.go` (new): `NewUpgrader()` (wires real `opentimestamps.UpgradeSequence`)
  over a testable injectable `seqUpgrade` seam; `buildUpgrader` closure (parse via panic-guarded
  `recoverRead` → keep Bitcoin-attested sequences verbatim, upgrade only pending ones → serialize →
  classify with `ots.Confirmed`); `Stamp(ctx, calendarURL, digest)` helper (exposed now, called by
  `stampRoot` later); `DefaultCalendarURL` const.
- `internal/otsclient/client_test.go` (new, test): offline confirmed/pending/transport-fault/garbage
  classify cases (fake `seqUpgrade`, no live calendar) + `Stamp` serialize round-trip.
- `internal/otsclient/testdata/*.ots` (new, fixtures): `hello-world.txt.ots`, `empty.ots`,
  `merkle1.txt.ots` copied verbatim from `internal/ots/testdata` (the external `ots verify` oracle).
- `internal/ots/ots.go`: `att.BitcoinBlockHeight > math.MaxInt64` fail-closed guard before the `int64`
  cast (imports `math`).
- `internal/ots/ots_test.go`: `TestOTSConfirmedHeightOverflow` (crafts an overflow-height `.ots` by
  reserializing a real fixture with `MaxInt64+1`; non-vacuous).
- `cmd/iscc-monitor/main.go`: `otsUpgradeInterval = 24h` const; `runOTSLoop` goroutine; `go
  runOTSLoop(ctx, st, otsclient.NewUpgrader(), logger)` next to `go serveMetrics`; `otsclient` import.

**Verification:** `mise run check` → GREEN (build + vet + test, all 23 packages `ok` incl. new
`internal/otsclient`). Per criterion:
- `gofmt -l .` (excl. `cauldron/`) — clean.
- `go mod verify` — all modules verified; `go mod tidy -diff` — tidy (no `opentimestamps` re-add).
- `go test -run TestOTS ./internal/ots` — pass incl. overflow case; **mutation-proven non-vacuous**:
  reverting the guard makes `TestOTSConfirmedHeightOverflow` FAIL ("want overflow error, got nil")
  while the `hello-world`/`empty`/`merkle1` golden rows still pass; restored → pass.
- `go test ./internal/otsclient` — pass: confirmed fixture → `{Confirmed: true, BTCHeight: 358391}`
  (re-classifies confirmed); pending fixture (fake upgrade returns unchanged) → `{Confirmed: false}`
  nil; transport fault → wrapped error; garbage → fail-closed. **Mutation-proven**: `BTCHeight: height+1`
  FAILS both rows; inverting the `ots.Confirmed` verdict FAILS both the confirmed and pending asserts
  (both branches load-bearing); restored → pass.
- `grep -rEl "follower\.OTSTick|otsclient\." cmd/` → `cmd/iscc-monitor/main.go` (non-empty; drift line
  closed).
- WASM purity: `GOOS=js GOARCH=wasm go build ./internal/{didweb,index,badge}` OK; each has 0 deps on
  `internal/ots`/`internal/otsclient`.
- Conformance/oracle: `didweb`/`index`/`certificate`/`logclient` re-run uncached = `ok`;
  `derive_vkey.py` reproduces `40b74463`/`22b08f3e` exactly (scratch cleaned).
- Binary smoke: builds; starts with the OTS goroutine wired; clean SIGINT exit 0.

**Next:** The follow-up named in `next.md` § Not In Scope: wire `otsclient.Stamp` into
`follower.stampRoot` (`follower.go:410-421`) so a stamped pending row actually carries `OTSBytes`/
`CalendarURLs` — until then `OTSTick`'s upgrade is a structural no-op in production (no row has
`OTSBytes` yet; the closure is fully exercised and tested here, the wiring is this step's deliverable).
That step finally lets a real root transit pending → Bitcoin-confirmed end-to-end. After it: the `.ots`
HTTP route (reads `OTSForRoot`) and certificate §5 BITCOIN ANCHOR (depend on confirmed rows existing).

**Notes:**
- **DEVIATION from a verification one-liner (not a design deviation):** `next.md` line 163-164 asserts
  `go list -deps ./internal/ots ./internal/otsclient | grep -c '^database/sql$' == 0`. `internal/ots`
  (the pure classify leaf, the real subject of the store-uncoupling rule) IS 0. `internal/otsclient`
  is **1** — unavoidable and correct: the `follower.Upgrader` contract is
  `func(ctx, store.OTSRecord) (...)`, so `otsclient` MUST import `internal/store` for the
  `store.OTSRecord` type, and `store` imports `database/sql`. This is the SAME transitive pull
  `internal/follower` already has (verified: `go list -deps ./internal/follower | grep -c database/sql`
  == 1) by the identical `store.OTSRecord` seam. The genuine load-bearing invariants all hold and were
  verified directly: **store stays a leaf** (no `internal/otsclient`/`internal/follower`/`net/http` in
  `go list -deps ./internal/store`); **follower stays anchoring-free** (no `otsclient`/`internal/ots`/
  `opentimestamps` import); the closure takes `store.OTSRecord` by value and opens NO DB handle. The
  `>= 0` check on `internal/ots` is the load-bearing one and passes.
- **Library finding (load-bearing for the closure design):** `opentimestamps.UpgradeSequence`
  unconditionally does a calendar HTTP GET — even on an already-Bitcoin-attested sequence, where it
  builds an empty-host URL (`http:///timestamp/...`) and FAILS. So the closure upgrades ONLY
  `file.GetPendingSequences()` and keeps `file.GetBitcoinAttestedSequences()` verbatim. This is the
  correct reading of `next.md`'s "for each sequence call UpgradeSequence" (intent: upgrade pending →
  confirmed); calling it on every sequence would break the offline confirmed-fixture test and is wrong
  against a confirmed root. Worth a learnings note.
- **OTS cadence:** chose `otsUpgradeInterval = 24h` (the milestone's "daily") as a `main.go` const
  rather than a new config key — adding `ISCC_MONITOR_OTS_INTERVAL` would have touched `config.go` (a
  4th non-test file) and is out of scope. Documented inline; a config knob is a trivial later add.
- **No pending row carries `OTSBytes` in production yet** (the `stampRoot` calendar-submit is the
  deferred follow-up), so `OTSTick`'s upgrade is currently a no-op against real data — the closure and
  ticker are the deliverable + fully tested; the no-op resolves with the next step. This matches
  `next.md` § Not In Scope.
- Scope: 3 non-test files modified (`internal/ots/ots.go`, `cmd/iscc-monitor/main.go`) + 1 created
  (`internal/otsclient/client.go`) = 3 within the ≤3 budget; tests + testdata not counted. Nothing in
  `## Not In Scope` touched (`follower.go` byte-unchanged, no `.ots` route, no certificate §5).
- The other three `normal` issues (`hubDomain` ForceQuery; §4/bundle `host:port` DID; §6 timestamp)
  are on untouched surfaces — not addressed (out of scope).
