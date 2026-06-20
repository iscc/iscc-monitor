# Handoff

## 2026-06-20 — Review of: Pure shrink-trigger detection (`CheckShrink`) in `internal/logclient`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance landed the first of M1's three RFC-6962 consistency triggers as a pure,
golden-testable unit exactly as `next.md` asked: `CheckShrink(prev, next uint64) bool == prev > 0 &&
next < prev`, plus a `ViolationKind` string type carrying only `ViolationShrink = "shrink"`. Additive
new file only (0 non-test/doc files modified), no new dependency, no follower wiring, no store calls.
All gates green; every load-bearing boundary is pinned by non-vacuous table tests.

**Verification:**
- [x] `mise run check` green — `go build ./... && go vet ./... && go test ./...` all `ok`
  (logclient/follower/store/didweb) on go1.24.
- [x] `gofmt -l .` — empty (no formatting failures across the whole tree).
- [x] `go test -count=1 -run TestCheckShrink ./internal/logclient` — PASS, 6 non-vacuous subtests
  (true and false cases both present).
- [x] `CheckShrink(10183, 10182) == true` (strict shrink) — PASS.
- [x] `CheckShrink(10183, 10183) == false` (equal — re-observation) — PASS.
- [x] `CheckShrink(10183, 10184) == false` (growth) — PASS.
- [x] `CheckShrink(0, 5) == false` (fresh-store zero, the load-bearing guard) — PASS.
- [x] `CheckShrink(0, 0) == false` — PASS. (Extra `CheckShrink(1, 0) == true` shrink-to-zero is a
  correct additional boundary.)
- [x] `go list -deps ./internal/logclient | grep -c '^net/http'` → 4 (unchanged; the additive file
  imports nothing — confirmed it appears with zero `GoFiles` imports).
- [x] `go list -m github.com/transparency-dev/merkle` → "not a known dependency"; `git status --short
  go.mod go.sum` empty (no dep added, manifest untouched).
- [x] Scope discipline — unpushed Go diff is exactly the two new files
  (`internal/logclient/consistency.go` + `_test.go`); nothing from `## Not In Scope` (no merkle dep,
  no `PollHub` wiring, no `RecordViolation`/`Freeze` calls, no tile fixtures, no fork/equivocation
  constants).
- [x] Quality-gate integrity — scanned all 3 unpushed commits (`@{upstream}..HEAD`): no `//nolint`,
  `t.Skip`/`SkipNow`, build tags, swallowed errors, or deleted assertions.
- [n/a] Conformance/oracle gate — pure size-arithmetic verdict; no signature / RFC-6962 proof / didweb
  / merkle code touched, so `notecheck` / `derive_vkey.py` / `fsck` parity is N/A. The logclient +
  didweb golden suites still pass, confirming no regression. The merkle-backed equivocation slice that
  follows *will* trip this gate.

**Issues found:** (none)

**Next:** The fork trigger (same `tree_size`, different `root`) is the natural dep-free follow-on — a
`[rootBytes]byte` compare against the stored root at equal size, composing with `CheckpointInfo.Root`
and `store.RecordCheckpoint`'s `UNIQUE(hub_id, tree_size, root)` the same way shrink composes with
`LastSize`. Landing fork next keeps `transparency-dev/merkle` + tile fixtures deferred to the single
equivocation slice that genuinely needs them. After fork, the wiring slice into `follower.PollHub`
(map `FollowState.LastSize → prev`, `CheckpointInfo.TreeSize → next`; on a true verdict call
`RecordViolation` + `Freeze`; the "exactly one alert" mechanism) is the composition step that turns
these pure verdicts into the M1 freeze behavior.

**Notes:**
- `CheckShrink` / `ViolationShrink` are an *intentional* export seam, referenced only by their own
  tests until the follower wiring slice lands — not dead code. `go vet` is clean; do not flag them.
- The `prev > 0` guard is the load-bearing edge: it keeps the fresh-store `FollowState{}.LastSize == 0`
  from being misread as a shrink. The table pins both that (`(0,5)→false`) and the strict decrease.
- Branch is `develop`; remote `origin` configured. Pushing on PASS.
