# Handoff

## 2026-06-21 — Port the M2 fsck leaf-hasher (`LeafHashes`) — entry-bundle → RFC-6962 leaf hashes

**Done:** Landed the pure entry-bundle leaf hasher `logclient.LeafHashes(bundle []byte) ([][]byte,
error)` — a verbatim-in-shape port of `runfsck`'s `leafHasher` (`api.EntryBundle{}.UnmarshalText` then
`rfc6962.DefaultHasher.HashLeaf` per entry, taking `h[:]`), matching the `fsck.New(...)` hasher
contract exactly. Added a table-driven golden test that cross-checks each hash against an
independently-computed `rfc6962` leaf hash over a self-encoded C2SP bundle. No source/dep changes
outside the two new files.

**Files changed:**
- `internal/logclient/leafhasher.go` (new): exports `LeafHashes`; imports only `fmt` +
  `tessera/api` + `merkle/rfc6962` (file-level WASM purity). Pre-sizes `out`, wraps the unmarshal
  error with `%w` as `logclient.LeafHashes: unmarshal entry bundle: %w`.
- `internal/logclient/leafhasher_test.go` (new): `TestLeafHashes` (3-entry bundle, differing lengths
  incl. a zero-length entry, asserts each hash byte-equals the independent `HashLeaf` and is exactly
  32 bytes), `TestLeafHashesEmpty` (nil + empty bundle → `len==0, err==nil`), `TestLeafHashesTruncated`
  (`[]byte{0x00,0x05,0x01}` → `err != nil`). The in-test `encodeBundle` is the independent encode path
  (distinct from the decode + hash paths under test), so the cross-check is not circular.

**Verification:** `mise run check` → green (all 10 packages `ok`; `go build ./...`, `go vet ./...`,
`go test ./...`). Per-criterion:
- [x] `gofmt -l .` empty.
- [x] `go test -run TestLeafHashes ./internal/logclient` passes (all 3 subtests PASS).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` exits 0 (file-level purity invariant holds).
- [x] `go build ./... && git diff --exit-code -- go.mod go.sum` exits 0 (no new dependency; go.mod/go.sum
      byte-identical — both `tessera/api` and `merkle/rfc6962` were already in the closure via
      `proofbuilder.go`).
- [x] 3-entry bundle assertion: returns 3 hashes, each 32 bytes, `out[i]` byte-equal to
      `rfc6962.DefaultHasher.HashLeaf(ri)`.
- [x] `LeafHashes(nil)`/empty → `len==0, err==nil`; truncated `[]byte{0x00,0x05,0x01}` → `err != nil`.

**Next:** Wire `LeafHashes` + the `SQLiteFetcher` into the real M2 `fsck` root-rebuild conformance
slice — the first slice to face the trust-root oracle in CI. That needs `tessera/fsck` (pulls
`net/http`/`otel`/`klog` — a heavier go.mod cost, deliberately deferred here), real on-disk tile/
entry-bundle fixtures under `testdata/live/`, and the inclusion cross-check against the hub's own
`IsccLogInclusionProof`. Natural companion: wiring `.github/workflows/` CI + the external `notecheck`
oracle (the remaining open `normal` issue), since the tree is tidy-clean.

**Notes:**
- **Oracle/conformance gate APPLIES (RFC-6962 leaf-hash crypto) and is satisfied by independent ground
  truth, not a tautology.** The test's `encodeBundle` (manual `binary.BigEndian.PutUint16` framing) is a
  distinct code path from `api.EntryBundle.UnmarshalText` (decode) and `rfc6962.DefaultHasher.HashLeaf`
  (hash); the assertion compares `LeafHashes` output against an independently-recomputed `HashLeaf`. The
  zero-length entry exercises the framing walk on a non-trivial empty record. `notecheck`/
  `derive_vkey.py`/`fsck` ground-truth oracles are N/A for this slice (no signature/did:web/tile-rebuild
  path; the real `fsck.New(...).Check(...)` cross-check is the deferred next slice).
- `LeafHashes` is an **intentional unused-until-wired export seam** (like the consistency triggers,
  `IsFull`, `LookupHubKey`) — consumed by the future `fsck` slice, no caller added per scope. `go vet`
  is clean; do not flag it as dead code.
- WASM-purity is enforced at the **file level**: the `logclient` package as a whole pulls `net/http`
  via `didresolve.go`, so the load-bearing invariant is this file's import cleanliness + the
  `GOOS=js GOARCH=wasm go build ./internal/logclient` still passing (verified). Both new imports were
  already proven WASM-clean by `proofbuilder.go`.
- No backward-incompatible API change, no design deviation, no human review needed.
