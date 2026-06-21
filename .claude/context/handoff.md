## 2026-06-21 — Review of: Serve computed inclusion proofs over HTTP from the local mirror

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` added a new leaf package `internal/proofserve` serving `GET
/inclusion?iscc_id=<id>[&index=<n>]`, which resolves the leaf seq via `SeqsForISCCID`, builds an
RFC-6962 inclusion proof from the hub's mirrored tiles via `InclusionProofFromTiles` (never re-hitting
the hub) against the accepted tree size (`FollowState.LastSize`), and returns JSON shaped like the
hub's `IsccLogInclusionProof`. It is mounted per hub behind a nested `*http.ServeMux` next to the
existing `tilesserve` mirror routes. The diff is exactly 2 production files (well within scope), every
Verification check passes, and the oracle gate is reviewer-mutation-proven non-vacuous.

**Verification:**
- [x] `mise run check` green — build + vet + all 13 packages `ok` (incl. `cmd/notecheck` oracle).
- [x] `gofmt -l .` empty — no formatting failures.
- [x] `go test -run TestInclusion -count=1 ./internal/proofserve` PASS — served proof verifies via
  `proof.VerifyInclusion` against the tree root for leaves `{0,5,255,256,260,299}` (both tile-boundary
  sides), wrong-leaf re-label rejected; unknown iscc_id → 404; missing param → 400; non-GET → 405;
  `LastSize == 0` → 404; unmatched path → 404.
- [x] `go test -run TestMirror -count=1 ./cmd/iscc-monitor` PASS — `/sb0.iscc.id/log/inclusion?iscc_id=…`
  → 200 verifiable JSON, `/sb0.iscc.id/log/checkpoint` → 200 (static mirror), `/metrics` → 200; prior
  mirror routing intact.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` → exit 0 (no schema/dep change).
- [x] Dep direction: `go list -deps ./internal/store | grep -E 'proofserve|net/http'` empty;
  `go list -deps ./internal/logclient | grep proofserve` empty — `proofserve → {store, logclient}`,
  never the reverse; `net/http` stays out of those closures.
- [x] WASM purity: `GOOS=js GOARCH=wasm go build ./internal/didweb` OK (untouched leaf).
- [x] **Oracle gate (RFC-6962 inclusion crypto) reviewer-verified non-vacuous** — two reverted
  mutations of `handler.go`: (1) serve `proof = nil` → `TestInclusionServedProofVerifies` FAILS
  ("wrong proof size 0, want N"); (2) build for `leafIndex+1` → FAILS (wrong-leaf verifies / 500 at
  last leaf). A green-but-wrong handler cannot ship. `notecheck`/`derive_vkey.py` correctly N/A — the
  checkpoint signature is never re-parsed or served here (response omits `checkpoint`; the client
  refetches `/checkpoint`; `AcceptCheckpoint` owns signature/root). CI `notecheck` parity job
  unchanged + green at the last-pushed SHA.

**Issues found:** (none new). Scope-clean: exactly 2 production files (`internal/proofserve/handler.go`
new, `cmd/iscc-monitor/main.go`); nothing from `## Not In Scope` was added (no consistency/entries
endpoint, no CORS/ETag/healthz, no signature re-verify, follower untouched). The pre-existing
`CheckpointAt` unordered-`LIMIT 1` issue stays open and was correctly out of scope for this slice — it
belongs to the consistency slice that revisits prior-root selection.

**Next:** The `consistency` endpoint slice — serve `ConsistencyProofFromTiles(smaller=prev,
larger=LastSize)` over the same per-hub nested mux (`GET /consistency?from=<n>`), reusing this routing.
That slice is the natural place to fix the open `CheckpointAt` unordered-`LIMIT 1` prior-root selection
(issues.md) and to start owning the M3-deferred CORS / caching / conditional-GET / healthz this slice
intentionally skipped.

**Notes:**
- **Nested-ServeMux mount detail is load-bearing and correct.** Because `hubHandler` is itself an
  `http.ServeMux` (`/inclusion` → proofserve, `/` → tilesserve), `mirrorHandler` strips only `"/"+
  Origin` (leaving the leading slash) — stripping the full trailing-slash prefix would 301-redirect
  every request. Reviewer confirmed with a standalone mux: exact `/inclusion` wins over the `/`
  subtree; `/checkpoint`, `/tile/...`, and `/inclusion/x` all fall through to tilesserve.
- **Default seq `seqs[0]` is genuinely the lowest committed seq** — `SeqsForISCCID` is `ORDER BY seq`
  ASC, so the documented default is deterministic. An explicit `&index=` must equal a committed seq
  (else 400 via `selectSeq`); `parseUint` rejects non-digits → 400; `leafIndex >= size` is guarded →
  404 (never 500/panic) on a stale/racing size.
- **CI / push:** branch `develop` is 3 commits ahead of `origin/develop` (define-next + advance + this
  review). CI was green at the last-pushed SHA `ae5238b`; the new commits will be CI-gated on push.
  `main` lags at `c59d380` and is merged by a human via a CI-gated PR — never pushed here.
- M2 remains IN_PROGRESS (inclusion proof now served; `consistency`/`entries` proofs still to serve).
  M3 / WASM / OTS not started — loop is not DONE, no human-only decision open → CONTINUE.
