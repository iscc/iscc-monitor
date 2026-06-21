# Next Work Package

## Step: Serve computed consistency proofs over HTTP from the local mirror

## Goal
Add `GET /consistency?from=<n>` to the per-hub proof surface: build an RFC-6962 consistency proof
relating the prior root at size `from` to the monitor's accepted root at `LastSize`, sourced entirely
from the hub's mirrored hash tiles (never re-hitting the hub). Inclusion is already served; consistency
is the next half of M2's "computed proofs served from the local store" Verify bar, and it lets a client
check its own prior `(size, root)` against the monitor's mirrored tree.

## Scope
- **Modify**: `internal/proofserve/handler.go` — route `GET /consistency?from=<n>` alongside the
  existing `/inclusion`: split the inner `Handler` closure into a path switch (`/inclusion` →
  `serveInclusion`, `/consistency` → `serveConsistency`, else 404). `serveConsistency` reads the prior
  root via `store.CheckpointAt(hubID, from)` and the accepted root via `store.CheckpointAt(hubID,
  LastSize)`, builds the proof via `logclient.ConsistencyProofFromTiles(ctx, f.ReadTile, from,
  LastSize)`, and writes JSON. Add the `internal/proofserve/handler_test.go` consistency cases.
- **Modify**: `internal/store/checkpoints.go` — fix the open `CheckpointAt` unordered-`LIMIT 1` issue:
  add `ORDER BY rowid` so the prior *accepted* root (recorded first → lowest rowid) is returned
  deterministically, never a later contradicting-evidence row at the same `tree_size`. Update the
  method doc to state the ordering. (test file `internal/store/checkpoints_test.go` is doc/test, not a
  production-file count.)
- **Modify**: `cmd/iscc-monitor/main.go` — comment-only: extend the `hubHandler` doc comment so it
  lists `/inclusion` AND `/consistency` as the per-hub proof routes. The route dispatch lives entirely
  inside `proofserve.Handler` (the existing `mux.Handle("/inclusion", proofserve.Handler(...))` already
  forwards the whole proof subtree once `/consistency` is added to the inner switch — but verify the
  mount: if `proofserve.Handler` is mounted only at the exact `/inclusion` path in `hubHandler`'s mux,
  change that mount so `/consistency` also reaches it, e.g. mount `proofserve.Handler` at a shared proof
  prefix or add a second `mux.Handle("/consistency", ...)` line. Adjust as the actual mount requires;
  keep this to the routing line(s) + doc.)
- **Reference**:
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — the inclusion sibling to mirror:
    method-gate, path check, `selectSeq`/`parseUint`, `writeEvidence` shape, leaf-package purity posture.
  - `/workspace/iscc-monitor/internal/proofserve/handler_test.go` — `buildMirror` (300-leaf
    `testonly.Tree` fixture), `getEvidence`, `decodeProof`, and the `proof.Verify*` ground-truth
    assertion pattern to reuse.
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder.go` —
    `ConsistencyProofFromTiles(ctx, fetch, smaller, larger)`; `smaller==0`/`smaller==larger` yield a
    nil proof WITHOUT touching the fetcher.
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `CheckpointAt`, `FollowState`.
  - `/workspace/iscc-monitor/cmd/iscc-monitor/main.go` — `hubHandler` mount of `proofserve.Handler`.
  - `/workspace/iscc-monitor/cauldron/iscc-hub/specs/iscc-log.md` §10.2 — consistency proof relates
    `R_M` to `R_N` for `M ≤ N`; if it does not verify, treat as split-view evidence.
  - `/workspace/iscc-monitor/cauldron/tessera/client/client.go` — `ProofBuilder.ConsistencyProof`
    (the original `ConsistencyProofFromTiles` ports).

## Not In Scope
- The `entries` proof/read endpoint (the third remaining M2 proof surface) — a separate later slice.
- CORS, caching, ETag / conditional-GET, and `/healthz` — still deferred (M3-owned); do not add them.
- Re-verifying the served consistency proof inside the handler, or re-parsing the checkpoint signature
  (that is `AcceptCheckpoint`'s job); the handler computes and serves, the client verifies.
- Wiring a `consistency`-evidence struct into the follower's `CheckEquivocation` freeze path — that
  trigger already builds its own proof; this is a read-only serving endpoint only.
- The other open `normal` issues (frozen-hubs-still-advance, `AcceptCheckpoint` context reuse,
  tile-writer `p` vocabulary, deep `AdvanceAccepted`, `CheckConsistency` collapse) — not this slice.

## Implementation Notes
- **Route shape.** `from` is the smaller (prior) tree size the client already trusts; the larger is the
  monitor's accepted `FollowState.LastSize`. So the call is
  `logclient.ConsistencyProofFromTiles(ctx, f.ReadTile, from, size)` with `size = fs.LastSize`. Mirror
  the inclusion handler: a method-gate (405), a path switch, and `serveConsistency` owning the flow +
  status mapping. Reuse `parseUint` for `from`.
- **Status mapping (match the inclusion handler's posture):** non-GET → 405; unmatched path → 404;
  missing/non-numeric `from` → 400; no accepted checkpoint (`LastSize == 0`) → 404; `from > LastSize`
  (a future/over-large prior — RFC-6962 requires `M ≤ N`) → 400; `from` has no recorded checkpoint row
  (`CheckpointAt` `found == false`) → 404; a tile not yet mirrored (wrapped `os.ErrNotExist`) → 404;
  any other read/build error → 500.
- **Degenerate boundaries.** `ConsistencyProofFromTiles` returns a **nil/empty** proof for `from == 0`
  AND `from == LastSize` (`proof.Consistency` yields no node IDs and never touches the fetcher) — serve
  a **200 with an empty `consistencyProof` array** (a valid degenerate proof), do NOT 400 them. `from ==
  LastSize` still needs the accepted-size checkpoint row to exist (for `secondSize`/roundtrip), so it is
  a 200 only when `LastSize > 0`.
- **Response shape.** There is NO hub-defined `IsccLogConsistencyProof` JSON (unlike inclusion — §10.2:
  the consistency proof is verifier-*computed*, not served by the hub). Define a small local response
  struct in `proofserve`, e.g. `{type:"IsccLogConsistencyProof", firstSize:<from>, secondSize:<LastSize>,
  consistencyProof:[base64-Std…]}`. base64-**Std** encode each hash (matching the inclusion encoding and
  `iscc_hub` convention). Keep `writeEvidence`'s drop-the-write-error-after-200 posture; a marshal of a
  fixed-shape struct of strings/uints cannot fail for content reasons.
- **Correctness rule (learnings — oracle gate).** `ConsistencyProofFromTiles` IS RFC-6962 crypto, so the
  **oracle gate APPLIES**; the test must be non-vacuous. Assert the served bytes
  `proof.VerifyConsistency(rfc6962.DefaultHasher, from, larger, served, fromRoot, largerRoot)` ACCEPTS —
  note the arg order `(hasher, size1, size2, proof, root1, root2)`: `proof` precedes the two roots,
  UNLIKE `VerifyInclusion` where `leafHash` precedes `proof`. Use the existing `buildMirror` 300-leaf
  fixture; get the two roots from `tree.HashAt(from)` and `tree.HashAt(larger)`, the ground-truth proof
  from `tree.ConsistencyProof(from, larger)`, and assert the served proof BYTE-EQUALS it AND verifies.
  Add a sharp negative: the proof for `(from, larger)` must NOT verify against a wrong `from'` root.
