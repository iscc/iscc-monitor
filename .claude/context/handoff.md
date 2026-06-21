## 2026-06-21 — Review of: Serve the raw tlog-tiles mirror (checkpoint / tile / entries) over HTTP from one hub's SQLiteFetcher

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `internal/tilesserve` adds one clean `Handler(f store.SQLiteFetcher) http.Handler` that
routes the three canonical tlog-tiles paths to `ReadCheckpoint`/`ReadTile`/`ReadEntryBundle` and serves
the raw mirror BLOBs verbatim, with correct 400/404/405/500 status mapping. The diff is exactly the two
new files + handoff (scope-clean), every `next.md` verification criterion passes, and the body-equality +
status assertions are reviewer-mutation-proven non-vacuous. The oracle gate is correctly N/A (no
trust-root path) — verified by dep-closure, not asserted.

**Verification:**
- [x] `mise run check` green — all 12 packages `ok` (build + vet + test).
- [x] `gofmt -l .` empty — no formatting failures.
- [x] `go test -run TestHandler -count=1 ./internal/tilesserve` PASS — all 9 sub-cases (full, partial,
  entries, checkpoint, 404 missing, 400 malformed tile, 400 malformed entries, 404 unmatched, 405 POST).
  (Test fn is `TestHandlerServesSeededBytes`; `-run TestHandler` matches it as a prefix.)
- [x] `git diff --quiet HEAD~1 -- internal/store/schema.sql go.mod go.sum` exit 0 — no schema/dep change
  (used HEAD~1, the pre-advance base, since the advance commit is HEAD).
- [x] `go list -deps ./internal/store | grep -E 'internal/tilesserve|net/http'` empty — store stays a
  leaf; `tilesserve → store`, never the reverse.
- [x] `GOOS=js GOARCH=wasm go build ./internal/didweb` exit 0 — WASM purity invariant untouched.
- [x] Full tile → 200 byte-equal seeded BLOB; never-mirrored tile → 404; malformed index → 400; POST →
  405 (all asserted, all mutation-proven non-vacuous).
- [x] Oracle/conformance gate correctly N/A — `go list -deps ./internal/tilesserve` pulls in no
  `internal/proof`/`logclient`/`didweb`/`merkle`/`note`/`rfc6962`; opaque BLOB transport only. CI
  `notecheck` parity job present + unchanged + green.
- [x] Gate-integrity scan over all unpushed commits (`origin/develop..HEAD`) — no `//nolint`/`t.Skip`/
  swallowed-error dodge/loosened gate. The lone `_, _ = w.Write` is the documented metricshttp idiom.

**Issues found:** (none) — the handler is correct, simple, and well-documented; no minor fixes needed.

**Next:** Wire `tilesserve.Handler` into `cmd/iscc-monitor/main.go` (the deferred binary slice): a per-hub
route prefix, a hub→origin router resolving the request's `HubID` from a path prefix or host, and a
read-only connection. That needs the multi-hub routing design `next.md` called out. In parallel, the
`consistency`/`inclusion` `verify-for-me` REST surface (M2/M3) builds on this inbound transport, consuming
`ConsistencyProofFromTiles`/`VerifyInclusionEvidence`.

**Notes:**
- **Routing robustness independently probed against the real tessera parser** (not just the 9 subcases):
  `tile/0` (no 2nd slash) → 400 via `strings.Cut` `ok=false`; `tile/0/`, `tile/0/abc` → parser error →
  400; `tile/0/001.p/44` → `width=44` passed straight through as the fetcher's `p` (shared 0==full). The
  `tile/entries/` switch case correctly precedes `tile/` (sub-prefix). No panic, no misroute.
- **Non-vacuousness proven by two reverted mutations:** (1) serving constant `"X"` fails all four
  byte-equal subtests; (2) collapsing the 404 mapping to 200 fails the never-mirrored + unmatched
  subtests. A green-but-wrong handler cannot ship.
- **6 open issues remain orthogonal + untouched** — none of the open-issue files (`checkpoints.go`,
  `accept.go`, `notecheck`, `follower.go`, `ingest.go`, `fetcher.go`, `tiles.go`, `consistency.go`) was
  modified this slice, so none is resolved or made stale. Backlog unchanged.
- **Intentional unwired export seam** (like prior M2 seams): no production caller yet (binary wiring is
  the next slice). `go vet` clean, not dead code. go.mod/go.sum/schema byte-identical across the full
  unpushed range.
- M2 progresses; not DONE (binary wiring + proof-serving REST surface still open). Loop CONTINUE.
