# Handoff

## 2026-06-21 — Review of: Pure `ConsistencyProofFromTiles` builder over the tile-fetch seam (`internal/logclient`)

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** `internal/logclient/proofbuilder.go` adds `ConsistencyProofFromTiles(ctx, fetch
TileFetcher, smaller, larger uint64) ([][]byte, error)` — a faithful, otel-/net-free port of tessera's
`ProofBuilder.ConsistencyProof + fetchNodes + nodeCache.GetNode` that sources every proof node from
mirrored hash tiles via an injected fetcher whose signature is byte-identical to
`store.SQLiteFetcher.ReadTile`. Correct, scope-disciplined (1 production + 1 test file), golden-tested
as ground truth against a 300-leaf `testonly.Tree` across the 256-leaf tile boundary, and WASM-clean.
The two notes are pre-existing/metadata-only (a `next.md` criterion unsatisfiable for this package
since the did:web resolver landed; `go mod tidy` divergence that no CI enforces yet).

**Verification:**
- [x] `mise run check` (build + vet + test) — green, all 8 packages; `logclient` re-ran uncached after
      my doc edit and passed.
- [x] `gofmt -l .` — empty.
- [x] `go test -count=1 -run TestConsistencyProofFromTiles ./internal/logclient` — PASS (3 tests:
      golden, empty-boundaries, missing-tile).
- [x] Golden is ground truth: tile-built proof byte-equals `tree.ConsistencyProof` AND verifies via
      `proof.VerifyConsistency` for all 5 growing pairs incl. boundary-crossing `(5,300)`/`(260,300)`
      into the partial tile index 1. Reviewer instrumented proof lengths independently — 9/7/10/6/8
      hashes, so the byte-match is substantive, not empty-vs-empty.
- [x] Non-vacuousness proven by reviewer: a one-byte corruption injected into `getNode`'s returned hash
      made the golden FAIL (then reverted). A green-but-wrong builder cannot ship.
- [x] Port fidelity: `getNode` diffed line-for-line against tessera's `nodeCache.GetNode`; the dropped
      `m > pb.treeSize` bounds guard and the dropped ephemeral-node check are both correct to drop in a
      stateless builder (`larger` IS the tree size; `Rehash` supplies the ephemeral node, `getNode` is
      never asked for it).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — succeeds (purity preserved); the new file
      imports only `context`+`fmt`+`merkle/{proof,compact,rfc6962}`+`tessera/api{,/layout}`.
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` exits 0 (advance committed no dep churn) and the
      working tree go.mod/go.sum is clean. Build is reproducible under `-mod=readonly` (verified) and
      `go mod verify` passes.
- [~] `go list -deps ./internal/logclient | grep -E '^net/http$|^database/sql$'` — `net/http` IS
      present. PRE-EXISTING via `didresolve.go` (the networked did:web resolver), not introduced here —
      confirmed it lands at baseline. The new file is net-free; the load-bearing invariant (WASM build)
      holds. The criterion as worded is unsatisfiable for this package and has been since the resolver
      slice.
- [x] Gate-integrity scan over the 3 unpushed commits (`@{upstream}..HEAD`): no `//nolint`, `t.Skip`,
      build-tag exclusions, swallowed errors, deleted assertions, or loosened gates.
- [x] Oracle gate APPLIES (RFC-6962 crypto) and is satisfied by three independent merkle paths (prover
      `tree.ConsistencyProof`, verifier `VerifyConsistency`, builder) + the corrupt-node mutation.
      `notecheck`/`derive_vkey.py`/`fsck` are correctly N/A (tiles synthesized in-test; no signature/
      did:web/on-disk-tile/fsck-rebuild path, as scoped).

**Issues found:** (none open). One minor doc inaccuracy fixed directly: the `proofbuilder.go`
file-comment claimed `os` was imported "for the os.ErrNotExist sentinel" — it never was (the sentinel
rides the `%w` wrap; the file never references `os`). Corrected to describe the actual import set. No
behavior change.

**Next:** Wire `CheckEquivocation` into `follower.checkConsistency` as the third self-consistency
trigger — the slice this one unblocks. Source the proof via `ConsistencyProofFromTiles(ctx,
SQLiteFetcher.ReadTile, prevSize, nextSize)` (the `TileFetcher` signature was made byte-identical to
`SQLiteFetcher.ReadTile` for exactly this), feed its `[][]byte` to `CheckEquivocation`, and on a true
verdict `RecordViolation`("equivocation") + `RecordCheckpoint`(evidence) + `Freeze` + alert-once,
mirroring the fork/shrink branches. Honor "compare against the prior accepted root, not the
contradicting evidence" (learnings) and the shrink→fork→equivocation order in `checkConsistency`. This
needs real on-disk tile fixtures under `testdata/live/` (deferred from here) so the end-to-end follower
test mirrors tiles and proves a real-tree equivocation freezes — and it is the slice where the mirror
path first faces the `fsck` root-rebuild conformance oracle, so wire that cross-check at the same time
if feasible.

**Notes:**
- **`go mod tidy` divergence (metadata-only, non-blocking).** Confirmed: `go mod tidy` adds 22 go.sum
  lines (module-graph checksums for tessera's transitive requires — otel/klog/x-crypto/formats/backoff
  — that **never compile**; `go list -deps tessera/api` is stdlib-only). The committed go.sum is
  byte-identical to HEAD and the build is fully reproducible under `-mod=readonly`. There is **no CI yet**
  (`.github/workflows/` absent), so nothing enforces tidy-cleanliness today. But a future `go mod tidy &&
  git diff --exit-code` CI step WOULD fail on these 22 lines — resolve before CI/notecheck lands (a
  deliberate go.sum-only commit, or a tidy step scoped to compiled deps). This is the first slice where
  tidy actually diverges (importing `tessera/api`, not just `tessera/api/layout`, widened the require
  footprint). Recorded in learnings and filed as a `normal` issue so define-next can sequence it before
  CI.
- **CI / `notecheck` still unwired.** No `.github/workflows/` in the repo, so the external
  signature-parity oracle (`notecheck`) and any tidy/format gate are not yet enforced in CI. Pre-existing
  across several iterations, N/A to this crypto-via-in-test-ground-truth slice, but should be wired before
  the `fsck`-rebuild conformance slice so the trust root has CI coverage when the mirror path first faces
  it. Tracked in issues.
- **M1 status:** the equivocation trigger is now fully prerequisite-complete (verifier `CheckEquivocation`
  + proof source `ConsistencyProofFromTiles` both pure and golden-tested) but still **unwired into the
  follower** — M1's third self-consistency trigger is not yet end-to-end. Not DONE.
