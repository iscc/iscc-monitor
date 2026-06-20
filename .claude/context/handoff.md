# Handoff

## 2026-06-20 — Review of: Wire CheckShrink + CheckFork into follower.PollHub (freeze + alert-once)

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `advance` composed the two landed pure verdicts (`CheckShrink`/`CheckFork`) into M1's
ADR-0006 freeze behavior inside `follower.PollHub`, plus the one supporting store read
(`CheckpointAt`). The diff is tight (2 production files + 2 test files), all gates are green, and every
Verification criterion in `next.md` passes when run individually and uncached. Ordering, the
`prevSize>0` guard reuse, the freeze-never-crash contract, alert-once gating, and no-auto-unfreeze are
all correct; tests drive the full chain through the outbound-fetch seam against the real sb0 fixtures
and assert on observable store rows via an independent read-only connection.

**Verification:**
- [x] `mise run check` — green (`go build ./...`, `go vet ./...`, `go test ./...` all `ok`:
  didweb/follower/logclient/store). Re-confirmed uncached (`go test -count=1 ./...`).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -count=1 -run TestPollHub ./internal/follower` — PASS (verified-advances +
  unverified-no-advance still green after the `alert AlertFunc` signature change).
- [x] `TestPollHubFork` — verified observation at size 10183 with a distinct root → `violations.kind ==
  "fork"`, `frozen == 1`, cursor stays at 10183, exactly one alert.
- [x] Re-detection — a 2nd poll of the frozen hub records a 2nd `violations` row, alert count stays 1
  (verified stable over `-count=20`).
- [x] Other hubs unaffected — a 2nd registered hub polled clean advances to 10183, `frozen == 0`.
- [x] Restart — reopen from the same path → `FollowState.Frozen == true`, both violation rows survive.
- [x] `TestPollHubShrink` — verified observation at 10183 < prior 20000 → `kind == "shrink"`, `frozen
  == 1`, cursor stays at 20000, exactly one alert.
- [x] `TestCheckpointAt` — `(root, raw)` round-trips; absent `(hubID, treeSize)` and absent hub →
  `found == false`, nil error.
- [x] `git status --short go.mod go.sum` empty — no dependency added; `go.mod`/`go.sum` byte-identical.
- [x] Scope — exactly the 2 production files `next.md` named (`internal/follower/follower.go`,
  `internal/store/checkpoints.go`) + the 2 test files; no `## Not In Scope` item done (no
  equivocation/merkle, no poll loop, no did:web cache write, no real alert transport — `AlertFunc` is
  the minimal func seam allowed).
- [x] Gate integrity — no `//nolint`/`t.Skip`/build-tag/swallowed-error/removed-assertion in the
  unpushed Go diff (all matches are `.claude/` prose). Store stays a leaf (no `net/http` in closure);
  follower production imports are `{context,fmt,logclient,store,time}` only.
- [x] Conformance/oracle gate — correctly N/A: no signature, RFC-6962 proof, didweb, or merkle code
  touched; `internal/proof` does not exist yet. didweb + logclient golden suites still pass.

**Issues found:** (none) — no minor fixes needed.

**Next:** The merkle-backed **equivocation** trigger (RFC-6962 consistency-proof failure across
*growing* sizes) is the last M1 self-consistency trigger and the next high-value slice. It needs
`transparency-dev/merkle` + tile fixtures and trips the conformance/oracle gate (`fsck` root-rebuild,
inclusion cross-check vs the hub's `IsccLogInclusionProof`, golden-vector parity). It plugs into the
same `checkConsistency` seam as a third branch returning `ViolationEquivocation` + a real `ProofJSON`.
Alternatively, the **poll-loop / single-writer goroutine wrapper** (backed-off evidence-only re-poll
cadence of a frozen hub) is the other unblocked slice — `PollHub` is currently one observation per
call.

**Notes:**
- **Re-detection determinism is an implicit dependency on SQLite rowid order** (now in `learnings.md`):
  after the first fork detection there are two rows at the same `tree_size` (prior seed root + the
  contradicting evidence root), and re-detection works only because `CheckpointAt`'s `LIMIT 1`
  (no `ORDER BY`) returns the lower-rowid (prior) row. Correct today and verified stable over 20 runs,
  but the equivocation/merkle slice that changes how the prior accepted root is selected MUST keep
  "compare against the prior *accepted* root, not the contradicting evidence."
- The fork test's `sb0FixtureRootB64` guard compares raw seed bytes to a base64 *string* (can never
  trip) and the seed is 33 bytes (`copy` into `[32]byte` truncates harmlessly) — harmless, not a
  defect. The genuine non-vacuousness is the `kind == "fork"` (not "shrink") assertion at equal size.
- Branch is `develop`; pushing on PASS per policy (never `main`).
