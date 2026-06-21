## 2026-06-21 — Serve computed consistency proofs over HTTP from the local mirror

**Done:** Added `GET /consistency?from=<n>` to the per-hub proof surface. `serveConsistency` reads the
prior root via `store.CheckpointAt(hubID, from)` and the accepted size via `FollowState.LastSize`,
builds the RFC-6962 consistency proof from the hub's mirrored tiles via
`logclient.ConsistencyProofFromTiles(ctx, f.ReadTile, from, size)` (never re-hitting the hub), and
serves it as JSON (`{type:"IsccLogConsistencyProof", firstSize, secondSize, consistencyProof:[base64-Std]}`).
Also fixed the open `CheckpointAt` unordered-`LIMIT 1` issue by adding `ORDER BY rowid` so the
first-recorded (prior accepted) root is returned deterministically when two rows share a `tree_size`.

**Files changed:**
- `internal/proofserve/handler.go`: split the inner `Handler` closure into a path switch
  (`/inclusion` → `serveInclusion`, `/consistency` → `serveConsistency`, else 404); added
  `serveConsistency`, the `ConsistencyEvidence` response struct, and `writeConsistency`; updated the
  package doc.
- `internal/store/checkpoints.go`: appended `ORDER BY rowid` to `CheckpointAt`'s query and updated the
  method doc to state the ordering (deterministic prior-root selection on a fork).
- `cmd/iscc-monitor/main.go`: `hubHandler` now mounts `proofserve.Handler` at BOTH `/inclusion` and
  `/consistency` exact paths (one shared handler, internal path switch); doc comment extended to list
  both proof routes.
- `internal/proofserve/consistency_test.go` (new): the oracle-gated consistency conformance cases.
- `internal/store/checkpoints_test.go`: added `TestCheckpointAtDeterministicOnFork`.

**Verification:** `mise run check` → green (build + vet + all 13 packages `ok`, incl. `notecheck`
oracle). Per-criterion:
- `gofmt -l .` empty.
- `go test -run TestConsistency -count=1 ./internal/proofserve` PASS — served proof for `from ∈
  {1,200,255,256,299}` (straddling the 256-leaf tile boundary, `larger=300`) base64-Std-decodes to
  bytes `proof.VerifyConsistency(hasher, from, larger, proof, fromRoot, largerRoot)` ACCEPTS,
  byte-equals `tree.ConsistencyProof(from, larger)`, and a wrong-prior-root negative is REJECTED.
- `go test -run TestCheckpointAt -count=1 ./internal/store` PASS — `TestCheckpointAtDeterministicOnFork`
  confirms the first-recorded (lowest-rowid) row is returned when two roots share a `tree_size`.
- Status checks (in proofserve test) PASS: missing/non-numeric `from` → 400; `from > LastSize` → 400;
  `LastSize == 0` → 404; unknown `from` (no checkpoint row) → 404; non-GET → 405; `from == 0` and
  `from == LastSize` → 200 with empty `consistencyProof`.
- `go test -run TestMirror -count=1 ./cmd/iscc-monitor` PASS (existing mirror/inclusion routing intact).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` → exit 0 (no schema/dep change).
- `go list -deps ./internal/store | grep -E 'proofserve|net/http'` empty (store stays a leaf).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` OK (WASM-shared verifier seam untouched).

**Oracle gate (RFC-6962 consistency crypto) — reviewer-reproducible mutation proof (both reverted):**
(1) corrupting `proof[0][0] ^= 0xff` before serving → `TestConsistencyServedProofVerifies` FAILS on
BOTH the byte-equality assert AND `proof.VerifyConsistency` ("does not verify"). (2) reversing
`CheckpointAt`'s `ORDER BY rowid` to `DESC` → `TestCheckpointAtDeterministicOnFork` FAILS (returns the
contradicting row). A green-but-wrong handler or a non-deterministic prior-root read cannot ship.
`notecheck`/`derive_vkey.py` correctly N/A — the consistency response carries no checkpoint and no
signature is re-parsed (the client refetches `/checkpoint`; `AcceptCheckpoint` owns signature/root).

**Next:** The `entries` proof/read endpoint — the third remaining M2 proof surface (serve the raw entry
bundle / record bytes for a leaf range from the mirror). That slice can reuse this `/consistency` +
`/inclusion` routing pattern in `hubHandler`'s switch, and is the natural place to start owning the
M3-deferred CORS / caching / conditional-GET / `/healthz` that this slice intentionally skipped.

**Notes:**
- **`from == 0` degenerate case needs a guard around `CheckpointAt`.** There is no checkpoint at size 0
  by construction, so `CheckpointAt(0)` returns `found == false`. `next.md`'s status table says
  "unknown `from` → 404" but separately says `from == 0` must be a 200 with an empty proof. I resolved
  this by skipping the `CheckpointAt` row requirement only when `from == 0` (the empty-tree prior); for
  `from == LastSize` and all other `from > 0`, the checkpoint row IS required (buildMirror records the
  accepted-size row, so the same-size roundtrip 200s). This is the only place the literal status table
  and the degenerate-boundary note conflicted; the chosen reading matches the M ≤ N RFC-6962 semantics.
- **`hubHandler` mount: two exact-path handles share one `proofserve.Handler`.** Because the inner
  `hubHandler` mux already routes `/` → tilesserve as a subtree, `/consistency` would otherwise fall
  through to tilesserve (404). Adding `mux.Handle("/consistency", proofs)` makes http.ServeMux's
  most-specific match send it to proofserve, whose internal switch dispatches. Verified by
  `TestMirror*` (existing mirror routing intact) and the proofserve route tests.
- **Scope-clean:** 3 production files (main.go is comment + 1 routing line; checkpoints.go is the
  `ORDER BY rowid` + doc; handler.go is the route + serve fn). Nothing from `## Not In Scope` added —
  no `entries` endpoint, no CORS/ETag/healthz, no in-handler re-verify, follower/freeze path untouched,
  the other open `normal` issues left alone. go.mod/go.sum/schema.sql byte-unchanged.
- **CI/push:** changes are committed on `develop` (loop branch); CI gates on push. `main` is untouched.
