# Next Work Package

## Step: Trap SIGTERM so `run()` shuts down gracefully under `docker stop`

## Advances
Closes the `critical` M-Deploy ops issue **"Trap SIGTERM so the container shuts down gracefully
(run() handles SIGINT only)"** and moves toward the M-Deploy Verify criterion (target.md, ADR-0013):

> **SIGTERM** cancels the run context: the process drains the in-flight poll, runs the deferred
> `store.Close()`, and exits `0` within the grace window — asserted by a test (start the binary, send
> `SIGTERM`, assert clean exit + store closed); reverting the SIGTERM registration in
> `signal.NotifyContext` makes that test FAIL.

This is the front-of-queue M-Deploy slice: M-Deploy is 0/Verify with 6 `critical` ops issues open, the
gate is now green on every host (so the in-repo Verify items verify cleanly), and the `review` handoff's
`**Next:**` names this exact step. It is also the cleanest unblock *before* the Dockerfile/GHCR work so
`docker stop` (which sends SIGTERM, not SIGINT) drains the store rather than `SIGKILL`-ing it mid-commit
over what the glossary calls *irreplaceable evidence*.

## Goal
Make `run()` cancel its run context on **SIGTERM** as well as SIGINT, so a container/orchestrator stop
drains the in-flight poll, runs the deferred `store.Close()`, and exits 0 — proven by a mutation-grade
test that fails if the SIGTERM registration is reverted.

## Scope
- **Modify**: `cmd/iscc-monitor/main.go` (1 production file — add `syscall.SIGTERM` to the shutdown
  `signal.NotifyContext` at `main.go:133`, extracted behind a tiny named seam so it is unit-testable;
  update the package/`run` docstrings that say "until SIGINT" / "on a clean SIGINT shutdown" to name
  SIGTERM too).
- **Create**: a test in `cmd/iscc-monitor/` (e.g. `shutdown_test.go`) that drives the seam and asserts
  SIGTERM cancels the context.
- **Reference**:
  - `.claude/context/learnings/cmd-monitor.md` — the `run()` wiring + the `Run returns ctx.Err()
    unwrapped → main.go's err != context.Canceled (==, not errors.Is) is correct` note. **Do not break
    that contract** — SIGTERM must cancel the *same* context whose cancel `loop.Run` returns as clean.
  - `cmd/iscc-monitor/main.go:133` — the current `signal.NotifyContext(context.Background(),
    os.Interrupt)` site, and `main.go:131` the `defer func() { _ = st.Close() }()` it must let run.
  - `cmd/iscc-monitor/main_test.go` — existing test style to match (no test classes; simple focused
    funcs; `t.TempDir()` stores; `Test…` names selectable by `-run`).
  - The open issue **"Trap SIGTERM …"** in `.claude/context/issues.md` (the ops acceptance wording: a
    `docker stop` shows graceful exit, no SIGKILL).

## Not In Scope
- **The Dockerfile + GHCR publish workflow** — the larger `critical` follow-on (its own ≤3-file step);
  this slice only fixes the signal trap that makes `docker stop` clean.
- **Version-stamping the binary** (`-ldflags` git SHA on `/healthz` / `GET /version`) — a separate
  M-Deploy slice; do not add it here.
- **`deploy/realm-testnet.txt`, the operability/deployment doc, the root `README.md`** — independent
  M-Deploy slices, not this step.
- **A configurable `stop_grace_period` / shutdown-timeout knob** — the issue *mentions* recommending a
  Compose `stop_grace_period`, but that is a doc line for the later deployment doc, NOT a new config key;
  do not touch `internal/config`. The existing 5s `serveMetrics` shutdown timeout is unchanged.
- **Restructuring `run()`'s body** beyond extracting the one signal seam — keep the change minimal; do
  not reorder the `store.Open` / `defer Close` / `registerHubs` / goroutine wiring.

## Implementation Notes
- **The fix is one line of behavior**: `signal.NotifyContext(context.Background(), os.Interrupt,
  syscall.SIGTERM)` (add the `"syscall"` import). `syscall.SIGTERM` is defined on **every** Go platform
  incl. Windows (Go maps it), so the production change stays cross-platform per CLAUDE.md — no build tag
  on `main.go`.
