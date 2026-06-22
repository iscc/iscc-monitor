## 2026-06-22 — Trap SIGTERM so `run()` shuts down gracefully under `docker stop`

**Done:** `run()` now cancels its shutdown context on **SIGTERM** as well as SIGINT, so a
container/orchestrator stop (docker / Compose / Kubernetes / systemd all send SIGTERM) drains the
in-flight poll, runs the deferred `store.Close()`, and exits 0 instead of being SIGKILLed mid-commit
over irreplaceable evidence. The single-line behavior change (`signal.NotifyContext(...,
os.Interrupt, syscall.SIGTERM)`) is extracted behind a tiny package-level `notifyShutdown()` seam so
the SIGTERM registration is unit-testable without driving the whole of `run()`.

**Files changed:**
- `cmd/iscc-monitor/main.go`: added the `"syscall"` import; extracted the shutdown-context
  registration into `notifyShutdown() (context.Context, context.CancelFunc)` which registers
  `os.Interrupt, syscall.SIGTERM`; `run()` now calls `ctx, stop := notifyShutdown()`; updated the
  package doc ("runs until SIGINT or SIGTERM") and the `run` doc ("on a clean SIGINT or SIGTERM
  shutdown Loop.Run returns ctx.Err()") to name SIGTERM per the evergreen-comment rule. No
  reordering of the `store.Open` / `defer Close` / `registerHubs` / goroutine wiring — change is
  minimal.
- `cmd/iscc-monitor/shutdown_test.go` (new, `//go:build unix`): `TestSIGTERMCancelsShutdownContext`
  drives the `notifyShutdown` seam directly, self-delivers a real `syscall.SIGTERM` via
  `syscall.Kill(syscall.Getpid(), …)`, and asserts the returned context's `Done()` fires within 2s;
  `defer stop()` tears down the process-wide trap so no sibling test swallows a later signal.

**Verification:** `mise run check` → **green** (all 27 packages `ok`; `go build` + `go vet` clean;
`gofmt -l .` empty across the whole tree).
- `go test -count=1 -run TestSIGTERM ./cmd/iscc-monitor` → **PASS** (filter catches the
  `TestSIGTERM…` name as required).
- **Mutation check (non-vacuous, run then restored):** replacing `syscall.SIGTERM` with
  `syscall.SIGUSR2` (keeps `syscall` imported so it is a *runtime* mutation, not a compile error)
  makes the test FAIL — the process takes SIGTERM's default disposition and is terminated
  (`signal: terminated`), so `<-ctx.Done()` never fires and the 2s deadline trips. The literal
  `next.md` revert (`os.Interrupt` only) also fails the test, as an unused-`syscall`-import build
  error, so the registration is load-bearing both structurally and at runtime. `main.go` restored
  byte-clean after each mutation; test green again.
- `grep -n "syscall.SIGTERM" cmd/iscc-monitor/main.go` → present at the `notifyShutdown` site; no
  bare `os.Interrupt`-only `signal.NotifyContext` remains in `cmd/iscc-monitor`.
- Cross-platform: `GOOS=windows GOARCH=amd64 go build ./cmd/iscc-monitor` succeeds — the
  unconditional `syscall.SIGTERM` production registration compiles on Windows; the Unix-only test
  *mechanism* (`syscall.Kill`/`Getpid`) is correctly excluded by `//go:build unix`, not skipped at
  runtime.

**Next:** The next front-of-queue M-Deploy `critical` slice is the **multi-stage Dockerfile + GHCR
publish workflow** (its own ≤3-file step, ADR-0013) — now unblocked because `docker stop` (SIGTERM)
drains the store cleanly. That step should also carry the `-ldflags` git-SHA build stamp on
`/healthz` or a `GET /version` (the version-stamp slice was deliberately deferred out of this one).
Cheap independent slices still open: a canonical mountable `deploy/realm-testnet.txt` (not the Go
`testdata` realm) + the documented instance-identity env values, the operability/deployment doc
(which is where the recommended Compose `stop_grace_period` line belongs), and the public
`README.md`.

**Notes:**
- Scope-clean: exactly 1 production file (within the ≤3 budget) + 1 test file. Nothing in
  `## Not In Scope` was touched — no Dockerfile, no GHCR workflow, no `internal/config` change (no
  `stop_grace_period`/shutdown-timeout knob), no realm doc, no `run()` body restructuring. The
  existing 5s `serveMetrics` shutdown timeout is unchanged.
- The `notifyShutdown` seam preserves the `Run returns ctx.Err()` contract the learnings flag:
  SIGTERM cancels the *same* `context.Background()`-rooted context whose cancel `loop.Run` returns,
  which `run()` already treats as clean via `err != context.Canceled` (the `==`, not `errors.Is`,
  comparison — left untouched and still correct).
- Oracle gate **N/A** — pure process-lifecycle wiring; no signature / RFC-6962 / Merkle / did:web /
  proof / `go.mod` / `go.sum` / `schema.sql` path touched (`git diff` is the two `cmd/iscc-monitor`
  files only). `proof/verify` purity and the WASM build are unaffected.
- Pre-existing working-tree change `M .claude/context/issues.md` was present at the start of this
  iteration (not mine); I did not stage or modify it.
