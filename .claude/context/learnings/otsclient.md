<!-- area: internal/otsclient (client.go, NewUpgrader/buildUpgrader/Stamp/recoverRead) -->
<!-- indexed-as: otsclient.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/otsclient` — OpenTimestamps calendar-HTTP transport adapter

The real `follower.Upgrader` closure + `Stamp` helper, wiring `nbd-wtf/opentimestamps`' networked
calendar calls to `internal/ots`' pure classifier. Lives OUTSIDE `internal/follower` so the follower
imports no anchoring package; `main.go`'s `runOTSLoop` is its only production caller. Read this before
touching OTS upgrade/stamp code.

## Durable traps

- **`internal/otsclient` pulls `database/sql` (count == 1) — and that is CORRECT, not a regression.** The
  `follower.Upgrader` contract is `func(ctx, store.OTSRecord) (...)`, so the adapter MUST import
  `internal/store` for the `store.OTSRecord` type, which transitively imports `database/sql`. This is the
  IDENTICAL transitive pull `internal/follower` already has (also 1) via the same `store.OTSRecord` seam.
  The load-bearing invariants still hold and are the ones to verify on every touch: `internal/store` stays
  a leaf (no `internal/otsclient`/`internal/follower`/`net/http` in its dep list, count 0); `internal/ots`
  stays at `database/sql` count 0; the closure takes `store.OTSRecord` by VALUE and opens no DB handle. Do
  NOT "fix" the otsclient count-1 by inventing a parallel record type — that would duplicate the store seam.
- **NOT WASM-pure (inherits `internal/ots` + `opentimestamps` `net/http`).** No WASM-shared leaf
  (`internal/{didweb,index,badge}`, future `proof/verify`) may import `internal/otsclient`. Verify on every
  touch: those three still `GOOS=js GOARCH=wasm go build` with 0 deps on `internal/ots`/`internal/otsclient`.
- **`opentimestamps.UpgradeSequence` unconditionally does a calendar HTTP GET — even on an
  already-Bitcoin-attested sequence**, where it builds an empty-host URL (`http:///timestamp/...`) and
  FAILS. So the closure upgrades ONLY `file.GetPendingSequences()` and keeps
  `file.GetBitcoinAttestedSequences()` VERBATIM (the offline confirmed-fixture test would break otherwise).
  Reading "for each sequence call UpgradeSequence" as "every sequence" is wrong against a confirmed root.
- **`UpgradeSequence` PANICS on a parseable-but-uncomputable proof, and the closure does NOT yet recover
  it (reviewer-confirmed, open `normal` issue).** `UpgradeSequence` → `seq.Compute(initial)` →
  `inst.Operation.Apply(...)`, and `opentimestamps/ots.go:46-54` defines `sha1`/`reverse`/`hexlify`/
  `keccak256` ops as `panic("… not implemented")` (plus `invalid instruction/attestation` panics at
  ots.go:175/254/259). `recoverRead` guards only `ReadFromFile`, NOT the `upgrade()` call at `client.go:86`,
  and `runOTSLoop` has no `recover` — so such a proof would crash the whole monitor process (ADR-0004 says
  OTS never crashes the follower). NOT exploitable today: `stampRoot` writes pending rows with EMPTY
  `OTSBytes` (the calendar-submit is deferred), so the closure fails at `recoverRead` ("invalid ots file
  header '': EOF") and `upgrade()` is unreachable until `OTSBytes` is populated. Wrap the upgrade body in
  the same panic-recover as `recoverRead` BEFORE/with the `stampRoot` calendar-submit step.
- **`UpgradeSequence` uses `http.DefaultClient` (no `Timeout`) with the deadline-free process context
  (open `normal` issue).** `runOTSLoop`'s ctx is `signal.NotifyContext(context.Background(), …)` — no
  deadline. A stalled calendar GET blocks one `OTSTick` pass indefinitely (starving later pending rows);
  it never blocks the FOLLOWER (separate goroutine off the poll path) so it is `normal`, not critical.
  Same exploitability profile as the panic (unreachable until `OTSBytes` populated, since `upgrade()` is
  where the GET fires). Fix: derive a per-request `context.WithTimeout` before each `upgrade()` call.
- **Fixtures are the same bundled `examples/*.ots` `internal/ots` uses, copied VERBATIM into
  `testdata/`** (reviewer cmp-verified byte-identical). The closure's confirmed verdict is pinned to the
  same external ground truth (`hello-world`→358391, `empty`→129405); a wrong height fails the assertion.
  Tests inject a fake `seqUpgrade` so the parse/upgrade-seam/serialize/classify path runs fully offline —
  never call a live calendar in `go test`.