- **Make it testable with a tiny seam, not by signalling through the whole `run()`.** Extract the
  registration into a package-level helper, e.g.
  `func notifyShutdown() (context.Context, context.CancelFunc) { return signal.NotifyContext(
  context.Background(), os.Interrupt, syscall.SIGTERM) }`, and call it from `run()` in place of the
  inline call (`ctx, stop := notifyShutdown(); defer stop()`). The test then calls `notifyShutdown()`
  directly, sends itself SIGTERM, and asserts the returned context's `Done()` fires within a short
  deadline. This keeps `run()` (which opens a real store, wires goroutines, and blocks on `loop.Run`)
  out of the test — driving the full `run()` would need env wiring and is the wrong seam.
- **Deliver the signal in-process** with `syscall.Kill(syscall.Getpid(), syscall.SIGTERM)` (Unix). Then
  `select { case <-ctx.Done(): /* pass */ case <-time.After(2 * time.Second): t.Fatal(...) }`. After the
  assertion, **call the returned `stop()` (cancel)** so the test un-registers the handler and does not
  leak a process-wide SIGTERM trap into sibling tests (`signal.NotifyContext` installs a *process*
  handler; use `defer stop()` / `t.Cleanup(stop)` — otherwise a later SIGTERM the OS sends could be
  swallowed). The main suite has no `t.Parallel`, so a transient process-wide trap inside one test is
  safe as long as it is torn down.
- **Cross-platform test hygiene:** `syscall.Kill` / `syscall.Getpid` self-signalling is Unix-only. Put
  the test file behind `//go:build unix` (or guard with `if runtime.GOOS == "windows" { t.Skip(...) }`)
  so `go build ./...` and `mise run check` stay green on Windows while the assertion runs on the Linux
  CI gate — this is a platform-capability skip of an OS-specific test *mechanism*, NOT a gate dodge of
  the feature (the production `syscall.SIGTERM` registration is unconditional and compiled on every OS).
- **Non-vacuous (mutation) requirement:** the test must FAIL if SIGTERM is dropped from the
  registration. Because `notifyShutdown` is the single seam, reverting it to `signal.NotifyContext(...,
  os.Interrupt)` leaves the SIGTERM handler unset → the process takes SIGTERM's *default* disposition
  (terminate), so `<-ctx.Done()` never fires and the test hits its `time.After` fatal. Confirm this by
  reverting locally and seeing the test fail, then restore.
- **Docstrings**: update the two evergreen comments that currently say "until SIGINT" (`main.go` package
  doc, ~lines 11-14) and "on a clean SIGINT shutdown Loop.Run returns ctx.Err()" (`run` doc, ~line 108)
  to read "SIGINT or SIGTERM", per CLAUDE.md "evergreen comments describe the current state."
- **Oracle gate is N/A** — pure process-lifecycle wiring; no signature / RFC-6962 / Merkle / did:web /
  proof / `go.mod` / `go.sum` / `schema.sql` path is touched. `proof/verify` purity and the WASM build
  are unaffected (this is `cmd/iscc-monitor`, not a WASM-shared leaf).

## Verification
- `mise run check` is green (build + vet + `gofmt -l .` empty + all packages `ok`).
- `go test -count=1 -run TestSIGTERM ./cmd/iscc-monitor` passes (name the test `TestSIGTERM…` so this
  filter catches it — the filter-shorthand caveat prior OTS reviews flagged).
- **Mutation check (run manually, then restore):** reverting `notifyShutdown` to
  `signal.NotifyContext(context.Background(), os.Interrupt)` makes `go test -run TestSIGTERM
  ./cmd/iscc-monitor` FAIL.
- `grep -n "syscall.SIGTERM" cmd/iscc-monitor/main.go` shows the registration is present, and no bare
  `os.Interrupt`-only `signal.NotifyContext` remains in `cmd/iscc-monitor`.

## Done When
`run()` registers SIGTERM (alongside SIGINT) on its shutdown context via a named seam, a `//go:build
unix` (or GOOS-guarded) `TestSIGTERM…` proves SIGTERM cancels that context and FAILS if the registration
is reverted, and `mise run check` is green.
