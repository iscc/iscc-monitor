## 2026-06-21 — CORS middleware on every public GET (M3 cross-cutting HTTP slice — part 1)

**Done:** Added `internal/corsmw`, a single stdlib-only middleware leaf
(`Handler(next http.Handler) http.Handler`) that sets `Access-Control-Allow-Origin: *` on every
response and short-circuits `OPTIONS` preflights with `204 No Content`, then wrapped the assembled
public mux once in `buildMux` (the lone route convergence point) so every served surface (`/metrics`,
`/healthz`, `/inclusion`, `/consistency`, `/entries`, raw tlog-tiles mirror) answers cross-origin
browser GETs uniformly without per-handler CORS code.

**Files changed:**
- `internal/corsmw/corsmw.go` (new): the middleware leaf — file docstring explaining why CORS lives in
  one place, single exported `Handler`, imports only `net/http`. Sets `Access-Control-Allow-Origin: *`
  BEFORE delegating (so it lands on 200/404/405/500 alike); on `OPTIONS` also sets
  `Access-Control-Allow-Methods: "GET, OPTIONS"` + `Access-Control-Allow-Headers: "*"`, writes 204, and
  returns without calling `next`; all other methods flow through unchanged.
- `cmd/iscc-monitor/main.go`: added the `internal/corsmw` import; `buildMux` now returns
  `corsmw.Handler(mux)` instead of the bare mux (return type stays `http.Handler`). Updated the
  `buildMux` doc comment to note the single CORS wrap.
- `internal/corsmw/corsmw_test.go` (new, test): table cases asserting GET delegation + `*` header,
  OPTIONS → 204 with `Allow-Methods` containing `GET` and the inner handler NOT invoked (sentinel
  `ran` flag), and a non-200 inner (`http.Error 404`) still carrying the `*` header.

**Verification:** `mise run check` → green (build + vet + test, all 15 packages `ok`); `gofmt -l .`
empty. Per-criterion:
- [x] `go test ./internal/corsmw` — PASS (3 table cases).
- [x] `go test -run TestMirror ./cmd/iscc-monitor` — PASS (router tests green through the now-wrapped mux).
- [x] `grep -rn "Access-Control" internal/proofserve internal/tilesserve internal/metricshttp internal/healthz`
  — empty (no per-handler duplication).
- [x] `git diff --quiet HEAD -- go.mod go.sum internal/store/schema.sql` — exits 0 (no dep/schema change).
- [x] `go list -deps ./internal/corsmw | grep -E 'internal/(store|logclient)'` — empty; closure is
  exactly `[net/http]`, only `iscc-monitor` dep-entry is itself (import-clean leaf).

**Next:** The SECOND M3 cross-cutting HTTP slice — caching / `Cache-Control` / conditional-GET
(`ETag` / `If-None-Match` / `Last-Modified`) headers, deliberately separated out of this package to
keep this one concern. It can mirror the same single-wrap-in-`buildMux` pattern (a second middleware
composed around `corsmw.Handler(mux)`), but note immutable-full vs mutable-partial tiles need different
cache policies, so it likely needs the handlers to signal cacheability rather than a blind wrap — worth
scoping carefully. Alternatively, drain an open `normal` issue, or start the proof-bundle JSON +
verify-for-me feature arc.

**Notes:**
- **No deviations from `next.md`.** Scope is exactly 2 production files (`corsmw.go` + `main.go` wire
  point) + 1 test file; nothing from `## Not In Scope` was touched (no caching/ETag, no
  `proofserve`/`tilesserve`/`metricshttp`/`healthz` handler edits, no `Vary`/`Allow-Credentials`, no
  open-issue cleanup). The `proofserve`/`tilesserve` "CORS out of scope" doc comments were left in
  place per `next.md` — they correctly defer to this middleware and don't claim CORS is now handled
  there; a future cleanup could tighten that wording but it's out of scope here (flagging for review,
  not fixing).
- **Oracle/conformance gate correctly N/A** — pure HTTP header wiring, no
  signature/RFC-6962/Merkle/did:web/fsck/proof path introduced; same posture as the `tilesserve` and
  `metricshttp` wiring slices. `go.mod`/`go.sum`/`schema.sql` byte-unchanged (`net/http` already in the
  closure). WASM purity rides `internal/didweb` (untouched); `corsmw` is not on the WASM-shared verifier
  path but stays stdlib-only anyway.
- **Wildcard `*` is the deliberate policy** (`next.md`): the monitor serves public, credential-free,
  read-only data, so no per-origin allow-list and no `Allow-Credentials` (which the browser would reject
  paired with `*`). The OPTIONS short-circuit is required because the inner GET-only handlers would 405
  a preflight, blocking the browser's real GET.
