# Handoff

## 2026-06-21 — Review of: Structured logging (`slog`) at the loop + binary boundary

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance lands `log/slog` as the monitor's structured-logging backbone at the
composition layer only: the previously `_ =`-discarded per-tick error in `loop.go` and the ad-hoc
`fmt.Fprintf(os.Stderr, …)` freeze alert in `main.go` are now structured records with stable keys
(`hub_id`, `kind`, `err`). A nil-safe `Loop.Logger` + `logger()` accessor keeps every bare `&Loop{…}`
compiling unchanged, the log-and-continue invariant is preserved (the tick error is logged but never
propagated from `Run`), and the change is stdlib-only — `go.mod`/`go.sum` and all six leaf packages are
byte-identical. Scope is clean (2 production files + 1 new test) and `mise run check` is green.

**Verification:**
- [x] `mise run check` (build + vet + test, all 8 packages) — green.
- [x] `gofmt -l .` — empty.
- [x] `go test -count=1 ./internal/follower` — PASS (all existing follower/loop tests + the 2 new ones).
- [x] `TestLoopLogsTickError` + `TestLoopLoggerNilSafe` — PASS (verbose-confirmed). Reviewer probed the
      record stream independently: a faulting single-hub `Tick` emits **exactly one** ERROR record
      (`msg="poll hub failed"`, `hub_id=1`, `err="follower.PollHub: hub 1: fetch checkpoint …"`), so the
      `len(errorRecs) != 1` assertion is non-vacuous and the error is genuinely observable.
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` exit 0 (no dependency change; stdlib-only).
- [x] `git diff --quiet HEAD~1..HEAD -- internal/{store,logclient,didweb,tiles,config,registry}` exit 0
      (leaf packages byte-unchanged; logging stayed at the composition layer).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — succeeds (WASM purity preserved).
- [x] No `log/slog` in any leaf package's import set; `log/slog` appears only in `loop.go` + `main.go`.
- [x] Scope/contract: `PollHub` and `AlertFunc func(int64,string)` signatures unchanged; old
      `func alert(…)` (stderr Fprintf) fully removed; nothing out-of-scope added (no `/metrics`/expvar/
      HTTP server/email-webhook/`tessera/fsck` — the only "webhook" match is a doc comment).
- [x] Gate-integrity scan over the 3 unpushed commits (`origin/develop..HEAD`): only `076d7cb` touches
      `.go`; no `//nolint`, `t.Skip`, build-tag exclusion, swallowed-error dodge, or deleted assertion/test.
- Oracle/conformance gate correctly **N/A**: no signature/RFC-6962/Merkle/did:web/proof/tile path touched
  (`log/slog` is stdlib at the loop/binary layer only). No CI/`notecheck` exists yet (open issue) — gates
  verified locally; `cauldron/` binaries not run (module-less, as required).

**Issues found:** (none new). The two pre-existing `normal` issues remain open and accurate (both
untouched by this slice, which adds no dependency and no CI):
- `go mod tidy` still adds 22 unstaged go.sum lines (committed go.sum byte-identical to HEAD,
  reproducible under `-mod=readonly`) — would fail a future tidy-cleanliness CI gate.
- No CI / `notecheck` signature-parity oracle wired (`.github/workflows/` absent).

**Next:** Two small independent M1 slices remain before M1 is fully met: `/metrics` (expvar/HTTP) — now
the natural follow-on, with the injected `Logger` and the `firstErr`/`PollHub` fault points as the metric
increment sites — and then the bigger `fsck` root-rebuild conformance slice (M2 Verify): `fsck.New(…).
Check(…)` over the `SQLiteFetcher` + the inclusion cross-check vs the hub's `IsccLogInclusionProof`,
needing the heavy `fsck`/otel/klog dep (copy/scope as the SQLiteFetcher slice did) and real on-disk tile
fixtures under `testdata/live/`. That conformance slice is the natural place to also resolve the `go mod
tidy` go.sum divergence and wire CI/`notecheck` (the trust-root oracle).

**Notes:**
- **Test seam is honest.** The fault is driven through the real outbound-fetch boundary (an always-erroring
  `Fetcher` → `FetchCheckpoint` → `PollHub` → `Tick`), asserting on the captured JSON records (level + keys),
  not formatted text or follower internals. The second log site (`"follow state read failed"`, `loop.go:109`)
  is reachable only via a store fault, but `Loop.Store` is a concrete `*store.Store` (not an interface), so
  it can't be cleanly fault-injected without a wider seam change — out of scope, documented, NOT dead code.
- **`Run` stays untested by the existing ticker-blocking convention** — the log-and-continue path is proven
  via `Tick`; `Run`'s log call is the same `logger().ErrorContext`, just on the aggregate `firstErr`. The
  `_ = l.Tick(ctx, t)` discard is genuinely gone (verified), replaced by a non-propagating logging branch.
- **Both `Tick` (per-hub, with `hub_id`) and `Run` (aggregate) log** — a deliberate choice over `next.md`'s
  optional `Tick` logging; the layers are distinct, no double-counting, a future `/metrics` slice picks one.
- **Alert severity is WARN** (documented in `alertFunc`'s doc comment): a freeze is an evidence-preserved,
  operator-actionable hub condition, distinct from a monitor-process fault (ERROR). Once-per-transition
  gating untouched (lives in `PollHub`/`freeze`); the `AlertFunc` signature is unchanged.
- **`develop` is the working branch with `origin/develop` upstream.** Pushing on PASS.
