# Next Work Package

## Step: OpenTimestamps adapter — `internal/ots` confirmed-check over `nbd-wtf/opentimestamps`, golden-tested offline against the library's bundled `.ots` vectors

## Advances
Toward the **OTS / Bitcoin anchoring** milestone Verify criterion (`target.md` lines 199-203):

> stamp each distinct observed root daily … + background upgrade loop (pending → Bitcoin-confirmed) +
> serve `.ots`; **never blocks the follower**. **Verify:** a stamped root upgrades to Bitcoin-confirmed
> and the served `.ots` verifies with the standard `ots` client.

This is the **crypto-payload first sub-step** of that criterion, not more plumbing. The milestone as a
whole — real `Upgrader` closure + live `Run` wiring in `main.go` + the `.ots` route + certificate §5 — is
5+ files and cannot land as one ≤3-file step. The drift watch-line in `state.md` (lines 49-51) warns that
"a *fifth* plumbing step that still does not call `OTSTick` from `main.go` with a real calendar client
would read as drift." This step is deliberately **not** a fifth plumbing step: it is the
`go.mod`/`go.sum` dependency add (the first for this milestone) plus the pure parse/serialize/confirmed
seam every remaining sub-step reads through, and it is **where the `ots verify` crypto-oracle gate first
applies** (`target.md` lines 51-54; `state.md` line 120). The `Upgrader` closure + `main.go` wiring is the
named, immediate next sub-step (see `## Not In Scope`) and becomes a small closure *because* this seam
exists — without it the `Upgrader` would have to inline raw library calls inside `internal/follower`,
breaking the established "follower imports no anchoring package" decision (`follower.md` lines 118-124).

No `critical`/`normal` issue preempts this milestone work; the 3 open `normal` issues are all
"fix-on-next-touch" items in surfaces (`registry`, `certificate/handler.go`) this step does not touch.

## Goal
Add a pure `internal/ots` adapter wrapping `github.com/nbd-wtf/opentimestamps` that answers the one
question the upgrade loop, the `.ots` route, and certificate §5 all need: **given serialized `.ots`
bytes, is this root Bitcoin-confirmed yet, and at what block height?** Pin it to ground truth with a
network-free golden test over the library's own bundled `.ots` example fixtures.

