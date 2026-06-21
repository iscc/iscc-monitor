# Next Work Package

## Step: Structured logging (`slog`) at the loop + binary boundary

## Goal
Introduce `log/slog` as the monitor's structured-logging backbone, replacing the two documented
stderr placeholders — the swallowed per-tick error in `loop.go` and the ad-hoc `alert` print in
`main.go` — so every poll-loop fault and every freeze alert is emitted as a structured record with
stable keys. This advances M1's explicit "structured logs" scope item with a small, stdlib-only,
mechanically-testable slice that does NOT touch the crypto/oracle path or any leaf-pure package.

## Scope
- **Create**: (none — the new test may live in a new `internal/follower/logging_test.go` or be added
  to `loop_test.go`; test-only, your call.)
- **Modify**:
  - `internal/follower/loop.go` (add a nil-safe `Logger *slog.Logger` field to `Loop`; a `logger()`
    accessor that returns `l.Logger` or `slog.Default()`; in `Run`, log the per-tick error currently
    swallowed at `loop.go:132`; optionally log the per-hub fault inside `Tick` before it is folded
    into `firstErr`).
  - `cmd/iscc-monitor/main.go` (construct a `*slog.Logger` once in `run()`, pass it as `Loop.Logger`,
    and make the freeze `alert` emit a structured `slog` record instead of `fmt.Fprintf(os.Stderr,
    …)`; the top-level `main` fatal print may stay or move to `slog` — keep it one line either way).
- **Reference** (read, do not import or edit):
  - `internal/follower/loop.go` — the swallowed error at `loop.go:131-132`, the `Loop` struct at
    `loop.go:45-53`, and `Tick`'s `firstErr` folding at `loop.go:90-110`.
  - `cmd/iscc-monitor/main.go` — the `alert` placeholder at `main.go:104-106` and the `Loop`
    construction at `main.go:65-72`.
  - `internal/follower/loop_test.go` — existing `&Loop{…}` literals at lines 80 and 168 (the new
    field must be optional so these keep compiling unchanged).
  - `.claude/context/learnings.md` — "Follower composition" (the loop-cadence + log-and-continue +
    single-writer invariants that must not regress) and "Monitor binary" sections.

## Not In Scope
- The `fsck` root-rebuild conformance slice, on-disk tile fixtures, CI/`notecheck` wiring, and the
  `go mod tidy` go.sum divergence (the two open `normal` issues). This step adds NO dependency and
  must leave `go.mod`/`go.sum` byte-identical — do not import tessera `fsck`/`client`.
- `/metrics` / `expvar` / any HTTP server. Metrics is a separate later M1 slice.
- A real alert transport (email/webhook). `alert` stays a structured log emit, not delivery; the
  `AlertFunc func(hubID int64, kind string)` signature and the once-per-transition gating in
  `PollHub`/`freeze` stay exactly as they are.
- Adding logging inside `PollHub` or any leaf package (`logclient`, `store`, `didweb`, `tiles`,
  `config`, `registry`). Keep logging at the loop/binary composition layer so the pure/WASM-shared
  packages stay log-free and leaf-clean. Do NOT change `PollHub`'s signature.
- Reformatting or restructuring `loop.go`/`main.go` beyond the logging additions.

## Implementation Notes
- **Nil-safe injection, no signature churn.** Add `Logger *slog.Logger` to the `Loop` struct. Add a
  small unexported accessor `func (l *Loop) logger() *slog.Logger { if l.Logger != nil { return
  l.Logger }; return slog.Default() }` so a bare `&Loop{…}` (as in `loop_test.go` and any future
  caller) works with zero changes — `slog.Default()` is always safe. This keeps the two existing
  `&Loop{…}` test literals and the `main.go` construction compiling without edits beyond passing the
  logger where you choose to.
- **Replace the swallowed `Run` error.** At `loop.go:131-132` the per-tick error is `_ =`-discarded
  with a comment that "the error surfaces via the store/metrics later." Replace the discard with a
  structured emit only when `err != nil`, e.g. `l.logger().ErrorContext(ctx, "poll tick failed",
  "err", err)`. Preserve the load-bearing behavior from learnings: a tick error must STILL NOT stop
  the loop (a flaky hub never aborts the network), so log-and-continue — do not `return` the error
  from `Run`. `Run` still ends only on `ctx.Done()` and never panics.
