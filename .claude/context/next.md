# Next Work Package

## Step: verify-for-me JSON verdict — `GET /<domain>/log/verify?iscc_id=<id>`

## Advances
M3 — Trust API + dashboard. Closes the nearest unmet M3 Verify criterion:

> `GET /<domain>/log/verify?iscc_id=<known-id>` returns the documented `verify-for-me` JSON verdict
> (hub status + checkpoint `(size, root)` + inclusion result), and a malformed/unknown id returns the
> documented non-verified verdict, never a 5xx.

This is also the explicit `review` handoff `**Next:**` (the proof-bundle / verify-for-me REST surface)
and the #1 item in `state.md`'s convergence-driven order. M3 is 1/4 — this brings it to 2/4 and breaks
the loop's 5-iteration polish/infra streak by attacking a Verify criterion.

## Goal
Add a `verify` route to the existing per-hub `proofserve.Handler` that returns a single self-contained
JSON verdict for an `iscc_id`: the hub's persisted status, the accepted checkpoint `(size, root)`, and
a real RFC-6962 inclusion result (recomputed from the local mirror and Merkle-verified against the
accepted root). This is the monitor's weaker, caller-trusts-the-verdict path; it composes the proof
machinery M2 already built, so the in-browser verifier (later) can reuse the same store/fetcher seam.

