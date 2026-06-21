# Handoff

## 2026-06-21 — Review of: Port the M2 fsck leaf-hasher (`LeafHashes`) — entry-bundle → RFC-6962 leaf hashes

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance commit (`a260635`) lands `logclient.LeafHashes(bundle []byte) ([][]byte,
error)` — a verbatim-in-shape port of `runfsck`'s `leafHasher` (`api.EntryBundle{}.UnmarshalText` then
`rfc6962.DefaultHasher.HashLeaf` per entry, `h[:]`) — plus a table-driven golden test that cross-checks
each hash against an independently-computed `HashLeaf` over a self-framed C2SP bundle. Exactly two new
files, no source/dep changes elsewhere; `go.mod`/`go.sum` byte-identical. Faithful to the reference,
WASM-pure at the file level, and the golden is non-vacuous (reviewer mutation-proved it).

**Verification:**
- [x] `mise run check` — green (all 10 packages `ok`; build + vet + test).
- [x] `gofmt -l .` — empty.
- [x] `go test -count=1 -run TestLeafHashes ./internal/logclient` — all 3 subtests PASS (uncached).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — exits 0 (file-level purity invariant holds).
- [x] `git diff --exit-code HEAD~1..HEAD -- go.mod go.sum` — exits 0 (no new dependency).
- [x] 3-entry bundle (differing lengths incl. zero-length): returns 3 hashes, each exactly 32 bytes,
      `out[i]` byte-equal to `rfc6962.DefaultHasher.HashLeaf(ri)`.
- [x] `LeafHashes(nil)`/`{}` → `len==0, err==nil`; `{0x00,0x05,0x01}` truncated → `err != nil`. Confirmed
      these map exactly onto `EntryBundle.UnmarshalText`'s framing loop (read tessera source).
- [x] **Oracle gate APPLIES (RFC-6962 leaf-hash crypto) and is satisfied by independent ground truth.**
      The test's `encodeBundle` (`binary.BigEndian.PutUint16` framing) is a third code path distinct from
      the decode + hash paths under test. Mutation-proved non-vacuous: `h[0] ^= 0xff` → all 3 records FAIL
      the golden (reverted, tree clean). `notecheck`/`derive_vkey.py`/`fsck` correctly N/A for this slice
      (no signature/did:web/tile-rebuild path); trust-root conformance (didweb/logclient/follower) passes
      uncached and `derive_vkey.py` reproduces both vectors (`40b74463`/`22b08f3e`) — no regression.
- [x] **Gate-integrity clean** — no `//nolint`/`t.Skip`/build-tag/swallowed-error/deleted-assertion in
      the unpushed `.go` files; the only deviation from the reference is `%v`→`%w` (a strengthening).

**Issues found:** (none). No issue resolved this iteration. The open `normal` "No CI / `notecheck`
oracle wired" issue stays open — correctly out of scope here (`next.md` `## Not In Scope`).

**Next:** The real M2 `fsck` root-rebuild conformance slice — wire `LeafHashes` + `SQLiteFetcher` into
`fsck.New(...).Check(...)` over real on-disk tile/entry-bundle fixtures under `testdata/live/`, plus the
inclusion cross-check against the hub's own `IsccLogInclusionProof`. This is the first slice to face the
trust-root oracle for real (it pulls `tessera/fsck` → `net/http`/`otel`/`klog`, a heavier go.mod cost).
Its natural companion is wiring `.github/workflows/` CI + the external `notecheck` oracle (the open
`normal` issue), so the mirror path faces the oracle in CI when it first needs to. define-next picks the
smaller verifiable slice.

**Notes:**
- `LeafHashes` is an intentional unused-until-wired export seam (like the consistency triggers, `IsFull`,
  `LookupHubKey`) — `go vet` clean, NOT dead code. Its first caller is the deferred `fsck` slice.
- WASM purity is file-level: the `logclient` package pulls `net/http` via `didresolve.go`, so the
  load-bearing invariant is this file's import set (exactly `fmt` + `tessera/api` + `merkle/rfc6962`) +
  the `GOOS=js GOARCH=wasm` build still passing — both verified.
- The unpushed range also carries the loop's own `next.md`/`state.md` edits from earlier steps (out of
  review's purview to modify); the advance diff itself is just the two new files + handoff.
- Pushed to `origin/develop` (remote configured). M7 out of scope; no `critical`/`normal` blocker against
  M1, but the v1 DONE bar (M1→OTS Verify) is not met (M2+ pending) — loop continues.
