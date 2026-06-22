## 2026-06-22 — Review of: OpenTimestamps adapter (`internal/ots`) — confirmed-check over `nbd-wtf/opentimestamps`, golden-tested offline

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** The advance added a clean, minimal `internal/ots.Confirmed` adapter wrapping
`nbd-wtf/opentimestamps@v0.4.0`, pinned to the library's own bundled `.ots` example vectors (verified
byte-identical to the library examples = genuine external oracle, not self-referential). All gates are
green, the test is non-vacuous (both mutations reproduced + reverted independently), and the WASM-purity
invariant holds. One reviewer-confirmed Codex P2 (oversized-uint64 height wraps negative while reporting
confirmed) is filed as a `normal` issue — not exploitable yet (zero importers; no production caller), so
it does not block progress, but it keeps this from a clean PASS.

**Verification:**
- [x] `mise run check` — GREEN (build + vet + test; all 22 packages `ok`, `internal/ots` included).
- [x] `gofmt -l .` (excl. `cauldron/`) — clean.
- [x] `go get …@v0.4.0` landed — `git diff --name-only HEAD~1..HEAD -- go.mod go.sum` non-empty; deps are
  the OTS lib + btcsuite/decred/x-crypto indirects; `go mod verify` = all modules verified; `go mod tidy
  -diff` = tidy (no extra/missing entries).
- [x] `go test -count=1 -run TestOTS ./internal/ots` — passes; the `-run TestOTS` shorthand catches ALL
  three tests (`TestOTSConfirmed`, `…ParseError`, `…Garbage`) — the under-selection caveat the prior OTS
  store-test review flagged does NOT recur here.
- [x] `Confirmed(hello-world.txt.ots) == (true, 358391, nil)`; `Confirmed(empty.ots) == (true, 129405,
  nil)`; `Confirmed(merkle1.txt.ots) == (false, 0, nil)` — all assert.
- [x] `go list -deps ./internal/ots | grep -c '^database/sql$'` == 0 (store stays uncoupled).
- [x] Purity: `net/http` IS in the closure (anticipated — single-package library), but the load-bearing
  rule holds: no package imports `internal/ots` (zero importers), and `internal/{didweb,index,badge}` still
  `GOOS=js GOARCH=wasm go build` with 0 `internal/ots` deps each.
- [x] Mutation (reproduced + reverted): (1) `Confirmed` → always `(false,0,nil)` FAILS the confirmed rows
  (flag AND height); (2) drop the height literal (`return true, 0`) FAILS ONLY the height assertions,
  proving the pinned height is independently load-bearing.
- [x] Oracle / trust-root gate: bundled `.ots` fixtures byte-identical to the library examples (external
  ground truth). `derive_vkey.py` still reproduces `40b74463`/`22b08f3e`; `logclient`+`certificate`+`didweb`
  +`index` conformance re-run uncached = `ok`; CI's `notecheck` signature-parity oracle job intact
  (`.github/workflows/ci.yml` build+vet+test + accept/reject-corrupted). `.claude/.scratch/` removed.
- [x] Gate-circumvention scan over the 3 unpushed commits: no `//nolint`/`t.Skip`/build-tag/swallowed-error
  added; no deleted tests/assertions. The `recover()` in `recoverParse` is a documented FFI-boundary guard
  that still surfaces the panic as a returned error — NOT a gate-dodge.

**Issues found:** One `normal` (Codex-confirmed, see below). The 6 pre-existing low/normal issues are all on
untouched surfaces (registry, certificate handler, proofserve, dashboard, store) — none resolved or made
stale by this diff.

**Codex second opinion:** One finding, `[P2]` at `ots.go:67`: an `.ots` blob carrying a `BitcoinBlockHeight
> math.MaxInt64` (corrupt/malicious calendar response) wraps to a NEGATIVE int64 in the unchecked
`int64(att.BitcoinBlockHeight)` cast while still returning `confirmed=true`. **Triaged: CONFIRMED real** —
verified the library's `readVarUint` (utils.go:47) has no overflow cap and stores the raw uint64
(parsers.go:124), and reproduced the cast wrap (`MaxInt64+1 -> -9223372036854775808`). The package's own
docstring concedes the bytes are untrusted, so fail-closed (not wrap) is correct. Filed as a `normal`
`issues.md` entry. NOT escalated to NEEDS_WORK: `internal/ots` has zero importers and the store-write that
would persist a negative `btc_height` is the deferred next sub-step, so it does not block progress — but the
guard should land before/with the `Upgrader` that persists the value.

**Visual check:** n/a — no SSR surface changed (`internal/ots` is a pure adapter with no templates; the
diff is `ots.go` + test + testdata + `go.mod`/`go.sum`).

**Next:** The named immediate sub-step (`next.md` § Not In Scope): the real `Upgrader` closure + live `Run`
wiring in `main.go` — make `stampRoot` submit to a calendar (`opentimestamps.Stamp`) and persist the initial
sequence bytes into `OTSRecord.OTSBytes`/`CalendarURLs`; implement `follower.Upgrader` as a closure that
calls `opentimestamps.UpgradeSequence` then this adapter's `Confirmed`; add a `Run`-style ticker calling
`OTSTick`, wired into `cmd/iscc-monitor/main.go` off the poll path (`defer Stop()`, log-and-continue). That
closes the "no production caller" drift line (`state.md`). **Fold in the `>math.MaxInt64` height guard from
the new issue** when that step touches the adapter path, since that step is what first persists `BTCHeight`.
Keep all calendar/`UpgradeSequence` network calls off the poll path so the wiring does not regress "OTS
never blocks the follower."

**Notes:**
- Scope discipline: 2 non-test slots used (`ots.go` + `go.mod`/`go.sum` as one), within ≤3. Nothing in
  `## Not In Scope` was touched (no `main.go`, no `otsloop.go`, no `store/ots.go`, no `certificate`, no live
  calendar I/O). `BTCHeight int64` matches `UpgradeResult`/`OTSRecord` so the `Upgrader` plugs in seamlessly.
- This is the 5th step on the OTS milestone but is correctly NOT a "fifth plumbing step" — it is the first
  `go.mod` dependency add + the pure crypto-classify seam, and it is where the `ots verify` oracle gate first
  applies (the bundled-fixture golden test IS that gate). The drift watch-line is satisfied: the next step
  must close the bar with the real `Upgrader` + `main.go` wiring.
- New learnings: created `learnings/ots.md` + index pointer row (WASM-keep-out, `recoverParse` panic guard,
  the uncapped-uint64 height trap, bundled-fixtures-are-the-oracle, keep-network-off-poll-path).
