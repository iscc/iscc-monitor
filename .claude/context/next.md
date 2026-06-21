# Next Work Package

## Step: Server-rendered dashboard at `GET /` listing every realm hub (status + coverage)

## Advances
M3 Verify criterion (the nearest unmet one, brings M3 → 3/4):

> `GET /` returns `200 text/html` listing **every** realm hub with its glossary status + coverage
> window (golden-tested on a fixture store)

This is the third of M3's four Verify criteria. Two are met (CORS on every GET; verify-for-me JSON
verdict). The HTML log browser (`GET /<domain>/log/`) is the fourth and is the *next* sub-step in this
same HTML arc — see `## Not In Scope`. It is also the #1 item in `state.md`'s convergence-driven order
and the `review` handoff `**Next:**`. No `critical`/`normal` issue is open (the one open issue is
`low`, loop-skipped), so milestone work proceeds.

## Goal
Add a server-rendered HTML dashboard at the root path that enumerates every hub the monitor follows,
showing each hub's store-provable status (frozen / verified / inactive) and its coverage window
(`monitored_since` size + time, accepted `last_size`). This closes the third M3 Verify criterion and
gives the monitor its first human-facing surface.

## Scope
- **Create**:
  - `internal/dashboard/handler.go` — a new HTTP leaf: `Handler(st *store.Store) http.Handler` that
    renders the hub list as `200 text/html` from one store read. Use `html/template` with a single
    embedded template parsed once at package init (`template.Must`); GET-only (405 otherwise); handle
    only exact `/` (404 any other path — see Implementation Notes). This package owns the HTML;
    `cmd/iscc-monitor` only wires it. Mirror the leaf shape of `internal/healthz` / `internal/metricshttp`
    — a thin package-local handler whose only internal dep is `internal/store`.
  - `internal/dashboard/handler_test.go` — golden test at the HTTP seam (`httptest.ResponseRecorder`)
    against a fixture `store.Open(filepath.Join(t.TempDir(), "dash.db"))` populated with ≥2 hubs (one
    verified with coverage, one frozen) via `UpsertHub` + `AdvanceAccepted`/`AdvanceFollowState` +
    `SetCoverage` + `Freeze`. Assert: 200, `Content-Type: text/html; charset=utf-8`, body contains
    **every** hub's domain/origin and its status string (the frozen hub renders `frozen`, the verified
    one `verified`). Non-GET → 405; `GET /unknown` → 404.
- **Modify**:
  - `internal/store/hubs.go` (NEW file inside the existing `store` package — a store edit, not a new
    package) — add `ListHubs(ctx context.Context) ([]HubSummary, error)`: one query LEFT JOINing
    `hubs` + `follow_state` returning, per hub ordered by `hub_id`, the domain, origin, `active`,
    `last_size`, `frozen`, and `monitored_since_{size,time}`. Add the `HubSummary` struct. Store stays
    a leaf (returns plain Go types; imports only `database/sql` + stdlib; no `logclient`/`net/http`).
    Putting `ListHubs` in the existing `checkpoints.go` instead is acceptable — either way keep it ≤3
    production files and do NOT create a second package.
  - `cmd/iscc-monitor/main.go` — mount `dashboard.Handler(st)` at the **exact** path `/` in `buildMux`
    (`mux.Handle("/", dashboard.Handler(st))`), next to the existing exact `/metrics` and `/healthz`
    mounts; add the import.
  - `CLAUDE.md` — update the "Running a local dev instance" section: `/` no longer returns 404; it now
    serves the HTML dashboard. (Current text says "no dashboard UI yet (M3)" and "`/` returns 404".)
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/.claude/context/learnings/cmd-monitor.md` — `buildMux` mount discipline;
    exact-path vs trailing-slash subtree matching (the `/` exact mount must NOT shadow the per-hub
    `/<domain>/log/` subtrees or the exact `/metrics`/`/healthz`). Read before touching `main.go`.
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — the leaf-handler conventions
    and the post-status write-drop idiom (note: buffer the template into `bytes.Buffer` first, THEN
    write, so a render error is a 500 before any 200 — see Implementation Notes).
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — store-leaf purity rule and the
    `monitored_since` set-once coverage semantics before adding `ListHubs`.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` lines 475-486 (`hubStatus`) — the
    store-provable status mapping to mirror (`frozen` else `verified`); extend with `inactive` when
    `active == 0`.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `FollowState`, `CoverageInfo`,
    `UpsertHub`, `AdvanceAccepted`, `AdvanceFollowState`, `Freeze`, `SetCoverage` signatures for
    building the fixture store in the test.
  - `/workspace/iscc-monitor/internal/healthz/handler.go` — the thin leaf-handler + GET-gate shape to
    mirror.

