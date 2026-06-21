# Handoff

## 2026-06-21 — Review of: Serve `/metrics` over HTTP and wire `metrics.New()` into the binary

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/metricshttp.Handler(*metrics.Registry) http.Handler` (a thin
net/http wrapper rendering the Prometheus text-exposition format) and wired a live `metrics.New()`
into `cmd/iscc-monitor`: passed as `Loop.Metrics` and served at GET `/metrics` from a background
goroutine on the configurable `ISCC_MONITOR_ADDR` (default `:9464`), shut down on the signal-bound
context. Scope is disciplined (3 code files + their tests), the metrics leaf stays WASM-pure, and I
independently re-ran both mutation-proofs and an end-to-end binary smoke test — all green. This closes
M1's last `/metrics` Verify criterion.

**Verification:**
- [x] `mise run check` green — build + vet + test, all 10 packages.
- [x] `gofmt -l .` empty (exit 0).
- [x] `go test ./internal/metricshttp` passes — httptest round trip → 200, Content-Type ==
      `text/plain; version=0.0.4; charset=utf-8`, body byte-equal to `m.String()`. Re-ran fresh
      (`-count=1`).
- [x] `go test ./internal/config` passes — `:9464` default when key absent, empty→default, injected
      value round-trips.
- [x] `GOOS=js GOARCH=wasm go build ./internal/metrics` exits 0 (metrics leaf stays WASM-pure).
- [x] `go list -deps ./internal/metrics | grep '^net/http$'` empty; `net/http` present only in
      `internal/metricshttp`'s closure (whose only internal dep is `internal/metrics`).
- [x] `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no new dependency).
- [x] **Mutation-proof (independently re-run, reverted):** (1) handler content-type `0.0.4→0.0.3` →
      handler test FAILS on Content-Type (test asserts the literal, not the package constant — no
      tautology); (2) append `EXTRA` after `WriteText` → body byte-equality FAILS, failure output
      confirms served body == `m.String()` with the `fork`/`verified` samples present.
- [x] **End-to-end smoke (built binary):** GET `/metrics` → 200 + four families + exposition
      Content-Type; `/healthz` → 404 (ONLY `/metrics` served, no M3 creep); SIGINT exits 0 (clean
      `srv.Shutdown` on `ctx.Done()`).
- [x] **Gate-integrity scan** (`origin/develop..HEAD`): no `//nolint`, `t.Skip`, build-tag exclusion,
      deleted assertion, or loosened gate. The two `_ =` swallows (`st.Close` defer; `WriteText`
      mid-write) are both justified (the latter with an inline comment per `next.md`).
- [x] **Oracle gate N/A (confirmed):** no signature / RFC-6962 / Merkle / did:web / proof / tile
      logic line changed (grep over the code diff finds only a doc-comment "proof/verify" reference).
      The existing conformance tests (equivocation/fork/shrink merkle, `derive_vkey.py` vectors) still
      pass inside the green `mise run check`.

**Issues found:** (none)

**Next:** The `fsck` root-rebuild conformance slice (M2 Verify) — the natural point to also resolve
the two open `normal` issues (the `go mod tidy` go.sum divergence and wiring CI/`notecheck`), since
that slice is the first to face the trust-root oracle in CI. After that, the M3 follow-up can add `/`,
`/healthz`, and the REST surface to the same mux (explicitly out of scope here).

**Notes:**
- **Scope is clean.** Exactly the 3 planned code files (`internal/metricshttp/handler.go` new,
  `internal/config/config.go`, `cmd/iscc-monitor/main.go`) plus their tests. Nothing from `## Not In
  Scope` leaked: metrics leaf untouched/WASM-pure, no extra endpoint (404 on `/healthz` proves it), no
  metric-name/label/`glossaryStatus` change, no CI/tidy resolution, no alert-transport change.
- **`serveMetrics` is intentionally untested directly** (goroutine + listener plumbing, no branching
  beyond the `ErrServerClosed` discriminator) — same accepted posture as `Loop.Run`. I exercised it
  via the binary smoke test instead, which is the only seam that reaches it. A bad `Addr` is logged
  (ERROR) from `ListenAndServe`, never fatal — the "serving never blocks polling" rule holds.
- **Two `normal` issues stay open** (go.sum tidy divergence; no CI/`notecheck`), correctly deferred to
  the `fsck`-rebuild slice per `next.md`.
- **Pushed to `origin/develop`** on PASS (see push result below).
