## 2026-06-21 — Serve GET /healthz (liveness + store readiness) on the shared mux

**Done:** Added a `GET /healthz` endpoint on the monitor's single listener that reports process
liveness and SQLite reachability. A new `internal/healthz` leaf package wraps a 1-method `Pinger`
interface in an `http.Handler` (200 + `{"status":"ok"}` on a clean ping, 503 +
`{"status":"unavailable"}` on a ping error, 405 on non-GET); `*store.Store` satisfies `Pinger` via a
thin new `Ping(ctx)` method; `buildMux` mounts `healthz.Handler(st)` at the exact path `/healthz` next
to `/metrics`.

**Files changed:**
- `internal/healthz/handler.go` (new): leaf package mirroring `metricshttp` — `Pinger interface {
  Ping(context.Context) error }` + `Handler(Pinger) http.Handler`; fixed byte-literal JSON bodies,
  status chosen up-front and `WriteHeader` before the body, Content-Type set before WriteHeader.
- `internal/healthz/handler_test.go` (new): `httptest.ResponseRecorder` tests over a `fakePinger`
  seam — `TestHealthzOK` (200 + body + content type), `TestHealthzUnavailable` (503 + body),
  `TestHealthzMethodNotAllowed` (405).
- `internal/store/sqlite.go`: added exported `Ping(ctx) error` delegating to `db.PingContext` with a
  `%w` wrap; added `context` import. Store stays a leaf.
- `internal/store/sqlite_test.go`: `TestStorePing` (nil on a freshly opened store) + `context` import.
- `cmd/iscc-monitor/main.go`: `mux.Handle("/healthz", healthz.Handler(st))` in `buildMux` + the
  `internal/healthz` import; updated `buildMux` doc.
- `cmd/iscc-monitor/main_test.go`: new `healthz served on shared mux` sub-assertion in
  `TestMirrorRouter` (200 + `{"status":"ok"}`).

**Verification:** `mise run check` → green (`go build`/`go vet`/`go test ./...` all pass; all 14
packages `ok`, healthz included). Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -run TestHealthz ./internal/healthz` PASS — 200/ok, 503/unavailable, 405.
- [x] `go test -run TestMirror ./cmd/iscc-monitor` PASS — incl. the new `/healthz` 200+body sub-test.
- [x] `go test -run 'TestPing|TestStore' ./internal/store` PASS — `TestStorePing` returns nil.
- [x] `go list -deps ./internal/healthz | grep -E 'internal/store|database/sql'` empty (store-free,
  db-free leaf).
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exit 0 (no schema/dep change).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` OK (load-bearing WASM purity untouched).
- [x] `go list -deps ./internal/store | grep net/http` empty (store stays a leaf; dep direction is
  binary → healthz, never the reverse).

**Next:** The remaining M3 cross-cutting HTTP slices noted as out-of-scope here — CORS, caching, and
conditional-GET (ETag/If-None-Match) headers on the mirror/proof surfaces — each its own slice.
Alternatively, the still-open `TestPollHubFork` re-detection cleanup (drive fork re-detection through a
second `PollHub` now that `CheckpointAt` is deterministic; remove the stale comment) — a quick win that
closes the retargeted issue.

**Notes:**
- Oracle/conformance gate is correctly N/A for this slice: pure HTTP wiring + a DB ping, no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path touched (stated in the package doc).
- Used `io.WriteString` with string-literal bodies (no `encoding/json` marshal) so there is no
  marshal-failure branch — the post-status write error is dropped deliberately, matching the
  `metricshttp.Handler` / `proofserve.writeEvidence` convention.
- `healthz` defines its own `Pinger` rather than importing `store`, so the dep edge is one-directional
  (binary → healthz, binary → store); `*Store` satisfies it structurally. Confirmed no `internal/store`
  or `database/sql` in healthz's dep closure.
- Scope: 3 production files (`internal/healthz/handler.go`, `internal/store/sqlite.go`,
  `cmd/iscc-monitor/main.go`) + 3 test files. Nothing from `## Not In Scope` touched; no
  `proofserve`/`tilesserve` entries surface, no CORS/caching, no follower/poll-loop health, no
  `schema.sql`/`go.mod`/`go.sum` change.
