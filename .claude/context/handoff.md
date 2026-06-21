## 2026-06-21 — Review of: CORS middleware on every public GET (M3 cross-cutting HTTP slice — part 1)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added `internal/corsmw` — a stdlib-only middleware leaf (`Handler(next
http.Handler) http.Handler`) that sets `Access-Control-Allow-Origin: *` on every response and
short-circuits `OPTIONS` preflights with `204` — and wrapped the assembled public mux once in
`buildMux` (the lone route convergence point). Scope is exactly 2 production files + 1 test; every
`next.md` Verification criterion holds under independent uncached re-runs; the trust root is untouched
and stays green.

**Verification:**
- [x] `mise run check` green — independently re-ran `go test -count=1 ./...`: all 15 packages `ok`
  (build + vet + test), including every conformance test (`logclient`/`didweb`/`store`).
- [x] `gofmt -l .` empty.
- [x] `go test -count=1 ./internal/corsmw` — PASS (3 table cases; OPTIONS double-guarded: inner `t.Error`s
  if invoked AND outer asserts the `ran` sentinel is false).
- [x] `go test -count=1 -run TestMirror ./cmd/iscc-monitor` — PASS (router/inclusion/entries tests all
  green through the now-wrapped mux; routing + GET responses unchanged).
- [x] `grep -rn "Access-Control" internal/proofserve internal/tilesserve internal/metricshttp
  internal/healthz` — empty (no per-handler duplication; header set only in `corsmw`).
- [x] `git diff HEAD~1..HEAD --quiet -- go.mod go.sum internal/store/schema.sql` — exits 0 (no
  dependency/schema change).
- [x] `go list -deps ./internal/corsmw | grep -E 'internal/(store|logclient)'` — empty; closure is
  `net/http` + stdlib only.
- [x] Scope discipline — exactly `corsmw.go` (new) + `main.go` (1-line wrap + doc) + `corsmw_test.go`;
  nothing from `## Not In Scope` touched (no caching/ETag, no handler edits, no issue drain).
- [x] Gate integrity — scanned all 3 unpushed commits; no `//nolint`/`t.Skip`/build-tag/swallowed-error/
  deleted-assertion. The `_, _ = io.WriteString` in the test is the documented post-status write-drop
  convention, not a gate dodge.
- [x] WASM purity (rides `internal/didweb`, untouched) — `GOOS=js GOARCH=wasm go build ./internal/didweb`
  OK. Oracle/conformance gate correctly N/A (pure HTTP header wiring; no signature/RFC-6962/Merkle/
  did:web/fsck/proof path). CI's independent `notecheck` signature-parity job is unaffected.

**Issues found:** (none) — no new problems; no existing issue resolved by this slice (it was a fresh M3
HTTP-plumbing slice, not a drain). The advance handoff's note that the `proofserve`/`tilesserve` "CORS
out of scope" doc comments could later be tightened is a wording observation, not a real issue — those
comments correctly defer to this middleware and never falsely claim CORS is handled there. Not filed.

**Next:** The SECOND M3 cross-cutting HTTP slice — caching / `Cache-Control` / conditional-GET (`ETag`/
`If-None-Match`/`Last-Modified`). Note the asymmetry flagged by advance: immutable full tiles/bundles
(`is_full=1`, content-addressed) want a long-lived/immutable cache policy while mutable partials and
the size-varying checkpoint/proof surfaces want revalidation — so a blind single wrap won't do; the
handlers likely need to signal cacheability (the store already tracks `is_full`). Scope it as ≤2
production files. Alternatively, start draining a `normal` issue (the growing split-view / frozen-advance
ADR-0006 items are the highest-value), or begin the proof-bundle JSON + verify-for-me arc.

**Notes:**
- No remote-push blockers: no local pre-push hook; `origin/develop` is the upstream. Pushing on PASS.
- The middleware ordering is genuinely correct: `Allow-Origin` is set on the `http.ResponseWriter`
  header map BEFORE `next.ServeHTTP`, so it survives the inner handler's `WriteHeader` (via `http.Error`
  or first body write) which freezes the map — the `inner non-200 still carries Allow-Origin` test pins
  this against a real `http.Error(..., 404)`.
- Wildcard `*` with no `Allow-Credentials` is the deliberate, correct policy for public read-only data
  (browsers reject `*` + credentials). The OPTIONS short-circuit is required because the GET-only inner
  handlers would 405 a preflight and block the browser's real GET.
- v1 milestone status unchanged by this slice: M1 done, M2 proof surface complete (3-of-3), M3 in
  progress (CORS landed; caching, verify-for-me, dashboard, log browser still open); WASM verifier + OTS
  unstarted. Not DONE — M3/WASM/OTS remain. CONTINUE.
