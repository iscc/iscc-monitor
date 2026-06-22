# Next Work Package

## Step: Guard the live OTS `Stamp` calendar call with a `safeStamp` (timeout + panic-recover), symmetric with `safeUpgrade`

## Advances
This step does **not** close a remaining milestone Verify criterion directly — it **preempts a `normal`
issue** that the reachable milestone targets cannot displace this iteration:

> Issue: *"The production OTS stamp path has neither a panic-recover nor a per-request timeout (the
> upgrade path has both)"* — `normal`, `[review]` (Codex P1+P2).

It preempts milestone work because the only two milestone Verify criteria still open are **not
autonomously closable this iteration**: the WASM "published" half is **human-blocked** (Pages
repo-Settings — a workflow cannot self-enable it, `state.md` "Next Milestone" 2), the WASM signature
half is flagged **design-first / STOP-candidate** (browser did:web resolution is non-trivial — handoff
`**Next:**`), and the OTS **Bitcoin-confirmed** half is **offline-unprovable** (needs a live calendar +
real BTC confirmation). Against that, this `normal` is a genuine production-correctness defect on a
**freshly-live** path that directly violates the always-loaded rule *"OTS never blocks/crashes the
follower"* (ADR-0004): a malformed calendar Stamp response **panics → crashes the whole monitor**, and a
stalled calendar **hangs the OTS goroutine forever**, starving every later pending row. It has a clean
symmetric precedent (`safeUpgrade`), is a ≤3-file change, and is fully runnable-verifiable offline.

## Goal
Route the live `otsclient.Stamp` calendar submit through a `safeStamp` wrapper that derives a
per-request `context.WithTimeout` and a `recover()`-to-error guard — exactly as `safeUpgrade`
(`client.go:157`) does for the upgrade path — so a malformed or stalled calendar response surfaces as a
wrapped best-effort error the loop backs off on, never a process crash or a hung goroutine.

## Scope
- **Create**: (none)
- **Modify**:
  - `internal/otsclient/client.go` — introduce an injectable `stampFn` seam + `safeStamp`, route the
    exported `Stamp` through it (keep `Stamp`'s public signature unchanged so `cmd/iscc-monitor` is
    untouched). **One production file.**
  - `internal/otsclient/client_test.go` — add `TestStampPanicRecovered` + `TestStampBoundsContext`
    (test file — does not count against the 3-file budget).
- **Reference**:
  - `.claude/context/learnings/otsclient.md` — the `safeUpgrade`/`recoverRead` precedent and the durable
    rule "any new live OTS calendar call MUST go through such an FFI-boundary guard … exactly like
    `safeUpgrade`." Read before editing.
  - `internal/otsclient/client.go:146-167` (`safeUpgrade`) — the pattern to mirror verbatim.
  - `internal/otsclient/client_test.go:156-200` (`TestUpgradePanicRecovered`, `TestUpgradeBoundsContext`,
    `TestStampSerializesRoundTrip`) — the test shapes to mirror (offline fakes, no live calendar).
  - `/home/dev/go/pkg/mod/github.com/nbd-wtf/opentimestamps@v0.4.0/stamp.go` — `opentimestamps.Stamp`'s
    signature `func(ctx, calendarUrl string, digest [32]byte) (Sequence, error)`; it uses
    `http.DefaultClient` (no deadline) + the panic-prone `parseCalendarServerResponse` family.

## Not In Scope
- Do **not** touch the WASM verifier signature gap, the dossier WASM caller, or anything under
  `internal/verifier` / `cmd/wasm` — that is design-first / a separate milestone arc.
- Do **not** change `cmd/iscc-monitor/main.go` `stampFunc()` wiring — `Stamp`'s exported signature stays
  identical, so `main.go` needs no edit (keeps this to one production file).
- Do **not** add calendar redundancy / multiple-calendar back-off (later refinement, per the
  `DefaultCalendarURL` docstring) — only the timeout + panic guard.
- Do **not** "fix" the unrelated `low` OTS issues (`OTSTick` nil-Stamper fall-through, `-run TestOTS`
  filter naming) — the loop skips `low`.
- Do **not** call a live calendar in any test (no network in `go test` — inject a fake `stampFn`).

