# Handoff

## 2026-06-20 — internal/config: pure config loader leaf for the monitor binary

**Done:** Added `internal/config` — a pure, dependency-free `Load(get func(key string) (string, bool))
(Config, error)` that turns a flat injected key/value lookup into a validated, typed
`Config{DBPath, RealmPath string; Normal, Frozen time.Duration}`. It applies documented interval
defaults (Normal=5m, Frozen=1h), parses durations with `time.ParseDuration`, and fails closed
(wrapped, key-naming errors + zero `Config`) on a missing required path, an unparseable or
non-positive interval, and the load-bearing `Frozen >= Normal` back-off cross-check (ADR-0006).

**Files changed:**
- `internal/config/config.go`: the package — leading docstring documenting the four keys
  (`ISCC_MONITOR_DB`/`REALM`/`NORMAL`/`FROZEN`) and their defaults, the `Config` struct, the pure
  `Load` entry point, and two private helpers (`required`, `duration`). Imports are exactly
  `{fmt time}`; no `os`/`net`/`flag` in the load path (the `get` closure is injected).
- `internal/config/config_test.go`: table-driven + golden tests (not counted toward the 3-file limit).

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all `ok`; config +
the five existing packages). Per-criterion:
- [x] `gofmt -l internal/config` — prints nothing (ran `mise run fmt` first).
- [x] `go test -count=1 ./internal/config` — PASS (uncached, 0.003s).
- [x] `go test -count=1 -run TestLoad ./internal/config` — PASS.
- [x] `go list -deps ./internal/config | grep -E '^(net|net/http)$'` — empty; own `.Imports` =
  `[fmt time]` (pure leaf).
- [x] `git status --short go.mod go.sum` — empty (no dependency added).
- [x] Table/golden assertions: `TestLoadGolden` round-trips every field of a full input;
  `TestLoadDefaults` proves a minimal `{DB, realm}` loads with `Normal=5m`/`Frozen=1h` and asserts
  the defaults satisfy `Frozen >= Normal`; `TestLoad` covers single-override + `Frozen == Normal`
  accepted; `TestLoadErrors` covers missing/empty DB path, missing/empty realm path, unparseable
  Normal/Frozen, non-positive Normal/negative Frozen, and `Frozen < Normal` — each asserting a
  non-nil error that names the offending key AND the zero `Config` returned alongside it.

**Next:** `cmd/iscc-monitor/main.go` is now the unblocked slice and config's only consumer: build the
`os.LookupEnv`-backed `get` closure → `config.Load` → `os.ReadFile(cfg.RealmPath)` →
`registry.Parse` → `store.Open(cfg.DBPath)` → per `Entry` `store.UpsertHub(domain, origin, baseURL)`
to obtain `HubID`s → `[]follower.HubTarget` → build `follower.Loop{Normal: cfg.Normal, Frozen:
cfg.Frozen, …}` → `Loop.Run(ctx)`. That step must make the still-open **origin-export decision**
flagged in this work package: `UpsertHub` needs `<domain>/log` and `logclient.origin` is private —
either export `logclient.Origin` or carry origin on `registry.Entry`; do NOT add a second deriver
(learnings: "One `origin()` helper, golden-tested"). The blocking `Run` loop stays untestable so it
is correctly the binary step, not this leaf.

**Notes:**
- Scope discipline honored: 1 production file + 1 test, new leaf package only; nothing else touched.
  None of `## Not In Scope` was done — no `cmd/` binary, no `Loop.Run` wiring, no `logclient.origin`
  export/duplication, no realm-doc read / DB open / `os`/`net` in the load path, no YAML/JSON/TOML
  dep (go.mod/go.sum byte-identical).
- Key names: chose `ISCC_MONITOR_{DB,REALM,NORMAL,FROZEN}` env-style strings. They are package
  constants the binary references symbolically; the binary owns the actual `os.LookupEnv` calls, so
  these are not yet load-bearing against any external contract and can be renamed at wiring time
  cheaply if a flag-based surface is preferred instead.
- One small validation addition beyond the literal `Normal > 0`/`Frozen > 0` spec wording: a
  *present but* non-positive interval (`0s`, `-1m`) is rejected with a named error, matching the
  intent ("`Normal > 0`; `Frozen > 0`"). Absent keys still take the positive defaults.
- Oracle/conformance gate correctly N/A: the diff touches no proof/verify/didweb/merkle/consistency/
  signature/fsck/notecheck path — `internal/config` is a pure no-crypto, no-network value parser.
- Branch is `develop` (never `main`).
