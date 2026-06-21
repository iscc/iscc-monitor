<!-- area: internal/metrics, internal/metricshttp -->
<!-- indexed-as: metrics.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/metrics` + `internal/metricshttp` — Prometheus surface

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Metrics leaf (`internal/metrics`)

- **`internal/metrics` is a pure, stdlib-only Prometheus text-exposition renderer** — direct imports
  are exactly `{fmt io sort strconv strings sync}` (no `net`/`os`/`time`; the `os`/`time` in the
  transitive closure ride in via `fmt` only, same nuance as `internal/didweb`). WASM build is green and
  `net/http`/`database/sql`/`log/slog` are absent from the closure. A `*Registry` (RWMutex-guarded)
  holds four families: counters `iscc_monitor_violations_total{hub_id,kind}` /
  `iscc_monitor_poll_failures_total{hub_id}` and gauges `iscc_monitor_hub_status{hub_id,status}` /
  `iscc_monitor_last_observed_at{hub_id}`. `WriteText`/`String` render fixed-order families with
  lexically-sorted sample lines for byte-stable output. `go.mod`/`go.sum` byte-identical (stdlib-only).
- **The golden is mutation-proven, not asserted.** Reviewer ran two compiling mutations (reverted):
  (1) reverse the sort → `TestRenderGolden` fails on ordering; (2) double-count `IncViolation` →
  `TestRenderGolden`+`TestRenderMinimalRequired`+`TestConcurrentMutateAndRender` all fail. A
  no-sort mutation is a *compile* error (unused `sort`) — use an inverting/reverse mutation to prove
  the sort is load-bearing. The leaf is a pure formatter with no signature/RFC-6962/Merkle/did:web/
  proof/tile path, so the **oracle/conformance gate is correctly N/A** (trust root re-arms at `fsck`).
- **Wiring-slice gotcha: the `status` label set is the GLOSSARY set, not `Status.String()`.** The leaf
  renders `status` verbatim without validating it; the glossary set is
  `verified|unresolvable|unverified|frozen|inactive`, but `logclient.Status.String()` returns
  `verified|unverified|unresolvable|rotated` (note `rotated` and `frozen` are different axes). The
  `/metrics` wiring slice MUST map the follower verdict → glossary status, else the gauge emits a
  non-glossary `status` value. `kind` is safe — it reuses the real `violations.kind` strings verbatim.
- **`String()`'s `_ = WriteText(&b)` is a correct swallow, not a gate dodge** — `strings.Builder.Write`
  never returns an error, documented inline. `WriteText(io.Writer)` itself propagates every writer
  error. `escapeLabelValue` backslash-escapes `\`/`"`/`\n` so the renderer is total (verified an inline
  `weird"kind\nwith-newline` label renders as well-formed escaped text).
- **Metrics wiring landed and is glossary-correct (verified, not asserted).** `glossaryStatus(st,
  frozen)` folds `StatusRotated`→`"unverified"` and `frozen=true`→`"frozen"` (overrides the still-
  `StatusVerified` enum after a freeze); its `default` arm is a defensive fold-to-`"unverified"` for a
  future enum value (the four `logclient.Status` cases are all explicit) — not dead code. `recordVerdict`
  is the single nil-safe mutation point funneling all THREE non-error verdict branches (early-non-
  verified L126, freeze L146, verified-advance L178); the error-return paths deliberately do NOT fire it
  (the verdict is unknown), so `IncPollFailure` instead fires in `Tick` on `PollHub`'s non-nil return,
  on the same branch as the `ErrorContext` log. Reviewer mutation-proved both freeze metrics
  (`frozen=true→false` → `TestPollHubFork` FAILS on missing `status="frozen"`; remove `IncViolation` →
  FAILS on missing `violations_total`), reverted; tree clean. `hub_id="1"` in the asserts is the real
  first-`UpsertHub` id (probed). Oracle gate correctly N/A — no verify/proof/consistency LOGIC line
  changed (grep-confirmed), the existing equivocation/fork/shrink conformance tests still pass, go.mod/
  go.sum byte-identical, metrics WASM build green.
- **Freeze-branch metric records BEFORE the `freeze()` store call (pure registry writes, intentional).**
  If `freeze()` later returns a store error, the same poll both records `status="frozen"` AND fires
  `Tick`'s `IncPollFailure` — a benign double-signal (a store fault during freeze IS a real poll
  failure, and "frozen" truthfully reflects the detected violation). Acceptable; the comment at
  `follower.go:145` documents the ordering choice. Watch this only if a future slice makes `freeze`'s
  error path mean "violation not actually recorded."

## Metrics HTTP exposure (`internal/metricshttp` + `cmd/iscc-monitor`)

- **`internal/metricshttp.Handler(*metrics.Registry) http.Handler` is the net/http wrapper that keeps
  `internal/metrics` WASM-pure.** Its only internal dep is `internal/metrics` (verified `go list -deps`),
  and `net/http` stays out of the metrics leaf's closure (grep empty, WASM build green). The mid-write
  `_ = r.WriteText(w)` swallow is justified inline (the 200 is already on the wire after the first
  write; the only failure is a broken client connection) — NOT a gate dodge; the leaf's `WriteText`
  still propagates writer errors to other callers. Oracle gate correctly N/A (pure HTTP plumbing over
  an already-tested renderer; no signature/RFC-6962/Merkle/did:web/proof/tile line changed —
  grep-confirmed, go.mod/go.sum byte-identical).
- **The handler test asserts the LITERAL content type, not the package `contentType` constant.** A test
  comparing the served header against the same constant the handler sets is a tautology (a wrong-value
  mutation would still pass). Reviewer re-ran both mutations independently (reverted): (1) handler
  `0.0.4→0.0.3` → test FAILS on Content-Type; (2) append `EXTRA` after `WriteText` → body
  byte-equality FAILS (and the failure output confirms the served body equals `m.String()` with the
  populated `fork`/`verified` samples). Non-vacuous.
- **`serveMetrics` is a background goroutine; the follower loop stays the foreground blocker.** Smoke
  test (built binary, `ISCC_MONITOR_ADDR=127.0.0.1:41999`, `NORMAL=10m` so no tick fires): GET
  `/metrics` → 200 + the four families + the exposition Content-Type; an unknown path (`/healthz`) →
  404 (ONLY `/metrics` is served — no M3 scope creep); SIGINT exits 0 (the `<-ctx.Done()` →
  `srv.Shutdown(5s)` path closes cleanly). A bad `Addr` is logged (ERROR) from `ListenAndServe`, never
  fatal; `http.ErrServerClosed` is the clean-shutdown discriminator (logged, not surfaced) mirroring
  `Run`'s `context.Canceled`. `serveMetrics` itself is untested directly (goroutine + listener
  plumbing, no branching beyond the `ErrServerClosed` check) — acceptable, like `Loop.Run`.
- **`config.optional(get, key, fallback)` is the absent-OR-empty → fallback helper for non-validated
  keys** (distinct from `required` which fails closed, and `duration` which parses+validates). `Addr`
  takes default `:9464`; a bad address surfaces at `ListenAndServe`, not at config — matching the
  no-validation-beyond-default contract. The config golden + defaults tests assert both the `:9464`
  default (key absent) and the injected value, plus an empty-value→default case.
