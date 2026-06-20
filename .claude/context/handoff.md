# Handoff

## 2026-06-20 — cmd/iscc-monitor binary — wire config → registry → store → poll loop

**Done:** Stood up the `cmd/iscc-monitor` entrypoint that turns the four M1 leaves
(`config.Load` → `os.ReadFile`+`registry.Parse` → `store.Open`/`UpsertHub` → `follower.Loop.Run`)
into a runnable monitor process, with all registry→store wiring extracted into a testable
`registerHubs` helper. Resolved the open origin-export decision by adding a one-line
`logclient.Origin` wrapper that delegates to the existing private `origin` (no second deriver).

**Files changed:**
- `internal/logclient/origin.go`: added exported `Origin(baseURL string) (string, error)` — a
  one-line wrapper delegating to the private `origin`; private `origin` body and its two internal
  callers untouched.
- `internal/logclient/origin_test.go`: added `TestOriginExport` golden-asserting
  `Origin("https://sb0.iscc.id")=="sb0.iscc.id/log"` and `Origin("https://sb1.amlet.id")==
  "sb1.amlet.id/log"`.
- `cmd/iscc-monitor/main.go`: the binary entrypoint. Thin `main` (owns `os.Exit`) → `run()` does
  `config.Load(os.LookupEnv)` → `os.ReadFile(cfg.RealmPath)` (error wrapped with the path) →
  `registry.Parse` → `store.Open(cfg.DBPath)` (deferred close) → `signal.NotifyContext(…,
  os.Interrupt)` → `registerHubs` → `follower.Loop{…}.Run(ctx)`. `registerHubs` derives origin via
  `logclient.Origin(e.BaseURL)`, calls `UpsertHub(domain, origin, baseURL)`, builds `HubTarget`s.
  `alert` is a stderr one-liner placeholder.
- `cmd/iscc-monitor/main_test.go`: `TestRegisterHubs` drives `registerHubs` against a temp-dir store
  and the `internal/registry/testdata/realm.txt` fixture; asserts 2 targets, each `HubID>0` and
  `BaseURL=="https://"+domain`, and idempotency (second call → identical `HubID`s). Does not start
  `Run`.

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all `ok`, 7
packages). `gofmt -l .` → empty. Per-criterion:
- [x] `go build -o /tmp/iscc-monitor ./cmd/iscc-monitor` → exit 0.
- [x] `go test -run TestOrigin ./internal/logclient` → PASS (golden `Origin` path).
- [x] `go test -run TestRegisterHubs ./cmd/iscc-monitor` → PASS (2 targets, `HubID>0`, origin-correct
  base URL, idempotent).
- [x] `git diff -- go.mod go.sum` → empty (no dependency added; `net/http` already in the logclient
  closure via `didresolve.go`).
- [x] `env -u ISCC_MONITOR_DB -u ISCC_MONITOR_REALM /tmp/iscc-monitor` → exit 1, stderr
  `iscc-monitor: config: required key "ISCC_MONITOR_DB" is missing` (no panic).
- [x] End-to-end smoke run (offline, `NORMAL=10m` so no tick fires): registered the hub then clean
  SIGINT shutdown (exit 0); persisted `hubs` row = `domain=sb0.iscc.id origin=sb0.iscc.id/log
  base_url=https://sb0.iscc.id` — confirms origin is `<domain>/log`, never the bare domain.

**Next:** The merkle-backed **equivocation** trigger — the heavy slice deferred from this and prior
steps. It needs `transparency-dev/merkle` + tile fixtures and is the first slice to trip the
oracle/conformance gate (`fsck` root-rebuild over a `SQLiteFetcher`, inclusion cross-check vs the
hub's `IsccLogInclusionProof`, `notecheck` parity in CI). Alternatively, the lighter remaining M1
gaps surfaced by this wiring: coverage (`monitored_since`), structured logging to replace the stderr
placeholders (`alert` + the swallowed `Run` error), `/metrics`, and the `hub_keys` did:web cache
write (with the stale sb1 fixture refresh `22b08f3e`→`069d0f14`).

**Notes:**
- Oracle/conformance gate is correctly **N/A** for this step: nothing touches a
  proof/verify/didweb/merkle/consistency/signature/fsck/notecheck path. `Origin` only re-exports the
  already-golden-tested `origin` (no new derivation), and `go.mod`/`go.sum` are byte-identical.
- The whitespace-only-path config gap flagged in the prior handoff is now surfaced cleanly: a path
  that passes `config.Load` (presence-only) fails at `os.ReadFile` with the path named
  (`read realm document %q: %w`). No silent failure.
- `Run` returns `ctx.Err()` (`context.Canceled`) on SIGINT; `run()` treats that as a clean shutdown
  (returns nil), so a normal Ctrl-C exits 0. Any other `Run` error propagates to `os.Exit(1)`.
- `Loop.Run` itself is intentionally untested (blocking `select` over a ticker — the reviewer already
  ratified it as correctly untested); all branching lives in the injected-`now` `Tick`/`due()` and
  the new `registerHubs`. The smoke run above exercises the real `Run` path manually but is not a
  committed test (it would require sleeping/network).
- The stderr `alert` and the binary's reliance on `Loop.Run`'s documented swallowed per-tick error
  are placeholders explicitly in `next.md`'s Not-In-Scope (real alert transport + structured logging
  are later steps), not gate-dodging.
