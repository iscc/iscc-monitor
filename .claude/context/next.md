# Next Work Package

## Step: Serve GET /healthz (liveness + store readiness) on the shared mux

## Goal
Add the first of the M3-deferred cross-cutting HTTP concerns — a `/healthz` endpoint that reports
process liveness and SQLite reachability — on the monitor's single listener. Every public-facing
milestone (dashboard, REST, verify-for-me) needs a readiness probe; this lands the smallest
verifiable piece now and unblocks M3 without duplicating the already-served `entries` surface (the
raw entry-bundle BLOB is already served verbatim by `tilesserve`, so M2's served-proof bar —
inclusion + consistency + entries — is met).

## Scope
- **Create**: `internal/healthz/handler.go` — a tiny `net/http` leaf package, mirroring
  `internal/metricshttp/handler.go`. `Handler(p Pinger) http.Handler` where `Pinger` is a 1-method
  interface `Ping(ctx context.Context) error` (so the package depends on neither `store` nor
  `database/sql` — `*Store` satisfies it structurally, keeping the dep direction binary→healthz,
  never the reverse). On `GET`: call `p.Ping(r.Context())`; on nil error write `200` with a fixed
  JSON body `{"status":"ok"}`; on a Ping error write `503` with `{"status":"unavailable"}`. Non-GET
  → `405`.
- **Modify** (≤3 non-test/doc production files):
  1. `internal/store/sqlite.go` — add an exported `Ping(ctx context.Context) error` method on
     `*Store` that delegates to `s.db.PingContext(ctx)` and `%w`-wraps the error. `database/sql` and
     `fmt` are already imported; store stays a leaf.
  2. `cmd/iscc-monitor/main.go` — mount `healthz.Handler(st)` at `/healthz` inside `buildMux`
     (next to the existing `mux.Handle("/metrics", …)`); add the `internal/healthz` import.
- **Reference** (exact paths):
  - `/workspace/iscc-monitor/internal/metricshttp/handler.go` — the exact leaf-package +
    drop-write-error-after-200 pattern to mirror (Content-Type set before body; the method posture).
  - `/workspace/iscc-monitor/internal/metricshttp/handler_test.go` — the
    `httptest.ResponseRecorder` test shape to copy.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` lines 138–146 (`buildMux`) — where
    `/metrics` mounts on the shared mux; `/healthz` mounts the same way (an exact path, not a subtree).
  - `/workspace/iscc-monitor/internal/store/sqlite.go` lines 44–80 — `Store{db *sql.DB}`, `Open`,
    `Close`; the method shape to match.

## Not In Scope
- Do NOT add a `proofserve` `/entries` JSON endpoint — the raw entry-bundle BLOB is ALREADY served
  verbatim at the canonical `/tile/entries/<index...>` path by `tilesserve` (covered by
  `TestHandlerServesSeededBytes` "entry bundle"). A second entries surface over the same BLOB would
  duplicate it (DRY/YAGNI).
- Do NOT add CORS, caching, or conditional-GET headers here — `/healthz` lands first; those are
  their own later M3 slices.
- Do NOT report follower/poll-loop health, per-hub lag, frozen counts, or coverage in `/healthz` —
  this is process liveness + DB reachability only; the rich status surface is the M3 dashboard.
- Do NOT touch `internal/follower`, the `TestPollHubFork` cleanup, or any open `normal` issue here —
  those are separate slices.
- Do NOT change `schema.sql`, `go.mod`, or `go.sum` (no new dependency is needed).

## Implementation Notes
- **Leaf-package purity (Correctness rule: keep `net/http` out of `store`/`metrics`; one-directional
  deps).** `healthz` imports only `context`, `net/http`, and whatever it needs for the fixed body
  (`io.WriteString` with string literals, or `encoding/json` of a fixed-shape struct). It defines its
  own `Pinger interface { Ping(context.Context) error }` so it does NOT import `store` — `*Store`
  satisfies it structurally, even lighter than how `proofserve`/`tilesserve` depend on `store`.
  Verify the direction: `go list -deps ./internal/healthz | grep -E 'internal/store|database/sql'`
  must be empty.
- **`Store.Ping` is thin glue:** `func (s *Store) Ping(ctx context.Context) error { if err :=
  s.db.PingContext(ctx); err != nil { return fmt.Errorf("store.Ping: %w", err) }; return nil }`.
  `database/sql` + `fmt` are already in the file's imports. With `SetMaxOpenConns(1)` a Ping
  serializes on the single connection like every other read — fine for a liveness probe.
- **Handler body posture:** mirror `metricshttp.Handler`. Choose the `200`/`503` status up-front from
  the Ping result and call `w.WriteHeader` BEFORE the body so the error case is not masked by an
  implicit `200`; set `Content-Type: application/json` before `WriteHeader`. Write a FIXED
  string/byte-literal body (no marshal of dynamic data), so there is no marshal-failure branch —
  matching the "a fixed-shape body cannot fail for content reasons" convention in
  `proofserve.writeEvidence` / `metricshttp.Handler` (the post-status write error is dropped
  deliberately).
- **Mount as an exact path**, not a subtree: `mux.Handle("/healthz", healthz.Handler(st))` in
  `buildMux`. It sits next to `/metrics` on the same `*http.ServeMux` from `mirrorHandler`, so the
  per-hub mirror subtrees, `/metrics`, and `/healthz` all share one listener — the single-listener
  invariant in `serveMetrics`'s doc holds (no second socket).
- **Oracle/conformance gate is correctly N/A** for this slice: pure HTTP wiring + a DB ping, with no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path touched. State that explicitly in the
  package doc (matching how `tilesserve`/`metricshttp` note their N/A posture). The WASM purity
  invariant rides on `internal/didweb`, untouched here.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass).
- `gofmt -l .` is empty.
- `go test -run TestHealthz ./internal/healthz` passes: a `nil`-returning `Pinger` → `200` with body
  `{"status":"ok"}`; an error-returning `Pinger` → `503` with body `{"status":"unavailable"}`;
  `POST /healthz` → `405`.
- `go test -run TestMirror ./cmd/iscc-monitor` passes, including a new sub-assertion that
  `GET /healthz` on the shared mux returns `200`.
- `go test -run 'TestPing|TestStore' ./internal/store` passes: `Store.Ping` returns `nil` on a
  freshly opened store.
- `go list -deps ./internal/healthz | grep -E 'internal/store|database/sql'` is empty (healthz stays
  a store-free, db-free leaf).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).

## Done When
`/healthz` is served on the monitor's single listener returning `200` + `{"status":"ok"}` when the
store pings cleanly and `503` + `{"status":"unavailable"}` when it does not, with `mise run check`
green and every Verification check passing.