## Not In Scope
- The HTML **log browser** `GET /<domain>/log/` (the fourth M3 Verify criterion) — that is the next
  sub-step in this arc; do not start it here. It mounts under the per-hub subtree, a different mount.
- Lag / violations / OTS columns and the richer in-memory metrics statuses
  (`unverified`/`unresolvable`/`rotated`): the dashboard derives status from the **store-provable
  subset only** (frozen / verified / inactive), exactly as `proofserve.hubStatus` does, so it stays
  golden-testable on a fixture store. Threading the `metrics.Registry` into the dashboard is later work.
- Any CSS framework / JS / WASM progressive-enhancement hook — plain semantic HTML only. WASM is a
  separate milestone.
- ETag / Cache-Control / conditional-GET on the dashboard (not a Verify criterion; the page is dynamic).
- Changing CORS or the `corsmw` wrap — `buildMux` already wraps the whole mux once, so the dashboard
  inherits `Access-Control-Allow-Origin: *` for free.

## Implementation Notes
- **Mount collision (load-bearing):** `http.ServeMux` matches the most-specific registered pattern, so
  `mux.Handle("/", dashboard.Handler(st))` is the lowest priority — the existing `/metrics`,
  `/healthz`, and each `/<domain>/log/` subtree all still win. Mounting the dashboard at `/` therefore
  does NOT shadow any existing route; it only catches what matches nothing else. But `mux.Handle("/",
  ...)` also receives `/anything-unknown`, so the handler must guard: `if r.URL.Path != "/" {
  http.NotFound(w, r); return }`. Confirm the existing `cmd/iscc-monitor` mux tests still pass after
  wiring (they assert `/metrics` and the mirror subtrees resolve).
- **Status mapping:** reuse the `proofserve.hubStatus` logic and extend it for the realm-registry
  `inactive` glossary status: `active == 0` → `inactive`; else `frozen` → `frozen`; else `verified`.
  Do NOT invent `unverified`/`unresolvable`/`rotated` — those live in the in-memory metrics registry,
  are not store-provable, and are documented out of scope. Use the exact glossary strings (CLAUDE.md
  "Hub status").
- **Template safety:** use `html/template` (NOT `text/template`) so hub domains auto-escape. Render
  into a `bytes.Buffer` first; if `tmpl.Execute` errors, `http.Error(w, ..., 500)` and return. Only on
  success set `Content-Type: text/html; charset=utf-8`, `WriteHeader(200)`, then copy the buffer to
  `w` (a post-200 write-drop is fine — the page is already committed). This avoids a half-rendered 200.
- **Store leaf purity (learnings):** `ListHubs` returns `[]HubSummary` of plain types; it must not
  import `logclient` or `net/http`. Use `LEFT JOIN follow_state` so a hub with no follow_state row
  still appears (its `last_size`/`frozen` read as NULL/0 via `sql.NullInt64`/`sql.NullBool` →
  zero-value). Verify with `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/logclient'`
  → empty. The dashboard imports `store`, never the reverse.
- **Coverage rendering (correctness rule — ADR-0001 coverage honesty):** show the `monitored_since`
  size+time when set, else render an explicit "no coverage yet" — never imply pre-coverage guarantees
  (CLAUDE.md "Coverage" glossary). Show the accepted `last_size` as the current observed size. The
  dashboard must show the coverage window and never imply pre-coverage guarantees.
- **Skeleton-first:** this is the minimal verifiable dashboard skeleton — hub list + status + coverage,
  exactly the columns the Verify criterion names ("status + coverage window"). Lag/violations/OTS
  columns and the metrics registry are deferred; the log browser is the next sub-step in this arc.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -count=1 ./internal/dashboard` passes (the new golden HTTP-seam test).
- `go test -count=1 ./internal/store ./cmd/iscc-monitor` passes uncached (ListHubs + mux wiring intact).
- `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/logclient'` is empty
  (store stays a leaf); `git diff --stat -- go.mod go.sum internal/store/schema.sql` is empty
  (no schema/dep change — the dashboard reads existing columns only).
- At the HTTP seam on the fixture store: `GET /` → 200, `Content-Type: text/html; charset=utf-8`, and
  the body contains the domain/origin of **every** fixture hub plus each hub's status string (the
  frozen fixture hub renders `frozen`, the verified one `verified`).
- `GET /` with method `POST` → 405; `GET /unknown` → 404 (the handler serves only exact `/`).

## Done When
`GET /` serves a `200 text/html` dashboard listing every realm hub with its store-provable glossary
status and coverage window, golden-tested on a fixture store by `go test ./internal/dashboard`, with
`mise run check` green and `internal/store` still a leaf.
