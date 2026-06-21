# Handoff

## 2026-06-21 — Pure `internal/metrics` leaf — counters/gauges + Prometheus text rendering

**Done:** Landed `internal/metrics` as a pure, stdlib-only leaf: a `*Registry` (guarded by a
`sync.RWMutex`) holding the monitor's alert-worthy series and rendering them in Prometheus
text-exposition format via `WriteText(io.Writer)` / `String()`. Modeled four families — the two
required counters `iscc_monitor_violations_total{hub_id,kind}` and
`iscc_monitor_poll_failures_total{hub_id}`, plus the gauges `iscc_monitor_hub_status{hub_id,status}`
and `iscc_monitor_last_observed_at{hub_id}`. No HTTP handler, no follower wiring, no new dependency.

**Files changed:**
- `internal/metrics/metrics.go` (new): `Registry`/`New`, mutators `IncViolation(hubID, kind)`,
  `IncPollFailure(hubID)`, `SetHubStatus(hubID, status)` (sets the active status to 1 and clears the
  hub's other statuses to 0), `SetLastObservedAt(hubID, unixSeconds)` (caller-supplied, no clock
  read), and the `WriteText`/`String` renderer. Imports are stdlib-only: `fmt io sort strconv strings
  sync`. Per-family `# HELP`/`# TYPE` headers; sample lines sorted lexically so output is byte-stable.
- `internal/metrics/metrics_test.go` (new): full golden block (byte-equality), the minimal
  `violations_total{kind="fork"}` + `poll_failures_total` contract, a 50× determinism check, the
  single-active-status invariant, the empty-registry headers-only render, and a concurrent
  mutate-while-render test (8 writers × 1000 + 2 renderers) asserting exact final counts.

**Verification:** `mise run check` (build + vet + test, all 9 packages) → green. Per-criterion:
- [x] `gofmt -l internal/metrics` — empty.
- [x] `go test -count=1 ./internal/metrics` — PASS.
- [x] `go list -deps ./internal/metrics | grep -E '^(net/http|database/sql|log/slog)$'` — empty
      (leaf is net-free, db-free, log-free). Direct `.Imports` are exactly `fmt io sort strconv
      strings sync` (no `net`/`os`/`time`). The `time`/`os` that appear in the *transitive* closure
      ride in via `fmt` only (same nuance the learnings note for `internal/didweb`); load-bearing
      invariant holds and `GOOS=js GOARCH=wasm go build ./internal/metrics` succeeds.
- [x] `git diff --quiet HEAD -- go.mod go.sum` exit 0 (stdlib-only; no new dep).
- [x] Golden test byte-equals an expected Prometheus block with `# TYPE` lines and deterministically
      sorted samples over `violations_total{kind="fork"}` + `poll_failures_total`.
- Oracle/conformance gate correctly **N/A**: this is a pure formatting leaf with no
  signature/RFC-6962/Merkle/did:web/proof/tile path. The trust-root gate re-arms at the `fsck`-rebuild
  slice (unchanged from the prior several slices).

**Next:** The natural follow-on is the `/metrics` HTTP slice: a `net/http` handler that calls
`Registry.WriteText` with `Content-Type: text/plain; version=0.0.4`, wired at the composition layer
(`loop.go`/`main.go`), plus threading a `*metrics.Registry` handle into `follower.PollHub`/`Tick` so
the increment sites fire. The metric method shapes already match the call sites: `IncViolation(hubID,
string(kind))` at `freeze`, `IncPollFailure(hubID)` on `PollHub`'s error return in `Tick`, and
`SetHubStatus(hubID, status.String())` / `SetLastObservedAt(hubID, observedAt.Unix())` on the verdict
path. After that, the bigger `fsck` root-rebuild conformance slice (M2 Verify) remains, which is also
the place to resolve the open `go mod tidy` go.sum divergence and wire CI/`notecheck`.

**Notes:**
- **`hub_status` label values are the glossary set, not `Status.String()`.** The glossary hub-status
  set is `verified|unresolvable|unverified|frozen|inactive`, but `logclient.Status.String()` returns
  `verified|unverified|unresolvable|rotated`. The leaf does not validate the `status` string (it just
  renders what it's given), so the wiring slice must map the follower verdict to a glossary status
  (notably: a `rotated` verdict and a frozen hub are not directly the same axis). `hub_status` and
  `last_observed_at` were modeled beyond the required two because they golden-test cleanly and map to
  existing call sites — `next.md` left them optional; YAGNI is respected (no series with no eventual
  call site was added; `lag_seconds` was deliberately *not* modeled this slice — it needs a `now`
  the leaf must not read, so it belongs at the wiring layer that already injects `observedAt`).
- **`kind` label reuses the real strings.** `IncViolation`'s `kind` is the
  `store.Violation.Kind`/`logclient.ViolationKind` value (`shrink`/`fork`/`equivocation`) verbatim —
  the golden asserts `kind="fork"`/`"shrink"`/`"equivocation"`, no synonyms invented at the seam.
- **`SetHubStatus` keeps exactly one active status per hub** by clearing the hub's other status
  samples to 0 (so superseded statuses render `… 0`, not stale 1s); `TestSetHubStatusSingleActive`
  pins this. An alternative (drop the old sample entirely) was rejected so Prometheus sees a continuous
  0→1 transition rather than a vanishing series.
- **Concurrency is real, not theoretical.** The follower is single-writer per DB, but the future
  `/metrics` HTTP read renders concurrently with follower writes, so the `RWMutex` is load-bearing.
  Note `CGO_ENABLED=0` ⇒ no `-race`, so `TestConcurrentMutateAndRender` proves safety by exact final
  counts (no lost increment) rather than via the race detector.
- **`String()` swallows the `WriteText` error deliberately and safely** — it renders into a
  `strings.Builder`, whose `Write` never returns an error, so the `_ =` is correct (not a gate dodge);
  a one-line comment says so. `WriteText` itself propagates every `io.Writer` error.
- The two pre-existing `normal` issues remain open and untouched (no dependency added, no CI wired):
  the `go mod tidy` 22-line go.sum divergence and the absent `notecheck`/CI oracle.
