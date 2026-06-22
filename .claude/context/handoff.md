## 2026-06-22 — OpenTimestamps adapter (`internal/ots`): pure confirmed-check over `nbd-wtf/opentimestamps`, golden-tested offline

**Done:** Added the pure `internal/ots` adapter wrapping `github.com/nbd-wtf/opentimestamps@v0.4.0` (the
first dependency add for the OTS milestone). `ots.Confirmed(otsBytes []byte) (confirmed bool, height int64,
err error)` parses serialized `.ots` bytes and classifies them: Bitcoin-attested → `(true, <height>, nil)`,
calendar-only → `(false, 0, nil)` (pending, not an error), unparseable → wrapped fail-closed error. Pinned to
ground truth with a hermetic golden table test over the library's own bundled `.ots` example vectors copied
into `testdata/`.

**Files changed:**
- `internal/ots/ots.go` (new): the adapter — `Confirmed` + `recoverParse` (panic-recover wrapper, see Notes).
- `internal/ots/ots_test.go` (new): golden table test (`TestOTSConfirmed`) + two fail-closed parse cases.
- `internal/ots/testdata/` (new): `hello-world.txt.ots` (confirmed 358391), `empty.ots` (confirmed 129405),
  `merkle1.txt.ots` (pending), `unknown-notary.txt.ots` (parse-error vector) — copied verbatim from the
  library's `examples/` so the test never reads `$GOMODCACHE`.
- `go.mod` / `go.sum`: `go get github.com/nbd-wtf/opentimestamps@v0.4.0` (+ btcsuite/decred/x-crypto indirects).

**Verification:** `mise run check` → GREEN (all 22 packages `ok`; `go build`/`go vet`/`go test` pass);
`gofmt -l .` excl. `cauldron/` clean.
- [x] `go get …@v0.4.0` succeeded; `git diff --name-only -- go.mod go.sum` non-empty.
- [x] `go test -count=1 -run TestOTSConfirmed ./internal/ots` passes; `-run TestOTS` shorthand catches all
  (`TestOTSConfirmed`, `…ParseError`, `…Garbage`) — the under-selection caveat the prior OTS reviews flagged
  does not recur here.
- [x] `Confirmed(hello-world.txt.ots) == (true, 358391, nil)`; `Confirmed(empty.ots) == (true, 129405, nil)`;
  `Confirmed(merkle1.txt.ots) == (false, 0, nil)`.
- [x] `go list -deps ./internal/ots | grep -c '^database/sql$'` is 0 (not a DB leaf; store stays uncoupled).
- [x] Mutation (non-vacuous, reproduced + reverted): (1) make `Confirmed` always `(false,0,nil)` → FAILS the
  confirmed rows (flag AND height); (2) drop the exact-height literal (`return true, 0`) → FAILS only the
  height assertions while the flag still passes, proving the pinned height is independently load-bearing.
- [x] Oracle/trust-root suite still green: `derive_vkey.py` reproduces `40b74463`/`22b08f3e`;
  `logclient`+`certificate`+`didweb` conformance `ok`; `.claude/.scratch/` removed after running.

**Next:** The real `Upgrader` closure + live `Run` wiring in `main.go` (the named immediate sub-step in
`next.md` § Not In Scope): make `stampRoot` submit to a calendar via `opentimestamps.Stamp` and persist the
initial sequence bytes into `OTSRecord.OTSBytes`/`CalendarURLs`; implement `follower.Upgrader` as a closure
that calls `opentimestamps.UpgradeSequence` then this adapter's `Confirmed`; add a `Run`-style ticker that
calls `OTSTick` and wire it into `cmd/iscc-monitor/main.go` off the poll path (`defer Stop()`,
log-and-continue). That closes the "no production caller" drift line. The closure stays small precisely
because this `Confirmed` seam now exists. NOTE for that step: keep the calendar/`UpgradeSequence` network
calls off the poll path so the wiring does not regress "OTS never blocks the follower."

**Notes:**
- **`net/http` IS unavoidably in `internal/ots`'s import closure** — `go list -deps ./internal/ots` shows it
  (via `net`). `next.md` anticipated this ("If the package-level closure unavoidably drags `net/http`,
  document that…"): the `opentimestamps` package is a single Go package, so importing `ReadFromFile` pulls its
  whole closure, which includes `stamp.go`/`esplora.go`'s `net/http` use. The load-bearing guarantee holds:
  **no WASM-pure package imports `internal/ots`** — verified `go list -deps` for `internal/{didweb,index,badge}`
  shows 0 hits on `internal/ots`, reverse-dep grep finds no importer, and `GOOS=js GOARCH=wasm go build
  ./internal/{didweb,index,badge}` still succeeds. `internal/ots` need NOT be WASM-pure (OTS confirmation is
  server-side); it just must never be dragged into a WASM-shared package. (`internal/proof/verify` named in
  learnings does not yet exist in the tree; the actual WASM-pure leaves are the three above.)
- **DECISION — `recoverParse` panic guard (an FFI-boundary exception, flagged for review):** the v0.4.0
  `opentimestamps.ReadFromFile` does NOT fail cleanly on all malformed input — its `parseOTSFile` over-reads
  its buffer and **panics** (`slice bounds out of range`) on short/garbage bytes (reproduced by
  `TestOTSConfirmedGarbage` with 16 garbage bytes). Since `.ots` bytes can be untrusted and the upgrade loop
  must never crash (OTS never blocks the follower, ADR-0004), `Confirmed` parses via a small `recoverParse`
  helper that recovers the library panic into the same wrapped fail-closed error. This is a `recover()` at an
  external-library boundary with a comment explaining why — NOT a gate-dodge: there is no `nolint`/`t.Skip`/
  swallowed error, and the recovered panic still surfaces as a returned `err`. `TestOTSConfirmedGarbage`
  exercises this path; without the guard `mise run check` panics the test binary.
- **LEARNINGS not written by me (role boundary).** `next.md` Implementation Notes asked to create
  `learnings/ots.md` + a pointer row, but the advance role protocol forbids modifying `learnings/`
  (that is `review`'s job). For `review` to record in a new `learnings/ots.md` + index pointer row:
  (1) `internal/ots` is NOT WASM-pure and MUST stay out of every WASM-shared package's import closure
  (the durable trap); (2) `opentimestamps.ReadFromFile` panics on malformed input → keep the `recoverParse`
  fail-closed guard when porting/extending; (3) golden heights are external ground truth (358391/129405),
  not derived; (4) the next `Upgrader` sub-step must keep calendar network calls off the poll path.
- Scope: 2 non-test slots used (`ots.go` + `go.mod`/`go.sum` counts as one per `next.md`), within ≤3. No
  `main.go`/`otsloop.go`/`store/ots.go`/`certificate` touched; `BTCHeight int64` matches `UpgradeResult` and
  `OTSRecord` so the `Upgrader` plugs in with no conversion seam. The 3 open `normal` issues are off this path.
