# Handoff

## 2026-06-21 — Review of: Thread `*metrics.Registry` into `PollHub`/`Tick` and fire the increment sites

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** Advance wired the pure `internal/metrics` leaf into the follower so all four metric
series move on the live verdict path, with the load-bearing verdict→glossary remap correct
(`rotated`→`unverified`, freeze→`frozen`). Scope is exactly the 2 implementation + 2 test files
specified; `main.go`/`cmd` untouched and still compiling; go.mod/go.sum byte-identical. Tests are
seam-driven (assert on `Registry.String()`, not internals) and I mutation-proved two of them
non-vacuous.

**Verification:**
- [x] `mise run check` — green (build + vet + test, all 9 packages). Uncached
      `go test -count=1 ./internal/follower ./internal/metrics ./cmd/iscc-monitor` also green.
- [x] `gofmt -l .` — empty.
- [x] `go test -run TestPollHub ./internal/follower` — PASS (Equivocation/Fork/Shrink/VerifiedAdvances/
      UnverifiedDoesNotAdvance/CacheHitSkipsDidFetch, all updated for the new param).
- [x] `go test -run TestGlossaryStatus ./internal/follower` — PASS: verified / verified-but-frozen→
      frozen / unverified / unresolvable / rotated→unverified.
- [x] Verified-observation `PollHub` test asserts `hub_status{hub_id="1",status="verified"} 1` and
      `last_observed_at{hub_id="1"} 1781913600` (= `observedAt.Unix()`, non-zero).
- [x] Fork/freeze `PollHub` test asserts `violations_total{hub_id="1",kind="fork"} 1` and
      `hub_status{hub_id="1",status="frozen"} 1`.
- [x] `TestTickMetricsPollFailure` asserts `poll_failures_total{hub_id="1"} 1` via the real
      `errFetcher` fault through `Tick`.
- [x] `go list -deps ./internal/store | grep '^net/http$'` — empty (store stays a leaf; the
      follower→metrics edge does not leak in).
- [x] `git diff --quiet HEAD~1..HEAD -- go.mod go.sum` — exit 0 (metrics is in-module; no dep change).
- [x] `cmd/iscc-monitor/main.go` untouched; the named-field `&follower.Loop{…}` literal compiles
      unchanged with the new optional `Metrics` field (left nil — wiring is the next slice).
- [x] Gate-integrity scan over all unpushed commits (`origin/develop..HEAD`): no `//nolint`/`t.Skip`/
      build-tag/swallowed-error/deleted-test/deleted-assertion.
- [x] Oracle/conformance gate **N/A** (verified): no verify/proof/consistency/merkle LOGIC line changed
      (grep-confirmed); the existing equivocation/fork/shrink conformance tests still pass, so no
      trust-root regression. Metrics WASM build green (leaf purity intact).

**Issues found:** (none) — both pre-existing `normal` issues (go.sum tidy divergence; no CI/`notecheck`)
remain open and were correctly untouched by this slice (no dep added, no CI wired).

**Next:** The `/metrics` HTTP slice — a `net/http` handler calling `Registry.WriteText` with
`Content-Type: text/plain; version=0.0.4`, plus `metrics.New()` in `cmd/iscc-monitor/main.go` passed as
`Loop.Metrics` and served. That completes M1's `/metrics` deliverable (this slice only fired the
increment sites; `main.go` still leaves `Metrics` nil). After that, the `fsck` root-rebuild conformance
slice (M2 Verify) — which is the natural point to also resolve the two open `normal` issues (the go.sum
tidy divergence and wiring CI/`notecheck`), since that slice first faces the trust-root oracle in CI.

**Notes:**
- **Glossary remap verified, not just asserted.** `glossaryStatus` folds `StatusRotated`→`"unverified"`
  and `frozen=true`→`"frozen"` (overriding the still-`StatusVerified` enum after a freeze). Two reverted
  mutations confirm the asserts are load-bearing: (1) freeze-branch `frozen=true→false` renders
  `status="verified"` → `TestPollHubFork` FAILS; (2) removing `IncViolation` → FAILS on missing
  `violations_total`. Tree restored clean after each.
- **`recordVerdict` is the single nil-safe mutation point** for `hub_status`/`last_observed_at`,
  funneling all three non-error verdict branches (early-non-verified, freeze, verified-advance). Error-
  return paths deliberately skip it (the verdict is unknown), so `IncPollFailure` fires in `Tick` on
  `PollHub`'s non-nil return — same branch as the `ErrorContext` log. `hub_id="1"` in the asserts is the
  real first-`UpsertHub` id (probed and confirmed).
- **Benign double-signal to watch (not a defect):** the freeze branch records `status="frozen"` BEFORE
  the `freeze()` store call (pure registry writes, documented at `follower.go:145`). If `freeze()` ever
  returns a store error, the same poll both records "frozen" and fires `Tick`'s `IncPollFailure` — fine
  today (a store fault during freeze IS a real poll failure), but revisit if a future slice makes
  `freeze`'s error path mean "violation not recorded."
- **Push:** remote `origin` configured (branch `develop`); pushing on PASS. Never push `main`.
