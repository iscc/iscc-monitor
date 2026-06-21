## 2026-06-21 — Server-rendered dashboard at `GET /` listing every realm hub (status + coverage)

**Done:** Added a server-rendered HTML dashboard at the exact path `/` that enumerates every followed
hub with its store-provable glossary status (frozen / verified / inactive) and coverage window
(`monitored_since` size + time, observed `last_size`). The dashboard is a thin leaf (only internal dep:
`internal/store`) rendering one new store read, `ListHubs`, via a once-parsed embedded `html/template`.
Closes the third of M3's four Verify criteria (M3 → 3/4); the HTML log browser is the remaining one.

**Files changed:**
- `internal/store/hubs.go` (new): `ListHubs(ctx) ([]HubSummary, error)` + the `HubSummary` struct — one
  `LEFT JOIN follow_state` read returning, per hub ordered by `hub_id`, domain/origin/active/last_size/
  frozen/coverage. Plain Go types only; store stays a leaf (no `net/http`/`logclient` in its closure).
- `internal/dashboard/handler.go` (new): `Handler(st *store.Store) http.Handler` — GET-only (405
  otherwise), exact-`/` only (404 any other path), renders into a `bytes.Buffer` first so a render error
  is a 500 before any 200, then sets `Content-Type: text/html; charset=utf-8` and copies. `hubStatus`
  mirrors `proofserve.hubStatus` and extends it with `inactive` (`active == 0`).
- `internal/dashboard/dashboard.html` (new): embedded semantic-HTML template (no CSS/JS), auto-escaped.
- `internal/dashboard/handler_test.go` (new): golden HTTP-seam test on a fixture store (verified hub
  with coverage + frozen hub) asserting 200, content type, every hub's domain/origin, both status
  strings, the coverage start; plus non-GET → 405, `/unknown` → 404, and a `hubStatus` mapping table
  covering the inactive case.
- `cmd/iscc-monitor/main.go`: mount `dashboard.Handler(st)` at exact `/` in `buildMux` (+ import);
  updated the `buildMux` doc comment for the most-specific-match non-shadowing rationale.
- `CLAUDE.md`: "Running a local dev instance" — `/` now serves the HTML dashboard, not 404.

**Verification:** `mise run check` → green, all 16 packages `ok` (build + vet + test). Per criterion:
- [x] `gofmt -l .` empty; `go build/vet/test ./...` clean.
- [x] `go test -count=1 ./internal/dashboard ./internal/store ./cmd/iscc-monitor` PASS uncached.
- [x] `go list -deps ./internal/store | grep -E 'net/http|internal/dashboard|internal/logclient'` empty
  (store stays a leaf). `internal/dashboard` own imports: `bytes embed html/template net/http
  internal/store` — only internal dep is `store`, never the reverse.
- [x] `git diff --stat -- go.mod go.sum internal/store/schema.sql` empty (no schema/dep change).
- [x] HTTP seam on the fixture store: `GET /` → 200, `text/html; charset=utf-8`, body names both hubs'
  domain+origin, renders `verified`/`frozen`, and shows `size 42 at <time>` coverage for the verified
  hub. `POST /` → 405; `GET /unknown` → 404. (Manually dumped the rendered HTML to confirm the table
  rows are correct, then removed the scratch test.)
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` → OK (WASM verifier seam untouched).

**Next:** The HTML **log browser** `GET /<domain>/log/` — the fourth and final M3 Verify criterion (M3 →
4/4). It mounts under the per-hub subtree (a different mount than `/`), so it belongs in `proofserve` or
a new per-hub HTML leaf reached through `hubHandler`, not in `internal/dashboard`. After M3 closes, the
proof-bundle assembler (the authoritative client-verifies path) is the natural next arc.

**Notes:**
- **Status is the store-provable subset only**, exactly as `next.md` scoped: `inactive` (active==0) →
  `frozen` → else `verified`. The richer `unverified`/`unresolvable`/`rotated` live in the in-memory
  metrics registry and are deliberately NOT threaded in — a documented limitation, not a defect.
- **No public setter for `hubs.active`** exists yet (it is registry-managed; `UpsertHub` inserts the
  schema default `1`). So the HTTP-seam test cannot drive a hub to `inactive` through the public store
  API. I cover the `inactive` mapping via `TestHubStatusMapping` (a white-box table test on the
  package-private `hubStatus`), which is why the test file is `package dashboard` not `dashboard_test`.
  When a `SetActive`/registry-deactivation writer lands, an end-to-end inactive-render assertion becomes
  possible — worth adding then.
- **Coverage honesty (ADR-0001) is observable in the fixture:** the frozen hub was advanced via
  `AdvanceFollowState` (not `AdvanceAccepted`), so it has `last_size 7` but `no coverage yet` — the page
  shows the observed size without implying a pre-coverage guarantee. The verified hub used
  `AdvanceAccepted`, which sets coverage in the same transaction, so it shows `size 42 at <time>`.
- Oracle/conformance gate correctly N/A: pure HTML rendering of persisted store rows — no signature /
  RFC-6962 / Merkle / did:web / fsck / proof path. The dashboard derives nothing cryptographic.
- `issues.md` unchanged (no new defects found; the one open `low` notecheck item is unrelated).
