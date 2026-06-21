# Next Work Package

## Step: Wire tilesserve.Handler into the binary with a per-hub mirror router

## Goal
Mount the already-built `tilesserve.Handler` in `cmd/iscc-monitor` so each followed hub's mirrored
tlog-tiles artifacts are actually *served* over HTTP at its canonical origin path
(`/<origin>/checkpoint`, `/<origin>/tile/...`, `/<origin>/tile/entries/...`). This turns the
built-but-unserved raw-mirror surface into a live inbound transport and is the foundation the M2/M3
`verify-for-me` proof-serving slice builds on.

## Scope
- **Modify**: `cmd/iscc-monitor/main.go` (add a `mirrorHandler` router helper + mount it on the same
  mux as `/metrics`; pass per-hub origins through from `registerHubs`). This is the only non-test
  production file.
- **Modify (tests)**: `cmd/iscc-monitor/main_test.go` (add a router test).
- **Reference**:
  - `/workspace/iscc-monitor/internal/tilesserve/handler.go` — `Handler(f store.SQLiteFetcher) http.Handler`;
    note it trims exactly ONE leading slash and then matches `checkpoint` / `tile/...` /
    `tile/entries/...`, so it MUST be mounted such that it sees `/checkpoint` etc. (use
    `http.StripPrefix` to drop the per-hub prefix down to the single leading slash).
  - `/workspace/iscc-monitor/internal/store/fetcher.go` — `SQLiteFetcher{Store, HubID}` is a plain
    struct literal (no constructor); reads are read-only over the shared single connection.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — existing `serveMetrics` mux pattern,
    `registerHubs` (already computes `org, _ := logclient.Origin(e.BaseURL)`), and the `run` wiring.
  - `/workspace/iscc-monitor/internal/follower/loop.go` — `HubTarget{HubID int64; BaseURL string}`
    (note: it carries no origin field; the router needs origin, so thread it from `registerHubs`).
  - `/workspace/iscc-monitor/internal/logclient/origin.go` — `Origin(baseURL)` → `<domain>/log`.

## Not In Scope
- **No proof-computing endpoints** (`inclusion`/`consistency`/`entries`-as-proofs). This slice serves
  only the raw static mirror BLOBs `tilesserve` already exposes; the `ProofBuilder` /
  `verify-for-me` REST surface is the next M2 slice.
- **No CORS, caching, conditional GET, healthz, or dashboard** — `tilesserve` already documents these
  as out of scope, and M3 owns them.
- **Do not change `tilesserve.Handler`'s routing or signature**, and do not touch
  `internal/store`/`internal/follower`/`schema.sql`/`go.mod`/`go.sum`.
- **Do not address the 6 open `normal` issues** here (this slice does not modify
  `checkpoints.go`/`accept.go`/`follower.go`/`ingest.go`/`fetcher.go`/`tiles.go`/`consistency.go`).
- Do not add a `HubID`→origin store lookup; reuse the origin `registerHubs` already derives.

## Implementation Notes
- **Routing key = the hub origin path.** Each hub's origin is `<domain>/log` (e.g. `sb0.iscc.id/log`).
  Mount each hub at prefix `"/" + origin + "/"` (e.g. `/sb0.iscc.id/log/`) and `http.StripPrefix` that
  prefix so the wrapped `tilesserve.Handler` sees `/checkpoint`, `/tile/0/000`, etc. — exactly the
  single-leading-slash suffix the handler trims. Register the prefix `"/sb0.iscc.id/log/"` (with a
  trailing slash, so `http.ServeMux` does subtree matching) on the same mux that already serves
  `/metrics`.
- **Thread origin from `registerHubs`.** `registerHubs` already computes `org` per entry but currently
  discards it (only `HubID`+`BaseURL` flow into `HubTarget`). The cleanest minimal change: have a
  sibling helper return a small `[]hubRoute{HubID int64; Origin string}` slice (re-deriving origin via
  `logclient.Origin`, OR returned alongside the existing `targets`), then a pure, testable helper
  `mirrorHandler(st *store.Store, routes []hubRoute) http.Handler` builds the mux. Keep `main`/`run`
  thin and put the per-hub `StripPrefix` + `tilesserve.Handler(store.SQLiteFetcher{Store: st, HubID:
  r.HubID})` loop in the helper so it is unit testable without binding a socket. Define the `hubRoute`
  struct local to `package main`.
- **Mount on the metrics mux.** Extend `serveMetrics` (or factor a `buildMux` helper it calls) so the
  one `http.Server` serves both `/metrics` and the per-hub mirror subtrees — avoid a second listener.
  Keep `/metrics` registered exactly as today (sole behavior unchanged).
- **Read-only reuse of the single connection.** `SQLiteFetcher` reads through the store's one open
  connection (`SetMaxOpenConns(1)`, single-writer discipline, ADR-0005/0007); concurrent HTTP reads
  share that connection — this is fine (reads serialize on it), do NOT open a second DB handle.
- **Correctness rule (learnings):** origin is `<domain>/log`, never the bare domain — the mount prefix
  must use the full origin, so a request to `/sb0.iscc.id/checkpoint` (missing `/log`) does NOT match.
  Reuse `logclient.Origin`, never hand-build the path.
- **Status passthrough:** the handler already maps 400/404/405/500 and an unmatched path under a hub
  prefix to 404; an entirely unknown top-level prefix falls through to the mux's default 404. No extra
  status logic needed in the router.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestMirror -count=1 ./cmd/iscc-monitor` passes (the new router test).
- New router test asserts, against a store seeded with one hub + a checkpoint BLOB (reuse the
  `registerHubs`/store seeding pattern already in `main_test.go`, served via a direct `ServeHTTP`
  against `httptest.NewRecorder`):
  - `GET /sb0.iscc.id/log/checkpoint` → 200 with body byte-equal to the seeded checkpoint BLOB.
  - `GET /sb0.iscc.id/log/tile/0/000` (an unmirrored path under the hub prefix) → 404.
  - `GET /sb0.iscc.id/checkpoint` (origin missing the `/log` segment) → 404 (wrong prefix, not matched).
  - `GET /metrics` still → 200 (the existing route is untouched on the shared mux).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dependency
  change — this is pure wiring of existing packages).
- `go list -deps ./internal/store | grep -E 'internal/tilesserve|net/http'` is empty (store stays a
  leaf; the binary depends on `tilesserve`→`store`, never the reverse).

## Done When
`tilesserve.Handler` is mounted per-hub on the binary's HTTP server at each hub's `<origin>/` prefix
and all Verification criteria pass — a real `GET /<origin>/checkpoint` returns the mirrored bytes while
`/metrics` and the store-leaf/no-dep-change invariants remain intact.
