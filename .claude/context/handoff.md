## 2026-06-21 — Serve computed inclusion proofs over HTTP from the local mirror

**Done:** Added a new leaf package `internal/proofserve` exposing `Handler(st *store.Store, hubID
int64) http.Handler` that serves `GET /inclusion?iscc_id=<id>` (optionally `&index=<n>`) by resolving
the leaf seq via `SeqsForISCCID`, building an RFC-6962 inclusion proof from the hub's mirrored tiles
via `logclient.InclusionProofFromTiles` (never re-hitting the hub) against the accepted tree size
(`FollowState.LastSize`), and returning JSON shaped like the hub's `IsccLogInclusionProof`. Mounted it
per hub on the binary's combined mux next to the `tilesserve` mirror routes.

**Files changed:**
- `internal/proofserve/handler.go` (new): the proof-serving leaf package; resolves iscc_id → seq
  (schema-agnostic, ADR-0008), defaults to `seqs[0]` when `index` is absent, base64-Std-encodes the
  proof hashes into `logclient.InclusionEvidence`'s field shape. Status mapping: non-GET 405, non-
  `/inclusion` 404, missing `iscc_id` 400, bad `index` 400, no accepted checkpoint / unknown iscc_id /
  uncovered leaf / unmirrored tile 404, else 500.
- `internal/proofserve/handler_test.go` (new): oracle-gated conformance test over a verified 300-leaf
  mirror; plus the 400/404/405/no-checkpoint negatives.
- `cmd/iscc-monitor/main.go`: new `hubHandler` combines `proofserve.Handler` (`/inclusion`) and
  `tilesserve.Handler` (everything else) behind a per-hub `*http.ServeMux`; `mirrorHandler` now strips
  the prefix WITHOUT its trailing slash (load-bearing — see Notes) so the inner mux gets leading-slash
  paths.
- `cmd/iscc-monitor/main_test.go`: added `TestMirrorInclusionRoute` (binary-level proof mount, served
  proof re-verified) and confirmed the existing `TestMirrorRouter` still passes under the strip change.

**Verification:** `mise run check` → green (build + vet + all 13 packages `ok`); `gofmt -l .` empty.
- `go test -run TestInclusion -count=1 ./internal/proofserve` PASS — happy-path proof verifies via
  `proof.VerifyInclusion` against the tree root for leaves `{0,5,255,256,260,299}` (both tile-boundary
  sides); unknown iscc_id → 404; missing param → 400; non-GET → 405; `LastSize == 0` → 404.
- `go test -run TestMirror -count=1 ./cmd/iscc-monitor` PASS — `/sb0.iscc.id/log/inclusion?iscc_id=…`
  → 200 verifiable JSON, `/sb0.iscc.id/log/checkpoint` → 200, `/metrics` → 200; old mirror routing
  (checkpoint byte-equal, missing-`/log` 404) intact.
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` → exit 0 (no schema/dep change).
- `go list -deps ./internal/store | grep -E 'proofserve|net/http'` empty; `go list -deps
  ./internal/logclient | grep proofserve` empty — `proofserve` depends on both, never the reverse.
- WASM purity (`GOOS=js GOARCH=wasm go build ./internal/didweb`) OK.

**Next:** The `consistency` endpoint slice — serve `ConsistencyProofFromTiles(smaller=prev,
larger=LastSize)` over the same per-hub mux (`GET /consistency?from=<n>`), reusing this routing. That
slice is also the natural place to revisit the `CheckpointAt` unordered-`LIMIT 1` prior-root selection
(still open) and to start owning the M3-deferred CORS / caching / conditional-GET / healthz this slice
intentionally skipped. Wiring a `verify-for-me` inbound consumer of `/inclusion` is a separate later
concern.

**Notes:**
- **Oracle gate APPLIES and is mutation-proven non-vacuous.** Two mutations (both reverted): (1) serve
  an empty proof → `TestInclusionServedProofVerifies` FAILS; (2) build the proof for `leafIndex+1` →
  FAILS with root-mismatch / wrong-proof-size across the boundary leaves. The test's three independent
  paths (the `testonly.Tree` prover, `InclusionProofFromTiles` inside the handler, and
  `proof.VerifyInclusion`) are non-circular; a green-but-wrong handler cannot ship. `notecheck` /
  `derive_vkey.py` correctly N/A (no signature/did:web path — the checkpoint is not re-parsed or even
  served here; `AcceptCheckpoint` owns that).
- **Load-bearing mount detail (candidate for learnings):** because `hubHandler` is itself an
  `http.ServeMux`, `StripPrefix` must leave the leading slash on the suffix (strip `"/"+Origin`, NOT
  `"/"+Origin+"/"`) or the inner mux 301-redirects every request (`/checkpoint` without a leading slash
  → "Moved Permanently"). The prior slice mounted `tilesserve.Handler` directly (which tolerates a
  slash-less path via `TrimPrefix`), so stripping the full trailing-slash prefix worked; a nested
  ServeMux does not tolerate it. The mount prefix itself keeps its trailing slash for subtree matching.
  Verified: the pre-existing `TestMirrorRouter` (checkpoint byte-equal + missing-`/log` 404) still
  passes under the strip change.
- **Default seq choice:** with `index` absent the handler proves `seqs[0]` (the first/lowest committed
  seq), documented in `selectSeq` — `iscc_id → seq` is one-to-many (ADR-0008), so this is the
  deterministic default; an explicit `index` must be one of the committed seqs (else 400), never a
  silently-substituted one.
- **`checkpoint` field omitted** from the response (per Not In Scope) — the client refetches
  `/checkpoint` from the mirror. The served `{type, treeSize, leafIndex, inclusionProof}` is still
  shape-compatible with `logclient.InclusionEvidence` / `VerifyInclusionEvidence`.
- Test totals: +8 `func Test` (6 in proofserve, 1 added in cmd, plus the existing TestMirrorRouter kept).
