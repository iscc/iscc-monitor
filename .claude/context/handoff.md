## 2026-06-21 — verify-for-me JSON verdict — `GET /<domain>/log/verify?iscc_id=<id>`

**Done:** Added the `/verify` route to the existing per-hub `proofserve.Handler`: a single
self-contained JSON verdict (`VerifyVerdict`) carrying the hub's persisted status, the accepted
checkpoint `(size, root)`, and a REAL RFC-6962 inclusion result recomputed from the local mirror and
Merkle-verified against the accepted root. Wired it on the per-hub mux in `cmd/iscc-monitor/main.go`
and documented the endpoint in `CLAUDE.md`. This is the weaker verify-for-me path (caller trusts the
verdict); an id-shaped input fault is always a 200 verdict, never a 5xx.

**Files changed:**
- `internal/proofserve/handler.go`: added `case "/verify"` to the path switch, the `serveVerify`
  function, the `hubStatus` helper, the `VerifyVerdict` response struct, and the `writeVerdict` writer
  (post-200 write-drop idiom). Added `merkle/proof` + `merkle/rfc6962` imports. Updated the `Handler`
  doc comment.
- `cmd/iscc-monitor/main.go`: added `mux.Handle("/verify", proofs)` next to the three existing exact
  mounts; updated the `hubHandler` doc comment (three → four proof routes).
- `CLAUDE.md`: added the `GET /<domain>/log/verify?iscc_id=<id>` line to the dev-instance endpoint list.
- `internal/proofserve/verify_test.go` (new): `buildVerifyMirror` (mirrors hash tiles AND entry
  bundles from the same leaf data + projections + accepted checkpoint, with a `corruptRoot` flag),
  `getVerdict` HTTP-seam helper, and 7 tests.

**Verification:** `mise run check` → green (all 15 packages `ok`); `gofmt -l .` empty. Per-criterion:
- `go test -count=1 -run TestVerify ./internal/proofserve` → PASS (golden known-id across the 256-leaf
  boundary {0,5,255,256,260,299}, unknown-id 200 verdict, missing-id 200 verdict, non-vacuous
  corrupted-root negative, frozen hub_status, no-accepted-checkpoint, leaf-hash-matches-tree).
- `go test -count=1 ./internal/proofserve ./cmd/iscc-monitor` → PASS uncached (route mount + handler).
- `go list -deps ./internal/store | grep -E 'proofserve|net/http'` → empty (store stays a leaf);
  `go list -deps ./internal/logclient | grep proofserve` → empty.
- `git diff --stat -- go.mod go.sum internal/store/schema.sql` → empty (byte-unchanged).
- `GOOS=js GOARCH=wasm go build ./internal/didweb` → exit 0 (WASM verifier seam untouched).
- Mirror-consuming pkgs re-ran uncached green: `follower`, `logclient`, `tilesserve`, `store`.
- Known-id verdict: HTTP 200, `verified==true`, `included==true`, `tree_size==300`, `leaf_index==5`,
  base64-decoded `root == tree.Hash()`. Unknown id: HTTP 200, `verified==false`, non-empty `reason`.

**Mutation-proven non-vacuous (oracle gate APPLIES — RFC-6962 inclusion crypto):** reverted-mutated
`serveVerify` to ignore the `proof.VerifyInclusion` result (`included := true`) → over a byte-accurate
mirror with a corrupted accepted root, `TestVerifyInclusionIsNonVacuous` FAILS on all three asserts
(verified/included/reason); reverted → green. A green-but-wrong handler cannot ship. The unknown-id
test catches a hardcoded `verified:true`.

**Next:** The other two open M3 Verify criteria are the HTML dashboard (`GET /`) and the HTML log
browser (`GET /<domain>/log/`) — the next sub-steps in this same arc. After those, the authoritative
path: a full proof-bundle assembler packaging `{checkpoint, inclusion proof, record bytes, hub key,
ots?}` into one downloadable client-verifiable artifact (the WASM verifier reuses the same
store/SQLiteFetcher seam). `serveVerify` already composes every piece the bundle needs except the
resolved hub key (in `hub_keys`) and the raw checkpoint bytes (`CheckpointAt` returns `raw` too,
currently discarded with `_`).

**Notes:**
- **`hub_status` is the store-provable glossary subset only.** `serveVerify` derives it from
  `FollowState`: `frozen` if `Frozen`, else `verified` (only signature-verified checkpoints advance
  `LastSize`, ADR-0006). Per Not-In-Scope, I did NOT add a status column or thread the metrics registry
  into proofserve. The richer statuses (unverified/unresolvable/rotated/inactive) live in the in-memory
  metrics registry and are not visible to this route — a known, documented limitation for this slice.
- **`buildVerifyMirror` does NOT reuse `buildMirror` (handler_test.go)** because `buildMirror` mirrors
  only hash tiles, not entry bundles — and `serveVerify` reads the leaf's record bytes from an entry
  bundle to hash them (the inclusion check needs the real leaf hash). `buildVerifyMirror` mirrors BOTH,
  framing entry bundles from the same `"leaf-%08d"` data the tree is built from, so
  `HashLeaf(record) == tree.LeafHash(i)`. `TestVerifyLeafHashMatchesTree` pins that assumption. It does
  reuse `nodeHash`, `leafISCCID`, `mirrorTree`, and `mirrorLeaves` from `handler_test.go`.
- **A mirror-not-caught-up miss is a 200 verdict, not a 5xx.** An entry-bundle/tile `os.ErrNotExist`
  or `ErrLeafOutOfBundle` → `{verified:false, reason:"tile not mirrored"}` (the leaf is accepted but
  the mirror lags). Genuine infra faults (DB read errors on `FollowState`/`CheckpointAt`/
  `SeqsForISCCID`/bundle-read, a non-`os.ErrNotExist` proof build error, or a `CheckpointAt`
  not-found at the accepted size) → 500.
- **`CheckpointAt(size)` not-found at the accepted size is treated as a 500 infra fault**, not a
  verdict: `LastSize > 0` means a checkpoint was recorded at that size, so a missing row is a real
  store inconsistency, not bad id input.
- Scope clean: 3 production files (handler.go, main.go, CLAUDE.md doc) + 1 test file, within the ≤3
  ceiling (docs/tests excluded). go.mod/go.sum/schema byte-unchanged. No `//nolint`/`t.Skip`/swallowed
  error introduced (the `writeVerdict` post-200 write-drop is the documented package idiom shared by
  `writeEvidence`/`writeConsistency`/`writeRecord`, not a new dodge).