## Scope
- **Create**: (none — the route lives in the existing handler file; the test is a new `_test.go`)
- **Modify**:
  - `internal/proofserve/handler.go` — add the `"/verify"` case to the `Handler` path switch, a
    `serveVerify` function, and a `VerifyVerdict` response struct. (Adds `merkle/proof` +
    `merkle/rfc6962` imports — both already used by this package's tests.)
  - `cmd/iscc-monitor/main.go` — mount `/verify` on the per-hub mux in `hubHandler` (one
    `mux.Handle("/verify", proofs)` line next to the existing three exact mounts).
  - `CLAUDE.md` — add the new `GET /<domain>/log/verify?iscc_id=<id>` line to the
    "Running a local dev instance" endpoint list, so the documented HTTP surface stays in sync.
- **Reference** (read before editing — exact paths):
  - `/workspace/iscc-monitor/internal/proofserve/handler.go` — existing `serveInclusion` / `serveEntries`
    flow, `selectSeq`, `parseUint`, `writeEvidence` post-status-write idiom (port the guard ladder from
    `serveInclusion`; reuse the bundle-read pattern from `serveEntries`).
  - `/workspace/iscc-monitor/internal/proofserve/handler_test.go` — the `buildMirror` 300-leaf
    `testonly.Tree` fixture, `leafISCCID`, and the `getEvidence` HTTP-seam test idiom (the new test
    reuses all of these).
  - `/workspace/iscc-monitor/internal/store/checkpoints.go` — `FollowState` (`LastSize`, `Frozen`) and
    `CheckpointAt(ctx, hubID, size)` (the verdict's status + `(size, root)` source).
  - `/workspace/iscc-monitor/internal/logclient/entries.go` — `RecordBytesFromBundle`,
    `ErrLeafOutOfBundle`; `/workspace/iscc-monitor/internal/logclient/proofbuilder.go` —
    `InclusionProofFromTiles`.
  - `/workspace/iscc-monitor/.claude/context/learnings/http-surface.md` — proofserve route conventions,
    status mapping, post-status write-drop, mutation-proven non-vacuousness expectation, dep direction
    `proofserve → {store, logclient}` never reverse, and the `VerifyInclusion(hasher, index, size,
    leafHash, proof, root)` arg-order gotcha.
  - `/workspace/iscc-monitor/.claude/context/learnings/store.md` — `SQLiteFetcher` p→width and the
    partial→full fallback for the final bundle (`serveEntries` already shows the
    `tiles.PartialTileSize(0, bundleIndex, size)` pattern).

## Not In Scope
- The HTML dashboard (`GET /`) and the HTML log browser (`GET /<domain>/log/`) — the other two open M3
  Verify criteria. They are the next sub-steps in this same arc, not this slice.
- A full proof-bundle assembler that packages `{checkpoint, inclusion proof, record bytes, hub key,
  ots?}` into one downloadable artifact (the authoritative client-verifies-it-itself path). This slice
  is the *weaker* verify-for-me verdict only; the bundle is a later slice.
- Persisting richer hub status (verified/unverified/unresolvable/rotated) into the store. Today only
  `FollowState.Frozen` and "has an accepted checkpoint" are persisted; the metrics registry holds the
  rest in memory. Derive the verdict's `hub_status` from what the store persists; do NOT add a status
  column or thread the metrics registry into proofserve.
- CORS / Cache-Control / ETag / conditional-GET on `/verify` (CORS already lands via the single
  `corsmw.Handler` wrap at the mux; per-route caching for the size-dependent proof surfaces is a
  separate later slice, like the other proofserve routes).
- The WASM verifier and OTS anchoring (later milestones).

## Implementation Notes
- **Route wiring mirrors the existing three exact mounts.** In `proofserve.Handler`, add
  `case "/verify": serveVerify(w, r, st, f, hubID)` to the `switch r.URL.Path`. In
  `cmd/iscc-monitor/main.go` `hubHandler`, add `mux.Handle("/verify", proofs)` — exact mounts beat the
  `/` → tilesserve subtree via `http.ServeMux` most-specific match (same as `/inclusion`).
- **`serveVerify` reuses `serveInclusion`'s guard ladder, but a bad id is a 200 verdict, never a 5xx.**
  The Verify criterion requires a malformed/unknown id to return the *documented non-verified verdict,
  never a 5xx*. So the route returns HTTP 200 with `verified:false` + a `reason` for every id-shaped
  input fault, reserving non-200 for genuine infra faults:
  - Missing/empty `iscc_id` → 200 verdict `{verified:false, reason:"missing iscc_id"}` (the
    most-literal satisfaction of "never a 5xx for a bad id"; do NOT 400 it the way `serveInclusion`
    does — verify-for-me always yields a verdict for id input).
  - `FollowState`/`CheckpointAt`/bundle-read DB faults → 500 (a real infra fault, allowed — these are
    not "a bad id").
  - `LastSize == 0` (no accepted checkpoint) → 200 verdict `{verified:false, reason:"no accepted
    checkpoint"}`, `hub_status` from `Frozen`.
  - `SeqsForISCCID` empty (unknown id) → 200 verdict `{verified:false, reason:"iscc_id not found"}`.
  - leaf resolved + `leafIndex < size`: build + verify the inclusion proof (below) →
    `{verified:true, included:true, tree_size, leaf_index, root}`. A tile/bundle miss
    (`errors.Is(err, os.ErrNotExist)`) or `ErrLeafOutOfBundle` → 200 verdict
    `{verified:false, reason:"tile not mirrored"}` (the leaf is accepted but the mirror has not caught
    up — a verdict, not a 5xx). A non-`os.ErrNotExist` build/read error → 500.
- **The inclusion result is a REAL Merkle check, not a stub (the oracle gate APPLIES).** Compute the
  accepted root via `st.CheckpointAt(ctx, hubID, size)` (returns the persisted `root []byte` for the
  accepted size — the RFC-6962 tree head the monitor vouches for). Read the leaf bytes the same way
  `serveEntries` does (`bundleIndex := leafIndex / tiles.TileWidth`, `offset := leafIndex %
  tiles.TileWidth`, `p := tiles.PartialTileSize(0, bundleIndex, size)`, `f.ReadEntryBundle(ctx,
  bundleIndex, p)` → `logclient.RecordBytesFromBundle(bundle, offset)`). Compute
  `leafHash := rfc6962.DefaultHasher.HashLeaf(record)`. Build the proof with
  `logclient.InclusionProofFromTiles(ctx, f.ReadTile, leafIndex, size)`. Verify with
  `proof.VerifyInclusion(rfc6962.DefaultHasher, leafIndex, size, leafHash, builtProof, root)` →
  `included := (err == nil)`. **Arg-order gotcha (learnings):** `VerifyInclusion(hasher, index, size,
  leafHash, proof, root)` — `leafHash` precedes `proof`, unlike `VerifyConsistency`.
- **`hub_status` is the glossary subset the store can prove:** `frozen` if `FollowState.Frozen`; else
  `verified` if `LastSize > 0` (only signature-verified checkpoints advance `LastSize`, ADR-0006). Do
  not invent a status the store cannot substantiate. Use the exact glossary strings (`CLAUDE.md` "Hub
  status" entry) — no synonyms.
- **`VerifyVerdict` struct** (snake_case JSON, like `ConsistencyEvidence`): e.g.
  `{ "iscc_id": string, "hub_status": string, "tree_size": uint64, "root": string (base64-Std of the
  accepted root), "leaf_index": uint64, "included": bool, "verified": bool, "reason": string }`.
  `reason` is empty on success; `verified` is the overall verdict (true only when the leaf resolved AND
  `proof.VerifyInclusion` accepted). Keep `root` base64-Std to match the package's existing encoding.
  Reuse the `writeEvidence` post-status-write-drop idiom (`json.NewEncoder(w).Encode`, drop the
  post-200 write error — a fixed-shape struct marshal cannot fail for content reasons).
- **Dep direction must hold:** `proofserve → {store, logclient, merkle}`, never the reverse. Adding
  `merkle/proof` + `merkle/rfc6962` to `proofserve` is fine (the test already imports both; go.sum
  carries them via logclient). go.mod/go.sum/schema must stay byte-unchanged.
- **Correctness rule (learnings.md, ADR-0008):** `iscc_id → seq` is one-to-many and verification is
  schema-agnostic — default to `seqs[0]` via `selectSeq`; interpret nothing about the id.
- **Test (golden, HTTP seam, non-vacuous):** add `internal/proofserve/verify_test.go` reusing
  `buildMirror(t, 300)`. Assert `GET /verify?iscc_id=` + `leafISCCID(5)` → 200, `verified:true`,
  `included:true`, `tree_size==300`, `leaf_index==5`, `root` base64-decodes to `tree.Hash()`. Assert an
  unknown id → 200, `verified:false`, non-empty `reason`, never 5xx. Cross the 256-leaf tile boundary
  (e.g. `leafISCCID(256)`) so the partial-bundle path is exercised. Make it mutation-resistant: a
  handler that hard-codes `verified:true` must fail the unknown-id case, and one that drops
  `proof.VerifyInclusion` must fail a corrupted-root / wrong-leaf negative — note this expectation for
  the advance author so the inclusion check is non-vacuous (matching `serveInclusion`'s test posture).

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...` all pass) and
  `gofmt -l .` is empty.
- `go test -count=1 -run TestVerify ./internal/proofserve` passes (golden + unknown-id + boundary).
- `go test -count=1 ./internal/proofserve ./cmd/iscc-monitor` passes uncached (route mount + handler).
- `go list -deps ./internal/store | grep -E 'proofserve|net/http'` is empty (store stays a leaf);
  `git diff --stat -- go.mod go.sum internal/store/schema.sql` is empty.
- `GOOS=js GOARCH=wasm go build ./internal/didweb` exits 0 (WASM verifier seam untouched).
- For the fixture log: `GET /verify?iscc_id=<leaf-5-id>` returns HTTP 200 with `verified==true`,
  `included==true`, `tree_size==300`, `leaf_index==5`, and base64-decoded `root == tree.Hash()`; an
  unknown `iscc_id` returns HTTP 200 with `verified==false` and a non-empty `reason` (never 5xx).

## Done When
`mise run check` is green and the verify-for-me route returns the documented JSON verdict (real
Merkle-verified inclusion result + hub status + checkpoint `(size, root)`) for a known id and a
non-verified 200 verdict for an unknown/malformed id, proven at the HTTP seam by
`go test -run TestVerify ./internal/proofserve`.
