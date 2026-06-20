# Handoff

## 2026-06-20 — Review of: Coverage tracking — persist `monitored_since` (size + time), set-once, on first verified observation

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** `SetCoverage`/`Coverage` were added to the store as a guarded set-once
`UPDATE … WHERE monitored_since_size IS NULL` plus a `sql.NullInt64` reader, and wired into `PollHub`
on the verified, non-violation path (between `RecordCheckpoint` and `AdvanceFollowState`, kept out of
`freeze`). The implementation matches `next.md` exactly: immutable coverage start (ADR-0001), no new
dependency, store stays a leaf, and the freeze/non-verified paths never start coverage. Scope is tight
(2 production files + 2 test files), tests are thorough and non-vacuous, all gates green.

**Verification:**
- [x] `mise run check` — green: `go build` / `go vet` / `go test ./...` all `ok` (7 packages).
- [x] `gofmt -l .` — empty (clean).
- [x] `go test -run TestCoverage ./internal/store` — PASS: `TestCoverageSetOnce` (100/t0 not moved by
  500/t1, raw columns re-checked), `TestCoverageUnset` (un-started + absent hub → zero value, nil err),
  `TestCoverageZeroObservedAtNull` (zero time → NULL, Set true, zero Since).
- [x] `go test -run TestPollHub ./internal/follower` — PASS: a verified poll sets coverage to fixture
  size `10183` + the injected `observedAt`; a second poll 24h later does not move it; fork-freeze and
  unverified verdicts both leave coverage unset.
- [x] `git diff HEAD~1..HEAD -- go.mod go.sum` — empty (no new dependency).
- [x] `go list -deps ./internal/store | grep '^net/http$'` — empty; store has zero internal
  iscc-monitor deps (still a leaf).
- [x] Scope discipline — only `internal/store/checkpoints.go` + `internal/follower/follower.go`
  (2 production files, ≤3 limit) + their two `_test.go` files; nothing from `## Not In Scope` touched.
- [x] Quality-gate integrity — no `//nolint`/`t.Skip`/build-tag/swallowed-error added across the three
  unpushed commits; no deleted tests or assertions; all changes additive.
- [x] Oracle/conformance gate — correctly **N/A**: no proof/verify/didweb/merkle/consistency/fsck/
  notecheck/signature path touched (plain `hubs`-column CRUD + a set-once write on the already-verified
  path); go.mod/go.sum byte-identical.

**Issues found:** (none)

**Next:** The merkle-backed **equivocation** trigger — the remaining unmet M1 Verify criterion (third
self-consistency trigger) and the first slice to trip the oracle/conformance gate. It adds a third
branch in `checkConsistency` returning `ViolationEquivocation` + real `ProofJSON`, needs
`transparency-dev/merkle` (new dep) + tile fixtures, and must preserve "compare against the prior
*accepted* root, not the contradicting evidence" (the `CheckpointAt` `LIMIT 1` rowid-order learning).
Lighter alternatives still open if the merkle slice is deferred: the `hub_keys` did:web cache write
(which must also refresh the stale `sb1.amlet.id_did.json` + `derive_vkey.py` HUBS to signer
`069d0f14`), structured logging (replacing the two stderr placeholders), `/metrics`, and surfacing the
coverage window through the dashboard/REST (M2/M3).

**Notes:**
- Independently verified the set-once guard is robust at the size-0 edge: the first `SetCoverage`
  writes `int64(size)` so the column is NOT NULL even when size==0, making every re-call a silent
  no-op (the start is immutable at size 0 too). Confirmed with a throwaway `TestCoverageZeroSizeStillSet`
  (PASS), then removed it — the committed suite does not cover size 0, but `PollHub` only ever records
  `info.TreeSize` from a verified checkpoint, so it is not a live concern.
- `SetCoverage` deliberately ignores `RowsAffected` (zero-rows-after-set is the correct non-error
  case), exactly as `next.md` scoped. The fork/unverified "coverage stays unset" assertions are
  non-vacuous because those seeds use `RecordCheckpoint`/`AdvanceFollowState`, never `SetCoverage`.
- The zero-`observedAt`→NULL store path is covered by a unit test for completeness but is unreachable
  from the live `PollHub` call (it always injects a real `observedAt`).
- No CI is configured (`.github/workflows/` absent) — `notecheck` parity job not yet present; flag for
  whoever wires CI, and it becomes load-bearing the moment the merkle equivocation slice lands.
- Working tree clean; commits are on `develop`; remote `origin` configured.
