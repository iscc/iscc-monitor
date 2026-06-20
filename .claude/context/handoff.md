# Handoff

## 2026-06-20 — Pure RFC-6962 consistency-proof verifier (`CheckEquivocation`) + `transparency-dev/merkle` dep

**Done:** Added the third freeze trigger's pure building block to `internal/logclient/consistency.go`:
`ViolationEquivocation ViolationKind = "equivocation"` and `CheckEquivocation(prevSize, prevRoot,
nextSize, nextRoot, consistencyProof) (violated bool, err error)`, which verifies an RFC-6962
consistency proof via `github.com/transparency-dev/merkle` (`proof.VerifyConsistency` +
`rfc6962.DefaultHasher`) and reports `violated=true` only when the proof FAILS on a strictly-growing
pair. Golden-tested against a real RFC-6962 tree built in-test from `testonly.Tree`, with two mutation
checks proving the golden is non-vacuous.

**Files changed:**
- `internal/logclient/consistency.go`: added the `merkle/proof` + `merkle/rfc6962` imports,
  `ViolationEquivocation`, and `CheckEquivocation`; rewrote the package doc (the "equivocation deferred
  to a merkle-backed slice" and "import-free of any new dep" lines are gone — all three triggers now
  live here).
- `internal/logclient/consistency_test.go`: added `TestCheckEquivocation` (table-driven golden over a
  ground-truth tree: valid-proof→false, corrupted-root→true, corrupted-proof→true, plus the three
  non-growing boundaries→false), `TestCheckEquivocationBoundariesSkipVerify` (boundaries return
  `(false,nil)` even when fed garbage proof+roots, proving they short-circuit before
  `VerifyConsistency`), `TestViolationEquivocationKind`, and a `rootArray` helper.
- `go.mod` / `go.sum`: added `github.com/transparency-dev/merkle v0.0.2` (now a direct require);
  `go mod tidy` pulled `github.com/google/go-cmp v0.6.0` as a transitive indirect (test dep of
  `merkle/testonly`). Module directive stays `go 1.24.0`, no `toolchain` line.

**Verification:** `mise run check` → green (build + vet + test, all 7 packages ok; re-run uncached).
- [x] `gofmt -l .` empty (whole tree).
- [x] `go test -count=1 -run TestCheckEquivocation ./internal/logclient` PASS (6 subtests +
  `TestCheckEquivocationBoundariesSkipVerify`'s 3 subtests).
- [x] `go test -count=1 -run TestViolationEquivocationKind ./internal/logclient` PASS;
  `string(ViolationEquivocation) == "equivocation"`.
- [x] Valid consistency proof for growing `(M=7, N=11)` → `(false, nil)`; same call with corrupted
  `nextRoot` → `(true, nil)`; corrupted proof element → `(true, nil)`.
- [x] `prevSize==0`, `nextSize==prevSize` (fork), and `nextSize<prevSize` (shrink) all return
  `(false, nil)` AND skip `VerifyConsistency` (asserted by feeding garbage proof+roots and still
  getting `false`).
- [x] `go list -m github.com/transparency-dev/merkle` → `v0.0.2`; `go.mod` directive still `go 1.24.0`,
  no `toolchain` line; `go mod tidy` is a no-op diff (verified both `go.mod` and `go.sum`).
- [x] `git diff --quiet HEAD -- internal/store/schema.sql internal/store/checkpoints.go
  internal/follower/follower.go` exit 0 (no store/follower change).

**Oracle/conformance gate (APPLIES — this is RFC-6962/merkle crypto):** The golden vector is ground
truth from `rfc6962.DefaultHasher`, not author-asserted: `testonly.Tree` builds a real append-only tree
over 11 leaves, `HashAt(M)`/`HashAt(N)` give the roots and `ConsistencyProof(M, N)` the valid proof
(uses `proof.Consistency` + `Nodes.Rehash` internally — the merkle library's own ground truth). Two
mutation checks were run and removed: (1) forcing `CheckEquivocation` to always return `(false,nil)`
fails the two corrupted cases; (2) dropping the growing guard fails all three boundary-skip cases. So a
green-but-wrong verify (accepting a malformed proof, or verifying on the wrong boundaries) cannot ship.
`notecheck`/`derive_vkey.py`/`fsck` are still N/A here — no signature, did:web, or tile/fsck path is
touched; this is consistency-proof Merkle math only.

**Next:** Wire `CheckEquivocation` into the follower (the NEXT slice, explicitly deferred here): add the
third branch to `follower.checkConsistency` so a verified growing observation maps
`FollowState.LastSize → prevSize`, the stored root at that size → `prevRoot`, `info.TreeSize → nextSize`,
`info.Root → nextRoot`, plus a fetched consistency proof → `CheckEquivocation`, returning
`ViolationEquivocation` on a true verdict (→ `RecordViolation` + `Freeze` + alert-once, evidence-only,
no advance). That wiring needs the consistency-proof hashes, which requires the tile-fetch /
`SQLiteFetcher` / `ProofBuilder` slice to obtain them from mirrored hash tiles — so the realistic order
is tile-fetch first (to source proofs), then the follower branch.

**Notes:**
- I used `merkle/testonly.Tree` (a test-only package, imported only in `_test.go`) instead of hand-wiring
  `compact.RangeFactory` + `Nodes.Rehash`. `next.md` named `compact` as the preferred builder and a
  hand-built tree as the acceptable minimum; `testonly.Tree` is the library's own ground-truth tree (it
  composes `proof.Consistency` + `Nodes.Rehash` over `rfc6962.DefaultHasher` exactly as a hand-rolled
  `compact` build would), so it satisfies the "ground truth from the hasher, never a hard-coded magic
  root" rule while staying well within the file budget. Flagging the choice since it deviates from the
  literal `compact.RangeFactory` suggestion — it is strictly stronger (same hasher, less bespoke test
  code), not a shortcut.
- Error-vs-violation discipline is implemented as documented: ANY `VerifyConsistency` failure on the
  growing path becomes `(true, nil)` — a non-verifying proof is the evidence, never a poll error. The
  returned `err` is currently always `nil` from this function (I did not add an explicit root-length
  validation, since the `[rootBytes]byte` array params make a wrong-length root unrepresentable at the
  type level — `prevRoot[:]`/`nextRoot[:]` are always 32 bytes). The `err` return and the `wantErr`
  table column are kept for the documented contract and future input validation if the signature ever
  loosens.
- `CheckEquivocation`/`ViolationEquivocation` are an intentional unused-until-wired export seam (same as
  shrink/fork before their wiring) — `go vet` is clean and they should not be flagged as dead code.
- `go-cmp v0.6.0` entered `go.sum` as an indirect transitive of `merkle/testonly`; it is test-only and
  CGO-free, no concern.
