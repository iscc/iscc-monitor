# Next Work Package

## Step: CORS middleware on every public GET (M3 cross-cutting HTTP slice — part 1)

## Goal
Add a single CORS middleware leaf that wraps the monitor's one public mux, so every served
surface (`/metrics`, `/healthz`, `/inclusion`, `/consistency`, `/entries`, and the raw tlog-tiles
mirror) answers cross-origin browser GETs uniformly. This is the first piece of M3's
"CORS on every public GET" requirement (target.md M3) and the prerequisite for the in-browser
verifier app (`monitor.iscc.codes`) and the dashboard fetching a monitor instance's data client-side.

## Scope
- **Create**: `internal/corsmw/corsmw.go` — a `Handler(next http.Handler) http.Handler` middleware
  that sets `Access-Control-Allow-Origin: *` on every response and answers `OPTIONS` preflights with
  `204 No Content`.
- **Modify**: `cmd/iscc-monitor/main.go` — wrap the assembled mux in `buildMux` (the single place all
  public routes converge) with `corsmw.Handler(...)` so the wrap applies once, uniformly. Add the
  `internal/corsmw` import.
- **Create (test, not counted toward the 3-file limit)**: `internal/corsmw/corsmw_test.go`.
- **Reference**:
  - `/workspace/iscc-monitor/internal/metricshttp/handler.go` — the canonical tiny-leaf-package shape
    (package docstring, single exported `Handler`, stdlib-only import) to mirror.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — `buildMux` (lines 145-150) is the wrap point;
    `serveMetrics` (line 123) hands `buildMux(...)` straight to `http.Server.Handler`, so one wrap
    covers the single listener and every mounted subtree.
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` and
    `/workspace/iscc-monitor/internal/tilesserve/handler.go` — both already document
    "CORS … intentionally out of scope for this slice"; this step is what those notes defer to. Do NOT
    change those handlers — CORS belongs in the one middleware, never duplicated per handler.
  - `/workspace/iscc-monitor/.claude/context/target.md` — M3 line "REST surface (CORS on every public GET)".
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main_test.go` — `TestMirrorRouter` /
    `TestMirrorInclusionRoute` / `TestMirrorEntriesRoute` exercise the mux; they must stay green
    through the wrapped mux.

## Not In Scope
- Caching / `Cache-Control` / conditional-GET (`ETag` / `If-None-Match` / `Last-Modified`) headers —
  those are the SECOND M3 cross-cutting slice, deliberately separated to keep this ≤2 production files
  and one concern.
- The `verify-for-me` verdict surface, the `/` landing page, the server-rendered dashboard, or the log
  browser — later M3 feature slices.
- Per-origin allow-listing, `Access-Control-Allow-Credentials`, or `Vary: Origin` — the monitor serves
  public, credential-free, read-only data, so wildcard `*` is the correct and simplest policy. Do NOT
  combine `Allow-Credentials: true` with `*` (the browser rejects that pairing).
- Touching `proofserve` / `tilesserve` / `metricshttp` / `healthz` handlers or their "out of scope"
  doc comments — the middleware wraps them; it does not edit them.
- Draining any open `normal` issue (`TestPollHubFork` cleanup, frozen-advance, `AcceptCheckpoint`
  context reuse, tile-writer `p` vocab, `AdvanceAccepted`, `CheckConsistency` collapse) — weighed and
  deferred to land the coherent M3 HTTP-plumbing arc first.

## Implementation Notes
- **Single middleware, applied once.** Mirror `metricshttp`'s package shape: a file-level docstring
  explaining why CORS lives in its own leaf (the policy is defined in exactly one place and rides
  every route from `buildMux`, never duplicated per handler), and one exported function
  `func Handler(next http.Handler) http.Handler`. It imports only `net/http`.
- **Behavior:**
  - On EVERY request, set `Access-Control-Allow-Origin: *` BEFORE delegating, so the header is present
    on 200, 404, 405, and 500 alike.
  - If `r.Method == http.MethodOptions`: treat as a preflight — also set
    `Access-Control-Allow-Methods: "GET, OPTIONS"` and `Access-Control-Allow-Headers: "*"`, then
    `w.WriteHeader(http.StatusNoContent)` and RETURN without calling `next` (the inner handlers only
    speak GET and would 405 an `OPTIONS`; the preflight must succeed so the browser proceeds to the
    real GET).
  - Otherwise delegate to `next.ServeHTTP(w, r)` unchanged — GET/HEAD/POST flow through to the
    existing 405/404/200 logic untouched. Only `OPTIONS` is short-circuited; a non-GET non-OPTIONS
    request still reaches the inner handler and gets the existing 405.
- **Header order matters with `http.Error`.** Set `Access-Control-Allow-Origin` BEFORE `next.ServeHTTP`:
  the inner handlers call `w.WriteHeader` (via `http.Error` or the first body write), after which header
  mutations are ignored. Setting it in the middleware first guarantees it lands on every response,
  including error bodies.
- **Wire point.** In `main.go`'s `buildMux`, return `corsmw.Handler(mux)` instead of the bare `mux`
  (keep `buildMux`'s return type `http.Handler` — it already is). This is the lone convergence point:
  `serveMetrics` feeds `buildMux(...)`'s result directly to `http.Server.Handler`, so one wrap covers
  the single listener and every mounted subtree (metrics, healthz, and all per-hub mirror/proof routes).
- **Relevant Correctness rule (learnings.md, "HTTP leaf" pattern + `proof/verify` purity rule):** keep
  new leaf packages import-clean — `corsmw` must import only `net/http` (no `store`, no `logclient`),
  preserving the one-directional dependency graph the other HTTP leaves (`metricshttp`, `healthz`,
  `tilesserve`) maintain. It is NOT on the WASM-shared verifier path (that rides `internal/didweb`), but
  staying stdlib-only keeps it trivially correct.
- **No new dependency:** `net/http` is already in the closure, so `go.mod` / `go.sum` / `schema.sql`
  stay byte-unchanged. The oracle/conformance gate is correctly N/A here (pure HTTP header wiring; no
  signature / RFC-6962 / Merkle / did:web / fsck / proof path) — the same posture the `tilesserve` and
  `metricshttp` wiring slices took.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .`
  empty).
- `go test ./internal/corsmw` passes with table cases asserting:
  - a GET response carries `Access-Control-Allow-Origin: *` and the inner handler's status/body are
    delegated through unchanged;
  - an `OPTIONS` request returns `204 No Content` with `Access-Control-Allow-Origin: *` and an
    `Access-Control-Allow-Methods` containing `GET`, and the inner handler is NOT invoked (assert via a
    sentinel handler that records whether it ran);
  - the `Access-Control-Allow-Origin: *` header is present even when the inner handler writes a
    non-200 (an inner `http.Error(w, ..., 404)` still carries the CORS header).
- `go test -run TestMirror ./cmd/iscc-monitor` passes — the existing router tests still go green through
  the now-wrapped mux (proves the wrap did not break routing or change GET responses).
- `grep -rn "Access-Control" internal/proofserve internal/tilesserve internal/metricshttp internal/healthz`
  is empty (no per-handler duplication; the header is set only in `internal/corsmw`).
- `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` exits 0 (no dependency/schema change).
- `go list -deps ./internal/corsmw | grep -E 'iscc-monitor/internal/(store|logclient)'` is empty (leaf
  stays import-clean).

## Done When
`corsmw.Handler` wraps the public mux once via `buildMux`, every public GET (and error) response
carries `Access-Control-Allow-Origin: *`, `OPTIONS` preflights succeed with `204`, and all
Verification checks above pass with `mise run check` green.
