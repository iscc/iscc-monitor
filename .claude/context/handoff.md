## 2026-06-21 — Review of: Serve GET /healthz (liveness + store readiness) on the shared mux

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added a `GET /healthz` endpoint (200 + `{"status":"ok"}` clean / 503 +
`{"status":"unavailable"}` on a ping error / 405 non-GET) as a tiny `internal/healthz` leaf that mirrors
`metricshttp`, plus a thin `Store.Ping` and the `buildMux` mount. The work is faithful to `next.md`,
scope-clean (exactly 3 production `.go` files, nothing from `## Not In Scope`), and the leaf-purity
invariant holds both directions — `healthz` imports neither `store` nor `database/sql`, and `store`
imports neither `net/http` nor `healthz`. Full suite green uncached (14 packages), gofmt clean, WASM
purity untouched.

**Verification:**
- [x] `mise run check` green — `go build`/`go vet`/`go test ./...` all pass; full suite re-run
  **uncached** with `go test -count=1 ./...` → all 14 packages `ok` (incl. `healthz`, `follower`,
  `proofserve`, `store`, `notecheck`).
- [x] `gofmt -l .` empty (exit 0).
- [x] `go test -run TestHealthz ./internal/healthz` PASS — `TestHealthzOK` (200 + body + JSON
  content-type), `TestHealthzUnavailable` (503 + body), `TestHealthzMethodNotAllowed` (405).
- [x] `go test -run TestMirror ./cmd/iscc-monitor` PASS — `TestMirrorRouter` (incl. the new `healthz
  served on shared mux` sub-test: 200 + `{"status":"ok"}`) and `TestMirrorInclusionRoute` intact.
- [x] `go test -run 'TestPing|TestStore' ./internal/store` PASS — `TestStorePing` returns nil on a
  freshly opened store.
- [x] `go list -deps ./internal/healthz | grep -E 'internal/store|database/sql'` empty (store-free,
  db-free leaf). Reverse direction also verified: `go list -deps ./internal/store | grep -E
  'net/http|internal/healthz'` empty (store stays a leaf; dep is binary → healthz, never reverse).
- [x] `git diff --quiet HEAD~1 -- internal/store/schema.sql go.mod go.sum` exit 0 (no schema/dep change).
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` OK (load-bearing WASM verifier seam untouched).
- [x] **Oracle / trust-root gate correctly N/A** — the diff touches no signature / RFC-6962 / Merkle /
  proof code / `internal/didweb` / fork-shrink-equivocation logic; it is pure HTTP wiring + a
  `db.PingContext` delegation. CI `notecheck` signature-parity oracle job is present and load-bearing
  in `.github/workflows/ci.yml` (accept real sb0 checkpoint + reject corrupted sig). N/A here, green
  there.
- [x] Gate-integrity scan over `@{upstream}..HEAD` (3 commits ahead): no `//nolint`/`t.Skip`/
  build-tag/swallowed-err additions; no Go test funcs or assertions deleted (the lone `178 func Test`
  match is `state.md` prose, not code). The `_, _ = io.WriteString` write-drop is the documented
  `metricshttp`/`proofserve` post-status convention (a fixed-body write error is unrecoverable after
  `WriteHeader`), not a check dodge.

**Issues found:** (none) — no new issues. No open issue was resolved by this slice (the `TestPollHubFork`
cleanup and the other `normal` items were explicitly out of scope), so `issues.md` is unchanged.

**Next:** The remaining M3 cross-cutting HTTP slices deferred here — CORS, caching, and conditional-GET
(ETag/If-None-Match) on the mirror/proof surfaces — each its own slice. Alternatively close the retargeted
`TestPollHubFork` re-detection issue: drive fork re-detection through a second `PollHub` now that
`CheckpointAt` is deterministic, and delete the stale "unordered LIMIT 1 … non-deterministic" comment
(`grep -rn "unordered LIMIT 1" internal/` should end empty) — a quick win.

**Notes:**
- Scope is exactly 3 production `.go` files (`internal/healthz/handler.go`, `internal/store/sqlite.go`,
  `cmd/iscc-monitor/main.go`) + 3 test files. Verified nothing from `## Not In Scope` was touched: no
  `/entries` proofserve endpoint, no CORS/caching/ETag header *code* (the "ETag" string appears only in
  this handoff's prose), `internal/follower` untouched, `schema.sql`/`go.mod`/`go.sum` byte-unchanged.
- `/healthz` mounts as an EXACT path, distinct from the per-hub `/<domain>/log/` *subtree* prefixes
  (which need their trailing slash for `ServeMux` subtree matching) — no routing collision, single
  listener preserved.
- Open `normal` issues remain for `define-next` to weigh against the state→target gap
  (frozen-hubs-still-advance, `AcceptCheckpoint` context reuse, tile-writer `p` vocabulary, deep
  `AdvanceAccepted`, `CheckConsistency` collapse, the `TestPollHubFork` cleanup). M7 (gossip/cosigner)
  stays out of scope and never blocks DONE. No open `critical` issue → Loop CONTINUE, not DONE (M3
  dashboard/REST surfaces are still unbuilt).
