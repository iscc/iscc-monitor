# Next Work Package

## Step: `internal/config` — pure config loader leaf for the monitor binary

## Goal
Add the pure, golden-testable configuration package that the (next-step) `cmd/iscc-monitor`
binary will consume: turn a flat key/value lookup plus the on-disk realm-document path into a
validated, typed `Config` (DB path, realm-doc path, poll intervals). This is the last pure leaf the
binary wiring needs before the loop can be assembled, and it is the cleanest no-crypto, no-network
slice toward "the monitor actually runs and survives restart" (M1).

## Goal-fit (state → target gap)
M1's first Verify half (`origin`/`verifierKey`/single-poll) plus two of three triggers (shrink+fork
freeze) are met end-to-end, the poll loop drives `PollHub` on a cadence, and the pure realm-registry
parser landed. The named M1 list opens with "**config** + realm registry + …"; the registry exists,
config does not (`internal/config/` is absent). Both review and state name the `cmd/iscc-monitor`
binary as the unblocked slice, but that binary needs validated startup values (DB path, realm-doc
path, poll intervals) *and* it forces an as-yet-unmade decision about how it obtains each hub's
`origin` for `store.UpsertHub`. Splitting that decision and the untestable blocking `Run` loop off
into the binary step, and landing the **pure, golden-testable config leaf first**, follows the loop's
"pure functions before I/O, runnable+testable before infrastructure" ordering. The heavier
merkle-equivocation trigger (new dep + tile fixtures + oracle gate) and the `hub_keys`/
`derive_vkey.py` refresh stay deferred to their own steps.

## Scope
- **Create**:
  - `/workspace/iscc-monitor/internal/config/config.go` — the package (pure: parse + validate; no
    `os`/`net`/`flag` in the load path. Take the raw lookup as an injected
    `func(key string) (string, bool)` so it is unit-testable without touching the environment).
  - `/workspace/iscc-monitor/internal/config/config_test.go` — table-driven + golden tests (test
    file, not counted toward the 3-file limit).
- **Modify**: (none — this step adds a leaf package only)
- **Reference**:
  - `/workspace/iscc-monitor/internal/registry/registry.go` — the sibling pure-leaf style to mirror
    (leading package docstring, single pure entry point, fail-closed wrapped errors, minimal
    stdlib-only imports).
  - `/workspace/iscc-monitor/internal/follower/loop.go` lines 45–53 — the `Loop` fields config feeds
    (`Normal`, `Frozen time.Duration`; the docstring documents `Frozen >= Normal` as the back-off).
  - `/workspace/iscc-monitor/internal/store/sqlite.go` lines 59–76 — `Open(path string)` is the
    DB-path consumer (config supplies that path).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` lines 56–77 — `UpsertHub(ctx, domain,
    origin, baseURL)` needs `origin` (`<domain>/log`), which `logclient.origin` keeps private; see
    Not In Scope (this is a binary-wiring decision, not config's).
  - `/workspace/iscc-monitor/.claude/adr/0007-*.md` (one DB file per network) and `0005`/`0006` — the
    rules behind "one DB path" and the freeze back-off the `Frozen` interval encodes.

## Not In Scope
- **Do NOT create `cmd/iscc-monitor/main.go` or wire `Loop.Run` this step.** The binary is the next
  slice: it has an untestable blocking `Run` loop and forces the origin-export decision below, so
  keeping it separate keeps this step a clean, fully-testable leaf.
- **Do NOT export, duplicate, or re-derive `logclient.origin`.** `UpsertHub` needs `<domain>/log`,
  but how the binary obtains origin without a second deriver (export `logclient.Origin` vs. carry it
  on `registry.Entry`) is a *binary-wiring* decision for the next step. Config must not pre-empt it
  or grow an `origin()` of its own (learnings: "One `origin()` helper, golden-tested"; highest-
  probability bug).
- Do NOT read the realm document or open the DB here — config only *parses/validates* values
  (the realm-doc path stays a string). The binary does the actual `os.ReadFile` / `registry.Parse` /
  `store.Open`. Keep `os`/`net`/`net/http` out of the load path.
- Do NOT add a YAML/JSON/TOML dependency — `go.mod`/`go.sum` must stay byte-identical (KISS: a flat
  key/value `Load` is enough; the realm registry itself is already a flat text format).
- No coverage tracking (`monitored_since`), structured logs, `/metrics`, or alert transport — each is
  its own later M1 step.

## Implementation Notes
- Mirror `internal/registry` exactly for style: a leading package docstring stating purpose + the
  config keys + their defaults, a small typed result struct, one pure entry point, fail-closed
  wrapped errors naming the bad key, and a minimal stdlib-only import set (`fmt`, `strings`, `time`).
- Suggested surface (adjust names to taste; keep it minimal):
  - `type Config struct { DBPath string; RealmPath string; Normal, Frozen time.Duration }`.
  - `func Load(get func(key string) (string, bool)) (Config, error)` — pure: pull each key via
    `get`, apply documented defaults for the intervals, parse durations with `time.ParseDuration`,
    validate, return. Injecting `get` (instead of reading `os.Getenv`/`flag` here) is what keeps it
    I/O-free and unit-testable; the binary passes an `os.LookupEnv`-backed closure next step.
- Validation rules to encode (each is a golden/table assertion):
  - `DBPath` and `RealmPath` required (absent/empty → wrapped error naming the missing key).
  - `Normal > 0`; `Frozen > 0`; **`Frozen >= Normal`** (the `loop.go` back-off invariant — a
    `Frozen < Normal` config is rejected, not silently accepted). This is the load-bearing cross-check
    that ties config to ADR-0006's backed-off evidence-only cadence.
  - An unparseable duration (`time.ParseDuration` error) is wrapped + named, not swallowed.
  - Sensible defaults when an interval key is absent (e.g. `Normal=5m`, `Frozen=1h`) so a minimal
    config with only the two paths loads cleanly; document the defaults in the docstring.
- Relevant learnings / correctness rules:
  - **One DB file per network (ADR-0007)** — `DBPath` is a single network's DB file; config carries
    one path, not a list (multi-network is out of scope here).
  - **Freeze back-off (ADR-0006)** — the `Frozen` interval is the evidence-only re-poll cadence;
    enforcing `Frozen >= Normal` here is *why* the loop's `due()` actually backs a frozen hub off.
  - Keep errors fail-closed and wrapped (`fmt.Errorf("config: ... %q: %w", key, err)`), mirroring
    `registry.Parse`, and return the zero `Config` alongside any error so the binary surfaces a
    precise startup failure rather than a half-built config.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass).
- `gofmt -l internal/config` prints nothing.
- `go test -count=1 ./internal/config` passes (uncached).
- `go test -count=1 -run TestLoad ./internal/config` passes.
- `go list -deps ./internal/config | grep -E '^(net|net/http)$'` prints nothing (pure leaf; no
  network in the closure).
- `git status --short go.mod go.sum` is empty (no dependency added).
- Table/golden assertions prove: a minimal input `{DBPath, RealmPath}` loads with the documented
  default intervals; an input with `Frozen < Normal` returns a non-nil wrapped error; a missing
  required path returns a non-nil error naming the key; an unparseable interval returns a non-nil
  error; a valid full input round-trips every field.

## Done When
`internal/config` exists as a pure, dependency-free leaf whose golden/table tests pass under
`mise run check`, with the `Frozen >= Normal` and required-path validations covered, no new
dependency, and no `cmd/` binary, `logclient.origin` export, or real I/O introduced.