## Implementation Notes
- **Mirror `safeUpgrade` exactly.** Add a `stampFn` seam type
  `type stampFn func(ctx context.Context, calendarURL string, digest [32]byte) (opentimestamps.Sequence, error)`
  (the signature of `opentimestamps.Stamp`). Add a `buildStamper(stamp stampFn) func(ctx, calendarURL,
  digest) ([]byte, error)` holding the existing File-assembly/serialize, and have the exported `Stamp`
  delegate to `buildStamper(opentimestamps.Stamp)(...)` — so tests can inject a fake `stampFn` offline,
  the same way `buildUpgrader(seqUpgrade)` is tested. (This is the parallel of the
  `NewUpgrader`/`buildUpgrader` split.)
- **`safeStamp`** wraps *only* the `stampFn` call (not the whole `buildStamper` body — matching how
  `safeUpgrade` wraps only `upgrade(...)`, not the serialize/classify): derive
  `ctx, cancel := context.WithTimeout(ctx, stampTimeout)` with `defer cancel()`, and a deferred
  `recover()` that sets the named-return error to a wrapped `"otsclient.Stamp: ... panicked: %v"`. Add a
  `const stampTimeout = 30 * time.Second` next to `upgradeTimeout` with an evergreen docstring noting it
  is a best-effort transport bound, not a safety gate (same wording class as `upgradeTimeout`).
- **Preserve the existing error wrap.** `Stamp` already wraps `opentimestamps.Stamp`'s error as
  `"otsclient.Stamp: %q: %w"`; keep that wrapping for the non-panic transport-fault path so the
  error-string contract callers see is unchanged.
- **Always-loaded rule applied:** *"OTS never blocks/crashes the follower (ADR-0004)"* + the
  `otsclient.md` durable rule. The `recover()` is a fail-closed FFI boundary returning an `err`, **not**
  a gate-dodge — do not `//nolint`, `t.Skip`, or swallow.
- **Purity unchanged:** `internal/otsclient` is already NOT WASM-pure (imports `opentimestamps` +
  `internal/ots`); this adds no import beyond `context`/`time` (both already imported). No WASM-shared
  leaf (`internal/{didweb,index,badge}`, `internal/proof/verify`) may import it.
- **Tests (offline, mirroring the upgrade tests):**
  - `TestStampPanicRecovered` — inject a `stampFn` that `panic(...)`s; assert `buildStamper(panicFn)(...)`
    (or `safeStamp` directly) returns a non-nil wrapped error and does **not** crash. Mutation anchor:
    deleting the `recover()` in `safeStamp` makes this test panic the binary.
  - `TestStampBoundsContext` — inject a `stampFn` that captures the `ctx` it receives; assert the
    captured `ctx` reports a deadline (`_, ok := ctx.Deadline(); ok == true`). Reverting `safeStamp` to
    the bare `ctx` makes this FAIL. (Mirror `TestUpgradeBoundsContext`, `client_test.go:182`.)
  - Leave `TestStampSerializesRoundTrip` intact — the File-assembly seam is unchanged (route it through
    `buildStamper` with a fake `stampFn` if the refactor moves the assembly, else leave as-is).

## Verification
- `mise run check` is green (build + vet + `go test ./...` + `gofmt -l .` empty).
- `mise exec -- go test -count=1 -run 'TestStamp|TestUpgrade' ./internal/otsclient` passes (forced fresh).
- `TestStampPanicRecovered` passes; deleting the `recover()` in `safeStamp` makes it panic/fail
  (mutation-proven, mirroring `TestUpgradePanicRecovered`).
- `TestStampBoundsContext` passes; reverting `safeStamp` to pass the bare `ctx` (no `WithTimeout`) makes
  it FAIL (the captured ctx reports no deadline).
- `git diff --stat` shows only `internal/otsclient/client.go` + `internal/otsclient/client_test.go`
  (`cmd/iscc-monitor/main.go` unchanged).
- `GOOS=js GOARCH=wasm go build ./internal/didweb ./internal/index ./internal/badge` still succeeds with
  no new dependency on `internal/otsclient` (purity guard, per `otsclient.md`).

## Done When
`Stamp` routes its calendar call through a timeout-bounded, panic-recovered `safeStamp` (symmetric with
`safeUpgrade`), `TestStampPanicRecovered` + `TestStampBoundsContext` pass and are mutation-proven, and
`mise run check` is green with `main.go` untouched.
