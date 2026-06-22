# Learnings — `internal/ots` (OpenTimestamps adapter)

Pure parse/classify leaf over `github.com/nbd-wtf/opentimestamps@v0.4.0`. Answers the one question the
upgrade loop, the `.ots` route, and certificate §5 need: given serialized `.ots` bytes, is this root
Bitcoin-confirmed yet and at what height? Does NO network I/O. Read this before touching OTS code.

## Durable traps

- **`internal/ots` is NOT WASM-pure and MUST stay out of every WASM-shared package's import closure.**
  `go list -deps ./internal/ots` shows `net`/`net/http` (the library is a single Go package, so importing
  the parse symbols drags its `stamp.go`/`esplora.go` `net/http` use). That is fine — OTS confirmation is
  server-side. The load-bearing rule: no WASM-pure leaf (`internal/{didweb,index,badge}`, future
  `proof/verify`) may import `internal/ots`. Verify on every touch: those three still `GOOS=js GOARCH=wasm
  go build`, and `go list -deps` of each shows 0 hits on `internal/ots`. `database/sql` stays out of the
  closure (count must be 0) so the store stays uncoupled.
- **`opentimestamps.ReadFromFile` PANICS (slice bounds out of range) on some malformed input** instead of
  returning an error. Since `.ots` bytes can be untrusted and OTS must never block the follower (ADR-0004),
  `Confirmed` parses via the `recoverParse` panic-recover wrapper that converts the panic into the same
  wrapped fail-closed error. Keep this guard when porting/extending — it is an FFI-boundary `recover()`
  with a comment, not a gate-dodge (no `nolint`/`t.Skip`; the recovered panic still surfaces as a returned
  `err`). `TestOTSConfirmedGarbage` exercises it; without the guard `mise run check` panics the test binary.
- **`Attestation.BitcoinBlockHeight` is `uint64` and the library's `readVarUint` has NO cap** (utils.go:47
  accumulates a full uint64 with `shift += 7`, no overflow guard; parsers.go:124 stores it raw). So a
  syntactically valid but corrupt/malicious `.ots` blob can carry a height above `math.MaxInt64`, which the
  raw `int64(att.BitcoinBlockHeight)` cast wraps to a NEGATIVE int64 while still returning `confirmed=true`.
  A real Bitcoin height never overflows int64, but the adapter accepts untrusted blobs — so the height
  MUST be bounds-checked (`> math.MaxInt64` → wrapped fail-closed error) before the cast. (Open `normal`
  issue at review time; not yet exploitable because there is no production caller.)
- **The bundled `examples/*.ots` are the external `ots verify` oracle, NOT self-referential.** They carry
  real Bitcoin attestations produced by the OTS ecosystem; the golden test pins `Confirmed`'s verdict to
  them. Heights are external ground-truth literals (`hello-world.txt.ots`→358391, `empty.ots`→129405),
  never derived from this code. Copy fixtures verbatim into `testdata/` (hermetic; never read `$GOMODCACHE`
  at test time) — verified byte-identical to the library examples. `unknown-notary.txt.ots` carries a
  deliberately-unsupported attestation type and ERRORS on parse (use only as a parse-error negative case,
  never as "pending"). Keep the test non-vacuous: assert pending `confirmed==false AND height==0` and the
  confirmed fixtures the EXACT literal height (a height-drop mutation must fail independently of the flag).
- **The next sub-step's `Upgrader` must keep calendar/`UpgradeSequence` network calls OFF the poll path** so
  the live wiring does not regress "OTS never blocks the follower." This adapter does no I/O, so it honors
  that trivially; the regression risk is downstream in the closure that calls `Confirmed`.
