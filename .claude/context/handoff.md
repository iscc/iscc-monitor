# Handoff

## 2026-06-21 — Structured logging (`slog`) at the loop + binary boundary

**Done:** Introduced `log/slog` as the monitor's structured-logging backbone at the composition layer
only. The previously-swallowed per-tick error in `loop.go` and the ad-hoc `alert` stderr print in
`main.go` are now structured records with stable keys; a nil-safe `Loop.Logger` field keeps every bare
`&Loop{…}` working unchanged. Stdlib-only, no dependency added, no leaf/pure package touched.

**Files changed:**
- `internal/follower/loop.go`: added a nil-safe `Logger *slog.Logger` field + unexported `logger()`
  accessor (falls back to `slog.Default()`); `Run` now emits the per-tick error as an `ErrorContext`
  record instead of `_ =`-discarding it (still log-and-continue — never returns the error, only
  `ctx.Done()` ends the loop); `Tick` logs each per-hub fault (FollowState read + PollHub) with a
  `hub_id` attribute before folding it into `firstErr`. Doc comments updated to match.
- `cmd/iscc-monitor/main.go`: `run()` builds a `*slog.Logger` over `slog.NewTextHandler(os.Stderr,
  nil)` once, sets it as default, and injects it via `Loop.Logger`; the package-level `alert` print
  was replaced by `alertFunc(logger)` returning a closure that emits `logger.Warn("hub frozen",
  "hub_id", …, "kind", …)` (WARN — a freeze is operator-actionable, not a process fault). Top-level
  `main` fatal print stays one line.
- `internal/follower/logging_test.go` (new, test-only): injects a JSON-handler-backed `*slog.Logger`
  over a `bytes.Buffer`, forces a deterministic tick fault via an always-erroring fetcher, and asserts
  exactly one `level==ERROR` record carrying `err` + `hub_id` while `Tick` returns the error without
  aborting; a second test proves the nil-`Logger` path falls back to `slog.Default()` without panicking.

**Verification:** `mise run check` → green (build + vet + test, all 8 packages). Per criterion:
- [x] `mise run check` green; `gofmt -l .` empty (ran `mise run fmt` first).
- [x] `go test -count=1 ./internal/follower` PASS (all existing follower/loop tests + the 2 new ones).
- [x] `TestLoopLogsTickError` + `TestLoopLoggerNilSafe` PASS (verbose-confirmed): exactly one ERROR
      record with `err`+`hub_id`, error surfaced from `Tick` without aborting.
- [x] `git diff --quiet HEAD -- go.mod go.sum` exit 0 (no dependency change; stdlib-only).
- [x] `git diff --quiet HEAD -- internal/store internal/logclient internal/didweb internal/tiles
      internal/config internal/registry` exit 0 (leaf packages byte-unchanged).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (WASM purity preserved).
- [x] No `log/slog` in any leaf package's import set (`go list -f` grep empty).
- Oracle/conformance gate correctly **N/A**: no signature/RFC-6962/Merkle/did:web/proof/tile path
  touched (`log/slog` is stdlib at the loop/binary layer only).

**Next:** The `fsck` root-rebuild conformance slice (M2 Verify) remains the big open item — `fsck.New(
…).Check(…)` over the `SQLiteFetcher` + the inclusion cross-check vs the hub's
`IsccLogInclusionProof`, needing the heavy `fsck`/otel/klog dep (copy/scope as the SQLiteFetcher slice
did) and real on-disk tile fixtures under `testdata/live/`. That slice is also the natural place to
resolve the pre-existing `go mod tidy` go.sum divergence and wire CI/`notecheck`. M1's remaining slice
is `/metrics` (expvar/HTTP), a small independent step now that the structured-log backbone exists — the
same injected `Logger` and the `firstErr`/`PollHub` fault points are the natural metric increment sites.

**Notes:**
- **Scope:** 2 production files modified (`loop.go`, `main.go`) + 1 new test file — within the ≤3-file
  scope (tests excluded). No refactor beyond the logging additions.
- **Test error-path choice:** drove the deterministic fault through the outbound-fetch seam (an
  always-erroring `Fetcher` → `FetchCheckpoint` fails → `PollHub` returns non-nil → `Tick` folds it),
  not a faulting store (`Loop.Store` is a concrete `*store.Store`, not an interface — no clean store
  fault injection without a wider seam change, which is out of scope). Asserts on observable store/log
  outputs only, never follower internals.
- **`Run` stays untested by design** (it blocks on a `time.NewTicker`); per the existing loop-test
  convention the log-and-continue path is proven via `Tick` with an injected `now`. The `Run` log call
  is the same `logger().ErrorContext` the `Tick` path exercises, just on the aggregate `firstErr`.
- **Both `Tick` and `Run` log.** `next.md` made the per-hub `Tick` logging optional; I added it because
  it carries the actionable `hub_id` (the `Run`-level record only has the aggregate error). `Run`'s
  record is the loop-level "a tick failed" signal. No double-counting concern — they are different
  layers; a future `/metrics` slice can pick whichever it wants.
- **Alert severity is WARN, not ERROR** (documented in `alertFunc`'s doc comment): a freeze is a
  detected, evidence-preserved, operator-actionable hub condition — distinct from a monitor-process
  fault (which is ERROR). Once-per-transition gating is untouched (lives in `PollHub`/`freeze`); the
  `AlertFunc func(int64, string)` signature is unchanged.
- **`slog.SetDefault(logger)` is set in `run()`** so any future leaf-free call site inherits it, but the
  loop still reads its injected `Logger` field first via the accessor — the binary explicitly passes
  `Logger: logger`, so it never relies on the global.
