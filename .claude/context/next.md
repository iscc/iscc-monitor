# Next Work Package

## Step: `cmd/iscc-monitor` binary — wire config → registry → store → poll loop

## Goal
Stand up the `cmd/iscc-monitor` entrypoint that turns the four landed leaves
(`config.Load`, `registry.Parse`, `store.Open`/`UpsertHub`, `follower.Loop`) into a running monitor
process. This is config's only consumer and the slice that makes M1 a runnable program rather than a
library of disconnected pieces.

## Scope
- **Create**:
  - `cmd/iscc-monitor/main.go` — the binary entrypoint plus a testable `registerHubs` helper.
  - `cmd/iscc-monitor/main_test.go` — table/golden test for `registerHubs` (test file, not counted
    against the 3-file budget).
- **Modify**:
  - `internal/logclient/origin.go` — add an exported `Origin(baseURL string) (string, error)` that
    **delegates to the existing private `origin`** (a one-line wrapper). Do NOT rename `origin` or
    touch its body; the two internal callers (`checkpoint.go`, `didresolve.go`) keep calling the
    private `origin` unchanged. This resolves the open origin-export decision without a second deriver.
- **Reference** (read for context; never import the `cauldron/` trees):
  - `/workspace/iscc-monitor/internal/config/config.go` — `Load(get) (Config, error)`; fields
    `DBPath/RealmPath/Normal/Frozen`; key constants.
  - `/workspace/iscc-monitor/internal/registry/registry.go` — `Parse([]byte) ([]Entry, error)`,
    `Entry{Domain, BaseURL}`.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` lines 56–77 — `UpsertHub(ctx, domain,
    origin, baseURL) (int64, error)`.
  - `/workspace/iscc-monitor/internal/store/sqlite.go` lines 59–80 — `Open(path) (*Store, error)`,
    `Close()`.
  - `/workspace/iscc-monitor/internal/follower/loop.go` lines 31–53,121–135 — `Loop{Store, Fetcher,
    Targets, Normal, Frozen, Alert}`, `HubTarget{HubID, BaseURL}`, `Run(ctx) error`.
  - `/workspace/iscc-monitor/internal/logclient/didresolve.go` lines 48–56 — `NewHTTPFetcher(c
    *http.Client) Fetcher` (the production Fetcher for `Loop.Fetcher`).
  - `/workspace/iscc-monitor/internal/follower/follower.go` lines 41–46 — `AlertFunc` signature for the
    placeholder `Alert`.

## Not In Scope
- The merkle-backed **equivocation** trigger, `transparency-dev/merkle`, and tile fixtures — that is the
  separate heavy slice that trips the oracle gate; do not add the dep or fixtures here.
- The `hub_keys` did:web cache write and the stale sb1 fixture refresh (`22b08f3e`→`069d0f14`).
- Coverage (`monitored_since`), structured logging, `/metrics`, and real alert transport — the `Alert`
  here is a minimal stderr/log one-liner placeholder, not a delivery system.
- **Do not test `Loop.Run`** (a blocking `select` over a ticker — untestable without sleeping; the
  reviewer already ratified `Run` as correctly untested). Keep all branching/wiring in `registerHubs`.
- Do not rename the private `logclient.origin` or change its body; do not give `Origin` its own copy of
  the derivation math (it must delegate to `origin`). Do not carry origin on `registry.Entry`.

## Implementation Notes
- **Origin export (the open decision):** add `func Origin(baseURL string) (string, error) { return
  origin(baseURL) }` to `origin.go` with a one-line docstring. Rationale per learnings ("One `origin()`
  helper, golden-tested against both live hubs"): there stays exactly **one** derivation; `Origin` only
  exposes it so the binary can pass `<domain>/log` to `store.UpsertHub`'s `origin` argument. Carrying
  origin on `registry.Entry` would create a second derivation path and is rejected.
- **`main.go` structure** — keep `main` thin, push logic into the helper:
  - `get := os.LookupEnv` — `os.LookupEnv` already has the `func(string) (string, bool)` shape
    `config.Load` wants; pass it directly.
  - `cfg, err := config.Load(get)`; on error print to `os.Stderr` and `os.Exit(1)` (the `main` shell
    owns process exit — keep `os.Exit` out of the testable helper).
  - `data, err := os.ReadFile(cfg.RealmPath)` then `entries, err := registry.Parse(data)`. Config
    learning: a whitespace-only path passes `config.Load` and fails here at `os.ReadFile` — wrap that
    error with the path so the startup failure is clear.
  - `st, err := store.Open(cfg.DBPath)`; defer close with `defer func() { _ = st.Close() }()`
    (`Close` returns an error).
  - **Extract** `func registerHubs(ctx context.Context, st *store.Store, entries []registry.Entry)
    ([]follower.HubTarget, error)`: for each `Entry`, derive `org, err := logclient.Origin(e.BaseURL)`
    (wrap + name the bad domain on error), call `id, err := st.UpsertHub(ctx, e.Domain, org,
    e.BaseURL)`, append `follower.HubTarget{HubID: id, BaseURL: e.BaseURL}`. Return the slice (first
    error short-circuits). This is the unit `main_test.go` drives.
  - Build `loop := &follower.Loop{Store: st, Fetcher: logclient.NewHTTPFetcher(nil), Targets: targets,
    Normal: cfg.Normal, Frozen: cfg.Frozen, Alert: <log-to-stderr one-liner>}` and call `loop.Run(ctx)`
    where `ctx` comes from `signal.NotifyContext(context.Background(), os.Interrupt)` so SIGINT cleanly
    cancels and `Run` returns `ctx.Err()`.
  - File starts with a docstring (project convention) explaining it is the monitor entrypoint.
- **`main_test.go`** drives `registerHubs` only: `store.Open(filepath.Join(t.TempDir(), "test.db"))`,
  build the two real testnet `Entry`s (`sb0.iscc.id`, `sb1.amlet.id`) inline or via
  `registry.Parse` of the fixture at `internal/registry/testdata/realm.txt`, call `registerHubs`, and
  assert: (a) two targets returned; (b) each `HubTarget.BaseURL == "https://"+domain` and `HubID > 0`;
  (c) **idempotency** — a second `registerHubs` call returns identical `HubID`s (`UpsertHub` is
  idempotent on domain). Keep it a `func Test`, no test class. The test must not start `Run`.
- Correctness rule in play: **Origin = `<domain>/log`, never the bare domain** (learnings, highest-prob
  bug) — feed `Origin(BaseURL)` to `UpsertHub`'s `origin` arg, never `e.Domain`.
- Oracle/conformance gate is correctly **N/A** here: nothing touches a proof/verify/merkle/signature/
  fsck path. `Origin` only re-exports the already-golden-tested `origin`; `go.mod`/`go.sum` stay
  byte-identical (no new dependency — `net/http` is already in the logclient closure via `didresolve.go`).

## Verification
- `mise run check` is green (`go build ./...` now also compiles `cmd/iscc-monitor`; `go vet ./...`;
  `go test ./...` all pass) and `gofmt -l .` prints nothing.
- `go build -o /tmp/iscc-monitor ./cmd/iscc-monitor` exits 0 (binary compiles).
- `go test -run TestOrigin ./internal/logclient` passes — the exported `Origin` path stays golden:
  `Origin("https://sb0.iscc.id") == "sb0.iscc.id/log"` and
  `Origin("https://sb1.amlet.id") == "sb1.amlet.id/log"`.
- `go test -run TestRegisterHubs ./cmd/iscc-monitor` passes: two targets, each `HubID > 0`, each
  `BaseURL == "https://"+domain`, and a second `registerHubs` call returns identical `HubID`s
  (idempotent).
- `git diff -- go.mod go.sum` is empty (no dependency added).
- Missing-env failure surfaces cleanly: `env -u ISCC_MONITOR_DB -u ISCC_MONITOR_REALM
  /tmp/iscc-monitor` exits non-zero and prints a `config: required key` message to stderr (no panic).

## Done When
`cmd/iscc-monitor` builds and runs as the wired config→registry→store→loop entrypoint, `registerHubs`
is golden-tested for both real testnet hubs (idempotent, origin-correct), and every Verification check
passes with `mise run check` green and `go.mod`/`go.sum` byte-identical.
