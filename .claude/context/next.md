# Next Work Package

## Step: Pure `internal/metrics` leaf — counters/gauges + Prometheus text rendering

## Goal
Land the metrics registry as a pure, stdlib-only leaf that holds the monitor's
alert-worthy series and renders them in Prometheus text-exposition format. This is
the first slice of M1's remaining `/metrics` gap; the HTTP `/metrics` handler and the
follower wiring are deliberately separate later slices (mirroring this codebase's
established "pure leaf first, wire next" pattern — consistency triggers, store CRUD,
and the config loader all landed this way).

## Scope
- **Create**: `internal/metrics/metrics.go` (the pure registry + Prometheus text renderer)
- **Create**: `internal/metrics/metrics_test.go` (table-driven golden tests on rendered output)
- **Modify**: (none — no production wiring this slice)
- **Reference**:
  - `/workspace/iscc-monitor/.claude/plans/cosmic-baking-octopus.md` lines 87–94 — the
    authoritative metric series list: per-hub `status`, `violations_total{kind}`,
    `unresolvable`, poll/availability failures, `lag_seconds`, `last_observed_at`.
  - `/workspace/iscc-monitor/.claude/prd/0001-iscc-monitor-v1.md` lines 84, 222–232 —
    `/metrics` is an asserted *observable output*; Prometheus is the primary alert path.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — read the `RecordViolation`/
    `Freeze`/`FollowState`/`Coverage` shapes so the metric *names/labels* you choose line up
    with what a later wiring slice will feed them (e.g. `kind` matches `violations.kind`,
    `hub_id` is the natural per-hub label). Do NOT call into store here.
  - `/workspace/iscc-monitor/internal/follower/follower.go` — to confirm the eventual
    increment sites (violation, unverified/unresolvable, poll fault) so the leaf's public
    methods are shaped to those call sites, without importing follower.

## Not In Scope
- **No HTTP server / `http.Handler` / `/metrics` route** — no `net/http`, no `ListenAndServe`.
  The handler is the next slice. Keeping this leaf net-free keeps it trivially unit-testable
  and avoids pulling `net/http` into a fresh package.
- **No follower/loop/main wiring** — do not touch `follower.go`, `loop.go`, or `main.go`.
  Threading a metrics handle through `PollHub`/`Tick` is a separate slice (it touches ≤3
  production files on its own and deserves its own step).
- **No `prometheus/client_golang` dependency** — render the text format from stdlib only.
  Adding that heavy dep would widen the already-flagged `go mod tidy` go.sum divergence (open
  `normal` issue) for no benefit at this leaf stage; a later slice may swap the renderer if a
  real need appears (YAGNI). `go.mod`/`go.sum` MUST stay byte-identical.
- **No webhook / email transport** — the generic outbound webhook (PRD §21) is a separate later
  step; this slice is metrics only.
- **No `time.Now()` inside the leaf** — if you model `lag_seconds`/`last_observed_at`, take the
  unix-seconds/observed values as method arguments (injected), never read the clock in the leaf,
  so tests stay deterministic (same discipline as `due()`/`AcceptCheckpoint`'s injected `observedAt`).

## Implementation Notes
- Model a small `Registry` struct holding the series. Guard the maps with a `sync.Mutex`/
  `sync.RWMutex`: even though the follower is single-writer per DB, a later `/metrics` HTTP read
  path renders concurrently with follower writes, so the leaf must be safe under a read-while-write
  race without depending on the caller being single-threaded. A one-line comment noting the
  single-writer follower is fine, but correctness must not *rely* on it.
- Series to model (names are the contract a later wiring slice fills — pick stable
  Prometheus-idiomatic names: snake_case, `_total` suffix on counters, an `iscc_monitor_` prefix):
  - `iscc_monitor_violations_total{hub_id=…,kind=…}` — counter; label `kind` ∈
    `shrink|fork|equivocation` (exactly the `store.Violation.Kind` / `violations.kind` strings —
    reuse them, do not invent new label values).
  - `iscc_monitor_poll_failures_total{hub_id=…}` — counter (transport/garbled-body faults).
  - `iscc_monitor_hub_status{hub_id=…,status=…}` — gauge (1 for the current status). Statuses are
    the glossary set `verified|unresolvable|unverified|frozen|inactive`. (Optional this slice.)
  - `iscc_monitor_last_observed_at{hub_id=…}` — gauge, unix seconds (caller-supplied). (Optional.)
  - `iscc_monitor_lag_seconds{hub_id=…}` — gauge, caller-supplied (no clock read). (Optional.)
  - Choose the minimal coherent subset you can fully golden-test; you need not wire all five, but
    `violations_total{kind}` and `poll_failures_total` MUST be present and golden-tested — they map
    to the existing freeze + poll-fault sites the wiring slice will use.
- Public API shape: counter mutators like `IncViolation(hubID int64, kind string)` and
  `IncPollFailure(hubID int64)`; gauge setters (if modeled) take the value as an arg. A render
  method `func (r *Registry) WriteText(w io.Writer) error` (or `String() string`) emits valid
  Prometheus text-exposition format: a `# HELP`/`# TYPE` header per metric family followed by
  sample lines (`name{label="v",…} value`). **Sort label sets and sample lines deterministically**
  (map iteration order is random) so the golden output is byte-stable.
- Keep imports stdlib-only: e.g. `fmt`, `io`, `sort`, `strconv`, `strings`, `sync`. No
  `net`/`net/http`/`os`/`database/sql`/`time`/`log/slog` in the leaf. Start the file with a
  docstring per repo convention; write concise evergreen docstrings on each exported symbol.
- **Correctness rule (learnings):** this is a pure leaf with no signature/RFC-6962/Merkle/did:web/
  proof/tile path, so the **oracle/conformance gate is correctly N/A** here — state that in the
  handoff. The trust-root gate re-arms only at the `fsck`-rebuild slice.

## Verification
- `mise run check` is green (`go build ./... && go vet ./... && go test ./...`).
- `gofmt -l internal/metrics` prints nothing (file is formatted).
- `go test -count=1 ./internal/metrics` passes.
- `go list -deps ./internal/metrics | grep -E '^(net/http|database/sql|log/slog)$'` is empty
  (leaf is net-free, db-free, and log-free).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 after the change (stdlib-only; no new dep).
- A golden test asserts the rendered text for a registry with at least one
  `violations_total{kind="fork"}` increment and one `poll_failures_total` increment byte-equals an
  expected Prometheus block (with `# TYPE` lines and deterministically sorted samples) — proving the
  renderer is stable and the `kind` label uses the real `violations.kind` string.

## Done When
`internal/metrics` exists as a pure, stdlib-only leaf whose render method emits a deterministic
Prometheus text block golden-tested over `violations_total{kind}` + `poll_failures_total`,
`mise run check` is green, `gofmt -l internal/metrics` is empty, the package closure is
net-/db-/log-free, and `go.mod`/`go.sum` are byte-identical to HEAD.
