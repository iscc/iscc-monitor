# Next Work Package

## Step: Real OTS `Upgrader` + `main.go` wiring — first production caller for `OTSTick`/`ots.Confirmed`

## Advances
Target → **OTS / Bitcoin anchoring** milestone:

> stamp each distinct observed root daily (`UNIQUE(hub, tree_size, root)`) + background upgrade loop
> (pending → Bitcoin-confirmed) + serve `.ots`; **never blocks the follower**. **Verify:** a stamped
> root upgrades to Bitcoin-confirmed and the served `.ots` verifies with the standard `ots` client.

This is the step `state.md` (§ Next Milestone #1) and the prior `review` (handoff `**Next:**`) both name as
mandatory now: the real `Upgrader` closure + the live `Run` goroutine wired into `cmd/iscc-monitor/main.go`,
giving `follower.OTSTick` and `ots.Confirmed` their **first production caller**. The drift watch-line is at
its edge — five OTS plumbing steps have landed with **zero production callers** (`OTSTick`/`ots.Confirmed`
unreferenced in `cmd/`, re-verified this iteration); another seam-only step would read as drift. This step
closes that line: it is the increment after which a stamped pending root can actually transit
pending → Bitcoin-confirmed through a real `opentimestamps.UpgradeSequence` + `ots.Confirmed` classify,
driven by a production ticker. It also closes the open `normal` issue
**"`ots.Confirmed` wraps an oversized `uint64` Bitcoin height to a negative `int64`"** — folded in per the
handoff: fix it on the step that first persists `BTCHeight` through a real `Upgrader`.

## Goal
Make a stamped root's pending → confirmed upgrade run in production: add a real `internal/otsclient`
calendar-HTTP `Upgrader` closure (`UpgradeSequence` → serialize → `ots.Confirmed`), fix the
`>math.MaxInt64` height fail-closed guard in `internal/ots`, and wire a background OTS ticker calling
`follower.OTSTick` into `cmd/iscc-monitor/main.go` — off the poll path, log-and-continue, so OTS never
blocks the follower (ADR-0004).

## Scope
- **Create**: `/workspace/iscc-monitor/internal/otsclient/client.go` — new package: the calendar-HTTP
  `Upgrader` builder (a closure of type `follower.Upgrader`) that calls `opentimestamps.UpgradeSequence`
  then `ots.Confirmed`, plus a `Stamp` helper that submits a digest and returns the initial serialized
  pending sequence bytes. Thin wrapper over `github.com/nbd-wtf/opentimestamps` + `internal/ots`.
- **Create**: `/workspace/iscc-monitor/internal/otsclient/client_test.go` — offline test of the closure's
  classify/serialize seam against bundled `.ots` fixtures (test file, not counted).
- **Modify** (≤3 non-test files):
  1. `/workspace/iscc-monitor/internal/ots/ots.go` — fold in the
     `att.BitcoinBlockHeight > math.MaxInt64` fail-closed guard before the `int64` cast (the `normal` issue).
  2. `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — build the real `Upgrader` from
     `internal/otsclient`, start a background OTS ticker goroutine (sibling of `serveMetrics`) that calls
     `follower.OTSTick(ctx, st, upgrader, time.Now(), logger)` on its own cadence, `defer`-stopped, off the
     poll path, log-and-continue.
  3. `/workspace/iscc-monitor/internal/ots/ots_test.go` — add the non-vacuous overflow case (test file,
     not counted toward the 3).
- **Reference**:
  - `/workspace/iscc-monitor/.claude/context/learnings/ots.md` — WASM-keep-out; `recoverParse` panic guard;
    the uncapped-uint64 height trap (the exact fix this step lands); bundled-fixtures-are-the-oracle;
    keep network off the poll path; "submission/upgrade belongs to the `Upgrader` closure that calls
    `Confirmed`".
  - `/workspace/iscc-monitor/.claude/context/learnings/follower.md` (§ "OTS upgrade-loop control core" +
    § "`Run` is deliberately untested") — `OTSTick` contract, `Upgrader` func-seam shape, the
    log-and-continue / never-block discipline; the ticker-wrapper-is-deferred note this step resolves;
    "follower imports no anchoring package" (why the closure lives outside `internal/follower`).
  - `/workspace/iscc-monitor/.claude/context/learnings/cmd-monitor.md` — `serveMetrics` background-goroutine
    + `ctx`-shutdown pattern to mirror for the OTS ticker; thin-`main` / single-`os.Exit` discipline.
  - `/workspace/iscc-monitor/internal/follower/otsloop.go` — `OTSTick(ctx, st, up, now, logger)` +
    `Upgrader`/`UpgradeResult{Confirmed, OTSBytes, BTCHeight}` signatures the closure must satisfy.
  - `/workspace/iscc-monitor/internal/store/ots.go` — `OTSRecord` fields (`OTSBytes`, `CalendarURLs`,
    `BTCHeight`, `Root`) the closure reads.
  - `/workspace/iscc-monitor/internal/ots/ots.go` + `/workspace/iscc-monitor/internal/ots/ots_test.go` —
    `Confirmed` shape + the bundled `.ots` fixtures (`hello-world.txt.ots`, `empty.ots`, `merkle1.txt.ots`)
    the closure's confirmed/pending classify branch can be tested against offline.
  - The `opentimestamps` API (verified present via `go doc`):
    `Stamp(ctx, calendarUrl string, digest [32]byte) (Sequence, error)`,
    `UpgradeSequence(ctx, seq Sequence, initial []byte) (Sequence, error)`,
    `type File struct { Digest []byte; Sequences []Sequence }`, `(File).SerializeToFile() []byte`,
    `ReadFromFile(data []byte) (*File, error)`, `(Sequence).GetAttestation() Attestation`,
    `(File).GetBitcoinAttestedSequences() []Sequence`.

## Not In Scope
- **`stampRoot` calendar submission.** Leave `/workspace/iscc-monitor/internal/follower/follower.go`'s
  `stampRoot` (lines 410-421) as the local-insert-only pending row it is. Wiring `opentimestamps.Stamp`
  into `stampRoot` (to populate `OTSRecord.OTSBytes`/`CalendarURLs` with the *initial* sequence bytes) is
  its own follow-up sub-step — keep it out so this step stays ≤3 non-test files and `follower.go` stays
  untouched. `internal/otsclient` exposes the `Stamp` helper this step, but the follower does not call it
  yet. (Consequence: in production no pending row yet carries `OTSBytes`, so `OTSTick`'s upgrade is a no-op
  until that follow-up lands — the closure is still exercised and tested here; the wiring is the deliverable.)
- **The `.ots` HTTP route** (reads `OTSForRoot`) — a separate sub-step on the proofserve/tilesserve surface.
- **Certificate §5 BITCOIN ANCHOR** (`HasClause5`, `/workspace/iscc-monitor/internal/certificate/handler.go:286`)
  and the Bitcoin-anchor panel — depend on confirmed rows existing; later sub-step.
- **Adding an `OTSRun` exported method to `internal/follower`.** Keep the ticker in `main.go` (sibling of
  `serveMetrics`), so `internal/follower` stays untouched and the wiring lives in the binary where the
  thin-`main` learning puts it.
- The other three `normal` issues (`hubDomain` ForceQuery fail-open; §4/bundle `host:port` DID encoding;
  §6 record timestamp) — not on any surface this step touches.
- WASM verifier; the M-UI exit visual-pass gate.

## Implementation Notes
- **Package boundary (why a new `internal/otsclient`, not the follower):** `Upgrader` is a func seam
  precisely so `internal/follower` imports no anchoring package (`learnings/follower.md`). The real closure
  imports BOTH `github.com/nbd-wtf/opentimestamps` AND `internal/ots`, so it MUST live outside the follower.
  `internal/otsclient` is the calendar-HTTP transport adapter; `main.go` constructs the closure from it and
  passes it as `follower.Upgrader`. This keeps `internal/ots` (the pure classify leaf) and
  `internal/otsclient` (the network transport) cleanly separate — matching `learnings/ots.md`.
- **The `Upgrader` closure body** (must satisfy `func(ctx context.Context, r store.OTSRecord)
  (follower.UpgradeResult, error)`):
  1. `opentimestamps.ReadFromFile(r.OTSBytes)` to recover the pending `File` (its `.Digest` is the stamped
     root, its `.Sequences` are the initial pending sequences). For each sequence call
     `opentimestamps.UpgradeSequence(ctx, seq, file.Digest)`; assemble the upgraded `File{Digest, Sequences}`
     and `SerializeToFile()` for the new `OTSBytes`. Document the contract: `r.OTSBytes` is the serialized
     initial sequence(s) that `stampRoot`'s future calendar-submit will persist (this step does not yet
     persist them, hence the Not-In-Scope note — but the closure's shape is fixed against it now).
  2. Classify the serialized upgraded bytes with `ots.Confirmed(upgradedBytes)` — do NOT re-implement the
     attestation walk; `ots.Confirmed` is the oracle-gated classifier.
  3. Return `follower.UpgradeResult{Confirmed: true, OTSBytes: upgradedBytes, BTCHeight: height}` on
     confirmed; `{Confirmed: false}` with nil error when still pending; a wrapped error on a transport
     fault. Network/transport faults are best-effort — return the error and let `OTSTick` back off
     (never freeze, ADR-0004).
- **`Stamp` helper** (exposed now, called by `stampRoot` later): `Stamp(ctx context.Context, calendarURL
  string, digest [32]byte) (otsBytes []byte, err error)` — wraps `opentimestamps.Stamp`, builds
  `File{Digest: digest[:], Sequences: []Sequence{seq}}`, returns `file.SerializeToFile()`. Calendar URL(s)
  come from a package const default (a public OpenTimestamps calendar, e.g.
  `https://alice.btc.calendar.opentimestamps.org`); a single default is fine for v1 — redundancy is a later
  refinement (`learnings.md`: "calendars are best-effort with backoff + redundancy"). Test the serialize
  round-trip in isolation; do NOT assert against a live calendar (no network in `go test`).
- **Height guard (`internal/ots/ots.go`, the `normal` issue):** before
  `return true, int64(att.BitcoinBlockHeight), nil`, add
  `if att.BitcoinBlockHeight > math.MaxInt64 { return false, 0, fmt.Errorf("ots.Confirmed: bitcoin height %d overflows int64", att.BitcoinBlockHeight) }`.
  Import `math`. Fail-closed (the package docstring already concedes untrusted bytes); a real Bitcoin height
  never overflows int64 so the existing golden fixtures are unaffected. **Correctness rule
  (`learnings/ots.md`):** the library's `readVarUint` has NO overflow cap, so the raw cast wraps negative —
  the guard MUST precede the cast.
- **`main.go` ticker (sibling of `serveMetrics`):** add a `runOTSLoop(ctx, st, upgrader, logger)` goroutine:
  `ticker := time.NewTicker(<cadence>)`, `defer ticker.Stop()`, `select { case <-ctx.Done(): return; case
  t := <-ticker.C: if err := follower.OTSTick(ctx, st, upgrader, t, logger); err != nil { logger.ErrorContext(...) } }`.
  Start it with `go runOTSLoop(...)` next to `go serveMetrics(...)`. Pick an OTS-specific cadence (daily-ish
  is fine, or reuse `cfg.Normal` for a dev instance — document the choice; the milestone says "daily").
  **It must NOT run on the follower's poll path** (`learnings.md`: "OTS never blocks the follower") — it is
  its own goroutine off `loop.Run`. Mirror `serveMetrics`'s `ctx`-cancel shutdown; `OTSTick` already
  injects `now` so use the ticker's `t`.
- **Correctness rule (`learnings.md` index):** `internal/ots` is NOT WASM-pure and must stay out of every
  WASM-shared closure. `internal/otsclient` imports `internal/ots` + `opentimestamps` (both server-side), so
  it inherits that — verify no WASM-pure leaf (`internal/{didweb,index,badge}`) gains an `internal/otsclient`
  or `internal/ots` dep. `database/sql` must stay out of `internal/ots`/`internal/otsclient` closures (the
  store stays a leaf; the closure takes `store.OTSRecord` by value, never imports the store's DB).
- **Oracle gate:** the `Upgrader`'s confirmed verdict flows through `ots.Confirmed`, whose bundled-`.ots`
  golden test IS the `ots verify` oracle gate (`learnings/ots.md`). The height-guard change touches that
  classify path, so the `internal/ots` oracle test must stay green AND gain a non-vacuous overflow case.
- **Don't weaken a gate:** no `//nolint`/`t.Skip`/build-tag/swallowed-error. The ticker's log-and-continue
  (a non-nil `OTSTick` result is logged, not returned) is the documented never-block discipline, identical
  to `loop.go`'s `Run` — not a swallowed error.
- **Docstrings:** file-level docstring on `client.go` (the calendar-HTTP adapter providing the real
  `Upgrader`/`Stamp`); evergreen per-function docstrings; no "new"/"improved" wording (CLAUDE.md).

## Verification
- `mise run check` is green (build + vet + test all packages incl. new `internal/otsclient`; `gofmt -l .`
  empty excluding `cauldron/`).
- `go mod verify` passes and `go mod tidy -diff` is clean (no `opentimestamps` re-add needed — it is already
  in `go.mod`; the new package only imports it).
- `go test -count=1 -run TestOTS ./internal/ots` passes, **including a new overflow case**: a crafted
  attestation height `> math.MaxInt64` makes `Confirmed` return `(false, 0, err)`; reverting the guard makes
  that case FAIL (non-vacuous) while the existing `hello-world`/`empty`/`merkle1` golden rows still pass.
- `go test -count=1 ./internal/otsclient` passes: the `Upgrader` closure, fed an already-confirmed bundled
  `.ots` fixture (so no network), returns `UpgradeResult{Confirmed: true, BTCHeight: <fixture height>}` and
  serialized `OTSBytes` that `ots.Confirmed` re-classifies as confirmed; a pending fixture returns
  `{Confirmed: false}` with nil error. Do NOT call a live calendar in the test.
- `grep -rEl "follower\.OTSTick|otsclient\." /workspace/iscc-monitor/cmd/` is non-empty —
  `OTSTick`/the real `Upgrader` now have a production caller (the drift line closes); the prior
  zero-callers grep no longer holds.
- WASM purity holds: `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge`
  succeeds, and `go list -deps ./internal/badge | grep -c -e 'internal/ots' -e 'internal/otsclient'` == 0.
- `go list -deps ./internal/ots ./internal/otsclient | grep -c '^database/sql$'` == 0 (store stays
  uncoupled).

## Done When
`mise run check` is green with `internal/otsclient` providing the real `Upgrader`/`Stamp`, `cmd/iscc-monitor`
starting an off-poll-path OTS ticker that calls `follower.OTSTick` with that `Upgrader` (first production
caller), and `internal/ots.Confirmed` fail-closing on a `>math.MaxInt64` height — all proven by the
verification commands above (overflow case non-vacuous, `otsclient` confirmed/pending classify offline, WASM
purity + store-uncoupling intact).
