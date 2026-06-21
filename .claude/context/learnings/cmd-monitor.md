<!-- area: cmd/iscc-monitor (main.go, registerHubs, buildMux) -->
<!-- indexed-as: cmd-monitor.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `cmd/iscc-monitor` — binary wiring

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Monitor binary (`cmd/iscc-monitor`)

- **The binary is the only consumer that wires all four M1 leaves**: `config.Load(os.LookupEnv)` →
  `os.ReadFile(RealmPath)` → `registry.Parse` → `store.Open` → `registerHubs` → `follower.Loop.Run`.
  `main` stays thin (owns the single `os.Exit`); all branching lives in the testable `registerHubs`
  (`Loop.Run` is correctly untested — blocking ticker select). Verified end-to-end offline: a long
  `NORMAL=10m` means no tick fires, SIGINT exits 0, and the `hubs` rows persist with
  `origin == <domain>/log` (sb0→`sb0.iscc.id/log`, sb1→`sb1.amlet.id/log`) — never the bare domain.
- **`Origin(baseURL)` is a one-line re-export of the private `origin`, NOT a second deriver** — the
  golden `TestOrigin` vectors cover it because `Origin` delegates; `TestOriginExport` only re-asserts
  the two live-hub vectors. The private `origin` body and its two internal callers stayed byte-identical.
- **`Loop.lastPoll` is lazily inited inside `Tick` (`if l.lastPoll == nil`), so the binary can build
  `&follower.Loop{…}` without setting the unexported `lastPoll`** — no nil-map-write panic. Confirms
  the follower author handled the bare-struct construction the binary relies on.
- **`Run` returns `ctx.Err()` unwrapped**, so `main.go`'s `err != context.Canceled` (a `==`, not
  `errors.Is`) is correct for a `signal.NotifyContext(os.Interrupt)` cancel → clean exit 0. If `Run`
  ever wraps the cancel error, switch to `errors.Is`; today the bare `==` holds (smoke-verified).
- **Whitespace-only `RealmPath` now fails cleanly at the binary's `os.ReadFile`** with the path named
  (`read realm document "   ": open …: no such file or directory`), closing the config presence-only
  gap noted in earlier learnings — the binary owns the fs error, config owns presence.
- **`tilesserve.Handler` is now wired per-hub on the one HTTP server** via `mirrorHandler(st, routes)`
  + `buildMux(st, routes, m)` (mirror subtrees + `/metrics` on a single `*http.ServeMux`); `serveMetrics`
  serves that combined mux on the lone listener (no second socket). The mount is the load-bearing
  detail: `mux.Handle("/"+origin+"/", http.StripPrefix("/"+origin+"/", h))` — the **trailing slash**
  arms `http.ServeMux` subtree matching, and `StripPrefix` down to the single leading slash gives the
  handler exactly the `/checkpoint`-style suffix it trims. Reviewer re-ran the trailing-slash mutation
  (drop the `+"/"` → 404 on the 200-byte-equal subtest, reverted → green): a green-but-misrouted router
  cannot ship. Origin is `<domain>/log` (full, never bare), re-derived via `logclient.Origin` in
  `registerHubs` (now returns index-aligned `([]HubTarget, []hubRoute, error)`) — a request missing
  `/log` does not match the prefix and 404s. `hubRoute{HubID, Origin}` is package-local to `main`
  (`HubTarget` carries no origin; the follower needs none). All per-hub `SQLiteFetcher`s share the
  store's single connection (ADR-0005/0007). Oracle gate correctly N/A — pure HTTP wiring of existing
  packages, `tilesserve` serves opaque BLOBs (no signature/RFC-6962/Merkle/did:web/fsck path);
  store stays a leaf (`go list -deps ./internal/store | grep tilesserve` empty — dep is binary→
  tilesserve→store), go.mod/go.sum/schema byte-identical.
- **`/healthz` is a `metricshttp`-style leaf that pings the store WITHOUT importing it.** `internal/
  healthz` declares its own 1-method `Pinger interface { Ping(context.Context) error }` (NOT a `store`
  import); `*store.Store` satisfies it structurally via a thin `Ping(ctx) error` → `db.PingContext`
  (`%w`-wrapped). Verified the dep stays one-directional both ways: `go list -deps ./internal/healthz`
  has neither `internal/store` nor `database/sql`, and `go list -deps ./internal/store` has neither
  `net/http` nor `internal/healthz`. Mounts as an EXACT path `mux.Handle("/healthz", …)` next to
  `/metrics` in `buildMux` — no collision with the per-hub `/<domain>/log/` *subtree* prefixes (those
  need the trailing slash to match; `/healthz` is exact). The `_, _ = io.WriteString(w, body)` drop is
  the documented post-status-write convention (a fixed byte-literal body cannot fail for content
  reasons; only a broken client conn, unrecoverable after `WriteHeader`), identical to `metricshttp`
  /`proofserve.writeEvidence` — NOT a swallowed-error gate dodge. Oracle gate correctly N/A (HTTP wiring
  + a DB ping; no signature/RFC-6962/Merkle/did:web/fsck/proof path); WASM purity rides `internal/
  didweb` (untouched, builds green); go.mod/go.sum/schema byte-identical.
