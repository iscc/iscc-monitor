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
- **settled: the `>math.MaxInt64` height guard landed (`ots.go`, mutation-proven).** `readVarUint`
  (utils.go:47) has no overflow cap, so a corrupt/malicious `.ots` blob can carry a height above
  `math.MaxInt64`; `Confirmed` now fail-closes (`> math.MaxInt64` → wrapped error) BEFORE the
  `int64(att.BitcoinBlockHeight)` cast. `TestOTSConfirmedHeightOverflow` pins it (reviewer reverted the
  guard → FAIL with "want overflow error, got nil"; restored → green). Keep the guard before any cast.
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
- **Two classifiers, one core: `Confirmed` (digest-agnostic) vs `ConfirmedFor(otsBytes, root)`
  (digest-bound).** The upgrade loop + `.ots` route key on the row's OWN root, so they trust the write
  path's (root, proof) pairing → `Confirmed`. A self-verifiable surface (certificate §5) must NOT — it
  calls `ConfirmedFor`, which fail-closes (wrapped error, same contract as a parse failure) unless
  `bytes.Equal(file.Digest, root)`, so the caller's `cerr == nil` guard treats a mismatch exactly like an
  unparseable proof (silent decline, never confirmed, never 500). Both share the private `classify(file)`
  holding the attestation + `>MaxInt64` overflow guard — keep ONE parse + ONE overflow guard; the digest
  check interposes between parse and classify so a mismatched proof never reaches the int64 cast. When
  adding a NEW root-asserting caller, use `ConfirmedFor`, never `Confirmed` (a built proof is not a
  verified one). `TestOTSConfirmedForDigestBound` pins it; removing the `bytes.Equal` gate makes the
  mismatch subcase return `(true, 358391, nil)` and FAIL.
