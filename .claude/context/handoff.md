# Handoff

## 2026-06-21 — Thread `*metrics.Registry` into `PollHub`/`Tick` and fire the increment sites

**Done:** Wired the pure `internal/metrics` leaf into the follower so the four metric series move on
the live verdict path. Added a `m *metrics.Registry` param to `PollHub` (fires `SetHubStatus`/
`SetLastObservedAt` on every non-error verdict and `IncViolation` on the freeze branch, all nil-safe)
and a nil-safe `Loop.Metrics *metrics.Registry` field threaded through `Tick`, which fires
`IncPollFailure` on `PollHub`'s error return. The verdict→status label uses a new pure
`glossaryStatus` mapper (glossary set, NOT `Status.String()`): freeze→`frozen`, rotated→`unverified`.

**Files changed:**
- `internal/follower/follower.go`: added `metrics` import; pure `glossaryStatus(st, frozen) string`
  mapper; `m *metrics.Registry` param on `PollHub`; nil-safe `recordVerdict` helper (the single
  `hub_status`/`last_observed_at` mutation point, funneling the three verdict branches: early
  non-verified, freeze, verified-advance); `IncViolation(hubID, string(kind))` on the freeze branch.
- `internal/follower/loop.go`: added `metrics` import; nil-safe `Loop.Metrics *metrics.Registry`
  field; pass `l.Metrics` to `PollHub`; `IncPollFailure(target.HubID)` on `PollHub`'s error branch in
  `Tick` (same branch as the existing `ErrorContext` log), nil-safe.
- `internal/follower/follower_test.go` (test): updated all 10 `PollHub(...)` call sites for the new
  param; added `assertMetric` helper (asserts on `Registry.String()` rendered output); added metric
  assertions to the fork (`violations_total{kind="fork"} 1` + `hub_status{status="frozen"} 1`),
  verified (`hub_status{status="verified"} 1` + non-zero `last_observed_at`), and unverified
  (`hub_status{status="unverified"} 1`) cases; added `TestGlossaryStatus` golden table.
- `internal/follower/loop_test.go` (test): added `metrics` import; added `TestTickMetricsPollFailure`
  driving an `errFetcher` fault and asserting `poll_failures_total{hub_id="1"} 1`.

**Verification:** `mise run check` → green (build + vet + test, all 9 packages; uncached
`go test -count=1` on touched packages + `cmd` also green). `gofmt -l .` empty.
- [x] `go test -run TestPollHub ./internal/follower` — PASS (all existing PollHub tests, updated for
      the new param).
- [x] `go test -run TestGlossaryStatus ./internal/follower` — PASS: `verified`/`frozen`/`unverified`/
      `unresolvable`, and `StatusRotated,false → "unverified"`.
- [x] Verified-observation PollHub test asserts `hub_status{hub_id="1",status="verified"} 1` and
      `last_observed_at{hub_id="1"} 1781913600` (= `observedAt.Unix()`, non-zero).
- [x] Fork/freeze PollHub test asserts `violations_total{hub_id="1",kind="fork"} 1` and
      `hub_status{hub_id="1",status="frozen"} 1`.
- [x] `TestTickMetricsPollFailure` asserts a non-zero `poll_failures_total{hub_id="1"} 1`.
- [x] `go list -deps ./internal/store | grep '^net/http$'` — empty (store stays a leaf; the new
      follower→metrics edge does not leak into store).
- [x] `git diff --quiet HEAD -- go.mod go.sum` exit 0 (metrics is already in-module; no dep change).
- [x] `cmd/iscc-monitor/main.go` untouched and still compiles (the `&follower.Loop{…}` named-field
      literal compiles unchanged with the new optional `Metrics` field).

**Next:** The `/metrics` HTTP slice — a `net/http` handler calling `Registry.WriteText` with
`Content-Type: text/plain; version=0.0.4`, plus constructing `metrics.New()` in `cmd/iscc-monitor/
main.go` and passing it as `Loop.Metrics` + serving the handler. That is the deliberate second half of
M1's `/metrics` slice (this slice only fired the increment sites). After that, the `fsck` root-rebuild
conformance slice (M2 Verify), which is also where the open go.sum tidy divergence + CI/`notecheck`
should be resolved.

**Notes:**
- **Glossary mapping is honored and mutation-proven.** `status` uses the glossary set
  `verified|unresolvable|unverified|frozen`, never `Status.String()`'s `rotated`. Two reverted
  mutations confirm the assertions are non-vacuous: (1) dropping the `frozen=true` flag on the freeze
  branch renders `status="verified"` → `TestPollHubFork` FAILS; (2) removing the `IncViolation` call →
  `TestPollHubFork` FAILS on the missing `violations_total`. Both reverted; tree restored.
- **Param placement:** `m *metrics.Registry` is the last param of `PollHub` (after `alert`), and
  `Loop.Metrics` mirrors the nil-safe `Logger`/`logger()` pattern. Every mutator call is guarded
  `if m != nil` / `if l.Metrics != nil` (or inside the nil-safe `recordVerdict`), so a nil registry =
  metrics disabled, no panic, and metrics writes never alter control flow (ADR-0006).
- **Increment-site placement matches `next.md`:** `IncPollFailure` fires in `Tick` (not `PollHub`,
  since a fetch fault returns early before the verdict); `SetHubStatus`/`SetLastObservedAt` fire on
  all three non-error verdict branches via `recordVerdict`; `IncViolation` fires on the freeze branch
  and re-fires on re-detection (cumulative, correct — `alert` stays once-per-transition, the counter
  does not). `last_observed_at` uses `observedAt.Unix()` (the leaf is clock-free; caller supplies it).
- **Scope:** exactly 2 implementation files (`follower.go`, `loop.go`) + 2 test files; `main.go`/`cmd/`
  untouched as scoped. `go.mod`/`go.sum` byte-identical. Dependency direction stays
  `follower → {logclient, store, metrics}` (metrics is a pure leaf; no `net/http`/`database/sql`
  enters the store/metrics closures).
- **Oracle/conformance gate correctly N/A** for this slice — only call-site wiring of an existing pure
  leaf; no signature/RFC-6962/Merkle/did:web/proof/tile path changed. No go.mod/go.sum/schema change.
- Both pre-existing `normal` issues (go.sum tidy divergence; no CI/`notecheck` oracle) remain open and
  untouched — correct, since this slice added no dep and wired no CI.
