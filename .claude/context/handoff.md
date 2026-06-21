## 2026-06-21 — Review of: Wire tilesserve.Handler into the binary with a per-hub mirror router

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` mounted the already-built `tilesserve.Handler` per-hub on the monitor binary's
single HTTP server, so each followed hub's mirrored tlog-tiles artifacts are now served at its
canonical `/<origin>/...` prefix alongside `/metrics`. The change is one production file + its test,
exactly as `next.md` scoped it; routing is mutation-proven non-vacuous and every Verification criterion
passes independently.

**Verification:**
- [x] `mise run check` green — all 12 packages `ok` (build + vet + test); `gofmt -l .` empty.
- [x] `go test -run TestMirror -count=1 ./cmd/iscc-monitor` PASS (uncached) — all 4 subtests:
  `GET /sb0.iscc.id/log/checkpoint` → 200 byte-equal to the seeded BLOB; `/sb0.iscc.id/log/tile/0/000`
  (unmirrored under prefix) → 404; `/sb0.iscc.id/checkpoint` (missing `/log`) → 404; `/metrics` → 200.
- [x] `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exit 0 — no schema/dep change.
- [x] `go list -deps ./internal/store | grep -E 'internal/tilesserve|net/http'` empty — store stays a
  leaf (closure is `internal/tiles`+self only); the dep is binary→tilesserve→store, never the reverse.
- [x] Non-vacuousness re-proven by reviewer: dropping the trailing slash from the mount prefix
  (`"/"+r.Origin` instead of `"/"+r.Origin+"/"`) fails the 200-byte-equal subtest with 404; reverted →
  green. A green-but-misrouted router cannot ship. Files restored byte-identical to HEAD.
- [x] Scope discipline — only `cmd/iscc-monitor/main.go` (1 production file) + `main_test.go` touched.
  Every `## Not In Scope` item honored: no proof/CORS/cache/healthz endpoints, `tilesserve.Handler`
  routing/signature untouched, `store`/`follower`/`schema.sql`/`go.mod`/`go.sum` byte-identical, no
  `HubID`→origin store lookup (origin re-derived via `logclient.Origin` as instructed).
- [x] Gate-integrity scan over unpushed commits (`origin/develop..HEAD`) — no `//nolint`/`t.Skip`/
  build-tag exclusion/swallowed error/deleted assertion. The two `return nil, nil, ...` edits are the
  legitimate 3-value-signature widening, not removed assertions.
- [x] WASM purity invariant holds (`GOOS=js GOARCH=wasm go build ./internal/didweb` → OK).

**Issues found:** (none) — the 6 open `normal` issues are orthogonal (none of
`checkpoints.go`/`accept.go`/`follower.go`/`ingest.go`/`fetcher.go`/`tiles.go`/`consistency.go` was
touched this slice) and remain valid; none was resolved here, so none deleted.

**Next:** The M2 proof-computing `verify-for-me` REST surface (`inclusion`/`consistency`/`entries`-as-
proofs) building on this inbound transport — it consumes `ConsistencyProofFromTiles` /
`InclusionProofFromTiles` / `VerifyInclusionEvidence` over the same `SQLiteFetcher`, and (per the M3
split) will own CORS, caching, conditional GET, and healthz that this slice intentionally deferred.
Consider opportunistically resolving the `CheckpointAt` unordered-`LIMIT 1` issue if the proof slice
revisits prior-root selection.

**Notes:**
- **Oracle/conformance gate correctly N/A** for this slice — pure HTTP wiring of existing packages, no
  signature/RFC-6962/Merkle/did:web/fsck path introduced (`tilesserve` serves opaque BLOBs). It re-arms
  at the proof-computing `verify-for-me` slice.
- **Load-bearing wiring detail (now in learnings):** the trailing slash on the mount prefix arms
  `http.ServeMux` subtree matching, and `StripPrefix` down to the single leading slash gives the handler
  exactly the `/checkpoint`-style suffix it trims. Origin must be the full `<domain>/log` — a request
  missing `/log` 404s (asserted + reviewer-confirmed).
- Test totals: **171 `func Test`** (up from 170 — the new `TestMirrorRouter` + 4 subtests).
- Pushed to `origin/develop` on PASS. A human merges `develop → main` via the CI-gated PR.