- **`CheckpointAt` fix.** Append `ORDER BY rowid` to the `SELECT root, raw … WHERE hub_id=? AND
  tree_size=?` query. Rationale (issues.md): after a fork freeze two rows share a `tree_size`; the prior
  accepted row was recorded first (lower rowid), the contradicting-evidence row later, so `ORDER BY
  rowid` deterministically returns the prior accepted root — making both this handler's prior-root read
  and `follower`'s fork re-detection deterministic. Keep `found=false` on no row, the `[]byte` return
  (store stays a leaf — no `logclient` import), and the existing error wraps. Note the test fixture
  records `CheckpointRecord{Raw: []byte("checkpoint")}` at the accepted size, so `CheckpointAt(LastSize)`
  finds a row — good; for `from < LastSize` the test must also record a checkpoint row at `from`'s size
  (or pick `from` values for which `buildMirror` recorded one). `buildMirror` currently records ONLY the
  one checkpoint at `size`; extend the fixture (or the test) to record a prior checkpoint at each tested
  `from` with the correct `tree.HashAt(from)` root, so `serveConsistency`'s `CheckpointAt(from)` succeeds.
- **Leaf-package purity.** `proofserve` already imports `store` + `logclient` + `net/http`; do NOT add a
  `net/http`/`proofserve` edge into `store` or `logclient`. No new dep — `ConsistencyProofFromTiles`,
  `CheckpointAt`, and `proof.VerifyConsistency` are all already in the module closure, so
  `go.mod`/`go.sum`/`schema.sql` stay byte-unchanged.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass, `gofmt -l .`
  empty).
- `go test -run TestConsistency -count=1 ./internal/proofserve` passes — the served proof for sampled
  `(from, larger)` pairs straddling the 256-leaf tile boundary (e.g. `from ∈ {1,200,255,256,299}`,
  `larger == 300`) base64-Std-decodes to bytes `proof.VerifyConsistency` ACCEPTS against the tree's two
  roots, byte-equals `tree.ConsistencyProof(from, larger)`, and a wrong-`from` negative is REJECTED.
- `go test -run TestCheckpointAt -count=1 ./internal/store` passes — `CheckpointAt` returns the
  first-recorded (lowest-rowid) row deterministically when two rows share a `tree_size`.
- Status checks pass (assert in the proofserve test): missing/non-numeric `from` → 400; `from > LastSize`
  → 400; `LastSize == 0` → 404; unknown `from` size (no row) → 404; non-GET → 405; unmatched path → 404;
  `from == LastSize` → 200 with an empty `consistencyProof`.
- `go test -run TestMirror -count=1 ./cmd/iscc-monitor` still passes (existing mirror/inclusion routing
  intact).
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).
- `go list -deps ./internal/store | grep -E 'proofserve|net/http'` is empty (store stays a leaf).

## Done When
`advance` has the `/consistency` endpoint serving an oracle-verified RFC-6962 consistency proof from the
local mirror, the deterministic `CheckpointAt` `ORDER BY rowid` fix landed, and every Verification check
passes.
