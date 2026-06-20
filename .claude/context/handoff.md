# Handoff

## 2026-06-20 — Review of: Pure fork-trigger detection (`CheckFork`) in `internal/logclient`

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance landed M1's second RFC-6962 self-consistency trigger exactly as `next.md`
asked: a pure `CheckFork(prevSize, prevRoot [rootBytes]byte, nextSize, nextRoot [rootBytes]byte) bool
== prevSize > 0 && nextSize == prevSize && nextRoot != prevRoot`, plus `ViolationFork ViolationKind =
"fork"`, both extending the existing `consistency.go` (no new file). Dep-free (array `!=`, no `bytes`
import), no follower wiring, no store calls, doc updated so only equivocation stays deferred. All
gates green; every load-bearing boundary is pinned by a non-vacuous table.

**Verification:**
- [x] `mise run check` green — `go build ./...`, `go vet ./...`, `go test ./...` all `ok`
  (didweb/follower/logclient/store) on go1.24.
- [x] `gofmt -l .` (whole tree) — empty.
- [x] `go test -count=1 -run TestCheckFork ./internal/logclient` — PASS, 5 subtests (true + false).
- [x] `string(ViolationFork) == "fork"` — PASS (`TestViolationForkKind`).
- [x] `CheckFork(10183, rootA, 10183, rootB) == true` (same size, different root) — PASS.
- [x] `CheckFork(10183, rootA, 10183, rootA) == false` (identical root, re-observation) — PASS.
- [x] `CheckFork(10183, rootA, 10182, rootB) == false` (shrink) — PASS.
- [x] `CheckFork(10183, rootA, 10184, rootB) == false` (growth with differing roots) — PASS.
- [x] `CheckFork(0, zeroRoot, 5, rootB) == false` (fresh-store `prevSize == 0` guard) — PASS.
- [x] Test setup guard `if rootA == rootB { t.Fatal }` makes "different root" non-vacuous — present.
- [x] No dep added — `git status --short go.mod go.sum` empty; `go list -m
  github.com/transparency-dev/merkle` → "not a known dependency"; `net/http` count in logclient deps
  still 4; package imports unchanged (no `bytes`).
- [x] Scope discipline — across all 3 unpushed commits, exactly one non-test/doc Go file
  (`consistency.go`) modified (≤3 budget). Nothing from `## Not In Scope`: no merkle dep, no `PollHub`
  wiring, no `RecordViolation`/`Freeze` calls, no tile fixtures, no equivocation trigger, no store /
  `accept.go` / `verify.go` / didweb edits.
- [x] Quality-gate integrity — scanned all unpushed commits (`@{upstream}..HEAD`); the only
  `nolint`/`t.Skip` string matches are in `.claude/` prose, none in added Go lines.
- [n/a] Conformance/oracle gate — pure size/root array comparison; no signature, RFC-6962 proof,
  didweb, or merkle code touched, so `notecheck` / `derive_vkey.py` / `fsck` parity is N/A. didweb +
  logclient golden suites still pass (no regression). The merkle-backed equivocation slice will trip
  this gate.

**Issues found:** (none)

**Next:** Wire `CheckShrink` + `CheckFork` into `follower.PollHub` — the composition slice both pure
verdicts were left unwired for. Map `FollowState.LastSize → prevSize` and the stored root at that size
(a `checkpoints` lookup, since `LastRoot` is intentionally NOT persisted in `follow_state`) →
prevRoot; map `CheckpointInfo.TreeSize/Root → nextSize/nextRoot`; on a true verdict call
`RecordViolation` + `Freeze`, with the "exactly one alert" mechanism. Test through the outbound-fetch
boundary with synthetic same-size-different-root / shrink fixtures, asserting `violations.kind` +
`frozen=1` + exactly one alert + other hubs unaffected + evidence surviving restart. Keep
`transparency-dev/merkle` + tile fixtures deferred to the single equivocation slice.

**Notes:**
- `CheckFork` / `ViolationFork` are an *intentional* unused-until-wired export seam (same as
  `CheckShrink` / `ViolationShrink`), referenced only by their own tests until the follower wiring
  slice lands — not dead code. `go vet` is clean.
- The `prevSize > 0` guard is load-bearing exactly like shrink's: it keeps the fresh-store
  `FollowState{}.LastSize == 0` + its zero `[32]byte` root from being misread as a fork. Array `!=` is
  Go's elementwise compare on `[32]byte` — no `bytes` import; the only `bytes` token in the file is a
  comment explaining why.
- Branch is `develop`; remote `origin` configured. Pushing on PASS.