## Scope
- **Create**: `/workspace/iscc-monitor/internal/ots/ots.go` — the adapter: `Confirmed(otsBytes []byte) (confirmed bool, height int64, err error)`. Keep the surface minimal (YAGNI); add a thin serialize/parse helper ONLY if needed to make the test hermetic.
- **Create**: `/workspace/iscc-monitor/internal/ots/ots_test.go` — golden table test over the bundled example vectors (test file, not counted).
- **Create**: `/workspace/iscc-monitor/internal/ots/testdata/` — copy the **load-bearing fixtures** into the repo (see Implementation Notes) so the test is hermetic and never reads the module cache.
- **Modify**: `/workspace/iscc-monitor/go.mod` + `/workspace/iscc-monitor/go.sum` — `go get github.com/nbd-wtf/opentimestamps@v0.4.0` (the first dependency add for this milestone; counts as one of the ≤3 non-test slots alongside `ots.go`).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/follower.md` (lines 108-130 — the `Upgrader` func-seam decision + the OTS-never-blocks invariant this adapter serves; `UpgradeResult{Confirmed, OTSBytes, BTCHeight}` shape).
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — the `OTSRecord` columns + `unixOrNil`/zero-as-NULL conventions the loop carries (`BTCHeight int64`).
  - `/workspace/iscc-monitor/internal/follower/otsloop.go` — the `UpgradeResult{Confirmed bool; OTSBytes []byte; BTCHeight int64}` this adapter must feed.
  - `/workspace/iscc-monitor/internal/store/ots.go` — `OTSRecord` (the `OTSBytes []byte`, `BTCHeight int64` columns).
  - `/workspace/iscc-monitor/cauldron/iscc-hub/specs/iscc-log.md` §12.1 (line 356 — Bitcoin anchoring via OpenTimestamps).

## Not In Scope
- **The real `Upgrader` closure + live `Run` wiring in `main.go`** — the immediate NEXT sub-step. It will (a) make `stampRoot` actually submit to a calendar via `opentimestamps.Stamp` and persist the initial sequence bytes into `OTSRecord.OTSBytes`/`CalendarURLs` (today `stampRoot` writes a bare pending row with no bytes — `internal/follower/follower.go:410-421`), (b) implement the `follower.Upgrader` as a closure that calls `opentimestamps.UpgradeSequence` then this adapter's `Confirmed`, and (c) add the `Run`-style ticker wrapper that calls `OTSTick` and wire it into `cmd/iscc-monitor/main.go` off the poll path (`defer Stop()`, log-and-continue). That closes the "no production caller" drift line.
- The `.ots` HTTP route (reads `OTSForRoot`) — a later sub-step.
- Certificate **§5 BITCOIN ANCHOR** (`HasClause5` at `internal/certificate/handler.go:286`) — a later sub-step.
- **Live calendar / Bitcoin RPC I/O in this step.** No `opentimestamps.Stamp`, no `UpgradeSequence`, no `Verify(bitcoin, …)`, no `NewEsploraClient`. This adapter only *parses* already-serialized `.ots` bytes and classifies them; network submission/upgrade belongs to the `Upgrader` sub-step.
- The deferred `host:port` DID-encoding `normal` issues — not on this path; `handler.go` is untouched here.

## Implementation Notes
- **Library API (verified against v0.4.0 offline):** `opentimestamps.ReadFromFile(data []byte) (*File, error)` parses serialized `.ots` bytes; `File.GetBitcoinAttestedSequences() []Sequence` returns the sequences that terminate in a Bitcoin attestation; `Sequence.GetAttestation() Attestation` yields `Attestation{BitcoinBlockHeight uint64, CalendarServerURL string}`. So `Confirmed` is: parse → if `len(GetBitcoinAttestedSequences()) > 0` take `[0].GetAttestation().BitcoinBlockHeight` → `(true, int64(height), nil)`; else `(false, 0, nil)` (still-pending is NOT an error). A parse error from `ReadFromFile` is wrapped and returned (`fmt.Errorf("ots.Confirmed: parse: %w", err)`) — fail-closed, never a silent confirmed.
- **`BitcoinBlockHeight` is `uint64`; the store/`UpgradeResult` carry `int64`.** Cast `int64(height)`; a real Bitcoin height never overflows int64. Match `UpgradeResult.BTCHeight int64` and `OTSRecord.BTCHeight int64` so the next sub-step plugs in with no conversion seam.
- **Golden vectors = the library's own bundled `examples/*.ots` (true external oracle, NOT self-referential).** These `.ots` files were produced by the real OpenTimestamps ecosystem (real Bitcoin attestations), so asserting our adapter's verdict against them is the `ots verify` oracle this milestone's Verify demands — the LLM reviewer is explicitly NOT ground truth for this path (`target.md` lines 51-54). Probed offline (under `$(go env GOMODCACHE)/github.com/nbd-wtf/opentimestamps@v0.4.0/examples/`), the deterministic classification is:
  - **Confirmed:** `hello-world.txt.ots` → height **358391**; `empty.ots` → height **129405**.
  - **Pending (not confirmed):** `incomplete.txt.ots`, `merkle1.txt.ots`, `merkle2.txt.ots`, `two-calendars.txt.ots` → `(false, 0)`.
  - **AVOID** `known-and-unknown-notary.txt.ots` / `unknown-notary.txt.ots` — they carry a deliberately-unsupported attestation type and `ReadFromFile` *errors* on them; useful only as a parse-error negative case, never as "pending".
  - Copy at minimum `hello-world.txt.ots` (confirmed, height 358391) + `merkle1.txt.ots` (pending) into `internal/ots/testdata/` so the test is hermetic (do NOT read `$GOMODCACHE` at test time). Hardcode the expected `(confirmed, height)` as literals — ground truth, not derived from our code. Optionally add `empty.ots` as a second confirmed case.
- **Purity:** keep `internal/ots` a pure parser leaf — its import closure must stay free of `net`/`net/http`/`database/sql` for the symbols it actually uses. The library's `bitcoind`/`esplora` clients pull `net/http`, so import **only** the parse/serialize symbols. Decide WASM-shareability by `go list -deps ./internal/ots | grep '^net/http$'` and record the verdict in `.claude/context/learnings/` (create `learnings/ots.md` + a pointer row): `internal/ots` need NOT be WASM-pure like `proof/verify` (OTS confirmation is server-side), but it must not drag `net/http` into anything that IS WASM-shared. If the package-level closure unavoidably drags `net/http`, document that and confirm no WASM-pure package imports `internal/ots`.
- **Docstrings:** file-level docstring states the adapter wraps `nbd-wtf/opentimestamps` to classify serialized `.ots` proofs for the upgrade loop / `.ots` route / §5; per-function evergreen docstrings; no "new"/"improved" wording (CLAUDE.md).
- **Correctness rule (learnings.md):** "OTS never blocks the follower; calendars are best-effort." This adapter does NO I/O, so it trivially honors that — but note in the learnings file that the *next* sub-step's `Upgrader` must keep the network calls off the poll path so the wiring step doesn't regress it.
- **Oracle gate now APPLIES to this package** (it interprets a Bitcoin attestation). The bundled-fixture golden test IS that gate; it must be non-vacuous — a test where flipping `Confirmed`'s `len(...) > 0` to `>= 0` or dropping the height assertion still passes is a vacuous gate. Assert the pending fixture `confirmed == false` AND `height == 0`, and the confirmed fixture the exact literal height.

## Verification
- `go get github.com/nbd-wtf/opentimestamps@v0.4.0` succeeds and `go.mod`/`go.sum` carry the dependency (`git diff --name-only -- go.mod go.sum` is non-empty).
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .` excl. `cauldron/` empty).
- `go test -count=1 -run TestOTSConfirmed ./internal/ots` passes (name the test `TestOTSConfirmed…` so the documented `-run TestOTS` shorthand catches it — the filter caveat the prior OTS reviews flagged).
- Assertion: `ots.Confirmed(<hello-world.txt.ots bytes>) == (true, 358391, nil)`.
- Assertion: `ots.Confirmed(<empty.ots bytes>) == (true, 129405, nil)` (if `empty.ots` is included).
- Assertion: `ots.Confirmed(<merkle1.txt.ots bytes>) == (false, 0, nil)` (pending, not an error).
- `go list -deps ./internal/ots | grep -c '^database/sql$'` is 0 (adapter is not a DB leaf; store stays uncoupled).
- Mutation (non-vacuous, document in the advance handoff): reverting `Confirmed` to ignore the attested-sequence check (e.g. always return `(false, 0, nil)`) FAILS the confirmed-fixture assertions; dropping the exact-height literal would let a height regression pass, so the test pins the literal.

## Done When
`internal/ots.Confirmed` correctly classifies the bundled OpenTimestamps `.ots` golden vectors (the exact
confirmed height for the attested fixtures, pending-not-error for a pending fixture) under a non-vacuous
test, the `nbd-wtf/opentimestamps` dependency is added to `go.mod`/`go.sum`, and `mise run check` is green.
