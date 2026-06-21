# Handoff

## 2026-06-21 — Serve `/metrics` over HTTP and wire `metrics.New()` into the binary

**Done:** Added a thin `internal/metricshttp.Handler(*metrics.Registry) http.Handler` that renders the
registry in Prometheus text-exposition format, and wired a live `metrics.New()` into
`cmd/iscc-monitor`: it is passed as `Loop.Metrics` and served at GET `/metrics` from a background
goroutine on the new configurable `ISCC_MONITOR_ADDR` (default `:9464`), shut down on the same
signal-bound context so serving never blocks the follower loop. This closes M1's last `/metrics`
Verify criterion (production now both collects and exposes the four series).

**Files changed:**
- `internal/metricshttp/handler.go` (new): `Handler` sets `Content-Type: text/plain; version=0.0.4;
  charset=utf-8` then calls `r.WriteText(w)`. Imports only `net/http` + `internal/metrics`. The
  mid-write error is dropped with an inline comment (the 200 is already sent on first write; the only
  failure mode is a broken client connection).
- `internal/metricshttp/handler_test.go` (new): `httptest` round trip with a pre-populated registry
  (`SetHubStatus(1,"verified")`, `IncViolation(1,"fork")`) asserting HTTP 200, the literal exposition
  Content-Type, and body byte-equal to `m.String()`.
- `internal/config/config.go`: added `Config.Addr string`, `ISCC_MONITOR_ADDR` key, `defaultAddr =
  ":9464"`, a small `optional(get,key,fallback)` helper (absent/empty → default, no validation — a bad
  address surfaces at `ListenAndServe`), and updated the package + struct doc comments.
- `internal/config/config_test.go`: `Addr`/`ISCC_MONITOR_ADDR` coverage — `:9464` default when absent,
  empty falls back, injected value round-trips; existing struct-equality cases updated for the new field.
- `cmd/iscc-monitor/main.go`: construct `m := metrics.New()`, pass `Metrics: m` into the `&follower.Loop{…}`
  literal, and `go serveMetrics(ctx, cfg.Addr, m, logger)` — a `*http.ServeMux` routing `/metrics` →
  `metricshttp.Handler(m)` behind an `*http.Server` that `Shutdown`s with a 5s-timeout context on
  `ctx.Done()`; `http.ErrServerClosed` is the clean-shutdown signal (logged, not surfaced), mirroring how
  `Run` treats `context.Canceled`.

**Verification:** `mise run check` → **green** (build + vet + test, all 10 packages). Per criterion:
- [x] `mise run check` green; `gofmt -l .` empty.
- [x] `go test ./internal/metricshttp` passes: httptest round trip → 200, Content-Type ==
      `text/plain; version=0.0.4; charset=utf-8`, body byte-equal to `m.String()`.
- [x] `GOOS=js GOARCH=wasm go build ./internal/metrics` exits 0 (metrics leaf stays WASM-pure).
- [x] `go list -deps ./internal/metrics | grep '^net/http$'` empty (grep exit 1); `net/http` is present
      only in `internal/metricshttp`'s closure, by design.
- [x] `go test ./internal/config` passes with the new `Addr` default (`:9464` absent) and injected value.
- [x] `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no new dependency; `net/http` is stdlib).

**Mutation-proof (non-vacuousness, all reverted):** (1) content-type `0.0.4→0.0.3` → handler test FAILS
(the test asserts the literal exposition string, not the package constant — see Notes); (2) appending
`EXTRA` after `WriteText` → body byte-equality FAILS, and the diff confirms the served body equals
`m.String()` with the populated `fork`/`verified` samples present. Tree restored clean after each;
`grep version=0.0` confirms the committed constant is `0.0.4`.

**Next:** The `fsck` root-rebuild conformance slice (M2 Verify) — the natural point to also resolve the
two open `normal` issues (the `go mod tidy` go.sum divergence and wiring CI/`notecheck`), since that
slice first faces the trust-root oracle in CI. With `/metrics` exposed, an M3 follow-up could add `/`,
`/healthz`, and the REST surface to the same mux, but those are explicitly out of scope here.

**Notes:**
- **Oracle/conformance gate is N/A for this slice (per `next.md`).** No signature / RFC-6962 / Merkle /
  did:web / proof / tile logic line changed — pure HTTP plumbing over the already-tested renderer.
  `review` should not expect an oracle re-run. The metrics WASM-purity invariant (the only correctness
  rule this slice touches) is verified green above.
- **Handler test asserts the LITERAL content type, not the `contentType` package constant.** My first
  mutation attempt changed the constant and the test still passed (both sides referenced the same
  constant — a tautology). I hardened the test to compare against a literal `"text/plain; version=0.0.4;
  charset=utf-8"`; the re-run mutation then FAILED as required. Worth keeping in mind generally: a test
  comparing a header value against the same constant the handler uses cannot catch a wrong value.
- **Mid-write `WriteText` error is deliberately dropped with an inline comment**, not a silent swallow:
  the 200 is already on the wire after the first write, so there is no recoverable second status — the
  only failure here is a broken client connection. `WriteText`'s `io.Writer` error contract is otherwise
  honored (the leaf still propagates writer errors to other callers).
- **`serveMetrics` is intentionally background and non-blocking**; the follower loop stays the foreground
  blocker. A bad `Addr` is logged (ERROR) from `ListenAndServe`, never fatal — consistent with the
  "serving never blocks polling" rule. It is untested directly (like `Loop.Run`): it is goroutine +
  ticker/listener plumbing with no branching beyond the `ErrServerClosed` discriminator; all testable
  logic (the handler, the config default) is covered. If `review` wants a smoke test, `registerHubs`-style
  coverage would require a wider seam than this slice warrants.
- **No new dependency / go.mod untouched** — `net/http`, `errors`, `time` are all stdlib.
