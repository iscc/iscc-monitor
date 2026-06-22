## 2026-06-22 — Review of: Trap SIGTERM so `run()` shuts down gracefully under `docker stop`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance adds `syscall.SIGTERM` (alongside `os.Interrupt`) to `run()`'s shutdown
`signal.NotifyContext`, extracted behind a tiny package-level `notifyShutdown()` seam, plus a
`//go:build unix` test that self-delivers a real SIGTERM and asserts the returned context cancels. The
change is exactly the one line of behavior `next.md` asked for; scope is 1 production file + 1 test file,
mutation-proven non-vacuous, cross-platform-clean, and `mise run check` is green. It closes the `critical`
M-Deploy ops issue so `docker stop` (SIGTERM, not SIGINT) now drains the store instead of SIGKILL-ing it
mid-commit over irreplaceable evidence.

**Verification:**
- [x] `mise run check` green — all 27 packages `ok`; `go build` + `go vet` clean.
- [x] `go test -count=1 -run TestSIGTERM ./cmd/iscc-monitor` — PASS (`TestSIGTERMCancelsShutdownContext`,
  the name matches the documented `-run TestSIGTERM` filter shorthand).
- [x] **Mutation check (run, then restored byte-clean):** replacing `syscall.SIGTERM` with
  `syscall.SIGUSR2` (keeps `syscall` imported → runtime, not compile, mutation) makes the test FAIL —
  the process takes SIGTERM's default disposition and is terminated (`signal: terminated`), so
  `<-ctx.Done()` never fires and the 2s deadline trips. `main.go` restored; `git status` shows no
  residual diff; test green again. Non-vacuous confirmed.
- [x] `gofmt -l .` empty across the whole tree; touched files clean.
- [x] `grep -n "syscall.SIGTERM" cmd/iscc-monitor/main.go` → present at the `notifyShutdown` site; the
  only `signal.NotifyContext` in `cmd/iscc-monitor` is the SIGTERM-registering one (no bare
  `os.Interrupt`-only call remains).
- [x] Cross-platform: `GOOS=windows GOARCH=amd64 go build ./cmd/iscc-monitor` AND `go vet ./cmd/iscc-monitor`
  succeed — the production registration is unconditional and compiles on Windows; the Unix-only test is
  correctly EXCLUDED there (`GOOS=windows go list -f '{{.TestGoFiles}}'` lists only `main_test.go`; Linux
  lists `main_test.go shutdown_test.go`). The build tag is a platform-capability guard on the test's
  `syscall.Kill`/`Getpid` mechanism, not a gate dodge.
- [x] Scope discipline — 1 prod file (within ≤3) + 1 test file; nothing in `## Not In Scope` touched (no
  Dockerfile, no GHCR workflow, no `internal/config` `stop_grace_period` knob, no realm doc, no `run()`
  body restructuring; the 5s `serveMetrics` shutdown timeout is unchanged).
- [x] Oracle gate **N/A** — pure process-lifecycle wiring; `git diff --name-only` = the two
  `cmd/iscc-monitor` files only. `go.mod` / `go.sum` / `internal/store/schema.sql` byte-identical
  (empty diff). No signature / RFC-6962 / Merkle / did:web / proof path; `proof/verify` purity + WASM
  build unaffected.
- [x] Gate-integrity scan of all unpushed commits (`af20446..HEAD`) — no `//nolint`, no `t.Skip`/
  `SkipNow`, no swallowed error, no deleted assertion. The only `//go:build` is the `unix` tag on the
  test file (justified platform guard, above).

**Issues found:** (none) — no new defects. Deleted the resolved `critical` *"Trap SIGTERM so the
container shuts down gracefully"* issue: the test proves the context cancels on SIGTERM, and the
unchanged `run()` wiring (deferred `st.Close()`, `Run` returning `ctx.Err()` treated as clean) satisfies
the rest of its acceptance ("`docker stop` shows graceful exit, no SIGKILL"). The issue's residual
`stop_grace_period` doc-line recommendation is subsumed by the open M-Deploy operability/deployment-doc
slice (it is a doc line, never a config key — `next.md` Not-In-Scope was explicit), so the entry is fully
closed.

**Codex second opinion:** Clean — completed after ~4 min (it self-launched a thorough cross-platform
`go tool dist list` build sweep of the `syscall.SIGTERM` portability claim, then `gofmt -d` + `go vet` +
`git show --check`, all succeeding per `/tmp/codex-review.log`). Final verdict: *"The change correctly
routes shutdown through a SIGTERM-aware signal context and the added test covers that behavior. I did not
find any introduced correctness issues."* No findings to triage; its independent cross-platform/format/vet
checks corroborate my own (mutation, Windows build+vet+test-exclusion, gate-integrity scan).

**Visual check:** n/a — no SSR surface changed. The diff is `cmd/iscc-monitor` process-lifecycle wiring
(signal registration + a test); it renders nothing. No `.dc.html` mockup comparison applies.

**Next:** The front-of-queue M-Deploy `critical` is now the **multi-stage Dockerfile + GHCR publish
workflow** (its own ≤3-file step, ADR-0013) — unblocked because `docker stop` (SIGTERM) now drains the
store cleanly. Fold in the `-ldflags` git-SHA build stamp surfaced on `/healthz` JSON or a tiny
`GET /version` (the version-stamp slice was deferred out of the SIGTERM step). Cheap independent slices
still open: `deploy/realm-testnet.txt` (a canonical mountable realm, not the Go `testdata` path) + the
documented instance-identity env values, the operability/deployment doc (the natural home for the
recommended Compose `stop_grace_period` line and the persistence/egress/exposure ops asks), and the
public root `README.md` (a `target.md` "Done When" requirement → DONE is not reachable until it exists).
M-Deploy is 0/Verify with 5 `critical` ops issues remaining after this close.

**Notes:**
- The test sends SIGTERM *before* selecting on `ctx.Done()`; this is correct, not racy —
  `signal.NotifyContext` registers the handler synchronously before `notifyShutdown()` returns, so a
  signal delivered afterward cannot be lost. `defer stop()` tears down the process-wide trap; the main
  suite has no `t.Parallel`, so no sibling test races on the handler. Matches the `next.md` design exactly.
- The `notifyShutdown` seam preserves the `Run returns ctx.Err()` contract `learnings/cmd-monitor.md`
  flags: SIGTERM cancels the SAME `context.Background()`-rooted context whose cancel `loop.Run` returns,
  which `run()` already treats as clean via `err != context.Canceled` (the bare `==`, left untouched and
  still correct — the cancel error is not wrapped).
- Pre-existing working-tree change `M .claude/context/issues.md` was present at iteration start (from a
  prior step); the advance correctly did not stage it. This review's only `issues.md` change is the
  resolved-SIGTERM deletion above.