- **Structured alert in the binary.** Rewrite `alert` in `main.go` to emit `logger.Warn("hub
  frozen", "hub_id", hubID, "kind", kind)` (or `Error`-level — pick one and say why in the
  docstring). Because `alert` is a package-level func passed by value as `AlertFunc`, the cleanest
  wiring is to build the `*slog.Logger` in `run()` and capture it in a closure assigned to
  `Loop.Alert` (e.g. `Alert: func(id int64, kind string) { logger.Warn("hub frozen", …) }`), so the
  logger is injected rather than a global. Keep the once-per-transition semantics (the gating lives
  in `PollHub`/`freeze`, untouched).
- **Handler choice.** Use `slog.New(slog.NewTextHandler(os.Stderr, nil))` (or `JSONHandler`) in
  `run()`; stderr keeps parity with today's output destination. You may also `slog.SetDefault(logger)`
  so leaf-free call sites inherit it — but the loop must still read its injected `Logger` field first
  (the accessor handles the fallback).
- **Testability.** The new loop behavior is verified with a capturing handler: build a `*slog.Logger`
  over a `bytes.Buffer`-backed `slog.NewTextHandler`/`NewJSONHandler` (or a small custom recording
  handler), inject it via `Loop.Logger`, force a `Tick`/`Run` error path (e.g. a `Store` whose
  `FollowState` faults, or a fetcher that errors so `PollHub` returns non-nil), and assert exactly
  one `level==ERROR` record carrying an `"err"` attribute is emitted while the loop/tick does NOT
  abort. Prefer asserting on the captured record (level + `err`/`hub_id` keys), not on formatted
  text, so the test is format-stable. `Run` blocks on a ticker — prefer testing the log-and-continue
  path via `Tick` (deterministic, injected `now`) rather than `Run`, matching the existing loop-test
  convention that `Run` is intentionally untested.
- **Relevant Correctness rule (learnings.md).** "`proof/verify` is pure (no `net`/`os`/`sqlite`)" —
  do not add `slog`/`os` to any leaf package; logging belongs only in `loop.go` (which imports
  nothing leaf-breaking today) and `main.go` (which already imports `os`). Also keep the `loop.go`
  `due()`/`Tick` single-writer + log-and-continue invariants from the "Follower composition"
  learnings intact.
- **No dep / oracle impact.** `log/slog` is stdlib; the oracle/conformance gate is correctly N/A
  (no signature/RFC-6962/Merkle/did:web/proof/tile path touched). `go.mod`/`go.sum` must stay
  byte-identical — verify with `git diff --quiet HEAD -- go.mod go.sum`.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/follower` passes (all existing follower + loop tests, plus the new
  structured-logging test).
- A new follower test (e.g. `TestLoopLogsTickError`) captures `slog` output via an injected
  `Loop.Logger` over a recording handler, forces a tick error (faulting `Store`/fetcher), and
  asserts: exactly one `ERROR`-level record carrying an `"err"` attribute is emitted AND `Tick`
  returns the error without aborting (the loop would continue) — i.e. the previously-swallowed error
  is now observable.
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change; stdlib-only).
- `git diff --quiet HEAD -- internal/store internal/logclient internal/didweb internal/tiles
  internal/config internal/registry` exits 0 (leaf packages byte-unchanged; logging stayed at the
  composition layer).
- `GOOS=js GOARCH=wasm go build ./internal/logclient` succeeds (WASM purity preserved — no leaf
  package gained an `slog`/`os` import).

## Done When
`mise run check` is green, the new capturing-handler test proves a tick error is emitted as exactly
one structured `ERROR` record without aborting the loop, the `alert` placeholder emits a structured
`slog` record instead of a raw stderr print, and `go.mod`/`go.sum` plus every leaf package are
byte-identical to HEAD.
