# Next Work Package

## Step: Serve computed inclusion proofs over HTTP from the local mirror

## Goal
Close the first half of M2's remaining Verify bar by serving an `inclusion` proof — computed
from the local `SQLiteFetcher` tiles via `InclusionProofFromTiles`, never re-hitting the hub — at a
per-hub HTTP endpoint, so a client can fetch a verifiable inclusion proof for an `iscc_id` and check
it against the hub's own `IsccLogInclusionProof`.

## Scope
- **Create**: `internal/proofserve/handler.go` — a new leaf package exposing
  `Handler(st *store.Store, hubID int64) http.Handler` that serves
  `GET /inclusion?iscc_id=<id>` (and optionally `&index=<n>`) by resolving the leaf seq, building the
  proof from mirrored tiles, and returning JSON. Plus `internal/proofserve/handler_test.go`.
- **Modify**: `cmd/iscc-monitor/main.go` — mount the proof handler per hub on the same mux, under the
  hub's `/<origin>/` subtree, next to the existing `tilesserve.Handler` mirror routes (reuse the
  existing `hubRoute` list; no new store lookup).
- **Reference**:
  - `/workspace/iscc-monitor/internal/logclient/proofbuilder.go` — `InclusionProofFromTiles(ctx, fetch TileFetcher, index, size uint64)` and the `TileFetcher` shape (== `SQLiteFetcher.ReadTile`).
  - `/workspace/iscc-monitor/internal/logclient/inclusioncheck.go` — `InclusionEvidence` JSON shape (`{type, checkpoint, treeSize, leafIndex, inclusionProof}`, proof base64-**Std**) to mirror for the response body.
  - `/workspace/iscc-monitor/internal/store/iscc_index.go` — `SeqsForISCCID(ctx, hubID, isccID) ([]uint64, error)` (one-to-many, ADR-0008).
  - `/workspace/iscc-monitor/internal/store/fetcher.go` — `SQLiteFetcher.ReadTile` (pass straight into `TileFetcher`).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `FollowState(ctx, hubID) (FollowState, error)` → `LastSize` is the latest accepted tree size to prove against.
  - `/workspace/iscc-monitor/internal/tilesserve/handler.go` + `/workspace/iscc-monitor/internal/metricshttp/handler.go` — the status-mapping / mounting style to match (400/404/405/500, octet vs JSON content type, `StripPrefix` mount).
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/log_tree.py:208` `inclusion_evidence` — the hub's evidence-member field names + base64-Std proof encoding to be byte-compatible with.

## Not In Scope
- The `consistency` and `entries`-as-proof endpoints — separate later slices (this step is inclusion only).
- CORS, caching, conditional GET (ETag/If-None-Match), and `/healthz` — explicitly deferred per the M3 split; do **not** add them here.
- Re-verifying the embedded checkpoint signature/root, or returning the signed checkpoint body (serving the proof hashes + leafIndex + treeSize is enough for this slice; the client refetches `/checkpoint` from the mirror).
- Wiring the proof endpoint back into `PollHub`/`follower` as a self-check caller (the inbound `verify-for-me` consumer is a later concern).
- Touching `internal/store` schema, `internal/logclient`, `internal/tilesserve`, `go.mod`/`go.sum`, or the `CheckpointAt` `ORDER BY` issue (that one belongs to the consistency slice that revisits prior-root selection).

## Implementation Notes
- **New leaf package `internal/proofserve`** (mirrors `tilesserve`/`metricshttp`): it depends on
  `internal/store` and `internal/logclient`, never the reverse, so `net/http` stays out of those
  closures. Start the file with a docstring stating its purpose and the "computed from the local
  mirror, never re-hitting the hub" invariant.
- **Handler signature**: `Handler(st *store.Store, hubID int64) http.Handler`. Inside, build
  `f := store.SQLiteFetcher{Store: st, HubID: hubID}` and pass `f.ReadTile` as the
  `logclient.TileFetcher` — the signatures are byte-identical (learnings: "`TileFetcher` signature is
  byte-identical to `store.SQLiteFetcher.ReadTile`"), so no adapter is needed.
- **Request flow** for `GET /inclusion`:
  1. Reject non-GET with 405 (match `tilesserve`).
  2. Read `iscc_id` query param; empty → 400.
  3. `size := fs.LastSize` from `store.FollowState(ctx, hubID)`; if `size == 0` (no accepted
     checkpoint yet) → 404 (nothing to prove against). This is the accepted tree the monitor
     vouches for — do **not** parse a size out of the raw checkpoint bytes in this slice.
  4. `seqs, err := st.SeqsForISCCID(ctx, hubID, isccID)`. Empty list → 404. An `index` query param,
     when present, must be one of `seqs` (else 400/404); when absent, default to `seqs[0]`
     (document this choice — `iscc_id → seq` is one-to-many per ADR-0008, so picking the first
     committed seq is the deterministic default; do not silently prove a seq that is not in the list).
  5. Guard `leafIndex < size` before building (it should hold, since the seq was committed into the
     accepted tree, but a stale/racing size could make it false → 404, never a 500/panic).
  6. `proof, err := logclient.InclusionProofFromTiles(ctx, f.ReadTile, leafIndex, size)`.
     A wrapped `os.ErrNotExist` (a tile not yet mirrored) → 404; any other error → 500.
- **Response body** (JSON, content type `application/json`): reuse the field names from
  `logclient.InclusionEvidence` so the monitor's output is shape-compatible with the hub's
  `IsccLogInclusionProof` and directly feedable to `VerifyInclusionEvidence` — at minimum
  `{"type":"IsccLogInclusionProof","treeSize":<size>,"leafIndex":<leafIndex>,"inclusionProof":[<base64-Std hashes>]}`.
  Encode each proof hash with `base64.StdEncoding` (learnings + iscc-hub `inclusion_evidence` both use
  base64-Std). Omitting `checkpoint` is fine for this slice (see Not In Scope) — but if you include it,
  it must be the raw mirror checkpoint bytes, not a re-derived string.
- **Mount in `main.go`**: in `mirrorHandler` (or a sibling), for each `hubRoute` also register the
  proof handler under the same `"/" + Origin + "/"` subtree. Cleanest: build a small per-hub
  `*http.ServeMux` that routes `/inclusion` to `proofserve.Handler(st, r.HubID)` and everything else to
  the existing `tilesserve.Handler`, then `StripPrefix`-mount that combined mux at the prefix — so
  `GET /sb0.iscc.id/log/inclusion?iscc_id=…` reaches the proof handler and `/sb0.iscc.id/log/checkpoint`
  still reaches the mirror. Keep the trailing-slash + `StripPrefix` discipline from learnings (the
  trailing slash arms `http.ServeMux` subtree matching).
- **Correctness rules in play**: (a) ADR-0008 — `iscc_id → seq` is one-to-many and schema-agnostic; do
  not interpret the ISCC-ID, just index by seq and prove the chosen leaf. (b) The proof/verify purity
  rule does not bind this package (it imports `net/http` + `store`), but keep `logclient` and `store`
  import-clean as they are — `proofserve` depends on them, never the reverse.
- **Oracle gate APPLIES** (this serves RFC-6962 inclusion crypto): the test must assert the served
  proof is real ground truth, not just a 200. Build a verified mirror fixture (reuse the
  `testonly.Tree` / `buildVerifiedMirror` / seeded-`SQLiteFetcher` pattern the follower inclusion test
  already uses), then assert the served `inclusionProof` base64-decodes to bytes that
  `proof.VerifyInclusion(rfc6962.DefaultHasher, leafIndex, size, tree.LeafHash(leafIndex), got, tree.HashAt(size))`
  accepts — and that a wrong-leaf served proof would fail it. A green-but-wrong handler (e.g. one
  serving an empty or constant proof) must not pass.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestInclusion ./internal/proofserve` passes (uncached: add `-count=1`), including:
  a happy-path served proof that `proof.VerifyInclusion` accepts against the fixture tree's root;
  `iscc_id` not in `iscc_index` → 404; missing `iscc_id` param → 400; non-GET → 405; `LastSize == 0`
  (no accepted checkpoint) → 404.
- `go test -run TestMirror ./cmd/iscc-monitor` still passes (the proof mount does not break existing
  mirror routing): `/sb0.iscc.id/log/checkpoint` → 200, `/sb0.iscc.id/log/inclusion?iscc_id=<seeded>`
  → 200 JSON, `/metrics` → 200.
- `git diff --quiet HEAD -- internal/store/schema.sql go.mod go.sum` exits 0 (no schema/dep change).
- `go list -deps ./internal/store | grep -E 'internal/proofserve|net/http'` is empty and
  `go list -deps ./internal/logclient | grep internal/proofserve` is empty (the new package depends on
  them, never the reverse).

## Done When
`advance` has a new `internal/proofserve` package serving an oracle-verified inclusion proof from the
local mirror, mounted per hub in the binary, with every Verification check passing.
