# Next Work Package

## Step: Thread `*metrics.Registry` into `PollHub`/`Tick` and fire the increment sites

## Goal
Wire the pure `internal/metrics` leaf into the follower so the four metric series
actually move on the live verdict path — mapping the follower verdict to the
**glossary** hub-status set, not `Status.String()`. This is the first half of M1's
remaining `/metrics` slice (the increment call sites); the HTTP handler + `main.go`
server wiring is the deliberate next step.

## Scope
- **Modify**: `internal/follower/follower.go` (add a `*metrics.Registry` param to
  `PollHub`; fire `SetHubStatus`/`SetLastObservedAt` on every verdict, `IncViolation`
  in the freeze path; add the verdict→glossary status mapper)
- **Modify**: `internal/follower/loop.go` (add a nil-safe `Loop.Metrics *metrics.Registry`
  field; pass it to `PollHub`; fire `IncPollFailure` on `PollHub`'s error return in `Tick`)
- **Tests (not counted against the 3-file budget)**: update the existing `PollHub(...)` call
  sites in `internal/follower/*_test.go` for the new param; add assertions on the registry's
  rendered output (`Registry.String()`) for the verified / fork / unverified cases; add a
  `glossaryStatus` unit test. `cmd/iscc-monitor/main.go` is **NOT** touched this slice (it
  constructs `&follower.Loop{…}` with named fields, so a new optional field compiles unchanged).
- **Reference**:
  - `/workspace/iscc-monitor/internal/metrics/metrics.go` — the mutator API
    (`IncViolation(hubID, kind)`, `IncPollFailure(hubID)`, `SetHubStatus(hubID, status)`,
    `SetLastObservedAt(hubID, unixSeconds)`) and the glossary `status`-label contract.
  - `/workspace/iscc-monitor/internal/logclient/accept.go` — the `Status` enum
    (`StatusVerified|StatusUnverified|StatusUnresolvable|StatusRotated`) the mapper consumes.
  - `/workspace/iscc-monitor/internal/follower/follower.go` — `PollHub`/`checkConsistency`/`freeze`
    (the increment call sites).
  - `/workspace/iscc-monitor/internal/follower/loop.go` — `Loop`/`Tick` (the `IncPollFailure` site
    and the field threading; mirror the nil-safe `Logger`/`logger()` pattern).

## Not In Scope
- The `/metrics` **HTTP handler** (`net/http` serving `Registry.WriteText` with
  `Content-Type: text/plain; version=0.0.4`) and the `main.go` server + `Loop.Metrics`
  construction — that is the immediate **next** slice. This step only fires the increment sites.
- Real alert transport (email/webhook) — `alertFunc` stays the WARN slog placeholder.
- A `lag_seconds` series (it needs a `now` the metrics leaf must not read — already deferred, YAGNI).
- Persisting `hub_status`/`last_observed_at` to SQLite — the registry is in-memory only.
- Changing the `metrics` leaf itself (no new metric families, no API changes).

## Implementation Notes
- **Glossary mapping is the load-bearing correctness rule (carried forward from review/learnings).**
  The `status` label MUST be the glossary set `verified|unresolvable|unverified|frozen|inactive`,
  **not** `logclient.Status.String()` (which returns `verified|unverified|unresolvable|`**`rotated`**).
  Add a small pure mapper in `follower.go`, e.g. `glossaryStatus(st logclient.Status, frozen bool) string`:
  - `frozen == true` → `"frozen"` (takes precedence — a violation froze the hub even though the
    signature was `StatusVerified`).
  - else `StatusVerified` → `"verified"`, `StatusUnverified` → `"unverified"`,
    `StatusUnresolvable` → `"unresolvable"`, `StatusRotated` → `"unverified"`
    (out-of-window key = internally-broken / non-accepted; the glossary has no `rotated`, and a
    rotated checkpoint does not advance accepted state, so it folds into `unverified`).
  - `"inactive"` is the realm-registry removed/paused state — **never produced by `PollHub`** (the
    follower only polls active hubs); do not emit it here.
- **Make the registry optional and nil-safe** so existing `&follower.Loop{…}` literals and the binary
  keep compiling: add `Loop.Metrics *metrics.Registry`, and in `PollHub` accept an `m *metrics.Registry`
  param; guard every mutator call with `if m != nil { … }` (mirror the nil-safe `Logger`/`logger()`
  pattern already in `loop.go`). A nil registry = metrics disabled, no panic.
- **Increment-site placement (match the existing verdict structure in `PollHub`):**
  - `IncPollFailure(hubID)` fires in **`Tick`** on `PollHub`'s non-nil error return (the transport /
    garbled-body fault path), alongside the existing `ErrorContext` log — NOT inside `PollHub`, because
    a fetch/accept fault returns early before the verdict is known. Keep it on the same error branch.
  - `SetHubStatus(hubID, glossaryStatus(status, frozen))` + `SetLastObservedAt(hubID, observedAt.Unix())`
    fire on the **verdict path** in `PollHub` for the non-error outcomes: the early non-verified return
    (`status != StatusVerified`), the freeze return, and the verified-advance tail. Use `observedAt.Unix()`
    for the timestamp (the leaf is clock-free; the caller supplies the value — already injected, never
    `time.Now()`).
  - `IncViolation(hubID, string(kind))` fires in the **freeze** path (in or alongside `freeze`), keyed on
    the same `kind` string already passed to `RecordViolation`/`alert`. It re-fires on every re-detection
    (the counter is cumulative — that is correct; `alert` stays once-per-transition, `IncViolation` does not).
