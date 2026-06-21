# Next Work Package

## Step: Port the M2 fsck leaf-hasher (`LeafHashes`) — entry-bundle → RFC-6962 leaf hashes

## Goal
Land the pure entry-bundle leaf hasher that `fsck.New(...)` takes as its hasher argument — the
foundational, dep-clean unit of the M2 root-rebuild conformance. It decodes a tlog-tiles entry bundle
and RFC-6962-hashes each entry, so a later slice can feed it (plus the `SQLiteFetcher`) into the real
`fsck` integrity check. This is the smallest verifiable step toward M2's "fsck rebuilds each accepted
root" Verify bar, and it adds no new dependency.

## Scope
- **Create**: `internal/logclient/leafhasher.go` — one exported pure function (plus its file docstring).
- **Create**: `internal/logclient/leafhasher_test.go` — table-driven golden test.
- **Modify**: (none — no non-test/doc source file changes; `go.mod`/`go.sum` stay byte-identical)
- **Reference**:
  - `cauldron/iscc-hub/conformance/runfsck/main.go` — the `leafHasher(bundle []byte) ([][]byte, error)`
    reference (lines 24–35): `api.EntryBundle{}.UnmarshalText` then `rfc6962.DefaultHasher.HashLeaf(e)`
    per entry, returning `h[:]`. This is *exactly* the `fsck.New(...)` hasher signature.
  - `/home/dev/go/pkg/mod/github.com/transparency-dev/tessera@v1.0.2/api/state.go` — `EntryBundle`
    (lines 66–93): `UnmarshalText` parses the C2SP framing (2-byte big-endian length prefix + data per
    entry) into `Entries [][]byte`. Use this to write the in-test bundle encoder (mirror it:
    `binary.BigEndian.PutUint16(prefix, uint16(len(rec)))` then `prefix || rec`, concatenated).
  - `cauldron/iscc-hub/iscc_hub/log_tree.py` — `_leaf_hashes` / `merkle.leaf_hash` (lines 69–81): the
    hub-side ground truth that each committed record is RFC-6962-leaf-hashed in index order.
  - `internal/logclient/proofbuilder.go` — the in-package precedent that already imports
    `tessera/api` + `merkle/rfc6962` and documents the WASM-purity invariant; match its import style and
    file-docstring conventions.

## Not In Scope
- **No `fsck.New(...).Check(...)`**, no `tessera/fsck` / `tessera/client` import, no `cmd/fsck` wrapper.
  (`cauldron/tessera/fsck` is not even vendored here — only `cauldron/tessera/client` is — and `fsck`
  pulls `net/http`/`otel`/`klog`: a separate, heavier slice with its own go.mod cost. This step stays
  dep-clean.)
- No new conformance package, no `internal/conformance`, no real on-disk tile/entry-bundle fixtures
  under `testdata/live/` (the fixture slice is its own later step).
- No inclusion cross-check against the hub's `IsccLogInclusionProof` (the other half of the M2 Verify
  bar — a later slice).
- No CI / `notecheck` wiring (the open `normal` issue — a separate slice; this step touches no
  signature/proof CI surface).
- No follower wiring; `LeafHashes` is an intentional unused-until-wired export seam (like the
  consistency triggers and `IsFull`), consumed by the future `fsck` slice. Do not add a caller.

## Implementation Notes
- **Port `runfsck`'s `leafHasher` verbatim in shape**, exporting it. Suggested signature, matching the
  `fsck` hasher contract exactly:
  `func LeafHashes(bundle []byte) ([][]byte, error)`.
  Body: `eb := &api.EntryBundle{}; if err := eb.UnmarshalText(bundle); err != nil { return nil, fmt.Errorf("logclient.LeafHashes: unmarshal entry bundle: %w", err) }`,
  then for each `e := range eb.Entries` append `h := rfc6962.DefaultHasher.HashLeaf(e); out = append(out, h[:])`.
  Pre-size `out := make([][]byte, 0, len(eb.Entries))`.
- **Purity (Correctness rule: `proof/verify` is pure — keep WASM-shareable).** Import ONLY
  `fmt` + `github.com/transparency-dev/tessera/api` + `github.com/transparency-dev/merkle/rfc6962`. Do
  NOT import `net`/`net/http`/`database/sql`/`os`. The `logclient` *package* already pulls `net/http`
  via `didresolve.go`, so the load-bearing invariant is the **file-level** import cleanliness plus the
  `GOOS=js GOARCH=wasm go build ./internal/logclient` still building — both `tessera/api` and
  `merkle/rfc6962` are already proven WASM-clean by `proofbuilder.go`.
- **`HashLeaf` returns a 32-byte `[]byte`**; take `h[:]` to get the `[]byte` element exactly as
  `runfsck` does. Each returned hash is 32 bytes.
- **Error edge cases** (assert at least one): an empty bundle (`len == 0`) decodes to zero entries →
  `(len == 0, err == nil)` (non-nil-empty or nil slice both fine; assert on `len`). A truncated bundle
  (a 2-byte length prefix promising more bytes than remain) must surface the wrapped `UnmarshalText`
  error — feed `[]byte{0x00, 0x05, 0x01}` (claims 5 data bytes, only 1 present) and assert `err != nil`.
- **Oracle/conformance gate APPLIES (this is RFC-6962 leaf-hash crypto)** and must be satisfied by
  independent ground truth, not a tautology. Build the in-test entry bundle by encoding raw record
  bytes with the C2SP framing yourself (the encoder above), then assert each `LeafHashes(bundle)[i]`
  equals `rfc6962.DefaultHasher.HashLeaf(records[i])` computed independently in the test. The decode
  path (`api.EntryBundle.UnmarshalText`) and the hash path are distinct from the test's encode path, so
  the cross-check is not circular. Use ≥3 entries of differing lengths (including a zero-length entry)
  so the framing-walk is non-vacuous, and assert each hash is exactly 32 bytes.

## Verification
- `mise run check` is green (`go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .` empty).
- `go test -run TestLeafHashes ./internal/logclient` passes.
- `GOOS=js GOARCH=wasm go build ./internal/logclient` exits 0 (file-level purity invariant holds).
- `go build ./... && git diff --exit-code -- go.mod go.sum` exits 0 (no new dependency added).
- Assertion: for a 3-entry bundle with records `r0,r1,r2`, `LeafHashes(encode(r0,r1,r2))` returns 3
  hashes, each 32 bytes, with `out[i]` byte-equal to `rfc6962.DefaultHasher.HashLeaf(ri)`.
- Assertion: `LeafHashes(nil)` (or empty bundle) → `len == 0`, `err == nil`; a truncated bundle
  (`[]byte{0x00,0x05,0x01}`) → `err != nil`.

## Done When
`internal/logclient/leafhasher.go` exports the pure entry-bundle leaf hasher, every Verification check
passes (including the WASM build and the byte-identical go.mod/go.sum), and the golden test
cross-checks against independently-computed `rfc6962` leaf hashes.
