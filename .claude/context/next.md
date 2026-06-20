# Next Work Package

## Step: Pure RFC-6962 consistency-proof verifier (`CheckEquivocation`) + `transparency-dev/merkle` dep

## Goal
Land the dep-bearing, pure building block of M1's third freeze trigger: a golden-tested
`CheckEquivocation` that verifies an RFC-6962 consistency proof relates a prior accepted root at
size M to a new root at size N (M < N), returning `true` when the proof FAILS to verify (= a
self-consistency violation / split view). This is the one Merkle primitive the later follower
wiring + tile-fetch slices will compose; landing it pure and golden-tested first keeps the
conformance/oracle gate honest before any I/O or tile fixtures exist.

## Scope
- **Create**: (none — extend the existing consistency file)
- **Modify**:
  - `internal/logclient/consistency.go` — add `ViolationEquivocation ViolationKind = "equivocation"`
    and `CheckEquivocation(prevSize uint64, prevRoot [rootBytes]byte, nextSize uint64, nextRoot [rootBytes]byte, proof [][]byte) (violated bool, err error)`. Update the file's package doc, which
    currently says equivocation is "deferred to a merkle-backed slice" and that the file "stays
    import-free of any new dep" — both lines change with this step.
  - `go.mod` / `go.sum` — add `github.com/transparency-dev/merkle v0.0.2` (this is the dep change for
    this step; it is the exact reuse target named in `target.md`'s Stack block).
  - `internal/logclient/consistency_test.go` — add `TestCheckEquivocation` (golden, table-driven) and
    `TestViolationEquivocationKind` (test file, not counted against the 3-file budget).
- **Reference**:
  - `/workspace/iscc-monitor/cauldron/iscc-hub/specs/iscc-log.md` §10.2 "Consistency proof"
    (lines 296–305): a failing consistency proof between trusted size M and later size N is evidence
    of a split view.
  - `/workspace/iscc-monitor/cauldron/iscc-hub/iscc_hub/log_tree.py` (`tree_head`, `inclusion_proof`)
    for how the hub builds RFC-6962 structures (Python reference; do NOT import — Go reuses
    `transparency-dev/merkle`).
  - `/workspace/iscc-monitor/internal/logclient/verify.go` line 35 (`const rootBytes = 32`) — the
    existing root-array type.
  - `/workspace/iscc-monitor/internal/logclient/consistency.go` (the shrink/fork twins this sits beside).

## Not In Scope
- **No follower wiring.** Do NOT touch `internal/follower/*`, `checkConsistency`, or `PollHub`.
  Mapping `FollowState` + a fetched consistency proof into a `CheckEquivocation` call (and the third
  branch of `checkConsistency` returning `ViolationEquivocation`) is the NEXT slice.
- **No tile fetching / `SQLiteFetcher` / `ProofBuilder`.** Obtaining the real consistency-proof
  hashes from mirrored hash tiles is M2-adjacent work; this step verifies a proof it is GIVEN.
- **No new live fixtures.** Do NOT add tile/entry-bundle files under `testdata/live/`. The golden
  vector is built in-test from a known RFC-6962 tree (see notes), not from captured hub tiles.
- **No `store`/schema changes.** `internal/store/schema.sql` and `internal/store/checkpoints.go`
  stay byte-identical.
- Do not change `internal/didweb` purity or the WASM build target — `logclient` already imports
  `net/http` (via `didresolve.go`), so adding `merkle` here does not introduce a new WASM constraint.

## Implementation Notes
- **API (verified by `go doc` against `v0.0.2`):**
  `proof.VerifyConsistency(hasher merkle.LogHasher, size1, size2 uint64, proof [][]byte, root1, root2 []byte) error`
  in `github.com/transparency-dev/merkle/proof`; pass `rfc6962.DefaultHasher` (a `*rfc6962.Hasher`
  that satisfies `merkle.LogHasher`) from `github.com/transparency-dev/merkle/rfc6962`. Requires
  `0 <= size1 <= size2`.
- **Semantics (load-bearing — mirror the shrink/fork guards):** `CheckEquivocation` returns
  `(violated, err)`. The trigger fires (`violated=true`) when `VerifyConsistency` returns a non-nil
  error for a *growing* pair (`prevSize > 0 && nextSize > prevSize`) — the hub presented two roots a
  consistent append-only log could never both produce. Boundaries that are NOT this trigger's
  concern (each returns `false, nil`, never an error, and must NOT call `VerifyConsistency`):
  `prevSize == 0` (fresh store, nothing accepted yet — matches the `CheckShrink`/`CheckFork`
  `prev > 0` guard); `nextSize == prevSize` (fork's concern, a same-size root compare);
  `nextSize < prevSize` (shrink's concern). Only the strictly-growing case calls `VerifyConsistency`.
  A *successful* verification means the log is consistent → `false, nil`.
- **Error vs. violation discipline (ADR-0006 "freeze, never crash"):** A failed proof is a *verdict*
  (`violated=true, err=nil`), NOT a Go error — the caller will freeze on it. The simplest correct
  contract: convert ANY `VerifyConsistency` failure on the growing path to `violated=true, err=nil`
  (the proof not verifying IS the evidence). Document this clearly: a non-verifying proof must never
  surface as a poll error that could abort the loop. Reserve the returned `err` for nothing in this
  pure layer unless you choose to validate input root lengths — if you do, an obviously-malformed
  input (wrong-length root slice) may return a non-nil `err`, but a failing-yet-well-formed proof
  must stay `violated=true, err=nil`.
- **Pass roots as slices:** `VerifyConsistency` wants `root1, root2 []byte`; pass `prevRoot[:]` and
  `nextRoot[:]`. Keep the `[rootBytes]byte` array params for signature symmetry with `CheckFork`.
- **Imports:** add `github.com/transparency-dev/merkle/proof` and `.../rfc6962` to `consistency.go`.
  This is the first non-stdlib dep in this file — update the file doc and drop the "import-free of any
  new dep" sentence.
- **go.mod hygiene (learnings — the `modernc`/`x/mod` precedent):** `merkle v0.0.2` declares a low
  `go` directive; after `go get` confirm the module directive stays `go 1.24.0` with NO `toolchain`
  line (drop any auto-injected `toolchain go1.24.x`). Run `go mod tidy` and verify it is a no-op
  diff afterward. If `v0.0.2` forces the directive above `1.24.0`, fall back to `v0.0.1` and note why
  in the commit/handoff.
- **Correctness rule (learnings / `target.md` oracle gate):** this IS crypto / RFC-6962 code, so the
  conformance/oracle gate APPLIES — a green-but-wrong verify (e.g. one accepting a malformed proof)
  must not ship on an LLM PASS alone. The golden vector must be ground truth, not author-asserted.
  Build a real RFC-6962 tree IN-TEST with `github.com/transparency-dev/merkle/compact`
  (`compact.RangeFactory{Hash: rfc6962.DefaultHasher.HashChildren}`, leaves via
  `rfc6962.DefaultHasher.HashLeaf`) to obtain `rootM`, `rootN`, and a VALID consistency proof: use
  `proof.Consistency(M, N)` to get the `Nodes`, then read each node hash out of the compact range.
  Positive (consistent) case must verify → `violated=false`; negative case flips one byte of `rootN`
  (or corrupts a proof element) → `violated=true`. Make it non-vacuous: assert the consistent and
  corrupted roots actually differ before asserting opposite verdicts, mirroring the
  `if rootA == rootB { t.Fatal }` guard in `TestCheckFork`. If wiring `compact` proves heavier than
  this step's budget, the acceptable minimum is a hand-built two-/three-leaf tree whose interior and
  root hashes are derived in-test directly from `rfc6962.DefaultHasher` (still ground-truth from the
  hasher) — never a hard-coded magic root.

## Verification
- `mise run check` is green (build + vet + test, all packages; `gofmt -l .` empty).
- `go test -count=1 -run TestCheckEquivocation ./internal/logclient` passes.
- `go test -count=1 -run TestViolationEquivocationKind ./internal/logclient` passes and
  `string(ViolationEquivocation) == "equivocation"`.
- A valid consistency proof for a growing pair `(M, N), M < N` yields
  `CheckEquivocation(...) == (false, nil)`; the same call with a corrupted `nextRoot` (or proof)
  yields `(true, nil)`.
- `CheckEquivocation` with `prevSize == 0` (and with `nextSize <= prevSize`) returns `(false, nil)`
  and does NOT call `VerifyConsistency` (the fresh-store / non-growing guards, asserted in the table).
- `go list -m github.com/transparency-dev/merkle` reports `v0.0.2` (or the documented `v0.0.1`
  fallback); the `go.mod` module directive is still `go 1.24.0` with no `toolchain` line; `go mod
  tidy` produces a no-op diff.
- `git diff --quiet HEAD -- internal/store/schema.sql internal/store/checkpoints.go internal/follower/follower.go`
  exits 0 (no store/follower change in this slice).

## Done When
`CheckEquivocation` and `ViolationEquivocation` exist as a pure, golden-tested RFC-6962
consistency-proof verifier backed by `transparency-dev/merkle`, with `mise run check` green and
every Verification criterion passing — leaving follower wiring and tile-fetch for the next slices.