- **Status-on-freeze ordering:** the freeze branch in `PollHub` returns `freeze(...)`; that helper still
  returns `(StatusVerified, nil)`. Set `hub_status = glossaryStatus(status, /*frozen=*/true)` (→ `"frozen"`)
  and `last_observed_at` on that branch — do not rely on the `Status` enum alone, since the enum is still
  `StatusVerified` after a freeze. Pure registry writes, so placement before/after the `freeze` call is fine.
- **Imports:** `internal/follower` already imports `logclient`/`store`/`time`; add
  `github.com/iscc/iscc-monitor/internal/metrics`. The dependency direction stays
  `follower → {logclient, store, metrics}` (metrics is a pure leaf), so `net/http`/`database/sql` never
  enter the store/metrics closures. No `go.mod`/`go.sum` change (metrics is already an in-module package).
- **Correctness rules honored:** ADR-0006 freeze-not-crash is unchanged (metrics writes never alter
  control flow); the single-writer discipline holds (only `Tick`/`PollHub` in the one owning goroutine
  write the registry; the future HTTP reader takes the registry's own RWMutex). `kind` reuses the real
  `violations.kind` strings verbatim (safe per learnings); only `status` needs the glossary remap.
  Oracle/conformance gate is **N/A** for this slice (no signature/RFC-6962/Merkle/did:web/proof/tile path
  changes — only call-site wiring of an existing pure leaf); `go.mod`/`go.sum` stay byte-identical.
- **Test seam:** assert on the **observable rendered output** (`Registry.String()` containing the
  expected lines), never on follower internals (PRD seam-based testing). Construct a real `metrics.New()`
  in the tests and pass it through `PollHub`/the `Loop`.

## Verification
- `mise run check` is green (`go build ./...` && `go vet ./...` && `go test ./...`), `gofmt -l .` empty.
- `go test -run TestPollHub ./internal/follower` passes (all existing `PollHub` tests, updated for the
  new param, still green).
- `go test -run TestGlossaryStatus ./internal/follower` passes: asserts
  `glossaryStatus(StatusVerified,false)=="verified"`, `glossaryStatus(StatusVerified,true)=="frozen"`,
  `glossaryStatus(StatusUnverified,false)=="unverified"`,
  `glossaryStatus(StatusUnresolvable,false)=="unresolvable"`, `glossaryStatus(StatusRotated,false)=="unverified"`.
- A follower test drives a verified observation through `PollHub` with a real `metrics.New()` and asserts
  `Registry.String()` contains `iscc_monitor_hub_status{hub_id="<id>",status="verified"} 1` and a non-zero
  `iscc_monitor_last_observed_at{hub_id="<id>"}`.
- A fork/freeze follower test asserts `Registry.String()` contains
  `iscc_monitor_violations_total{hub_id="<id>",kind="fork"} 1` and `…hub_status{…,status="frozen"} 1`.
- A `Tick`-level test driving a fetch fault asserts `Registry.String()` contains a non-zero
  `iscc_monitor_poll_failures_total{hub_id="<id>"}`.
- `go list -deps ./internal/store | grep '^net/http$'` stays empty (store remains a leaf; the new
  follower→metrics edge does not leak into store).
- `git diff --quiet HEAD -- go.mod go.sum` exits 0 (no dependency change — `metrics` is already in-module).

## Done When
`mise run check` is green and the follower fires all four metric series on the live verdict path with the
glossary-status mapping (rotated→unverified, freeze→frozen) proven by `Registry.String()` assertions and
the `glossaryStatus` unit test — with the registry threaded nil-safely so the binary still compiles unchanged.
