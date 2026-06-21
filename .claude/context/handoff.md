## 2026-06-21 — Wire tilesserve.Handler into the binary with a per-hub mirror router

**Done:** Mounted the already-built `tilesserve.Handler` per-hub on the monitor binary's single HTTP
server, so each followed hub's mirrored tlog-tiles artifacts are now served at its canonical origin
prefix (`/<origin>/checkpoint`, `/<origin>/tile/...`, `/<origin>/tile/entries/...`) alongside the
existing `/metrics`. This turns the built-but-unserved raw-mirror surface into a live inbound
transport (the foundation for the M2/M3 `verify-for-me` proof-serving slice).

**Files changed:**
- `cmd/iscc-monitor/main.go`: added package-local `hubRoute{HubID, Origin}`; `registerHubs` now
  returns index-aligned `([]follower.HubTarget, []hubRoute, error)` (re-using the `org` it already
  derives); added `mirrorHandler(st, routes)` (per-hub `http.StripPrefix("/"+origin+"/")` →
  `tilesserve.Handler(store.SQLiteFetcher{...})` on a `*http.ServeMux`) and `buildMux(st, routes, m)`
  (mirror subtrees + `/metrics` on one mux); `serveMetrics` now serves that combined mux on the one
  listener. `/metrics` registration behavior unchanged.
- `cmd/iscc-monitor/main_test.go`: updated `TestRegisterHubs` for the new 3-value signature (asserts
  routes are index-aligned and carry the full `<domain>/log` origin); added `TestMirrorRouter` driving
  `buildMux` directly via `httptest.NewRecorder`/`ServeHTTP` (no socket).

**Verification:** `mise run check` → green (all 12 packages `ok`; build + vet + test). Per-criterion:
- [x] `mise run check` green; `gofmt -l .` empty.
- [x] `go test -run TestMirror -count=1 ./cmd/iscc-monitor` PASS — all 4 subtests:
  `GET /sb0.iscc.id/log/checkpoint` → 200 byte-equal seeded BLOB; `GET /sb0.iscc.id/log/tile/0/000`
  (unmirrored under prefix) → 404; `GET /sb0.iscc.id/checkpoint` (missing `/log`) → 404; `GET /metrics`
  → 200.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exit 0 — no schema/dep change.
- [x] `go list -deps ./internal/store | grep -E 'internal/tilesserve|net/http'` empty — store stays a
  leaf; the binary depends on `tilesserve → store`, never the reverse.

**Next:** The raw static mirror is now served. The next M2 slice is the proof-computing
`verify-for-me` REST surface (`inclusion`/`consistency`/`entries`-as-proofs) building on this inbound
transport — it consumes `ConsistencyProofFromTiles` / `InclusionProofFromTiles` /
`VerifyInclusionEvidence` over the same `SQLiteFetcher`, and (per the M3 split) will own CORS, caching,
conditional GET, and healthz that this slice intentionally deferred.

**Notes:**
- **Non-vacuousness proven by a reverted mutation:** dropping the trailing slash from the mount prefix
  (`"/"+r.Origin` instead of `"/"+r.Origin+"/"`, which kills `http.ServeMux` subtree matching) fails
  the 200-byte-equal subtest; reverted → green. A green-but-misrouted router cannot ship.
- **Scope-clean:** only the one production file + its test changed. `tilesserve.Handler`'s routing and
  signature are untouched; `internal/store`/`internal/follower`/`schema.sql`/`go.mod`/`go.sum` are
  byte-identical to HEAD. No `HubID→origin` store lookup was added — the origin is re-derived in
  `registerHubs` via `logclient.Origin`, exactly as the prior code already did.
- **`StripPrefix` down to the single leading slash** is the key wiring detail: the handler trims
  exactly one leading slash, so mounting at `/<origin>/` and stripping `/<origin>/` leaves the handler
  seeing `/checkpoint`, `/tile/0/000`, etc. The full-`<origin>` (`<domain>/log`) prefix is load-bearing
  — a request missing `/log` does not match and 404s (asserted).
- **Single connection reused** by all per-hub `SQLiteFetcher`s through the shared `*store.Store`
  (`SetMaxOpenConns(1)`, reads serialize, ADR-0005/0007); no second DB handle opened.
- Oracle/conformance gate correctly N/A for this slice — pure HTTP wiring of existing packages, no
  signature/RFC-6962/Merkle/did:web/fsck path introduced (`tilesserve` serves opaque BLOBs).
- The 6 open `normal` issues are orthogonal and untouched (none of
  `checkpoints.go`/`accept.go`/`follower.go`/`ingest.go`/`fetcher.go`/`tiles.go`/`consistency.go` was
  modified).
