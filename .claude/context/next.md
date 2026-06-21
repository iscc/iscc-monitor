# Next Work Package

## Step: Serve `/metrics` over HTTP and wire `metrics.New()` into the binary

## Goal
Close M1's last open `/metrics` Verify criterion: the follower already fires all four metric
series, but nothing is exposed and `main.go` leaves `Loop.Metrics` nil. This step adds a
testable `net/http` handler rendering the registry in Prometheus text format and wires a live
`metrics.New()` registry into `cmd/iscc-monitor` so production both collects and serves metrics.

## Scope
- **Create**: `internal/metricshttp/handler.go` — a thin `net/http` wrapper:
  `Handler(r *metrics.Registry) http.Handler` returning an `http.HandlerFunc` that sets
  `Content-Type: text/plain; version=0.0.4; charset=utf-8` and calls `r.WriteText(w)`.
- **Modify**:
  - `cmd/iscc-monitor/main.go` — construct `metrics.New()`, pass it as `Loop.Metrics`, and start
    an `http.Server` on the configured address serving `/metrics` via `metricshttp.Handler`, shut
    down on the same signal-bound context. Must NOT block the follower loop.
  - `internal/config/config.go` — add an optional `ISCC_MONITOR_ADDR` key (default `:9464`) as a
    new `Config.Addr string` field so the listen address is configurable; document it in the
    package doc comment alongside the existing keys.
- **Reference**:
  - `/workspace/iscc-monitor/internal/metrics/metrics.go` — `New()`, `WriteText(io.Writer) error`,
    `String()` (the surface to wrap; do NOT add `net/http` here — it must stay WASM-pure).
  - `/workspace/iscc-monitor/internal/follower/loop.go` — `Loop.Metrics *metrics.Registry` field
    (the wiring point) and how `Tick`/`PollHub` already consume it nil-safely.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — current `run()`/`registerHubs` structure
    and the existing `signal.NotifyContext` shutdown to hang the server off.
  - `/workspace/iscc-monitor/internal/config/config.go` — the `required`/`duration` helpers, the
    optional-key pattern, and the `Config` struct to extend with `Addr`.
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` — the only existing `net/http`
    consumer, for the import/idiom precedent (no new dependency; stdlib only).

## Not In Scope
- Do NOT add `net/http` to `internal/metrics` — keep that leaf WASM-pure (verified by
  `GOOS=js GOARCH=wasm go build ./internal/metrics`). The handler lives in the new
  `internal/metricshttp` package only, never as a method on `*Registry`.
- Do NOT add any other endpoint (`/`, `/healthz`, REST surface, dashboard, CORS) — those are M3.
  This step serves exactly `/metrics`.
- Do NOT change the metric names, label sets, or the `glossaryStatus` remap (all already landed
  and golden-tested); this step only exposes what is already collected.
- Do NOT wire CI / `notecheck` or resolve the `go mod tidy` go.sum divergence (both open `normal`
  issues) — they belong with the later `fsck`-rebuild conformance slice, not here.
- Do NOT change the alert transport (still the WARN slog placeholder) — a separate later step.

## Implementation Notes
- **No new dependency.** `net/http` is stdlib; `internal/metricshttp` imports only `net/http` +
  `internal/metrics`. `go.mod`/`go.sum` must stay byte-identical
  (`git diff --quiet HEAD -- go.mod go.sum` exits 0).
- **Handler shape.** `Handler(r *metrics.Registry) http.Handler`. In the `HandlerFunc`: set the
  header `Content-Type: text/plain; version=0.0.4; charset=utf-8` (the Prometheus text-exposition
  content type) BEFORE writing the body, then `if err := r.WriteText(w); err != nil { ... }`. Since
  `WriteText` writes directly to the `ResponseWriter`, a mid-write error cannot un-send the 200 —
  handle it narrowly with an inline comment (do not swallow silently without a note). `Handler`
  requires a non-nil registry — document that in the doc comment; the binary always passes a real
  one, so no nil-guard branch is needed.
- **`main.go` wiring.** Build `m := metrics.New()` once, pass `Metrics: m` into the
  `&follower.Loop{…}` literal (it is an optional named field — just add one line). Start the metrics
  server in a goroutine BEFORE `loop.Run(ctx)` so serving never blocks polling: build a
  `*http.ServeMux` routing `/metrics` → `metricshttp.Handler(m)`, then
  `srv := &http.Server{Addr: cfg.Addr, Handler: mux}`; run `srv.ListenAndServe()` in a goroutine and
  on `ctx.Done()` call `srv.Shutdown(...)` with a short-timeout context.
  `errors.Is(err, http.ErrServerClosed)` from `ListenAndServe` is the normal-shutdown signal, not an
  error to surface — mirror how `Run`'s `context.Canceled` is treated as clean. The follower loop
  stays the foreground blocker; the HTTP server is the background goroutine.
- **Config.** Add `Addr string` to `Config` and an `ISCC_MONITOR_ADDR` key with default `:9464`
  (an exotic non-standard port per the project port convention; pick this fixed value and keep it).
  Reuse the existing optional-key pattern — absent/empty → default. No validation beyond the default
  is required (a bad address surfaces at `ListenAndServe`). Update the package doc comment's
  "Configuration keys" block and the `Config` struct doc to mention `Addr`. The `config` test must
  assert the `:9464` default when the key is absent and the injected value when present.
- **Correctness rule (learnings).** `internal/metrics` is the pure, stdlib-only WASM-shareable leaf
  (the `proof/verify`-style purity rule) — `net/http` must not enter its closure. That is the reason
  the handler is a separate package. Verify the leaf's WASM build stays green after the change.
- **Oracle/conformance gate is N/A** for this slice: no signature / RFC-6962 / Merkle / did:web /
  proof / tile logic line changes — it is pure HTTP plumbing over an already-tested renderer. Note
  this in the handoff so `review` does not expect an oracle re-run.

## Verification
- `mise run check` is green (`go build ./...` + `go vet ./...` + `go test ./...` all pass).
- `gofmt -l .` lists nothing.
- `go test ./internal/metricshttp` passes: an `httptest.NewServer(metricshttp.Handler(m))` round
  trip, with `m` pre-populated (`m.SetHubStatus(1,"verified")`, `m.IncViolation(1,"fork")`), returns
  HTTP 200, `Content-Type` header equal to `text/plain; version=0.0.4; charset=utf-8`, and a body
  byte-equal to `m.String()` (so the served bytes match the renderer exactly).
- `GOOS=js GOARCH=wasm go build ./internal/metrics` exits 0 (the metrics leaf stays WASM-pure; the
  new `net/http` import did NOT land in it).
- `go list -deps ./internal/metrics | grep '^net/http$'` is empty.
- `go test ./internal/config` passes with the new `Addr`/`ISCC_MONITOR_ADDR` default (`:9464` when
  the key is absent; the injected value when present).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no new dependency).

## Done When
`internal/metricshttp.Handler` serves `metrics.Registry.WriteText` with the Prometheus content type,
`cmd/iscc-monitor` constructs `metrics.New()`, passes it as `Loop.Metrics`, and serves it on the
configured `ISCC_MONITOR_ADDR` without blocking the follower — and every Verification check passes.
