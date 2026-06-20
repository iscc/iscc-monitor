# Handoff

## 2026-06-20 — Pure shrink-trigger detection (`CheckShrink`) in `internal/logclient`

**Done:** Added the first of M1's three RFC-6962 consistency triggers as a pure, golden-testable unit:
`CheckShrink(prev, next uint64) bool` returns true only on a strict tree-size decrease with `prev > 0`
(a hub rewriting history backwards against this monitor). Additive new file only — no new dependency,
no follower wiring, no store calls.

**Files changed:**
- `internal/logclient/consistency.go` (new): the pure `CheckShrink` predicate plus a `ViolationKind`
  string type with the single `ViolationShrink ViolationKind = "shrink"` constant. File docstring
  documents that fork (same size, different root) and equivocation (consistency-proof failure) are
  deferred to the merkle-backed slices, so the next scoper sees the seam.
- `internal/logclient/consistency_test.go` (new, test): table-driven `TestCheckShrink` pinning all five
  required boundaries plus a `shrink-to-zero` case, and `TestViolationShrinkKind` pinning the kind
  string to `"shrink"`.

**Verification:** `mise run check` → green (`go build ./... && go vet ./... && go test ./...` all exit
0 on go1.24; logclient/follower/store/didweb all `ok`).
- [x] `gofmt -l internal/logclient/consistency.go internal/logclient/consistency_test.go` → empty.
- [x] `go test -count=1 -run TestCheckShrink ./internal/logclient` → PASS (6 non-vacuous subtests).
- [x] `CheckShrink(10183, 10182) == true` (strict shrink) — PASS.
- [x] `CheckShrink(10183, 10183) == false` (equal — re-observation) — PASS.
- [x] `CheckShrink(10183, 10184) == false` (growth) — PASS.
- [x] `CheckShrink(0, 5) == false` (fresh store) — PASS.
- [x] `CheckShrink(0, 0) == false` — PASS.
- [x] `go list -deps ./internal/logclient | grep -c '^net/http'` → 4 (unchanged by the additive file;
  the count comes from the pre-existing networked code, not `consistency.go`).
- [x] `go list -m github.com/transparency-dev/merkle` → "not a known dependency" (no dep added).
- [x] `git status --short go.mod go.sum` → empty (go.mod/go.sum untouched).
- [n/a] Conformance/oracle gate — this is a pure size-arithmetic verdict; no signature / RFC-6962 proof
  / didweb / merkle code touched, so `notecheck` / `derive_vkey.py` / `fsck` parity is N/A. (The
  merkle-backed fork/equivocation slices that follow *will* trip that gate.)

**Next:** The fork trigger (same `tree_size`, different `root`) is the natural follow-on and is also
detectable without `transparency-dev/merkle` — it is a `[rootBytes]byte` comparison against the stored
root for an equal size, composing with `CheckpointInfo.Root` and `store.RecordCheckpoint`'s
UNIQUE(hub_id, tree_size, root) the same way shrink composes with `LastSize`. Landing fork next keeps
the merkle dep + tile fixtures deferred to the single equivocation slice that genuinely needs them.
After fork, the wiring slice into `follower.PollHub` (map `FollowState.LastSize → prev`,
`CheckpointInfo.TreeSize → next`; on a true verdict call `RecordViolation` + `Freeze`; the
"exactly one alert" mechanism) is the next composition step.

**Notes:**
- Kept the signature `CheckShrink(prev, next uint64)` (two sizes, not whole `CheckpointInfo` structs)
  exactly as `next.md` suggested, so the unit stays trivially table-testable; the caller maps verdicts
  to store structs at the call site, mirroring how `PollHub` already maps the verdict chain.
- Added the small `ViolationKind` type with only `ViolationShrink = "shrink"` (the named-result option
  `next.md` permitted). Deliberately did NOT pre-add fork/equivocation constants (YAGNI) — they land
  with their detectors. `string(ViolationShrink)` equals the `store.Violation.Kind` value the
  persistence step populates.
- `logclient` imports are unchanged by this file (it imports nothing); `consistency.go` is pure stdlib-
  free arithmetic. No `t.Skip` / `//nolint` / swallowed errors / build tags.
- Branch is `develop`.
