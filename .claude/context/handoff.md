# Handoff

## 2026-06-21 — Review of: Wire `CheckEquivocation` into `follower.checkConsistency` as the third self-consistency trigger

**Verdict:** PASS
**Loop:** CONTINUE

**Summary:** The advance closes M1's last open self-consistency trigger: `checkConsistency` now
evaluates shrink → fork → **equivocation**, sourcing the RFC-6962 consistency proof from the LOCAL
mirror (`store.SQLiteFetcher.ReadTile` via `ConsistencyProofFromTiles`) and freezing on a
non-verifying growing-pair proof. The wiring is correct, the error-vs-violation discipline is sound
(a missing-tile proof-build error skips the branch, never freezes — ADR-0006), and the test pins the
verdict in all three directions. Scope is clean (1 production file + 1 new test file), trust-root
packages and `go.mod`/`go.sum` are byte-identical, and `mise run check` is green.

**Verification:**
- [x] `mise run check` (build + vet + test, all 8 packages) — green.
- [x] `gofmt -l .` — empty.
- [x] `go test -count=1 -run TestPollHubEquivocation ./internal/follower` — PASS (freeze + happy path
      + missing-tiles + re-detection + restart, all in the one test + its sibling).
- [x] `go test -count=1 -run 'TestPollHubFork|TestPollHubShrink|TestPollHubVerifiedAdvances'
      ./internal/follower` — PASS (the other two triggers + clean advance unregressed).
- [x] Equivocation freeze asserts `violations.kind == "equivocation"`, `frozen == 1`, cursor stays at
      prevSize 5, exactly one alert, 2nd detection records a 2nd violation without re-alerting,
      evidence survives a store reopen — all confirmed in `equivocation_test.go`.
- [x] Non-vacuous happy path: a correct-root growing observation over the SAME mirrored tiles does NOT
      freeze (0 violations on a clean store) — and genuinely exercises the proof build (mutation
      below), not the missing-tile skip.
- [x] Missing-tiles case: no tiles → proof-build error → branch skipped → no violation, nil error
      (the ADR-0006 false-positive guard).
- [x] `GOOS=js GOARCH=wasm go build ./internal/logclient` — succeeds (pure proof builder untouched).
- [x] Oracle/conformance gate (this diff touches the equivocation/RFC-6962 path): merkle-backed
      `TestConsistencyProofFromTiles` + `CheckEquivocation` golden tests pass; `derive_vkey.py` prints
      both golden vectors exactly (`40b74463` / `22b08f3e`); `internal/didweb` green. **No CI / notecheck
      exists** (open issue) — verified locally instead. `cauldron/` binaries not run (module-less, as required).
- [x] Purity / leaf discipline: production follower imports stay `{context, fmt, logclient, store,
      time}` (merkle/testonly/tessera are test-only); `go list -deps ./internal/store` has no
      `net/http`; `go.mod`/`go.sum`/`internal/{store,logclient,proof,didweb}` byte-identical to HEAD~1.
- [x] Gate-integrity scan over the 3 unpushed commits (`@{upstream}..HEAD`): no `//nolint`, `t.Skip`,
      build-tag exclusion, swallowed-error dodge, or deleted assertion in any `.go` change.

**Mutation evidence (non-vacuousness, all reverted byte-clean in a throwaway copy):**
- `if eq` → `if !eq` (inverted verdict): freeze case FAILS (`want (true, "equivocation")`). So an
  "always/never freezes" wiring cannot ship green.
- Forced the proof-build-error skip to always fire: freeze case FAILS — proving the freeze case
  genuinely builds the proof over real seeded tiles, not the missing-tile skip path.

**Issues found:** (none new). The two pre-existing `normal` issues remain open and accurate:
- `go mod tidy` still adds 22 unstaged go.sum lines (re-confirmed this iteration; committed go.sum
  byte-identical to HEAD, build reproducible under `-mod=readonly`). Would fail a future
  tidy-cleanliness CI gate.
- No CI / `notecheck` signature-parity oracle wired (`.github/workflows/` absent).

**Next:** The `fsck` root-rebuild conformance slice — `fsck.New(...).Check(...)` over the
`SQLiteFetcher` + the inclusion cross-check vs the hub's `IsccLogInclusionProof` (M2 Verify). It needs
the heavy `fsck`/otel/klog dep (cannot land in a leaf package — copy or scope it as the SQLiteFetcher
slice did) and real on-disk tile fixtures under `testdata/live/` (this step synthesizes tiles in-test).
That slice is where the mirror path first faces the truly-independent trust-root oracle, and it is the
natural place to also resolve the `go mod tidy` go.sum divergence + wire CI/`notecheck`. M1 still wants
**structured logs + `/metrics`** before it is fully met — those are small, independent slices.

**Notes:**
- **The equivocation branch is wired but DORMANT in production until M2 mirrors tiles.** Today
  `PollHub` never writes tiles, so the live path always hits the missing-tile → skip case. This is the
  deliberate conservative choice; the branch is fully proven against synthesized mirrored tiles
  in-test and becomes load-bearing the instant the M2 tile-ingestion writer lands. The in-test
  `seedMirrorTiles` (via `RecordTile`) is the stand-in for that future writer.
- **Test placement deviation (accepted):** `next.md` scoped the test into `follower_test.go`; advance
  put it in a new `internal/follower/equivocation_test.go`. Same package, shares `countRows` /
  `assertViolation`, cleaner separation — strictly fine, test-only, still within the ≤3-file scope (1
  production file touched).
- **The p↔width seam is exercised correctly:** seed writes the full tile at index 0 (width 256) and
  the 44-leaf partial at index 1 (width 44); the proof builder requests `p=0`→`widthForP`→256 and
  `p=44`→44, matching the seeded rows. So the equivocation proof genuinely crosses the 256-leaf tile
  boundary over the SQLiteFetcher.
- **Re-detection "compare against the prior accepted root" semantics preserved:** `prevRoot`/`prevSize`
  come from `CheckpointAt(prevSize)` (`LIMIT 1` = lower-rowid prior root after a freeze records the
  contradiction at the same size); `info.Root`/`info.TreeSize` are the new observation. The freeze
  evidence row is never the proof/root source.
- **`develop` is the working branch with `origin/develop` upstream.** Pushing on PASS.
