# Handoff

## 2026-06-20 — Review of: Pure RFC-6962 consistency-proof verifier (`CheckEquivocation`) + `transparency-dev/merkle` dep

**Verdict:** PASS_WITH_NOTES
**Loop:** CONTINUE

**Summary:** `advance` added the M1 third freeze trigger's pure building block —
`CheckEquivocation` + `ViolationEquivocation` in `internal/logclient/consistency.go`, backed by a real
`github.com/transparency-dev/merkle v0.0.2` consistency-proof verify — with a golden test built from a
genuine RFC-6962 tree. Scope is tight (1 production file + 1 test + the authorized go.mod/go.sum dep),
all gates are green, and the oracle/conformance gate (this IS RFC-6962 crypto) is satisfied: the golden
is ground truth from `rfc6962.DefaultHasher`, and I independently re-ran two mutations that both fail
the suite. PASS_WITH_NOTES (not plain PASS) only because I trimmed one over-promising doc line and want
the `testonly`-vs-`compact` deviation on the record — both cosmetic, neither blocks.

**Verification:**
- [x] `mise run check` green — build + vet + test, all 7 packages ok (re-run uncached).
- [x] `gofmt -l .` empty (whole tree).
- [x] `go test -count=1 -run TestCheckEquivocation ./internal/logclient` PASS (6 subtests + the 3
  `TestCheckEquivocationBoundariesSkipVerify` subtests).
- [x] `go test -count=1 -run TestViolationEquivocationKind ./internal/logclient` PASS;
  `string(ViolationEquivocation) == "equivocation"`.
- [x] Valid proof for growing `(M=7, N=11)` → `(false, nil)`; corrupted `nextRoot` → `(true, nil)`;
  corrupted proof element → `(true, nil)`.
- [x] `prevSize==0`, `nextSize==prevSize`, `nextSize<prevSize` all return `(false, nil)` AND skip
  `VerifyConsistency` (asserted by feeding garbage proof + roots and still getting false).
- [x] `go list -m github.com/transparency-dev/merkle` → `v0.0.2`; `go.mod` directive still `go 1.24.0`,
  no `toolchain` line; `go mod tidy` no-op; `go mod verify` passes.
- [x] `git diff --quiet HEAD~1..HEAD -- internal/store/schema.sql internal/store/checkpoints.go
  internal/follower/follower.go` exit 0 (no store/follower change).
- [x] **Oracle/conformance gate (APPLIES — RFC-6962/merkle crypto):** golden is ground truth, not
  author-asserted. `testonly.New(rfc6962.DefaultHasher)` builds a real append-only tree; the prover
  (`ConsistencyProof`/`HashAt`) and verifier (`proof.VerifyConsistency`) are independent merkle code
  paths, so the cross-check is not a tautology. I re-ran two mutations (then reverted): (1) force
  `(false,nil)` always → corrupted-root + corrupted-proof cases FAIL; (2) drop the growing guard → all
  three boundary-skip cases FAIL. A green-but-wrong verify cannot ship. `notecheck`/`derive_vkey.py`/
  `fsck` correctly N/A (no signature/did:web/tile path). No CI configured yet, so the external
  `notecheck` job is not a gate this iteration.
- [x] Gate-integrity scan over the 3 unpushed commits: no `//nolint`, `t.Skip`, build-tag exclusions,
  swallowed errors, or deleted tests/assertions.

**Issues found:** (none blocking)
- *Minor (fixed by reviewer):* the `CheckEquivocation` doc promised a `(false, non-nil err)` path for a
  "wrong-length root" — but the `[rootBytes]byte` array params make a wrong-length root unrepresentable,
  so that path can never fire. I trimmed the comment to state the actual contract (`err` always nil
  today; return kept for signature symmetry / future slice-param loosening). Comment-only, no behavior
  change; build/vet/test stay green.
- *Note (not a defect):* the test uses `transparency-dev/merkle/testonly.Tree` rather than `next.md`'s
  literal `compact.RangeFactory` suggestion. `next.md` permitted any in-test tree whose roots/proof are
  "ground truth from the hasher, never a hard-coded magic root"; `testonly.Tree` runs over the same
  `rfc6962.DefaultHasher` and is the library's own reference tree — strictly stronger (less bespoke
  test code), not a shortcut. Accepted.

**Next:** Wire `CheckEquivocation` into the follower — add the third branch to
`follower.checkConsistency` (map `FollowState.LastSize → prevSize`, the stored root at that size →
`prevRoot`, `info.TreeSize → nextSize`, `info.Root → nextRoot`, plus a fetched consistency proof),
returning `ViolationEquivocation` on a true verdict (→ `RecordViolation` + `RecordCheckpoint` evidence +
`Freeze` + alert-once, no advance). That branch needs the consistency-proof hashes, which only the
tile-fetch / `SQLiteFetcher` / `ProofBuilder` slice can source from mirrored hash tiles — so the
realistic order is tile-fetch first (to obtain proofs), then the follower branch.

**Notes:**
- Working branch is `develop`, 3 commits ahead of `origin/develop` (incl. this review). Remote
  configured; pushing `develop` on this PASS_WITH_NOTES verdict.
- `go-cmp v0.6.0` is in `go.sum` only as a transitive test-dep of `merkle/testonly` (`go mod why` →
  "main module does not need" it); not a direct require, CGO-free — no concern.
- `internal/proof` does not exist yet, so its `net`/`os`/`sqlite` purity rule is not yet applicable.
  `consistency.go` lives in `logclient`, which already pulls `net/http`/`os` via `didresolve.go`/
  `checkpoint.go`, so adding `merkle` introduces no new WASM/import constraint.
