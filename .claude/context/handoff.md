# Handoff

## 2026-06-21 — Review of: Pure `internal/metrics` leaf — counters/gauges + Prometheus text rendering

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` landed `internal/metrics` as a pure, stdlib-only Prometheus text-exposition
renderer exactly as `next.md` scoped — a `*Registry` (RWMutex-guarded) over four families (the two
required counters `violations_total{hub_id,kind}` / `poll_failures_total{hub_id}` plus the gauges
`hub_status{hub_id,status}` / `last_observed_at{hub_id}`), with no HTTP handler, no follower wiring,
and no new dependency. All gates are green, the golden test is mutation-proven non-vacuous, and the
leaf is net-/db-/log-/clock-free with `go.mod`/`go.sum` byte-identical to HEAD.

**Verification:**
- [x] `mise run check` (build + vet + test, all 9 packages) — green; full `go test -count=1 ./...`
      (uncached) also green.
- [x] `gofmt -l internal/metrics` empty; `gofmt -l .` (whole tree) empty.
- [x] `go test -count=1 ./internal/metrics` — PASS.
- [x] `go list -deps ./internal/metrics | grep -E '^(net/http|database/sql|log/slog)$'` — empty.
      Direct `.Imports` are exactly `[fmt io sort strconv strings sync]` (no `net`/`os`/`time`); the
      `os`/`time` in the transitive closure ride in via `fmt` only. `GOOS=js GOARCH=wasm go build
      ./internal/metrics` — OK.
- [x] `git diff --quiet HEAD -- go.mod go.sum` exit 0 (stdlib-only; no new dep). Also unchanged across
      the advance commit (`HEAD~1..HEAD`).
- [x] Golden test byte-equals an expected Prometheus block (`# HELP`/`# TYPE` lines, deterministically
      sorted samples) over `violations_total{kind="fork"}` + `poll_failures_total`. **Mutation-proven:**
      reverse-sort → `TestRenderGolden` fails on ordering; double-count `IncViolation` → three tests
      fail. A green-but-wrong renderer cannot ship. (Both mutations reverted; tree restored
      byte-identical.)
- [x] Scope discipline — diff is exactly `internal/metrics/metrics.go` + `metrics_test.go` (+ handoff);
      no `follower.go`/`loop.go`/`main.go`/`cmd/` touched, honoring `## Not In Scope`.
- [x] Quality-gate integrity — scanned all unpushed commits (`@{upstream}..HEAD`): no
      `//nolint`/`t.Skip`/build-tag/swallowed-error gate dodges. The two `_ =` blanks are legitimate
      (`String()`'s `strings.Builder.Write` never errors — documented inline; the test reader discards
      render output intentionally).
- [x] Oracle/conformance gate **correctly N/A** — pure formatter, no
      signature/RFC-6962/Merkle/did:web/proof/tile path. Confirmed the diff touches none of those paths;
      the trust-root gate re-arms at the `fsck`-rebuild slice (unchanged).

**Issues found:** (none) — both pre-existing `normal` issues (go.sum tidy divergence; no CI/`notecheck`
oracle) remain open and untouched, which is correct: this slice added no dep and wired no CI.

**Next:** The `/metrics` HTTP slice — a `net/http` handler calling `Registry.WriteText` with
`Content-Type: text/plain; version=0.0.4`, wired at the composition layer (`loop.go`/`main.go`), plus
threading a `*metrics.Registry` into `follower.PollHub`/`Tick` so the increment sites fire
(`IncViolation` at freeze, `IncPollFailure` on `PollHub`'s error return in `Tick`, `SetHubStatus` /
`SetLastObservedAt` on the verdict path). After that, the `fsck` root-rebuild conformance slice (M2
Verify), which is also where the open go.sum divergence + CI/`notecheck` should be resolved.

**Notes:**
- **Wiring-slice gotcha (carried forward, now also in learnings): the `status` label set is the
  GLOSSARY set, not `Status.String()`.** The leaf renders `status` verbatim without validating it; the
  glossary set is `verified|unresolvable|unverified|frozen|inactive` but `logclient.Status.String()`
  returns `verified|unverified|unresolvable|rotated` (`rotated` and `frozen` are different axes). The
  wiring slice MUST map follower verdict → glossary status before calling `SetHubStatus`. `kind` is
  safe (reuses the real `violations.kind` strings verbatim, golden-asserted).
- `lag_seconds` was deliberately NOT modeled (it needs a `now` the leaf must not read) — YAGNI
  respected; no series without an eventual call site was added.
- `CGO_ENABLED=0` ⇒ no `-race`, so `TestConcurrentMutateAndRender` proves safety by exact final counts
  (8 writers × 1000, no lost increment) rather than via the race detector — sound given the constraint.
- The rendered output is well-formed Prometheus text-exposition format (independently inspected): per-
  family `# HELP`/`# TYPE`, `name{labels} value` samples, and `escapeLabelValue` correctly
  backslash-escapes `\`/`"`/`\n` so the renderer stays total.
